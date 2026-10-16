---
name: Debug Go
description: Debug code Go trong TuiGoTiengViet mà không vi phạm chính sách no-diagnostics — debug bằng test (docSim, setupAppModeTest), delve, go test -run/-v, debug UI Win32 in-process, đồng bộ bug engine qua 3 port Go/Swift/session. Dùng khi sửa bug Go, trace lỗi gõ phím, hoặc điều tra crash/hook.
---

# Debug Go — TuiGoTiengViet

Khi nào dùng: sửa bug trong `app/` (main, engine, updater), `linux/` (IBus
engine, session, settings), hoặc cần trace hành vi gõ phím / UI Windows.

## Ràng buộc sống còn: KHÔNG diagnostic logging

AGENTS.md cấm app tự ghi log trên mọi nền tảng: không `log/slog`, `fmt.Print*`,
`println`, ghi `os.Stderr`/`os.Stdout`, `crash.log`, hook/heartbeat output,
setting debug-log. `scripts/check-no-diagnostics.py` (chạy bởi mọi check script)
sẽ fail nếu thêm — kể cả HTTP client ngoài `app/updater/`.

- **`*_test.go` được miễn** (`check-no-diagnostics.py` skip file test):
  `t.Log`, `fmt.Print` trong test là cách debug chuẩn của repo này.
- Sửa tạm product code để print rồi **revert trước khi commit**, và luôn chạy
  `python -P scripts/check-no-diagnostics.py` trước khi kết thúc.
- Giữ panic containment của Win32 callback và hook recovery **độc lập** với logging.

## Vòng lặp debug chuẩn

1. **Tái hiện bằng test.** Sửa bug = thêm test đỏ trước.
   - Engine (`app/engine/engine_test.go`): dùng harness `docSim` — mô phỏng
     "xóa Backs ký tự, chèn Out" rồi assert document. Nhanh nhất để debug Telex/VNI/macro.
   - Package `main` (`app/*_test.go`): dùng pattern `setupAppModeTest`
     (`app/app_mode_test.go:10`) — lưu/restore toàn bộ global qua `t.Cleanup`,
     `focusAppForTest` giả lập focus app mà không đổi focus desktop user.
   - Linux session: `linux/internal/session/session_test.go`.
2. **Chạy thu hẹp, không cần build cả app:**
   ```powershell
   cd app
   $env:GOWORK="off"; go test ./engine/ -run TestTelex -v -count=1
   $env:GOWORK="off"; go test ./... ; go vet ./...
   ```
   (`GOWORK=off` là quy ước của repo — mọi check script đều set vậy.)
3. **Cần breakpoint → delve** (cài nếu thiếu:
   `go install github.com/go-delve/delve/cmd/dlv@latest`):
   ```powershell
   cd app
   $env:GOWORK="off"; dlv test ./engine -- -test.run TestMacro
   ```
   Với GUI: `dlv exec .\tuigo.exe` rồi breakpoint trong `settingsProc`/`wndProc`.
   Tránh step lâu trong low-level keyboard hook (`app/hook.go`) — nó chạy dưới
   input stack của hệ thống; đặt breakpoint ở entry rồi `continue`.

## Debug UI Win32

- Ưu tiên **test in-process** (`package main`): gọi trực tiếp `settingsCommand(id)`,
  `navHitAt(x, y)`, `paintSettingsCached`… thay vì dựng message loop.
  Xem skill `e2e-ui-testing` để "nhấn nút" thật qua `SendMessage(WM_COMMAND)`.
- State của checkbox nằm trong map `swState` **không phải** `BM_GETCHECK`.
- Nút theme **destroy và tạo lại** cửa sổ settings (`settings_ui.go` ~2133/2152):
  sau khi click phải `openSettings()` lại, `ctlHnd` mới có hiệu lực.
- Test nào ghi `tuigo.ini` (qua `saveSettings()`) phải snapshot/restore file
  (`iniPath()` = `tuigo.ini` cạnh exe) trong `t.Cleanup`.

## Bug gõ phím: 3 port phải đồng bộ

Engine có 3 bản: Go (`app/engine`), Swift (`ios/Sources/TuiGoEngine`, chạy cả
macOS), Session Go cho IBus (`linux/internal/session`). Sửa logic gõ phím/macro
thì chạy regression **cả 3**: `go test ./engine/`, `swift test` (trong `ios/`),
`go test ./...` (trong `linux/`). Macro ký tự ngoài BMP phải encode UTF-16
surrogate — chạy test Unicode/reload trên cả hai port.

## Kết thúc

- `go vet ./...` + `go test ./...` pass với `GOWORK=off`.
- Chạy check script của nền tảng vừa sửa (bảng trong skill `e2e-ui-testing`).
- `python -P scripts/check-no-diagnostics.py` pass.
- Ghi evidence vào `checks.md`.
