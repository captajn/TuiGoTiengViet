package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var gHwnd uintptr
var gMsgTaskbarCreated uint32 // broadcast when Explorer (re)creates the taskbar

// appVersion is injected at release time via -ldflags "-X main.appVersion=…";
// local builds keep this default.
var appVersion = "0.0.1"

// recoverCrash must be called from a deferred func in every windows.NewCallback
// path — a panic escaping the callback kills the process silently.
func recoverCrash(where string) {
	r := recover()
	if r == nil {
		return
	}
	f, err := os.OpenFile(filepath.Join(exeDir(), "crash.log"),
		os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "=== %s panic in %s: %v\n%s\n",
		time.Now().Format("2006-01-02 15:04:05"), where, r, debug.Stack())
}

// logLine appends one timestamped line to crash.log (lifecycle diagnostics).
func logLine(s string) {
	f, err := os.OpenFile(filepath.Join(exeDir(), "crash.log"),
		os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s %s\n", time.Now().Format("2006-01-02 15:04:05"), s)
}

func wndProc(hwnd uintptr, msg uint32, wp, lp uintptr) uintptr {
	defer func() { recoverCrash("wndproc") }()
	switch msg {
	case WM_TRAYICON:
		switch uint32(lp) {
		case WM_RBUTTONUP, WM_CONTEXTMENU:
			showTrayMenu()
		case WM_LBUTTONUP, WM_LBUTTONDBLCLK:
			openSettings() // click opens the panel; V/E toggles via hotkey/menu
		}
		return 0
	case WM_COMMAND:
		switch wp & 0xFFFF {
		case idmToggle:
			toggleVietKey("tray-menu")
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
		if wp == 2 { // HUD flash timeout
			if !cfg.ShowHud && gHudHwnd != 0 {
				pShowWindow.Call(gHudHwnd, 0) // SW_HIDE
			}
			return 0
		}
		hookWatchdog()
		if !gTrayAdded {
			trayAdd() // Explorer wasn't ready at boot — retry
		}
		tickCount++
		if tickCount%10 == 0 {
			logLine("alive") // heartbeat — if the app dies, the gap shows when
		}
		return 0
	case WM_DESTROY:
		saveSettings()
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
	// Route stderr into crash.log: Go *fatal errors* (e.g. access violations
	// inside Win32 calls) bypass recover() and write to stderr — invisible
	// in a windowsgui build unless we give stderr a real handle.
	if f, err := os.OpenFile(filepath.Join(exeDir(), "crash.log"),
		os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err == nil {
		windows.SetStdHandle(windows.STD_ERROR_HANDLE, windows.Handle(f.Fd()))
	}
	defer func() { recoverCrash("main") }()
	logLine("main: entered")

	mtx, err := windows.CreateMutex(nil, true, utf16ptr("BoGoTiengViet.SingleInstance"))
	logLine(fmt.Sprintf("main: CreateMutex err=%v lastErr=%v", err, windows.GetLastError()))
	if err == nil && windows.GetLastError() == windows.ERROR_ALREADY_EXISTS {
		logLine("main: already exists -> exit")
		return
	}
	_ = mtx

	eng, _ := newEngine()
	gEngine = eng
	gEngine.Setup()
	gEngine.SetInputMethod(ImTelex)
	logLine("main: after Setup")

	dir := exeDir()
	gEngine.LoadMacro(filepath.Join(dir, "macro.txt"))
	gEngine.LoadKeymap(filepath.Join(dir, "keymap.txt"))
	loadExclusions(dir)
	loadStartup()
	settingsExisted := loadSettings()
	gEngine.SetVietKey(gVietKey)
	logLine("main: after loadSettings")

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
	logLine(fmt.Sprintf("main: gHwnd=%d", gHwnd))

	applyTheme()
	logLine("main: after applyTheme")
	gIconVn = makeTrayIcon(true)
	gIconEn = makeTrayIcon(false)
	tm, _, _ := pRegisterWindowMsg.Call(uintptr(unsafe.Pointer(utf16ptr("TaskbarCreated"))))
	gMsgTaskbarCreated = uint32(tm)
	trayAdd()
	trayUpdate()
	logLine("main: after trayAdd")
	hudInit()
	logLine("main: after hudInit")
	// Elevated + startup: this instance can register the scheduled task
	// without another UAC prompt (user just granted elevation).
	if isElevated() && cfg.RunAsAdmin && cfg.RunAtStartup {
		registerAdminTask()
	}
	// Show the panel on first run (onboarding), or at Windows startup when the
	// user opted in. A manual relaunch opens quietly in the tray — double-click
	// the tray icon for settings.
	if !settingsExisted || (isStartupLaunch() && cfg.ShowOnLaunch) {
		logLine("main: before openSettings")
		openSettings()
		logLine("main: after openSettings")
	} else if !isStartupLaunch() {
		hudFlash() // manual relaunch: flash HUD so the user sees it started
	}

	logLine("main: before installHook")
	if !installHook() {
		logLine("main: installHook failed!")
		pMessageBox.Call(0, uintptr(unsafe.Pointer(utf16ptr("Không cài được keyboard hook."))),
			uintptr(unsafe.Pointer(utf16ptr("Tui Gõ"))), 0x10)
		os.Exit(1)
	}
	logLine("main: after installHook")
	pSetTimer.Call(gHwnd, 1, 30000, 0)

	// Optional companion updater: if tuigo-updater.exe sits next to us, hand
	// it off (it self-throttles). Deleting that file = fully offline again.
	os.WriteFile(filepath.Join(dir, "version.txt"), []byte(appVersion), 0644)
	os.Remove(filepath.Join(dir, "tuigo.old.exe"))
	os.Remove(filepath.Join(dir, "tuigo-updater.old.exe"))
	if _, err := os.Stat(filepath.Join(dir, "tuigo-updater.exe")); err == nil {
		if c := exec.Command(filepath.Join(dir, "tuigo-updater.exe")); c.Start() == nil {
			c.Process.Release()
		}
	}

	var m msg
	var r uintptr
	for {
		r, _, _ = pGetMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			logLine(fmt.Sprintf("main: GetMessage <= 0: r=%d", int32(r)))
			break
		}
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		pDispatchMessage.Call(uintptr(unsafe.Pointer(&m)))
	}

	pUnhookWindowsHook.Call(gHook)
	pUnhookWindowsHook.Call(gMouseHook)
	trayDelete()
	// Distinguish a clean exit (WM_QUIT → r=0) from silent death: if the
	// process vanishes without this line in crash.log, something external
	// (AV/TerminateProcess) killed it.
	logLine(fmt.Sprintf("exit clean: GetMessage=%d msg=0x%X", int32(r), uint32(m.message)))
}

var hookCb, mouseCb uintptr

func installHook() bool {
	if hookCb == 0 {
		hookCb = windows.NewCallback(lowLevelKbProc)
	}
	if mouseCb == 0 {
		mouseCb = windows.NewCallback(lowLevelMouseProc)
	}
	gHook, _, _ = pSetWindowsHookEx.Call(WH_KEYBOARD_LL, hookCb, 0, 0)
	if gMouseHook == 0 {
		gMouseHook, _, _ = pSetWindowsHookEx.Call(WH_MOUSE_LL, mouseCb, 0, 0)
	}
	return gHook != 0
}

var (
	gPingSent   bool
	gSeenAtPing uint64
	tickCount   int
)

// hookWatchdog pings the hook with a harmless injected key-up (VK 0xE8,
// unassigned) every timer tick. Injected events still pass through LL hooks,
// so if the counter didn't move since last tick the hook is dead — reinstall.
// Runs on the hook-owning thread via WM_TIMER.
func hookWatchdog() {
	if gPingSent && gHookSeen == gSeenAtPing && gHook != 0 {
		pUnhookWindowsHook.Call(gHook)
		installHook()
		logLine("hook dead: reinstalled")
	}
	gSeenAtPing = gHookSeen
	gSending = true
	sendInput([]input{keyEvent(0xE8, 0, KEYEVENTF_KEYUP)})
	gSending = false
	gPingSent = true
}
