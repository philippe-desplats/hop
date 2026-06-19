package core

import (
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/philippe-desplats/hop/internal/store"
)

const frecencyVersion = 1

// maxAge bounds the total rank mass of the frecency table. Once the sum of all
// ranks exceeds it, every rank is scaled down proportionally (zoxide's aging
// model), so fresh projects can overtake formerly hot ones instead of fighting
// an ever-growing rank forever.
const maxAge = 10000

type fEntry struct {
	Rank       float64 `json:"rank"`
	LastAccess int64   `json:"last_access"` // unix seconds
}

// Frecency tracks how often and how recently each path was visited.
type Frecency struct {
	Version int                `json:"version"`
	Entries map[string]*fEntry `json:"entries"`
}

// FrecencyPath is the canonical location of the frecency file.
func FrecencyPath() string { return filepath.Join(StateDir(), "frecency.json") }

func emptyFrecency() *Frecency {
	return &Frecency{Version: frecencyVersion, Entries: map[string]*fEntry{}}
}

// LoadFrecency reads the frecency DB, returning an empty one on any error
// (missing or corrupt) so callers never need to handle recovery.
func LoadFrecency() *Frecency {
	f := emptyFrecency()
	if err := store.Load(FrecencyPath(), f); err != nil {
		return emptyFrecency()
	}
	if f.Entries == nil {
		f.Entries = map[string]*fEntry{}
	}
	// Drop any null entry (only reachable via external corruption) so sorters and
	// scorers never dereference a nil pointer.
	for k, v := range f.Entries {
		if v == nil {
			delete(f.Entries, k)
		}
	}
	return f
}

// AddFrecency records a visit to path. With blocking=false it uses a try-lock
// and silently skips when the lock is busy (a missed increment is harmless).
func AddFrecency(path string, now time.Time, blocking bool) (bool, error) {
	f := emptyFrecency()
	return store.Update(FrecencyPath(), f, blocking, func() error {
		if f.Version != frecencyVersion || f.Entries == nil {
			f.Version = frecencyVersion
			f.Entries = map[string]*fEntry{}
		}
		e := f.Entries[path]
		if e == nil {
			e = &fEntry{}
			f.Entries[path] = e
		}
		e.Rank++
		e.LastAccess = now.Unix()
		ageEntries(f.Entries)
		return nil
	})
}

// ageEntries caps the table's total rank mass. When the sum of all ranks
// exceeds maxAge it scales every rank by maxAge/sum, then drops entries whose
// rank fell below 1 (they have lost their learned signal). LastAccess is left
// untouched so recency buckets stay accurate. Pure helper for testability.
func ageEntries(entries map[string]*fEntry) {
	sum := 0.0
	for _, e := range entries {
		if e != nil {
			sum += e.Rank
		}
	}
	if sum <= maxAge {
		return
	}
	factor := maxAge / sum
	for path, e := range entries {
		if e == nil {
			delete(entries, path)
			continue
		}
		e.Rank *= factor
		if e.Rank < 1 {
			delete(entries, path)
		}
	}
}

// SeedFrecency bulk-seeds ranks in a single atomic write, used by `hop import`.
// Each path's rank is raised to at least the seeded value (an existing higher
// rank is never lowered) and its LastAccess is stamped to now. Aging runs once
// at the end so a large import is bounded without aging each seed in turn.
func SeedFrecency(seeds map[string]float64, now time.Time) error {
	if len(seeds) == 0 {
		return nil
	}
	f := emptyFrecency()
	_, err := store.Update(FrecencyPath(), f, true, func() error {
		if f.Version != frecencyVersion || f.Entries == nil {
			f.Version = frecencyVersion
			f.Entries = map[string]*fEntry{}
		}
		ts := now.Unix()
		for path, rank := range seeds {
			e := f.Entries[path]
			if e == nil {
				e = &fEntry{}
				f.Entries[path] = e
			}
			if rank > e.Rank {
				e.Rank = rank
			}
			e.LastAccess = ts
		}
		ageEntries(f.Entries)
		return nil
	})
	return err
}

// NthMostRecentExcept returns the nth most recently accessed entry (n=1 is the
// most recent) other than exclude, or "" if there are fewer than n. Powers the
// jump-list `p -`, `p -2`, `p -3`.
func (f *Frecency) NthMostRecentExcept(exclude string, n int) string {
	if n < 1 {
		return ""
	}
	paths := make([]string, 0, len(f.Entries))
	for path, e := range f.Entries {
		if e != nil && path != exclude {
			paths = append(paths, path)
		}
	}
	sort.Slice(paths, func(i, j int) bool {
		return f.Entries[paths[i]].LastAccess > f.Entries[paths[j]].LastAccess
	})
	if n > len(paths) {
		return ""
	}
	return paths[n-1]
}

// PruneFrecency drops entries whose directory no longer exists, returning the
// number removed.
func PruneFrecency() (int, error) {
	f := emptyFrecency()
	removed := 0
	_, err := store.Update(FrecencyPath(), f, true, func() error {
		for path := range f.Entries {
			if _, statErr := os.Stat(path); statErr != nil {
				delete(f.Entries, path)
				removed++
			}
		}
		return nil
	})
	return removed, err
}

// Score is the zoxide-style frecency: rank weighted by recency buckets.
func (f *Frecency) Score(path string, now time.Time) float64 {
	e := f.Entries[path]
	if e == nil {
		return 0
	}
	age := now.Sub(time.Unix(e.LastAccess, 0))
	if age < 0 {
		age = 0
	}
	return e.Rank * recencyMultiplier(age)
}

func recencyMultiplier(age time.Duration) float64 {
	switch {
	case age < time.Hour:
		return 4
	case age < 24*time.Hour:
		return 2
	case age < 7*24*time.Hour:
		return 0.5
	default:
		return 0.25
	}
}
