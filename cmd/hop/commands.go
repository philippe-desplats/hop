package main

import (
	"fmt"
	"os"
	"os/exec"
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
}

func cmdNav(args []string) {
	settings := core.LoadSettings()
	cfg := core.ScanConfig(settings)
	idx, _ := core.LoadIndexOrBuild(cfg, true)
	frec := core.LoadFrecency()
	now := time.Now()
	weights := core.RankWeights{Fuzzy: settings.Resolver.WFuzzy, Frecency: settings.Resolver.WFrecency}
	opts := action.Options{Editor: settings.Actions.Editor, ShowTmux: settings.Actions.ShowTmux}

	// `p -` : jump back to the previous project (second most recent visit).
	if len(args) == 1 && args[0] == "-" {
		cwd, _ := os.Getwd()
		prev := frec.MostRecentExcept(core.CanonicalDir(cwd))
		if prev == "" {
			fmt.Fprintln(os.Stderr, i18n.T("cli.no_prev"))
			os.Exit(1)
		}
		emitOutcome(action.Outcome{Cd: prev})
		return
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
	cfg := core.ScanConfig(core.LoadSettings())
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
	if idx, err := core.LoadIndex(); err == nil {
		fmt.Printf("  %-8s %-42s %d\n", "index", core.IndexPath(), len(idx.Projects))
	} else {
		fmt.Printf("  %-8s %-42s %s\n", "index", core.IndexPath(), i18n.T("cli.doctor.index_missing"))
	}
}
