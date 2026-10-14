package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/BurntSushi/toml"
)

// Settings is the user config (~/.config/hop/config.toml).
type Settings struct {
	UI       UISettings       `toml:"ui"`
	AI       AISettings       `toml:"ai"`
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

// AISettings selects the AI assistant bound to the Hub's c/r actions.
type AISettings struct {
	// Tool: "auto" (first installed in preference order), or a name
	// ("claude" / "codex" / "aider" / "gemini").
	Tool string `toml:"tool"`
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
	ShowTmux bool   `toml:"show_tmux"` // legacy: include the tmux action (superseded by Multiplexer)
	// Multiplexer drives the t action: "auto" (tmux if present, else zellij),
	// "tmux", "zellij", or "off". When empty, show_tmux=true maps to "auto".
	Multiplexer string         `toml:"multiplexer"`
	Custom      []CustomAction `toml:"custom"` // user-defined [[actions.custom]] entries
}

// CustomAction is a user-defined Hub action ([[actions.custom]]).
type CustomAction struct {
	Key        string `toml:"key"`         // single letter, must not collide with a built-in
	Label      string `toml:"label"`       // menu label
	Command    string `toml:"command"`     // {path} and {name} are substituted
	NeedsGit   bool   `toml:"needs_git"`   // hide outside a git repo
	InTerminal bool   `toml:"in_terminal"` // true: run in the shell after cd; false: launch detached
}

type ScanSettings struct {
	Roots     []string `toml:"roots"`
	MaxDepth  int      `toml:"max_depth"`
	Ignore    []string `toml:"ignore"`
	Worktrees bool     `toml:"worktrees"` // also index git worktrees living outside the roots
}

func DefaultSettings() Settings {
	return Settings{
		UI:      UISettings{Language: "auto", Theme: "auto"},
		AI:      AISettings{Tool: "auto"},
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
	if strings.TrimSpace(s.AI.Tool) == "" {
		s.AI.Tool = "auto"
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

// actionExamples documents multiplexer and [[actions.custom]] in a file that
// sets neither. Once either is set, renderActions emits the real values in its
// place, so a save never turns a user's action into an example.
const actionExamples = `# Multiplexeur de l'action "t" : "auto" (tmux sinon zellij), "tmux", "zellij" ou "off".
# Prend le pas sur show_tmux quand il est défini ; décommente pour choisir explicitement.
# multiplexer = "auto"
# Actions personnalisées (optionnel) : chaque bloc ajoute une touche au menu.
# [[actions.custom]]
# key = "y"                  # une lettre, hors touches natives (z c r g o f t)
# label = "ouvrir dans Cursor"
# command = "cursor {path}"  # {path} et {name} sont substitués
# needs_git = false
# in_terminal = false        # false : lancé détaché (GUI) ; true : exécuté dans le shell après le cd
`

// tomlString renders v as a TOML basic string, quotes included. It is the one
// place the config writer escapes a string: Go's %q is Go escaping, so a label
// holding BEL, VT or ESC came out as \a, \v or \x1b, three escapes TOML does not
// define. The decoder then refused the whole file, LoadSettings fell back to the
// defaults, and the next save overwrote what the user had written.
//
// TOML text is UTF-8, so a value that is not valid UTF-8 has no representation
// here and returns an error rather than a file nothing can read back.
func tomlString(v string) (string, error) {
	if !utf8.ValidString(v) {
		return "", fmt.Errorf("cannot write %q to TOML: not valid UTF-8", v)
	}
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range v {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\b':
			b.WriteString(`\b`)
		case '\t':
			b.WriteString(`\t`)
		case '\n':
			b.WriteString(`\n`)
		case '\f':
			b.WriteString(`\f`)
		case '\r':
			b.WriteString(`\r`)
		default:
			// TOML allows any character but the quote, the backslash, and the
			// control characters, tab excepted. The rest go to \uXXXX.
			if r < 0x20 || r == 0x7f {
				fmt.Fprintf(&b, `\u%04X`, r)
				continue
			}
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String(), nil
}

// tomlEncoder escapes the strings of one rendered config, keeping the first
// value it could not write. The renderers stay simple format calls that way:
// they substitute enc.str everywhere they used %q, and the caller tests err
// once instead of at every field.
type tomlEncoder struct{ err error }

// str returns v as a TOML basic string, or an empty string when v cannot be
// written. The placeholder never reaches a file: renderConfig returns err.
func (e *tomlEncoder) str(v string) string {
	q, err := tomlString(v)
	if err != nil {
		if e.err == nil {
			e.err = err
		}
		return `""`
	}
	return q
}

// list returns items as a TOML array of basic strings.
func (e *tomlEncoder) list(items []string) string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = e.str(it)
	}
	return "[" + strings.Join(out, ", ") + "]"
}

// renderActions serialises the part of [actions] the editor does not expose:
// the multiplexer and the custom action blocks. The returned text ends with a
// blank line, so it drops into the template just above [scan].
func renderActions(a ActionsSettings, enc *tomlEncoder) string {
	if a.Multiplexer == "" && len(a.Custom) == 0 {
		return actionExamples + "\n"
	}
	var b strings.Builder
	if a.Multiplexer != "" {
		b.WriteString(`# Multiplexeur de l'action "t" : "auto" (tmux sinon zellij), "tmux", "zellij" ou "off".` + "\n")
		b.WriteString("# Prend le pas sur show_tmux quand il est défini.\n")
		fmt.Fprintf(&b, "multiplexer = %s\n", enc.str(a.Multiplexer))
	}
	if len(a.Custom) > 0 {
		b.WriteString("\n# Actions personnalisées : chaque bloc ajoute une touche au menu.\n")
		for i, c := range a.Custom {
			if i > 0 {
				b.WriteString("\n")
			}
			b.WriteString("[[actions.custom]]\n")
			fmt.Fprintf(&b, "key = %s\n", enc.str(c.Key))
			fmt.Fprintf(&b, "label = %s\n", enc.str(c.Label))
			fmt.Fprintf(&b, "command = %s\n", enc.str(c.Command))
			fmt.Fprintf(&b, "needs_git = %t\n", c.NeedsGit)
			fmt.Fprintf(&b, "in_terminal = %t\n", c.InTerminal)
		}
	}
	b.WriteString("\n")
	return b.String()
}

// renderConfig serialises settings to commented TOML. It returns an error when
// a value cannot be written as TOML, so the caller never puts on disk a file it
// could not read back.
func renderConfig(s Settings) (string, error) {
	enc := &tomlEncoder{}
	out := fmt.Sprintf(`# hop configuration

[ui]
# Langue de l'interface : "auto" (selon $LANG), ou "en" / "fr" / "es" / "pt".
language = %s
# Thème : "auto" (détecte le fond du terminal), ou "light" / "dark".
theme = %s

[ai]
# Assistant IA des touches c/r : "auto" (premier installé), ou "claude" / "codex" / "aider" / "gemini".
tool = %s

[shell]
# Nom de la fonction shell quotidienne (le raccourci qui fait le cd).
command = %s

[hub]
# Accès aux actions du Hub :
#   "tab"   = filtre pur, Tab ouvre le menu d'actions (défaut)
#   "shift" = comme tab, plus des raccourcis en MAJUSCULE directs depuis la liste
#   "enter" = Entrée ouvre le menu d'actions au lieu d'un cd direct
action_access = %s

[actions]
# Commande de l'action "ouvrir dans l'éditeur".
editor = %s
# Afficher l'action "session tmux" dans le menu (hérité).
show_tmux = %t
%s[scan]
# Où chercher les projets, profondeur max, et dossiers ignorés.
roots = %s
max_depth = %d
ignore = %s
# Indexer aussi les worktrees git situés hors des racines (un appel git par dépôt au scan).
worktrees = %t

[resolver]
# Classement d'un fragment : poids de la qualité de match vs frécence.
# min_margin : si l'écart 1er/2e est en dessous, le Hub s'ouvre au lieu de sauter.
w_fuzzy = %g
w_frecency = %g
min_margin = %g
`,
		enc.str(s.UI.Language), enc.str(s.UI.Theme), enc.str(s.AI.Tool), enc.str(s.Shell.Command),
		enc.str(s.Hub.ActionAccess), enc.str(s.Actions.Editor), s.Actions.ShowTmux,
		renderActions(s.Actions, enc),
		enc.list(s.Scan.Roots), s.Scan.MaxDepth, enc.list(s.Scan.Ignore), s.Scan.Worktrees,
		s.Resolver.WFuzzy, s.Resolver.WFrecency, s.Resolver.MinMargin)
	if enc.err != nil {
		return "", enc.err
	}
	return out, nil
}

// SaveSettings writes settings atomically (temp file + rename). Rendering comes
// first, so a value TOML cannot hold fails the save and leaves the file the user
// already had in place.
func SaveSettings(s Settings) error {
	data, err := renderConfig(s)
	if err != nil {
		return err
	}
	path := ConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(data), 0o600); err != nil {
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
