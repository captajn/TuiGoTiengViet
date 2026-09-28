package main

// Theme system: "Pháp Bảo" palette — xianxia 鎏金/灵玉/朱砂 on Thanh Đại (dark)
// or Bạch Ngọc / 宣纸 (light). 0=auto (follow Windows), 1=dark, 2=light.

import (
	"golang.org/x/sys/windows/registry"
)

// colors are COLORREF (0x00BBGGRR), kept as uintptr for direct proc calls
var (
	colBg          uintptr // window background
	colCard        uintptr // card fill
	colBorder      uintptr // card/pill border
	colText        uintptr // primary text
	colGold        uintptr // 鎏金 hoàng kim accent — branding, title, border glow
	colAmber       uintptr // 琥珀 hổ phách — glowing paw, accents
	colJade        uintptr // 灵玉 ngọc bích / mint — HUD dot, active switches
	colVermilion   uintptr // 朱砂 chu sa — stamps, badges, red seal
	colMuted       uintptr // secondary text
	colPill        uintptr // switch off track
	colGoldDim     uintptr // 鎏金 phai — ornament hairlines, celestial arcs
	colBtnBg       uintptr // owner-drawn button background
	colBtnHover    uintptr // button hover / active
	colBtnBorder   uintptr // button border
	colBtnText     uintptr // button text
	colNavActiveBg uintptr // active sidebar nav tab fill
	colNavActiveBd uintptr // active sidebar nav tab border
)

type palette struct {
	bg, card, border, text, gold, amber, jade, vermilion uintptr
	muted, pill, goldDim, btnBg, btnHover, btnBorder     uintptr
	btnText, navActiveBg, navActiveBd                    uintptr
}

// 墨青·夜 (Mặc Thanh · Dạ) — theo promo tiên hiệp: nền lam-ngọc sâu,
// card ngọc bích, vàng đồng cổ ấm, công tắc ngọc bích phát sáng.
var palDark = palette{
	bg:          0x25211B, // #1B2125 slate
	card:        0x302B24, // #242B30
	border:      0x443D34, // #343D44
	text:        0xEDF2EF, // #EFF2ED
	gold:        0x78B9D8,
	amber:       0x63C5F2,
	jade:        0xB9D46C, // #6CD4B9
	vermilion:   0x697BEB,
	muted:       0xAAA89C, // #9CA8AA
	pill:        0x494237,
	goldDim:     0x667D88,
	btnBg:       0x39322A,
	btnHover:    0x473F34,
	btnBorder:   0x5A5145,
	btnText:     0xEDF2EF,
	navActiveBg: 0x3B3D24, // #243D3B
	navActiveBd: 0x3B3D24,
}

// Warm white, readable secondary text, and a soft mint selection surface.
var palLight = palette{
	bg:          0xF5F7F7, // #F7F7F5
	card:        0xFFFFFF,
	border:      0xDFE5E3, // #E3E5DF
	text:        0x2C3024, // #24302C
	gold:        0x2E7CA6,
	amber:       0x1F7FC7,
	jade:        0x647F19, // #197F64
	vermilion:   0x2437B3,
	muted:       0x697568, // #687569
	pill:        0xD9E0DC,
	goldDim:     0xBAC8CF,
	btnBg:       0xFFFFFF,
	btnHover:    0xEDF3F0,
	btnBorder:   0xC8D3CD,
	btnText:     0x2C3024,
	navActiveBg: 0xE7F0E2, // #E2F0E7
	navActiveBd: 0xE7F0E2,
}

const (
	themeAuto = iota
	themeDark
	themeLight
)

func windowsDark() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`, registry.READ)
	if err != nil {
		return true
	}
	defer k.Close()
	v, _, err := k.GetIntegerValue("AppsUseLightTheme")
	return err == nil && v == 0
}

func isDark() bool {
	switch cfg.Theme {
	case themeDark:
		return true
	case themeLight:
		return false
	}
	return windowsDark()
}

func deleteGdiObj(h *uintptr) {
	if *h != 0 {
		pDeleteObject.Call(*h)
		*h = 0
	}
}

var (
	brBg, brCard, brBorder, brNavActive, brJade, brHero, brFooter, brRadioActive uintptr
	penBorder, penGold, penGoldDim, penNavActive, penJade, penMuted, penWhite2   uintptr
)

func applyTheme() {
	p := palDark
	if !isDark() {
		p = palLight
	}
	colBg, colCard, colBorder = p.bg, p.card, p.border
	colText, colMuted = p.text, p.muted
	colGold, colAmber, colJade = p.gold, p.amber, p.jade
	colVermilion = p.vermilion
	colPill, colGoldDim = p.pill, p.goldDim
	colBtnBg, colBtnHover, colBtnBorder = p.btnBg, p.btnHover, p.btnBorder
	colBtnText, colNavActiveBg, colNavActiveBd = p.btnText, p.navActiveBg, p.navActiveBd

	// clean up existing brushes/pens
	deleteGdiObj(&brBg)
	deleteGdiObj(&brCard)
	deleteGdiObj(&brBorder)
	deleteGdiObj(&brNavActive)
	deleteGdiObj(&brJade)
	deleteGdiObj(&brHero)
	deleteGdiObj(&brFooter)
	deleteGdiObj(&brRadioActive)
	deleteGdiObj(&penBorder)
	deleteGdiObj(&penGold)
	deleteGdiObj(&penGoldDim)
	deleteGdiObj(&penNavActive)
	deleteGdiObj(&penJade)
	deleteGdiObj(&penMuted)
	deleteGdiObj(&penWhite2)

	// recreate brushes/pens
	brBg, _, _ = pCreateSolidBrush.Call(colBg)
	brCard, _, _ = pCreateSolidBrush.Call(colCard)
	brBorder, _, _ = pCreateSolidBrush.Call(colBorder)
	brNavActive, _, _ = pCreateSolidBrush.Call(colNavActiveBg)
	brJade, _, _ = pCreateSolidBrush.Call(colJade)
	brHero, _, _ = pCreateSolidBrush.Call(colCard)
	brFooter, _, _ = pCreateSolidBrush.Call(colBg)
	brRadioActive, _, _ = pCreateSolidBrush.Call(colNavActiveBg)

	penBorder, _, _ = pCreatePen.Call(PS_SOLID, 1, colBorder)
	penGold, _, _ = pCreatePen.Call(PS_SOLID, 1, colGold)
	penGoldDim, _, _ = pCreatePen.Call(PS_SOLID, 1, colGoldDim)
	penNavActive, _, _ = pCreatePen.Call(PS_SOLID, 1, colNavActiveBd)
	penJade, _, _ = pCreatePen.Call(PS_SOLID, 1, colJade)
	penMuted, _, _ = pCreatePen.Call(PS_SOLID, 1, colMuted)
	penWhite2, _, _ = pCreatePen.Call(PS_SOLID, 2, 0xFFFFFF)
}
