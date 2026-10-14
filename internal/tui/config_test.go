package tui

import (
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/philippe-desplats/hop/internal/core"
	"github.com/philippe-desplats/hop/internal/i18n"
)

func TestConfigCycleSelectAndSave(t *testing.T) {
	var m tea.Model = newConfigModel(core.DefaultSettings())
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})  // command -> action_access
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight}) // tab -> shift
	if got := m.(configModel).settings().Hub.ActionAccess; got != "shift" {
		t.Fatalf("after right, access = %q, want shift", got)
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter}) // save
	cm := m.(configModel)
	if !cm.saved {
		t.Fatal("enter should save")
	}
	if cm.settings().Hub.ActionAccess != "shift" {
		t.Fatal("saved settings should reflect the edit")
	}
}

func TestConfigEditTextField(t *testing.T) {
	var m tea.Model = newConfigModel(core.DefaultSettings()) // cursor on "command"
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})      // remove "p"
	m = typeRunes(m, "pp")
	if got := m.(configModel).settings().Shell.Command; got != "pp" {
		t.Fatalf("command = %q, want pp", got)
	}
}

// The editor shows neither the multiplexer nor the custom actions, so settings
// must hand them back exactly as it got them.
func TestConfigKeepsUnexposedActionSettings(t *testing.T) {
	s := core.DefaultSettings()
	s.Actions.Multiplexer = "zellij"
	s.Actions.Custom = []core.CustomAction{{Key: "y", Label: "open in Cursor", Command: "cursor {path}"}}

	got := newConfigModel(s).settings()
	if got.Actions.Multiplexer != "zellij" {
		t.Errorf("multiplexer = %q, want zellij", got.Actions.Multiplexer)
	}
	if len(got.Actions.Custom) != 1 || got.Actions.Custom[0] != s.Actions.Custom[0] {
		t.Errorf("custom actions = %+v, want %+v", got.Actions.Custom, s.Actions.Custom)
	}
}

// focusField moves the cursor onto the field with the given id.
func focusField(t *testing.T, m tea.Model, id string) tea.Model {
	t.Helper()
	for range m.(configModel).fields {
		cm := m.(configModel)
		if cm.fields[cm.cursor].id == id {
			return m
		}
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	t.Fatalf("no field %q in the editor", id)
	return m
}

// A root or an ignore pattern holding a space must not be split in two by a
// save the user never asked for.
func TestConfigKeepsUntouchedListsVerbatim(t *testing.T) {
	s := core.DefaultSettings()
	s.Scan.Roots = []string{"/Users/me/My Projects"}
	s.Scan.Ignore = []string{"node modules"}

	got := newConfigModel(s).settings()
	if len(got.Scan.Roots) != 1 || got.Scan.Roots[0] != s.Scan.Roots[0] {
		t.Errorf("roots = %q, want %q", got.Scan.Roots, s.Scan.Roots)
	}
	if len(got.Scan.Ignore) != 1 || got.Scan.Ignore[0] != s.Scan.Ignore[0] {
		t.Errorf("ignore = %q, want %q", got.Scan.Ignore, s.Scan.Ignore)
	}
}

func TestConfigReparsesEditedLists(t *testing.T) {
	s := core.DefaultSettings()
	s.Scan.Roots = []string{"/a"}
	var m tea.Model = newConfigModel(s)
	m = focusField(t, m, "roots")
	m = typeRunes(m, " /b")

	got := m.(configModel).settings().Scan.Roots
	if len(got) != 2 || got[0] != "/a" || got[1] != "/b" {
		t.Errorf("roots after an edit = %q, want [/a /b]", got)
	}
}

func TestConfigEditCanDeleteARoot(t *testing.T) {
	s := core.DefaultSettings()
	s.Scan.Roots = []string{"/a", "/b"}
	var m tea.Model = newConfigModel(s)
	m = focusField(t, m, "roots")
	for range 3 { // "/a /b" -> "/a"
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	}

	got := m.(configModel).settings().Scan.Roots
	if len(got) != 1 || got[0] != "/a" {
		t.Errorf("roots after deleting one = %q, want [/a]", got)
	}
}

// A language hop does not ship still belongs to the user: the editor must show
// it and hand it back, not silently rewrite it to the first option.
func TestConfigKeepsUnknownLanguage(t *testing.T) {
	s := core.DefaultSettings()
	s.UI.Language = "de"
	m := newConfigModel(s)

	if got := m.settings().UI.Language; got != "de" {
		t.Errorf("language = %q, want de", got)
	}
	if view := stripANSI(m.View()); !strings.Contains(view, "auto en fr es pt [de]") {
		t.Errorf("language row does not show de as current:\n%s", view)
	}
}

func TestConfigSelectOptionsUnchangedForKnownValues(t *testing.T) {
	s := core.DefaultSettings()
	s.UI.Language, s.UI.Theme, s.Hub.ActionAccess = "fr", "dark", "enter"
	m := newConfigModel(s)

	want := map[string][]string{
		"language":      {"auto", "en", "fr", "es", "pt"},
		"theme":         {"auto", "light", "dark"},
		"action_access": {"tab", "shift", "enter"},
	}
	for _, f := range m.fields {
		if opts, ok := want[f.id]; ok && !slices.Equal(f.options, opts) {
			t.Errorf("%s options = %q, want %q", f.id, f.options, opts)
		}
	}
	if got := m.settings(); got.UI.Language != "fr" || got.UI.Theme != "dark" || got.Hub.ActionAccess != "enter" {
		t.Errorf("known values changed: %+v", got)
	}
}

func TestConfigCyclingLanguageReachesEveryShippedLanguage(t *testing.T) {
	var m tea.Model = newConfigModel(core.DefaultSettings())
	m = focusField(t, m, "language")

	seen := map[string]bool{}
	for range 5 {
		cm := m.(configModel)
		seen[cm.settings().UI.Language] = true
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	}
	for _, lang := range []string{"auto", "en", "fr", "es", "pt"} {
		if !seen[lang] {
			t.Errorf("cycling never reached %q (saw %v)", lang, seen)
		}
	}
}

// Update takes a value receiver, so a keystroke must land in the model it
// returns and nowhere else — the caller's copy included.
func TestConfigUpdateLeavesTheCallersModelAlone(t *testing.T) {
	m0 := newConfigModel(core.DefaultSettings()) // cursor on "command", value "p"
	edited := typeRunes(m0, "XYZ")

	if got := m0.settings().Shell.Command; got != "p" {
		t.Errorf("caller's model changed: command = %q, want p", got)
	}
	if got := edited.(configModel).settings().Shell.Command; got != "pXYZ" {
		t.Errorf("returned model: command = %q, want pXYZ", got)
	}
}

func TestConfigUpdateLeavesTheCallersSelectAlone(t *testing.T) {
	m0 := newConfigModel(core.DefaultSettings())
	focused := focusField(t, m0, "action_access")
	edited, _ := focused.Update(tea.KeyMsg{Type: tea.KeyRight})

	if got := m0.settings().Hub.ActionAccess; got != "tab" {
		t.Errorf("caller's model changed: access = %q, want tab", got)
	}
	if got := edited.(configModel).settings().Hub.ActionAccess; got != "shift" {
		t.Errorf("returned model: access = %q, want shift", got)
	}
}

func TestConfigEveryFieldStaysEditable(t *testing.T) {
	ids := newConfigModel(core.DefaultSettings()).fields
	if len(ids) != 9 {
		t.Fatalf("editor has %d fields, want 9", len(ids))
	}
	for _, f := range ids {
		t.Run(f.id, func(t *testing.T) {
			var m tea.Model = newConfigModel(core.DefaultSettings())
			m = focusField(t, m, f.id)
			before := m.(configModel).fields[m.(configModel).cursor]
			if f.kind == fieldSelect {
				m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
			} else {
				m = typeRunes(m, "x")
			}
			after := m.(configModel).fields[m.(configModel).cursor]
			if after.value == before.value && after.idx == before.idx {
				t.Errorf("a keystroke left %s unchanged (%+v)", f.id, after)
			}
		})
	}
}

// clearField empties the text field under the cursor, one backspace at a time.
func clearField(t *testing.T, m tea.Model) tea.Model {
	t.Helper()
	cm := m.(configModel)
	for range []rune(cm.fields[cm.cursor].value) {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	}
	cm = m.(configModel)
	if got := cm.fields[cm.cursor].value; got != "" {
		t.Fatalf("field %q still holds %q", cm.fields[cm.cursor].id, got)
	}
	return m
}

// A save hop cannot honour must be refused, not absorbed. max_depth used to be
// parsed with a silent fallback, so a typo ("4O" for "40") left the depth
// exactly as it was while hop printed "config saved"; and a value core rejects
// (0, -3) was written to config.toml and replaced at the next load, leaving the
// file and the behaviour disagreeing. A cleared required field was saved empty
// and coerced back the same way.
func TestConfigRefusesSavesItCannotHonour(t *testing.T) {
	i18n.SetLanguage("en")
	cases := []struct {
		field, typed string
		wantSaved    bool
	}{
		{"max_depth", "abc", false},
		{"max_depth", "4O", false}, // the typo for 40
		{"max_depth", "", false},
		{"max_depth", "0", false},  // core would turn it back into 7
		{"max_depth", "-3", false}, // same
		{"max_depth", "4", true},
		{"max_depth", " 5 ", true}, // stray spaces are the user being human
		{"command", "", false},
		{"command", "hop", true},
		{"editor", "", false},
		{"editor", "nvim", true},
		{"roots", "", false},
		{"roots", "  ", false},
		{"roots", "~/code", true},
		{"ignore", "", true}, // ignoring nothing is a legitimate choice
	}
	for _, tc := range cases {
		t.Run(tc.field+"="+tc.typed, func(t *testing.T) {
			var m tea.Model = newConfigModel(core.DefaultSettings())
			m = focusField(t, m, tc.field)
			m = clearField(t, m)
			m = typeRunes(m, tc.typed)
			m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})

			cm := m.(configModel)
			if cm.saved != tc.wantSaved {
				t.Errorf("%s = %q: saved = %v, want %v", tc.field, tc.typed, cm.saved, tc.wantSaved)
			}
			if tc.wantSaved && cmd == nil {
				t.Error("a save must quit the editor")
			}
			if !tc.wantSaved && cmd != nil {
				t.Error("a refusal must leave the editor open so the value can be fixed")
			}
		})
	}
}

// A silent refusal is the same defect with the sign flipped: the user has to be
// told which field hop could not accept.
func TestConfigShowsWhichFieldItRefused(t *testing.T) {
	i18n.SetLanguage("en")
	var m tea.Model = newConfigModel(core.DefaultSettings()) // max_depth = 7
	m = focusField(t, m, "max_depth")
	m = clearField(t, m)
	m = typeRunes(m, "abc")
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	cm := m.(configModel)
	if got := cm.settings().Scan.MaxDepth; got != 7 {
		t.Errorf("depth = %d, want the untouched 7", got)
	}
	view := stripANSI(cm.View())
	if !strings.Contains(view, "whole number") {
		t.Errorf("the refusal must name what hop expected, got:\n%s", view)
	}
	// Fixing the value clears the marker and lets the save through.
	m = clearField(t, m)
	m = typeRunes(m, "4")
	if view := stripANSI(m.(configModel).View()); strings.Contains(view, "whole number") {
		t.Errorf("the marker must go once the field is being fixed, got:\n%s", view)
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	cm = m.(configModel)
	if !cm.saved {
		t.Fatal("a corrected depth must save")
	}
	if got := cm.settings().Scan.MaxDepth; got != 4 {
		t.Errorf("depth = %d, want 4", got)
	}
}

func TestConfigEscDoesNotSave(t *testing.T) {
	var m tea.Model = newConfigModel(core.DefaultSettings())
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.(configModel).saved {
		t.Fatal("esc should not save")
	}
}
