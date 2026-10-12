package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupAppModeTest(t *testing.T) {
	t.Helper()
	oldConfig, oldGlobal := cfg, gVietKey
	oldRemembered, oldPerApp := gRememberedAppViet, gPerAppViet
	oldEngine, oldAutoCap := gEngine, gAutoCapNext
	oldIM := gIM
	oldKeyHwnd, oldLastIM := gLastKeyHwnd, gLastIM
	oldClip, oldTCVN, oldSafeDel, oldCaret := gForceClip, gForceTCVN, gSafeDel, gCaretAtEnd
	oldCtrl, oldAlt, oldShift, oldWin, oldHotkey := gCtrlHeld, gAltHeld, gShiftHeld, gWinHeld, gHotkeyDown
	oldHwnd, oldSpec, oldExe := lastFgHwnd, lastFgSpec, lastFgExe
	oldInputApp := gLastInputApp
	t.Cleanup(func() {
		cfg, gVietKey = oldConfig, oldGlobal
		gRememberedAppViet, gPerAppViet = oldRemembered, oldPerApp
		gEngine, gAutoCapNext = oldEngine, oldAutoCap
		gIM = oldIM
		gLastKeyHwnd, gLastIM = oldKeyHwnd, oldLastIM
		gForceClip, gForceTCVN, gSafeDel, gCaretAtEnd = oldClip, oldTCVN, oldSafeDel, oldCaret
		gCtrlHeld, gAltHeld, gShiftHeld, gWinHeld, gHotkeyDown = oldCtrl, oldAlt, oldShift, oldWin, oldHotkey
		lastFgHwnd, lastFgSpec, lastFgExe = oldHwnd, oldSpec, oldExe
		gLastInputApp = oldInputApp
	})
	gEngine, _ = newEngine()
	cfg = config{RememberAppMode: true}
	gVietKey = true
	gRememberedAppViet = map[string]bool{}
	gPerAppViet = map[string]bool{}
	gLastInputApp = inputAppContext{}
	gCtrlHeld, gAltHeld, gShiftHeld, gWinHeld, gHotkeyDown = false, false, false, false, false
}

func focusAppForTest(exe string, spec appSpec) {
	// Exercise real callers without moving focus on the user's desktop.
	lastFgHwnd, _, _ = pGetForegroundWindow.Call()
	lastFgSpec, lastFgExe = spec, exe
	gLastInputApp = inputAppContext{spec, exe, lastFgHwnd}
}

func TestRememberAppModeSwitchAndRestart(t *testing.T) {
	setupAppModeTest(t)
	normal := appSpec{im: -1}
	focusAppForTest("game.exe", normal)
	if !flipVietKey() || effectiveViet() || !gVietKey {
		t.Fatal("disabling the game must leave the global default enabled")
	}
	focusAppForTest("notepad.exe", normal)
	if !effectiveViet() {
		t.Fatal("another app must still type Vietnamese")
	}
	focusAppForTest("game.exe", normal)
	if effectiveViet() {
		t.Fatal("returning to the game must restore English")
	}
	// The hook must use the same decision as the tray/HUD.
	cfg.HotkeyMods, cfg.HotkeyVk = 4, 'Z'
	for _, ch := range "aas" {
		if handleKeyDown(&kbdLLHookStruct{vkCode: uint32(strings.ToUpper(string(ch))[0])}) {
			t.Fatal("remembered English app must pass Telex letters through")
		}
	}
	p := filepath.Join(t.TempDir(), "tuigo.ini")
	gRememberedAppViet["chrome.exe"] = true
	if err := saveSettingsTo(p); err != nil {
		t.Fatal(err)
	}
	vals, err := readIni(p)
	if err != nil {
		t.Fatal(err)
	}
	gRememberedAppViet = nil
	loadRememberedAppModes(vals)
	if vals["RememberAppMode"] != "1" || effectiveViet() || !gRememberedAppViet["chrome.exe"] {
		t.Fatal("saved mode must survive reload")
	}
	if !flipVietKey() || !effectiveViet() {
		t.Fatal("turning Vietnamese back on must replace the saved English mode")
	}
	if err := saveSettingsTo(p); err != nil {
		t.Fatal(err)
	}
	vals, _ = readIni(p)
	loadRememberedAppModes(vals)
	if !effectiveViet() {
		t.Fatal("re-enabled Vietnamese must survive reload")
	}
}

func TestRememberAppModeQuickKeys(t *testing.T) {
	setupAppModeTest(t)
	focusAppForTest("game.exe", appSpec{im: -1})
	cfg.HotkeyMods, cfg.HotkeyVk, cfg.FKeys = 4, 'Z', 1|2|16
	gAltHeld = true
	if !handleKeyDown(&kbdLLHookStruct{vkCode: 'Z'}) || effectiveViet() || !gVietKey {
		t.Fatal("Alt+Z must save English only for this app")
	}
	if !handleKeyDown(&kbdLLHookStruct{vkCode: 'Z'}) || effectiveViet() {
		t.Fatal("hotkey auto-repeat must not re-toggle")
	}
	gAltHeld = false
	if !handleKeyDown(&kbdLLHookStruct{vkCode: VK_F1}) || !effectiveViet() {
		t.Fatal("F1 must replace the remembered English choice")
	}
	if !handleKeyDown(&kbdLLHookStruct{vkCode: VK_F2}) || effectiveViet() {
		t.Fatal("F2 must save English again")
	}
	if !handleKeyDown(&kbdLLHookStruct{vkCode: VK_F12}) || effectiveViet() {
		t.Fatal("resetting the typing buffer must preserve remembered app choices")
	}
	for i := 0; i < 20; i++ {
		if !flipVietKey() || effectiveViet() != (i%2 == 0) || !gVietKey {
			t.Fatalf("rapid per-app toggle %d was lost or changed the global mode", i)
		}
	}
}

func TestRememberAppModeProfiles(t *testing.T) {
	setupAppModeTest(t)
	gRememberedAppViet = map[string]bool{"app.exe": true}
	gVietKey = false
	for _, tc := range []struct {
		name string
		mode int
		want bool
	}{
		{"normal", appModeNone, true},
		{"clipboard", appModeClip, true},
		{"manual", appModeManual, true},
		{"lock", appModeLock, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			focusAppForTest("app.exe", appSpec{mode: tc.mode, im: ImVni, tcvn: true})
			if effectiveViet() != tc.want {
				t.Fatal("profile and remembered mode disagree")
			}
			if tc.mode == appModeLock && flipVietKey() {
				t.Fatal("remembering must not bypass a locked profile")
			}
		})
	}
	// A manually excluded app can be enabled and keep that choice on return.
	gRememberedAppViet = map[string]bool{}
	focusAppForTest("manual.exe", appSpec{mode: appModeManual, im: -1})
	if effectiveViet() || !flipVietKey() || !effectiveViet() || gVietKey {
		t.Fatal("manual exclusion must toggle per-app without changing the global default")
	}
}

func TestRememberAppModeDisabledAndUnknown(t *testing.T) {
	setupAppModeTest(t)
	gRememberedAppViet["game.exe"] = false
	focusAppForTest("game.exe", appSpec{im: -1})
	cfg.RememberAppMode = false
	if !effectiveViet() || !flipVietKey() || gVietKey || gRememberedAppViet["game.exe"] {
		t.Fatal("disabled feature must keep the global toggle and preserve saved choices")
	}
	cfg.RememberAppMode = true
	gVietKey = true
	if effectiveViet() {
		t.Fatal("re-enabling must restore the saved choice")
	}
	focusAppForTest("", appSpec{im: -1})
	if !flipVietKey() || gVietKey {
		t.Fatal("unknown process must fall back to the global toggle")
	}
	if _, ok := gRememberedAppViet[""]; ok {
		t.Fatal("unknown processes must never share an empty app identity")
	}
}

func TestLoadRememberedAppModesValidation(t *testing.T) {
	setupAppModeTest(t)
	loadRememberedAppModes(map[string]string{
		"AppMode.NotePad.EXE": "0", "AppMode.browser.exe": "1",
		"AppMode.": "0", "AppMode.bad.exe": "invalid",
		"AppMode.dir/app.exe": "1", "AppMode.bad\nname.exe": "0",
		"Unrelated": "1",
	})
	if len(gRememberedAppViet) != 2 || gRememberedAppViet["notepad.exe"] || !gRememberedAppViet["browser.exe"] {
		t.Fatalf("unexpected restored choices: %v", gRememberedAppViet)
	}
	loadRememberedAppModes(map[string]string{})
	if len(gRememberedAppViet) != 0 {
		t.Fatal("old settings without saved apps must start with an empty map")
	}
}

func TestSaveRememberedAppModesFailure(t *testing.T) {
	setupAppModeTest(t)
	p := filepath.Join(t.TempDir(), "settings.ini")
	if err := os.Mkdir(p, 0755); err != nil {
		t.Fatal(err)
	}
	if err := saveSettingsTo(p); err == nil {
		t.Fatal("a failed rename must report the persistence error")
	}
}

func TestRememberModeUnseenAppDoesNotInheritGlobalEnglish(t *testing.T) {
	setupAppModeTest(t)
	gVietKey = false
	if !vietForApp(appSpec{im: -1}, "notepad.exe") {
		t.Fatal("an unseen app must default to Vietnamese when remembering is on")
	}
	if vietForApp(appSpec{mode: appModeManual, im: -1}, "game.exe") {
		t.Fatal("an explicitly excluded app must still default to English")
	}
	gRememberedAppViet["notepad.exe"] = false
	if vietForApp(appSpec{im: -1}, "notepad.exe") {
		t.Fatal("a saved English choice must override the new-app default")
	}
}

func TestEnableRememberModePreservesTypingApp(t *testing.T) {
	setupAppModeTest(t)
	cfg.RememberAppMode = false
	gVietKey = false
	focusAppForTest("chrome.exe", appSpec{im: -1})
	setRememberAppMode(true)
	if effectiveViet() || gRememberedAppViet["chrome.exe"] {
		t.Fatal("enabling must preserve the typing app's existing English mode")
	}
	focusAppForTest("notepad.exe", appSpec{im: -1})
	if !effectiveViet() {
		t.Fatal("other apps must default to Vietnamese instead of inheriting global English")
	}
}

func TestTrayAndSettingsKeepTypingApp(t *testing.T) {
	setupAppModeTest(t)
	chrome := inputAppContext{appSpec{im: -1}, "chrome.exe", 100}
	notepad := inputAppContext{appSpec{im: -1}, "notepad.exe", 200}
	resolveInputApp(chrome, "Chrome_WidgetWin_1", false)
	for _, cls := range []string{"Shell_TrayWnd", "Shell_SecondaryTrayWnd", "NotifyIconOverflowWindow", "TopLevelWindowForOverflowXamlIsland"} {
		shell := inputAppContext{appSpec{im: -1}, "explorer.exe", 300}
		if got := resolveInputApp(shell, cls, false); got != chrome {
			t.Fatalf("%s replaced the typing app: %+v", cls, got)
		}
	}
	settings := inputAppContext{appSpec{im: -1}, "tuigo.exe", 400}
	if got := resolveInputApp(settings, "TuiGoSettings", true); got != chrome {
		t.Fatal("opening settings must not replace the typing app")
	}
	// Capture on tray mouse-up; the user changes focus before the delayed timer.
	captured := gLastInputApp
	resolveInputApp(notepad, "Notepad", false)
	if !flipVietKeyForApp(captured) || gRememberedAppViet["chrome.exe"] {
		t.Fatal("deferred tray toggle must affect the captured Chrome app")
	}
	if !vietForApp(notepad.spec, notepad.exe) {
		t.Fatal("a focus change during the double-click interval must not toggle Notepad")
	}
	if _, ok := gRememberedAppViet["explorer.exe"]; ok {
		t.Fatal("tray click must never remember a preference for Explorer")
	}
	if _, ok := gRememberedAppViet["tuigo.exe"]; ok {
		t.Fatal("settings must never remember a preference for Tui Go")
	}
	// A real Explorer file-management window is a valid typing app.
	explorer := inputAppContext{appSpec{im: -1}, "explorer.exe", 500}
	if got := resolveInputApp(explorer, "CabinetWClass", false); got != explorer {
		t.Fatal("ordinary Explorer windows must keep their own mode")
	}
}

func TestNormalTypingWithRememberedModes(t *testing.T) {
	setupAppModeTest(t)
	app := inputAppContext{appSpec{im: -1}, "notepad.exe", 100}
	typeWord := func(keys string) string {
		gEngine.Reset()
		gEngine.SetVietKey(vietForApp(app.spec, app.exe))
		var out []rune
		for _, ch := range keys {
			res := gEngine.Filter(uint32(ch))
			if !res.Consumed {
				out = append(out, ch)
				continue
			}
			out = out[:len(out)-res.Backs]
			for _, ch := range res.Out {
				out = append(out, rune(ch))
			}
		}
		return string(out)
	}
	if got := typeWord("aas"); got != "ấ" {
		t.Fatalf("normal Telex with remembering on: %q", got)
	}
	flipVietKeyForApp(app)
	if got := typeWord("aas"); got != "aas" {
		t.Fatalf("explicit English mode: %q", got)
	}
	flipVietKeyForApp(app)
	if got := typeWord("aas"); got != "ấ" {
		t.Fatalf("Vietnamese after re-enabling: %q", got)
	}
}

// Drive the production keyboard handler, while preventing SendInput from
// writing into the user's foreground window. The real engine still runs.
type typingModeProbe struct {
	Engine
	on bool
}

func (p *typingModeProbe) SetVietKey(on bool) {
	p.on = on
	p.Engine.SetVietKey(on)
}

func (p *typingModeProbe) Filter(ch uint32) Result {
	p.Engine.Filter(ch)
	return Result{}
}

func (p *typingModeProbe) Backspace() Result {
	p.Engine.Backspace()
	return Result{}
}

func TestOrdinaryTypingNeverSwitchesToEnglish(t *testing.T) {
	setupAppModeTest(t)
	focusAppForTest("normal-typing.exe", appSpec{im: -1})
	cfg.RememberAppMode = false
	setRememberAppMode(true)
	cfg.HotkeyMods, cfg.HotkeyVk = 4, 'Z'
	probe := &typingModeProbe{Engine: gEngine, on: true}
	gEngine = probe
	// Includes every letter, Vietnamese tone keys, digits, punctuation,
	// whitespace and ordinary editing keys. None is a V/E command.
	for _, im := range []int{ImTelex, ImVni} {
		oldIM := gIM
		gIM = im
		for pass := 0; pass < 50; pass++ {
			for _, vk := range []uint32{
				'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M',
				'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z',
				'0', '1', '2', '3', '4', '5', '6', '7', '8', '9',
				VK_SPACE, VK_BACK, VK_LEFT, VK_RIGHT, VK_RETURN, VK_TAB,
			} {
				handleKeyDown(&kbdLLHookStruct{vkCode: vk})
				if !probe.on || !effectiveViet() || !gRememberedAppViet["normal-typing.exe"] {
					t.Fatalf("method %d, pass %d, vk %d: normal typing switched V to E", im, pass, vk)
				}
			}
		}
		gIM = oldIM
	}
}
