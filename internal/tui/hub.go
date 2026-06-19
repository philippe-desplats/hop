// Package tui renders the interactive project Hub (Bubble Tea). Two modes: a
// fuzzy list ranked by frecency, and an action menu for the highlighted project
// (cd, Zed, Claude, git, remote, Finder, tmux).
package tui

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/philippe-desplats/hop/internal/action"
	"github.com/philippe-desplats/hop/internal/core"
	"github.com/philippe-desplats/hop/internal/i18n"
)

func init() {
	// Bubble Tea renders to /dev/tty, but lipgloss detects color from stdout,
	// which is a pipe under $(hop nav ...); without this it strips every color
	// to monochrome. Force truecolor so the Hub actually renders styled.
	lipgloss.SetColorProfile(termenv.TrueColor)
}

const (
	modeList = iota
	modeActions
)

// ac is an adaptive color: light variant for light terminals, dark for dark.
func ac(light, dark string) lipgloss.AdaptiveColor {
	return lipgloss.AdaptiveColor{Light: light, Dark: dark}
}

var (
	promptStyle  = lipgloss.NewStyle().Foreground(ac("#b26a00", "#ffcc66")).Bold(true)
	selStyle     = lipgloss.NewStyle().Foreground(ac("#1f6fb2", "#66ccff")).Bold(true)
	keyStyle     = lipgloss.NewStyle().Foreground(ac("#2e7d32", "#88cc77")).Bold(true)
	dimStyle     = lipgloss.NewStyle().Foreground(ac("#6b7488", "#7788aa"))
	okStyle      = lipgloss.NewStyle().Foreground(ac("#2e7d32", "#88cc77"))
	warnStyle    = lipgloss.NewStyle().Foreground(ac("#b26a00", "#ffcc66"))
	nameStyle    = lipgloss.NewStyle().Foreground(ac("#33406b", "#bbc3ff"))
	pathStyle    = lipgloss.NewStyle().Foreground(ac("#8a93a8", "#5a6b8c"))
	selBg        = ac("#cfe0ff", "#21314f")
	selBarStyle  = lipgloss.NewStyle().Background(selBg)
	selTextStyle = lipgloss.NewStyle().Background(selBg).Foreground(ac("#1a2440", "#eaf1ff")).Bold(true)
	selHLStyle   = lipgloss.NewStyle().Background(selBg).Foreground(ac("#8a3d00", "#ffd86b")).Bold(true)
	sepStyle     = lipgloss.NewStyle().Foreground(ac("#ccd3de", "#2b3b57"))
	countStyle   = lipgloss.NewStyle().Foreground(ac("#1f6fb2", "#66ccff")).Bold(true)
	hlStyle      = lipgloss.NewStyle().Foreground(ac("#9a4f00", "#ffd86b")).Bold(true) // matched chars

	// Stable per-category tints (hashed by category name), light/dark variants.
	catPalette = []lipgloss.Style{
		lipgloss.NewStyle().Foreground(ac("#5b3fd6", "#9b8cff")),
		lipgloss.NewStyle().Foreground(ac("#1f7da6", "#5fb3d4")),
		lipgloss.NewStyle().Foreground(ac("#2f7d3a", "#7fb37a")),
		lipgloss.NewStyle().Foreground(ac("#8a6a17", "#c2a86a")),
		lipgloss.NewStyle().Foreground(ac("#b04a4a", "#c98a8a")),
		lipgloss.NewStyle().Foreground(ac("#2f5fa6", "#6f9bd4")),
		lipgloss.NewStyle().Foreground(ac("#6a3fa0", "#a98ac9")),
		lipgloss.NewStyle().Foreground(ac("#1f8a78", "#5fc2b0")),
	}
)

// setupColors forces truecolor and selects the light/dark palette variant.
func setupColors(tty *os.File, theme string) {
	r := lipgloss.DefaultRenderer()
	r.SetColorProfile(termenv.TrueColor)
	r.SetHasDarkBackground(resolveDark(theme, tty))
}

// resolveDark decides between the light and dark palette. An explicit theme wins.
// "auto" prefers the COLORFGBG hint (set by many terminals, never blocks), then
// falls back to an OSC 11 query on the tty, defaulting to dark when the terminal
// stays silent. Some terminals (notably macOS Terminal.app) never answer OSC 11,
// so "auto" can guess dark there; set [ui] theme = "light" explicitly in that case.
func resolveDark(theme string, tty *os.File) bool {
	switch theme {
	case "light":
		return false
	case "dark":
		return true
	}
	if dark, ok := colorFGBGDark(); ok {
		return dark
	}
	return termenv.NewOutput(tty).HasDarkBackground()
}

// colorFGBGDark reads the COLORFGBG hint ("fg;bg" or "fg;default;bg"). Background
// indices 0-6 and 8 are dark, 7 and 9-15 are light. Returns ok=false when the
// variable is absent or unparseable.
func colorFGBGDark() (dark, ok bool) {
	v := os.Getenv("COLORFGBG")
	if v == "" {
		return false, false
	}
	parts := strings.Split(v, ";")
	n, err := strconv.Atoi(strings.TrimSpace(parts[len(parts)-1]))
	if err != nil {
		return false, false
	}
	switch n {
	case 0, 1, 2, 3, 4, 5, 6, 8:
		return true, true
	default:
		return false, true
	}
}

func catStyle(category string) lipgloss.Style {
	if category == "" {
		return pathStyle
	}
	h := 0
	for _, c := range category {
		h = h*31 + int(c)
	}
	if h < 0 {
		h = -h
	}
	return catPalette[h%len(catPalette)]
}

// gitMsg carries an async-loaded git preview for one project.
type gitMsg struct {
	path string
	info core.GitInfo
}

func loadGitCmd(path string) tea.Cmd {
	return func() tea.Msg { return gitMsg{path: path, info: core.LoadGitInfo(path)} }
}

type model struct {
	projects  []core.Project
	frec      *core.Frecency
	roots     []string
	access    string // "tab" | "shift" | "enter"
	opts      action.Options
	weights   core.RankWeights
	query     string
	matches   []core.Match
	cursor    int
	height    int
	width     int
	mode      int
	chosen    *core.Project
	actionKey string
	git       map[string]core.GitInfo // async git preview cache
	gitReq    map[string]bool         // paths already requested
	pinned    map[string]bool         // pinned project paths (favorites)
}

func newModel(projects []core.Project, frec *core.Frecency, roots []string, access string, opts action.Options, weights core.RankWeights) model {
	m := model{
		projects: projects, frec: frec, roots: roots, access: access, opts: opts, weights: weights, height: 20,
		git:    map[string]core.GitInfo{},
		gitReq: map[string]bool{},
		pinned: core.LoadPins().Set(),
	}
	m.refilter()
	return m
}

// withGit issues an async load for the highlighted project's git info when it is
// not cached yet, so scrolling never blocks on git.
func (m model) withGit() (tea.Model, tea.Cmd) {
	p := m.current()
	if p == nil {
		return m, nil
	}
	if _, done := m.git[p.Path]; done || m.gitReq[p.Path] {
		return m, nil
	}
	m.gitReq[p.Path] = true
	return m, loadGitCmd(p.Path)
}

func (m *model) refilter() {
	m.matches = core.Resolve(m.projects, m.frec, strings.Fields(m.query), time.Now(), m.weights)
	// On the bare list (no query), float pinned favorites to the top.
	if strings.TrimSpace(m.query) == "" && len(m.pinned) > 0 {
		sort.SliceStable(m.matches, func(i, j int) bool {
			return m.pinned[m.matches[i].Project.Path] && !m.pinned[m.matches[j].Project.Path]
		})
	}
	if m.cursor >= len(m.matches) {
		m.cursor = len(m.matches) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

// togglePin pins or unpins p, persists it, and refreshes the list (re-floating
// favorites on the bare list).
func (m *model) togglePin(p core.Project) {
	if m.pinned[p.Path] {
		_, _ = core.RemovePin(p.Path)
		delete(m.pinned, p.Path)
	} else {
		_, _ = core.AddPin(p.Path)
		m.pinned[p.Path] = true
	}
	m.refilter()
}

func (m model) current() *core.Project {
	if m.cursor >= 0 && m.cursor < len(m.matches) {
		p := m.matches[m.cursor].Project
		return &p
	}
	return nil
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height, m.width = msg.Height, msg.Width
		return m.withGit() // kick off the first git preview load
	case gitMsg:
		m.git[msg.path] = msg.info
		return m, nil
	case tea.KeyMsg:
		if m.mode == modeActions {
			return m.updateActions(msg)
		}
		return m.updateList(msg)
	}
	return m, nil
}

func (m model) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// "shift" mode: an uppercase action letter fires that action directly. Plain
	// (lowercase) typing still filters, so this never blocks search. macOS-safe:
	// no Option/Meta key, which composes characters on macOS.
	if m.access == "shift" && msg.Type == tea.KeyRunes && len(msg.Runes) == 1 && unicode.IsUpper(msg.Runes[0]) {
		if p := m.current(); p != nil {
			key := strings.ToLower(string(msg.Runes))
			if _, ok := action.ByKey(key, *p, m.opts); ok {
				m.chosen, m.actionKey = p, key
				return m, tea.Quit
			}
		}
		// uppercase non-action letter: fall through and filter normally
	}
	switch msg.String() {
	case "ctrl+c", "esc":
		m.chosen = nil
		return m, tea.Quit
	case "enter":
		// "enter" mode: Enter opens the action menu instead of cd.
		if m.access == "enter" {
			if m.current() != nil {
				m.mode = modeActions
			}
			return m, nil
		}
		if p := m.current(); p != nil {
			m.chosen, m.actionKey = p, "enter"
		}
		return m, tea.Quit
	case "tab":
		if m.current() != nil {
			m.mode = modeActions
		}
	case "up", "ctrl+p":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "ctrl+n":
		if m.cursor < len(m.matches)-1 {
			m.cursor++
		}
	case "backspace":
		if r := []rune(m.query); len(r) > 0 {
			m.query = string(r[:len(r)-1])
			m.cursor = 0
			m.refilter()
		}
	default:
		switch msg.Type {
		case tea.KeyRunes:
			m.query += string(msg.Runes)
		case tea.KeySpace:
			m.query += " "
		default:
			return m, nil
		}
		m.cursor = 0
		m.refilter()
	}
	return m.withGit()
}

func (m model) updateActions(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	p := m.current()
	if p == nil {
		m.mode = modeList
		return m, nil
	}
	switch msg.String() {
	case "ctrl+c":
		m.chosen = nil
		return m, tea.Quit
	case "esc", "tab":
		m.mode = modeList
		return m, nil
	case "p":
		m.togglePin(*p)
		m.mode = modeList
		return m, nil
	default:
		if _, ok := action.ByKey(msg.String(), *p, m.opts); ok {
			m.chosen, m.actionKey = p, msg.String()
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.mode == modeActions {
		return m.viewActions()
	}
	return m.viewList()
}

func (m model) viewList() string {
	var b strings.Builder

	// Header: title, query with a cursor, and a match counter.
	b.WriteString("\n  " + promptStyle.Render("hop") + dimStyle.Render(" ❯ ") + m.query + selStyle.Render("▏"))
	b.WriteString("   " + countStyle.Render(fmt.Sprintf("%d/%d", len(m.matches), len(m.projects))) + "\n\n")

	visible := m.height - 9
	if visible < 1 {
		visible = 10
	}
	start := 0
	if m.cursor >= visible {
		start = m.cursor - visible + 1
	}
	end := min(start+visible, len(m.matches))
	for i := start; i < end; i++ {
		b.WriteString(m.renderRow(i) + "\n")
	}
	if len(m.matches) == 0 {
		b.WriteString("   " + dimStyle.Render(i18n.T("hub.no_match")) + "\n")
	}

	// Separator + footer (git preview, optional shift legend, key hints).
	b.WriteString("\n  " + sepStyle.Render(strings.Repeat("─", m.contentWidth())) + "\n")
	if pv := m.gitPreview(); pv != "" {
		b.WriteString("  " + pv + "\n")
	}
	if m.access == "shift" {
		if p := m.current(); p != nil {
			b.WriteString("  " + shiftLegend(*p, m.opts) + "\n")
		}
	}
	hint := i18n.T("hub.hint.tab")
	switch m.access {
	case "shift":
		hint = i18n.T("hub.hint.shift")
	case "enter":
		hint = i18n.T("hub.hint.enter")
	}
	b.WriteString("  " + dimStyle.Render(hint))
	return b.String()
}

func (m model) contentWidth() int {
	if m.width <= 4 {
		return 76
	}
	return min(m.width-4, 100)
}

func (m model) renderRow(i int) string {
	p := m.matches[i].Project
	disp := core.DisplayPath(p.Path, m.roots)
	kws := strings.Fields(strings.ToLower(m.query))
	if i == m.cursor {
		return m.renderSelected(disp, kws)
	}
	gutter := "    "
	if m.pinned[p.Path] {
		gutter = "  " + warnStyle.Render("★") + " "
	}
	return gutter + styledPath(disp, p.Category, kws)
}

// renderSelected draws the highlighted full-width selection bar, keeping matched
// characters visible (yellow) on the bar.
func (m model) renderSelected(disp string, keywords []string) string {
	runes := []rune(disp)
	matched := matchMask(disp, keywords)
	var b strings.Builder
	b.WriteString(selTextStyle.Render("  ▸ "))
	for i := 0; i < len(runes); {
		hl := matched[i]
		j := i
		for j < len(runes) && matched[j] == hl {
			j++
		}
		seg := string(runes[i:j])
		if hl {
			b.WriteString(selHLStyle.Render(seg))
		} else {
			b.WriteString(selTextStyle.Render(seg))
		}
		i = j
	}
	return selBarStyle.Width(m.contentWidth()).Render(b.String())
}

// matchMask marks the runes of disp covered by any keyword (case-insensitive).
func matchMask(disp string, keywords []string) []bool {
	low := []rune(strings.ToLower(disp))
	mask := make([]bool, len(low))
	for _, kw := range keywords {
		kr := []rune(kw)
		if len(kr) == 0 {
			continue
		}
		for i := 0; i+len(kr) <= len(low); i++ {
			if string(low[i:i+len(kr)]) == kw {
				for j := range kr {
					mask[i+j] = true
				}
			}
		}
	}
	return mask
}

// styledPath tints the leading category segment, brightens the project name,
// dims the middle path, and highlights characters that match the query.
func styledPath(disp, category string, keywords []string) string {
	runes := []rune(disp)
	matched := matchMask(disp, keywords)
	firstSlash, lastSlash := -1, -1
	for i, r := range runes {
		if r == '/' {
			if firstSlash < 0 {
				firstSlash = i
			}
			lastSlash = i
		}
	}
	styleByID := [4]lipgloss.Style{hlStyle, catStyle(category), nameStyle, pathStyle}
	idFor := func(i int) int {
		switch {
		case matched[i]:
			return 0
		case firstSlash >= 0 && i < firstSlash:
			return 1
		case i > lastSlash:
			return 2
		default:
			return 3
		}
	}
	var b strings.Builder
	for i := 0; i < len(runes); {
		id := idFor(i)
		j := i
		for j < len(runes) && idFor(j) == id {
			j++
		}
		b.WriteString(styleByID[id].Render(string(runes[i:j])))
		i = j
	}
	return b.String()
}

// gitPreview is a one-line git summary for the highlighted project, read from
// the async cache ("…" while still loading).
func (m model) gitPreview() string {
	p := m.current()
	if p == nil {
		return ""
	}
	info, ok := m.git[p.Path]
	if !ok {
		return dimStyle.Render("⎇ …")
	}
	if !info.IsRepo {
		return dimStyle.Render(i18n.T("hub.git.none"))
	}
	state := okStyle.Render("✓ " + i18n.T("hub.git.clean"))
	if info.Dirty {
		state = warnStyle.Render("● " + i18n.T("hub.git.dirty"))
	}
	line := keyStyle.Render("⎇ "+info.Branch) + "  " + state
	if info.LastCommit != "" {
		c := info.LastCommit
		if r := []rune(c); len(r) > 50 {
			c = string(r[:50]) + "…"
		}
		line += dimStyle.Render("  " + c)
		if info.LastWhen != "" {
			line += dimStyle.Render(" · " + info.LastWhen)
		}
	}
	return line
}

// shiftLegend lists the uppercase-letter shortcuts available for p, so the shift
// mode is discoverable instead of invisible.
func shiftLegend(p core.Project, opts action.Options) string {
	var b strings.Builder
	b.WriteString(dimStyle.Render(i18n.T("hub.shift_prefix")))
	first := true
	for _, s := range action.All(opts) {
		if s.Key == "enter" || !s.Available(p) {
			continue
		}
		if !first {
			b.WriteString(dimStyle.Render(" · "))
		}
		first = false
		b.WriteString(keyStyle.Render(strings.ToUpper(s.Key)) + dimStyle.Render(" "+s.Short))
	}
	return b.String()
}

func (m model) viewActions() string {
	p := m.current()
	var b strings.Builder
	b.WriteString(promptStyle.Render("hop ❯ ") + selStyle.Render(p.Name) + dimStyle.Render("  "+core.DisplayPath(p.Path, m.roots)) + "\n\n")
	for _, s := range action.All(m.opts) {
		if !s.Available(*p) {
			continue
		}
		key := s.Key
		if key == "enter" {
			key = "↵"
		}
		b.WriteString("  " + keyStyle.Render(fmt.Sprintf("%-3s", key)) + s.Label + "\n")
	}
	pinLabel := i18n.T("action.pin")
	if m.pinned[p.Path] {
		pinLabel = i18n.T("action.unpin")
	}
	b.WriteString("  " + keyStyle.Render(fmt.Sprintf("%-3s", "p")) + warnStyle.Render("★ ") + pinLabel + "\n")
	b.WriteString("\n" + dimStyle.Render(i18n.T("hub.actions.hint")))
	return b.String()
}

// Run shows the Hub on the controlling terminal and returns the chosen project
// and the selected action key. chosen is nil when the user cancels; err is
// non-nil when there is no tty (caller should fall back to a listing).
func Run(projects []core.Project, frec *core.Frecency, roots []string, access string, opts action.Options, weights core.RankWeights, query, theme string) (chosen *core.Project, actionKey string, err error) {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = tty.Close() }()
	setupColors(tty, theme)

	mdl := newModel(projects, frec, roots, access, opts, weights)
	if query != "" {
		mdl.query = query
		mdl.refilter()
	}
	prog := tea.NewProgram(
		mdl,
		tea.WithInput(tty),
		tea.WithOutput(tty),
		tea.WithAltScreen(),
	)
	final, err := prog.Run()
	if err != nil {
		return nil, "", err
	}
	fm, _ := final.(model)
	return fm.chosen, fm.actionKey, nil
}
