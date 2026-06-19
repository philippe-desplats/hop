# hop demo playground.
#
# SOURCE this file (do not execute it) so the `p` function and `cd` work in your
# current shell:
#
#     source sample/demo.sh        # zsh or bash
#
# It scans ONLY a throwaway copy of sample/projects (in ~/hop-demo) and keeps
# hop's index in a temp dir, so your real ~/.config/hop and project index are
# never touched. Open a fresh shell afterwards to return to your normal setup.
#
# Note: no `set -euo pipefail` here on purpose, this is sourced into your
# interactive shell and must not change its options or exit it on error.

# Resolve sample/ directory whether sourced from bash or zsh.
if [ -n "${BASH_SOURCE:-}" ]; then
  _hop_src="${BASH_SOURCE[0]}"
elif [ -n "${ZSH_VERSION:-}" ]; then
  _hop_src="${(%):-%x}"
else
  _hop_src="$0"
fi
_HOP_SAMPLE_DIR="$(cd "$(dirname "$_hop_src")" && pwd)"

HOP_DEMO_DIR="${HOP_DEMO_DIR:-$HOME/hop-demo}"
_HOP_STATE="$(mktemp -d 2>/dev/null || echo "/tmp/hop-demo-state.$$")"

export XDG_CONFIG_HOME="$_HOP_STATE/config"
export XDG_STATE_HOME="$_HOP_STATE/state"
mkdir -p "$XDG_CONFIG_HOME/hop"

# Backdated timestamp helper: BSD/macOS `date -v`, GNU `date -d`.
_hop_days_ago() {
  date -v-"$1"d '+%Y-%m-%dT%H:%M:%S' 2>/dev/null || date -d "$1 days ago" '+%Y-%m-%dT%H:%M:%S'
}

# Fresh copy of the sample projects.
rm -rf "$HOP_DEMO_DIR"
mkdir -p "$HOP_DEMO_DIR"
cp -R "$_HOP_SAMPLE_DIR/projects/." "$HOP_DEMO_DIR/"

# Turn a few of them into git repos with backdated commits, so the Hub shows the
# localized git preview (branch + age). The others stay plain folders.
_hop_seed_repo() {
  _hop_dir="$1"
  _hop_age="$2"
  _hop_branch="${3:-main}"
  (
    cd "$HOP_DEMO_DIR/$_hop_dir" || return
    git init -q -b "$_hop_branch" 2>/dev/null || { git init -q; git checkout -q -b "$_hop_branch" 2>/dev/null; }
    git add -A
    GIT_AUTHOR_DATE="$(_hop_days_ago "$_hop_age")" GIT_COMMITTER_DATE="$(_hop_days_ago "$_hop_age")" \
      git -c user.name=demo -c user.email=demo@example.com commit -q -m "init"
  )
}
_hop_seed_repo work/acme-api 0
_hop_seed_repo work/web-monorepo 3
_hop_seed_repo side/blog 14 trunk
_hop_seed_repo experiments/llm-playground 1

# Show the scan root as ~/... in `hop config` (hop expands ~ at load time, so the
# scan still works) instead of an absolute /Users/... path that would leak a
# username into a recording, when the demo dir lives under $HOME.
case "$HOP_DEMO_DIR" in
  "$HOME"/*) _hop_root_conf="~/${HOP_DEMO_DIR#"$HOME"/}" ;;
  *) _hop_root_conf="$HOP_DEMO_DIR" ;;
esac

# Demo config: index only the demo dir. English + dark theme so a recorded GIF is
# deterministic regardless of your locale and terminal background.
cat > "$XDG_CONFIG_HOME/hop/config.toml" <<EOF
[ui]
language = "en"
theme = "dark"
[shell]
command = "p"
[hub]
action_access = "tab"
[actions]
editor = "zed"
show_tmux = false
[scan]
roots = ["$_hop_root_conf"]
max_depth = 5
ignore = ["node_modules", "vendor"]
EOF

# Wire the `p` function for the current shell only.
if [ -n "${ZSH_VERSION:-}" ]; then
  eval "$(command hop init zsh)"
else
  eval "$(command hop init bash)"
fi

command hop scan >/dev/null 2>&1

# Pre-warm frecency so the ranking looks lived-in.
command hop add "$HOP_DEMO_DIR/work/acme-api" >/dev/null 2>&1
command hop add "$HOP_DEMO_DIR/work/web-monorepo" >/dev/null 2>&1

printf '\n  hop demo ready (indexing only %s)\n' "$HOP_DEMO_DIR"
printf '  try:  p api   .   p   .   p web   .   Tab opens actions   .   p -\n\n'
