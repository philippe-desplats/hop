package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestMainUnknownCommand re-executes the test binary so main's os.Exit path runs
// for real. main is an ordinary function of this package, so the child can point
// os.Args at the invocation under test and call it directly.
//
// The config pins [ui] language = "en", so the assertions below quote the
// English catalog wording rather than whatever locale the test host runs under.
func TestMainUnknownCommand(t *testing.T) {
	if os.Getenv("HOP_MAIN_CHILD") == "1" {
		os.Args = []string{"hop", "nope"}
		main()
		os.Exit(0) // exit before the test runner prints PASS to stdout
	}

	configHome := t.TempDir()
	if err := os.MkdirAll(filepath.Join(configHome, "hop"), 0o750); err != nil {
		t.Fatal(err)
	}
	toml := "[ui]\nlanguage = \"en\"\n"
	if err := os.WriteFile(filepath.Join(configHome, "hop", "config.toml"), []byte(toml), 0o600); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestMainUnknownCommand")
	cmd.Env = []string{
		"HOP_MAIN_CHILD=1",
		"HOME=" + t.TempDir(),
		"XDG_CONFIG_HOME=" + configHome,
		"XDG_STATE_HOME=" + filepath.Join(t.TempDir(), "state"),
		"PATH=" + t.TempDir(),       // empty: an unknown command never shells out
		"GOCOVERDIR=" + t.TempDir(), // silence the cover runtime warning on a coverage build
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	code := 0
	if ee, ok := err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatalf("subprocess failed to run: %v\n%s", err, stderr.String())
	}

	if code != 2 {
		t.Errorf("exit = %d, want 2 (usage error)\nstderr: %s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "unknown command") {
		t.Errorf("stderr must carry the English wording, got:\n%s", stderr.String())
	}
	if strings.Contains(stderr.String(), "commande inconnue") {
		t.Errorf("an English user must not be served the French wording, got:\n%s", stderr.String())
	}
	if !strings.Contains(stderr.String(), `"nope"`) {
		t.Errorf("stderr must quote the rejected argument, got:\n%s", stderr.String())
	}
}
