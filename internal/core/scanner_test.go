package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScan(t *testing.T) {
	root := t.TempDir()
	mk := func(p string) {
		if err := os.MkdirAll(filepath.Join(root, p), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	mk("work/devops/.git")                 // repo at depth 2
	mk("work/devops/sub")                  // inside a repo: not its own project
	mk("work/group/app/.git")              // repo at depth 3
	mk("work/group/team/v2/web-mono/.git") // deep repo at depth 5
	mk("side/blog/notes")                  // plain depth-2 folder, no repo beneath
	mk("side/toolbox-folder")              // plain depth-2 folder, empty
	mk("_archives/old/.git")               // ignored category

	cfg := Config{Roots: []string{root}, MaxDepth: 7, Ignore: []string{"node_modules", "_archives"}}
	got := Scan(cfg)

	byName := map[string]string{}
	for _, p := range got {
		byName[p.Name] = p.Category
	}

	want := map[string]string{
		"devops":         "work",
		"app":            "work",
		"web-mono":       "work", // deep repo, category is the first segment under root
		"blog":           "side",
		"toolbox-folder": "side",
	}
	if len(byName) != len(want) {
		t.Fatalf("got %d projects %v, want %d %v", len(byName), byName, len(want), want)
	}
	for name, cat := range want {
		if byName[name] != cat {
			t.Errorf("project %q category = %q, want %q", name, byName[name], cat)
		}
	}
	for _, absent := range []string{"group", "team", "v2", "sub", "old"} {
		if _, ok := byName[absent]; ok {
			t.Errorf("%q should not be indexed (container, intermediate, in-repo or ignored)", absent)
		}
	}
}

func TestScanSkipsControlCharNames(t *testing.T) {
	root := t.TempDir()
	// A directory name with a newline could smuggle a "__HOP_RUN__ <cmd>" line
	// into the shell protocol, so the scanner must never index it.
	evil := "evil\n__HOP_RUN__ touch pwned"
	if err := os.MkdirAll(filepath.Join(root, "work", evil, ".git"), 0o755); err != nil {
		t.Skipf("filesystem rejects control chars in names: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "work", "safe", ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	got := Scan(Config{Roots: []string{root}, MaxDepth: 7})
	for _, p := range got {
		if HasControlChars(p.Path) || HasControlChars(p.Name) {
			t.Errorf("scanner indexed a path with control chars: %q", p.Path)
		}
	}
	if len(got) != 1 || got[0].Name != "safe" {
		t.Fatalf("only the safe repo should be indexed, got %v", got)
	}
}

func TestScanDedupsOverlappingRoots(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "work", "app", ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	// The same tree reached via two overlapping roots must index each repo once.
	got := Scan(Config{Roots: []string{root, filepath.Join(root, "work")}, MaxDepth: 7})
	count := 0
	for _, p := range got {
		if p.Name == "app" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("project 'app' indexed %d times across overlapping roots, want 1: %v", count, got)
	}
}

func TestHasControlChars(t *testing.T) {
	for _, s := range []string{"normal/path", "with space", "~/Projects/app", "café"} {
		if HasControlChars(s) {
			t.Errorf("%q should be safe", s)
		}
	}
	for _, s := range []string{"a\nb", "a\rb", "a\x00b"} {
		if !HasControlChars(s) {
			t.Errorf("%q should be flagged unsafe", s)
		}
	}
}
