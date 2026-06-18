# hop

[![CI](https://github.com/philippe-desplats/hop/actions/workflows/ci.yml/badge.svg)](https://github.com/philippe-desplats/hop/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/philippe-desplats/hop)](https://goreportcard.com/report/github.com/philippe-desplats/hop)

A fast, AI-agnostic project switcher for your terminal. Jump to any project by
frecency, then drop into your editor, your AI assistant, or a custom action,
without retyping paths.

Where `zoxide` stops at `cd`, `hop` keeps going: one keystroke takes you from
"which project was that again" to "open, and start working".

```text
$ p api
  ~/code/acme-api

$ p              # no argument opens the interactive Hub
  ┌ hop ─────────────────────────────────────────────┐
  │ > acme-api          ~/code/acme-api               │
  │   web-shop          ~/code/web-shop               │
  │   blog              ~/code/side/blog              │
  └───────────────────────────────────────────────────┘
  enter cd · c Claude · z editor · g git · f Finder
```

## Why

`hop` is a single binary that prints a target; a small shell function named `p`
performs the `cd` (a child process cannot change its parent shell's working
directory, the same model `zoxide` uses). Jumps learn from your habits through
frecency (frequency plus recency).

## Features

- Instant jump by keyword, ranked by frecency.
- Ordered multi-keyword matching to narrow by sub-path (`p acme web`).
- Interactive fuzzy Hub with an action menu (cd, editor, AI assistant, git,
  remote, Finder, tmux).
- `p -` to jump back to the previous project.
- TOML configuration with an interactive editor (`hop config`).
- Adaptive light and dark themes, and UI in English, French, Spanish and
  Portuguese.
- Works in zsh, bash and fish, with TAB-completion of project names.

## Installation

```sh
go install github.com/philippe-desplats/hop/cmd/hop@latest
```

A Homebrew tap is planned for the first tagged release.

Then wire the shell integration into your shell startup file:

```sh
eval "$(hop init zsh)"          # zsh   (~/.zshrc)
eval "$(hop init bash)"         # bash  (~/.bashrc)
hop init fish | source          # fish  (~/.config/fish/config.fish)
```

`--cmd NAME` picks a different function name (default `p`), and `p <TAB>`
completes project names.

Open a new shell. The first invocation indexes your project roots automatically.

## Usage

```sh
p api            # jump straight to the best match (e.g. acme-api)
p web            # -> ~/code/web-shop
p acme web       # multiple keywords: narrow by ordered sub-path
p -              # jump back to the previous project
p                # open the interactive Hub (fuzzy list, up/down + Enter = cd)
hop scan         # reindex on demand
hop doctor       # configuration diagnostics
```

Keywords are ordered and each must appear in the path (the `zoxide` model). The
last keyword is favored against the project name. A freshly created project is
found automatically: on a miss, `hop` reindexes once and retries before giving
up.

## Configuration

`hop config` opens an interactive editor (up/down to move, left/right to select,
type to edit text fields, `enter` or `ctrl+s` to save). The file
`~/.config/hop/config.toml` is created automatically:

```toml
[ui]
language = "auto"        # auto (from $LANG), or en / fr / es / pt
theme = "auto"           # auto (detect terminal background), or light / dark

[shell]
command = "p"            # name of the daily shortcut

[hub]
# tab: Tab opens the menu · shift: UPPERCASE keys act directly · enter: Enter opens the menu
action_access = "tab"

[actions]
editor = "zed"           # command for the "open in editor" action
show_tmux = false        # show the tmux action in the menu

[scan]
roots = ["~/code"]       # where to look for projects
max_depth = 7
ignore = ["node_modules", "vendor", "_archives"]
```

Most settings are re-read on every `p`, so they take effect immediately. The
command name (`[shell] command`) is read when the integration is sourced, so it
takes effect on the next shell start.

## How it works

- A git-repo-first scanner walks your roots, treating each git repository (and
  any leaf project folder) as a target.
- Frecency ranks targets by how often and how recently you visit them.
- The Hub (Bubble Tea) renders the fuzzy list and the per-project action menu.
- State (index, frecency) is owned by a single store with atomic, lock-guarded
  JSON writes.

## Development

```sh
go build ./...
go test ./...
go vet ./...
```

## Security

See [SECURITY.md](SECURITY.md) for how to report a vulnerability and the
project's supply-chain posture.

## License

[MIT](LICENSE)
