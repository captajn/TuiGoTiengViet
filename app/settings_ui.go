package main

// Settings window: calm jade accents, native Vietnamese typography,
// a five-tab vector sidebar, and compact settings groups.

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
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
	name string
	desc string
}{
	{"Tổng quan", "Trạng thái và tùy chọn chung"},
	{"Gõ tiếng Việt", "Telex, VNI và nâng cao"},
	{"Phím tắt", "Tùy chỉnh tổ hợp phím"},
	{"Giao diện", "Chủ đề và hiển thị"},
	{"Giới thiệu", "Thông tin ứng dụng"},
}

// ids of per-tab STATICs
var (
	stIMSection, stLbIM, stLbCharset                         int
	stSwitchSection, stLbHotkey, stHotkeyHint, stFKeySection int
	stMacroSection, stMacroHint1, stMacroHint2               int
	stSysSection, stLbTheme, stExcludeSection, stExcludeHint int
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
	cRadioVni
	cRadioViqr
	cLangBtn
	cUpdateBtn
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

// Top titlebar: minimalist brand, subtitle, and system-style window controls.
func drawTopHeader(dc uintptr, hwnd uintptr, winW int32) {
	gfx := NewGfx(dc)
	if gfx != nil {
		drawMinimalBrandMark(gfx, float32(dp(16)), float32(dp(8)), float32(dp(32)))
		gfx.Destroy()
	}

	pSetBkMode.Call(dc, TRANSPARENT)
	pSetTextColor.Call(dc, colText)
	oldF, _, _ := pSelectObject.Call(dc, fontTitle)
	trc := rect{dp(60), dp(7), dp(280), dp(29)}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr("Tui Gõ"))),
		^uintptr(0), uintptr(unsafe.Pointer(&trc)),
		DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

	pSetTextColor.Call(dc, colMuted)
	pSelectObject.Call(dc, fontHint)
	sbc := rect{dp(60), dp(27), dp(280), dp(43)}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr("Gõ tiếng Việt tối giản & hiện đại"))),
		^uintptr(0), uintptr(unsafe.Pointer(&sbc)),
		DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

	pSelectObject.Call(dc, oldF)

	drawWindowControls(dc, hwnd, winW)
}

// Sidebar navigation renderer with zero-aliased vector icons and liquid glass pills
func drawNav(dc uintptr) {
	gfx := NewGfx(dc)
	defer func() {
		if gfx != nil {
			gfx.Destroy()
		}
	}()

	for i := 0; i < len(tabItems); i++ {
		r := navRect(i)
		active := (i == gActiveTab)
		hover := (i == gHoveredTab)

		x, y, w, h := float32(r.left), float32(r.top), float32(r.right-r.left), float32(r.bottom-r.top)
		radius := float32(dp(8))

		if active {
			if gfx != nil {
				// Liquid glass active pill
				bgTop := HexAlphaARGB(0x35, argbAccent&0x00FFFFFF)
				bgBottom := HexAlphaARGB(0x18, argbAccent&0x00FFFFFF)
				gfx.FillRoundedRectGradient(x, y, w, h, radius, bgTop, bgBottom)
				gfx.DrawRoundedRect(x, y, w, h, radius, HexAlphaARGB(0x60, argbAccent&0x00FFFFFF), 1.0)
				// Left indicator bar
				gfx.FillRoundedRect(x+float32(dp(3)), y+float32(dp(8)), float32(dp(3)), h-float32(dp(16)), 1.5, argbAccent)
			}
		} else if hover {
			if gfx != nil {
				gfx.FillRoundedRect(x, y, w, h, radius, HexAlphaARGB(0x18, 0xFFFFFF))
			}
		}

		iconCol := argbMuted
		if active {
			iconCol = argbAccent
		}
		ix := float32(r.left + dp(16))
		iy := float32(r.top + (r.bottom-r.top)/2)
		if gfx != nil {
			drawSmoothVectorIcon(gfx, ix+float32(dp(8)), iy, float32(dp(18)), i, iconCol)
		}

		pSetBkMode.Call(dc, TRANSPARENT)
		titleColor := colText
		if active {
			titleColor = colGold
		}
		pSetTextColor.Call(dc, titleColor)
		oldF, _, _ := pSelectObject.Call(dc, fontBold)
		trc := rect{r.left + dp(46), r.top + dp(6), r.right - dp(6), r.top + dp(24)}
		pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr(tabItems[i].name))),
			^uintptr(0), uintptr(unsafe.Pointer(&trc)),
			DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

		pSetTextColor.Call(dc, colMuted)
		pSelectObject.Call(dc, fontHint)
		drc := rect{r.left + dp(46), r.top + dp(24), r.right - dp(6), r.bottom - dp(4)}
		pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr(tabItems[i].desc))),
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
	drawTitleBlock(dc, cx, cw, tabItems[tab].name, tabItems[tab].desc)
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
	case 3: // Appearance, system, and app exclusions
		drawSettingsCard(dc, rect{cx, ct, cx + cw, ct + dp(268)})
		drawSettingsCard(dc, rect{cx, ct + dp(274), cx + cw, ct + dp(388)})
	}
}

// drawOverviewTab renders the rich mockup content of Tab 0
func drawOverviewTab(dc uintptr, cx, ct, cw, winH int32) {
	drawTitleBlock(dc, cx, cw, "Tổng quan", "Thiết lập các tùy chọn cơ bản để sử dụng Tui Gõ")

	// Hero Card
	heroRc := rect{cx, ct, cx + cw, ct + dp(56)}
	drawSettingsCard(dc, heroRc)

	// Hero Card text — status display only (no master switch, EVKey-style)
	textX := cx + dp(18)
	pSetTextColor.Call(dc, colText)
	pSelectObject.Call(dc, fontBold)
	htrc := rect{textX, ct + dp(8), cx + cw - dp(140), ct + dp(30)}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr("Chế độ gõ hiện tại"))),
		^uintptr(0), uintptr(unsafe.Pointer(&htrc)),
		DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

	pSetTextColor.Call(dc, colMuted)
	pSelectObject.Call(dc, fontHint)
	hdrc := rect{textX, ct + dp(30), cx + cw - dp(140), ct + dp(48)}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr("Bấm phím tắt để chuyển Tiếng Việt / English"))),
		^uintptr(0), uintptr(unsafe.Pointer(&hdrc)),
		DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

	// Mode indicator on the right where the switch used to be
	modeTxt := "● Tiếng Việt"
	modeCol := colJade
	if !effectiveViet() {
		modeTxt = "● English"
		modeCol = colMuted
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
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr("Chế độ gõ"))),
		^uintptr(0), uintptr(unsafe.Pointer(&sec1)),
		DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

	pSetTextColor.Call(dc, colMuted)
	pSelectObject.Call(dc, fontHint)
	sec1Sub := rect{cx, ct + dp(86), cx + cw, ct + dp(102)}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr("Chọn kiểu gõ phù hợp với thói quen của bạn"))),
		^uintptr(0), uintptr(unsafe.Pointer(&sec1Sub)),
		DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

	// Section 2: "Tùy chọn chung"
	pSetTextColor.Call(dc, colText)
	pSelectObject.Call(dc, fontSection)
	sec2 := rect{cx, ct + dp(160), cx + cw, ct + dp(180)}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr("Tùy chọn chung"))),
		^uintptr(0), uintptr(unsafe.Pointer(&sec2)),
		DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

	// Section 3: "Ngôn ngữ hiển thị"
	pSetTextColor.Call(dc, colText)
	pSelectObject.Call(dc, fontSection)
	sec3 := rect{cx, ct + dp(286), cx + cw - dp(150), ct + dp(306)}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr("Ngôn ngữ hiển thị"))),
		^uintptr(0), uintptr(unsafe.Pointer(&sec3)),
		DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

	pSetTextColor.Call(dc, colMuted)
	pSelectObject.Call(dc, fontHint)
	sec3Sub := rect{cx, ct + dp(306), cx + cw - dp(150), ct + dp(322)}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr("Ngôn ngữ của giao diện Tui Gõ"))),
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
	gfx := NewGfx(dc)
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

	stTitle := "Tiếng Việt đang hoạt động"
	if !effectiveViet() {
		stTitle = "Đang tắt tiếng Việt (English)"
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
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr("Phiên bản "+appVersion+" | Cập nhật lần cuối: "+appBuildStamp))),
		^uintptr(0), uintptr(unsafe.Pointer(&st2)),
		DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

	pSetTextColor.Call(dc, colGold)
	pSelectObject.Call(dc, fontTagline)
	scRc := rect{cx + cw - dp(260), ftRc.top + dp(8), cx + cw - dp(16), ftRc.bottom - dp(8)}
	if compactFooter {
		scRc = rect{sx + dp(18), ftRc.top + dp(43), cx + cw - dp(16), ftRc.bottom - dp(3)}
	}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr("Cùng bạn trên mọi hành trình ♥"))),
		^uintptr(0), uintptr(unsafe.Pointer(&scRc)),
		DT_RIGHT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

	_ = winH
}

// drawAboutTab keeps the app details in the same card system as Settings.
func drawAboutTab(dc uintptr, cx, ct, cw, winH int32) {
	drawTitleBlock(dc, cx, cw, "Giới thiệu", "Phiên bản và thông tin ứng dụng")
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
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr("Phiên bản "+appVersion))), ^uintptr(0), uintptr(unsafe.Pointer(&ver)), DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)
	pSetTextColor.Call(dc, colText)
	desc := rect{cx + dp(20), ct + dp(105), cx + cw - dp(20), ct + dp(148)}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr("Bộ gõ tiếng Việt cho Windows, hỗ trợ nhiều kiểu gõ và gõ tắt. Cài đặt được lưu trên máy của bạn."))), ^uintptr(0), uintptr(unsafe.Pointer(&desc)), DT_LEFT|DT_WORDBREAK|DT_NOPREFIX)

	pSelectObject.Call(dc, fontSection)
	info := rect{cx + dp(20), ct + dp(188), cx + cw - dp(20), ct + dp(212)}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr("Tính năng"))), ^uintptr(0), uintptr(unsafe.Pointer(&info)), DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)
	pSelectObject.Call(dc, fontNormal)
	rows := []struct{ label, value string }{
		{"Kiểu gõ", "Telex · VNI · VIQR"},
		{"Bảng mã", "Unicode · TCVN3"},
		{"Tùy chỉnh", "Phím tắt · Ứng dụng loại trừ"},
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

	pSelectObject.Call(dc, oldF)
}

func drawLangBtn(di *drawItemStruct) {
	dc := di.hDC
	rc := di.rcItem
	pressed := di.itemState&ODS_SELECTED != 0

	gfx := NewGfx(dc)
	if gfx != nil {
		gfx.DrawGlassCard(float32(rc.left), float32(rc.top), float32(rc.right-rc.left), float32(rc.bottom-rc.top), float32(dp(6)), isDark(), pressed)
		gfx.Destroy()
	}

	pSetBkMode.Call(dc, TRANSPARENT)
	pSetTextColor.Call(dc, colText)
	oldF, _, _ := pSelectObject.Call(dc, fontNormal)
	trc := rect{rc.left + dp(12), rc.top, rc.right - dp(24), rc.bottom}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr("Tiếng Việt"))),
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

	// ---- Tab 0: Tổng quan (Overview matching mockup) ----
	curTab = 0
	// 3 Radio pills for Kiểu gõ
	addRadio(cRadioTelex, "Telex", "Phổ biến, dễ sử dụng")
	addRadio(cRadioVni, "VNI", "Tương thích rộng rãi")
	addRadio(cRadioViqr, "VIQR", "Dành cho nâng cao")
	// 6 Checkboxes for Tùy chọn chung (2 columns)
	addCheckbox(cStartup, "Khởi động cùng hệ thống")
	addCheckbox(cRestore, "Tự động khôi phục từ sai")
	addCheckbox(cAutoCap, "Tự động viết hoa chữ đầu")
	addCheckbox(cModern, "Bỏ dấu kiểu mới (oà, uý)")
	addCheckbox(cSpell, "Kiểm tra chính tả tiếng Việt")
	addCheckbox(cFreeMark, "Cho phép bỏ dấu tự do")
	// Language dropdown button
	addBtn(cLangBtn, "Tiếng Việt")

	// ---- Tab 1: Gõ tiếng Việt (Advanced input methods & Charset & Macro) ----
	curTab = 1
	stIMSection = addSection("KIỂU GÕ NÂNG CAO")
	stLbIM = addLabel("Kiểu gõ")
	cb := mkCtl(h, ctlDef{cIMCombo, "COMBOBOX", CBS_DROPDOWNLIST | WS_TABSTOP, "", 0, 0, 10, 10})
	comboAdd(cb, []string{"Telex", "Telex đơn giản", "VNI", "VIQR",
		"Telex + VNI", "MS-VI", "Tự định nghĩa"}, 0)
	if isDark() {
		darkCtl(cb)
	}
	stLbCharset = addLabel("Bảng mã")
	cs := mkCtl(h, ctlDef{cCharset, "COMBOBOX", CBS_DROPDOWNLIST | WS_TABSTOP, "", 0, 0, 10, 10})
	comboAdd(cs, []string{"Unicode", "TCVN3"}, cfg.Charset)
	if isDark() {
		darkCtl(cs)
	}

	stMacroSection = addSection("GÕ TẮT (MACRO)")
	addSw(cMacro, "Cho phép gõ tắt")
	addSw(cMacroAlways, "Gõ tắt cả khi tắt Tiếng Việt")
	addBtn(cMacroEdit, "Mở bảng gõ tắt")
	stMacroHint1 = addHint("Mỗi dòng: từ_gõ_tắt:nội_dung  (vd: vn:Việt Nam)")
	stMacroHint2 = addHint("File macro.txt nằm cạnh tuigo.exe")

	// ---- Tab 2: Phím tắt ----
	curTab = 2
	stSwitchSection = addSection("PHÍM CHUYỂN TIẾNG VIỆT")
	addSw(cModCtrl, "Ctrl")
	addSw(cModShift, "Shift")
	addSw(cModAlt, "Alt")
	addSw(cModWin, "Win")
	stLbHotkey = addLabel("Phím")
	hotkeyEdit := mkCtl(h, ctlDef{cHotkeyBox, "EDIT", WS_TABSTOP | 0x0001 | 0x0008, "", 0, 0, 10, 10})
	pSendMessage.Call(hotkeyEdit, 0x00C5 /* EM_LIMITTEXT */, 1, 0)
	stHotkeyHint = addHint("A-Z hoặc 0-9 · có thể để trống")
	addSw(cUseCtrlShift, "Thêm Ctrl + Shift làm phím chuyển")
	stFKeySection = addSection("PHÍM NHANH (F-KEYS)")
	addSw(cF1, "F1 — Bật tiếng Việt")
	addSw(cF2, "F2 — Tắt tiếng Việt")
	addSw(cF5, "F5 — Mở cài đặt")
	addSw(cF9, "F9 — Bật/tắt gõ tắt")
	addSw(cF12, "F12 — Xóa bộ đệm gõ")

	// ---- Tab 3: Giao diện & Hệ thống ----
	curTab = 3
	stSysSection = addSection("GIAO DIỆN & HỆ THỐNG")
	btnIDs[cThemeAuto] = true
	btnIDs[cThemeLight] = true
	btnIDs[cThemeDark] = true
	mkCtl(h, ctlDef{cThemeAuto, "BUTTON", BS_OWNERDRAW | WS_TABSTOP, "Hệ thống", 0, 0, 10, 10})
	mkCtl(h, ctlDef{cThemeLight, "BUTTON", BS_OWNERDRAW | WS_TABSTOP, "Sáng", 0, 0, 10, 10})
	mkCtl(h, ctlDef{cThemeDark, "BUTTON", BS_OWNERDRAW | WS_TABSTOP, "Tối", 0, 0, 10, 10})
	addSw(cShowHud, "Hiện thanh trạng thái nổi")
	addSw(cSound, "Âm thanh khi chuyển chế độ")
	addSw(cSkipLayout, "Tự tắt khi bàn phím không phải kiểu US")
	addSw(cClipboard, "Dùng Clipboard cho app không nhận phím")
	addSw(cAdmin, "Chạy với quyền Admin")
	addSw(cShowDlg, "Hiện bảng này khi khởi động Windows")
	addBtn(cExcludeBrowse, "Chọn app…")
	addBtn(cExcludeEdit, "Danh sách loại trừ…")
	stExcludeSection = addSection("ỨNG DỤNG LOẠI TRỪ")
	stExcludeHint = addHint("Chọn app → tắt gõ trong app đó · sửa danh sách để dùng |lock |clip |tcvn |vni")

	// ---- Tab 4: Giới thiệu ----
	curTab = 4
	addBtn(cUpdateBtn, "Kiểm tra cập nhật")

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
	railW = dp(216)
	cx := railW + dp(24)
	cw := winW - cx - dp(24)
	ct := dp(106)

	// Tab 0 — Tổng quan
	pw := (cw - 2*dp(6)) / 3
	place(cRadioTelex, cx, ct+dp(106), pw, dp(48))
	place(cRadioVni, cx+pw+dp(6), ct+dp(106), pw, dp(48))
	place(cRadioViqr, cx+2*(pw+dp(6)), ct+dp(106), pw, dp(48))
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

	// Tab 3 — Giao diện & Hệ thống
	place(stSysSection, cx+dp(16), ct+dp(12), cw-dp(32), dp(20))
	btnW := (cw - dp(32) - dp(16)) / 3
	place(cThemeAuto, cx+dp(16), ct+dp(38), btnW, dp(48))
	place(cThemeLight, cx+dp(16)+btnW+dp(8), ct+dp(38), btnW, dp(48))
	place(cThemeDark, cx+dp(16)+2*(btnW+dp(8)), ct+dp(38), cw-dp(16)-(cx+dp(16)+2*(btnW+dp(8))), dp(48))

	sysSw := []int{cShowHud, cSound, cSkipLayout, cClipboard, cAdmin, cShowDlg}
	for row, id := range sysSw {
		place(id, cx+dp(16), ct+dp(98)+int32(row)*dp(27), cw-dp(32), dp(26))
	}
	place(stExcludeSection, cx+dp(16), ct+dp(278), cw-dp(32), dp(20))
	place(cExcludeBrowse, cx+dp(16), ct+dp(306), dp(120), dp(30))
	place(cExcludeEdit, cx+dp(144), ct+dp(306), dp(170), dp(30))
	place(stExcludeHint, cx+dp(16), ct+dp(344), cw-dp(32), dp(36))

	// Tab 4 — Giới thiệu (button sits inside the second card)
	place(cUpdateBtn, cx+cw-dp(20)-dp(170), ct+dp(317), dp(170), dp(32))

	_ = winH
	pInvalidateRect.Call(gSettingsHwnd, 0, 1)
}

func navRect(i int) rect {
	y := dp(56) + int32(i)*dp(52)
	return rect{dp(10), y, railW - dp(10), y + dp(46)}
}

func setTab(i int) {
	gActiveTab = i
	for _, hctl := range allCtls {
		t := ctlTab[hctl]
		vis := uintptr(SW_HIDE)
		if t < 0 || t == i {
			vis = SW_SHOW
		}
		pShowWindow.Call(hctl, vis)
	}
	if gSettingsHwnd != 0 {
		pInvalidateRect.Call(gSettingsHwnd, 0, 1)
	}
}

func imIdx(im int) int {
	for i, v := range imList {
		if v == im {
			return i
		}
	}
	return 0
}

// repaintRadioGroup refreshes the three "Chế độ gõ" cards. They are child
// HWNDs — InvalidateRect on the parent never reaches them, so without this
// the previously selected card kept its stale "selected" pixels next to the
// freshly clicked one.
func repaintRadioGroup() {
	for _, id := range []int{cRadioTelex, cRadioVni, cRadioViqr} {
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
	for _, tid := range []int{cThemeAuto, cThemeLight, cThemeDark} {
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

func browseExe() string {
	var buf [1024]uint16
	filter := utf16buf("Ứng dụng (*.exe)\x00*.exe\x00Tất cả file (*.*)\x00*.*\x00")
	title := utf16ptr("Chọn app cần loại trừ")
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

	pAppendMenu.Call(hMenu, 0, 1, uintptr(unsafe.Pointer(utf16ptr("Unicode (Mặc định)"))))
	pAppendMenu.Call(hMenu, 0, 2, uintptr(unsafe.Pointer(utf16ptr("TCVN3 (Tiêu chuẩn VN 3)"))))

	var pt point
	pGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))

	r, _, _ := pTrackPopupMenu.Call(hMenu, 0x0100|0x0002,
		uintptr(pt.x), uintptr(pt.y), 0, gSettingsHwnd, 0)
	switch r {
	case 1:
		cfg.Charset = 0
		saveSettings()
		pSendMessage.Call(ctlHnd[cCharset], CB_SETCURSEL, 0, 0)
		pInvalidateRect.Call(gSettingsHwnd, 0, 1)
	case 2:
		cfg.Charset = 1
		saveSettings()
		pSendMessage.Call(ctlHnd[cCharset], CB_SETCURSEL, 1, 0)
		pInvalidateRect.Call(gSettingsHwnd, 0, 1)
	}
}

// launchUpdater runs the optional companion updater (-now = skip throttle,
// always report). tuigo.exe itself stays offline; deleting the updater exe
// disables update checks entirely.
func launchUpdater(now bool) {
	up := filepath.Join(exeDir(), "tuigo-updater.exe")
	if _, err := os.Stat(up); err != nil {
		pMessageBox.Call(gSettingsHwnd,
			uintptr(unsafe.Pointer(utf16ptr("Không thấy tuigo-updater.exe cạnh tuigo.exe.\nTải bản zip đầy đủ hoặc tải trên trang Releases của GitHub."))),
			uintptr(unsafe.Pointer(utf16ptr("Tui Gõ"))), 0x30)
		return
	}
	args := ""
	if now {
		args = "-now"
	}
	pShellExecute.Call(0, uintptr(unsafe.Pointer(utf16ptr("open"))),
		uintptr(unsafe.Pointer(utf16ptr(up))),
		uintptr(unsafe.Pointer(utf16ptr(args))),
		uintptr(unsafe.Pointer(utf16ptr(exeDir()))), SW_SHOW)
}

func settingsCommand(id int) {
	o := gEngine.GetOptions()
	switch id {
	case cUpdateBtn:
		launchUpdater(true)
		return
	case cRadioTelex:
		setIM(ImTelex)
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
	defer func() { recoverCrash("settings", recover()) }()
	switch msg {
	case WM_CREATE:
		gHoveredTab = -1
		gCaptionHover, gCaptionDown = -1, -1
		buildControls(hwnd)
		return 0
	case WM_MOUSEMOVE:
		x, y := int32(int16(lp&0xFFFF)), int32(int16((lp>>16)&0xFFFF))
		var client rect
		pGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&client)))
		caption := captionAt(client.right, x, y)
		if caption != gCaptionHover {
			gCaptionHover = caption
			r := captionBarRect(client.right)
			pInvalidateRect.Call(hwnd, uintptr(unsafe.Pointer(&r)), 0)
		}
		hovered := -1
		for i := range tabItems {
			r := navRect(i)
			if x >= r.left && x < r.right && y >= r.top && y < r.bottom {
				hovered = i
				break
			}
		}
		if hovered != gHoveredTab {
			gHoveredTab = hovered
			r := rect{0, dp(50), railW, dp(320)}
			pInvalidateRect.Call(hwnd, uintptr(unsafe.Pointer(&r)), 1)
		}
		track := navMouseTrack{flags: 2 /* TME_LEAVE */, hwnd: hwnd}
		track.size = uint32(unsafe.Sizeof(track))
		pTrackMouseEvent.Call(uintptr(unsafe.Pointer(&track)))
		return 0
	case 0x02A3: // WM_MOUSELEAVE
		if gCaptionHover != -1 {
			gCaptionHover = -1
			var client rect
			pGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&client)))
			r := captionBarRect(client.right)
			pInvalidateRect.Call(hwnd, uintptr(unsafe.Pointer(&r)), 0)
		}
		if gHoveredTab != -1 {
			gHoveredTab = -1
			r := rect{0, dp(50), railW, dp(320)}
			pInvalidateRect.Call(hwnd, uintptr(unsafe.Pointer(&r)), 1)
		}
		return 0
	case WM_SIZE:
		if wp != 1 {
			updateSettingsRegion(hwnd)
			layoutAll()
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
			for i := range tabItems {
				r := navRect(i)
				if x >= r.left && x < r.right && y >= r.top && y < r.bottom {
					setTab(i)
					return 0
				}
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
			case btnIDs[id]:
				pressed := di.itemState&ODS_SELECTED != 0
				drawSmoothButton(di.hDC, di.rcItem, id, pressed)
			}
			return 1
		}
	case WM_CTLCOLORSTATIC, WM_CTLCOLORBTN:
		col := uintptr(colText)
		if goldCtls[lp] {
			col = colText
		} else if mutedCtls[lp] {
			col = colMuted
		}
		pSetTextColor.Call(wp, col)
		pSetBkMode.Call(wp, TRANSPARENT)
		if bgCtls[lp] {
			return brBg
		}
		return brCard
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
		// Paint into a back buffer, then blit once: drawing straight onto the
		// window DC lets DWM present the bare background mid-paint (the white
		// flash on open/tab switch).
		mem, _, _ := pCreateCompatibleDC.Call(dc)
		bmp, _, _ := pCreateCompatibleBitmap.Call(dc, uintptr(winW), uintptr(winH))
		if mem != 0 && bmp != 0 {
			oldBmp, _, _ := pSelectObject.Call(mem, bmp)
			paintSettings(mem, hwnd)
			pBitBlt.Call(dc, 0, 0, uintptr(winW), uintptr(winH), mem, 0, 0, SRCCOPY)
			pSelectObject.Call(mem, oldBmp)
			pDeleteObject.Call(bmp)
			pDeleteDC.Call(mem)
		} else {
			if bmp != 0 {
				pDeleteObject.Call(bmp)
			}
			if mem != 0 {
				pDeleteDC.Call(mem)
			}
			paintSettings(dc, hwnd)
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
		gSettingsHwnd = 0
		gCaptionHover, gCaptionDown = -1, -1
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

// paintSettings renders the full client area into dc (back buffer or screen).
func paintSettings(dc, hwnd uintptr) {
	var rc rect
	pGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rc)))
	winW, winH := rc.right, rc.bottom

	// 1. Fill entire window background
	pFillRect.Call(dc, uintptr(unsafe.Pointer(&rc)), brBg)

	// 2. Minimal liquid glass inner border with top specular highlight
	gfx := NewGfx(dc)
	if gfx != nil {
		gfx.DrawRoundedRect(0.5, 0.5, float32(winW)-1.0, float32(winH)-1.0, float32(dp(8)), argbBorder, 1.0)
		gfx.DrawLine(float32(dp(8)), 1.0, float32(winW-dp(8)), 1.0, HexAlphaARGB(0x35, 0xFFFFFF), 1.0)
		gfx.Destroy()
	}

	// 3. Top bar: brand title, subtitle, and window controls
	drawTopHeader(dc, hwnd, winW)

	// 4. Sidebar vertical divider
	divLine := rect{railW, dp(50), railW + 1, winH - dp(14)}
	pFillRect.Call(dc, uintptr(unsafe.Pointer(&divLine)), brBorder)

	// 5. Sidebar tabs
	drawNav(dc)

	// 6. Right Content Pane depending on tab
	cx := railW + dp(24)
	cw := winW - cx - dp(24)
	ct := dp(106)

	switch gActiveTab {
	case 0:
		drawOverviewTab(dc, cx, ct, cw, winH)
	case 4:
		drawAboutTab(dc, cx, ct, cw, winH)
	default:
		drawSettingsTab(dc, gActiveTab, cx, ct, cw)
	}
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
		uintptr(unsafe.Pointer(utf16ptr("Tui Gõ — Cài đặt"))),
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
