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

func TestSaveSettingsKeepsMultiplexerAndCustomActions(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if _, err := EnsureConfig(); err != nil {
		t.Fatal(err)
	}
	cfg := "[actions]\n" +
		"multiplexer = \"zellij\"\n\n" +
		"[[actions.custom]]\n" +
		"key = \"y\"\n" +
		"label = \"open in Cursor\"\n" +
		"command = \"cursor {path}\"\n" +
		"needs_git = true\n" +
		"in_terminal = true\n"
	if err := os.WriteFile(ConfigPath(), []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}

	// A save that changes nothing must not drop what the editor never showed.
	if err := SaveSettings(LoadSettings()); err != nil {
		t.Fatal(err)
	}

	s := LoadSettings()
	if s.Actions.Multiplexer != "zellij" {
		t.Errorf("multiplexer = %q, want zellij", s.Actions.Multiplexer)
	}
	if len(s.Actions.Custom) != 1 {
		t.Fatalf("custom actions = %d, want 1", len(s.Actions.Custom))
	}
	want := CustomAction{Key: "y", Label: "open in Cursor", Command: "cursor {path}", NeedsGit: true, InTerminal: true}
	if s.Actions.Custom[0] != want {
		t.Errorf("custom action = %+v, want %+v", s.Actions.Custom[0], want)
	}
}

func TestSaveSettingsEscapesCustomActionStrings(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	s := DefaultSettings()
	s.Actions.Custom = []CustomAction{{
		Key:     "y",
		Label:   `say "hi"`,
		Command: `run --flag="a b" C:\tools\x`,
	}}
	if err := SaveSettings(s); err != nil {
		t.Fatal(err)
	}
	got := LoadSettings().Actions.Custom
	if len(got) != 1 || got[0] != s.Actions.Custom[0] {
		t.Errorf("custom action round-trip = %+v, want %+v", got, s.Actions.Custom)
	}
}

// TestSaveSettingsEscapesControlCharacters pins the save as TOML rather than as
// Go source. Go's %q spells BEL, VT and ESC as \a, \v and \x1b, none of which
// TOML defines, so a decoder refuses the whole file: LoadSettings then falls
// back to the defaults and the next save overwrites what the user wrote.
func TestSaveSettingsEscapesControlCharacters(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	s := DefaultSettings()
	s.Actions.Custom = []CustomAction{{
		Key:     "y",
		Label:   "bell \x07 vt \x0b esc \x1b",
		Command: "run \x1b[1m{path}\x1b[0m",
	}}
	if err := SaveSettings(s); err != nil {
		t.Fatal(err)
	}
	got := LoadSettings().Actions.Custom
	if len(got) != 1 || got[0] != s.Actions.Custom[0] {
		t.Errorf("custom action round-trip = %+v, want %+v", got, s.Actions.Custom)
	}
}

// TestSaveSettingsRefusesInvalidUTF8 covers the input TOML cannot represent at
// all. Writing it would produce a file no decoder can read, so the save fails
// and whatever the user already had on disk stays there.
func TestSaveSettingsRefusesInvalidUTF8(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := SaveSettings(DefaultSettings()); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(ConfigPath())
	if err != nil {
		t.Fatal(err)
	}

	s := DefaultSettings()
	s.Actions.Custom = []CustomAction{{Key: "y", Label: "bad \xff\xfe", Command: "run {path}"}}
	if err := SaveSettings(s); err == nil {
		t.Error("SaveSettings accepted a label that is not valid UTF-8")
	}

	after, err := os.ReadFile(ConfigPath())
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Error("a refused save rewrote the existing config file")
	}
}

func TestRenderConfigCommentsOutUnsetActionExamples(t *testing.T) {
	def, err := renderConfig(DefaultSettings())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"# multiplexer = \"auto\"", "# [[actions.custom]]"} {
		if !strings.Contains(def, want) {
			t.Errorf("default config lost the example line %q", want)
		}
	}

	s := DefaultSettings()
	s.Actions.Multiplexer = "tmux"
	s.Actions.Custom = []CustomAction{{Key: "y", Command: "cursor {path}"}}
	set, err := renderConfig(s)
	if err != nil {
		t.Fatal(err)
	}
	for _, unwanted := range []string{"# multiplexer = ", "# [[actions.custom]]", "# key = "} {
		if strings.Contains(set, unwanted) {
			t.Errorf("config with values kept the example line %q", unwanted)
		}
	}
	for _, want := range []string{"multiplexer = \"tmux\"", "[[actions.custom]]\nkey = \"y\""} {
		if !strings.Contains(set, want) {
			t.Errorf("rendered config missing %q", want)
		}
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
