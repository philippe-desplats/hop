package core

import (
	"os"
	"path/filepath"

	"github.com/philippe-desplats/hop/internal/store"
)

const extrasVersion = 1

// Extras holds folders the user added by hand with `hop track`, so they show up
// in the search list even without a git repository (e.g. ~/Downloads). Paths are
// canonical. They are merged into the index on every build, so `hop scan` never
// drops them.
type Extras struct {
	Version int      `json:"version"`
	Paths   []string `json:"paths"`
}

// ExtrasPath is the canonical location of the tracked-folders file.
func ExtrasPath() string { return filepath.Join(StateDir(), "extras.json") }

func emptyExtras() *Extras { return &Extras{Version: extrasVersion, Paths: []string{}} }

// LoadExtras reads the tracked-folders file, returning an empty set on any error.
func LoadExtras() *Extras {
	e := emptyExtras()
	if err := store.Load(ExtrasPath(), e); err != nil {
		return emptyExtras()
	}
	return e
}

// AddExtra records path (idempotent); added is false when it was already tracked.
func AddExtra(path string) (added bool, err error) {
	e := emptyExtras()
	_, err = store.Update(ExtrasPath(), e, true, func() error {
		e.Version = extrasVersion
		for _, x := range e.Paths {
			if x == path {
				return nil
			}
		}
		e.Paths = append(e.Paths, path)
		added = true
		return nil
	})
	return added, err
}

// AddExtras records several paths in a single atomic write (idempotent on paths
// already tracked), returning how many were newly added. Used by `hop import` so
// a bulk track does not pay one file write per folder.
func AddExtras(paths []string) (added int, err error) {
	if len(paths) == 0 {
		return 0, nil
	}
	e := emptyExtras()
	_, err = store.Update(ExtrasPath(), e, true, func() error {
		e.Version = extrasVersion
		have := make(map[string]bool, len(e.Paths))
		for _, x := range e.Paths {
			have[x] = true
		}
		for _, p := range paths {
			if have[p] {
				continue
			}
			have[p] = true
			e.Paths = append(e.Paths, p)
			added++
		}
		return nil
	})
	return added, err
}

// RemoveExtra drops path; removed is false when it was not tracked.
func RemoveExtra(path string) (removed bool, err error) {
	e := emptyExtras()
	_, err = store.Update(ExtrasPath(), e, true, func() error {
		out := e.Paths[:0]
		for _, x := range e.Paths {
			if x == path {
				removed = true
				continue
			}
			out = append(out, x)
		}
		e.Paths = out
		return nil
	})
	return removed, err
}

// PruneExtras drops tracked paths whose directory no longer exists, returning the
// number removed, so `hop clean` keeps the store tidy alongside pins and frecency.
func PruneExtras() (int, error) {
	e := emptyExtras()
	removed := 0
	_, err := store.Update(ExtrasPath(), e, true, func() error {
		out := e.Paths[:0]
		for _, x := range e.Paths {
			if _, statErr := os.Stat(x); statErr != nil {
				removed++
				continue
			}
			out = append(out, x)
		}
		e.Paths = out
		return nil
	})
	return removed, err
}

// mergeExtras appends tracked folders to the scanned projects, skipping any the
// scan already found (so a tracked folder that later gains a .git keeps the
// scanned entry and its category) and any whose directory has gone missing.
func mergeExtras(scanned []Project) []Project {
	extras := LoadExtras()
	if len(extras.Paths) == 0 {
		return scanned
	}
	seen := make(map[string]bool, len(scanned))
	for _, p := range scanned {
		seen[p.Path] = true
	}
	for _, path := range extras.Paths {
		if seen[path] {
			continue
		}
		fi, err := os.Stat(path)
		if err != nil || !fi.IsDir() {
			continue
		}
		seen[path] = true
		scanned = append(scanned, Project{Name: filepath.Base(path), Path: path})
	}
	return scanned
}
