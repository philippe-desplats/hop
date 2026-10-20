package i18n

import (
	"strings"
	"testing"
)

// TestCatalogKeyParity holds every shipped language to the English key set. A
// missing key resolves to English, so the gap is silent at runtime: the user
// gets one sentence of the interface in a language they did not choose.
func TestCatalogKeyParity(t *testing.T) {
	en := catalog[EN]
	if len(en) == 0 {
		t.Fatal("the English catalog is empty")
	}
	for lang, msgs := range catalog {
		if lang == EN {
			continue
		}
		for key := range en {
			if _, ok := msgs[key]; !ok {
				t.Errorf("%s is missing the key %q", lang, key)
			}
		}
		for key := range msgs {
			if _, ok := en[key]; !ok {
				t.Errorf("%s carries the key %q, which English does not", lang, key)
			}
		}
	}
}

// TestHelpMentionsFlags holds the help page of every language to the same
// discoverability. Key parity cannot see this: cli.help is one key per language,
// so a flag added to the English page and forgotten in the other three leaves
// those users with no way to find it short of reading the README.
func TestHelpMentionsFlags(t *testing.T) {
	// Each entry is a subcommand's help line and the flag that line has to carry,
	// for a flag whose own command has no help page of its own.
	for _, c := range []struct{ command, flag string }{
		{"hop query ", "--limit"},
		{"hop clean", "--dry-run"},
	} {
		for lang, msgs := range catalog {
			var cmdLine string
			for _, line := range strings.Split(msgs["cli.help"], "\n") {
				if strings.HasPrefix(strings.TrimSpace(line), c.command) {
					cmdLine = line
					break
				}
			}
			if cmdLine == "" {
				t.Errorf("%s: the help page has no `%s` line", lang, strings.TrimSpace(c.command))
				continue
			}
			if !strings.Contains(cmdLine, c.flag) {
				t.Errorf("%s: the `%s` line must mention %s, got %q",
					lang, strings.TrimSpace(c.command), c.flag, cmdLine)
			}
		}
	}
}
