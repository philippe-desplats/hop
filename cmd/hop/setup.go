package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/philippe-desplats/hop/internal/action"
	"github.com/philippe-desplats/hop/internal/core"
	"github.com/philippe-desplats/hop/internal/i18n"
	"github.com/philippe-desplats/hop/internal/tui"
)

// cmdSetup runs the first-run guided configuration: detect candidate folders,
// editors and assistants, let the user confirm in a wizard (or fall back to a
// non-interactive preset with no tty), save the config, build the index, and
// optionally wire the shell integration into the user's rc file.
func cmdSetup(_ []string) {
	settings := core.LoadSettings()
	roots := core.DiscoverRoots()
	editors := action.DetectEditors()
	shell, rc, line, knownShell := detectShell()
	wired := shellAlreadyWired(rc)

	edited, confirmed, writeShell, err := tui.RunSetup(tui.SetupInput{
		Settings:     settings,
		Roots:        roots,
		Editors:      editors,
		Assistants:   action.DetectAssistants(),
		ShellName:    shell,
		RcLabel:      rc,
		AlreadyWired: wired,
	})
	if err != nil { // no tty: apply preset defaults, never touch the rc unprompted
		edited, confirmed, writeShell = presetSettings(settings, roots, editors, nil), true, false
	}
	if !confirmed {
		fmt.Fprintln(os.Stderr, i18n.T("setup.cancelled"))
		return
	}
	if err := core.SaveSettings(edited); err != nil {
		fatal(err)
	}
	idx := core.BuildAndSaveIndex(core.ScanConfig(edited))

	wroteShell := false
	// An unrecognised shell is never wired automatically: rc would be ~/.zshrc,
	// which that shell does not read. The summary prints the line to adapt instead.
	if writeShell && !wired && knownShell {
		if werr := wireShell(rc, line); werr != nil {
			fmt.Fprintln(os.Stderr, i18n.Tf("setup.shell_failed", rc))
		} else {
			wroteShell = true
		}
	}
	printSetupSummary(idx, rc, line, wired, wroteShell, knownShell)
}

// presetSettings folds auto-detected defaults onto settings without prompting:
// every folder holding a repo (or the first folder when none do) becomes a scan
// root, the first detected editor is selected, and the assistant stays "auto".
func presetSettings(s core.Settings, roots []core.RootCandidate, editors []action.Editor, _ []string) core.Settings {
	var chosen []string
	for _, r := range roots {
		if r.Repos > 0 {
			chosen = append(chosen, r.Path)
		}
	}
	if len(chosen) == 0 && len(roots) > 0 {
		chosen = []string{roots[0].Path}
	}
	if len(chosen) > 0 {
		s.Scan.Roots = chosen
	}
	if len(editors) > 0 {
		s.Actions.Editor = editors[0].Bin
	}
	s.AI.Tool = "auto"
	return s
}

// detectShell maps $SHELL to the shell name, its rc file and the exact line the
// user must add to wire up the daily `p` function.
//
// known tells the caller whether $SHELL was actually recognised. The zsh triple
// is returned either way, because it is still the most useful thing to suggest,
// but on an unknown shell it is only a suggestion: dash, ksh, sh, nushell and
// elvish never source ~/.zshrc, so writing into it would create a file the shell
// ignores and report a success the user cannot observe. An unset $SHELL, common
// under a GUI launcher, a container or `env -i`, stays a zsh default.
func detectShell() (shell, rc, line string, known bool) {
	const (
		zshRc   = "~/.zshrc"
		zshLine = `eval "$(hop init zsh)"`
	)
	switch filepath.Base(os.Getenv("SHELL")) {
	case "fish":
		return "fish", "~/.config/fish/config.fish", "hop init fish | source", true
	case "bash":
		return "bash", "~/.bashrc", `eval "$(hop init bash)"`, true
	case "zsh", ".": // filepath.Base("") is ".": $SHELL is unset
		return "zsh", zshRc, zshLine, true
	default:
		return "zsh", zshRc, zshLine, false
	}
}

// expandTilde resolves a leading ~ to the user's home directory.
func expandTilde(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			if p == "~" {
				return home
			}
			return filepath.Join(home, p[2:])
		}
	}
	return p
}

// shellAlreadyWired reports whether the rc file already sources hop, so setup
// never offers to add a duplicate line. Only an active line counts: a commented
// one wires nothing, and treating it as present would suppress the offer to add
// the real line and then claim an integration the next shell will not have.
func shellAlreadyWired(rc string) bool {
	data, err := os.ReadFile(expandTilde(rc))
	if err != nil {
		return false
	}
	for line := range strings.Lines(string(data)) {
		code := strings.TrimSpace(line)
		if code == "" || strings.HasPrefix(code, "#") {
			continue
		}
		if strings.Contains(code, "hop init") {
			return true
		}
	}
	return false
}

// wireShell appends the integration line to the rc file (creating it and its
// parent directory if needed), so the daily `p` function works on the next shell.
func wireShell(rc, line string) error {
	full := expandTilde(rc)
	if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
		return err
	}
	//nolint:gosec // rc is our own constant path from detectShell, not untrusted input; 0644 is the expected mode for a shell rc
	f, err := os.OpenFile(full, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	_, err = f.WriteString("\n# hop shell integration\n" + line + "\n")
	return err
}

func printSetupSummary(idx *core.Index, rc, line string, wired, wroteShell, knownShell bool) {
	fmt.Fprintln(os.Stderr, i18n.Tf("setup.done", len(idx.Projects)))
	fmt.Fprintln(os.Stderr)
	switch {
	case wroteShell:
		fmt.Fprintln(os.Stderr, i18n.Tf("setup.shell_done", rc))
	case wired:
		fmt.Fprintln(os.Stderr, i18n.Tf("setup.shell_present", rc))
	default:
		if !knownShell {
			fmt.Fprintln(os.Stderr, i18n.Tf("setup.shell_unknown", os.Getenv("SHELL")))
		}
		fmt.Fprintln(os.Stderr, i18n.Tf("setup.shell_hint", rc))
		fmt.Fprintln(os.Stderr, "    "+line)
	}
}
