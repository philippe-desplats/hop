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
	if s.Actions.Editor != "cursor" {
		t.Errorf("editor = %q, want cursor", s.Actions.Editor)
	}
	if s.AI.Tool != "auto" {
		t.Errorf("ai = %q, want auto", s.AI.Tool)
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
