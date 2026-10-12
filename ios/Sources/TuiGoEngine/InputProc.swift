// inputproc.cpp port — key event classification + keymap handling

struct KeyEvent {
    var keyCode: Int = 0
    var evType: Int = VneNormal
    var chType: Int = 0
    var vnSym: VnLexiName = VnlNonVnChar
    var tone: Int = -1
}

final class InputProcessor {
    private(set) var im = ImTelex
    var keyMap = [Int](repeating: VneNormal, count: 256)

    init() {
        setIM(ImTelex)
    }

    func setIM(_ im: Int) {
        self.im = im
        switch im {
        case ImVni: useBuiltIn(vniMethodMapping)
        case ImViqr: useBuiltIn(viqrMethodMapping)
        case ImMsVi: useBuiltIn(msViMethodMapping)
        case ImTelexSimple: useBuiltIn(simpleTelexMethodMapping)
        case ImTelexVni: useBuiltIn(telexMethodMapping + vniMethodMapping)
        default:
            self.im = ImTelex
            useBuiltIn(telexMethodMapping)
        }
    }

    func setIMKeyMap(_ keyMap: [Int]) {
        im = ImUsrKeymap
        for i in 0..<256 { self.keyMap[i] = keyMap[i] }
    }

    private func useBuiltIn(_ map: [KeyMapping]) {
        for i in 0..<256 { keyMap[i] = VneNormal }
        for m in map {
            keyMap[m.key] = m.action
            if m.action < VneCount {
                if isLower(m.key) {
                    keyMap[toUpper(m.key)] = m.action
                } else if isUpper(m.key) {
                    keyMap[toLower(m.key)] = m.action
                }
            }
        }
    }

    func keyCodeToEvent(_ keyCode: Int) -> KeyEvent {
        var ev = KeyEvent(keyCode: keyCode, evType: VneNormal, tone: -1)
        if keyCode > 255 {
            ev.vnSym = isoToVnLexi(keyCode)
            ev.chType = ev.vnSym == VnlNonVnChar ? UkcNonVn : UkcVn
        } else {
            ev.chType = ukcMap[keyCode]
            ev.evType = keyMap[keyCode]
            if ev.evType >= VneTone0 && ev.evType <= VneTone5 {
                ev.tone = ev.evType - VneTone0
            }
            if ev.evType >= VneCount {
                ev.chType = UkcVn
                ev.vnSym = ev.evType - VneCount
                ev.evType = VneMapChar
            } else {
                ev.vnSym = isoToVnLexi(keyCode)
            }
        }
        return ev
    }

    func keyCodeToSymbol(_ keyCode: Int) -> KeyEvent {
        var ev = KeyEvent(keyCode: keyCode, evType: VneNormal, tone: -1)
        ev.vnSym = isoToVnLexi(keyCode)
        if keyCode > 255 {
            ev.chType = ev.vnSym == VnlNonVnChar ? UkcNonVn : UkcVn
        } else {
            ev.chType = ukcMap[keyCode]
        }
        return ev
    }

    func getCharType(_ keyCode: Int) -> Int {
        if keyCode > 255 {
            return isoToVnLexi(keyCode) == VnlNonVnChar ? UkcNonVn : UkcVn
        }
        return ukcMap[keyCode]
    }
}

func isLower(_ c: Int) -> Bool { c >= "a".ord && c <= "z".ord }
func isUpper(_ c: Int) -> Bool { c >= "A".ord && c <= "Z".ord }
func toLower(_ c: Int) -> Int { isUpper(c) ? c + 32 : c }
func toUpper(_ c: Int) -> Int { isLower(c) ? c - 32 : c }
func isAlpha(_ c: Int) -> Bool { isLower(c) || isUpper(c) }
