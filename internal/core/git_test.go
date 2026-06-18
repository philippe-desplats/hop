package core

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestLoadGitInfoNonRepo(t *testing.T) {
	if got := LoadGitInfo(t.TempDir()); got.IsRepo {
		t.Errorf("a plain dir should not be a repo: %+v", got)
	}
}

func TestLoadGitInfoRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	env := append(os.Environ(),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	run := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = env
		if err := cmd.Run(); err != nil {
			t.Fatalf("git %v: %v", args, err)
		}
	}
	run("init", "-q", "-b", "main")
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-q", "-m", "initial thing")

	info := LoadGitInfo(dir)
	if !info.IsRepo || info.Branch != "main" || info.LastCommit != "initial thing" {
		t.Fatalf("got %+v", info)
	}
	if info.Dirty {
		t.Error("clean tree should not be dirty")
	}

	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("changed"), 0o644); err != nil {
		t.Fatal(err)
	}
	if info := LoadGitInfo(dir); !info.Dirty {
		t.Error("modified tree should be dirty")
	}
}
