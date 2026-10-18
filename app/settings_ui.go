package main

// Settings window: calm jade accents, native Vietnamese typography,
// a five-tab vector sidebar, and compact settings groups.

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sync/atomic"
	"time"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

var comdlg32 = windows.NewLazySystemDLL("comdlg32.dll")

var (
	pSendMessage     = user32.NewProc("SendMessageW")
	pShellExecute    = sh32.NewProc("ShellExecuteW")
	pShowWindow      = user32.NewProc("ShowWindow")
	pSetWndText      = user32.NewProc("SetWindowTextW")
	pGetWndText      = user32.NewProc("GetWindowTextW")
	pGetDpiForWindow = user32.NewProc("GetDpiForWindow")
	pGetDpiForSystem = user32.NewProc("GetDpiForSystem")
	pGetClientRect   = user32.NewProc("GetClientRect")
	pMapWindowPoints = user32.NewProc("MapWindowPoints")
	pIsZoomed        = user32.NewProc("IsZoomed")
	pIsIconic        = user32.NewProc("IsIconic")
)

const (
	WS_CHILD           = 0x40000000
	WS_VISIBLE         = 0x10000000
	WS_OVERLAPPED      = 0x00000000
	WS_CAPTION         = 0x00C00000
	WS_SYSMENU         = 0x00080000
	WS_TABSTOP         = 0x00010000
	WS_THICKFRAME      = 0x00040000
	WS_MINIMIZEBOX     = 0x00020000
	WS_MAXIMIZEBOX     = 0x00010000
	BS_AUTOCHECKBOX    = 0x0003
	CBS_DROPDOWNLIST   = 0x0003
	BN_CLICKED         = 0
	CBN_SELCHANGE      = 1
	BM_GETCHECK        = 0x00F0
	BM_SETCHECK        = 0x00F1
	BST_CHECKED        = 1
	CB_ADDSTRING       = 0x0143
	CB_SETCURSEL       = 0x014E
	CB_GETCURSEL       = 0x0147
	WM_SETFONT         = 0x0030
	WM_CREATE          = 0x0001
	WM_CLOSE           = 0x0010
	WM_SIZE            = 0x0005
	WM_GETMINMAXINFO   = 0x0024
	WM_DPICHANGED      = 0x02E0
	WM_ERASEBKGND      = 0x0014
	WM_CTLCOLORSTATIC  = 0x0138
	WM_CTLCOLORBTN     = 0x0135
	WM_CTLCOLORLISTBOX = 0x0134
	WM_CTLCOLOREDIT    = 0x0133
	WM_DRAWITEM        = 0x002B
	SW_HIDE            = 0
	BS_OWNERDRAW       = 0x000B
	ODT_BUTTON         = 4
	ODS_SELECTED       = 0x0001
	SW_SHOW            = 5
	SW_RESTORE         = 9
	SW_MINIMIZE        = 6
	SW_MAXIMIZE        = 3
	SWP_NOMOVE         = 0x0002
	WS_CLIPCHILDREN    = 0x02000000
	SRCCOPY            = 0x00CC0020
	RDW_INVALIDATE     = 0x0001
	RDW_ERASE          = 0x0004
	RDW_ALLCHILDREN    = 0x0080
	RDW_UPDATENOW      = 0x0100
)

var (
	rethemed      bool
	gSettingsHwnd uintptr
	gSettingsIcon uintptr // cached — loadAppIcon creates a new HICON each call
	// Cached once: the settings window is created/destroyed repeatedly and
	// every NewCallback call permanently allocates a runtime callback thunk.
	settingsProcCb uintptr

	// appBuildStamp is the running exe's modification time — the "last
	// update" shown in the footer. The updater replaces the exe in place,
	// so its mtime tracks the installed build.
	appBuildStamp = func() string {
		exe, err := os.Executable()
		if err == nil {
			if fi, err := os.Stat(exe); err == nil {
				return fi.ModTime().Format("02/01/2006 15:04")
			}
		}
		return "?"
	}()

	ctlHnd        = map[int]uintptr{}
	allCtls       []uintptr
	goldCtls      = map[uintptr]bool{}
	mutedCtls     = map[uintptr]bool{}
	bgCtls        = map[uintptr]bool{}
	swState       = map[int]bool{}
	swText        = map[int]string{}
	btnText       = map[int]string{}
	btnIDs        = map[int]bool{}
	isCheckbox    = map[int]bool{}
	isRadio       = map[int]bool{}
	radioSubtitle = map[int]string{}
	ctlTab        = map[uintptr]int{}
	curTab        = -1
	gActiveTab    int
	gHoveredTab   = -1
	gCaptionHover = -1
	gCaptionDown  = -1

	fontNormal  uintptr
	fontBold    uintptr
	fontTitle   uintptr
	fontSection uintptr
	fontHint    uintptr
	fontTagline uintptr

	gDpi  int32 = 96
	railW int32
)

// dp converts a 96-dpi design pixel to the current monitor's DPI.
func dp(v int32) int32 { return int32(int64(v) * int64(gDpi) / 96) }

var tabItems = []struct {
	name   string
	desc   string
	nameEn string
	descEn string
}{
	{"Tổng quan", "Trạng thái và tùy chọn chung", "Overview", "Status and general options"},
	{"Gõ tiếng Việt", "Telex, VNI và nâng cao", "Typing", "Telex, VNI and macros"},
	{"Phím tắt", "Tùy chỉnh tổ hợp phím", "Shortcuts", "Customize toggle & hotkeys"},
	{"Hệ thống", "Hệ thống và ứng dụng loại trừ", "System", "System & app exclusions"},
	{"Giao diện", "Chủ đề và hiển thị", "Appearance", "Color mode & theme styles"},
	{"Giới thiệu", "Thông tin ứng dụng", "About", "Application information"},
}

func tabName(i int) string {
	return tr(tabItems[i].name, tabItems[i].nameEn)
}

func tabDesc(i int) string {
	return tr(tabItems[i].desc, tabItems[i].descEn)
}

// ids of per-tab STATICs
var (
	stIMSection, stLbIM, stLbCharset                                                             int
	stSwitchSection, stLbHotkey, stHotkeyHint, stFKeySection                                     int
	stMacroSection, stMacroHint1, stMacroHint2                                                   int
	stThemeModeSection, stStyleSection, stSysSection, stLbTheme, stExcludeSection, stExcludeHint int
	stTraySection, stTrayHint                                                                 int
	stAccentLabel                                                                             int
)

// control IDs
const (
	cIMCombo = 100 + iota
	cModern
	cFreeMark
	cSpell
	cRestore
	cMacro
	cMacroAlways
	cMacroEdit
	cSkipLayout
	cAutoCap
	cClipboard
	cStartup
	cAdmin
	cShowDlg
	cSound
	cShowHud
	cTheme
	cThemeAuto
	cThemeLight
	cThemeDark
	cToggleCombo
	cExcludeEdit
	cCharset
	cModCtrl
	cModShift
	cModAlt
	cModWin
	cHotkeyBox
	cUseCtrlShift
	cF1
	cF2
	cF5
	cF9
	cF12
	cExcludeBrowse
	// Mockup Tab 0 controls:
	cRadioTelex
	cRadioTelexSimple
	cRadioVni
	cRadioViqr
	cLangBtn
	cUpdateBtn
	cRememberAppMode
	cThemeJade
	cThemeIosPure
	cThemeCyberNeon
	cThemeMatchaLatte
	cThemeMaterialTeal
	cThemeBasic
	cTrayV
	cTrayVBg
	cTrayE
	cTrayEBg
	cTrayReset
	cAccent
	cAccentText
	cAccentReset
)

var imList = []int{ImTelex, ImTelexSimple, ImVni, ImViqr, ImTelexVni, ImMsVi, ImUsrKeymap}

type ctlDef struct {
	id         int
	class      string
	style      uint32
	text       string
	x, y, w, h int32
}

func mkCtl(parent uintptr, d ctlDef) uintptr {
	h, _, _ := pCreateWindowEx.Call(0,
		uintptr(unsafe.Pointer(utf16ptr(d.class))),
		uintptr(unsafe.Pointer(utf16ptr(d.text))),
		uintptr(d.style|WS_CHILD|WS_VISIBLE),
		uintptr(d.x), uintptr(d.y), uintptr(d.w), uintptr(d.h),
		parent, uintptr(d.id), 0, 0)
	if d.id > 0 {
		ctlHnd[d.id] = h
	}
	allCtls = append(allCtls, h)
	ctlTab[h] = curTab
	if d.class != "STATIC" && d.style&BS_OWNERDRAW == 0 && isDark() {
		darkCtl(h)
	}
	return h
}

var stSeq = 299

func statID() int { stSeq++; return stSeq }

var gbH uintptr

func addLabel(text string) int {
	id := statID()
	mkCtl(gbH, ctlDef{id, "STATIC", 0, text, 0, 0, 10, 10})
	return id
}

func addSection(text string) int {
	id := statID()
	h := mkCtl(gbH, ctlDef{id, "STATIC", 0x0080, text, 0, 0, 10, 10})
	pSendMessage.Call(h, WM_SETFONT, fontSection, 1)
	goldCtls[h] = true
	return id
}

func addHint(text string) int {
	id := statID()
	h := mkCtl(gbH, ctlDef{id, "STATIC", 0, text, 0, 0, 10, 10})
	mutedCtls[h] = true
	return id
}

func addSw(id int, text string) {
	swText[id] = text
	swIDs[id] = true
	mkCtl(gbH, ctlDef{id, "BUTTON", BS_OWNERDRAW | WS_TABSTOP, "", 0, 0, 10, 10})
}

func addBtn(id int, text string) {
	btnIDs[id] = true
	btnText[id] = text
	mkCtl(gbH, ctlDef{id, "BUTTON", BS_OWNERDRAW | WS_TABSTOP, text, 0, 0, 10, 10})
}

func addCheckbox(id int, text string) {
	isCheckbox[id] = true
	btnIDs[id] = true
	btnText[id] = text
	mkCtl(gbH, ctlDef{id, "BUTTON", BS_OWNERDRAW | WS_TABSTOP, text, 0, 0, 10, 10})
}

func addRadio(id int, title, desc string) {
	isRadio[id] = true
	btnIDs[id] = true
	btnText[id] = title
	radioSubtitle[id] = desc
	mkCtl(gbH, ctlDef{id, "BUTTON", BS_OWNERDRAW | WS_TABSTOP, title, 0, 0, 10, 10})
}

type drawItemStruct struct {
	ctlType    uint32
	ctlID      uint32
	itemID     uint32
	itemAction uint32
	itemState  uint32
	hwndItem   uintptr
	hDC        uintptr
	rcItem     rect
	itemData   uintptr
}

func diamond(dc uintptr, x, y, r int32, col uintptr) {
	br, _, _ := pCreateSolidBrush.Call(col)
	oldB, _, _ := pSelectObject.Call(dc, br)
	pp, _, _ := pCreatePen.Call(PS_SOLID, 1, col)
	oldP, _, _ := pSelectObject.Call(dc, pp)
	pts := []point{{x, y - r}, {x + r, y}, {x, y + r}, {x - r, y}}
	pPolygon.Call(dc, uintptr(unsafe.Pointer(&pts[0])), uintptr(len(pts)))
	pSelectObject.Call(dc, oldP)
	pDeleteObject.Call(pp)
	pSelectObject.Call(dc, oldB)
	pDeleteObject.Call(br)
}

// drawLogo paints the stylized mint jade dual-leaf brand logo with gold spark.
func drawLogo(dc uintptr, cx, cy int32) {
	lbr, _, _ := pCreateSolidBrush.Call(colJade)
	oldB, _, _ := pSelectObject.Call(dc, lbr)
	nullPen, _, _ := pGetStockObject.Call(NULL_PEN)
	oldP, _, _ := pSelectObject.Call(dc, nullPen)

	// Left taller crescent leaf curving gracefully upward-right
	pts1 := []point{
		{cx - dp(4), cy + dp(7)},
		{cx - dp(9), cy + dp(1)},
		{cx - dp(8), cy - dp(5)},
		{cx - dp(3), cy - dp(10)},
		{cx - dp(1), cy - dp(3)},
		{cx - dp(3), cy + dp(3)},
	}
	pPolygon.Call(dc, uintptr(unsafe.Pointer(&pts1[0])), uintptr(len(pts1)))

	// Right shorter rounded leaf curving upward-left
	pts2 := []point{
		{cx + dp(2), cy + dp(6)},
		{cx + dp(7), cy + dp(2)},
		{cx + dp(8), cy - dp(3)},
		{cx + dp(4), cy - dp(6)},
		{cx + dp(2), cy - dp(1)},
		{cx + dp(1), cy + dp(4)},
	}
	pPolygon.Call(dc, uintptr(unsafe.Pointer(&pts2[0])), uintptr(len(pts2)))

	pSelectObject.Call(dc, oldP)
	pSelectObject.Call(dc, oldB)
	pDeleteObject.Call(lbr)

	diamond(dc, cx, cy-dp(5), dp(2), colGold)
}

const (
	captionMinimize = iota
	captionMaximize
	captionClose
)

// One geometry source for painting and hit testing. Each control has a full
// 46x42 dpi-scaled target; glyphs stay centered in that target.
func captionRect(winW int32, id int) rect {
	left := winW - dp(int32((3-id)*46))
	return rect{left, 0, left + dp(46), dp(42)}
}

func captionBarRect(winW int32) rect {
	return rect{captionRect(winW, captionMinimize).left, 0, winW, dp(42)}
}

func captionAt(winW, x, y int32) int {
	if y < 0 || y >= dp(42) {
		return -1
	}
	for id := captionMinimize; id <= captionClose; id++ {
		r := captionRect(winW, id)
		if x >= r.left && x < r.right {
			return id
		}
	}
	return -1
}

func drawWindowControls(dc uintptr, hwnd uintptr, winW int32) {
	for id := captionMinimize; id <= captionClose; id++ {
		r := captionRect(winW, id)
		hovered := gCaptionHover == id
		pressed := gCaptionDown == id && hovered
		if hovered || pressed {
			fill := colCard
			if pressed {
				fill = colPill
			}
			if id == captionClose {
				fill = 0x002311E8 // Windows close red (#E81123)
				if pressed {
					fill = 0x00100CAD
				}
			}
			br, _, _ := pCreateSolidBrush.Call(fill)
			pFillRect.Call(dc, uintptr(unsafe.Pointer(&r)), br)
			pDeleteObject.Call(br)
		}

		ink := colText
		if id == captionClose && hovered {
			ink = 0xFFFFFF
		}
		pen, _, _ := pCreatePen.Call(PS_SOLID, uintptr(max(int32(1), dp(2))), ink)
		oldPen, _, _ := pSelectObject.Call(dc, pen)
		cx, cy := (r.left+r.right)/2, dp(21)
		line := func(x1, y1, x2, y2 int32) {
			pMoveToEx.Call(dc, uintptr(x1), uintptr(y1), 0)
			pLineTo.Call(dc, uintptr(x2), uintptr(y2))
		}
		switch id {
		case captionMinimize:
			line(cx-dp(6), cy+dp(4), cx+dp(7), cy+dp(4))
		case captionMaximize:
			nullBrush, _, _ := pGetStockObject.Call(NULL_BRUSH)
			oldBrush, _, _ := pSelectObject.Call(dc, nullBrush)
			if zoomed, _, _ := pIsZoomed.Call(hwnd); zoomed != 0 {
				pRectangle.Call(dc, uintptr(cx-dp(6)), uintptr(cy-dp(2)), uintptr(cx+dp(3)), uintptr(cy+dp(6)))
				line(cx-dp(3), cy-dp(5), cx+dp(6), cy-dp(5))
				line(cx+dp(5), cy-dp(5), cx+dp(5), cy+dp(3))
			} else {
				pRectangle.Call(dc, uintptr(cx-dp(5)), uintptr(cy-dp(5)), uintptr(cx+dp(6)), uintptr(cy+dp(5)))
			}
			pSelectObject.Call(dc, oldBrush)
		case captionClose:
			line(cx-dp(5), cy-dp(5), cx+dp(6), cy+dp(6))
			line(cx+dp(5), cy-dp(5), cx-dp(6), cy+dp(6))
		}
		pSelectObject.Call(dc, oldPen)
		pDeleteObject.Call(pen)
	}
}

func toggleSettingsMaximize(hwnd uintptr) {
	if zoomed, _, _ := pIsZoomed.Call(hwnd); zoomed != 0 {
		pShowWindow.Call(hwnd, SW_RESTORE)
	} else {
		pShowWindow.Call(hwnd, SW_MAXIMIZE)
	}
}

func updateSettingsRegion(hwnd uintptr) {
	applyWindowChrome(hwnd)
}

func settingsHasRoundCorners() bool {
	if !gDwmRoundCorners {
		return false
	}
	if gSettingsHwnd != 0 {
		if zoomed, _, _ := pIsZoomed.Call(gSettingsHwnd); zoomed != 0 {
			return false
		}
	}
	return true
}

// Top titlebar static branding (gradient bar, brand mark, title, and subtitle).
func drawTopHeaderBase(dc uintptr, winW int32) {
	gfx := NewGfx(dc)
	if gfx != nil {
		// Subtle translucent specular bar along top edge
		topR := float32(0)
		if settingsHasRoundCorners() {
			topR = float32(dp(8))
		}
		gfx.FillRoundedRectGradient(0, 0, float32(winW), float32(dp(44)), topR, HexAlphaARGB(0x18, 0xFFFFFF), HexAlphaARGB(0x04, 0xFFFFFF))
		gfx.DrawLine(0, float32(dp(44)), float32(winW), float32(dp(44)), HexAlphaARGB(0x18, 0xFFFFFF), 1.0)
		drawMinimalBrandMark(gfx, float32(dp(16)), float32(dp(6)), float32(dp(32)))
		gfx.Destroy()
	}

	pSetBkMode.Call(dc, TRANSPARENT)
	pSetTextColor.Call(dc, colText)
	oldF, _, _ := pSelectObject.Call(dc, fontTitle)
	trc := rect{dp(60), dp(5), dp(320), dp(27)}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr("Tui Gõ"))),
		^uintptr(0), uintptr(unsafe.Pointer(&trc)),
		DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

	pSetTextColor.Call(dc, colMuted)
	pSelectObject.Call(dc, fontHint)
	sbc := rect{dp(60), dp(25), dp(340), dp(41)}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr(tr("Gõ tiếng Việt tối giản & hiện đại", "Minimalist & modern Vietnamese input")))),
		^uintptr(0), uintptr(unsafe.Pointer(&sbc)),
		DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

	pSelectObject.Call(dc, oldF)
}

// Top titlebar: minimalist brand, subtitle, and system-style window controls.
func drawTopHeader(dc uintptr, hwnd uintptr, winW int32) {
	drawTopHeaderBase(dc, winW)
	drawWindowControls(dc, hwnd, winW)
}

// Sidebar navigation renderer with zero-aliased vector icons, style-aware squircle badges, and liquid glass pills
func drawNav(dc uintptr) {
	drawNavClipped(dc, rect{0, 0, railW, dp(450)})
}

func drawNavClipped(dc uintptr, clip rect) {
	var gfx *Gfx
	defer func() {
		if gfx != nil {
			gfx.Destroy()
		}
	}()

	dark := isDark()
	iosBadgeCols := [6]uint32{0xFF007AFF, 0xFF34C759, 0xFFFF9500, 0xFFAF52DE, 0xFF64748B, 0xFF5856D6}
	matchaLightCols := [6]uint32{0xFFA8D5BA, 0xFFF7C59F, 0xFFFDE68A, 0xFFB7E4C7, 0xFFCBD5E1, 0xFFF4A261}

	for i := 0; i < len(tabItems); i++ {
		if !rectsIntersect(clip, navDirtyRect(i)) {
			continue
		}
		if gfx == nil {
			gfx = NewGfx(dc)
		}
		r := navRect(i)
		active := (i == gActiveTab)
		hover := (i == gHoveredTab)

		x, y, w, h := float32(r.left), float32(r.top), float32(r.right-r.left), float32(r.bottom-r.top)
		radius := float32(dp(9))

		if active {
			if gfx != nil {
				gfx.DrawGlassPill(x, y, w, h, radius, dark, true, false)
				// Left indicator bar in active style accent; on a solid-accent
				// pill (Cơ bản) flip it to the on-accent ink.
				barCol := argbAccent
				if cfg.Style == styleBasic {
					barCol = accentOnArgb()
				}
				gfx.FillRoundedRect(x+float32(dp(3)), y+float32(dp(9)), float32(dp(4)), h-float32(dp(18)), 2.0, barCol)
			}
		} else if hover {
			if gfx != nil {
				gfx.DrawGlassPill(x, y, w, h, radius, dark, false, true)
			}
		}

		// Style-aware icon squircle badge (28x28dp) matching the HTML demo
		bx := float32(r.left + dp(11))
		by := float32(r.top+(r.bottom-r.top)/2) - float32(dp(14))
		bsz := float32(dp(28))
		br := float32(dp(7))
		iconCol := argbMuted
		if active {
			iconCol = argbAccent
		}

		if gfx != nil {
			switch cfg.Style {
			case styleIosPure:
				// Apple iOS 18 Settings colorful filled squircles
				bCol := iosBadgeCols[i%len(iosBadgeCols)]
				if !active && !hover {
					bCol = HexAlphaARGB(0xD8, bCol&0x00FFFFFF)
				}
				gfx.FillRoundedRect(bx, by, bsz, bsz, br, bCol)
				gfx.DrawRoundedRect(bx, by, bsz, bsz, br, HexAlphaARGB(0x45, 0xFFFFFF), 1.0)
				iconCol = 0xFFFFFFFF

			case styleCyberNeon:
				// Cyber Neon glowing frame badge
				br = float32(dp(6))
				fillCol := uint32(0xFF0B111E)
				rimCol := uint32(0xFF0EA5E9)
				if !dark {
					fillCol = 0xFFE0F2FE
					rimCol = 0xFF0284C7
				}
				if active {
					if dark {
						fillCol = 0xFF082F49
						rimCol = 0xFF00F2FE
						iconCol = 0xFF00F2FE
					} else {
						fillCol = 0xFF0284C7
						rimCol = 0xFF0369A1
						iconCol = 0xFFFFFFFF
					}
				} else if dark {
					iconCol = 0xFF38BDF8
				} else {
					iconCol = 0xFF0284C7
				}
				gfx.FillRoundedRect(bx, by, bsz, bsz, br, fillCol)
				gfx.DrawRoundedRect(bx, by, bsz, bsz, br, rimCol, 1.2)

			case styleMatchaLatte:
				// Warm Matcha / Peach / Caramel soft squircle badge
				br = float32(dp(9))
				if dark {
					if active {
						gfx.FillRoundedRect(bx, by, bsz, bsz, br, 0xFF40916C)
						iconCol = 0xFFF4ECE1
					} else {
						gfx.FillRoundedRect(bx, by, bsz, bsz, br, 0xFF24352C)
						gfx.DrawRoundedRect(bx, by, bsz, bsz, br, 0xFF385446, 1.0)
						iconCol = 0xFF74C69D
					}
				} else {
					if active {
						gfx.FillRoundedRect(bx, by, bsz, bsz, br, 0xFF2D6A4F)
						iconCol = 0xFFFFFDF9
					} else {
						gfx.FillRoundedRect(bx, by, bsz, bsz, br, matchaLightCols[i%len(matchaLightCols)])
						gfx.DrawRoundedRect(bx, by, bsz, bsz, br, 0xFFD5C3AE, 1.0)
						iconCol = 0xFF3D2E24
					}
				}

			case styleMaterialTeal:
				// Material 3 Expressive circular/capsule tonal container
				br = bsz / 2.0
				if dark {
					if active {
						gfx.FillRoundedRect(bx, by, bsz, bsz, br, 0xFF2DD4BF)
						iconCol = 0xFF0E171D
					} else {
						gfx.FillRoundedRect(bx, by, bsz, bsz, br, 0xFF1E3545)
						iconCol = 0xFF5EEAD4
					}
				} else {
					if active {
						gfx.FillRoundedRect(bx, by, bsz, bsz, br, 0xFF0D9488)
						iconCol = 0xFFFFFFFF
					} else {
						gfx.FillRoundedRect(bx, by, bsz, bsz, br, 0xFFCCFBF1)
						iconCol = 0xFF0F766E
					}
				}

			default:
				if cfg.Style == styleBasic {
					// Cơ bản: flat badge — solid accent when active, card tint otherwise
					br = float32(dp(6))
					if active {
						gfx.FillRoundedRect(bx, by, bsz, bsz, br, argbAccent)
						iconCol = accentOnArgb()
					} else {
						gfx.FillRoundedRect(bx, by, bsz, bsz, br, argbKeycap)
						gfx.DrawRoundedRect(bx, by, bsz, bsz, br, argbBorder, 1.0)
						iconCol = argbMuted
					}
					break
				}
				// Default Ngọc Bích Liquid Glass squircle
				if active {
					gfx.FillRoundedRectGradient(bx, by, bsz, bsz, br, argbAccent, HexAlphaARGB(0xD8, argbSwitchOn&0x00FFFFFF))
					if dark {
						iconCol = 0xFF0F1115
					} else {
						iconCol = 0xFFFFFFFF
					}
				} else {
					bgTint := HexAlphaARGB(0x18, argbAccent&0x00FFFFFF)
					gfx.FillRoundedRect(bx, by, bsz, bsz, br, bgTint)
					gfx.DrawRoundedRect(bx, by, bsz, bsz, br, HexAlphaARGB(0x40, argbAccent&0x00FFFFFF), 1.0)
				}
			}

			// Icon kind is positional (3=sliders, 4=gear); sidebar order now
			// puts System (gear) at index 3 and Appearance (sliders) at 4.
			iconKind := i
			switch i {
			case 3:
				iconKind = 4
			case 4:
				iconKind = 3
			}
			drawSmoothVectorIcon(gfx, bx+bsz/2.0, by+bsz/2.0, float32(dp(15)), iconKind, iconCol)
		}

		pSetBkMode.Call(dc, TRANSPARENT)
		titleColor := colText
		descColor := colMuted
		if active {
			titleColor = colGold
			if cfg.Style == styleBasic {
				// Pill is filled solid accent — draw ink on top of it
				titleColor = accentOnCol()
				descColor = accentOnCol()
			}
		}
		pSetTextColor.Call(dc, titleColor)
		oldF, _, _ := pSelectObject.Call(dc, fontBold)
		trc := rect{r.left + dp(46), r.top + dp(6), r.right - dp(4), r.top + dp(24)}
		pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr(tabName(i)))),
			^uintptr(0), uintptr(unsafe.Pointer(&trc)),
			DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

		pSetTextColor.Call(dc, descColor)
		pSelectObject.Call(dc, fontHint)
		drc := rect{r.left + dp(46), r.top + dp(24), r.right - dp(4), r.bottom - dp(4)}
		pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr(tabDesc(i)))),
			^uintptr(0), uintptr(unsafe.Pointer(&drc)),
			DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

		pSelectObject.Call(dc, oldF)
	}
}

// drawTitleBlock renders the shared tab header with a consistent type hierarchy.
func drawTitleBlock(dc uintptr, cx, cw int32, title, desc string) {
	pSetBkMode.Call(dc, TRANSPARENT)
	pSetTextColor.Call(dc, colText)
	oldF, _, _ := pSelectObject.Call(dc, fontTitle)
	t1 := rect{cx, dp(54), cx + cw, dp(78)}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr(title))),
		^uintptr(0), uintptr(unsafe.Pointer(&t1)),
		DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

	pSetTextColor.Call(dc, colMuted)
	pSelectObject.Call(dc, fontHint)
	s1 := rect{cx, dp(78), cx + cw, dp(98)}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr(desc))),
		^uintptr(0), uintptr(unsafe.Pointer(&s1)),
		DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)
	pSelectObject.Call(dc, oldF)

}

func drawSettingsCard(dc uintptr, r rect) {
	gfx := NewGfx(dc)
	if gfx == nil {
		return
	}
	defer gfx.Destroy()
	gfx.DrawGlassCard(float32(r.left), float32(r.top), float32(r.right-r.left), float32(r.bottom-r.top), float32(dp(10)), isDark(), false)
}

func drawSettingsTab(dc uintptr, tab int, cx, ct, cw int32) {
	drawTitleBlock(dc, cx, cw, tabName(tab), tabDesc(tab))
	switch tab {
	case 1: // Input method and macros
		drawSettingsCard(dc, rect{cx, ct, cx + cw, ct + dp(100)})
		drawSettingsCard(dc, rect{cx, ct + dp(112), cx + cw, ct + dp(318)})
	case 2: // Hotkey and function keys
		drawSettingsCard(dc, rect{cx, ct, cx + cw, ct + dp(180)})
		drawSettingsCard(dc, rect{cx, ct + dp(192), cx + cw, ct + dp(388)})
		// The native EDIT has no border; this frame gives it the same shape and
		// tint as the selected modifier keys.
		field := rect{cx + dp(70), ct + dp(97), cx + dp(142), ct + dp(125)}
		gfx := NewGfx(dc)
		if gfx != nil {
			fx, fy, fw, fh := float32(field.left), float32(field.top), float32(field.right-field.left), float32(field.bottom-field.top)
			gfx.FillRoundedRect(fx, fy, fw, fh, float32(dp(6)), HexAlphaARGB(0x30, argbAccent&0x00FFFFFF))
			gfx.DrawRoundedRect(fx, fy, fw, fh, float32(dp(6)), argbAccent, 1.2)
			gfx.Destroy()
		}
	case 3: // System & app exclusions
		drawSettingsCard(dc, rect{cx, ct, cx + cw, ct + dp(214)})
		drawSettingsCard(dc, rect{cx, ct + dp(226), cx + cw, ct + dp(374)})
	case 4: // Appearance: color mode & theme style + tray icon customizer
		drawSettingsCard(dc, rect{cx, ct, cx + cw, ct + dp(94)})
		// Style card grows an extra row for the accent pickers on Cơ bản
		card2Bot, card3Top := dp(224), dp(236)
		if cfg.Style == styleBasic {
			card2Bot, card3Top = dp(272), dp(284)
		}
		drawSettingsCard(dc, rect{cx, ct + dp(106), cx + cw, ct + card2Bot})
		drawSettingsCard(dc, rect{cx, ct + card3Top, cx + cw, ct + card3Top + dp(104)})
	}
}

// drawOverviewTab renders the rich mockup content of Tab 0
func drawOverviewTab(dc uintptr, cx, ct, cw, winH int32) {
	drawTitleBlock(dc, cx, cw, tr("Tổng quan", "Overview"), tr("Thiết lập các tùy chọn cơ bản để sử dụng Tui Gõ", "Configure essential options for Tui Gõ"))

	// Hero Card with active glass depth
	heroRc := rect{cx, ct, cx + cw, ct + dp(56)}
	gfx := NewGfx(dc)
	if gfx != nil {
		gfx.DrawGlassCardExt(float32(heroRc.left), float32(heroRc.top), float32(heroRc.right-heroRc.left), float32(heroRc.bottom-heroRc.top), float32(dp(10)), isDark(), false, effectiveViet())
		gfx.Destroy()
	}

	// Hero Card text — status display only (no master switch, EVKey-style).
	// On solid-fill styles the active hero is painted in accent — flip all
	// its inks to the on-accent color or text drowns on the mint fill.
	heroSolid := effectiveViet() && solidAccentSelected()
	heroText, heroSub := colText, colMuted
	if heroSolid {
		heroText = accentOnCol()
		heroSub = accentOnCol()
	}
	textX := cx + dp(18)
	pSetTextColor.Call(dc, heroText)
	pSelectObject.Call(dc, fontBold)
	htrc := rect{textX, ct + dp(8), cx + cw - dp(140), ct + dp(30)}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr(tr("Chế độ gõ hiện tại", "Current input mode")))),
		^uintptr(0), uintptr(unsafe.Pointer(&htrc)),
		DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

	pSetTextColor.Call(dc, heroSub)
	pSelectObject.Call(dc, fontHint)
	hdrc := rect{textX, ct + dp(30), cx + cw - dp(140), ct + dp(48)}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr(tr("Bấm phím tắt để chuyển Tiếng Việt / English", "Press shortcut to toggle Vietnamese / English")))),
		^uintptr(0), uintptr(unsafe.Pointer(&hdrc)),
		DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

	// Mode indicator on the right where the switch used to be
	modeTxt := tr("● Tiếng Việt", "● Vietnamese")
	modeCol := colJade
	if !effectiveViet() {
		modeTxt = "● English"
		modeCol = colMuted
	}
	if heroSolid {
		modeCol = accentOnCol()
	}
	pSetTextColor.Call(dc, modeCol)
	pSelectObject.Call(dc, fontBold)
	mrc := rect{cx + cw - dp(140), ct, cx + cw - dp(18), ct + dp(56)}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr(modeTxt))),
		^uintptr(0), uintptr(unsafe.Pointer(&mrc)),
		DT_RIGHT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

	// Section 1: "Chế độ gõ"
	pSetTextColor.Call(dc, colText)
	pSelectObject.Call(dc, fontSection)
	sec1 := rect{cx, ct + dp(66), cx + cw, ct + dp(86)}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr(tr("Chế độ gõ", "Input method")))),
		^uintptr(0), uintptr(unsafe.Pointer(&sec1)),
		DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

	pSetTextColor.Call(dc, colMuted)
	pSelectObject.Call(dc, fontHint)
	sec1Sub := rect{cx, ct + dp(86), cx + cw, ct + dp(102)}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr(tr("Chọn kiểu gõ phù hợp với thói quen của bạn (rê chuột để xem cách gõ)", "Choose your preferred typing method (hover to view typing guide)")))),
		^uintptr(0), uintptr(unsafe.Pointer(&sec1Sub)),
		DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

	// Section 2: "Tùy chọn chung"
	pSetTextColor.Call(dc, colText)
	pSelectObject.Call(dc, fontSection)
	sec2 := rect{cx, ct + dp(160), cx + cw, ct + dp(180)}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr(tr("Tùy chọn chung", "General options")))),
		^uintptr(0), uintptr(unsafe.Pointer(&sec2)),
		DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

	// Section 3: "Ngôn ngữ hiển thị"
	pSetTextColor.Call(dc, colText)
	pSelectObject.Call(dc, fontSection)
	sec3 := rect{cx, ct + dp(286), cx + cw - dp(150), ct + dp(306)}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr(tr("Ngôn ngữ hiển thị", "Display language")))),
		^uintptr(0), uintptr(unsafe.Pointer(&sec3)),
		DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

	pSetTextColor.Call(dc, colMuted)
	pSelectObject.Call(dc, fontHint)
	sec3Sub := rect{cx, ct + dp(306), cx + cw - dp(150), ct + dp(322)}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr(tr("Ngôn ngữ của giao diện Tui Gõ", "Interface language of Tui Gõ")))),
		^uintptr(0), uintptr(unsafe.Pointer(&sec3Sub)),
		DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

	// Section 4: Footer Status Card
	footerHeight := dp(48)
	compactFooter := cw < dp(520)
	if compactFooter {
		footerHeight = dp(64)
	}
	ftRc := rect{cx, ct + dp(336), cx + cw, ct + dp(336) + footerHeight}
	drawSettingsCard(dc, ftRc)

	sx := cx + dp(22)
	sy := ftRc.top + dp(24)
	gfx = NewGfx(dc)
	if gfx != nil {
		dotCol := argbSwitchOn
		if !effectiveViet() {
			dotCol = argbMuted
		}
		// Glowing outer ring
		gfx.FillCircle(float32(sx), float32(sy), float32(dp(8)), HexAlphaARGB(0x28, dotCol&0x00FFFFFF))
		// Core status dot
		gfx.FillCircle(float32(sx), float32(sy), float32(dp(4)), dotCol)
		gfx.Destroy()
	}

	stTitle := tr("Tiếng Việt đang hoạt động", "Vietnamese input is active")
	if !effectiveViet() {
		stTitle = tr("Đang tắt tiếng Việt (English)", "Vietnamese input is off (English)")
	}
	pSetTextColor.Call(dc, colText)
	pSelectObject.Call(dc, fontBold)
	statusRight := cx + cw - dp(260)
	if compactFooter {
		statusRight = cx + cw - dp(16)
	}
	st1 := rect{sx + dp(18), ftRc.top + dp(6), statusRight, ftRc.top + dp(25)}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr(stTitle))),
		^uintptr(0), uintptr(unsafe.Pointer(&st1)),
		DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

	pSetTextColor.Call(dc, colMuted)
	pSelectObject.Call(dc, fontHint)
	st2 := rect{sx + dp(18), ftRc.top + dp(25), statusRight, ftRc.top + dp(44)}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr(tr("Phiên bản "+appVersion+" | Cập nhật lần cuối: "+appBuildStamp, "Version "+appVersion+" | Last updated: "+appBuildStamp)))),
		^uintptr(0), uintptr(unsafe.Pointer(&st2)),
		DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

	pSetTextColor.Call(dc, colGold)
	pSelectObject.Call(dc, fontTagline)
	scRc := rect{cx + cw - dp(260), ftRc.top + dp(8), cx + cw - dp(16), ftRc.bottom - dp(8)}
	if compactFooter {
		scRc = rect{sx + dp(18), ftRc.top + dp(43), cx + cw - dp(16), ftRc.bottom - dp(3)}
	}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr(tr("Cùng bạn trên mọi hành trình ♥", "With you on every journey ♥")))),
		^uintptr(0), uintptr(unsafe.Pointer(&scRc)),
		DT_RIGHT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

	_ = winH
}

// drawAboutTab keeps the app details in the same card system as Settings.
func drawAboutTab(dc uintptr, cx, ct, cw, winH int32) {
	drawTitleBlock(dc, cx, cw, tabName(5), tabDesc(5))
	drawSettingsCard(dc, rect{cx, ct, cx + cw, ct + dp(160)})
	drawSettingsCard(dc, rect{cx, ct + dp(172), cx + cw, ct + dp(362)})

	mascotS := dp(72)
	mascotX, mascotY := cx+dp(20), ct+dp(20)
	drawAppMascot(dc, mascotX, mascotY, mascotS, mascotS, dp(12))
	pSetBkMode.Call(dc, TRANSPARENT)
	oldFont, _, _ := pSelectObject.Call(dc, fontTitle)
	pSetTextColor.Call(dc, colText)
	title := rect{mascotX + mascotS + dp(20), ct + dp(25), cx + cw - dp(20), ct + dp(55)}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr("Tui Gõ"))), ^uintptr(0), uintptr(unsafe.Pointer(&title)), DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)
	pSelectObject.Call(dc, fontNormal)
	pSetTextColor.Call(dc, colMuted)
	ver := rect{mascotX + mascotS + dp(20), ct + dp(58), cx + cw - dp(20), ct + dp(80)}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr(tr("Phiên bản "+appVersion, "Version "+appVersion)))), ^uintptr(0), uintptr(unsafe.Pointer(&ver)), DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)
	pSetTextColor.Call(dc, colText)
	desc := rect{cx + dp(20), ct + dp(105), cx + cw - dp(20), ct + dp(148)}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr(tr("Bộ gõ tiếng Việt cho Windows, hỗ trợ nhiều kiểu gõ và gõ tắt. Cài đặt được lưu trên máy của bạn.", "Modern Vietnamese input method for Windows with multi-method and macro support. All settings are stored locally.")))), ^uintptr(0), uintptr(unsafe.Pointer(&desc)), DT_LEFT|DT_WORDBREAK|DT_NOPREFIX)

	pSelectObject.Call(dc, fontSection)
	info := rect{cx + dp(20), ct + dp(188), cx + cw - dp(20), ct + dp(212)}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr(tr("Tính năng", "Features")))), ^uintptr(0), uintptr(unsafe.Pointer(&info)), DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)
	pSelectObject.Call(dc, fontNormal)
	rows := []struct{ label, value string }{
		{tr("Kiểu gõ", "Input methods"), "Telex · Simple Telex · VNI · VIQR"},
		{tr("Bảng mã", "Charsets"), "Unicode · TCVN3"},
		{tr("Tùy chỉnh", "Customization"), tr("Phím tắt · Ứng dụng loại trừ", "Shortcuts · App exclusions")},
	}
	for i, row := range rows {
		y := ct + dp(220) + int32(i)*dp(27)
		pSetTextColor.Call(dc, colMuted)
		label := rect{cx + dp(20), y, cx + dp(130), y + dp(22)}
		pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr(row.label))), ^uintptr(0), uintptr(unsafe.Pointer(&label)), DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)
		pSetTextColor.Call(dc, colText)
		value := rect{cx + dp(135), y, cx + cw - dp(20), y + dp(22)}
		pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr(row.value))), ^uintptr(0), uintptr(unsafe.Pointer(&value)), DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)
	}
	pSelectObject.Call(dc, oldFont)
	_ = winH
}

func drawButton(di *drawItemStruct) {
	dc := di.hDC
	rc := di.rcItem
	pressed := di.itemState&ODS_SELECTED != 0

	pFillRect.Call(dc, uintptr(unsafe.Pointer(&rc)), brCard)

	bgCol := colBtnBg
	if pressed {
		bgCol = colBtnHover
	}
	bbr, _, _ := pCreateSolidBrush.Call(bgCol)
	oldB, _, _ := pSelectObject.Call(dc, bbr)
	pp, _, _ := pCreatePen.Call(PS_SOLID, 1, colBtnBorder)
	oldP, _, _ := pSelectObject.Call(dc, pp)

	pRoundRect.Call(dc, uintptr(rc.left), uintptr(rc.top),
		uintptr(rc.right), uintptr(rc.bottom), uintptr(dp(8)), uintptr(dp(8)))

	pSelectObject.Call(dc, oldP)
	pDeleteObject.Call(pp)
	pSelectObject.Call(dc, oldB)
	pDeleteObject.Call(bbr)

	txt := btnText[int(di.ctlID)]
	pSetBkMode.Call(dc, TRANSPARENT)
	pSetTextColor.Call(dc, colBtnText)
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

func drawCheckbox(di *drawItemStruct) {
	dc := di.hDC
	rc := di.rcItem
	id := int(di.ctlID)
	checked := swState[id]

	pFillRect.Call(dc, uintptr(unsafe.Pointer(&rc)), brBg)

	boxS := dp(18)
	by := rc.top + (rc.bottom-rc.top-boxS)/2
	bx := rc.left

	if checked {
		oldB, _, _ := pSelectObject.Call(dc, brJade)
		oldP, _, _ := pSelectObject.Call(dc, penJade)
		pRoundRect.Call(dc, uintptr(bx), uintptr(by), uintptr(bx+boxS), uintptr(by+boxS), uintptr(dp(4)), uintptr(dp(4)))
		pSelectObject.Call(dc, oldP)
		pSelectObject.Call(dc, oldB)

		// A fat check mark reads at a glance; a 1-2px tick on tinted jade
		// looked like noise at smaller sizes.
		chkPen, _, _ := pCreatePen.Call(PS_SOLID, uintptr(max(int32(2), dp(3))), 0xFFFFFF)
		oldP2, _, _ := pSelectObject.Call(dc, chkPen)
		pts := []point{
			{bx + dp(4), by + dp(9)},
			{bx + dp(7), by + dp(13)},
			{bx + dp(14), by + dp(5)},
		}
		pPolyline.Call(dc, uintptr(unsafe.Pointer(&pts[0])), uintptr(len(pts)))
		pSelectObject.Call(dc, oldP2)
		pDeleteObject.Call(chkPen)
	} else {
		// penBorder is nearly invisible on both palettes — same treatment
		// as the unselected radio ring (muted gray outline).
		oldB, _, _ := pSelectObject.Call(dc, brHero)
		oldP, _, _ := pSelectObject.Call(dc, penMuted)
		pRoundRect.Call(dc, uintptr(bx), uintptr(by), uintptr(bx+boxS), uintptr(by+boxS), uintptr(dp(4)), uintptr(dp(4)))
		pSelectObject.Call(dc, oldP)
		pSelectObject.Call(dc, oldB)
	}

	pSetBkMode.Call(dc, TRANSPARENT)
	pSetTextColor.Call(dc, colText)
	oldF, _, _ := pSelectObject.Call(dc, fontNormal)
	trc := rect{bx + boxS + dp(8), rc.top, rc.right, rc.bottom}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr(btnText[id]))),
		^uintptr(0), uintptr(unsafe.Pointer(&trc)),
		DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)
	pSelectObject.Call(dc, oldF)
}

func drawRadio(di *drawItemStruct) {
	dc := di.hDC
	rc := di.rcItem
	id := int(di.ctlID)

	selected := false
	switch id {
	case cRadioTelex:
		selected = (gIM == ImTelex)
	case cRadioTelexSimple:
		selected = (gIM == ImTelexSimple)
	case cRadioVni:
		selected = (gIM == ImVni)
	case cRadioViqr:
		selected = (gIM == ImViqr)
	}

	bgBr := brHero
	bdPen := penBorder
	if selected {
		bgBr = brRadioActive
		bdPen = penJade
	}
	oldB, _, _ := pSelectObject.Call(dc, bgBr)
	oldP, _, _ := pSelectObject.Call(dc, bdPen)
	pRoundRect.Call(dc, uintptr(rc.left), uintptr(rc.top),
		uintptr(rc.right), uintptr(rc.bottom), uintptr(dp(8)), uintptr(dp(8)))
	pSelectObject.Call(dc, oldP)
	pSelectObject.Call(dc, oldB)

	cx := rc.left + dp(13)
	cy := rc.top + (rc.bottom-rc.top)/2
	r := dp(6)

	if selected {
		oldP2, _, _ := pSelectObject.Call(dc, penJade)
		nullB, _, _ := pGetStockObject.Call(NULL_BRUSH)
		oldB2, _, _ := pSelectObject.Call(dc, nullB)
		pEllipse.Call(dc, uintptr(cx-r), uintptr(cy-r), uintptr(cx+r), uintptr(cy+r))
		pSelectObject.Call(dc, oldB2)
		pSelectObject.Call(dc, oldP2)

		oldB3, _, _ := pSelectObject.Call(dc, brJade)
		nullPen, _, _ := pGetStockObject.Call(NULL_PEN)
		oldP3, _, _ := pSelectObject.Call(dc, nullPen)
		ir := dp(3)
		pEllipse.Call(dc, uintptr(cx-ir), uintptr(cy-ir), uintptr(cx+ir), uintptr(cy+ir))
		pSelectObject.Call(dc, oldP3)
		pSelectObject.Call(dc, oldB3)
	} else {
		oldP2, _, _ := pSelectObject.Call(dc, penMuted)
		nullB, _, _ := pGetStockObject.Call(NULL_BRUSH)
		oldB2, _, _ := pSelectObject.Call(dc, nullB)
		pEllipse.Call(dc, uintptr(cx-r), uintptr(cy-r), uintptr(cx+r), uintptr(cy+r))
		pSelectObject.Call(dc, oldB2)
		pSelectObject.Call(dc, oldP2)
	}

	pSetBkMode.Call(dc, TRANSPARENT)
	pSetTextColor.Call(dc, colText)
	oldF, _, _ := pSelectObject.Call(dc, fontBold)
	if radioSubtitle[id] != "" {
		trc := rect{rc.left + dp(23), rc.top + dp(6), rc.right - dp(2), rc.top + dp(24)}
		pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr(btnText[id]))),
			^uintptr(0), uintptr(unsafe.Pointer(&trc)),
			DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

		pSetTextColor.Call(dc, colMuted)
		pSelectObject.Call(dc, fontHint)
		drc := rect{rc.left + dp(23), rc.top + dp(24), rc.right - dp(2), rc.bottom - dp(4)}
		pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr(radioSubtitle[id]))),
			^uintptr(0), uintptr(unsafe.Pointer(&drc)),
			DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)
	} else {
		trc := rect{rc.left + dp(23), rc.top, rc.right - dp(2), rc.bottom}
		pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr(btnText[id]))),
			^uintptr(0), uintptr(unsafe.Pointer(&trc)),
			DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)
	}

	pSelectObject.Call(dc, oldF)
}

func drawLangBtn(di *drawItemStruct) {
	dc := di.hDC
	rc := di.rcItem
	pressed := di.itemState&ODS_SELECTED != 0

	fillParentBg(dc, rc, di.hwndItem)

	gfx := NewGfx(dc)
	if gfx != nil {
		gfx.DrawGlassCard(float32(rc.left), float32(rc.top), float32(rc.right-rc.left), float32(rc.bottom-rc.top), float32(dp(6)), isDark(), pressed)
		gfx.Destroy()
	}

	pSetBkMode.Call(dc, TRANSPARENT)
	pSetTextColor.Call(dc, colText)
	oldF, _, _ := pSelectObject.Call(dc, fontNormal)
	trc := rect{rc.left + dp(12), rc.top, rc.right - dp(24), rc.bottom}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr(tr("Tiếng Việt", "English")))),
		^uintptr(0), uintptr(unsafe.Pointer(&trc)),
		DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

	pSetTextColor.Call(dc, colMuted)
	chrc := rect{rc.right - dp(22), rc.top, rc.right - dp(8), rc.bottom}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr("▾"))),
		^uintptr(0), uintptr(unsafe.Pointer(&chrc)),
		DT_CENTER|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

	pSelectObject.Call(dc, oldF)
}

// Modifiers are choices within one shortcut, so display them as compact
// selectable keys instead of full-size on/off switches.
func drawModifier(di *drawItemStruct) {
	dc := di.hDC
	rc := di.rcItem
	id := int(di.ctlID)
	selected := swState[id]
	pFillRect.Call(dc, uintptr(unsafe.Pointer(&rc)), brCard)
	brush, pen, color := brCard, penBorder, colText
	if selected {
		brush, pen, color = brRadioActive, penJade, colJade
	}
	oldBrush, _, _ := pSelectObject.Call(dc, brush)
	oldPen, _, _ := pSelectObject.Call(dc, pen)
	pRoundRect.Call(dc, uintptr(rc.left), uintptr(rc.top), uintptr(rc.right), uintptr(rc.bottom),
		uintptr(dp(7)), uintptr(dp(7)))
	pSelectObject.Call(dc, oldPen)
	pSelectObject.Call(dc, oldBrush)
	pSetBkMode.Call(dc, TRANSPARENT)
	pSetTextColor.Call(dc, color)
	oldFont, _, _ := pSelectObject.Call(dc, fontBold)
	textRc := rc
	if di.itemState&ODS_SELECTED != 0 {
		textRc.top += dp(1)
	}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr(swText[id]))),
		^uintptr(0), uintptr(unsafe.Pointer(&textRc)), DT_CENTER|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)
	pSelectObject.Call(dc, oldFont)
}

func drawSwitch(di *drawItemStruct) {
	dc := di.hDC
	rc := di.rcItem
	on := swState[int(di.ctlID)]

	pFillRect.Call(dc, uintptr(unsafe.Pointer(&rc)), brCard)

	pillH := dp(20)
	pillW := dp(36)
	py := rc.top + (rc.bottom-rc.top-pillH)/2
	pill := rect{rc.left, py, rc.left + pillW, py + pillH}
	pillCol := uintptr(colPill)
	knobCol := uintptr(colMuted)
	if on {
		pillCol = colJade
		knobCol = 0xFFFFFF
	}

	pbr, _, _ := pCreateSolidBrush.Call(pillCol)
	oldB, _, _ := pSelectObject.Call(dc, pbr)
	var pp uintptr
	if on {
		pp, _, _ = pCreatePen.Call(PS_SOLID, 1, pillCol)
	} else {
		pp, _, _ = pCreatePen.Call(PS_SOLID, 1, colBorder)
	}
	oldP, _, _ := pSelectObject.Call(dc, pp)
	pRoundRect.Call(dc, uintptr(pill.left), uintptr(pill.top),
		uintptr(pill.right), uintptr(pill.bottom), uintptr(pillH), uintptr(pillH))
	pSelectObject.Call(dc, oldP)
	pDeleteObject.Call(pp)

	kx := pill.left + dp(3)
	if on {
		kx = pill.right - dp(17)
	}
	kbr, _, _ := pCreateSolidBrush.Call(knobCol)
	pSelectObject.Call(dc, kbr)
	nullPen, _, _ := pGetStockObject.Call(NULL_PEN)
	oldKP, _, _ := pSelectObject.Call(dc, nullPen)
	pEllipse.Call(dc, uintptr(kx), uintptr(py+dp(3)),
		uintptr(kx+dp(14)), uintptr(py+dp(17)))
	pSelectObject.Call(dc, oldKP)
	pSelectObject.Call(dc, oldB)
	pDeleteObject.Call(kbr)
	pDeleteObject.Call(pbr)

	pSetBkMode.Call(dc, TRANSPARENT)
	pSetTextColor.Call(dc, colText)
	oldF, _, _ := pSelectObject.Call(dc, fontNormal)
	trc := rect{rc.left + dp(46), rc.top, rc.right, rc.bottom}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr(swText[int(di.ctlID)]))),
		^uintptr(0), uintptr(unsafe.Pointer(&trc)),
		DT_VCENTER|DT_SINGLELINE)
	pSelectObject.Call(dc, oldF)
}

func setCheck(id int, on bool) {
	swState[id] = on
	pInvalidateRect.Call(ctlHnd[id], 0, 1)
}

func getCheck(id int) bool {
	return swState[id]
}

func comboAdd(h uintptr, items []string, sel int) {
	for _, s := range items {
		pSendMessage.Call(h, CB_ADDSTRING, 0, uintptr(unsafe.Pointer(utf16ptr(s))))
	}
	pSendMessage.Call(h, CB_SETCURSEL, uintptr(sel), 0)
}

func comboSel(id int) int {
	r, _, _ := pSendMessage.Call(ctlHnd[id], CB_GETCURSEL, 0, 0)
	return int(r)
}

func applySettingsFont() {
	fontNormal, _, _ = pCreateFont.Call(^uintptr(dp(12)), 0, 0, 0, 400, 0, 0, 0,
		DEFAULT_CHARSET, 0, 0, CLEARTYPE_QUALITY, 0,
		uintptr(unsafe.Pointer(utf16ptr("Segoe UI"))))
	fontBold, _, _ = pCreateFont.Call(^uintptr(dp(12)), 0, 0, 0, 600, 0, 0, 0,
		DEFAULT_CHARSET, 0, 0, CLEARTYPE_QUALITY, 0,
		uintptr(unsafe.Pointer(utf16ptr("Segoe UI"))))
	fontTitle, _, _ = pCreateFont.Call(^uintptr(dp(16)), 0, 0, 0, 700, 0, 0, 0,
		DEFAULT_CHARSET, 0, 0, CLEARTYPE_QUALITY, 0,
		uintptr(unsafe.Pointer(utf16ptr("Segoe UI"))))
	fontSection, _, _ = pCreateFont.Call(^uintptr(dp(13)), 0, 0, 0, 600, 0, 0, 0,
		DEFAULT_CHARSET, 0, 0, CLEARTYPE_QUALITY, 0,
		uintptr(unsafe.Pointer(utf16ptr("Segoe UI"))))
	fontHint, _, _ = pCreateFont.Call(^uintptr(dp(10)), 0, 0, 0, 400, 0, 0, 0,
		DEFAULT_CHARSET, 0, 0, CLEARTYPE_QUALITY, 0,
		uintptr(unsafe.Pointer(utf16ptr("Segoe UI"))))
	// Use the same Vietnamese-capable family as the rest of the UI. A valid
	// Segoe Script HFONT does not guarantee coverage of Vietnamese diacritics.
	fontTagline, _, _ = pCreateFont.Call(^uintptr(dp(11)), 0, 0, 0, 400, 1, 0, 0,
		DEFAULT_CHARSET, 0, 0, CLEARTYPE_QUALITY, 0,
		uintptr(unsafe.Pointer(utf16ptr("Segoe UI"))))
}

func rebuildFonts() {
	deleteGdiObj(&fontNormal)
	deleteGdiObj(&fontBold)
	deleteGdiObj(&fontTitle)
	deleteGdiObj(&fontSection)
	deleteGdiObj(&fontHint)
	deleteGdiObj(&fontTagline)
	applySettingsFont()
	if gTooltipHwnd != 0 {
		pSendMessage.Call(gTooltipHwnd, WM_SETFONT, fontNormal, 1)
		pSendMessage.Call(gTooltipHwnd, 0x0418 /* TTM_SETMAXTIPWIDTH */, 0, uintptr(dp(340)))
	}
	for _, hctl := range allCtls {
		switch {
		case goldCtls[hctl]:
			pSendMessage.Call(hctl, WM_SETFONT, fontSection, 1)
		case mutedCtls[hctl]:
			pSendMessage.Call(hctl, WM_SETFONT, fontHint, 1)
		default:
			pSendMessage.Call(hctl, WM_SETFONT, fontNormal, 1)
		}
	}
}

var (
	comctl32            = windows.NewLazySystemDLL("comctl32.dll")
	pInitCommonControls = comctl32.NewProc("InitCommonControls")
	gTooltipHwnd        uintptr
)

type toolInfoW struct {
	cbSize   uint32
	uFlags   uint32
	hwnd     uintptr
	uId      uintptr
	rect     rect
	hinst    uintptr
	lpszText *uint16
}

func attachControlTooltip(parent, ctl uintptr, text string) {
	if gTooltipHwnd == 0 || ctl == 0 || text == "" {
		return
	}
	ti := toolInfoW{
		cbSize:   uint32(unsafe.Sizeof(toolInfoW{})),
		uFlags:   0x0001 | 0x0010, // TTF_IDISHWND | TTF_SUBCLASS
		hwnd:     parent,
		uId:      ctl,
		lpszText: utf16ptr(text),
	}
	pSendMessage.Call(gTooltipHwnd, 0x0432 /* TTM_ADDTOOLW */, 0, uintptr(unsafe.Pointer(&ti)))
}

func getDpi(hwnd uintptr) int32 {
	if r, _, _ := pGetDpiForWindow.Call(hwnd); r > 0 {
		return int32(r)
	}
	return 96
}

// systemDpi reads the primary display DPI — needed before the window exists.
func systemDpi() int32 {
	if r, _, _ := pGetDpiForSystem.Call(); r > 0 {
		return int32(r)
	}
	return 96
}

func buildControls(h uintptr) {
	gSettingsHwnd = h
	gbH = h
	defer func() { gbH = 0 }()
	ctlHnd = map[int]uintptr{}
	goldCtls = map[uintptr]bool{}
	mutedCtls = map[uintptr]bool{}
	bgCtls = map[uintptr]bool{}
	swState = map[int]bool{}
	swText = map[int]string{}
	btnText = map[int]string{}
	btnIDs = map[int]bool{}
	isCheckbox = map[int]bool{}
	isRadio = map[int]bool{}
	radioSubtitle = map[int]string{}
	ctlTab = map[uintptr]int{}
	allCtls = nil
	stSeq = 299
	gDpi = getDpi(h)
	applySettingsFont()

	pInitCommonControls.Call()
	if gTooltipHwnd != 0 {
		pDestroyWindow.Call(gTooltipHwnd)
		gTooltipHwnd = 0
	}
	gTooltipHwnd, _, _ = pCreateWindowEx.Call(
		0x00000008, // WS_EX_TOPMOST
		uintptr(unsafe.Pointer(utf16ptr("tooltips_class32"))),
		0,
		0x80000000|0x01|0x02, // WS_POPUP | TTS_ALWAYSTIP | TTS_NOPREFIX
		0, 0, 0, 0,
		h, 0, 0, 0)
	if gTooltipHwnd != 0 {
		pSendMessage.Call(gTooltipHwnd, WM_SETFONT, fontNormal, 1)
		pSendMessage.Call(gTooltipHwnd, 0x0418 /* TTM_SETMAXTIPWIDTH */, 0, uintptr(dp(340)))
		pSendMessage.Call(gTooltipHwnd, 0x0403 /* TTM_SETDELAYTIME */, 3 /* TTDT_INITIAL */, 200)
	}

	// ---- Tab 0: Tổng quan (Overview matching mockup) ----
	curTab = 0
	// 4 Radio pills for Kiểu gõ (single-line, with hover tooltips explaining rules)
	addRadio(cRadioTelex, "Telex", "")
	addRadio(cRadioTelexSimple, "Simple Telex", "")
	addRadio(cRadioVni, "VNI", "")
	addRadio(cRadioViqr, "VIQR", "")
	attachControlTooltip(h, ctlHnd[cRadioTelex], tr(
		"Telex chuẩn (w gõ ư/ơ/ă):\r\n• Chữ: aa→â, aw→ă, ee→ê, oo→ô, ow→ơ, uw/w→ư, dd→đ\r\n• Dấu: s (sắc), f (huyền), r (hỏi), x (ngã), j (nặng), z (xóa dấu)",
		"Standard Telex (w types ư/ơ/ă):\r\n• Letters: aa→â, aw→ă, ee→ê, oo→ô, ow→ơ, uw/w→ư, dd→đ\r\n• Tones: s (acute), f (grave), r (hook), x (tilde), j (dot), z (clear)"))
	attachControlTooltip(h, ctlHnd[cRadioTelexSimple], tr(
		"Simple Telex (gõ w giữ nguyên chữ w):\r\n• Chữ: aa→â, aw→ă, ee→ê, oo→ô, ow→ơ, uw→ư, dd→đ\r\n• Dấu: s (sắc), f (huyền), r (hỏi), x (ngã), j (nặng), z (xóa dấu)",
		"Simple Telex (standalone w stays w):\r\n• Letters: aa→â, aw→ă, ee→ê, oo→ô, ow→ơ, uw→ư, dd→đ\r\n• Tones: s (acute), f (grave), r (hook), x (tilde), j (dot), z (clear)"))
	attachControlTooltip(h, ctlHnd[cRadioVni], tr(
		"Kiểu gõ VNI (dùng phím số):\r\n• Chữ: 6 (â, ê, ô), 7 (ơ, ư), 8 (ă), 9 (đ)\r\n• Dấu: 1 (sắc), 2 (huyền), 3 (hỏi), 4 (ngã), 5 (nặng), 0 (xóa dấu)",
		"VNI input method (number keys):\r\n• Letters: 6 (â, ê, ô), 7 (ơ, ư), 8 (ă), 9 (đ)\r\n• Tones: 1 (acute), 2 (grave), 3 (hook), 4 (tilde), 5 (dot), 0 (clear)"))
	attachControlTooltip(h, ctlHnd[cRadioViqr], tr(
		"Kiểu gõ VIQR (dấu câu quy ước):\r\n• Chữ: ^ (â, ê, ô), + hoặc * (ơ, ư), ( (ă), dd (đ)\r\n• Dấu: ' (sắc), ` (huyền), ? (hỏi), ~ (ngã), . (nặng), - (xóa dấu)",
		"VIQR input method (punctuation marks):\r\n• Letters: ^ (â, ê, ô), + or * (ơ, ư), ( (ă), dd (đ)\r\n• Tones: ' (acute), ` (grave), ? (hook), ~ (tilde), . (dot), - (clear)"))

	// 6 Checkboxes for Tùy chọn chung (2 columns)
	addCheckbox(cStartup, tr("Khởi động cùng hệ thống", "Launch at Windows startup"))
	addCheckbox(cRestore, tr("Tự động khôi phục từ sai", "Auto-restore non-Vietnamese words"))
	addCheckbox(cAutoCap, tr("Tự động viết hoa chữ đầu", "Auto-capitalize first letter"))
	addCheckbox(cModern, tr("Bỏ dấu kiểu mới (oà, uý)", "Modern tone placement (oà, uý)"))
	addCheckbox(cSpell, tr("Kiểm tra chính tả tiếng Việt", "Vietnamese spell check"))
	addCheckbox(cFreeMark, tr("Cho phép bỏ dấu tự do", "Allow free tone marking"))
	// Language dropdown button
	addBtn(cLangBtn, tr("Tiếng Việt", "English"))

	// ---- Tab 1: Gõ tiếng Việt (Advanced input methods & Charset & Macro) ----
	curTab = 1
	stIMSection = addSection(tr("KIỂU GÕ NÂNG CAO", "ADVANCED INPUT METHOD"))
	stLbIM = addLabel(tr("Kiểu gõ", "Method"))
	cb := mkCtl(h, ctlDef{cIMCombo, "COMBOBOX", CBS_DROPDOWNLIST | WS_TABSTOP, "", 0, 0, 10, 10})
	comboAdd(cb, []string{"Telex", "Simple Telex", "VNI", "VIQR",
		"Telex + VNI", "MS-VI", tr("Tự định nghĩa", "Custom keymap")}, 0)
	if isDark() {
		darkCtl(cb)
	}
	stLbCharset = addLabel(tr("Bảng mã", "Charset"))
	cs := mkCtl(h, ctlDef{cCharset, "COMBOBOX", CBS_DROPDOWNLIST | WS_TABSTOP, "", 0, 0, 10, 10})
	comboAdd(cs, []string{"Unicode", "TCVN3"}, cfg.Charset)
	if isDark() {
		darkCtl(cs)
	}

	stMacroSection = addSection(tr("GÕ TẮT (MACRO)", "MACROS (TEXT EXPANSION)"))
	addSw(cMacro, tr("Cho phép gõ tắt", "Enable macros"))
	addSw(cMacroAlways, tr("Gõ tắt cả khi tắt Tiếng Việt", "Expand macros even in English mode"))
	addBtn(cMacroEdit, tr("Mở bảng gõ tắt", "Edit macro list"))
	stMacroHint1 = addHint(tr("Mỗi dòng: từ_gõ_tắt:nội_dung  (vd: vn:Việt Nam)", "Format per line: trigger:expansion  (e.g. vn:Việt Nam)"))
	stMacroHint2 = addHint(tr("File macro.txt nằm cạnh tuigo.exe", "Stored in macro.txt next to tuigo.exe"))

	// ---- Tab 2: Phím tắt ----
	curTab = 2
	stSwitchSection = addSection(tr("PHÍM CHUYỂN TIẾNG VIỆT", "LANGUAGE TOGGLE SHORTCUT"))
	addSw(cModCtrl, "Ctrl")
	addSw(cModShift, "Shift")
	addSw(cModAlt, "Alt")
	addSw(cModWin, "Win")
	stLbHotkey = addLabel(tr("Phím", "Key"))
	hotkeyEdit := mkCtl(h, ctlDef{cHotkeyBox, "EDIT", WS_TABSTOP | 0x0001 | 0x0008, "", 0, 0, 10, 10})
	pSendMessage.Call(hotkeyEdit, 0x00C5 /* EM_LIMITTEXT */, 1, 0)
	stHotkeyHint = addHint(tr("A-Z hoặc 0-9 · có thể để trống", "A-Z or 0-9 · can be left blank"))
	addSw(cUseCtrlShift, tr("Thêm Ctrl + Shift làm phím chuyển", "Also toggle with Ctrl + Shift"))
	stFKeySection = addSection(tr("PHÍM NHANH (F-KEYS)", "FUNCTION HOTKEYS (F-KEYS)"))
	addSw(cF1, tr("F1 — Bật tiếng Việt", "F1 — Enable Vietnamese"))
	addSw(cF2, tr("F2 — Tắt tiếng Việt", "F2 — Disable Vietnamese (English)"))
	addSw(cF5, tr("F5 — Mở cài đặt", "F5 — Open settings"))
	addSw(cF9, tr("F9 — Bật/tắt gõ tắt", "F9 — Toggle macros"))
	addSw(cF12, tr("F12 — Xóa bộ đệm gõ", "F12 — Clear typing buffer"))

	// ---- Tab 3: Hệ thống ----
	curTab = 3
	stSysSection = addSection(tr("TÙY CHỌN HỆ THỐNG", "SYSTEM OPTIONS"))
	addSw(cShowHud, tr("Hiện nút trạng thái nổi (V/E)", "Show mini floating badge (V/E)"))
	addSw(cSound, tr("Âm thanh khi chuyển chế độ", "Play sound when toggling mode"))
	addSw(cSkipLayout, tr("Tự tắt khi bàn phím không phải kiểu US", "Auto-disable on non-US keyboard layouts"))
	addSw(cClipboard, tr("Dùng Clipboard cho app không nhận phím", "Use Clipboard fallback for incompatible apps"))
	addSw(cAdmin, tr("Chạy với quyền Admin", "Run as Administrator"))
	addSw(cShowDlg, tr("Hiện bảng này khi khởi động Windows", "Show settings window on startup"))
	addSw(cRememberAppMode, tr("Tự nhớ bật/tắt tiếng Việt theo ứng dụng", "Remember V/E mode per application"))
	addBtn(cExcludeBrowse, tr("Chọn app…", "Select app…"))
	addBtn(cExcludeEdit, tr("Danh sách loại trừ…", "Exclusion list…"))
	stExcludeSection = addSection(tr("ỨNG DỤNG LOẠI TRỪ", "EXCLUDED APPLICATIONS"))
	stExcludeHint = addHint(tr("Tự nhớ: chuyển V/E trong app, lần sau giữ nguyên.\nChọn app để loại trừ thủ công · |lock |clip |tcvn |vni", "Per-app memory: remembers your last V/E state per app.\nOr add manual rules in exclude.txt · |lock |clip |tcvn |vni"))

	// ---- Tab 4: Giao diện ----
	curTab = 4
	stThemeModeSection = addSection(tr("CHẾ ĐỘ MÀU", "COLOR MODE"))
	stStyleSection = addSection(tr("PHONG CÁCH CHỦ ĐỀ", "THEME STYLE"))
	for _, tid := range []int{
		cThemeAuto, cThemeLight, cThemeDark,
		cThemeBasic, cThemeJade, cThemeIosPure, cThemeCyberNeon, cThemeMatchaLatte, cThemeMaterialTeal,
	} {
		btnIDs[tid] = true
	}
	mkCtl(h, ctlDef{cThemeAuto, "BUTTON", BS_OWNERDRAW | WS_TABSTOP, tr("Hệ thống", "System"), 0, 0, 10, 10})
	mkCtl(h, ctlDef{cThemeLight, "BUTTON", BS_OWNERDRAW | WS_TABSTOP, tr("Sáng", "Light"), 0, 0, 10, 10})
	mkCtl(h, ctlDef{cThemeDark, "BUTTON", BS_OWNERDRAW | WS_TABSTOP, tr("Tối", "Dark"), 0, 0, 10, 10})
	mkCtl(h, ctlDef{cThemeBasic, "BUTTON", BS_OWNERDRAW | WS_TABSTOP, tr("Cơ bản", "Basic"), 0, 0, 10, 10})
	mkCtl(h, ctlDef{cThemeJade, "BUTTON", BS_OWNERDRAW | WS_TABSTOP, tr("Ngọc Bích", "Jade Glass"), 0, 0, 10, 10})
	mkCtl(h, ctlDef{cThemeIosPure, "BUTTON", BS_OWNERDRAW | WS_TABSTOP, "iOS 18 Pro", 0, 0, 10, 10})
	mkCtl(h, ctlDef{cThemeCyberNeon, "BUTTON", BS_OWNERDRAW | WS_TABSTOP, "Cyber Neon", 0, 0, 10, 10})
	mkCtl(h, ctlDef{cThemeMatchaLatte, "BUTTON", BS_OWNERDRAW | WS_TABSTOP, "Matcha Latte", 0, 0, 10, 10})
	mkCtl(h, ctlDef{cThemeMaterialTeal, "BUTTON", BS_OWNERDRAW | WS_TABSTOP, "Material 3", 0, 0, 10, 10})
	stTraySection = addSection(tr("BIỂU TƯỢNG KHAY HỆ THỐNG", "TRAY ICON"))
	for _, tid := range []int{cTrayV, cTrayVBg, cTrayE, cTrayEBg} {
		btnIDs[tid] = true
	}
	mkCtl(h, ctlDef{cTrayV, "BUTTON", BS_OWNERDRAW | WS_TABSTOP, "", 0, 0, 10, 10})
	mkCtl(h, ctlDef{cTrayVBg, "BUTTON", BS_OWNERDRAW | WS_TABSTOP, "", 0, 0, 10, 10})
	mkCtl(h, ctlDef{cTrayE, "BUTTON", BS_OWNERDRAW | WS_TABSTOP, "", 0, 0, 10, 10})
	mkCtl(h, ctlDef{cTrayEBg, "BUTTON", BS_OWNERDRAW | WS_TABSTOP, "", 0, 0, 10, 10})
	btnText[cTrayV] = tr("Chữ V", "V letter")
	btnText[cTrayVBg] = tr("Nền V", "V bg")
	btnText[cTrayE] = tr("Chữ E", "E letter")
	btnText[cTrayEBg] = tr("Nền E", "E bg")
	addBtn(cTrayReset, tr("Mặc định", "Reset"))
	stTrayHint = addHint(tr("Bấm ô màu để đổi màu biểu tượng khay V/E", "Click a swatch to recolor the V/E tray icon"))
	if cfg.Style == styleBasic {
		// Accent picker — only meaningful on the flat Cơ bản style
		stAccentLabel = addLabel(tr("Màu nhấn", "Accent color"))
		btnIDs[cAccent] = true
		mkCtl(h, ctlDef{cAccent, "BUTTON", BS_OWNERDRAW | WS_TABSTOP, "", 0, 0, 10, 10})
		btnText[cAccent] = tr("Nền", "Fill")
		btnIDs[cAccentText] = true
		mkCtl(h, ctlDef{cAccentText, "BUTTON", BS_OWNERDRAW | WS_TABSTOP, "", 0, 0, 10, 10})
		btnText[cAccentText] = tr("Chữ", "Text")
		addBtn(cAccentReset, tr("Mặc định", "Reset"))
	}

	// ---- Tab 5: Giới thiệu ----
	curTab = 5
	addBtn(cUpdateBtn, tr("Kiểm tra cập nhật", "Check for updates"))

	curTab = -1

	for _, hctl := range allCtls {
		switch {
		case goldCtls[hctl]:
			pSendMessage.Call(hctl, WM_SETFONT, fontSection, 1)
		case mutedCtls[hctl]:
			pSendMessage.Call(hctl, WM_SETFONT, fontHint, 1)
		default:
			pSendMessage.Call(hctl, WM_SETFONT, fontNormal, 1)
		}
	}
	layoutAll()
	syncControls()
	setTab(gActiveTab) // reopen on the tab the user last had open
}

func place(id int, x, y, w, h int32) {
	pSetWindowPos.Call(ctlHnd[id], 0, uintptr(x), uintptr(y),
		uintptr(w), uintptr(h), SWP_NOZORDER|SWP_NOACTIVATE)
}

func layoutAll() {
	if gSettingsHwnd == 0 || len(allCtls) == 0 {
		return
	}
	var rc rect
	pGetClientRect.Call(gSettingsHwnd, uintptr(unsafe.Pointer(&rc)))
	winW, winH := rc.right, rc.bottom
	railW = dp(232)
	cx := railW + dp(24)
	cw := winW - cx - dp(24)
	ct := dp(106)

	// Tab 0 — Tổng quan (4 radio cards on 1 row)
	pw := (cw - 3*dp(6)) / 4
	place(cRadioTelex, cx, ct+dp(106), pw, dp(44))
	place(cRadioTelexSimple, cx+pw+dp(6), ct+dp(106), pw, dp(44))
	place(cRadioVni, cx+2*(pw+dp(6)), ct+dp(106), pw, dp(44))
	rx4 := cx + 3*(pw+dp(6))
	place(cRadioViqr, rx4, ct+dp(106), cx+cw-rx4, dp(44))
	cw2 := (cw - dp(24)) / 2
	place(cStartup, cx, ct+dp(190), cw2, dp(26))
	place(cRestore, cx, ct+dp(220), cw2, dp(26))
	place(cAutoCap, cx, ct+dp(250), cw2, dp(26))
	place(cModern, cx+cw2+dp(24), ct+dp(190), cw2, dp(26))
	place(cSpell, cx+cw2+dp(24), ct+dp(220), cw2, dp(26))
	place(cFreeMark, cx+cw2+dp(24), ct+dp(250), cw2, dp(26))
	place(cLangBtn, cx+cw-dp(140), ct+dp(294), dp(140), dp(32))

	// Tab 1 — Gõ tiếng Việt
	place(stIMSection, cx+dp(16), ct+dp(12), cw-dp(32), dp(20))
	place(stLbIM, cx+dp(16), ct+dp(42), dp(82), dp(22))
	place(cIMCombo, cx+dp(110), ct+dp(40), cw-dp(126), dp(200))
	place(stLbCharset, cx+dp(16), ct+dp(72), dp(82), dp(22))
	place(cCharset, cx+dp(110), ct+dp(70), cw-dp(126), dp(140))
	place(stMacroSection, cx+dp(16), ct+dp(124), cw-dp(32), dp(20))
	place(cMacro, cx+dp(16), ct+dp(158), cw-dp(32), dp(26))
	place(cMacroAlways, cx+dp(16), ct+dp(190), cw-dp(32), dp(26))
	place(cMacroEdit, cx+dp(16), ct+dp(224), dp(160), dp(30))
	place(stMacroHint1, cx+dp(16), ct+dp(264), cw-dp(32), dp(20))
	place(stMacroHint2, cx+dp(16), ct+dp(288), cw-dp(32), dp(20))

	// Tab 2 — Phím tắt
	place(stSwitchSection, cx+dp(16), ct+dp(12), cw-dp(32), dp(20))
	// Four short labels fit cleanly on one row, including at high DPI.
	mw := (cw - dp(32) - 3*dp(8)) / 4
	for i, id := range []int{cModCtrl, cModShift, cModAlt, cModWin} {
		place(id, cx+dp(16)+int32(i)*(mw+dp(8)), ct+dp(44), mw, dp(34))
	}
	place(stLbHotkey, cx+dp(16), ct+dp(100), dp(48), dp(24))
	place(cHotkeyBox, cx+dp(74), ct+dp(101), dp(64), dp(20))
	place(stHotkeyHint, cx+dp(154), ct+dp(100), cw-dp(170), dp(24))
	place(cUseCtrlShift, cx+dp(16), ct+dp(139), cw-dp(32), dp(28))
	place(stFKeySection, cx+dp(16), ct+dp(204), cw-dp(32), dp(20))
	place(cF1, cx+dp(16), ct+dp(234), cw-dp(32), dp(26))
	place(cF2, cx+dp(16), ct+dp(261), cw-dp(32), dp(26))
	place(cF5, cx+dp(16), ct+dp(288), cw-dp(32), dp(26))
	place(cF9, cx+dp(16), ct+dp(315), cw-dp(32), dp(26))
	place(cF12, cx+dp(16), ct+dp(342), cw-dp(32), dp(26))

	// Tab 3 — Hệ thống
	place(stSysSection, cx+dp(16), ct+dp(12), cw-dp(32), dp(20))
	sysSw := []int{cShowHud, cSound, cSkipLayout, cClipboard, cAdmin, cShowDlg}
	for row, id := range sysSw {
		place(id, cx+dp(16), ct+dp(40)+int32(row)*dp(27), cw-dp(32), dp(25))
	}
	place(stExcludeSection, cx+dp(16), ct+dp(238), cw-dp(32), dp(20))
	place(cRememberAppMode, cx+dp(16), ct+dp(266), cw-dp(32), dp(25))
	place(cExcludeBrowse, cx+dp(16), ct+dp(298), dp(120), dp(28))
	place(cExcludeEdit, cx+dp(144), ct+dp(298), dp(170), dp(28))
	place(stExcludeHint, cx+dp(16), ct+dp(332), cw-dp(32), dp(32))

	// Tab 4 — Giao diện
	place(stThemeModeSection, cx+dp(16), ct+dp(12), cw-dp(32), dp(20))
	btnW := (cw - dp(32) - dp(16)) / 3
	place(cThemeAuto, cx+dp(16), ct+dp(40), btnW, dp(38))
	place(cThemeLight, cx+dp(16)+btnW+dp(8), ct+dp(40), btnW, dp(38))
	themeX3 := cx + dp(16) + 2*(btnW+dp(8))
	place(cThemeDark, themeX3, ct+dp(40), cx+cw-dp(16)-themeX3, dp(38))

	place(stStyleSection, cx+dp(16), ct+dp(118), cw-dp(32), dp(20))
	styleIDs := []int{cThemeBasic, cThemeJade, cThemeIosPure, cThemeCyberNeon, cThemeMatchaLatte, cThemeMaterialTeal}
	btnNW := (cw - dp(32) - int32(len(styleIDs)-1)*dp(6)) / int32(len(styleIDs))
	for i, tid := range styleIDs {
		tx := cx + dp(16) + int32(i)*(btnNW+dp(6))
		tw := btnNW
		if i == len(styleIDs)-1 {
			tw = cx + cw - dp(16) - tx
		}
		place(tid, tx, ct+dp(146), tw, dp(64))
	}
	// Accent pickers live at the bottom of the style card — Cơ bản only.
	// The card grows dp(48) when visible, so everything below shifts by accOff.
	accOff := int32(0)
	if ctlHnd[cAccent] != 0 {
		accOff = dp(48)
		place(stAccentLabel, cx+dp(16), ct+dp(218), cw-dp(32), dp(20))
		aw := dp(110)
		place(cAccent, cx+dp(16), ct+dp(240), aw, dp(28))
		place(cAccentText, cx+dp(16)+aw+dp(8), ct+dp(240), aw, dp(28))
		place(cAccentReset, cx+dp(24)+aw*2+dp(8), ct+dp(240), dp(72), dp(28))
	}
	place(stTraySection, cx+dp(16), ct+dp(248)+accOff, cw-dp(32), dp(20))
	trayY := ct + dp(272) + accOff
	trayW := (cw - dp(32) - dp(16) - 3*dp(8) - dp(80)) / 4
	trayIDs := []int{cTrayV, cTrayVBg, cTrayE, cTrayEBg}
	for i, id := range trayIDs {
		tx := cx + dp(16) + int32(i)*(trayW+dp(8))
		place(id, tx, trayY, trayW, dp(34))
	}
	place(cTrayReset, cx+cw-dp(16)-dp(72), trayY, dp(72), dp(34))
	place(stTrayHint, cx+dp(16), ct+dp(310)+accOff, cw-dp(32), dp(20))

	// Tab 5 — Giới thiệu (button sits inside the second card)
	place(cUpdateBtn, cx+cw-dp(20)-dp(170), ct+dp(317), dp(170), dp(32))

	_ = winH
	pInvalidateRect.Call(gSettingsHwnd, 0, 1)
}

func navRect(i int) rect {
	y := dp(56) + int32(i)*dp(52)
	return rect{dp(10), y, railW - dp(10), y + dp(46)}
}

func navDirtyRect(i int) rect {
	r := navRect(i)
	pad := dp(3)
	return rect{r.left - pad, r.top - pad, r.right + pad, r.bottom + pad}
}

// navHitAt maps a sidebar point to a tab index, splitting the 6dp gap
// between adjacent pills at its midpoint so vertical cursor sweeps transition
// smoothly from tab to tab without flashing an unhovered (-1) intermediate frame.
func navHitAt(x, y int32) int {
	if x < dp(10) || x >= railW-dp(10) {
		return -1
	}
	firstTop := dp(56)
	pitch := dp(52)
	totalEnd := firstTop + int32(len(tabItems))*pitch - dp(6)
	if y < firstTop || y >= totalEnd {
		return -1
	}
	idx := int((y - firstTop + dp(3)) / pitch)
	if idx < 0 || idx >= len(tabItems) {
		return -1
	}
	return idx
}

func rectsIntersect(a, b rect) bool {
	return a.left < b.right && a.right > b.left && a.top < b.bottom && a.bottom > b.top
}

func rectUnion(a, b rect) rect {
	if a.right <= a.left || a.bottom <= a.top {
		return b
	}
	if b.right <= b.left || b.bottom <= b.top {
		return a
	}
	if b.left < a.left {
		a.left = b.left
	}
	if b.top < a.top {
		a.top = b.top
	}
	if b.right > a.right {
		a.right = b.right
	}
	if b.bottom > a.bottom {
		a.bottom = b.bottom
	}
	return a
}

const (
	SWP_NOREDRAW   = 0x0008
	SWP_SHOWWINDOW = 0x0040
	SWP_HIDEWINDOW = 0x0080
)

func setTab(i int) {
	gActiveTab = i
	gStaticValid = false
	for _, hctl := range allCtls {
		t := ctlTab[hctl]
		flag := uintptr(SWP_HIDEWINDOW)
		if t < 0 || t == i {
			flag = SWP_SHOWWINDOW
		}
		pSetWindowPos.Call(hctl, 0, 0, 0, 0, 0,
			SWP_NOMOVE|SWP_NOSIZE|SWP_NOZORDER|SWP_NOACTIVATE|SWP_NOREDRAW|flag)
	}
	redrawSettings()
}

// redrawSettings repaints the parent first, then every child, so children
// copy a background that already belongs to the current tab.
func redrawSettings() {
	gStaticValid = false
	if gSettingsHwnd != 0 {
		pRedrawWindow.Call(gSettingsHwnd, 0, 0, RDW_INVALIDATE|RDW_ALLCHILDREN|RDW_UPDATENOW)
	}
}

// Three-stage offscreen cache:
//   - gBackdropDC: full-window atmospheric gradient + color blooms + top header base + divider
//                  (depends only on window size, style, and dark mode)
//   - gStaticDC:   gBackdropDC + active tab's static content cards/text/mascot
//                  (depends on window size, style, dark mode, active tab, and effectiveViet)
//   - gBgDC:       complete composite frame (gStaticDC + window controls + sidebar nav pills)
//                  kept so child controls copy their slice via fillParentBg, and hover changes
//                  only BitBlt + redraw the 1-2 affected sidebar pills or caption buttons.
var (
	gBackdropDC, gBackdropBmp, gBackdropOld uintptr
	gBackdropW, gBackdropH                  int32
	gBackdropStyle, gBackdropLang           int
	gBackdropDark, gBackdropValid           bool

	gStaticDC, gStaticBmp, gStaticOld      uintptr
	gStaticW, gStaticH                     int32
	gStaticTab, gStaticStyle, gStaticLang  int
	gStaticDark, gStaticViet, gStaticValid bool

	gBgDC, gBgBmp, gBgOld uintptr
	gBgW, gBgH            int32
	gMouseTracking        bool
)

func releaseBgCache() {
	if gBgDC != 0 {
		pSelectObject.Call(gBgDC, gBgOld)
		pDeleteDC.Call(gBgDC)
	}
	if gBgBmp != 0 {
		pDeleteObject.Call(gBgBmp)
	}
	gBgDC, gBgBmp, gBgOld, gBgW, gBgH = 0, 0, 0, 0, 0

	if gStaticDC != 0 {
		pSelectObject.Call(gStaticDC, gStaticOld)
		pDeleteDC.Call(gStaticDC)
	}
	if gStaticBmp != 0 {
		pDeleteObject.Call(gStaticBmp)
	}
	gStaticDC, gStaticBmp, gStaticOld, gStaticW, gStaticH = 0, 0, 0, 0, 0
	gStaticValid = false

	if gBackdropDC != 0 {
		pSelectObject.Call(gBackdropDC, gBackdropOld)
		pDeleteDC.Call(gBackdropDC)
	}
	if gBackdropBmp != 0 {
		pDeleteObject.Call(gBackdropBmp)
	}
	gBackdropDC, gBackdropBmp, gBackdropOld, gBackdropW, gBackdropH = 0, 0, 0, 0, 0
	gBackdropValid = false

	releaseMascotCache()
}

// fillParentBg paints rc (child client coords) with the parent pixels
// behind child.
func fillParentBg(dc uintptr, rc rect, child uintptr) {
	if gBgDC == 0 || child == 0 || gSettingsHwnd == 0 {
		pFillRect.Call(dc, uintptr(unsafe.Pointer(&rc)), brBg)
		return
	}
	var pt point
	pMapWindowPoints.Call(child, gSettingsHwnd, uintptr(unsafe.Pointer(&pt)), 1)
	pBitBlt.Call(dc, uintptr(rc.left), uintptr(rc.top),
		uintptr(rc.right-rc.left), uintptr(rc.bottom-rc.top),
		gBgDC, uintptr(pt.x+rc.left), uintptr(pt.y+rc.top), SRCCOPY)
}

func imIdx(im int) int {
	for i, v := range imList {
		if v == im {
			return i
		}
	}
	return 0
}

// repaintRadioGroup refreshes the four "Chế độ gõ" cards. They are child
// HWNDs — InvalidateRect on the parent never reaches them, so without this
// the previously selected card kept its stale "selected" pixels next to the
// freshly clicked one.
func repaintRadioGroup() {
	for _, id := range []int{cRadioTelex, cRadioTelexSimple, cRadioVni, cRadioViqr} {
		if h := ctlHnd[id]; h != 0 {
			pInvalidateRect.Call(h, 0, 1)
		}
	}
}

func setIM(im int) {
	setInputMethod(im)
	if ctlHnd[cIMCombo] != 0 {
		pSendMessage.Call(ctlHnd[cIMCombo], CB_SETCURSEL, uintptr(imIdx(im)), 0)
	}
	if gSettingsHwnd != 0 {
		pInvalidateRect.Call(gSettingsHwnd, 0, 1)
	}
}

func syncControls() {
	if gSettingsHwnd == 0 {
		return
	}
	pSendMessage.Call(ctlHnd[cIMCombo], CB_SETCURSEL, uintptr(imIdx(gIM)), 0)
	pSendMessage.Call(ctlHnd[cCharset], CB_SETCURSEL, uintptr(cfg.Charset), 0)
	for _, tid := range []int{
		cThemeAuto, cThemeLight, cThemeDark,
		cThemeBasic, cThemeJade, cThemeIosPure, cThemeCyberNeon, cThemeMatchaLatte, cThemeMaterialTeal,
	} {
		if ctlHnd[tid] != 0 {
			pInvalidateRect.Call(ctlHnd[tid], 0, 1)
		}
	}

	o := gEngine.GetOptions()
	setCheck(cModern, o.ModernStyle != 0)
	setCheck(cFreeMark, o.FreeMarking != 0)
	setCheck(cSpell, o.SpellCheckEnabled != 0)
	setCheck(cRestore, o.AutoNonVnRestore != 0)
	setCheck(cMacro, o.MacroEnabled != 0)
	setCheck(cMacroAlways, o.AlwaysMacro != 0)
	setCheck(cSkipLayout, cfg.SkipNonUSLayout)
	setCheck(cAutoCap, cfg.AutoCap)
	setCheck(cClipboard, cfg.UseClipboard)
	setCheck(cStartup, cfg.RunAtStartup)
	setCheck(cAdmin, cfg.RunAsAdmin)
	setCheck(cShowDlg, cfg.ShowOnLaunch)
	setCheck(cSound, cfg.SoundOnToggle)
	setCheck(cShowHud, cfg.ShowHud)
	setCheck(cRememberAppMode, cfg.RememberAppMode)
	setCheck(cModCtrl, cfg.HotkeyMods&1 != 0)
	setCheck(cModShift, cfg.HotkeyMods&2 != 0)
	setCheck(cModAlt, cfg.HotkeyMods&4 != 0)
	setCheck(cModWin, cfg.HotkeyMods&8 != 0)
	setCheck(cUseCtrlShift, cfg.UseCtrlShift)
	setCheck(cF1, cfg.FKeys&1 != 0)
	setCheck(cF2, cfg.FKeys&2 != 0)
	setCheck(cF5, cfg.FKeys&4 != 0)
	setCheck(cF9, cfg.FKeys&8 != 0)
	setCheck(cF12, cfg.FKeys&16 != 0)

	keyTxt := ""
	if cfg.HotkeyVk != 0 {
		keyTxt = string(rune(cfg.HotkeyVk))
	}
	pSetWndText.Call(ctlHnd[cHotkeyBox], uintptr(unsafe.Pointer(utf16ptr(keyTxt))))
}

const excludeTemplate = `; App loại trừ — mỗi dòng một tên .exe
;   notepad.exe        tắt gõ trong app này, Alt+Z bật tạm cho riêng app
;   notepad.exe|lock   tắt hẳn, Alt+Z không tác dụng
;   notepad.exe|clip   gõ được, gửi chữ qua Clipboard
;   cad.exe|tcvn       gõ được, xuất bảng mã TCVN3
;   cad.exe|vni        dùng kiểu gõ VNI riêng cho app này
;   cad.exe|tcvn,vni   kết hợp nhiều tuỳ chọn bằng dấu phẩy
`

// openFileNameW mirrors Win32 OPENFILENAMEW (152 bytes on 64-bit).
// Go pads uintptr fields automatically — do NOT add manual padding.
type openFileNameW struct {
	lStructSize       uint32
	hwndOwner         uintptr
	hInstance         uintptr
	lpstrFilter       uintptr
	lpstrCustomFilter uintptr
	nMaxCustFilter    uint32
	nFilterIndex      uint32
	lpstrFile         uintptr
	nMaxFile          uint32
	lpstrFileTitle    uintptr
	nMaxFileTitle     uint32
	lpstrInitialDir   uintptr
	lpstrTitle        uintptr
	flags             uint32
	nFileOffset       uint16
	nFileExtension    uint16
	lpstrDefExt       uintptr
	lCustData         uintptr
	lpfnHook          uintptr
	lpTemplateName    uintptr
	pvReserved        uintptr
	dwReserved        uint32
	flagsEx           uint32
}

const (
	OFN_HIDEREADONLY  = 0x00000004
	OFN_PATHMUSTEXIST = 0x00000800
	OFN_FILEMUSTEXIST = 0x00001000
)

func init() {
	if unsafe.Sizeof(openFileNameW{}) != 152 {
		panic("OPENFILENAMEW struct size mismatch")
	}
}

func utf16buf(s string) *uint16 {
	u := utf16.Encode([]rune(s))
	u = append(u, 0)
	return &u[0]
}

var pGetOpenFileName = comdlg32.NewProc("GetOpenFileNameW")
var pChooseColor = comdlg32.NewProc("ChooseColorW")

// chooseColorW mirrors Win32 CHOOSECOLORW (36 bytes on 64-bit).
type chooseColorW struct {
	lStructSize  uint32
	hwndOwner    uintptr
	hInstance    uintptr
	rgbResult    uint32
	lpCustColors *uint32
	flags        uint32
	lCustData    uintptr
	lpfnHook     uintptr
	lpTemplate   *uint16
}

var gCustColors [16]uint32 // custom-color slots persist for the session

// pickColor opens the standard color dialog seeded with the given RGB
// (0xRRGGBB). Returns the chosen RGB and false on cancel.
func pickColor(seed int) (int, bool) {
	cc := chooseColorW{
		lStructSize:  uint32(unsafe.Sizeof(chooseColorW{})),
		hwndOwner:    gSettingsHwnd,
		lpCustColors: &gCustColors[0],
		flags:        0x0001 | 0x0002, // CC_INIT | CC_FULLOPEN
	}
	if seed >= 0 {
		cc.rgbResult = uint32(rgb2ref(seed))
	} else {
		cc.flags &^= 0x0001
	}
	r, _, _ := pChooseColor.Call(uintptr(unsafe.Pointer(&cc)))
	if r == 0 {
		return 0, false
	}
	// ChooseColor returns COLORREF (BGR) — flip back to stored RGB.
	c := int(cc.rgbResult)
	return (c&0xFF)<<16 | (c & 0xFF00) | (c>>16)&0xFF, true
}

func browseExe() string {
	var buf [1024]uint16
	filter := utf16buf(tr("Ứng dụng (*.exe)\x00*.exe\x00Tất cả file (*.*)\x00*.*\x00", "Applications (*.exe)\x00*.exe\x00All files (*.*)\x00*.*\x00"))
	title := utf16ptr(tr("Chọn app cần loại trừ", "Select application to exclude"))
	ofn := openFileNameW{
		lStructSize: uint32(unsafe.Sizeof(openFileNameW{})),
		hwndOwner:   gSettingsHwnd,
		lpstrFilter: uintptr(unsafe.Pointer(filter)),
		lpstrFile:   uintptr(unsafe.Pointer(&buf[0])),
		nMaxFile:    uint32(len(buf)),
		lpstrTitle:  uintptr(unsafe.Pointer(title)),
		lpstrDefExt: uintptr(unsafe.Pointer(utf16ptr("exe"))),
		flags:       OFN_HIDEREADONLY | OFN_PATHMUSTEXIST | OFN_FILEMUSTEXIST,
	}
	r, _, _ := pGetOpenFileName.Call(uintptr(unsafe.Pointer(&ofn)))
	if r == 0 {
		return ""
	}
	return windows.UTF16ToString(buf[:])
}

func appendExclude(name string) {
	p := filepath.Join(exeDir(), "exclude.txt")
	if _, err := os.Stat(p); os.IsNotExist(err) {
		os.WriteFile(p, []byte(excludeTemplate), 0644)
	}
	f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s\n", name)
}

func editFile(name string) {
	p := filepath.Join(exeDir(), name)
	if _, err := os.Stat(p); os.IsNotExist(err) {
		content := "; " + name + "\n"
		if name == "exclude.txt" {
			content = excludeTemplate
		}
		os.WriteFile(p, []byte(content), 0644)
	}
	pShellExecute.Call(0, uintptr(unsafe.Pointer(utf16ptr("open"))),
		uintptr(unsafe.Pointer(utf16ptr("notepad.exe"))),
		uintptr(unsafe.Pointer(utf16ptr(p))), 0, SW_SHOW)
}

func showLangMenu() {
	hMenu, _, _ := pCreatePopupMenu.Call()
	defer pDestroyMenu.Call(hMenu)

	flagVi := uintptr(0)
	flagEn := uintptr(0)
	if cfg.UILang == 0 {
		flagVi = MF_CHECKED
	} else {
		flagEn = MF_CHECKED
	}
	pAppendMenu.Call(hMenu, flagVi, 1, uintptr(unsafe.Pointer(utf16ptr("Tiếng Việt"))))
	pAppendMenu.Call(hMenu, flagEn, 2, uintptr(unsafe.Pointer(utf16ptr("English"))))

	var pt point
	pGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))

	r, _, _ := pTrackPopupMenu.Call(hMenu, 0x0100|0x0002,
		uintptr(pt.x), uintptr(pt.y), 0, gSettingsHwnd, 0)
	newLang := cfg.UILang
	switch r {
	case 1:
		newLang = 0
	case 2:
		newLang = 1
	default:
		return
	}
	if newLang != cfg.UILang {
		cfg.UILang = newLang
		saveSettings()
		syncTrayState()
		rethemed = true
		pDestroyWindow.Call(gSettingsHwnd)
	}
}

// launchUpdater runs the optional companion updater (-now = skip throttle,
// always report). tuigo.exe itself stays offline; deleting the updater exe
// disables update checks entirely.
func launchUpdater(now bool) {
	up := filepath.Join(exeDir(), "tuigo-updater.exe")
	if _, err := os.Stat(up); err != nil {
		pMessageBox.Call(gSettingsHwnd,
			uintptr(unsafe.Pointer(utf16ptr(tr("Không thấy tuigo-updater.exe cạnh tuigo.exe.\nTải bản zip đầy đủ hoặc tải trên trang Releases của GitHub.", "tuigo-updater.exe was not found next to tuigo.exe.\nDownload the full zip package from GitHub Releases.")))),
			uintptr(unsafe.Pointer(utf16ptr("Tui Gõ"))), 0x30)
		return
	}
	if !updaterRunning.CompareAndSwap(false, true) {
		return
	}
	var args []string
	if now {
		args = []string{"-now"}
	}
	// Same reason as the startup spawn in main.go: process creation can block
	// 10-15s on AV/SmartScreen scans, and this thread owns the LL hooks.
	go func() {
		defer updaterRunning.Store(false)
		defer func() { _ = recover() }()
		c := exec.Command(up, args...)
		c.Dir = exeDir()
		if c.Start() == nil {
			c.Wait()
		}
	}()
}

var updaterRunning atomic.Bool

func settingsCommand(id int) {
	o := gEngine.GetOptions()
	switch id {
	case cUpdateBtn:
		launchUpdater(true)
		return
	case cRadioTelex:
		setIM(ImTelex)
		return
	case cRadioTelexSimple:
		setIM(ImTelexSimple)
		return
	case cRadioVni:
		setIM(ImVni)
		return
	case cRadioViqr:
		setIM(ImViqr)
		return
	case cLangBtn:
		showLangMenu()
		return
	case cIMCombo:
		sel := comboSel(cIMCombo)
		if sel >= 0 && sel < len(imList) {
			setIM(imList[sel])
		}
		return
	case cModern:
		swState[cModern] = !swState[cModern]
		o.ModernStyle = b2i32(swState[cModern])
		pInvalidateRect.Call(ctlHnd[cModern], 0, 1)
	case cFreeMark:
		swState[cFreeMark] = !swState[cFreeMark]
		o.FreeMarking = b2i32(swState[cFreeMark])
		pInvalidateRect.Call(ctlHnd[cFreeMark], 0, 1)
	case cSpell:
		swState[cSpell] = !swState[cSpell]
		o.SpellCheckEnabled = b2i32(swState[cSpell])
		pInvalidateRect.Call(ctlHnd[cSpell], 0, 1)
	case cRestore:
		swState[cRestore] = !swState[cRestore]
		o.AutoNonVnRestore = b2i32(swState[cRestore])
		pInvalidateRect.Call(ctlHnd[cRestore], 0, 1)
	case cMacro:
		o.MacroEnabled = b2i32(getCheck(cMacro))
	case cMacroAlways:
		o.AlwaysMacro = b2i32(getCheck(cMacroAlways))
	case cSkipLayout:
		cfg.SkipNonUSLayout = getCheck(cSkipLayout)
	case cAutoCap:
		swState[cAutoCap] = !swState[cAutoCap]
		cfg.AutoCap = swState[cAutoCap]
		pInvalidateRect.Call(ctlHnd[cAutoCap], 0, 1)
	case cClipboard:
		cfg.UseClipboard = getCheck(cClipboard)
	case cStartup:
		swState[cStartup] = !swState[cStartup]
		cfg.RunAtStartup = swState[cStartup]
		applyStartup()
		pInvalidateRect.Call(ctlHnd[cStartup], 0, 1)
	case cAdmin:
		cfg.RunAsAdmin = getCheck(cAdmin)
		if cfg.RunAsAdmin && !isElevated() {
			saveSettings()
			relaunchElevated()
			return
		}
		applyStartup()
	case cShowDlg:
		cfg.ShowOnLaunch = getCheck(cShowDlg)
	case cSound:
		cfg.SoundOnToggle = getCheck(cSound)
	case cShowHud:
		cfg.ShowHud = getCheck(cShowHud)
		hudUpdate()
	case cRememberAppMode:
		setRememberAppMode(getCheck(cRememberAppMode))
		syncTrayState()
	case cModCtrl, cModShift, cModAlt, cModWin:
		m := 0
		for i, mid := range []int{cModCtrl, cModShift, cModAlt, cModWin} {
			if getCheck(mid) {
				m |= 1 << i
			}
		}
		cfg.HotkeyMods = m
	case cUseCtrlShift:
		cfg.UseCtrlShift = getCheck(cUseCtrlShift)
	case cF1, cF2, cF5, cF9, cF12:
		f := 0
		for i, fid := range []int{cF1, cF2, cF5, cF9, cF12} {
			if getCheck(fid) {
				f |= 1 << i
			}
		}
		cfg.FKeys = f
	case cMacroEdit:
		editFile("macro.txt")
	case cExcludeBrowse:
		if p := browseExe(); p != "" {
			appendExclude(filepath.Base(p))
			loadExclusions(exeDir())
		}
	case cExcludeEdit:
		editFile("exclude.txt")
		loadExclusions(exeDir())
	case cTrayV, cTrayVBg, cTrayE, cTrayEBg:
		slot := &cfg.TrayV
		seed := cfg.TrayV
		switch id {
		case cTrayVBg:
			slot, seed = &cfg.TrayVBg, cfg.TrayVBg
		case cTrayE:
			slot, seed = &cfg.TrayE, cfg.TrayE
		case cTrayEBg:
			slot, seed = &cfg.TrayEBg, cfg.TrayEBg
		}
		if seed < 0 {
			seed = traySwatchColor(id) // start picker at the effective color
		}
		if rgb, ok := pickColor(seed); ok {
			*slot = rgb
			saveSettings()
			syncTrayState()
			pInvalidateRect.Call(ctlHnd[id], 0, 1)
		}
		return
	case cTrayReset:
		cfg.TrayV, cfg.TrayVBg, cfg.TrayE, cfg.TrayEBg = -1, -1, -1, -1
		saveSettings()
		syncTrayState()
		for _, tid := range []int{cTrayV, cTrayVBg, cTrayE, cTrayEBg} {
			pInvalidateRect.Call(ctlHnd[tid], 0, 1)
		}
		return
	case cAccent:
		seed := cfg.Accent
		if seed < 0 {
			seed = int(argbAccent & 0x00FFFFFF)
		}
		if rgb, ok := pickColor(seed); ok {
			cfg.Accent = rgb
			saveSettings()
			rethemed = true
			pDestroyWindow.Call(gSettingsHwnd)
		}
		return
	case cAccentText:
		seed := cfg.AccentText
		if seed < 0 {
			// seed with the auto-picked ink so the dialog opens at current
			ac := argbAccent & 0x00FFFFFF
			lum := 0.299*float64(ac>>16&0xFF) + 0.587*float64(ac>>8&0xFF) + 0.114*float64(ac&0xFF)
			if lum > 140 {
				seed = 0x0F1115
			} else {
				seed = 0xFFFFFF
			}
		}
		if rgb, ok := pickColor(seed); ok {
			cfg.AccentText = rgb
			saveSettings()
			rethemed = true
			pDestroyWindow.Call(gSettingsHwnd)
		}
		return
	case cAccentReset:
		if cfg.Accent >= 0 || cfg.AccentText >= 0 {
			cfg.Accent, cfg.AccentText = -1, -1
			saveSettings()
			rethemed = true
			pDestroyWindow.Call(gSettingsHwnd)
		}
		return
	case cThemeAuto, cThemeLight, cThemeDark:
		newTheme := themeAuto
		switch id {
		case cThemeLight:
			newTheme = themeLight
		case cThemeDark:
			newTheme = themeDark
		}
		if newTheme != cfg.Theme {
			cfg.Theme = newTheme
			saveSettings()
			rethemed = true
			pDestroyWindow.Call(gSettingsHwnd)
			return
		}
	case cThemeBasic, cThemeJade, cThemeIosPure, cThemeCyberNeon, cThemeMatchaLatte, cThemeMaterialTeal:
		newStyle := styleBasic
		switch id {
		case cThemeJade:
			newStyle = styleJade
		case cThemeIosPure:
			newStyle = styleIosPure
		case cThemeCyberNeon:
			newStyle = styleCyberNeon
		case cThemeMatchaLatte:
			newStyle = styleMatchaLatte
		case cThemeMaterialTeal:
			newStyle = styleMaterialTeal
		}
		if newStyle != cfg.Style {
			cfg.Style = newStyle
			saveSettings()
			rethemed = true
			pDestroyWindow.Call(gSettingsHwnd)
			return
		}
	}
	gEngine.SetOptions(o)
	saveSettings()
}

func b2i32(b bool) int32 {
	if b {
		return 1
	}
	return 0
}

var swIDs = map[int]bool{}

type minmaxinfo struct {
	ptReserved     point
	ptMaxSize      point
	ptMaxPosition  point
	ptMinTrackSize point
	ptMaxTrackSize point
}

func settingsProc(hwnd uintptr, msg uint32, wp, lp uintptr) uintptr {
	defer func() { _ = recover() }()
	switch msg {
	case WM_CREATE:
		gHoveredTab = -1
		gCaptionHover, gCaptionDown = -1, -1
		gMouseTracking = false
		buildControls(hwnd)
		return 0
	case WM_MOUSEMOVE:
		x, y := int32(int16(lp&0xFFFF)), int32(int16((lp>>16)&0xFFFF))
		winW := gBgW
		if winW <= 0 {
			var client rect
			pGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&client)))
			winW = client.right
		}
		caption := captionAt(winW, x, y)
		if caption != gCaptionHover {
			gCaptionHover = caption
			r := captionBarRect(winW)
			pInvalidateRect.Call(hwnd, uintptr(unsafe.Pointer(&r)), 0)
		}
		hovered := navHitAt(x, y)
		if hovered != gHoveredTab {
			oldHover := gHoveredTab
			gHoveredTab = hovered
			if oldHover >= 0 && oldHover < len(tabItems) {
				r := navDirtyRect(oldHover)
				pInvalidateRect.Call(hwnd, uintptr(unsafe.Pointer(&r)), 0)
			}
			if hovered >= 0 && hovered < len(tabItems) {
				r := navDirtyRect(hovered)
				pInvalidateRect.Call(hwnd, uintptr(unsafe.Pointer(&r)), 0)
			}
		}
		if !gMouseTracking {
			track := navMouseTrack{flags: 2 /* TME_LEAVE */, hwnd: hwnd}
			track.size = uint32(unsafe.Sizeof(track))
			if r, _, _ := pTrackMouseEvent.Call(uintptr(unsafe.Pointer(&track))); r != 0 {
				gMouseTracking = true
			}
		}
		return 0
	case 0x02A3: // WM_MOUSELEAVE
		gMouseTracking = false
		if gCaptionHover != -1 {
			gCaptionHover = -1
			var client rect
			pGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&client)))
			r := captionBarRect(client.right)
			pInvalidateRect.Call(hwnd, uintptr(unsafe.Pointer(&r)), 0)
		}
		if gHoveredTab != -1 {
			oldHover := gHoveredTab
			gHoveredTab = -1
			if oldHover >= 0 && oldHover < len(tabItems) {
				r := navDirtyRect(oldHover)
				pInvalidateRect.Call(hwnd, uintptr(unsafe.Pointer(&r)), 0)
			}
		}
		return 0
	case WM_SIZE:
		if wp != 1 {
			layoutAll()
			redrawSettings()
		}
		return 0
	case WM_GETMINMAXINFO:
		r, _, _ := pDefWindowProc.Call(hwnd, uintptr(msg), wp, lp)
		mmi := (*minmaxinfo)(*(*unsafe.Pointer)(unsafe.Pointer(&lp)))
		// Min size = smallest that still fits the tallest tab's content
		// (~ct106 + 430px of controls + footer margin).
		mmi.ptMinTrackSize = point{dp(700), dp(520)}
		return r
	case WM_DPICHANGED:
		if d := int32(wp & 0xFFFF); d > 0 {
			gDpi = d
		}
		rebuildFonts()
		sr := (*rect)(*(*unsafe.Pointer)(unsafe.Pointer(&lp)))
		pSetWindowPos.Call(hwnd, 0, uintptr(sr.left), uintptr(sr.top),
			uintptr(sr.right-sr.left), uintptr(sr.bottom-sr.top),
			SWP_NOZORDER|SWP_NOACTIVATE)
		updateSettingsRegion(hwnd)
		layoutAll()
		return 0
	case WM_COMMAND:
		id := int(wp & 0xFFFF)
		code := int(wp >> 16)
		if code == BN_CLICKED {
			if swIDs[id] {
				setCheck(id, !getCheck(id))
			}
			settingsCommand(id)
			return 0
		}
		if code == CBN_SELCHANGE {
			switch id {
			case cIMCombo:
				settingsCommand(cIMCombo)
			case cCharset:
				sel := comboSel(cCharset)
				if sel >= 0 {
					cfg.Charset = sel
					saveSettings()
				}
			case cTheme:
				sel := comboSel(cTheme)
				if sel >= 0 && sel != cfg.Theme {
					cfg.Theme = sel
					saveSettings()
					rethemed = true
					pDestroyWindow.Call(hwnd)
				}
			}
			return 0
		}
		if code == 0x0300 && id == cHotkeyBox {
			var buf [8]uint16
			pGetWndText.Call(ctlHnd[cHotkeyBox],
				uintptr(unsafe.Pointer(&buf[0])), 8)
			cfg.HotkeyVk = 0
			if c := buf[0]; c != 0 {
				if c >= 'a' && c <= 'z' {
					c -= 32
				}
				cfg.HotkeyVk = int(c)
			}
			saveSettings()
			return 0
		}
	case WM_LBUTTONDOWN:
		var rc rect
		pGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rc)))
		winW := rc.right
		x := int32(int16(lp & 0xFFFF))
		y := int32(int16(lp >> 16))

		// Capture the whole button. Activating on mouse-up avoids accidental
		// close/minimize when the pointer is dragged away.
		if id := captionAt(winW, x, y); id != -1 {
			gCaptionHover, gCaptionDown = id, id
			pSetCapture.Call(hwnd)
			r := captionRect(winW, id)
			pInvalidateRect.Call(hwnd, uintptr(unsafe.Pointer(&r)), 0)
			return 0
		}

		// Top header area: window dragging
		if y < dp(46) {
			// Drag window natively
			pReleaseCapture.Call()
			pSendMessage.Call(hwnd, 0x00A1, 2, 0)
			return 0
		}

		// Sidebar tabs
		if x < railW {
			if idx := navHitAt(x, y); idx >= 0 {
				if idx != gActiveTab {
					setTab(idx)
				}
				return 0
			}
		}
	case WM_LBUTTONUP:
		if gCaptionDown != -1 {
			id := gCaptionDown
			gCaptionDown = -1
			pReleaseCapture.Call()
			var client rect
			pGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&client)))
			x, y := int32(int16(lp&0xFFFF)), int32(int16((lp>>16)&0xFFFF))
			r := captionRect(client.right, id)
			pInvalidateRect.Call(hwnd, uintptr(unsafe.Pointer(&r)), 0)
			if captionAt(client.right, x, y) == id {
				switch id {
				case captionMinimize:
					pShowWindow.Call(hwnd, SW_MINIMIZE)
				case captionMaximize:
					toggleSettingsMaximize(hwnd)
				case captionClose:
					// Destroy, same as WM_CLOSE: hiding keeps ~10MB of
					// controls and buffers alive with no path to free them.
					pDestroyWindow.Call(hwnd)
				}
			}
			return 0
		}
	case 0x0215: // WM_CAPTURECHANGED
		if gCaptionDown != -1 {
			gCaptionDown = -1
			pInvalidateRect.Call(hwnd, 0, 0)
		}
		return 0
	case WM_LBUTTONDBLCLK:
		x, y := int32(int16(lp&0xFFFF)), int32(int16((lp>>16)&0xFFFF))
		var client rect
		pGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&client)))
		if y >= 0 && y < dp(46) && captionAt(client.right, x, y) == -1 {
			toggleSettingsMaximize(hwnd)
			return 0
		}

	case WM_DRAWITEM:
		di := (*drawItemStruct)(*(*unsafe.Pointer)(unsafe.Pointer(&lp)))
		if di.ctlType == ODT_BUTTON {
			id := int(di.ctlID)
			switch {
			case isCheckbox[id]:
				drawSmoothCheckbox(di.hDC, di.rcItem, id, swState[id])
			case isRadio[id]:
				selected := false
				switch id {
				case cRadioTelex:
					selected = (gIM == ImTelex)
				case cRadioTelexSimple:
					selected = (gIM == ImTelexSimple)
				case cRadioVni:
					selected = (gIM == ImVni)
				case cRadioViqr:
					selected = (gIM == ImViqr)
				}
				drawSmoothRadioCard(di.hDC, di.rcItem, id, selected)
			case id >= cModCtrl && id <= cModWin:
				pressed := di.itemState&ODS_SELECTED != 0
				drawSmoothModifierKeycap(di.hDC, di.rcItem, id, swState[id], pressed)
			case id == cLangBtn:
				drawLangBtn(di)
			case swIDs[id]:
				drawSmoothSwitch(di.hDC, di.rcItem, id, swState[id])
			case id == cThemeAuto || id == cThemeLight || id == cThemeDark:
				pressed := di.itemState&ODS_SELECTED != 0
				drawSmoothThemeButton(di.hDC, di.rcItem, id, pressed)
			case id == cThemeBasic || id == cThemeJade || id == cThemeIosPure || id == cThemeCyberNeon || id == cThemeMatchaLatte || id == cThemeMaterialTeal:
				pressed := di.itemState&ODS_SELECTED != 0
				drawSmoothStyleCard(di.hDC, di.rcItem, id, pressed)
			case id == cTrayV || id == cTrayVBg || id == cTrayE || id == cTrayEBg || id == cAccent || id == cAccentText:
				pressed := di.itemState&ODS_SELECTED != 0
				drawTraySwatch(di.hDC, di.rcItem, id, pressed)
			case btnIDs[id]:
				pressed := di.itemState&ODS_SELECTED != 0
				drawSmoothButton(di.hDC, di.rcItem, id, pressed)
			}
			return 1
		}
	case WM_CTLCOLORSTATIC, WM_CTLCOLORBTN:
		col := uintptr(colText)
		if goldCtls[lp] {
			col = colGold
		} else if mutedCtls[lp] {
			col = colMuted
		}
		pSetTextColor.Call(wp, col)
		pSetBkMode.Call(wp, TRANSPARENT)
		// A NULL brush alone never erases: labels kept the previous tab's
		// text underneath. Paint the parent's pixels first, then draw text.
		var crc rect
		pGetClientRect.Call(lp, uintptr(unsafe.Pointer(&crc)))
		fillParentBg(wp, crc, lp)
		nullB, _, _ := pGetStockObject.Call(NULL_BRUSH)
		return nullB
	case WM_CTLCOLORLISTBOX:
		pSetTextColor.Call(wp, colText)
		pSetBkMode.Call(wp, TRANSPARENT)
		return brCard
	case WM_CTLCOLOREDIT:
		if lp == ctlHnd[cHotkeyBox] {
			pSetTextColor.Call(wp, colJade)
			pSetBkMode.Call(wp, TRANSPARENT)
			return brRadioActive
		}
		pSetTextColor.Call(wp, colText)
		pSetBkMode.Call(wp, TRANSPARENT)
		return brCard
	case WM_ERASEBKGND:
		return 1 // WM_PAINT repaints everything — no separate erase pass
	case WM_PAINT:
		var ps paintStruct
		dc, _, _ := pBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		var rc rect
		pGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rc)))
		winW, winH := rc.right, rc.bottom
		if winW > 0 && winH > 0 {
			paintSettingsCached(dc, hwnd, winW, winH, ps.rcPaint)
		}
		pEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		return 0
	case WM_CLOSE:
		// Destroy rather than hide: a hidden settings window keeps ~10MB
		// of controls, paint buffers and Go heap alive forever. Reopening
		// rebuilds everything via WM_CREATE anyway.
		pDestroyWindow.Call(hwnd)
		return 0
	case WM_DESTROY:
		if gTooltipHwnd != 0 {
			pDestroyWindow.Call(gTooltipHwnd)
			gTooltipHwnd = 0
		}
		gSettingsHwnd = 0
		gCaptionHover, gCaptionDown = -1, -1
		gMouseTracking = false
		releaseBgCache()
		deleteGdiObj(&fontNormal)
		deleteGdiObj(&fontBold)
		deleteGdiObj(&fontTitle)
		deleteGdiObj(&fontSection)
		deleteGdiObj(&fontHint)
		deleteGdiObj(&fontTagline)
		if rethemed {
			rethemed = false
			applyTheme()
			openSettings()
		} else {
			// The window's controls and buffers are now garbage — hand the
			// freed pages back so idle RSS drops again after the UI closes.
			go func() {
				time.Sleep(200 * time.Millisecond)
				runtime.GC()
				debug.FreeOSMemory()
				// FreeOSMemory decommits free spans but they still count in
				// the working set until the memory manager trims — (HANDLE)-1,
				// -1, -1 asks Windows to do it now so Task Manager shows it.
				pTrimWorkingSet.Call(^uintptr(0), ^uintptr(0), ^uintptr(0))
			}()
		}
		return 0
	}
	r, _, _ := pDefWindowProc.Call(hwnd, uintptr(msg), wp, lp)
	return r
}

func allocOffscreenDC(refDC uintptr, w, h int32) (mem, bmp, old uintptr) {
	mem, _, _ = pCreateCompatibleDC.Call(refDC)
	bmp, _, _ = pCreateCompatibleBitmap.Call(refDC, uintptr(w), uintptr(h))
	if mem != 0 && bmp != 0 {
		old, _, _ = pSelectObject.Call(mem, bmp)
		return mem, bmp, old
	}
	if bmp != 0 {
		pDeleteObject.Call(bmp)
	}
	if mem != 0 {
		pDeleteDC.Call(mem)
	}
	return 0, 0, 0
}

func paintSettingsBackdrop(dc uintptr, winW, winH int32, dark bool) {
	gfx := NewGfx(dc)
	if gfx != nil {
		gfx.FillAtmosphericGlassBg(float32(winW), float32(winH), dark)
		if settingsHasRoundCorners() {
			gfx.DrawRoundedRect(0.5, 0.5, float32(winW)-1.0, float32(winH)-1.0, float32(dp(8)), argbBorder, 1.0)
			gfx.DrawLine(float32(dp(8)), 1.0, float32(winW-dp(8)), 1.0, HexAlphaARGB(0x40, 0xFFFFFF), 1.0)
		} else {
			gfx.DrawRect(0.5, 0.5, float32(winW)-1.0, float32(winH)-1.0, argbBorder, 1.0)
			gfx.DrawLine(1.0, 1.0, float32(winW)-1.0, 1.0, HexAlphaARGB(0x40, 0xFFFFFF), 1.0)
		}
		gfx.Destroy()
	} else {
		rc := rect{0, 0, winW, winH}
		pFillRect.Call(dc, uintptr(unsafe.Pointer(&rc)), brBg)
	}
	drawTopHeaderBase(dc, winW)
	divLine := rect{railW, dp(50), railW + 1, winH - dp(14)}
	pFillRect.Call(dc, uintptr(unsafe.Pointer(&divLine)), brBorder)
}

func paintSettingsTabContent(dc uintptr, tab int, winW, winH int32) {
	cx := railW + dp(24)
	cw := winW - cx - dp(24)
	ct := dp(106)
	switch tab {
	case 0:
		drawOverviewTab(dc, cx, ct, cw, winH)
	case 5:
		drawAboutTab(dc, cx, ct, cw, winH)
	default:
		drawSettingsTab(dc, tab, cx, ct, cw)
	}
}

func paintSettingsCached(dc, hwnd uintptr, winW, winH int32, dirty rect) {
	if gBgDC == 0 || gBgW != winW || gBgH != winH {
		releaseBgCache()
		gBgDC, gBgBmp, gBgOld = allocOffscreenDC(dc, winW, winH)
		if gBgDC != 0 {
			gBgW, gBgH = winW, winH
		}
		gStaticDC, gStaticBmp, gStaticOld = allocOffscreenDC(dc, winW, winH)
		if gStaticDC != 0 {
			gStaticW, gStaticH = winW, winH
		}
		gBackdropDC, gBackdropBmp, gBackdropOld = allocOffscreenDC(dc, winW, winH)
		if gBackdropDC != 0 {
			gBackdropW, gBackdropH = winW, winH
		}
	}
	if gBgDC == 0 || gStaticDC == 0 || gBackdropDC == 0 {
		paintSettings(dc, hwnd)
		return
	}

	dark := isDark()
	viet := effectiveViet()
	needBackdrop := !gBackdropValid || gBackdropStyle != cfg.Style || gBackdropDark != dark || gBackdropLang != cfg.UILang
	if needBackdrop {
		paintSettingsBackdrop(gBackdropDC, winW, winH, dark)
		gBackdropStyle = cfg.Style
		gBackdropDark = dark
		gBackdropLang = cfg.UILang
		gBackdropValid = true
		gStaticValid = false
	}

	needStatic := !gStaticValid || gStaticTab != gActiveTab || gStaticStyle != cfg.Style || gStaticDark != dark || gStaticViet != viet || gStaticLang != cfg.UILang
	if needStatic {
		pBitBlt.Call(gStaticDC, 0, 0, uintptr(winW), uintptr(winH), gBackdropDC, 0, 0, SRCCOPY)
		paintSettingsTabContent(gStaticDC, gActiveTab, winW, winH)
		gStaticTab = gActiveTab
		gStaticStyle = cfg.Style
		gStaticDark = dark
		gStaticViet = viet
		gStaticLang = cfg.UILang
		gStaticValid = true

		// Populate full composite gBgDC so child controls always read complete parent pixels via fillParentBg
		pBitBlt.Call(gBgDC, 0, 0, uintptr(winW), uintptr(winH), gStaticDC, 0, 0, SRCCOPY)
		drawWindowControls(gBgDC, hwnd, winW)
		drawNav(gBgDC)
		pBitBlt.Call(dc, 0, 0, uintptr(winW), uintptr(winH), gBgDC, 0, 0, SRCCOPY)
		return
	}

	// Fast path (sidebar hover, caption hover, or partial window expose):
	// gStaticDC is already up to date — only restore and repaint the dirty region!
	paintRc := dirty
	if paintRc.right <= paintRc.left || paintRc.bottom <= paintRc.top {
		paintRc = rect{0, 0, winW, winH}
	}
	restoreRc := paintRc
	capRc := captionBarRect(winW)
	if rectsIntersect(restoreRc, capRc) {
		restoreRc = rectUnion(restoreRc, capRc)
	}
	for i := range tabItems {
		nr := navDirtyRect(i)
		if rectsIntersect(restoreRc, nr) {
			restoreRc = rectUnion(restoreRc, nr)
		}
	}
	if restoreRc.left < 0 {
		restoreRc.left = 0
	}
	if restoreRc.top < 0 {
		restoreRc.top = 0
	}
	if restoreRc.right > winW {
		restoreRc.right = winW
	}
	if restoreRc.bottom > winH {
		restoreRc.bottom = winH
	}
	rw, rh := restoreRc.right-restoreRc.left, restoreRc.bottom-restoreRc.top
	if rw <= 0 || rh <= 0 {
		return
	}

	pBitBlt.Call(gBgDC, uintptr(restoreRc.left), uintptr(restoreRc.top),
		uintptr(rw), uintptr(rh), gStaticDC, uintptr(restoreRc.left), uintptr(restoreRc.top), SRCCOPY)
	if rectsIntersect(restoreRc, capRc) {
		drawWindowControls(gBgDC, hwnd, winW)
	}
	if restoreRc.left < railW {
		drawNavClipped(gBgDC, restoreRc)
	}
	pBitBlt.Call(dc, uintptr(restoreRc.left), uintptr(restoreRc.top),
		uintptr(rw), uintptr(rh), gBgDC, uintptr(restoreRc.left), uintptr(restoreRc.top), SRCCOPY)
}

// paintSettings renders the full client area into dc (fallback when offscreen DCs are unavailable).
func paintSettings(dc, hwnd uintptr) {
	var rc rect
	pGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rc)))
	winW, winH := rc.right, rc.bottom

	paintSettingsBackdrop(dc, winW, winH, isDark())
	drawWindowControls(dc, hwnd, winW)
	drawNav(dc)
	paintSettingsTabContent(dc, gActiveTab, winW, winH)
}

func openSettings() {
	if gSettingsHwnd != 0 {
		syncControls() // state may have changed via tray menu/hotkeys while hidden
		// Repaint while still hidden so the first visible frame is complete.
		pRedrawWindow.Call(gSettingsHwnd, 0, 0,
			RDW_INVALIDATE|RDW_ERASE|RDW_ALLCHILDREN|RDW_UPDATENOW)
		if iconic, _, _ := pIsIconic.Call(gSettingsHwnd); iconic != 0 {
			pShowWindow.Call(gSettingsHwnd, SW_RESTORE)
		} else {
			pShowWindow.Call(gSettingsHwnd, SW_SHOW)
		}
		pSetForegroundWnd.Call(gSettingsHwnd)
		return
	}
	cls := utf16ptr("BoGoSettingsWnd")
	if settingsProcCb == 0 {
		settingsProcCb = windows.NewCallback(settingsProc)
	}
	if gSettingsIcon == 0 {
		gSettingsIcon = loadAppIcon()
	}
	hIcon := gSettingsIcon
	wc := wndClassEx{
		cbSize:        uint32(unsafe.Sizeof(wndClassEx{})),
		style:         0x0008,         // CS_DBLCLKS: double-click titlebar to maximize
		lpfnWndProc:   settingsProcCb, // cached — each NewCallback grows the runtime's callback table
		hIcon:         hIcon,
		hIconSm:       hIcon,
		hCursor:       loadArrowCursor(),
		lpszClassName: cls,
	}
	pRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc)))

	// Resolve DPI BEFORE sizing: content layout uses dp() scaled by the real
	// DPI — a stale 96 here creates a window too small for scaled controls.
	gDpi = systemDpi()
	winW := dp(840)
	winH := dp(560)
	screenW, _, _ := pGetSystemMetrics.Call(0)
	screenH, _, _ := pGetSystemMetrics.Call(1)
	// Clamp to screen so small displays (or high DPI) never overflow.
	if winW > int32(screenW)-dp(40) {
		winW = int32(screenW) - dp(40)
	}
	if winH > int32(screenH)-dp(40) {
		winH = int32(screenH) - dp(40)
	}
	posX := (int32(screenW) - winW) / 2
	posY := (int32(screenH) - winH) / 2

	// WS_CLIPCHILDREN keeps the parent paint out of child controls —
	// otherwise every repaint overdraws them and they flicker back on top.
	gSettingsHwnd, _, _ = pCreateWindowEx.Call(
		0x00040000,
		uintptr(unsafe.Pointer(cls)),
		uintptr(unsafe.Pointer(utf16ptr(tr("Tui Gõ — Cài đặt", "Tui Gõ — Settings")))),
		0x80000000|0x02000000|0x00020000|0x00010000,
		uintptr(posX), uintptr(posY), uintptr(winW), uintptr(winH),
		0, 0, 0, 0)

	updateSettingsRegion(gSettingsHwnd)

	// Paint once while still hidden so the window never presents blank.
	pRedrawWindow.Call(gSettingsHwnd, 0, 0,
		RDW_INVALIDATE|RDW_ERASE|RDW_ALLCHILDREN|RDW_UPDATENOW)
	pShowWindow.Call(gSettingsHwnd, SW_SHOW)
	pSetForegroundWnd.Call(gSettingsHwnd)
}
