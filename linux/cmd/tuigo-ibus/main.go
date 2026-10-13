// tuigo-ibus is the IBus engine daemon for Tui Go. ibus-daemon spawns it
// via the component XML (exec line); --standalone registers the component
// on a running daemon for development.
//
// Key handling mirrors the Windows shell: printable chars go through the
// engine, navigation/modifier chords reset it, Backspace goes through
// engine.Backspace. Output reaches the app as preedit/commit text — see
// internal/session for why committed text is never edited.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"time"
	"unicode/utf16"
	"unicode/utf8"

	ibus "github.com/BambooEngine/goibus"
	"github.com/godbus/dbus/v5"

	"tuigo-linux/config"
	"tuigo-linux/internal/session"
	"tuigo/engine"
)

// IBus key state masks (same values as X11/GDK modifiers).
const (
	ibusShiftMask   = 1 << 0
	ibusLockMask    = 1 << 1 // Caps Lock
	ibusControlMask = 1 << 2
	ibusMod1Mask    = 1 << 3 // Alt
	ibusMod4Mask    = 1 << 6 // Super
	ibusReleaseMask = 1 << 30
)

// IBus keyvals are X11 keysyms; the ones we need:
const (
	keyBackSpace  = 0xFF08
	keyTab        = 0xFF09
	keyISOLeftTab = 0xFE20
	keyReturn     = 0xFF0D
	keyEscape     = 0xFF1B

	keyHome       = 0xFF50
	keyLeft       = 0xFF51
	keyUp         = 0xFF52
	keyRight      = 0xFF53
	keyDown       = 0xFF54
	keyPageUp     = 0xFF55
	keyPageDown   = 0xFF56
	keyEnd        = 0xFF57
	keyInsert     = 0xFF63
	keyMenu       = 0xFF67
	keyKPEnter    = 0xFF8D
	keyKPDelete   = 0xFF9F
	keyKPBegin    = 0xFF9D
	keyPrint      = 0xFF61
	keyScrollLock = 0xFF14
	keyPause      = 0xFF13
	keyNumLock    = 0xFF7F
	keyDelete     = 0xFFFF

	keyF1  = 0xFFBE
	keyF35 = 0xFFE0

	keyShiftL   = 0xFFE1
	keyShiftR   = 0xFFE2
	keyControlL = 0xFFE3
	keyControlR = 0xFFE4
	keyCapsLock = 0xFFE5
	keyMetaL    = 0xFFE7
	keyMetaR    = 0xFFE8
	keyAltL     = 0xFFE9
	keyAltR     = 0xFFEA
	keySuperL   = 0xFFEB
	keySuperR   = 0xFFEC

	keyDeadBase    = 0xFE50     // dead keys start here
	keyUnicodeBase = 0x01000000 // direct-unicode keysyms: 0x01000000|codepoint
)

func isResetKey(k uint32) bool {
	switch k {
	case keyReturn, keyKPEnter, keyTab, keyISOLeftTab, keyEscape,
		keyLeft, keyRight, keyUp, keyDown, keyHome, keyEnd,
		keyPageUp, keyPageDown, keyDelete, keyKPDelete, keyInsert,
		keyMenu, keyPrint, keyPause, keyNumLock, keyScrollLock, keyCapsLock,
		keyKPBegin:
		return true
	}
	return k >= keyF1 && k <= keyF35
}

func isCtrlOrShift(k uint32) bool {
	return k == keyControlL || k == keyControlR || k == keyShiftL || k == keyShiftR
}

// keyvalToChar maps an IBus keyval to the char the engine expects.
// Latin-1 keysyms 0x20–0xFF are already Unicode codepoints; the
// 0x01000000+ range encodes arbitrary Unicode directly. Numpad keys map
// to their ASCII equivalents so VNI digits work on the keypad.
func keyvalToChar(k uint32) (uint32, bool) {
	if k >= 0x20 && k <= 0xFF {
		return k, true
	}
	if k >= keyUnicodeBase && k <= 0x0110FFFF {
		return k & 0xFFFFFF, true
	}
	switch k {
	case 0xFF80: // KP_Space
		return ' ', true
	case 0xFFAA: // KP_Multiply
		return '*', true
	case 0xFFAB: // KP_Add
		return '+', true
	case 0xFFAC: // KP_Separator
		return ',', true
	case 0xFFAD: // KP_Subtract
		return '-', true
	case 0xFFAE: // KP_Decimal
		return '.', true
	case 0xFFAF: // KP_Divide
		return '/', true
	case 0xFF95: // KP_Home etc with numlock off — treat as navigation
		return 0, false
	}
	if k >= 0xFFB0 && k <= 0xFFB9 { // KP_0..KP_9
		return '0' + (k - 0xFFB0), true
	}
	if k >= 0xFF91 && k <= 0xFF9C { // other keypad nav/edit keys
		return 0, false
	}
	return 0, false
}

func u16ToString(units []uint16) string {
	return string(utf16.Decode(units))
}

// -----------------------------------------------------------------------

var engineCounter uint64

type tuigoEngine struct {
	ibus.Engine
	eng  *engine.Engine
	sess *session.Session

	cfg       config.Settings
	viet      bool
	ctrlHeld  bool
	shiftHeld bool

	// clean-chord tracking for the ctrl+shift toggle (fires on release)
	chordArmed bool
	chordClean bool
	modDownAt  time.Time
	modUpAt    time.Time
}

func newTuigoEngine(conn *dbus.Conn, path dbus.ObjectPath) *tuigoEngine {
	e := &tuigoEngine{Engine: ibus.BaseEngine(conn, path)}
	e.reloadConfig()
	return e
}

// reloadConfig re-reads tuigo.ini + macro/keymap files. Called on Enable so
// edits made in tuigo-settings take effect when the IME activates.
func (e *tuigoEngine) reloadConfig() {
	if e.eng == nil {
		e.eng = engine.New()
	}
	cfg, _ := config.Load()
	e.cfg = cfg
	opts := cfg.EngineOptions()
	e.eng.SetOptions(&opts)
	e.eng.SetInputMethod(cfg.InputMethod)
	e.eng.LoadKeyMapFile(config.KeymapPath())
	e.eng.LoadMacroFile(config.MacroPath())

	e.viet = cfg.VietKey
	if e.sess == nil {
		e.sess = session.New(e.eng, cfg.VietKey)
	} else {
		e.apply(e.sess.SetViet(cfg.VietKey))
	}
	e.sess.AutoCap = cfg.AutoCap
}

func (e *tuigoEngine) hasToggle(name string) bool {
	for _, k := range e.cfg.ToggleKeys {
		if k == name {
			return true
		}
	}
	return false
}

// apply turns a session.Action into IBus signals.
// Returns whether the key event was consumed.
func (e *tuigoEngine) apply(a session.Action) bool {
	if len(a.Commit) > 0 || len(a.Preedit) == 0 {
		e.UpdatePreeditText(ibus.NewText(""), 0, false)
	}
	if len(a.Commit) > 0 {
		e.CommitText(ibus.NewText(u16ToString(a.Commit)))
	}
	if len(a.Preedit) > 0 {
		t := ibus.NewText(u16ToString(a.Preedit))
		n := uint32(utf8.RuneCountInString(t.Text))
		t.AppendAttr(ibus.IBUS_ATTR_TYPE_UNDERLINE,
			ibus.IBUS_ATTR_UNDERLINE_SINGLE, 0, n)
		e.UpdatePreeditText(t, n, true)
	}
	return !a.Forward
}

func (e *tuigoEngine) toggleViet() {
	e.viet = !e.viet
	e.cfg.VietKey = e.viet
	e.cfg.Save()
	e.apply(e.sess.SetViet(e.viet))
	e.RegisterProperties(e.props())
}

func (e *tuigoEngine) onModDown(keyval uint32) {
	if !e.ctrlHeld && !e.shiftHeld {
		e.modDownAt = time.Now()
		e.chordClean = true
		e.chordArmed = false
	}
	switch keyval {
	case keyControlL, keyControlR:
		e.ctrlHeld = true
	case keyShiftL, keyShiftR:
		e.shiftHeld = true
	}
	if e.hasToggle("ctrl+shift") && e.ctrlHeld && e.shiftHeld &&
		!e.chordArmed && e.chordClean &&
		time.Since(e.modDownAt) < 800*time.Millisecond {
		e.chordArmed = true
	}
}

func (e *tuigoEngine) onModRelease(keyval uint32) {
	if !isCtrlOrShift(keyval) {
		return
	}
	wasCtrl := keyval == keyControlL || keyval == keyControlR
	if wasCtrl {
		e.ctrlHeld = false
	} else {
		e.shiftHeld = false
	}
	if !e.chordArmed {
		return
	}
	if e.ctrlHeld || e.shiftHeld {
		// first of the pair released — remember when
		e.modUpAt = time.Now()
		return
	}
	e.chordArmed = false
	if e.chordClean && time.Since(e.modUpAt) < 700*time.Millisecond {
		e.toggleViet()
	}
}

// ------------------------------------------------------------------
// org.freedesktop.IBus.Engine interface
// ------------------------------------------------------------------

func (e *tuigoEngine) ProcessKeyEvent(keyval uint32, keycode uint32, state uint32) (bool, *dbus.Error) {
	fmt.Fprintf(os.Stderr, "tuigo-ibus: key %x code %d state %x\n", keyval, keycode, state)
	if state&ibusReleaseMask != 0 {
		e.onModRelease(keyval)
		return false, nil
	}

	// toggle hotkeys checked before the modifier gates below
	if e.hasToggle("alt+z") && (keyval == 'z' || keyval == 'Z') &&
		state&ibusMod1Mask != 0 && state&ibusControlMask == 0 {
		e.toggleViet()
		return true, nil
	}
	if e.hasToggle("ctrl+space") && keyval == ' ' &&
		state&ibusControlMask != 0 {
		e.toggleViet()
		return true, nil
	}

	if isCtrlOrShift(keyval) {
		e.onModDown(keyval)
		return false, nil
	}
	// other modifier-only keys: pass through, don't disturb the word
	switch keyval {
	case keyAltL, keyAltR, keyMetaL, keyMetaR, keySuperL, keySuperR:
		return false, nil
	}
	if e.ctrlHeld || e.shiftHeld {
		e.chordClean = false
	}

	// Ctrl/Alt/Super shortcuts bypass the engine and invalidate the word.
	if state&(ibusControlMask|ibusMod1Mask|ibusMod4Mask) != 0 {
		e.apply(e.sess.Reset())
		return false, nil
	}

	if isResetKey(keyval) {
		if keyval == keyReturn || keyval == keyKPEnter {
			e.apply(e.sess.EnterPressed())
		} else {
			e.apply(e.sess.Reset())
		}
		return false, nil
	}

	if keyval == keyBackSpace {
		return e.apply(e.sess.Backspace()), nil
	}

	e.eng.ShiftPressed = state&ibusShiftMask != 0
	e.eng.CapsLockOn = state&ibusLockMask != 0

	ch, ok := keyvalToChar(keyval)
	if !ok {
		// keysym we can't map (dead keys, non-Latin layouts): commit the
		// pending word and let the app handle the key itself.
		a := e.sess.Reset()
		a.Forward = true
		return e.apply(a), nil
	}
	return e.apply(e.sess.Feed(ch)), nil
}

func (e *tuigoEngine) FocusIn() *dbus.Error {
	return nil
}

func (e *tuigoEngine) FocusOut() *dbus.Error {
	e.apply(e.sess.Reset())
	return nil
}

func (e *tuigoEngine) Reset() *dbus.Error {
	e.apply(e.sess.Reset())
	return nil
}

func (e *tuigoEngine) Enable() *dbus.Error {
	e.reloadConfig()
	e.RegisterProperties(e.props())
	return nil
}

func (e *tuigoEngine) Disable() *dbus.Error {
	e.apply(e.sess.Reset())
	return nil
}

func (e *tuigoEngine) SetCapabilities(cap uint32) *dbus.Error { return nil }
func (e *tuigoEngine) SetCursorLocation(x, y, w, h int32) *dbus.Error {
	return nil
}
func (e *tuigoEngine) SetSurroundingText(text dbus.Variant, cursorIndex, anchorPos uint32) *dbus.Error {
	return nil
}
func (e *tuigoEngine) PageUp() *dbus.Error             { return nil }
func (e *tuigoEngine) PageDown() *dbus.Error           { return nil }
func (e *tuigoEngine) CursorUp() *dbus.Error           { return nil }
func (e *tuigoEngine) CursorDown() *dbus.Error         { return nil }
func (e *tuigoEngine) PropertyShow(string) *dbus.Error { return nil }
func (e *tuigoEngine) PropertyHide(string) *dbus.Error { return nil }
func (e *tuigoEngine) CandidateClicked(index, button, state uint32) *dbus.Error {
	return nil
}

func (e *tuigoEngine) PropertyActivate(propName string, propState uint32) *dbus.Error {
	switch propName {
	case "vietkey":
		if (propState == ibus.PROP_STATE_CHECKED) != e.viet {
			e.toggleViet()
		}
	case "im_telex":
		e.setIM(engine.ImTelex)
	case "im_vni":
		e.setIM(engine.ImVni)
	case "im_viqr":
		e.setIM(engine.ImViqr)
	case "im_msvi":
		e.setIM(engine.ImMsVi)
	case "im_telexvni":
		e.setIM(engine.ImTelexVni)
	case "open_settings":
		openSettings()
	}
	return nil
}

func (e *tuigoEngine) setIM(im int) {
	e.cfg.InputMethod = im
	e.cfg.Save()
	e.eng.SetInputMethod(im)
	e.RegisterProperties(e.props())
}

func (e *tuigoEngine) props() *ibus.PropList {
	vietState := uint32(ibus.PROP_STATE_UNCHECKED)
	if e.viet {
		vietState = ibus.PROP_STATE_CHECKED
	}
	radio := func(key, label string, active bool) *ibus.Property {
		st := uint32(ibus.PROP_STATE_UNCHECKED)
		if active {
			st = ibus.PROP_STATE_CHECKED
		}
		return ibus.NewProperty(key, ibus.PROP_TYPE_RADIO, label, "", label, true, true, st)
	}
	props := ibus.NewPropList(
		ibus.NewProperty("vietkey", ibus.PROP_TYPE_TOGGLE,
			"Tiếng Việt", "", "Bật/tắt gõ tiếng Việt", true, true, vietState),
		ibus.NewPropertyWithChild("input_method", ibus.PROP_TYPE_MENU,
			"Kiểu gõ", "", "Kiểu gõ", true, true, 0, *ibus.NewPropList(
				radio("im_telex", "Telex", e.cfg.InputMethod == engine.ImTelex),
				radio("im_vni", "VNI", e.cfg.InputMethod == engine.ImVni),
				radio("im_viqr", "VIQR", e.cfg.InputMethod == engine.ImViqr),
				radio("im_msvi", "MS Vi (AllCode)", e.cfg.InputMethod == engine.ImMsVi),
				radio("im_telexvni", "Telex + VNI", e.cfg.InputMethod == engine.ImTelexVni),
			)),
		ibus.NewProperty("sep", ibus.PROP_TYPE_SEPARATOR, "", "", "", false, true, 0),
		ibus.NewProperty("open_settings", ibus.PROP_TYPE_NORMAL,
			"Cài đặt Tui Gõ…", "", "Mở cửa sổ cài đặt", true, true, 0),
	)
	return props
}

func openSettings() {
	dir, err := os.Executable()
	if err != nil {
		return
	}
	bin := filepath.Join(filepath.Dir(dir), "tuigo-settings")
	if _, err := os.Stat(bin); err != nil {
		bin = "tuigo-settings" // fall back to PATH
	}
	exec.Command(bin).Start()
}

// -----------------------------------------------------------------------

var (
	ibusMode   = flag.Bool("ibus", false, "spawned by ibus-daemon (normal mode)")
	standalone = flag.Bool("standalone", false, "register component on the running ibus-daemon (dev mode)")
	execPath   = flag.String("exec", "", "exec path to embed in the registered component (standalone mode)")
)

func component() *ibus.Component {
	self, err := os.Executable()
	if err != nil || *execPath != "" {
		self = *execPath
	}
	icon := filepath.Join(filepath.Dir(self), "tuigo.png")
	c := ibus.NewComponent(
		"org.freedesktop.IBus.TuiGo",
		"Vietnamese input method (Telex/VNI/VIQR)",
		"0.0.1", "GPL-3.0", "Tui Go", "",
		self+" --ibus", "",
	)
	ed := ibus.FullEngineDesc(
		"tuigo", "Tui Gõ (Vietnamese)",
		"Bộ gõ tiếng Việt — Telex, VNI, VIQR",
		"vi", "GPL-3.0", "Tui Go", icon, "us",
		0, "", "VI", filepath.Join(filepath.Dir(self), "tuigo-settings"),
		"", "", "0.0.1", "",
	)
	c.AddEngine(ed)
	return c
}

func main() {
	flag.Parse()
	fmt.Fprintf(os.Stderr, "tuigo-ibus: starting (ibus=%v standalone=%v)\n",
		*ibusMode, *standalone)

	bus, err := ibus.NewBusE()
	if err != nil {
		fmt.Fprintln(os.Stderr, "tuigo-ibus:", err)
		os.Exit(1)
	}
	defer bus.Close()

	if err := bus.RequestNameChecked("org.freedesktop.IBus.TuiGo", 0); err != nil {
		fmt.Fprintln(os.Stderr, "tuigo-ibus:", err)
		os.Exit(1)
	}

	if *standalone {
		call := bus.RegisterComponent(component())
		if call.Err != nil {
			fmt.Fprintln(os.Stderr, "tuigo-ibus: register component:", call.Err)
			os.Exit(1)
		}
	}

	conn := bus.GetDbusConn()
	ibus.NewFactory(conn, func(conn *dbus.Conn, engineName string) dbus.ObjectPath {
		n := atomic.AddUint64(&engineCounter, 1)
		path := dbus.ObjectPath("/org/freedesktop/IBus/Engine/" + strconv.FormatUint(n, 10))
		ibus.PublishEngine(conn, path, newTuigoEngine(conn, path))
		fmt.Fprintf(os.Stderr, "tuigo-ibus: created engine %s for %q\n", path, engineName)
		return path
	})

	fmt.Fprintln(os.Stderr, "tuigo-ibus: factory ready")
	<-bus.Done()
}
