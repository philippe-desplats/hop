package main

import (
	"strings"
	"testing"
)

func TestBashIntegration(t *testing.T) {
	out := bashIntegration("p")
	for _, want := range []string{
		"p() {",
		"__HOP_CD__ ",
		"__HOP_RUN__ ",
		"complete -F _hop_complete p",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("bash integration missing %q", want)
		}
	}
	// The frecency hook must append to PROMPT_COMMAND, never clobber it.
	if !strings.Contains(out, `PROMPT_COMMAND="_hop_record${PROMPT_COMMAND:+; $PROMPT_COMMAND}"`) {
		t.Error("bash hook should append to PROMPT_COMMAND")
	}
	// The background write must be wrapped in a subshell, else bash job control
	// prints "[1]+ Done ..." on the next prompt after every navigation.
	if !strings.Contains(out, `( command hop add "$PWD" >/dev/null 2>&1 & )`) {
		t.Error("bash frecency hook should background in a subshell to stay silent")
	}
	if pp := bashIntegration("pp"); !strings.Contains(pp, "unalias pp 2>/dev/null") || strings.Contains(pp, "unalias p 2>/dev/null") {
		t.Error("custom name not applied to bash integration")
	}
}

func TestZshIntegration(t *testing.T) {
	out := zshIntegration("p")
	for _, want := range []string{
		"function p {",
		"__HOP_CD__ ",
		"__HOP_RUN__ ",
		"add-zsh-hook chpwd _hop_chpwd",
		"_hop_complete()", // completion is wired (was missing before)
		"compdef _hop_complete p",
		"~/.zshrc", // help points at a file zsh actually sources
	} {
		if !strings.Contains(out, want) {
			t.Errorf("zsh integration missing %q", want)
		}
	}
	if strings.Contains(out, "~/.zsh_init") {
		t.Error("zsh integration must not reference the non-standard ~/.zsh_init")
	}
	if pp := zshIntegration("pp"); !strings.Contains(pp, "compdef _hop_complete pp") {
		t.Error("custom name not applied to zsh completion")
	}
}

func TestFishIntegration(t *testing.T) {
	out := fishIntegration("p")
	for _, want := range []string{
		"function p\n",
		"__HOP_CD__ ",
		"__HOP_RUN__ ",
		"--on-variable PWD",
		"complete -c p ",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("fish integration missing %q", want)
		}
	}
	if pp := fishIntegration("pp"); !strings.Contains(pp, "function pp\n") || strings.Contains(pp, "function p\n") {
		t.Error("custom name not applied to fish integration")
	}
}

func TestCompleteMatches(t *testing.T) {
	names := []string{"acme-api", "web-shop", "blog", "acme-web", "blog"}
	if got := completeMatches(names, ""); len(got) != 4 {
		t.Errorf("empty fragment should return 4 distinct names, got %v", got)
	}
	got := completeMatches(names, "acme")
	if len(got) != 2 || got[0] != "acme-api" || got[1] != "acme-web" {
		t.Errorf("fragment 'acme' = %v, want [acme-api acme-web]", got)
	}
	if got := completeMatches(names, "WEB"); len(got) != 2 {
		t.Errorf("match should be case-insensitive, got %v", got)
	}
	if got := completeMatches(names, "zzz"); len(got) != 0 {
		t.Errorf("no match should return empty, got %v", got)
	}
}
