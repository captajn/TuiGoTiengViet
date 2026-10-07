package main

import (
	"unsafe"
)

// ui_components.go provides smooth, zero-aliasing UI controls powered by GDI+
// with modern Minimalist and Liquid Glass styling.

// drawSmoothSwitch renders a modern iOS/Fluent-style pill toggle switch with GDI+.
func drawSmoothSwitch(dc uintptr, rc rect, id int, on bool) {
	gfx := NewGfx(dc)
	if gfx == nil {
		return
	}
	defer gfx.Destroy()

	// Fill background of control with card color
	gfx.FillRect(float32(rc.left), float32(rc.top), float32(rc.right-rc.left), float32(rc.bottom-rc.top), argbCard)

	pillH := float32(dp(22))
	pillW := float32(dp(40))
	py := float32(rc.top) + (float32(rc.bottom-rc.top)-pillH)/2.0
	px := float32(rc.left)
	radius := pillH / 2.0

	// Track styling with subtle Liquid Glass gradient
	if on {
		// Active track: vibrant accent with subtle specular top
		topCol := argbSwitchOn
		bottomCol := HexAlphaARGB(0xEE, argbSwitchOn&0x00FFFFFF)
		gfx.FillRoundedRectGradient(px, py, pillW, pillH, radius, topCol, bottomCol)
	} else {
		// Inactive track: neutral muted slate
		topCol := argbSwitchOff
		bottomCol := HexAlphaARGB(0xDD, argbSwitchOff&0x00FFFFFF)
		gfx.FillRoundedRectGradient(px, py, pillW, pillH, radius, topCol, bottomCol)
		gfx.DrawRoundedRect(px, py, pillW, pillH, radius, argbBorder, 1.0)
	}

	// Thumb (circular knob)
	knobD := pillH - float32(dp(4))
	knobR := knobD / 2.0
	kx := px + float32(dp(2))
	if on {
		kx = px + pillW - knobD - float32(dp(2))
	}
	ky := py + float32(dp(2))

	// Subtle knob drop shadow
	gfx.FillCircle(kx+knobR, ky+knobR+0.75, knobR, HexAlphaARGB(0x35, 0x000000))
	// Pure white thumb with subtle sheen
	gfx.FillCircle(kx+knobR, ky+knobR, knobR, 0xFFFFFFFF)

	// Draw label text
	pSetBkMode.Call(dc, TRANSPARENT)
	pSetTextColor.Call(dc, colText)
	oldF, _, _ := pSelectObject.Call(dc, fontNormal)
	trc := rect{rc.left + dp(50), rc.top, rc.right, rc.bottom}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr(swText[id]))),
		^uintptr(0), uintptr(unsafe.Pointer(&trc)),
		DT_VCENTER|DT_SINGLELINE)
	pSelectObject.Call(dc, oldF)
}

// drawSmoothCheckbox renders an anti-aliased rounded checkbox.
func drawSmoothCheckbox(dc uintptr, rc rect, id int, checked bool) {
	gfx := NewGfx(dc)
	if gfx == nil {
		return
	}
	defer gfx.Destroy()

	gfx.FillRect(float32(rc.left), float32(rc.top), float32(rc.right-rc.left), float32(rc.bottom-rc.top), argbBg)

	boxS := float32(dp(18))
	by := float32(rc.top) + (float32(rc.bottom-rc.top)-boxS)/2.0
	bx := float32(rc.left)
	r := float32(dp(4))

	if checked {
		// Active: filled with accent color
		gfx.FillRoundedRectGradient(bx, by, boxS, boxS, r, argbAccent, HexAlphaARGB(0xDD, argbAccent&0x00FFFFFF))
		// Crisp anti-aliased checkmark
		p1x, p1y := bx+float32(dp(4)), by+float32(dp(9))
		p2x, p2y := bx+float32(dp(7)), by+float32(dp(13))
		p3x, p3y := bx+float32(dp(14)), by+float32(dp(5))
		thk := float32(max(int32(2), dp(2)))
		gfx.DrawLine(p1x, p1y, p2x, p2y, 0xFFFFFFFF, thk)
		gfx.DrawLine(p2x, p2y, p3x, p3y, 0xFFFFFFFF, thk)
	} else {
		// Inactive: subtle frosted card surface with border
		gfx.FillRoundedRect(bx, by, boxS, boxS, r, argbCard)
		gfx.DrawRoundedRect(bx, by, boxS, boxS, r, argbBorder, 1.2)
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

// drawSmoothRadioCard renders an anti-aliased radio selection card with Liquid Glass styling.
func drawSmoothRadioCard(dc uintptr, rc rect, id int, selected bool) {
	gfx := NewGfx(dc)
	if gfx == nil {
		return
	}
	defer gfx.Destroy()

	x, y, w, h := float32(rc.left), float32(rc.top), float32(rc.right-rc.left), float32(rc.bottom-rc.top)
	r := float32(dp(8))

	if selected {
		// Selected: subtle accent tinted glass card
		bgTop := HexAlphaARGB(0x25, argbAccent&0x00FFFFFF)
		bgBottom := HexAlphaARGB(0x12, argbAccent&0x00FFFFFF)
		gfx.FillRoundedRectGradient(x, y, w, h, r, bgTop, bgBottom)
		gfx.DrawRoundedRect(x, y, w, h, r, argbAccent, 1.2)
		// Top specular highlight
		gfx.DrawLine(x+r, y+1, x+w-r, y+1, HexAlphaARGB(0x50, 0xFFFFFF), 1.0)
	} else {
		// Unselected: neutral liquid glass surface
		gfx.DrawGlassCard(x, y, w, h, r, isDark(), false)
	}

	// Radio bullet
	cx := x + float32(dp(16))
	cy := y + h/2.0
	outerR := float32(dp(6))

	if selected {
		gfx.DrawCircle(cx, cy, outerR, argbAccent, 1.5)
		gfx.FillCircle(cx, cy, float32(dp(3)), argbAccent)
	} else {
		gfx.DrawCircle(cx, cy, outerR, argbMuted, 1.2)
	}

	pSetBkMode.Call(dc, TRANSPARENT)
	pSetTextColor.Call(dc, colText)
	oldF, _, _ := pSelectObject.Call(dc, fontBold)
	trc := rect{int32(cx + outerR + float32(dp(10))), rc.top + dp(6), rc.right - dp(4), rc.top + dp(24)}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr(btnText[id]))),
		^uintptr(0), uintptr(unsafe.Pointer(&trc)),
		DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

	pSetTextColor.Call(dc, colMuted)
	pSelectObject.Call(dc, fontHint)
	drc := rect{int32(cx + outerR + float32(dp(10))), rc.top + dp(24), rc.right - dp(4), rc.bottom - dp(4)}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr(radioSubtitle[id]))),
		^uintptr(0), uintptr(unsafe.Pointer(&drc)),
		DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

	pSelectObject.Call(dc, oldF)
}

// drawSmoothButton renders a button with liquid glass sheen and anti-aliasing.
func drawSmoothButton(dc uintptr, rc rect, id int, pressed bool) {
	gfx := NewGfx(dc)
	if gfx == nil {
		return
	}
	defer gfx.Destroy()

	x, y, w, h := float32(rc.left), float32(rc.top), float32(rc.right-rc.left), float32(rc.bottom-rc.top)
	r := float32(dp(6))

	gfx.DrawGlassCard(x, y, w, h, r, isDark(), pressed)

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

// drawSmoothModifierKeycap renders a modern mechanical keycap badge.
func drawSmoothModifierKeycap(dc uintptr, rc rect, id int, selected, pressed bool) {
	gfx := NewGfx(dc)
	if gfx == nil {
		return
	}
	defer gfx.Destroy()

	x, y, w, h := float32(rc.left), float32(rc.top), float32(rc.right-rc.left), float32(rc.bottom-rc.top)
	r := float32(dp(6))

	if selected {
		bgTop := HexAlphaARGB(0x35, argbAccent&0x00FFFFFF)
		bgBottom := HexAlphaARGB(0x18, argbAccent&0x00FFFFFF)
		gfx.FillRoundedRectGradient(x, y, w, h, r, bgTop, bgBottom)
		gfx.DrawRoundedRect(x, y, w, h, r, argbAccent, 1.2)
	} else {
		gfx.DrawGlassCard(x, y, w, h, r, isDark(), pressed)
	}

	pSetBkMode.Call(dc, TRANSPARENT)
	txtCol := colText
	if selected {
		txtCol = colGold
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

	case 4: // Information: sleek circle with 'i'
		gfx.DrawCircle(cx, cy, size*0.46, color, thk)
		gfx.FillCircle(cx, cy-size*0.22, float32(dp(2)), color)
		gfx.DrawLine(cx, cy-size*0.06, cx, cy+size*0.22, color, thk)
	}
}

// drawMinimalBrandMark draws a sleek typographic keycap logo [ V ] for the header.
func drawMinimalBrandMark(gfx *Gfx, x, y, size float32) {
	if gfx == nil {
		return
	}
	r := float32(dp(7))
	// Frosted keycap background
	bgTop := HexAlphaARGB(0x40, argbAccent&0x00FFFFFF)
	bgBottom := HexAlphaARGB(0x15, argbAccent&0x00FFFFFF)
	gfx.FillRoundedRectGradient(x, y, size, size, r, bgTop, bgBottom)
	gfx.DrawRoundedRect(x, y, size, size, r, argbAccent, 1.2)
	// Top specular highlight
	gfx.DrawLine(x+r, y+1, x+size-r, y+1, HexAlphaARGB(0x70, 0xFFFFFF), 1.0)

	// Minimal geometric "V" symbol
	thk := float32(max(int32(2), dp(2)))
	cx := x + size/2.0
	cy := y + size/2.0
	armW := size * 0.24
	armH := size * 0.22
	gfx.DrawLine(cx-armW, cy-armH, cx, cy+armH, 0xFFFFFFFF, thk)
	gfx.DrawLine(cx, cy+armH, cx+armW, cy-armH, 0xFFFFFFFF, thk)
}

// drawSmoothThemeButton renders a modern segmented theme option with icon and label.
func drawSmoothThemeButton(dc uintptr, rc rect, id int, pressed bool) {
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

	if selected {
		// Active option: jade green tinted liquid glass card
		bgTop := HexAlphaARGB(0x35, argbAccent&0x00FFFFFF)
		bgBottom := HexAlphaARGB(0x18, argbAccent&0x00FFFFFF)
		gfx.FillRoundedRectGradient(x, y, w, h, r, bgTop, bgBottom)
		gfx.DrawRoundedRect(x, y, w, h, r, argbAccent, 1.4)
		// Specular top highlight line
		gfx.DrawLine(x+r, y+1, x+w-r, y+1, HexAlphaARGB(0x60, 0xFFFFFF), 1.0)
	} else {
		// Inactive option: frosted glass surface
		gfx.DrawGlassCard(x, y, w, h, r, isDark(), pressed)
	}

	// Draw Icon
	iconCol := argbMuted
	if selected {
		iconCol = argbAccent
	}
	iconX := x + float32(dp(18))
	iconY := y + h/2.0
	iconSz := float32(dp(18))
	thk := float32(max(int32(1), dp(2)))

	switch themeMode {
	case themeAuto: // System / Monitor icon
		sw := iconSz * 0.9
		sh := iconSz * 0.65
		sx := iconX - sw/2.0
		sy := iconY - sh/2.0 - float32(dp(2))
		gfx.DrawRoundedRect(sx, sy, sw, sh, float32(dp(2)), iconCol, thk)
		// Stand
		gfx.DrawLine(iconX, sy+sh, iconX, sy+sh+float32(dp(4)), iconCol, thk)
		gfx.DrawLine(iconX-float32(dp(5)), sy+sh+float32(dp(4)), iconX+float32(dp(5)), sy+sh+float32(dp(4)), iconCol, thk)

	case themeLight: // Sun icon (circle + rays)
		sr := iconSz * 0.28
		gfx.DrawCircle(iconX, iconY, sr, iconCol, thk)
		// 8 rays
		rayLen := float32(dp(3))
		rayDist := sr + float32(dp(2))
		gfx.DrawLine(iconX, iconY-rayDist-rayLen, iconX, iconY-rayDist, iconCol, thk)
		gfx.DrawLine(iconX, iconY+rayDist, iconX, iconY+rayDist+rayLen, iconCol, thk)
		gfx.DrawLine(iconX-rayDist-rayLen, iconY, iconX-rayDist, iconY, iconCol, thk)
		gfx.DrawLine(iconX+rayDist, iconY, iconX+rayDist+rayLen, iconY, iconCol, thk)
		d := (rayDist) * 0.707
		dl := (rayDist + rayLen) * 0.707
		gfx.DrawLine(iconX-dl, iconY-dl, iconX-d, iconY-d, iconCol, thk)
		gfx.DrawLine(iconX+d, iconY+d, iconX+dl, iconY+dl, iconCol, thk)
		gfx.DrawLine(iconX-dl, iconY+dl, iconX-d, iconY+d, iconCol, thk)
		gfx.DrawLine(iconX+d, iconY-d, iconX+dl, iconY-dl, iconCol, thk)

	case themeDark: // Moon crescent icon
		mr := iconSz * 0.42
		gfx.DrawCircle(iconX, iconY, mr, iconCol, thk)
		gfx.DrawCircle(iconX+float32(dp(3)), iconY-float32(dp(1)), mr*0.82, argbCard, thk+0.5)
	}

	// Text: Title & Subtitle
	pSetBkMode.Call(dc, TRANSPARENT)
	titleCol := colText
	if selected {
		titleCol = colGold
	}
	pSetTextColor.Call(dc, titleCol)
	oldF, _, _ := pSelectObject.Call(dc, fontBold)

	title := "Hệ thống"
	sub := "Theo Windows"
	if themeMode == themeLight {
		title = "Sáng"
		sub = "Bạch Ngọc"
	} else if themeMode == themeDark {
		title = "Tối"
		sub = "Thanh Đại"
	}

	textLeft := int32(iconX + iconSz/2.0 + float32(dp(10)))
	trc := rect{textLeft, rc.top + dp(6), rc.right - dp(4), rc.top + dp(24)}
	if pressed {
		trc.top += dp(1)
	}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr(title))),
		^uintptr(0), uintptr(unsafe.Pointer(&trc)),
		DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

	pSetTextColor.Call(dc, colMuted)
	pSelectObject.Call(dc, fontHint)
	drc := rect{textLeft, rc.top + dp(24), rc.right - dp(4), rc.bottom - dp(4)}
	if pressed {
		drc.top += dp(1)
	}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr(sub))),
		^uintptr(0), uintptr(unsafe.Pointer(&drc)),
		DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

	pSelectObject.Call(dc, oldF)
}
