package main

import (
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// These tests drive `hop clean` through runHopCLI (help_test.go) rather than by
// calling cmdClean: what --dry-run has to prove is a listing on one stream, an
// exit code, and three store files that came out byte-identical, and only a real
// process writing real files can show that.

// cleanFixture is one child's state: a home, the live project it holds, and the
// dead paths its three stores point at.
type cleanFixture struct {
	hopHome
	live string
	dead []string
}

// stores are the three files `hop clean` prunes, in the order the report does
// not depend on.
var stores = []string{"frecency.json", "pins.json", "extras.json"}

// cleanHome lays out an isolated home with one live project and the three
// stores, each seeded with dead paths. The dead paths are deliberately spread
// over the stores, and one sits in all three at once: a pinned project is a
// visited one too, so that overlap is the normal case, not an edge case.
func cleanHome(t *testing.T) cleanFixture {
	t.Helper()
	fx := cleanFixture{hopHome: newHopHome(t, englishConfig)}

	fx.live = filepath.Join(fx.home, "live-project")
	if err := os.MkdirAll(fx.live, 0o750); err != nil {
		t.Fatal(err)
	}
	// Two dead paths under the home, to prove the preview collapses it to ~, and
	// one outside it, which has no ~ form and must be printed as it is.
	deadA := filepath.Join(fx.home, "gone-a")
	deadB := filepath.Join(fx.home, "gone-b")
	deadOutside := filepath.Join(t.TempDir(), "gone-outside")
	fx.dead = []string{deadA, deadB, deadOutside}

	frecency := `{"version":1,"entries":{` +
		`"` + fx.live + `":{"rank":9,"last_access":1},` +
		`"` + deadA + `":{"rank":3,"last_access":1},` +
		`"` + deadB + `":{"rank":2,"last_access":1},` +
		`"` + deadOutside + `":{"rank":1,"last_access":1}}}`
	pins := `{"version":1,"paths":["` + fx.live + `","` + deadA + `"]}`
	extras := `{"version":1,"paths":["` + deadA + `","` + deadB + `"]}`

	if err := os.MkdirAll(filepath.Join(fx.stateHome, "hop"), 0o700); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{
		"frecency.json": frecency,
		"pins.json":     pins,
		"extras.json":   extras,
	} {
		if err := os.WriteFile(fx.stateFile(name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return fx
}

// stateSnapshot reads every file under the state directory, keyed by name, so a
// test can prove a command wrote nothing at all: a changed byte, a dropped file
// and a newly created one each show up in the comparison.
func stateSnapshot(t *testing.T, h hopHome) map[string]string {
	t.Helper()
	dir := filepath.Join(h.stateHome, "hop")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	snap := map[string]string{}
	for _, e := range entries {
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		snap[e.Name()] = string(data)
	}
	return snap
}

// assertStateUntouched compares the state directory against an earlier
// snapshot, naming what moved: a rewritten store, a deleted one, or a file the
// run created (the lock files count, a command that only reads takes no lock).
func assertStateUntouched(t *testing.T, h hopHome, before map[string]string, what string) {
	t.Helper()
	after := stateSnapshot(t, h)
	for name, want := range before {
		switch have, ok := after[name]; {
		case !ok:
			t.Errorf("%s is gone, %s must not delete a store", name, what)
		case have != want:
			t.Errorf("%s was rewritten, %s must not touch it\nbefore:\n%s\nafter:\n%s", name, what, want, have)
		}
	}
	for name := range after {
		if _, ok := before[name]; !ok {
			t.Errorf("%s created %s, it must only read", what, name)
		}
	}
}

// stderrLines splits a report into its lines, returning nothing for an empty
// report rather than one empty line.
func stderrLines(stderr string) []string {
	trimmed := strings.TrimRight(stderr, "\n")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "\n")
}

// TestCleanDryRunPreviewsEveryDeadPath is the whole point of the flag: the user
// gets to read the list before the deletion, so every dead path is named, in the
// ~-collapsed form the rest of the CLI displays, and the count closes the
// report. A path held by several stores is one line, because what is about to be
// lost is a project, not a store row.
func TestCleanDryRunPreviewsEveryDeadPath(t *testing.T) {
	fx := cleanHome(t)

	got := runHopCLI(t, fx.hopHome, "clean", "--dry-run")
	if got.code != 0 {
		t.Fatalf("exit = %d, want 0\nstderr: %s", got.code, got.stderr)
	}
	if got.stdout != "" {
		t.Errorf("a preview is a report for a human, nothing belongs on stdout, got:\n%s", got.stdout)
	}

	lines := stderrLines(got.stderr)
	if len(lines) != len(fx.dead)+1 {
		t.Fatalf("the preview must be %d paths and one count, got %d lines:\n%s",
			len(fx.dead), len(lines), got.stderr)
	}
	// Asserted as a set: the order the three stores are read in is not something
	// a user can observe, while a missing path is.
	want := map[string]bool{
		"~/gone-a": true,
		"~/gone-b": true,
		fx.dead[2]: true, // outside the home, so it has no ~ form
	}
	for _, line := range lines[:len(lines)-1] {
		if !want[line] {
			t.Errorf("unexpected preview line %q, want one of %v", line, slices.Sorted(maps.Keys(want)))
		}
		delete(want, line)
	}
	if len(want) != 0 {
		t.Errorf("the preview left out %v, so a user would lose them without warning",
			slices.Sorted(maps.Keys(want)))
	}
	if count := lines[len(lines)-1]; !strings.Contains(count, "3") {
		t.Errorf("the last line must count the 3 dead paths, got %q", count)
	}
}

// TestCleanDryRunWritesNothing is the promise that makes the preview worth
// having: reading the list must not be the act that empties the stores.
func TestCleanDryRunWritesNothing(t *testing.T) {
	fx := cleanHome(t)
	before := stateSnapshot(t, fx.hopHome)

	got := runHopCLI(t, fx.hopHome, "clean", "--dry-run")
	if got.code != 0 {
		t.Fatalf("exit = %d, want 0\nstderr: %s", got.code, got.stderr)
	}
	assertStateUntouched(t, fx.hopHome, before, "a dry run")
}

// TestCleanDryRunIsRepeatable keeps the preview readable: frecency entries live
// in a map, so without a stable order the same state would list its paths in a
// different sequence on every run, and a user comparing two previews could not
// tell a reshuffle from a change.
func TestCleanDryRunIsRepeatable(t *testing.T) {
	fx := cleanHome(t)

	first := runHopCLI(t, fx.hopHome, "clean", "--dry-run")
	second := runHopCLI(t, fx.hopHome, "clean", "--dry-run")
	if first.stderr != second.stderr {
		t.Errorf("two previews of one unchanged state must read the same\nfirst:\n%s\nsecond:\n%s",
			first.stderr, second.stderr)
	}
}

// TestCleanWithoutTheFlagStillPrunes is the other half of the contract: the
// committed run keeps doing exactly what it did, dropping every dead entry from
// the three stores, keeping the live project, and reporting a single count with
// no listing above it.
func TestCleanWithoutTheFlagStillPrunes(t *testing.T) {
	fx := cleanHome(t)

	got := runHopCLI(t, fx.hopHome, "clean")
	if got.code != 0 {
		t.Fatalf("exit = %d, want 0\nstderr: %s", got.code, got.stderr)
	}
	if lines := stderrLines(got.stderr); len(lines) != 1 {
		t.Errorf("a committed run reports one count and lists nothing, got:\n%s", got.stderr)
	} else if !strings.Contains(lines[0], "removed") {
		t.Errorf("the report must say what it did, got %q", lines[0])
	}

	for _, name := range stores {
		data, err := os.ReadFile(fx.stateFile(name))
		if err != nil {
			t.Fatal(err)
		}
		for _, dead := range fx.dead {
			if strings.Contains(string(data), dead) {
				t.Errorf("%s still holds the dead path %s:\n%s", name, dead, data)
			}
		}
	}
	// frecency.json and pins.json held the live project; extras.json never did.
	for _, name := range []string{"frecency.json", "pins.json"} {
		data, err := os.ReadFile(fx.stateFile(name))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), fx.live) {
			t.Errorf("%s dropped the live project %s:\n%s", name, fx.live, data)
		}
	}
}

// TestCleanRefusesAnUnknownArgument is what makes --dry-run safe to type: clean
// discarded its arguments, so a mistyped `hop clean --dry-runn` pruned the three
// stores and exited 0, with nothing in the report to tell it apart from the
// preview the user meant to ask for. clean takes no positional argument either,
// so any token it does not know is a usage error, and the refusal comes before
// the pruning rather than after it.
func TestCleanRefusesAnUnknownArgument(t *testing.T) {
	for _, argv := range [][]string{
		{"clean", "--bogus"},
		{"clean", "--dry-runn"},
		{"clean", "--dry-run", "--bogus"},
		{"clean", "--bogus", "--dry-run"},
		{"clean", "oops"},
		{"clean", "-n"},
	} {
		t.Run(strings.Join(argv, " "), func(t *testing.T) {
			fx := cleanHome(t)
			before := stateSnapshot(t, fx.hopHome)

			got := runHopCLI(t, fx.hopHome, argv...)
			if got.code != 2 {
				t.Errorf("hop %v exit = %d, want 2 (usage error)\nstdout: %s\nstderr: %s",
					argv, got.code, got.stdout, got.stderr)
			}
			if got.stdout != "" {
				t.Errorf("hop %v must leave stdout empty, got:\n%s", argv, got.stdout)
			}
			if strings.TrimSpace(got.stderr) == "" {
				t.Errorf("hop %v must name the argument it refused, got nothing on stderr", argv)
			}
			assertStateUntouched(t, fx.hopHome, before, "a refused argument")
		})
	}
}
