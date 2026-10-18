package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var gHwnd uintptr
var gMsgTaskbarCreated uint32 // broadcast when Explorer (re)creates the taskbar

// appVersion is injected at release time via -ldflags "-X main.appVersion=…";
// local builds keep this default.
var appVersion = "0.0.9"

func wndProc(hwnd uintptr, msg uint32, wp, lp uintptr) uintptr {
	defer func() { _ = recover() }()
	switch msg {
	case WM_MOUSE_DOWN:
		flushPendingMouseClick()
		return 0
	case WM_TOGGLE_VN:
		src := "hotkey"
		switch wp {
		case 1:
			src = "ctrl+shift"
		case 2:
			src = "f1"
		case 3:
			src = "f2"
		}
		toggleEffects(src)
		return 0
	case WM_TRAYICON:
		switch uint32(lp) {
		case WM_RBUTTONUP, WM_CONTEXTMENU:
			gTrayDblclkArmed = false
			showTrayMenu()
		case WM_LBUTTONUP:
			if gTrayDblclkArmed {
				// Second click of a double-click — settings already opened.
				gTrayDblclkArmed = false
				pKillTimer.Call(gHwnd, 4)
				break
			}
			// Defer the toggle past the double-click window: a following
			// DBLCLK opens settings instead — firing now would toggle twice.
			gTrayToggleApp = currentInputApp()
			t, _, _ := pGetDoubleClickTime.Call()
			pSetTimer.Call(gHwnd, 4, t, 0)
		case WM_LBUTTONDBLCLK:
			gTrayDblclkArmed = true
			pKillTimer.Call(gHwnd, 4)
			openSettings()
		}
		return 0
	case WM_COMMAND:
		switch wp & 0xFFFF {
		case idmToggle:
			toggleVietKeyForApp("tray-menu", gTrayMenuApp)
		case idmTelex:
			setInputMethod(ImTelex)
		case idmTelexSimple:
			setInputMethod(ImTelexSimple)
		case idmVni:
			setInputMethod(ImVni)
		case idmViqr:
			setInputMethod(ImViqr)
		case idmSpell, idmModern, idmMacro, idmRestore:
			toggleOption(wp & 0xFFFF)
		case idmSettings:
			openSettings()
		case idmExit:
			pDestroyWindow.Call(hwnd)
		}
		return 0
	case WM_TIMER:
		if wp == 4 { // deferred tray single-click — no double-click followed
			pKillTimer.Call(gHwnd, 4)
			toggleVietKeyForApp("tray-icon", gTrayToggleApp)
			return 0
		}
		if wp == 2 { // HUD flash timeout
			if !cfg.ShowHud && gHudHwnd != 0 && hudShown {
				pShowWindow.Call(gHudHwnd, 0) // SW_HIDE
				hudShown = false
			}
			return 0
		}
		// Resync tracked modifier state every 500ms (timer 3) as well as timer 1:
		// a keyup can be missed (secure desktop, UAC prompt, hook gap) leaving
		// stale held flags that corrupt the Ctrl+Shift chord tracking.
		if gCtrlHeld && !keyDown(VK_LCONTROL) && !keyDown(VK_RCONTROL) {
			gCtrlHeld = false
		}
		if gShiftHeld && !keyDown(VK_LSHIFT) && !keyDown(VK_RSHIFT) {
			gShiftHeld = false
		}
		if gAltHeld && !keyDown(VK_LMENU) && !keyDown(VK_RMENU) {
			gAltHeld = false
		}
		if gWinHeld && !keyDown(VK_LWIN) && !keyDown(VK_RWIN) {
			gWinHeld = false
		}
		if gHotkeyDown && cfg.HotkeyVk != 0 && !keyDown(uint32(cfg.HotkeyVk)) {
			gHotkeyDown = false
		}
		if wp == 3 {
			syncTrayState()
			return 0
		}
		hookWatchdog()
		if !gTrayAdded {
			trayAdd() // Explorer wasn't ready at boot — retry
		}
		return 0
	case WM_WTSSESSION_CHANGE:
		// Lock/unlock crosses to the secure desktop: Windows may silently
		// drop our LL hooks, and GetLastInputInfo keeps moving on the lock
		// screen — reinstall on unlock instead of trusting the watchdog.
		if wp == WTS_SESSION_UNLOCK {
			reinstallHooks()
		}
		return 0
	case WM_DESTROY:
		saveSettingsSync()
		pPostQuitMessage.Call(0)
		return 0
	}
	if msg == gMsgTaskbarCreated && gMsgTaskbarCreated != 0 {
		gTrayAdded = false
		trayAdd() // Explorer (re)started — re-add our icon
		return 0
	}
	r, _, _ := pDefWindowProc.Call(hwnd, uintptr(msg), wp, lp)
	return r
}

func exeDir() string {
	var buf [windows.MAX_PATH]uint16
	n, _ := windows.GetModuleFileName(0, &buf[0], windows.MAX_PATH)
	return filepath.Dir(windows.UTF16ToString(buf[:n]))
}

func main() {
	runtime.LockOSThread() // hook + message loop must live on one thread
	defer func() { _ = recover() }()

	mtx, err := windows.CreateMutex(nil, true, utf16ptr("BoGoTiengViet.SingleInstance"))
	if err == nil && windows.GetLastError() == windows.ERROR_ALREADY_EXISTS {
		return
	}
	if err != nil {
		// Access denied = another instance owns the mutex at a different
		// integrity level (elevated vs normal). Running anyway would stack
		// two keyboard hooks and double-process every key.
		return
	}
	_ = mtx

	eng, _ := newEngine()
	gEngine = eng
	gEngine.Setup()
	gEngine.SetInputMethod(ImTelex)

	dir := exeDir()
	gEngine.LoadMacro(filepath.Join(dir, "macro.txt"))
	gEngine.LoadKeymap(filepath.Join(dir, "keymap.txt"))
	loadExclusions(dir)
	loadStartup()
	settingsExisted := loadSettings()
	gEngine.SetVietKey(gVietKey)

	// hidden message-only window for tray callbacks
	className := utf16ptr("BoGoTiengVietWnd")
	wc := wndClassEx{
		cbSize:        uint32(unsafe.Sizeof(wndClassEx{})),
		lpfnWndProc:   windows.NewCallback(wndProc),
		lpszClassName: className,
	}
	pRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc)))
	gHwnd, _, _ = pCreateWindowEx.Call(0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(utf16ptr("Tui Gõ"))),
		0, 0, 0, 0, 0, HWND_MESSAGE, 0, 0, 0)

	applyTheme()
	refreshTrayIcons()
	tm, _, _ := pRegisterWindowMsg.Call(uintptr(unsafe.Pointer(utf16ptr("TaskbarCreated"))))
	gMsgTaskbarCreated = uint32(tm)
	trayAdd()
	trayUpdate()
	hudInit()
	// Session lock/unlock notifications: needed to reinstall LL hooks that
	// Windows drops across the secure-desktop switch. Register on both —
	// the message-only hwnd is a fallback if HUD creation failed.
	pWTSRegister.Call(gHwnd, 0) // NOTIFY_FOR_THIS_SESSION
	pWTSRegister.Call(gHudHwnd, 0)
	// Elevated + startup: this instance can register the scheduled task
	// without another UAC prompt (user just granted elevation).
	if isElevated() && cfg.RunAsAdmin && cfg.RunAtStartup {
		registerAdminTask()
	}
	// Show the panel on first run (onboarding), or at Windows startup when the
	// user opted in. A manual relaunch opens quietly in the tray — double-click
	// the tray icon for settings.
	if !settingsExisted || (isStartupLaunch() && cfg.ShowOnLaunch) {
		openSettings()
	} else if !isStartupLaunch() {
		hudFlash() // manual relaunch: flash HUD so the user sees it started
	}

	if !installHook() {
		pMessageBox.Call(0, uintptr(unsafe.Pointer(utf16ptr("Không cài được keyboard hook."))),
			uintptr(unsafe.Pointer(utf16ptr("Tui Gõ"))), 0x10)
		os.Exit(1)
	}
	pSetTimer.Call(gHwnd, 1, 30000, 0)
	pSetTimer.Call(gHwnd, 3, 500, 0) // tray follows the foreground app and theme

	// Optional companion updater: if tuigo-updater.exe sits next to us, hand
	// it off (it self-throttles). Deleting that file = fully offline again.
	// Spawn on a goroutine after a short delay: CreateProcess blocks 10-15s
	// on machines whose AV/SmartScreen cloud-scans a fresh unsigned exe, and
	// on this thread that stall freezes the LL hooks = dead keyboard.
	os.WriteFile(filepath.Join(dir, "version.txt"), []byte(appVersion), 0644)
	os.Remove(filepath.Join(dir, "tuigo.old.exe"))
	os.Remove(filepath.Join(dir, "tuigo-updater.old.exe"))
	go func() {
		time.Sleep(5 * time.Second)
		up := filepath.Join(dir, "tuigo-updater.exe")
		if _, err := os.Stat(up); err != nil {
			return
		}
		if c := exec.Command(up); c.Start() == nil {
			c.Process.Release()
		}
	}()

	var m msg
	var r uintptr
	for {
		r, _, _ = pGetMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		pDispatchMessage.Call(uintptr(unsafe.Pointer(&m)))
	}

	pUnhookWindowsHook.Call(gHook)
	if tid := atomic.LoadUint32(&gMouseThreadID); tid != 0 {
		pPostThreadMessage.Call(uintptr(tid), WM_UNHOOK_MOUSE, 0, 0)
	}
	trayDelete()

}

var (
	hookCb, mouseCb uintptr
	gMouseThreadID  uint32
	mouseHookOnce   sync.Once
)

func runMouseHookLoop(ready chan<- struct{}) {
	runtime.LockOSThread()
	defer func() { _ = recover() }()
	tid, _, _ := pGetCurrentThreadId.Call()
	atomic.StoreUint32(&gMouseThreadID, uint32(tid))
	if mouseCb == 0 {
		mouseCb = windows.NewCallback(lowLevelMouseProc)
	}
	mh, _, _ := pSetWindowsHookEx.Call(WH_MOUSE_LL, mouseCb, 0, 0)
	atomic.StoreUintptr(&gMouseHook, mh)
	close(ready)

	var m msg
	for {
		r, _, _ := pGetMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		switch m.message {
		case WM_REINSTALL_MOUSE_HOOK:
			if h := atomic.SwapUintptr(&gMouseHook, 0); h != 0 {
				pUnhookWindowsHook.Call(h)
			}
			nh, _, _ := pSetWindowsHookEx.Call(WH_MOUSE_LL, mouseCb, 0, 0)
			atomic.StoreUintptr(&gMouseHook, nh)
		case WM_UNHOOK_MOUSE:
			if h := atomic.SwapUintptr(&gMouseHook, 0); h != 0 {
				pUnhookWindowsHook.Call(h)
			}
			return
		}
	}
}

func installHook() bool {
	if hookCb == 0 {
		hookCb = windows.NewCallback(lowLevelKbProc)
	}
	gHook, _, _ = pSetWindowsHookEx.Call(WH_KEYBOARD_LL, hookCb, 0, 0)
	started := false
	mouseHookOnce.Do(func() {
		started = true
		ready := make(chan struct{})
		go runMouseHookLoop(ready)
		<-ready
	})
	if !started {
		if tid := atomic.LoadUint32(&gMouseThreadID); tid != 0 {
			pPostThreadMessage.Call(uintptr(tid), WM_REINSTALL_MOUSE_HOOK, 0, 0)
		}
	}
	return gHook != 0
}

// reinstallHooks force-reinstalls both hooks. Handles are zeroed first —
// Windows can remove a hook silently, leaving a stale non-zero handle that
// would otherwise make the reinstall a no-op.
func reinstallHooks() {
	if gHook != 0 {
		pUnhookWindowsHook.Call(gHook)
		gHook = 0
	}
	installHook()
}

var (
	gPrevInputTick uint32
	gPrevHookSeen  uint64
	gPrevMouseSeen uint64
)

type lastInputInfo struct {
	cbSize uint32
	dwTime uint32
}

var pGetLastInputInfo = user32.NewProc("GetLastInputInfo")

// hookWatchdog detects a silently-removed LL hook WITHOUT injecting input:
// GetLastInputInfo reports the last real input tick system-wide — if it
// advanced but neither hook counter moved, our hooks are dead → reinstall.
// (The old synthetic key-up ping reset the idle timer and blocked display
// sleep — injection must not happen on a timer.)
func hookWatchdog() {
	var lii lastInputInfo
	lii.cbSize = uint32(unsafe.Sizeof(lii))
	if r, _, _ := pGetLastInputInfo.Call(uintptr(unsafe.Pointer(&lii))); r == 0 {
		return
	}
	mouseSeen := atomic.LoadUint64(&gMouseSeen)
	inputMoved := lii.dwTime != gPrevInputTick
	hooksMoved := gHookSeen != gPrevHookSeen || mouseSeen != gPrevMouseSeen
	if inputMoved && !hooksMoved {
		reinstallHooks()
	}
	gPrevInputTick = lii.dwTime
	gPrevHookSeen = gHookSeen
	gPrevMouseSeen = mouseSeen
}
