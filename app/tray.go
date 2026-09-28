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
var traySize int
var trayLastViet bool

func refreshTrayIcons() {
	dark := isDark()
	size, _, _ := pGetSystemMetrics.Call(49) // SM_CXSMICON
	if size < 16 || size > 128 {
		size = 16
	}
	if gIconVn != 0 && gIconEn != 0 && trayDark == dark && traySize == int(size) {
		return
	}
	p := palDark
	if !dark {
		p = palLight
	}
	vn, en := makeTrayIcon(true, int(size), p), makeTrayIcon(false, int(size), p)
	if vn == 0 || en == 0 {
		pDestroyIcon.Call(vn)
		pDestroyIcon.Call(en)
		return // retain the previous pair if allocation failed
	}
	oldVn, oldEn := gIconVn, gIconEn
	gIconVn, gIconEn = vn, en
	trayDark, traySize = dark, int(size)
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
	changed := effectiveViet() != trayLastViet
	themeChanged := trayDark != isDark()
	refreshTrayIcons()
	if changed || themeChanged {
		trayUpdate()
		if gHudHwnd != 0 {
			pInvalidateRect.Call(gHudHwnd, 0, 1)
		}
		if gSettingsHwnd != 0 {
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
		copy(nid.szTip[:], windows.StringToUTF16("Tiếng Việt đang bật"))
	} else {
		nid.hIcon = gIconEn
		copy(nid.szTip[:], windows.StringToUTF16("Tiếng Việt đang tắt"))
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
	menu, _, _ := pCreatePopupMenu.Call()
	imMenu, _, _ := pCreatePopupMenu.Call()
	o := gEngine.GetOptions()

	appendMenu(menu, MF_STRING|checked(gVietKey), idmToggle, "Gõ tiếng Việt\tAlt+Z")
	appendMenu(imMenu, MF_STRING|checked(gIM == ImTelex), idmTelex, "Telex")
	appendMenu(imMenu, MF_STRING|checked(gIM == ImTelexSimple), idmTelexSimple, "Simple Telex")
	appendMenu(imMenu, MF_STRING|checked(gIM == ImVni), idmVni, "VNI")
	appendMenu(imMenu, MF_STRING|checked(gIM == ImViqr), idmViqr, "VIQR")
	appendMenu(menu, MF_STRING|MF_POPUP, imMenu, "Kiểu gõ")
	appendMenu(menu, MF_SEPARATOR, 0, "")
	appendMenu(menu, MF_STRING|checked(o.SpellCheckEnabled != 0), idmSpell, "Kiểm tra chính tả")
	appendMenu(menu, MF_STRING|checked(o.ModernStyle != 0), idmModern, "Bỏ dấu kiểu mới (oà, uý)")
	appendMenu(menu, MF_STRING|checked(o.MacroEnabled != 0), idmMacro, "Gõ tắt")
	appendMenu(menu, MF_STRING|checked(o.AutoNonVnRestore != 0), idmRestore, "Tự phục hồi từ sai")
	appendMenu(menu, MF_SEPARATOR, 0, "")
	appendMenu(menu, MF_STRING, idmSettings, "Cài đặt…")
	appendMenu(menu, MF_SEPARATOR, 0, "")
	appendMenu(menu, MF_STRING, idmExit, "Thoát")

	var pt point
	pGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	pSetForegroundWnd.Call(gHwnd)
	pTrackPopupMenu.Call(menu, TPM_BOTTOMALIGN|TPM_LEFTALIGN,
		uintptr(pt.x), uintptr(pt.y), 0, gHwnd, 0)
	pDestroyMenu.Call(menu)
}

func toggleVietKey(src string) {
	spec, exe := foregroundApp()
	logLine("toggleVN via " + src)
	if spec.mode == appModeLock {
		return
	}
	if spec.mode == appModeManual { // per-app toggle: does not touch global state
		gPerAppViet[exe] = !gPerAppViet[exe]
		trayUpdate()
		if cfg.SoundOnToggle {
			pMessageBeep.Call(0x00000040) // MB_ICONASTERISK
		}
		hudUpdate()
		hudFlash()
		pInvalidateRect.Call(gSettingsHwnd, 0, 1) // refresh mode indicator
		return
	}
	gVietKey = !gVietKey
	gEngine.SetVietKey(gVietKey) // off: pass-through, macros still expand
	if cfg.SoundOnToggle {
		pMessageBeep.Call(0x00000040) // MB_ICONASTERISK
	}
	trayUpdate()
	hudUpdate()
	hudFlash()
	pInvalidateRect.Call(gSettingsHwnd, 0, 1) // refresh mode indicator
	saveSettings()
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
