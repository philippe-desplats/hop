package core

import (
	"path/filepath"
	"time"

	"github.com/philippe-desplats/hop/internal/store"
)

const frecencyVersion = 1

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
		return nil
	})
}

// MostRecentExcept returns the path of the most recently accessed entry whose
// path differs from exclude, or "" if there is none. Used by `p -`.
func (f *Frecency) MostRecentExcept(exclude string) string {
	best := ""
	var bestAt int64 = -1
	for path, e := range f.Entries {
		if path == exclude {
			continue
		}
		if e.LastAccess > bestAt {
			bestAt = e.LastAccess
			best = path
		}
	}
	return best
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
