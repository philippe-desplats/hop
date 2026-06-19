package core

import (
	"strings"
	"testing"
	"time"
)

// FuzzResolve drives the core ranking function with arbitrary query and project
// strings. The query and the project path feed straight into byte-level string
// indexing: matchKeywords advances `pos += idx + len(kw)` over a lowercased
// copy of the path, so malformed or multi-byte Unicode input (where lowercasing
// can change the byte length) is exactly where a slice-bounds panic would hide.
// The fuzzer asserts Resolve's structural guarantees: it never returns more
// matches than it was given, every match comes from the input set, and results
// stay ordered by descending Final score.
func FuzzResolve(f *testing.F) {
	seeds := []struct{ query, name, path string }{
		{"ops", "ops-tools", "/p/work/ops-tools"},
		{"acme web", "web-monorepo", "/p/work/acme/web-monorepo"},
		{"", "blog", "/p/side/blog"},
		{"İ", "İstanbul", "/p/work/İstanbul"}, // dotted capital I lowercases to a longer byte string
		{"ß", "straße", "/p/str/straße"},      // sharp s
		{"\U0001f680", "rocket", "/p/fun/\U0001f680-rocket"},
		{"a b c", "weird", "//\x00/.."},
	}
	for _, s := range seeds {
		f.Add(s.query, s.name, s.path)
	}

	now := time.Now()
	const stablePath = "/p/work/stable"
	f.Fuzz(func(t *testing.T, query, name, path string) {
		projects := []Project{
			{Name: name, Path: path, Category: "fuzz"},
			{Name: "stable", Path: stablePath, Category: "work"},
		}
		keywords := strings.Fields(query)

		matches := Resolve(projects, emptyFrecency(), keywords, now, DefaultWeights())

		if len(matches) > len(projects) {
			t.Fatalf("Resolve returned %d matches for %d projects", len(matches), len(projects))
		}
		known := map[string]bool{path: true, stablePath: true}
		for i, m := range matches {
			if !known[m.Project.Path] {
				t.Fatalf("match %d has a path not in the input set: %q", i, m.Project.Path)
			}
			if i > 0 && matches[i-1].Final < m.Final {
				t.Fatalf("matches out of order: [%d].Final=%g < [%d].Final=%g", i-1, matches[i-1].Final, i, m.Final)
			}
		}
	})
}
