package core

import (
	"os"
	"strings"
	"testing"
)

func TestSaveSettingsRoundTrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	s := DefaultSettings()
	s.Hub.ActionAccess = "enter"
	if err := SaveSettings(s); err != nil {
		t.Fatal(err)
	}
	if got := LoadSettings().Hub.ActionAccess; got != "enter" {
		t.Fatalf("round-trip = %q, want enter", got)
	}
	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "# hop configuration") {
		t.Error("saved config lost its comments")
	}
}

func TestSettingsLoad(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	if got := LoadSettings().Hub.ActionAccess; got != "tab" {
		t.Fatalf("missing config should default to tab, got %q", got)
	}

	created, err := EnsureConfig()
	if err != nil || !created {
		t.Fatalf("EnsureConfig: created=%v err=%v", created, err)
	}
	if _, err := os.Stat(ConfigPath()); err != nil {
		t.Fatalf("config file not written: %v", err)
	}

	write := func(s string) {
		if err := os.WriteFile(ConfigPath(), []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("[hub]\naction_access = \"shift\"\n")
	if got := LoadSettings().Hub.ActionAccess; got != "shift" {
		t.Errorf("ActionAccess = %q, want shift", got)
	}
	write("[hub]\naction_access = \"bogus\"\n")
	if got := LoadSettings().Hub.ActionAccess; got != "tab" {
		t.Errorf("invalid value should coerce to tab, got %q", got)
	}
}

func TestSettingsAIAndCustom(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	// Absent [ai] defaults to auto (backward compat with v1.x configs).
	if got := LoadSettings().AI.Tool; got != "auto" {
		t.Errorf("missing [ai] should default to auto, got %q", got)
	}

	if _, err := EnsureConfig(); err != nil {
		t.Fatal(err)
	}
	cfg := "[ai]\ntool = \"codex\"\n\n" +
		"[[actions.custom]]\n" +
		"key = \"y\"\n" +
		"label = \"open in Cursor\"\n" +
		"command = \"cursor {path}\"\n" +
		"needs_git = true\n" +
		"in_terminal = false\n"
	if err := os.WriteFile(ConfigPath(), []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	s := LoadSettings()
	if s.AI.Tool != "codex" {
		t.Errorf("AI.Tool = %q, want codex", s.AI.Tool)
	}
	if len(s.Actions.Custom) != 1 {
		t.Fatalf("custom actions = %d, want 1", len(s.Actions.Custom))
	}
	if c := s.Actions.Custom[0]; c.Key != "y" || c.Command != "cursor {path}" || !c.NeedsGit || c.InTerminal {
		t.Errorf("decoded custom action = %+v", c)
	}

	if err := os.WriteFile(ConfigPath(), []byte("[ai]\ntool = \"\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := LoadSettings().AI.Tool; got != "auto" {
		t.Errorf("empty tool should coerce to auto, got %q", got)
	}
}
