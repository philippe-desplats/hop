package core

import "testing"

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
