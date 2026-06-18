package main

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/philippe-desplats/hop/internal/action"
	"github.com/philippe-desplats/hop/internal/core"
)

// captureStdout runs fn with os.Stdout redirected to a pipe and returns what it
// printed. emitOutcome writes the shell protocol to stdout via fmt.Printf.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() { os.Stdout = old }()
	fn()
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func TestEmitOutcome(t *testing.T) {
	cases := []struct {
		name string
		out  action.Outcome
		want string
	}{
		{"cd only", action.Outcome{Cd: "/p/x"}, "__HOP_CD__ /p/x\n"},
		{"cd and run", action.Outcome{Cd: "/p/x", Run: "claude"}, "__HOP_CD__ /p/x\n__HOP_RUN__ claude\n"},
		{"run only", action.Outcome{Run: "git status"}, "__HOP_RUN__ git status\n"},
		{"empty emits nothing", action.Outcome{}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := captureStdout(t, func() { emitOutcome(tc.out) })
			if got != tc.want {
				t.Errorf("emitOutcome(%+v) = %q, want %q", tc.out, got, tc.want)
			}
		})
	}
}

func TestActionOutcome(t *testing.T) {
	p := core.Project{Name: "demo", Path: "/p/demo"}
	opts := action.Options{
		Editor: "zed",
		AI:     action.Assistant{Name: "claude", Run: []string{"claude"}, Resume: []string{"claude", "--resume"}},
		HasAI:  true,
	}
	cases := []struct {
		name    string
		key     string
		wantCd  string
		wantRun string
	}{
		{"enter is a bare cd", "enter", "/p/demo", ""},
		{"claude runs the assistant", "c", "/p/demo", "claude"},
		{"resume passes --resume", "r", "/p/demo", "claude --resume"},
		{"unknown key falls back to cd", "qzx", "/p/demo", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := actionOutcome(tc.key, p, opts)
			if got.Cd != tc.wantCd || got.Run != tc.wantRun {
				t.Errorf("actionOutcome(%q) = {Cd:%q Run:%q}, want {Cd:%q Run:%q}",
					tc.key, got.Cd, got.Run, tc.wantCd, tc.wantRun)
			}
		})
	}
}
