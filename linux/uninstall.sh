#!/bin/sh
# Remove tuigo files installed by install.sh. Config in
# ~/.config/tuigo/ is left alone.
set -eu

PREFIX="${PREFIX:-$HOME/.local/lib/tuigo}"
COMPONENT_DIR="${XDG_DATA_HOME:-$HOME/.local/share}/ibus/component"
SUDO=""
if [ "${1:-}" = "--system" ]; then
	PREFIX=/usr/lib/tuigo
	COMPONENT_DIR=/usr/share/ibus/component
	SUDO=sudo
fi

$SUDO rm -f "$COMPONENT_DIR/tuigo.xml"
$SUDO rm -rf "$PREFIX"
echo "Removed $PREFIX and $COMPONENT_DIR/tuigo.xml"
echo "Restart ibus (ibus restart) and remove the Tui Go input source."
