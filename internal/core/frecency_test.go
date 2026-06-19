package core

import (
	"testing"
	"time"

	"github.com/philippe-desplats/hop/internal/store"
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

func TestNthMostRecentExceptSkipsNilEntries(t *testing.T) {
	now := time.Now()
	// A null entry is only reachable via external corruption; it must be skipped,
	// not dereferenced (the sort comparator would otherwise panic).
	f := &Frecency{Version: frecencyVersion, Entries: map[string]*fEntry{
		"/current":  {Rank: 1, LastAccess: now.Unix()},
		"/previous": {Rank: 1, LastAccess: now.Add(-1 * time.Minute).Unix()},
		"/null":     nil,
	}}
	if got := f.NthMostRecentExcept("/current", 1); got != "/previous" {
		t.Errorf("NthMostRecentExcept(/current, 1) = %q, want /previous", got)
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

func TestAgeEntriesBelowCapIsNoOp(t *testing.T) {
	entries := map[string]*fEntry{
		"/a": {Rank: 3, LastAccess: 100},
		"/b": {Rank: 5, LastAccess: 200},
	}
	ageEntries(entries)
	if entries["/a"].Rank != 3 || entries["/b"].Rank != 5 {
		t.Errorf("below the cap ranks must stay untouched, got /a=%v /b=%v", entries["/a"].Rank, entries["/b"].Rank)
	}
	if len(entries) != 2 {
		t.Errorf("no entry should be dropped below the cap, got %d", len(entries))
	}
}

func TestAgeEntriesScalesAndPrunesAtCap(t *testing.T) {
	// One hot path holds most of the mass; a sprinkling of tiny ones together
	// push the sum past maxAge and should be pruned after scaling.
	entries := map[string]*fEntry{"/hot": {Rank: maxAge, LastAccess: 999}}
	for i := 0; i < 200; i++ {
		entries[string(rune('a'+i%26))+string(rune('0'+i/26))] = &fEntry{Rank: 5, LastAccess: int64(i)}
	}
	ageEntries(entries)

	sum := 0.0
	for _, e := range entries {
		sum += e.Rank
		if e.Rank < 1 {
			t.Errorf("an entry with rank < 1 survived aging: %v", e.Rank)
		}
	}
	if sum > maxAge+1e-9 {
		t.Errorf("total rank after aging = %v, want <= maxAge (%d)", sum, maxAge)
	}
	if _, ok := entries["/hot"]; !ok {
		t.Error("the dominant hot entry must survive aging")
	}
	if len(entries) == 0 {
		t.Fatal("aging must never empty a non-trivial table")
	}
}

func TestAgeEntriesKeepsHotAboveStale(t *testing.T) {
	now := time.Now()
	f := &Frecency{Version: frecencyVersion, Entries: map[string]*fEntry{
		"/stale": {Rank: maxAge * 0.9, LastAccess: now.Add(-90 * 24 * time.Hour).Unix()},
		"/hot":   {Rank: maxAge * 0.9, LastAccess: now.Unix()},
	}}
	ageEntries(f.Entries)
	if _, ok := f.Entries["/hot"]; !ok {
		t.Fatal("the hot entry must survive")
	}
	if f.Score("/hot", now) <= f.Score("/stale", now) {
		t.Error("a recent path must still outrank an equally-ranked stale one after aging")
	}
}

func TestAddFrecencyAgesAndPersistsBounded(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	now := time.Now()
	hot := t.TempDir()
	// Seed the table just under the cap, then a single visit crosses it and must
	// trigger aging inside AddFrecency (not only in the pure helper).
	seed := &Frecency{Version: frecencyVersion, Entries: map[string]*fEntry{
		hot:      {Rank: maxAge - 1, LastAccess: now.Unix()},
		"/stale": {Rank: 2, LastAccess: now.Add(-90 * 24 * time.Hour).Unix()},
	}}
	if err := store.Save(FrecencyPath(), seed); err != nil {
		t.Fatal(err)
	}
	if _, err := AddFrecency(hot, now, true); err != nil {
		t.Fatal(err)
	}
	f := LoadFrecency()
	if len(f.Entries) == 0 {
		t.Fatal("aging must never empty the table")
	}
	if _, ok := f.Entries[hot]; !ok {
		t.Error("the freshly-visited hot path must survive aging")
	}
	sum := 0.0
	for _, e := range f.Entries {
		sum += e.Rank
	}
	if sum > maxAge+1e-9 {
		t.Errorf("persisted total rank = %v, want <= maxAge (%d)", sum, maxAge)
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
