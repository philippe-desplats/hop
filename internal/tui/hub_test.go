package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/philippe-desplats/hop/internal/action"
	"github.com/philippe-desplats/hop/internal/core"
)

func stripANSI(s string) string {
	var b strings.Builder
	inEsc := false
	for _, r := range s {
		if inEsc {
			if r == 'm' {
				inEsc = false
			}
			continue
		}
		if r == 0x1b {
			inEsc = true
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
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
	leg := shiftLegend(p, action.Options{Editor: "zed", ShowTmux: true, AI: action.Assistant{Name: "claude", Run: []string{"claude"}, Resume: []string{"claude", "--resume"}}, HasAI: true})
	for _, want := range []string{"zed", "claude", "resume", "Finder", "tmux"} {
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
