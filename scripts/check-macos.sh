#!/bin/sh
# macOS (InputMethodKit) build check — mirrors check-ios.sh.
# Runs on macOS only: swift test + xcodegen + unsigned xcodebuild, then
# packages the .app bundle. Icons are generated from app/logo.jpg into
# macos/Resources/ before project generation so they get bundled.
set -eu
repo=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)

# engine unit tests (shared Swift package)
cd "$repo/ios" && swift test

cd "$repo/macos"
command -v xcodegen >/dev/null
command -v xcodebuild >/dev/null

# icons from the shipped mascot (app/logo.jpg) — same source as mobile.
iconset=build/AppIcon.iconset
mkdir -p "$iconset" build
sips -s format tiff -Z 32 "$repo/app/logo.jpg" --out Resources/icon.tiff >/dev/null
for px in 16 32 128 256 512; do
  sips -Z $px "$repo/app/logo.jpg" --out "$iconset/icon_${px}x${px}.png" >/dev/null
  dbl=$((px*2))
  sips -Z $dbl "$repo/app/logo.jpg" --out "$iconset/icon_${px}x${px}@2x.png" >/dev/null
done
iconutil -c icns "$iconset" -o Resources/AppIcon.icns

xcodegen
xcodebuild -project TuiGoMac.xcodeproj -scheme TuiGoMac \
  -configuration Release -derivedDataPath "$repo/build/macos" \
  build CODE_SIGNING_ALLOWED=NO CODE_SIGNING_REQUIRED=NO

appdir="$repo/build/macos/Build/Products/Release/Tui Go.app"
test -d "$appdir"
test -f "$appdir/Contents/Resources/icon.tiff"
test -f "$appdir/Contents/Resources/AppIcon.icns"
echo "check-macos: PASS -> $appdir"
