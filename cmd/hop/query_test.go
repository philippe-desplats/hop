package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/philippe-desplats/hop/internal/core"
)

// encodeArgv and decodeArgv are the two ends of the argv channel used by the
// subprocess tests of this package (query and import): argv travels to the
// child as a JSON array in one environment variable, so an argument containing
// a space, or any argument boundary at all, survives the trip. Splitting the
// variable on whitespace in the child would make `hop query "my project"`
// inexpressible.
func encodeArgv(t *testing.T, argv []string) string {
	t.Helper()
	raw, err := json.Marshal(argv)
	if err != nil {
		t.Fatalf("encoding argv %q: %v", argv, err)
	}
	return string(raw)
}

// decodeArgv runs in the child, where t is not available: a malformed argv is a
// broken test harness, not a command failure, so it exits with a code no
// command under test uses.
func decodeArgv(raw string) []string {
	var argv []string
	if err := json.Unmarshal([]byte(raw), &argv); err != nil {
		fmt.Fprintf(os.Stderr, "child: cannot decode argv %q: %v\n", raw, err)
		os.Exit(99)
	}
	return argv
}

func TestFirstSafePathSkipsControlChars(t *testing.T) {
	matches := []core.Match{
		{Project: core.Project{Path: "/p/with\nnewline"}}, // unsafe: would smuggle a protocol line
		{Project: core.Project{Path: "/p/clean"}},
	}
	got, ok := firstSafePath(matches)
	if !ok || got != "/p/clean" {
		t.Errorf("firstSafePath = (%q, %v), want (/p/clean, true)", got, ok)
	}

	if _, ok := firstSafePath([]core.Match{{Project: core.Project{Path: "/p/bad\x00"}}}); ok {
		t.Error("a sole control-char path must be refused")
	}
	if _, ok := firstSafePath(nil); ok {
		t.Error("no matches must return ok=false")
	}
}

func TestSafePathsFiltersControlChars(t *testing.T) {
	matches := []core.Match{
		{Project: core.Project{Path: "/p/a"}},
		{Project: core.Project{Path: "/p/b\r"}},
		{Project: core.Project{Path: "/p/c"}},
	}
	got := safePaths(matches)
	if len(got) != 2 || got[0] != "/p/a" || got[1] != "/p/c" {
		t.Errorf("safePaths = %v, want [/p/a /p/c]", got)
	}
}

// TestCmdQuerySubprocess re-executes the test binary so cmdQuery's os.Exit paths
// run for real, against a scanned index of repos under a temp ~/Projects.
func TestCmdQuerySubprocess(t *testing.T) {
	if os.Getenv("HOP_QUERY_CHILD") == "1" {
		cmdQuery(decodeArgv(os.Getenv("HOP_QUERY_ARGV")))
		os.Exit(0) // exit before the test runner prints PASS to stdout
	}

	// "my project" exercises an argument containing a space. "my-app/project-x"
	// is a decoy: it is unreachable with the single keyword "my project", but it
	// outranks "my project" as soon as that argument is split into the two
	// keywords "my" and "project", so the space-carrying case below fails loudly
	// if the seam ever loses the argument boundaries again.
	projects := []string{"alpha", "beta", "my project", filepath.Join("my-app", "project-x")}
	home := t.TempDir()
	for _, name := range projects {
		if err := os.MkdirAll(filepath.Join(home, "Projects", name, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	run := func(t *testing.T, argv ...string) (string, int) {
		t.Helper()
		cmd := exec.Command(os.Args[0], "-test.run=TestCmdQuerySubprocess")
		cmd.Env = []string{
			"HOP_QUERY_CHILD=1",
			"HOP_QUERY_ARGV=" + encodeArgv(t, argv),
			"HOME=" + home,
			"XDG_STATE_HOME=" + filepath.Join(t.TempDir(), "state"),
			"XDG_CONFIG_HOME=" + t.TempDir(),
			"PATH=" + t.TempDir(),       // empty: query never shells out
			"GOCOVERDIR=" + t.TempDir(), // silence the cover runtime warning on a coverage build
		}
		out, err := cmd.CombinedOutput()
		code := 0
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else if err != nil {
			t.Fatalf("subprocess failed to run: %v", err)
		}
		return string(out), code
	}

	t.Run("resolves to a plain path, no sentinel", func(t *testing.T) {
		out, code := run(t, "alpha")
		if code != 0 {
			t.Fatalf("exit = %d, want 0\n%s", code, out)
		}
		line := strings.TrimSpace(out)
		if strings.Contains(line, "__HOP_CD__") {
			t.Errorf("query must print a plain path, got sentinel: %q", line)
		}
		if !strings.HasSuffix(line, filepath.Join("Projects", "alpha")) {
			t.Errorf("path = %q, want it to end in Projects/alpha", line)
		}
	})

	t.Run("a single keyword containing a space resolves", func(t *testing.T) {
		out, code := run(t, "my project")
		if code != 0 {
			t.Fatalf("exit = %d, want 0\n%s", code, out)
		}
		// The index stores canonical paths, so compare against a resolved home.
		root, err := filepath.EvalSymlinks(home)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := strings.TrimSpace(out), filepath.Join(root, "Projects", "my project"); got != want {
			t.Errorf("path = %q, want %q", got, want)
		}
	})

	t.Run("no match exits 1 with empty stdout", func(t *testing.T) {
		out, code := run(t, "zzznomatch")
		if code != 1 {
			t.Errorf("exit = %d, want 1 (no match)\n%s", code, out)
		}
		if strings.TrimSpace(out) != "" {
			t.Errorf("no-match stdout must be empty, got %q", out)
		}
	})

	t.Run("--list prints every project", func(t *testing.T) {
		out, code := run(t, "--list")
		if code != 0 {
			t.Fatalf("exit = %d, want 0\n%s", code, out)
		}
		lines := strings.Split(strings.TrimSpace(out), "\n")
		if len(lines) != len(projects) {
			t.Errorf("--list printed %d lines, want %d: %q", len(lines), len(projects), out)
		}
	})
}

// The --limit tests below drive the CLI through runHopCLI (help_test.go) rather
// than through the merged-stream helper above: three quarters of what they have
// to prove is "a message on stderr and an empty stdout", which a combined stream
// cannot show.

// limitProjects is the fixture for the cap: 4 projects, enough for a cap below,
// at and above the count.
var limitProjects = []string{"alpha", "beta", "gamma", "delta"}

// newQueryHome lays out an isolated home with one git repository per name under
// ~/Projects, the layout hop indexes, and pins the child's language so the
// assertions can quote the English catalog.
func newQueryHome(t *testing.T, names ...string) hopHome {
	t.Helper()
	h := newHopHome(t, englishConfig)
	for _, name := range names {
		if err := os.MkdirAll(filepath.Join(h.home, "Projects", name, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return h
}

// listLines splits a --list stdout into its paths, returning nothing for an
// empty stdout rather than one empty line.
func listLines(stdout string) []string {
	trimmed := strings.TrimSpace(stdout)
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "\n")
}

// TestQueryListLimitTruncates holds the cap to a truncation of the uncapped
// listing: the N lines printed are the first N lines of the same command run
// without --limit, so the flag can never reorder what --list ranks. Comparing
// against that output rather than against a hard-coded ranking keeps the test
// out of the resolver's business.
func TestQueryListLimitTruncates(t *testing.T) {
	h := newQueryHome(t, limitProjects...)
	full := runHopCLI(t, h, "query", "--list")
	if full.code != 0 {
		t.Fatalf("hop query --list exit = %d, want 0\nstderr: %s", full.code, full.stderr)
	}
	all := listLines(full.stdout)
	if len(all) != len(limitProjects) {
		t.Fatalf("hop query --list printed %d lines, want %d:\n%s", len(all), len(limitProjects), full.stdout)
	}

	capped := runHopCLI(t, h, "query", "--list", "--limit", "2")
	if capped.code != 0 {
		t.Fatalf("exit = %d, want 0\nstderr: %s", capped.code, capped.stderr)
	}
	got := listLines(capped.stdout)
	if len(got) != 2 {
		t.Fatalf("--limit 2 printed %d lines, want 2:\n%s", len(got), capped.stdout)
	}
	for i, want := range all[:2] {
		if got[i] != want {
			t.Errorf("line %d = %q, want %q (the cap must truncate, never reorder)", i+1, got[i], want)
		}
	}
}

// TestQueryListLimitAboveTheMatchCount pins the generous cap: it is not an
// error, it prints everything. Together with the untouched --list case above,
// this is what guards "stdout is byte-for-byte what it is today".
func TestQueryListLimitAboveTheMatchCount(t *testing.T) {
	h := newQueryHome(t, limitProjects...)
	full := runHopCLI(t, h, "query", "--list")
	got := runHopCLI(t, h, "query", "--list", "--limit", "99")
	if got.code != 0 {
		t.Fatalf("exit = %d, want 0\nstderr: %s", got.code, got.stderr)
	}
	if got.stdout != full.stdout {
		t.Errorf("a cap above the match count must print the whole list:\ngot:\n%s\nwant:\n%s", got.stdout, full.stdout)
	}
}

// TestQueryListLimitSpellings holds the three ways of writing the same capped
// request to the same answer: the equals spelling, the reversed flag order, and
// a repeated --limit, which resolves to the last occurrence the way every other
// Unix tool allows a generated command line to append an override.
func TestQueryListLimitSpellings(t *testing.T) {
	h := newQueryHome(t, limitProjects...)
	want := runHopCLI(t, h, "query", "--list", "--limit", "2")
	if want.code != 0 {
		t.Fatalf("the reference run exit = %d, want 0\nstderr: %s", want.code, want.stderr)
	}
	// Pin the reference itself, otherwise an uncapped listing would satisfy every
	// comparison below by agreeing with an equally uncapped reference.
	if n := len(listLines(want.stdout)); n != 2 {
		t.Fatalf("the reference run printed %d lines, want 2:\n%s", n, want.stdout)
	}

	for _, argv := range [][]string{
		{"query", "--list", "--limit=2"},
		{"query", "--limit", "2", "--list"},
		{"query", "--limit=2", "--list"},
		{"query", "--list", "--limit", "3", "--limit", "2"},
		{"query", "--list", "--limit=3", "--limit=2"},
	} {
		got := runHopCLI(t, h, argv...)
		if got.code != 0 {
			t.Errorf("hop %v exit = %d, want 0\nstderr: %s", argv, got.code, got.stderr)
			continue
		}
		if got.stdout != want.stdout {
			t.Errorf("hop %v stdout =\n%s\nwant:\n%s", argv, got.stdout, want.stdout)
		}
	}
}

// TestQueryLimitRefusals holds every bad --limit to the same answer: a message
// on stderr, a completely empty stdout and exit code 2. The empty stdout is the
// point of the whole set, `paths=$(hop query --list --limit $N)` must either
// hold paths or hold nothing, never a silently uncapped listing.
func TestQueryLimitRefusals(t *testing.T) {
	cases := []struct {
		name string
		argv []string
	}{
		{"zero", []string{"query", "--list", "--limit", "0"}},
		{"negative with the equals spelling", []string{"query", "--list", "--limit=-3"}},
		{"not a number", []string{"query", "--list", "--limit", "abc"}},
		{"out of int range", []string{"query", "--list", "--limit", "99999999999999999999"}},
		{"bare, with nothing after it", []string{"query", "--list", "--limit"}},
		{"a value that looks like a flag", []string{"query", "--list", "--limit", "-3"}},
		{"without --list", []string{"query", "--limit", "5"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := runHopCLI(t, newQueryHome(t, limitProjects...), c.argv...)
			if got.code != 2 {
				t.Errorf("hop %v exit = %d, want 2 (usage error)\nstdout: %s\nstderr: %s", c.argv, got.code, got.stdout, got.stderr)
			}
			if got.stdout != "" {
				t.Errorf("hop %v must leave stdout empty, got:\n%s", c.argv, got.stdout)
			}
			if strings.TrimSpace(got.stderr) == "" {
				t.Errorf("hop %v must say what is wrong on stderr, got nothing", c.argv)
			}
		})
	}
}

// TestQueryRefusesUnknownDoubleDashFlags is the behaviour change the flag needs
// to be safe: a misspelled --limit used to fall into the keyword bucket, which
// --list ignores, so `hop query --list --limitt 5` printed the whole list and
// exited 0. Single-dash tokens are deliberately left alone, they stay keywords.
func TestQueryRefusesUnknownDoubleDashFlags(t *testing.T) {
	for _, argv := range [][]string{
		{"query", "--list", "--limitt", "5"},
		{"query", "--list", "--nope"},
		{"query", "--nope"},
	} {
		got := runHopCLI(t, newQueryHome(t, limitProjects...), argv...)
		if got.code != 2 {
			t.Errorf("hop %v exit = %d, want 2 (usage error)\nstdout: %s\nstderr: %s", argv, got.code, got.stdout, got.stderr)
		}
		if got.stdout != "" {
			t.Errorf("hop %v must leave stdout empty, got:\n%s", argv, got.stdout)
		}
		if strings.TrimSpace(got.stderr) == "" {
			t.Errorf("hop %v must name the unknown flag on stderr, got nothing", argv)
		}
	}

	// A single dash is still a keyword, so a miss is a miss: exit 1, empty stdout.
	keyword := runHopCLI(t, newQueryHome(t, limitProjects...), "query", "-zzznomatch")
	if keyword.code != 1 {
		t.Errorf("a single-dash token must stay a keyword: exit = %d, want 1\nstdout: %s\nstderr: %s",
			keyword.code, keyword.stdout, keyword.stderr)
	}
	if keyword.stdout != "" {
		t.Errorf("a keyword miss must leave stdout empty, got:\n%s", keyword.stdout)
	}
}

// TestQueryHelpIsStillAnswered guards the one invocation the refusal above
// could plausibly have swallowed: --help is a -- flag that cmdQuery does not
// know, and it must keep being answered by the top-level dispatch, on stdout,
// with exit 0 and without building an index.
func TestQueryHelpIsStillAnswered(t *testing.T) {
	for _, flag := range []string{"--help", "-h"} {
		h := newQueryHome(t, limitProjects...)
		got := runHopCLI(t, h, "query", flag)
		if got.code != 0 {
			t.Errorf("hop query %s exit = %d, want 0\nstderr: %s", flag, got.code, got.stderr)
		}
		if !strings.Contains(got.stdout, "Usage:") {
			t.Errorf("hop query %s must print the help page, got:\n%s", flag, got.stdout)
		}
		if _, err := os.Stat(h.indexFile()); !os.IsNotExist(err) {
			t.Errorf("hop query %s built an index at %s (stat err = %v)", flag, h.indexFile(), err)
		}
	}
}

// TestQueryListLimitOnAnEmptyIndex keeps "no projects" distinguishable from
// "bad usage": the cap does not turn an empty result into a usage error.
func TestQueryListLimitOnAnEmptyIndex(t *testing.T) {
	got := runHopCLI(t, newQueryHome(t), "query", "--list", "--limit", "5")
	if got.code != 1 {
		t.Errorf("exit = %d, want 1 (no projects)\nstdout: %s\nstderr: %s", got.code, got.stdout, got.stderr)
	}
	if got.stdout != "" {
		t.Errorf("an empty index must leave stdout empty, got:\n%s", got.stdout)
	}
}
