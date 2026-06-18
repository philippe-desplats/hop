package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/philippe-desplats/hop/internal/core"
	"github.com/philippe-desplats/hop/internal/i18n"
)

var cmdNameRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// cmdInit prints the shell integration. The daily function name defaults to "p"
// and is overridable with --cmd NAME (or --cmd=NAME).
func cmdInit(args []string) {
	shell := "zsh"
	name := core.LoadSettings().Shell.Command // config value; --cmd overrides below
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--cmd":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "hop: --cmd requires a name")
				os.Exit(2)
			}
			name = args[i+1]
			i++
		case strings.HasPrefix(a, "--cmd="):
			name = strings.TrimPrefix(a, "--cmd=")
		case strings.HasPrefix(a, "-"):
			fmt.Fprintf(os.Stderr, "hop: unknown flag %q\n", a)
			os.Exit(2)
		default:
			shell = a
		}
	}
	if !cmdNameRe.MatchString(name) {
		fmt.Fprintf(os.Stderr, "hop: invalid function name %q (letters, digits, underscore; not starting with a digit)\n", name)
		os.Exit(2)
	}
	switch shell {
	case "zsh":
		if created, _ := core.EnsureConfig(); created {
			fmt.Fprintln(os.Stderr, i18n.Tf("cli.config_created", core.ConfigPath()))
		}
		fmt.Print(zshIntegration(name))
	default:
		fmt.Fprintf(os.Stderr, "hop: unsupported shell %q (zsh only for now)\n", shell)
		os.Exit(2)
	}
}

// zshIntegration renders the sourced function under the chosen command name.
func zshIntegration(cmd string) string {
	const tmpl = `# hop shell integration, add  eval "$(hop init zsh)"  to ~/.zsh_init
#
# Reclaim the name in case an alias shadows it (e.g. common-aliases' p='ps -f').
# Cheatsheet:
#   %[1]s <kw>         jump straight to the best project
#   %[1]s <kw> <kw>    narrow by sub-path, e.g. %[1]s acme web
#   %[1]s -            jump back to the previous project
#   %[1]s              open the interactive fuzzy Hub (Enter to cd)
#   %[1]s @name        pinned bookmark (v1.1)
# Hub keys (v1.0):
#   enter cd · z Zed · c Claude · r Claude --resume · g git status
#   o remote repo · f Finder · t tmux session
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
`
	return fmt.Sprintf(tmpl, cmd)
}
