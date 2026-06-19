package core

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestParseWorktreePaths(t *testing.T) {
	out := "worktree /home/u/main\nHEAD abc\nbranch refs/heads/main\n\n" +
		"worktree /home/u/main-feature\nHEAD def\nbranch refs/heads/feature\n\n" +
		"worktree /home/u/detached\nHEAD 123\ndetached\n"
	got := parseWorktreePaths(out)
	want := []string{"/home/u/main", "/home/u/main-feature", "/home/u/detached"}
	if len(got) != len(want) {
		t.Fatalf("got %d paths, want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("path[%d] = %q, want %q", i, got[i], want[i])
		}
	}
	if parseWorktreePaths("") != nil {
		t.Error("empty output should yield no paths")
	}
}

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	full := append([]string{"-C", dir,
		"-c", "user.email=t@example.com", "-c", "user.name=t", "-c", "commit.gpgsign=false"}, args...)
	if out, err := exec.Command("git", full...).CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// initRepoWithCommit makes dir a git repo with one commit (a worktree needs a
// commit to branch from).
func initRepoWithCommit(t *testing.T, dir string) {
	t.Helper()
	if err := exec.Command("git", "init", "-q", dir).Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-q", "-m", "init")
}

func indexHasBase(idx *Index, base string) (Project, bool) {
	for _, p := range idx.Projects {
		if filepath.Base(p.Path) == base {
			return p, true
		}
	}
	return Project{}, false
}

func TestExpandWorktreesIndexesExternalOnlyWhenEnabled(t *testing.T) {
	requireGit(t)
	t.Setenv("XDG_STATE_HOME", t.TempDir()) // isolate LoadExtras inside BuildIndex

	base := t.TempDir()
	root := filepath.Join(base, "root")
	repo := filepath.Join(root, "main-repo")
	initRepoWithCommit(t, repo)

	external := filepath.Join(base, "main-repo-feature") // lives OUTSIDE the root
	runGit(t, repo, "worktree", "add", "-q", "-b", "feature", external)

	cfg := Config{Roots: []string{root}, MaxDepth: 7}

	off := BuildIndex(cfg)
	if _, ok := indexHasBase(off, "main-repo"); !ok {
		t.Fatal("the main repo should always be indexed")
	}
	if _, ok := indexHasBase(off, "main-repo-feature"); ok {
		t.Error("an external worktree must NOT be indexed when worktrees=false")
	}

	cfg.Worktrees = true
	on := BuildIndex(cfg)
	wt, ok := indexHasBase(on, "main-repo-feature")
	if !ok {
		t.Fatal("the external worktree should be indexed when worktrees=true")
	}
	if wt.Category != "worktree" {
		t.Errorf("worktree category = %q, want %q", wt.Category, "worktree")
	}
}

func TestExpandWorktreesDedupsInRootWorktree(t *testing.T) {
	requireGit(t)
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	base := t.TempDir()
	root := filepath.Join(base, "root")
	repo := filepath.Join(root, "main-repo")
	initRepoWithCommit(t, repo)

	// A worktree that lives UNDER the root: the scanner already finds it, so the
	// expansion must not add it a second time.
	internal := filepath.Join(root, "main-repo-inroot")
	runGit(t, repo, "worktree", "add", "-q", "-b", "inroot", internal)

	cfg := Config{Roots: []string{root}, MaxDepth: 7, Worktrees: true}
	idx := BuildIndex(cfg)

	count := 0
	want := CanonicalDir(internal)
	for _, p := range idx.Projects {
		if p.Path == want {
			count++
		}
	}
	if count != 1 {
		t.Errorf("in-root worktree indexed %d times, want exactly 1", count)
	}
}

func BenchmarkExpandWorktrees(b *testing.B) {
	if _, err := exec.LookPath("git"); err != nil {
		b.Skip("git not available")
	}
	base := b.TempDir()
	repo := filepath.Join(base, "repo")
	if err := exec.Command("git", "init", "-q", repo).Run(); err != nil {
		b.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "f"), []byte("x\n"), 0o644); err != nil {
		b.Fatal(err)
	}
	for _, a := range [][]string{{"add", "."}, {"-c", "user.email=t@e", "-c", "user.name=t", "commit", "-q", "-m", "i"}} {
		if err := exec.Command("git", append([]string{"-C", repo}, a...)...).Run(); err != nil {
			b.Fatal(err)
		}
	}
	_ = exec.Command("git", "-C", repo, "worktree", "add", "-q", "-b", "f", filepath.Join(base, "wt")).Run()
	scanned := []Project{{Name: "repo", Path: CanonicalDir(repo)}}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = expandWorktrees(scanned)
	}
}
