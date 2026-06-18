package store

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

type counter struct {
	Version int `json:"version"`
	N       int `json:"n"`
}

// TestUpdateConcurrent proves the lock serialises read-modify-write so no
// increment is lost under concurrent writers.
func TestUpdateConcurrent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "c.json")
	const workers = 64
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c := &counter{Version: 1}
			if _, err := Update(path, c, true, func() error { c.N++; return nil }); err != nil {
				t.Errorf("update: %v", err)
			}
		}()
	}
	wg.Wait()

	got := &counter{}
	if err := Load(path, got); err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.N != workers {
		t.Fatalf("counter = %d, want %d", got.N, workers)
	}
}

// TestUpdateRecoversFromCorrupt verifies a garbage file is reset, not fatal.
func TestUpdateRecoversFromCorrupt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "c.json")
	if err := os.WriteFile(path, []byte("}{ not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	c := &counter{Version: 1}
	if _, err := Update(path, c, true, func() error { c.N = 7; return nil }); err != nil {
		t.Fatalf("update: %v", err)
	}
	got := &counter{}
	if err := Load(path, got); err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.N != 7 {
		t.Fatalf("counter = %d, want 7 (corrupt file should reset)", got.N)
	}
}

// TestLoadMissing returns an error wrapping os.ErrNotExist.
func TestLoadMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nope.json")
	if err := Load(path, &counter{}); !os.IsNotExist(err) {
		t.Fatalf("expected not-exist error, got %v", err)
	}
}
