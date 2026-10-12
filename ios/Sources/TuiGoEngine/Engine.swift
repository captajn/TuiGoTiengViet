// Swift port of UkEngine (ukengine.cpp), specialized to Unicode (UTF-16) output.
// Mirrors the Go port in app/engine/engine.go and the Kotlin port in android/.

public final class Options {
    public var freeMarking = 0
    public var modernStyle = 0
    public var macroEnabled = 0
    public var useUnicodeClipboard = 0
    public var alwaysMacro = 0
    public var strictSpellCheck = 0
    public var useIME = 0
    public var spellCheckEnabled = 0
    public var autoNonVnRestore = 0

    public init() {}

    func copyFrom(_ o: Options) {
        freeMarking = o.freeMarking; modernStyle = o.modernStyle
        macroEnabled = o.macroEnabled; useUnicodeClipboard = o.useUnicodeClipboard
        alwaysMacro = o.alwaysMacro; strictSpellCheck = o.strictSpellCheck
        useIME = o.useIME; spellCheckEnabled = o.spellCheckEnabled
        autoNonVnRestore = o.autoNonVnRestore
    }
}

// Result of processing one key.
public struct Result {
    public var consumed = false   // swallow the key (engine produced output)
    public var backs = 0          // number of UTF-16 units to backspace
    public var out = [UInt16]()   // UTF-16 output to insert
    public var keyOut = false     // out contains raw key codes, not characters

    public init() {}
}

final class KeyBufEntry {
    var ev = KeyEvent()
    var converted = false
}

final class WordInfo {
    // info for word ending at this position
    var form = 0                        // vnw_*
    var c1Offset = 0
    var vOffset = 0
    var c2Offset = 0

    // union { VowelSeq vseq; ConSeq cseq } — single storage, cast on read
    var seq = 0

    // info for current symbol
    var caps = 0
    var tone = 0
    var vnSym: VnLexiName = 0           // canonical symbol, after caps & tone removed; -1 for non-Vn
    var keyCode = 0

    func vseq() -> VowelSeq { seq }
    func cseq() -> ConSeq { seq }
    func setVseq(_ v: VowelSeq) { seq = v }
    func setCseq(_ c: ConSeq) { seq = c }
}

final class MacroStore {
    // key: StdVnChar list (internal codes — any bijective key works)
    var items = [[StdVnChar]: [StdVnChar]]()

    func lookup(_ key: [StdVnChar]) -> [StdVnChar]? {
        items.isEmpty ? nil : items[key]
    }
}

public final class Engine {
    private var buffer = (0..<MaxUkEngine).map { _ in WordInfo() }
    private var keyStrokes = (0..<MaxUkEngine).map { _ in KeyBufEntry() }

    private var changePos = 0
    private var backs = 0
    private let bufSize = MaxUkEngine
    private var current = -1
    private var singleMode = false
    private let keyBufSize = MaxUkEngine
    private var keyCurrent = -1
    private var toEscape = false
    private var outputWritten = false
    private var reverted = false
    private var keyRestored = false
    private var keyRestoring = false
    private var outType = OutChar

    // keyboard state (replaces m_keyCheckFunc)
    public var shiftPressed = false
    public var capsLockOn = false

    private var out = [UInt16]()

    private let opts = Options()
    private var vietKey = true
    private let input = InputProcessor()
    let macStore = MacroStore()

    private var usedAsMapChar = false // static flag in processTelexW

    public init() {
        // UniKey defaults: strict Telex (FreeMarking=0) can't reach back
        // through 'i' — "khoiwr" would stay literal instead of "khởi".
        opts.freeMarking = 1
        opts.modernStyle = 1
        opts.spellCheckEnabled = 1
        opts.macroEnabled = 1
    }

    public func setInputMethod(_ im: Int) { input.setIM(im) }
    public func setKeyMap(_ m: [Int]) { input.setIMKeyMap(m) }
    public func setVietKey(_ v: Bool) { vietKey = v }
    public func setOptions(_ o: Options) { opts.copyFrom(o) }
    public func getOptions() -> Options { opts }
    public func setSingleMode() { singleMode = true }
    public func atWordBeginning() -> Bool { current < 0 || buffer[current].form == VnwEmpty }

    private func reset() {
        current = -1
        keyCurrent = -1
        singleMode = false
        toEscape = false
    }

    public func resetBuf() { reset() }

    public func pass(_ keyCode: Int) {
        var ev = input.keyCodeToEvent(keyCode)
        processAppend(&ev)
    }

    // ----------------------------------------------------------
    private func getSeqSteps(_ first: Int, _ last: Int) -> Int {
        if last < first { return 0 }
        return last - first + 1 // Unicode output: 1 entry = 1 UTF-16 unit
    }

    private func markChange(_ pos: Int) {
        if pos < changePos {
            backs += getSeqSteps(pos, changePos - 1)
            changePos = pos
        }
    }

    private func stdCharToUTF16(_ sc: StdVnChar) -> UInt16? {
        if sc == InvalidStdChar { return nil }
        if sc >= 0 && sc < VnStdCharOffset { return UInt16(truncatingIfNeeded: sc) }
        let idx = sc - VnStdCharOffset
        if idx >= 0 && idx < TotalVnChars { return unicodeTable[idx] }
        return nil
    }

    // ----------------------------------------------------------
    private func writeOutput() {
        out.removeAll(keepingCapacity: true)
        guard changePos <= current else { // Go loop is empty in this case
            outputWritten = true
            return
        }
        for i in changePos...current {
            let stdChar: StdVnChar
            if buffer[i].vnSym != VnlNonVnChar {
                var sc = buffer[i].vnSym + VnStdCharOffset
                if buffer[i].caps != 0 { sc -= 1 }
                if buffer[i].tone != 0 { sc += buffer[i].tone * 2 }
                stdChar = sc
            } else {
                stdChar = isoToStdVnChar(buffer[i].keyCode)
            }
            if let c = stdCharToUTF16(stdChar) { out.append(c) }
        }
        outputWritten = true
    }

    // ----------------------------------------------------------
    // isValidCV / isValidVC / isValidCVC
    // ----------------------------------------------------------
    private func isValidCV(_ c: ConSeq, _ v: VowelSeq) -> Bool {
        if c == CsNil || v == VsNil { return true }
        let vInfo = vSeqList[v]

        if (c == CsGi && vInfo.v[0] == Vnl_i) ||
            (c == CsQu && vInfo.v[0] == Vnl_u) {
            return false // gi doesn't go with i, qu doesn't go with u
        }

        if c == CsK {
            // k can only go with the following vowel sequences
            let kVseq: [VowelSeq] = [
                Vs_e, Vs_i, Vs_y, Vs_er, Vs_eo, Vs_eu,
                Vs_eru, Vs_ia, Vs_ie, Vs_ier, Vs_ieu, Vs_ieru,
            ]
            return kVseq.contains(v)
        }
        return true
    }

    private func isValidVC(_ v: VowelSeq, _ c: ConSeq) -> Bool {
        if v == VsNil || c == CsNil { return true }
        if vSeqList[v].conSuffix == 0 { return false }
        if !cSeqList[c].suffix { return false }
        return vcPairSet.contains(v * 64 + c)
    }

    private func isValidCVC(_ c1: ConSeq, _ v: VowelSeq, _ c2: ConSeq) -> Bool {
        if v == VsNil { return c1 == CsNil || c2 != CsNil }
        if c1 == CsNil { return isValidVC(v, c2) }
        if c2 == CsNil { return isValidCV(c1, v) }
        let okCV = isValidCV(c1, v)
        let okVC = isValidVC(v, c2)
        if okCV && okVC { return true }
        if !okVC {
            // exceptions: vc fails but cvc passes
            // quyn, quynh
            if c1 == CsQu && v == Vs_y && (c2 == CsN || c2 == CsNh) { return true }
            // gieng, gie^ng
            if c1 == CsGi && (v == Vs_e || v == Vs_er) && (c2 == CsN || c2 == CsNg) { return true }
        }
        return false
    }

    // ----------------------------------------------------------
    private func getTonePosition(_ vs: VowelSeq, _ terminated: Bool) -> Int {
        if vs < 0 || vs >= vSeqList.count {
            return 0 // invalid seq — C++ OOB-reads silently; must not crash
        }
        let info = vSeqList[vs]
        if info.length == 1 { return 0 }
        if info.roofPos != -1 { return info.roofPos }
        if info.hookPos != -1 {
            if vs == Vs_uhoh || vs == Vs_uhohi || vs == Vs_uhohu { //u+o+, u+o+u, u+o+i
                return 1
            }
            return info.hookPos
        }
        if info.length == 3 { return 1 }
        if opts.modernStyle != 0 && (vs == Vs_oa || vs == Vs_oe || vs == Vs_uy) {
            return 1
        }
        if terminated { return 0 }
        return 1
    }

    // ----------------------------------------------------------
    private func processRoof(_ ev: inout KeyEvent) -> Int {
        if !vietKey || current < 0 || buffer[current].vOffset < 0 {
            return processAppend(&ev)
        }

        let target: VnLexiName
        switch ev.evType {
        case VneRoofA: target = Vnl_ar
        case VneRoofE: target = Vnl_er
        case VneRoofO: target = Vnl_or
        default: target = VnlNonVnChar
        }

        var newVs: VowelSeq
        var chPos: Int
        var roofRemoved = false

        let vEnd = current - buffer[current].vOffset
        let vs = buffer[vEnd].vseq()
        let vStart = vEnd - (vSeqList[vs].length - 1)
        let curTonePos = vStart + getTonePosition(vs, vEnd == current)
        let tone = buffer[curTonePos].tone

        var doubleChangeUO = false
        if vs == Vs_uho || vs == Vs_uhoh || vs == Vs_uhoi || vs == Vs_uhohi {
            //special cases: u+o+ -> uo^, u+o -> uo^, u+o+i -> uo^i, u+oi -> uo^i
            newVs = lookupVSeq(Vnl_u, Vnl_or, vSeqList[vs].v[2])
            doubleChangeUO = true
        } else {
            newVs = vSeqList[vs].withRoof
        }

        let pInfo: VowelSeqInfo

        if newVs == VsNil {
            if vSeqList[vs].roofPos == -1 {
                return processAppend(&ev) //roof is not applicable
            }

            //a roof already exists -> undo roof
            let curCh = buffer[vStart + vSeqList[vs].roofPos].vnSym
            if target != VnlNonVnChar && curCh != target {
                return processAppend(&ev) //specific roof and the roof character don't match
            }

            let newCh: VnLexiName
            if curCh == Vnl_ar {
                newCh = Vnl_a
            } else if curCh == Vnl_er {
                newCh = Vnl_e
            } else {
                newCh = Vnl_o
            }
            chPos = vStart + vSeqList[vs].roofPos

            if opts.freeMarking == 0 && chPos != current {
                return processAppend(&ev)
            }

            markChange(chPos)
            buffer[chPos].vnSym = newCh

            if vSeqList[vs].length == 3 {
                newVs = lookupVSeq(buffer[vStart].vnSym, buffer[vStart + 1].vnSym, buffer[vStart + 2].vnSym)
            } else if vSeqList[vs].length == 2 {
                newVs = lookupVSeq(buffer[vStart].vnSym, buffer[vStart + 1].vnSym, VnlNonVnChar)
            } else {
                newVs = lookupVSeq(buffer[vStart].vnSym, VnlNonVnChar, VnlNonVnChar)
            }

            pInfo = vSeqList[newVs]
            roofRemoved = true
        } else {
            pInfo = vSeqList[newVs]
            if target != VnlNonVnChar && pInfo.v[pInfo.roofPos] != target {
                return processAppend(&ev)
            }

            //check validity of new VC and CV
            var c1 = CsNil
            var c2 = CsNil
            if buffer[current].c1Offset != -1 {
                c1 = buffer[current - buffer[current].c1Offset].cseq()
            }
            if buffer[current].c2Offset != -1 {
                c2 = buffer[current - buffer[current].c2Offset].cseq()
            }
            if !isValidCVC(c1, newVs, c2) {
                return processAppend(&ev)
            }

            chPos = doubleChangeUO ? vStart : vStart + pInfo.roofPos
            if opts.freeMarking == 0 && chPos != current {
                return processAppend(&ev)
            }
            markChange(chPos)
            if doubleChangeUO {
                buffer[vStart].vnSym = Vnl_u
                buffer[vStart + 1].vnSym = Vnl_or
            } else {
                buffer[chPos].vnSym = pInfo.v[pInfo.roofPos]
            }
        }

        for i in 0..<pInfo.length { //update sub-sequences
            buffer[vStart + i].setVseq(pInfo.sub[i])
        }

        //check if tone re-position is needed
        let newTonePos = vStart + getTonePosition(newVs, vEnd == current)
        if curTonePos != newTonePos && tone != 0 {
            markChange(newTonePos)
            buffer[newTonePos].tone = tone
            markChange(curTonePos)
            buffer[curTonePos].tone = 0
        }

        if roofRemoved {
            singleMode = false
            processAppend(&ev)
            reverted = true
        }

        return 1
    }

    // ----------------------------------------------------------
    // can only be called from processHook
    // ----------------------------------------------------------
    private func processHookWithUO(_ ev: inout KeyEvent) -> Int {
        var newVs: VowelSeq
        var hookRemoved = false
        let removeWithUndo = true

        if opts.freeMarking == 0 && buffer[current].vOffset != 0 {
            return processAppend(&ev)
        }

        let vEnd = current - buffer[current].vOffset
        let vs = buffer[vEnd].vseq()
        let vStart = vEnd - (vSeqList[vs].length - 1)
        let v = vSeqList[vs].v
        let curTonePos = vStart + getTonePosition(vs, vEnd == current)
        let tone = buffer[curTonePos].tone

        switch ev.evType {
        case VneHookU:
            if v[0] == Vnl_u {
                newVs = vSeqList[vs].withHook
                markChange(vStart)
                buffer[vStart].vnSym = Vnl_uh
            } else { // v[0] = vnl_uh, -> uo
                newVs = lookupVSeq(Vnl_u, Vnl_o, v[2])
                markChange(vStart)
                buffer[vStart].vnSym = Vnl_u
                buffer[vStart + 1].vnSym = Vnl_o
                hookRemoved = true
            }
        case VneHookO:
            if v[1] == Vnl_o || v[1] == Vnl_or {
                if vEnd == current && vSeqList[vs].length == 2 &&
                    buffer[current].form == VnwCV && buffer[current - 2].cseq() == CsTh {
                    // o|o^ -> o+
                    newVs = vSeqList[vs].withHook
                    markChange(vStart + 1)
                    buffer[vStart + 1].vnSym = Vnl_oh
                } else {
                    newVs = lookupVSeq(Vnl_uh, Vnl_oh, v[2])
                    if v[0] == Vnl_u {
                        markChange(vStart)
                        buffer[vStart].vnSym = Vnl_uh
                        buffer[vStart + 1].vnSym = Vnl_oh
                    } else {
                        markChange(vStart + 1)
                        buffer[vStart + 1].vnSym = Vnl_oh
                    }
                }
            } else { // v[1] = vnl_oh, -> uo
                newVs = lookupVSeq(Vnl_u, Vnl_o, v[2])
                if v[0] == Vnl_uh {
                    markChange(vStart)
                    buffer[vStart].vnSym = Vnl_u
                    buffer[vStart + 1].vnSym = Vnl_o
                } else {
                    markChange(vStart + 1)
                    buffer[vStart + 1].vnSym = Vnl_o
                }
                hookRemoved = true
            }
        default: //vneHookAll, vneHookUO:
            if v[0] == Vnl_u {
                if v[1] == Vnl_o || v[1] == Vnl_or {
                    //uo -> uo+ if prefixed by "th"
                    if (vs == Vs_uo || vs == Vs_uor) && vEnd == current &&
                        buffer[current].form == VnwCV && buffer[current - 2].cseq() == CsTh {
                        newVs = Vs_uoh
                        markChange(vStart + 1)
                        buffer[vStart + 1].vnSym = Vnl_oh
                    } else {
                        //uo -> u+o+
                        newVs = vSeqList[vs].withHook
                        markChange(vStart)
                        buffer[vStart].vnSym = Vnl_uh
                        newVs = vSeqList[newVs].withHook
                        buffer[vStart + 1].vnSym = Vnl_oh
                    }
                } else { //uo+ -> u+o+
                    newVs = vSeqList[vs].withHook
                    markChange(vStart)
                    buffer[vStart].vnSym = Vnl_uh
                }
            } else { //v[0] == vnl_uh
                if v[1] == Vnl_o { // u+o -> u+o+
                    newVs = vSeqList[vs].withHook
                    markChange(vStart + 1)
                    buffer[vStart + 1].vnSym = Vnl_oh
                } else { //v[1] == vnl_oh, u+o+ -> uo
                    newVs = lookupVSeq(Vnl_u, Vnl_o, v[2])
                    markChange(vStart)
                    buffer[vStart].vnSym = Vnl_u
                    buffer[vStart + 1].vnSym = Vnl_o
                    hookRemoved = true
                }
            }
        }

        let p = vSeqList[newVs]
        for i in 0..<p.length { //update sub-sequences
            buffer[vStart + i].setVseq(p.sub[i])
        }

        //check if tone re-position is needed
        let newTonePos = vStart + getTonePosition(newVs, vEnd == current)
        if curTonePos != newTonePos && tone != 0 {
            markChange(newTonePos)
            buffer[newTonePos].tone = tone
            markChange(curTonePos)
            buffer[curTonePos].tone = 0
        }

        if hookRemoved && removeWithUndo {
            singleMode = false
            processAppend(&ev)
            reverted = true
        }

        return 1
    }

    // ----------------------------------------------------------
    private func processHook(_ ev: inout KeyEvent) -> Int {
        if !vietKey || current < 0 || buffer[current].vOffset < 0 {
            return processAppend(&ev)
        }

        var newVs: VowelSeq
        var chPos: Int
        var hookRemoved = false
        let pInfo: VowelSeqInfo

        let vEnd = current - buffer[current].vOffset
        let vs = buffer[vEnd].vseq()

        let v = vSeqList[vs].v

        if vSeqList[vs].length > 1 &&
            ev.evType != VneBowl &&
            (v[0] == Vnl_u || v[0] == Vnl_uh) &&
            (v[1] == Vnl_o || v[1] == Vnl_oh || v[1] == Vnl_or) {
            return processHookWithUO(&ev)
        }

        let vStart = vEnd - (vSeqList[vs].length - 1)
        let curTonePos = vStart + getTonePosition(vs, vEnd == current)
        let tone = buffer[curTonePos].tone

        newVs = vSeqList[vs].withHook
        if newVs == VsNil {
            if vSeqList[vs].hookPos == -1 {
                return processAppend(&ev) //hook is not applicable
            }

            //a hook already exists -> undo hook
            let curCh = buffer[vStart + vSeqList[vs].hookPos].vnSym
            let newCh: VnLexiName
            if curCh == Vnl_ab {
                newCh = Vnl_a
            } else if curCh == Vnl_uh {
                newCh = Vnl_u
            } else {
                newCh = Vnl_o
            }
            chPos = vStart + vSeqList[vs].hookPos
            if opts.freeMarking == 0 && chPos != current {
                return processAppend(&ev)
            }

            switch ev.evType {
            case VneHookU:
                if curCh != Vnl_uh { return processAppend(&ev) }
            case VneHookO:
                if curCh != Vnl_oh { return processAppend(&ev) }
            case VneBowl:
                if curCh != Vnl_ab { return processAppend(&ev) }
            default:
                if ev.evType == VneHookUO && curCh == Vnl_ab { return processAppend(&ev) }
            }

            markChange(chPos)
            buffer[chPos].vnSym = newCh

            if vSeqList[vs].length == 3 {
                newVs = lookupVSeq(buffer[vStart].vnSym, buffer[vStart + 1].vnSym, buffer[vStart + 2].vnSym)
            } else if vSeqList[vs].length == 2 {
                newVs = lookupVSeq(buffer[vStart].vnSym, buffer[vStart + 1].vnSym, VnlNonVnChar)
            } else {
                newVs = lookupVSeq(buffer[vStart].vnSym, VnlNonVnChar, VnlNonVnChar)
            }

            pInfo = vSeqList[newVs]
            hookRemoved = true
        } else {
            pInfo = vSeqList[newVs]

            switch ev.evType {
            case VneHookU:
                if pInfo.v[pInfo.hookPos] != Vnl_uh { return processAppend(&ev) }
            case VneHookO:
                if pInfo.v[pInfo.hookPos] != Vnl_oh { return processAppend(&ev) }
            case VneBowl:
                if pInfo.v[pInfo.hookPos] != Vnl_ab { return processAppend(&ev) }
            default: //vneHook_uo, vneHookAll
                if ev.evType == VneHookUO && pInfo.v[pInfo.hookPos] == Vnl_ab { return processAppend(&ev) }
            }

            //check validity of new VC and CV
            var c1 = CsNil
            var c2 = CsNil
            if buffer[current].c1Offset != -1 {
                c1 = buffer[current - buffer[current].c1Offset].cseq()
            }
            if buffer[current].c2Offset != -1 {
                c2 = buffer[current - buffer[current].c2Offset].cseq()
            }
            if !isValidCVC(c1, newVs, c2) {
                return processAppend(&ev)
            }

            chPos = vStart + pInfo.hookPos
            if opts.freeMarking == 0 && chPos != current {
                return processAppend(&ev)
            }

            markChange(chPos)
            buffer[chPos].vnSym = pInfo.v[pInfo.hookPos]
        }

        for i in 0..<pInfo.length { //update sub-sequences
            buffer[vStart + i].setVseq(pInfo.sub[i])
        }

        //check if tone re-position is needed
        let newTonePos = vStart + getTonePosition(newVs, vEnd == current)
        if curTonePos != newTonePos && tone != 0 {
            markChange(newTonePos)
            buffer[newTonePos].tone = tone
            markChange(curTonePos)
            buffer[curTonePos].tone = 0
        }

        if hookRemoved {
            singleMode = false
            processAppend(&ev)
            reverted = true
        }

        return 1
    }

    // ----------------------------------------------------------
    private func processTone(_ ev: inout KeyEvent) -> Int {
        if current < 0 || !vietKey {
            return processAppend(&ev)
        }

        if buffer[current].form == VnwC &&
            (buffer[current].cseq() == CsGi || buffer[current].cseq() == CsGin) {
            let p = buffer[current].cseq() == CsGi ? current : current - 1
            if buffer[p].tone == 0 && ev.tone == 0 {
                return processAppend(&ev)
            }
            markChange(p)
            if buffer[p].tone == ev.tone {
                buffer[p].tone = 0
                singleMode = false
                processAppend(&ev)
                reverted = true
                return 1
            }
            buffer[p].tone = ev.tone
            return 1
        }

        if buffer[current].vOffset < 0 {
            return processAppend(&ev)
        }

        let vEnd = current - buffer[current].vOffset
        let vs = buffer[vEnd].vseq()
        let info = vSeqList[vs]
        if opts.spellCheckEnabled != 0 && opts.freeMarking == 0 && info.complete == 0 {
            return processAppend(&ev)
        }

        if buffer[current].form == VnwVC || buffer[current].form == VnwCVC {
            let cs = buffer[current].cseq()
            if (cs == CsC || cs == CsCh || cs == CsP || cs == CsT) &&
                (ev.tone == 2 || ev.tone == 3 || ev.tone == 4) {
                return processAppend(&ev) // c, ch, p, t suffixes don't allow ` ? ~
            }
        }

        let toneOffset = getTonePosition(vs, vEnd == current)
        let tonePos = vEnd - (info.length - 1) + toneOffset
        if buffer[tonePos].tone == 0 && ev.tone == 0 {
            return processAppend(&ev)
        }

        if buffer[tonePos].tone == ev.tone {
            markChange(tonePos)
            buffer[tonePos].tone = 0
            singleMode = false
            processAppend(&ev)
            reverted = true
            return 1
        }

        markChange(tonePos)
        buffer[tonePos].tone = ev.tone
        return 1
    }

    // ----------------------------------------------------------
    private func processDd(_ ev: inout KeyEvent) -> Int {
        if !vietKey || current < 0 {
            return processAppend(&ev)
        }

        var pos: Int

        // we want to allow dd even in non-vn sequence, because dd is used a lot in abbreviation
        // we allow dd only if preceding character is not a vowel
        if buffer[current].form == VnwNonVn &&
            buffer[current].vnSym == Vnl_d &&
            (current - 1 < 0 || buffer[current - 1].vnSym == VnlNonVnChar || !isVnVowel(buffer[current - 1].vnSym)) {
            singleMode = true
            pos = current
            markChange(pos)
            buffer[pos].setCseq(CsDd)
            buffer[pos].vnSym = Vnl_dd
            buffer[pos].form = VnwC
            buffer[pos].c1Offset = 0
            buffer[pos].c2Offset = -1
            buffer[pos].vOffset = -1
            return 1
        }

        if buffer[current].c1Offset < 0 {
            return processAppend(&ev)
        }

        pos = current - buffer[current].c1Offset
        if opts.freeMarking == 0 && pos != current {
            return processAppend(&ev)
        }

        if buffer[pos].cseq() == CsD {
            markChange(pos)
            buffer[pos].setCseq(CsDd)
            buffer[pos].vnSym = Vnl_dd
            return 1
        }

        if buffer[pos].cseq() == CsDd {
            //undo dd
            markChange(pos)
            buffer[pos].setCseq(CsD)
            buffer[pos].vnSym = Vnl_d
            singleMode = false
            processAppend(&ev)
            reverted = true
            return 1
        }

        return processAppend(&ev)
    }

    // ----------------------------------------------------------
    private func processMapChar(_ ev: inout KeyEvent) -> Int {
        if capsLockOn {
            ev.vnSym = changeCase(ev.vnSym)
        }

        let ret0 = processAppend(&ev)
        if !vietKey {
            return ret0
        }

        if current >= 0 && buffer[current].form != VnwEmpty &&
            buffer[current].form != VnwNonVn {
            return 1
        }

        if current < 0 {
            return 0
        }

        // mapChar doesn't apply
        current -= 1
        let entry = buffer[current]

        var undo = false
        // test if undo is needed
        if entry.form != VnwEmpty && entry.form != VnwNonVn {
            var prevSym = entry.vnSym
            if entry.caps != 0 {
                prevSym -= 1
            }
            if prevSym == ev.vnSym {
                if entry.form != VnwC {
                    let vEnd = current - entry.vOffset
                    let vs = buffer[vEnd].vseq()
                    let vStart = vEnd - vSeqList[vs].length + 1
                    let curTonePos = vStart + getTonePosition(vs, vEnd == current)
                    let tone = buffer[curTonePos].tone
                    markChange(current)
                    current -= 1

                    //check if tone position is needed
                    if tone != 0 && current >= 0 &&
                        (buffer[current].form == VnwV || buffer[current].form == VnwCV) {
                        let newVs = buffer[current].vseq()
                        let newTonePos = vStart + getTonePosition(newVs, true)
                        if newTonePos != curTonePos {
                            markChange(newTonePos)
                            buffer[newTonePos].tone = tone
                            markChange(curTonePos)
                            buffer[curTonePos].tone = 0
                        }
                    }
                } else {
                    markChange(current)
                    current -= 1
                }
                undo = true
            }
        }

        ev.evType = VneNormal
        ev.chType = input.getCharType(ev.keyCode)
        ev.vnSym = isoToVnLexi(ev.keyCode)
        let ret = processAppend(&ev)
        if undo {
            singleMode = false
            reverted = true
            return 1
        }
        return ret
    }

    // ----------------------------------------------------------
    private func processTelexW(_ ev: inout KeyEvent) -> Int {
        if !vietKey {
            return processAppend(&ev)
        }

        var ret: Int

        if usedAsMapChar {
            ev.evType = VneMapChar
            ev.vnSym = isUpper(ev.keyCode & 0xFF) ? VnlUh : Vnl_uh
            if capsLockOn {
                ev.vnSym = changeCase(ev.vnSym)
            }
            ev.chType = UkcVn
            ret = processMapChar(&ev)
            if ret == 0 {
                if current >= 0 {
                    current -= 1
                }
                usedAsMapChar = false
                ev.evType = VneHookAll
                return processHook(&ev)
            }
            return ret
        }

        ev.evType = VneHookAll
        usedAsMapChar = false
        ret = processHook(&ev)
        if ret == 0 {
            if current >= 0 {
                current -= 1
            }
            ev.evType = VneMapChar
            ev.vnSym = isUpper(ev.keyCode & 0xFF) ? VnlUh : Vnl_uh
            if capsLockOn {
                ev.vnSym = changeCase(ev.vnSym)
            }
            ev.chType = UkcVn
            usedAsMapChar = true
            return processMapChar(&ev)
        }
        return ret
    }

    // ----------------------------------------------------------
    @discardableResult
    private func processAppend(_ ev: inout KeyEvent) -> Int {
        switch ev.chType {
        case UkcReset:
            if ev.keyCode == EnterChar {
                if opts.macroEnabled != 0 && macroMatch(&ev) {
                    return 1
                }
            }
            reset()
            return 0
        case UkcWordBreak:
            singleMode = false
            return processWordEnd(&ev)
        case UkcVn:
            if isVnVowel(ev.vnSym) {
                let v = stdVnNoTone[vnToLower(ev.vnSym)]
                if current >= 0 && buffer[current].form == VnwC &&
                    ((buffer[current].cseq() == CsQ && v == Vnl_u) ||
                        (buffer[current].cseq() == CsG && v == Vnl_i)) {
                    return appendConsonnant(&ev) //process u after q, i after g as consonnants
                }
                return appendVowel(&ev)
            }
            return appendConsonnant(&ev)
        default: // UkcNonVn (the UkcWordBreak arm in the Go switch is dead code)
            current += 1
            let entry = buffer[current]
            entry.form = VnwNonVn
            entry.c1Offset = -1
            entry.c2Offset = -1
            entry.vOffset = -1
            entry.keyCode = ev.keyCode
            entry.vnSym = vnToLower(ev.vnSym)
            entry.caps = entry.vnSym != ev.vnSym ? 1 : 0
            entry.tone = 0
            return 0
        }
    }

    // ----------------------------------------------------------
    private func appendVowel(_ ev: inout KeyEvent) -> Int {
        current += 1
        let entry = buffer[current]

        let lowerSym = vnToLower(ev.vnSym)
        let canSym = stdVnNoTone[lowerSym]

        entry.vnSym = canSym
        entry.caps = lowerSym != ev.vnSym ? 1 : 0
        entry.tone = (lowerSym - canSym) / 2
        entry.keyCode = ev.keyCode

        if current == 0 || !vietKey {
            entry.form = VnwV
            entry.c1Offset = -1
            entry.c2Offset = -1
            entry.vOffset = 0
            entry.setVseq(lookupVSeq(canSym, VnlNonVnChar, VnlNonVnChar))

            if !vietKey || isAlpha(entry.keyCode) {
                return 0
            }
            markChange(current)
            return 1
        }

        let prev = buffer[current - 1]
        var vs: VowelSeq
        var newVs: VowelSeq
        var cs: ConSeq
        var prevTonePos: Int
        var tone: Int
        var newTone: Int
        var tonePos: Int
        var newTonePos: Int

        switch prev.form {
        case VnwEmpty:
            entry.form = VnwV
            entry.c1Offset = -1
            entry.c2Offset = -1
            entry.vOffset = 0
            newVs = lookupVSeq(canSym, VnlNonVnChar, VnlNonVnChar)
            entry.setVseq(newVs)

        case VnwNonVn, VnwCVC, VnwVC:
            entry.form = VnwNonVn
            entry.c1Offset = -1
            entry.c2Offset = -1
            entry.vOffset = -1

        case VnwV, VnwCV:
            vs = prev.vseq()
            prevTonePos = (current - 1) - (vSeqList[vs].length - 1) + getTonePosition(vs, true)
            tone = buffer[prevTonePos].tone

            if lowerSym != canSym && tone != 0 { //new sym has a tone, but there's already a preceding tone
                newVs = VsNil
            } else if vSeqList[vs].length == 3 {
                newVs = VsNil
            } else if vSeqList[vs].length == 2 {
                newVs = lookupVSeq(vSeqList[vs].v[0], vSeqList[vs].v[1], canSym)
            } else {
                newVs = lookupVSeq(vSeqList[vs].v[0], canSym, VnlNonVnChar)
            }

            if newVs != VsNil && prev.form == VnwCV {
                cs = buffer[current - 1 - prev.c1Offset].cseq()
                if !isValidCV(cs, newVs) {
                    newVs = VsNil
                }
            }

            if newVs == VsNil {
                entry.form = VnwNonVn
                entry.c1Offset = -1
                entry.c2Offset = -1
                entry.vOffset = -1
            } else {
                entry.form = prev.form
                entry.c1Offset = prev.form == VnwCV ? prev.c1Offset + 1 : -1
                entry.c2Offset = -1
                entry.vOffset = 0
                entry.setVseq(newVs)
                entry.tone = 0

                newTone = (lowerSym - canSym) / 2
                if tone == 0 {
                    if newTone != 0 {
                        tone = newTone
                        tonePos = getTonePosition(newVs, true) + ((current - 1) - vSeqList[vs].length + 1)
                        markChange(tonePos)
                        buffer[tonePos].tone = tone
                        return 1
                    }
                } else {
                    newTonePos = getTonePosition(newVs, true) + ((current - 1) - vSeqList[vs].length + 1)
                    if newTonePos != prevTonePos {
                        markChange(prevTonePos)
                        buffer[prevTonePos].tone = 0
                        markChange(newTonePos)
                        if newTone != 0 {
                            tone = newTone
                        }
                        buffer[newTonePos].tone = tone
                        return 1
                    }
                    if newTone != 0 && newTone != tone {
                        tone = newTone
                        markChange(prevTonePos)
                        buffer[prevTonePos].tone = tone
                        return 1
                    }
                }
            }

        case VnwC:
            newVs = lookupVSeq(canSym, VnlNonVnChar, VnlNonVnChar)
            cs = prev.cseq()
            if !isValidCV(cs, newVs) {
                entry.form = VnwNonVn
                entry.c1Offset = -1
                entry.c2Offset = -1
                entry.vOffset = -1
            } else {
                entry.form = VnwCV
                entry.c1Offset = 1
                entry.c2Offset = -1
                entry.vOffset = 0
                entry.setVseq(newVs)

                if cs == CsGi && prev.tone != 0 {
                    if entry.tone == 0 {
                        entry.tone = prev.tone
                    }
                    markChange(current - 1)
                    prev.tone = 0
                    return 1
                }
            }

        default:
            break
        }

        if isAlpha(entry.keyCode) {
            return 0
        }
        markChange(current)
        return 1
    }

    // ----------------------------------------------------------
    private func appendConsonnant(_ ev: inout KeyEvent) -> Int {
        var complexEvent = false
        current += 1
        let entry = buffer[current]

        let lowerSym = vnToLower(ev.vnSym)

        entry.vnSym = lowerSym
        entry.caps = lowerSym != ev.vnSym ? 1 : 0
        entry.keyCode = ev.keyCode
        entry.tone = 0

        if current == 0 || !vietKey {
            entry.form = VnwC
            entry.c1Offset = 0
            entry.c2Offset = -1
            entry.vOffset = -1
            entry.setCseq(lookupCSeq(lowerSym, VnlNonVnChar, VnlNonVnChar))
            return 0
        }

        var cs: ConSeq
        var newCs: ConSeq
        var c1: ConSeq
        var vs: VowelSeq
        var newVs: VowelSeq
        var isValid: Bool

        let prev = buffer[current - 1]

        switch prev.form {
        case VnwNonVn:
            entry.form = VnwNonVn
            entry.c1Offset = -1
            entry.c2Offset = -1
            entry.vOffset = -1
            return 0
        case VnwEmpty:
            entry.form = VnwC
            entry.c1Offset = 0
            entry.c2Offset = -1
            entry.vOffset = -1
            entry.setCseq(lookupCSeq(lowerSym, VnlNonVnChar, VnlNonVnChar))
            return 0
        case VnwV, VnwCV:
            vs = prev.vseq()
            newVs = vs
            if vs == Vs_uoh || vs == Vs_uho {
                newVs = Vs_uhoh
            }

            c1 = CsNil
            if prev.c1Offset != -1 {
                c1 = buffer[current - 1 - prev.c1Offset].cseq()
            }

            newCs = lookupCSeq(lowerSym, VnlNonVnChar, VnlNonVnChar)
            isValid = isValidCVC(c1, newVs, newCs)

            if isValid {
                //check u+o -> u+o+
                if vs == Vs_uho {
                    markChange(current - 1)
                    prev.vnSym = Vnl_oh
                    prev.setVseq(Vs_uhoh)
                    complexEvent = true
                } else if vs == Vs_uoh {
                    markChange(current - 2)
                    buffer[current - 2].vnSym = Vnl_uh
                    buffer[current - 2].setVseq(Vs_uh)
                    prev.setVseq(Vs_uhoh)
                    complexEvent = true
                }

                if prev.form == VnwV {
                    entry.form = VnwVC
                    entry.c1Offset = -1
                    entry.c2Offset = 0
                    entry.vOffset = 1
                } else { //prev == vnw_cv
                    entry.form = VnwCVC
                    entry.c1Offset = prev.c1Offset + 1
                    entry.c2Offset = 0
                    entry.vOffset = 1
                }
                entry.setCseq(newCs)

                //reposition tone if needed
                let oldIdx = (current - 1) - (vSeqList[vs].length - 1) + getTonePosition(vs, true)
                if buffer[oldIdx].tone != 0 {
                    let newIdx = (current - 1) - (vSeqList[newVs].length - 1) + getTonePosition(newVs, false)
                    if newIdx != oldIdx {
                        markChange(newIdx)
                        buffer[newIdx].tone = buffer[oldIdx].tone
                        markChange(oldIdx)
                        buffer[oldIdx].tone = 0
                        return 1
                    }
                }
            } else {
                entry.form = VnwNonVn
                entry.c1Offset = -1
                entry.c2Offset = -1
                entry.vOffset = -1
            }

            return complexEvent ? 1 : 0
        default: // VnwC, VnwVC, VnwCVC
            cs = prev.cseq()
            if cs < 0 || cs >= cSeqList.count {
                // prev is a consonant entry with no real seq (bare h/j/w…) —
                // C++ reads CSeqList[-1] out of bounds here; must not crash
                newCs = CsNil
            } else if cSeqList[cs].length == 3 {
                newCs = CsNil
            } else if cSeqList[cs].length == 2 {
                newCs = lookupCSeq(cSeqList[cs].c[0], cSeqList[cs].c[1], lowerSym)
            } else {
                newCs = lookupCSeq(cSeqList[cs].c[0], lowerSym, VnlNonVnChar)
            }

            if newCs != CsNil && (prev.form == VnwVC || prev.form == VnwCVC) {
                // Check CVC combination
                c1 = CsNil
                if prev.c1Offset != -1 {
                    c1 = buffer[current - 1 - prev.c1Offset].cseq()
                }

                let vIdx = (current - 1) - prev.vOffset
                vs = buffer[vIdx].vseq()
                isValid = isValidCVC(c1, vs, newCs)
                if !isValid {
                    newCs = CsNil
                }
            }

            if newCs == CsNil {
                entry.form = VnwNonVn
                entry.c1Offset = -1
                entry.c2Offset = -1
                entry.vOffset = -1
            } else {
                if prev.form == VnwC {
                    entry.form = VnwC
                    entry.c1Offset = 0
                    entry.c2Offset = -1
                    entry.vOffset = -1
                } else if prev.form == VnwVC {
                    entry.form = VnwVC
                    entry.c1Offset = -1
                    entry.c2Offset = 0
                    entry.vOffset = prev.vOffset + 1
                } else { //vnw_cvc
                    entry.form = VnwCVC
                    entry.c1Offset = prev.c1Offset + 1
                    entry.c2Offset = 0
                    entry.vOffset = prev.vOffset + 1
                }
                entry.setCseq(newCs)
            }
            return 0
        }
    }

    // ----------------------------------------------------------
    private func processEscChar(_ ev: inout KeyEvent) -> Int {
        if vietKey && current >= 0 &&
            buffer[current].form != VnwEmpty && buffer[current].form != VnwNonVn {
            toEscape = true
        }
        return processAppend(&ev)
    }

    // ----------------------------------------------------------
    // This can be called only after other processing have been done.
    // The new event is supposed to be put into m_buffer already
    // ----------------------------------------------------------
    private func processNoSpellCheck(_ ev: inout KeyEvent) -> Int {
        let entry = buffer[current]
        if isVnVowel(entry.vnSym) {
            entry.form = VnwV
            entry.vOffset = 0
            entry.setVseq(lookupVSeq(entry.vnSym, VnlNonVnChar, VnlNonVnChar))
            entry.c1Offset = -1
            entry.c2Offset = -1
        } else {
            entry.form = VnwC
            entry.c1Offset = 0
            entry.c2Offset = -1
            entry.vOffset = -1
            entry.setCseq(lookupCSeq(entry.vnSym, VnlNonVnChar, VnlNonVnChar))
        }

        if ev.evType == VneNormal &&
            ((entry.keyCode >= "a".ord && entry.keyCode <= "z".ord) ||
                (entry.keyCode >= "A".ord && entry.keyCode <= "Z".ord)) {
            return 0
        }
        markChange(current)
        return 1
    }

    // ----------------------------------------------------------
    // main entry point
    // ----------------------------------------------------------
    private func process(_ keyCode: Int) -> Result {
        prepareBuffer()
        backs = 0
        changePos = current + 1
        out.removeAll(keepingCapacity: true)
        outputWritten = false
        reverted = false
        keyRestored = false
        keyRestoring = false
        outType = OutChar

        var ev = input.keyCodeToEvent(keyCode)

        var ret: Int
        if !toEscape {
            ret = dispatch(&ev)
        } else {
            toEscape = false
            if current < 0 || ev.evType == VneNormal || ev.evType == VneEscChar {
                ret = processAppend(&ev)
            } else {
                current -= 1
                processAppend(&ev)
                markChange(current) //this will assign m_backs to 1 and mark the character for output
                ret = 1
            }
        }

        if vietKey &&
            current >= 0 && buffer[current].form == VnwNonVn &&
            ev.chType == UkcVn &&
            (opts.spellCheckEnabled == 0 || singleMode) {
            //The spell check has failed, but because we are in non-spellcheck mode,
            //we consider the new character as the beginning of a new word
            ret = processNoSpellCheck(&ev)
        }

        //we add key to key buffer only if that key has not caused a reset
        if current >= 0 {
            ev.chType = input.getCharType(ev.keyCode)
            keyCurrent += 1
            if keyCurrent < MaxUkEngine {
                keyStrokes[keyCurrent].ev = ev // struct copy — matches Go value semantics
                keyStrokes[keyCurrent].converted = ret != 0 && !keyRestored
            }
        }

        var res = Result()
        res.keyOut = outType == OutKey
        if ret == 0 {
            return res
        }

        res.backs = backs
        if !outputWritten {
            writeOutput()
        }
        res.out = out
        res.consumed = true
        return res
    }

    private func dispatch(_ ev: inout KeyEvent) -> Int {
        switch ev.evType {
        case VneRoofAll, VneRoofA, VneRoofE, VneRoofO:
            return processRoof(&ev)
        case VneHookAll, VneHookUO, VneHookU, VneHookO, VneBowl:
            return processHook(&ev)
        case VneDd:
            return processDd(&ev)
        case VneTone0, VneTone1, VneTone2, VneTone3, VneTone4, VneTone5:
            return processTone(&ev)
        case VneTelexW:
            return processTelexW(&ev)
        case VneMapChar:
            return processMapChar(&ev)
        case VneEscChar:
            return processEscChar(&ev)
        default: //vneNormal
            return processAppend(&ev)
        }
    }

    // ----------------------------------------------------------
    private func synchKeyStrokeBuffer() {
        //synchronize with key-stroke buffer
        if keyCurrent >= 0 {
            keyCurrent -= 1
        }
        if current >= 0 && buffer[current].form == VnwEmpty {
            //in character buffer, we have reached a word break,
            //so we also need to move key stroke pointer backward to corresponding word break
            while keyCurrent >= 0 && keyStrokes[keyCurrent].ev.chType != UkcWordBreak {
                keyCurrent -= 1
            }
        }
    }

    // ----------------------------------------------------------
    private func processBackspace() -> Result {
        var res = Result()
        res.keyOut = false
        out.removeAll(keepingCapacity: true)
        if !vietKey || current < 0 {
            return res
        }

        backs = 0
        changePos = current + 1
        markChange(current)

        if current == 0 ||
            buffer[current].form == VnwEmpty ||
            buffer[current].form == VnwNonVn ||
            buffer[current].form == VnwC ||
            buffer[current - 1].form == VnwC ||
            buffer[current - 1].form == VnwCVC ||
            buffer[current - 1].form == VnwVC {
            current -= 1
            res.backs = backs
            synchKeyStrokeBuffer()
            res.consumed = backs > 1
            return res
        }

        let vEnd = current - buffer[current].vOffset
        if vEnd < 0 {
            current -= 1
            res.backs = backs
            synchKeyStrokeBuffer()
            res.consumed = backs > 1
            return res
        }
        let vs = buffer[vEnd].vseq()
        let newVs = buffer[current - 1].vseq()
        if vs < 0 || vs >= vSeqList.count || newVs < 0 || newVs >= vSeqList.count {
            // cell carries no valid vowel seq (empty/nonVn) — C++ would silently
            // read out of bounds here; fall back to the plain decrement path
            current -= 1
            res.backs = backs
            synchKeyStrokeBuffer()
            res.consumed = backs > 1
            return res
        }
        let vStart = vEnd - vSeqList[vs].length + 1
        let curTonePos = vStart + getTonePosition(vs, vEnd == current)
        let newTonePos = vStart + getTonePosition(newVs, true)
        let tone = buffer[curTonePos].tone

        if tone == 0 || curTonePos == newTonePos ||
            (curTonePos == current && buffer[current].tone != 0) {
            current -= 1
            res.backs = backs
            synchKeyStrokeBuffer()
            res.consumed = backs > 1
            return res
        }

        markChange(newTonePos)
        buffer[newTonePos].tone = tone
        markChange(curTonePos)
        buffer[curTonePos].tone = 0
        current -= 1
        synchKeyStrokeBuffer()
        res.backs = backs
        writeOutput()
        res.out = out
        res.consumed = true
        return res
    }

    // ----------------------------------------------------
    // make sure there are at least 10 entries available
    // ----------------------------------------------------
    private func prepareBuffer() {
        //prepare symbol buffer
        if current >= 0 && current + 10 >= bufSize {
            // Get rid of at least half of the current entries
            // don't get rid from the middle of a word.
            var rid = current / 2
            while buffer[rid].form != VnwEmpty && rid < current {
                rid += 1
            }
            if rid == current {
                current = -1
            } else {
                rid += 1
                let seg = Array(buffer[rid...current])
                buffer.replaceSubrange(0..<(current + 1 - rid), with: seg)
                current -= rid
            }
        }

        //prepare key stroke buffer
        if keyCurrent > 0 && keyCurrent + 1 >= keyBufSize {
            // Get rid of at least half of the current entries
            let rid = keyCurrent / 2
            let seg = Array(keyStrokes[rid...keyCurrent])
            keyStrokes.replaceSubrange(0..<(keyCurrent + 1 - rid), with: seg)
            keyCurrent -= rid
        }
    }

    // ----------------------------------------------------
    private func macroMatch(_ ev: inout KeyEvent) -> Bool {
        if shiftPressed && (ev.keyCode == " ".ord || ev.keyCode == EnterChar) {
            return false
        }

        var pMacText: [StdVnChar]? = nil
        var key = [StdVnChar](repeating: 0, count: MaxMacroKeyLen + 1)
        var i = current
        var j: Int

        while i >= 0 && (current - i + 1) < MaxMacroKeyLen {
            while i >= 0 && buffer[i].form != VnwEmpty && (current - i + 1) < MaxMacroKeyLen {
                i -= 1
            }
            if i >= 0 && buffer[i].form != VnwEmpty {
                return false
            }

            if i >= 0 {
                if buffer[i].vnSym != VnlNonVnChar {
                    key[0] = buffer[i].vnSym + VnStdCharOffset
                    if buffer[i].caps != 0 {
                        key[0] -= 1
                    }
                    key[0] += buffer[i].tone * 2
                } else {
                    key[0] = buffer[i].keyCode
                }
            }

            j = i + 1
            while j <= current {
                if buffer[j].vnSym != VnlNonVnChar {
                    key[j - i] = buffer[j].vnSym + VnStdCharOffset
                    if buffer[j].caps != 0 {
                        key[j - i] -= 1
                    }
                    key[j - i] += buffer[j].tone * 2
                } else {
                    key[j - i] = buffer[j].keyCode
                }
                j += 1
            }
            let keyLen = current - i + 1
            //search macro table
            pMacText = macStore.lookup(Array(key[1..<keyLen]))
            if pMacText != nil {
                i += 1 //mark the position where change is needed
                break
            }
            if i >= 0 {
                pMacText = macStore.lookup(Array(key[0..<keyLen]))
                if pMacText != nil {
                    break
                }
            }
            i -= 1
        }

        guard let pMacText else {
            return false
        }

        markChange(i)
        out.removeAll(keepingCapacity: true)
        for sc in pMacText {
            if let c = stdCharToUTF16(sc) { out.append(c) }
        }

        //write the last input character
        let vnChar: StdVnChar = ev.vnSym != VnlNonVnChar
            ? ev.vnSym + VnStdCharOffset
            : ev.keyCode
        if let c = stdCharToUTF16(vnChar) { out.append(c) }
        let b = backs //store backs before calling reset
        reset()
        outputWritten = true
        backs = b
        return true
    }

    // ----------------------------------------------------
    private func doRestoreKeyStrokes() -> Bool {
        outType = OutKey
        if !lastWordHasVnMark() {
            return false
        }

        backs = 0
        changePos = current + 1

        var keyStart = keyCurrent
        var converted = false
        while keyStart >= 0 && keyStrokes[keyStart].ev.chType != UkcWordBreak {
            if keyStrokes[keyStart].converted {
                converted = true
            }
            keyStart -= 1
        }
        keyStart += 1

        if !converted {
            //no key stroke has been converted, so it doesn't make sense to restore key strokes
            return false
        }

        while current >= 0 && buffer[current].form != VnwEmpty {
            current -= 1
        }
        markChange(current + 1)

        out.removeAll(keepingCapacity: true)
        keyRestoring = true
        for i in keyStart...keyCurrent { // keyStart <= keyCurrent guaranteed: converted=true implies non-empty span

            out.append(UInt16(truncatingIfNeeded: keyStrokes[i].ev.keyCode))
            var ev = input.keyCodeToSymbol(keyStrokes[i].ev.keyCode)
            keyStrokes[i].converted = false
            processAppend(&ev)
        }
        keyRestoring = false

        return true
    }

    // --------------------------------------------------
    private func processWordEnd(_ ev: inout KeyEvent) -> Int {
        if opts.macroEnabled != 0 && macroMatch(&ev) {
            return 1
        }

        if opts.spellCheckEnabled == 0 || singleMode || current < 0 || keyRestoring {
            current += 1
            let entry = buffer[current]
            entry.form = VnwEmpty
            entry.c1Offset = -1
            entry.c2Offset = -1
            entry.vOffset = -1
            entry.keyCode = ev.keyCode
            entry.vnSym = vnToLower(ev.vnSym)
            entry.caps = entry.vnSym != ev.vnSym ? 1 : 0
            return 0
        }

        if opts.autoNonVnRestore != 0 && lastWordIsNonVn() {
            if doRestoreKeyStrokes() {
                keyRestored = true
                outputWritten = true
            }
        }

        current += 1
        let entry = buffer[current]
        entry.form = VnwEmpty
        entry.c1Offset = -1
        entry.c2Offset = -1
        entry.vOffset = -1
        entry.keyCode = ev.keyCode
        entry.vnSym = vnToLower(ev.vnSym)
        entry.caps = entry.vnSym != ev.vnSym ? 1 : 0

        if keyRestored {
            out.append(UInt16(truncatingIfNeeded: ev.keyCode))
            return 1
        }

        return 0
    }

    // -----------------------------------------------------------
    // Test if last word is a non-Vietnamese word
    // -----------------------------------------------------------
    private func lastWordIsNonVn() -> Bool {
        if current < 0 {
            return false
        }

        switch buffer[current].form {
        case VnwNonVn:
            // Abbreviation exception: an all-consonant word carrying a đ mark
            // (HĐ, CĐT, ĐH…) is valid Vietnamese shorthand — keep it as typed
            // instead of restoring the raw keystrokes.
            var hasDd = false
            var i = current
            while i >= 0 && buffer[i].form != VnwEmpty {
                let sym = buffer[i].vnSym
                if isVnVowel(sym) {
                    return true // has a vowel: a real (invalid) word — restore
                }
                if sym == Vnl_dd {
                    hasDd = true
                }
                i -= 1
            }
            return !hasDd
        case VnwEmpty, VnwC:
            return false
        case VnwV, VnwCV:
            return vSeqList[buffer[current].vseq()].complete == 0
        default: // VnwVC, VnwCVC
            let vIndex = current - buffer[current].vOffset
            let vs = buffer[vIndex].vseq()
            if vSeqList[vs].complete == 0 {
                return true
            }
            let cs = buffer[current].cseq()
            var c1 = CsNil
            if buffer[current].c1Offset != -1 {
                c1 = buffer[current - buffer[current].c1Offset].cseq()
            }

            if !isValidCVC(c1, vs, cs) {
                return true
            }

            let tonePos = (vIndex - vSeqList[vs].length + 1) + getTonePosition(vs, false)
            let tone = buffer[tonePos].tone
            if (cs == CsC || cs == CsCh || cs == CsP || cs == CsT) &&
                (tone == 2 || tone == 3 || tone == 4) {
                return true
            }
        }
        return false
    }

    // -----------------------------------------------------------
    // Test if last word has a Vietnamese mark, that is tones, decorators
    // -----------------------------------------------------------
    private func lastWordHasVnMark() -> Bool {
        var i = current
        while i >= 0 && buffer[i].form != VnwEmpty {
            let sym = buffer[i].vnSym
            if sym != VnlNonVnChar {
                if isVnVowel(sym) {
                    if buffer[i].tone != 0 {
                        return true
                    }
                }
                if sym != stdVnRootChar[sym] {
                    return true
                }
            }
            i -= 1
        }
        return false
    }

    // ----------------------------------------------------------
    // Public API matching the shell's Engine interface
    // ----------------------------------------------------------
    public func filter(_ keyCode: Int) -> Result {
        process(keyCode)
    }

    public func backspace() -> Result {
        processBackspace()
    }

    public func restoreKeyStrokes() -> Result {
        var res = Result()
        res.keyOut = true
        out.removeAll(keepingCapacity: true)
        if doRestoreKeyStrokes() {
            res.backs = backs
            res.out = out
            res.consumed = true
        }
        return res
    }

    @discardableResult
    public func loadMacroFile(_ path: String) -> Bool {
        loadMacroFileImpl(path)
    }

    @discardableResult
    public func loadKeyMapFile(_ path: String) -> Bool {
        guard let m = loadKeyMapFileImpl(path) else { return false }
        setKeyMap(m)
        return true
    }
}
