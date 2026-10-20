package core

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"
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

// TerminalSafe renders s for printing on a terminal. hop prints names it did not
// choose: an entry of the state directory is named by whoever can write that
// directory, which is the very thing `hop doctor` reports on. A name holding
// ESC [1A CR moves the cursor back over the verdicts already drawn and rewrites
// them, so without this the account a report accuses is the account that writes
// it. A value that needs escaping comes back quoted, so the escapes read as what
// they are rather than as part of the name. Printable text, accents included,
// is returned untouched, and so is the empty string.
func TerminalSafe(s string) string {
	if !utf8.ValidString(s) {
		return strconv.Quote(s) // its bytes are the terminal's to interpret, not ours
	}
	for _, r := range s {
		if !strconv.IsPrint(r) {
			return strconv.Quote(s)
		}
	}
	return s
}

// ExpandHome resolves a leading ~ to the user's home directory. It is exported
// because the CLI expands the same tildes as the config does: `hop track ~/x`
// and `roots = ["~/x"]` have to land on the same path, which a second copy of
// this function cannot guarantee past its next edit.
func ExpandHome(p string) string {
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
