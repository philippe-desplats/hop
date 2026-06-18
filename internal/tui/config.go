package tui

import (
	"os"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/philippe-desplats/hop/internal/core"
	"github.com/philippe-desplats/hop/internal/i18n"
)

const (
	fieldSelect = iota
	fieldText
)

type configField struct {
	id      string
	label   string
	help    string
	kind    int
	options []string // select
	idx     int      // select
	value   string   // text
}

type configModel struct {
	base   core.Settings // loaded settings, so fields not in the editor survive a save
	fields []configField
	cursor int
	saved  bool
}

func indexOf(opts []string, v string) int {
	for i, o := range opts {
		if o == v {
			return i
		}
	}
	return 0
}

func boolIdx(b bool) int {
	if b {
		return 1
	}
	return 0
}

func newConfigModel(s core.Settings) configModel {
	access := []string{"tab", "shift", "enter"}
	langs := []string{"auto", "en", "fr", "es", "pt"}
	themes := []string{"auto", "light", "dark"}
	onoff := []string{i18n.T("config.opt.no"), i18n.T("config.opt.yes")}
	return configModel{
		base: s,
		fields: []configField{
			{id: "command", kind: fieldText, label: i18n.T("config.field.command"), value: s.Shell.Command,
				help: i18n.T("config.help.command")},
			{id: "action_access", kind: fieldSelect, label: i18n.T("config.field.access"), options: access, idx: indexOf(access, s.Hub.ActionAccess),
				help: i18n.T("config.help.access")},
			{id: "editor", kind: fieldText, label: i18n.T("config.field.editor"), value: s.Actions.Editor,
				help: i18n.T("config.help.editor")},
			{id: "show_tmux", kind: fieldSelect, label: i18n.T("config.field.tmux"), options: onoff, idx: boolIdx(s.Actions.ShowTmux)},
			{id: "roots", kind: fieldText, label: i18n.T("config.field.roots"), value: strings.Join(s.Scan.Roots, " "),
				help: i18n.T("config.help.roots")},
			{id: "max_depth", kind: fieldText, label: i18n.T("config.field.depth"), value: strconv.Itoa(s.Scan.MaxDepth),
				help: i18n.T("config.help.depth")},
			{id: "ignore", kind: fieldText, label: i18n.T("config.field.ignore"), value: strings.Join(s.Scan.Ignore, " "),
				help: i18n.T("config.help.ignore")},
			{id: "language", kind: fieldSelect, label: i18n.T("config.field.language"), options: langs, idx: indexOf(langs, s.UI.Language)},
			{id: "theme", kind: fieldSelect, label: i18n.T("config.field.theme"), options: themes, idx: indexOf(themes, s.UI.Theme)},
		},
	}
}

// settings rebuilds Settings from the current field values, preserving any
// settings the editor does not expose (e.g. [resolver]).
func (m configModel) settings() core.Settings {
	s := m.base
	for _, f := range m.fields {
		switch f.id {
		case "command":
			s.Shell.Command = strings.TrimSpace(f.value)
		case "action_access":
			s.Hub.ActionAccess = f.options[f.idx]
		case "editor":
			s.Actions.Editor = strings.TrimSpace(f.value)
		case "show_tmux":
			s.Actions.ShowTmux = f.idx == 1
		case "roots":
			s.Scan.Roots = strings.Fields(f.value)
		case "max_depth":
			if n, err := strconv.Atoi(strings.TrimSpace(f.value)); err == nil {
				s.Scan.MaxDepth = n
			}
		case "ignore":
			s.Scan.Ignore = strings.Fields(f.value)
		case "language":
			s.UI.Language = f.options[f.idx]
		case "theme":
			s.UI.Theme = f.options[f.idx]
		}
	}
	return s
}

func (m configModel) Init() tea.Cmd { return nil }

func (m configModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch key.String() {
	case "ctrl+c", "esc":
		m.saved = false
		return m, tea.Quit
	case "enter", "ctrl+s":
		m.saved = true
		return m, tea.Quit
	case "up", "ctrl+p":
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil
	case "down", "ctrl+n":
		if m.cursor < len(m.fields)-1 {
			m.cursor++
		}
		return m, nil
	}

	f := &m.fields[m.cursor]
	if f.kind == fieldSelect {
		switch key.String() {
		case "left", "h":
			f.idx = (f.idx - 1 + len(f.options)) % len(f.options)
		case "right", "l":
			f.idx = (f.idx + 1) % len(f.options)
		}
		return m, nil
	}
	// text field
	switch key.String() {
	case "backspace":
		if r := []rune(f.value); len(r) > 0 {
			f.value = string(r[:len(r)-1])
		}
	default:
		switch key.Type {
		case tea.KeyRunes:
			f.value += string(key.Runes)
		case tea.KeySpace:
			f.value += " "
		}
	}
	return m, nil
}

func (m configModel) View() string {
	var b strings.Builder
	b.WriteString(promptStyle.Render(i18n.T("config.title")) + "\n\n")
	for i, f := range m.fields {
		marker, label := "  ", dimStyle.Render(f.label)
		if i == m.cursor {
			marker, label = selStyle.Render("▸ "), selStyle.Render(f.label)
		}
		b.WriteString(marker + label + "\n    ")
		if f.kind == fieldSelect {
			for j, opt := range f.options {
				if j == f.idx {
					b.WriteString(keyStyle.Render("["+opt+"]") + " ")
				} else {
					b.WriteString(dimStyle.Render(opt) + " ")
				}
			}
		} else if i == m.cursor {
			b.WriteString(keyStyle.Render(f.value + "▏"))
		} else {
			b.WriteString(keyStyle.Render(f.value))
		}
		b.WriteString("\n")
		if i == m.cursor && f.help != "" {
			b.WriteString("    " + dimStyle.Render(f.help) + "\n")
		}
		b.WriteString("\n")
	}
	b.WriteString(dimStyle.Render(i18n.T("config.hint")))
	return b.String()
}

// RunConfig shows the settings editor on the controlling terminal and returns
// the edited settings plus whether the user asked to save. err is non-nil when
// there is no tty (caller should fall back to printing).
func RunConfig(s core.Settings) (edited core.Settings, saved bool, err error) {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return s, false, err
	}
	defer func() { _ = tty.Close() }()
	setupColors(tty, s.UI.Theme)

	final, err := tea.NewProgram(
		newConfigModel(s),
		tea.WithInput(tty),
		tea.WithOutput(tty),
		tea.WithAltScreen(),
	).Run()
	if err != nil {
		return s, false, err
	}
	fm, _ := final.(configModel)
	return fm.settings(), fm.saved, nil
}
