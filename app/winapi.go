package main

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32  = windows.NewLazySystemDLL("user32.dll")
	gdi32   = windows.NewLazySystemDLL("gdi32.dll")
	sh32    = windows.NewLazySystemDLL("shell32.dll")
	dwmapi  = windows.NewLazySystemDLL("dwmapi.dll")
	uxtheme = windows.NewLazySystemDLL("uxtheme.dll")
	kern32  = windows.NewLazySystemDLL("kernel32.dll")

	pSetWindowsHookEx   = user32.NewProc("SetWindowsHookExW")
	pCallNextHookEx     = user32.NewProc("CallNextHookEx")
	pUnhookWindowsHook  = user32.NewProc("UnhookWindowsHookEx")
	pSetTimer           = user32.NewProc("SetTimer")
	pRegisterWindowMsg  = user32.NewProc("RegisterWindowMessageW")
	pGetMessage         = user32.NewProc("GetMessageW")
	pTranslateMessage   = user32.NewProc("TranslateMessage")
	pDispatchMessage    = user32.NewProc("DispatchMessageW")
	pSendInput          = user32.NewProc("SendInput")
	pGetAsyncKeyState   = user32.NewProc("GetAsyncKeyState")
	pGetKeyState        = user32.NewProc("GetKeyState")
	pToUnicodeEx        = user32.NewProc("ToUnicodeEx")
	pGetKeyboardLayout  = user32.NewProc("GetKeyboardLayout")
	pCreateWindowEx     = user32.NewProc("CreateWindowExW")
	pRegisterClassEx    = user32.NewProc("RegisterClassExW")
	pDefWindowProc      = user32.NewProc("DefWindowProcW")
	pDestroyWindow      = user32.NewProc("DestroyWindow")
	pPostQuitMessage    = user32.NewProc("PostQuitMessage")
	pCreatePopupMenu    = user32.NewProc("CreatePopupMenu")
	pAppendMenu         = user32.NewProc("AppendMenuW")
	pTrackPopupMenu     = user32.NewProc("TrackPopupMenu")
	pDestroyMenu        = user32.NewProc("DestroyMenu")
	pSetForegroundWnd   = user32.NewProc("SetForegroundWindow")
	pGetCursorPos       = user32.NewProc("GetCursorPos")
	pMessageBox         = user32.NewProc("MessageBoxW")
	pGetDC              = user32.NewProc("GetDC")
	pReleaseDC          = user32.NewProc("ReleaseDC")
	pFillRect           = user32.NewProc("FillRect")
	pDrawText           = user32.NewProc("DrawTextW")
	pCreateIconIndirect = user32.NewProc("CreateIconIndirect")
	pDestroyIcon        = user32.NewProc("DestroyIcon")
	pMessageBeep        = user32.NewProc("MessageBeep")

	pShellNotifyIcon = sh32.NewProc("Shell_NotifyIconW")

	pCreateCompatibleDC     = gdi32.NewProc("CreateCompatibleDC")
	pCreateCompatibleBitmap = gdi32.NewProc("CreateCompatibleBitmap")
	pCreateBitmap           = gdi32.NewProc("CreateBitmap")
	pCreateSolidBrush       = gdi32.NewProc("CreateSolidBrush")
	pCreateFont             = gdi32.NewProc("CreateFontW")
	pSelectObject           = gdi32.NewProc("SelectObject")
	pDeleteObject           = gdi32.NewProc("DeleteObject")
	pDeleteDC               = gdi32.NewProc("DeleteDC")
	pSetBkMode              = gdi32.NewProc("SetBkMode")
	pSetTextColor           = gdi32.NewProc("SetTextColor")
	pRoundRect              = gdi32.NewProc("RoundRect")
	pRectangle              = gdi32.NewProc("Rectangle")
	pCreatePen              = gdi32.NewProc("CreatePen")
	pPolygon                = gdi32.NewProc("Polygon")
	pArc                    = gdi32.NewProc("Arc")
	pGetStockObject         = gdi32.NewProc("GetStockObject")
	pPolyline               = gdi32.NewProc("Polyline")
	pMoveToEx               = gdi32.NewProc("MoveToEx")
	pLineTo                 = gdi32.NewProc("LineTo")
	pEllipse                = gdi32.NewProc("Ellipse")
	pCreateRoundRectRgn     = gdi32.NewProc("CreateRoundRectRgn")
	pSetWindowRgn           = user32.NewProc("SetWindowRgn")
	pBeginPaint             = user32.NewProc("BeginPaint")
	pEndPaint               = user32.NewProc("EndPaint")
	pSetCapture             = user32.NewProc("SetCapture")
	pReleaseCapture         = user32.NewProc("ReleaseCapture")
	pInvalidateRect         = user32.NewProc("InvalidateRect")
	pGetWindowRect          = user32.NewProc("GetWindowRect")
	pScreenToClient         = user32.NewProc("ScreenToClient")
	pSelectClipRgn          = gdi32.NewProc("SelectClipRgn")
	pStretchDIBits          = gdi32.NewProc("StretchDIBits")
	pSetStretchBltMode      = gdi32.NewProc("SetStretchBltMode")

	pGetModuleHandle  = kern32.NewProc("GetModuleHandleW")
	pExtractIcon      = sh32.NewProc("ExtractIconW")
	pDwmSetWndAttr    = dwmapi.NewProc("DwmSetWindowAttribute")
	pSetWindowTheme   = uxtheme.NewProc("SetWindowTheme")
	pLoadCursor       = user32.NewProc("LoadCursorW")
	pSetWindowPos     = user32.NewProc("SetWindowPos")
	pGetSystemMetrics = user32.NewProc("GetSystemMetrics")
)

// loadAppIcon returns the application mascot icon or falls back to exe icon.
func loadAppIcon() uintptr {
	if ic := createMascotIcon(32); ic != 0 {
		return ic
	}
	var exe [windows.MAX_PATH]uint16
	windows.GetModuleFileName(0, &exe[0], windows.MAX_PATH)
	h, _, _ := pExtractIcon.Call(0, uintptr(unsafe.Pointer(&exe[0])), 0)
	if h > 1 { // ExtractIconW returns >1 on success
		return h
	}
	return 0
}

// darkTitlebar sets the Win10/11 immersive dark frame to match the theme.
func darkTitlebar(hwnd uintptr) {
	on := int32(0)
	if isDark() {
		on = 1
	}
	const DWMWA_USE_IMMERSIVE_DARK_MODE = 20
	pDwmSetWndAttr.Call(hwnd, DWMWA_USE_IMMERSIVE_DARK_MODE,
		uintptr(unsafe.Pointer(&on)), 4)
}

// darkCtl applies the dark theme to a standard control (Win10 1809+).
func darkCtl(hwnd uintptr) {
	pSetWindowTheme.Call(hwnd,
		uintptr(unsafe.Pointer(utf16ptr("DarkMode_CFD"))), 0)
}

func loadArrowCursor() uintptr {
	c, _, _ := pLoadCursor.Call(0, IDC_ARROW)
	return c
}

const (
	WH_KEYBOARD_LL   = 13
	WH_MOUSE_LL      = 14
	WM_LBUTTONDOWN   = 0x0201
	WM_RBUTTONDOWN   = 0x0204
	WM_MBUTTONDOWN   = 0x0207
	WM_KEYDOWN       = 0x0100
	WM_SYSKEYDOWN    = 0x0104
	WM_KEYUP         = 0x0101
	WM_SYSKEYUP      = 0x0105
	WM_COMMAND       = 0x0111
	WM_DESTROY       = 0x0002
	WM_ENDSESSION    = 0x0016
	WM_TIMER         = 0x0113
	WM_RBUTTONUP     = 0x0205
	WM_LBUTTONDBLCLK = 0x0203
	WM_CONTEXTMENU   = 0x007B
	WM_APP           = 0x8000
	WM_TRAYICON      = WM_APP + 1

	LLKHF_INJECTED          = 0x10
	LLKHF_LOWER_IL_INJECTED = 0x02

	VK_BACK     = 0x08
	VK_TAB      = 0x09
	VK_RETURN   = 0x0D
	VK_SHIFT    = 0x10
	VK_CONTROL  = 0x11
	VK_MENU     = 0x12
	VK_PAUSE    = 0x13
	VK_CAPITAL  = 0x14
	VK_ESCAPE   = 0x1B
	VK_SPACE    = 0x20
	VK_PRIOR    = 0x21
	VK_NEXT     = 0x22
	VK_END      = 0x23
	VK_HOME     = 0x24
	VK_LEFT     = 0x25
	VK_UP       = 0x26
	VK_RIGHT    = 0x27
	VK_DOWN     = 0x28
	VK_SNAPSHOT = 0x2C
	VK_INSERT   = 0x2D
	VK_DELETE   = 0x2E
	VK_LWIN     = 0x5B
	VK_RWIN     = 0x5C
	VK_APPS     = 0x5D
	VK_NUMLOCK  = 0x90
	VK_SCROLL   = 0x91
	VK_LSHIFT   = 0xA0
	VK_RSHIFT   = 0xA1
	VK_LCONTROL = 0xA2
	VK_RCONTROL = 0xA3
	VK_LMENU    = 0xA4
	VK_RMENU    = 0xA5
	VK_F1       = 0x70
	VK_F2       = 0x71
	VK_F5       = 0x74
	VK_F9       = 0x78
	VK_F12      = 0x7B
	VK_F24      = 0x87

	INPUT_KEYBOARD    = 1
	KEYEVENTF_KEYUP   = 0x0002
	KEYEVENTF_UNICODE = 0x0004

	MF_STRING    = 0x0000
	MF_CHECKED   = 0x0008
	MF_POPUP     = 0x0010
	MF_SEPARATOR = 0x0800

	TPM_BOTTOMALIGN = 0x0020
	TPM_LEFTALIGN   = 0x0000

	NIM_ADD     = 0x00
	NIM_MODIFY  = 0x01
	NIM_DELETE  = 0x02
	NIF_MESSAGE = 0x01
	NIF_ICON    = 0x02
	NIF_TIP     = 0x04

	HWND_MESSAGE = ^uintptr(2) // (HWND)-3

	DT_LEFT             = 0x00
	DT_CENTER           = 0x01
	DT_RIGHT            = 0x02
	DT_VCENTER          = 0x04
	DT_WORDBREAK        = 0x10
	DT_SINGLELINE       = 0x20
	DT_NOPREFIX         = 0x0800
	TRANSPARENT         = 1
	FW_BOLD             = 700
	ANTIALIASED_QUALITY = 4
	CLEARTYPE_QUALITY   = 5
	IDC_ARROW           = 32512
	PS_SOLID            = 0
	NULL_BRUSH          = 5
	NULL_PEN            = 8
	PS_DOT              = 2
	DEFAULT_CHARSET     = 1

	ERROR_ALREADY_EXISTS = 183
)

type kbdLLHookStruct struct {
	vkCode      uint32
	scanCode    uint32
	flags       uint32
	time        uint32
	dwExtraInfo uintptr
}

type point struct{ x, y int32 }

type rect struct{ left, top, right, bottom int32 }

type msg struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      point
}

type wndClassEx struct {
	cbSize        uint32
	style         uint32
	lpfnWndProc   uintptr
	cbClsExtra    int32
	cbWndExtra    int32
	hInstance     uintptr
	hIcon         uintptr
	hCursor       uintptr
	hbrBackground uintptr
	lpszMenuName  *uint16
	lpszClassName *uint16
	hIconSm       uintptr
}

type iconInfo struct {
	fIcon    int32
	xHotspot uint32
	yHotspot uint32
	hbmMask  uintptr
	hbmColor uintptr
}

type bitmapInfo struct {
	bmiHeader bitmapInfoHeader
	bmiColors [1]uint32
}

type bitmapInfoHeader struct {
	biSize          uint32
	biWidth         int32
	biHeight        int32
	biPlanes        uint16
	biBitCount      uint16
	biCompression   uint32
	biSizeImage     uint32
	biXPelsPerMeter int32
	biYPelsPerMeter int32
	biClrUsed       uint32
	biClrImportant  uint32
}

type guid struct {
	data1 uint32
	data2 uint16
	data3 uint16
	data4 [8]byte
}

type notifyIconData struct {
	cbSize            uint32
	hWnd              uintptr
	uID               uint32
	uFlags            uint32
	uCallbackMessage  uint32
	hIcon             uintptr
	szTip             [128]uint16
	dwState           uint32
	dwStateMask       uint32
	szInfo            [256]uint16
	uTimeoutOrVersion uint32
	szInfoTitle       [64]uint16
	dwInfoFlags       uint32
	guidItem          guid
	hBalloonIcon      uintptr
}

type keybdInput struct {
	wVk         uint16
	wScan       uint16
	dwFlags     uint32
	time        uint32
	dwExtraInfo uintptr
}

// INPUT on x64: type(4) + pad(4) + 32-byte union = 40 bytes
type input struct {
	Type uint32
	_    uint32
	Ki   keybdInput
	_    [8]byte
}

func utf16ptr(s string) *uint16 {
	p, _ := windows.UTF16PtrFromString(s)
	return p
}

func init() {
	if unsafe.Sizeof(input{}) != 40 {
		panic("INPUT struct size mismatch")
	}
}

func sendInput(inputs []input) {
	pSendInput.Call(uintptr(len(inputs)), uintptr(unsafe.Pointer(&inputs[0])), unsafe.Sizeof(input{}))
}
