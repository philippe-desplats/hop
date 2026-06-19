package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/philippe-desplats/hop/internal/core"
)

func TestFirstSafePathSkipsControlChars(t *testing.T) {
	matches := []core.Match{
		{Project: core.Project{Path: "/p/with\nnewline"}}, // unsafe: would smuggle a protocol line
		{Project: core.Project{Path: "/p/clean"}},
	}
	got, ok := firstSafePath(matches)
	if !ok || got != "/p/clean" {
		t.Errorf("firstSafePath = (%q, %v), want (/p/clean, true)", got, ok)
	}

	if _, ok := firstSafePath([]core.Match{{Project: core.Project{Path: "/p/bad\x00"}}}); ok {
		t.Error("a sole control-char path must be refused")
	}
	if _, ok := firstSafePath(nil); ok {
		t.Error("no matches must return ok=false")
	}
}

func TestSafePathsFiltersControlChars(t *testing.T) {
	matches := []core.Match{
		{Project: core.Project{Path: "/p/a"}},
		{Project: core.Project{Path: "/p/b\r"}},
		{Project: core.Project{Path: "/p/c"}},
	}
	got := safePaths(matches)
	if len(got) != 2 || got[0] != "/p/a" || got[1] != "/p/c" {
		t.Errorf("safePaths = %v, want [/p/a /p/c]", got)
	}
}

// TestCmdQuerySubprocess re-executes the test binary so cmdQuery's os.Exit paths
// run for real, against a scanned index of two repos under a temp ~/Projects.
func TestCmdQuerySubprocess(t *testing.T) {
	if os.Getenv("HOP_QUERY_CHILD") == "1" {
		cmdQuery(strings.Fields(os.Getenv("HOP_QUERY_ARGS")))
		os.Exit(0) // exit before the test runner prints PASS to stdout
	}

	home := t.TempDir()
	for _, name := range []string{"alpha", "beta"} {
		if err := os.MkdirAll(filepath.Join(home, "Projects", name, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	run := func(t *testing.T, args string) (string, int) {
		t.Helper()
		cmd := exec.Command(os.Args[0], "-test.run=TestCmdQuerySubprocess")
		cmd.Env = []string{
			"HOP_QUERY_CHILD=1",
			"HOP_QUERY_ARGS=" + args,
			"HOME=" + home,
			"XDG_STATE_HOME=" + filepath.Join(t.TempDir(), "state"),
			"XDG_CONFIG_HOME=" + t.TempDir(),
			"PATH=" + t.TempDir(),       // empty: query never shells out
			"GOCOVERDIR=" + t.TempDir(), // silence the cover runtime warning on a coverage build
		}
		out, err := cmd.CombinedOutput()
		code := 0
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else if err != nil {
			t.Fatalf("subprocess failed to run: %v", err)
		}
		return string(out), code
	}

	t.Run("resolves to a plain path, no sentinel", func(t *testing.T) {
		out, code := run(t, "alpha")
		if code != 0 {
			t.Fatalf("exit = %d, want 0\n%s", code, out)
		}
		line := strings.TrimSpace(out)
		if strings.Contains(line, "__HOP_CD__") {
			t.Errorf("query must print a plain path, got sentinel: %q", line)
		}
		if !strings.HasSuffix(line, filepath.Join("Projects", "alpha")) {
			t.Errorf("path = %q, want it to end in Projects/alpha", line)
		}
	})

	t.Run("no match exits 1 with empty stdout", func(t *testing.T) {
		out, code := run(t, "zzznomatch")
		if code == 0 {
			t.Errorf("expected non-zero exit on no match, got 0")
		}
		if strings.TrimSpace(out) != "" {
			t.Errorf("no-match stdout must be empty, got %q", out)
		}
	})

	t.Run("--list prints every project", func(t *testing.T) {
		out, code := run(t, "--list")
		if code != 0 {
			t.Fatalf("exit = %d, want 0\n%s", code, out)
		}
		lines := strings.Split(strings.TrimSpace(out), "\n")
		if len(lines) != 2 {
			t.Errorf("--list printed %d lines, want 2: %q", len(lines), out)
		}
	})
}
