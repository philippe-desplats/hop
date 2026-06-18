package action

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/philippe-desplats/hop/internal/core"
)

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
	p := core.Project{Name: "we!rd name.v2", Path: "/p/x"}
	s, _ := ByKey("t", p, Options{ShowTmux: true})
	out := s.Do(p)
	const prefix = "tmux new-session -A -s "
	if !strings.HasPrefix(out.Run, prefix) {
		t.Fatalf("unexpected run: %q", out.Run)
	}
	if name := strings.TrimPrefix(out.Run, prefix); strings.ContainsAny(name, " !.") {
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
