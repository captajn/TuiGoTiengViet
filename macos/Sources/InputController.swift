import Cocoa
import InputMethodKit
import TuiGoEngine

// One InputController per client context (per text field). Mirrors the
// Windows hook logic: printable chars feed the engine, navigation and
// modifier chords reset it, Ctrl+Shift / Alt+Z toggle Vietnamese.
@objc(TuiGoInputController)
final class InputController: IMKInputController {

    private var session: Session!
    private var cfg = TuiGoConfig.load()

    // Ctrl+Shift clean-chord state (fires on release, like Windows)
    private var ctrlHeld = false
    private var shiftHeld = false
    private var chordArmed = false
    private var chordClean = false
    private var noneHeldAt = Date.distantPast
    private var firstReleaseAt = Date.distantPast

    override init(server: IMKServer!, delegate: Any!, client inputClient: Any!) {
        super.init(server: server, delegate: delegate, client: inputClient)
        reloadConfig()
    }

    private func reloadConfig() {
        cfg = TuiGoConfig.load()
        if session == nil {
            session = Session(viet: cfg.vietKey)
        } else {
            apply(session.setViet(cfg.vietKey))
        }
        session.autoCap = cfg.autoCap
        session.eng.setOptions(cfg.engineOptions())
        session.eng.setInputMethod(cfg.inputMethod)
        _ = session.eng.loadMacroFile(TuiGoConfig.macroURL.path)
        _ = session.eng.loadKeyMapFile(TuiGoConfig.keymapURL.path)
    }

    override func activateServer(_ sender: Any!) {
        super.activateServer(sender)
        reloadConfig() // pick up settings/macro edits
    }

    override func deactivateServer(_ sender: Any!) {
        apply(session.reset())
        super.deactivateServer(sender)
    }

    override func commitComposition(_ sender: Any!) -> Bool {
        apply(session.reset())
        return true
    }

    override func recognizedEvents(_ sender: Any!) -> Int {
        return Int(NSEvent.EventTypeMask([.keyDown, .flagsChanged]).rawValue)
    }

    // MARK: key handling

    private static let resetKeyCodes: Set<UInt16> = [
        36, 76,          // Return, KP_Enter
        48,              // Tab
        53,              // Escape
        123, 124, 125, 126, // arrows
        115, 119, 116, 121, // Home, End, PageUp, PageDown
        117, 114,        // forward Delete, Insert/Help
        122, 120, 99, 118, 96, 97, 98, 100, 101, 109, 103, 111, // F1..F12
    ]

    override func handle(_ event: NSEvent!, client sender: Any!) -> Bool {
        guard let event = event else { return false }
        if event.type == .flagsChanged {
            handleFlags(event)
            return false
        }
        guard event.type == .keyDown else { return false }

        let mods = event.modifierFlags
            .intersection(.deviceIndependentFlagsMask)

        // hotkeys — checked before the shortcut gate
        if cfg.toggles.contains("alt+z") && mods.contains(.option)
            && !mods.contains(.control) && !mods.contains(.command)
            && event.charactersIgnoringModifiers?.lowercased() == "z" {
            toggleViet()
            return true
        }

        // any non-modifier key dirties a pending chord
        chordClean = chordClean && !(ctrlHeld || shiftHeld)

        // Ctrl/Option/Command shortcuts bypass the engine, word invalidated.
        if mods.contains(.control) || mods.contains(.option) || mods.contains(.command) {
            apply(session.reset())
            return false
        }

        if Self.resetKeyCodes.contains(event.keyCode) {
            if event.keyCode == 36 || event.keyCode == 76 {
                apply(session.enterPressed())
            } else {
                apply(session.reset())
            }
            return false
        }

        if event.keyCode == 51 { // Backspace
            return apply(session.backspace())
        }

        session.eng.shiftPressed = mods.contains(.shift)
        session.eng.capsLockOn = mods.contains(.capsLock)

        guard let chars = event.characters, !chars.isEmpty else {
            return false // dead-key step etc: pass through, keep word
        }
        // feed each produced codepoint (normally one)
        var consumed = true
        for scalar in chars.unicodeScalars {
            if !apply(session.feed(scalar.value)) {
                consumed = false
            }
        }
        return consumed
    }

    private func handleFlags(_ event: NSEvent) {
        let m = event.modifierFlags.intersection(.deviceIndependentFlagsMask)
        let ctrl = m.contains(.control), shift = m.contains(.shift)
        if m.contains(.option) || m.contains(.command) {
            chordClean = false
            chordArmed = false
        }
        if !ctrlHeld && !shiftHeld {
            noneHeldAt = Date()
            chordClean = true
        }
        if ctrl && shift && !(ctrlHeld && shiftHeld) && chordClean
            && Date().timeIntervalSince(noneHeldAt) < 0.8 {
            chordArmed = true
        }
        if chordArmed && !ctrl && !shift {
            chordArmed = false
            if chordClean && Date().timeIntervalSince(firstReleaseAt) < 0.7 {
                toggleViet()
            }
        } else if chordArmed && !(ctrl && shift) {
            firstReleaseAt = Date()
        }
        ctrlHeld = ctrl
        shiftHeld = shift
    }

    private func toggleViet() {
        cfg.vietKey = !cfg.vietKey
        try? cfg.save()
        apply(session.setViet(cfg.vietKey))
    }

    // MARK: applying session actions to the client

    private func apply(_ a: Session.Action) -> Bool {
        let client = self.client() as? IMKTextInput
        let notFound = NSRange(location: NSNotFound, length: 0)
        if !a.commit.isEmpty || a.preedit.isEmpty {
            client?.setMarkedText("", selectionRange: NSRange(location: 0, length: 0),
                                  replacementRange: notFound)
        }
        if !a.commit.isEmpty {
            client?.insertText(Self.u16(a.commit), replacementRange: notFound)
        }
        if !a.preedit.isEmpty {
            let str = Self.u16(a.preedit)
            client?.setMarkedText(str,
                selectionRange: NSRange(location: str.utf16.count, length: 0),
                replacementRange: notFound)
        }
        return !a.forward
    }

    private static func u16(_ units: [UInt16]) -> String {
        return String(utf16CodeUnits: units, count: units.count)
    }

    // MARK: input menu

    override func menu() -> NSMenu! {
        let menu = NSMenu()
        let vietItem = NSMenuItem(
            title: "Tiếng Việt", action: #selector(toggleFromMenu(_:)),
            keyEquivalent: "")
        vietItem.target = self
        vietItem.state = session.viet ? .on : .off
        menu.addItem(vietItem)

        let imMenu = NSMenu()
        for (label, im) in Self.inputMethods {
            let item = NSMenuItem(title: label,
                action: #selector(setIMFromMenu(_:)), keyEquivalent: "")
            item.target = self
            item.tag = im
            item.state = cfg.inputMethod == im ? .on : .off
            imMenu.addItem(item)
        }
        let imItem = NSMenuItem(title: "Kiểu gõ", action: nil, keyEquivalent: "")
        imItem.submenu = imMenu
        menu.addItem(imItem)

        menu.addItem(.separator())
        let settings = NSMenuItem(
            title: "Cài đặt Tui Gõ…",
            action: #selector(openSettingsFromMenu(_:)), keyEquivalent: "")
        settings.target = self
        menu.addItem(settings)
        return menu
    }

    private static let inputMethods: [(String, Int)] = [
        ("Telex", 0), ("VNI", 1), ("VIQR", 2),
        ("MS Vi (AllCode)", 3), ("Telex đơn giản", 5), ("Telex + VNI", 6),
    ]

    @objc private func toggleFromMenu(_ sender: Any?) { toggleViet() }

    @objc private func setIMFromMenu(_ sender: NSMenuItem) {
        cfg.inputMethod = sender.tag
        try? cfg.save()
        session.eng.setInputMethod(sender.tag)
    }

    @objc private func openSettingsFromMenu(_ sender: Any?) {
        AppDelegate.showSettings()
    }
}
