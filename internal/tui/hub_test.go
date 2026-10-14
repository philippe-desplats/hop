package tui

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/philippe-desplats/hop/internal/action"
	"github.com/philippe-desplats/hop/internal/core"
)

// stripANSI drops the complete ANSI escape sequences of s and keeps every other
// byte, control runes included, so an assertion can tell styling apart from
// payload: anything stripANSI removes is a sequence it recognised in full.
func stripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if n := escapeLen(s[i:]); n > 0 {
			i += n
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		b.WriteRune(r)
		i += size
	}
	return b.String()
}

// escapeLen returns the byte length of the ANSI escape sequence starting s, or
// 0 when s does not start with a complete one. An unterminated or unknown
// sequence counts as none, so the escape is kept instead of swallowing the rest
// of the string.
func escapeLen(s string) int {
	if len(s) < 2 || s[0] != 0x1b {
		return 0
	}
	switch s[1] {
	case '[': // CSI: parameter and intermediate bytes, then a final byte
		for i := 2; i < len(s); i++ {
			switch c := s[i]; {
			case c >= 0x40 && c <= 0x7e:
				return i + 1
			case c >= 0x20 && c <= 0x3f:
			default:
				return 0
			}
		}
	case ']': // OSC: terminated by BEL or by ST (ESC backslash)
		for i := 2; i < len(s); i++ {
			if s[i] == '\a' {
				return i + 1
			}
			if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '\\' {
				return i + 2
			}
		}
	}
	return 0
}

func TestStripANSI(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"sgr colours", "\x1b[31mred\x1b[0m", "red"},
		{"csi final byte other than m", "\x1b[2Jcleared", "cleared"},
		{"csi with no parameters", "plain\x1b[Kend", "plainend"},
		{"osc terminated by bel", "\x1b]0;t\aafter", "after"},
		{"osc terminated by st", "\x1b]0;t\x1b\\after", "after"},
		{"no escape at all", "plain", "plain"},
		{"multibyte text survives", "héllo\x1b[0m", "héllo"},
		{"lone control rune is kept", "a\ab", "a\ab"},
		{"escape that starts no sequence is kept", "a\x1bb", "a\x1bb"},
		{"unterminated csi is kept", "a\x1b[31", "a\x1b[31"},
		{"unterminated osc is kept", "a\x1b]0;t", "a\x1b]0;t"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := stripANSI(tc.in); got != tc.want {
				t.Errorf("stripANSI(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// matchIndex locates path in m.matches: core.Resolve reorders the list, so a test
// must never assume the project it built sits at index 0.
func matchIndex(m model, path string) int {
	for i, mt := range m.matches {
		if mt.Project.Path == path {
			return i
		}
	}
	return -1
}

func TestHubSanitizesGitPreview(t *testing.T) {
	m := newModel(sample(), &core.Frecency{}, []string{"/p"}, "tab", action.Options{}, core.DefaultWeights())
	p := m.current()
	if p == nil {
		t.Fatal("no current project")
	}

	// Branch, subject and age all come from git, i.e. from whoever authored the
	// commit. Assert on the raw string: stripANSI would remove the very escape
	// this test is about.
	m.git[p.Path] = core.GitInfo{
		IsRepo:     true,
		Branch:     "main\x1b[2J",
		LastCommit: "fix: \x1b[2J\x1b[H\x1b[31mTOTALLY CLEAN\x1b[0m",
		LastWhen:   "2 hours\nago",
	}
	pv := m.gitPreview()
	if strings.Contains(pv, "\x1b[2J") {
		t.Errorf("git preview leaks a screen-clear escape: %q", pv)
	}
	if strings.Contains(pv, "\x1b[H") {
		t.Errorf("git preview leaks a cursor-home escape: %q", pv)
	}
	if n := strings.Count(pv, "\n"); n != 0 {
		t.Errorf("git preview spans %d extra lines: %q", n, pv)
	}

	// The 50-rune budget must count visible characters, so truncation cannot cut
	// a sanitized subject mid-escape either.
	m.git[p.Path] = core.GitInfo{
		IsRepo:     true,
		Branch:     "main",
		LastCommit: strings.Repeat("a", 45) + "\x1b[31mred",
	}
	pv = m.gitPreview()
	if strings.Contains(pv, "\x1b[31m") {
		t.Errorf("truncated subject still carries a foreign colour escape: %q", pv)
	}
}

func TestHubGitPreviewKeepsOrdinaryText(t *testing.T) {
	m := newModel(sample(), &core.Frecency{}, []string{"/p"}, "tab", action.Options{}, core.DefaultWeights())
	p := m.current()
	if p == nil {
		t.Fatal("no current project")
	}
	m.git[p.Path] = core.GitInfo{
		IsRepo:     true,
		Branch:     "feature/café-☕",
		LastCommit: "feat: ajoute la préférence ✨",
		LastWhen:   "il y a 2 heures",
	}
	pv := stripANSI(m.gitPreview())
	for _, want := range []string{"feature/café-☕", "feat: ajoute la préférence ✨", "il y a 2 heures"} {
		if !strings.Contains(pv, want) {
			t.Errorf("git preview dropped ordinary text %q: %q", want, pv)
		}
	}
}

func TestHubSanitizesRowPath(t *testing.T) {
	const (
		escPath = "/p/work/evil\x1b[2Jrepo"
		nlPath  = "/p/work/two\nlines"
	)
	projects := []core.Project{
		{Name: "evil", Path: escPath, Category: "work"},
		{Name: "two", Path: nlPath, Category: "work"},
		{Name: "ops-tools", Path: "/p/work/ops-tools", Category: "work"},
	}
	m := newModel(projects, &core.Frecency{}, []string{"/p"}, "tab", action.Options{}, core.DefaultWeights())
	for _, path := range []string{escPath, nlPath} {
		i := matchIndex(m, path)
		if i < 0 {
			t.Fatalf("project %q is not in the match list", path)
		}
		for _, sel := range []bool{true, false} {
			m.cursor = i
			if !sel {
				m.cursor = (i + 1) % len(m.matches)
			}
			row := m.renderRow(i)
			if strings.Contains(row, "\x1b[2J") {
				t.Errorf("row (selected=%v) leaks a screen-clear escape: %q", sel, row)
			}
			if n := strings.Count(row, "\n"); n != 0 {
				t.Errorf("row (selected=%v) spans %d extra lines: %q", sel, n, row)
			}
		}
	}
}

func TestHubSanitizesActionMenu(t *testing.T) {
	projects := []core.Project{{Name: "evil\x1b[2J", Path: "/p/work/evil\x1b[2Jrepo", Category: "work"}}
	m := newModel(projects, &core.Frecency{}, []string{"/p"}, "tab", action.Options{}, core.DefaultWeights())
	menu := m.viewActions()
	if strings.Contains(menu, "\x1b[2J") {
		t.Errorf("action menu leaks a screen-clear escape: %q", menu)
	}
}

func TestFit(t *testing.T) {
	const disp = "work/acme/platform/services/billing/internal/handlers/subscription-renewal-worker"
	cases := []struct {
		in   string
		w    int
		want string
	}{
		{"work/ops-tools", 76, "work/ops-tools"}, // fits: returned as is
		{"work/ops-tools", 14, "work/ops-tools"}, // exactly fits
		{disp, 36, "work/acme/pl…cription-renewal-worker"},
		{disp, 10, "wor…worker"},
		{disp, 2, "w…"},
		{disp, 1, "…"},
		{disp, 0, ""},
		// Wide runes are counted in columns, not in runes.
		{"work/café/☕☕☕☕☕☕☕☕/projet-émoji", 20, "work/ca…projet-émoji"},
	}
	for _, tc := range cases {
		got := fit(tc.in, tc.w)
		if got != tc.want {
			t.Errorf("fit(%q, %d) = %q, want %q", tc.in, tc.w, got, tc.want)
		}
		if w := lipgloss.Width(got); w > tc.w {
			t.Errorf("fit(%q, %d) is %d columns wide", tc.in, tc.w, w)
		}
	}
}

func TestHubRowStaysOneLine(t *testing.T) {
	const long = "/p/work/acme/platform/services/billing/internal/handlers/subscription-renewal-worker"
	projects := []core.Project{
		{Name: "subscription-renewal-worker", Path: long, Category: "work"},
		{Name: "ops-tools", Path: "/p/work/ops-tools", Category: "work"},
	}
	m := newModel(projects, &core.Frecency{}, []string{"/p"}, "tab", action.Options{}, core.DefaultWeights())
	i := matchIndex(m, long)
	if i < 0 {
		t.Fatalf("project %q is not in the match list", long)
	}
	for _, width := range []int{80, 60, 40} {
		for _, sel := range []bool{true, false} {
			m.width = width
			m.cursor = i
			if !sel {
				m.cursor = (i + 1) % len(m.matches)
			}
			row := m.renderRow(i)
			if n := strings.Count(row, "\n"); n != 0 {
				t.Errorf("width %d selected=%v: row spans %d extra lines: %q", width, sel, n, row)
			}
			if got, max := len([]rune(stripANSI(row))), m.contentWidth(); got > max {
				t.Errorf("width %d selected=%v: row is %d runes, content width is %d: %q", width, sel, got, max, row)
			}
		}
	}
}

// TestHubShortPathRendersUnchanged pins the no-op case: a path that fits must go
// through the row renderer exactly as it did before truncation existed.
func TestHubShortPathRendersUnchanged(t *testing.T) {
	m := newModel(sample(), &core.Frecency{}, []string{"/p"}, "tab", action.Options{}, core.DefaultWeights())
	i := matchIndex(m, "/p/work/ops-tools")
	if i < 0 {
		t.Fatal("ops-tools is not in the match list")
	}
	disp := core.DisplayPath("/p/work/ops-tools", []string{"/p"})

	m.cursor = i
	if got, want := m.renderRow(i), m.renderSelected(disp, nil); got != want {
		t.Errorf("selected row = %q, want %q", got, want)
	}
	m.cursor = (i + 1) % len(m.matches)
	if got, want := m.renderRow(i), "    "+styledPath(disp, "work", nil); got != want {
		t.Errorf("row = %q, want %q", got, want)
	}
}

// hubFloorLines is the smallest view the Hub can draw: its chrome (header, the
// two blank lines, separator, git preview, key hints) is 7 fixed lines, plus
// the one row it always keeps so the cursor stays visible. A height below that
// cannot be honoured without dropping the chrome, which this ticket does not
// touch; what matters is that a short terminal no longer gets 10 rows.
const hubFloorLines = 8

func TestHubVisibleRowsRespectHeight(t *testing.T) {
	var projects []core.Project
	for i := 0; i < 30; i++ {
		projects = append(projects, core.Project{
			Name:     fmt.Sprintf("p%02d", i),
			Path:     fmt.Sprintf("/p/work/p%02d", i),
			Category: "work",
		})
	}
	m := newModel(projects, &core.Frecency{}, []string{"/p"}, "tab", action.Options{}, core.DefaultWeights())

	lines := func(h int) int {
		m.height = h
		return strings.Count(stripANSI(m.viewList()), "\n") + 1
	}
	for h := 1; h <= 12; h++ {
		if got, max := lines(h), max(h, hubFloorLines); got > max {
			t.Errorf("height %d renders %d lines, want at most %d", h, got, max)
		}
	}
	// A short terminal must get fewer rows than a taller one, never more.
	tall := lines(12)
	for h := 1; h <= 9; h++ {
		if got := lines(h); got >= tall {
			t.Errorf("height %d renders %d lines, height 12 renders %d: short terminals must get fewer rows", h, got, tall)
		}
	}
	// Heights at or above 10 keep today's behaviour exactly.
	for h, want := range map[int]int{10: 8, 11: 9, 12: 10} {
		if got := lines(h); got != want {
			t.Errorf("height %d renders %d lines, want %d", h, got, want)
		}
	}
}

func TestSanitize(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"plain", "plain"},
		{"work/café-☕", "work/café-☕"},
		{"a\x1b[2Jb", "a[2Jb"},
		{"two\nlines", "twolines"},
		{"bell\ahere", "bellhere"},
		{"tab\there", "tabhere"},
	}
	for _, tc := range cases {
		if got := sanitize(tc.in); got != tc.want {
			t.Errorf("sanitize(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestStyledPathPreservesText(t *testing.T) {
	disp := "work/acme/web-monorepo"
	styled := styledPath(disp, "work", []string{"acme", "web"})
	if !strings.ContainsRune(styled, 0x1b) {
		t.Error("expected ANSI color codes (truecolor must be forced even off-tty)")
	}
	if got := stripANSI(styled); got != disp {
		t.Fatalf("styled text = %q, want %q", got, disp)
	}
	// no keywords still preserves the text
	if got := stripANSI(styledPath(disp, "work", nil)); got != disp {
		t.Fatalf("unfiltered styled text = %q, want %q", got, disp)
	}
}

func TestShiftLegend(t *testing.T) {
	p := core.Project{Name: "x", Path: "/tmp/hop-no-such-repo-xyz"} // not a git repo
	opts := action.Options{Editor: "zed", Multiplexer: "tmux", AI: action.Assistant{Name: "claude", Run: []string{"claude"}, Resume: []string{"claude", "--resume"}}, HasAI: true}
	leg := shiftLegend(p, opts, false)
	// Platform-independent actions only; the opener-gated file-manager/remote
	// actions are covered in the action package where PATH can be stubbed.
	for _, want := range []string{"zed", "claude", "resume", "tmux", "pin"} {
		if !strings.Contains(leg, want) {
			t.Errorf("legend missing %q: %s", want, leg)
		}
	}
	for _, absent := range []string{"git", "remote"} {
		if strings.Contains(leg, absent) {
			t.Errorf("legend should hide %q outside a repo: %s", absent, leg)
		}
	}
}

func sample() []core.Project {
	return []core.Project{
		{Name: "ops-tools", Path: "/p/work/ops-tools", Category: "work"},
		{Name: "blog", Path: "/p/side/blog", Category: "side"},
		{Name: "toolbox", Path: "/p/work/toolbox", Category: "work"},
	}
}

func typeRunes(m tea.Model, s string) tea.Model {
	for _, r := range s {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	return m
}

// aiOpts returns Options with a resolved assistant so the c/r actions exist.
func aiOpts() action.Options {
	return action.Options{AI: action.Assistant{Name: "claude", Run: []string{"claude"}, Resume: []string{"claude", "--resume"}}, HasAI: true}
}

func TestHubPinnedFloatsFirst(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	if _, err := core.AddPin("/p/work/toolbox"); err != nil {
		t.Fatal(err)
	}
	m := newModel(sample(), &core.Frecency{}, []string{"/p"}, "tab", action.Options{}, core.DefaultWeights())
	if len(m.matches) == 0 || m.matches[0].Project.Path != "/p/work/toolbox" {
		t.Fatalf("pinned project should float to top on the bare list, got %+v", m.matches)
	}
}

func TestHubTogglePin(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	var m tea.Model = newModel(sample(), &core.Frecency{}, []string{"/p"}, "tab", aiOpts(), core.DefaultWeights())
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab}) // open the action menu
	p := m.(model).current()
	if p == nil {
		t.Fatal("no current project")
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}) // pin
	mm := m.(model)
	if mm.mode != modeList {
		t.Error("toggling pin should return to the list")
	}
	if !mm.pinned[p.Path] {
		t.Errorf("%s should be pinned after toggle", p.Path)
	}
	if !core.LoadPins().Set()[p.Path] {
		t.Error("pin should be persisted to disk")
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}) // unpin
	if m.(model).pinned[p.Path] {
		t.Error("second toggle should unpin")
	}
}

// pOpts returns Options carrying a custom action bound to "p", the key the pin
// toggle also wants.
func pOpts() action.Options {
	o := aiOpts()
	o.Custom = []core.CustomAction{{Key: "p", Label: "publish", Command: "make publish", InTerminal: true}}
	return o
}

// menuRowsForKey counts the action-menu rows bound to key. Rows are indented by
// two spaces and start with the key padded to three columns, which tells them
// apart from the header and the hint line.
func menuRowsForKey(menu, key string) int {
	n := 0
	for _, line := range strings.Split(stripANSI(menu), "\n") {
		if strings.HasPrefix(line, "  ") && strings.HasPrefix(strings.TrimLeft(line, " "), key+" ") {
			n++
		}
	}
	return n
}

func TestHubPinRowWithoutCustomAction(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	m := newModel(sample(), &core.Frecency{}, []string{"/p"}, "tab", aiOpts(), core.DefaultWeights())
	menu := m.viewActions()
	if n := menuRowsForKey(menu, "p"); n != 1 {
		t.Errorf("action menu shows %d rows bound to p, want the pin row:\n%s", n, stripANSI(menu))
	}
	if !strings.Contains(stripANSI(menu), "pin to favorites") {
		t.Errorf("action menu should offer the pin toggle:\n%s", stripANSI(menu))
	}
}

func TestHubCustomActionWinsOverPin(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	var m tea.Model = newModel(sample(), &core.Frecency{}, []string{"/p"}, "tab", pOpts(), core.DefaultWeights())
	p := m.(model).current()
	if p == nil {
		t.Fatal("no current project")
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab}) // open the action menu
	if n := menuRowsForKey(m.(model).viewActions(), "p"); n != 1 {
		t.Errorf("action menu shows %d rows bound to p, want 1:\n%s", n, stripANSI(m.(model).viewActions()))
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	mm := m.(model)
	if mm.chosen == nil || mm.actionKey != "p" {
		t.Fatalf("'p' should fire the custom action, got chosen=%v key=%q", mm.chosen, mm.actionKey)
	}
	if mm.pinned[p.Path] {
		t.Errorf("'p' should not pin %s when a custom action claims the key", p.Path)
	}
}

func TestHubShiftCustomActionWinsOverPin(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	var m tea.Model = newModel(sample(), &core.Frecency{}, []string{"/p"}, "shift", pOpts(), core.DefaultWeights())
	p := m.(model).current()
	if p == nil {
		t.Fatal("no current project")
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'P'}})
	mm := m.(model)
	if mm.chosen == nil || mm.actionKey != "p" {
		t.Fatalf("Shift+P should fire the custom action, got chosen=%v key=%q", mm.chosen, mm.actionKey)
	}
	if mm.pinned[p.Path] {
		t.Errorf("Shift+P should not pin %s when a custom action claims the key", p.Path)
	}
	leg := stripANSI(shiftLegend(*p, pOpts(), false))
	if n := strings.Count(leg, "P "); n != 1 {
		t.Errorf("legend lists %d entries bound to P, want 1: %s", n, leg)
	}
	if strings.Contains(leg, "pin") {
		t.Errorf("legend should not offer the pin toggle when a custom action owns P: %s", leg)
	}
}

func TestHubFilterAndSelect(t *testing.T) {
	var m tea.Model = newModel(sample(), &core.Frecency{}, []string{"/p"}, "tab", action.Options{}, core.DefaultWeights())
	m = typeRunes(m, "ops")
	mm := m.(model)
	if len(mm.matches) != 1 || mm.matches[0].Project.Name != "ops-tools" {
		t.Fatalf("filter 'ops' = %+v, want only ops-tools", mm.matches)
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if got := m.(model).chosen; got == nil || got.Name != "ops-tools" {
		t.Fatalf("enter should select ops-tools, got %+v", got)
	}
}

func TestHubSpaceMultiKeyword(t *testing.T) {
	projects := []core.Project{
		{Name: "web-monorepo", Path: "/p/work/acme/web-monorepo", Category: "work"},
		{Name: "web-shop", Path: "/p/work/globex/web-shop", Category: "work"}, // web, no acme
	}
	var m tea.Model = newModel(projects, &core.Frecency{}, []string{"/p"}, "tab", action.Options{}, core.DefaultWeights())
	m = typeRunes(m, "acme")
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = typeRunes(m, "web")
	mm := m.(model)
	if mm.query != "acme web" {
		t.Fatalf("query = %q, want 'acme web'", mm.query)
	}
	if len(mm.matches) != 1 || mm.matches[0].Project.Name != "web-monorepo" {
		t.Fatalf("matches = %+v, want only web-monorepo", mm.matches)
	}
}

func TestHubCursorMoves(t *testing.T) {
	var m tea.Model = newModel(sample(), &core.Frecency{}, []string{"/p"}, "tab", action.Options{}, core.DefaultWeights())
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.(model).cursor != 1 {
		t.Fatalf("down should move cursor to 1, got %d", m.(model).cursor)
	}
}

func TestHubActionMode(t *testing.T) {
	var m tea.Model = newModel(sample(), &core.Frecency{}, []string{"/p"}, "tab", aiOpts(), core.DefaultWeights())
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.(model).mode != modeActions {
		t.Fatal("tab should open the action menu")
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	mm := m.(model)
	if mm.chosen == nil || mm.actionKey != "c" {
		t.Fatalf("'c' should select the Claude action, got chosen=%v key=%q", mm.chosen, mm.actionKey)
	}
}

func TestHubActionEscReturnsToList(t *testing.T) {
	var m tea.Model = newModel(sample(), &core.Frecency{}, []string{"/p"}, "tab", action.Options{}, core.DefaultWeights())
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	mm := m.(model)
	if mm.mode != modeList || mm.chosen != nil {
		t.Fatalf("esc in action menu should return to list without selecting (mode=%d chosen=%v)", mm.mode, mm.chosen)
	}
}

func TestHubShiftPin(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	var m tea.Model = newModel(sample(), &core.Frecency{}, []string{"/p"}, "shift", aiOpts(), core.DefaultWeights())
	p := m.(model).current()
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'P'}}) // Shift+P toggles pin
	mm := m.(model)
	if mm.chosen != nil {
		t.Error("Shift+P should not exit the Hub")
	}
	if !mm.pinned[p.Path] {
		t.Errorf("Shift+P should pin %s", p.Path)
	}
}

func TestHubShiftMode(t *testing.T) {
	var m tea.Model = newModel(sample(), &core.Frecency{}, []string{"/p"}, "shift", aiOpts(), core.DefaultWeights())
	// Uppercase C fires the Claude action directly from the list, no Tab needed.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'C'}})
	mm := m.(model)
	if mm.chosen == nil || mm.actionKey != "c" {
		t.Fatalf("uppercase C should fire the Claude action, got chosen=%v key=%q", mm.chosen, mm.actionKey)
	}
}

func TestHubShiftLowercaseStillFilters(t *testing.T) {
	var m tea.Model = newModel(sample(), &core.Frecency{}, []string{"/p"}, "shift", action.Options{}, core.DefaultWeights())
	// Lowercase letters must keep filtering, never trigger an action.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	mm := m.(model)
	if mm.chosen != nil {
		t.Fatal("lowercase should filter, not fire an action")
	}
	if mm.query != "c" {
		t.Fatalf("lowercase should go to the query, got %q", mm.query)
	}
}

func TestHubEnterModeOpensMenu(t *testing.T) {
	var m tea.Model = newModel(sample(), &core.Frecency{}, []string{"/p"}, "enter", action.Options{}, core.DefaultWeights())
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	mm := m.(model)
	if mm.mode != modeActions || mm.chosen != nil {
		t.Fatalf("in 'enter' mode, Enter should open the menu without cd (mode=%d chosen=%v)", mm.mode, mm.chosen)
	}
}

func TestHubEscCancels(t *testing.T) {
	var m tea.Model = newModel(sample(), &core.Frecency{}, []string{"/p"}, "tab", action.Options{}, core.DefaultWeights())
	m = typeRunes(m, "ops")
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.(model).chosen != nil {
		t.Fatal("esc must not select a project")
	}
}
