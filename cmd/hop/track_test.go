package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestCmdTrackSubprocess re-executes the test binary so the exit code of
// `hop track` / `hop untrack` is observed for real. A rejected argument used to
// leave the exit code at 0, so `hop track ~/Dowloads && echo ok` printed ok on a
// typo and a provisioning script reported success having tracked nothing.
func TestCmdTrackSubprocess(t *testing.T) {
	if os.Getenv("HOP_TRACK_CHILD") == "1" {
		argv := decodeArgv(os.Getenv("HOP_TRACK_ARGV"))
		if os.Getenv("HOP_TRACK_UNTRACK") == "1" {
			cmdUntrack(argv)
		} else {
			cmdTrack(argv)
		}
		os.Exit(0) // exit before the test runner prints PASS to stdout
	}

	run := func(t *testing.T, untrack bool, argv ...string) (string, int, string) {
		t.Helper()
		home := t.TempDir()
		stateDir := filepath.Join(t.TempDir(), "state")
		untrackFlag := "0"
		if untrack {
			untrackFlag = "1"
		}
		cmd := exec.Command(os.Args[0], "-test.run=TestCmdTrackSubprocess")
		cmd.Dir = home
		cmd.Env = []string{
			"HOP_TRACK_CHILD=1",
			"HOP_TRACK_UNTRACK=" + untrackFlag,
			"HOP_TRACK_ARGV=" + encodeArgv(t, argv),
			"HOME=" + home,
			"XDG_STATE_HOME=" + stateDir,
			"XDG_CONFIG_HOME=" + t.TempDir(),
			"PATH=" + t.TempDir(),       // empty: track never shells out
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

	extras := func(t *testing.T, stateDir string) string {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(stateDir, "hop", "extras.json"))
		if err != nil {
			return ""
		}
		return string(data)
	}

	t.Run("a path that is not a directory fails", func(t *testing.T) {
		out, code, stateDir := run(t, false, "/definitely/not/here")
		if code != 1 {
			t.Errorf("exit = %d, want 1 (the argument was rejected)\n%s", code, out)
		}
		if !strings.Contains(out, "/definitely/not/here") {
			t.Errorf("stderr must name the rejected path, got:\n%s", out)
		}
		if got := extras(t, stateDir); got != "" {
			t.Errorf("nothing should be tracked, extras = %s", got)
		}
	})

	t.Run("a real directory is tracked and exits 0", func(t *testing.T) {
		dir := t.TempDir()
		out, code, stateDir := run(t, false, dir)
		if code != 0 {
			t.Fatalf("exit = %d, want 0\n%s", code, out)
		}
		if got := extras(t, stateDir); !strings.Contains(got, filepath.Base(dir)) {
			t.Errorf("extras should hold %q, got %s", dir, got)
		}
	})

	// The rule: exit 1 as soon as one argument failed, like git. The arguments
	// that did succeed are still applied, so the failure is reported without
	// losing the work.
	t.Run("one bad argument among good ones still fails", func(t *testing.T) {
		dir := t.TempDir()
		out, code, stateDir := run(t, false, dir, "/definitely/not/here")
		if code != 1 {
			t.Errorf("exit = %d, want 1 (one argument was rejected)\n%s", code, out)
		}
		if got := extras(t, stateDir); !strings.Contains(got, filepath.Base(dir)) {
			t.Errorf("the good argument must still be tracked, extras = %s", got)
		}
	})

	t.Run("a control-char path is refused before it is stored", func(t *testing.T) {
		out, code, stateDir := run(t, false, "/tmp/evil\nrm -rf")
		if code != 1 {
			t.Errorf("exit = %d, want 1 (unsafe path)\n%s", code, out)
		}
		if got := extras(t, stateDir); got != "" {
			t.Errorf("an unsafe path must never reach extras.json, got %s", got)
		}
	})

	t.Run("untrack reports a rejected argument too", func(t *testing.T) {
		out, code, _ := run(t, true, "/tmp/evil\nrm -rf")
		if code != 1 {
			t.Errorf("exit = %d, want 1 (unsafe path)\n%s", code, out)
		}
	})
}
