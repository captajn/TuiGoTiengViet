import Foundation
import TuiGoEngine

// Settings shared between the host app and the keyboard extension via the
// App Group container (works in extensions without "Allow Full Access").
enum Prefs {
    static let suiteName = "group.com.tuigo"

    static var defaults: UserDefaults {
        UserDefaults(suiteName: suiteName) ?? .standard
    }

    // Same keys as android/Prefs.kt
    static let keyIM = "input_method"
    static let keyFreeMarking = "free_marking"
    static let keyModernStyle = "modern_style"
    static let keySpellCheck = "spell_check"
    static let keyMacro = "macro"
    static let keyAutoNonVn = "auto_nonvn_restore"
    static let keyDoubleSpace = "double_space_period"
    static let keyAutoCapitalize = "auto_capitalize"
    static let keyVietOn = "viet_on"
    static let keyNumberRow = "number_row"
    static let keyKeyHeight = "key_height"
    static let keyKeyGap = "key_gap"
    static let keyTheme = "key_theme"
    static let keyClipboardSuggest = "clipboard_suggest"
    static let keyClipboardHistory = "clipboard_history"
    static let keyClipboardMax = "clipboard_history_max"
    static let keyVietPerApp = "viet_per_app"

    static func keyHeight() -> Double {
        min(68, max(44, defaults.object(forKey: keyKeyHeight) as? Double ?? 52))
    }
    static func keyGap() -> Double {
        min(8, max(0, defaults.object(forKey: keyKeyGap) as? Double ?? 4))
    }
    static func numberRow() -> Bool {
        defaults.bool(forKey: keyNumberRow) || inputMethod() == ImVni
    }
    static func clipboardSuggest() -> Bool {
        defaults.object(forKey: keyClipboardSuggest) as? Bool ?? true
    }
    static func clipboardHistory() -> Bool {
        defaults.object(forKey: keyClipboardHistory) as? Bool ?? true
    }
    static func clipboardHistoryMax() -> Int {
        // Settings binds the slider through a Double — read via NSNumber.
        min(2000, max(5, (defaults.object(forKey: keyClipboardMax) as? NSNumber)?.intValue ?? 50))
    }
    static func inputMethod() -> Int {
        let saved = defaults.object(forKey: keyIM) as? Int ?? ImTelex
        return [ImTelex, ImVni, ImViqr].contains(saved) ? saved : ImTelex
    }
    static func migrateInputMethod() {
        // Called only by the containing app; keyboard access can be read-only.
        defaults.set(inputMethod(), forKey: keyIM)
    }

    static var sharedContainer: URL? {
        FileManager.default.containerURL(forSecurityApplicationGroupIdentifier: suiteName)
    }

    static var macroURL: URL? {
        sharedContainer?.appendingPathComponent("macro.txt")
    }

    static func vietOn() -> Bool {
        defaults.object(forKey: keyVietOn) as? Bool ?? true
    }

    static func doubleSpacePeriod() -> Bool {
        defaults.object(forKey: keyDoubleSpace) as? Bool ?? true
    }

    static func autoCapitalize() -> Bool {
        defaults.object(forKey: keyAutoCapitalize) as? Bool ?? true
    }

    // ---- per-app language memory ----------------------------------------------
    // iOS does not expose the host app's identity to keyboard extensions;
    // documentIdentifier is the closest stable key (one UUID per field/host).

    static func vietForApp(_ id: String?) -> Bool? {
        guard let id else { return nil }
        let map = defaults.dictionary(forKey: keyVietPerApp) as? [String: Bool] ?? [:]
        return map[id]
    }

    /// Remember the toggle per document (same contract as Android Prefs.setViet).
    /// keyVietOn is only the default for hosts without a recorded choice —
    /// toggling inside one app must not change other apps.
    static func setViet(_ id: String?, on: Bool) {
        guard let id else { return }
        var map = defaults.dictionary(forKey: keyVietPerApp) as? [String: Bool] ?? [:]
        map[id] = on
        if map.count > 300 { map = [id: on] }
        defaults.set(map, forKey: keyVietPerApp)
    }

    /// Apply persisted options onto a live engine.
    static func apply(to engine: Engine) {
        let d = defaults
        engine.setInputMethod(inputMethod())
        func on(_ key: String) -> Int {
            (d.object(forKey: key) as? Bool ?? true) ? 1 : 0
        }
        let o = Options()
        o.freeMarking = on(keyFreeMarking)
        o.modernStyle = on(keyModernStyle)
        o.spellCheckEnabled = on(keySpellCheck)
        o.macroEnabled = on(keyMacro)
        o.autoNonVnRestore = on(keyAutoNonVn)
        engine.setOptions(o)
        engine.setVietKey(vietOn())
    }
}
