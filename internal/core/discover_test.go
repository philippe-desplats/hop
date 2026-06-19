package core

import (
	"os"
	"path/filepath"
	"testing"
)

func mkRepo(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o750); err != nil {
		t.Fatal(err)
	}
}

func TestDiscoverRoots(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	mkRepo(t, filepath.Join(home, "code", "repo1"))          // depth 1 repo
	mkRepo(t, filepath.Join(home, "code", "group", "repo2")) // depth 2 repo
	if err := os.MkdirAll(filepath.Join(home, "work"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(home, "Music"), 0o750); err != nil { // not a candidate
		t.Fatal(err)
	}

	got := DiscoverRoots()
	if len(got) != 2 {
		t.Fatalf("got %d candidates %+v, want 2 (code, work)", len(got), got)
	}
	if got[0].Path != "~/code" || got[0].Repos != 2 {
		t.Errorf("first = %+v, want ~/code with 2 repos", got[0])
	}
	if got[1].Path != "~/work" || got[1].Repos != 0 {
		t.Errorf("second = %+v, want ~/work with 0 repos", got[1])
	}
}

func TestDiscoverRootsFindsCustomNamesByContent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	mkRepo(t, filepath.Join(home, "my-stuff", "proj"))    // custom name, has a repo
	mkRepo(t, filepath.Join(home, "Downloads", "cloned")) // denylisted location
	if err := os.MkdirAll(filepath.Join(home, "scratch"), 0o750); err != nil {
		t.Fatal(err) // a repo-less custom dir
	}

	paths := map[string]int{}
	for _, c := range DiscoverRoots() {
		paths[c.Path] = c.Repos
	}
	if paths["~/my-stuff"] != 1 {
		t.Errorf("expected ~/my-stuff found by content with 1 repo, got %v", paths)
	}
	if _, ok := paths["~/Downloads"]; ok {
		t.Error("~/Downloads must be excluded (denylisted)")
	}
	if _, ok := paths["~/scratch"]; ok {
		t.Error("a repo-less custom dir must not be proposed")
	}
}

func TestCountReposIgnoresDependencyDirs(t *testing.T) {
	root := t.TempDir()
	mkRepo(t, filepath.Join(root, "node_modules", "pkg")) // must NOT count
	mkRepo(t, filepath.Join(root, "real"))                // counts
	if n := countRepos(root, 0); n != 1 {
		t.Fatalf("countRepos = %d, want 1 (node_modules ignored)", n)
	}
}

func TestCountReposStopsAtRepoAndDepth(t *testing.T) {
	root := t.TempDir()
	mkRepo(t, filepath.Join(root, "a"))                  // depth 0 repo, a leaf
	mkRepo(t, filepath.Join(root, "a", "nested"))        // inside a repo: must NOT be counted
	mkRepo(t, filepath.Join(root, "b", "c", "deep"))     // depth 2 repo, within bound
	mkRepo(t, filepath.Join(root, "x", "y", "z", "far")) // depth 3, beyond discoverMaxDepth
	if n := countRepos(root, 0); n != 2 {
		t.Fatalf("countRepos = %d, want 2 (a and b/c/deep)", n)
	}
}

func TestHomeRelative(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cases := map[string]string{
		home:                          "~",
		filepath.Join(home, "code"):   "~/code",
		filepath.Join(home, "a", "b"): "~/a/b",
		"/etc/hosts":                  "/etc/hosts",
	}
	for in, want := range cases {
		if got := HomeRelative(in); got != want {
			t.Errorf("HomeRelative(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestHasIndex(t *testing.T) {
	state := t.TempDir()
	t.Setenv("XDG_STATE_HOME", state)
	if HasIndex() {
		t.Fatal("HasIndex should be false before any scan")
	}
	if err := os.MkdirAll(StateDir(), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(IndexPath(), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !HasIndex() {
		t.Fatal("HasIndex should be true once the index file exists")
	}
}
