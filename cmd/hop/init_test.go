package main

import (
	"strings"
	"testing"
)

func TestZshIntegrationDefaultName(t *testing.T) {
	out := zshIntegration("p")
	for _, want := range []string{"unalias p 2>/dev/null", "function p {", "add-zsh-hook chpwd _hop_chpwd"} {
		if !strings.Contains(out, want) {
			t.Errorf("integration missing %q", want)
		}
	}
}

func TestZshIntegrationCustomName(t *testing.T) {
	out := zshIntegration("pp")
	if !strings.Contains(out, "function pp {") || !strings.Contains(out, "unalias pp 2>/dev/null") {
		t.Errorf("custom name not applied:\n%s", out)
	}
	if strings.Contains(out, "function p {") {
		t.Error("default name leaked into custom integration")
	}
	// Internal helpers stay fixed regardless of the public name.
	if !strings.Contains(out, "function _hop_chpwd {") {
		t.Error("internal hook name should not change")
	}
}

func TestCmdNameValidation(t *testing.T) {
	valid := []string{"p", "pp", "go_proj", "_x", "Hop2"}
	invalid := []string{"", "2p", "p p", "p-x", "p.x", "p/x"}
	for _, n := range valid {
		if !cmdNameRe.MatchString(n) {
			t.Errorf("%q should be valid", n)
		}
	}
	for _, n := range invalid {
		if cmdNameRe.MatchString(n) {
			t.Errorf("%q should be invalid", n)
		}
	}
}
