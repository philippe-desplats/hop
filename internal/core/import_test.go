package core

import (
	"path/filepath"
	"testing"
	"time"
)

func TestParseZoxideLine(t *testing.T) {
	cases := []struct {
		name      string
		in        string
		wantPath  string
		wantScore float64
		wantOK    bool
	}{
		{"score first", "123.45 /home/user/foo", "/home/user/foo", 123.45, true},
		{"leading spaces", "   12.00 /home/user/bar", "/home/user/bar", 12, true},
		{"path with spaces", "5.0 /home/user/My Projects/app", "/home/user/My Projects/app", 5, true},
		{"tab separated", "9\t/srv/repo", "/srv/repo", 9, true},
		{"blank line", "", "", 0, false},
		{"no path", "42", "", 0, false},
		{"non-numeric score", "abc /home/user/foo", "", 0, false},
		{"only spaces", "   ", "", 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path, score, ok := parseZoxideLine(tc.in)
			if ok != tc.wantOK || path != tc.wantPath || score != tc.wantScore {
				t.Errorf("parseZoxideLine(%q) = (%q, %v, %v), want (%q, %v, %v)",
					tc.in, path, score, ok, tc.wantPath, tc.wantScore, tc.wantOK)
			}
		})
	}
}

func TestParseZoxideOutputSkipsJunk(t *testing.T) {
	out := "100 /a\ngarbage line\n50 /b\n\n  3.5 /c\n"
	entries := ParseZoxideOutput(out)
	if len(entries) != 3 {
		t.Fatalf("got %d entries, want 3: %+v", len(entries), entries)
	}
	if entries[0].Path != "/a" || entries[2].Score != 3.5 {
		t.Errorf("unexpected parse: %+v", entries)
	}
}

// canonRepo creates dir with a .git marker (so isRepo reports true) and returns
// its canonical path, matching what ImportZoxide computes internally.
func canonRepo(t *testing.T, dir string) string {
	t.Helper()
	mkRepo(t, dir)
	return CanonicalDir(dir)
}

func TestImportZoxideClassifiesEntries(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	now := time.Now()

	repo := canonRepo(t, filepath.Join(t.TempDir(), "newrepo"))
	indexedProj := CanonicalDir(t.TempDir()) // already known to hop
	plain := CanonicalDir(t.TempDir())       // exists but not a repo
	missing := filepath.Join(t.TempDir(), "gone")

	entries := []ZoxideEntry{
		{Path: repo, Score: 80},
		{Path: indexedProj, Score: 40},
		{Path: plain, Score: 30},
		{Path: missing, Score: 99},
	}
	indexed := map[string]bool{indexedProj: true}

	imported, tracked, skipped, err := ImportZoxide(entries, indexed, now, false)
	if err != nil {
		t.Fatal(err)
	}
	if imported != 2 {
		t.Errorf("imported = %d, want 2 (repo + indexed)", imported)
	}
	if tracked != 1 {
		t.Errorf("tracked = %d, want 1 (only the new repo)", tracked)
	}
	if skipped != 2 {
		t.Errorf("skipped = %d, want 2 (non-repo + missing)", skipped)
	}

	// The git repo joined the tracked-folders store.
	extras := LoadExtras()
	if len(extras.Paths) != 1 || extras.Paths[0] != repo {
		t.Errorf("extras = %v, want [%s]", extras.Paths, repo)
	}
	// Frecency seeded for repo and the indexed project, not for the skipped ones.
	f := LoadFrecency()
	if _, ok := f.Entries[repo]; !ok {
		t.Error("repo should have seeded frecency")
	}
	if _, ok := f.Entries[indexedProj]; !ok {
		t.Error("indexed project should have seeded frecency")
	}
	if _, ok := f.Entries[plain]; ok {
		t.Error("a non-repo must not be seeded")
	}
	if got := f.Entries[repo].Rank; got != 80 {
		t.Errorf("repo seeded rank = %v, want 80", got)
	}
}

func TestImportZoxideDryRunWritesNothing(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	now := time.Now()
	repo := canonRepo(t, filepath.Join(t.TempDir(), "repo"))

	imported, tracked, skipped, err := ImportZoxide([]ZoxideEntry{{Path: repo, Score: 10}}, map[string]bool{}, now, true)
	if err != nil {
		t.Fatal(err)
	}
	if imported != 1 || tracked != 1 || skipped != 0 {
		t.Errorf("dry-run counts = (%d, %d, %d), want (1, 1, 0)", imported, tracked, skipped)
	}
	if len(LoadExtras().Paths) != 0 {
		t.Error("dry-run must not write to extras")
	}
	if len(LoadFrecency().Entries) != 0 {
		t.Error("dry-run must not write to frecency")
	}
}

func TestImportZoxideNeverLowersExistingRank(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	now := time.Now()
	repo := canonRepo(t, filepath.Join(t.TempDir(), "repo"))
	// hop already learned a high rank; a low zoxide score must not clobber it.
	if err := SeedFrecency(map[string]float64{repo: 500}, now); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := ImportZoxide([]ZoxideEntry{{Path: repo, Score: 3}}, map[string]bool{repo: true}, now, false); err != nil {
		t.Fatal(err)
	}
	if got := LoadFrecency().Entries[repo].Rank; got != 500 {
		t.Errorf("rank = %v, want 500 (import must not lower it)", got)
	}
}

func TestImportZoxideSeedsAtLeastOne(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	now := time.Now()
	repo := canonRepo(t, filepath.Join(t.TempDir(), "repo"))
	if _, _, _, err := ImportZoxide([]ZoxideEntry{{Path: repo, Score: 0.2}}, map[string]bool{repo: true}, now, false); err != nil {
		t.Fatal(err)
	}
	if got := LoadFrecency().Entries[repo].Rank; got != 1 {
		t.Errorf("tiny zoxide score seeded rank = %v, want floor of 1", got)
	}
}
