package main

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/philippe-desplats/hop/internal/action"
	"github.com/philippe-desplats/hop/internal/core"
)

// captureStdout runs fn with os.Stdout redirected to a pipe and returns what it
// printed. emitOutcome writes the shell protocol to stdout via fmt.Printf.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() { os.Stdout = old }()
	fn()
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func TestEmitOutcome(t *testing.T) {
	cases := []struct {
		name string
		out  action.Outcome
		want string
	}{
		{"cd only", action.Outcome{Cd: "/p/x"}, "__HOP_CD__ /p/x\n"},
		{"cd and run", action.Outcome{Cd: "/p/x", Run: "claude"}, "__HOP_CD__ /p/x\n__HOP_RUN__ claude\n"},
		{"run only", action.Outcome{Run: "git status"}, "__HOP_RUN__ git status\n"},
		{"empty emits nothing", action.Outcome{}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := captureStdout(t, func() { emitOutcome(tc.out) })
			if got != tc.want {
				t.Errorf("emitOutcome(%+v) = %q, want %q", tc.out, got, tc.want)
			}
		})
	}
}

func TestActionOutcome(t *testing.T) {
	p := core.Project{Name: "demo", Path: "/p/demo"}
	opts := action.Options{
		Editor: "zed",
		AI:     action.Assistant{Name: "claude", Run: []string{"claude"}, Resume: []string{"claude", "--resume"}},
		HasAI:  true,
	}
	cases := []struct {
		name    string
		key     string
		wantCd  string
		wantRun string
	}{
		{"enter is a bare cd", "enter", "/p/demo", ""},
		{"claude runs the assistant", "c", "/p/demo", "claude"},
		{"resume passes --resume", "r", "/p/demo", "claude --resume"},
		{"unknown key falls back to cd", "qzx", "/p/demo", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := actionOutcome(tc.key, p, opts)
			if got.Cd != tc.wantCd || got.Run != tc.wantRun {
				t.Errorf("actionOutcome(%q) = {Cd:%q Run:%q}, want {Cd:%q Run:%q}",
					tc.key, got.Cd, got.Run, tc.wantCd, tc.wantRun)
			}
		})
	}
}

// TestCmdNavSubprocess re-executes the test binary so cmdNav's os.Exit paths run
// for real, against a scanned index under a temp ~/Projects. One project is
// named "my-hop" on purpose: the resolver matches keywords as a substring of the
// lowercased full path, so an unparsed "-h" would match it and cd the user into
// it. A leading "-" must be a flag namespace, never a keyword.
func TestCmdNavSubprocess(t *testing.T) {
	if os.Getenv("HOP_NAV_CHILD") == "1" {
		if seed := os.Getenv("HOP_NAV_SEED"); seed != "" {
			cmdAdd([]string{seed}) // record a visit so `nav -` has somewhere to go back to
			os.Exit(0)
		}
		cmdNav(decodeArgv(os.Getenv("HOP_NAV_ARGV")))
		os.Exit(0) // exit before the test runner prints PASS to stdout
	}

	home := t.TempDir()
	for _, name := range []string{"alpha", "my-hop"} {
		if err := os.MkdirAll(filepath.Join(home, "Projects", name, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	// run executes one child. stateDir is shared between calls that need to see
	// each other's frecency; seed, when set, records a visit instead of navigating.
	run := func(t *testing.T, stateDir, seed string, argv ...string) (string, string, int) {
		t.Helper()
		cmd := exec.Command(os.Args[0], "-test.run=TestCmdNavSubprocess")
		cmd.Dir = home // cwd outside every project, so none is excluded from the jump list
		cmd.Env = []string{
			"HOP_NAV_CHILD=1",
			"HOP_NAV_SEED=" + seed,
			"HOP_NAV_ARGV=" + encodeArgv(t, argv),
			"HOME=" + home,
			"XDG_STATE_HOME=" + stateDir,
			"XDG_CONFIG_HOME=" + t.TempDir(),
			"PATH=" + t.TempDir(),       // empty: nav never shells out here
			"GOCOVERDIR=" + t.TempDir(), // silence the cover runtime warning on a coverage build
		}
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		code := 0
		if err := cmd.Run(); err != nil {
			ee, ok := err.(*exec.ExitError)
			if !ok {
				t.Fatalf("subprocess failed to run: %v", err)
			}
			code = ee.ExitCode()
		}
		return stdout.String(), stderr.String(), code
	}

	t.Run("an unparsed leading dash is a usage error, not a keyword", func(t *testing.T) {
		for _, arg := range []string{"-h", "--help", "-0", "-x", "--"} {
			t.Run(arg, func(t *testing.T) {
				stdout, stderr, code := run(t, filepath.Join(t.TempDir(), "state"), "", arg)
				if code != 2 {
					t.Errorf("exit = %d, want 2 (usage error)\nstdout: %s\nstderr: %s", code, stdout, stderr)
				}
				if strings.Contains(stdout, "__HOP_CD__") {
					t.Errorf("%q must never emit a cd sentinel, got stdout: %q", arg, stdout)
				}
				if !strings.Contains(stderr, "p -2") {
					t.Errorf("stderr must show the jump-back usage, got: %s", stderr)
				}
			})
		}
	})

	t.Run("a keyword containing a hyphen still resolves", func(t *testing.T) {
		stdout, stderr, code := run(t, filepath.Join(t.TempDir(), "state"), "", "my-hop")
		if code != 0 {
			t.Fatalf("exit = %d, want 0\nstdout: %s\nstderr: %s", code, stdout, stderr)
		}
		if !strings.Contains(stdout, "__HOP_CD__") || !strings.Contains(stdout, "my-hop") {
			t.Errorf("stdout = %q, want a cd sentinel onto my-hop", stdout)
		}
	})

	t.Run("jump-back still jumps", func(t *testing.T) {
		stateDir := filepath.Join(t.TempDir(), "state")
		visited := filepath.Join(home, "Projects", "alpha")
		if _, stderr, code := run(t, stateDir, visited); code != 0 {
			t.Fatalf("seeding the jump list exited %d: %s", code, stderr)
		}
		stdout, stderr, code := run(t, stateDir, "", "-")
		if code != 0 {
			t.Fatalf("exit = %d, want 0\nstdout: %s\nstderr: %s", code, stdout, stderr)
		}
		if !strings.Contains(stdout, "__HOP_CD__") || !strings.Contains(stdout, "alpha") {
			t.Errorf("stdout = %q, want a cd sentinel back onto alpha", stdout)
		}
	})

	t.Run("jump-back with an empty jump list is not a usage error", func(t *testing.T) {
		stdout, stderr, code := run(t, filepath.Join(t.TempDir(), "state"), "", "-2")
		if code != 1 {
			t.Errorf("exit = %d, want 1 (nothing to go back to)\nstdout: %s\nstderr: %s", code, stdout, stderr)
		}
	})
}
