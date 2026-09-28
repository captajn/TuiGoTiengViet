package main

import "unsafe"

var pStretchBlt = gdi32.NewProc("StretchBlt")
var pTrackMouseEvent = user32.NewProc("TrackMouseEvent")

type navMouseTrack struct {
	size, flags uint32
	hwnd        uintptr
	hoverTime   uint32
}

// drawNavIcon draws all navigation symbols on the same 24-unit grid. Render
// at 4x the target size, then downsample so diagonal and curved strokes remain
// smooth at fractional Windows display scales. No icon font is required.
func drawNavIcon(dc uintptr, x, y, size int32, kind int, color, background uintptr) {
	const sample = 4
	n := size * sample
	mem, _, _ := pCreateCompatibleDC.Call(dc)
	if mem == 0 {
		return
	}
	defer pDeleteDC.Call(mem)
	bmp, _, _ := pCreateCompatibleBitmap.Call(dc, uintptr(n), uintptr(n))
	if bmp == 0 {
		return
	}
	defer pDeleteObject.Call(bmp)
	oldBitmap, _, _ := pSelectObject.Call(mem, bmp)
	defer pSelectObject.Call(mem, oldBitmap)
	bg, _, _ := pCreateSolidBrush.Call(background)
	r := rect{0, 0, n, n}
	pFillRect.Call(mem, uintptr(unsafe.Pointer(&r)), bg)
	pDeleteObject.Call(bg)

	u := func(v int32) uintptr { return uintptr((v*n + 12) / 24) }
	pen, _, _ := pCreatePen.Call(PS_SOLID, u(2), color)
	oldPen, _, _ := pSelectObject.Call(mem, pen)
	defer pDeleteObject.Call(pen)
	defer pSelectObject.Call(mem, oldPen)
	nullBrush, _, _ := pGetStockObject.Call(NULL_BRUSH)
	oldBrush, _, _ := pSelectObject.Call(mem, nullBrush)
	defer pSelectObject.Call(mem, oldBrush)
	line := func(coords ...int32) {
		pMoveToEx.Call(mem, u(coords[0]), u(coords[1]), 0)
		for i := 2; i < len(coords); i += 2 {
			pLineTo.Call(mem, u(coords[i]), u(coords[i+1]))
		}
	}
	circle := func(l, t, r, b int32) {
		pEllipse.Call(mem, u(l), u(t), u(r), u(b))
	}
	switch kind {
	case 0: // Home
		line(3, 10, 12, 3, 21, 10)
		line(5, 9, 5, 21, 10, 21, 10, 14, 14, 14, 14, 21, 19, 21, 19, 9)
	case 1: // Keyboard
		pRoundRect.Call(mem, u(2), u(5), u(22), u(19), u(4), u(4))
		for _, row := range []int32{9, 12} {
			for _, col := range []int32{6, 10, 14, 18} {
				line(col, row, col+1, row)
			}
		}
		line(7, 16, 17, 16)
	case 2: // Command key: four connected loops
		pRoundRect.Call(mem, u(3), u(3), u(9), u(9), u(5), u(5))
		pRoundRect.Call(mem, u(15), u(3), u(21), u(9), u(5), u(5))
		pRoundRect.Call(mem, u(3), u(15), u(9), u(21), u(5), u(5))
		pRoundRect.Call(mem, u(15), u(15), u(21), u(21), u(5), u(5))
		line(9, 6, 9, 18)
		line(15, 6, 15, 18)
		line(6, 9, 18, 9)
		line(6, 15, 18, 15)
	case 3: // Appearance / sliders
		line(4, 3, 4, 7)
		line(4, 13, 4, 21)
		circle(1, 7, 7, 13)
		line(12, 3, 12, 12)
		line(12, 18, 12, 21)
		circle(9, 12, 15, 18)
		line(20, 3, 20, 5)
		line(20, 11, 20, 21)
		circle(17, 5, 23, 11)
	case 4: // Information
		circle(2, 2, 22, 22)
		line(12, 7, 12, 8)
		line(10, 11, 12, 11, 12, 17)
		line(10, 17, 14, 17)
	}
	oldMode, _, _ := pSetStretchBltMode.Call(dc, 4 /* HALFTONE */)
	pStretchBlt.Call(dc, uintptr(x), uintptr(y), uintptr(size), uintptr(size),
		mem, 0, 0, uintptr(n), uintptr(n), 0x00CC0020 /* SRCCOPY */)
	pSetStretchBltMode.Call(dc, oldMode)
}
