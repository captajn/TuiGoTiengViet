package main

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
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
	}{
		{"dark", palDark},
		{"light", palLight},
		{"ios_light", palIosLight},
		{"ios_dark", palIosDark},
		{"cyber_dark", palCyberDark},
		{"cyber_light", palCyberLight},
		{"matcha_light", palMatchaLight},
		{"matcha_dark", palMatchaDark},
		{"material_dark", palMaterialDark},
		{"material_light", palMaterialLight},
		{"minimal_dark", palMinimalDark},
		{"minimal_light", palMinimalLight},
	} {
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
	oldInputApp := gLastInputApp
	oldGlobal, oldPerApp := gVietKey, gPerAppViet
	oldVn, oldEn := gIconVn, gIconEn
	gIconVn, gIconEn = 11, 22 // sentinel handles; notifyIcon does not dereference them
	defer func() {
		lastFgHwnd, lastFgSpec, lastFgExe = oldHwnd, oldSpec, oldExe
		gLastInputApp = oldInputApp
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
			focusAppForTest("tray-test.exe", appSpec{mode: tc.mode, im: -1})
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
	oldInputApp := gLastInputApp
	defer func() {
		gEngine, gVietKey, gPerAppViet = oldEngine, oldGlobal, oldPerApp
		lastFgHwnd, lastFgSpec, lastFgExe = oldHwnd, oldSpec, oldExe
		gLastInputApp = oldInputApp
	}()
	gEngine = eng
	focusAppForTest("flip-test.exe", appSpec{im: -1})
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

	focusAppForTest("flip-test.exe", appSpec{mode: appModeLock, im: -1})
	if flipVietKey() {
		t.Fatal("lock-mode app must ignore the toggle")
	}

	focusAppForTest("flip-test.exe", appSpec{mode: appModeManual, im: -1})
	gPerAppViet = map[string]bool{"flip-test.exe": false}
	if !flipVietKey() || !gPerAppViet["flip-test.exe"] || !gVietKey {
		t.Fatal("manual mode must flip only the per-app flag")
	}

	// A later global toggle is the master switch: it drops every per-app
	// override so excluded apps return to their excluded default instead
	// of staying un-excluded for the rest of the session.
	focusAppForTest("flip-test.exe", appSpec{im: -1})
	if !flipVietKey() || len(gPerAppViet) != 0 {
		t.Fatal("global toggle must clear per-app overrides")
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

func TestSidebarHoverPerformanceAndHitTest(t *testing.T) {
	oldDpi, oldRailW := gDpi, railW
	oldActive, oldHovered := gActiveTab, gHoveredTab
	oldCfg := cfg
	defer func() {
		releaseBgCache()
		deleteGdiObj(&fontNormal)
		deleteGdiObj(&fontBold)
		deleteGdiObj(&fontTitle)
		deleteGdiObj(&fontSection)
		deleteGdiObj(&fontHint)
		deleteGdiObj(&fontTagline)
		gDpi, railW = oldDpi, oldRailW
		gActiveTab, gHoveredTab = oldActive, oldHovered
		cfg = oldCfg
		applyTheme()
	}()

	gDpi = 96
	railW = dp(232)
	applySettingsFont()

	// Verify continuous vertical sweep across all 5 tabs has no -1 dead-zone in the 6dp gaps
	firstR := navRect(0)
	lastR := navRect(len(tabItems) - 1)
	midX := (firstR.left + firstR.right) / 2
	prevIdx := 0
	for y := firstR.top; y < lastR.bottom; y++ {
		idx := navHitAt(midX, y)
		if idx < 0 || idx >= len(tabItems) {
			t.Fatalf("navHitAt(%d, %d) returned %d inside sidebar list", midX, y, idx)
		}
		if idx < prevIdx || idx > prevIdx+1 {
			t.Fatalf("non-monotonic tab transition at y=%d: %d -> %d", y, prevIdx, idx)
		}
		prevIdx = idx
	}
	if got := navHitAt(midX, firstR.top-1); got != -1 {
		t.Fatalf("above first tab: got %d, want -1", got)
	}
	if got := navHitAt(midX, lastR.bottom); got != -1 {
		t.Fatalf("below last tab: got %d, want -1", got)
	}

	// Benchmark cached hover repaint on the last tab (About tab with 1024x1024 mascot)
	winW, winH := dp(840), dp(560)
	screen, _, _ := pGetDC.Call(0)
	if screen == 0 {
		t.Skip("no screen DC available")
	}
	defer pReleaseDC.Call(0, screen)
	dc, bmp, old := allocOffscreenDC(screen, winW, winH)
	if dc == 0 {
		t.Fatal("failed to allocate test DC")
	}
	defer func() {
		pSelectObject.Call(dc, old)
		pDeleteDC.Call(dc)
		pDeleteObject.Call(bmp)
	}()

	gActiveTab = len(tabItems) - 1

	// Sweep hover across all tabs: full paint once, then dirty-rect repaints
	// (4 passes x all tabs = 20 steps).
	sweep := func() time.Duration {
		gHoveredTab = -1
		paintSettingsCached(dc, 0, winW, winH, rect{0, 0, winW, winH})
		start := time.Now()
		steps := 0
		for pass := 0; pass < 4; pass++ {
			for i := 0; i < len(tabItems); i++ {
				oldH := gHoveredTab
				gHoveredTab = i
				dirty := navDirtyRect(i)
				if oldH >= 0 {
					dirty = rectUnion(dirty, navDirtyRect(oldH))
				}
				paintSettingsCached(dc, 0, winW, winH, dirty)
				steps++
			}
		}
		return time.Since(start) / time.Duration(steps)
	}

	// Heaviest visual path first (Jade dark glass), then the flat Terminal
	// mode — both must stay under the same dirty-rect repaint budget.
	cfg.Theme, cfg.Style = themeDark, styleJade
	applyTheme()
	if avg := sweep(); avg > 5*time.Millisecond {
		t.Fatalf("jade sidebar hover repaint too slow: avg %v/step (want < 5ms)", avg)
	} else {
		t.Logf("jade dark hover repaint: avg %v/step", avg)
	}

	cfg.Theme, cfg.Style = themeDark, styleMinimal
	applyTheme()
	if avg := sweep(); avg > 5*time.Millisecond {
		t.Fatalf("minimal sidebar hover repaint too slow: avg %v/step (want < 5ms)", avg)
	} else {
		t.Logf("minimal dark hover repaint: avg %v/step", avg)
	}
}

// Terminal is a flat, neutral, square-cornered style: its palettes carry no
// jade brand tokens, applyTheme plumbs them into the GDI+ globals, radii
// collapse to zero and selected cards take the solid-accent code path.
func TestMinimalStyleFlat(t *testing.T) {
	oldCfg := cfg
	defer func() {
		cfg = oldCfg
		applyTheme()
	}()

	for _, tc := range []struct {
		name  string
		theme int
		pal   palette
	}{
		{"light", themeLight, palMinimalLight},
		{"dark", themeDark, palMinimalDark},
	} {
		cfg.Theme, cfg.Style = tc.theme, styleMinimal
		applyTheme()
		if got := curPalette(); got != tc.pal {
			t.Fatalf("%s: curPalette() did not return the minimal palette", tc.name)
		}
		if argbAccent != tc.pal.aAccent || argbSwitchOn != tc.pal.aSwitchOn {
			t.Fatalf("%s: applyTheme did not plumb minimal tokens: accent %#x switch %#x",
				tc.name, argbAccent, argbSwitchOn)
		}
		for _, rgb := range []uint32{tc.pal.aAccent & 0x00FFFFFF, tc.pal.aSwitchOn & 0x00FFFFFF} {
			if rgb == 0x197F64 || rgb == 0x3ED3B8 {
				t.Fatalf("%s: minimal accent token is jade %#x — the style must stay neutral", tc.name, rgb)
			}
		}
	}

	cfg.Theme, cfg.Style = themeDark, styleMinimal
	if r := styleRadius(float32(dp(8))); r != 0 {
		t.Fatalf("styleRadius under minimal = %v, want 0", r)
	}
	if !solidAccentSelected() {
		t.Fatal("minimal selected cards must use the solid-accent path")
	}
	cfg.Style = styleJade
	if r := styleRadius(8.0); r != 8.0 {
		t.Fatalf("styleRadius(8) under jade = %v, want 8", r)
	}
}

// The 500 ms tray timer must reuse the existing HICONs when nothing changed.
// A missing trayCols update made the cache check miss on every tick, so both
// icons were rebuilt and the old handles destroyed twice per second.
func TestTrayIconCacheReuse(t *testing.T) {
	oldCfg := cfg
	oldDark, oldTheme, oldStyle, oldSize, oldCols := trayDark, trayTheme, trayStyle, traySize, trayCols
	oldVn, oldEn, oldAdded := gIconVn, gIconEn, gTrayAdded
	defer func() {
		pDestroyIcon.Call(gIconVn)
		pDestroyIcon.Call(gIconEn)
		cfg = oldCfg
		trayDark, trayTheme, trayStyle, traySize, trayCols = oldDark, oldTheme, oldStyle, oldSize, oldCols
		gIconVn, gIconEn, gTrayAdded = oldVn, oldEn, oldAdded
	}()

	cfg.Theme, cfg.Style = 1, 0
	cfg.TrayV, cfg.TrayVBg, cfg.TrayE, cfg.TrayEBg = 0x112233, 0x445566, 0x778899, 0xAABBCC
	gIconVn, gIconEn, gTrayAdded = 0, 0, false
	trayCols = [4]int{}

	refreshTrayIcons()
	if gIconVn == 0 || gIconEn == 0 {
		t.Fatal("first refresh did not create both icons")
	}

	vn, en := gIconVn, gIconEn
	refreshTrayIcons()
	if gIconVn != vn || gIconEn != en {
		t.Fatal("unchanged state rebuilt the tray icons")
	}
	if trayCols != trayCustom() {
		t.Fatalf("trayCols = %v, want %v", trayCols, trayCustom())
	}

	cfg.TrayV = 0x010203
	refreshTrayIcons()
	if gIconVn == vn || gIconEn == en {
		t.Fatal("color change did not rebuild the tray icons")
	}
	if trayCols != trayCustom() {
		t.Fatalf("trayCols after change = %v, want %v", trayCols, trayCustom())
	}
}

func TestUILangAndMiniHudAndWin10Corners(t *testing.T) {
	oldLang := cfg.UILang
	oldRound := gDwmRoundCorners
	defer func() {
		cfg.UILang = oldLang
		gDwmRoundCorners = oldRound
	}()

	cfg.UILang = 0
	if got := tr("Tiếng Việt", "English"); got != "Tiếng Việt" {
		t.Fatalf("UILang=0: got %q", got)
	}
	cfg.UILang = 1
	if got := tr("Tiếng Việt", "English"); got != "English" {
		t.Fatalf("UILang=1: got %q", got)
	}

	// Verify mini HUD badge renders transparent corners (fixing Win10 square corner artifacts)
	for _, on := range []bool{true, false} {
		pixels := make([]uint32, 34*34)
		renderHudPixels(pixels, 34, on, palDark)
		if topLeftA := pixels[0] >> 24; topLeftA != 0 {
			t.Fatalf("renderHudPixels(on=%v) top-left corner alpha = %d, want 0", on, topLeftA)
		}
		if centerA := pixels[17*34+17] >> 24; centerA == 0 {
			t.Fatalf("renderHudPixels(on=%v) center pixel alpha = 0, want opaque", on)
		}
	}
}

// Steady-state memory guard: repeated paint cycles (full static rebuilds +
// dirty-rect fast paths) must not grow the process — neither the Win32 GDI
// handle count nor the Go heap beyond GC headroom. Covers a carded style and
// Terminal so neither rendering path can regress into a slow leak.
func TestPaintCycleMemoryStability(t *testing.T) {
	oldDpi, oldRailW := gDpi, railW
	oldActive, oldHovered := gActiveTab, gHoveredTab
	oldCfg := cfg
	defer func() {
		releaseBgCache()
		deleteGdiObj(&fontNormal)
		deleteGdiObj(&fontBold)
		deleteGdiObj(&fontTitle)
		deleteGdiObj(&fontSection)
		deleteGdiObj(&fontHint)
		deleteGdiObj(&fontTagline)
		gDpi, railW = oldDpi, oldRailW
		gActiveTab, gHoveredTab = oldActive, oldHovered
		cfg = oldCfg
		applyTheme()
	}()

	gDpi = 96
	railW = dp(232)
	applySettingsFont()

	getGuiResources := user32.NewProc("GetGuiResources")
	curProc, _, _ := windows.NewLazySystemDLL("kernel32.dll").NewProc("GetCurrentProcess").Call()
	const grGdiObjects = 0
	gdiCount := func() uint32 {
		v, _, _ := getGuiResources.Call(curProc, grGdiObjects)
		return uint32(v)
	}

	winW, winH := dp(840), dp(560)
	screen, _, _ := pGetDC.Call(0)
	if screen == 0 {
		t.Skip("no screen DC available")
	}
	defer pReleaseDC.Call(0, screen)
	dc, bmp, old := allocOffscreenDC(screen, winW, winH)
	if dc == 0 {
		t.Fatal("failed to allocate test DC")
	}
	defer func() {
		pSelectObject.Call(dc, old)
		pDeleteDC.Call(dc)
		pDeleteObject.Call(bmp)
	}()

	gActiveTab = 0
	full := rect{0, 0, winW, winH}
	for _, tc := range []struct {
		name  string
		style int
	}{
		{"jade", styleJade},
		{"terminal", styleMinimal},
	} {
		cfg.Theme, cfg.Style = themeDark, tc.style
		applyTheme()
		paintSettingsCached(dc, 0, winW, winH, full) // warm every cache layer

		runtime.GC()
		before := gdiCount()
		var memBefore runtime.MemStats
		runtime.ReadMemStats(&memBefore)

		for i := 0; i < 30; i++ {
			gStaticValid = false // force the full static-layer rebuild path
			paintSettingsCached(dc, 0, winW, winH, full)
			// dirty-rect fast path (sidebar hover region)
			paintSettingsCached(dc, 0, winW, winH, rect{dp(10), dp(56), railW - dp(10), dp(110)})
		}

		runtime.GC()
		after := gdiCount()
		var memAfter runtime.MemStats
		runtime.ReadMemStats(&memAfter)

		gdiDelta := int64(after) - int64(before)
		heapDelta := int64(memAfter.HeapAlloc) - int64(memBefore.HeapAlloc)
		t.Logf("%s: GDI %d→%d (%+d), heap %d→%d (%+d)",
			tc.name, before, after, gdiDelta, memBefore.HeapAlloc, memAfter.HeapAlloc, heapDelta)
		if gdiDelta > 0 {
			t.Fatalf("%s: GDI object leak — %d → %d across 30 paint cycles", tc.name, before, after)
		}
		const heapTolerance = 4 << 20 // 4 MB GC headroom
		if heapDelta > heapTolerance {
			t.Fatalf("%s: heap grew %+d bytes across 30 cycles (want ≤ %d)", tc.name, heapDelta, heapTolerance)
		}
	}
}
