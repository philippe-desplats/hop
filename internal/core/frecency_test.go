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
