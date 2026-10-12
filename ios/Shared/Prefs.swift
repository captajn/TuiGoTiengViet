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
    static let keyVietOn = "viet_on"

    static var macroURL: URL? {
        FileManager.default
            .containerURL(forSecurityApplicationGroupIdentifier: suiteName)?
            .appendingPathComponent("macro.txt")
    }

    static func vietOn() -> Bool {
        defaults.object(forKey: keyVietOn) as? Bool ?? true
    }

    static func doubleSpacePeriod() -> Bool {
        defaults.object(forKey: keyDoubleSpace) as? Bool ?? true
    }

    /// Apply persisted options onto a live engine.
    static func apply(to engine: Engine) {
        let d = defaults
        let im = d.object(forKey: keyIM) as? Int ?? ImTelex
        switch im {
        case ImVni, ImTelexSimple, ImTelexVni:
            engine.setInputMethod(im)
        default:
            engine.setInputMethod(ImTelex)
        }
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
