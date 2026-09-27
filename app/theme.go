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
	bg:          0x302A0B, // #0B2A30 deep ink teal
	card:        0x403812, // #123840 jade card
	border:      0x5E542A, // #2A545E jade card border
	text:        0xE3EEF2, // #F2EEE3 warm ivory
	gold:        0x78B9D8, // #D8B978 antique gold
	amber:       0x63C5F2, // #F2C563 gold halo
	jade:        0xB8D33E, // #3ED3B8 luminous mint jade (switches ON, badges)
	vermilion:   0x354BDE, // #DE4B35 cinnabar red
	muted:       0xA4A98F, // #8FA9A4 soft celadon mist
	pill:        0x463F16, // #163F46 switch off track
	goldDim:     0x5A8DA0, // #A08D5A dim antique gold filigree
	btnBg:       0x504817, // #174850 tablet button fill
	btnHover:    0x645A1E, // #1E5A64 tablet button hover
	btnBorder:   0x847A3A, // #3A7A84 jade button border
	btnText:     0xE3EEF2, // #F2EEE3 button text
	navActiveBg: 0x544B1A, // #1A4B54 active tab jade pill
	navActiveBd: 0x78B9D8, // #D8B978 active tab gold border
}

// Minh Ngọc · Nhật (Bright Jade · Day): clean trung tính — nền xám sáng
// phẳng, card trắng tinh, border xám mờ, vàng đồng chỉ ở điểm nhấn.
var palLight = palette{
	bg:          0xF3F4F4, // #F4F4F3 flat neutral off-white
	card:        0xFFFFFF, // #FFFFFF pure white card
	border:      0xDEE2E3, // #E3E2DE faint gray hairline
	text:        0x20211F, // #1F2120 near-black ink
	gold:        0x2E7CA6, // #A67C2E bronze gold accent
	amber:       0x1F7FC7, // #C77F1F warm amber
	jade:        0x8A9E2E, // #2E9E8A deep mint jade (switches ON)
	vermilion:   0x2437B3, // #B33724 cinnabar red
	muted:       0x767A7A, // #7A7A76 neutral gray
	pill:        0xDCDDE0, // #E0DDDC light gray track
	goldDim:     0xBAC8CF, // #CFC8BA faint ornament line
	btnBg:       0xF4F5F5, // #F5F5F4 button fill
	btnHover:    0xFFFFFF, // #FFFFFF pure white hover
	btnBorder:   0xCACACD, // #CDCACA soft gray-gold border
	btnText:     0x20211F, // #1F2120 button text
	navActiveBg: 0xE9EBEC, // #ECEBE9 neutral active plaque
	navActiveBd: 0x3E8AB0, // #B08A3E active tab gold border
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
