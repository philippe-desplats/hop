package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestPruneFailureSubprocess drives `hop scan` and `hop clean` in a child whose
// frecency prune cannot possibly succeed, and asserts each command reports it
// the way its own contract requires: scan warns and still exits 0 (the scan
// itself worked, and that is the command's job), clean exits non-zero (pruning
// is all it does).
//
// `hop scan` used to run the same three prunes as `hop clean` with their errors
// discarded — `removed, _ := core.PruneFrecency()` — so a frecency table that
// could no longer be rewritten printed the usual "N projects indexed" summary
// and exited 0. The user got no signal at all while dead entries kept ranking.
//
// Making the prune fail needs care. The ticket's suggestion — chmod 0400 on
// $XDG_STATE_HOME/hop — is not usable for scan: SaveIndex writes index.json
// into that same directory first and calls fatal, so the run never reaches the
// prune. What this test does instead is leave the directory writable and put a
// *directory* where the prune's lock file belongs: store.acquire opens it with
// O_RDWR, which fails with EISDIR for any uid, root included. The data file
// itself is untouched and index.json is written normally, so the only failing
// step is the one under test.
func TestPruneFailureSubprocess(t *testing.T) {
	if os.Getenv("HOP_PRUNE_CHILD") == "1" {
		if os.Getenv("HOP_PRUNE_CMD") == "clean" {
			cmdClean(nil)
		} else {
			cmdScan(nil)
		}
		os.Exit(0) // exit before the test runner prints PASS to stdout
	}

	// run executes the named command in a child whose frecency prune is broken,
	// and returns its stderr and exit code.
	run := func(t *testing.T, command string) (string, int) {
		t.Helper()
		stateDir := filepath.Join(t.TempDir(), "state")
		hopState := filepath.Join(stateDir, "hop")
		if err := os.MkdirAll(hopState, 0o700); err != nil {
			t.Fatal(err)
		}
		// A dead entry, so a working prune would have had something to remove.
		frecency := `{"version":1,"entries":{"/nonexistent/gone":{"rank":10,"last_access":1}}}`
		if err := os.WriteFile(filepath.Join(hopState, "frecency.json"), []byte(frecency), 0o600); err != nil {
			t.Fatal(err)
		}
		lock := filepath.Join(hopState, "frecency.json.lock")
		if err := os.Mkdir(lock, 0o700); err != nil {
			t.Fatal(err)
		}

		configDir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(configDir, "hop"), 0o700); err != nil {
			t.Fatal(err)
		}
		// One empty scan root: the scan has to succeed, it just has nothing to find.
		conf := "[scan]\nroots = [\"" + t.TempDir() + "\"]\n"
		if err := os.WriteFile(filepath.Join(configDir, "hop", "config.toml"), []byte(conf), 0o600); err != nil {
			t.Fatal(err)
		}

		cmd := exec.Command(os.Args[0], "-test.run=TestPruneFailureSubprocess")
		cmd.Env = []string{
			"HOP_PRUNE_CHILD=1",
			"HOP_PRUNE_CMD=" + command,
			"HOME=" + t.TempDir(),
			"XDG_STATE_HOME=" + stateDir,
			"XDG_CONFIG_HOME=" + configDir,
			"PATH=" + t.TempDir(),       // empty: neither command shells out
			"GOCOVERDIR=" + t.TempDir(), // silence the cover runtime warning on a coverage build
		}
		var outBuf, errBuf bytes.Buffer
		cmd.Stdout, cmd.Stderr = &outBuf, &errBuf
		err := cmd.Run()
		code := 0
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else if err != nil {
			t.Fatalf("subprocess failed to run: %v\n%s", err, errBuf.String())
		}
		return errBuf.String(), code
	}

	// The warning is asserted through the lock path it has to carry rather than
	// through a particular sentence: the message is a catalog key, so its wording
	// moves with the user's language and with any rewording of a translation,
	// while the cause does not. The absence of a pruned count is checked against
	// the English wording, which is what the child renders: it calls cmdScan
	// directly, so i18n stays on its default.
	t.Run("scan warns and still exits 0", func(t *testing.T) {
		stderr, code := run(t, "scan")
		if code != 0 {
			t.Errorf("exit = %d, want 0: the scan succeeded, only the prune failed\nstderr: %s", code, stderr)
		}
		if !strings.Contains(stderr, "frecency.json.lock") {
			t.Errorf("a failing prune must be reported, not discarded; stderr was:\n%s", stderr)
		}
		if strings.Contains(stderr, "hop: removed 0") {
			t.Errorf("nothing was pruned, so scan must print no count; stderr was:\n%s", stderr)
		}
	})

	t.Run("clean exits non-zero", func(t *testing.T) {
		stderr, code := run(t, "clean")
		if code == 0 {
			t.Errorf("exit = 0, want non-zero: pruning is all `hop clean` does\nstderr: %s", stderr)
		}
		if !strings.Contains(stderr, "frecency.json.lock") {
			t.Errorf("the failure must name its cause; stderr was:\n%s", stderr)
		}
	})
}
