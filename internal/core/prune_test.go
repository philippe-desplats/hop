package core

import (
	"slices"
	"testing"
	"time"
)

// TestDeadPathsNamesEachLostPathOnce holds the preview `hop clean --dry-run`
// reads from to what the user is about to lose: one line per project, not one
// per store row. A pinned project is a visited one too, so a path sitting in
// several stores is the normal case, and the fixture records the dead paths out
// of order because the sorted result is part of the contract.
func TestDeadPathsNamesEachLostPathOnce(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	live := t.TempDir()
	zulu, alpha := "/does/not/exist/zulu", "/does/not/exist/alpha"
	now := time.Now()
	for _, p := range []string{live, zulu, alpha} {
		if _, err := AddFrecency(p, now, true); err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range []string{live, zulu} {
		if _, err := AddPin(p); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := AddExtra(zulu); err != nil {
		t.Fatal(err)
	}

	want := []string{alpha, zulu}
	if got := DeadPaths(); !slices.Equal(got, want) {
		t.Errorf("DeadPaths = %v, want %v (each lost path once, in a stable order)", got, want)
	}
}

// TestDeadPathsOnAFreshInstall keeps the preview answerable before any store
// exists: a first-run `hop clean --dry-run` has nothing to list, which is not
// the same thing as a crash.
func TestDeadPathsOnAFreshInstall(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	if got := DeadPaths(); len(got) != 0 {
		t.Errorf("DeadPaths = %v, want nothing when no store has been written yet", got)
	}
}
