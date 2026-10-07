package main

import (
	"math"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	gdiplus = windows.NewLazySystemDLL("gdiplus.dll")

	pGdiplusStartup          = gdiplus.NewProc("GdiplusStartup")
	pGdiplusShutdown         = gdiplus.NewProc("GdiplusShutdown")
	pGdipCreateFromHDC       = gdiplus.NewProc("GdipCreateFromHDC")
	pGdipDeleteGraphics      = gdiplus.NewProc("GdipDeleteGraphics")
	pGdipSetSmoothingMode    = gdiplus.NewProc("GdipSetSmoothingMode")
	pGdipSetPixelOffsetMode  = gdiplus.NewProc("GdipSetPixelOffsetMode")
	pGdipSetTextRendering    = gdiplus.NewProc("GdipSetTextRenderingHint")
	pGdipCreateSolidFill     = gdiplus.NewProc("GdipCreateSolidFill")
	pGdipDeleteBrush         = gdiplus.NewProc("GdipDeleteBrush")
	pGdipCreatePen1          = gdiplus.NewProc("GdipCreatePen1")
	pGdipDeletePen           = gdiplus.NewProc("GdipDeletePen")
	pGdipCreatePath          = gdiplus.NewProc("GdipCreatePath")
	pGdipDeletePath          = gdiplus.NewProc("GdipDeletePath")
	pGdipResetPath           = gdiplus.NewProc("GdipResetPath")
	pGdipAddPathArc          = gdiplus.NewProc("GdipAddPathArc")
	pGdipAddPathLine         = gdiplus.NewProc("GdipAddPathLine")
	pGdipClosePathFigure     = gdiplus.NewProc("GdipClosePathFigure")
	pGdipFillPath                = gdiplus.NewProc("GdipFillPath")
	pGdipDrawPath                = gdiplus.NewProc("GdipDrawPath")
	pGdipFillRectangle           = gdiplus.NewProc("GdipFillRectangle")
	pGdipDrawRectangle           = gdiplus.NewProc("GdipDrawRectangle")
	pGdipFillEllipse             = gdiplus.NewProc("GdipFillEllipse")
	pGdipDrawEllipse             = gdiplus.NewProc("GdipDrawEllipse")
	pGdipDrawLine                = gdiplus.NewProc("GdipDrawLine")
	pGdipCreateLineBrushFromRect = gdiplus.NewProc("GdipCreateLineBrushFromRect")
)

const (
	SmoothingModeAntiAlias       = 2
	SmoothingModeHighQuality     = 4
	PixelOffsetModeHighQuality   = 2
	PixelOffsetModeHalf          = 4
	TextRenderingHintClearType   = 5
	UnitPixel                    = 2
	FillModeAlternate            = 0
)

type gdiplusStartupInput struct {
	GdiplusVersion           uint32
	DebugEventCallback       uintptr
	SuppressBackgroundThread bool
	SuppressExternalCodecs   bool
}

var (
	gdipToken uintptr
	gdipOnce  sync.Once
)

// initGDIPlus initializes the GDI+ subsystem once for the process.
func initGDIPlus() bool {
	gdipOnce.Do(func() {
		input := gdiplusStartupInput{
			GdiplusVersion: 1,
		}
		var token uintptr
		r, _, _ := pGdiplusStartup.Call(
			uintptr(unsafe.Pointer(&token)),
			uintptr(unsafe.Pointer(&input)),
			0,
		)
		if r == 0 {
			gdipToken = token
		}
	})
	return gdipToken != 0
}

// ARGB creates a 32-bit ARGB color for GDI+ from alpha, red, green, blue.
func ARGB(a, r, g, b byte) uint32 {
	return (uint32(a) << 24) | (uint32(r) << 16) | (uint32(g) << 8) | uint32(b)
}

// RGBA creates a 32-bit fully opaque ARGB color for GDI+ from red, green, blue.
func RGBA(r, g, b byte) uint32 {
	return ARGB(255, r, g, b)
}

// HexARGB converts a 24-bit 0xRRGGBB color to 32-bit opaque 0xFFRRGGBB.
func HexARGB(rgb uint32) uint32 {
	return 0xFF000000 | rgb
}

// HexAlphaARGB converts an alpha byte and 24-bit 0xRRGGBB color to 32-bit ARGB.
func HexAlphaARGB(a byte, rgb uint32) uint32 {
	return (uint32(a) << 24) | (rgb & 0x00FFFFFF)
}

// Gfx wraps a GDI+ Graphics context with high-level anti-aliased drawing helpers.
type Gfx struct {
	native uintptr
}

// NewGfx creates a new GDI+ Graphics context from an HDC and sets high-quality anti-aliasing.
func NewGfx(hdc uintptr) *Gfx {
	if !initGDIPlus() {
		return nil
	}
	var g uintptr
	r, _, _ := pGdipCreateFromHDC.Call(hdc, uintptr(unsafe.Pointer(&g)))
	if r != 0 || g == 0 {
		return nil
	}
	pGdipSetSmoothingMode.Call(g, SmoothingModeAntiAlias)
	pGdipSetPixelOffsetMode.Call(g, PixelOffsetModeHighQuality)
	pGdipSetTextRendering.Call(g, TextRenderingHintClearType)
	return &Gfx{native: g}
}

// Destroy releases the GDI+ Graphics context.
func (g *Gfx) Destroy() {
	if g != nil && g.native != 0 {
		pGdipDeleteGraphics.Call(g.native)
		g.native = 0
	}
}

// FillRect fills an axis-aligned rectangle with a solid color.
func (g *Gfx) FillRect(x, y, w, h float32, color uint32) {
	if g == nil || g.native == 0 || w <= 0 || h <= 0 {
		return
	}
	var brush uintptr
	pGdipCreateSolidFill.Call(uintptr(color), uintptr(unsafe.Pointer(&brush)))
	if brush != 0 {
		pGdipFillRectangle.Call(g.native, brush,
			*(*uintptr)(unsafe.Pointer(&x)),
			*(*uintptr)(unsafe.Pointer(&y)),
			*(*uintptr)(unsafe.Pointer(&w)),
			*(*uintptr)(unsafe.Pointer(&h)),
		)
		pGdipDeleteBrush.Call(brush)
	}
}

// DrawRect draws a 1px or thicker stroke rectangle with a solid color.
func (g *Gfx) DrawRect(x, y, w, h float32, color uint32, lineWidth float32) {
	if g == nil || g.native == 0 || w <= 0 || h <= 0 {
		return
	}
	var pen uintptr
	pGdipCreatePen1.Call(uintptr(color), *(*uintptr)(unsafe.Pointer(&lineWidth)), UnitPixel, uintptr(unsafe.Pointer(&pen)))
	if pen != 0 {
		pGdipDrawRectangle.Call(g.native, pen,
			*(*uintptr)(unsafe.Pointer(&x)),
			*(*uintptr)(unsafe.Pointer(&y)),
			*(*uintptr)(unsafe.Pointer(&w)),
			*(*uintptr)(unsafe.Pointer(&h)),
		)
		pGdipDeletePen.Call(pen)
	}
}

// createRoundRectPath creates a closed GraphicsPath of a rounded rectangle.
func createRoundRectPath(x, y, w, h, radius float32) uintptr {
	var path uintptr
	pGdipCreatePath.Call(FillModeAlternate, uintptr(unsafe.Pointer(&path)))
	if path == 0 {
		return 0
	}

	maxR := float32(math.Min(float64(w), float64(h))) / 2.0
	if radius > maxR {
		radius = maxR
	}
	if radius <= 0 {
		radius = 0.1
	}
	d := radius * 2

	// Helper to pass float32 as uintptr in syscall
	f2u := func(f float32) uintptr {
		return *(*uintptr)(unsafe.Pointer(&f))
	}

	// Top-left arc
	pGdipAddPathArc.Call(path, f2u(x), f2u(y), f2u(d), f2u(d), f2u(180), f2u(90))
	// Top line & Top-right arc
	pGdipAddPathArc.Call(path, f2u(x+w-d), f2u(y), f2u(d), f2u(d), f2u(270), f2u(90))
	// Right line & Bottom-right arc
	pGdipAddPathArc.Call(path, f2u(x+w-d), f2u(y+h-d), f2u(d), f2u(d), f2u(0), f2u(90))
	// Bottom line & Bottom-left arc
	pGdipAddPathArc.Call(path, f2u(x), f2u(y+h-d), f2u(d), f2u(d), f2u(90), f2u(90))
	pGdipClosePathFigure.Call(path)

	return path
}

// FillRoundedRect fills a rounded rectangle with smooth subpixel anti-aliasing.
func (g *Gfx) FillRoundedRect(x, y, w, h, radius float32, color uint32) {
	if g == nil || g.native == 0 || w <= 0 || h <= 0 {
		return
	}
	path := createRoundRectPath(x, y, w, h, radius)
	if path == 0 {
		return
	}
	defer pGdipDeletePath.Call(path)

	var brush uintptr
	pGdipCreateSolidFill.Call(uintptr(color), uintptr(unsafe.Pointer(&brush)))
	if brush != 0 {
		pGdipFillPath.Call(g.native, brush, path)
		pGdipDeleteBrush.Call(brush)
	}
}

// DrawRoundedRect strokes a rounded rectangle with smooth subpixel anti-aliasing.
func (g *Gfx) DrawRoundedRect(x, y, w, h, radius float32, color uint32, lineWidth float32) {
	if g == nil || g.native == 0 || w <= 0 || h <= 0 {
		return
	}
	// Inset slightly by half the stroke width for sharp pixel alignment
	half := lineWidth / 2.0
	path := createRoundRectPath(x+half, y+half, w-lineWidth, h-lineWidth, radius)
	if path == 0 {
		return
	}
	defer pGdipDeletePath.Call(path)

	var pen uintptr
	pGdipCreatePen1.Call(uintptr(color), *(*uintptr)(unsafe.Pointer(&lineWidth)), UnitPixel, uintptr(unsafe.Pointer(&pen)))
	if pen != 0 {
		pGdipDrawPath.Call(g.native, pen, path)
		pGdipDeletePen.Call(pen)
	}
}

// FillCircle fills an anti-aliased circle at (cx, cy) with radius r.
func (g *Gfx) FillCircle(cx, cy, r float32, color uint32) {
	if g == nil || g.native == 0 || r <= 0 {
		return
	}
	var brush uintptr
	pGdipCreateSolidFill.Call(uintptr(color), uintptr(unsafe.Pointer(&brush)))
	if brush != 0 {
		x, y, d := cx-r, cy-r, r*2
		f2u := func(f float32) uintptr { return *(*uintptr)(unsafe.Pointer(&f)) }
		pGdipFillEllipse.Call(g.native, brush, f2u(x), f2u(y), f2u(d), f2u(d))
		pGdipDeleteBrush.Call(brush)
	}
}

// DrawCircle strokes an anti-aliased circle at (cx, cy) with radius r.
func (g *Gfx) DrawCircle(cx, cy, r float32, color uint32, lineWidth float32) {
	if g == nil || g.native == 0 || r <= 0 {
		return
	}
	var pen uintptr
	pGdipCreatePen1.Call(uintptr(color), *(*uintptr)(unsafe.Pointer(&lineWidth)), UnitPixel, uintptr(unsafe.Pointer(&pen)))
	if pen != 0 {
		x, y, d := cx-r, cy-r, r*2
		f2u := func(f float32) uintptr { return *(*uintptr)(unsafe.Pointer(&f)) }
		pGdipDrawEllipse.Call(g.native, pen, f2u(x), f2u(y), f2u(d), f2u(d))
		pGdipDeletePen.Call(pen)
	}
}

// DrawLine draws an anti-aliased line from (x1, y1) to (x2, y2).
func (g *Gfx) DrawLine(x1, y1, x2, y2 float32, color uint32, lineWidth float32) {
	if g == nil || g.native == 0 {
		return
	}
	var pen uintptr
	pGdipCreatePen1.Call(uintptr(color), *(*uintptr)(unsafe.Pointer(&lineWidth)), UnitPixel, uintptr(unsafe.Pointer(&pen)))
	if pen != 0 {
		f2u := func(f float32) uintptr { return *(*uintptr)(unsafe.Pointer(&f)) }
		pGdipDrawLine.Call(g.native, pen, f2u(x1), f2u(y1), f2u(x2), f2u(y2))
		pGdipDeletePen.Call(pen)
	}
}

type gpRectF struct {
	X, Y, Width, Height float32
}

// FillRoundedRectGradient fills a rounded rectangle with a smooth vertical linear gradient.
func (g *Gfx) FillRoundedRectGradient(x, y, w, h, radius float32, colorTop, colorBottom uint32) {
	if g == nil || g.native == 0 || w <= 0 || h <= 0 {
		return
	}
	path := createRoundRectPath(x, y, w, h, radius)
	if path == 0 {
		return
	}
	defer pGdipDeletePath.Call(path)

	rc := gpRectF{x, y, w, h}
	var brush uintptr
	pGdipCreateLineBrushFromRect.Call(
		uintptr(unsafe.Pointer(&rc)),
		uintptr(colorTop),
		uintptr(colorBottom),
		1, // LinearGradientModeVertical
		0, // WrapModeTile
		uintptr(unsafe.Pointer(&brush)),
	)
	if brush != 0 {
		pGdipFillPath.Call(g.native, brush, path)
		pGdipDeleteBrush.Call(brush)
	}
}

// DrawGlassCard renders a liquid glass surface with translucent frosted gradient,
// subtle anti-aliased border, and a delicate top specular sheen line.
func (g *Gfx) DrawGlassCard(x, y, w, h, radius float32, isDark bool, hover bool) {
	if isDark {
		// Dark Mode Liquid Glass
		bgTop := HexAlphaARGB(0x38, 0x222632)    // ~22% opacity #222632
		bgBottom := HexAlphaARGB(0x22, 0x161820) // ~13% opacity #161820
		border := HexAlphaARGB(0x35, 0xFFFFFF)   // 21% white border
		sheen := HexAlphaARGB(0x45, 0xFFFFFF)    // 27% top specular sheen
		if hover {
			bgTop = HexAlphaARGB(0x48, 0x2B303F)
			bgBottom = HexAlphaARGB(0x30, 0x1E212B)
			border = HexAlphaARGB(0x50, 0xFFFFFF)
		}
		g.FillRoundedRectGradient(x, y, w, h, radius, bgTop, bgBottom)
		g.DrawRoundedRect(x, y, w, h, radius, border, 1.0)
		if radius > 4 && w > radius*2 {
			g.DrawLine(x+radius, y+1, x+w-radius, y+1, sheen, 1.0)
		}
	} else {
		// Light Mode Liquid Glass
		bgTop := HexAlphaARGB(0xF0, 0xFFFFFF)    // 94% white
		bgBottom := HexAlphaARGB(0xD0, 0xF4F6F9) // 81% light slate
		border := HexAlphaARGB(0x60, 0xD0D8E2)   // subtle frosted border
		sheen := HexAlphaARGB(0xAA, 0xFFFFFF)    // pure white top specular
		if hover {
			bgTop = HexAlphaARGB(0xFA, 0xFFFFFF)
			bgBottom = HexAlphaARGB(0xE0, 0xECF0F6)
			border = HexAlphaARGB(0x80, 0xB0BCCB)
		}
		g.FillRoundedRectGradient(x, y, w, h, radius, bgTop, bgBottom)
		g.DrawRoundedRect(x, y, w, h, radius, border, 1.0)
		if radius > 4 && w > radius*2 {
			g.DrawLine(x+radius, y+1, x+w-radius, y+1, sheen, 1.0)
		}
	}
}

