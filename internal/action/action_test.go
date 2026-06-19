package action

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/philippe-desplats/hop/internal/core"
)

func TestInvalidCustomActions(t *testing.T) {
	custom := []core.CustomAction{
		{Key: "y", Label: "ok", Command: "cursor {path}"}, // valid
		{Key: "", Command: "x"},                           // no key
		{Key: "n", Command: ""},                           // no command
		{Key: "z", Command: "x"},                          // reserved built-in key
	}
	bad := InvalidCustomActions(custom)
	if len(bad) != 3 {
		t.Fatalf("expected 3 invalid entries reported, got %d: %v", len(bad), bad)
	}
	if InvalidCustomActions(nil) != nil {
		t.Error("no custom actions should report nothing")
	}
}

func TestActionOutcomes(t *testing.T) {
	p := core.Project{Name: "web-app", Path: "/p/x"}
	opts := Options{AI: Assistant{Name: "claude", Run: []string{"claude"}, Resume: []string{"claude", "--resume"}}, HasAI: true}
	want := map[string]Outcome{
		"enter": {Cd: "/p/x"},
		"c":     {Cd: "/p/x", Run: "claude"},
		"r":     {Cd: "/p/x", Run: "claude --resume"},
	}
	for key, exp := range want {
		s, ok := ByKey(key, p, opts)
		if !ok {
			t.Fatalf("action %q not found", key)
		}
		if got := s.Do(p); got != exp {
			t.Errorf("%q Do = %+v, want %+v", key, got, exp)
		}
	}
}

func TestTmuxSessionSanitized(t *testing.T) {
	t.Setenv("TMUX", "") // resolve the outside-a-session branch deterministically
	p := core.Project{Name: "we!rd name.v2", Path: "/p/x"}
	s, ok := ByKey("t", p, Options{Multiplexer: "tmux"})
	if !ok {
		t.Fatal("t action should be present when a multiplexer is set")
	}
	out := s.Do(p)
	const prefix = "tmux new-session -A -s "
	if !strings.HasPrefix(out.Run, prefix) {
		t.Fatalf("unexpected run: %q", out.Run)
	}
	rest := strings.TrimPrefix(out.Run, prefix)
	name := strings.SplitN(rest, " ", 2)[0]
	if strings.ContainsAny(name, " !.") {
		t.Errorf("session name not sanitized: %q", name)
	}
}

func TestGitActionsNeedRepo(t *testing.T) {
	dir := t.TempDir()
	p := core.Project{Name: "x", Path: dir}
	if _, ok := ByKey("g", p, Options{}); ok {
		t.Error("g should be unavailable without .git")
	}
	if _, ok := ByKey("o", p, Options{}); ok {
		t.Error("o should be unavailable without .git")
	}
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, ok := ByKey("g", p, Options{}); !ok {
		t.Error("g should be available once .git exists")
	}
}
