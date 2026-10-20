package core

import (
	"os/exec"
	"strings"
	"testing"
)

// TestTerminalSafe guards the two halves of the promise: a name hop prints comes
// out unchanged when every rune of it is printable, and comes out escaped as
// soon as one of them is something a terminal acts on. hop prints file names it
// did not choose, and a terminal reading ESC [1A from one of them redraws the
// lines already on screen.
func TestTerminalSafe(t *testing.T) {
	t.Run("printable text is returned as it is", func(t *testing.T) {
		for _, s := range []string{
			"",
			"~/.local/state/hop/index.json",
			"~/Projets/café/mon dossier",       // accents and spaces are printable
			`~/weird "quoted" name/back\slash`, // so are quotes and backslashes
			"~/项目/图书",
		} {
			if got := TerminalSafe(s); got != s {
				t.Errorf("TerminalSafe(%q) = %q, want it unchanged", s, got)
			}
		}
	})

	t.Run("anything a terminal acts on is escaped", func(t *testing.T) {
		for _, s := range []string{
			"a\x1b[1A\x1b[2K",       // cursor up, erase line: rewrites the verdict above
			"a\rstate  ~/hop  OK",   // carriage return: rewrites its own line
			"a\nstate  ~/hop  OK",   // a newline in a name forges a whole line
			"a\x00b",                // NUL
			"a\x7fb",                // DEL
			"a\x9bb",                // the single-byte CSI some terminals still honour
			"a\xff\xfeb",            // invalid UTF-8, whose bytes a terminal reads as it likes
			"\x1b]0;window title\a", // an OSC sequence
		} {
			got := TerminalSafe(s)
			if got == s {
				t.Errorf("TerminalSafe(%q) returned it unchanged", s)
			}
			for _, r := range got {
				if r < 0x20 || r == 0x7f || r == 0x9b {
					t.Errorf("TerminalSafe(%q) = %q, which still carries %q", s, got, r)
				}
			}
		}
	})
}

// TestShellQuote asks a real shell, because what the function promises is a
// property of the shell and not of the string: `hop doctor` prints its fix as a
// command to paste, and the name inside that command was chosen by whoever can
// write the state directory, which is the very account the line reports on. An
// entry named `$(touch PWNED)` must reach chmod as those 14 characters, not as a
// command the shell runs on the way.
func TestShellQuote(t *testing.T) {
	// hostile holds, next to the plain names, every way a name can carry a shell
	// out of the word it was meant to stay inside.
	hostile := []string{
		"index.json",
		"$(touch PWNED)",
		"`touch PWNED`",
		"a;touch PWNED",
		"a&&touch PWNED",
		"a|touch PWNED",
		"a\ntouch PWNED", // a newline in a word is a command separator too
		"$HOME",
		"${HOME}",
		"a b c",
		"it's mine",    // the quote the quoting itself has to survive
		`a"b`,          // and the other one
		`back\slash`,   //
		"glob*?[a-z]",  // nothing may be expanded against the directory either
		"~evil",        // a tilde a shell would read as a login name, here part of the name
		"a~b",          //
		"Projets/café", // accents stay accents
		"项目",           //
		"",             // nothing at all is still one word
	}

	t.Run("a quoted word reaches the command as the name it is", func(t *testing.T) {
		for _, name := range hostile {
			word, ok := ShellQuote(name)
			if !ok {
				continue // judged unquotable, the subtest below holds that half
			}
			// printf writes its argument and nothing else, so whatever comes back
			// is exactly the word the shell handed the command.
			out, err := exec.Command("/bin/sh", "-c", "printf %s "+word).Output()
			if err != nil {
				t.Errorf("ShellQuote(%q) = %s, which the shell refused: %v", name, word, err)
				continue
			}
			if string(out) != name {
				t.Errorf("ShellQuote(%q) = %s, which the shell read as %q", name, word, out)
			}
		}
	})

	t.Run("a leading tilde is still the shell's to expand", func(t *testing.T) {
		// doctor shows its paths home-relative, so the ~ it writes has to keep
		// working once the rest of the path is quoted. That tilde comes from hop,
		// the name after it does not.
		const home = "/tmp/hop-shellquote-home"
		for _, tc := range []struct{ path, want string }{
			{"~/.local/state/hop/index.json", home + "/.local/state/hop/index.json"},
			{"~/.local/state/hop/$(touch PWNED)", home + "/.local/state/hop/$(touch PWNED)"},
			{"~", home},
		} {
			word, ok := ShellQuote(tc.path)
			if !ok {
				t.Errorf("ShellQuote(%q) found no word for a printable path", tc.path)
				continue
			}
			cmd := exec.Command("/bin/sh", "-c", "printf %s "+word)
			cmd.Env = []string{"HOME=" + home}
			out, err := cmd.Output()
			if err != nil {
				t.Errorf("ShellQuote(%q) = %s, which the shell refused: %v", tc.path, word, err)
				continue
			}
			if string(out) != tc.want {
				t.Errorf("ShellQuote(%q) = %s, which the shell read as %q, want %q", tc.path, word, out, tc.want)
			}
		}
	})

	t.Run("a plain path is left readable", func(t *testing.T) {
		// Quotes around every path would be noise on the line a user actually
		// reads, and the lines they read are almost always this shape.
		for _, p := range []string{"~/.local/state/hop", "~/.local/state/hop/index.json", "/etc/hop.toml"} {
			word, ok := ShellQuote(p)
			if !ok || word != p {
				t.Errorf("ShellQuote(%q) = %s, %v, want it unchanged", p, word, ok)
			}
		}
	})

	t.Run("a name no shell word can carry gets none", func(t *testing.T) {
		// There is no shell quoting for a byte a terminal acts on: single quotes
		// would pass it through untouched, and the Go escaping that makes it
		// readable needs double quotes, inside which a $ is live again. So such a
		// name comes back escaped for reading, with no promise attached.
		for _, name := range []string{
			"a\x1b[1A$(touch PWNED)",
			"a\rb",
			"a\x00b",
			"a\xffb",
		} {
			word, ok := ShellQuote(name)
			if ok {
				t.Errorf("ShellQuote(%q) = %s, ok, but no shell can carry it", name, word)
			}
			if word != TerminalSafe(name) {
				t.Errorf("ShellQuote(%q) = %q, want what TerminalSafe renders, %q", name, word, TerminalSafe(name))
			}
			if strings.ContainsAny(word, "\x1b\r\x00") {
				t.Errorf("ShellQuote(%q) = %q, which still carries a byte the terminal acts on", name, word)
			}
		}
	})
}
