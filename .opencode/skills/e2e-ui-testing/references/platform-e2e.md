# Platform E2E & check scripts — TuiGoTiengViet

Chạy đúng check script của nền tảng vừa sửa. Mọi script đều chạy
`python -P scripts/check-no-diagnostics.py` trước.

| Nền tảng | Lệnh | Chạy cái gì |
|---|---|---|
| Windows | `powershell -File scripts/check-diagnostics.ps1` | no-diagnostics + `GOWORK=off go test ./...` + `go vet ./...` trong `app/` |
| Android | `powershell -File scripts/check-mobile.ps1` | no-diagnostics + `check-palette.py` + Gradle `:engine:test :app:assembleDebug :app:assembleDebugAndroidTest :app:lintDebug` (log: `build/mobile-redesign/android-final.log`) |
| Android + device | `powershell -File scripts/check-mobile.ps1 -Device -Serial emulator-5554` | cài `app-debug.apk` + `app-debug-androidTest.apk`, chạy `am instrument -w com.tuigo.tiengviet.test/com.tuigo.MobileSmokeTest`, yêu cầu output `PASS: 3 exclusive selections` |
| Android release | `check-mobile.ps1 -Release` | cần `keystore.properties`; **bắt buộc** `android/keystore/tuigo-release.jks`, build từ `:app:assembleRelease` |
| iOS | `sh scripts/check-ios.sh` (macOS) | no-diagnostics + palette + `swift test` + `xcodegen` + `xcodebuild` Debug simulator (unsigned) |
| macOS | `sh scripts/check-macos.sh` (macOS) | no-diagnostics + `swift test` + icon từ `app/logo.jpg` + `xcodebuild` Release unsigned, assert bundle `Tui Go.app` |
| Linux | `sh scripts/check-linux.sh` (Ubuntu) | no-diagnostics + `go test ./...` trong `linux/` + build `tuigo-ibus` + Fyne settings + e2e thật |
| Palette/brand | `python -P scripts/check-palette.py` | 7 token màu khớp `app/theme.go` ↔ `UiTheme.kt` ↔ `colors.xml` ↔ `GlassTheme.swift`; hash `app/logo.jpg`; icon iOS |

Cũng chạy `scripts/check-contributors.ps1` khi đụng commit metadata, và
`powershell -File scripts/check-diagnostics.ps1` khi sửa Windows/updater.

## Test Go hiện có

```powershell
cd app
$env:GOWORK="off"; go test ./... -v        # main + engine + updater
$env:GOWORK="off"; go test ./engine/       # engine thuần, nhanh
```

- `app/engine/engine_test.go` — 15 test Telex/VNI/macro/TCVN/backspace; harness `docSim`.
- `app/tray_test.go` — icon, hit-test sidebar (cần phiên desktop tương tác; tự skip nếu không có DC).
- `app/app_mode_test.go` — remember app mode; pattern lưu/restore global.
- `app/updater/main_test.go` — updater không ghi diagnostics.
- `linux/internal/session/session_test.go` — preedit session IBus.
- `ios/Tests/TuiGoEngineTests/EngineTests.swift` — `swift test` trong `ios/`.

Không có build tag nào; package `main` chỉ biên dịch trên Windows (import
`golang.org/x/sys/windows`).

## Android UI test — pattern trong `MobileSmokeTest.kt`

Framework-free: dùng `Instrumentation` + view tree thật.

- Tìm view: `findViewWithTag("input-methods" | "bottom-nav" | "nav-tab-typing" |
  "nav-tab-theme" | "nav-tab-macro" | "keyboard-themes" | "macro-editor" |
  "macro-feedback")`; tìm nút theo text đệ quy (ví dụ `"Lưu gõ tắt"`).
- Click: `performClick()` trên view thật; assert RadioGroup độc quyền
  (`PASS: 3 exclusive selections`), persist prefs, macro save/validate/
  reject/write-failure (tạo `macro.txt.new` thành thư mục để gây lỗi ghi).
- **Restore toàn bộ prefs/file trong `finally`** — không để lại trạng thái trên máy.

## Linux e2e — `linux/test/e2e.py`

Chạy qua `check-linux.sh` với `xvfb-run dbus-run-session` + `ibus-daemon` thật:

- Tạo `IBus.InputContext`, feed key event giả (`ic.process_key_event`).
- Nghe signal `commit-text` / `update-preedit-text`, assert:
  `"vieejt "` → `"việt "`, `"dduowngf xin chaof "` → `"đường xin chào "`.
- Pattern: `type_str` + `check(name, got, want)`, exit 1 nếu fail.

## Screenshot & evidence

- Thay đổi layout ⇒ xem screenshot thật (Android emulator, iOS simulator,
  Windows app, Linux under xvfb). Build pass không chứng minh UI render đúng.
- Điều hướng system (gesture / 3-button nav) trên Android: verify bằng screenshot
  thật, không suy luận từ code.
- Ghi vào `checks.md`: pass/fail/unresolved, lệnh đã chạy, output chính, tên
  screenshot. Retry bước fail tối đa 3 lần. Không claim check iOS/macOS pass nếu
  chưa chạy trên macOS. Windows source review không phải bằng chứng UIKit/SwiftUI
  compile/render.
