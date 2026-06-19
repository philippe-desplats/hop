package core

import (
	"os"
	"path/filepath"

	"github.com/philippe-desplats/hop/internal/store"
)

const pinsVersion = 1

// Pins holds the user's pinned project paths (favorites floated to the top of
// the Hub). Paths are canonical.
type Pins struct {
	Version int      `json:"version"`
	Paths   []string `json:"paths"`
}

// PinsPath is the canonical location of the pins file.
func PinsPath() string { return filepath.Join(StateDir(), "pins.json") }

func emptyPins() *Pins { return &Pins{Version: pinsVersion, Paths: []string{}} }

// LoadPins reads the pins file, returning an empty set on any error.
func LoadPins() *Pins {
	p := emptyPins()
	if err := store.Load(PinsPath(), p); err != nil {
		return emptyPins()
	}
	return p
}

// Set returns the pinned paths as a lookup map.
func (p *Pins) Set() map[string]bool {
	m := make(map[string]bool, len(p.Paths))
	for _, x := range p.Paths {
		m[x] = true
	}
	return m
}

// AddPin pins path (idempotent); added is false when it was already pinned.
func AddPin(path string) (added bool, err error) {
	p := emptyPins()
	_, err = store.Update(PinsPath(), p, true, func() error {
		p.Version = pinsVersion
		for _, x := range p.Paths {
			if x == path {
				return nil
			}
		}
		p.Paths = append(p.Paths, path)
		added = true
		return nil
	})
	return added, err
}

// PrunePins drops pinned paths whose directory no longer exists, returning the
// number removed. Mirrors PruneFrecency so `hop clean` keeps both stores tidy.
func PrunePins() (int, error) {
	p := emptyPins()
	removed := 0
	_, err := store.Update(PinsPath(), p, true, func() error {
		out := p.Paths[:0]
		for _, x := range p.Paths {
			if _, statErr := os.Stat(x); statErr != nil {
				removed++
				continue
			}
			out = append(out, x)
		}
		p.Paths = out
		return nil
	})
	return removed, err
}

// RemovePin unpins path; removed is false when it was not pinned.
func RemovePin(path string) (removed bool, err error) {
	p := emptyPins()
	_, err = store.Update(PinsPath(), p, true, func() error {
		out := p.Paths[:0]
		for _, x := range p.Paths {
			if x == path {
				removed = true
				continue
			}
			out = append(out, x)
		}
		p.Paths = out
		return nil
	})
	return removed, err
}
