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
