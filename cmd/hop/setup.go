package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/philippe-desplats/hop/internal/action"
	"github.com/philippe-desplats/hop/internal/core"
	"github.com/philippe-desplats/hop/internal/i18n"
	"github.com/philippe-desplats/hop/internal/tui"
)

// cmdSetup runs the first-run guided configuration: detect candidate folders,
// editors and assistants, let the user confirm in a wizard (or fall back to a
// non-interactive preset with no tty), then save the config, build the index and
// print the one shell line left to paste.
func cmdSetup(_ []string) {
	settings := core.LoadSettings()
	roots := core.DiscoverRoots()
	editors := action.DetectEditors()
	assistants := action.DetectAssistants()

	edited, confirmed, err := tui.RunSetup(settings, roots, editors, assistants)
	if err != nil { // no tty: apply the same defaults the wizard preselects
		edited, confirmed = presetSettings(settings, roots, editors, assistants), true
	}
	if !confirmed {
		fmt.Fprintln(os.Stderr, i18n.T("setup.cancelled"))
		return
	}
	if err := core.SaveSettings(edited); err != nil {
		fatal(err)
	}
	idx := core.BuildAndSaveIndex(core.ScanConfig(edited))
	printSetupSummary(idx)
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
// user must add to wire up the daily `p` function. Defaults to zsh.
func detectShell() (shell, rc, line string) {
	switch filepath.Base(os.Getenv("SHELL")) {
	case "fish":
		return "fish", "~/.config/fish/config.fish", "hop init fish | source"
	case "bash":
		return "bash", "~/.bashrc", `eval "$(hop init bash)"`
	default:
		return "zsh", "~/.zshrc", `eval "$(hop init zsh)"`
	}
}

func printSetupSummary(idx *core.Index) {
	_, rc, line := detectShell()
	fmt.Fprintln(os.Stderr, i18n.Tf("setup.done", len(idx.Projects)))
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, i18n.Tf("setup.shell_hint", rc))
	fmt.Fprintln(os.Stderr, "    "+line)
}
