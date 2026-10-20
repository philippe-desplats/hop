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

// TestHelpMentionsQueryLimit holds the help page of every language to the same
// discoverability. Key parity cannot see this: cli.help is one key per language,
// so a flag added to the English page and forgotten in the other three leaves
// those users with no way to find it short of reading the README.
func TestHelpMentionsQueryLimit(t *testing.T) {
	for lang, msgs := range catalog {
		help := msgs["cli.help"]
		var queryLine string
		for _, line := range strings.Split(help, "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "hop query ") {
				queryLine = line
				break
			}
		}
		if queryLine == "" {
			t.Errorf("%s: the help page has no `hop query` line", lang)
			continue
		}
		if !strings.Contains(queryLine, "--limit") {
			t.Errorf("%s: the `hop query` line must mention --limit, got %q", lang, queryLine)
		}
	}
}
