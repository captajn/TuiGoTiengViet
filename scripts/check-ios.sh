#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$repo/ios"
python3 -P "$repo/scripts/check-no-diagnostics.py"
python3 -P "$repo/scripts/check-palette.py"
swift test
command -v xcodegen >/dev/null
command -v xcodebuild >/dev/null
xcodegen
xcodebuild -project TuiGo.xcodeproj -scheme TuiGo \
  -destination 'generic/platform=iOS Simulator' -configuration Debug \
  -derivedDataPath "$repo/build/mobile-redesign/ios" \
  build CODE_SIGNING_ALLOWED=NO CODE_SIGNING_REQUIRED=NO
