package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/philippe-desplats/hop/internal/action"
	"github.com/philippe-desplats/hop/internal/core"
	"github.com/philippe-desplats/hop/internal/i18n"
	"github.com/philippe-desplats/hop/internal/tui"
)

func cmdScan(_ []string) {
	cfg := core.ScanConfig(core.LoadSettings())
	idx := core.BuildIndex(cfg)
	if err := core.SaveIndex(idx); err != nil {
		fatal(err)
	}
	cats := map[string]bool{}
	for _, p := range idx.Projects {
		if p.Category != "" {
			cats[p.Category] = true
		}
	}
	fmt.Fprintln(os.Stderr, i18n.Tf("cli.scan_summary", len(idx.Projects), len(cats)))
	if removed, _ := core.PruneFrecency(); removed > 0 {
		fmt.Fprintln(os.Stderr, i18n.Tf("cli.pruned", removed))
	}
}

// parseJumpBack parses "-", "-2", "-3" into the nth-most-recent index (n >= 1).
// ok is false when arg is not a jump-back token.
func parseJumpBack(arg string) (int, bool) {
	rest := strings.TrimPrefix(arg, "-")
	if rest == "" {
		return 1, true
	}
	if n, err := strconv.Atoi(rest); err == nil && n >= 1 {
		return n, true
	}
	return 0, false
}

// cmdClean prunes frecency entries whose directory no longer exists.
func cmdClean(_ []string) {
	removed, err := core.PruneFrecency()
	if err != nil {
		fatal(err)
	}
	fmt.Fprintln(os.Stderr, i18n.Tf("cli.pruned", removed))
}

// cmdPin pins the project matching the keywords so it floats to the top of the
// Hub; cmdUnpin removes it.
func cmdPin(args []string)   { pinOrUnpin(args, true) }
func cmdUnpin(args []string) { pinOrUnpin(args, false) }

func pinOrUnpin(args []string, pin bool) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "hop: usage: hop pin|unpin <keyword>")
		os.Exit(2)
	}
	settings := core.LoadSettings()
	cfg := core.ScanConfig(settings)
	idx, _ := core.LoadIndexOrBuild(cfg, true)
	weights := core.RankWeights{Fuzzy: settings.Resolver.WFuzzy, Frecency: settings.Resolver.WFrecency}
	matches := core.Resolve(idx.Projects, core.LoadFrecency(), args, time.Now(), weights)
	if len(matches) == 0 {
		fmt.Fprintln(os.Stderr, i18n.Tf("cli.no_project", strings.Join(args, " ")))
		os.Exit(1)
	}
	p := matches[0].Project
	if pin {
		if _, err := core.AddPin(p.Path); err != nil {
			fatal(err)
		}
		fmt.Fprintln(os.Stderr, i18n.Tf("cli.pinned", p.Name))
		return
	}
	if _, err := core.RemovePin(p.Path); err != nil {
		fatal(err)
	}
	fmt.Fprintln(os.Stderr, i18n.Tf("cli.unpinned", p.Name))
}

func cmdNav(args []string) {
	settings := core.LoadSettings()
	cfg := core.ScanConfig(settings)
	idx, _ := core.LoadIndexOrBuild(cfg, true)
	frec := core.LoadFrecency()
	now := time.Now()
	weights := core.RankWeights{Fuzzy: settings.Resolver.WFuzzy, Frecency: settings.Resolver.WFrecency}
	ai, hasAI := action.ResolveAssistant(settings.AI.Tool)
	opts := action.Options{
		Editor:   settings.Actions.Editor,
		ShowTmux: settings.Actions.ShowTmux,
		AI:       ai,
		HasAI:    hasAI,
		Custom:   settings.Actions.Custom,
	}

	// Jump-list: `p -` (previous), `p -2`, `p -3` (nth most recent, excluding cwd).
	if len(args) == 1 && strings.HasPrefix(args[0], "-") {
		if n, ok := parseJumpBack(args[0]); ok {
			cwd, _ := os.Getwd()
			prev := frec.NthMostRecentExcept(core.CanonicalDir(cwd), n)
			if prev == "" {
				fmt.Fprintln(os.Stderr, i18n.T("cli.no_prev"))
				os.Exit(1)
			}
			emitOutcome(action.Outcome{Cd: prev})
			return
		}
	}

	matches := core.Resolve(idx.Projects, frec, args, now, weights)

	if len(args) == 0 {
		proj, key, err := tui.Run(idx.Projects, frec, cfg.Roots, settings.Hub.ActionAccess, opts, weights, "", settings.UI.Theme)
		if err != nil {
			printFrequent(matches, cfg.Roots) // no tty: plain listing
			return
		}
		if proj == nil {
			return // cancelled
		}
		emitOutcome(actionOutcome(key, *proj, opts))
		return
	}

	// Opportunistic rescan: a freshly created project may not be indexed yet.
	if len(matches) == 0 {
		idx = core.BuildAndSaveIndex(cfg)
		matches = core.Resolve(idx.Projects, frec, args, now, weights)
	}
	if len(matches) == 0 {
		fmt.Fprintln(os.Stderr, i18n.Tf("cli.no_project", strings.Join(args, " ")))
		os.Exit(1)
	}

	// Ambiguous fragment (top two scores too close): open the Hub pre-filtered
	// instead of guessing. Falls back to jumping when there is no terminal.
	if len(matches) >= 2 && matches[0].Final-matches[1].Final < settings.Resolver.MinMargin {
		if proj, key, err := tui.Run(idx.Projects, frec, cfg.Roots, settings.Hub.ActionAccess, opts, weights, strings.Join(args, " "), settings.UI.Theme); err == nil {
			if proj != nil {
				emitOutcome(actionOutcome(key, *proj, opts))
			}
			return
		}
	}

	// Centralized sentinel emission; the p() shell function consumes it.
	emitOutcome(action.Outcome{Cd: matches[0].Project.Path})
}

func actionOutcome(key string, p core.Project, opts action.Options) action.Outcome {
	if spec, ok := action.ByKey(key, p, opts); ok {
		return spec.Do(p)
	}
	return action.Outcome{Cd: p.Path} // default: cd
}

// emitOutcome writes the protocol the `p` shell function parses: a cd target
// and/or a command to run after cd.
func emitOutcome(o action.Outcome) {
	if o.Cd != "" {
		fmt.Printf("__HOP_CD__ %s\n", o.Cd)
	}
	if o.Run != "" {
		fmt.Printf("__HOP_RUN__ %s\n", o.Run)
	}
}

func printFrequent(matches []core.Match, roots []string) {
	if len(matches) == 0 {
		fmt.Fprintln(os.Stderr, i18n.T("cli.no_index"))
		return
	}
	n := len(matches)
	if n > 12 {
		n = 12
	}
	fmt.Fprintln(os.Stderr, i18n.Tf("cli.frequent_header", len(matches)))
	for i, m := range matches[:n] {
		fmt.Fprintf(os.Stderr, "  %2d  %s\n", i+1, core.DisplayPath(m.Project.Path, roots))
	}
	fmt.Fprintln(os.Stderr, i18n.T("cli.tip"))
}

func cmdAdd(args []string) {
	if len(args) == 0 {
		return
	}
	cfg := core.ScanConfig(core.LoadSettings())
	p := core.CanonicalDir(args[0])
	if !core.UnderRoots(p, cfg.Roots) {
		return // only learn paths inside configured roots
	}
	_, _ = core.AddFrecency(p, time.Now(), false) // try-lock, best effort
}

// cmdPrompt prints the name of the project containing the current directory
// (deepest match), or nothing. Meant for a prompt segment, so it is fast and
// never triggers a scan.
func cmdPrompt(_ []string) {
	idx, err := core.LoadIndex()
	if err != nil {
		return
	}
	cwd, err := os.Getwd()
	if err != nil {
		return
	}
	cwd = core.CanonicalDir(cwd)
	best, name := "", ""
	for _, p := range idx.Projects {
		if cwd == p.Path || strings.HasPrefix(cwd, p.Path+string(os.PathSeparator)) {
			if len(p.Path) > len(best) {
				best, name = p.Path, p.Name
			}
		}
	}
	if name != "" {
		fmt.Print(name)
	}
}

// cmdComplete prints project names matching the fragment, one per line, for shell
// completion. It reads the index only (no scan) so it stays fast on every TAB.
func cmdComplete(args []string) {
	idx, err := core.LoadIndex()
	if err != nil {
		return
	}
	names := make([]string, len(idx.Projects))
	for i, p := range idx.Projects {
		names[i] = p.Name
	}
	for _, n := range completeMatches(names, strings.Join(args, " ")) {
		fmt.Println(n)
	}
}

// completeMatches returns the distinct names containing frag (case-insensitive),
// preserving order. An empty fragment returns all distinct names.
func completeMatches(names []string, frag string) []string {
	frag = strings.ToLower(frag)
	seen := map[string]bool{}
	var out []string
	for _, n := range names {
		if seen[n] {
			continue
		}
		if frag == "" || strings.Contains(strings.ToLower(n), frag) {
			seen[n] = true
			out = append(out, n)
		}
	}
	return out
}

func cmdConfig(_ []string) {
	if _, err := core.EnsureConfig(); err != nil {
		fatal(err)
	}
	s := core.LoadSettings()
	edited, saved, err := tui.RunConfig(s)
	if err != nil { // no tty: print current config instead
		fmt.Println(core.ConfigPath())
		fmt.Printf("action_access = %q\n", s.Hub.ActionAccess)
		return
	}
	if saved {
		if err := core.SaveSettings(edited); err != nil {
			fatal(err)
		}
		fmt.Fprintln(os.Stderr, i18n.Tf("cli.config_saved", core.ConfigPath()))
	}
}

func cmdDoctor(_ []string) {
	settings := core.LoadSettings()
	cfg := core.ScanConfig(settings)
	rootLbl, binLbl := i18n.T("cli.doctor.root"), i18n.T("cli.doctor.bin")
	fmt.Println("hop doctor")
	for _, r := range cfg.Roots {
		status := "OK"
		if fi, err := os.Stat(r); err != nil || !fi.IsDir() {
			status = "✗"
		}
		fmt.Printf("  %-8s %-42s %s\n", rootLbl, r, status)
	}
	for _, bin := range []string{"zed", "claude", "git", "tmux"} {
		status := "OK"
		if _, err := exec.LookPath(bin); err != nil {
			status = "✗"
		}
		fmt.Printf("  %-8s %-42s %s\n", binLbl, bin, status)
	}
	list := strings.Join(action.DetectAssistants(), ", ")
	if list == "" {
		list = "(none found)"
	}
	active := "none"
	if ai, ok := action.ResolveAssistant(settings.AI.Tool); ok {
		active = ai.Name
	}
	fmt.Printf("  %-8s %-42s %s\n", "ai", list, "active: "+active)
	if idx, err := core.LoadIndex(); err == nil {
		fmt.Printf("  %-8s %-42s %d\n", "index", core.IndexPath(), len(idx.Projects))
	} else {
		fmt.Printf("  %-8s %-42s %s\n", "index", core.IndexPath(), i18n.T("cli.doctor.index_missing"))
	}
}
