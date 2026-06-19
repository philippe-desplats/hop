package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtras(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	if added, err := AddExtra("/p/dl"); err != nil || !added {
		t.Fatalf("AddExtra = %v/%v, want true/nil", added, err)
	}
	if added, _ := AddExtra("/p/dl"); added {
		t.Error("re-tracking should be idempotent (added=false)")
	}
	if paths := LoadExtras().Paths; len(paths) != 1 || paths[0] != "/p/dl" {
		t.Errorf("extras not persisted: %v", paths)
	}

	if removed, err := RemoveExtra("/p/dl"); err != nil || !removed {
		t.Fatalf("RemoveExtra = %v/%v, want true/nil", removed, err)
	}
	if removed, _ := RemoveExtra("/p/dl"); removed {
		t.Error("removing an untracked path should report removed=false")
	}
}

func TestMergeExtrasAppendsExistingDirsOnly(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	real := t.TempDir()
	if _, err := AddExtra(real); err != nil {
		t.Fatal(err)
	}
	if _, err := AddExtra("/does/not/exist/" + filepath.Base(real)); err != nil {
		t.Fatal(err)
	}

	merged := mergeExtras(nil)
	if len(merged) != 1 || merged[0].Path != real {
		t.Fatalf("mergeExtras = %+v, want only the existing dir %q", merged, real)
	}
	if merged[0].Name != filepath.Base(real) {
		t.Errorf("extra Name = %q, want %q", merged[0].Name, filepath.Base(real))
	}
}

func TestMergeExtrasSkipsAlreadyScanned(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	real := t.TempDir()
	if _, err := AddExtra(real); err != nil {
		t.Fatal(err)
	}
	scanned := []Project{{Name: "scanned", Path: real, Category: "work"}}
	merged := mergeExtras(scanned)
	if len(merged) != 1 {
		t.Fatalf("mergeExtras duplicated an already-scanned path: %+v", merged)
	}
	if merged[0].Category != "work" {
		t.Errorf("scanned entry should win and keep its category, got %+v", merged[0])
	}
}

func TestPruneExtras(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	real := t.TempDir()
	gone := filepath.Join(t.TempDir(), "removed")
	if err := os.Mkdir(gone, 0o750); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{real, gone} {
		if _, err := AddExtra(p); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Remove(gone); err != nil {
		t.Fatal(err)
	}

	removed, err := PruneExtras()
	if err != nil || removed != 1 {
		t.Fatalf("PruneExtras = %d/%v, want 1/nil", removed, err)
	}
	if paths := LoadExtras().Paths; len(paths) != 1 || paths[0] != real {
		t.Errorf("after prune: %v, want only %q", paths, real)
	}
}
