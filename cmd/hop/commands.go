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
	removed, _ := core.PruneFrecency()
	pins, _ := core.PrunePins()
	extras, _ := core.PruneExtras()
	if removed+pins+extras > 0 {
		fmt.Fprintln(os.Stderr, i18n.Tf("cli.pruned", removed+pins+extras))
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

// cmdClean prunes frecency, pin and tracked-folder entries whose directory no
// longer exists.
func cmdClean(_ []string) {
	removed, err := core.PruneFrecency()
	if err != nil {
		fatal(err)
	}
	pins, err := core.PrunePins()
	if err != nil {
		fatal(err)
	}
	extras, err := core.PruneExtras()
	if err != nil {
		fatal(err)
	}
	fmt.Fprintln(os.Stderr, i18n.Tf("cli.pruned", removed+pins+extras))
}

// cmdTrack adds folders to the search list so they appear even without a git
// repository; cmdUntrack removes them. With no argument both act on the current
// directory.
func cmdTrack(args []string)   { trackOrUntrack(args, true) }
func cmdUntrack(args []string) { trackOrUntrack(args, false) }

func trackOrUntrack(args []string, track bool) {
	if len(args) == 0 {
		cwd, err := os.Getwd()
		if err != nil {
			fatal(err)
		}
		args = []string{cwd}
	}
	changed := false
	for _, arg := range args {
		path := core.CanonicalDir(expandTilde(arg))
		if core.HasControlChars(path) {
			fmt.Fprintln(os.Stderr, i18n.T("cli.unsafe_path"))
			continue
		}
		label := core.HomeRelative(path)
		if track {
			//nolint:gosec // path is the folder the user explicitly asked to track, already canonicalized and control-char checked; we only stat it to confirm it is a directory.
			if fi, err := os.Stat(path); err != nil || !fi.IsDir() {
				fmt.Fprintln(os.Stderr, i18n.Tf("cli.track_not_dir", arg))
				continue
			}
			added, err := core.AddExtra(path)
			if err != nil {
				fatal(err)
			}
			if added {
				changed = true
				fmt.Fprintln(os.Stderr, i18n.Tf("cli.tracked", label))
			} else {
				fmt.Fprintln(os.Stderr, i18n.Tf("cli.track_already", label))
			}
			continue
		}
		removed, err := core.RemoveExtra(path)
		if err != nil {
			fatal(err)
		}
		if removed {
			changed = true
			fmt.Fprintln(os.Stderr, i18n.Tf("cli.untracked", label))
		} else {
			fmt.Fprintln(os.Stderr, i18n.Tf("cli.track_not_found", label))
		}
	}
	// Rebuild so the change is reflected on the next `p` without waiting for a scan.
	if changed {
		core.BuildAndSaveIndex(core.ScanConfig(core.LoadSettings()))
	}
}

// cmdImport seeds hop's ranking from another tool's history. Only zoxide is
// supported today: `hop import --from zoxide [--dry-run]`. It shells out to
// `zoxide query --list --score` (the stable text interface, never the binary db)
// and never fails silently: it always prints the imported/tracked/skipped counts.
func cmdImport(args []string) {
	source := "zoxide"
	dryRun := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--dry-run":
			dryRun = true
		case a == "--from":
			if i+1 < len(args) {
				i++
				source = args[i]
			}
		case strings.HasPrefix(a, "--from="):
			source = strings.TrimPrefix(a, "--from=")
		default:
			fmt.Fprintln(os.Stderr, i18n.Tf("cli.import_unknown_flag", a))
			os.Exit(2)
		}
	}
	if source != "zoxide" {
		fmt.Fprintln(os.Stderr, i18n.Tf("cli.import_unknown_source", source))
		os.Exit(2)
	}
	if _, err := exec.LookPath("zoxide"); err != nil {
		fmt.Fprintln(os.Stderr, i18n.T("cli.import_no_zoxide"))
		os.Exit(1)
	}
	out, err := exec.Command("zoxide", "query", "--list", "--score").Output()
	if err != nil {
		fmt.Fprintln(os.Stderr, i18n.Tf("cli.import_failed", err))
		os.Exit(1)
	}
	entries := core.ParseZoxideOutput(string(out))

	cfg := core.ScanConfig(core.LoadSettings())
	idx, _ := core.LoadIndexOrBuild(cfg, true)
	indexed := make(map[string]bool, len(idx.Projects))
	for _, p := range idx.Projects {
		indexed[p.Path] = true
	}

	imported, tracked, skipped, err := core.ImportZoxide(entries, indexed, time.Now(), dryRun)
	if err != nil {
		fatal(err)
	}
	if dryRun {
		fmt.Fprintln(os.Stderr, i18n.Tf("cli.import_dry", imported, tracked, skipped))
		return
	}
	fmt.Fprintln(os.Stderr, i18n.Tf("cli.import_done", imported, tracked, skipped))
	if tracked > 0 {
		core.BuildAndSaveIndex(cfg) // surface newly tracked folders on the next `p`
	}
}

// cmdQuery resolves keywords and prints the best match's path to stdout, as a
// plain string (no sentinel, no frecency write) for scripting and composition
// with other tools, the hop counterpart of `zoxide query`. With no keyword it
// prints the top-frecency path; with --list it prints every match path,
// frecency-descending, ignoring any keyword. No match exits 1 with empty stdout.
func cmdQuery(args []string) {
	list := false
	var kws []string
	for _, a := range args {
		if a == "--list" {
			list = true
			continue
		}
		kws = append(kws, a)
	}
	if list {
		kws = nil // --list lists everything by frecency, keywords are ignored
	}

	settings := core.LoadSettings()
	cfg := core.ScanConfig(settings)
	idx, _ := core.LoadIndexOrBuild(cfg, false)
	frec := core.LoadFrecency()
	now := time.Now()
	weights := core.RankWeights{Fuzzy: settings.Resolver.WFuzzy, Frecency: settings.Resolver.WFrecency}

	matches := core.Resolve(idx.Projects, frec, kws, now, weights)
	// Opportunistic rescan on a keyword miss, mirroring cmdNav, so a freshly
	// created project resolves.
	if len(matches) == 0 && len(kws) > 0 {
		idx = core.BuildAndSaveIndex(cfg)
		matches = core.Resolve(idx.Projects, frec, kws, now, weights)
	}

	if list {
		paths := safePaths(matches)
		if len(paths) == 0 {
			os.Exit(1)
		}
		for _, p := range paths {
			fmt.Println(p)
		}
		return
	}
	if path, ok := firstSafePath(matches); ok {
		fmt.Println(path)
		return
	}
	os.Exit(1)
}

// firstSafePath returns the best match's path, skipping any path that would be
// unsafe to print (a control char could smuggle a second protocol line into a
// caller that eval's the output). ok is false when no safe match exists.
func firstSafePath(matches []core.Match) (string, bool) {
	for _, m := range matches {
		if !core.HasControlChars(m.Project.Path) {
			return m.Project.Path, true
		}
	}
	return "", false
}

// safePaths returns every match path that is safe to print, preserving order.
func safePaths(matches []core.Match) []string {
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		if !core.HasControlChars(m.Project.Path) {
			out = append(out, m.Project.Path)
		}
	}
	return out
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
	if !core.HasIndex() {
		fmt.Fprintln(os.Stderr, i18n.T("cli.setup_hint")) // first run: point at `hop setup`
	}
	cfg := core.ScanConfig(settings)
	idx, _ := core.LoadIndexOrBuild(cfg, true)
	frec := core.LoadFrecency()
	now := time.Now()
	weights := core.RankWeights{Fuzzy: settings.Resolver.WFuzzy, Frecency: settings.Resolver.WFrecency}
	ai, hasAI := action.ResolveAssistant(settings.AI.Tool)
	opts := action.Options{
		Editor:      settings.Actions.Editor,
		Multiplexer: action.ResolveMultiplexer(settings.Actions.Multiplexer, settings.Actions.ShowTmux),
		AI:          ai,
		HasAI:       hasAI,
		Custom:      settings.Actions.Custom,
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
// and/or a command to run after cd. The shell eval's this output, so a Cd or Run
// holding a newline could smuggle an extra __HOP_RUN__ line; such values are
// refused (exit non-zero, the shell function then aborts without cd/eval).
func emitOutcome(o action.Outcome) {
	if core.HasControlChars(o.Cd) || core.HasControlChars(o.Run) {
		fmt.Fprintln(os.Stderr, i18n.T("cli.unsafe_path"))
		os.Exit(1)
	}
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
	if core.HasControlChars(p) {
		return // never index a path that could break the shell protocol
	}
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
	for _, bin := range []string{settings.Actions.Editor, "git", "tmux"} {
		status := "OK"
		if _, err := exec.LookPath(bin); err != nil {
			status = "✗"
		}
		fmt.Printf("  %-8s %-42s %s\n", binLbl, bin, status)
	}
	var editorBins []string
	for _, e := range action.DetectEditors() {
		editorBins = append(editorBins, e.Bin)
	}
	edlist := strings.Join(editorBins, ", ")
	if edlist == "" {
		edlist = "(none found)"
	}
	fmt.Printf("  %-8s %-42s %s\n", "editor", edlist, "active: "+settings.Actions.Editor)
	list := strings.Join(action.DetectAssistants(), ", ")
	if list == "" {
		list = "(none found)"
	}
	active := "none"
	if ai, ok := action.ResolveAssistant(settings.AI.Tool); ok {
		active = ai.Name
	}
	fmt.Printf("  %-8s %-42s %s\n", "ai", list, "active: "+active)
	for _, bad := range action.InvalidCustomActions(settings.Actions.Custom) {
		fmt.Printf("  %-8s %-42s %s\n", "action", bad, "✗")
	}
	if idx, err := core.LoadIndex(); err == nil {
		fmt.Printf("  %-8s %-42s %d\n", "index", core.IndexPath(), len(idx.Projects))
	} else {
		fmt.Printf("  %-8s %-42s %s\n", "index", core.IndexPath(), i18n.T("cli.doctor.index_missing"))
	}
}
