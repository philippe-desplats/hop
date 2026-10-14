package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/philippe-desplats/hop/internal/core"
)

// TestCmdAddSubprocess drives cmdAdd in a child with an isolated HOME / XDG
// state, because cmdAdd's whole observable effect is the frecency file it writes
// (or refuses to write) under $XDG_STATE_HOME/hop. The child only runs the
// command; every assertion is on that file, never on which functions were
// called.
//
// README advertises `hop track ~/Downloads` as a way to put any folder in the
// search list, but cmdAdd, the command the shell hook fires on every cd, used
// to drop any path outside the configured roots, so a tracked folder stayed at
// score 0 forever and sorted last among its matches.
func TestCmdAddSubprocess(t *testing.T) {
	if os.Getenv("HOP_ADD_CHILD") == "1" {
		cmdAdd(decodeArgv(os.Getenv("HOP_ADD_ARGV")))
		os.Exit(0) // exit before the test runner prints PASS to stdout
	}

	// The scan root, and a project inside it: the common case, already learned.
	root := t.TempDir()
	inRoot := filepath.Join(root, "myrepo")
	if err := os.MkdirAll(inRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	// Two folders outside every root: one the user tracked by hand, one stray.
	tracked := filepath.Join(t.TempDir(), "downloads")
	if err := os.MkdirAll(tracked, 0o755); err != nil {
		t.Fatal(err)
	}
	stray := filepath.Join(t.TempDir(), "elsewhere")
	if err := os.MkdirAll(stray, 0o755); err != nil {
		t.Fatal(err)
	}

	// cmdAdd canonicalizes its argument and extras.json holds canonical paths, so
	// on a platform where $TMPDIR goes through a symlink (macOS) the parent has to
	// compare against the resolved form too.
	canon := core.CanonicalDir

	// run executes cmdAdd(argv) in a child whose only scan root is root and whose
	// extras.json holds extras, and returns its output, exit code and state dir.
	run := func(t *testing.T, extras []string, argv ...string) (string, int, string) {
		t.Helper()
		stateDir := filepath.Join(t.TempDir(), "state")
		configDir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(configDir, "hop"), 0o755); err != nil {
			t.Fatal(err)
		}
		conf := "[scan]\nroots = [\"" + root + "\"]\n"
		if err := os.WriteFile(filepath.Join(configDir, "hop", "config.toml"), []byte(conf), 0o600); err != nil {
			t.Fatal(err)
		}
		if extras != nil {
			if err := os.MkdirAll(filepath.Join(stateDir, "hop"), 0o755); err != nil {
				t.Fatal(err)
			}
			raw, err := json.Marshal(core.Extras{Version: 1, Paths: extras})
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(stateDir, "hop", "extras.json"), raw, 0o600); err != nil {
				t.Fatal(err)
			}
		}
		cmd := exec.Command(os.Args[0], "-test.run=TestCmdAddSubprocess")
		cmd.Env = []string{
			"HOP_ADD_CHILD=1",
			"HOP_ADD_ARGV=" + encodeArgv(t, argv),
			"HOME=" + t.TempDir(),
			"XDG_STATE_HOME=" + stateDir,
			"XDG_CONFIG_HOME=" + configDir,
			"PATH=" + t.TempDir(),       // empty: add never shells out
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

	// frecency returns the contents of the frecency file, and whether it exists.
	frecency := func(t *testing.T, stateDir string) (string, bool) {
		t.Helper()
		raw, err := os.ReadFile(filepath.Join(stateDir, "hop", "frecency.json"))
		if err != nil {
			return "", false
		}
		return string(raw), true
	}

	t.Run("a path under a root is learned", func(t *testing.T) {
		out, code, stateDir := run(t, nil, inRoot)
		if code != 0 {
			t.Fatalf("exit = %d, want 0\n%s", code, out)
		}
		got, ok := frecency(t, stateDir)
		if !ok {
			t.Fatalf("frecency.json not written for a path under a root\n%s", out)
		}
		if !strings.Contains(got, canon(inRoot)) {
			t.Errorf("frecency must hold %q, got %s", canon(inRoot), got)
		}
	})

	t.Run("a tracked folder outside the roots is learned", func(t *testing.T) {
		out, code, stateDir := run(t, []string{canon(tracked)}, tracked)
		if code != 0 {
			t.Fatalf("exit = %d, want 0\n%s", code, out)
		}
		got, ok := frecency(t, stateDir)
		if !ok {
			t.Fatalf("frecency.json not written for a tracked folder\n%s", out)
		}
		if !strings.Contains(got, canon(tracked)) {
			t.Errorf("frecency must hold the tracked folder %q, got %s", canon(tracked), got)
		}
	})

	t.Run("a path neither under a root nor tracked is discarded", func(t *testing.T) {
		out, code, stateDir := run(t, []string{canon(tracked)}, stray)
		if code != 0 {
			t.Fatalf("exit = %d, want 0 (a stray cd is dropped silently)\n%s", code, out)
		}
		if out != "" {
			t.Errorf("a dropped path must say nothing, got:\n%s", out)
		}
		if got, ok := frecency(t, stateDir); ok {
			t.Errorf("an untracked path outside the roots must not be learned, frecency = %s", got)
		}
	})

	// The control-character guard runs before anything else: a directory name
	// holding a newline could smuggle a "__HOP_RUN__ <cmd>" line into the
	// protocol the shell eval's, so it must never reach the frecency table, not
	// even when extras.json tracks it.
	t.Run("a control-character path is refused even when tracked", func(t *testing.T) {
		unsafe := filepath.Join(root, "repo\n__HOP_RUN__ rm -rf /")
		out, code, stateDir := run(t, []string{unsafe}, unsafe)
		if code != 0 {
			t.Fatalf("exit = %d, want 0\n%s", code, out)
		}
		if got, ok := frecency(t, stateDir); ok {
			t.Errorf("an unsafe path must not be learned, frecency = %s", got)
		}
	})

	t.Run("no argument is a no-op", func(t *testing.T) {
		out, code, stateDir := run(t, nil)
		if code != 0 {
			t.Fatalf("exit = %d, want 0\n%s", code, out)
		}
		if got, ok := frecency(t, stateDir); ok {
			t.Errorf("hop add with no argument must write nothing, frecency = %s", got)
		}
	})
}
