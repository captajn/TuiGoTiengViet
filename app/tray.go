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

var (
	pPrivateExtractIcons = user32.NewProc("PrivateExtractIconsW")
	pDrawIconEx          = user32.NewProc("DrawIconEx")
)

const (
	DI_MASK   = 1
	DI_NORMAL = 3
)

// makeTrayIcon renders the embedded mascot at 32px with a status dot
// bottom-right (jade = Vietnamese on, muted = off) -> HICON.
func makeTrayIcon(on bool) uintptr {
	const sz = 32
	if len(gLogoBgra) == 0 {
		initLogo()
	}

	screen, _, _ := pGetDC.Call(0)
	memC, _, _ := pCreateCompatibleDC.Call(screen)
	bmp, _, _ := pCreateCompatibleBitmap.Call(screen, sz, sz)
	var maskBits [sz * sz / 8]byte
	mask, _, _ := pCreateBitmap.Call(sz, sz, 1, 1,
		uintptr(unsafe.Pointer(&maskBits[0])))
	oldC, _, _ := pSelectObject.Call(memC, bmp)

	// Draw high-res chibi cultivation mascot scaled to 32x32 with rounded corners
	drawAppMascot(memC, 0, 0, sz, sz, 6)

	// Status dot bottom-right on color plane (jade when typing VN, muted when English)
	dotCol := colMuted
	if on {
		dotCol = colJade
	}
	dbr, _, _ := pCreateSolidBrush.Call(dotCol)
	oldB, _, _ := pSelectObject.Call(memC, dbr)
	penD, _, _ := pCreatePen.Call(PS_SOLID, 1, 0x101514) // dark rim around dot for contrast
	oldP, _, _ := pSelectObject.Call(memC, penD)
	pEllipse.Call(memC, 20, 20, 31, 31)
	pSelectObject.Call(memC, oldP)
	pDeleteObject.Call(penD)
	pSelectObject.Call(memC, oldB)
	pDeleteObject.Call(dbr)

	pSelectObject.Call(memC, oldC)

	ii := iconInfo{fIcon: 1, hbmMask: mask, hbmColor: bmp}
	ic, _, _ := pCreateIconIndirect.Call(uintptr(unsafe.Pointer(&ii)))
	pDeleteObject.Call(bmp)
	pDeleteObject.Call(mask)
	pDeleteDC.Call(memC)
	pReleaseDC.Call(0, screen)
	return ic
}

func notifyIcon(flags uint32) *notifyIconData {
	nid := &notifyIconData{
		cbSize:           uint32(unsafe.Sizeof(notifyIconData{})),
		hWnd:             gHwnd,
		uID:              1,
		uFlags:           flags,
		uCallbackMessage: WM_TRAYICON,
	}
	if gVietKey {
		nid.hIcon = gIconVn
		copy(nid.szTip[:], windows.StringToUTF16("Tui Gõ — đang bật tiếng Việt"))
	} else {
		nid.hIcon = gIconEn
		copy(nid.szTip[:], windows.StringToUTF16("Tui Gõ — đang tắt (English)"))
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
}

func trayUpdate() {
	pShellNotifyIcon.Call(NIM_MODIFY, uintptr(unsafe.Pointer(notifyIcon(NIF_ICON|NIF_TIP))))
}

func trayDelete() {
	pShellNotifyIcon.Call(NIM_DELETE, uintptr(unsafe.Pointer(notifyIcon(0))))
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
	if spec.mode == appModeManual { // per-app toggle: does not touch global state
		gPerAppViet[exe] = !gPerAppViet[exe]
		if cfg.SoundOnToggle {
			pMessageBeep.Call(0x00000040) // MB_ICONASTERISK
		}
		hudUpdate()
		hudFlash()
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
