# Mobile verification

- Read plan.md and progress.md before resuming a staged mobile task. Respect the user's approved scope.
- Run `powershell -File scripts/check-mobile.ps1` after Android changes. Add `-Device -Serial emulator-5554` only with a prepared test emulator; this installs debug APKs and runs instrumented checks. Do not uninstall or overwrite a user's physical-device app without authorization.
- Run `sh scripts/check-ios.sh` on macOS after iOS changes. Windows source review is not evidence that UIKit/SwiftUI compile or render correctly.
- Android RadioGroup: assign each child a unique View ID, check the persisted selection after all children are added, then attach the group listener. Confirm exactly one selection and that the live engine uses it.
- The mobile input-method selector mirrors the desktop's three primary modes: Telex, VNI, VIQR. Normalize legacy simple/hybrid prefs to Telex. VIQR needs the extra symbols page (#+=) for ^, ~ and grave accent; test the real keyboard as well as the engine.
- System navigation insets are reserved space, not app-owned touch targets. Do not draw a duplicate IME hide/globe control there; verify gesture and three-button navigation with real screenshots.
- Macro UI must validate before writing, report success only after a successful atomic write, and preserve existing data on I/O failure. The IME must reload edited, emptied or removed lists on activation.
- StdVnChar reserves values above BMP; encode astral macro text as UTF-16 surrogate units. Run the Unicode/reload regression tests on both ports when changing macro handling.
- Keep light/dark palette resources and native theme tokens consistent. Review screenshots after layout changes; build and unit tests do not establish visual correctness.
- Mobile colors follow the shipped Windows jade style in app/theme.go: light accent #197F64 on #F8FAFC; dark accent #3ED3B8 on #0F1115. Keep Android resources and iOS adaptive colors in sync; do not switch back to purple without a user request.
- Use the actual Windows mascot from app/logo.jpg for mobile branding. `scripts/sync-mobile-logo.ps1` copies the exact logo and creates opaque iOS icon sizes; check-mobile/check-ios validate logo hashes and icon metadata. Keep generated mobile assets tracked; do not replace the logo with a generic Android icon.
- Record pass/fail/unresolved and real evidence in checks.md. Retry a failed step at most three times; do not claim unavailable iOS checks passed.

- Mobile release metadata comes from scripts/mobile-release.py: Android/iOS versions must match the mobile-v* tag, filenames tuigo-vX.Y.Z.apk/.ipa, display title Tui Gõ Mobile vX.Y.Z. Check out the release tag when dispatching; never package an arbitrary APK variant. Inspect the embedded keyboard extension and mark unsigned IPA explicitly.
- Use scripts/publish-mobile-tag.ps1 to check local AND remote tag targets and stop on every Git failure before pushing. A missing remote tag does not mean the local tag is absent. Preserve a preexisting local tag; never continue a push after tag creation fails.
- Symbol pages include / and * plus the Gboard reference symbols. Keep VIQR marks and existing currencies/brackets reachable; verify actual touch input and page transitions on the built keyboard.

# macOS + Linux ports

- macOS (`macos/`): InputMethodKit app reusing ios/Sources/TuiGoEngine (SwiftPM, macOS 12). Single input mode `com.tuigo.inputmethod.TuiGo`; VN/EN is a session flag toggled via input menu or Ctrl+Shift/Alt+Z. Verify with `sh scripts/check-macos.sh` on macOS.
- Linux (`linux/`): IBus engine in Go (`cmd/tuigo-ibus`, module tuigo-linux, `replace tuigo => ../app`). Settings GUI `settings/` is a separate Go module (fyne.io/fyne needs cgo; godbus must stay v5.1.0 for goibus — newer godbus drops AuthCookieSha1 on Unix). Verify with `sh scripts/check-linux.sh` on Ubuntu.
- IME shells can't delete committed text (IBus/IMK have no backchannel backspace): keep the word in preedit/marked text until a word boundary. `Session` (linux/internal/session, macos/Sources/Session.swift) handles this — swallow unconsumed mid-word chars into the model instead of forwarding.
- Engine returns Consumed=false with Backs>0 for plain backspace — consume it yourself when the deleted char lives in preedit.
- Shared config: tuigo.ini (same key names as Windows) + macro.txt + keymap.txt. Linux: ~/.config/tuigo/, macOS: ~/Library/Application Support/TuiGo/.

# Contributor metadata

- Do not add the removed Devin bot co-author trailer to new commits. For this cleanup, run `powershell -File scripts/check-contributors.ps1` to verify the isolated candidate against its commit map. Preserve release tags and use an explicit force-with-lease for any approved history rewrite; align local branch refs only after proving identical trees and preserving working file hashes.

# Privacy and diagnostics

- User requires no app-owned diagnostic logging on Windows, Linux, macOS, Android or iOS, and no debug-log setting. Do not reintroduce crash.log writers, hook/heartbeat output, stderr redirection or updater exit monitoring. Keep Win32 callback panic containment and input-hook recovery independent of logging.
- Run `python -P scripts/check-no-diagnostics.py` after product changes; native platform check scripts also run it. It rejects diagnostic/telemetry sinks and input-app HTTP clients, and verifies Android has no INTERNET permission/backup and iOS does not request Full Access. Only the separate optional Windows updater may access the network.
- Run `powershell -File scripts/check-diagnostics.ps1` after Windows/updater changes. Preserve existing logs and local user edits; do not remove user files as part of disabling logging. OS/framework crash reporting is separate from app-owned diagnostics and must not be disabled or described as controlled by this source guard.
- README must describe all five platforms and actual local config/macro/per-app data, optional updater traffic, clipboard behavior and iOS access limitations. Do not claim absolute privacy, signed iOS artifacts or native/runtime verification without evidence.
