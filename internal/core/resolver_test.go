package core

import (
	"testing"
	"time"
)

func sampleProjects() []Project {
	return []Project{
		{Name: "ops-tools", Path: "/p/work/ops-tools", Category: "work"},
		{Name: "blog", Path: "/p/side/blog", Category: "side"},
		{Name: "toolbox", Path: "/p/work/toolbox", Category: "work"},
	}
}

func TestResolveFragment(t *testing.T) {
	now := time.Now()
	frec := &Frecency{Version: frecencyVersion, Entries: map[string]*fEntry{}}
	m := Resolve(sampleProjects(), frec, []string{"ops"}, now, DefaultWeights())
	if len(m) != 1 || m[0].Project.Name != "ops-tools" {
		t.Fatalf("fragment 'ops' = %+v, want single ops-tools", m)
	}
}

func TestResolveFrecencyWins(t *testing.T) {
	now := time.Now()
	frec := &Frecency{Version: frecencyVersion, Entries: map[string]*fEntry{
		"/p/work/toolbox": {Rank: 5, LastAccess: now.Unix()},
	}}
	m := Resolve(sampleProjects(), frec, nil, now, DefaultWeights())
	if len(m) != 3 {
		t.Fatalf("no keywords should return all 3, got %d", len(m))
	}
	if m[0].Project.Name != "toolbox" {
		t.Errorf("most frecent should rank first, got %q", m[0].Project.Name)
	}
}

func TestResolveNameBeatsPathColdStart(t *testing.T) {
	now := time.Now()
	frec := &Frecency{Version: frecencyVersion, Entries: map[string]*fEntry{}}
	projects := []Project{
		{Name: "k6", Path: "/p/devops/k6", Category: "devops"},           // "ops" only in path
		{Name: "DevOps", Path: "/p/shpv/DevOps", Category: "shpv"},       // "ops" mid-word, shorter
		{Name: "ops-tools", Path: "/p/work/ops-tools", Category: "work"}, // "ops" is a token
	}
	m := Resolve(projects, frec, []string{"ops"}, now, DefaultWeights())
	if len(m) == 0 || m[0].Project.Name != "ops-tools" {
		t.Fatalf("cold-start 'ops' = %+v, want ops-tools first (name token beats substring/path)", m)
	}
}

func TestResolveMultiKeyword(t *testing.T) {
	now := time.Now()
	frec := &Frecency{Version: frecencyVersion, Entries: map[string]*fEntry{}}
	projects := []Project{
		{Name: "web-monorepo", Path: "/p/work/acme/web-monorepo", Category: "work"},
		{Name: "web-shop", Path: "/p/work/globex/web-shop", Category: "work"}, // web but no acme
		{Name: "acme-api", Path: "/p/work/acme/api", Category: "work"},
	}
	// "acme web" must hit only the path that contains acme THEN web.
	m := Resolve(projects, frec, []string{"acme", "web"}, now, DefaultWeights())
	if len(m) != 1 || m[0].Project.Name != "web-monorepo" {
		t.Fatalf("'acme web' = %+v, want only web-monorepo", m)
	}
	// Order matters: "web acme" should not match (acme comes before web in the path).
	if m := Resolve(projects, frec, []string{"web", "acme"}, now, DefaultWeights()); len(m) != 0 {
		t.Fatalf("'web acme' (wrong order) should not match, got %+v", m)
	}
}

func TestResolveNoMatch(t *testing.T) {
	now := time.Now()
	frec := &Frecency{Version: frecencyVersion, Entries: map[string]*fEntry{}}
	if m := Resolve(sampleProjects(), frec, []string{"zzz-nope"}, now, DefaultWeights()); len(m) != 0 {
		t.Fatalf("no-match fragment should return 0, got %d", len(m))
	}
}
