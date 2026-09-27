package main

import (
	"bytes"
	_ "embed"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"unsafe"
)

//go:embed logo.jpg
var logoJpgData []byte

var (
	gLogoBgra []byte
	gLogoW    int32
	gLogoH    int32
)

func initLogo() {
	if len(logoJpgData) == 0 {
		return
	}
	img, _, err := image.Decode(bytes.NewReader(logoJpgData))
	if err != nil {
		return
	}
	b := img.Bounds()
	gLogoW = int32(b.Dx())
	gLogoH = int32(b.Dy())
	gLogoBgra = make([]byte, gLogoW*gLogoH*4)

	idx := 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := img.At(x, y).RGBA()
			// Win32 DIB expects BGRA, top-down
			gLogoBgra[idx+0] = byte(bl >> 8)
			gLogoBgra[idx+1] = byte(g >> 8)
			gLogoBgra[idx+2] = byte(r >> 8)
			gLogoBgra[idx+3] = 0xFF
			idx += 4
		}
	}
}

// drawAppMascot renders the high-res chibi character logo into the given rect
// with rounded corners and an optional delicate golden hairline border.
func drawAppMascot(dc uintptr, x, y, w, h int32, radius int32) {
	if len(gLogoBgra) == 0 {
		initLogo()
	}
	if len(gLogoBgra) == 0 {
		drawLogo(dc, x+w/2, y+h/2)
		return
	}

	var rgn uintptr
	if radius > 0 {
		rgn, _, _ = pCreateRoundRectRgn.Call(
			uintptr(x), uintptr(y),
			uintptr(x+w+1), uintptr(y+h+1),
			uintptr(radius), uintptr(radius),
		)
		pSelectClipRgn.Call(dc, rgn)
	}

	var bi bitmapInfo
	bi.bmiHeader.biSize = uint32(unsafe.Sizeof(bitmapInfoHeader{}))
	bi.bmiHeader.biWidth = gLogoW
	bi.bmiHeader.biHeight = -gLogoH // negative for top-down DIB
	bi.bmiHeader.biPlanes = 1
	bi.bmiHeader.biBitCount = 32
	bi.bmiHeader.biCompression = 0 // BI_RGB

	pSetStretchBltMode.Call(dc, 4 /* HALFTONE */)
	pStretchDIBits.Call(
		dc,
		uintptr(x), uintptr(y), uintptr(w), uintptr(h),
		0, 0, uintptr(gLogoW), uintptr(gLogoH),
		uintptr(unsafe.Pointer(&gLogoBgra[0])),
		uintptr(unsafe.Pointer(&bi)),
		0,          /* DIB_RGB_COLORS */
		0x00CC0020, /* SRCCOPY */
	)

	if radius > 0 && rgn != 0 {
		pSelectClipRgn.Call(dc, 0)
		pDeleteObject.Call(rgn)

		// Subtle golden rim around mascot
		nullBr, _, _ := pGetStockObject.Call(NULL_BRUSH)
		oldB, _, _ := pSelectObject.Call(dc, nullBr)
		if penGoldDim != 0 {
			oldP, _, _ := pSelectObject.Call(dc, penGoldDim)
			pRoundRect.Call(dc, uintptr(x), uintptr(y), uintptr(x+w), uintptr(y+h), uintptr(radius), uintptr(radius))
			pSelectObject.Call(dc, oldP)
		}
		pSelectObject.Call(dc, oldB)
	}
}

// createMascotIcon creates an HICON from the embedded mascot image at the specified size.
func createMascotIcon(sz int32) uintptr {
	if len(gLogoBgra) == 0 {
		initLogo()
	}
	if len(gLogoBgra) == 0 {
		return 0
	}
	screen, _, _ := pGetDC.Call(0)
	memC, _, _ := pCreateCompatibleDC.Call(screen)
	bmp, _, _ := pCreateCompatibleBitmap.Call(screen, uintptr(sz), uintptr(sz))
	var maskBits [4096]byte
	mask, _, _ := pCreateBitmap.Call(uintptr(sz), uintptr(sz), 1, 1, uintptr(unsafe.Pointer(&maskBits[0])))
	oldC, _, _ := pSelectObject.Call(memC, bmp)

	drawAppMascot(memC, 0, 0, sz, sz, sz/4)

	pSelectObject.Call(memC, oldC)
	ii := iconInfo{fIcon: 1, hbmMask: mask, hbmColor: bmp}
	ic, _, _ := pCreateIconIndirect.Call(uintptr(unsafe.Pointer(&ii)))
	pDeleteObject.Call(bmp)
	pDeleteObject.Call(mask)
	pDeleteDC.Call(memC)
	pReleaseDC.Call(0, screen)
	return ic
}

