package main

import (
	"unsafe"
)

var (
	gHook         uintptr
	gSending      bool // reentrancy guard for our own SendInput
	gChordClean   = true
	gChordPending bool // Ctrl+Shift is complete — toggle fires on release
	gCtrlHeld     bool
	gShiftHeld    bool
	gEngine       Engine
	gVietKey      = true
)

func keyDown(vk uint32) bool {
	r, _, _ := pGetAsyncKeyState.Call(uintptr(vk))
	return r&0x8000 != 0
}

func injectOutput(res Result) {
	if (cfg.Charset == 1 || gForceTCVN) && !res.KeyOut {
		gEngine.ConvertTCVN(res.Out)
	}
	if (cfg.UseClipboard || gForceClip) && !res.KeyOut && len(res.Out) > 0 {
		injectViaClipboard(res)
		return
	}
	inputs := make([]input, 0, res.Backs*2+len(res.Out)*2)
	for i := 0; i < res.Backs; i++ {
		inputs = append(inputs, keyEvent(VK_BACK, 0, 0), keyEvent(VK_BACK, 0, KEYEVENTF_KEYUP))
	}
	for _, ch := range res.Out {
		inputs = append(inputs,
			keyEvent(0, ch, KEYEVENTF_UNICODE),
			keyEvent(0, ch, KEYEVENTF_UNICODE|KEYEVENTF_KEYUP))
	}
	if len(inputs) > 0 {
		gSending = true
		sendInput(inputs)
		gSending = false
	}
}

// vkToChar translates a virtual key to the ASCII char the engine expects.
func vkToChar(vk, scan uint32) (byte, bool) {
	var kb [256]byte
	if keyDown(VK_SHIFT) {
		kb[VK_SHIFT] = 0x80
	}
	if r, _, _ := pGetKeyState.Call(VK_CAPITAL); r&1 != 0 {
		kb[VK_CAPITAL] = 1
	}
	var buf [8]uint16
	layout, _, _ := pGetKeyboardLayout.Call(0)
	r, _, _ := pToUnicodeEx.Call(uintptr(vk), uintptr(scan),
		uintptr(unsafe.Pointer(&kb[0])), uintptr(unsafe.Pointer(&buf[0])),
		8, 0, layout)
	if int32(r) == -1 { // dead key: flush layout state
		pToUnicodeEx.Call(uintptr(vk), uintptr(scan),
			uintptr(unsafe.Pointer(&kb[0])), uintptr(unsafe.Pointer(&buf[0])),
			8, 0, layout)
		return 0, false
	}
	if int32(r) <= 0 {
		return 0, false
	}
	c := buf[0]
	if c < 0x20 || c > 0x7E {
		return 0, false // engine only speaks ASCII
	}
	return byte(c), true
}

func isResetKey(vk uint32) bool {
	switch vk {
	case VK_RETURN, VK_TAB, VK_ESCAPE,
		VK_LEFT, VK_RIGHT, VK_UP, VK_DOWN,
		VK_HOME, VK_END, VK_PRIOR, VK_NEXT,
		VK_DELETE, VK_INSERT,
		VK_LWIN, VK_RWIN, VK_APPS,
		VK_NUMLOCK, VK_SCROLL, VK_PAUSE, VK_SNAPSHOT, VK_CAPITAL:
		return true
	}
	return vk >= VK_F1 && vk <= VK_F24
}

func isModifierKey(vk uint32) bool {
	switch vk {
	case VK_SHIFT, VK_LSHIFT, VK_RSHIFT,
		VK_CONTROL, VK_LCONTROL, VK_RCONTROL,
		VK_MENU, VK_LMENU, VK_RMENU:
		return true
	}
	return false
}

// handleKeyDown returns true if the key must be swallowed.
func handleKeyDown(p *kbdLLHookStruct) bool {
	vk := p.vkCode
	spec, exe := foregroundApp()
	gForceClip = spec.mode == appModeClip
	gForceTCVN = spec.tcvn

	// per-app input method override (only touch engine when it changes)
	wantIM := gIM
	if spec.im >= 0 {
		wantIM = spec.im
	}
	if wantIM != gLastIM {
		gEngine.SetInputMethod(wantIM)
		gLastIM = wantIM
	}

	// hard-excluded app: everything passes through, toggle ignored
	if spec.mode == appModeLock {
		gEngine.Reset()
		gAutoCapNext = false
		return false
	}

	// toggle hotkey: configured modifiers + key
	if cfg.HotkeyVk != 0 && vk == uint32(cfg.HotkeyVk) && modsMatch(cfg.HotkeyMods) {
		toggleVietKey()
		return true
	}
	if vk == VK_LSHIFT || vk == VK_RSHIFT || vk == VK_LCONTROL || vk == VK_RCONTROL {
		// Track held state ourselves instead of GetAsyncKeyState — inside an
		// LL hook the async state can lag the event stream just enough to
		// drop a fast chord tap.
		wasChord := gCtrlHeld && gShiftHeld
		switch vk {
		case VK_LCONTROL, VK_RCONTROL:
			gCtrlHeld = true
		case VK_LSHIFT, VK_RSHIFT:
			gShiftHeld = true
		}
		// Modifier-only chord (Ctrl+Shift): ARM it here but fire on release —
		// toggling on press steals real combos like Ctrl+Shift+S (Firefox
		// screenshot). Any non-modifier key while held marks it dirty.
		modChord := cfg.UseCtrlShift || (cfg.HotkeyVk == 0 && cfg.HotkeyMods == 3)
		if modChord && !wasChord && gCtrlHeld && gShiftHeld {
			gChordPending = true
			gChordClean = true
		}
		return false // never swallow modifiers
	}

	// quick-action F keys (raw, no modifiers held)
	if cfg.FKeys != 0 && vk >= VK_F1 && vk <= VK_F24 &&
		!keyDown(VK_CONTROL) && !keyDown(VK_MENU) && !keyDown(VK_SHIFT) {
		switch vk {
		case VK_F1:
			if cfg.FKeys&1 != 0 {
				if !effectiveViet() {
					toggleVietKey()
				}
				return true
			}
		case VK_F2:
			if cfg.FKeys&2 != 0 {
				if effectiveViet() {
					toggleVietKey()
				}
				return true
			}
		case VK_F5:
			if cfg.FKeys&4 != 0 {
				openSettings()
				return true
			}
		case VK_F9:
			if cfg.FKeys&8 != 0 {
				toggleMacro()
				return true
			}
		case VK_F12:
			if cfg.FKeys&16 != 0 {
				gEngine.Reset()
				gAutoCapNext = false
				gPerAppViet = map[string]bool{}
				if cfg.SoundOnToggle {
					pMessageBeep.Call(0x00000040)
				}
				return true
			}
		}
	}
	if gChordPending && gCtrlHeld && gShiftHeld {
		gChordClean = false // e.g. Ctrl+Shift+Arrow text selection
	}

	// bypass entirely: manually-excluded app (until toggled on) or non-US layout
	viet := gVietKey
	if spec.mode == appModeManual {
		viet = gPerAppViet[exe]
	}
	gEngine.SetVietKey(viet)
	if cfg.SkipNonUSLayout && !foregroundLayoutIsUS() {
		gEngine.Reset()
		gAutoCapNext = false
		return false
	}

	if isResetKey(vk) {
		gEngine.Reset()
		gAutoCapNext = cfg.AutoCap && vk == VK_RETURN
		return false
	}
	if isModifierKey(vk) {
		return false
	}
	// shortcuts (Ctrl/Alt held) bypass the engine
	if keyDown(VK_CONTROL) || keyDown(VK_MENU) {
		return false
	}

	if vk == VK_BACK {
		res := gEngine.Backspace()
		if !res.Consumed {
			return false
		}
		injectOutput(res)
		return true
	}

	ch, ok := vkToChar(vk, p.scanCode)
	if !ok {
		return false
	}

	// auto-capitalize after ". ! ?" or Enter
	forceCap := cfg.AutoCap && gAutoCapNext && ch >= 'a' && ch <= 'z'
	if forceCap {
		ch -= 'a' - 'A'
	}

	gEngine.SetCapsState(keyDown(VK_SHIFT), capsLockOn())
	res := gEngine.Filter(uint32(ch))

	if cfg.AutoCap {
		switch {
		case ch == '.' || ch == '!' || ch == '?':
			gAutoCapNext = true
		case ch == ' ':
			// keep pending through whitespace
		default:
			gAutoCapNext = false
		}
	}

	if !res.Consumed {
		if forceCap {
			// engine passed it through; inject the uppercase letter ourselves
			injectOutput(Result{Out: []uint16{uint16(ch)}})
			return true
		}
		return false
	}
	injectOutput(res)
	return true
}

func capsLockOn() bool {
	r, _, _ := pGetKeyState.Call(VK_CAPITAL)
	return r&1 != 0
}

// Hook liveness: Windows silently removes a low-level hook whose callback
// exceeds LowLevelHooksTimeout — the app keeps running but keys stop being
// processed. gHookSeen bumps on every event (including our own ping below);
// the WM_TIMER watchdog in wndProc uses it to detect a dead hook.
var gHookSeen uint64

func lowLevelKbProc(nCode int, wParam, lParam uintptr) uintptr {
	// A panic escaping a windows.NewCallback can't unwind through Windows
	// frames — the process dies instantly and silently. Catch it here.
	defer func() { recoverCrash("hook") }()
	gHookSeen++
	if nCode == 0 && !gSending { // HC_ACTION
		p := (*kbdLLHookStruct)(*(*unsafe.Pointer)(unsafe.Pointer(&lParam)))
		if p.flags&(LLKHF_INJECTED|LLKHF_LOWER_IL_INJECTED) == 0 {
			switch wParam {
			case WM_KEYUP, WM_SYSKEYUP:
				switch p.vkCode {
				case VK_LCONTROL, VK_RCONTROL, VK_LSHIFT, VK_RSHIFT:
					// Chord ends when a modifier releases: toggle only if
					// the chord stayed clean (no other key while held).
					if gChordPending && gChordClean && gCtrlHeld && gShiftHeld {
						toggleVietKey()
					}
					gChordPending = false
					if p.vkCode == VK_LCONTROL || p.vkCode == VK_RCONTROL {
						gCtrlHeld = false
					} else {
						gShiftHeld = false
					}
				}
				if isModifierKey(p.vkCode) {
					gChordClean = true
				}
			case WM_KEYDOWN, WM_SYSKEYDOWN:
				if handleKeyDown(p) {
					return 1
				}
			}
		}
	}
	r, _, _ := pCallNextHookEx.Call(gHook, uintptr(nCode), wParam, lParam)
	return r
}
