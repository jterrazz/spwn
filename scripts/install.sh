#!/bin/sh
set -e

REPO="jterrazz/studio-station"
BINARY="station"
INSTALL_DIR="$HOME/.local/bin"

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH" && exit 1 ;;
esac

echo "Installing $BINARY ($OS/$ARCH)..."

ASSET="$BINARY-$OS-$ARCH"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

# value: the machine tiers install the binary they just built, from a local file,
# with no tag read and no checksum; everything after the download is the same.
if [ -n "${STATION_INSTALL_FROM:-}" ]; then
  if [ ! -f "$STATION_INSTALL_FROM" ]; then
    echo "STATION_INSTALL_FROM: no file at $STATION_INSTALL_FROM" && exit 1
  fi
  TAG="local"
  cp "$STATION_INSTALL_FROM" "$WORK/$ASSET"
else
  TAG="$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p')"
  if [ -z "$TAG" ]; then
    echo "Failed to fetch latest release" && exit 1
  fi

  BASE_URL="https://github.com/$REPO/releases/download/$TAG"
  curl -fsSL "$BASE_URL/$ASSET" -o "$WORK/$ASSET"
  curl -fsSL "$BASE_URL/checksums.txt" -o "$WORK/checksums.txt"

  # invariant: a binary whose line is missing from checksums.txt, or whose hash differs, is never installed.
  EXPECTED="$(awk -v asset="$ASSET" '$2 == asset { print $1 }' "$WORK/checksums.txt")"
  if [ -z "$EXPECTED" ]; then
    echo "checksums.txt has no line for $ASSET" && exit 1
  fi
  if command -v sha256sum >/dev/null 2>&1; then
    ACTUAL="$(sha256sum "$WORK/$ASSET" | awk '{ print $1 }')"
  else
    ACTUAL="$(shasum -a 256 "$WORK/$ASSET" | awk '{ print $1 }')"
  fi
  if [ "$EXPECTED" != "$ACTUAL" ]; then
    echo "Checksum mismatch for $ASSET" && exit 1
  fi
fi

mkdir -p "$INSTALL_DIR"
mv "$WORK/$ASSET" "$INSTALL_DIR/$BINARY"
chmod +x "$INSTALL_DIR/$BINARY"
rm -f "$HOME/.studio/bin/workstation"
echo "✓ Installed $INSTALL_DIR/$BINARY ($TAG)"

# outside: ~/.local/bin may not be on PATH yet on a fresh machine, so init runs by path
# and takes the arguments of this script (sh -s -- --name macbook). Piped from curl, init
# asks its questions on the terminal. With none, a Mac with no name yet is refused
# (exit 2) with the command that names it, and the install stands.
if [ -t 1 ] && (: </dev/tty) 2>/dev/null; then
  "$INSTALL_DIR/station" init "$@" </dev/tty
else
  status=0
  "$INSTALL_DIR/station" init "$@" || status=$?
  [ "$status" -eq 0 ] || [ "$status" -eq 2 ] || exit "$status"
fi

echo "✓ Done. Run 'exec zsh', then 'station'"
