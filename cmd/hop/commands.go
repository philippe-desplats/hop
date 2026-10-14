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
	// A prune failure is a warning here, not fatal: the scan is what the user
	// asked for and it succeeded. But it is never silent either: a frecency
	// table that can no longer be rewritten keeps ranking dead entries, and the
	// user has to hear about it to go and fix the permissions.
	pruned, err := pruneDeadPaths()
	if err != nil {
		fmt.Fprintln(os.Stderr, i18n.Tf("cli.prune_failed", err))
	}
	if pruned > 0 {
		fmt.Fprintln(os.Stderr, i18n.Tf("cli.pruned", pruned))
	}
}

// pruneDeadPaths drops the frecency, pin and tracked-folder entries whose
// directory no longer exists, returning how many went in total. It stops at the
// first failure, so the count it returns on an error is what was actually
// removed before it.
//
// `hop scan` and `hop clean` share it because they prune the same three stores;
// running the sequence twice is how the two came to disagree about what a
// failure means in the first place.
func pruneDeadPaths() (int, error) {
	removed, err := core.PruneFrecency()
	if err != nil {
		return removed, err
	}
	pins, err := core.PrunePins()
	if err != nil {
		return removed + pins, err
	}
	extras, err := core.PruneExtras()
	return removed + pins + extras, err
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
// longer exists. Pruning is all it does, so a failure is fatal, and the count is
// printed even at zero: that line is the command's whole answer.
func cmdClean(_ []string) {
	pruned, err := pruneDeadPaths()
	if err != nil {
		fatal(err)
	}
	fmt.Fprintln(os.Stderr, i18n.Tf("cli.pruned", pruned))
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
	changed, failed := false, false
	for _, arg := range args {
		path := core.CanonicalDir(core.ExpandHome(arg))
		if core.HasControlChars(path) {
			fmt.Fprintln(os.Stderr, i18n.T("cli.unsafe_path"))
			failed = true
			continue
		}
		label := core.HomeRelative(path)
		if track {
			//nolint:gosec // path is the folder the user explicitly asked to track, already canonicalized and control-char checked; we only stat it to confirm it is a directory.
			if fi, err := os.Stat(path); err != nil || !fi.IsDir() {
				fmt.Fprintln(os.Stderr, i18n.Tf("cli.track_not_dir", arg))
				failed = true
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
	// Exit code rule, git's: one rejected argument fails the command, even when
	// the others went through. `hop track ~/Dowloads && …` must not print ok on a
	// typo, and a script tracking a list must not report success having tracked
	// none of them. An argument that was simply already (un)tracked is a no-op,
	// not a failure.
	if failed {
		os.Exit(1)
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
		// A --from without a value is a usage error, like `hop init --cmd`:
		// swallowing it would run a committed import, writing extras.json and
		// frecency.json, on a `hop import --from zoxide --dry-run` the user lost
		// the tail of, with nothing to tell it apart from the intended one.
		case a == "--from":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, i18n.T("cli.import_from_no_source"))
				os.Exit(2)
			}
			i++
			source = args[i]
		case strings.HasPrefix(a, "--from="):
			source = strings.TrimPrefix(a, "--from=")
			if source == "" {
				fmt.Fprintln(os.Stderr, i18n.T("cli.import_from_no_source"))
				os.Exit(2)
			}
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
		// Separate keys, not one "pin|unpin" message: the user typed exactly one
		// of the two, and naming both leaves them to work out which they got wrong.
		usage := "cli.unpin_usage"
		if pin {
			usage = "cli.pin_usage"
		}
		fmt.Fprintln(os.Stderr, i18n.T(usage))
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
	// A leading `-` is a flag namespace, never a keyword: keywords are matched as
	// a substring of the lowercased full path, so handing an unparsed `-h` to the
	// resolver would cd the user into the first project whose path contains "-h".
	if len(args) == 1 && strings.HasPrefix(args[0], "-") {
		n, ok := parseJumpBack(args[0])
		if !ok {
			fmt.Fprintln(os.Stderr, i18n.Tf("cli.nav_unknown_option", args[0]))
			fmt.Fprintln(os.Stderr, i18n.T("cli.nav_jump_usage"))
			os.Exit(2)
		}
		cwd, _ := os.Getwd()
		prev := frec.NthMostRecentExcept(core.CanonicalDir(cwd), n)
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
	if !core.UnderRoots(p, cfg.Roots) && !isTracked(p) {
		return // only learn paths under a configured root, or tracked folders
	}
	_, _ = core.AddFrecency(p, time.Now(), false) // try-lock, best effort
}

// isTracked reports whether p is one of the folders `hop track` recorded. A
// tracked folder is in the index just like a scanned one, so visiting it has to
// earn frecency, or it would sort last among its matches forever.
//
// It reads extras.json, so cmdAdd only consults it once the roots test has
// already failed: the shell hook calls `hop add` on every cd and the common
// case, a path under a root, must stay the in-memory check it was.
func isTracked(p string) bool {
	for _, x := range core.LoadExtras().Paths {
		if x == p {
			return true
		}
	}
	return false
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
	if err != nil {
		fatalUnlessNoTTY(err)
		printSettings(s) // no tty: print the current config instead
		return
	}
	if saved {
		if err := core.SaveSettings(edited); err != nil {
			fatal(err)
		}
		fmt.Fprintln(os.Stderr, i18n.Tf("cli.config_saved", core.ConfigPath()))
	}
}

// printSettings lists every field the config editor exposes, in the order the
// editor shows them, under the path of the file holding them. It is what a user
// without a terminal gets instead of the editor, so printing one field would
// leave them knowing nothing about their editor, their roots or their language.
// The values are printed quoted, close to how the file holds them, but the
// output carries no section headers, so a line is not a config.toml fragment on
// its own: it belongs under the right section, which the reader has to supply.
func printSettings(s core.Settings) {
	fmt.Println(core.ConfigPath())
	fmt.Printf("command = %q\n", s.Shell.Command)
	fmt.Printf("action_access = %q\n", s.Hub.ActionAccess)
	fmt.Printf("editor = %q\n", s.Actions.Editor)
	fmt.Printf("show_tmux = %t\n", s.Actions.ShowTmux)
	fmt.Printf("roots = %s\n", tomlList(s.Scan.Roots))
	fmt.Printf("max_depth = %d\n", s.Scan.MaxDepth)
	fmt.Printf("ignore = %s\n", tomlList(s.Scan.Ignore))
	fmt.Printf("language = %q\n", s.UI.Language)
	fmt.Printf("theme = %q\n", s.UI.Theme)
}

// tomlList renders a string slice as a TOML array of quoted strings.
func tomlList(vals []string) string {
	quoted := make([]string, len(vals))
	for i, v := range vals {
		quoted[i] = fmt.Sprintf("%q", v)
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

func cmdDoctor(_ []string) {
	settings := core.LoadSettings()
	cfg := core.ScanConfig(settings)
	rootLbl, binLbl := i18n.T("cli.doctor.root"), i18n.T("cli.doctor.bin")
	okLbl, noneFound := i18n.T("cli.doctor.ok"), i18n.T("cli.doctor.none_found")
	fmt.Println(i18n.T("cli.doctor.title"))
	for _, r := range cfg.Roots {
		status := okLbl
		if fi, err := os.Stat(r); err != nil || !fi.IsDir() {
			status = "✗"
		}
		fmt.Printf("  %-8s %-42s %s\n", rootLbl, r, status)
	}
	for _, bin := range []string{settings.Actions.Editor, "git", "tmux"} {
		status := okLbl
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
		edlist = noneFound
	}
	fmt.Printf("  %-8s %-42s %s\n", i18n.T("cli.doctor.editor"), edlist, i18n.Tf("cli.doctor.active", settings.Actions.Editor))
	list := strings.Join(action.DetectAssistants(), ", ")
	if list == "" {
		list = noneFound
	}
	active := i18n.T("cli.doctor.none")
	if ai, ok := action.ResolveAssistant(settings.AI.Tool); ok {
		active = ai.Name
	}
	fmt.Printf("  %-8s %-42s %s\n", i18n.T("cli.doctor.ai"), list, i18n.Tf("cli.doctor.active", active))
	for _, bad := range action.InvalidCustomActions(settings.Actions.Custom) {
		fmt.Printf("  %-8s %-42s %s\n", i18n.T("cli.doctor.action"), bad, "✗")
	}
	idxLbl := i18n.T("cli.doctor.index")
	if idx, err := core.LoadIndex(); err == nil {
		fmt.Printf("  %-8s %-42s %d\n", idxLbl, core.IndexPath(), len(idx.Projects))
	} else {
		fmt.Printf("  %-8s %-42s %s\n", idxLbl, core.IndexPath(), i18n.T("cli.doctor.index_missing"))
	}
}
