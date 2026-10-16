# Win32 UI automation — TuiGo Windows app

App dùng Win32 thuần qua `syscall.NewLazyDLL` (`app/winapi.go`), không framework.
Test điều khiển UI **in-process trong `package main`** (`app/*_test.go`) — được
truy cập mọi hằng số, global và handler, và được miễn lệnh cấm diagnostic logging.

## Cửa sổ và cách tìm

| Cửa sổ | Class | Title | Tạo ở |
|---|---|---|---|
| Main (tray + hook dispatch) | `BoGoTiengVietWnd` | `Tui Gõ` | `app/main.go:181` — **message-only** (parent `HWND_MESSAGE`) |
| Settings | `BoGoSettingsWnd` | `Tui Gõ — Cài đặt` / `Tui Gõ — Settings` | `app/settings_ui.go:2720` (`openSettings()` ở `:2668`) |
| HUD badge | `BoGoHudWnd` | `BoGoHUD` | `app/hud.go` |

- In-process: dùng `gHwnd`, `gSettingsHwnd` trực tiếp; tạo settings window bằng `openSettings()`.
- Out-of-process (ví dụ từ updater — `app/updater/main.go:166`): `FindWindowW("BoGoTiengVietWnd")`;
  nếu miss vì message-only window thì thử `FindWindowExW(HWND_MESSAGE, …)`.

## "Nhấn nút" — đường xử lý thật

`settingsProc` xử lý `WM_COMMAND` (`app/settings_ui.go:2264`):

```go
case WM_COMMAND:
    id := int(wp & 0xFFFF)
    code := int(wp >> 16)
    if code == BN_CLICKED {        // BN_CLICKED == 0
        if swIDs[id] { setCheck(id, !getCheck(id)) }   // flip swState
        settingsCommand(id)                             // handler thật
        return 0
    }
    if code == CBN_SELCHANGE { ... }                    // combo cIMCombo/cCharset/cTheme
    if code == 0x0300 /* EN_CHANGE */ && id == cHotkeyBox { ... }
```

Cách click trong test (đồng bộ — `SendMessage` gọi `settingsProc` ngay):

```go
pSendMessage.Call(gSettingsHwnd, WM_COMMAND, uintptr(id), ctlHnd[id]) // BN_CLICKED = 0
```

- `BM_CLICK` (0x00F5) gửi tới handle nút cũng hoạt động, **nhưng** đừng assert
  bằng `BM_GETCHECK`: state checkbox sống trong `swState`/`setCheck`/`getCheck`
  (`settings_ui.go:1220`), không phải Win32 button state. Đọc `swState[id]` trong test.
- Combo: `pSendMessage.Call(ctlHnd[id], CB_SETCURSEL, uintptr(sel), 0)` rồi gửi
  `WM_COMMAND` với `wp = id | (CBN_SELCHANGE<<16)`; đọc lại bằng `CB_GETCURSEL`
  (xem `comboSel`, `settings_ui.go:1236`).
- EDIT hotkey (`cHotkeyBox`): đổi text rồi gửi `EN_CHANGE` (0x0300) nếu cần
  kích hoạt handler ghi `cfg.HotkeyVk`.

## Control ID (`app/settings_ui.go:171`, iota từ 100)

Dùng **tên hằng số** trong test, đừng hardcode số (số tự tính lại khi đổi thứ tự):

| ID | Tên | Loại / nhãn |
|---|---|---|
| 100 | `cIMCombo` | COMBOBOX "Kiểu gõ / Method" (7 mục, `imList`) |
| 101–104 | `cModern`, `cFreeMark`, `cSpell`, `cRestore` | switch pill |
| 105–107 | `cMacro`, `cMacroAlways`, `cMacroEdit` | gõ tắt; `cMacroEdit` mở notepad `macro.txt` |
| 108–115 | `cSkipLayout`…`cShowHud` | toggle hệ thống (clipboard, startup, admin, sound, HUD…) |
| 117–119 | `cThemeAuto`, `cThemeLight`, `cThemeDark` | chế độ theme |
| 120–122 | `cToggleCombo`, `cExcludeEdit`, `cCharset` | COMBOBOX Unicode/TCVN3 |
| 123–133 | `cModCtrl`…`cF12` | tab phím tắt (EDIT `cHotkeyBox`=127) |
| 134 | `cExcludeBrowse` | mở `GetOpenFileNameW` thật — tránh trong test tự động |
| 135–138 | `cRadioTelex`, `cRadioTelexSimple`, `cRadioVni`, `cRadioViqr` | radio kiểu gõ tab 0 |
| 139–141 | `cLangBtn`, `cUpdateBtn`, `cRememberAppMode` | ngôn ngữ / kiểm tra cập nhật (spawn updater!) / remember app mode |
| 142–146 | `cThemeJade`…`cThemeMaterialTeal` | theme style |

Tất cả nút là `BS_OWNERDRAW` (tự vẽ, `drawButton` `settings_ui.go:931`) — không
có nút Apply/Save: **mọi click đều ghi `cfg` + `saveSettings()` ngay**
(`settingsCommand` kết thúc ở `settings_ui.go:2156`).

## Các bẫy cần nhớ

1. **Nút theme hủy cửa sổ**: `cTheme*`/`cThemeAuto|Light|Dark` gọi
   `DestroyWindow(gSettingsHwnd)` rồi dựng lại (`settings_ui.go:2133, 2152`).
   Sau khi click: chờ/kiểm tra `gSettingsHwnd` mới, `ctlHnd` map lại.
2. **Caption min/max/close và sidebar tab không phải control** — hit-test tọa độ
   trong `WM_LBUTTONDOWN` (`captionAt`, `settings_ui.go:2309`). Muốn click thì
   gửi `WM_LBUTTONDOWN`/`WM_LBUTTONUP` với tọa độ client, hoặc gọi handler trực tiếp.
3. **Không có message loop trong test** — chỉ dùng `SendMessage` (đồng bộ), không
   `PostMessage` trừ khi tự pump `GetMessage`/`DispatchMessage`.
4. **`settingsCommand` ghi file thật**: snapshot `tuigo.ini` (`iniPath()`) và
   restore trong `t.Cleanup` (template đã làm sẵn).
5. **Không click thật ra desktop**: tránh `cExcludeBrowse` (dialog), `cUpdateBtn`
   (spawn `tuigo-updater.exe`), `cMacroEdit`/`cExcludeEdit` (mở notepad), `cAdmin`
   (relaunch elevated). Test handler của chúng bằng cách assert side effect gián
   tiếp hoặc skip.
6. **Menu tray**: ID `idmToggle=1 … idmSettings=10, idmExit=11` (`app/tray.go:10`);
   `PostMessage(gHwnd, WM_COMMAND, idm…, 0)` (lúc này cần message loop thật —
   hoặc gọi handler trực tiếp trong test). `WM_TOGGLE_VN = WM_APP+2` với
   `wp` = 1(ctrl+shift)/2(F1)/3(F2)/other(hotkey) (`app/main.go:29`).

## Pattern test hiện có để bắt chước

- `app/tray_test.go:286` — `TestSidebarHoverPerformanceAndHitTest`: dùng GDI
  offscreen DC, skip nếu không có screen DC (cần phiên desktop tương tác).
- `app/app_mode_test.go:10` — `setupAppModeTest`: lưu/restore **toàn bộ** global
  qua `t.Cleanup`; `focusAppForTest` giả lập focus không làm động desktop user.
