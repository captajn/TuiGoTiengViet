package main

// Theme system: "Pháp Bảo" palette — xianxia 鎏金/灵玉/朱砂 on Thanh Đại (dark)
// or Bạch Ngọc / 宣纸 (light). 0=auto (follow Windows), 1=dark, 2=light.

import (
	"time"

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

// Cơ bản — plain flat theme: solid surfaces, no gradient blooms, no glass
// translucency. Neutral Windows grays with the jade brand accent.
var palBasicDark = palette{
	bg:          0x202020, // #202020
	card:        0x383838, // #383838 — matches Win32 dark-combo face tone
	border:      0x4A4A4A, // #4A4A4A
	text:        0xF0F0F0, // #F0F0F0
	gold:        0xB8D33E, // #3ED3B8
	amber:       0xB8D33E, // #3ED3B8
	jade:        0xB8D33E, // #3ED3B8
	vermilion:   0x4444EF, // #EF4444
	muted:       0xA0A0A0, // #A0A0A0
	pill:        0x4A4A4A, // #4A4A4A
	goldDim:     0x404040, // #404040
	btnBg:       0x333333, // #333333
	btnHover:    0x3D3D3D, // #3D3D3D
	btnBorder:   0x4A4A4A, // #4A4A4A
	btnText:     0xF0F0F0, // #F0F0F0
	navActiveBg: 0x383838, // #383838
	navActiveBd: 0xB8D33E, // #3ED3B8

	aBg:        0xFF202020,
	aCard:      0xFF383838,
	aBorder:    0xFF4A4A4A,
	aText:      0xFFF0F0F0,
	aMuted:     0xFFA0A0A0,
	aAccent:    0xFF3ED3B8,
	aSwitchOn:  0xFF3ED3B8,
	aSwitchOff: 0xFF4A4A4A,
	aKeycap:    0xFF333333,
}

var palBasicLight = palette{
	bg:          0xF3F3F3, // #F3F3F3
	card:        0xFFFFFF, // #FFFFFF
	border:      0xD9D9D9, // #D9D9D9
	text:        0x1B1B1B, // #1B1B1B
	gold:        0x647F19, // #197F64
	amber:       0x647F19, // #197F64
	jade:        0x647F19, // #197F64
	vermilion:   0x2626DC, // #DC2626
	muted:       0x6B6B6B, // #6B6B6B
	pill:        0xD1D1D1, // #D1D1D1
	goldDim:     0xE0E0E0, // #E0E0E0
	btnBg:       0xFFFFFF, // #FFFFFF
	btnHover:    0xF0F0F0, // #F0F0F0
	btnBorder:   0xD1D1D1, // #D1D1D1
	btnText:     0x1B1B1B, // #1B1B1B
	navActiveBg: 0xE0E0E0, // #E0E0E0
	navActiveBd: 0x647F19, // #197F64

	aBg:        0xFFF3F3F3,
	aCard:      0xFFFFFFFF,
	aBorder:    0xFFD9D9D9,
	aText:      0xFF1B1B1B,
	aMuted:     0xFF6B6B6B,
	aAccent:    0xFF197F64,
	aSwitchOn:  0xFF197F64,
	aSwitchOff: 0xFFD1D1D1,
	aKeycap:    0xFFF0F0F0,
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

// 1A. Apple iOS 18 Pro (Light) — Crisp Silver-Gray & Pure White 3D Cards with Apple Blue (#007AFF) & iOS Green (#34C759)
var palIosLight = palette{
	bg:          0xF0E9E5, // #E5E9F0
	card:        0xFFFFFF, // #FFFFFF
	border:      0xD8CDC5, // #C5CDD8
	text:        0x1E1C1C, // #1C1C1E
	gold:        0xFF7A00, // #007AFF Apple Blue
	amber:       0xD65658, // #5856D6 Apple Indigo
	jade:        0xFF7A00, // #007AFF Apple Blue
	vermilion:   0x303BFF, // #FF3B30 Apple Red
	muted:       0x666363, // #636366
	pill:        0xDBD5D1, // #D1D5DB
	goldDim:     0xDBD5D1, // #D1D5DB
	btnBg:       0xFFFFFF, // #FFFFFF
	btnHover:    0xF8F4F2, // #F2F4F8
	btnBorder:   0xD8CDC5, // #C5CDD8
	btnText:     0x1E1C1C, // #1C1C1E
	navActiveBg: 0xFFEBDC, // #DCEBFF
	navActiveBd: 0xFF7A00, // #007AFF

	aBg:        0xFFE5E9F0,
	aCard:      0xFFFFFFFF,
	aBorder:    0xFFC5CDD8,
	aText:      0xFF1C1C1E,
	aMuted:     0xFF636366,
	aAccent:    0xFF007AFF,
	aSwitchOn:  0xFF34C759,
	aSwitchOff: 0xFFD1D5DB,
	aKeycap:    0xFFFFFFFF,
}

// 1B. Apple iOS 18 Pro (Dark) — Pure OLED Black (#000000), Charcoal Elevated Card (#1C1C1E) & iOS Dark Blue (#0A84FF)
var palIosDark = palette{
	bg:          0x000000, // #000000
	card:        0x1E1C1C, // #1C1C1E
	border:      0x3A3838, // #38383A
	text:        0xFFFFFF, // #FFFFFF
	gold:        0xFF840A, // #0A84FF iOS Dark Blue
	amber:       0xE65C5E, // #5E5CE6 iOS Dark Indigo
	jade:        0xFF840A, // #0A84FF
	vermilion:   0x3A45FF, // #FF453A
	muted:       0x938E8E, // #8E8E93
	pill:        0x3D3939, // #39393D
	goldDim:     0x3A3838, // #38383A
	btnBg:       0x2E2C2C, // #2C2C2E
	btnHover:    0x3C3A3A, // #3A3A3C
	btnBorder:   0x4A4848, // #48484A
	btnText:     0xFFFFFF, // #FFFFFF
	navActiveBg: 0x3D240A, // #0A243D
	navActiveBd: 0xFF840A, // #0A84FF

	aBg:        0xFF000000,
	aCard:      0xFF1C1C1E,
	aBorder:    0xFF38383A,
	aText:      0xFFFFFFFF,
	aMuted:     0xFF8E8E93,
	aAccent:    0xFF0A84FF,
	aSwitchOn:  0xFF30D158,
	aSwitchOff: 0xFF39393D,
	aKeycap:    0xFF2C2C2E,
}

// 2A. Cyber Neon (Dark AMOLED) — True Black (#04060A), Glowing Cyan (#00F2FE) & Neon Mint (#10B981)
var palCyberDark = palette{
	bg:          0x0A0604, // #04060A
	card:        0x1E110B, // #0B111E
	border:      0xE9A50E, // #0EA5E9
	text:        0xFEF2E0, // #E0F2FE
	gold:        0xFEF200, // #00F2FE Electric Cyan
	amber:       0xF8BD38, // #38BDF8
	jade:        0x81B910, // #10B981 Neon Mint
	vermilion:   0x5E3FF4, // #F43F5E
	muted:       0xFCD37D, // #7DD3FC
	pill:        0x3B2216, // #16223B
	goldDim:     0x6E4A0C, // #0C4A6E
	btnBg:       0x2A170F, // #0F172A
	btnHover:    0x3B291E, // #1E293B
	btnBorder:   0xE9A50E, // #0EA5E9
	btnText:     0xFEF2E0, // #E0F2FE
	navActiveBg: 0x492F08, // #082F49
	navActiveBd: 0xFEF200, // #00F2FE

	aBg:        0xFF04060A,
	aCard:      0xFF0B111E,
	aBorder:    0xFF0EA5E9,
	aText:      0xFFE0F2FE,
	aMuted:     0xFF7DD3FC,
	aAccent:    0xFF00F2FE,
	aSwitchOn:  0xFF10B981,
	aSwitchOff: 0xFF16223B,
	aKeycap:    0xFF0F172A,
}

// 2B. Cyber Neon (Light Daylight) — Ice Cyan Lab (#E0F2FE), Crisp White & Electric Sky (#0284C7) + Neon Mint (#10B981)
var palCyberLight = palette{
	bg:          0xFEF2E0, // #E0F2FE
	card:        0xFFFFFF, // #FFFFFF
	border:      0xF8BD38, // #38BDF8
	text:        0x492F08, // #082F49
	gold:        0xC78402, // #0284C7 Electric Sky
	amber:       0xD4B606, // #06B6D4 Cyan
	jade:        0x699605, // #059669 Emerald
	vermilion:   0x481DE1, // #E11D48
	muted:       0xA16903, // #0369A1
	pill:        0xFDE6BA, // #BAE6FD
	goldDim:     0xFCDB7D, // #7DD3FC
	btnBg:       0xFFF9F0, // #F0F9FF
	btnHover:    0xFEEFE0, // #E0EFFE
	btnBorder:   0xF8BD38, // #38BDF8
	btnText:     0x492F08, // #082F49
	navActiveBg: 0xFDE6BA, // #BAE6FD
	navActiveBd: 0xC78402, // #0284C7

	aBg:        0xFFE0F2FE,
	aCard:      0xFFFFFFFF,
	aBorder:    0xFF38BDF8,
	aText:      0xFF082F49,
	aMuted:     0xFF0369A1,
	aAccent:    0xFF0284C7,
	aSwitchOn:  0xFF10B981,
	aSwitchOff: 0xFFBAE6FD,
	aKeycap:    0xFFF0F9FF,
}

// 3A. Pastel Matcha Latte (Light) — Warm Butter Cream (#EDE4D8), Milk Foam (#FFFDF9) & Matcha (#2D6A4F) + Peach (#E07A5F)
var palMatchaLight = palette{
	bg:          0xD8E4ED, // #EDE4D8
	card:        0xF9FDFF, // #FFFDF9
	border:      0xAEC3D5, // #D5C3AE
	text:        0x242E3D, // #3D2E24 Rich Espresso Brown
	gold:        0x4F6A2D, // #2D6A4F Deep Matcha
	amber:       0x5F7AE0, // #E07A5F Terra Peach
	jade:        0x6C9140, // #40916C Matcha Leaf
	vermilion:   0x5F7AE0, // #E07A5F Terra Peach
	muted:       0x5D6B7D, // #7D6B5D Warm Mocha
	pill:        0xB6C9DC, // #DCC9B6 Warm Biscuit
	goldDim:     0xAEC3D5, // #D5C3AE
	btnBg:       0xE0ECF5, // #F5ECE0
	btnHover:    0xCBDCE8, // #E8DCCB
	btnBorder:   0x96ADC2, // #C2AD96
	btnText:     0x242E3D, // #3D2E24
	navActiveBg: 0xDCF3D8, // #D8F3DC
	navActiveBd: 0x4F6A2D, // #2D6A4F

	aBg:        0xFFEDE4D8,
	aCard:      0xFFFFFDF9,
	aBorder:    0xFFD5C3AE,
	aText:      0xFF3D2E24,
	aMuted:     0xFF7D6B5D,
	aAccent:    0xFF2D6A4F,
	aSwitchOn:  0xFF40916C,
	aSwitchOff: 0xFFDCC9B6,
	aKeycap:    0xFFF5ECE0,
}

// 3B. Pastel Matcha Latte (Dark Espresso) — Roasted Forest (#121A16), Matcha Card (#1B2822) & Creamy Matcha (#74C69D)
var palMatchaDark = palette{
	bg:          0x161A12, // #121A16
	card:        0x22281B, // #1B2822
	border:      0x39442D, // #2D4439
	text:        0xE1ECF4, // #F4ECE1 Warm Oat Cream
	gold:        0x9DC674, // #74C69D Creamy Matcha
	amber:       0x61A2F4, // #F4A261 Warm Peach
	jade:        0x88B752, // #52B788 Matcha Leaf
	vermilion:   0x61A2F4, // #F4A261
	muted:       0xA6B895, // #95B8A6 Soft Sage
	pill:        0x333D2A, // #2A3D33
	goldDim:     0x39442D, // #2D4439
	btnBg:       0x2C3524, // #24352C
	btnHover:    0x39442D, // #2D4439
	btnBorder:   0x4A593B, // #3B594A
	btnText:     0xE1ECF4, // #F4ECE1
	navActiveBg: 0x2E3F24, // #243F2E
	navActiveBd: 0x9DC674, // #74C69D

	aBg:        0xFF121A16,
	aCard:      0xFF1B2822,
	aBorder:    0xFF2D4439,
	aText:      0xFFF4ECE1,
	aMuted:     0xFF95B8A6,
	aAccent:    0xFF74C69D,
	aSwitchOn:  0xFF52B788,
	aSwitchOff: 0xFF2A3D33,
	aKeycap:    0xFF24352C,
}

// 4A. Material 3 Expressive (Dark) — Pixel Slate-Teal (#0E171D) & Expressive Aqua Pill (#2DD4BF)
var palMaterialDark = palette{
	bg:          0x1D170E, // #0E171D
	card:        0x302517, // #172530
	border:      0x56442A, // #2A4456
	text:        0xF1F4E6, // #E6F4F1
	gold:        0xBFD42D, // #2DD4BF Expressive Teal
	amber:       0xD4EA5E, // #5EEAD4
	jade:        0xBFD42D, // #2DD4BF
	vermilion:   0x8571FB, // #FB7185
	muted:       0xB8A888, // #88A8B8
	pill:        0x4A3B24, // #243B4A
	goldDim:     0x45351E, // #1E3545
	btnBg:       0x3E311E, // #1E313E
	btnHover:    0x524128, // #284152
	btnBorder:   0x6D5636, // #36566D
	btnText:     0xF1F4E6, // #E6F4F1
	navActiveBg: 0x3E3C13, // #133C3E
	navActiveBd: 0xBFD42D, // #2DD4BF

	aBg:        0xFF0E171D,
	aCard:      0xFF172530,
	aBorder:    0xFF2A4456,
	aText:      0xFFE6F4F1,
	aMuted:     0xFF88A8B8,
	aAccent:    0xFF2DD4BF,
	aSwitchOn:  0xFF2DD4BF,
	aSwitchOff: 0xFF243B4A,
	aKeycap:    0xFF1E313E,
}

// 4B. Material 3 Expressive (Light) — Pixel Mint-Teal Daylight (#E6F7F5), Tonal Aqua (#CCFBF1) & Deep Teal (#0F766E)
var palMaterialLight = palette{
	bg:          0xF5F7E6, // #E6F7F5
	card:        0xFFFFFF, // #FFFFFF
	border:      0xD8E699, // #99E6D8
	text:        0x282B0D, // #0D2B28
	gold:        0x6E760F, // #0F766E Deep Teal
	amber:       0xA6B814, // #14B8A6
	jade:        0x88940D, // #0D9488 Teal Switch
	vermilion:   0x481DE1, // #E11D48
	muted:       0x666B3D, // #3D6B66
	pill:        0xDFE8B2, // #B2E8DF
	goldDim:     0xD8E699, // #99E6D8
	btnBg:       0xF1FBCC, // #CCFBF1
	btnHover:    0xE4F699, // #99F6E4
	btnBorder:   0xC8DC80, // #80DCC8
	btnText:     0x282B0D, // #0D2B28
	navActiveBg: 0xF1FBCC, // #CCFBF1
	navActiveBd: 0x6E760F, // #0F766E

	aBg:        0xFFE6F7F5,
	aCard:      0xFFFFFFFF,
	aBorder:    0xFF99E6D8,
	aText:      0xFF0D2B28,
	aMuted:     0xFF3D6B66,
	aAccent:    0xFF0F766E,
	aSwitchOn:  0xFF0D9488,
	aSwitchOff: 0xFFB2E8DF,
	aKeycap:    0xFFCCFBF1,
}

var (
	palIosPure      = palIosLight
	palCyberNeon    = palCyberDark
	palMatchaLatte  = palMatchaLight
	palMaterialTeal = palMaterialDark
)

const (
	themeAuto = iota
	themeDark
	themeLight
)

const (
	styleBasic = iota
	styleJade
	styleIosPure
	styleCyberNeon
	styleMatchaLatte
	styleMaterialTeal
)

var (
	winDarkCached bool
	winDarkAt     time.Time
)

func windowsDark() bool {
	if !winDarkAt.IsZero() && time.Since(winDarkAt) < 2*time.Second {
		return winDarkCached
	}
	winDarkAt = time.Now()
	k, err := registry.OpenKey(registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`, registry.READ)
	if err != nil {
		winDarkCached = true
		return true
	}
	defer k.Close()
	v, _, err := k.GetIntegerValue("AppsUseLightTheme")
	winDarkCached = err == nil && v == 0
	return winDarkCached
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

func curPalette() palette {
	dark := isDark()
	switch cfg.Style {
	case styleBasic:
		if dark {
			return palBasicDark
		}
		return palBasicLight
	case styleIosPure:
		if dark {
			return palIosDark
		}
		return palIosLight
	case styleCyberNeon:
		if dark {
			return palCyberDark
		}
		return palCyberLight
	case styleMatchaLatte:
		if dark {
			return palMatchaDark
		}
		return palMatchaLight
	case styleMaterialTeal:
		if dark {
			return palMaterialDark
		}
		return palMaterialLight
	}
	if dark {
		return palDark
	}
	return palLight
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
	winDarkAt = time.Time{}
	p := curPalette()
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

	// Cơ bản lets the user recolor the accent (Giao diện → Màu nền).
	if cfg.Style == styleBasic && cfg.Accent >= 0 {
		rgb := cfg.Accent
		c := uintptr(rgb&0xFF)<<16 | uintptr(rgb&0xFF00) | uintptr((rgb>>16)&0xFF)
		colGold, colAmber, colJade = c, c, c
		argbAccent = 0xFF000000 | uint32(rgb)
		argbSwitchOn = argbAccent
	}

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

	refreshTrayIcons()
	hudUpdate()
}
