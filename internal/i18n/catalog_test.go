package i18n

import "testing"

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
