package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestOperatorMessagesAreLocalized runs the usage errors of two commands under
// [ui] language = "fr" and asserts their English literals are gone, so no single
// command can answer half in the user's language and half in English.
//
// It asserts the absence of the English wording rather than the presence of a
// particular French sentence: the point is that the string participates in the
// catalog at all, and that stays true when a translation is reworded.
//
// The child goes through main rather than calling cmdInit or cmdPin directly,
// because main is where the active language is selected from the settings; a
// direct call would leave i18n on its English default and observe nothing.
func TestOperatorMessagesAreLocalized(t *testing.T) {
	if argv := os.Getenv("HOP_I18N_ARGV"); argv != "" {
		os.Args = append([]string{"hop"}, decodeArgv(argv)...)
		main()
		os.Exit(0) // exit before the test runner prints PASS to stdout
	}

	configHome := t.TempDir()
	if err := os.MkdirAll(filepath.Join(configHome, "hop"), 0o750); err != nil {
		t.Fatal(err)
	}
	toml := "[ui]\nlanguage = \"fr\"\n"
	if err := os.WriteFile(filepath.Join(configHome, "hop", "config.toml"), []byte(toml), 0o600); err != nil {
		t.Fatal(err)
	}

	run := func(t *testing.T, argv ...string) (stderr string, code int) {
		t.Helper()
		cmd := exec.Command(os.Args[0], "-test.run=TestOperatorMessagesAreLocalized")
		cmd.Env = []string{
			"HOP_I18N_ARGV=" + encodeArgv(t, argv),
			"HOME=" + t.TempDir(),
			"XDG_CONFIG_HOME=" + configHome,
			"XDG_STATE_HOME=" + filepath.Join(t.TempDir(), "state"),
			"PATH=" + t.TempDir(),       // empty: neither usage error shells out
			"GOCOVERDIR=" + t.TempDir(), // silence the cover runtime warning on a coverage build
		}
		var outBuf, errBuf bytes.Buffer
		cmd.Stdout, cmd.Stderr = &outBuf, &errBuf
		err := cmd.Run()
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else if err != nil {
			t.Fatalf("subprocess failed to run: %v\n%s", err, errBuf.String())
		}
		return errBuf.String(), code
	}

	t.Run("hop init --cmd without a name", func(t *testing.T) {
		stderr, code := run(t, "init", "--cmd")
		if code != 2 {
			t.Errorf("exit = %d, want 2 (usage error)\nstderr: %s", code, stderr)
		}
		if strings.Contains(stderr, "requires a name") {
			t.Errorf("a French user must not be served the English wording, got:\n%s", stderr)
		}
	})

	t.Run("hop query --list --limit with a bad value", func(t *testing.T) {
		stderr, code := run(t, "query", "--list", "--limit", "0")
		if code != 2 {
			t.Errorf("exit = %d, want 2 (usage error)\nstderr: %s", code, stderr)
		}
		if strings.Contains(stderr, "invalid --limit value") {
			t.Errorf("a French user must not be served the English wording, got:\n%s", stderr)
		}
	})

	t.Run("hop clean with an argument it does not know", func(t *testing.T) {
		stderr, code := run(t, "clean", "--bogus")
		if code != 2 {
			t.Errorf("exit = %d, want 2 (usage error)\nstderr: %s", code, stderr)
		}
		if strings.Contains(stderr, "unknown argument") {
			t.Errorf("a French user must not be served the English wording, got:\n%s", stderr)
		}
	})

	t.Run("hop pin without a keyword", func(t *testing.T) {
		stderr, code := run(t, "pin")
		if strings.Contains(stderr, "unpin") {
			t.Errorf("the message must name only the command invoked, got:\n%s", stderr)
		}
		if code != 2 {
			t.Errorf("exit = %d, want 2 (usage error)\nstderr: %s", code, stderr)
		}
		if strings.Contains(stderr, "usage:") {
			t.Errorf("a French user must not be served the English wording, got:\n%s", stderr)
		}
	})
}
