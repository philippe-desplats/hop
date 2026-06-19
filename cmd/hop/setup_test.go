package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/philippe-desplats/hop/internal/action"
	"github.com/philippe-desplats/hop/internal/core"
)

func TestPresetSettings(t *testing.T) {
	roots := []core.RootCandidate{{Path: "~/code", Repos: 5}, {Path: "~/work", Repos: 0}, {Path: "~/dev", Repos: 2}}
	editors := []action.Editor{{Name: "VS Code", Bin: "code"}}
	got := presetSettings(core.DefaultSettings(), roots, editors, []string{"claude"})

	if len(got.Scan.Roots) != 2 || got.Scan.Roots[0] != "~/code" || got.Scan.Roots[1] != "~/dev" {
		t.Errorf("roots = %v, want repo-bearing only [~/code ~/dev]", got.Scan.Roots)
	}
	if got.Actions.Editor != "code" {
		t.Errorf("editor = %q, want code", got.Actions.Editor)
	}
	if got.AI.Tool != "auto" {
		t.Errorf("ai = %q, want auto", got.AI.Tool)
	}
}

func TestPresetSettingsFallsBackToFirstRoot(t *testing.T) {
	roots := []core.RootCandidate{{Path: "~/a", Repos: 0}, {Path: "~/b", Repos: 0}}
	got := presetSettings(core.DefaultSettings(), roots, nil, nil)
	if len(got.Scan.Roots) != 1 || got.Scan.Roots[0] != "~/a" {
		t.Errorf("roots = %v, want fallback [~/a]", got.Scan.Roots)
	}
}

func TestPresetSettingsNoRootsKeepsDefault(t *testing.T) {
	base := core.DefaultSettings()
	got := presetSettings(base, nil, nil, nil)
	if len(got.Scan.Roots) != len(base.Scan.Roots) || got.Scan.Roots[0] != base.Scan.Roots[0] {
		t.Errorf("roots = %v, want unchanged default %v", got.Scan.Roots, base.Scan.Roots)
	}
}

func TestWireShellAppendsAndDetects(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	rc := "~/.zshrc"

	if shellAlreadyWired(rc) {
		t.Fatal("must not report wired before the rc exists")
	}
	if err := wireShell(rc, `eval "$(hop init zsh)"`); err != nil {
		t.Fatal(err)
	}
	if !shellAlreadyWired(rc) {
		t.Error("must detect the integration after writing")
	}
	data, err := os.ReadFile(filepath.Join(home, ".zshrc"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `eval "$(hop init zsh)"`) {
		t.Errorf("rc is missing the eval line: %q", data)
	}
}

func TestWireShellCreatesFishConfigDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	rc := "~/.config/fish/config.fish" // parent dir does not exist yet

	if err := wireShell(rc, "hop init fish | source"); err != nil {
		t.Fatal(err)
	}
	if !shellAlreadyWired(rc) {
		t.Error("fish config should be wired after writing")
	}
}

func TestDetectShell(t *testing.T) {
	cases := map[string]struct{ shell, rc string }{
		"/bin/zsh":            {"zsh", "~/.zshrc"},
		"/usr/bin/bash":       {"bash", "~/.bashrc"},
		"/usr/local/bin/fish": {"fish", "~/.config/fish/config.fish"},
		"":                    {"zsh", "~/.zshrc"}, // unknown falls back to zsh
	}
	for shellPath, want := range cases {
		t.Setenv("SHELL", shellPath)
		shell, rc, line := detectShell()
		if shell != want.shell || rc != want.rc {
			t.Errorf("detectShell(%q) = %q,%q, want %q,%q", shellPath, shell, rc, want.shell, want.rc)
		}
		if line == "" {
			t.Errorf("detectShell(%q) returned an empty shell line", shellPath)
		}
	}
}
