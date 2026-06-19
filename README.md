<div align="center">

# hop

**A fast, AI-agnostic project switcher for your terminal.**

Jump to any project by frecency, then drop into your editor, your AI assistant, or a custom action, without retyping paths.

[![CI](https://github.com/philippe-desplats/hop/actions/workflows/ci.yml/badge.svg)](https://github.com/philippe-desplats/hop/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/philippe-desplats/hop?sort=semver)](https://github.com/philippe-desplats/hop/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/philippe-desplats/hop)](https://goreportcard.com/report/github.com/philippe-desplats/hop)

<img src="docs/demo.gif" alt="hop in action" width="720" />

</div>

Where `zoxide` stops at `cd`, `hop` keeps going: one keystroke takes you from "which project was that again" to "open, and start working".

## Why

`hop` is a single static binary that prints a target; a small shell function named `p` performs the `cd` (a child process cannot change its parent shell's working directory, the same model `zoxide` uses). Jumps learn from your habits through frecency, frequency plus recency, so the project you reach for most is one keystroke away.

The catch with most switchers is they stop at the jump. `hop` treats the jump as the start: from the interactive Hub you open the project in your editor, hand it to your AI assistant, peek at its git status, or run your own action, without ever leaving the keyboard.

## Features

- **Instant jump by keyword**, ranked by frecency (`p api`).
- **Ordered multi-keyword matching** to narrow by sub-path (`p acme web`).
- **Interactive fuzzy Hub** with a per-project action menu: cd, editor, AI assistant, git, remote, Finder, tmux, and your own custom actions.
- **AI-agnostic**: the assistant keys auto-detect Claude, Codex, Aider or Gemini, and you can pin a specific one in config.
- **Pin favorites** so they float to the top of the Hub, from the CLI or with a single key in the Hub.
- **Jump back** to where you were with `p -` (and `p -2`, `p -3`).
- **Forgets dead paths**: projects whose folder is gone are pruned.
- **Multi-shell**: zsh, bash and fish, with TAB-completion of project names.
- **Adaptive light and dark themes**, UI in English, French, Spanish and Portuguese.
- **TOML configuration** with an interactive editor (`hop config`).

## Installation

### 1. Get the binary

```sh
# Homebrew (after the first tagged release)
brew install philippe-desplats/tap/hop

# or from source, with Go 1.24+
go install github.com/philippe-desplats/hop/cmd/hop@latest
```

### 2. Wire the shell integration

Add the matching line to your shell startup file, then open a new shell.

<details>
<summary><b>zsh</b> (<code>~/.zshrc</code>)</summary>

```sh
eval "$(hop init zsh)"
```
</details>

<details>
<summary><b>bash</b> (<code>~/.bashrc</code>)</summary>

```sh
eval "$(hop init bash)"
```
</details>

<details>
<summary><b>fish</b> (<code>~/.config/fish/config.fish</code>)</summary>

```sh
hop init fish | source
```
</details>

`--cmd NAME` picks a different function name (default `p`), and `p <TAB>` completes project names. The first invocation indexes your project roots automatically.

## Usage

```sh
p api            # jump straight to the best match (e.g. acme-api)
p acme web       # multiple keywords: narrow by ordered sub-path
p -              # jump back to the previous project (p -2, p -3 go further)
p                # open the interactive Hub (fuzzy list, up/down + Enter = cd)

hop pin web      # pin a project so it floats to the top of the Hub (marked with a star)
hop unpin web    # remove a pin
hop scan         # reindex on demand
hop clean        # forget projects whose folder no longer exists
hop doctor       # configuration diagnostics
```

In the Hub, type to filter, `Tab` opens the action menu, and from there a single key fires an action: `z` editor, `c` AI assistant, `r` resume (when the assistant supports it), `g` git, `o` remote, `f` Finder, `t` tmux (when enabled), `p` pin, plus any custom action you defined.

Keywords are ordered and each must appear in the path (the `zoxide` model). A freshly created project is found automatically: on a miss, `hop` reindexes once and retries before giving up.

## Try it without touching your setup

The repository ships a self-contained playground. It indexes only a throwaway copy of `sample/projects` and keeps its index in a temp directory, so your real configuration and index are never touched.

```sh
source sample/demo.sh
p api        # jump straight to acme-api
p            # open the Hub, type to filter, Tab for actions
p -          # jump back
```

See [`sample/README.md`](sample/README.md) for details and for regenerating the demo GIF with [VHS](https://github.com/charmbracelet/vhs).

## Configuration

`hop config` opens an interactive editor. The file `~/.config/hop/config.toml` (honoring `XDG_CONFIG_HOME`) is created automatically:

```toml
[ui]
language = "auto"        # auto (from $LANG), or en / fr / es / pt
theme = "auto"           # auto (detect terminal background), or light / dark

[ai]
tool = "auto"            # auto (first installed), or claude / codex / aider / gemini

[shell]
command = "p"            # name of the daily shortcut

[hub]
# tab: Tab opens the menu · shift: UPPERCASE keys act directly · enter: Enter opens the menu
action_access = "tab"

[actions]
editor = "zed"           # single executable for the "open in editor" action
show_tmux = false        # show the tmux action in the menu

# Optional custom actions, each adds a key to the menu:
# [[actions.custom]]
# key = "y"
# label = "open in Cursor"
# command = "cursor {path}"   # {path} and {name} are substituted
# needs_git = false
# in_terminal = false         # false: launch detached (GUI) · true: run in the shell after cd

[scan]
roots = ["~/Projects"]   # where to look for projects
max_depth = 7
ignore = ["node_modules", "vendor", "_archives"]

[resolver]
# Ranking weights for a one-shot jump, and the margin below which the Hub opens
# instead of guessing between close matches.
w_fuzzy = 0.6
w_frecency = 0.4
min_margin = 0.15
```

Most settings are re-read on every `p`, so they take effect immediately. The command name (`[shell] command`) is read when the integration is sourced, so it takes effect on the next shell start.

## How it works

- A git-repo-first scanner walks your roots, treating each git repository (and any leaf project folder) as a target.
- Frecency ranks targets by how often and how recently you visit them.
- The Hub (built on [Bubble Tea](https://github.com/charmbracelet/bubbletea)) renders the fuzzy list and the per-project action menu.
- State (index, frecency, pins) is owned by a single store with atomic, lock-guarded JSON writes.

## Development

```sh
go build ./...
go test ./...
go vet ./...
```

## Security

See [SECURITY.md](SECURITY.md) for how to report a vulnerability and the project's supply-chain posture.

## License

[MIT](LICENSE)
