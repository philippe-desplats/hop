package tui

import (
	"errors"
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/philippe-desplats/hop/internal/action"
	"github.com/philippe-desplats/hop/internal/core"
	"github.com/philippe-desplats/hop/internal/i18n"
)

func TestNewSetupModelPreselectsReposAndDefaults(t *testing.T) {
	m := newSetupModel(SetupInput{
		Settings:   core.DefaultSettings(),
		Roots:      []core.RootCandidate{{Path: "~/code", Repos: 3}, {Path: "~/empty", Repos: 0}},
		Editors:    []action.Editor{{Name: "Cursor", Bin: "cursor"}, {Name: "Zed", Bin: "zed"}},
		Assistants: []string{"claude", "codex"},
		ShellName:  "zsh",
		RcLabel:    "~/.zshrc",
	})

	if !m.rootSel[0] || m.rootSel[1] {
		t.Errorf("rootSel = %v, want only the repo-bearing folder checked", m.rootSel)
	}
	if m.aiOptions[0] != "auto" || len(m.aiOptions) != 3 {
		t.Errorf("aiOptions = %v, want [auto claude codex]", m.aiOptions)
	}
	if !m.writeShell {
		t.Error("writeShell should default to true when the rc is not already wired")
	}
	s := m.settings()
	if len(s.Scan.Roots) != 1 || s.Scan.Roots[0] != "~/code" {
		t.Errorf("roots = %v, want [~/code]", s.Scan.Roots)
	}
	// The base settings configure "zed", which the detected list offers second:
	// the wizard must come up on it rather than on the first detected editor.
	if s.Actions.Editor != "zed" {
		t.Errorf("editor = %q, want the configured zed", s.Actions.Editor)
	}
	if s.AI.Tool != "auto" {
		t.Errorf("ai = %q, want auto", s.AI.Tool)
	}
}

// A re-run of the wizard must come up on the editor and the assistant the user
// already configured. settings() writes back whatever is preselected, so coming
// up on the first detected editor and on "auto" replaces chosen values with
// detected ones — and README documents re-running setup as supported.
func TestNewSetupModelPreselectsTheConfiguredEditorAndAssistant(t *testing.T) {
	base := core.DefaultSettings()
	base.Actions.Editor, base.AI.Tool = "nvim", "codex"
	m := newSetupModel(SetupInput{
		Settings:   base,
		Roots:      []core.RootCandidate{{Path: "~/code", Repos: 3}},
		Editors:    []action.Editor{{Name: "Cursor", Bin: "cursor"}, {Name: "Neovim", Bin: "nvim"}},
		Assistants: []string{"claude", "codex"},
	})

	s := m.settings()
	if s.Actions.Editor != "nvim" {
		t.Errorf("editor = %q, want the configured nvim", s.Actions.Editor)
	}
	if s.AI.Tool != "codex" {
		t.Errorf("ai = %q, want the configured codex", s.AI.Tool)
	}
}

// With nothing configured yet there is nothing to preserve, so the wizard keeps
// its first-run preselections.
func TestNewSetupModelFirstRunTakesTheFirstDetectedEditor(t *testing.T) {
	m := newSetupModel(SetupInput{
		Settings:   core.Settings{}, // a config hop has never written
		Roots:      []core.RootCandidate{{Path: "~/code", Repos: 1}},
		Editors:    []action.Editor{{Name: "Cursor", Bin: "cursor"}, {Name: "Zed", Bin: "zed"}},
		Assistants: []string{"claude"},
	})

	s := m.settings()
	if s.Actions.Editor != "cursor" {
		t.Errorf("editor = %q, want the first detected cursor", s.Actions.Editor)
	}
	if s.AI.Tool != "auto" {
		t.Errorf("ai = %q, want auto", s.AI.Tool)
	}
}

// A configured editor the wizard cannot offer — not on PATH, so not detected —
// cannot be preselected: the cursor lands on the first offered editor, and
// confirming the step replaces the configured binary with that one. The step
// shows the replacement, which is the only thing standing between the user and
// losing the value; this test pins the fallback, not an endorsement of it.
func TestNewSetupModelFallsBackWhenTheConfiguredEditorIsNotDetected(t *testing.T) {
	base := core.DefaultSettings()
	base.Actions.Editor = "emacs"
	m := newSetupModel(SetupInput{
		Settings: base,
		Editors:  []action.Editor{{Name: "Cursor", Bin: "cursor"}, {Name: "Zed", Bin: "zed"}},
	})

	if m.editorIdx != 0 {
		t.Errorf("editorIdx = %d, want the 0 fallback for an undetected editor", m.editorIdx)
	}
	if got := m.settings().Actions.Editor; got != "cursor" {
		t.Errorf("editor = %q, want the fallback cursor", got)
	}
}

func TestNewSetupModelFallsBackToFirstRootAndKeepsDefaults(t *testing.T) {
	base := core.DefaultSettings()
	m := newSetupModel(SetupInput{
		Settings: base,
		Roots:    []core.RootCandidate{{Path: "~/a", Repos: 0}, {Path: "~/b", Repos: 0}},
	}) // no editors, no assistants

	if !m.rootSel[0] || m.rootSel[1] {
		t.Errorf("rootSel = %v, want first checked as fallback", m.rootSel)
	}
	s := m.settings()
	if s.Actions.Editor != base.Actions.Editor {
		t.Errorf("editor = %q, want unchanged default %q", s.Actions.Editor, base.Actions.Editor)
	}
	if s.AI.Tool != "auto" {
		t.Errorf("ai = %q, want auto", s.AI.Tool)
	}
}

// Unchecking every folder cannot mean "keep the configured roots": settings()
// falls back to base when the selection is empty, and the confirm screen is
// built from that same call, so the wizard would present a root the user just
// deselected as their choice. The step must refuse to advance and say so.
func TestSetupRootsStepRefusesAnEmptySelection(t *testing.T) {
	i18n.SetLanguage("en")
	base := core.DefaultSettings()
	base.Scan.Roots = []string{"~/work"}
	var m tea.Model = newSetupModel(SetupInput{
		Settings: base,
		Roots:    []core.RootCandidate{{Path: "~/work", Repos: 2}, {Path: "~/misc", Repos: 0}},
	})
	m = uncheckEveryRoot(t, m)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	sm, ok := m.(setupModel)
	if !ok {
		t.Fatalf("Update returned a %T, want setupModel", m)
	}
	if sm.step != stepRoots {
		t.Fatalf("step = %d, want to stay on the roots step (%d)", sm.step, stepRoots)
	}
	if view := stripANSI(sm.View()); !strings.Contains(view, "at least one folder") {
		t.Errorf("the refusal must be on screen, got:\n%s", view)
	}
}

// Checking a folder clears the refusal and lets the wizard move on, so the
// guard costs a corrected selection and nothing more.
func TestSetupRootsStepAdvancesOnceAFolderIsChecked(t *testing.T) {
	var m tea.Model = newSetupModel(SetupInput{
		Settings: core.DefaultSettings(),
		Roots:    []core.RootCandidate{{Path: "~/work", Repos: 2}, {Path: "~/misc", Repos: 0}},
	})
	m = uncheckEveryRoot(t, m)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter}) // refused
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace}) // check ~/misc, under the cursor
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	sm := m.(setupModel)
	if sm.step != stepEditor {
		t.Fatalf("step = %d, want stepEditor (%d) once a folder is checked", sm.step, stepEditor)
	}
	if got := sm.settings().Scan.Roots; len(got) != 1 || got[0] != "~/misc" {
		t.Errorf("roots = %v, want the checked [~/misc]", got)
	}
	if view := stripANSI(sm.View()); strings.Contains(view, "at least one folder") {
		t.Errorf("the refusal must be gone once a folder is checked, got:\n%s", view)
	}
}

// With no candidate folder at all there is nothing to check, so the guard must
// not lock the user inside the step: the only way out would be esc.
func TestSetupRootsStepAdvancesWhenThereIsNothingToChoose(t *testing.T) {
	var m tea.Model = newSetupModel(SetupInput{Settings: core.DefaultSettings()}) // no roots
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if got := m.(setupModel).step; got != stepEditor {
		t.Fatalf("step = %d, want stepEditor (%d) when no folder was found", got, stepEditor)
	}
}

// RunSetup and RunConfig fail for two unrelated reasons: there is no terminal
// to draw on, which the caller can answer non-interactively, and the program
// itself failed after the screen was up, which means what the user did is lost.
// Reporting the second as the first is how cmd/hop came to overwrite a config
// with a preset the user never saw, and to exit 0 on a crash.
func TestRunnersReportAMissingTTYApartFromAnyOtherFailure(t *testing.T) {
	t.Run("no tty", func(t *testing.T) {
		stubTTY(t, func() (*os.File, error) { return nil, errors.New("open /dev/tty: device not configured") })

		if _, _, _, err := RunSetup(SetupInput{Settings: core.DefaultSettings()}); !errors.Is(err, ErrNoTTY) {
			t.Errorf("RunSetup err = %v, want it to wrap ErrNoTTY", err)
		}
		if _, _, err := RunConfig(core.DefaultSettings()); !errors.Is(err, ErrNoTTY) {
			t.Errorf("RunConfig err = %v, want it to wrap ErrNoTTY", err)
		}
	})

	// A tty that opens and a program that then fails: the error must not claim
	// the terminal was missing, because the caller answers that one by writing a
	// configuration of its own.
	t.Run("program failure", func(t *testing.T) {
		stubTTY(t, func() (*os.File, error) {
			f, err := os.Open(os.DevNull) // opens, but reading input returns EOF at once
			if err != nil {
				t.Fatal(err)
			}
			if err := f.Close(); err != nil { // closed: bubbletea cannot read from it
				t.Fatal(err)
			}
			return f, nil
		})

		_, _, _, err := RunSetup(SetupInput{Settings: core.DefaultSettings()})
		if err == nil {
			t.Fatal("RunSetup must fail when its program cannot run")
		}
		if errors.Is(err, ErrNoTTY) {
			t.Errorf("RunSetup err = %v, must not be reported as a missing tty", err)
		}
		_, _, err = RunConfig(core.DefaultSettings())
		if err == nil {
			t.Fatal("RunConfig must fail when its program cannot run")
		}
		if errors.Is(err, ErrNoTTY) {
			t.Errorf("RunConfig err = %v, must not be reported as a missing tty", err)
		}
	})
}

// stubTTY substitutes how the runners reach the controlling terminal, which no
// test can make unopenable on its own, and restores it afterwards.
func stubTTY(t *testing.T, open func() (*os.File, error)) {
	t.Helper()
	previous := openTTY
	openTTY = open
	t.Cleanup(func() { openTTY = previous })
}

// uncheckEveryRoot walks the roots step and toggles off every checked folder,
// leaving the cursor on the last one.
func uncheckEveryRoot(t *testing.T, m tea.Model) tea.Model {
	t.Helper()
	for i := range m.(setupModel).rootSel {
		if m.(setupModel).rootSel[i] {
			m, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
		}
		if i < len(m.(setupModel).rootSel)-1 {
			m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
		}
	}
	for i, ok := range m.(setupModel).rootSel {
		if ok {
			t.Fatalf("root %d is still checked: %v", i, m.(setupModel).rootSel)
		}
	}
	return m
}

func TestNewSetupModelSkipsShellStepWhenAlreadyWired(t *testing.T) {
	m := newSetupModel(SetupInput{
		Settings:     core.DefaultSettings(),
		Roots:        []core.RootCandidate{{Path: "~/code", Repos: 1}},
		ShellName:    "zsh",
		RcLabel:      "~/.zshrc",
		AlreadyWired: true,
	})
	if m.writeShell {
		t.Error("writeShell must be false when the integration is already present")
	}
	m.step = stepShell
	if m.itemCount() != 0 {
		t.Errorf("shell step should offer no choices when already wired, got %d", m.itemCount())
	}
}
