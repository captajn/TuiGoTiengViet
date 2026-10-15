---
name: E2E & UI Testing
description: E2E và test giao diện cho TuiGoTiengViet — nhấn nút/settings UI Win32 (SendMessage WM_COMMAND), instrumented UI test Android, e2e IBus Linux, chụp screenshot xác minh visual, ghi evidence vào checks.md. Dùng khi test end-to-end, test UI, mô phỏng nhấn nút, hoặc xác minh layout.
---

# E2E & UI Testing — TuiGoTiengViet

Khi nào dùng skill này: cần test end-to-end, test giao diện, mô phỏng nhấn nút
(settings UI Windows, bàn phím Android/iOS, IBus Linux), hoặc xác minh layout sau
khi thay đổi giao diện.

## Nguyên tắc bắt buộc

1. **Build/unit test pass ≠ UI đã đúng.** Sau thay đổi layout phải xem screenshot
   thật (AGENTS.md: "Review screenshots after layout changes").
2. **Ghi evidence vào `checks.md`**: pass/fail/unresolved + bằng chứng thật
   (tên test, output, screenshot). Bước fail được retry tối đa 3 lần. Không claim
   check iOS/macOS đã pass nếu không chạy được trên máy macOS.
3. **Không thêm diagnostic logging** để debug (xem skill `debug-go`). Test file
   (`*_test.go`, `*Test.kt`, test Swift) được phép `fmt.Print`/`t.Log`.
4. **Không ghi đè app trên thiết bị thật của user** khi chạy instrumented test
   (check-mobile `-Device` chỉ dùng emulator đã chuẩn bị).

## Workflow tổng quát

1. Xác định nền tảng chứa thay đổi → chạy check script tương ứng
   (bảng ở `references/platform-e2e.md`).
2. Với bug UI/nút bấm: viết test tái hiện **trước** khi sửa (red → fix → green).
3. Chạy test thu hẹp (`-run`, `--tests`, filter Gradle) rồi chạy full check script.
4. Chụp/xem screenshot nếu layout thay đổi; ghi kết quả vào `checks.md`.

## Windows — nhấn nút trong settings UI

App dùng Win32 thuần (`app/settings_ui.go`, `app/winapi.go`). Nguyên tắc:
**test in-process trong `package main`** (giống `app/tray_test.go`,
`app/app_mode_test.go`) và "nhấn nút" bằng `SendMessage(gSettingsHwnd, WM_COMMAND, id, ctlHnd[id])`
— `SendMessage` gọi `settingsProc` đồng bộ nên đi đúng đường xử lý thật.

- Control ID hằng số trong `app/settings_ui.go` (iota từ 100): dùng tên
  (`cRadioVni`, `cMacroEdit`, `cUpdateBtn`…) chứ **không hardcode số**.
- Chi tiết message, bảng ID, các bẫy (swState, theme recreate cửa sổ, caption
  hit-test, message-only window): đọc `references/win32-ui-automation.md`.
- Template sẵng: `scripts/ui_click_test.go` — copy vào `app/` và chỉnh sửa.

## Android — UI test instrumented

`android/app/src/androidTest/kotlin/com/tuigo/MobileSmokeTest.kt` là mô hình chuẩn:
không dùng UI test framework, tìm view bằng `findViewWithTag` / text nút
(ví dụ `"Lưu gõ tắt"`), click bằng `performClick()`, restore toàn bộ prefs/file
trong `finally`. Chạy qua `powershell -File scripts/check-mobile.ps1 -Device
-Serial emulator-5554` (yêu cầu output `PASS: 3 exclusive selections`).

## Linux — e2e thật với IBus

`linux/test/e2e.py` (chạy qua `sh scripts/check-linux.sh`, có `xvfb-run
dbus-run-session` + `ibus-daemon` thật): feeds fake key events qua
`ic.process_key_event`, assert `commit-text`/`update-preedit-text`
(ví dụ `"vieejt "` → `"việt "`).

## iOS / macOS

Không có UI test framework trong repo; verify bằng `sh scripts/check-ios.sh` /
`sh scripts/check-macos.sh` (build simulator/app bundle) **kèm screenshot thiết bị
hoặc simulator** cho thay đổi layout. Ghi rõ trong `checks.md` nếu chỉ review
source mà không build được.

Checklist trước khi kết thúc:

- [ ] Test tái hiện bug (nếu có) đã pass; `go test`/instrumented/e2e tương ứng chạy đúng.
- [ ] `python -P scripts/check-no-diagnostics.py` pass sau thay đổi product code.
- [ ] Screenshot đã xem nếu layout đổi.
- [ ] `checks.md` cập nhật pass/fail + evidence thật.
