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
	colGold        uintptr // primary accent (modern azure blue)
	colAmber       uintptr // secondary accent
	colJade        uintptr // active switch / indicator (emerald)
	colVermilion   uintptr // badges / alerts
	colMuted       uintptr // secondary text
	colPill        uintptr // switch off track
	colGoldDim     uintptr // subtle divider / keycap border
	colBtnBg       uintptr // button background
	colBtnHover    uintptr // button hover
	colBtnBorder   uintptr // button border
	colBtnText     uintptr // button text
	colNavActiveBg uintptr // active sidebar nav tab fill
	colNavActiveBd uintptr // active sidebar nav tab border

	// GDI+ 32-bit ARGB tokens for smooth anti-aliased rendering
	argbBg        uint32
	argbCard      uint32
	argbBorder    uint32
	argbText      uint32
	argbMuted     uint32
	argbAccent    uint32
	argbSwitchOn  uint32
	argbSwitchOff uint32
	argbKeycap    uint32
)

type palette struct {
	bg, card, border, text, gold, amber, jade, vermilion uintptr
	muted, pill, goldDim, btnBg, btnHover, btnBorder     uintptr
	btnText, navActiveBg, navActiveBd                    uintptr
	aBg, aCard, aBorder, aText, aMuted, aAccent          uint32
	aSwitchOn, aSwitchOff, aKeycap                       uint32
}

// Minimal Obsidian / Deep Zinc Dark Mode (Liquid Glass với tông xanh lá ngọc bích theo logo)
var palDark = palette{
	bg:          0x15110F, // #0F1115
	card:        0x201A18, // #181A20
	border:      0x352B27, // #272B35
	text:        0xFCFAF8, // #F8FAFC
	gold:        0xB8D33E, // #3ED3B8 xanh lá ngọc bích dạ quang (luminous mint jade)
	amber:       0x70CDBB, // #BBCD70
	jade:        0xB8D33E, // #3ED3B8 xanh lá ngọc bích
	vermilion:   0x4444EF, // #EF4444
	muted:       0xB8A394, // #94A3B8
	pill:        0x463833, // #333846
	goldDim:     0x3B291E, // #1E293B
	btnBg:       0x302622, // #222630
	btnHover:    0x40332D, // #2D3340
	btnBorder:   0x55443D, // #3D4455
	btnText:     0xFCFAF8, // #F8FAFC
	navActiveBg: 0x3B3D24, // #243D3B nền ngọc bích trầm
	navActiveBd: 0xB8D33E, // #3ED3B8 viền ngọc bích

	aBg:        0xFF0F1115,
	aCard:      0xFF181A20,
	aBorder:    0xFF272B35,
	aText:      0xFFF8FAFC,
	aMuted:     0xFF94A3B8,
	aAccent:    0xFF3ED3B8, // #3ED3B8 Xanh lá ngọc bích chủ đạo theo logo
	aSwitchOn:  0xFF3ED3B8, // #3ED3B8
	aSwitchOff: 0xFF333846,
	aKeycap:    0xFF222630,
}

// Minimal Pristine Slate / Pure White Light Mode với tông xanh lá ngọc bích theo logo
var palLight = palette{
	bg:          0xFCFAF8, // #F8FAFC
	card:        0xFFFFFF, // #FFFFFF
	border:      0xF0E8E2, // #E2E8F0
	text:        0x2A170F, // #0F172A
	gold:        0x647F19, // #197F64 xanh lá ngọc bích trầm (rich jade green)
	amber:       0x409E2E, // #2E9E40
	jade:        0x647F19, // #197F64 xanh lá ngọc bích
	vermilion:   0x2626DC, // #DC2626
	muted:       0x8B7464, // #64748B
	pill:        0xE1D5CB, // #CBD5E1
	goldDim:     0xF0E8E2, // #E2E8F0
	btnBg:       0xFFFFFF, // #FFFFFF
	btnHover:    0xF9F5F1, // #F1F5F9
	btnBorder:   0xE1D5CB, // #CBD5E1
	btnText:     0x2A170F, // #0F172A
	navActiveBg: 0xE7F0E2, // #E2F0E7 nền ngọc bích sáng dịu
	navActiveBd: 0x647F19, // #197F64 viền ngọc bích

	aBg:        0xFFF8FAFC,
	aCard:      0xFFFFFFFF,
	aBorder:    0xFFE2E8F0,
	aText:      0xFF0F172A,
	aMuted:     0xFF64748B,
	aAccent:    0xFF197F64, // #197F64 Xanh lá ngọc bích chủ đạo theo logo
	aSwitchOn:  0xFF197F64, // #197F64
	aSwitchOff: 0xFFCBD5E1,
	aKeycap:    0xFFF1F5F9,
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
	argbBg, argbCard, argbBorder = p.aBg, p.aCard, p.aBorder
	argbText, argbMuted, argbAccent = p.aText, p.aMuted, p.aAccent
	argbSwitchOn, argbSwitchOff, argbKeycap = p.aSwitchOn, p.aSwitchOff, p.aKeycap

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
