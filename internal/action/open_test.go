package action

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/philippe-desplats/hop/internal/core"
)

func TestOpenerMatchesPlatform(t *testing.T) {
	want := "xdg-open"
	if runtime.GOOS == "darwin" {
		want = "open"
	}
	if got := opener(); got != want {
		t.Fatalf("opener() = %q, want %q for GOOS=%s", got, want, runtime.GOOS)
	}
}

func TestFinderAndRemoteRequireOpener(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o750); err != nil {
		t.Fatal(err)
	}
	p := core.Project{Name: "x", Path: dir}

	t.Run("hidden when no opener on PATH", func(t *testing.T) {
		stubLookPath(t) // nothing installed, including the opener
		if _, ok := ByKey("f", p, Options{}); ok {
			t.Error("file-manager action must be hidden without an opener")
		}
		if _, ok := ByKey("o", p, Options{}); ok {
			t.Error("remote action must be hidden without an opener")
		}
	})

	t.Run("shown when the opener exists", func(t *testing.T) {
		stubLookPath(t, opener())
		if _, ok := ByKey("f", p, Options{}); !ok {
			t.Error("file-manager action must appear when the opener exists")
		}
		if _, ok := ByKey("o", p, Options{}); !ok {
			t.Error("remote action must appear when the opener exists in a git repo")
		}
	})
}
