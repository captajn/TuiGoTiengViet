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
//	app.exe|safedel    selection-safe delete (fields whose autocomplete
//	                   selects its suggestion; browsers get this auto)
//	app.exe|vni        use VNI in this app (telex/stelex/viqr/msvi too)
//	app.exe|clip,vni   flags combine with commas
const (
	appModeNone = iota
	appModeManual
	appModeLock
	appModeClip
)

type appSpec struct {
	mode    int
	im      int  // -1 = use global input method
	tcvn    bool // convert output to TCVN3
	safeDel bool // selection-safe delete (autocomplete fields)
}

var gExcluded = map[string]appSpec{}
var gPerAppViet = map[string]bool{} // per-app Vietnamese state (manual mode)
var gForceClip bool                 // foreground app forces clipboard send
var gForceTCVN bool                 // foreground app wants TCVN3 output
var gSafeDel bool                   // foreground app needs selection-safe delete

// gCaretAtEnd tracks whether the edit caret is believed to sit at the end of
// the text (or a live selection may exist). Only then is a forward-Delete
// harmless: mid-text it would erase the real character after the caret.
// Cleared by backward caret moves (Left/Up/Home/PgUp) and mouse clicks; set
// by End/Down/PgDn and focus changes.
var gCaretAtEnd = true
var gLastIM = -1 // input method currently set on the engine

// Apps whose text fields inline-autocomplete with a live selection (browser
// omniboxes, Explorer's address bar). A plain Backspace there deletes the
// SELECTED suggestion tail instead of the previous char — "hà" comes out
// "haà". Selection-safe delete is used for these automatically.
var gSafeDelExes = map[string]bool{
	"chrome.exe": true, "msedge.exe": true, "firefox.exe": true,
	"brave.exe": true, "opera.exe": true, "vivaldi.exe": true,
	"arc.exe": true, "whale.exe": true, "browser.exe": true, // browser.exe = Cốc Cốc
	"explorer.exe": true,
}
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

// foregroundApp returns the per-app spec, exe name and HWND of the active
// window.
func foregroundApp() (appSpec, string, uintptr) {
	hwnd, _, _ := pGetForegroundWindow.Call()
	if hwnd == lastFgHwnd {
		return lastFgSpec, lastFgExe, hwnd
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
	return spec, name, hwnd
}

// modsMatch reports whether the currently held modifiers equal bitmask m
// (bit0 Ctrl, bit1 Shift, bit2 Alt, bit3 Win). Reads the hook-tracked state:
// GetAsyncKeyState lags the LL event stream and also sees injected keys —
// AltGr and the phantom Ctrl Windows sends while Alt is held, which would
// otherwise randomly disqualify Alt-based hotkeys.
func modsMatch(m int) bool {
	ctrl := gCtrlHeld
	if !ctrl && gAltHeld {
		// A real Ctrl press while Alt is held is indistinguishable from the
		// phantom (injected, and never tracked) — except on the RIGHT side:
		// the phantom is always left Ctrl.
		ctrl = keyDown(VK_RCONTROL)
	}
	return ctrl == (m&1 != 0) &&
		gShiftHeld == (m&2 != 0) &&
		gAltHeld == (m&4 != 0) &&
		gWinHeld == (m&8 != 0)
}

// effectiveViet is the Vietnamese state in effect for the foreground window.
func effectiveViet() bool {
	spec, exe, _ := foregroundApp()
	if spec.mode == appModeLock {
		return false
	}
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
				case "safedel":
					spec.safeDel = true
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

func lpUint16(v uintptr) *uint16 {
	return (*uint16)(*(*unsafe.Pointer)(unsafe.Pointer(&v)))
}

func lpUintptr(v uintptr) *uintptr {
	return (*uintptr)(*(*unsafe.Pointer)(unsafe.Pointer(&v)))
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

// appendBackspaces emits N char deletions. delPairs selects the legacy
// Delete+Backspace pair used when a field may hold a live selection we did
// not get to inspect (autocomplete suggestion tail in omniboxes): the
// forward-Delete clears the selection first so the Backspace hits real
// text. With no selection at end-of-text the Delete is a no-op — but
// mid-text it erases the character after the caret, so pairs are only
// used when the caret is believed to be at the end and no probe ran.
func appendBackspaces(inputs []input, n int, delPairs bool) []input {
	if delPairs {
		for i := 0; i < n; i++ {
			inputs = append(inputs,
				keyEvent(VK_DELETE, 0, 0), keyEvent(VK_DELETE, 0, KEYEVENTF_KEYUP),
				keyEvent(VK_BACK, 0, 0), keyEvent(VK_BACK, 0, KEYEVENTF_KEYUP))
		}
		return inputs
	}
	for i := 0; i < n; i++ {
		inputs = append(inputs, keyEvent(VK_BACK, 0, 0), keyEvent(VK_BACK, 0, KEYEVENTF_KEYUP))
	}
	return inputs
}

// probeSelectionCut asks the focused field to cut its active selection and
// watches the clipboard sequence number. Returns the number of UTF-16
// units that were cut (0 when nothing was selected) and whether the probe
// succeeded at all — on failure the caller must fall back to the caret
// heuristic.
//
// This resolves the ambiguity keystrokes alone cannot: after typing 'a'
// in an omnibox the field may hold no selection, an inline suggestion
// tail, or may even have selected the 'a' itself (Edge). Only a real
// selection change tells them apart. Ctrl+X with no selection is a no-op,
// and the user's clipboard is preserved via an OleGetClipboard snapshot.
func probeSelectionCut() (cut int, ok bool) {
	var saved uintptr
	pOleInitialize.Call(0)
	if r, _, _ := pOleGetClipboard.Call(uintptr(unsafe.Pointer(&saved))); int32(r) < 0 || saved == 0 {
		return 0, false
	}
	seq0, _, _ := pClipboardSeq.Call()
	gSending = true
	sendInput([]input{
		keyEvent(VK_CONTROL, 0, 0), keyEvent('X', 0, 0),
		keyEvent('X', 0, KEYEVENTF_KEYUP), keyEvent(VK_CONTROL, 0, KEYEVENTF_KEYUP)})
	gSending = false

	// An app that honours Ctrl+X writes the clipboard within a few ms; a
	// field with no selection never writes at all, so keep the wait short —
	// this runs inside the keyboard hook and delays the key queue.
	deadline := time.Now().Add(30 * time.Millisecond)
	for {
		seq, _, _ := pClipboardSeq.Call()
		if seq != seq0 {
			cut = clipboardTextLen()
			pOleSetClipboard.Call(saved) // hand the user's clipboard back
			comRelease(saved)
			return cut, true
		}
		if time.Now().After(deadline) {
			// No clipboard write: no selection was active (or the field
			// refuses to cut) — nothing was clobbered.
			comRelease(saved)
			return 0, true
		}
		time.Sleep(time.Millisecond)
	}
}

// clipboardTextLen returns the length in UTF-16 units of the text
// currently on the clipboard (0 if none or it cannot be read).
func clipboardTextLen() int {
	n := 0
	if r, _, _ := pOpenClipboard.Call(gHwnd); r != 0 {
		if f, _, _ := pIsCBFormat.Call(cfUnicodeText); f != 0 {
			if h, _, _ := pGetClipboardData.Call(cfUnicodeText); h != 0 {
				if p, _, _ := pGlobalLock.Call(h); p != 0 {
					for n < 1<<20 && *lpUint16(p) != 0 {
						n++
						p += 2
					}
					pGlobalUnlock.Call(h)
				}
			}
		}
		pCloseClipboard.Call()
	}
	return n
}

// comRelease invokes IUnknown::Release (vtable slot 2) on a COM object.
func comRelease(obj uintptr) {
	vt := *lpUintptr(obj)
	release := *lpUintptr(vt + 2*unsafe.Sizeof(uintptr(0)))
	syscall.SyscallN(release, obj)
}

// injectViaClipboard sends backspaces via SendInput and the text via
// clipboard paste — fallback for apps that ignore injected keys.
func injectViaClipboard(res Result, delPairs bool) {
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
	inputs := make([]input, 0, res.Backs*4+4)
	inputs = appendBackspaces(inputs, res.Backs, delPairs)
	inputs = append(inputs,
		keyEvent(VK_CONTROL, 0, 0), keyEvent('V', 0, 0),
		keyEvent('V', 0, KEYEVENTF_KEYUP), keyEvent(VK_CONTROL, 0, KEYEVENTF_KEYUP))
	gSending = true
	sendInput(inputs)
	gSending = false

	// restore clipboard after the target app processed the paste
	go func(saved windows.Handle, sz int) {
		defer func() { recoverCrash("clipboard", recover()) }()
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
