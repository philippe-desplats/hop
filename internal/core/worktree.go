package core

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// expandWorktrees appends git worktrees that the scan did not already find,
// typically linked worktrees living outside the configured roots (the common
// `../project-feature` layout). Worktrees under a root are already indexed (the
// scanner stats `.git`, which exists as a file in a linked worktree), so they
// dedup against the scanned set and are not added twice.
//
// It runs `git worktree list --porcelain` once per scanned repo; this one git
// call per repo is why the feature sits behind the opt-in [scan] worktrees flag.
func expandWorktrees(scanned []Project) []Project {
	seen := make(map[string]bool, len(scanned))
	for _, p := range scanned {
		seen[p.Path] = true
	}
	// Snapshot the scanned repos first: appending to the slice as we go must not
	// make us query worktrees we just discovered.
	repos := make([]string, 0, len(scanned))
	for _, p := range scanned {
		if isRepo(p.Path) {
			repos = append(repos, p.Path)
		}
	}
	for _, repo := range repos {
		for _, wt := range gitWorktrees(repo) {
			cp := CanonicalDir(wt)
			if seen[cp] || HasControlChars(cp) {
				continue
			}
			if fi, err := os.Stat(cp); err != nil || !fi.IsDir() {
				continue // a pruned or missing worktree
			}
			seen[cp] = true
			scanned = append(scanned, Project{Name: filepath.Base(cp), Path: cp, Category: "worktree"})
		}
	}
	return scanned
}

// gitWorktrees returns the worktree paths reported by `git worktree list
// --porcelain` for the repo at dir, or nil on any error (not a repo, old git).
func gitWorktrees(dir string) []string {
	//nolint:gosec // fixed "git" binary with constant args; dir is a project path from our own scanned index
	out, err := exec.Command("git", "-C", dir, "worktree", "list", "--porcelain").Output()
	if err != nil {
		return nil
	}
	return parseWorktreePaths(string(out))
}

// parseWorktreePaths extracts the paths from `git worktree list --porcelain`
// output, where each record starts with a `worktree <path>` line.
func parseWorktreePaths(out string) []string {
	var paths []string
	for _, line := range strings.Split(out, "\n") {
		if rest, ok := strings.CutPrefix(line, "worktree "); ok {
			if rest = strings.TrimSpace(rest); rest != "" {
				paths = append(paths, rest)
			}
		}
	}
	return paths
}
