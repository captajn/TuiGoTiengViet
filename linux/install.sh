#!/bin/sh
# Install tuigo (IBus Vietnamese IME) for the current user.
#
#   ./install.sh            user install, no sudo (recommended)
#   ./install.sh --system   system-wide install under /usr (needs sudo)
#   ./uninstall.sh          remove
#
# Builds the binaries with `go build` when the go toolchain is present;
# otherwise expects prebuilt binaries in ./dist (release tarball layout).
set -eu

PREFIX="${PREFIX:-$HOME/.local/lib/tuigo}"
COMPONENT_DIR="${XDG_DATA_HOME:-$HOME/.local/share}/ibus/component"
SUDO=""
if [ "${1:-}" = "--system" ]; then
	PREFIX=/usr/lib/tuigo
	COMPONENT_DIR=/usr/share/ibus/component
	SUDO=sudo
fi

HERE=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
cd "$HERE"

if command -v go >/dev/null 2>&1; then
	echo "==> building tuigo-ibus"
	go build -trimpath -ldflags "-s -w" -o dist/tuigo-ibus ./cmd/tuigo-ibus
	echo "==> building tuigo-settings"
	(cd settings && go build -trimpath -ldflags "-s -w" -o ../dist/tuigo-settings .)
else
	echo "==> go not found; using prebuilt binaries in ./dist"
fi
test -x dist/tuigo-ibus || { echo "missing dist/tuigo-ibus"; exit 1; }

$SUDO mkdir -p "$PREFIX" "$COMPONENT_DIR"
$SUDO cp dist/tuigo-ibus "$PREFIX/tuigo-ibus"
if [ -x dist/tuigo-settings ]; then $SUDO cp dist/tuigo-settings "$PREFIX/tuigo-settings"; fi
$SUDO cp data/logo.jpg "$PREFIX/logo.jpg"
sed "s|@PREFIX@|$PREFIX|g" data/tuigo.xml | $SUDO tee "$COMPONENT_DIR/tuigo.xml" >/dev/null

echo "==> installed to $PREFIX"
echo "Now restart ibus:  ibus restart"
echo "Then add 'Tui Go (Vietnamese)' in Settings -> Keyboard -> Input Sources."
echo "Config dir: ${XDG_CONFIG_HOME:-$HOME/.config}/tuigo/ (tuigo.ini, macro.txt, keymap.txt)"
