# hop setup wizard demo playground.
#
# SOURCE this file (do not execute it):
#
#     source sample/setup-demo.sh
#
# It builds a throwaway $HOME with a few project folders and stub tools, then
# points hop at it, so `hop setup` shows a clean, machine-independent wizard.
# Your real home, config, index and PATH are never touched: everything lives in
# a temp dir. Open a fresh shell afterwards to restore your environment.
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

_HOP_REAL_HOP="$(command -v hop)" # capture before PATH is narrowed below

_HOP_TMP="$(mktemp -d 2>/dev/null || echo "/tmp/hop-setup-demo.$$")"
_HOP_HOME="$_HOP_TMP/home"
_HOP_BIN="$_HOP_TMP/bin"
mkdir -p "$_HOP_HOME" "$_HOP_BIN"

# A few project folders with git repos, including a non-standard name (~/Labs)
# that only the by-content discovery can find.
_hop_seed() {
  mkdir -p "$_HOP_HOME/$1"
  (
    cd "$_HOP_HOME/$1" || return
    git init -q
    git -c user.name=demo -c user.email=demo@example.com commit -q --allow-empty -m init
  )
}
_hop_seed "Projects/acme-api"
_hop_seed "Projects/web-monorepo"
_hop_seed "Developments/side-blog"
_hop_seed "Labs/llm-playground"

# Stub editors and an AI CLI so the wizard shows a clean, deterministic list.
# They are only ever detected on PATH, never executed by setup.
cp "$_HOP_REAL_HOP" "$_HOP_BIN/hop"
for _t in cursor code zed claude; do
  printf '#!/bin/sh\n' >"$_HOP_BIN/$_t"
  chmod +x "$_HOP_BIN/$_t"
done

# Isolate everything: throwaway home, config and state; a controlled PATH (only
# the fresh hop and stubs are visible); a known shell so the wizard offers
# ~/.zshrc; and an empty rc so it offers to wire the integration.
export HOME="$_HOP_HOME"
export XDG_CONFIG_HOME="$_HOP_HOME/.config"
export XDG_STATE_HOME="$_HOP_HOME/.local/state"
export SHELL="/bin/zsh"
export PATH="$_HOP_BIN:/usr/bin:/bin"
: >"$_HOP_HOME/.zshrc"
