import Foundation

public func macroValidationError(_ text: String) -> String? {
    var keys = Set<String>()
    for (index, raw) in text.components(separatedBy: .newlines).enumerated() {
        let line = raw.replacingOccurrences(of: "\u{FEFF}", with: "").trimmingCharacters(in: .whitespacesAndNewlines)
        if line.isEmpty || line.hasPrefix(";") { continue }
        func error(_ message: String) -> String { "Dòng \(index + 1): \(message)" }
        guard let colon = line.firstIndex(of: ":"), colon != line.startIndex else { return error("dùng định dạng từ tắt:nội dung.") }
        let key = String(line[..<colon])
        if key.unicodeScalars.contains(where: { CharacterSet.whitespacesAndNewlines.union(.controlCharacters).contains($0) }) { return error("từ tắt không chứa khoảng trắng.") }
        if key.unicodeScalars.count >= MaxMacroKeyLen { return error("từ tắt tối đa 15 ký tự.") }
        if line[line.index(after: colon)...].trimmingCharacters(in: .whitespacesAndNewlines).isEmpty { return error("nội dung thay thế không được trống.") }
        if !keys.insert(key).inserted { return error("từ tắt '\(key)' bị trùng.") }
        if keys.count > MaxMacroItems { return error("tối đa 1024 mục gõ tắt.") }
    }
    return nil
}

// Convert one codepoint to StdVnChar (Unicode -> VNSTANDARD conversion path).
private func codePointToStdVn(_ r: UInt32) -> StdVnChar {
    if r < 0x10000, let sc = uniToStdVn[UInt16(r)] {
        return sc
    }
    return StdVnChar(r)
}

func textToStdVn(_ s: String) -> [StdVnChar] {
    s.unicodeScalars.flatMap { scalar -> [StdVnChar] in
        if scalar.value > 0xFFFF { return String(scalar).utf16.map { StdVnChar($0) } }
        return [codePointToStdVn(scalar.value)]
    }
}

extension MacroStore {
    // item format "key:text"
    @discardableResult
    func addItem(_ item: String) -> Bool {
        guard let pos = item.firstIndex(of: ":") else { return false }
        var key = String(item[item.startIndex..<pos])
        if key.unicodeScalars.count > MaxMacroKeyLen - 1 {
            let cut = key.unicodeScalars.index(key.unicodeScalars.startIndex, offsetBy: MaxMacroKeyLen - 1)
            key = String(key.unicodeScalars[key.unicodeScalars.startIndex..<cut])
        }
        let text = String(item[item.index(after: pos)...])
        if items.count >= MaxMacroItems { return false }
        items[textToStdVn(key)] = textToStdVn(text)
        return true
    }
}

extension Engine {
    // UTF-8 macro file; lines "key:text", ';' comments,
    // header ";[DO NOT DELETE THIS LINE]***version=n" tolerated.
    func loadMacroFileImpl(_ fname: String) -> Bool {
        guard let raw = try? String(contentsOfFile: fname, encoding: .utf8) else {
            return false
        }
        macStore.items.removeAll()
        for var line in raw.split(separator: "\n", omittingEmptySubsequences: false).map(String.init) {
            // strip UTF-8 BOM on first line if present
            if line.hasPrefix("\u{FEFF}") { line.removeFirst() }
            line = line.trimmingCharacters(in: .whitespacesAndNewlines)
            if line.isEmpty || line.hasPrefix(";") { continue }
            macStore.addItem(line)
        }
        return true
    }

    // In-memory variant for app bundles / tests.
    public func loadMacroText(_ text: String) {
        macStore.items.removeAll()
        for var line in text.split(separator: "\n", omittingEmptySubsequences: false).map(String.init) {
            if line.hasPrefix("\u{FEFF}") { line.removeFirst() }
            line = line.trimmingCharacters(in: .whitespacesAndNewlines)
            if line.isEmpty || line.hasPrefix(";") { continue }
            macStore.addItem(line)
        }
    }
}
