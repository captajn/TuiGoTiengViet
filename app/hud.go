package main

// Floating mini capsule HUD: High-End 3-row widget matching mockup
// (media_1790104583638.jpg) with brand logo, [V]/[E] badge, [Telex ▾] dropdown,
// mini toggle switch, menu dots ···, and live status indicator line.

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	WS_POPUP         = 0x80000000
	WS_EX_TOPMOST    = 0x00000008
	WS_EX_TOOLWINDOW = 0x00000080
	WS_EX_NOACTIVATE = 0x08000000
	WS_EX_LAYERED    = 0x00080000

	WM_PAINT     = 0x000F
	WM_LBUTTONUP = 0x0202
	WM_MOUSEMOVE = 0x0200
	WM_NCHITTEST = 0x0084

	SWP_NOSIZE     = 0x0001
	SWP_NOZORDER   = 0x0004
	SWP_NOACTIVATE = 0x0010

	hudW = 168
	hudH = 74
)

var (
	gHudHwnd  uintptr
	hudDragOn bool
	hudMoved  bool
	hudDragPt point
	hudWinPt  point
)

type paintStruct struct {
	hdc         uintptr
	fErase      int32
	rcPaint     rect
	fRestore    int32
	fIncUpdate  int32
	rgbReserved [32]byte
}

func imName() string {
	switch gIM {
	case ImTelexSimple:
		return "STelex"
	case ImVni:
		return "VNI"
	case ImViqr:
		return "VIQR"
	case ImTelexVni:
		return "T+VNI"
	case ImMsVi:
		return "MS-VI"
	case ImUsrKeymap:
		return "Custom"
	}
	return "Telex"
}

func hudPaint(hwnd uintptr) {
	var ps paintStruct
	dc, _, _ := pBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))

	// 1. Rounded capsule background with fine gold hairline
	br, _, _ := pCreateSolidBrush.Call(colBg)
	hudBdCol := colBorder
	if isDark() {
		hudBdCol = colGold
	}
	pen, _, _ := pCreatePen.Call(PS_SOLID, 1, hudBdCol)
	oldB, _, _ := pSelectObject.Call(dc, br)
	oldP, _, _ := pSelectObject.Call(dc, pen)
	pRoundRect.Call(dc, 0, 0, hudW, hudH, 16, 16)
	pSelectObject.Call(dc, oldP)
	pDeleteObject.Call(pen)
	pSelectObject.Call(dc, oldB)
	pDeleteObject.Call(br)

	// 2. Row 1: Logo + "Tui Gõ"
	drawLogo(dc, 14, 14)
	pSetBkMode.Call(dc, TRANSPARENT)
	pSetTextColor.Call(dc, colText)
	oldF, _, _ := pSelectObject.Call(dc, fontBold)
	trc := rect{26, 4, hudW - 24, 22}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr("Tui Gõ"))),
		^uintptr(0), uintptr(unsafe.Pointer(&trc)),
		DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

	// Pin diamond
	diamond(dc, hudW-12, 12, 2, colGoldDim)

	// 3. Row 2: [ V ] badge + [ Telex ▾ ] pill + mini switch + ···
	vnOn := effectiveViet()

	// [ V ] / [ E ] badge
	badgeCol := colPill
	badgeTextCol := colMuted
	badgeLetter := "E"
	if vnOn {
		badgeCol = colJade
		badgeTextCol = 0x302A0B
		if !isDark() {
			badgeTextCol = 0xFFFFFF
		}
		badgeLetter = "V"
	}
	bbr, _, _ := pCreateSolidBrush.Call(badgeCol)
	oldB, _, _ = pSelectObject.Call(dc, bbr)
	nullPen, _, _ := pGetStockObject.Call(NULL_PEN)
	oldP, _, _ = pSelectObject.Call(dc, nullPen)
	pRoundRect.Call(dc, 8, 25, 30, 47, 6, 6)
	pSelectObject.Call(dc, oldP)
	pSelectObject.Call(dc, oldB)
	pDeleteObject.Call(bbr)

	pSetTextColor.Call(dc, badgeTextCol)
	pSelectObject.Call(dc, fontBold)
	blrc := rect{8, 25, 30, 47}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr(badgeLetter))),
		^uintptr(0), uintptr(unsafe.Pointer(&blrc)),
		DT_CENTER|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

	// [ Telex ▾ ] pill
	imbr, _, _ := pCreateSolidBrush.Call(colNavActiveBg)
	oldB, _, _ = pSelectObject.Call(dc, imbr)
	impen, _, _ := pCreatePen.Call(PS_SOLID, 1, colBorder)
	oldP, _, _ = pSelectObject.Call(dc, impen)
	pRoundRect.Call(dc, 35, 25, 96, 47, 6, 6)
	pSelectObject.Call(dc, oldP)
	pDeleteObject.Call(impen)
	pSelectObject.Call(dc, oldB)
	pDeleteObject.Call(imbr)

	pSetTextColor.Call(dc, colText)
	pSelectObject.Call(dc, fontHint)
	imrc := rect{40, 25, 84, 47}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr(imName()))),
		^uintptr(0), uintptr(unsafe.Pointer(&imrc)),
		DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)
	chrc := rect{83, 25, 93, 47}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr("▾"))),
		^uintptr(0), uintptr(unsafe.Pointer(&chrc)),
		DT_CENTER|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

	// Mini switch
	swTrack := colPill
	swKnob := colMuted
	swKx := int32(102 + 2)
	if vnOn {
		swTrack = colJade
		swKnob = 0xFFFFFF
		swKx = int32(102 + 30 - 14)
	}
	sbr, _, _ := pCreateSolidBrush.Call(swTrack)
	oldB, _, _ = pSelectObject.Call(dc, sbr)
	pRoundRect.Call(dc, 102, 28, 132, 44, 16, 16)
	pSelectObject.Call(dc, oldB)
	pDeleteObject.Call(sbr)

	knbr, _, _ := pCreateSolidBrush.Call(swKnob)
	oldB, _, _ = pSelectObject.Call(dc, knbr)
	pEllipse.Call(dc, uintptr(swKx), 30, uintptr(swKx+12), 42)
	pSelectObject.Call(dc, oldB)
	pDeleteObject.Call(knbr)

	// Menu dots ···
	pSetTextColor.Call(dc, colMuted)
	dotsRc := rect{138, 25, 162, 47}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr("···"))),
		^uintptr(0), uintptr(unsafe.Pointer(&dotsRc)),
		DT_CENTER|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

	// 4. Row 3: ● Tiếng Việt đang hoạt động
	dotCol := colMuted
	statusTxt := "Đang tắt tiếng Việt"
	if vnOn {
		dotCol = colJade
		statusTxt = "Tiếng Việt đang hoạt động"
	}
	dtbr, _, _ := pCreateSolidBrush.Call(dotCol)
	oldB, _, _ = pSelectObject.Call(dc, dtbr)
	pEllipse.Call(dc, 10, 56, 16, 62)
	pSelectObject.Call(dc, oldB)
	pDeleteObject.Call(dtbr)

	pSetTextColor.Call(dc, colMuted)
	pSelectObject.Call(dc, fontHint)
	stRc := rect{22, 50, hudW - 8, 68}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr(statusTxt))),
		^uintptr(0), uintptr(unsafe.Pointer(&stRc)),
		DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

	pSelectObject.Call(dc, oldF)
	pEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
}

func hudProc(hwnd uintptr, msg uint32, wp, lp uintptr) uintptr {
	defer func() { recoverCrash("hud") }()
	switch msg {
	case WM_PAINT:
		hudPaint(hwnd)
		return 0
	case WM_LBUTTONDOWN:
		hudDragOn = true
		hudMoved = false
		pSetCapture.Call(hwnd)
		var pt point
		pGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
		hudDragPt = pt
		var rc rect
		pGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&rc)))
		hudWinPt = point{rc.left, rc.top}
		return 0
	case WM_MOUSEMOVE:
		if hudDragOn {
			var pt point
			pGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
			dx, dy := pt.x-hudDragPt.x, pt.y-hudDragPt.y
			if dx < -2 || dx > 2 || dy < -2 || dy > 2 {
				hudMoved = true
			}
			if hudMoved {
				pSetWindowPos.Call(hwnd, 0,
					uintptr(hudWinPt.x+dx), uintptr(hudWinPt.y+dy),
					0, 0, SWP_NOSIZE|SWP_NOZORDER|SWP_NOACTIVATE)
			}
		}
		return 0
	case WM_LBUTTONUP:
		if hudDragOn {
			hudDragOn = false
			pReleaseCapture.Call()
			if !hudMoved {
				// Check where user clicked
				var pt point
				pGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
				var rc rect
				pGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&rc)))
				cx := pt.x - rc.left
				cy := pt.y - rc.top

				if cy >= 25 && cy <= 47 && cx >= 35 && cx <= 96 {
					// Clicked [ Telex ▾ ]: show IM popup
					hMenu, _, _ := pCreatePopupMenu.Call()
					pAppendMenu.Call(hMenu, 0, 1, uintptr(unsafe.Pointer(utf16ptr("Telex"))))
					pAppendMenu.Call(hMenu, 0, 2, uintptr(unsafe.Pointer(utf16ptr("VNI"))))
					pAppendMenu.Call(hMenu, 0, 3, uintptr(unsafe.Pointer(utf16ptr("VIQR"))))
					r, _, _ := pTrackPopupMenu.Call(hMenu, 0x0100|0x0002,
						uintptr(pt.x), uintptr(pt.y), 0, hwnd, 0)
					pDestroyMenu.Call(hMenu)
					switch r {
					case 1:
						setInputMethod(ImTelex)
					case 2:
						setInputMethod(ImVni)
					case 3:
						setInputMethod(ImViqr)
					}
				} else if (cy >= 25 && cy <= 47 && cx >= 135) || cy < 22 {
					// Clicked ··· or header: show main context menu
					showTrayMenu()
				} else {
					// Clicked badge, switch, or status line: toggle VietKey
					toggleVietKey()
				}
			} else {
				var rc rect
				pGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&rc)))
				cfg.HudX, cfg.HudY = int(rc.left), int(rc.top)
				saveSettings()
			}
		}
		return 0
	case WM_RBUTTONUP:
		showTrayMenu()
		return 0
	case WM_ERASEBKGND:
		return 1
	}
	r, _, _ := pDefWindowProc.Call(hwnd, uintptr(msg), wp, lp)
	return r
}

func hudDefaultPos() (int32, int32) {
	cx, _, _ := pGetSystemMetrics.Call(0)
	cy, _, _ := pGetSystemMetrics.Call(1)
	return int32(cx) - hudW - 24, int32(cy) - hudH - 72
}

func hudInit() {
	cls := utf16ptr("BoGoHudWnd")
	wc := wndClassEx{
		cbSize:        uint32(unsafe.Sizeof(wndClassEx{})),
		lpfnWndProc:   windows.NewCallback(hudProc),
		hCursor:       loadArrowCursor(),
		lpszClassName: cls,
	}
	pRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc)))

	x, y := int32(cfg.HudX), int32(cfg.HudY)
	if cfg.HudX == 0 && cfg.HudY == 0 {
		x, y = hudDefaultPos()
	}
	gHudHwnd, _, _ = pCreateWindowEx.Call(
		WS_EX_TOPMOST|WS_EX_TOOLWINDOW|WS_EX_NOACTIVATE,
		uintptr(unsafe.Pointer(cls)),
		uintptr(unsafe.Pointer(utf16ptr("BoGoHUD"))),
		WS_POPUP,
		uintptr(x), uintptr(y), hudW, hudH,
		0, 0, 0, 0)

	rgn, _, _ := pCreateRoundRectRgn.Call(0, 0, hudW+1, hudH+1, 16, 16)
	pSetWindowRgn.Call(gHudHwnd, rgn, 1)

	if cfg.ShowHud {
		pShowWindow.Call(gHudHwnd, 4)
	}
}

func hudUpdate() {
	if gHudHwnd == 0 {
		return
	}
	pInvalidateRect.Call(gHudHwnd, 0, 1)
	if cfg.ShowHud {
		pShowWindow.Call(gHudHwnd, 4)
	} else {
		pShowWindow.Call(gHudHwnd, 0)
	}
}

// hudFlash briefly shows the HUD as toggle feedback even when the
// persistent HUD is off — otherwise switching V/E feels invisible.
func hudFlash() {
	if cfg.ShowHud || gHudHwnd == 0 {
		return
	}
	pInvalidateRect.Call(gHudHwnd, 0, 1)
	pShowWindow.Call(gHudHwnd, 4)     // SW_SHOWNA
	pSetTimer.Call(gHwnd, 2, 1200, 0) // wndProc hides it on timer 2
}
