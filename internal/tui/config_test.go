package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/philippe-desplats/hop/internal/core"
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

func TestConfigEscDoesNotSave(t *testing.T) {
	var m tea.Model = newConfigModel(core.DefaultSettings())
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.(configModel).saved {
		t.Fatal("esc should not save")
	}
}
