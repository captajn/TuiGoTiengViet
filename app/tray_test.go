package main

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Exercise the actual Win32 conversion: incorrect DIB layout or mask stride
// can yield a valid-looking PNG but an empty or corrupted notification icon.
func TestTrayIconHandles(t *testing.T) {
	getIconInfo := user32.NewProc("GetIconInfo")
	getObject := gdi32.NewProc("GetObjectW")
	type bitmap struct {
		kind, width, height, widthBytes int32
		planes, bitsPixel               uint16
		bits                            uintptr
	}
	for _, theme := range []struct {
		name string
		p    palette
	}{{"dark", palDark}, {"light", palLight}} {
		for _, size := range []int{16, 20, 24, 32, 48} {
			for _, on := range []bool{true, false} {
				letter := "E"
				if on {
					letter = "V"
				}
				name := fmt.Sprintf("%s-%s-%d", theme.name, letter, size)
				t.Run(name, func(t *testing.T) {
					icon := makeTrayIcon(on, size, theme.p)
					if icon == 0 {
						t.Fatal("CreateIconIndirect failed")
					}
					defer pDestroyIcon.Call(icon)
					var info iconInfo
					if r, _, _ := getIconInfo.Call(icon, uintptr(unsafe.Pointer(&info))); r == 0 {
						t.Fatal("GetIconInfo failed")
					}
					defer pDeleteObject.Call(info.hbmColor)
					defer pDeleteObject.Call(info.hbmMask)
					for _, h := range []uintptr{info.hbmColor, info.hbmMask} {
						var bm bitmap
						if r, _, _ := getObject.Call(h, unsafe.Sizeof(bm), uintptr(unsafe.Pointer(&bm))); r == 0 {
							t.Fatal("GetObject failed")
						}
						if bm.width != int32(size) || bm.height != int32(size) {
							t.Fatalf("unexpected native bitmap size: %dx%d", bm.width, bm.height)
						}
					}
					checkTrayComposite(t, icon, size, on, theme.p)
					// Optional visual inspection of the same renderer used by the app.
					if dir := os.Getenv("TUIGO_TRAY_PREVIEW"); dir != "" {
						if err := os.MkdirAll(dir, 0755); err != nil {
							t.Fatal(err)
						}
						f, err := os.Create(filepath.Join(dir, name+".png"))
						if err != nil {
							t.Fatal(err)
						}
						defer f.Close()
						if err := png.Encode(f, renderTrayIcon(size, on, theme.p)); err != nil {
							t.Fatal(err)
						}
					}
				})
			}
		}
	}
}

func checkTrayComposite(t *testing.T, icon uintptr, size int, on bool, p palette) {
	t.Helper()
	bi := bitmapInfo{}
	bi.bmiHeader.biSize = uint32(unsafe.Sizeof(bitmapInfoHeader{}))
	bi.bmiHeader.biWidth, bi.bmiHeader.biHeight = int32(size), -int32(size)
	bi.bmiHeader.biPlanes, bi.bmiHeader.biBitCount = 1, 32
	var bits unsafe.Pointer
	bmp, _, _ := pCreateDIBSection.Call(0, uintptr(unsafe.Pointer(&bi)), 0, uintptr(unsafe.Pointer(&bits)), 0, 0)
	if bmp == 0 {
		t.Fatal("preview DIB allocation failed")
	}
	defer pDeleteObject.Call(bmp)
	dc, _, _ := pCreateCompatibleDC.Call(0)
	if dc == 0 {
		t.Fatal("preview DC allocation failed")
	}
	defer pDeleteDC.Call(dc)
	old, _, _ := pSelectObject.Call(dc, bmp)
	defer pSelectObject.Call(dc, old)
	pixels := unsafe.Slice((*byte)(bits), size*size*4)
	clear(pixels)
	if r, _, _ := user32.NewProc("DrawIconEx").Call(dc, 0, 0, icon, uintptr(size), uintptr(size), 0, 0, 3); r == 0 {
		t.Fatal("DrawIconEx failed")
	}
	gdi32.NewProc("GdiFlush").Call()
	want := renderTrayIcon(size, on, p)
	for i := 0; i < len(pixels); i += 4 {
		for c := 0; c < 3; c++ {
			delta := int(pixels[i+c]) - int(want.Pix[i+2-c])
			if delta < -1 || delta > 1 {
				t.Fatalf("native alpha composite differs at pixel %d channel %d: delta=%d", i/4, c, delta)
			}
		}
	}
}

func TestTrayEffectiveState(t *testing.T) {
	oldHwnd, oldSpec, oldExe := lastFgHwnd, lastFgSpec, lastFgExe
	oldGlobal, oldPerApp := gVietKey, gPerAppViet
	oldVn, oldEn := gIconVn, gIconEn
	gIconVn, gIconEn = 11, 22 // sentinel handles; notifyIcon does not dereference them
	defer func() {
		lastFgHwnd, lastFgSpec, lastFgExe = oldHwnd, oldSpec, oldExe
		gVietKey, gPerAppViet = oldGlobal, oldPerApp
		gIconVn, gIconEn = oldVn, oldEn
	}()
	for _, tc := range []struct {
		name              string
		mode              int
		global, per, want bool
	}{
		{"global-on", appModeNone, true, false, true},
		{"global-off", appModeNone, false, true, false},
		{"manual-off", appModeManual, true, false, false},
		{"manual-on", appModeManual, false, true, true},
		{"locked", appModeLock, true, true, false},
		{"clipboard", appModeClip, true, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Seed the existing foreground cache without changing desktop focus.
			lastFgHwnd, _, _ = pGetForegroundWindow.Call()
			lastFgSpec, lastFgExe = appSpec{mode: tc.mode, im: -1}, "tray-test.exe"
			gVietKey = tc.global
			gPerAppViet = map[string]bool{lastFgExe: tc.per}
			if got := effectiveViet(); got != tc.want {
				t.Fatalf("effectiveViet = %v, want %v", got, tc.want)
			}
			nid := notifyIcon(NIF_ICON | NIF_TIP)
			icon, tip := gIconEn, "Tiếng Việt đang tắt"
			if tc.want {
				icon, tip = gIconVn, "Tiếng Việt đang bật"
			}
			if nid.hIcon != icon || windows.UTF16ToString(nid.szTip[:]) != tip {
				t.Fatal("tray icon/tooltip disagree with effective input state")
			}
		})
	}
}

// flipVietKey must apply every call — no debounce, so rapid hotkey presses
// each land. Lock-mode apps still ignore the toggle; manual mode flips the
// per-app flag only.
func TestFlipVietKeyRapid(t *testing.T) {
	eng, err := newEngine()
	if err != nil {
		t.Fatal(err)
	}
	oldEngine, oldGlobal, oldPerApp := gEngine, gVietKey, gPerAppViet
	oldHwnd, oldSpec, oldExe := lastFgHwnd, lastFgSpec, lastFgExe
	defer func() {
		gEngine, gVietKey, gPerAppViet = oldEngine, oldGlobal, oldPerApp
		lastFgHwnd, lastFgSpec, lastFgExe = oldHwnd, oldSpec, oldExe
	}()
	gEngine = eng
	lastFgHwnd, _, _ = pGetForegroundWindow.Call()
	lastFgSpec, lastFgExe = appSpec{im: -1}, "flip-test.exe"
	gVietKey = true
	for i := 0; i < 20; i++ {
		if !flipVietKey() {
			t.Fatalf("flip %d ignored", i)
		}
		want := i%2 != 0
		if gVietKey != want {
			t.Fatalf("flip %d: gVietKey = %v, want %v", i, gVietKey, want)
		}
	}

	lastFgSpec = appSpec{mode: appModeLock, im: -1}
	if flipVietKey() {
		t.Fatal("lock-mode app must ignore the toggle")
	}

	lastFgSpec = appSpec{mode: appModeManual, im: -1}
	gPerAppViet = map[string]bool{"flip-test.exe": false}
	if !flipVietKey() || !gPerAppViet["flip-test.exe"] || !gVietKey {
		t.Fatal("manual mode must flip only the per-app flag")
	}
}

// appendBackspaces must never emit a forward-Delete unless delPairs is set
// — mid-text it would erase the real char after the caret ("bánh" lost its
// 'n'). Pairs remain the fallback for end-of-text suggestions when no
// cut-probe ran.
func TestAppendBackspaces(t *testing.T) {
	vks := func(in []input) []uint16 {
		out := make([]uint16, len(in))
		for i, in := range in {
			out[i] = in.Ki.wVk
		}
		return out
	}
	equal := func(a, b []uint16) bool {
		if len(a) != len(b) {
			return false
		}
		for i := range a {
			if a[i] != b[i] {
				return false
			}
		}
		return true
	}

	// plain mode: Backspace only
	if got := vks(appendBackspaces(nil, 2, false)); !equal(got, []uint16{VK_BACK, VK_BACK, VK_BACK, VK_BACK}) {
		t.Fatalf("plain mode: got %v", got)
	}

	// pair mode (selection unresolved, caret at end): Delete+Backspace
	want := []uint16{VK_DELETE, VK_DELETE, VK_BACK, VK_BACK}
	if got := vks(appendBackspaces(nil, 1, true)); !equal(got, want) {
		t.Fatalf("pair mode: got %v", got)
	}
}

// modsMatch reads hook-tracked modifier state (in-order, injection-filtered)
// — GetAsyncKeyState lag or the phantom Ctrl under Alt must not disqualify.
func TestModsMatchTrackedState(t *testing.T) {
	old := [4]bool{gCtrlHeld, gShiftHeld, gAltHeld, gWinHeld}
	defer func() {
		gCtrlHeld, gShiftHeld, gAltHeld, gWinHeld = old[0], old[1], old[2], old[3]
	}()
	set := func(c, s, a, w bool) {
		gCtrlHeld, gShiftHeld, gAltHeld, gWinHeld = c, s, a, w
	}
	for _, tc := range []struct {
		name       string
		c, s, a, w bool
		mask       int
		want       bool
	}{
		{"alt-only matches alt", false, false, true, false, 4, true},
		{"alt-only vs ctrl+alt", false, false, true, false, 5, false},
		{"ctrl+shift matches", true, true, false, false, 3, true},
		{"shift alone misses chord", false, true, false, false, 3, false},
		{"bare matches none", false, false, false, false, 0, true},
		{"stray shift misses bare", false, true, false, false, 0, false},
		{"win matches", false, false, false, true, 8, true},
		{"extra shift misses alt", false, true, true, false, 4, false},
	} {
		set(tc.c, tc.s, tc.a, tc.w)
		if got := modsMatch(tc.mask); got != tc.want {
			t.Fatalf("%s: modsMatch(%d) = %v, want %v", tc.name, tc.mask, got, tc.want)
		}
	}
}
