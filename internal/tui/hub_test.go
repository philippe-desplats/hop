package tui

import (
	"strings"
	"testing"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"

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
