// Package action is the Hub's modular action registry. Each action turns a
// selected project into an Outcome (what the shell should do) and may perform an
// in-process side effect first (launching a GUI app or a browser). Adding an
// action is one entry in All().
package action

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/philippe-desplats/hop/internal/core"
	"github.com/philippe-desplats/hop/internal/i18n"
)

// Outcome is what the shell does after the Hub exits: cd somewhere and/or run a
// command (eval'd by the `p` function in the user's interactive shell).
type Outcome struct {
	Cd  string
	Run string
}

// Options tune the action set from user settings.
type Options struct {
	Editor   string              // command for the "open in editor" action (default zed)
	ShowTmux bool                // include the tmux action
	AI       Assistant           // assistant bound to c/r (resolved by ResolveAssistant)
	HasAI    bool                // false when no assistant is installed (c/r hidden)
	Custom   []core.CustomAction // user-defined [[actions.custom]] entries
}

// Spec is a single Hub action.
type Spec struct {
	Key   string // "enter" or a single letter
	Label string // full label for the action menu
	Short string // compact label for the alt-mode legend
	avail func(core.Project) bool
	do    func(core.Project) Outcome
}

// Available reports whether the action applies to p.
func (s Spec) Available(p core.Project) bool {
	return s.avail == nil || s.avail(p)
}

// Do performs the action's side effect (if any) and returns the shell Outcome.
func (s Spec) Do(p core.Project) Outcome { return s.do(p) }

var sessionRe = regexp.MustCompile(`[^A-Za-z0-9_-]+`)

// reservedKeys cannot be reused by custom actions.
var reservedKeys = map[string]bool{
	"enter": true, "z": true, "c": true, "r": true,
	"g": true, "o": true, "f": true, "t": true,
}

// All returns the ordered action set for the given options. Order drives the
// menu order.
func All(o Options) []Spec {
	editor := o.Editor
	if editor == "" {
		editor = "zed"
	}
	specs := []Spec{
		{Key: "enter", Label: i18n.T("action.cd"), Short: "cd", do: func(p core.Project) Outcome {
			return Outcome{Cd: p.Path}
		}},
		{Key: "z", Label: i18n.T("action.editor") + " (" + editor + ")", Short: editor, do: func(p core.Project) Outcome {
			launch(editor, p.Path)
			return Outcome{Cd: p.Path}
		}},
	}
	if o.HasAI {
		ai := o.AI
		specs = append(specs, Spec{Key: "c", Label: i18n.Tf("action.ai", ai.Name), Short: ai.Name, do: func(p core.Project) Outcome {
			return Outcome{Cd: p.Path, Run: ai.runCmd()}
		}})
		if ai.hasResume() {
			specs = append(specs, Spec{Key: "r", Label: i18n.Tf("action.ai.resume", ai.Name), Short: i18n.T("action.short.resume"), do: func(p core.Project) Outcome {
				return Outcome{Cd: p.Path, Run: ai.resumeCmd()}
			}})
		}
	}
	specs = append(specs,
		Spec{Key: "g", Label: i18n.T("action.git"), Short: "git", avail: hasGit, do: func(p core.Project) Outcome {
			return Outcome{Cd: p.Path, Run: "git status"}
		}},
		Spec{Key: "o", Label: i18n.T("action.remote"), Short: i18n.T("action.short.remote"), avail: hasGit, do: func(p core.Project) Outcome {
			if url := remoteURL(p.Path); url != "" {
				launch("open", url)
			}
			return Outcome{Cd: p.Path}
		}},
		Spec{Key: "f", Label: i18n.T("action.finder"), Short: "Finder", do: func(p core.Project) Outcome {
			launch("open", p.Path)
			return Outcome{Cd: p.Path}
		}},
	)
	if o.ShowTmux {
		specs = append(specs, Spec{Key: "t", Label: i18n.T("action.tmux"), Short: "tmux", do: func(p core.Project) Outcome {
			name := strings.Trim(sessionRe.ReplaceAllString(p.Name, "-"), "-")
			if name == "" {
				name = "hop"
			}
			return Outcome{Cd: p.Path, Run: "tmux new-session -A -s " + name}
		}})
	}
	for _, c := range o.Custom {
		if s, ok := customSpec(c); ok {
			specs = append(specs, s)
		}
	}
	return specs
}

// customSpec turns a CustomAction into a Spec, or ok=false when it is invalid
// (empty key/command or a key reserved by a built-in). Invalid entries are
// skipped silently here; hop doctor reports them.
func customSpec(c core.CustomAction) (Spec, bool) {
	if c.Key == "" || c.Command == "" || reservedKeys[c.Key] {
		return Spec{}, false
	}
	spec := Spec{Key: c.Key, Label: c.Label, Short: c.Label}
	if c.NeedsGit {
		spec.avail = hasGit
	}
	spec.do = func(p core.Project) Outcome {
		if c.InTerminal {
			return Outcome{Cd: p.Path, Run: substituteShell(c.Command, p)}
		}
		if fields := substituteArgv(c.Command, p); len(fields) > 0 {
			launch(fields[0], fields[1:]...)
		}
		return Outcome{Cd: p.Path}
	}
	return spec, true
}

// InvalidCustomActions returns one human-readable reason per custom action that
// customSpec will skip (empty key, empty command, or a key reserved by a
// built-in), so `hop doctor` can surface misconfigured entries.
func InvalidCustomActions(custom []core.CustomAction) []string {
	var bad []string
	for _, c := range custom {
		switch {
		case c.Key == "":
			bad = append(bad, "(no key): every custom action needs a key")
		case c.Command == "":
			bad = append(bad, c.Key+": missing command")
		case reservedKeys[c.Key]:
			bad = append(bad, c.Key+": key reserved by a built-in action")
		}
	}
	return bad
}

// substituteArgv splits command into argv tokens and replaces {path}/{name} in
// each, so a detached launch never goes through a shell.
func substituteArgv(command string, p core.Project) []string {
	fields := strings.Fields(command)
	for i, f := range fields {
		f = strings.ReplaceAll(f, "{path}", p.Path)
		fields[i] = strings.ReplaceAll(f, "{name}", p.Name)
	}
	return fields
}

// substituteShell replaces {path}/{name} with shell-quoted values, for a command
// the p() function will eval in the user's shell.
func substituteShell(command string, p core.Project) string {
	command = strings.ReplaceAll(command, "{path}", shellQuote(p.Path))
	return strings.ReplaceAll(command, "{name}", shellQuote(p.Name))
}

// shellQuote wraps s in single quotes for POSIX shells, escaping embedded single
// quotes as the standard '\” sequence.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// ByKey returns the action bound to key if it is available for p.
func ByKey(key string, p core.Project, o Options) (Spec, bool) {
	for _, s := range All(o) {
		if s.Key == key && s.Available(p) {
			return s, true
		}
	}
	return Spec{}, false
}

// launch starts a detached program with an explicit argv (no shell), so it
// neither blocks the Hub nor risks shell injection.
func launch(name string, args ...string) {
	// name is a built-in constant ("open"), the editor from config, or a custom
	// action command. All originate from the user's own config.toml, treated as
	// trusted; args are project paths and no shell is involved.
	cmd := exec.Command(name, args...) //nolint:gosec // argv from the user's own local config, not remote/untrusted input
	if err := cmd.Start(); err == nil && cmd.Process != nil {
		_ = cmd.Process.Release()
	}
}

func hasGit(p core.Project) bool {
	_, err := os.Stat(filepath.Join(p.Path, ".git"))
	return err == nil
}

// remoteURL derives a browser URL from origin, returning "" if it is not http(s).
func remoteURL(dir string) string {
	//nolint:gosec // fixed "git" binary with constant args; dir is a project path from our own index
	out, err := exec.Command("git", "-C", dir, "remote", "get-url", "origin").Output()
	if err != nil {
		return ""
	}
	url := strings.TrimSpace(string(out))
	if strings.HasPrefix(url, "git@") { // git@host:group/repo.git -> https://host/group/repo
		url = strings.TrimSuffix(url, ".git")
		url = strings.Replace(url, ":", "/", 1)
		url = strings.Replace(url, "git@", "https://", 1)
	}
	url = strings.TrimSuffix(url, ".git")
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return ""
	}
	return url
}
