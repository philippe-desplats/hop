package action

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/philippe-desplats/hop/internal/core"
)

func stubLookPath(t *testing.T, present ...string) {
	t.Helper()
	orig := lookPath
	t.Cleanup(func() { lookPath = orig })
	set := map[string]bool{}
	for _, p := range present {
		set[p] = true
	}
	lookPath = func(bin string) (string, error) {
		if set[bin] {
			return "/usr/bin/" + bin, nil
		}
		return "", os.ErrNotExist
	}
}

func TestResolveAssistant(t *testing.T) {
	t.Run("auto picks first installed in preference order", func(t *testing.T) {
		stubLookPath(t, "claude", "codex")
		if a, ok := ResolveAssistant("auto"); !ok || a.Name != "claude" {
			t.Fatalf("auto = %q/%v, want claude", a.Name, ok)
		}
	})
	t.Run("explicit tool wins", func(t *testing.T) {
		stubLookPath(t, "claude", "codex")
		if a, ok := ResolveAssistant("codex"); !ok || a.Name != "codex" {
			t.Fatalf("explicit codex = %q/%v", a.Name, ok)
		}
	})
	t.Run("explicit but missing falls back to auto", func(t *testing.T) {
		stubLookPath(t, "codex") // aider not present
		if a, ok := ResolveAssistant("aider"); !ok || a.Name != "codex" {
			t.Fatalf("missing aider should fall back to codex, got %q/%v", a.Name, ok)
		}
	})
	t.Run("none installed", func(t *testing.T) {
		stubLookPath(t) // nothing
		if a, ok := ResolveAssistant("auto"); ok {
			t.Fatalf("none installed should be ok=false, got %q", a.Name)
		}
	})
}

func TestNoAssistantHidesCR(t *testing.T) {
	p := core.Project{Name: "x", Path: "/p/x"}
	if _, ok := ByKey("c", p, Options{HasAI: false}); ok {
		t.Error("c must be hidden without an assistant")
	}
	if _, ok := ByKey("r", p, Options{HasAI: false}); ok {
		t.Error("r must be hidden without an assistant")
	}
}

func TestResumeHiddenWithoutResume(t *testing.T) {
	p := core.Project{Name: "x", Path: "/p/x"}
	opts := Options{AI: Assistant{Name: "aider", Run: []string{"aider"}}, HasAI: true} // no Resume
	if _, ok := ByKey("c", p, opts); !ok {
		t.Error("c should exist for aider")
	}
	if _, ok := ByKey("r", p, opts); ok {
		t.Error("r must be hidden when the assistant has no resume")
	}
}

func TestCustomActionInTerminalQuotesSpaces(t *testing.T) {
	p := core.Project{Name: "my app", Path: "/p/my project"}
	opts := Options{Custom: []core.CustomAction{{Key: "y", Label: "open", Command: "cursor {path}", InTerminal: true}}}
	s, ok := ByKey("y", p, opts)
	if !ok {
		t.Fatal("custom action y not found")
	}
	out := s.Do(p)
	if want := "cursor '/p/my project'"; out.Run != want {
		t.Errorf("Run = %q, want %q", out.Run, want)
	}
	if out.Cd != p.Path {
		t.Errorf("Cd = %q, want %q", out.Cd, p.Path)
	}
}

func TestCustomActionDetachedSetsNoRun(t *testing.T) {
	p := core.Project{Name: "x", Path: "/p/x"}
	opts := Options{Custom: []core.CustomAction{{Key: "y", Label: "open", Command: "true {path}", InTerminal: false}}}
	s, ok := ByKey("y", p, opts)
	if !ok {
		t.Fatal("custom action y not found")
	}
	out := s.Do(p) // launches "true /p/x" detached, harmless
	if out.Run != "" {
		t.Errorf("detached action must not set Run, got %q", out.Run)
	}
	if out.Cd != p.Path {
		t.Errorf("Cd = %q, want %q", out.Cd, p.Path)
	}
}

func TestCustomActionReservedKeyRejected(t *testing.T) {
	p := core.Project{Name: "x", Path: "/p/x"}
	opts := Options{Custom: []core.CustomAction{{Key: "c", Label: "nope", Command: "true"}}}
	if _, ok := ByKey("c", p, opts); ok {
		t.Error("custom action on reserved key 'c' must be rejected")
	}
}

func TestCustomActionEmptyCommandSkipped(t *testing.T) {
	p := core.Project{Name: "x", Path: "/p/x"}
	opts := Options{Custom: []core.CustomAction{{Key: "y", Label: "x", Command: ""}}}
	if _, ok := ByKey("y", p, opts); ok {
		t.Error("custom action with empty command must be skipped")
	}
}

func TestCustomActionNeedsGit(t *testing.T) {
	dir := t.TempDir()
	p := core.Project{Name: "x", Path: dir}
	opts := Options{Custom: []core.CustomAction{{Key: "y", Label: "g", Command: "true", NeedsGit: true}}}
	if _, ok := ByKey("y", p, opts); ok {
		t.Error("needs_git custom action must be hidden outside a git repo")
	}
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o750); err != nil {
		t.Fatal(err)
	}
	if _, ok := ByKey("y", p, opts); !ok {
		t.Error("needs_git custom action must appear once .git exists")
	}
}

func TestShellQuote(t *testing.T) {
	cases := map[string]string{
		"/p/x":          "'/p/x'",
		"/p/my project": "'/p/my project'",
		"a'b":           `'a'\''b'`,
	}
	for in, want := range cases {
		if got := shellQuote(in); got != want {
			t.Errorf("shellQuote(%q) = %q, want %q", in, got, want)
		}
	}
}
