package main

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	idmToggle = iota + 1
	idmTelex
	idmTelexSimple
	idmVni
	idmViqr
	idmSpell
	idmModern
	idmMacro
	idmRestore
	idmSettings
	idmExit
)

var (
	gIconVn, gIconEn uintptr
	gIM              = ImTelex
)

// Icon handles are recreated only when theme or display scale changes.
var trayDark bool
var trayTheme = -1
var trayStyle = -1
var trayCols [4]int
var traySize int
var trayLastViet bool
var gTrayToggleApp inputAppContext // target captured before the deferred single click
var gTrayMenuApp inputAppContext

// Double-click sequence is LBUTTONUP → DBLCLK → LBUTTONUP: the second UP must
// not re-arm the deferred toggle, or settings opens AND V/E still flips.
var gTrayDblclkArmed bool

func trayCustom() [4]int {
	return [4]int{cfg.TrayV, cfg.TrayVBg, cfg.TrayE, cfg.TrayEBg}
}

func refreshTrayIcons() {
	dark := isDark()
	size, _, _ := pGetSystemMetrics.Call(49) // SM_CXSMICON
	if size < 16 || size > 128 {
		size = 16
	}
	cols := trayCustom()
	if gIconVn != 0 && gIconEn != 0 && trayDark == dark && trayTheme == cfg.Theme && trayStyle == cfg.Style && trayCols == cols && traySize == int(size) {
		return
	}
	p := curPalette()
	vn, en := makeTrayIcon(true, int(size), p), makeTrayIcon(false, int(size), p)
	if vn == 0 || en == 0 {
		pDestroyIcon.Call(vn)
		pDestroyIcon.Call(en)
	trayCols = cols
		return // retain the previous pair if allocation failed
	}
	oldVn, oldEn := gIconVn, gIconEn
	gIconVn, gIconEn = vn, en
	trayDark, trayTheme, trayStyle, traySize = dark, cfg.Theme, cfg.Style, int(size)
	if gTrayAdded {
		trayUpdate()
	}
	if oldVn != 0 {
		pDestroyIcon.Call(oldVn)
	}
	if oldEn != 0 {
		pDestroyIcon.Call(oldEn)
	}
}

// Runs on the UI thread, including when focus changes without a keystroke.
func syncTrayState() {
	changed := effectiveViet() != trayLastViet || trayCols != trayCustom()
	themeChanged := trayDark != isDark() || trayTheme != cfg.Theme || trayStyle != cfg.Style
	refreshTrayIcons()
	if changed || themeChanged {
		trayUpdate()
		hudUpdate()
		if gSettingsHwnd != 0 && (themeChanged || gActiveTab == 0) {
			pInvalidateRect.Call(gSettingsHwnd, 0, 1)
		}
	}
}

func notifyIcon(flags uint32) *notifyIconData {
	nid := &notifyIconData{
		cbSize:           uint32(unsafe.Sizeof(notifyIconData{})),
		hWnd:             gHwnd,
		uID:              1,
		uFlags:           flags,
		uCallbackMessage: WM_TRAYICON,
	}
	if effectiveViet() {
		nid.hIcon = gIconVn
		copy(nid.szTip[:], windows.StringToUTF16(tr("Tiếng Việt đang bật", "Vietnamese: ON")))
	} else {
		nid.hIcon = gIconEn
		copy(nid.szTip[:], windows.StringToUTF16(tr("Tiếng Việt đang tắt", "Vietnamese: OFF (English)")))
	}
	return nid
}

var gTrayAdded bool

func trayAdd() {
	r, _, _ := pShellNotifyIcon.Call(NIM_ADD,
		uintptr(unsafe.Pointer(notifyIcon(NIF_ICON|NIF_TIP|NIF_MESSAGE))))
	// At logon Explorer may not be ready yet — NIM_ADD can fail; the
	// WM_TIMER watchdog retries until it lands. TaskbarCreated broadcast
	// re-adds after every Explorer (re)start too.
	gTrayAdded = r != 0
	trayLastViet = effectiveViet()
}

func trayUpdate() {
	r, _, _ := pShellNotifyIcon.Call(NIM_MODIFY, uintptr(unsafe.Pointer(notifyIcon(NIF_ICON|NIF_TIP))))
	gTrayAdded = r != 0
	trayLastViet = effectiveViet()
}

func trayDelete() {
	pShellNotifyIcon.Call(NIM_DELETE, uintptr(unsafe.Pointer(notifyIcon(0))))
	gTrayAdded = false
	pDestroyIcon.Call(gIconVn)
	pDestroyIcon.Call(gIconEn)
	gIconVn, gIconEn = 0, 0
}

func appendMenu(menu uintptr, flags uint32, id uintptr, text string) {
	pAppendMenu.Call(menu, uintptr(flags), id, uintptr(unsafe.Pointer(utf16ptr(text))))
}

func checked(b bool) uint32 {
	if b {
		return MF_CHECKED
	}
	return 0
}

func showTrayMenu() {
	gTrayMenuApp = currentInputApp()
	menu, _, _ := pCreatePopupMenu.Call()
	imMenu, _, _ := pCreatePopupMenu.Call()
	o := gEngine.GetOptions()

	appendMenu(menu, MF_STRING|checked(effectiveViet()), idmToggle, tr("Gõ tiếng Việt\tAlt+Z", "Vietnamese Input\tAlt+Z"))
	appendMenu(imMenu, MF_STRING|checked(gIM == ImTelex), idmTelex, "Telex")
	appendMenu(imMenu, MF_STRING|checked(gIM == ImTelexSimple), idmTelexSimple, "Simple Telex")
	appendMenu(imMenu, MF_STRING|checked(gIM == ImVni), idmVni, "VNI")
	appendMenu(imMenu, MF_STRING|checked(gIM == ImViqr), idmViqr, "VIQR")
	appendMenu(menu, MF_STRING|MF_POPUP, imMenu, tr("Kiểu gõ", "Input Method"))
	appendMenu(menu, MF_SEPARATOR, 0, "")
	appendMenu(menu, MF_STRING|checked(o.SpellCheckEnabled != 0), idmSpell, tr("Kiểm tra chính tả", "Spell Check"))
	appendMenu(menu, MF_STRING|checked(o.ModernStyle != 0), idmModern, tr("Bỏ dấu kiểu mới (oà, uý)", "Modern Tone Placement (oà, uý)"))
	appendMenu(menu, MF_STRING|checked(o.MacroEnabled != 0), idmMacro, tr("Gõ tắt", "Macros"))
	appendMenu(menu, MF_STRING|checked(o.AutoNonVnRestore != 0), idmRestore, tr("Tự phục hồi từ sai", "Auto-restore Non-VN Words"))
	appendMenu(menu, MF_SEPARATOR, 0, "")
	appendMenu(menu, MF_STRING, idmSettings, tr("Cài đặt…", "Settings…"))
	appendMenu(menu, MF_SEPARATOR, 0, "")
	appendMenu(menu, MF_STRING, idmExit, tr("Thoát", "Exit"))

	var pt point
	pGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	pSetForegroundWnd.Call(gHwnd)
	pTrackPopupMenu.Call(menu, TPM_BOTTOMALIGN|TPM_LEFTALIGN,
		uintptr(pt.x), uintptr(pt.y), 0, gHwnd, 0)
	pDestroyMenu.Call(menu)
}

// flipVietKey flips the effective Vietnamese state. Cheap — pure memory
// ops plus the cached foreground-app lookup — so it can run inside the
// keyboard hook and the very next keystroke already sees the new state.
// Returns false when the foreground app is locked (toggle ignored).
func flipVietKey() bool {
	app := currentInputApp()
	return flipVietKeyForApp(app)
}

func flipVietKeyForApp(app inputAppContext) bool {
	spec, exe := app.spec, app.exe
	if spec.mode == appModeLock {
		return false
	}
	if cfg.RememberAppMode && exe != "" {
		on := !vietForApp(spec, exe)
		gRememberedAppViet[exe] = on
		gEngine.Reset()
		gAutoCapNext = false
		gEngine.SetVietKey(on)
		return true
	}
	if spec.mode == appModeManual { // per-app toggle: does not touch global state
		gPerAppViet[exe] = !gPerAppViet[exe]
		return true
	}
	gVietKey = !gVietKey
	// The global toggle is the master switch: drop every per-app override
	// so an excluded app toggled on earlier returns to exclusion — without
	// this it stays un-excluded for the whole session with no way back.
	gPerAppViet = map[string]bool{}
	gEngine.SetVietKey(gVietKey) // off: pass-through, macros still expand
	return true
}

// toggleEffects is the heavyweight half of a toggle — beep, tray
// RPC, HUD update, async ini save. Posted onto the message loop so the hook
// never pays for it.
func toggleEffects(_ string) {
	spec, _, _ := foregroundApp()
	if cfg.SoundOnToggle {
		pMessageBeep.Call(0x00000040) // MB_ICONASTERISK
	}
	trayUpdate()
	hudUpdate()
	if gSettingsHwnd != 0 && gActiveTab == 0 {
		pInvalidateRect.Call(gSettingsHwnd, 0, 1) // refresh mode indicator on Overview tab
	}
	if cfg.RememberAppMode || spec.mode != appModeManual {
		saveSettings()
	}
}

func toggleVietKey(src string) {
	if flipVietKey() {
		toggleEffects(src)
	}
}

func toggleVietKeyForApp(src string, app inputAppContext) {
	if flipVietKeyForApp(app) {
		toggleEffects(src)
	}
}

// toggleMacro flips macro expansion (F9 quick action).
func toggleMacro() {
	o := gEngine.GetOptions()
	o.MacroEnabled = 1 - o.MacroEnabled
	gEngine.SetOptions(o)
	if ctlHnd[cMacro] != 0 { // keep settings UI in sync if open
		setCheck(cMacro, o.MacroEnabled != 0)
	}
	saveSettings()
	if cfg.SoundOnToggle {
		pMessageBeep.Call(0x00000040)
	}
}

func setInputMethod(im int) {
	gIM = im
	gEngine.SetInputMethod(im)
	gLastIM = im
	hudUpdate()
	repaintRadioGroup() // covers tray-menu switches while settings is open
	saveSettings()
}

func toggleOption(id uintptr) {
	o := gEngine.GetOptions()
	switch id {
	case idmSpell:
		o.SpellCheckEnabled ^= 1
	case idmModern:
		o.ModernStyle ^= 1
	case idmMacro:
		o.MacroEnabled ^= 1
	case idmRestore:
		o.AutoNonVnRestore ^= 1
	}
	gEngine.SetOptions(o)
	saveSettings()
}
