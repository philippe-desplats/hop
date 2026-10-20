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

// ShellQuote renders p as a single word a shell reads literally, and reports
// whether it could. `hop doctor` prints its fix as a command to paste, and the
// name inside that command belongs to whoever can write the state directory,
// which is the very account the line reports on: pasted bare, an entry named
// `$(...)` runs the planter's command, and it keeps being offered on every run
// after the owner has closed the hole that allowed the plant. Single quotes stop
// every expansion a shell does, and a quote inside the name is broken out of
// them and put back as a literal, in the one form sh, bash, zsh and fish all
// read alike:
//
//	'it'\''s mine'
//
// A leading ~/ stays outside the quotes, since hop and not the name is what put
// that tilde there and the user needs the shell to expand it.
//
// A path holding a byte a terminal acts on has no such word: single quotes would
// pass that byte through untouched, and the escaping that makes it readable needs
// double quotes, inside which a $ is live again. It comes back as TerminalSafe
// renders it, with false, and the caller offers no command for it.
func ShellQuote(p string) (string, bool) {
	if safe := TerminalSafe(p); safe != p {
		return safe, false
	}
	if p == "~" {
		return p, true
	}
	prefix := ""
	// A shell only expands a tilde that reaches it unquoted, slash included, so
	// the ~/ is kept out of the quotes whole.
	if strings.HasPrefix(p, "~/") {
		prefix, p = "~/", p[2:]
	}
	if p != "" && !strings.ContainsFunc(p, needsShellQuote) {
		return prefix + p, true
	}
	return prefix + "'" + strings.ReplaceAll(p, "'", `'\''`) + "'", true
}

// needsShellQuote reports whether r has to sit inside quotes to reach a command
// as itself. What is left bare is the set a shell has no rule for: letters,
// digits, and the punctuation that shows up in the paths hop prints.
func needsShellQuote(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		return false
	}
	return !strings.ContainsRune("-_./:@+,=%", r)
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
