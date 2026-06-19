#!/bin/sh
# hop installer for systems without Homebrew or Go.
#
#   curl -fsSL https://raw.githubusercontent.com/philippe-desplats/hop/master/install.sh | sh
#
# Environment overrides:
#   HOP_VERSION       tag to install (e.g. v2.1.0); default: latest release
#   HOP_INSTALL_DIR   target directory; default: ~/.local/bin
#
# It downloads the release tarball for your OS/arch, verifies its SHA-256 against
# the published checksums, installs the binary, and drops the macOS quarantine
# flag. It does not modify your shell config: run `hop setup` afterwards.
set -eu

REPO="philippe-desplats/hop"
INSTALL_DIR="${HOP_INSTALL_DIR:-$HOME/.local/bin}"

err() { printf 'hop-install: %s\n' "$1" >&2; exit 1; }
info() { printf '%s\n' "$1"; }

[ -n "${HOME:-}" ] || err "HOME is not set"
command -v uname >/dev/null 2>&1 || err "required tool not found: uname"
command -v tar >/dev/null 2>&1 || err "required tool not found: tar"
if command -v curl >/dev/null 2>&1; then
  fetch() { curl -fsSL -o "$1" "$2"; }
  fetch_url() { curl -fsSLI -o /dev/null -w '%{url_effective}' "$1"; }
elif command -v wget >/dev/null 2>&1; then
  fetch() { wget -qO "$1" "$2"; }
  fetch_url() { wget -qS --max-redirect=10 -O /dev/null "$1" 2>&1 | awk '/^  Location: /{u=$2} END{print u}'; }
else
  err "need curl or wget"
fi

os="$(uname -s)"
case "$os" in
  Darwin) os="darwin" ;;
  Linux) os="linux" ;;
  *) err "unsupported OS: $os (use Homebrew or 'go install' instead)" ;;
esac

arch="$(uname -m)"
case "$arch" in
  x86_64 | amd64) arch="amd64" ;;
  arm64 | aarch64) arch="arm64" ;;
  *) err "unsupported architecture: $arch" ;;
esac

version="${HOP_VERSION:-}"
if [ -z "$version" ]; then
  version="$(fetch_url "https://github.com/$REPO/releases/latest" | sed 's#.*/tag/##')"
  [ -n "$version" ] || err "could not determine the latest version; set HOP_VERSION"
fi

num="${version#v}" # the asset filename carries the version without a leading v
asset="hop_${num}_${os}_${arch}.tar.gz"
base="https://github.com/$REPO/releases/download/$version"

info "Installing hop $version ($os/$arch) to $INSTALL_DIR"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

fetch "$tmp/$asset" "$base/$asset" || err "download failed: $base/$asset"
fetch "$tmp/checksums.txt" "$base/checksums.txt" || err "could not download checksums.txt"

(
  cd "$tmp"
  line="$(grep " $asset\$" checksums.txt || true)"
  if [ -z "$line" ]; then
    err "no checksum entry for $asset"
  elif command -v sha256sum >/dev/null 2>&1; then
    printf '%s\n' "$line" | sha256sum -c - >/dev/null 2>&1 || err "checksum mismatch for $asset"
  elif command -v shasum >/dev/null 2>&1; then
    printf '%s\n' "$line" | shasum -a 256 -c - >/dev/null 2>&1 || err "checksum mismatch for $asset"
  else
    info "warning: no sha256 tool found, skipping checksum verification"
  fi
)

tar -xzf "$tmp/$asset" -C "$tmp" hop || err "could not extract hop from $asset"

mkdir -p "$INSTALL_DIR"
mv "$tmp/hop" "$INSTALL_DIR/hop"
chmod +x "$INSTALL_DIR/hop"

# The binary is ad-hoc signed but not notarized; drop any quarantine flag so a
# clean macOS Gatekeeper does not block it.
if [ "$os" = "darwin" ] && command -v xattr >/dev/null 2>&1; then
  xattr -d com.apple.quarantine "$INSTALL_DIR/hop" 2>/dev/null || true
fi

info ""
info "hop $version installed to $INSTALL_DIR/hop"

case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *)
    info ""
    info "$INSTALL_DIR is not on your PATH. Add this to your shell config:"
    info "  export PATH=\"$INSTALL_DIR:\$PATH\""
    ;;
esac

info ""
info "Next, run the guided setup (folders, editor, assistant):"
info "  hop setup"
