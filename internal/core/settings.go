package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// Settings is the user config (~/.config/hop/config.toml).
type Settings struct {
	UI       UISettings       `toml:"ui"`
	Shell    ShellSettings    `toml:"shell"`
	Hub      HubSettings      `toml:"hub"`
	Actions  ActionsSettings  `toml:"actions"`
	Scan     ScanSettings     `toml:"scan"`
	Resolver ResolverSettings `toml:"resolver"`
}

// UISettings holds presentation preferences.
type UISettings struct {
	// Language: "auto" (detect from $LANG), or "en" / "fr" / "es" / "pt".
	Language string `toml:"language"`
	// Theme: "auto" (detect terminal background), or "light" / "dark".
	Theme string `toml:"theme"`
}

// ResolverSettings tunes how a fragment is ranked and when ambiguity opens the Hub.
type ResolverSettings struct {
	WFuzzy    float64 `toml:"w_fuzzy"`    // weight of match quality
	WFrecency float64 `toml:"w_frecency"` // weight of frecency
	MinMargin float64 `toml:"min_margin"` // below this gap (1st vs 2nd), open the Hub instead of jumping
}

type ShellSettings struct {
	// Command is the daily shell function name (default "p"). --cmd overrides it.
	Command string `toml:"command"`
}

type HubSettings struct {
	// ActionAccess: "tab" (Tab opens the menu), "shift" (tab plus uppercase-letter
	// direct shortcuts), or "enter" (Enter opens the menu).
	ActionAccess string `toml:"action_access"`
}

type ActionsSettings struct {
	Editor   string `toml:"editor"`    // command for the "open in editor" action
	ShowTmux bool   `toml:"show_tmux"` // include the tmux action in the menu
}

type ScanSettings struct {
	Roots    []string `toml:"roots"`
	MaxDepth int      `toml:"max_depth"`
	Ignore   []string `toml:"ignore"`
}

func DefaultSettings() Settings {
	return Settings{
		UI:      UISettings{Language: "auto", Theme: "auto"},
		Shell:   ShellSettings{Command: "p"},
		Hub:     HubSettings{ActionAccess: "tab"},
		Actions: ActionsSettings{Editor: "zed", ShowTmux: false},
		Scan: ScanSettings{
			Roots:    []string{"~/Projects"},
			MaxDepth: 7,
			Ignore:   []string{"node_modules", "vendor", "_archives"},
		},
		Resolver: ResolverSettings{WFuzzy: 0.6, WFrecency: 0.4, MinMargin: 0.15},
	}
}

// ConfigPath is ~/.config/hop/config.toml (honouring XDG_CONFIG_HOME).
func ConfigPath() string {
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "hop", "config.toml")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "hop", "config.toml")
}

// LoadSettings reads the config over the defaults, so absent keys keep their
// default. Invalid values are coerced.
func LoadSettings() Settings {
	s := DefaultSettings()
	if _, err := toml.DecodeFile(ConfigPath(), &s); err != nil {
		return DefaultSettings()
	}
	switch s.Hub.ActionAccess {
	case "tab", "shift", "enter":
	default:
		s.Hub.ActionAccess = "tab"
	}
	if strings.TrimSpace(s.UI.Language) == "" {
		s.UI.Language = "auto"
	}
	switch s.UI.Theme {
	case "auto", "light", "dark":
	default:
		s.UI.Theme = "auto"
	}
	if strings.TrimSpace(s.Shell.Command) == "" {
		s.Shell.Command = "p"
	}
	if strings.TrimSpace(s.Actions.Editor) == "" {
		s.Actions.Editor = "zed"
	}
	if s.Scan.MaxDepth <= 0 {
		s.Scan.MaxDepth = 7
	}
	if len(s.Scan.Roots) == 0 {
		s.Scan.Roots = []string{"~/Projects"}
	}
	if s.Resolver.WFuzzy <= 0 && s.Resolver.WFrecency <= 0 {
		s.Resolver.WFuzzy, s.Resolver.WFrecency = 0.6, 0.4
	}
	if s.Resolver.MinMargin < 0 || s.Resolver.MinMargin > 1 {
		s.Resolver.MinMargin = 0.15
	}
	return s
}

// renderConfig serialises settings to commented TOML.
func renderConfig(s Settings) string {
	quote := func(items []string) string {
		out := make([]string, len(items))
		for i, it := range items {
			out[i] = fmt.Sprintf("%q", it)
		}
		return "[" + strings.Join(out, ", ") + "]"
	}
	return fmt.Sprintf(`# hop configuration

[ui]
# Langue de l'interface : "auto" (selon $LANG), ou "en" / "fr" / "es" / "pt".
language = %q
# Thème : "auto" (détecte le fond du terminal), ou "light" / "dark".
theme = %q

[shell]
# Nom de la fonction shell quotidienne (le raccourci qui fait le cd).
command = %q

[hub]
# Accès aux actions du Hub :
#   "tab"   = filtre pur, Tab ouvre le menu d'actions (défaut)
#   "shift" = comme tab, plus des raccourcis en MAJUSCULE directs depuis la liste
#   "enter" = Entrée ouvre le menu d'actions au lieu d'un cd direct
action_access = %q

[actions]
# Commande de l'action "ouvrir dans l'éditeur".
editor = %q
# Afficher l'action "session tmux" dans le menu.
show_tmux = %t

[scan]
# Où chercher les projets, profondeur max, et dossiers ignorés.
roots = %s
max_depth = %d
ignore = %s

[resolver]
# Classement d'un fragment : poids de la qualité de match vs frécence.
# min_margin : si l'écart 1er/2e est en dessous, le Hub s'ouvre au lieu de sauter.
w_fuzzy = %g
w_frecency = %g
min_margin = %g
`,
		s.UI.Language, s.UI.Theme, s.Shell.Command, s.Hub.ActionAccess, s.Actions.Editor, s.Actions.ShowTmux,
		quote(s.Scan.Roots), s.Scan.MaxDepth, quote(s.Scan.Ignore),
		s.Resolver.WFuzzy, s.Resolver.WFrecency, s.Resolver.MinMargin)
}

// SaveSettings writes settings atomically (temp file + rename).
func SaveSettings(s Settings) error {
	path := ConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(renderConfig(s)), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// EnsureConfig writes the default config file when none exists.
func EnsureConfig() (bool, error) {
	if _, err := os.Stat(ConfigPath()); err == nil {
		return false, nil
	}
	if err := SaveSettings(DefaultSettings()); err != nil {
		return false, err
	}
	return true, nil
}
