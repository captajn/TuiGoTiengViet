import Foundation

// User keymap file format (usrkeymap.cpp):
//
//	X = ActionLabel    (single char key = label)
//	; comment
private let ukEvLabels: [String: Int] = [
    "Tone0": VneTone0,
    "Tone1": VneTone1,
    "Tone2": VneTone2,
    "Tone3": VneTone3,
    "Tone4": VneTone4,
    "Tone5": VneTone5,
    "Roof-All": VneRoofAll,
    "Roof-A": VneRoofA,
    "Roof-E": VneRoofE,
    "Roof-O": VneRoofO,
    "Hook-Bowl": VneHookAll,
    "Hook-UO": VneHookUO,
    "Hook-U": VneHookU,
    "Hook-O": VneHookO,
    "Bowl": VneBowl,
    "D-Mark": VneDd,
    "Telex-W": VneTelexW,
    "Escape": VneEscChar,
    "DD": VneCount + VnlDD,
    "dd": VneCount + Vnl_dd,
    "A^": VneCount + VnlAr,
    "a^": VneCount + Vnl_ar,
    "A(": VneCount + VnlAb,
    "a(": VneCount + Vnl_ab,
    "E^": VneCount + VnlEr,
    "e^": VneCount + Vnl_er,
    "O^": VneCount + VnlOr,
    "o^": VneCount + Vnl_or,
    "O+": VneCount + VnlOh,
    "o+": VneCount + Vnl_oh,
    "U+": VneCount + VnlUh,
    "u+": VneCount + Vnl_uh,
]

// Parses UniKey keymap.txt content into a 256-entry keymap.
func parseKeyMap(_ text: String) -> [Int] {
    var keyMap = [Int](repeating: VneNormal, count: 256)
    var assigned = Set<Int>()

    for raw in text.split(separator: "\n", omittingEmptySubsequences: false) {
        var line = String(raw)
        if let idx = line.firstIndex(of: ";") {
            line = String(line[line.startIndex..<idx])
        }
        let parts = line.split(separator: "=", maxSplits: 1).map {
            $0.trimmingCharacters(in: .whitespaces)
        }
        guard parts.count == 2 else { continue }
        let name = parts[0], value = parts[1]
        // Go: len(name)==1 means one byte (ASCII); same check via utf8 view
        guard name.utf8.count == 1, let action = ukEvLabels[value] else { continue }
        let c = Int(name.utf8.first!)
        if assigned.contains(c) { continue } // already assigned, don't accept this map
        keyMap[c] = action
        assigned.insert(c)
        if action < VneCount {
            keyMap[toLower(c)] = action
            keyMap[toUpper(c)] = action
        }
    }
    return keyMap
}

// Parses a UniKey keymap.txt file into a 256-entry keymap (nil if unreadable).
func loadKeyMapFileImpl(_ fname: String) -> [Int]? {
    guard let text = try? String(contentsOfFile: fname, encoding: .utf8) else {
        return nil
    }
    return parseKeyMap(text)
}
