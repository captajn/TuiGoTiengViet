package main

// Settings window: High-End Xianxia / Chinese Fantasy modern UI matching
// the Tui Gõ Xianxia mockup (media_1790104583638.jpg).
// Custom borderless window with golden cloud filigree (祥云), custom titlebar,
// 5-tab vector icon sidebar, Overview Hero Card, horizontal radio pills,
// 2-column checkbox grid, and status footer banner.

import (
	"fmt"
	"os"
	"path/filepath"
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
	SWP_NOMOVE         = 0x0002
)

var (
	rethemed      bool
	gSettingsHwnd uintptr
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

	fontNormal  uintptr
	fontBold    uintptr
	fontTitle   uintptr
	fontSection uintptr
	fontHint    uintptr
	fontScript  uintptr

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
	stSysSection, stLbTheme, stExcludeHint                   int
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
	cHeroSwitch
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

// Auspicious golden cloud curves (祥云) at window top-right
func drawWindowCloudFiligree(dc uintptr, winW int32) {
	nullBr, _, _ := pGetStockObject.Call(NULL_BRUSH)
	oldB, _, _ := pSelectObject.Call(dc, nullBr)
	oldP, _, _ := pSelectObject.Call(dc, penGoldDim)

	// Soft scrolling arcs kept safely to the left of window controls
	pArc.Call(dc, uintptr(winW-dp(230)), uintptr(-dp(15)), uintptr(winW-dp(170)), uintptr(dp(25)),
		uintptr(winW-dp(230)), uintptr(dp(10)), uintptr(winW-dp(170)), uintptr(dp(10)))
	pArc.Call(dc, uintptr(winW-dp(185)), uintptr(-dp(8)), uintptr(winW-dp(125)), uintptr(dp(28)),
		uintptr(winW-dp(185)), uintptr(dp(12)), uintptr(winW-dp(125)), uintptr(dp(12)))

	r := dp(7)
	cx := winW - dp(145)
	cy := dp(16)
	pArc.Call(dc, uintptr(cx-r), uintptr(cy-r), uintptr(cx+r), uintptr(cy+r),
		uintptr(cx), uintptr(cy-r), uintptr(cx+r), uintptr(cy))

	diamond(dc, winW-dp(240), dp(14), dp(3), colGold)
	diamond(dc, winW-dp(205), dp(22), dp(2), colGoldDim)
	diamond(dc, winW-dp(160), dp(8), dp(2), colGoldDim)
	diamond(dc, winW-dp(118), dp(16), dp(2), colGold)

	pSelectObject.Call(dc, oldP)
	pSelectObject.Call(dc, oldB)
}

// Window control buttons: — □ ✕
func drawWindowControls(dc uintptr, winW int32) {
	minRc := rect{winW - dp(96), dp(8), winW - dp(70), dp(34)}
	maxRc := rect{winW - dp(66), dp(8), winW - dp(40), dp(34)}
	clsRc := rect{winW - dp(36), dp(8), winW - dp(10), dp(34)}

	ctrlCol := colMuted
	if isDark() {
		ctrlCol = 0x8FA2A0 // soft luminous silver-teal
	}
	pp, _, _ := pCreatePen.Call(PS_SOLID, 2, ctrlCol)
	oldP, _, _ := pSelectObject.Call(dc, pp)

	// Minimize '—'
	pMoveToEx.Call(dc, uintptr(minRc.left+dp(6)), uintptr(minRc.top+dp(13)), 0)
	pLineTo.Call(dc, uintptr(minRc.right-dp(6)), uintptr(minRc.top+dp(13)))

	// Maximize '□'
	nullBr, _, _ := pGetStockObject.Call(NULL_BRUSH)
	oldB, _, _ := pSelectObject.Call(dc, nullBr)
	pRectangle.Call(dc, uintptr(maxRc.left+dp(7)), uintptr(maxRc.top+dp(8)),
		uintptr(maxRc.right-dp(7)), uintptr(maxRc.bottom-dp(10)))
	pSelectObject.Call(dc, oldB)

	// Close '✕'
	pMoveToEx.Call(dc, uintptr(clsRc.left+dp(7)), uintptr(clsRc.top+dp(8)), 0)
	pLineTo.Call(dc, uintptr(clsRc.right-dp(7)), uintptr(clsRc.bottom-dp(10)))
	pMoveToEx.Call(dc, uintptr(clsRc.right-dp(7)), uintptr(clsRc.top+dp(8)), 0)
	pLineTo.Call(dc, uintptr(clsRc.left+dp(7)), uintptr(clsRc.bottom-dp(10)))

	pSelectObject.Call(dc, oldP)
	pDeleteObject.Call(pp)
}

// Top titlebar: Mascot + "Tui Gõ" + Subtitle + Window controls + Golden cloud filigree
func drawTopHeader(dc uintptr, winW int32) {
	drawAppMascot(dc, dp(14), dp(6), dp(36), dp(36), dp(8))

	pSetBkMode.Call(dc, TRANSPARENT)
	pSetTextColor.Call(dc, colText)
	oldF, _, _ := pSelectObject.Call(dc, fontTitle)
	trc := rect{dp(58), dp(7), dp(280), dp(29)}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr("Tui Gõ"))),
		^uintptr(0), uintptr(unsafe.Pointer(&trc)),
		DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

	pSetTextColor.Call(dc, colMuted)
	pSelectObject.Call(dc, fontHint)
	sbc := rect{dp(58), dp(27), dp(280), dp(43)}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr("Gõ tiếng Việt thông minh hơn"))),
		^uintptr(0), uintptr(unsafe.Pointer(&sbc)),
		DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

	pSelectObject.Call(dc, oldF)

	drawWindowCloudFiligree(dc, winW)
	drawWindowControls(dc, winW)
}

// Vector icons for the 5 sidebar tabs
func drawIconHome(dc uintptr, x, y, s int32, col uintptr) {
	pp, _, _ := pCreatePen.Call(PS_SOLID, 2, col)
	oldP, _, _ := pSelectObject.Call(dc, pp)

	pts := []point{
		{x + s/2, y + 2},
		{x + s - 2, y + s*45/100},
		{x + 2, y + s*45/100},
		{x + s/2, y + 2},
	}
	pPolyline.Call(dc, uintptr(unsafe.Pointer(&pts[0])), uintptr(len(pts)))

	wPts := []point{
		{x + 4, y + s*45/100},
		{x + 4, y + s - 2},
		{x + s - 4, y + s - 2},
		{x + s - 4, y + s*45/100},
	}
	pPolyline.Call(dc, uintptr(unsafe.Pointer(&wPts[0])), uintptr(len(wPts)))

	dPts := []point{
		{x + s/2 - 2, y + s - 2},
		{x + s/2 - 2, y + s*6/10},
		{x + s/2 + 2, y + s*6/10},
		{x + s/2 + 2, y + s - 2},
	}
	pPolyline.Call(dc, uintptr(unsafe.Pointer(&dPts[0])), uintptr(len(dPts)))

	pSelectObject.Call(dc, oldP)
	pDeleteObject.Call(pp)
}

func drawIconKeyboard(dc uintptr, x, y, s int32, col uintptr) {
	pp, _, _ := pCreatePen.Call(PS_SOLID, 1, col)
	oldP, _, _ := pSelectObject.Call(dc, pp)
	nullBr, _, _ := pGetStockObject.Call(NULL_BRUSH)
	oldB, _, _ := pSelectObject.Call(dc, nullBr)

	pRoundRect.Call(dc, uintptr(x+2), uintptr(y+3), uintptr(x+s-2), uintptr(y+s-3), uintptr(dp(4)), uintptr(dp(4)))

	// Key dots
	for ki := int32(0); ki < 4; ki++ {
		pMoveToEx.Call(dc, uintptr(x+5+ki*3), uintptr(y+7), 0)
		pLineTo.Call(dc, uintptr(x+6+ki*3), uintptr(y+7))
	}
	// Spacebar line
	pMoveToEx.Call(dc, uintptr(x+6), uintptr(y+s-7), 0)
	pLineTo.Call(dc, uintptr(x+s-6), uintptr(y+s-7))

	pSelectObject.Call(dc, oldB)
	pSelectObject.Call(dc, oldP)
	pDeleteObject.Call(pp)
}

func drawIconShortcut(dc uintptr, x, y, s int32, col uintptr) {
	pp, _, _ := pCreatePen.Call(PS_SOLID, 1, col)
	oldP, _, _ := pSelectObject.Call(dc, pp)
	nullBr, _, _ := pGetStockObject.Call(NULL_BRUSH)
	oldB, _, _ := pSelectObject.Call(dc, nullBr)

	pRoundRect.Call(dc, uintptr(x+2), uintptr(y+2), uintptr(x+s-2), uintptr(y+s-2), uintptr(dp(4)), uintptr(dp(4)))

	pMoveToEx.Call(dc, uintptr(x+5), uintptr(y+8), 0)
	pLineTo.Call(dc, uintptr(x+5), uintptr(y+5))
	pLineTo.Call(dc, uintptr(x+8), uintptr(y+5))

	pMoveToEx.Call(dc, uintptr(x+s-8), uintptr(y+5), 0)
	pLineTo.Call(dc, uintptr(x+s-5), uintptr(y+5))
	pLineTo.Call(dc, uintptr(x+s-5), uintptr(y+8))

	pMoveToEx.Call(dc, uintptr(x+5), uintptr(y+s-8), 0)
	pLineTo.Call(dc, uintptr(x+5), uintptr(y+s-5))
	pLineTo.Call(dc, uintptr(x+8), uintptr(y+s-5))

	pMoveToEx.Call(dc, uintptr(x+s-8), uintptr(y+s-5), 0)
	pLineTo.Call(dc, uintptr(x+s-5), uintptr(y+s-5))
	pLineTo.Call(dc, uintptr(x+s-5), uintptr(y+s-8))

	pSelectObject.Call(dc, oldB)
	pSelectObject.Call(dc, oldP)
	pDeleteObject.Call(pp)
}

func drawIconPalette(dc uintptr, x, y, s int32, col uintptr) {
	pp, _, _ := pCreatePen.Call(PS_SOLID, 1, col)
	oldP, _, _ := pSelectObject.Call(dc, pp)
	nullBr, _, _ := pGetStockObject.Call(NULL_BRUSH)
	oldB, _, _ := pSelectObject.Call(dc, nullBr)

	pEllipse.Call(dc, uintptr(x+2), uintptr(y+2), uintptr(x+s-2), uintptr(y+s-2))
	pEllipse.Call(dc, uintptr(x+s/2), uintptr(y+s/2), uintptr(x+s*7/10), uintptr(y+s*7/10))

	diamond(dc, x+s*3/10, y+s*4/10, dp(1), col)
	diamond(dc, x+s/2, y+s*3/10, dp(1), col)

	pSelectObject.Call(dc, oldB)
	pSelectObject.Call(dc, oldP)
	pDeleteObject.Call(pp)
}

func drawIconInfo(dc uintptr, x, y, s int32, col uintptr) {
	pp, _, _ := pCreatePen.Call(PS_SOLID, 1, col)
	oldP, _, _ := pSelectObject.Call(dc, pp)
	nullBr, _, _ := pGetStockObject.Call(NULL_BRUSH)
	oldB, _, _ := pSelectObject.Call(dc, nullBr)

	pEllipse.Call(dc, uintptr(x+2), uintptr(y+2), uintptr(x+s-2), uintptr(y+s-2))
	diamond(dc, x+s/2, y+s*32/100, dp(1), col)
	pMoveToEx.Call(dc, uintptr(x+s/2), uintptr(y+s*46/100), 0)
	pLineTo.Call(dc, uintptr(x+s/2), uintptr(y+s*76/100))

	pSelectObject.Call(dc, oldB)
	pSelectObject.Call(dc, oldP)
	pDeleteObject.Call(pp)
}

// Sidebar navigation renderer
func drawNav(dc uintptr) {
	for i := 0; i < len(tabItems); i++ {
		r := navRect(i)
		active := (i == gActiveTab)

		if active {
			oldB, _, _ := pSelectObject.Call(dc, brNavActive)
			oldP, _, _ := pSelectObject.Call(dc, penNavActive)
			pRoundRect.Call(dc, uintptr(r.left), uintptr(r.top),
				uintptr(r.right), uintptr(r.bottom), uintptr(dp(10)), uintptr(dp(10)))
			pSelectObject.Call(dc, oldP)
			pSelectObject.Call(dc, oldB)

			// Gold indicator bar on left (điểm nhấn 鎏金)
			gbar, _, _ := pCreateSolidBrush.Call(colGold)
			oldB2, _, _ := pSelectObject.Call(dc, gbar)
			nullPen, _, _ := pGetStockObject.Call(NULL_PEN)
			oldP2, _, _ := pSelectObject.Call(dc, nullPen)
			pRoundRect.Call(dc, uintptr(r.left+dp(4)), uintptr(r.top+dp(10)),
				uintptr(r.left+dp(7)), uintptr(r.bottom-dp(10)), uintptr(dp(3)), uintptr(dp(3)))
			pSelectObject.Call(dc, oldP2)
			pSelectObject.Call(dc, oldB2)
			pDeleteObject.Call(gbar)
		}

		iconCol := colMuted
		if active {
			iconCol = colJade
		}
		ix := r.left + dp(14)
		iy := r.top + (r.bottom-r.top-dp(20))/2
		is := dp(20)
		switch i {
		case 0:
			drawIconHome(dc, ix, iy, is, iconCol)
		case 1:
			drawIconKeyboard(dc, ix, iy, is, iconCol)
		case 2:
			drawIconShortcut(dc, ix, iy, is, iconCol)
		case 3:
			drawIconPalette(dc, ix, iy, is, iconCol)
		case 4:
			drawIconInfo(dc, ix, iy, is, iconCol)
		}

		pSetBkMode.Call(dc, TRANSPARENT)
		pSetTextColor.Call(dc, colText)
		oldF, _, _ := pSelectObject.Call(dc, fontBold)
		trc := rect{r.left + dp(42), r.top + dp(6), r.right - dp(6), r.top + dp(24)}
		pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr(tabItems[i].name))),
			^uintptr(0), uintptr(unsafe.Pointer(&trc)),
			DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

		pSetTextColor.Call(dc, colMuted)
		pSelectObject.Call(dc, fontHint)
		drc := rect{r.left + dp(42), r.top + dp(24), r.right - dp(6), r.bottom - dp(4)}
		pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr(tabItems[i].desc))),
			^uintptr(0), uintptr(unsafe.Pointer(&drc)),
			DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

		pSelectObject.Call(dc, oldF)
	}
}

// drawTitleBlock renders the shared tab header: title + subtitle + a gold
// hairline with a lozenge end-cap (kiểu dải phân mục trong promo tiên hiệp).
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

	// gold hairline fading right, lozenge at both ends
	hbr, _, _ := pCreateSolidBrush.Call(colGoldDim)
	hl := rect{cx, dp(101), cx + dp(150), dp(102)}
	pFillRect.Call(dc, uintptr(unsafe.Pointer(&hl)), hbr)
	pDeleteObject.Call(hbr)
	diamond(dc, cx, dp(101), dp(3), colGold)
	diamond(dc, cx+dp(158), dp(101), dp(2), colGoldDim)
}

// drawOverviewTab renders the rich mockup content of Tab 0
func drawOverviewTab(dc uintptr, cx, ct, cw, winH int32) {
	drawTitleBlock(dc, cx, cw, "Tổng quan", "Thiết lập các tùy chọn cơ bản để sử dụng Tui Gõ")

	// Hero Card
	heroRc := rect{cx, ct, cx + cw, ct + dp(56)}
	oldB, _, _ := pSelectObject.Call(dc, brHero)
	oldP, _, _ := pSelectObject.Call(dc, penBorder)
	pRoundRect.Call(dc, uintptr(heroRc.left), uintptr(heroRc.top),
		uintptr(heroRc.right), uintptr(heroRc.bottom), uintptr(dp(10)), uintptr(dp(10)))
	pSelectObject.Call(dc, oldP)
	pSelectObject.Call(dc, oldB)

	// Hero Card text
	textX := cx + dp(18)
	pSetTextColor.Call(dc, colText)
	pSelectObject.Call(dc, fontBold)
	htrc := rect{textX, ct + dp(8), cx + cw - dp(64), ct + dp(30)}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr("Bật gõ tiếng Việt"))),
		^uintptr(0), uintptr(unsafe.Pointer(&htrc)),
		DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

	pSetTextColor.Call(dc, colMuted)
	pSelectObject.Call(dc, fontHint)
	hdrc := rect{textX, ct + dp(30), cx + cw - dp(64), ct + dp(48)}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr("Chuyển đổi giữa Tiếng Việt và English"))),
		^uintptr(0), uintptr(unsafe.Pointer(&hdrc)),
		DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

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
	ftRc := rect{cx, ct + dp(336), cx + cw, ct + dp(336) + dp(48)}
	oldB, _, _ = pSelectObject.Call(dc, brFooter)
	oldP, _, _ = pSelectObject.Call(dc, penBorder)
	pRoundRect.Call(dc, uintptr(ftRc.left), uintptr(ftRc.top),
		uintptr(ftRc.right), uintptr(ftRc.bottom), uintptr(dp(10)), uintptr(dp(10)))
	pSelectObject.Call(dc, oldP)
	pSelectObject.Call(dc, oldB)

	sx := cx + dp(22)
	sy := ftRc.top + dp(24)
	sr := dp(6)
	sunCol := colJade
	if !effectiveViet() {
		sunCol = colMuted
	}
	sbr, _, _ := pCreateSolidBrush.Call(sunCol)
	oldB, _, _ = pSelectObject.Call(dc, sbr)
	nullP, _, _ := pGetStockObject.Call(NULL_PEN)
	oldP, _, _ = pSelectObject.Call(dc, nullP)
	pEllipse.Call(dc, uintptr(sx-sr), uintptr(sy-sr), uintptr(sx+sr), uintptr(sy+sr))
	pSelectObject.Call(dc, oldP)
	pSelectObject.Call(dc, oldB)
	pDeleteObject.Call(sbr)

	// Celestial radiating sun rays
	sp, _, _ := pCreatePen.Call(PS_SOLID, 1, sunCol)
	oldP, _, _ = pSelectObject.Call(dc, sp)
	r1 := sr + dp(2)
	r2 := sr + dp(4)
	d1 := int32(float64(r1) * 0.707)
	d2 := int32(float64(r2) * 0.707)
	pMoveToEx.Call(dc, uintptr(sx+r1), uintptr(sy), 0)
	pLineTo.Call(dc, uintptr(sx+r2), uintptr(sy))
	pMoveToEx.Call(dc, uintptr(sx-r1), uintptr(sy), 0)
	pLineTo.Call(dc, uintptr(sx-r2), uintptr(sy))
	pMoveToEx.Call(dc, uintptr(sx), uintptr(sy+r1), 0)
	pLineTo.Call(dc, uintptr(sx), uintptr(sy+r2))
	pMoveToEx.Call(dc, uintptr(sx), uintptr(sy-r1), 0)
	pLineTo.Call(dc, uintptr(sx), uintptr(sy-r2))
	pMoveToEx.Call(dc, uintptr(sx+d1), uintptr(sy+d1), 0)
	pLineTo.Call(dc, uintptr(sx+d2), uintptr(sy+d2))
	pMoveToEx.Call(dc, uintptr(sx-d1), uintptr(sy+d1), 0)
	pLineTo.Call(dc, uintptr(sx-d2), uintptr(sy+d2))
	pMoveToEx.Call(dc, uintptr(sx+d1), uintptr(sy-d1), 0)
	pLineTo.Call(dc, uintptr(sx+d2), uintptr(sy-d2))
	pMoveToEx.Call(dc, uintptr(sx-d1), uintptr(sy-d1), 0)
	pLineTo.Call(dc, uintptr(sx-d2), uintptr(sy-d2))
	pSelectObject.Call(dc, oldP)
	pDeleteObject.Call(sp)

	stTitle := "Tiếng Việt đang hoạt động"
	if !effectiveViet() {
		stTitle = "Đang tắt tiếng Việt (English)"
	}
	pSetTextColor.Call(dc, colText)
	pSelectObject.Call(dc, fontBold)
	st1 := rect{sx + dp(18), ftRc.top + dp(6), cx + cw - dp(260), ftRc.top + dp(25)}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr(stTitle))),
		^uintptr(0), uintptr(unsafe.Pointer(&st1)),
		DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

	pSetTextColor.Call(dc, colMuted)
	pSelectObject.Call(dc, fontHint)
	st2 := rect{sx + dp(18), ftRc.top + dp(25), cx + cw - dp(260), ftRc.bottom - dp(4)}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr("Phiên bản "+appVersion+" | Cập nhật lần cuối: 2026"))),
		^uintptr(0), uintptr(unsafe.Pointer(&st2)),
		DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

	pSetTextColor.Call(dc, colGold)
	pSelectObject.Call(dc, fontScript)
	scRc := rect{cx + cw - dp(260), ftRc.top + dp(8), cx + cw - dp(16), ftRc.bottom - dp(8)}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr("Cùng bạn trên mọi hành trình ❤"))),
		^uintptr(0), uintptr(unsafe.Pointer(&scRc)),
		DT_RIGHT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

	_ = winH
}

// drawAboutTab renders Tab 4
func drawAboutTab(dc uintptr, cx, ct, cw, winH int32) {
	drawTitleBlock(dc, cx, cw, "Giới thiệu", "Thông tin ứng dụng và bản quyền")

	mascotS := dp(110)
	mascotX := cx + (cw-mascotS)/2
	mascotY := ct + dp(16)

	cardRc := rect{cx, ct, cx + cw, mascotY + mascotS + dp(272)}
	oldB, _, _ := pSelectObject.Call(dc, brHero)
	oldP, _, _ := pSelectObject.Call(dc, penBorder)
	pRoundRect.Call(dc, uintptr(cardRc.left), uintptr(cardRc.top),
		uintptr(cardRc.right), uintptr(cardRc.bottom), uintptr(dp(12)), uintptr(dp(12)))
	pSelectObject.Call(dc, oldP)
	pSelectObject.Call(dc, oldB)

	drawAppMascot(dc, mascotX, mascotY, mascotS, mascotS, dp(16))

	// Imperial gold double frame around the mascot (khung vàng như chân dung promo)
	nullBr, _, _ := pGetStockObject.Call(NULL_BRUSH)
	oldMB, _, _ := pSelectObject.Call(dc, nullBr)
	gpen2, _, _ := pCreatePen.Call(PS_SOLID, 2, colGold)
	oldMP, _, _ := pSelectObject.Call(dc, gpen2)
	pRoundRect.Call(dc, uintptr(mascotX-dp(3)), uintptr(mascotY-dp(3)),
		uintptr(mascotX+mascotS+dp(3)), uintptr(mascotY+mascotS+dp(3)),
		uintptr(dp(20)), uintptr(dp(20)))
	pSelectObject.Call(dc, penGoldDim)
	pRoundRect.Call(dc, uintptr(mascotX-dp(6)), uintptr(mascotY-dp(6)),
		uintptr(mascotX+mascotS+dp(6)), uintptr(mascotY+mascotS+dp(6)),
		uintptr(dp(23)), uintptr(dp(23)))
	pSelectObject.Call(dc, oldMP)
	pSelectObject.Call(dc, oldMB)
	pDeleteObject.Call(gpen2)

	pSetTextColor.Call(dc, colText)
	pSelectObject.Call(dc, fontTitle)
	appTitle := rect{cx, mascotY + mascotS + dp(10), cx + cw, mascotY + mascotS + dp(34)}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr("Tui Gõ"))),
		^uintptr(0), uintptr(unsafe.Pointer(&appTitle)),
		DT_CENTER|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

	pSetTextColor.Call(dc, colGold)
	pSelectObject.Call(dc, fontHint)
	ver := rect{cx, mascotY + mascotS + dp(34), cx + cw, mascotY + mascotS + dp(52)}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr("Phiên bản "+appVersion+" — Pure Go & Win32 Engine"))),
		^uintptr(0), uintptr(unsafe.Pointer(&ver)),
		DT_CENTER|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

	pSetTextColor.Call(dc, colText)
	pSelectObject.Call(dc, fontNormal)
	desc := rect{cx + dp(24), mascotY + mascotS + dp(60), cx + cw - dp(24), mascotY + mascotS + dp(100)}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr("Bộ gõ tiếng Việt siêu nhẹ, hiệu năng cực cao, thiết kế theo phong cách Tiên hiệp phương Đông hiện đại. Hoàn toàn chạy nội bộ, an toàn, tôn trọng quyền riêng tư người dùng."))),
		^uintptr(0), uintptr(unsafe.Pointer(&desc)),
		DT_LEFT|DT_WORDBREAK|DT_NOPREFIX)

	pSetTextColor.Call(dc, colGold)
	pSelectObject.Call(dc, fontSection)
	scTitle := rect{cx + dp(24), mascotY + mascotS + dp(108), cx + cw - dp(24), mascotY + mascotS + dp(128)}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr("Phím tắt tiện ích nhanh"))),
		^uintptr(0), uintptr(unsafe.Pointer(&scTitle)),
		DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

	pSetTextColor.Call(dc, colMuted)
	pSelectObject.Call(dc, fontNormal)
	keys := []string{
		"Alt + Z  hoặc  Ctrl + Shift  :  Chuyển đổi Tiếng Việt / Tiếng Anh",
		"F5  :  Mở giao diện Cài đặt này",
		"F9  :  Bật / Tắt chế độ gõ tắt (Macro)",
		"F12 :  Xóa đệm gõ tức thì khi cần nhập mã hoặc mật khẩu",
	}
	for ki, kstr := range keys {
		krc := rect{cx + dp(28), mascotY + mascotS + dp(132) + int32(ki)*dp(22), cx + cw - dp(24), mascotY + mascotS + dp(154) + int32(ki)*dp(22)}
		pDrawText.Call(dc, uintptr(unsafe.Pointer(utf16ptr(kstr))),
			^uintptr(0), uintptr(unsafe.Pointer(&krc)),
			DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)
	}

	_ = winH
}

func drawButton(di *drawItemStruct) {
	dc := di.hDC
	rc := di.rcItem
	pressed := di.itemState&ODS_SELECTED != 0

	pFillRect.Call(dc, uintptr(unsafe.Pointer(&rc)), brBg)

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

		oldP2, _, _ := pSelectObject.Call(dc, penWhite2)
		pts := []point{
			{bx + dp(4), by + dp(9)},
			{bx + dp(7), by + dp(13)},
			{bx + dp(14), by + dp(5)},
		}
		pPolyline.Call(dc, uintptr(unsafe.Pointer(&pts[0])), uintptr(len(pts)))
		pSelectObject.Call(dc, oldP2)
	} else {
		oldB, _, _ := pSelectObject.Call(dc, brHero)
		oldP, _, _ := pSelectObject.Call(dc, penBorder)
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

func drawHeroSwitch(di *drawItemStruct) {
	dc := di.hDC
	rc := di.rcItem
	on := effectiveViet()

	pFillRect.Call(dc, uintptr(unsafe.Pointer(&rc)), brHero)

	pillH := dp(24)
	pillW := dp(44)
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
		kx = pill.right - dp(21)
	}
	kbr, _, _ := pCreateSolidBrush.Call(knobCol)
	pSelectObject.Call(dc, kbr)
	nullPen, _, _ := pGetStockObject.Call(NULL_PEN)
	oldKP, _, _ := pSelectObject.Call(dc, nullPen)
	pEllipse.Call(dc, uintptr(kx), uintptr(py+dp(3)),
		uintptr(kx+dp(18)), uintptr(py+dp(21)))
	pSelectObject.Call(dc, oldKP)
	pSelectObject.Call(dc, oldB)
	pDeleteObject.Call(kbr)
	pDeleteObject.Call(pbr)
}

func drawLangBtn(di *drawItemStruct) {
	dc := di.hDC
	rc := di.rcItem
	pressed := di.itemState&ODS_SELECTED != 0

	pFillRect.Call(dc, uintptr(unsafe.Pointer(&rc)), brBg)

	bgCol := colBtnBg
	if pressed {
		bgCol = colBtnHover
	}
	bbr, _, _ := pCreateSolidBrush.Call(bgCol)
	oldB, _, _ := pSelectObject.Call(dc, bbr)
	pp, _, _ := pCreatePen.Call(PS_SOLID, 1, colBorder)
	oldP, _, _ := pSelectObject.Call(dc, pp)
	pRoundRect.Call(dc, uintptr(rc.left), uintptr(rc.top),
		uintptr(rc.right), uintptr(rc.bottom), uintptr(dp(6)), uintptr(dp(6)))
	pSelectObject.Call(dc, oldP)
	pDeleteObject.Call(pp)
	pSelectObject.Call(dc, oldB)
	pDeleteObject.Call(bbr)

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

func drawSwitch(di *drawItemStruct) {
	dc := di.hDC
	rc := di.rcItem
	on := swState[int(di.ctlID)]

	pFillRect.Call(dc, uintptr(unsafe.Pointer(&rc)), brBg)

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
	fontScript, _, _ = pCreateFont.Call(^uintptr(dp(11)), 0, 0, 0, 400, 1, 0, 0,
		DEFAULT_CHARSET, 0, 0, CLEARTYPE_QUALITY, 0,
		uintptr(unsafe.Pointer(utf16ptr("Segoe Script"))))
	if fontScript == 0 {
		fontScript, _, _ = pCreateFont.Call(^uintptr(dp(11)), 0, 0, 0, 400, 1, 0, 0,
			DEFAULT_CHARSET, 0, 0, CLEARTYPE_QUALITY, 0,
			uintptr(unsafe.Pointer(utf16ptr("Segoe UI"))))
	}
}

func rebuildFonts() {
	deleteGdiObj(&fontNormal)
	deleteGdiObj(&fontBold)
	deleteGdiObj(&fontTitle)
	deleteGdiObj(&fontSection)
	deleteGdiObj(&fontHint)
	deleteGdiObj(&fontScript)
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
	// Hero Switch
	addBtn(cHeroSwitch, "")
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
	mkCtl(h, ctlDef{cHotkeyBox, "EDIT", WS_TABSTOP | 0x00800000 | 0x0001, "", 0, 0, 10, 10})
	stHotkeyHint = addHint("A-Z hoặc 0-9 · trống = Ctrl+Shift")
	addSw(cUseCtrlShift, "Dùng thêm Ctrl+Shift")
	stFKeySection = addSection("PHÍM NHANH (F-KEYS)")
	addSw(cF1, "F1 — Bật tiếng Việt")
	addSw(cF2, "F2 — Tắt tiếng Việt")
	addSw(cF5, "F5 — Mở cài đặt")
	addSw(cF9, "F9 — Bật/tắt gõ tắt")
	addSw(cF12, "F12 — Xóa bộ đệm gõ")

	// ---- Tab 3: Giao diện & Hệ thống ----
	curTab = 3
	stSysSection = addSection("GIAO DIỆN & HỆ THỐNG")
	stLbTheme = addLabel("Giao diện")
	cth := mkCtl(h, ctlDef{cTheme, "COMBOBOX", CBS_DROPDOWNLIST | WS_TABSTOP, "", 0, 0, 10, 10})
	comboAdd(cth, []string{"Theo hệ thống", "Tối (Thanh Đại)", "Sáng (Bạch Ngọc)"}, cfg.Theme)
	if isDark() {
		darkCtl(cth)
	}
	addSw(cShowHud, "Hiện thanh trạng thái nổi")
	addSw(cSound, "Âm thanh khi chuyển chế độ")
	addSw(cSkipLayout, "Tự tắt khi bàn phím không phải kiểu US")
	addSw(cClipboard, "Dùng Clipboard cho app không nhận phím")
	addSw(cAdmin, "Chạy với quyền Admin")
	addSw(cShowDlg, "Hiện bảng này khi khởi động Windows")
	addBtn(cExcludeBrowse, "Chọn app…")
	addBtn(cExcludeEdit, "Danh sách loại trừ…")
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
	setTab(0)
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
	cx := railW + dp(28)
	cw := winW - cx - dp(28)
	ct := dp(106)

	// Tab 0 — Tổng quan
	place(cHeroSwitch, cx+cw-dp(56), ct+dp(16), dp(44), dp(24))
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
	place(stIMSection, cx, ct, cw, dp(20))
	place(stLbIM, cx, ct+dp(32), dp(80), dp(22))
	place(cIMCombo, cx+dp(90), ct+dp(30), cw-dp(90), dp(200))
	place(stLbCharset, cx, ct+dp(68), dp(80), dp(22))
	place(cCharset, cx+dp(90), ct+dp(66), cw-dp(90), dp(140))
	place(stMacroSection, cx, ct+dp(108), cw, dp(20))
	place(cMacro, cx, ct+dp(136), cw, dp(24))
	place(cMacroAlways, cx, ct+dp(164), cw, dp(24))
	place(cMacroEdit, cx, ct+dp(198), dp(160), dp(30))
	place(stMacroHint1, cx, ct+dp(236), cw, dp(20))
	place(stMacroHint2, cx, ct+dp(258), cw, dp(20))

	// Tab 2 — Phím tắt
	place(stSwitchSection, cx, ct, cw, dp(20))
	// 2×2 grid: a single row of 4 pills is too narrow at high DPI and
	// clips labels ("Shift"→"Shif").
	mw := (cw - dp(8)) / 2
	place(cModCtrl, cx, ct+dp(30), mw, dp(24))
	place(cModShift, cx+mw+dp(8), ct+dp(30), mw, dp(24))
	place(cModAlt, cx, ct+dp(60), mw, dp(24))
	place(cModWin, cx+mw+dp(8), ct+dp(60), mw, dp(24))
	place(stLbHotkey, cx, ct+dp(96), dp(50), dp(22))
	place(cHotkeyBox, cx+dp(56), ct+dp(94), dp(48), dp(24))
	place(stHotkeyHint, cx+dp(116), ct+dp(96), cw-dp(116), dp(20))
	place(cUseCtrlShift, cx, ct+dp(130), cw, dp(24))
	place(stFKeySection, cx, ct+dp(166), cw, dp(20))
	place(cF1, cx, ct+dp(194), cw, dp(22))
	place(cF2, cx, ct+dp(220), cw, dp(22))
	place(cF5, cx, ct+dp(246), cw, dp(22))
	place(cF9, cx, ct+dp(272), cw, dp(22))
	place(cF12, cx, ct+dp(298), cw, dp(22))

	// Tab 3 — Giao diện & Hệ thống
	place(stSysSection, cx, ct, cw, dp(20))
	place(stLbTheme, cx, ct+dp(30), dp(76), dp(22))
	place(cTheme, cx+dp(80), ct+dp(28), dp(180), dp(140))
	sysSw := []int{cShowHud, cSound, cSkipLayout, cClipboard, cAdmin, cShowDlg}
	for row, id := range sysSw {
		place(id, cx, ct+dp(64)+int32(row)*dp(26), cw, dp(24))
	}
	place(cExcludeBrowse, cx, ct+dp(228), dp(120), dp(30))
	place(cExcludeEdit, cx+dp(128), ct+dp(228), dp(170), dp(30))
	place(stExcludeHint, cx, ct+dp(266), cw, dp(48))

	// Tab 4 — Giới thiệu (button sits inside the card's bottom-right)
	mascotY := ct + dp(16)
	mascotS := dp(110)
	place(cUpdateBtn, cx+cw-dp(24)-dp(170), mascotY+mascotS+dp(228), dp(170), dp(32))

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
	pSendMessage.Call(ctlHnd[cTheme], CB_SETCURSEL, uintptr(cfg.Theme), 0)

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

type openFileNameW struct {
	lStructSize       uint32
	_                 uint32
	hwndOwner         uintptr
	hInstance         uintptr
	lpstrFilter       uintptr
	lpstrCustomFilter uintptr
	nMaxCustFilter    uint32
	_                 uint32
	nFilterIndex      uint32
	_                 uint32
	lpstrFile         uintptr
	nMaxFile          uint32
	_                 uint32
	lpstrFileTitle    uintptr
	nMaxFileTitle     uint32
	_                 uint32
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
	if unsafe.Sizeof(openFileNameW{}) != 160 {
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
	case cHeroSwitch:
		toggleVietKey()
		pInvalidateRect.Call(gSettingsHwnd, 0, 1)
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
	defer func() { recoverCrash("settings") }()
	switch msg {
	case WM_CREATE:
		buildControls(hwnd)
		return 0
	case WM_SIZE:
		if wp != 1 {
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

		// Top header area: window controls & window dragging
		if y < dp(46) {
			if x >= winW-dp(36) && x <= winW-dp(8) { // Close button
				pShowWindow.Call(hwnd, SW_HIDE)
				return 0
			}
			if x >= winW-dp(68) && x <= winW-dp(40) { // Maximize / Restore
				return 0
			}
			if x >= winW-dp(100) && x <= winW-dp(72) { // Minimize -> Hide to tray
				pShowWindow.Call(hwnd, SW_HIDE)
				return 0
			}
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

		// Tab 0 Hero Card click: toggles VietKey
		if gActiveTab == 0 {
			heroRc := rect{railW + dp(24), dp(106), winW - dp(24), dp(106) + dp(56)}
			if x >= heroRc.left && x <= heroRc.right && y >= heroRc.top && y <= heroRc.bottom {
				settingsCommand(cHeroSwitch)
				return 0
			}
		}

	case WM_DRAWITEM:
		di := (*drawItemStruct)(*(*unsafe.Pointer)(unsafe.Pointer(&lp)))
		if di.ctlType == ODT_BUTTON {
			id := int(di.ctlID)
			switch {
			case isCheckbox[id]:
				drawCheckbox(di)
			case isRadio[id]:
				drawRadio(di)
			case id == cHeroSwitch:
				drawHeroSwitch(di)
			case id == cLangBtn:
				drawLangBtn(di)
			case swIDs[id]:
				drawSwitch(di)
			case btnIDs[id]:
				drawButton(di)
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
		if bgCtls[lp] {
			return brBg
		}
		return brCard
	case WM_CTLCOLORLISTBOX:
		pSetTextColor.Call(wp, colText)
		pSetBkMode.Call(wp, TRANSPARENT)
		return brCard
	case WM_CTLCOLOREDIT:
		textCol := colText
		if lp == ctlHnd[cHotkeyBox] {
			textCol = colJade
			if !isDark() {
				textCol = colVermilion
			}
		}
		pSetTextColor.Call(wp, textCol)
		pSetBkMode.Call(wp, TRANSPARENT)
		return brCard
	case WM_ERASEBKGND:
		var rc rect
		pGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rc)))
		winW, winH := rc.right, rc.bottom

		// 1. Fill entire window background
		pFillRect.Call(wp, uintptr(unsafe.Pointer(&rc)), brBg)

		// 2. Outer hairline border with rounded corners — full gold in dark
		// mode, a faint dim line in light mode to keep it clean.
		outerPen := penGold
		if !isDark() {
			outerPen = penGoldDim
		}
		oldB, _, _ := pSelectObject.Call(wp, brBg)
		oldP, _, _ := pSelectObject.Call(wp, outerPen)
		pRoundRect.Call(wp, 0, 0, uintptr(winW), uintptr(winH), uintptr(dp(18)), uintptr(dp(18)))
		pSelectObject.Call(wp, oldP)
		pSelectObject.Call(wp, oldB)

		// 3. Top bar: Brand title, subtitle, golden cloud filigree, controls
		drawTopHeader(wp, winW)

		// 4. Sidebar vertical divider
		divLine := rect{railW, dp(50), railW + 1, winH - dp(14)}
		pFillRect.Call(wp, uintptr(unsafe.Pointer(&divLine)), brBorder)

		// 5. Sidebar tabs
		drawNav(wp)

		// 6. Right Content Pane depending on tab
		cx := railW + dp(24)
		cw := winW - cx - dp(24)
		ct := dp(106)

		switch gActiveTab {
		case 0:
			drawOverviewTab(wp, cx, ct, cw, winH)
		case 4:
			drawAboutTab(wp, cx, ct, cw, winH)
		default:
			drawTitleBlock(wp, cx, cw, tabItems[gActiveTab].name, tabItems[gActiveTab].desc)
		}

		return 1
	case WM_CLOSE:
		pShowWindow.Call(hwnd, SW_HIDE)
		return 0
	case WM_DESTROY:
		gSettingsHwnd = 0
		if rethemed {
			rethemed = false
			applyTheme()
			openSettings()
		}
		return 0
	}
	r, _, _ := pDefWindowProc.Call(hwnd, uintptr(msg), wp, lp)
	return r
}

func openSettings() {
	if gSettingsHwnd != 0 {
		pShowWindow.Call(gSettingsHwnd, SW_SHOW)
		pSetForegroundWnd.Call(gSettingsHwnd)
		return
	}
	cls := utf16ptr("BoGoSettingsWnd")
	hIcon := loadAppIcon()
	wc := wndClassEx{
		cbSize:        uint32(unsafe.Sizeof(wndClassEx{})),
		lpfnWndProc:   windows.NewCallback(settingsProc),
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

	gSettingsHwnd, _, _ = pCreateWindowEx.Call(
		0x00040000,
		uintptr(unsafe.Pointer(cls)),
		uintptr(unsafe.Pointer(utf16ptr("Tui Gõ — Cài đặt"))),
		0x80000000|0x00020000|0x00010000,
		uintptr(posX), uintptr(posY), uintptr(winW), uintptr(winH),
		0, 0, 0, 0)

	rgn, _, _ := pCreateRoundRectRgn.Call(0, 0, uintptr(winW+1), uintptr(winH+1), uintptr(dp(18)), uintptr(dp(18)))
	pSetWindowRgn.Call(gSettingsHwnd, rgn, 1)

	pShowWindow.Call(gSettingsHwnd, SW_SHOW)
	pSetForegroundWnd.Call(gSettingsHwnd)
}
