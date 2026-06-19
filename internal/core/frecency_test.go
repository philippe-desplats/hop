package core

import (
	"testing"
	"time"
)

func TestScoreRecencyBuckets(t *testing.T) {
	now := time.Now()
	f := &Frecency{Version: frecencyVersion, Entries: map[string]*fEntry{
		"recent-rare":  {Rank: 1, LastAccess: now.Add(-30 * time.Minute).Unix()},     // <1h  -> x4 = 4
		"old-frequent": {Rank: 3, LastAccess: now.Add(-48 * time.Hour).Unix()},       // <1w  -> x0.5 = 1.5
		"ancient":      {Rank: 10, LastAccess: now.Add(-90 * 24 * time.Hour).Unix()}, // x0.25 = 2.5
	}}

	if got := f.Score("recent-rare", now); got != 4 {
		t.Errorf("recent-rare score = %v, want 4", got)
	}
	if got := f.Score("old-frequent", now); got != 1.5 {
		t.Errorf("old-frequent score = %v, want 1.5", got)
	}
	if f.Score("recent-rare", now) <= f.Score("old-frequent", now) {
		t.Error("a recent rare project should outrank an older frequent one")
	}
	if got := f.Score("never-seen", now); got != 0 {
		t.Errorf("unknown path score = %v, want 0", got)
	}
}

func TestMostRecentExcept(t *testing.T) {
	now := time.Now()
	f := &Frecency{Version: frecencyVersion, Entries: map[string]*fEntry{
		"/current":  {Rank: 1, LastAccess: now.Unix()},
		"/previous": {Rank: 1, LastAccess: now.Add(-1 * time.Minute).Unix()},
		"/older":    {Rank: 1, LastAccess: now.Add(-1 * time.Hour).Unix()},
	}}
	if got := f.MostRecentExcept("/current"); got != "/previous" {
		t.Errorf("MostRecentExcept(/current) = %q, want /previous", got)
	}
	if got := f.MostRecentExcept(""); got != "/current" {
		t.Errorf("MostRecentExcept(none) = %q, want /current", got)
	}
	empty := &Frecency{Version: frecencyVersion, Entries: map[string]*fEntry{}}
	if got := empty.MostRecentExcept("/x"); got != "" {
		t.Errorf("empty MostRecentExcept = %q, want empty", got)
	}
}

func TestNthMostRecentExcept(t *testing.T) {
	f := &Frecency{Version: frecencyVersion, Entries: map[string]*fEntry{
		"/a": {LastAccess: 300},
		"/b": {LastAccess: 200},
		"/c": {LastAccess: 100},
	}}
	if got := f.NthMostRecentExcept("", 1); got != "/a" {
		t.Errorf("1st = %q, want /a", got)
	}
	if got := f.NthMostRecentExcept("", 2); got != "/b" {
		t.Errorf("2nd = %q, want /b", got)
	}
	if got := f.NthMostRecentExcept("/a", 1); got != "/b" {
		t.Errorf("1st excluding /a = %q, want /b", got)
	}
	if got := f.NthMostRecentExcept("", 4); got != "" {
		t.Errorf("out of range = %q, want empty", got)
	}
	if got := f.NthMostRecentExcept("", 0); got != "" {
		t.Errorf("n=0 = %q, want empty", got)
	}
}

func TestPruneFrecency(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	live := t.TempDir()
	now := time.Now()
	if _, err := AddFrecency(live, now, true); err != nil {
		t.Fatal(err)
	}
	if _, err := AddFrecency("/no/such/hop/path/xyz", now, true); err != nil {
		t.Fatal(err)
	}
	removed, err := PruneFrecency()
	if err != nil {
		t.Fatal(err)
	}
	if removed != 1 {
		t.Errorf("removed = %d, want 1", removed)
	}
	f := LoadFrecency()
	if _, ok := f.Entries[live]; !ok {
		t.Error("live path should survive prune")
	}
	if _, ok := f.Entries["/no/such/hop/path/xyz"]; ok {
		t.Error("dead path should be pruned")
	}
}
