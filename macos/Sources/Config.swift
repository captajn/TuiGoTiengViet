import Foundation
import TuiGoEngine

// Shared config for the macOS port. Same key=value ini as Windows/Linux.
// Files live in ~/Library/Application Support/TuiGo/:
//   tuigo.ini, macro.txt, keymap.txt
struct TuiGoConfig {
    var vietKey = true
    var inputMethod = 0            // engine.ImTelex
    var spellCheck = true
    var modernStyle = true
    var freeMarking = true
    var macroEnabled = true
    var autoNonVnRestore = false
    var autoCap = false
    var toggles: [String] = ["ctrl+shift"]  // + "alt+z"
    var launchAtLogin = false      // LaunchAgent, not saved to ini

    static var configDir: URL {
        let base = FileManager.default.urls(
            for: .applicationSupportDirectory, in: .userDomainMask).first!
        return base.appendingPathComponent("TuiGo")
    }
    static var iniURL: URL    { configDir.appendingPathComponent("tuigo.ini") }
    static var macroURL: URL  { configDir.appendingPathComponent("macro.txt") }
    static var keymapURL: URL { configDir.appendingPathComponent("keymap.txt") }

    static func load() -> TuiGoConfig {
        var c = TuiGoConfig()
        guard let m = readIni(iniURL) else { return c }
        func int(_ k: String) -> Int? { m[k].flatMap(Int.init) }
        func bit(_ k: String, _ d: Bool) -> Bool {
            m[k].map { $0 == "1" } ?? d
        }
        if let v = int("VietKey") { c.vietKey = v != 0 }
        if let v = int("InputMethod") { c.inputMethod = v }
        c.spellCheck = bit("SpellCheck", c.spellCheck)
        c.modernStyle = bit("ModernStyle", c.modernStyle)
        c.freeMarking = bit("FreeMarking", c.freeMarking)
        c.macroEnabled = bit("MacroEnabled", c.macroEnabled)
        c.autoNonVnRestore = bit("AutoNonVnRestore", c.autoNonVnRestore)
        c.autoCap = bit("AutoCap", c.autoCap)
        if let v = m["ToggleKey"] {
            c.toggles = v.split(separator: ",")
                .map { $0.trimmingCharacters(in: .whitespaces).lowercased() }
                .filter { ["ctrl+shift", "alt+z"].contains($0) }
        }
        c.launchAtLogin = FileManager.default.fileExists(atPath: launchAgentPath)
        return c
    }

    func engineOptions() -> Options {
        let o = Options()
        o.spellCheckEnabled = spellCheck ? 1 : 0
        o.modernStyle = modernStyle ? 1 : 0
        o.freeMarking = freeMarking ? 1 : 0
        o.macroEnabled = macroEnabled ? 1 : 0
        o.autoNonVnRestore = autoNonVnRestore ? 1 : 0
        return o
    }

    func save() throws {
        try FileManager.default.createDirectory(
            at: Self.configDir, withIntermediateDirectories: true)
        func b(_ v: Bool) -> String { v ? "1" : "0" }
        var s = "[TuiGo]\n"
        s += "VietKey=\(b(vietKey))\n"
        s += "InputMethod=\(inputMethod)\n"
        s += "SpellCheck=\(b(spellCheck))\n"
        s += "ModernStyle=\(b(modernStyle))\n"
        s += "FreeMarking=\(b(freeMarking))\n"
        s += "MacroEnabled=\(b(macroEnabled))\n"
        s += "AutoNonVnRestore=\(b(autoNonVnRestore))\n"
        s += "AutoCap=\(b(autoCap))\n"
        s += "ToggleKey=\(toggles.joined(separator: ","))\n"
        let tmp = Self.iniURL.appendingPathExtension("tmp")
        try s.write(to: tmp, atomically: true, encoding: .utf8)
        _ = try? FileManager.default.removeItem(at: Self.iniURL)
        try FileManager.default.moveItem(at: tmp, to: Self.iniURL)
    }

    // LaunchAgent for "khởi động cùng hệ thống" — works without signing.
    static var launchAgentPath: String {
        NSHomeDirectory() + "/Library/LaunchAgents/com.tuigo.inputmethod.plist"
    }

    func setLaunchAtLogin(_ on: Bool) throws {
        let path = Self.launchAgentPath
        if !on {
            try? FileManager.default.removeItem(atPath: path)
            return
        }
        let exe = Bundle.main.executablePath ?? ""
        let plist = """
        <?xml version="1.0" encoding="UTF-8"?>
        <!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" \
        "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
        <plist version="1.0"><dict>
            <key>Label</key><string>com.tuigo.inputmethod</string>
            <key>ProgramArguments</key><array><string>\(exe)</string></array>
            <key>RunAtLoad</key><true/>
        </dict></plist>
        """
        try plist.write(toFile: path, atomically: true, encoding: .utf8)
    }

    static func readIni(_ url: URL) -> [String: String]? {
        guard let data = try? String(contentsOf: url, encoding: .utf8) else {
            return nil
        }
        var m: [String: String] = [:]
        for line in data.split(separator: "\n") {
            let l = line.trimmingCharacters(in: .whitespaces)
            if l.isEmpty || l.hasPrefix(";") || l.hasPrefix("#") || l.hasPrefix("[") {
                continue
            }
            if let eq = l.firstIndex(of: "=") {
                let k = l[l.startIndex..<eq].trimmingCharacters(in: .whitespaces)
                let v = l[l.index(after: eq)...].trimmingCharacters(in: .whitespaces)
                m[k] = v
            }
        }
        return m
    }
}
