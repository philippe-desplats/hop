package core

import "testing"

func TestPins(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	if added, err := AddPin("/p/a"); err != nil || !added {
		t.Fatalf("AddPin(/p/a) = %v/%v, want true/nil", added, err)
	}
	if added, _ := AddPin("/p/a"); added {
		t.Error("re-pinning should be idempotent (added=false)")
	}
	if _, err := AddPin("/p/b"); err != nil {
		t.Fatal(err)
	}

	set := LoadPins().Set()
	if !set["/p/a"] || !set["/p/b"] {
		t.Errorf("pins not persisted: %v", set)
	}

	if removed, err := RemovePin("/p/a"); err != nil || !removed {
		t.Fatalf("RemovePin(/p/a) = %v/%v, want true/nil", removed, err)
	}
	set = LoadPins().Set()
	if set["/p/a"] {
		t.Error("/p/a should be unpinned")
	}
	if !set["/p/b"] {
		t.Error("/p/b should remain pinned")
	}
	if removed, _ := RemovePin("/p/zzz"); removed {
		t.Error("removing an unpinned path should report removed=false")
	}
}
