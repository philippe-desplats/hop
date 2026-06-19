package core

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// RootCandidate is a conventional project directory proposed by hop setup. Path
// is the ~-collapsed form written to config; Repos is how many git repositories
// were found within (used to rank candidates and preselect the best ones).
type RootCandidate struct {
	Path  string
	Repos int
}

// candidateRoots are conventional project folder names probed under $HOME. They
// are proposed even when empty, so a fresh ~/Projects still shows up.
var candidateRoots = []string{
	"Projects", "projects", "Developer", "Development", "Developments",
	"code", "Code", "dev", "work", "workspace", "src", "repos", "git", "Sites",
}

// homeSkip are top-level home folders that are never project roots; they are
// excluded from the by-content scan so it never proposes ~/Downloads and friends.
var homeSkip = map[string]bool{
	"Library": true, "Applications": true, "Music": true, "Pictures": true,
	"Movies": true, "Videos": true, "Public": true, "Desktop": true,
	"Downloads": true, "Documents": true, "Templates": true,
}

// discoverIgnore are folders never descended into while counting repos, so a
// dependency tree never slows the scan or inflates the count.
var discoverIgnore = map[string]bool{"node_modules": true, "vendor": true}

// discoverMaxDepth bounds the repo count walk so a deep tree never stalls setup.
const discoverMaxDepth = 3

// DiscoverRoots proposes scan roots for hop setup. It probes the conventional
// folder names (kept even when empty) and additionally returns any other
// top-level home folder that actually contains git repositories, so a custom
// layout like ~/Developments is found by content rather than by guessing names.
// Results are ordered by repo count descending. Read-only and best-effort.
func DiscoverRoots() []RootCandidate {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	var out []RootCandidate
	var seen []os.FileInfo

	add := func(abs string, requireRepos bool) {
		fi, err := os.Stat(abs) // follows symlinks
		if err != nil || !fi.IsDir() {
			return
		}
		if sameAsAny(fi, seen) {
			return // a duplicate (e.g. "code" vs "Code" on a case-insensitive FS)
		}
		n := countRepos(abs, 0)
		if requireRepos && n == 0 {
			return
		}
		seen = append(seen, fi)
		out = append(out, RootCandidate{Path: HomeRelative(abs), Repos: n})
	}

	for _, name := range candidateRoots {
		add(filepath.Join(home, name), false)
	}
	if entries, err := os.ReadDir(home); err == nil {
		for _, e := range entries {
			name := e.Name()
			if strings.HasPrefix(name, ".") || homeSkip[name] {
				continue
			}
			if !e.IsDir() && e.Type()&os.ModeSymlink == 0 {
				continue // a plain file, never a project root
			}
			add(filepath.Join(home, name), true) // only by-content matches with repos
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Repos > out[j].Repos })
	return out
}

// countRepos counts git repositories under dir up to discoverMaxDepth, never
// descending past a repo (a repo is a leaf, same rule as the scanner).
func countRepos(dir string, depth int) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	n := 0
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") || discoverIgnore[e.Name()] {
			continue
		}
		child := filepath.Join(dir, e.Name())
		if isRepo(child) {
			n++
			continue
		}
		if depth+1 < discoverMaxDepth {
			n += countRepos(child, depth+1)
		}
	}
	return n
}

// sameAsAny reports whether fi is the same directory as any already-accepted one.
func sameAsAny(fi os.FileInfo, seen []os.FileInfo) bool {
	for _, s := range seen {
		if os.SameFile(fi, s) {
			return true
		}
	}
	return false
}
