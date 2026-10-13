package main

import (
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/philippe-desplats/hop/internal/core"
	"github.com/philippe-desplats/hop/internal/i18n"
)

var cmdNameRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// defaultCmdName is the daily function name used when nothing valid overrides it.
const defaultCmdName = "p"

// supportedShells are the shells hop init can emit an integration for.
var supportedShells = []string{"zsh", "bash", "fish"}

// isShellName reports whether name is one of the shells hop init knows. Such a
// name is refused as a function name: the integration would shadow the shell's
// own binary, so typing `zsh` would run hop nav instead of starting a subshell.
func isShellName(name string) bool {
	return slices.Contains(supportedShells, name)
}

// validCmdName reports whether name is usable as the daily shell function name.
// The pattern is deliberately tight: the name is interpolated into the three
// integration templates, so anything outside it could inject shell syntax.
func validCmdName(name string) bool {
	return cmdNameRe.MatchString(name) && !isShellName(name)
}

// cmdInit prints the shell integration. The daily function name defaults to "p"
// and is overridable with --cmd NAME (or --cmd=NAME).
//
// The two sources of that name are not treated alike. A bad --cmd value is a
// usage error: the user is right there to read the message. A bad config value
// only earns a warning and a fall back to "p", because `hop init` is what the rc
// file wraps in eval, so exiting would leave every new shell without the
// function and with nothing pointing at the offending line.
func cmdInit(args []string) {
	shell := "zsh"
	flagName, fromFlag := "", false
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--cmd":
			// A following flag is a missing name, not a malformed one: reject it
			// here rather than letting it fall through to the name pattern.
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(os.Stderr, "hop: --cmd requires a name")
				os.Exit(2)
			}
			flagName, fromFlag = args[i+1], true
			i++
		case strings.HasPrefix(a, "--cmd="):
			flagName, fromFlag = strings.TrimPrefix(a, "--cmd="), true
		case strings.HasPrefix(a, "-"):
			fmt.Fprintf(os.Stderr, "hop: unknown flag %q\n", a)
			os.Exit(2)
		default:
			shell = a
		}
	}
	name := flagName
	if !fromFlag {
		name = core.LoadSettings().Shell.Command
	}
	if !validCmdName(name) {
		switch {
		case !fromFlag:
			fmt.Fprintf(os.Stderr, "hop: ignoring invalid [shell] command %q in %s, using %q instead\n",
				name, core.ConfigPath(), defaultCmdName)
			name = defaultCmdName
		case isShellName(name):
			fmt.Fprintf(os.Stderr, "hop: --cmd %s is ambiguous: %s is a shell name, not a function name.\n"+
				"  did you mean  hop init %s  (default name %q), or  hop init %s --cmd NAME ?\n",
				name, name, name, defaultCmdName, name)
			os.Exit(2)
		default:
			fmt.Fprintf(os.Stderr, "hop: invalid function name %q (letters, digits, underscore; not starting with a digit)\n", name)
			os.Exit(2)
		}
	}
	emit := func(integration string) {
		if created, _ := core.EnsureConfig(); created {
			fmt.Fprintln(os.Stderr, i18n.Tf("cli.config_created", core.ConfigPath()))
		}
		fmt.Print(integration)
	}
	switch shell {
	case "zsh":
		emit(zshIntegration(name))
	case "bash":
		emit(bashIntegration(name))
	case "fish":
		emit(fishIntegration(name))
	default:
		fmt.Fprintf(os.Stderr, "hop: unsupported shell %q (zsh, bash, fish)\n", shell)
		os.Exit(2)
	}
}

// zshIntegration renders the sourced function under the chosen command name.
func zshIntegration(cmd string) string {
	const tmpl = `# hop shell integration, add  eval "$(hop init zsh)"  to ~/.zshrc
#
# Reclaim the name in case an alias shadows it (e.g. common-aliases' p='ps -f').
# Cheatsheet:
#   %[1]s <kw>         jump straight to the best project
#   %[1]s <kw> <kw>    narrow by sub-path, e.g. %[1]s acme web
#   %[1]s -            jump back to the previous project
#   %[1]s              open the interactive fuzzy Hub (Enter to cd)
# Hub keys: enter cd · z editor · c AI assistant · r resume · g git
#   o remote · f files · t tmux/zellij · plus your [[actions.custom]] keys
unalias %[1]s 2>/dev/null
# 'function %[1]s' (not '%[1]s()') so a still-active alias is not expanded at parse time.
function %[1]s {
  local out line dir cmd
  out="$(command hop nav "$@")" || return
  while IFS= read -r line; do
    if [[ "$line" == "__HOP_CD__ "* ]]; then
      dir="${line#__HOP_CD__ }"
    elif [[ "$line" == "__HOP_RUN__ "* ]]; then
      cmd="${line#__HOP_RUN__ }"
    fi
  done <<< "$out"
  [[ -n "$dir" ]] && builtin cd -- "$dir"
  [[ -n "$cmd" ]] && eval "$cmd"
}
# Learn from every directory change (%[1]s or a manual cd), zoxide-style.
# Anti-storm: skip an unchanged $PWD; hop add is detached and non-blocking.
typeset -g _hop_last_pwd=""
function _hop_chpwd {
  [[ "$PWD" == "$_hop_last_pwd" ]] && return
  _hop_last_pwd="$PWD"
  command hop add "$PWD" &>/dev/null &!
}
autoload -Uz add-zsh-hook && add-zsh-hook chpwd _hop_chpwd
# Completion: project names from the index (needs compinit, which most zsh setups load).
_hop_complete() {
  local -a names
  names=(${(f)"$(command hop complete "${words[CURRENT]}")"})
  compadd -a names
}
(( $+functions[compdef] )) && compdef _hop_complete %[1]s
`
	return fmt.Sprintf(tmpl, cmd)
}

// bashIntegration renders the bash function, frecency hook and completion.
func bashIntegration(cmd string) string {
	const tmpl = `# hop shell integration, add  eval "$(hop init bash)"  to ~/.bashrc
# Cheatsheet:
#   %[1]s <kw>         jump straight to the best project
#   %[1]s <kw> <kw>    narrow by sub-path, e.g. %[1]s acme web
#   %[1]s -            jump back to the previous project
#   %[1]s              open the interactive fuzzy Hub (Enter to cd)
# Hub keys: enter cd · z editor · c AI assistant · r resume · g git
#   o remote · f files · t tmux/zellij · plus your [[actions.custom]] keys
unalias %[1]s 2>/dev/null
%[1]s() {
  local out line dir cmd
  out="$(command hop nav "$@")" || return
  while IFS= read -r line; do
    case "$line" in
      "__HOP_CD__ "*) dir="${line#__HOP_CD__ }" ;;
      "__HOP_RUN__ "*) cmd="${line#__HOP_RUN__ }" ;;
    esac
  done <<< "$out"
  [ -n "$dir" ] && cd -- "$dir"
  [ -n "$cmd" ] && eval "$cmd"
}
# Frecency hook: bash has no chpwd; append to PROMPT_COMMAND (never clobber it).
_hop_last_pwd=""
_hop_record() {
  [ "$PWD" = "$_hop_last_pwd" ] && return
  _hop_last_pwd="$PWD"
  # Subshell so bash job control never prints "[1]+ Done ..." at the next prompt.
  ( command hop add "$PWD" >/dev/null 2>&1 & )
}
case "$PROMPT_COMMAND" in
  *_hop_record*) ;;
  *) PROMPT_COMMAND="_hop_record${PROMPT_COMMAND:+; $PROMPT_COMMAND}" ;;
esac
# Completion: project names from the index.
_hop_complete() {
  local cur="${COMP_WORDS[COMP_CWORD]}"
  COMPREPLY=( $(command hop complete "$cur") )
}
complete -F _hop_complete %[1]s
`
	return fmt.Sprintf(tmpl, cmd)
}

// fishIntegration renders the fish function, frecency hook and completion.
func fishIntegration(cmd string) string {
	const tmpl = `# hop shell integration, add  hop init fish | source  to ~/.config/fish/config.fish
# Cheatsheet:
#   %[1]s <kw>         jump straight to the best project
#   %[1]s <kw> <kw>    narrow by sub-path, e.g. %[1]s acme web
#   %[1]s -            jump back to the previous project
#   %[1]s              open the interactive fuzzy Hub (Enter to cd)
# Hub keys: enter cd · z editor · c AI assistant · r resume · g git
#   o remote · f files · t tmux/zellij · plus your [[actions.custom]] keys
function %[1]s
    set -l out (command hop nav $argv)
    or return
    set -l dir
    set -l cmd
    for line in $out
        if string match -q '__HOP_CD__ *' -- $line
            set dir (string replace '__HOP_CD__ ' '' -- $line)
        else if string match -q '__HOP_RUN__ *' -- $line
            set cmd (string replace '__HOP_RUN__ ' '' -- $line)
        end
    end
    test -n "$dir"; and cd -- $dir
    test -n "$cmd"; and eval $cmd
end
# Frecency hook: fish fires on every PWD change.
function _hop_record --on-variable PWD
    command hop add $PWD >/dev/null 2>&1 &
end
# Completion: project names from the index.
complete -c %[1]s -f -a "(command hop complete (commandline -ct))"
`
	return fmt.Sprintf(tmpl, cmd)
}
