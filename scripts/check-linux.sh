#!/bin/sh
# Linux (IBus) build + smoke check — mirrors check-ios.sh / check-macos.sh.
# Debian/Ubuntu packages needed:
#   go, python3-gi, gir1.2-ibus-1.0, ibus, dbus-x11, xvfb,
#   libgl1-mesa-dev xorg-dev  (Fyne settings GUI)
set -eu
repo=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
python3 -P "$repo/scripts/check-no-diagnostics.py"
cd "$repo/linux"
export GOWORK=off

mkdir -p dist
go test ./...
go build -trimpath -o dist/tuigo-ibus ./cmd/tuigo-ibus

# settings GUI — needs Fyne's cgo deps; skip gracefully if missing
if pkg-config --exists gl 2>/dev/null; then
  (cd settings && go build -trimpath -o ../dist/tuigo-settings .)
else
  echo "check-linux: SKIP tuigo-settings (GL dev headers not installed)"
fi

# component XML sanity: substitute the dist prefix and parse it
sed "s|@PREFIX@|$repo/linux/dist|g" data/tuigo.xml > dist/tuigo.xml
python3 -c "import xml.dom.minidom,sys; xml.dom.minidom.parse('dist/tuigo.xml')"

# end-to-end against a real ibus-daemon when the deps exist — install the
# component into the user component dir like install.sh does, then let
# ibus-daemon spawn the engine binary itself.
if command -v ibus-daemon >/dev/null && command -v xvfb-run >/dev/null \
   && command -v dbus-run-session >/dev/null && python3 -c \
   'import gi; gi.require_version("IBus","1.0")' 2>/dev/null; then
  cp data/logo.jpg dist/logo.jpg
  xvfb-run -a dbus-run-session -- sh -c '
    export XDG_CONFIG_HOME="$(mktemp -d)"   # keep e2e macro/ini writes out of the real config
    mkdir -p "$HOME/.local/share/ibus/component"
    sed "s|@PREFIX@|'"$repo"'/linux/dist|g" data/tuigo.xml \
      > "$HOME/.local/share/ibus/component/tuigo.xml"
    ibus-daemon -rv &
    sleep 3
    ibus list-engine 2>/dev/null | grep -i tuigo || echo "no tuigo engine listed"
    python3 test/e2e.py; rc=$?
    pgrep -af tuigo-ibus || echo "tuigo-ibus not running"
    exit $rc
  '
else
  echo "check-linux: SKIP ibus e2e (ibus-daemon/xvfb/python-gi missing)"
fi

echo "check-linux: PASS"
