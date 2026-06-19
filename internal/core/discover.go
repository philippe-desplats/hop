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

// candidateRoots are the conventional project folder names probed under $HOME.
var candidateRoots = []string{
	"Projects", "projects", "Developer", "code", "Code",
	"dev", "work", "src", "repos", "git",
}

// discoverMaxDepth bounds the repo count walk so a deep tree never stalls setup.
const discoverMaxDepth = 3

// DiscoverRoots probes conventional project folders under $HOME and returns those
// that exist, each annotated with a git-repo count, ordered by count descending
// then by probe order. It is read-only and best-effort: an unreadable folder
// simply counts zero.
func DiscoverRoots() []RootCandidate {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	var out []RootCandidate
	var seen []os.FileInfo
	for _, name := range candidateRoots {
		abs := filepath.Join(home, name)
		fi, err := os.Stat(abs)
		if err != nil || !fi.IsDir() {
			continue
		}
		// Dedup by identity: on case-insensitive filesystems "code" and "Code"
		// stat to the same directory and must not be listed twice. os.SameFile
		// compares inode/device, which case-string comparison cannot.
		if sameAsAny(fi, seen) {
			continue
		}
		seen = append(seen, fi)
		out = append(out, RootCandidate{
			Path:  HomeRelative(abs),
			Repos: countRepos(abs, 0),
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Repos > out[j].Repos })
	return out
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

// countRepos counts git repositories under dir up to discoverMaxDepth, never
// descending past a repo (a repo is a leaf, same rule as the scanner).
func countRepos(dir string, depth int) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	n := 0
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
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
