package core

import (
	"os"
	"path/filepath"
	"strings"
)

// StateDir is where hop keeps its index and frecency files.
func StateDir() string {
	if x := os.Getenv("XDG_STATE_HOME"); x != "" {
		return filepath.Join(x, "hop")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "state", "hop")
}

// HasControlChars reports whether s contains a newline, carriage return or NUL.
// The shell integration parses a newline-delimited protocol and eval's its
// output, so such a value must never be emitted or stored: a directory name
// holding a newline plus a "__HOP_RUN__ <cmd>" line would otherwise be executed.
func HasControlChars(s string) bool {
	return strings.ContainsAny(s, "\n\r\x00")
}

// expandHome resolves a leading ~ to the user's home directory.
func expandHome(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			if p == "~" {
				return home
			}
			return filepath.Join(home, p[2:])
		}
	}
	return p
}

// CanonicalDir returns an absolute, symlink-resolved, cleaned path.
func CanonicalDir(p string) string {
	if abs, err := filepath.Abs(p); err == nil {
		p = abs
	}
	if res, err := filepath.EvalSymlinks(p); err == nil {
		p = res
	}
	return filepath.Clean(p)
}

// DisplayPath shortens a project path for listing: relative to the first root
// that contains it, else with $HOME collapsed to ~.
func DisplayPath(path string, roots []string) string {
	for _, r := range roots {
		cr := CanonicalDir(r)
		if rel, err := filepath.Rel(cr, path); err == nil && !strings.HasPrefix(rel, "..") {
			return rel
		}
	}
	if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(path, home) {
		return "~" + strings.TrimPrefix(path, home)
	}
	return path
}

// HomeRelative collapses a leading $HOME to ~, for portable display and config
// storage. Paths outside home (or when home is unknown) are returned unchanged.
func HomeRelative(p string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	if p == home {
		return "~"
	}
	if strings.HasPrefix(p, home+string(os.PathSeparator)) {
		return "~" + p[len(home):]
	}
	return p
}

// UnderRoots reports whether p sits inside one of the configured roots.
func UnderRoots(p string, roots []string) bool {
	p = CanonicalDir(p)
	for _, r := range roots {
		r = CanonicalDir(r)
		if r == string(os.PathSeparator) {
			return true // a root of "/" contains every absolute path
		}
		if p == r || strings.HasPrefix(p, r+string(os.PathSeparator)) {
			return true
		}
	}
	return false
}
