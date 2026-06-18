package i18n

import (
	"testing"
	"time"
)

func TestRelAge(t *testing.T) {
	SetLanguage("fr")
	if got := RelAge(3 * time.Hour); got != "il y a 3h" {
		t.Errorf("fr 3h = %q", got)
	}
	if got := RelAge(2 * 24 * time.Hour); got != "il y a 2j" {
		t.Errorf("fr 2d = %q", got)
	}
	SetLanguage("en")
	if got := RelAge(3 * time.Hour); got != "3h ago" {
		t.Errorf("en 3h = %q", got)
	}
	if got := RelAge(10 * time.Second); got != "now" {
		t.Errorf("en now = %q", got)
	}
	SetLanguage("es")
	if got := RelAge(5 * time.Minute); got != "hace 5min" {
		t.Errorf("es 5min = %q", got)
	}
}

func TestResolveAndFallback(t *testing.T) {
	cases := map[string]Lang{
		"fr":          FR,
		"FR":          FR,
		"fr_FR.UTF-8": FR,
		"es_ES":       ES,
		"pt_BR":       PT,
		"en":          EN,
		"de":          EN, // unsupported -> English
		"":            EN, // env-less default -> English
	}
	for in, want := range cases {
		t.Setenv("LC_ALL", "")
		t.Setenv("LC_MESSAGES", "")
		t.Setenv("LANG", "")
		if got := resolve(in); got != want {
			t.Errorf("resolve(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTFallsBackToEnglishThenKey(t *testing.T) {
	SetLanguage("fr")
	if got := T("action.cd"); got != "cd ici" {
		t.Errorf("fr action.cd = %q", got)
	}
	SetLanguage("en")
	if got := T("action.cd"); got != "cd here" {
		t.Errorf("en action.cd = %q", got)
	}
	if got := T("nonexistent.key"); got != "nonexistent.key" {
		t.Errorf("unknown key should echo itself, got %q", got)
	}
}

func TestAutoDetectsFromEnv(t *testing.T) {
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_MESSAGES", "")
	t.Setenv("LANG", "es_ES.UTF-8")
	if got := resolve("auto"); got != ES {
		t.Errorf("auto with LANG=es_ES = %q, want es", got)
	}
}
