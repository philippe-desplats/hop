// Package i18n provides minimal UI translations (en, fr, es, pt) with an English
// fallback. The active language is set once at startup from settings or the
// environment; T/Tf are then used everywhere for user-facing text.
package i18n

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

type Lang string

const (
	EN Lang = "en"
	FR Lang = "fr"
	ES Lang = "es"
	PT Lang = "pt"
)

var supported = map[Lang]bool{EN: true, FR: true, ES: true, PT: true}

var (
	mu      sync.RWMutex
	current = EN
)

// SetLanguage sets the active language. "" or "auto" detects from the
// environment ($LC_ALL, $LC_MESSAGES, $LANG); anything unsupported falls back to
// English.
func SetLanguage(lang string) {
	mu.Lock()
	current = resolve(lang)
	mu.Unlock()
}

// Current returns the active language code.
func Current() Lang {
	mu.RLock()
	defer mu.RUnlock()
	return current
}

func resolve(lang string) Lang {
	lang = strings.ToLower(strings.TrimSpace(lang))
	if lang == "" || lang == "auto" {
		lang = detectEnv()
	}
	if len(lang) >= 2 {
		if l := Lang(lang[:2]); supported[l] {
			return l
		}
	}
	return EN
}

func detectEnv() string {
	for _, k := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if v := os.Getenv(k); v != "" {
			return v // e.g. "fr_FR.UTF-8"; resolve() takes the first two letters
		}
	}
	return "en"
}

// T returns the translation for key in the active language, falling back to
// English, then to the key itself.
func T(key string) string {
	lang := Current()
	if m, ok := catalog[lang]; ok {
		if s, ok := m[key]; ok {
			return s
		}
	}
	if s, ok := catalog[EN][key]; ok {
		return s
	}
	return key
}

// Tf is T followed by fmt.Sprintf.
func Tf(key string, args ...any) string {
	return fmt.Sprintf(T(key), args...)
}

// RelAge formats a duration as a localized relative age (e.g. "il y a 3h").
func RelAge(d time.Duration) string {
	if d < time.Minute {
		return T("time.now")
	}
	var n int
	var u string
	switch {
	case d < time.Hour:
		n, u = int(d/time.Minute), T("time.u.min")
	case d < 24*time.Hour:
		n, u = int(d/time.Hour), T("time.u.hour")
	case d < 7*24*time.Hour:
		n, u = int(d/(24*time.Hour)), T("time.u.day")
	case d < 30*24*time.Hour:
		n, u = int(d/(7*24*time.Hour)), T("time.u.week")
	case d < 365*24*time.Hour:
		n, u = int(d/(30*24*time.Hour)), T("time.u.month")
	default:
		n, u = int(d/(365*24*time.Hour)), T("time.u.year")
	}
	return Tf("time.fmt", fmt.Sprintf("%d%s", n, u))
}
