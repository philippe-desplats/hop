package action

import (
	"os"
	"strings"

	"github.com/philippe-desplats/hop/internal/core"
)

// ResolveMultiplexer picks the terminal multiplexer the t action should drive.
// setting is the [actions] multiplexer config ("auto" | "tmux" | "zellij" |
// "off"); showTmux is the legacy show_tmux flag, read only when setting is
// absent (then true maps to "auto"). It returns "tmux", "zellij", or "" when
// none is selected or the chosen one is not installed (the action then hides).
func ResolveMultiplexer(setting string, showTmux bool) string {
	setting = strings.ToLower(strings.TrimSpace(setting))
	if setting == "" {
		if showTmux {
			setting = "auto"
		} else {
			setting = "off"
		}
	}
	switch setting {
	case "tmux":
		return ifAvailable("tmux")
	case "zellij":
		return ifAvailable("zellij")
	case "auto":
		if m := ifAvailable("tmux"); m != "" {
			return m
		}
		return ifAvailable("zellij")
	default: // "off" or an unknown value
		return ""
	}
}

// ifAvailable returns bin when it is on PATH, else "".
func ifAvailable(bin string) string {
	if _, err := lookPath(bin); err == nil {
		return bin
	}
	return ""
}

// insideTmux / insideZellij report whether hop is already running inside a
// session, read from the env the interactive shell passed down.
func insideTmux() bool   { return os.Getenv("TMUX") != "" }
func insideZellij() bool { return os.Getenv("ZELLIJ") != "" }

// sessionName derives a multiplexer-safe session name from the project name,
// collapsing everything outside [A-Za-z0-9_-] and falling back to "hop".
func sessionName(p core.Project) string {
	name := strings.Trim(sessionRe.ReplaceAllString(p.Name, "-"), "-")
	if name == "" {
		name = "hop"
	}
	return name
}

// multiplexerRun builds the shell command the t action emits as Outcome.Run.
// The p() shell function cd's into the project then eval's this string, so the
// command runs from the project dir and a ";"-joined pair is valid. It is a pure
// function: the caller passes the inside-session flags (no env reads here) so it
// is fully unit-testable. The path is shell-quoted; the name is already
// sanitized by sessionName. An unknown mux yields "".
func multiplexerRun(mux, name, path string, inTmux, inZellij bool) string {
	q := shellQuote(path)
	switch mux {
	case "tmux":
		if inTmux {
			// Already in a session: create detached then switch, never nest.
			return "tmux new-session -dA -s " + name + " -c " + q + " ; tmux switch-client -t " + name
		}
		return "tmux new-session -A -s " + name + " -c " + q
	case "zellij":
		if inZellij {
			// zellij refuses nesting: open a tab in the current session instead.
			return "zellij action new-tab --cwd " + q + " --name " + name
		}
		return "zellij attach --create " + name
	}
	return ""
}
