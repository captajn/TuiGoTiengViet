import Foundation

// Convert one codepoint to StdVnChar (Unicode -> VNSTANDARD conversion path).
private func codePointToStdVn(_ r: UInt32) -> StdVnChar {
    if r < 0x10000, let sc = uniToStdVn[UInt16(r)] {
        return sc
    }
    return StdVnChar(r)
}

func textToStdVn(_ s: String) -> [StdVnChar] {
    s.unicodeScalars.map { codePointToStdVn($0.value) }
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
