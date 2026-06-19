package action

import (
	"testing"

	"github.com/philippe-desplats/hop/internal/core"
)

func TestMultiplexerRun(t *testing.T) {
	const name, path = "web-app", "/p/x"
	cases := []struct {
		name     string
		mux      string
		inTmux   bool
		inZellij bool
		want     string
	}{
		{"tmux outside", "tmux", false, false, "tmux new-session -A -s web-app -c '/p/x'"},
		{"tmux inside", "tmux", true, false, "tmux new-session -dA -s web-app -c '/p/x' ; tmux switch-client -t web-app"},
		{"zellij outside", "zellij", false, false, "zellij attach --create web-app"},
		{"zellij inside", "zellij", false, true, "zellij action new-tab --cwd '/p/x' --name web-app"},
		{"unknown mux", "screen", false, false, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := multiplexerRun(tc.mux, name, path, tc.inTmux, tc.inZellij); got != tc.want {
				t.Errorf("multiplexerRun(%q, in_tmux=%v, in_zellij=%v) = %q, want %q", tc.mux, tc.inTmux, tc.inZellij, got, tc.want)
			}
		})
	}
}

func TestMultiplexerRunQuotesSpacedPath(t *testing.T) {
	got := multiplexerRun("tmux", "proj", "/p/with space", false, false)
	const want = "tmux new-session -A -s proj -c '/p/with space'"
	if got != want {
		t.Errorf("got %q, want %q (path must be shell-quoted)", got, want)
	}
}

func TestResolveMultiplexer(t *testing.T) {
	cases := []struct {
		name     string
		setting  string
		showTmux bool
		present  []string
		want     string
	}{
		{"auto prefers tmux", "auto", false, []string{"tmux", "zellij"}, "tmux"},
		{"auto falls back to zellij", "auto", false, []string{"zellij"}, "zellij"},
		{"auto with neither hides", "auto", false, nil, ""},
		{"explicit tmux present", "tmux", false, []string{"tmux"}, "tmux"},
		{"explicit tmux absent hides", "tmux", false, nil, ""},
		{"explicit zellij present", "zellij", false, []string{"zellij"}, "zellij"},
		{"off hides even if installed", "off", true, []string{"tmux"}, ""},
		{"legacy show_tmux maps to auto", "", true, []string{"tmux"}, "tmux"},
		{"empty without show_tmux is off", "", false, []string{"tmux"}, ""},
		{"unknown value is off", "screen", true, []string{"tmux"}, ""},
		{"case-insensitive", "TMUX", false, []string{"tmux"}, "tmux"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stubLookPath(t, tc.present...)
			if got := ResolveMultiplexer(tc.setting, tc.showTmux); got != tc.want {
				t.Errorf("ResolveMultiplexer(%q, show_tmux=%v) = %q, want %q", tc.setting, tc.showTmux, got, tc.want)
			}
		})
	}
}

func TestMultiplexerActionHiddenWhenAbsent(t *testing.T) {
	p := core.Project{Name: "x", Path: "/p/x"}
	if _, ok := ByKey("t", p, Options{Multiplexer: ""}); ok {
		t.Error("the t action must be hidden when no multiplexer is resolved")
	}
	if _, ok := ByKey("t", p, Options{Multiplexer: "tmux"}); !ok {
		t.Error("the t action must be present when a multiplexer is resolved")
	}
}
