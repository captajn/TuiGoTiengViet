# checks.md — bằng chứng kiểm chứng

## 2026-10-15 — G6 + N1–N8 (Linux backlog batch)

Phạm vi: G6 (e2e mở rộng), N1 (ToggleKey format clash), N2 (Save xoá key
foreign, cả 2 phía), N3 (ImUsrKeymap silent reset), N4 (truncate ký tự ngoài
BMP), N5 (AlwaysMacro), N6 (sudo treo trong check-linux.sh), N7 (ETXTBSY khi
upgrade), N8 (data race D-Bus).

### Test mới (red → green)

| Test | File | Red trước fix |
| --- | --- | --- |
| `TestSavePreservesForeignKeys` | `linux/config/config_test.go` | FAIL: "Save dropped ToggleKey=2/Theme=3/HotkeyVk=90/AppMode.../; user comment" |
| `TestToggleKeyNumericFallsBackToDefaults` | `linux/config/config_test.go` | FAIL: `ToggleKeys = [], want [ctrl+shift]` (N1) |
| `TestToggleKeyNamesPreferred` | `linux/config/config_test.go` | FAIL: `ToggleKeys = [], want [alt+z ctrl+space]` |
| `TestToggleKeyNamesEmptyDisables` | `linux/config/config_test.go` | FAIL: `ToggleKeys = [ctrl+shift], want disabled` |
| `TestAlwaysMacroRoundTrip` | `linux/config/config_test.go` | FAIL: "AlwaysMacro lost in save/load" |
| `TestAstralCodepointStaysInPreedit` | `linux/internal/session/session_test.go` | FAIL: `preedit = [97 62976], want [97 55357 56832]` (0xF600 = uint16 truncate) |
| `TestAlwaysMacroExpandsInEnglish` | `linux/internal/session/session_test.go` | (compile red qua field mới; hành vi chạy cùng N5) |
| `TestEnglishWithoutAlwaysMacroForwards` | `linux/internal/session/session_test.go` | — (giữ hành vi mặc định) |
| `TestSaveSettingsToPreservesForeignKeys` | `app/settings_test.go` | (N2 mirror phía Windows) |
| `TestMergeIniPreserveIsIdempotent` | `app/settings_test.go` | — |
| `TestImChoicesCoverAllMethods` | `linux/config/config_test.go` | — (N3: chốt mọi method nằm trong bảng) |
| `TestViqrWord`, `TestVniWord` | `app/engine/engine_test.go` | — (chốt chuỗi gõ cho e2e: `vie^.t`, `vie65t`) |

### Kết quả

| Bước | Lệnh | Kết quả | Evidence |
| --- | --- | --- | --- |
| Privacy guard 5 nền tảng | `python -P scripts/check-no-diagnostics.py` | **PASS** | chạy trong check-diagnostics |
| Windows + updater | `powershell -File scripts/check-diagnostics.ps1` | **PASS** | `ok tuigo / ok tuigo/engine / ok tuigo/updater`, vet 0 |
| Linux module | `cd linux; GOWORK=off go vet ./...; go test ./... -count=1` | **PASS** | `ok cmd/tuigo-ibus 0.726s`, `ok config 0.473s`, `ok internal/session 0.439s` (27 test) |
| Settings GUI (Fyne) | `cd linux/settings; GOWORK=off go vet ./...` | **PASS** | vet exit 0 trên Windows (cgo/GL) |
| e2e syntax | `python -m py_compile linux/test/e2e.py` | **PASS** | exit 0 |
| Red→green N1/N2/N5 | test config chạy trước/sau fix | **PASS** | bảng red ở trên |
| Red→green N4 | test session chạy trước/sau fix | **PASS** | `[97 62976]` → `[97 55357 56832]` |
| IBus e2e thật (G6) | `sh scripts/check-linux.sh` | **UNRESOLVED** | cần Ubuntu — kịch bản mới (telex/phrases/backspace/macro/VNI/VIQR/toggle/preedit-đã viết, chờ chạy thật; check-linux.sh giờ cách ly XDG_CONFIG_HOME) |
| macOS/Android regression | — | **UNRESOLVED / PASS trước đó** | batch này không đổi engine mobile; G1 batch trước đã PASS Android, iOS chờ macOS |

### Ghi chú thiết kế

- **N5 AlwaysMacro**: engine (cả 3 port) luôn bung macro ở EN mode bất kể
  option (đã probe bằng test: got `"không "` cả khi AlwaysMacro=0). Scope đã làm:
  session-level — `AlwaysMacro=1` route typing qua engine ở EN mode (word vẫn
  ở preedit vì IME shell không xoá được text đã commit); `0` = pass-through
  thuần như trước. Checkbox Windows "Gõ tắt cả khi tắt Tiếng Việt" hiện là
  no-op sẵn có — cần quyết định riêng (gate ở engine cấp 5 platform).
- **N1**: key mới `ToggleKeyNames=` (Linux-owned); `ToggleKey` số (Windows
  legacy) → fallback `["ctrl+shift"]`, không mất hotkey. Windows giữ nguyên
  `ToggleKey=%d`.
- **N2**: cả `linux/config.Save()` lẫn `app saveSettingsTo` giờ merge — giữ
  mọi key foreign + comment; `AppMode.*` managed đầy đủ (xoá entry stale).
- **N6**: bỏ `sudo cp` system component trong check-linux.sh (component user
  là đủ) — hết nguy cơ treo chờ mật khẩu sudo.
- **N7**: install.sh cài qua `cp .new + mv -f` (hết ETXTBSY khi engine đang
  chạy); build settings fail không làm hỏng cả install.
- **N8**: `tuigoEngine` có `sync.Mutex` bọc ProcessKeyEvent/FocusOut/Reset/
  Enable/Disable/PropertyActivate.

### Thay đổi code (đợt này)

- `linux/config/config.go` + `config_test.go` (mới) — N1/N2/N5-config/ImChoices.
- `linux/internal/session/session.go` + test — N4 (utf16 surrogate), N5 (AlwaysMacro).
- `linux/cmd/tuigo-ibus/main.go` — N8 mutex, AlwaysMacro wiring; `configure.go`/`configure_test.go` (G2 đợt trước).
- `linux/settings/main.go` — checkbox AlwaysMacro; dùng `config.ImChoices()` (N3);
  **sửa thêm**: checkbox "Bật gõ tắt (macro)" tạo nhưng không nằm layout nào → đưa vào tab Gõ tắt.
- `linux/test/e2e.py` — G6: 7 kịch bản (telex, phrases, backspace, macro, VNI, VIQR, toggle) + assert preedit rỗng sau commit; backup/restore macro.txt.
- `linux/install.sh` — N7; `scripts/check-linux.sh` — N6 + XDG isolation cho e2e.
- `app/settings.go` + `app/settings_test.go` (mới) — N2 mirror phía Windows.

---

## 2026-10-15 — G1 + G2 (Linux prep: macro reload + keymap override)

Phạm vi: G1 (macro xoá/rỗng file không được nạp lại — `app/engine/macro.go`),
G2 (`keymap.txt` ghi đè kiểu gõ đã chọn — `linux/cmd/tuigo-ibus`),
đồng bộ 2 port mobile khi đổi macro handling (AGENTS.md).
Kèm sẵn trong working tree (đã review): G3 (panel menu thiếu Telex đơn giản),
G5 (icon `tuigo.png` → `logo.jpg`), G4-một-phần (mục "Keymap tuỳ chỉnh"
trong settings + radio `im_usrkeymap`).

### Test mới (red → green)

| Test | File | Red trước fix |
| --- | --- | --- |
| `TestMacroClearedWhenFileRemoved` | `linux/internal/session/session_test.go` | FAIL với `macro.go` cũ (git stash xác nhận: `--- FAIL: TestMacroClearedWhenFileRemoved`) |
| `TestMacroClearedWhenFileEmptied` | `linux/internal/session/session_test.go` | — (cùng đường nhánh early-return) |
| `TestMacroFileRemovedClearsList` | `app/engine/engine_test.go` | cùng bug, chạy trong `check-diagnostics.ps1` |
| `TestMacroFileEmptiedClearsList` | `app/engine/engine_test.go` | — |
| `TestKeymapDoesNotOverrideSelectedIM` | `linux/cmd/tuigo-ibus/configure_test.go` | FAIL với load vô điều kiện (`doc="đ", want "q"` — keymap.txt đánh cắp ImTelex) |
| `TestKeymapUsedWhenSelected` | `linux/cmd/tuigo-ibus/configure_test.go` | — (hành vi ImUsrKeymap phải giữ) |
| `TestMacroLoadedByConfigure` | `linux/cmd/tuigo-ibus/configure_test.go` | — (chốt wiring reloadConfig) |
| `macroFileRemovedClearsList` | `android/engine/src/test/.../EngineTest.kt` | — |
| `testMacroFileRemovedClearsList` | `ios/Tests/TuiGoEngineTests/EngineTests.swift` | — |

Ghi chú red-verify: sentinel keymap `q = dd` ('q' là ký tự thường ở mọi method
built-in; `w` là Telex-W nên đã phải đổi sau lần fail đầu — không phải bug).

### Kết quả

| Bước | Lệnh | Kết quả | Evidence |
| --- | --- | --- | --- |
| Privacy guard 5 nền tảng | `python -P scripts/check-no-diagnostics.py` | **PASS** | "PASS: no app-owned diagnostic logging or debug-log option on all five platforms" (24/5/10/7/17/5 product files) |
| Windows + updater | `powershell -File scripts/check-diagnostics.ps1` | **PASS** | guard PASS + `ok tuigo 1.774s / ok tuigo/engine 0.833s / ok tuigo/updater 3.361s`, vet exit 0 |
| Linux module | `cd linux; GOWORK=off go vet ./...; go test ./... -count=1` | **PASS** | `ok tuigo-linux/cmd/tuigo-ibus 1.493s`, `ok tuigo-linux/internal/session 1.023s` (16 test gồm 3 G2 mới, 2 G1 mới) |
| Red→green G1 | `git stash push -- app/engine/macro.go` → run test → `git stash pop` | **PASS** | OLD code: `--- FAIL: TestMacroClearedWhenFileRemoved`; fix restored: pass |
| Red→green G2 | tạm bỏ guard `ImUsrKeymap` trong `configureEngine` → run test → restore | **PASS** | unconditional: `im=0: keymap.txt hijacked the selected method, doc="đ", want "q"`; restored: `ok` |
| Android regression | `powershell -File scripts/check-mobile.ps1` | **PASS** | `:engine:test` (gồm `macroFileRemovedClearsList` mới) + `:app:assembleDebug` + `:app:lintDebug` — "BUILD SUCCESSFUL in 2m 27s, 48 actionable tasks"; guard privacy + palette + logo PASS; 1 warning deprecation `setDecorFitsSystemWindows` ở `MobileUi.kt:443` (không chặn) |
| iOS regression | `sh scripts/check-ios.sh` | **UNRESOLVED** | cần macOS — Windows không chạy được; chưa claim pass (AGENTS.md). Test `testMacroFileRemovedClearsList` đã thêm, chờ chạy `swift test` trên macOS |
| IBus e2e thật | `sh scripts/check-linux.sh` | **UNRESOLVED** | cần Ubuntu + ibus-daemon/xvfb — sẽ chạy khi có môi trường, mở rộng kịch bản theo G6 |
| Red→green G1 (mobile) | — | **UNRESOLVED** | test mobile thêm cùng thay đổi; chưa chạy red-verify riêng |

### Thay đổi code

- `app/engine/macro.go` — clear `macStore.items` trước `os.Open` (mọi nhánh xác định).
- `app/engine/engine_test.go` — 2 regression test G1.
- `android/engine/.../Macro.kt` + `EngineTest.kt` — clear trước early return + test.
- `ios/Sources/TuiGoEngine/Macro.swift` + `EngineTests.swift` — tương tự.
- `linux/cmd/tuigo-ibus/configure.go` (mới) — `configureEngine()`: keymap chỉ load khi `ImUsrKeymap`.
- `linux/cmd/tuigo-ibus/configure_test.go` (mới) — 3 test G2.
- `linux/cmd/tuigo-ibus/main.go` — `reloadConfig`/`setIM` dùng helper; `setIM(ImUsrKeymap)` load keymap ngay; radio `im_telexsimple` + `im_usrkeymap` trong panel; icon `logo.jpg`.
- `linux/internal/session/session_test.go` — 2 test G1.
- `linux/settings/main.go` — thêm "Keymap tuỳ chỉnh (keymap.txt)" vào danh sách kiểu gõ.
