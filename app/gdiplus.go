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
	if brush == 0 {
		return
	}
	defer pGdipDeleteBrush.Call(brush)

	if radius <= 0 {
		pGdipFillRectangle.Call(g.native, brush,
			*(*uintptr)(unsafe.Pointer(&x)),
			*(*uintptr)(unsafe.Pointer(&y)),
			*(*uintptr)(unsafe.Pointer(&w)),
			*(*uintptr)(unsafe.Pointer(&h)),
		)
		return
	}

	path := createRoundRectPath(x, y, w, h, radius)
	if path == 0 {
		return
	}
	defer pGdipDeletePath.Call(path)
	pGdipFillPath.Call(g.native, brush, path)
}

// DrawGlassCard renders a liquid glass surface with translucent frosted gradient,
// subtle anti-aliased border, ambient depth shadow, and a delicate top specular sheen line.
func (g *Gfx) DrawGlassCard(x, y, w, h, radius float32, isDark bool, hover bool) {
	g.DrawGlassCardExt(x, y, w, h, radius, isDark, hover, false)
}

// DrawGlassCardExt renders an advanced liquid glass surface with optional active accent glow.
func (g *Gfx) DrawGlassCardExt(x, y, w, h, radius float32, isDark bool, hover, active bool) {
	if g == nil || g.native == 0 || w <= 0 || h <= 0 {
		return
	}

	// Adjust corner radius subtly per style to match demo personality
	switch cfg.Style {
	case styleCyberNeon:
		if radius > 7 {
			radius = 7
		}
	case styleMatchaLatte:
		if radius >= 8 && radius < 13 {
			radius = 13
		}
	case styleMaterialTeal:
		if radius >= 8 && radius < 15 {
			radius = 15
		}
	}

	// 1. Ambient soft drop shadow or outer neon halo (Cơ bản stays flat)
	if h > 8 && w > 8 && cfg.Style != styleBasic {
		if cfg.Style == styleCyberNeon {
			glowCol := uint32(0x00F2FE)
			if !isDark {
				glowCol = 0x0284C7
			}
			if active {
				g.DrawRoundedRect(x-1.0, y-1.0, w+2.0, h+2.0, radius+1.0, HexAlphaARGB(0x45, glowCol), 3.0)
			} else {
				g.DrawRoundedRect(x-0.5, y-0.5, w+1.0, h+1.0, radius+0.5, HexAlphaARGB(0x22, glowCol), 2.0)
			}
		}
		shadowCol := HexAlphaARGB(0x16, 0x000000)
		if isDark {
			shadowCol = HexAlphaARGB(0x34, 0x000000)
		} else if cfg.Style == styleIosPure || cfg.Style == styleMatchaLatte {
			shadowCol = HexAlphaARGB(0x20, 0x000000)
		}
		g.FillRoundedRect(x, y+2.0, w, h, radius, shadowCol)
		g.FillRoundedRect(x+1.0, y+3.5, w-2.0, h-1.5, radius, HexAlphaARGB(0x10, 0x000000))
	}

	var bgTop, bgBottom, border, sheen uint32
	var borderW float32 = 1.1

	switch cfg.Style {
	case styleBasic:
		// Cơ bản: opaque flat card, solid accent fill when selected — no
		// glass translucency or tinted wash (hard to read on light gray).
		bgTop = HexAlphaARGB(0xFF, argbCard&0x00FFFFFF)
		bgBottom = bgTop
		border = HexAlphaARGB(0xFF, argbBorder&0x00FFFFFF)
		sheen = 0
		if active {
			bgTop = argbSwitchOn
			bgBottom = argbSwitchOn
			border = argbSwitchOn
			borderW = 1.2
		} else if hover {
			bgTop = HexAlphaARGB(0xFF, argbKeycap&0x00FFFFFF)
			bgBottom = bgTop
			border = HexAlphaARGB(0xFF, argbMuted&0x00FFFFFF)
		}

	case styleIosPure:
		if isDark {
			// Apple iOS 18 Pro Dark: Elevated #1C1C1E charcoal card over pure #000000 OLED black
			bgTop = HexAlphaARGB(0xFA, 0x242426)
			bgBottom = HexAlphaARGB(0xF8, 0x1C1C1E)
			border = HexAlphaARGB(0xB0, 0x38383A)
			sheen = HexAlphaARGB(0x35, 0xFFFFFF)
			if active {
				bgTop = HexAlphaARGB(0x42, 0x0A84FF)
				bgBottom = HexAlphaARGB(0x24, 0x0A84FF)
				border = HexAlphaARGB(0xF0, 0x0A84FF)
				sheen = HexAlphaARGB(0x65, 0xFFFFFF)
				borderW = 1.5
			} else if hover {
				bgTop = HexAlphaARGB(0xFF, 0x2C2C2E)
				bgBottom = HexAlphaARGB(0xFA, 0x242426)
				border = HexAlphaARGB(0xD0, 0x48484A)
			}
		} else {
			// Apple iOS 18 Pro Light: Crisp pure white 3D card over silver-gray grouped backdrop
			bgTop = HexAlphaARGB(0xFF, 0xFFFFFF)
			bgBottom = HexAlphaARGB(0xFC, 0xF8FAFC)
			border = HexAlphaARGB(0xA0, 0xC5CDD8)
			sheen = HexAlphaARGB(0xFF, 0xFFFFFF)
			if active {
				bgTop = HexAlphaARGB(0x24, 0x007AFF)
				bgBottom = HexAlphaARGB(0x14, 0x007AFF)
				border = HexAlphaARGB(0xE8, 0x007AFF)
				borderW = 1.5
			} else if hover {
				bgTop = HexAlphaARGB(0xFF, 0xFFFFFF)
				bgBottom = HexAlphaARGB(0xFF, 0xEFF4FA)
				border = HexAlphaARGB(0xC0, 0x94A3B8)
			}
		}

	case styleCyberNeon:
		if isDark {
			// Cyber Neon AMOLED Dark: Deep navy-black surface with glowing cyan/mint neon border
			bgTop = HexAlphaARGB(0xEA, 0x0E172A)
			bgBottom = HexAlphaARGB(0xF2, 0x080D18)
			border = HexAlphaARGB(0xA8, 0x0EA5E9)
			sheen = HexAlphaARGB(0x90, 0x00F2FE)
			borderW = 1.2
			if active {
				bgTop = HexAlphaARGB(0x45, 0x00F2FE)
				bgBottom = HexAlphaARGB(0x28, 0x10B981)
				border = HexAlphaARGB(0xFF, 0x00F2FE)
				sheen = HexAlphaARGB(0xD0, 0x00F2FE)
				borderW = 1.6
			} else if hover {
				bgTop = HexAlphaARGB(0xF0, 0x15233E)
				bgBottom = HexAlphaARGB(0xF5, 0x0C1526)
				border = HexAlphaARGB(0xE0, 0x38BDF8)
				sheen = HexAlphaARGB(0xB0, 0x00F2FE)
			}
		} else {
			// Cyber Neon Daylight: Crisp ice-white tech panel with electric sky/cyan rim
			bgTop = HexAlphaARGB(0xFC, 0xFFFFFF)
			bgBottom = HexAlphaARGB(0xF4, 0xF0F9FF)
			border = HexAlphaARGB(0xB8, 0x38BDF8)
			sheen = HexAlphaARGB(0xFF, 0xFFFFFF)
			borderW = 1.2
			if active {
				bgTop = HexAlphaARGB(0x38, 0x38BDF8)
				bgBottom = HexAlphaARGB(0x22, 0x10B981)
				border = HexAlphaARGB(0xFF, 0x0284C7)
				sheen = HexAlphaARGB(0xFF, 0xFFFFFF)
				borderW = 1.6
			} else if hover {
				bgTop = HexAlphaARGB(0xFF, 0xFFFFFF)
				bgBottom = HexAlphaARGB(0xFA, 0xE0F2FE)
				border = HexAlphaARGB(0xE0, 0x0284C7)
			}
		}

	case styleMatchaLatte:
		if isDark {
			// Matcha Espresso Dark: Rich roasted forest card with warm sage/matcha rim
			bgTop = HexAlphaARGB(0xF4, 0x213029)
			bgBottom = HexAlphaARGB(0xF8, 0x18241E)
			border = HexAlphaARGB(0xB5, 0x385446)
			sheen = HexAlphaARGB(0x45, 0xA8D5BA)
			borderW = 1.2
			if active {
				bgTop = HexAlphaARGB(0x48, 0x52B788)
				bgBottom = HexAlphaARGB(0x28, 0x2D6A4F)
				border = HexAlphaARGB(0xF0, 0x74C69D)
				sheen = HexAlphaARGB(0x80, 0xD8F3DC)
				borderW = 1.5
			} else if hover {
				bgTop = HexAlphaARGB(0xFA, 0x283B32)
				bgBottom = HexAlphaARGB(0xFC, 0x1E2D26)
				border = HexAlphaARGB(0xD8, 0x52B788)
			}
		} else {
			// Matcha Latte Light: Soft milk foam card with warm biscuit border & matcha accent
			bgTop = HexAlphaARGB(0xFF, 0xFFFDF9)
			bgBottom = HexAlphaARGB(0xF8, 0xF8F1E7)
			border = HexAlphaARGB(0xD0, 0xD5C3AE)
			sheen = HexAlphaARGB(0xFF, 0xFFFFFF)
			borderW = 1.3
			if active {
				bgTop = HexAlphaARGB(0x65, 0xA8D5BA)
				bgBottom = HexAlphaARGB(0x40, 0xA8D5BA)
				border = HexAlphaARGB(0xE8, 0x2D6A4F)
				borderW = 1.6
			} else if hover {
				bgTop = HexAlphaARGB(0xFF, 0xFFFFFF)
				bgBottom = HexAlphaARGB(0xFA, 0xEFE5D6)
				border = HexAlphaARGB(0xE0, 0xB89F86)
			}
		}

	case styleMaterialTeal:
		if isDark {
			// Material 3 Expressive Dark: Tonal Slate-Teal container with expressive aqua border
			bgTop = HexAlphaARGB(0xE6, 0x1B2D3A)
			bgBottom = HexAlphaARGB(0xF0, 0x13202A)
			border = HexAlphaARGB(0x78, 0x2DD4BF)
			sheen = HexAlphaARGB(0x60, 0x99F6E4)
			borderW = 1.2
			if active {
				bgTop = HexAlphaARGB(0x45, 0x2DD4BF)
				bgBottom = HexAlphaARGB(0x25, 0x14B8A6)
				border = HexAlphaARGB(0xF5, 0x2DD4BF)
				sheen = HexAlphaARGB(0x90, 0xFFFFFF)
				borderW = 1.5
			} else if hover {
				bgTop = HexAlphaARGB(0xF0, 0x233A4A)
				bgBottom = HexAlphaARGB(0xF5, 0x182936)
				border = HexAlphaARGB(0xB0, 0x2DD4BF)
			}
		} else {
			// Material 3 Expressive Light: Crisp white/aqua tonal surface with deep teal rim
			bgTop = HexAlphaARGB(0xFF, 0xFFFFFF)
			bgBottom = HexAlphaARGB(0xF6, 0xF0FDFA)
			border = HexAlphaARGB(0xC5, 0x5EEAD4)
			sheen = HexAlphaARGB(0xFF, 0xFFFFFF)
			borderW = 1.2
			if active {
				bgTop = HexAlphaARGB(0x75, 0xCCFBF1)
				bgBottom = HexAlphaARGB(0x50, 0x99F6E4)
				border = HexAlphaARGB(0xF0, 0x0D9488)
				borderW = 1.5
			} else if hover {
				bgTop = HexAlphaARGB(0xFF, 0xFFFFFF)
				bgBottom = HexAlphaARGB(0xFA, 0xCCFBF1)
				border = HexAlphaARGB(0xE0, 0x14B8A6)
			}
		}

	default:
		if isDark {
			bgTop = HexAlphaARGB(0x48, 0x242A38)
			bgBottom = HexAlphaARGB(0x2C, 0x141720)
			border = HexAlphaARGB(0x42, 0xFFFFFF)
			sheen = HexAlphaARGB(0x55, 0xFFFFFF)
			if active {
				bgTop = HexAlphaARGB(0x3C, argbAccent&0x00FFFFFF)
				bgBottom = HexAlphaARGB(0x20, argbAccent&0x00FFFFFF)
				border = HexAlphaARGB(0x95, argbAccent&0x00FFFFFF)
				sheen = HexAlphaARGB(0x75, 0xFFFFFF)
				borderW = 1.4
			} else if hover {
				bgTop = HexAlphaARGB(0x58, 0x2D3445)
				bgBottom = HexAlphaARGB(0x38, 0x1B1F2A)
				border = HexAlphaARGB(0x60, 0xFFFFFF)
				sheen = HexAlphaARGB(0x70, 0xFFFFFF)
			}
		} else {
			bgTop = HexAlphaARGB(0xFA, 0xFFFFFF)
			bgBottom = HexAlphaARGB(0xE8, 0xF1F5F9)
			border = HexAlphaARGB(0x88, 0xCBD5E1)
			sheen = HexAlphaARGB(0xE0, 0xFFFFFF)
			if active {
				bgTop = HexAlphaARGB(0xEE, 0xE6F7F2)
				bgBottom = HexAlphaARGB(0xDC, 0xD1EFE4)
				border = HexAlphaARGB(0xB0, argbAccent&0x00FFFFFF)
				sheen = HexAlphaARGB(0xF0, 0xFFFFFF)
				borderW = 1.4
			} else if hover {
				bgTop = HexAlphaARGB(0xFF, 0xFFFFFF)
				bgBottom = HexAlphaARGB(0xF0, 0xEBF2F8)
				border = HexAlphaARGB(0xA8, 0x94A3B8)
				sheen = HexAlphaARGB(0xF0, 0xFFFFFF)
			}
		}
	}

	g.FillRoundedRectGradient(x, y, w, h, radius, bgTop, bgBottom)
	g.DrawRoundedRect(x, y, w, h, radius, border, borderW)
	if radius > 4 && w > radius*2 {
		g.DrawLine(x+radius, y+1.0, x+w-radius, y+1.0, sheen, 1.0)
	}
}

// DrawGlassPill renders a tactile pill button with liquid glass specular highlight.
func (g *Gfx) DrawGlassPill(x, y, w, h, radius float32, isDark, active, hover bool) {
	if g == nil || g.native == 0 || w <= 0 || h <= 0 {
		return
	}

	if cfg.Style == styleBasic {
		// Cơ bản: flat opaque pill — solid accent only for the active nav tab.
		fill := argbKeycap
		border := argbBorder
		if active {
			fill = argbSwitchOn
			border = argbSwitchOn
		} else if hover {
			fill = HexAlphaARGB(0xFF, argbSwitchOff&0x00FFFFFF)
			border = argbSwitchOff
		}
		g.FillRoundedRect(x, y, w, h, radius, fill)
		g.DrawRoundedRect(x, y, w, h, radius, border, 1.0)
		return
	}

	if cfg.Style == styleMaterialTeal && radius >= 6 {
		radius = h / 2.0 // Material 3 Expressive full-capsule pill
	} else if cfg.Style == styleCyberNeon && radius > 6 {
		radius = 6
	}

	// Soft pill shadow
	g.FillRoundedRect(x, y+1.5, w, h, radius, HexAlphaARGB(0x22, 0x000000))

	accentRGB := argbAccent & 0x00FFFFFF

	if isDark {
		bgTop := HexAlphaARGB(0x88, argbKeycap&0x00FFFFFF)
		bgBottom := HexAlphaARGB(0x68, argbCard&0x00FFFFFF)
		border := HexAlphaARGB(0x78, argbBorder&0x00FFFFFF)
		if cfg.Style == styleJade {
			bgTop = HexAlphaARGB(0x45, 0x2A3140)
			bgBottom = HexAlphaARGB(0x28, 0x181C26)
			border = HexAlphaARGB(0x45, 0xFFFFFF)
		}
		sheen := HexAlphaARGB(0x55, 0xFFFFFF)

		if active {
			bgTop = HexAlphaARGB(0x4A, accentRGB)
			bgBottom = HexAlphaARGB(0x28, accentRGB)
			border = HexAlphaARGB(0xC0, accentRGB)
			sheen = HexAlphaARGB(0x80, 0xFFFFFF)
		} else if hover {
			bgTop = HexAlphaARGB(0xB0, argbKeycap&0x00FFFFFF)
			bgBottom = HexAlphaARGB(0x85, argbCard&0x00FFFFFF)
			border = HexAlphaARGB(0xA0, accentRGB)
			sheen = HexAlphaARGB(0x75, 0xFFFFFF)
		}
		g.FillRoundedRectGradient(x, y, w, h, radius, bgTop, bgBottom)
		g.DrawRoundedRect(x, y, w, h, radius, border, 1.1)
		if radius > 2 && w > radius*2 {
			g.DrawLine(x+radius*0.8, y+1.0, x+w-radius*0.8, y+1.0, sheen, 1.0)
		}
	} else {
		bgTop := HexAlphaARGB(0xFF, argbCard&0x00FFFFFF)
		bgBottom := HexAlphaARGB(0xF2, argbKeycap&0x00FFFFFF)
		border := HexAlphaARGB(0xB5, argbBorder&0x00FFFFFF)
		sheen := HexAlphaARGB(0xF0, 0xFFFFFF)

		if active {
			bgTop = HexAlphaARGB(0x32, accentRGB)
			bgBottom = HexAlphaARGB(0x1C, accentRGB)
			border = HexAlphaARGB(0xB0, accentRGB)
			sheen = HexAlphaARGB(0xFF, 0xFFFFFF)
		} else if hover {
			bgTop = HexAlphaARGB(0xFF, 0xFFFFFF)
			bgBottom = HexAlphaARGB(0xFA, argbKeycap&0x00FFFFFF)
			border = HexAlphaARGB(0xC0, accentRGB)
			sheen = HexAlphaARGB(0xFF, 0xFFFFFF)
		}
		g.FillRoundedRectGradient(x, y, w, h, radius, bgTop, bgBottom)
		g.DrawRoundedRect(x, y, w, h, radius, border, 1.1)
		if radius > 2 && w > radius*2 {
			g.DrawLine(x+radius*0.8, y+1.0, x+w-radius*0.8, y+1.0, sheen, 1.0)
		}
	}
}

// FillAtmosphericGlassBg renders a rich, style-aware window backdrop with ambient color blooms.
func (g *Gfx) FillAtmosphericGlassBg(w, h float32, isDark bool) {
	if g == nil || g.native == 0 || w <= 0 || h <= 0 {
		return
	}

	var colTop, colBottom, glow1, glow2 uint32
	switch cfg.Style {
	case styleBasic:
		// Cơ bản: flat solid background, no gradient, no ambient blooms.
		colTop = HexAlphaARGB(0xFF, argbBg&0x00FFFFFF)
		colBottom = colTop
		glow1, glow2 = 0, 0
	case styleIosPure:
		if isDark {
			colTop = HexARGB(0x000000)
			colBottom = HexARGB(0x080A10)
			glow1 = HexAlphaARGB(0x14, 0x0A84FF)
			glow2 = HexAlphaARGB(0x0E, 0x5E5CE6)
		} else {
			colTop = HexARGB(0xEAEEF4)
			colBottom = HexARGB(0xD8E0EA)
			glow1 = HexAlphaARGB(0x14, 0x007AFF)
			glow2 = HexAlphaARGB(0x10, 0x5856D6)
		}
	case styleCyberNeon:
		if isDark {
			colTop = HexARGB(0x04060A)
			colBottom = HexARGB(0x091120)
			glow1 = HexAlphaARGB(0x1A, 0x00F2FE)
			glow2 = HexAlphaARGB(0x14, 0x10B981)
		} else {
			colTop = HexARGB(0xE4F4FE)
			colBottom = HexARGB(0xC7EAFB)
			glow1 = HexAlphaARGB(0x20, 0x38BDF8)
			glow2 = HexAlphaARGB(0x18, 0x10B981)
		}
	case styleMatchaLatte:
		if isDark {
			colTop = HexARGB(0x141D18)
			colBottom = HexARGB(0x0D1310)
			glow1 = HexAlphaARGB(0x16, 0x52B788)
			glow2 = HexAlphaARGB(0x12, 0xF4A261)
		} else {
			colTop = HexARGB(0xF5EDE2)
			colBottom = HexARGB(0xE6D8C6)
			glow1 = HexAlphaARGB(0x28, 0xA8D5BA)
			glow2 = HexAlphaARGB(0x22, 0xF7C59F)
		}
	case styleMaterialTeal:
		if isDark {
			colTop = HexARGB(0x111E26)
			colBottom = HexARGB(0x091116)
			glow1 = HexAlphaARGB(0x18, 0x2DD4BF)
			glow2 = HexAlphaARGB(0x12, 0x38BDF8)
		} else {
			colTop = HexARGB(0xE8F8F5)
			colBottom = HexARGB(0xD0F2EC)
			glow1 = HexAlphaARGB(0x26, 0x5EEAD4)
			glow2 = HexAlphaARGB(0x1A, 0x2DD4BF)
		}
	default:
		if isDark {
			colTop = HexARGB(0x12151B)
			colBottom = HexARGB(0x0B0D12)
			glow1 = HexAlphaARGB(0x12, 0x3ED3B8)
			glow2 = HexAlphaARGB(0x0C, 0x197F64)
		} else {
			colTop = HexARGB(0xF8FAFC)
			colBottom = HexARGB(0xECF3F8)
			glow1 = HexAlphaARGB(0x14, 0x3ED3B8)
			glow2 = HexAlphaARGB(0x10, 0x197F64)
		}
	}
	g.FillRoundedRectGradient(0, 0, w, h, 0, colTop, colBottom)

	// Soft atmospheric ambient blooms in top-right and bottom-left (like the HTML demo)
	if glow1 != 0 {
		r1 := w * 0.34
		g.FillCircle(w*0.86, h*0.12, r1, glow1)
		g.FillCircle(w*0.86, h*0.12, r1*0.65, glow1)
	}
	if glow2 != 0 {
		r2 := w * 0.28
		g.FillCircle(w*0.18, h*0.88, r2, glow2)
	}
}


