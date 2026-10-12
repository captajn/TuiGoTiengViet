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
