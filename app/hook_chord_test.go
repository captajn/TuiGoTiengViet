package main

import (
	"sync/atomic"
	"testing"
	"time"
)

// setupChordTest isolates the Ctrl+Shift chord state for one test.
func setupChordTest(t *testing.T) {
	t.Helper()
	setupAppModeTest(t)
	oldPending, oldDirty, oldDown, oldUp, oldLast := gChordPending, gChordDirtyAt, gModDownAt, gModUpAt, gLastKeyAt
	t.Cleanup(func() {
		gChordPending, gChordDirtyAt, gModDownAt, gModUpAt, gLastKeyAt = oldPending, oldDirty, oldDown, oldUp, oldLast
		_ = atomic.SwapUint32(&gPendingMouseClick, 0)
	})
	cfg.UseCtrlShift = true
	focusAppForTest("notepad.exe", appSpec{im: -1})
	gChordPending, gChordDirtyAt, gModDownAt, gModUpAt, gLastKeyAt = false, time.Time{}, time.Time{}, time.Time{}, time.Time{}
}

// tick separates two events in time. The chord rules compare timestamps, and
// the OS clock is coarse enough that back-to-back calls can land in the same
// tick — real keys are never that close together.
func tick() { time.Sleep(3 * time.Millisecond) }

// armChord sends a Ctrl+Shift keydown pair through the real handler.
func armChord(t *testing.T) {
	t.Helper()
	if handleKeyDown(&kbdLLHookStruct{vkCode: VK_LCONTROL}) {
		t.Fatal("Ctrl must never be swallowed")
	}
	tick()
	if handleKeyDown(&kbdLLHookStruct{vkCode: VK_LSHIFT}) {
		t.Fatal("Shift must never be swallowed")
	}
}

// releaseChord drops the held modifiers the way the keyup path (or the
// watchdog, when those keyups are lost) does.
func releaseChord() {
	gCtrlHeld, gShiftHeld, gChordPending = false, false, false
	gModUpAt = time.Time{}
	tick()
}

// TestChordRetryAfterInterruptedChord covers issue #4, item 3: after a chord
// that a typed letter interrupted, the next Ctrl+Shift must still toggle.
// The dirty marker used to be a latch that only a modifier release cleared, so
// one interrupted attempt swallowed the next one — the reported "toggling the
// floating badge makes the Ctrl+Shift switch dead for a moment".
func TestChordRetryAfterInterruptedChord(t *testing.T) {
	setupChordTest(t)

	armChord(t)
	if !gChordPending || !pureChord() {
		t.Fatal("a clean Ctrl+Shift must arm the chord")
	}

	// A letter lands between the presses (Ctrl+Shift+S): the chord is dirty.
	tick()
	handleKeyDown(&kbdLLHookStruct{vkCode: 'S'})
	if pureChord() {
		t.Fatal("typing inside a chord must mark it dirty")
	}

	releaseChord()
	armChord(t)
	if !gChordPending {
		t.Fatal("an interrupted chord must not block the next Ctrl+Shift")
	}
	if !pureChord() {
		t.Fatal("the retry chord starts clean and must be allowed to fire")
	}
}

// TestChordClickHandling keeps the click path honest: a click while armed
// dirties the chord, a click before it started does not, and neither leaves a
// latch behind that blocks the next chord.
func TestChordClickHandling(t *testing.T) {
	setupChordTest(t)

	click := func() {
		t.Helper()
		atomic.StoreUint32(&gPendingMouseClick, 1)
		flushPendingMouseClick()
	}

	click()
	tick()
	armChord(t)
	if !gChordPending || !pureChord() {
		t.Fatal("a click before the chord must not dirty it")
	}

	tick()
	click()
	if pureChord() {
		t.Fatal("a click inside the chord must dirty it")
	}

	releaseChord()
	armChord(t)
	if !gChordPending {
		t.Fatal("a chord interrupted by a click must not block the next one")
	}
}
