package tui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/philippe-desplats/hop/internal/action"
	"github.com/philippe-desplats/hop/internal/core"
	"github.com/philippe-desplats/hop/internal/i18n"
)

// setup wizard steps, in order.
const (
	stepRoots = iota
	stepEditor
	stepAI
	stepShell
	stepConfirm
	stepCount
)

// SetupInput is everything the wizard needs to render and decide. The shell
// fields let it offer to wire the integration into the user's rc file.
type SetupInput struct {
	Settings     core.Settings
	Roots        []core.RootCandidate
	Editors      []action.Editor
	Assistants   []string
	ShellName    string // detected shell, e.g. "zsh"
	RcLabel      string // its rc file, e.g. "~/.zshrc"
	AlreadyWired bool   // the integration is already present in the rc
}

type setupModel struct {
	base core.Settings

	roots   []core.RootCandidate
	rootSel []bool // parallel to roots: checked folders

	editors   []action.Editor
	editorIdx int

	aiOptions []string // "auto" + detected assistant names
	aiIdx     int

	shellName    string
	rcLabel      string
	alreadyWired bool
	writeShell   bool // user chose to wire the rc (only when not alreadyWired)

	step      int
	cursor    int
	confirmed bool
}

// newSetupModel builds the wizard with sensible preselections: every folder that
// holds at least one repo is checked (or the first folder when none do), the
// first detected editor is chosen, the assistant defaults to "auto", and wiring
// the shell integration is offered (default yes) unless it is already present.
func newSetupModel(in SetupInput) setupModel {
	sel := make([]bool, len(in.Roots))
	any := false
	for i, r := range in.Roots {
		if r.Repos > 0 {
			sel[i] = true
			any = true
		}
	}
	if !any && len(in.Roots) > 0 {
		sel[0] = true
	}
	aiOptions := append([]string{"auto"}, in.Assistants...)
	return setupModel{
		base:         in.Settings,
		roots:        in.Roots,
		rootSel:      sel,
		editors:      in.Editors,
		aiOptions:    aiOptions,
		shellName:    in.ShellName,
		rcLabel:      in.RcLabel,
		alreadyWired: in.AlreadyWired,
		writeShell:   !in.AlreadyWired,
	}
}

// settings folds the current selections onto the loaded settings, leaving keys
// the wizard does not touch (e.g. [resolver], [ui]) untouched.
func (m setupModel) settings() core.Settings {
	s := m.base
	var roots []string
	for i, ok := range m.rootSel {
		if ok {
			roots = append(roots, m.roots[i].Path)
		}
	}
	if len(roots) > 0 {
		s.Scan.Roots = roots
	}
	if len(m.editors) > 0 {
		s.Actions.Editor = m.editors[m.editorIdx].Bin
	}
	if len(m.aiOptions) > 0 {
		s.AI.Tool = m.aiOptions[m.aiIdx]
	}
	return s
}

// itemCount is the number of navigable rows on the current step.
func (m setupModel) itemCount() int {
	switch m.step {
	case stepRoots:
		return len(m.roots)
	case stepEditor:
		return len(m.editors)
	case stepAI:
		return len(m.aiOptions)
	case stepShell:
		if m.alreadyWired {
			return 0 // informational only, nothing to choose
		}
		return 2 // yes / no
	default:
		return 0
	}
}

func (m setupModel) Init() tea.Cmd { return nil }

func (m setupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch key.String() {
	case "ctrl+c", "esc":
		m.confirmed = false
		return m, tea.Quit
	case "up", "ctrl+p", "k":
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil
	case "down", "ctrl+n", "j":
		if m.cursor < m.itemCount()-1 {
			m.cursor++
		}
		return m, nil
	case " ":
		if m.step == stepRoots && m.cursor < len(m.rootSel) {
			m.rootSel[m.cursor] = !m.rootSel[m.cursor]
		}
		return m, nil
	case "enter":
		return m.advance()
	}
	return m, nil
}

// advance records the cursor choice for single-select steps and moves to the
// next step, or confirms and quits on the final step.
func (m setupModel) advance() (tea.Model, tea.Cmd) {
	switch m.step {
	case stepEditor:
		if len(m.editors) > 0 {
			m.editorIdx = m.cursor
		}
	case stepAI:
		m.aiIdx = m.cursor
	case stepShell:
		if !m.alreadyWired {
			m.writeShell = m.cursor == 0 // option 0 = yes
		}
	case stepConfirm:
		m.confirmed = true
		return m, tea.Quit
	}
	m.step++
	m.cursor = 0
	switch m.step {
	case stepEditor:
		m.cursor = m.editorIdx
	case stepAI:
		m.cursor = m.aiIdx
	case stepShell:
		if !m.writeShell {
			m.cursor = 1
		}
	}
	return m, nil
}

func (m setupModel) View() string {
	var b strings.Builder
	b.WriteString(promptStyle.Render(i18n.T("setup.title")) + dimStyle.Render(fmt.Sprintf("   %d/%d", m.step+1, stepCount)) + "\n\n")
	switch m.step {
	case stepRoots:
		m.viewRoots(&b)
	case stepEditor:
		m.viewEditor(&b)
	case stepAI:
		m.viewAI(&b)
	case stepShell:
		m.viewShell(&b)
	case stepConfirm:
		m.viewConfirm(&b)
	}
	return b.String()
}

func (m setupModel) viewShell(b *strings.Builder) {
	b.WriteString(selStyle.Render(i18n.T("setup.shell.title")) + "\n\n")
	if m.alreadyWired {
		b.WriteString("  " + okStyle.Render("✓ ") + dimStyle.Render(i18n.Tf("setup.shell.already", m.rcLabel)) + "\n\n")
		b.WriteString(dimStyle.Render(i18n.T("setup.hint.next")))
		return
	}
	b.WriteString("  " + dimStyle.Render(i18n.Tf("setup.shell.prompt", m.rcLabel)) + "\n\n")
	for i, opt := range []string{i18n.T("setup.shell.yes"), i18n.T("setup.shell.no")} {
		m.viewRadio(b, i, opt)
	}
	b.WriteString("\n" + dimStyle.Render(i18n.T("setup.hint.single")))
}

func (m setupModel) viewRoots(b *strings.Builder) {
	b.WriteString(selStyle.Render(i18n.T("setup.roots.title")) + "\n\n")
	if len(m.roots) == 0 {
		b.WriteString("  " + dimStyle.Render(i18n.T("setup.roots.empty")) + "\n\n")
		b.WriteString(dimStyle.Render(i18n.T("setup.hint.next")))
		return
	}
	for i, r := range m.roots {
		check := dimStyle.Render("[ ]")
		if m.rootSel[i] {
			check = okStyle.Render("[x]")
		}
		marker, label := "  ", dimStyle.Render(r.Path)
		if i == m.cursor {
			marker, label = selStyle.Render("▸ "), selStyle.Render(r.Path)
		}
		fmt.Fprintf(b, "%s%s %-20s %s\n", marker, check, label, dimStyle.Render(i18n.Tf("setup.roots.repos", r.Repos)))
	}
	b.WriteString("\n" + dimStyle.Render(i18n.T("setup.hint.multi")))
}

func (m setupModel) viewEditor(b *strings.Builder) {
	b.WriteString(selStyle.Render(i18n.T("setup.editor.title")) + "\n\n")
	if len(m.editors) == 0 {
		b.WriteString("  " + dimStyle.Render(i18n.T("setup.editor.empty")) + "\n\n")
		b.WriteString(dimStyle.Render(i18n.T("setup.hint.next")))
		return
	}
	for i, e := range m.editors {
		m.viewRadio(b, i, e.Name+dimMuted(" ("+e.Bin+")"))
	}
	b.WriteString("\n" + dimStyle.Render(i18n.T("setup.hint.single")))
}

func (m setupModel) viewAI(b *strings.Builder) {
	b.WriteString(selStyle.Render(i18n.T("setup.ai.title")) + "\n\n")
	for i, name := range m.aiOptions {
		label := name
		if name == "auto" {
			label = i18n.T("setup.ai.auto")
		}
		m.viewRadio(b, i, label)
	}
	if len(m.aiOptions) <= 1 {
		b.WriteString("\n  " + dimStyle.Render(i18n.T("setup.ai.none")) + "\n")
	}
	b.WriteString("\n" + dimStyle.Render(i18n.T("setup.hint.single")))
}

// viewRadio renders one single-select row with a radio marker.
func (m setupModel) viewRadio(b *strings.Builder, i int, label string) {
	radio := dimStyle.Render("( )")
	if i == m.cursor {
		radio = keyStyle.Render("(•)")
	}
	marker, text := "  ", dimStyle.Render(label)
	if i == m.cursor {
		marker, text = selStyle.Render("▸ "), selStyle.Render(label)
	}
	fmt.Fprintf(b, "%s%s %s\n", marker, radio, text)
}

func (m setupModel) viewConfirm(b *strings.Builder) {
	s := m.settings()
	b.WriteString(selStyle.Render(i18n.T("setup.confirm.title")) + "\n\n")
	row := func(label, value string) {
		b.WriteString("  " + dimStyle.Render(fmt.Sprintf("%-10s", label)) + keyStyle.Render(value) + "\n")
	}
	row(i18n.T("setup.row.roots"), strings.Join(s.Scan.Roots, "  "))
	row(i18n.T("setup.row.editor"), s.Actions.Editor)
	row(i18n.T("setup.row.ai"), s.AI.Tool)
	switch {
	case m.alreadyWired:
		row(i18n.T("setup.row.shell"), i18n.T("setup.shellval.already"))
	case m.writeShell:
		row(i18n.T("setup.row.shell"), i18n.Tf("setup.shellval.write", m.rcLabel))
	default:
		row(i18n.T("setup.row.shell"), i18n.T("setup.shellval.skip"))
	}
	b.WriteString("\n" + dimStyle.Render(i18n.T("setup.hint.confirm")))
}

// dimMuted is a tiny helper to dim a parenthetical suffix inside a label.
func dimMuted(s string) string { return dimStyle.Render(s) }

// RunSetup shows the first-run wizard on the controlling terminal and returns the
// edited settings, whether the user confirmed, and whether they asked to wire the
// shell integration. err is non-nil when there is no tty, so the caller can fall
// back to a non-interactive preset.
func RunSetup(in SetupInput) (edited core.Settings, confirmed, writeShell bool, err error) {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return in.Settings, false, false, err
	}
	defer func() { _ = tty.Close() }()
	setupColors(tty, in.Settings.UI.Theme)

	final, err := tea.NewProgram(
		newSetupModel(in),
		tea.WithInput(tty),
		tea.WithOutput(tty),
		tea.WithAltScreen(),
	).Run()
	if err != nil {
		return in.Settings, false, false, err
	}
	fm, _ := final.(setupModel)
	return fm.settings(), fm.confirmed, fm.writeShell && !fm.alreadyWired, nil
}
