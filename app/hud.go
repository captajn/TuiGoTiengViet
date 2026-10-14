package main

// Compact floating V/E badge: lightweight 32-bit per-pixel alpha layered window
// (UpdateLayeredWindow) with zero square corner artifacts on both Windows 10 and 11.

import (
	"math"
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

	hudBaseSize = 34
)

var (
	gHudHwnd  uintptr
	hudShown  bool
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

type blendFunction struct {
	BlendOp             byte
	BlendFlags          byte
	SourceConstantAlpha byte
	AlphaFormat         byte
}

func hudSizePx() int32 {
	dpi := systemDpi()
	if gHudHwnd != 0 {
		if d := getDpi(gHudHwnd); d > 0 {
			dpi = d
		}
	}
	return int32(int64(hudBaseSize) * int64(dpi) / 96)
}

func renderHudPixels(pixels []uint32, sz int, vnOn bool, p palette) {
	fill, ink, border := trayBadgeColors(vnOn, p)
	toRGB := func(c uintptr) (uint8, uint8, uint8) {
		return uint8(c), uint8(c >> 8), uint8(c >> 16)
	}
	bgR, bgG, bgB := toRGB(fill)
	fgR, fgG, fgB := toRGB(ink)
	bdR, bdG, bdB := toRGB(border)

	vPoly := [][2]float64{{3.6, 3.6}, {6.0, 3.6}, {8.0, 9.7}, {10.0, 3.6}, {12.4, 3.6}, {9.25, 12.4}, {6.75, 12.4}}
	ePoly := [][2]float64{{4.6, 3.6}, {11.4, 3.6}, {11.4, 5.5}, {6.7, 5.5}, {6.7, 7.05}, {10.7, 7.05}, {10.7, 8.95}, {6.7, 8.95}, {6.7, 10.5}, {11.4, 10.5}, {11.4, 12.4}, {4.6, 12.4}}
	glyph := ePoly
	if vnOn {
		glyph = vPoly
	}
	insideGlyph := func(x, y float64) bool {
		inside := false
		j := len(glyph) - 1
		for i, a := range glyph {
			b := glyph[j]
			if (a[1] > y) != (b[1] > y) && x < (b[0]-a[0])*(y-a[1])/(b[1]-a[1])+a[0] {
				inside = !inside
			}
			j = i
		}
		return inside
	}
	sdfRound := func(x, y, cx, cy, half, radius float64) float64 {
		qx := math.Abs(x-cx) - (half - radius)
		qy := math.Abs(y-cy) - (half - radius)
		return math.Hypot(math.Max(qx, 0), math.Max(qy, 0)) + math.Min(math.Max(qx, qy), 0) - radius
	}

	const samples = 4
	invSamples2 := 1.0 / float64(samples*samples)

	for y := 0; y < sz; y++ {
		for x := 0; x < sz; x++ {
			var accR, accG, accB, accA float64
			for sy := 0; sy < samples; sy++ {
				for sx := 0; sx < samples; sx++ {
					px := (float64(x) + (float64(sx)+0.5)/samples) * 16.0 / float64(sz)
					py := (float64(y) + (float64(sy)+0.5)/samples) * 16.0 / float64(sz)

					dCard := sdfRound(px, py, 8.0, 7.8, 6.8, 3.3)
					if dCard <= 0 {
						r, g, b := float64(bgR), float64(bgG), float64(bgB)
						if dCard > -0.75 {
							r, g, b = float64(bdR), float64(bdG), float64(bdB)
						}
						gx := (px-8.0)*1.08 + 8.0
						gy := (py-7.8)*1.08 + 8.0
						if insideGlyph(gx, gy) {
							r, g, b = float64(fgR), float64(fgG), float64(fgB)
						}
						accR += r
						accG += g
						accB += b
						accA += 255.0
					} else {
						dSh := sdfRound(px, py, 8.0, 8.35, 6.8, 3.4)
						if dSh < 1.1 {
							t := 1.0 - math.Max(0, dSh)/1.1
							accA += 72.0 * t * t
						}
					}
				}
			}
			r := uint32(math.Min(255, accR*invSamples2+0.5))
			g := uint32(math.Min(255, accG*invSamples2+0.5))
			b := uint32(math.Min(255, accB*invSamples2+0.5))
			a := uint32(math.Min(255, accA*invSamples2+0.5))
			if r > a {
				r = a
			}
			if g > a {
				g = a
			}
			if b > a {
				b = a
			}
			pixels[y*sz+x] = (a << 24) | (r << 16) | (g << 8) | b
		}
	}
}

// hudRender paints the compact V/E squircle into a 32-bit premultiplied ARGB
// DIBSection and updates the layered window directly (no WM_PAINT or DWM frame).
func hudRender() {
	if gHudHwnd == 0 {
		return
	}
	sz := int(hudSizePx())
	if sz < 24 {
		sz = 24
	}

	screenDC, _, _ := pGetDC.Call(0)
	if screenDC == 0 {
		return
	}
	defer pReleaseDC.Call(0, screenDC)

	memDC, _, _ := pCreateCompatibleDC.Call(screenDC)
	if memDC == 0 {
		return
	}
	defer pDeleteDC.Call(memDC)

	bi := bitmapInfo{}
	bi.bmiHeader.biSize = uint32(unsafe.Sizeof(bitmapInfoHeader{}))
	bi.bmiHeader.biWidth = int32(sz)
	bi.bmiHeader.biHeight = -int32(sz) // top-down
	bi.bmiHeader.biPlanes = 1
	bi.bmiHeader.biBitCount = 32
	var bits unsafe.Pointer
	bmp, _, _ := pCreateDIBSection.Call(screenDC, uintptr(unsafe.Pointer(&bi)), 0, uintptr(unsafe.Pointer(&bits)), 0, 0)
	if bmp == 0 || bits == nil {
		return
	}
	defer pDeleteObject.Call(bmp)
	oldBmp, _, _ := pSelectObject.Call(memDC, bmp)
	defer pSelectObject.Call(memDC, oldBmp)

	pixels := unsafe.Slice((*uint32)(bits), sz*sz)
	renderHudPixels(pixels, sz, effectiveViet(), curPalette())

	var rc rect
	pGetWindowRect.Call(gHudHwnd, uintptr(unsafe.Pointer(&rc)))
	ptDst := point{rc.left, rc.top}
	wndSz := point{int32(sz), int32(sz)}
	ptSrc := point{0, 0}
	blend := blendFunction{
		BlendOp:             0, // AC_SRC_OVER
		BlendFlags:          0,
		SourceConstantAlpha: 255,
		AlphaFormat:         1, // AC_SRC_ALPHA
	}
	pUpdateLayeredWindow.Call(
		gHudHwnd,
		screenDC,
		uintptr(unsafe.Pointer(&ptDst)),
		uintptr(unsafe.Pointer(&wndSz)),
		memDC,
		uintptr(unsafe.Pointer(&ptSrc)),
		0,
		uintptr(unsafe.Pointer(&blend)),
		2, // ULW_ALPHA
	)
}

func hudProc(hwnd uintptr, msg uint32, wp, lp uintptr) uintptr {
	defer func() { _ = recover() }()
	switch msg {
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
				toggleVietKey("hud")
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
	case WM_DPICHANGED:
		hudRender()
		return 0
	case WM_ENDSESSION:
		if wp != 0 { // session is ending (logoff/shutdown) — save preferences
			saveSettingsSync()
		}
		return 0
	case WM_POWERBROADCAST:
		if wp == PBT_APMRESUME || wp == PBT_RESUMESUSPEND {
			reinstallHooks() // sleep/hibernate can silently drop LL hooks
		}
		return 1
	case WM_WTSSESSION_CHANGE:
		if wp == WTS_SESSION_UNLOCK {
			reinstallHooks()
		}
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
	sz := hudSizePx()
	return int32(cx) - sz - 24, int32(cy) - sz - 64
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

	sz := hudSizePx()
	x, y := int32(cfg.HudX), int32(cfg.HudY)
	cx, _, _ := pGetSystemMetrics.Call(0)
	cy, _, _ := pGetSystemMetrics.Call(1)
	if (cfg.HudX == 0 && cfg.HudY == 0) || x < 0 || y < 0 || x > int32(cx)-sz || y > int32(cy)-sz {
		x, y = hudDefaultPos()
	}
	gHudHwnd, _, _ = pCreateWindowEx.Call(
		WS_EX_TOPMOST|WS_EX_TOOLWINDOW|WS_EX_NOACTIVATE|WS_EX_LAYERED,
		uintptr(unsafe.Pointer(cls)),
		uintptr(unsafe.Pointer(utf16ptr("BoGoHUD"))),
		WS_POPUP,
		uintptr(x), uintptr(y), uintptr(sz), uintptr(sz),
		0, 0, 0, 0)

	hudRender()
	if cfg.ShowHud {
		pShowWindow.Call(gHudHwnd, 4) // SW_SHOWNA
		hudShown = true
	}
}

func hudUpdate() {
	if gHudHwnd == 0 {
		return
	}
	if cfg.ShowHud {
		hudRender()
		if !hudShown {
			pShowWindow.Call(gHudHwnd, 4) // SW_SHOWNA
			hudShown = true
		}
	} else if hudShown {
		pShowWindow.Call(gHudHwnd, 0) // SW_HIDE
		hudShown = false
	}
}

// hudFlash is a no-op when the user has disabled the floating status badge.
func hudFlash() {
	if cfg.ShowHud {
		hudUpdate()
	}
}
