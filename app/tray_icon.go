package main

import (
	"image"
	"image/color"
	"math"
	"unsafe"
)

// trayBadgeColors shares the HUD's jade accent. English is an outlined neutral
// keycap, so shape and letter distinguish the modes even without color.
func trayBadgeColors(on bool, p palette) (fill, ink, border uintptr) {
	if on {
		ink = palDark.bg
		if p.bg == palLight.bg {
			ink = palLight.card
		}
		return p.jade, ink, p.jade
	}
	return p.bg, p.text, p.muted
}

// renderTrayIcon draws on a 16-unit grid, supersampled at the actual tray size.
// Vector letters avoid ClearType fringes and font-dependent alignment at 16px.
// RGBA stores premultiplied alpha, including the transparent rounded corners.
func renderTrayIcon(size int, on bool, p palette) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	fill, ink, border := trayBadgeColors(on, p)
	rgb := func(c uintptr) color.RGBA {
		return color.RGBA{uint8(c), uint8(c >> 8), uint8(c >> 16), 255}
	}
	bg, fg, rim := rgb(fill), rgb(ink), rgb(border)
	v := [][2]float64{{3.5, 3.5}, {6, 3.5}, {8, 9.7}, {10, 3.5}, {12.5, 3.5}, {9.3, 12.5}, {6.7, 12.5}}
	e := [][2]float64{{4.5, 3.5}, {11.5, 3.5}, {11.5, 5.5}, {6.7, 5.5}, {6.7, 7}, {10.8, 7}, {10.8, 9}, {6.7, 9}, {6.7, 10.5}, {11.5, 10.5}, {11.5, 12.5}, {4.5, 12.5}}
	glyph := e
	if on {
		glyph = v
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
	rounded := func(x, y, inset, radius float64) bool {
		qx := math.Abs(x-8) - (8 - inset - radius)
		qy := math.Abs(y-8) - (8 - inset - radius)
		return math.Hypot(math.Max(qx, 0), math.Max(qy, 0))+math.Min(math.Max(qx, qy), 0) <= radius
	}
	const samples = 4
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			var r, g, b, a int
			for sy := 0; sy < samples; sy++ {
				for sx := 0; sx < samples; sx++ {
					px := (float64(x) + (float64(sx)+0.5)/samples) * 16 / float64(size)
					py := (float64(y) + (float64(sy)+0.5)/samples) * 16 / float64(size)
					if !rounded(px, py, 0, 3) {
						continue
					}
					c := bg
					if !on && !rounded(px, py, 1, 2) {
						c = rim
					}
					if insideGlyph(px, py) {
						c = fg
					}
					r += int(c.R)
					g += int(c.G)
					b += int(c.B)
					a += 255
				}
			}
			img.SetRGBA(x, y, color.RGBA{uint8(r / 16), uint8(g / 16), uint8(b / 16), uint8(a / 16)})
		}
	}
	return img
}

var pCreateDIBSection = gdi32.NewProc("CreateDIBSection")

func makeTrayIcon(on bool, size int, p palette) uintptr {
	img := renderTrayIcon(size, on, p)
	bi := bitmapInfo{}
	bi.bmiHeader.biSize = uint32(unsafe.Sizeof(bitmapInfoHeader{}))
	bi.bmiHeader.biWidth = int32(size)
	bi.bmiHeader.biHeight = -int32(size)
	bi.bmiHeader.biPlanes = 1
	bi.bmiHeader.biBitCount = 32
	var bits unsafe.Pointer
	bmp, _, _ := pCreateDIBSection.Call(0, uintptr(unsafe.Pointer(&bi)), 0, uintptr(unsafe.Pointer(&bits)), 0, 0)
	if bmp == 0 {
		return 0
	}
	defer pDeleteObject.Call(bmp)
	pixels := unsafe.Slice((*byte)(bits), len(img.Pix))
	// CreateIconIndirect takes straight BGRA and premultiplies it internally.
	// Supplying Go's premultiplied RGBA directly would darken the curved edges.
	for i := 0; i < len(pixels); i += 4 {
		c := color.NRGBAModel.Convert(color.RGBA{img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3]}).(color.NRGBA)
		pixels[i], pixels[i+1], pixels[i+2], pixels[i+3] = c.B, c.G, c.R, c.A
	}
	// Word-aligned 1bpp AND mask also supports callers that ignore alpha.
	stride := ((size + 15) / 16) * 2
	maskBits := make([]byte, stride*size)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			if img.RGBAAt(x, y).A == 0 {
				maskBits[y*stride+x/8] |= 0x80 >> uint(x%8)
			}
		}
	}
	mask, _, _ := pCreateBitmap.Call(uintptr(size), uintptr(size), 1, 1, uintptr(unsafe.Pointer(&maskBits[0])))
	if mask == 0 {
		return 0
	}
	defer pDeleteObject.Call(mask)
	ii := iconInfo{fIcon: 1, hbmMask: mask, hbmColor: bmp}
	ic, _, _ := pCreateIconIndirect.Call(uintptr(unsafe.Pointer(&ii)))
	return ic
}
