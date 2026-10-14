package tui

import (
	"testing"

	"github.com/philippe-desplats/hop/internal/action"
	"github.com/philippe-desplats/hop/internal/core"
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
