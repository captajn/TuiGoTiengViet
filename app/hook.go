package main

import (
	"fmt"
	"time"
	"unsafe"
)

var (
	gHook         uintptr
	gMouseHook    uintptr
	gSending      bool // reentrancy guard for our own SendInput
	gChordClean   = true
	gChordPending bool // Ctrl+Shift is complete — toggle fires on release
	gCtrlHeld     bool
	gShiftHeld    bool
	gAltHeld      bool
	gWinHeld      bool
	gHotkeyDown   bool      // hotkey vk is held — auto-repeat must not re-toggle
	gLastKeyHwnd  uintptr   // foreground window of the last key event
	gModDownAt    time.Time // when the first chord modifier went down
	gModUpAt      time.Time // when the first chord modifier came up
	gLastKeyAt    time.Time // last non-modifier keydown (chord purity check)
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
	// In selection-happy fields (browser omniboxes) what a "Backspace"
	// actually deletes depends on live state keystrokes can't reveal: an
	// inline suggestion tail may be selected, or the field may have
	// selected the just-typed char itself (Edge). Cut-probing resolves it.
	// Skipped while Shift is held so the injected chord stays a plain
	// Ctrl+X — Ctrl+Shift+X opens side panels in a few apps.
	delPairs := false
	var savedClip uintptr
	if gSafeDel && res.Backs > 0 && !res.KeyOut {
		if !gNoGuard {
			// VKey approach: suggestion-tail bait (U+202F) swallows any live
			// autocomplete tail cleanly without touching clipboard or adding
			// latency. delPairs is false because bait clears the selection.
			logLine(fmt.Sprintf("inject: backs=%d bait=0x202F caretEnd=%v",
				res.Backs, gCaretAtEnd))
		} else if !gShiftHeld {
			// Legacy cut-probe kept only for A/B testing (TUIGO_NO_GUARD set).
			delPairs = gCaretAtEnd
			cut, saved, ok := probeSelectionCut()
			if ok {
				delPairs = false // selection resolved: plain Backspaces suffice
				savedClip = saved
				if cut == res.Backs {
					res.Backs = 0 // the selection WAS the text to replace
				}
			}
			logLine(fmt.Sprintf("inject: backs=%d cut=%d ok=%v caretEnd=%v noGuard=true",
				res.Backs, cut, ok, gCaretAtEnd))
		}
	}
	if (cfg.UseClipboard || gForceClip) && !res.KeyOut && len(res.Out) > 0 {
		injectViaClipboard(res, delPairs)
		if savedClip != 0 {
			comRelease(savedClip)
		}
		return
	}
	inputs := make([]input, 0, res.Backs*4+len(res.Out)*2+6)
	// Suggestion-tail bait (VKey's approach): an omnibox holds its
	// autocomplete tail as a live selection that eats correction
	// backspaces ("hôm" → "hoôm") and can re-attach at ANY point —
	// between the probe and this batch, or between two batch events. A
	// typed char REPLACES the selection, so U+202F (narrow no-break
	// space — invisible, and matches no suggestion prefix) both eats
	// whatever tail exists and stops the field from suggesting again
	// while the backspaces run. One extra backspace deletes the bait.
	if gSafeDel && res.Backs > 0 && !res.KeyOut && !gNoGuard {
		inputs = append(inputs,
			keyEvent(0, 0x202F, KEYEVENTF_UNICODE),
			keyEvent(0, 0x202F, KEYEVENTF_UNICODE|KEYEVENTF_KEYUP))
		res.Backs++
	}
	inputs = appendBackspaces(inputs, res.Backs, delPairs)
	for _, ch := range res.Out {
		inputs = append(inputs,
			keyEvent(0, ch, KEYEVENTF_UNICODE),
			keyEvent(0, ch, KEYEVENTF_UNICODE|KEYEVENTF_KEYUP))
	}
	if len(inputs) > 0 {
		seq0, _, _ := pClipboardSeq.Call()
		gSending = true
		sendInput(inputs)
		gSending = false
		if savedClip != 0 {
			restoreClipboardLater(savedClip, seq0)
		}
	} else if savedClip != 0 {
		comRelease(savedClip)
	}
}

// vkToChar translates a virtual key to the ASCII char the engine expects.
func vkToChar(vk, scan uint32) (byte, bool) {
	var kb [256]byte
	if gShiftHeld {
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

// postToggle defers the actual toggle work (registry writes, tray RPC, HUD)
// to the message loop so the hook returns instantly — rapid hotkey presses
// then queue as cheap messages instead of serializing key events behind I/O.
// wParam identifies the source for the crash.log line.
func postToggle(src uintptr) {
	pPostMessage.Call(gHwnd, WM_TOGGLE_VN, src, 0)
}

// handleKeyDown returns true if the key must be swallowed.
func handleKeyDown(p *kbdLLHookStruct) bool {
	vk := p.vkCode
	spec, exe, hwnd := foregroundApp()
	if hwnd != gLastKeyHwnd {
		// Focus moved (Alt+Tab, taskbar click, Win+<n>) — the tracked
		// word belongs to the previous window's edit field.
		gLastKeyHwnd = hwnd
		gEngine.Reset()
		gAutoCapNext = false
		gCaretAtEnd = true // fresh focus: caret at end or whole text selected
	}
	gForceClip = spec.mode == appModeClip
	gForceTCVN = spec.tcvn
	gSafeDel = spec.safeDel || gSafeDelExes[exe]

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

	// toggle hotkey: configured modifiers + key. One toggle per press —
	// the auto-repeat stream while held must not toggle back and forth.
	if cfg.HotkeyVk != 0 && vk == uint32(cfg.HotkeyVk) && modsMatch(cfg.HotkeyMods) {
		if !gHotkeyDown {
			gHotkeyDown = true
			if flipVietKey() {
				postToggle(0)
			}
		}
		return true
	}
	if vk == VK_LSHIFT || vk == VK_RSHIFT || vk == VK_LCONTROL || vk == VK_RCONTROL {
		// Track held state ourselves instead of GetAsyncKeyState — inside an
		// LL hook the async state can lag the event stream just enough to
		// drop a fast chord tap.
		wasChord := gCtrlHeld && gShiftHeld
		noneHeld := !gCtrlHeld && !gShiftHeld
		// While Alt is held (Alt+Tab / Alt+Esc), Windows injects a phantom
		// Ctrl that reaches LL hooks — don't let it count toward the chord.
		altHeld := keyDown(VK_MENU)
		if !altHeld {
			switch vk {
			case VK_LCONTROL, VK_RCONTROL:
				gCtrlHeld = true
			case VK_LSHIFT, VK_RSHIFT:
				gShiftHeld = true
			}
		}
		if noneHeld {
			gModDownAt = time.Now()
		}
		// Modifier-only chord (Ctrl+Shift): ARM it here but fire on release —
		// toggling on press steals real combos like Ctrl+Shift+S (Firefox
		// screenshot). Any non-modifier key or mouse click while held marks it
		// dirty. The two presses must also land close together — a long-held
		// Shift (typing capitals) plus an accidental Ctrl tap is not a chord.
		modChord := cfg.UseCtrlShift || (cfg.HotkeyVk == 0 && cfg.HotkeyMods == 3)
		if modChord && !wasChord && gCtrlHeld && gShiftHeld &&
			time.Since(gModDownAt) < 800*time.Millisecond &&
			gLastKeyAt.Before(gModDownAt) && // no letters between the two presses
			!altHeld && !gWinHeld {
			gChordPending = true
			gChordClean = true
		}
		return false // never swallow modifiers
	}

	// Track Alt/Win like the chord modifiers — the rest of this function
	// reads tracked state because GetAsyncKeyState lags the event stream
	// inside an LL hook and sees injected keys (AltGr, the phantom Ctrl
	// Windows sends while Alt is held).
	switch vk {
	case VK_MENU, VK_LMENU, VK_RMENU:
		gAltHeld = true
	case VK_LWIN, VK_RWIN:
		gWinHeld = true
	}

	// quick-action F keys (raw, no modifiers held)
	if cfg.FKeys != 0 && vk >= VK_F1 && vk <= VK_F24 &&
		!gCtrlHeld && !gAltHeld && !gShiftHeld {
		switch vk {
		case VK_F1:
			if cfg.FKeys&1 != 0 {
				if !effectiveViet() && flipVietKey() {
					postToggle(2)
				}
				return true
			}
		case VK_F2:
			if cfg.FKeys&2 != 0 {
				if effectiveViet() && flipVietKey() {
					postToggle(3)
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
	gLastKeyAt = time.Now()
	if gChordPending {
		// Any non-chord key while armed kills the chord — even after the
		// first modifier released (Shift-held typing with a brushed Ctrl
		// must not toggle when the surviving letter lands before Shift-up).
		gChordClean = false
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
		// Caret-movement keys decide whether a forward-Delete in the
		// safeDel correction path is harmless (end-of-text/selection) or
		// eats a real character (mid-text).
		switch vk {
		case VK_LEFT, VK_UP, VK_HOME, VK_PRIOR, VK_RETURN:
			gCaretAtEnd = false
		case VK_END, VK_DOWN, VK_NEXT:
			gCaretAtEnd = true
		}
		return false
	}
	if isModifierKey(vk) {
		return false
	}
	// shortcuts (Ctrl/Alt/Win held) bypass the engine — and invalidate its
	// word: Ctrl+A/C/X/V, Ctrl+Backspace, Win combos change the text,
	// selection or focus behind the engine's back.
	if gCtrlHeld || gAltHeld || gWinHeld {
		gEngine.Reset()
		gAutoCapNext = false
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
	defer func() { recoverCrash("hook", recover()) }()
	gHookSeen++
	if nCode == 0 && !gSending { // HC_ACTION
		p := (*kbdLLHookStruct)(*(*unsafe.Pointer)(unsafe.Pointer(&lParam)))
		injected := p.flags&(LLKHF_INJECTED|LLKHF_LOWER_IL_INJECTED) != 0
		if !injected || (gTestMagic != 0 && p.dwExtraInfo == gTestMagic) {
			switch wParam {
			case WM_KEYUP, WM_SYSKEYUP:
				switch p.vkCode {
				case VK_LCONTROL, VK_RCONTROL, VK_LSHIFT, VK_RSHIFT:
					if p.vkCode == VK_LCONTROL || p.vkCode == VK_RCONTROL {
						gCtrlHeld = false
					} else {
						gShiftHeld = false
					}
					if gChordPending {
						if gCtrlHeld || gShiftHeld {
							// First release — the other modifier is still
							// held. Brushing Ctrl while holding Shift for
							// capitals lands here and must NOT toggle.
							gModUpAt = time.Now()
						} else {
							// Both released: toggle only if the chord stayed
							// clean and the pair was let go as one gesture.
							if gChordClean && time.Since(gModUpAt) < 700*time.Millisecond &&
								flipVietKey() {
								postToggle(1)
							}
							gChordPending = false
						}
					}
				case VK_MENU, VK_LMENU, VK_RMENU:
					gAltHeld = false
				case VK_LWIN, VK_RWIN:
					gWinHeld = false
				}
				if p.vkCode == uint32(cfg.HotkeyVk) {
					gHotkeyDown = false
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

// lowLevelMouseProc handles two jobs on any button press:
//   - dirty a pending Ctrl+Shift chord — Ctrl+Shift+Click (open link in new
//     tab, multi-select, Explorer shortcut-drag) must not toggle Vietnamese
//   - reset the engine — a click can move the caret, select a word, or
//     switch focus, so the tracked word no longer matches the edit field.
//     Without this, Backspace/typing "corrects" text at the wrong offset
//     (click before "xóa" + retype x → "õa", omnibox → "àáá").
var gMouseSeen uint64 // liveness counter, mirrors gHookSeen

func lowLevelMouseProc(nCode int, wParam, lParam uintptr) uintptr {
	defer func() { recoverCrash("mousehook", recover()) }()
	gMouseSeen++
	if nCode == 0 {
		switch wParam {
		case WM_LBUTTONDOWN, WM_RBUTTONDOWN, WM_MBUTTONDOWN, WM_XBUTTONDOWN,
			WM_NCLBUTTONDOWN, WM_NCRBUTTONDOWN, WM_NCMBUTTONDOWN, WM_NCXBUTTONDOWN:
			gEngine.Reset()
			gAutoCapNext = false
			// A click inside the window already receiving keys may land the
			// caret mid-text — forward-Delete there would erase a real
			// character. A click on another window refocuses it; the field
			// then typically selects all text or sits at end-of-text.
			var pt point
			pGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
			w, _, _ := pWindowFromPoint.Call(
				uintptr(uint32(pt.x)) | uintptr(uint32(pt.y))<<32)
			clicked, _, _ := pGetAncestor.Call(w, GA_ROOT)
			gCaretAtEnd = clicked != gLastKeyHwnd
			if gChordPending {
				gChordClean = false
			}
		}
	}
	r, _, _ := pCallNextHookEx.Call(gMouseHook, uintptr(nCode), wParam, lParam)
	return r
}
