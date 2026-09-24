package main

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// shell-level features not in the core engine:
// non-US layout bypass, per-app exclusion, auto-capitalization,
// clipboard injection, run-at-startup, configurable toggle key.

type config struct {
	SkipNonUSLayout bool
	AutoCap         bool
	UseClipboard    bool
	RunAtStartup    bool
	RunAsAdmin      bool
	ShowOnLaunch    bool // show settings dialog when launched at Windows startup
	SoundOnToggle   bool
	ShowHud         bool
	Charset         int  // 0=Unicode, 1=TCVN3
	ToggleKey       int  // legacy: migrated into HotkeyMods/HotkeyVk/UseCtrlShift
	HotkeyMods      int  // bit0 Ctrl, bit1 Shift, bit2 Alt, bit3 Win
	HotkeyVk        int  // virtual-key code of the toggle key (0 = modifiers only)
	UseCtrlShift    bool // extra Ctrl+Shift chord besides the custom hotkey
	FKeys           int  // bit0 F1=VN on, bit1 F2=VN off, bit2 F5=settings, bit3 F9=macro, bit4 F12=reset
	Theme           int  // 0=auto, 1=dark, 2=light
	HudX, HudY      int
}

var cfg config

// Per-app spec for exclude.txt:
//
//	app.exe            manual — excluded by default; the toggle key (Alt+Z /
//	                   Ctrl+Shift) turns Vietnamese on for THIS app only
//	app.exe|lock       hard-excluded: all keys pass through, toggle ignored
//	app.exe|clip       text is sent via clipboard paste in this app
//	app.exe|tcvn       output TCVN3 instead of Unicode in this app
//	app.exe|vni        use VNI in this app (telex/stelex/viqr/msvi too)
//	app.exe|clip,vni   flags combine with commas
const (
	appModeNone = iota
	appModeManual
	appModeLock
	appModeClip
)

type appSpec struct {
	mode int
	im   int  // -1 = use global input method
	tcvn bool // convert output to TCVN3
}

var gExcluded = map[string]appSpec{}
var gPerAppViet = map[string]bool{} // per-app Vietnamese state (manual mode)
var gForceClip bool                 // foreground app forces clipboard send
var gForceTCVN bool                 // foreground app wants TCVN3 output
var gLastIM = -1                    // input method currently set on the engine
var gAutoCapNext = false

// ---- layout / foreground app ----------------------------------------------

var (
	k32                     = windows.NewLazySystemDLL("kernel32.dll")
	pGetForegroundWindow    = user32.NewProc("GetForegroundWindow")
	pGetWindowThreadProcess = user32.NewProc("GetWindowThreadProcessId")
	pOpenProcess            = k32.NewProc("OpenProcess")
	pQueryFullProcImageName = k32.NewProc("QueryFullProcessImageNameW")
	pCloseHandle            = k32.NewProc("CloseHandle")
)

func foregroundLayoutIsUS() bool {
	hwnd, _, _ := pGetForegroundWindow.Call()
	if hwnd == 0 {
		return true
	}
	tid, _, _ := pGetWindowThreadProcess.Call(hwnd, 0)
	layout, _, _ := pGetKeyboardLayout.Call(tid)
	return uint32(layout)&0xFFFF == 0x0409 // LANG_ENGLISH/US
}

var lastFgHwnd uintptr
var lastFgSpec appSpec
var lastFgExe string

// foregroundApp returns the per-app spec and exe name of the active window.
func foregroundApp() (appSpec, string) {
	hwnd, _, _ := pGetForegroundWindow.Call()
	if hwnd == lastFgHwnd {
		return lastFgSpec, lastFgExe
	}
	spec, name := appSpec{im: -1}, ""
	var pid uint32
	pGetWindowThreadProcess.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if pid != 0 {
		// PROCESS_QUERY_LIMITED_INFORMATION
		h, _, _ := pOpenProcess.Call(0x1000, 0, uintptr(pid))
		if h != 0 {
			var buf [windows.MAX_PATH]uint16
			n := uint32(windows.MAX_PATH)
			r, _, _ := pQueryFullProcImageName.Call(h, 0,
				uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&n)))
			pCloseHandle.Call(h)
			if r != 0 {
				name = strings.ToLower(filepath.Base(windows.UTF16ToString(buf[:n])))
				if s, ok := gExcluded[name]; ok {
					spec = s
				}
			}
		}
	}
	lastFgHwnd, lastFgSpec, lastFgExe = hwnd, spec, name
	return spec, name
}

// modsMatch reports whether the currently held modifiers equal bitmask m
// (bit0 Ctrl, bit1 Shift, bit2 Alt, bit3 Win).
func modsMatch(m int) bool {
	lw, _, _ := pGetAsyncKeyState.Call(VK_LWIN)
	rw, _, _ := pGetAsyncKeyState.Call(VK_RWIN)
	win := (lw|rw)&0x8000 != 0
	return keyDown(VK_CONTROL) == (m&1 != 0) &&
		keyDown(VK_SHIFT) == (m&2 != 0) &&
		keyDown(VK_MENU) == (m&4 != 0) &&
		win == (m&8 != 0)
}

// effectiveViet is the Vietnamese state in effect for the foreground window.
func effectiveViet() bool {
	spec, exe := foregroundApp()
	if spec.mode == appModeManual {
		return gPerAppViet[exe]
	}
	return gVietKey
}

func loadExclusions(dir string) {
	f, err := os.Open(filepath.Join(dir, "exclude.txt"))
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.ToLower(strings.TrimSpace(sc.Text()))
		if line == "" || strings.HasPrefix(line, ";") {
			continue
		}
		name, spec := line, appSpec{mode: appModeManual, im: -1}
		if i := strings.IndexByte(line, '|'); i >= 0 {
			name = strings.TrimSpace(line[:i])
			for _, flag := range strings.Split(line[i+1:], ",") {
				switch strings.TrimSpace(flag) {
				case "lock", "off":
					spec.mode = appModeLock
				case "clip":
					spec.mode = appModeClip
				case "tcvn":
					spec.tcvn = true
				case "telex":
					spec.im = ImTelex
				case "stelex":
					spec.im = ImTelexSimple
				case "vni":
					spec.im = ImVni
				case "viqr":
					spec.im = ImViqr
				case "msvi":
					spec.im = ImMsVi
				case "tvni", "telexvni":
					spec.im = ImTelexVni
				}
			}
		}
		if name != "" {
			gExcluded[name] = spec
		}
	}
	_ = sc.Err()
}

// ---- auto-capitalization ---------------------------------------------------
// state lives in hook.go: gAutoCapNext

// ---- clipboard injection ----------------------------------------------------

const cfUnicodeText = 13

// lpBytes reinterprets a Win32 address uintptr (from GlobalLock etc.) as a
// *byte. The value is a real OS-owned pointer, valid for the call duration —
// written this way to avoid vet's unsafeptr false positive.
func lpBytes(v uintptr) *byte {
	return (*byte)(*(*unsafe.Pointer)(unsafe.Pointer(&v)))
}

var (
	pOpenClipboard    = user32.NewProc("OpenClipboard")
	pEmptyClipboard   = user32.NewProc("EmptyClipboard")
	pSetClipboardData = user32.NewProc("SetClipboardData")
	pGetClipboardData = user32.NewProc("GetClipboardData")
	pCloseClipboard   = user32.NewProc("CloseClipboard")
	pGlobalAlloc      = k32.NewProc("GlobalAlloc")
	pGlobalLock       = k32.NewProc("GlobalLock")
	pGlobalUnlock     = k32.NewProc("GlobalUnlock")
	pGlobalSize       = k32.NewProc("GlobalSize")
	pIsCBFormat       = user32.NewProc("IsClipboardFormatAvailable")
)

// injectViaClipboard sends backspaces via SendInput and the text via
// clipboard paste — fallback for apps that ignore injected keys.
func injectViaClipboard(res Result) {
	var saved windows.Handle
	var savedSz int
	pOpenClipboard.Call(gHwnd)
	if r, _, _ := pIsCBFormat.Call(cfUnicodeText); r != 0 {
		if h, _, _ := pGetClipboardData.Call(cfUnicodeText); h != 0 {
			if p, _, _ := pGlobalLock.Call(h); p != 0 {
				sz, _, _ := pGlobalSize.Call(h)
				savedSz = int(sz)
				cp, _, _ := pGlobalAlloc.Call(0x0042, uintptr(savedSz)) // GMEM_MOVEABLE|DDESHARE
				if cp != 0 {
					if dst, _, _ := pGlobalLock.Call(cp); dst != 0 {
						copy(unsafe.Slice(lpBytes(dst), savedSz),
							unsafe.Slice(lpBytes(p), savedSz))
						pGlobalUnlock.Call(cp)
						saved = windows.Handle(cp)
					}
				}
				pGlobalUnlock.Call(h)
			}
		}
	}
	// set new text
	text, _ := windows.UTF16FromString(utf16ToString(res.Out))
	h, _, _ := pGlobalAlloc.Call(0x0042, uintptr(len(text)*2))
	if h != 0 {
		if p, _, _ := pGlobalLock.Call(h); p != 0 {
			copy(unsafe.Slice(lpBytes(p), len(text)*2),
				unsafe.Slice((*byte)(unsafe.Pointer(&text[0])), len(text)*2))
			pGlobalUnlock.Call(h)
			pEmptyClipboard.Call()
			pSetClipboardData.Call(cfUnicodeText, h)
		}
	}
	pCloseClipboard.Call()

	// send backspaces + Ctrl+V
	inputs := make([]input, 0, res.Backs*2+4)
	for i := 0; i < res.Backs; i++ {
		inputs = append(inputs, keyEvent(VK_BACK, 0, 0), keyEvent(VK_BACK, 0, KEYEVENTF_KEYUP))
	}
	inputs = append(inputs,
		keyEvent(VK_CONTROL, 0, 0), keyEvent('V', 0, 0),
		keyEvent('V', 0, KEYEVENTF_KEYUP), keyEvent(VK_CONTROL, 0, KEYEVENTF_KEYUP))
	gSending = true
	sendInput(inputs)
	gSending = false

	// restore clipboard after the target app processed the paste
	go func(saved windows.Handle, sz int) {
		defer func() { recoverCrash("clipboard") }()
		time.Sleep(60 * time.Millisecond)
		pOpenClipboard.Call(gHwnd)
		pEmptyClipboard.Call()
		if saved != 0 {
			pSetClipboardData.Call(cfUnicodeText, uintptr(saved))
		}
		pCloseClipboard.Call()
	}(saved, savedSz)
}

func utf16ToString(u []uint16) string {
	return windows.UTF16ToString(u)
}

func keyEvent(vk, scan uint16, flags uint32) input {
	var in input
	in.Type = INPUT_KEYBOARD
	in.Ki.wVk = vk
	in.Ki.wScan = scan
	in.Ki.dwFlags = flags
	return in
}

// ---- run at startup ----------------------------------------------------------

const runKey = `Software\Microsoft\Windows\CurrentVersion\Run`
const taskName = "TuiGo (Admin)"
const legacyTaskName = "BoGoTiengViet (Admin)"

var pIsUserAnAdmin = sh32.NewProc("IsUserAnAdmin")

func isElevated() bool {
	r, _, _ := pIsUserAnAdmin.Call()
	return r != 0
}

// applyStartup manages both startup mechanisms:
//   - Run key: normal launch at logon (no admin)
//   - Scheduled task "highest": elevated launch at logon (needs admin to create)
func applyStartup() {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.SET_VALUE)
	if err == nil {
		defer k.Close()
		if cfg.RunAtStartup && !cfg.RunAsAdmin {
			exe, _ := os.Executable()
			k.SetStringValue("TuiGo", `"`+exe+`" -startup`)
		} else {
			k.DeleteValue("TuiGo")
			k.DeleteValue("BoGoTiengViet") // cleanup legacy
		}
	}
	if cfg.RunAtStartup && cfg.RunAsAdmin {
		registerAdminTask()
	} else if adminTaskExists() {
		deleteAdminTask()
	}
}

// schtasks runs schtasks.exe hidden; uses "runas" (UAC) only when needed.
// SEE_MASK_NO_CONSOLE keeps the elevated console app from flashing a
// black cmd window — plain ShellExecuteW can't suppress it.
func schtasks(args string) {
	exe := filepath.Join(os.Getenv("SystemRoot"), `System32\schtasks.exe`)
	if isElevated() {
		cmd := exec.Command(exe, strings.Fields(args)...)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		cmd.Run()
		return
	}
	runElevatedHidden(exe, args)
}

type shellExecInfo struct {
	cbSize       uint32
	fMask        uint32
	hwnd         uintptr
	lpVerb       *uint16
	lpFile       *uint16
	lpParameters *uint16
	lpDirectory  *uint16
	nShow        int32
	hInstApp     uintptr
	lpIDList     uintptr
	lpClass      *uint16
	hkeyClass    uintptr
	dwHotKey     uint32
	hMonitor     uintptr // union with hIcon
	hProcess     uintptr
}

const (
	seeMaskNoCloseProcess = 0x00000040
	seeMaskNoConsole      = 0x00008000
)

var pShellExecuteEx = sh32.NewProc("ShellExecuteExW")

// runElevatedHidden launches an elevated console tool without showing
// its console window (UAC prompt still appears — that's required).
func runElevatedHidden(exe, args string) {
	var si shellExecInfo
	si.cbSize = uint32(unsafe.Sizeof(si))
	si.fMask = seeMaskNoCloseProcess | seeMaskNoConsole
	si.lpVerb = utf16ptr("runas")
	si.lpFile = utf16ptr(exe)
	si.lpParameters = utf16ptr(args)
	si.nShow = 0 // SW_HIDE
	pShellExecuteEx.Call(uintptr(unsafe.Pointer(&si)))
	if si.hProcess != 0 {
		pCloseHandle.Call(si.hProcess)
	}
}

func adminTaskExists() bool {
	exe := filepath.Join(os.Getenv("SystemRoot"), `System32\schtasks.exe`)
	cmd := exec.Command(exe, "/Query", "/TN", taskName)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Run() == nil
}

// adminTaskPathOK reports whether the scheduled task still points at this
// executable. Moving the exe leaves a stale task that silently fails at
// logon — detectable without elevation (query is unprivileged).
func adminTaskPathOK() bool {
	exe := filepath.Join(os.Getenv("SystemRoot"), `System32\schtasks.exe`)
	cmd := exec.Command(exe, "/Query", "/TN", taskName, "/V", "/FO", "LIST")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	if err != nil {
		return true // can't verify — assume it's fine
	}
	me, err := os.Executable()
	if err != nil {
		return true
	}
	return strings.Contains(string(out), me)
}

func registerAdminTask() {
	exe, _ := os.Executable()
	tr := `"` + exe + `" -startup`
	schtasks(`/Create /F /TN "` + taskName + `" /SC ONLOGON /RL HIGHEST /TR "` + tr + `"`)
}

func deleteAdminTask() {
	schtasks(`/Delete /F /TN "` + taskName + `"`)
}

// relaunchElevated restarts the app with admin rights (UAC prompt).
func relaunchElevated() {
	exe, _ := os.Executable()
	pShellExecute.Call(0, uintptr(unsafe.Pointer(utf16ptr("runas"))),
		uintptr(unsafe.Pointer(utf16ptr(exe))), 0, 0, SW_SHOW)
	pPostQuitMessage.Call(0)
}

func isStartupLaunch() bool {
	for _, a := range os.Args[1:] {
		if a == "-startup" || a == "--startup" {
			return true
		}
	}
	return false
}

func loadStartup() {
	taskExists := adminTaskExists()
	taskOK := taskExists && adminTaskPathOK()
	cfg.RunAtStartup = taskExists
	if k, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.ALL_ACCESS); err == nil {
		defer k.Close()
		v, _, err := k.GetStringValue("TuiGo")
		if err != nil {
			v, _, err = k.GetStringValue("BoGoTiengViet") // legacy key
		}
		if err == nil {
			cfg.RunAtStartup = true
			// self-heal: exe was moved since the key was written — point it at
			// the current location or startup silently dies at next boot
			if exe, _ := os.Executable(); !strings.Contains(v, exe) {
				k.SetStringValue("TuiGo", `"`+exe+`" -startup`)
			}
			// admin task healthy again → the fallback Run key is redundant
			// (double-launch would just hit the single-instance mutex)
			if taskOK && cfg.RunAsAdmin {
				k.DeleteValue("TuiGo")
			}
		} else if taskExists && !taskOK && !isElevated() {
			// The admin task points at a moved/deleted exe and fixing it
			// needs elevation. Fall back to the Run key at THIS exe's path
			// so the app still starts next boot; an elevated run repairs
			// the task and applyStartup removes this key again.
			if exe, err := os.Executable(); err == nil {
				k.SetStringValue("TuiGo", `"`+exe+`" -startup`)
				cfg.RunAtStartup = true
			}
		}
	}
}
