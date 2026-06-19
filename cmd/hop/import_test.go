package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestCmdImportSubprocess re-executes the test binary as a child so the os.Exit
// paths in cmdImport are exercised for real. The parent sets up an isolated HOME
// / XDG state and a stub `zoxide` on PATH; the child runs cmdImport and exits.
func TestCmdImportSubprocess(t *testing.T) {
	if os.Getenv("HOP_IMPORT_CHILD") == "1" {
		cmdImport(strings.Fields(os.Getenv("HOP_IMPORT_ARGS")))
		os.Exit(0) // exit before the test runner prints PASS to stdout
	}

	repo := filepath.Join(t.TempDir(), "myrepo")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	plain := filepath.Join(t.TempDir(), "plainfolder")
	if err := os.MkdirAll(plain, 0o755); err != nil {
		t.Fatal(err)
	}

	run := func(t *testing.T, args string, withStub bool) (string, int, string) {
		t.Helper()
		home := t.TempDir()
		stateDir := filepath.Join(t.TempDir(), "state")
		configDir := t.TempDir()
		binDir := t.TempDir()
		pathVal := binDir
		if withStub {
			stub := "#!/bin/sh\nprintf '%s\\n' '80 " + repo + "' '30 " + plain + "'\n"
			if err := os.WriteFile(filepath.Join(binDir, "zoxide"), []byte(stub), 0o755); err != nil {
				t.Fatal(err)
			}
		}
		cmd := exec.Command(os.Args[0], "-test.run=TestCmdImportSubprocess")
		cmd.Env = []string{
			"HOP_IMPORT_CHILD=1",
			"HOP_IMPORT_ARGS=" + args,
			"HOME=" + home,
			"XDG_STATE_HOME=" + stateDir,
			"XDG_CONFIG_HOME=" + configDir,
			"PATH=" + pathVal,
			"GOCOVERDIR=" + t.TempDir(), // silence the cover runtime warning on a coverage build
		}
		out, err := cmd.CombinedOutput()
		code := 0
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else if err != nil {
			t.Fatalf("subprocess failed to run: %v", err)
		}
		return string(out), code, stateDir
	}

	t.Run("no zoxide on PATH errors", func(t *testing.T) {
		out, code, _ := run(t, "--from zoxide", false)
		if code == 0 {
			t.Errorf("expected non-zero exit when zoxide is absent, got 0\n%s", out)
		}
		if !strings.Contains(out, "not found") {
			t.Errorf("expected an install hint, got:\n%s", out)
		}
	})

	t.Run("real run tracks the repo and seeds frecency", func(t *testing.T) {
		out, code, stateDir := run(t, "--from zoxide", true)
		if code != 0 {
			t.Errorf("exit = %d, want 0\n%s", code, out)
		}
		if !strings.Contains(out, "imported") {
			t.Errorf("expected a count summary, got:\n%s", out)
		}
		extras, err := os.ReadFile(filepath.Join(stateDir, "hop", "extras.json"))
		if err != nil {
			t.Fatalf("extras.json not written: %v", err)
		}
		if !strings.Contains(string(extras), "myrepo") {
			t.Errorf("repo should be tracked, extras = %s", extras)
		}
		if strings.Contains(string(extras), "plainfolder") {
			t.Errorf("a non-repo must not be tracked, extras = %s", extras)
		}
		frec, err := os.ReadFile(filepath.Join(stateDir, "hop", "frecency.json"))
		if err != nil || !strings.Contains(string(frec), "myrepo") {
			t.Errorf("repo should be seeded in frecency, got err=%v frec=%s", err, frec)
		}
	})

	t.Run("dry-run writes nothing", func(t *testing.T) {
		out, code, stateDir := run(t, "--from zoxide --dry-run", true)
		if code != 0 {
			t.Errorf("exit = %d, want 0\n%s", code, out)
		}
		if !strings.Contains(out, "would import") {
			t.Errorf("expected a dry-run summary, got:\n%s", out)
		}
		if _, err := os.Stat(filepath.Join(stateDir, "hop", "extras.json")); err == nil {
			t.Error("dry-run must not write extras.json")
		}
		if _, err := os.Stat(filepath.Join(stateDir, "hop", "frecency.json")); err == nil {
			t.Error("dry-run must not write frecency.json")
		}
	})
}
