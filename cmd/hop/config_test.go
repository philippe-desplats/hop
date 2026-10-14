//go:build unix

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

// configToml holds a value per field the config editor exposes, each one
// distinctive enough that finding it in the output cannot be a coincidence.
const configToml = `[ui]
language = "es"
theme = "dark"

[shell]
command = "zp"

[hub]
action_access = "enter"

[actions]
editor = "my-editor-9000"
show_tmux = true

[scan]
roots = ["~/zzz-roots-a", "~/zzz-roots-b"]
max_depth = 42
ignore = ["zzz-ignored"]
`

// TestCmdConfigSubprocess runs cmdConfig in a child with no controlling
// terminal, so the config editor cannot start and the fallback listing runs.
// That listing used to print the config path and action_access only, hiding 8
// of the 9 editable fields from anyone checking their editor or their roots
// over SSH without a tty, or from a script.
func TestCmdConfigSubprocess(t *testing.T) {
	if os.Getenv("HOP_CONFIG_CHILD") == "1" {
		cmdConfig(nil)
		os.Exit(0) // exit before the test runner prints PASS to stdout
	}

	configDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(configDir, "hop"), 0o755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDir, "hop", "config.toml")
	if err := os.WriteFile(configPath, []byte(configToml), 0o600); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestCmdConfigSubprocess")
	// A new session detaches the child from the terminal running the tests, so
	// its /dev/tty cannot be opened: without this the editor would start for
	// real and the test would hang on whoever runs it from a terminal.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Env = []string{
		"HOP_CONFIG_CHILD=1",
		"HOME=" + t.TempDir(),
		"XDG_STATE_HOME=" + filepath.Join(t.TempDir(), "state"),
		"XDG_CONFIG_HOME=" + configDir,
		"PATH=" + t.TempDir(),       // empty: config never shells out
		"GOCOVERDIR=" + t.TempDir(), // silence the cover runtime warning on a coverage build
	}
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("exit = %v, want 0 (no tty is a legitimate use)\n%s", err, out)
	}

	stdout := string(out)
	if !strings.Contains(stdout, configPath) {
		t.Errorf("the fallback must print the config path %q, got:\n%s", configPath, stdout)
	}
	// One assertion per field the editor exposes, on the value rather than on the
	// layout: a line has to carry both the field and what config.toml holds for it.
	for _, f := range []struct{ field, value string }{
		{"command", "zp"},
		{"action_access", "enter"},
		{"editor", "my-editor-9000"},
		{"show_tmux", "true"},
		{"roots", "~/zzz-roots-a"},
		{"roots", "~/zzz-roots-b"},
		{"max_depth", "42"},
		{"ignore", "zzz-ignored"},
		{"language", "es"},
		{"theme", "dark"},
	} {
		if !lineHas(stdout, f.field, f.value) {
			t.Errorf("the fallback must print %s = %s, got:\n%s", f.field, f.value, stdout)
		}
	}
}

// lineHas reports whether a single output line mentions both the field and its
// value, so the assertion survives a change of layout but not a missing field.
func lineHas(out, field, value string) bool {
	for line := range strings.SplitSeq(out, "\n") {
		if strings.Contains(line, field) && strings.Contains(line, value) {
			return true
		}
	}
	return false
}
