package main

import (
	"unsafe"
)

// ui_components.go provides smooth, zero-aliasing UI controls powered by GDI+
// with modern Minimalist and Liquid Glass styling.

// drawSmoothSwitch renders a modern iOS/Fluent-style pill toggle switch with Liquid Glass GDI+.
func drawSmoothSwitch(dc uintptr, rc rect, id int, on bool) {
	fillParentBg(dc, rc, ctlHnd[id])

	gfx := NewGfx(dc)
	if gfx == nil {
		return
	}
	defer gfx.Destroy()

	pillH := float32(dp(22))
	pillW := float32(dp(40))
	py := float32(rc.top) + (float32(rc.bottom-rc.top)-pillH)/2.0
	px := float32(rc.left + dp(4))
	radius := pillH / 2.0

	// Ambient shadow for switch track
	gfx.FillRoundedRect(px, py+1.5, pillW, pillH, radius, HexAlphaARGB(0x25, 0x000000))

	// Track styling with Liquid Glass gradient
	if on {
		// Active track: vibrant style accent with specular top
		swRGB := argbSwitchOn & 0x00FFFFFF
		topCol := HexAlphaARGB(0xFA, swRGB)
		bottomCol := HexAlphaARGB(0xDC, swRGB)
		gfx.FillRoundedRectGradient(px, py, pillW, pillH, radius, topCol, bottomCol)
		borderCol := HexAlphaARGB(0x90, 0xFFFFFF)
		if cfg.Style == styleCyberNeon {
			borderCol = argbAccent
		}
		gfx.DrawRoundedRect(px, py, pillW, pillH, radius, borderCol, 1.1)
		// Specular top highlight
		gfx.DrawLine(px+radius, py+1.0, px+pillW-radius, py+1.0, HexAlphaARGB(0x75, 0xFFFFFF), 1.0)
	} else {
		// Inactive track: use active palette's switchOff and border tokens (warm biscuit on Matcha, cyber navy on Cyber, etc.)
		offRGB := argbSwitchOff & 0x00FFFFFF
		bdRGB := argbBorder & 0x00FFFFFF
		topAlpha := byte(0xE8)
		botAlpha := byte(0xD0)
		if isDark() {
			topAlpha = 0xB0
			botAlpha = 0x90
		}
		topCol := HexAlphaARGB(topAlpha, offRGB)
		bottomCol := HexAlphaARGB(botAlpha, offRGB)
		gfx.FillRoundedRectGradient(px, py, pillW, pillH, radius, topCol, bottomCol)
		gfx.DrawRoundedRect(px, py, pillW, pillH, radius, HexAlphaARGB(0xC0, bdRGB), 1.1)
	}

	// Thumb (circular knob)
	knobD := pillH - float32(dp(4))
	knobR := knobD / 2.0
	kx := px + float32(dp(2))
	if on {
		kx = px + pillW - knobD - float32(dp(2))
	}
	ky := py + float32(dp(2))

	// Soft tactile drop shadow under knob
	gfx.FillCircle(kx+knobR, ky+knobR+1.0, knobR, HexAlphaARGB(0x40, 0x000000))
	// Crisp thumb with subtle border
	thumbCol := uint32(0xFFFFFFFF)
	if cfg.Style == styleCyberNeon && isDark() && !on {
		thumbCol = 0xFF7DD3FC
	}
	gfx.FillCircle(kx+knobR, ky+knobR, knobR, thumbCol)
	gfx.DrawCircle(kx+knobR, ky+knobR, knobR, HexAlphaARGB(0x35, 0x000000), 0.6)

	// Draw label text
	pSetBkMode.Call(dc, TRANSPARENT)
	pSetTextColor.Call(dc, colText)
	oldF, _, _ := pSelectObject.Call(dc, fontNormal)
	trc := rect{rc.left + dp(54), rc.top, rc.right, rc.bottom}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr(swText[id]))),
		^uintptr(0), uintptr(unsafe.Pointer(&trc)),
		DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)
	pSelectObject.Call(dc, oldF)
}

// drawSmoothCheckbox renders an anti-aliased rounded checkbox with liquid glass styling.
func drawSmoothCheckbox(dc uintptr, rc rect, id int, checked bool) {
	fillParentBg(dc, rc, ctlHnd[id])

	gfx := NewGfx(dc)
	if gfx == nil {
		return
	}
	defer gfx.Destroy()

	boxS := float32(dp(18))
	by := float32(rc.top) + (float32(rc.bottom-rc.top)-boxS)/2.0
	bx := float32(rc.left + dp(4))
	r := float32(dp(4))

	// Box shadow
	gfx.FillRoundedRect(bx, by+1.0, boxS, boxS, r, HexAlphaARGB(0x15, 0x000000))

	if checked {
		// Active: glowing jade glass
		topCol := HexAlphaARGB(0xF5, argbAccent&0x00FFFFFF)
		botCol := HexAlphaARGB(0xD0, argbAccent&0x00FFFFFF)
		gfx.FillRoundedRectGradient(bx, by, boxS, boxS, r, topCol, botCol)
		gfx.DrawRoundedRect(bx, by, boxS, boxS, r, HexAlphaARGB(0x80, 0xFFFFFF), 1.0)
		gfx.DrawLine(bx+r, by+1.0, bx+boxS-r, by+1.0, HexAlphaARGB(0x90, 0xFFFFFF), 1.0)

		// Crisp anti-aliased checkmark
		p1x, p1y := bx+float32(dp(4)), by+float32(dp(9))
		p2x, p2y := bx+float32(dp(7)), by+float32(dp(13))
		p3x, p3y := bx+float32(dp(14)), by+float32(dp(5))
		thk := float32(max(int32(2), dp(2)))
		gfx.DrawLine(p1x, p1y, p2x, p2y, 0xFFFFFFFF, thk)
		gfx.DrawLine(p2x, p2y, p3x, p3y, 0xFFFFFFFF, thk)
	} else {
		// Inactive: clean card background with subtle border
		boxBg := argbCard
		borderCol := argbBorder
		gfx.FillRoundedRect(bx, by, boxS, boxS, r, boxBg)
		gfx.DrawRoundedRect(bx, by, boxS, boxS, r, borderCol, 1.2)
	}

	pSetBkMode.Call(dc, TRANSPARENT)
	pSetTextColor.Call(dc, colText)
	oldF, _, _ := pSelectObject.Call(dc, fontNormal)
	trc := rect{int32(bx + boxS + float32(dp(10))), rc.top, rc.right, rc.bottom}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr(btnText[id]))),
		^uintptr(0), uintptr(unsafe.Pointer(&trc)),
		DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)
	pSelectObject.Call(dc, oldF)
}

// solidAccentSelected reports whether the active style draws a solid accent
// fill under selected cards. Only Cơ bản does — every other style uses a
// translucent light tint where normal colText/colGold inks are correct.
func solidAccentSelected() bool {
	return cfg.Style == styleBasic
}

// accentOnCol picks the ink that reads on top of argbAccent: near-black on
// light accents (mint #3ED3B8, user picks), white on dark accents (#197F64).
// cfg.AccentText overrides the auto choice (RGB stored, return as COLORREF).
func accentOnCol() uintptr {
	if cfg.AccentText >= 0 {
		return rgb2ref(cfg.AccentText)
	}
	c := argbAccent & 0x00FFFFFF
	lum := 0.299*float64(c>>16&0xFF) + 0.587*float64(c>>8&0xFF) + 0.114*float64(c&0xFF)
	if lum > 140 {
		return 0x15110F // #0F1115
	}
	return 0xFFFFFF
}

// accentOnArgb is accentOnCol for GDI+ ARGB (nav icons, badge glyphs).
func accentOnArgb() uint32 {
	if cfg.AccentText >= 0 {
		return 0xFF000000 | uint32(cfg.AccentText&0x00FFFFFF)
	}
	c := argbAccent & 0x00FFFFFF
	lum := 0.299*float64(c>>16&0xFF) + 0.587*float64(c>>8&0xFF) + 0.114*float64(c&0xFF)
	if lum > 140 {
		return 0xFF0F1115
	}
	return 0xFFFFFFFF
}

// drawSmoothRadioCard renders an anti-aliased radio selection card with Liquid Glass styling.
func drawSmoothRadioCard(dc uintptr, rc rect, id int, selected bool) {
	fillParentBg(dc, rc, ctlHnd[id])

	gfx := NewGfx(dc)
	if gfx == nil {
		return
	}
	defer gfx.Destroy()

	x, y, w, h := float32(rc.left), float32(rc.top), float32(rc.right-rc.left), float32(rc.bottom-rc.top)
	r := float32(dp(10))

	// Draw full liquid glass card with active glow if selected
	gfx.DrawGlassCardExt(x, y, w, h, r, isDark(), false, selected)

	// Radio bullet
	cx := x + float32(dp(18))
	cy := y + h/2.0
	outerR := float32(dp(7))
	solid := selected && solidAccentSelected()
	bulletCol := argbAccent
	if solid {
		bulletCol = accentOnArgb()
	}

	if selected {
		// Radiant jade ring + glowing center
		gfx.DrawCircle(cx, cy, outerR, bulletCol, 1.6)
		gfx.FillCircle(cx, cy, float32(dp(4)), bulletCol)
		if !solid {
			gfx.FillCircle(cx, cy, float32(dp(2)), 0xFFFFFFFF)
		}
	} else {
		gfx.DrawCircle(cx, cy, outerR, argbMuted, 1.2)
	}

	titleCol := colText
	subCol := colMuted
	if solid {
		titleCol = accentOnCol()
		subCol = accentOnCol()
	}
	pSetBkMode.Call(dc, TRANSPARENT)
	pSetTextColor.Call(dc, titleCol)
	oldF, _, _ := pSelectObject.Call(dc, fontBold)
	sub := radioSubtitle[id]
	if sub == "" {
		trc := rect{int32(cx + outerR + float32(dp(10))), rc.top, rc.right - dp(4), rc.bottom}
		pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr(btnText[id]))),
			^uintptr(0), uintptr(unsafe.Pointer(&trc)),
			DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)
	} else {
		trc := rect{int32(cx + outerR + float32(dp(12))), rc.top + dp(6), rc.right - dp(4), rc.top + dp(24)}
		pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr(btnText[id]))),
			^uintptr(0), uintptr(unsafe.Pointer(&trc)),
			DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

		pSetTextColor.Call(dc, subCol)
		pSelectObject.Call(dc, fontHint)
		drc := rect{int32(cx + outerR + float32(dp(12))), rc.top + dp(24), rc.right - dp(4), rc.bottom - dp(4)}
		pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr(sub))),
			^uintptr(0), uintptr(unsafe.Pointer(&drc)),
			DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)
	}

	pSelectObject.Call(dc, oldF)
}

// drawSmoothButton renders a button with tactile liquid glass sheen and anti-aliasing.
func drawSmoothButton(dc uintptr, rc rect, id int, pressed bool) {
	fillParentBg(dc, rc, ctlHnd[id])

	gfx := NewGfx(dc)
	if gfx == nil {
		return
	}
	defer gfx.Destroy()

	x, y, w, h := float32(rc.left), float32(rc.top), float32(rc.right-rc.left), float32(rc.bottom-rc.top)
	r := float32(dp(8))

	gfx.DrawGlassPill(x, y, w, h, r, isDark(), false, pressed)

	txt := btnText[id]
	pSetBkMode.Call(dc, TRANSPARENT)
	pSetTextColor.Call(dc, colText)
	oldF, _, _ := pSelectObject.Call(dc, fontNormal)
	trc := rc
	if pressed {
		trc.top += dp(1)
	}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr(txt))),
		^uintptr(0), uintptr(unsafe.Pointer(&trc)),
		DT_CENTER|DT_VCENTER|DT_SINGLELINE)
	pSelectObject.Call(dc, oldF)
}

// traySwatchColor resolves the effective RGB for a tray swatch button:
// cfg value when set, else the fixed icon default.
func traySwatchColor(id int) int {
	switch id {
	case cTrayV:
		if cfg.TrayV >= 0 {
			return cfg.TrayV
		}
		return trayDefV
	case cTrayVBg:
		if cfg.TrayVBg >= 0 {
			return cfg.TrayVBg
		}
		return trayDefVBg
	case cTrayE:
		if cfg.TrayE >= 0 {
			return cfg.TrayE
		}
		return trayDefE
	case cTrayEBg:
		if cfg.TrayEBg >= 0 {
			return cfg.TrayEBg
		}
		return trayDefEBg
	case cAccent:
		if cfg.Accent >= 0 {
			return cfg.Accent
		}
		return int(argbAccent & 0x00FFFFFF)
	case cAccentText:
		if cfg.AccentText >= 0 {
			return cfg.AccentText
		}
		// show the auto-picked ink color (not the accent fill)
		c := argbAccent & 0x00FFFFFF
		lum := 0.299*float64(c>>16&0xFF) + 0.587*float64(c>>8&0xFF) + 0.114*float64(c&0xFF)
		if lum > 140 {
			return 0x0F1115
		}
		return 0xFFFFFF
	}
	return 0
}

// drawTraySwatch renders a color-picker button: color square + label.
func drawTraySwatch(dc uintptr, rc rect, id int, pressed bool) {
	fillParentBg(dc, rc, ctlHnd[id])

	gfx := NewGfx(dc)
	if gfx == nil {
		return
	}
	defer gfx.Destroy()

	x, y, w, h := float32(rc.left), float32(rc.top), float32(rc.right-rc.left), float32(rc.bottom-rc.top)
	r := float32(dp(8))
	gfx.DrawGlassPill(x, y, w, h, r, isDark(), false, pressed)

	// Color square
	rgb := traySwatchColor(id)
	argb := uint32(0xFF000000) | uint32(rgb)
	sq := float32(dp(16))
	sx := x + float32(dp(9))
	sy := y + (h-sq)/2.0
	gfx.FillRoundedRect(sx, sy, sq, sq, float32(dp(4)), argb)
	gfx.DrawRoundedRect(sx, sy, sq, sq, float32(dp(4)), HexAlphaARGB(0xA0, argbBorder&0x00FFFFFF), 1.0)
	if cfg.TrayV < 0 && id == cTrayV || cfg.TrayVBg < 0 && id == cTrayVBg ||
		cfg.TrayE < 0 && id == cTrayE || cfg.TrayEBg < 0 && id == cTrayEBg ||
		cfg.Accent < 0 && id == cAccent || cfg.AccentText < 0 && id == cAccentText {
		// Unset — small "auto" dot so the user sees it follows the theme
		gfx.FillCircle(sx+sq-float32(dp(2)), sy+float32(dp(2)), float32(dp(3)), argbMuted)
	}

	pSetBkMode.Call(dc, TRANSPARENT)
	pSetTextColor.Call(dc, colText)
	oldF, _, _ := pSelectObject.Call(dc, fontNormal)
	trc := rect{rc.left + dp(30), rc.top, rc.right - dp(4), rc.bottom}
	if pressed {
		trc.top += dp(1)
	}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr(btnText[id]))),
		^uintptr(0), uintptr(unsafe.Pointer(&trc)),
		DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)
	pSelectObject.Call(dc, oldF)
}

// drawSmoothModifierKeycap renders a modern mechanical keycap badge with liquid glass depth.
func drawSmoothModifierKeycap(dc uintptr, rc rect, id int, selected, pressed bool) {
	fillParentBg(dc, rc, ctlHnd[id])

	gfx := NewGfx(dc)
	if gfx == nil {
		return
	}
	defer gfx.Destroy()

	x, y, w, h := float32(rc.left), float32(rc.top), float32(rc.right-rc.left), float32(rc.bottom-rc.top)
	r := float32(dp(8))

	gfx.DrawGlassCardExt(x, y, w, h, r, isDark(), pressed, selected)

	pSetBkMode.Call(dc, TRANSPARENT)
	txtCol := colText
	if selected {
		txtCol = colGold
		if solidAccentSelected() {
			txtCol = accentOnCol()
		}
	}
	pSetTextColor.Call(dc, txtCol)
	oldFont, _, _ := pSelectObject.Call(dc, fontBold)
	textRc := rc
	if pressed {
		textRc.top += dp(1)
	}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr(swText[id]))),
		^uintptr(0), uintptr(unsafe.Pointer(&textRc)), DT_CENTER|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)
	pSelectObject.Call(dc, oldFont)
}

// drawSmoothVectorIcon draws crisp, anti-aliased navigation icons using GDI+ paths.
func drawSmoothVectorIcon(gfx *Gfx, cx, cy, size float32, kind int, color uint32) {
	if gfx == nil {
		return
	}
	thk := float32(max(int32(1), dp(2)))
	hs := size / 2.0
	x := cx - hs
	y := cy - hs

	switch kind {
	case 0: // Overview / Dashboard: 4-square grid with rounded corners
		gap := float32(dp(2))
		cell := (size - gap) / 2.0
		cr := float32(dp(2))
		gfx.DrawRoundedRect(x, y, cell, cell, cr, color, thk)
		gfx.DrawRoundedRect(x+cell+gap, y, cell, cell, cr, color, thk)
		gfx.DrawRoundedRect(x, y+cell+gap, cell, cell, cr, color, thk)
		gfx.DrawRoundedRect(x+cell+gap, y+cell+gap, cell, cell, cr, color, thk)

	case 1: // Keyboard / Typing: sleek keyboard outline + spacebar
		kw := size
		kh := size * 0.75
		ky := cy - kh/2.0
		gfx.DrawRoundedRect(x, ky, kw, kh, float32(dp(4)), color, thk)
		// Spacebar line
		spW := size * 0.5
		gfx.DrawLine(cx-spW/2, ky+kh*0.7, cx+spW/2, ky+kh*0.7, color, thk)
		// Dot keys
		gfx.FillCircle(cx-kw*0.25, ky+kh*0.35, 1.2, color)
		gfx.FillCircle(cx, ky+kh*0.35, 1.2, color)
		gfx.FillCircle(cx+kw*0.25, ky+kh*0.35, 1.2, color)

	case 2: // Hotkeys / Command: 4 interconnected loops (⌘ command symbol)
		r := size * 0.22
		gfx.DrawCircle(cx-r, cy-r, r, color, thk)
		gfx.DrawCircle(cx+r, cy-r, r, color, thk)
		gfx.DrawCircle(cx-r, cy+r, r, color, thk)
		gfx.DrawCircle(cx+r, cy+r, r, color, thk)
		gfx.DrawLine(cx-r, cy-r, cx-r, cy+r, color, thk)
		gfx.DrawLine(cx+r, cy-r, cx+r, cy+r, color, thk)
		gfx.DrawLine(cx-r, cy-r, cx+r, cy-r, color, thk)
		gfx.DrawLine(cx-r, cy+r, cx+r, cy+r, color, thk)

	case 3: // Appearance / Sliders: modern tuning sliders
		gfx.DrawLine(x+size*0.2, y+size*0.1, x+size*0.2, y+size*0.9, color, thk)
		gfx.DrawLine(x+size*0.5, y+size*0.1, x+size*0.5, y+size*0.9, color, thk)
		gfx.DrawLine(x+size*0.8, y+size*0.1, x+size*0.8, y+size*0.9, color, thk)
		// Knobs
		gfx.FillCircle(x+size*0.2, y+size*0.35, float32(dp(3)), color)
		gfx.FillCircle(x+size*0.5, y+size*0.65, float32(dp(3)), color)
		gfx.FillCircle(x+size*0.8, y+size*0.4, float32(dp(3)), color)

	case 4: // System / Cog: gear ring with 8 radial teeth and center hub
		rIn := size * 0.17
		rRing := size * 0.34
		rOut := size * 0.48
		gfx.DrawCircle(cx, cy, rRing, color, thk)
		gfx.DrawCircle(cx, cy, rIn, color, thk*0.85)
		// 4 cardinal teeth
		gfx.DrawLine(cx, cy-rOut, cx, cy-rRing, color, thk)
		gfx.DrawLine(cx, cy+rRing, cx, cy+rOut, color, thk)
		gfx.DrawLine(cx-rOut, cy, cx-rRing, cy, color, thk)
		gfx.DrawLine(cx+rRing, cy, cx+rOut, cy, color, thk)
		// 4 diagonal teeth (cos 45° ≈ 0.7071)
		d1 := rRing * 0.7071
		d2 := rOut * 0.7071
		gfx.DrawLine(cx-d2, cy-d2, cx-d1, cy-d1, color, thk)
		gfx.DrawLine(cx+d1, cy+d1, cx+d2, cy+d2, color, thk)
		gfx.DrawLine(cx-d2, cy+d2, cx-d1, cy+d1, color, thk)
		gfx.DrawLine(cx+d1, cy-d1, cx+d2, cy-d2, color, thk)

	case 5: // Information: sleek circle with 'i'
		gfx.DrawCircle(cx, cy, size*0.46, color, thk)
		gfx.FillCircle(cx, cy-size*0.22, float32(dp(2)), color)
		gfx.DrawLine(cx, cy-size*0.06, cx, cy+size*0.22, color, thk)
	}
}

// drawMinimalBrandMark draws a rich 3D typographic keycap logo [ V ] for the header.
func drawMinimalBrandMark(gfx *Gfx, x, y, size float32) {
	if gfx == nil {
		return
	}
	r := float32(dp(8))
	// Soft drop shadow
	gfx.FillRoundedRect(x, y+1.5, size, size, r, HexAlphaARGB(0x35, 0x000000))

	// Rich solid gradient badge so the [V] mark pops crisply in both Light and Dark modes
	accRGB := argbAccent & 0x00FFFFFF
	swRGB := argbSwitchOn & 0x00FFFFFF
	bgTop := HexAlphaARGB(0xFF, accRGB)
	bgBottom := HexAlphaARGB(0xE8, swRGB)
	glyphCol := uint32(0xFFFFFFFF)

	switch cfg.Style {
	case styleCyberNeon:
		if isDark() {
			bgTop = 0xFF0B111E
			bgBottom = 0xFF04060A
			glyphCol = 0xFF00F2FE
		} else {
			bgTop = 0xFF0284C7
			bgBottom = 0xFF0369A1
		}
	case styleMatchaLatte:
		if isDark() {
			bgTop = 0xFF40916C
			bgBottom = 0xFF2D6A4F
			glyphCol = 0xFFF4ECE1
		} else {
			bgTop = 0xFF40916C
			bgBottom = 0xFF2D6A4F
			glyphCol = 0xFFFFFDF9
		}
	case styleMaterialTeal:
		if isDark() {
			bgTop = 0xFF2DD4BF
			bgBottom = 0xFF14B8A6
			glyphCol = 0xFF0E171D
		} else {
			bgTop = 0xFF0D9488
			bgBottom = 0xFF0F766E
			glyphCol = 0xFFFFFFFF
		}
	default:
		if isDark() && cfg.Style == styleJade {
			bgTop = 0xFF3ED3B8
			bgBottom = 0xFF197F64
			glyphCol = 0xFF0F1115
		}
	}

	gfx.FillRoundedRectGradient(x, y, size, size, r, bgTop, bgBottom)
	borderCol := HexAlphaARGB(0x90, 0xFFFFFF)
	if cfg.Style == styleCyberNeon && isDark() {
		borderCol = 0xFF00F2FE
	}
	gfx.DrawRoundedRect(x, y, size, size, r, borderCol, 1.3)
	// Top specular highlight
	gfx.DrawLine(x+r, y+1.2, x+size-r, y+1.2, HexAlphaARGB(0x80, 0xFFFFFF), 1.0)

	// Minimal geometric "V" symbol
	thk := float32(max(int32(2), dp(2))) + 0.4
	cx := x + size/2.0
	cy := y + size/2.0
	armW := size * 0.24
	armH := size * 0.22
	gfx.DrawLine(cx-armW, cy-armH, cx, cy+armH, glyphCol, thk)
	gfx.DrawLine(cx, cy+armH, cx+armW, cy-armH, glyphCol, thk)
}

// drawSmoothThemeButton renders the Light / Dark / System mode segmented button.
func drawSmoothThemeButton(dc uintptr, rc rect, id int, pressed bool) {
	fillParentBg(dc, rc, ctlHnd[id])

	gfx := NewGfx(dc)
	if gfx == nil {
		return
	}
	defer gfx.Destroy()

	themeMode := themeAuto
	switch id {
	case cThemeLight:
		themeMode = themeLight
	case cThemeDark:
		themeMode = themeDark
	}

	selected := (cfg.Theme == themeMode)

	x, y, w, h := float32(rc.left), float32(rc.top), float32(rc.right-rc.left), float32(rc.bottom-rc.top)
	r := float32(dp(8))

	gfx.DrawGlassCardExt(x, y, w, h, r, isDark(), pressed, selected)

	// Draw Icon
	solid := selected && solidAccentSelected()
	iconCol := argbMuted
	if selected {
		iconCol = argbAccent
	}
	if solid {
		iconCol = accentOnArgb()
	}
	iconX := x + float32(dp(18))
	iconY := y + h/2.0
	iconSz := float32(dp(16))
	thk := float32(max(int32(1), dp(2)))

	switch themeMode {
	case themeAuto: // System / Monitor icon
		sw := iconSz * 0.9
		sh := iconSz * 0.65
		sx := iconX - sw/2.0
		sy := iconY - sh/2.0 - float32(dp(2))
		gfx.DrawRoundedRect(sx, sy, sw, sh, float32(dp(2)), iconCol, thk)
		gfx.DrawLine(iconX, sy+sh, iconX, sy+sh+float32(dp(3)), iconCol, thk)
		gfx.DrawLine(iconX-float32(dp(4)), sy+sh+float32(dp(3)), iconX+float32(dp(4)), sy+sh+float32(dp(3)), iconCol, thk)

	case themeLight: // Sun icon (circle + rays)
		sr := iconSz * 0.28
		gfx.DrawCircle(iconX, iconY, sr, iconCol, thk)
		rayLen := float32(dp(2))
		rayDist := sr + float32(dp(2))
		gfx.DrawLine(iconX, iconY-rayDist-rayLen, iconX, iconY-rayDist, iconCol, thk)
		gfx.DrawLine(iconX, iconY+rayDist, iconX, iconY+rayDist+rayLen, iconCol, thk)
		gfx.DrawLine(iconX-rayDist-rayLen, iconY, iconX-rayDist, iconY, iconCol, thk)
		gfx.DrawLine(iconX+rayDist, iconY, iconX+rayDist+rayLen, iconY, iconCol, thk)
		d := rayDist * 0.707
		dl := (rayDist + rayLen) * 0.707
		gfx.DrawLine(iconX-dl, iconY-dl, iconX-d, iconY-d, iconCol, thk)
		gfx.DrawLine(iconX+d, iconY+d, iconX+dl, iconY+dl, iconCol, thk)
		gfx.DrawLine(iconX-dl, iconY+dl, iconX-d, iconY+d, iconCol, thk)
		gfx.DrawLine(iconX+d, iconY-d, iconX+dl, iconY-dl, iconCol, thk)

	case themeDark: // Moon crescent icon
		mr := iconSz * 0.40
		gfx.DrawCircle(iconX, iconY, mr, iconCol, thk)
		moonBg := argbCard
		if solid {
			moonBg = argbSwitchOn // card fill is the accent — cut crescent from it
		}
		gfx.FillCircle(iconX+float32(dp(3)), iconY-float32(dp(2)), mr*0.75, moonBg)
	}

	pSetBkMode.Call(dc, TRANSPARENT)
	titleCol := colText
	if selected {
		titleCol = colGold
		if solid {
			titleCol = accentOnCol()
		}
	}
	pSetTextColor.Call(dc, titleCol)
	oldF, _, _ := pSelectObject.Call(dc, fontBold)

	title := tr("Hệ thống (Tự động)", "System (Auto)")
	switch themeMode {
	case themeLight:
		title = tr("Chế độ Sáng (Light)", "Light Mode")
	case themeDark:
		title = tr("Chế độ Tối (Dark)", "Dark Mode")
	}

	textLeft := int32(iconX + iconSz/2.0 + float32(dp(8)))
	trc := rect{textLeft, rc.top, rc.right - dp(6), rc.bottom}
	if pressed {
		trc.top += dp(1)
	}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr(title))),
		^uintptr(0), uintptr(unsafe.Pointer(&trc)),
		DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

	pSelectObject.Call(dc, oldF)
}

// drawSmoothStyleCard renders a visual theme style card with a live miniature keyboard/UI preview swatch.
func drawSmoothStyleCard(dc uintptr, rc rect, id int, pressed bool) {
	fillParentBg(dc, rc, ctlHnd[id])

	gfx := NewGfx(dc)
	if gfx == nil {
		return
	}
	defer gfx.Destroy()

	styleID := styleJade
	switch id {
	case cThemeBasic:
		styleID = styleBasic
	case cThemeIosPure:
		styleID = styleIosPure
	case cThemeCyberNeon:
		styleID = styleCyberNeon
	case cThemeMatchaLatte:
		styleID = styleMatchaLatte
	case cThemeMaterialTeal:
		styleID = styleMaterialTeal
	}

	selected := (cfg.Style == styleID)
	dark := isDark()

	x, y, w, h := float32(rc.left), float32(rc.top), float32(rc.right-rc.left), float32(rc.bottom-rc.top)
	r := float32(dp(10))

	gfx.DrawGlassCardExt(x, y, w, h, r, dark, pressed, selected)

	// Mini preview colors for this style in the current Light/Dark mode
	var pvBg, pvKey, pvFunc, pvBorder, pvAccent uint32
	title := tr("Ngọc Bích", "Jade Glass")
	sub := tr("Mặc định", "Default")

	switch styleID {
	case styleBasic:
		title = tr("Cơ bản", "Basic")
		sub = tr("Đơn giản", "Flat & simple")
		if dark {
			pvBg, pvKey, pvFunc, pvBorder, pvAccent = 0xFF202020, 0xFF333333, 0xFF2B2B2B, 0xFF3ED3B8, 0xFF3ED3B8
		} else {
			pvBg, pvKey, pvFunc, pvBorder, pvAccent = 0xFFF3F3F3, 0xFFFFFFFF, 0xFFE0E0E0, 0xFF197F64, 0xFF197F64
		}
	case styleJade:
		title = tr("Ngọc Bích", "Jade Glass")
		sub = "Liquid Glass"
		if dark {
			pvBg, pvKey, pvFunc, pvBorder, pvAccent = 0xFF0F1115, 0xFF232834, 0xFF1A1E27, 0xFF3ED3B8, 0xFF3ED3B8
		} else {
			pvBg, pvKey, pvFunc, pvBorder, pvAccent = 0xFFE8F0F6, 0xFFFFFFFF, 0xFFDCE6F0, 0xFF197F64, 0xFF197F64
		}
	case styleIosPure:
		title = "iOS 18 Pro"
		sub = "Apple Blue"
		if dark {
			pvBg, pvKey, pvFunc, pvBorder, pvAccent = 0xFF000000, 0xFF3A3A3C, 0xFF2C2C2E, 0xFF0A84FF, 0xFF0A84FF
		} else {
			pvBg, pvKey, pvFunc, pvBorder, pvAccent = 0xFFD1D5DB, 0xFFFFFFFF, 0xFFAEB5BF, 0xFF007AFF, 0xFF007AFF
		}
	case styleCyberNeon:
		title = "Cyber Neon"
		sub = "Cyan & Mint"
		if dark {
			pvBg, pvKey, pvFunc, pvBorder, pvAccent = 0xFF04060A, 0xFF0B111E, 0xFF16223B, 0xFF00F2FE, 0xFF10B981
		} else {
			pvBg, pvKey, pvFunc, pvBorder, pvAccent = 0xFFE0F2FE, 0xFFFFFFFF, 0xFFBAE6FD, 0xFF0284C7, 0xFF10B981
		}
	case styleMatchaLatte:
		title = "Matcha"
		sub = tr("Kem Trà Xanh", "Matcha Cream")
		if dark {
			pvBg, pvKey, pvFunc, pvBorder, pvAccent = 0xFF121A16, 0xFF24352C, 0xFF2D4439, 0xFF74C69D, 0xFFF4A261
		} else {
			pvBg, pvKey, pvFunc, pvBorder, pvAccent = 0xFFEDE4D8, 0xFFFFFDF9, 0xFFA8D5BA, 0xFF2D6A4F, 0xFFE07A5F
		}
	case styleMaterialTeal:
		title = "Material 3"
		sub = "Pixel Teal"
		if dark {
			pvBg, pvKey, pvFunc, pvBorder, pvAccent = 0xFF0E171D, 0xFF1E313E, 0xFF284152, 0xFF2DD4BF, 0xFF2DD4BF
		} else {
			pvBg, pvKey, pvFunc, pvBorder, pvAccent = 0xFFE0F7F4, 0xFFFFFFFF, 0xFFCCFBF1, 0xFF0D9488, 0xFF0D9488
		}
	}

	// Draw Miniature Keyboard/UI Swatch at top of card
	padX := float32(dp(8))
	swX := x + padX
	swY := y + float32(dp(6))
	swW := w - padX*2
	swH := float32(dp(24))
	swR := float32(dp(5))

	gfx.FillRoundedRect(swX, swY, swW, swH, swR, pvBg)
	bdThk := float32(1.0)
	if selected {
		bdThk = 1.5
	}
	gfx.DrawRoundedRect(swX, swY, swW, swH, swR, pvBorder, bdThk)

	// Top row of 4 mini keycaps inside swatch
	kGap := float32(dp(2))
	kPad := float32(dp(3))
	kw := (swW - kPad*2 - kGap*3) / 4.0
	kh := (swH - kPad*2 - kGap) / 2.0
	kr := float32(dp(2))
	if styleID == styleMaterialTeal {
		kr = kh / 2.0
	}
	for k := 0; k < 4; k++ {
		kx := swX + kPad + float32(k)*(kw+kGap)
		gfx.FillRoundedRect(kx, swY+kPad, kw, kh, kr, pvKey)
		if styleID == styleCyberNeon {
			gfx.DrawRoundedRect(kx, swY+kPad, kw, kh, kr, pvBorder, 0.7)
		}
	}
	// Bottom row: func key + spacebar + glowing accent Enter pill
	by := swY + kPad + kh + kGap
	gfx.FillRoundedRect(swX+kPad, by, kw*0.85, kh, kr, pvFunc)
	spX := swX + kPad + kw*0.85 + kGap
	spW := swW - kPad*2 - kw*1.95 - kGap*2
	gfx.FillRoundedRect(spX, by, spW, kh, kr, pvKey)
	enX := spX + spW + kGap
	enW := swW - kPad - (enX - swX)
	gfx.FillRoundedRect(enX, by, enW, kh, kr, pvAccent)

	// Active checkmark badge in top-right of swatch
	if selected {
		badgeR := float32(dp(6))
		bx := swX + swW - badgeR + float32(dp(1))
		byc := swY + badgeR - float32(dp(1))
		gfx.FillCircle(bx, byc, badgeR, pvBorder)
		tick := uint32(0xFFFFFFFF)
		if pvBorder&0x00FFFFFF == 0x3ED3B8 {
			tick = 0xFF0F1115
		}
		gfx.DrawLine(bx-float32(dp(3)), byc, bx-float32(dp(1)), byc+float32(dp(2)), tick, 1.5)
		gfx.DrawLine(bx-float32(dp(1)), byc+float32(dp(2)), bx+float32(dp(3)), byc-float32(dp(2)), tick, 1.5)
	}

	// Title & Subtitle below swatch
	pSetBkMode.Call(dc, TRANSPARENT)
	titleCol := colText
	subCol := colMuted
	if selected {
		titleCol = colGold
	}
	if selected && solidAccentSelected() {
		titleCol = accentOnCol()
		subCol = accentOnCol()
	}
	pSetTextColor.Call(dc, titleCol)
	oldF, _, _ := pSelectObject.Call(dc, fontBold)

	trc := rect{rc.left + dp(2), rc.top + dp(31), rc.right - dp(2), rc.top + dp(47)}
	if pressed {
		trc.top += dp(1)
	}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr(title))),
		^uintptr(0), uintptr(unsafe.Pointer(&trc)),
		DT_CENTER|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

	pSetTextColor.Call(dc, subCol)
	pSelectObject.Call(dc, fontHint)
	drc := rect{rc.left + dp(2), rc.top + dp(46), rc.right - dp(2), rc.bottom - dp(3)}
	if pressed {
		drc.top += dp(1)
	}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr(sub))),
		^uintptr(0), uintptr(unsafe.Pointer(&drc)),
		DT_CENTER|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

	pSelectObject.Call(dc, oldF)
}

