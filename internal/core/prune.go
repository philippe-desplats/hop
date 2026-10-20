package core

import (
	"os"
	"slices"
)

// DeadPaths returns the distinct paths `hop clean` would forget: every frecency,
// pin and tracked-folder entry whose directory no longer exists. A path recorded
// in several stores is returned once, because what the user is about to lose is
// a project, not the three rows that happen to hold it.
//
// It only reads, which is what lets `hop clean --dry-run` preview the deletion
// without taking a lock or rewriting a store. The result is sorted so two
// previews of the same state read the same way: frecency entries live in a map,
// whose iteration order would otherwise reshuffle the report on every run.
func DeadPaths() []string {
	var dead []string
	seen := map[string]bool{}
	consider := func(path string) {
		if seen[path] {
			return
		}
		seen[path] = true
		if _, err := os.Stat(path); err != nil {
			dead = append(dead, path)
		}
	}
	for path := range LoadFrecency().Entries {
		consider(path)
	}
	for _, path := range LoadPins().Paths {
		consider(path)
	}
	for _, path := range LoadExtras().Paths {
		consider(path)
	}
	slices.Sort(dead)
	return dead
}
