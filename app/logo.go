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

// scaleLogoBgra bilinear-scales the decoded logo (gLogoBgra, BGRA top-down)
// into dst as BGRA top-down w×h. Pure-Go replacement for StretchDIBits: the
// old GDI path left screen-compatible bitmaps uninitialized (black icons),
// so pixels are written directly into DIB section memory like the tray icons.
func scaleLogoBgra(dst []byte, w, h int32) {
	sw, sh := int(gLogoW), int(gLogoH)
	dw, dh := int(w), int(h)
	if sw <= 0 || sh <= 0 || dw <= 0 || dh <= 0 {
		return
	}
	for y := 0; y < dh; y++ {
		// Source sample center in src coords.
		fy := (float64(y)+0.5)*float64(sh)/float64(dh) - 0.5
		y0 := int(fy)
		fy -= float64(y0)
		if y0 < 0 {
			y0, fy = 0, 0
		}
		y1 := y0 + 1
		if y1 >= sh {
			y1 = sh - 1
		}
		row0, row1 := y0*sw*4, y1*sw*4
		drow := y * dw * 4
		for x := 0; x < dw; x++ {
			fx := (float64(x)+0.5)*float64(sw)/float64(dw) - 0.5
			x0 := int(fx)
			fx -= float64(x0)
			if x0 < 0 {
				x0, fx = 0, 0
			}
			x1 := x0 + 1
			if x1 >= sw {
				x1 = sw - 1
			}
			w00 := (1 - fx) * (1 - fy)
			w10 := fx * (1 - fy)
			w01 := (1 - fx) * fy
			w11 := fx * fy
			s00, s10 := row0+x0*4, row0+x1*4
			s01, s11 := row1+x0*4, row1+x1*4
			d := drow + x*4
			for c := 0; c < 4; c++ {
				v := float64(gLogoBgra[s00+c])*w00 +
					float64(gLogoBgra[s10+c])*w10 +
					float64(gLogoBgra[s01+c])*w01 +
					float64(gLogoBgra[s11+c])*w11
				dst[d+c] = byte(v + 0.5)
			}
		}
	}
}

var (
	gMascotDC, gMascotBmp, gMascotOld uintptr
	gMascotW, gMascotH                int32
)

func releaseMascotCache() {
	if gMascotDC != 0 {
		pSelectObject.Call(gMascotDC, gMascotOld)
		pDeleteDC.Call(gMascotDC)
	}
	if gMascotBmp != 0 {
		pDeleteObject.Call(gMascotBmp)
	}
	gMascotDC, gMascotBmp, gMascotOld, gMascotW, gMascotH = 0, 0, 0, 0, 0
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

	if gMascotDC == 0 || gMascotW != w || gMascotH != h {
		releaseMascotCache()
		// 32bpp DIB section: pixels written straight into bitmap memory.
		// Screen-compatible bitmaps + StretchDIBits left the icon black.
		var bi bitmapInfo
		bi.bmiHeader.biSize = uint32(unsafe.Sizeof(bitmapInfoHeader{}))
		bi.bmiHeader.biWidth = w
		bi.bmiHeader.biHeight = -h // negative for top-down DIB
		bi.bmiHeader.biPlanes = 1
		bi.bmiHeader.biBitCount = 32
		bi.bmiHeader.biCompression = 0 // BI_RGB
		var bits unsafe.Pointer
		bmp, _, _ := pCreateDIBSection.Call(dc, uintptr(unsafe.Pointer(&bi)), 0, uintptr(unsafe.Pointer(&bits)), 0, 0)
		mem, _, _ := pCreateCompatibleDC.Call(dc)
		if mem != 0 && bmp != 0 && bits != nil {
			scaleLogoBgra(unsafe.Slice((*byte)(bits), int(w)*int(h)*4), w, h)
			old, _, _ := pSelectObject.Call(mem, bmp)
			gMascotDC, gMascotBmp, gMascotOld, gMascotW, gMascotH = mem, bmp, old, w, h
		} else {
			if bmp != 0 {
				pDeleteObject.Call(bmp)
			}
			if mem != 0 {
				pDeleteDC.Call(mem)
			}
		}
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

	if gMascotDC != 0 {
		pBitBlt.Call(dc, uintptr(x), uintptr(y), uintptr(w), uintptr(h), gMascotDC, 0, 0, 0x00CC0020 /* SRCCOPY */)
	} else {
		// Cache build failed: draw via a throwaway DIB section + BitBlt
		// instead of StretchDIBits straight into the window DC.
		var bi bitmapInfo
		bi.bmiHeader.biSize = uint32(unsafe.Sizeof(bitmapInfoHeader{}))
		bi.bmiHeader.biWidth = w
		bi.bmiHeader.biHeight = -h // negative for top-down DIB
		bi.bmiHeader.biPlanes = 1
		bi.bmiHeader.biBitCount = 32
		bi.bmiHeader.biCompression = 0 // BI_RGB
		var bits unsafe.Pointer
		bmp, _, _ := pCreateDIBSection.Call(dc, uintptr(unsafe.Pointer(&bi)), 0, uintptr(unsafe.Pointer(&bits)), 0, 0)
		mem, _, _ := pCreateCompatibleDC.Call(dc)
		if bmp != 0 && bits != nil && mem != 0 {
			scaleLogoBgra(unsafe.Slice((*byte)(bits), int(w)*int(h)*4), w, h)
			old, _, _ := pSelectObject.Call(mem, bmp)
			pBitBlt.Call(dc, uintptr(x), uintptr(y), uintptr(w), uintptr(h), mem, 0, 0, 0x00CC0020 /* SRCCOPY */)
			pSelectObject.Call(mem, old)
		}
		if bmp != 0 {
			pDeleteObject.Call(bmp)
		}
		if mem != 0 {
			pDeleteDC.Call(mem)
		}
	}

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

// createMascotIcon creates an HICON from the embedded mascot image at the
// specified size. Pixels are scaled into a 32bpp DIB section — the same
// backing store the tray icons use — instead of StretchDIBits into a
// screen-compatible bitmap, which rendered the icon black.
func createMascotIcon(sz int32) uintptr {
	if len(gLogoBgra) == 0 {
		initLogo()
	}
	if len(gLogoBgra) == 0 {
		return 0
	}
	var bi bitmapInfo
	bi.bmiHeader.biSize = uint32(unsafe.Sizeof(bitmapInfoHeader{}))
	bi.bmiHeader.biWidth = sz
	bi.bmiHeader.biHeight = -sz // negative for top-down DIB
	bi.bmiHeader.biPlanes = 1
	bi.bmiHeader.biBitCount = 32
	bi.bmiHeader.biCompression = 0 // BI_RGB
	var bits unsafe.Pointer
	bmp, _, _ := pCreateDIBSection.Call(0, uintptr(unsafe.Pointer(&bi)), 0, uintptr(unsafe.Pointer(&bits)), 0, 0)
	if bmp == 0 || bits == nil {
		if bmp != 0 {
			pDeleteObject.Call(bmp)
		}
		return 0
	}
	scaleLogoBgra(unsafe.Slice((*byte)(bits), int(sz)*int(sz)*4), sz, sz)

	// Opaque AND mask — alpha comes from the color bitmap's channel.
	stride := ((int(sz) + 15) / 16) * 2
	maskBits := make([]byte, stride*int(sz))
	mask, _, _ := pCreateBitmap.Call(uintptr(sz), uintptr(sz), 1, 1, uintptr(unsafe.Pointer(&maskBits[0])))

	ii := iconInfo{fIcon: 1, hbmMask: mask, hbmColor: bmp}
	ic, _, _ := pCreateIconIndirect.Call(uintptr(unsafe.Pointer(&ii)))
	pDeleteObject.Call(bmp)
	if mask != 0 {
		pDeleteObject.Call(mask)
	}
	return ic
}
