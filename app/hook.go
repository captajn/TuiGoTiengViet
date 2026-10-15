package main

import (
	"sync/atomic"
	"time"
	"unsafe"
)

var (
	gHook         uintptr
	gMouseHook    uintptr
	gSending      bool // reentrancy guard for our own SendInput
	gChordClean   = true
	gChordPending bool // Ctrl+Shift is complete — toggle fires on release
	gShiftFirst   bool // true if Shift went down before Ctrl in the current chord
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
	// The bait path needs no cut probe. Keep the legacy probe only for
	// TUIGO_NO_GUARD; never log correction metadata from the keyboard hook.
	if gSafeDel && res.Backs > 0 && !res.KeyOut && gNoGuard && !gShiftHeld {
		delPairs = gCaretAtEnd
		cut, saved, ok := probeSelectionCut()
		if ok {
			delPairs = false
			savedClip = saved
			if cut == res.Backs {
				res.Backs = 0
			}
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
// wParam identifies the deferred toggle source.
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
		isCtrl := vk == VK_LCONTROL || vk == VK_RCONTROL
		// While Alt is held (Alt+Tab / Alt+Esc), Windows injects a phantom
		// Ctrl that reaches LL hooks — don't let it count toward the chord.
		altHeld := keyDown(VK_MENU)
		if !altHeld {
			if isCtrl {
				gCtrlHeld = true
			} else {
				gShiftHeld = true
			}
		}
		if noneHeld {
			gModDownAt = time.Now()
			gShiftFirst = !isCtrl
			gChordClean = true
		} else if !wasChord && gCtrlHeld && gShiftHeld && gChordClean {
			// The second modifier just joined the first — refresh the arm
			// timestamp regardless of which came first (Ctrl+Shift or
			// Shift+Ctrl). Without this, pressing Shift first and holding
			// it for >800ms before pressing Ctrl silently prevents the chord
			// from arming.
			gModDownAt = time.Now()
		}
		// Modifier-only chord (Ctrl+Shift): ARM it here but fire on release —
		// toggling on press steals real combos like Ctrl+Shift+S (Firefox
		// screenshot). Any non-modifier key or mouse click while held marks it
		// dirty. The two presses must also land close together — a long-held
		// Shift (typing capitals) plus an accidental Ctrl tap is not a chord.
		modChord := cfg.UseCtrlShift || (cfg.HotkeyVk == 0 && cfg.HotkeyMods == 3)
		if modChord && !wasChord && gCtrlHeld && gShiftHeld &&
			time.Since(gModDownAt) < 1200*time.Millisecond &&
			gLastKeyAt.Before(gModDownAt) && // no letters between the two presses
			gChordClean && !altHeld && !gWinHeld {
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
	viet := vietForApp(spec, exe)
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
	defer func() { _ = recover() }()
	gHookSeen++
	if nCode == 0 && !gSending { // HC_ACTION
		flushPendingMouseClick()
		p := (*kbdLLHookStruct)(*(*unsafe.Pointer)(unsafe.Pointer(&lParam)))
		injected := p.flags&(LLKHF_INJECTED|LLKHF_LOWER_IL_INJECTED) != 0
		if !injected || (gTestMagic != 0 && p.dwExtraInfo == gTestMagic) {
			switch wParam {
			case WM_KEYUP, WM_SYSKEYUP:
				switch p.vkCode {
				case VK_LCONTROL, VK_RCONTROL, VK_LSHIFT, VK_RSHIFT:
					isCtrl := p.vkCode == VK_LCONTROL || p.vkCode == VK_RCONTROL
					if isCtrl {
						gCtrlHeld = false
					} else {
						gShiftHeld = false
					}
					if gChordPending {
						if gShiftFirst && gShiftHeld && isCtrl {
							// Shift went down first and is still held while Ctrl was
							// released: wait for Shift release so brushing Ctrl while
							// holding Shift for capitals doesn't toggle before the letter.
							gModUpAt = time.Now()
						} else if !gCtrlHeld && !gShiftHeld && !gModUpAt.IsZero() {
							if gChordClean && time.Since(gModUpAt) < 900*time.Millisecond &&
								flipVietKey() {
								postToggle(1)
							}
							gChordPending = false
							gModUpAt = time.Time{}
						} else {
							// Clean Ctrl+Shift (either Shift released while Ctrl held,
							// or first release when Ctrl went down first): fire immediately!
							if gChordClean && flipVietKey() {
								postToggle(1)
							}
							gChordPending = false
							gModUpAt = time.Time{}
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
				if isModifierKey(p.vkCode) && !gCtrlHeld && !gShiftHeld {
					gChordClean = true
				}
			case WM_KEYDOWN, WM_SYSKEYDOWN:
				// Self-heal stale modifier flags on physical modifier press so a
				// missed keyup (window activation / UAC / focus switch) never
				// stalls Ctrl+Shift for 30 seconds until the timer runs.
				if !injected && gTestMagic == 0 {
					switch p.vkCode {
					case VK_LCONTROL, VK_RCONTROL:
						if gShiftHeld && !keyDown(VK_LSHIFT) && !keyDown(VK_RSHIFT) {
							gShiftHeld = false
						}
						if gAltHeld && !keyDown(VK_LMENU) && !keyDown(VK_RMENU) {
							gAltHeld = false
						}
						if gWinHeld && !keyDown(VK_LWIN) && !keyDown(VK_RWIN) {
							gWinHeld = false
						}
					case VK_LSHIFT, VK_RSHIFT:
						if gCtrlHeld && !keyDown(VK_LCONTROL) && !keyDown(VK_RCONTROL) {
							gCtrlHeld = false
						}
						if gAltHeld && !keyDown(VK_LMENU) && !keyDown(VK_RMENU) {
							gAltHeld = false
						}
						if gWinHeld && !keyDown(VK_LWIN) && !keyDown(VK_RWIN) {
							gWinHeld = false
						}
					}
				}
				if handleKeyDown(p) {
					return 1
				}
			}
		}
	}
	r, _, _ := pCallNextHookEx.Call(gHook, uintptr(nCode), wParam, lParam)
	return r
}

// lowLevelMouseProc runs on its own dedicated OS thread so UI painting in
// BoGoSettingsWnd can never delay WH_MOUSE_LL and stutter the system cursor.
// It handles two jobs on any button press:
//   - dirty a pending Ctrl+Shift chord — Ctrl+Shift+Click (open link in new
//     tab, multi-select, Explorer shortcut-drag) must not toggle Vietnamese
//   - reset the engine — a click can move the caret, select a word, or
//     switch focus, so the tracked word no longer matches the edit field.
//     Without this, Backspace/typing "corrects" text at the wrong offset
//     (click before "xóa" + retype x → "õa", omnibox → "àáá").
var (
	gMouseSeen         uint64 // liveness counter, mirrors gHookSeen (atomic)
	gPendingMouseClick uint32
	gPendingClickHwnd  uintptr
	gPendingClickFg    uintptr
)

func flushPendingMouseClick() {
	if atomic.SwapUint32(&gPendingMouseClick, 0) == 0 {
		return
	}
	clicked := atomic.LoadUintptr(&gPendingClickHwnd)
	fg := atomic.LoadUintptr(&gPendingClickFg)
	gEngine.Reset()
	gAutoCapNext = false
	// A click inside the window already receiving keys may land the
	// caret mid-text — forward-Delete there would erase a real
	// character. A click on another window refocuses it; the field
	// then typically selects all text or sits at end-of-text.
	if shellInputSurface(rootClassName(clicked)) || clicked == gSettingsHwnd || clicked == gHudHwnd {
		// Mouse-down ran before Explorer/settings took foreground.
		// Capture the typing app even if the user never typed a key in it.
		captureInputAppAt(fg)
	}
	gCaretAtEnd = clicked != gLastKeyHwnd
	if gChordPending {
		gChordClean = false
	}
}

func lowLevelMouseProc(nCode int, wParam, lParam uintptr) uintptr {
	defer func() { _ = recover() }()
	atomic.AddUint64(&gMouseSeen, 1)
	if nCode == 0 {
		switch wParam {
		case WM_LBUTTONDOWN, WM_RBUTTONDOWN, WM_MBUTTONDOWN, WM_XBUTTONDOWN,
			WM_NCLBUTTONDOWN, WM_NCRBUTTONDOWN, WM_NCMBUTTONDOWN, WM_NCXBUTTONDOWN:
			var pt point
			if lParam != 0 {
				pt = (*msLLHookStruct)(*(*unsafe.Pointer)(unsafe.Pointer(&lParam))).pt
			} else {
				pGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
			}
			w, _, _ := pWindowFromPoint.Call(
				uintptr(uint32(pt.x)) | uintptr(uint32(pt.y))<<32)
			clicked, _, _ := pGetAncestor.Call(w, GA_ROOT)
			fg, _, _ := pGetForegroundWindow.Call()
			atomic.StoreUintptr(&gPendingClickHwnd, clicked)
			atomic.StoreUintptr(&gPendingClickFg, fg)
			atomic.StoreUint32(&gPendingMouseClick, 1)
			if gHwnd != 0 {
				pPostMessage.Call(gHwnd, WM_MOUSE_DOWN, clicked, fg)
			} else {
				flushPendingMouseClick()
			}
		}
	}
	r, _, _ := pCallNextHookEx.Call(atomic.LoadUintptr(&gMouseHook), uintptr(nCode), wParam, lParam)
	return r
}
