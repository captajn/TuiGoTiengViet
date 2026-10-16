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
    mkdir -p "$HOME/.local/share/ibus/component"
    sed "s|@PREFIX@|'"$repo"'/linux/dist|g" data/tuigo.xml \
      > "$HOME/.local/share/ibus/component/tuigo.xml"
    if command -v sudo >/dev/null; then
      sudo cp "$HOME/.local/share/ibus/component/tuigo.xml" /usr/share/ibus/component/
    fi
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

# fcitx5 addon (Sen mode) — builds when fcitx5 dev files exist; the Sen
# machine + test build regardless, the addon target skips without them.
if command -v cmake >/dev/null && pkg-config --exists fcitx5core 2>/dev/null; then
  cmake -S "$repo/fcitx5" -B "$repo/fcitx5/build" -DCMAKE_BUILD_TYPE=Release
  cmake --build "$repo/fcitx5/build" -j
  (cd "$repo/fcitx5/build" && ctest --output-on-failure)
else
  echo "check-linux: SKIP fcitx5 addon (fcitx5 dev files not installed)"
fi

echo "check-linux: PASS"
