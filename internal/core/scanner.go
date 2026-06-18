package core

import (
	"os"
	"path/filepath"
	"strings"
)

// Scan walks the configured roots and returns the discovered projects.
//
// A project is primarily a git repository: repos are indexed at any depth (up to
// MaxDepth) and scanning never descends past a repo root. To also catch plain
// project folders that are not git repos (e.g. a docs folder), a depth-2
// directory with no repo anywhere beneath it is indexed too; a depth-2 directory
// that does contain repos is treated as a container and is not itself indexed.
func Scan(cfg Config) []Project {
	ignore := make(map[string]bool, len(cfg.Ignore))
	for _, n := range cfg.Ignore {
		ignore[n] = true
	}
	var out []Project
	for _, root := range cfg.Roots {
		croot := CanonicalDir(root)
		if fi, err := os.Stat(croot); err != nil || !fi.IsDir() {
			continue
		}
		walkDir(croot, croot, 0, cfg.MaxDepth, ignore, &out)
	}
	return out
}

// walkDir scans dir (at the given depth, root = 0), appends discovered projects
// to out, and returns how many git repositories were found in dir's subtree.
func walkDir(root, dir string, depth, maxDepth int, ignore map[string]bool, out *[]Project) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	repos := 0
	for _, e := range entries {
		name := e.Name()
		if ignore[name] || strings.HasPrefix(name, ".") {
			continue
		}
		child := filepath.Join(dir, name)
		fi, err := os.Stat(child) // follows symlinks
		if err != nil || !fi.IsDir() {
			continue
		}
		childDepth := depth + 1
		cpath := CanonicalDir(child)

		if isRepo(child) {
			*out = append(*out, newProject(root, name, cpath))
			repos++
			continue // a repo is a leaf
		}
		if childDepth >= maxDepth {
			continue
		}
		sub := walkDir(root, child, childDepth, maxDepth, ignore, out)
		repos += sub
		// A plain depth-2 folder with no repo beneath it is itself a project.
		if sub == 0 && childDepth == 2 {
			*out = append(*out, newProject(root, name, cpath))
		}
	}
	return repos
}

func isRepo(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil
}

func newProject(root, name, cpath string) Project {
	return Project{Name: name, Path: cpath, Category: categoryOf(root, cpath)}
}

func categoryOf(root, projPath string) string {
	rel, err := filepath.Rel(root, projPath)
	if err != nil {
		return ""
	}
	parts := strings.Split(rel, string(os.PathSeparator))
	if len(parts) >= 2 {
		return parts[0]
	}
	return ""
}
