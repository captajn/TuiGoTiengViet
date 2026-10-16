package com.tuigo.engine

// Kotlin port of UkEngine (ukengine.cpp), specialized to Unicode (UTF-16) output.
// Mirrors the Go port in app/engine/engine.go.

class Options(
    var freeMarking: Int = 0,
    var modernStyle: Int = 0,
    var macroEnabled: Int = 0,
    var useUnicodeClipboard: Int = 0,
    var alwaysMacro: Int = 0,
    var strictSpellCheck: Int = 0,
    var useIME: Int = 0,
    var spellCheckEnabled: Int = 0,
    var autoNonVnRestore: Int = 0,
) {
    fun copyFrom(o: Options) {
        freeMarking = o.freeMarking; modernStyle = o.modernStyle
        macroEnabled = o.macroEnabled; useUnicodeClipboard = o.useUnicodeClipboard
        alwaysMacro = o.alwaysMacro; strictSpellCheck = o.strictSpellCheck
        useIME = o.useIME; spellCheckEnabled = o.spellCheckEnabled
        autoNonVnRestore = o.autoNonVnRestore
    }
}

// Result of processing one key.
class Result {
    var consumed = false        // swallow the key (engine produced output)
    var backs = 0               // number of UTF-16 units to backspace
    var out = CharArray(0)      // UTF-16 output to insert
    var keyOut = false          // out contains raw key codes, not characters
}

internal class KeyBufEntry {
    val ev = KeyEvent()
    var converted = false
}

internal class WordInfo {
    // info for word ending at this position
    var form = 0                            // vnw_*
    var c1Offset = 0
    var vOffset = 0
    var c2Offset = 0

    // union { VowelSeq vseq; ConSeq cseq } — single storage, cast on read
    var seq = 0

    // info for current symbol
    var caps = 0
    var tone = 0
    var vnSym: VnLexiName = 0               // canonical symbol, after caps & tone removed; -1 for non-Vn
    var keyCode: Int = 0

    fun vseq(): VowelSeq = seq
    fun cseq(): ConSeq = seq
    fun setVseq(v: VowelSeq) { seq = v }
    fun setCseq(c: ConSeq) { seq = c }
}

internal class MacroStore {
    // key: StdVnChar list (internal codes — any bijective key works)
    val items = HashMap<List<StdVnChar>, List<StdVnChar>>()

    fun lookup(key: List<StdVnChar>): List<StdVnChar>? =
        if (items.isEmpty()) null else items[key]
}

class Engine {
    private val buffer = Array(MaxUkEngine) { WordInfo() }
    private val keyStrokes = Array(MaxUkEngine) { KeyBufEntry() }

    private var changePos = 0
    private var backs = 0
    private var bufSize = MaxUkEngine
    private var current = -1
    private var singleMode = false
    private var keyBufSize = MaxUkEngine
    private var keyCurrent = -1
    private var toEscape = false
    private var outputWritten = false
    private var reverted = false
    private var keyRestored = false
    private var keyRestoring = false
    private var outType = OutChar

    // keyboard state (replaces m_keyCheckFunc)
    var shiftPressed = false
    var capsLockOn = false
    var autoCapShifted = false

    private val out = ArrayList<Char>(MaxUkEngine)

    private val opts = Options()
    private var vietKey = true
    private val input = InputProcessor()
    internal val macStore = MacroStore()

    private var usedAsMapChar = false // static flag in processTelexW

    init {
        // UniKey defaults: strict Telex (FreeMarking=0) can't reach back
        // through 'i' — "khoiwr" would stay literal instead of "khởi".
        opts.freeMarking = 1
        opts.modernStyle = 1
        opts.spellCheckEnabled = 1
        opts.macroEnabled = 1
    }

    fun setInputMethod(im: Int) = input.setIM(im)
    fun setKeyMap(m: IntArray) = input.setIMKeyMap(m)
    fun setVietKey(v: Boolean) { vietKey = v }
    fun setOptions(o: Options) { opts.copyFrom(o) }
    fun getOptions(): Options = opts
    fun setSingleMode() { singleMode = true }
    fun atWordBeginning(): Boolean = current < 0 || buffer[current].form == VnwEmpty

    private fun reset() {
        current = -1
        keyCurrent = -1
        singleMode = false
        toEscape = false
    }

    fun resetBuf() = reset()

    fun pass(keyCode: Int) {
        val ev = input.keyCodeToEvent(keyCode)
        processAppend(ev)
    }

    // ----------------------------------------------------------
    private fun getSeqSteps(first: Int, last: Int): Int {
        if (last < first) return 0
        return last - first + 1 // Unicode output: 1 entry = 1 UTF-16 unit
    }

    private fun markChange(pos: Int) {
        if (pos < changePos) {
            backs += getSeqSteps(pos, changePos - 1)
            changePos = pos
        }
    }

    private fun stdCharToUTF16(sc: StdVnChar): Char? {
        if (sc == InvalidStdChar) return null
        if (sc in 0 until VnStdCharOffset) return Char(sc and 0xFFFF)
        val idx = sc - VnStdCharOffset
        if (idx in 0 until TotalVnChars) return Char(unicodeTable[idx])
        return null
    }

    // ----------------------------------------------------------
    private fun writeOutput() {
        out.clear()
        for (i in changePos..current) {
            val stdChar: StdVnChar
            if (buffer[i].vnSym != VnlNonVnChar) {
                var sc = buffer[i].vnSym + VnStdCharOffset
                if (buffer[i].caps != 0) sc--
                if (buffer[i].tone != 0) sc += buffer[i].tone * 2
                stdChar = sc
            } else {
                stdChar = isoToStdVnChar(buffer[i].keyCode)
            }
            stdCharToUTF16(stdChar)?.let { out.add(it) }
        }
        outputWritten = true
    }

    // ----------------------------------------------------------
    // isValidCV / isValidVC / isValidCVC
    // ----------------------------------------------------------
    private fun isValidCV(c: ConSeq, v: VowelSeq): Boolean {
        if (c == CsNil || v == VsNil) return true
        val vInfo = vSeqList[v]

        if ((c == CsGi && vInfo.v[0] == Vnl_i) ||
            (c == CsQu && vInfo.v[0] == Vnl_u)
        ) {
            return false // gi doesn't go with i, qu doesn't go with u
        }

        if (c == CsK) {
            // k can only go with the following vowel sequences
            val kVseq = intArrayOf(
                Vs_e, Vs_i, Vs_y, Vs_er, Vs_eo, Vs_eu,
                Vs_eru, Vs_ia, Vs_ie, Vs_ier, Vs_ieu, Vs_ieru
            )
            return v in kVseq
        }
        return true
    }

    private fun isValidVC(v: VowelSeq, c: ConSeq): Boolean {
        if (v == VsNil || c == CsNil) return true
        if (vSeqList[v].conSuffix == 0) return false
        if (!cSeqList[c].suffix) return false
        return vcPairSet.contains(v * 64 + c)
    }

    private fun isValidCVC(c1: ConSeq, v: VowelSeq, c2: ConSeq): Boolean {
        if (v == VsNil) return c1 == CsNil || c2 != CsNil
        if (c1 == CsNil) return isValidVC(v, c2)
        if (c2 == CsNil) return isValidCV(c1, v)
        val okCV = isValidCV(c1, v)
        val okVC = isValidVC(v, c2)
        if (okCV && okVC) return true
        if (!okVC) {
            // exceptions: vc fails but cvc passes
            // quyn, quynh
            if (c1 == CsQu && v == Vs_y && (c2 == CsN || c2 == CsNh)) return true
            // gieng, gie^ng
            if (c1 == CsGi && (v == Vs_e || v == Vs_er) && (c2 == CsN || c2 == CsNg)) return true
        }
        return false
    }

    // ----------------------------------------------------------
    private fun getTonePosition(vs: VowelSeq, terminated: Boolean): Int {
        if (vs < 0 || vs >= vSeqList.size) {
            return 0 // invalid seq — C++ OOB-reads silently; must not crash
        }
        val info = vSeqList[vs]
        if (info.length == 1) return 0
        if (info.roofPos != -1) return info.roofPos
        if (info.hookPos != -1) {
            if (vs == Vs_uhoh || vs == Vs_uhohi || vs == Vs_uhohu) { //u+o+, u+o+u, u+o+i
                return 1
            }
            return info.hookPos
        }
        if (info.length == 3) return 1
        if (opts.modernStyle != 0 && (vs == Vs_oa || vs == Vs_oe || vs == Vs_uy)) {
            return 1
        }
        if (terminated) return 0
        return 1
    }

    // ----------------------------------------------------------
    private fun processRoof(ev: KeyEvent): Int {
        if (!vietKey || current < 0 || buffer[current].vOffset < 0) {
            return processAppend(ev)
        }

        val target: VnLexiName = when (ev.evType) {
            VneRoofA -> Vnl_ar
            VneRoofE -> Vnl_er
            VneRoofO -> Vnl_or
            else -> VnlNonVnChar
        }

        var newVs: VowelSeq
        var changePos: Int
        var roofRemoved = false

        val vEnd = current - buffer[current].vOffset
        val vs = buffer[vEnd].vseq()
        val vStart = vEnd - (vSeqList[vs].length - 1)
        val curTonePos = vStart + getTonePosition(vs, vEnd == current)
        val tone = buffer[curTonePos].tone

        var doubleChangeUO = false
        if (vs == Vs_uho || vs == Vs_uhoh || vs == Vs_uhoi || vs == Vs_uhohi) {
            //special cases: u+o+ -> uo^, u+o -> uo^, u+o+i -> uo^i, u+oi -> uo^i
            newVs = lookupVSeq(Vnl_u, Vnl_or, vSeqList[vs].v[2])
            doubleChangeUO = true
        } else {
            newVs = vSeqList[vs].withRoof
        }

        val pInfo: VowelSeqInfo

        if (newVs == VsNil) {
            if (vSeqList[vs].roofPos == -1) {
                return processAppend(ev) //roof is not applicable
            }

            //a roof already exists -> undo roof
            val curCh = buffer[vStart + vSeqList[vs].roofPos].vnSym
            if (target != VnlNonVnChar && curCh != target) {
                return processAppend(ev) //specific roof and the roof character don't match
            }

            val newCh = when (curCh) {
                Vnl_ar -> Vnl_a
                Vnl_er -> Vnl_e
                else -> Vnl_o
            }
            changePos = vStart + vSeqList[vs].roofPos

            if (opts.freeMarking == 0 && changePos != current) {
                return processAppend(ev)
            }

            markChange(changePos)
            buffer[changePos].vnSym = newCh

            newVs = when (vSeqList[vs].length) {
                3 -> lookupVSeq(buffer[vStart].vnSym, buffer[vStart + 1].vnSym, buffer[vStart + 2].vnSym)
                2 -> lookupVSeq(buffer[vStart].vnSym, buffer[vStart + 1].vnSym, VnlNonVnChar)
                else -> lookupVSeq(buffer[vStart].vnSym, VnlNonVnChar, VnlNonVnChar)
            }

            pInfo = vSeqList[newVs]
            roofRemoved = true
        } else {
            pInfo = vSeqList[newVs]
            if (target != VnlNonVnChar && pInfo.v[pInfo.roofPos] != target) {
                return processAppend(ev)
            }

            //check validity of new VC and CV
            var c1 = CsNil
            var c2 = CsNil
            if (buffer[current].c1Offset != -1) {
                c1 = buffer[current - buffer[current].c1Offset].cseq()
            }
            if (buffer[current].c2Offset != -1) {
                c2 = buffer[current - buffer[current].c2Offset].cseq()
            }
            if (!isValidCVC(c1, newVs, c2)) {
                return processAppend(ev)
            }

            changePos = if (doubleChangeUO) vStart else vStart + pInfo.roofPos
            if (opts.freeMarking == 0 && changePos != current) {
                return processAppend(ev)
            }
            markChange(changePos)
            if (doubleChangeUO) {
                buffer[vStart].vnSym = Vnl_u
                buffer[vStart + 1].vnSym = Vnl_or
            } else {
                buffer[changePos].vnSym = pInfo.v[pInfo.roofPos]
            }
        }

        for (i in 0 until pInfo.length) { //update sub-sequences
            buffer[vStart + i].setVseq(pInfo.sub[i])
        }

        //check if tone re-position is needed
        val newTonePos = vStart + getTonePosition(newVs, vEnd == current)
        if (curTonePos != newTonePos && tone != 0) {
            markChange(newTonePos)
            buffer[newTonePos].tone = tone
            markChange(curTonePos)
            buffer[curTonePos].tone = 0
        }

        if (roofRemoved) {
            singleMode = false
            processAppend(ev)
            reverted = true
        }

        return 1
    }

    // ----------------------------------------------------------
    // can only be called from processHook
    // ----------------------------------------------------------
    private fun processHookWithUO(ev: KeyEvent): Int {
        var newVs: VowelSeq
        var hookRemoved = false
        val removeWithUndo = true

        if (opts.freeMarking == 0 && buffer[current].vOffset != 0) {
            return processAppend(ev)
        }

        val vEnd = current - buffer[current].vOffset
        val vs = buffer[vEnd].vseq()
        val vStart = vEnd - (vSeqList[vs].length - 1)
        val v = vSeqList[vs].v
        val curTonePos = vStart + getTonePosition(vs, vEnd == current)
        val tone = buffer[curTonePos].tone

        when (ev.evType) {
            VneHookU -> {
                if (v[0] == Vnl_u) {
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
            }
            VneHookO -> {
                if (v[1] == Vnl_o || v[1] == Vnl_or) {
                    if (vEnd == current && vSeqList[vs].length == 2 &&
                        buffer[current].form == VnwCV && buffer[current - 2].cseq() == CsTh
                    ) {
                        // o|o^ -> o+
                        newVs = vSeqList[vs].withHook
                        markChange(vStart + 1)
                        buffer[vStart + 1].vnSym = Vnl_oh
                    } else {
                        newVs = lookupVSeq(Vnl_uh, Vnl_oh, v[2])
                        if (v[0] == Vnl_u) {
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
                    if (v[0] == Vnl_uh) {
                        markChange(vStart)
                        buffer[vStart].vnSym = Vnl_u
                        buffer[vStart + 1].vnSym = Vnl_o
                    } else {
                        markChange(vStart + 1)
                        buffer[vStart + 1].vnSym = Vnl_o
                    }
                    hookRemoved = true
                }
            }
            else -> { //vneHookAll, vneHookUO:
                if (v[0] == Vnl_u) {
                    if (v[1] == Vnl_o || v[1] == Vnl_or) {
                        //uo -> uo+ if prefixed by "th"; u+o^ -> uo+ (uoow -> uơ)
                        if ((vs == Vs_uo || vs == Vs_uor) && vEnd == current &&
                            buffer[current].form == VnwCV && buffer[current - 2].cseq() == CsTh ||
                            (v[1] == Vnl_or && vSeqList[vs].length == 2)
                        ) {
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
                    if (v[1] == Vnl_o) { // u+o -> u+o+
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
        }

        val p = vSeqList[newVs]
        for (i in 0 until p.length) { //update sub-sequences
            buffer[vStart + i].setVseq(p.sub[i])
        }

        //check if tone re-position is needed
        val newTonePos = vStart + getTonePosition(newVs, vEnd == current)
        if (curTonePos != newTonePos && tone != 0) {
            markChange(newTonePos)
            buffer[newTonePos].tone = tone
            markChange(curTonePos)
            buffer[curTonePos].tone = 0
        }

        if (hookRemoved && removeWithUndo) {
            singleMode = false
            processAppend(ev)
            reverted = true
        }

        return 1
    }

    // ----------------------------------------------------------
    private fun processHook(ev: KeyEvent): Int {
        if (!vietKey || current < 0 || buffer[current].vOffset < 0) {
            return processAppend(ev)
        }

        var newVs: VowelSeq
        var changePos: Int
        var hookRemoved = false
        val pInfo: VowelSeqInfo

        val vEnd = current - buffer[current].vOffset
        val vs = buffer[vEnd].vseq()

        val v = vSeqList[vs].v

        if (vSeqList[vs].length > 1 &&
            ev.evType != VneBowl &&
            (v[0] == Vnl_u || v[0] == Vnl_uh) &&
            (v[1] == Vnl_o || v[1] == Vnl_oh || v[1] == Vnl_or)
        ) {
            return processHookWithUO(ev)
        }

        val vStart = vEnd - (vSeqList[vs].length - 1)
        val curTonePos = vStart + getTonePosition(vs, vEnd == current)
        val tone = buffer[curTonePos].tone

        newVs = vSeqList[vs].withHook
        if (newVs == VsNil) {
            if (vSeqList[vs].hookPos == -1) {
                return processAppend(ev) //hook is not applicable
            }

            //a hook already exists -> undo hook
            val curCh = buffer[vStart + vSeqList[vs].hookPos].vnSym
            val newCh = when (curCh) {
                Vnl_ab -> Vnl_a
                Vnl_uh -> Vnl_u
                else -> Vnl_o
            }
            changePos = vStart + vSeqList[vs].hookPos
            if (opts.freeMarking == 0 && changePos != current) {
                return processAppend(ev)
            }

            when (ev.evType) {
                VneHookU -> if (curCh != Vnl_uh) return processAppend(ev)
                VneHookO -> if (curCh != Vnl_oh) return processAppend(ev)
                VneBowl -> if (curCh != Vnl_ab) return processAppend(ev)
                else -> if (ev.evType == VneHookUO && curCh == Vnl_ab) return processAppend(ev)
            }

            markChange(changePos)
            buffer[changePos].vnSym = newCh

            newVs = when (vSeqList[vs].length) {
                3 -> lookupVSeq(buffer[vStart].vnSym, buffer[vStart + 1].vnSym, buffer[vStart + 2].vnSym)
                2 -> lookupVSeq(buffer[vStart].vnSym, buffer[vStart + 1].vnSym, VnlNonVnChar)
                else -> lookupVSeq(buffer[vStart].vnSym, VnlNonVnChar, VnlNonVnChar)
            }

            pInfo = vSeqList[newVs]
            hookRemoved = true
        } else {
            pInfo = vSeqList[newVs]

            when (ev.evType) {
                VneHookU -> if (pInfo.v[pInfo.hookPos] != Vnl_uh) return processAppend(ev)
                VneHookO -> if (pInfo.v[pInfo.hookPos] != Vnl_oh) return processAppend(ev)
                VneBowl -> if (pInfo.v[pInfo.hookPos] != Vnl_ab) return processAppend(ev)
                else -> { //vneHook_uo, vneHookAll
                    if (ev.evType == VneHookUO && pInfo.v[pInfo.hookPos] == Vnl_ab) return processAppend(ev)
                }
            }

            //check validity of new VC and CV
            var c1 = CsNil
            var c2 = CsNil
            if (buffer[current].c1Offset != -1) {
                c1 = buffer[current - buffer[current].c1Offset].cseq()
            }
            if (buffer[current].c2Offset != -1) {
                c2 = buffer[current - buffer[current].c2Offset].cseq()
            }
            if (!isValidCVC(c1, newVs, c2)) {
                return processAppend(ev)
            }

            changePos = vStart + pInfo.hookPos
            if (opts.freeMarking == 0 && changePos != current) {
                return processAppend(ev)
            }

            markChange(changePos)
            buffer[changePos].vnSym = pInfo.v[pInfo.hookPos]
        }

        for (i in 0 until pInfo.length) { //update sub-sequences
            buffer[vStart + i].setVseq(pInfo.sub[i])
        }

        //check if tone re-position is needed
        val newTonePos = vStart + getTonePosition(newVs, vEnd == current)
        if (curTonePos != newTonePos && tone != 0) {
            markChange(newTonePos)
            buffer[newTonePos].tone = tone
            markChange(curTonePos)
            buffer[curTonePos].tone = 0
        }

        if (hookRemoved) {
            singleMode = false
            processAppend(ev)
            reverted = true
        }

        return 1
    }

    // ----------------------------------------------------------
    private fun processTone(ev: KeyEvent): Int {
        if (current < 0 || !vietKey) {
            return processAppend(ev)
        }

        if (buffer[current].form == VnwC &&
            (buffer[current].cseq() == CsGi || buffer[current].cseq() == CsGin)
        ) {
            val p = if (buffer[current].cseq() == CsGi) current else current - 1
            if (buffer[p].tone == 0 && ev.tone == 0) {
                return processAppend(ev)
            }
            markChange(p)
            if (buffer[p].tone == ev.tone) {
                buffer[p].tone = 0
                singleMode = false
                processAppend(ev)
                reverted = true
                return 1
            }
            buffer[p].tone = ev.tone
            return 1
        }

        if (buffer[current].vOffset < 0) {
            return processAppend(ev)
        }

        val vEnd = current - buffer[current].vOffset
        val vs = buffer[vEnd].vseq()
        val info = vSeqList[vs]
        if (opts.spellCheckEnabled != 0 && opts.freeMarking == 0 && info.complete == 0) {
            return processAppend(ev)
        }

        if (buffer[current].form == VnwVC || buffer[current].form == VnwCVC) {
            val cs = buffer[current].cseq()
            if ((cs == CsC || cs == CsCh || cs == CsP || cs == CsT) &&
                (ev.tone == 2 || ev.tone == 3 || ev.tone == 4)
            ) {
                return processAppend(ev) // c, ch, p, t suffixes don't allow ` ? ~
            }
        }

        val toneOffset = getTonePosition(vs, vEnd == current)
        val tonePos = vEnd - (info.length - 1) + toneOffset
        if (buffer[tonePos].tone == 0 && ev.tone == 0) {
            return processAppend(ev)
        }

        if (buffer[tonePos].tone == ev.tone) {
            markChange(tonePos)
            buffer[tonePos].tone = 0
            singleMode = false
            processAppend(ev)
            reverted = true
            return 1
        }

        markChange(tonePos)
        buffer[tonePos].tone = ev.tone
        return 1
    }

    // ----------------------------------------------------------
    private fun processDd(ev: KeyEvent): Int {
        if (!vietKey || current < 0) {
            return processAppend(ev)
        }

        var pos: Int

        // we want to allow dd even in non-vn sequence, because dd is used a lot in abbreviation
        // we allow dd only if preceding character is not a vowel
        if (buffer[current].form == VnwNonVn &&
            buffer[current].vnSym == Vnl_d &&
            (current - 1 < 0 || buffer[current - 1].vnSym == VnlNonVnChar || !isVnVowel(buffer[current - 1].vnSym))
        ) {
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

        if (buffer[current].c1Offset < 0) {
            return processAppend(ev)
        }

        pos = current - buffer[current].c1Offset
        if (opts.freeMarking == 0 && pos != current) {
            return processAppend(ev)
        }

        if (buffer[pos].cseq() == CsD) {
            markChange(pos)
            buffer[pos].setCseq(CsDd)
            buffer[pos].vnSym = Vnl_dd
            return 1
        }

        if (buffer[pos].cseq() == CsDd) {
            //undo dd
            markChange(pos)
            buffer[pos].setCseq(CsD)
            buffer[pos].vnSym = Vnl_d
            singleMode = false
            processAppend(ev)
            reverted = true
            return 1
        }

        return processAppend(ev)
    }

    // ----------------------------------------------------------
    private fun processMapChar(ev: KeyEvent): Int {
        if (capsLockOn) {
            ev.vnSym = changeCase(ev.vnSym)
        }

        val ret0 = processAppend(ev)
        if (!vietKey) {
            return ret0
        }

        if (current >= 0 && buffer[current].form != VnwEmpty &&
            buffer[current].form != VnwNonVn
        ) {
            return 1
        }

        if (current < 0) {
            return 0
        }

        // mapChar doesn't apply
        current--
        val entry = buffer[current]

        var undo = false
        // test if undo is needed
        if (entry.form != VnwEmpty && entry.form != VnwNonVn) {
            var prevSym = entry.vnSym
            if (entry.caps != 0) {
                prevSym -= 1
            }
            if (prevSym == ev.vnSym) {
                if (entry.form != VnwC) {
                    val vEnd = current - entry.vOffset
                    val vs = buffer[vEnd].vseq()
                    val vStart = vEnd - vSeqList[vs].length + 1
                    val curTonePos = vStart + getTonePosition(vs, vEnd == current)
                    val tone = buffer[curTonePos].tone
                    markChange(current)
                    current--

                    //check if tone position is needed
                    if (tone != 0 && current >= 0 &&
                        (buffer[current].form == VnwV || buffer[current].form == VnwCV)
                    ) {
                        val newVs = buffer[current].vseq()
                        val newTonePos = vStart + getTonePosition(newVs, true)
                        if (newTonePos != curTonePos) {
                            markChange(newTonePos)
                            buffer[newTonePos].tone = tone
                            markChange(curTonePos)
                            buffer[curTonePos].tone = 0
                        }
                    }
                } else {
                    markChange(current)
                    current--
                }
                undo = true
            }
        }

        ev.evType = VneNormal
        ev.chType = input.getCharType(ev.keyCode)
        ev.vnSym = isoToVnLexi(ev.keyCode)
        val ret = processAppend(ev)
        if (undo) {
            singleMode = false
            reverted = true
            return 1
        }
        return ret
    }

    // ----------------------------------------------------------
    private fun processTelexW(ev: KeyEvent): Int {
        if (autoCapShifted && shiftPressed && isUpper(ev.keyCode and 0xFF) && atWordBeginning()) {
            autoCapShifted = false
            ev.evType = VneNormal
            return processAppend(ev)
        }
        if (!vietKey) {
            return processAppend(ev)
        }

        var ret: Int

        if (usedAsMapChar) {
            ev.evType = VneMapChar
            ev.vnSym = if (isUpper(ev.keyCode and 0xFF)) VnlUh else Vnl_uh
            if (capsLockOn) {
                ev.vnSym = changeCase(ev.vnSym)
            }
            ev.chType = UkcVn
            ret = processMapChar(ev)
            if (ret == 0) {
                if (current >= 0) {
                    current--
                }
                usedAsMapChar = false
                ev.evType = VneHookAll
                return processHook(ev)
            }
            return ret
        }

        ev.evType = VneHookAll
        usedAsMapChar = false
        ret = processHook(ev)
        if (ret == 0) {
            if (current >= 0) {
                current--
            }
            ev.evType = VneMapChar
            ev.vnSym = if (isUpper(ev.keyCode and 0xFF)) VnlUh else Vnl_uh
            if (capsLockOn) {
                ev.vnSym = changeCase(ev.vnSym)
            }
            ev.chType = UkcVn
            usedAsMapChar = true
            return processMapChar(ev)
        }
        return ret
    }

    // ----------------------------------------------------------
    private fun processAppend(ev: KeyEvent): Int {
        when (ev.chType) {
            UkcReset -> {
                if (ev.keyCode == EnterChar) {
                    if (opts.macroEnabled != 0 && macroMatch(ev)) {
                        return 1
                    }
                }
                reset()
                return 0
            }
            UkcWordBreak -> {
                singleMode = false
                return processWordEnd(ev)
            }
            UkcVn -> {
                if (isVnVowel(ev.vnSym)) {
                    val v = stdVnNoTone[vnToLower(ev.vnSym)]
                    if (current >= 0 && buffer[current].form == VnwC &&
                        ((buffer[current].cseq() == CsQ && v == Vnl_u) ||
                            (buffer[current].cseq() == CsG && v == Vnl_i))
                    ) {
                        return appendConsonnant(ev) //process u after q, i after g as consonnants
                    }
                    return appendVowel(ev)
                }
                return appendConsonnant(ev)
            }
            else -> { // UkcNonVn (and the dead UkcWordBreak arm in the Go switch)
                current++
                val entry = buffer[current]
                entry.form = VnwNonVn
                entry.c1Offset = -1
                entry.c2Offset = -1
                entry.vOffset = -1
                entry.keyCode = ev.keyCode
                entry.vnSym = vnToLower(ev.vnSym)
                entry.caps = if (entry.vnSym != ev.vnSym) 1 else 0
                entry.tone = 0
                return 0
            }
        }
    }

    // ----------------------------------------------------------
    private fun appendVowel(ev: KeyEvent): Int {
        current++
        val entry = buffer[current]

        val lowerSym = vnToLower(ev.vnSym)
        val canSym = stdVnNoTone[lowerSym]

        entry.vnSym = canSym
        entry.caps = if (lowerSym != ev.vnSym) 1 else 0
        entry.tone = (lowerSym - canSym) / 2
        entry.keyCode = ev.keyCode

        if (current == 0 || !vietKey) {
            entry.form = VnwV
            entry.c1Offset = -1
            entry.c2Offset = -1
            entry.vOffset = 0
            entry.setVseq(lookupVSeq(canSym, VnlNonVnChar, VnlNonVnChar))

            if (!vietKey || isAlpha(entry.keyCode)) {
                return 0
            }
            markChange(current)
            return 1
        }

        val prev = buffer[current - 1]
        var vs: VowelSeq
        var newVs: VowelSeq
        var cs: ConSeq
        var prevTonePos: Int
        var tone: Int
        var newTone: Int
        var tonePos: Int
        var newTonePos: Int

        when (prev.form) {
            VnwEmpty -> {
                entry.form = VnwV
                entry.c1Offset = -1
                entry.c2Offset = -1
                entry.vOffset = 0
                newVs = lookupVSeq(canSym, VnlNonVnChar, VnlNonVnChar)
                entry.setVseq(newVs)
            }

            VnwNonVn, VnwCVC, VnwVC -> {
                entry.form = VnwNonVn
                entry.c1Offset = -1
                entry.c2Offset = -1
                entry.vOffset = -1
            }

            VnwV, VnwCV -> {
                vs = prev.vseq()
                prevTonePos = (current - 1) - (vSeqList[vs].length - 1) + getTonePosition(vs, true)
                tone = buffer[prevTonePos].tone

                if (lowerSym != canSym && tone != 0) { //new sym has a tone, but there's already a preceding tone
                    newVs = VsNil
                } else {
                    newVs = when (vSeqList[vs].length) {
                        3 -> VsNil
                        2 -> lookupVSeq(vSeqList[vs].v[0], vSeqList[vs].v[1], canSym)
                        else -> lookupVSeq(vSeqList[vs].v[0], canSym, VnlNonVnChar)
                    }
                }

                if (newVs != VsNil && prev.form == VnwCV) {
                    cs = buffer[current - 1 - prev.c1Offset].cseq()
                    if (!isValidCV(cs, newVs)) {
                        newVs = VsNil
                    }
                }

                if (newVs == VsNil) {
                    entry.form = VnwNonVn
                    entry.c1Offset = -1
                    entry.c2Offset = -1
                    entry.vOffset = -1
                } else {
                    entry.form = prev.form
                    entry.c1Offset = if (prev.form == VnwCV) prev.c1Offset + 1 else -1
                    entry.c2Offset = -1
                    entry.vOffset = 0
                    entry.setVseq(newVs)
                    entry.tone = 0

                    newTone = (lowerSym - canSym) / 2
                    if (tone == 0) {
                        if (newTone != 0) {
                            tone = newTone
                            tonePos = getTonePosition(newVs, true) + ((current - 1) - vSeqList[vs].length + 1)
                            markChange(tonePos)
                            buffer[tonePos].tone = tone
                            return 1
                        }
                    } else {
                        newTonePos = getTonePosition(newVs, true) + ((current - 1) - vSeqList[vs].length + 1)
                        if (newTonePos != prevTonePos) {
                            markChange(prevTonePos)
                            buffer[prevTonePos].tone = 0
                            markChange(newTonePos)
                            if (newTone != 0) {
                                tone = newTone
                            }
                            buffer[newTonePos].tone = tone
                            return 1
                        }
                        if (newTone != 0 && newTone != tone) {
                            tone = newTone
                            markChange(prevTonePos)
                            buffer[prevTonePos].tone = tone
                            return 1
                        }
                    }
                }
            }

            VnwC -> {
                newVs = lookupVSeq(canSym, VnlNonVnChar, VnlNonVnChar)
                cs = prev.cseq()
                if (!isValidCV(cs, newVs)) {
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

                    if (cs == CsGi && prev.tone != 0) {
                        if (entry.tone == 0) {
                            entry.tone = prev.tone
                        }
                        markChange(current - 1)
                        prev.tone = 0
                        return 1
                    }
                }
            }
        }

        if (isAlpha(entry.keyCode)) {
            return 0
        }
        markChange(current)
        return 1
    }

    // ----------------------------------------------------------
    private fun appendConsonnant(ev: KeyEvent): Int {
        var complexEvent = false
        current++
        val entry = buffer[current]

        val lowerSym = vnToLower(ev.vnSym)

        entry.vnSym = lowerSym
        entry.caps = if (lowerSym != ev.vnSym) 1 else 0
        entry.keyCode = ev.keyCode
        entry.tone = 0

        if (current == 0 || !vietKey) {
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
        var isValid: Boolean

        val prev = buffer[current - 1]

        when (prev.form) {
            VnwNonVn -> {
                entry.form = VnwNonVn
                entry.c1Offset = -1
                entry.c2Offset = -1
                entry.vOffset = -1
                return 0
            }
            VnwEmpty -> {
                entry.form = VnwC
                entry.c1Offset = 0
                entry.c2Offset = -1
                entry.vOffset = -1
                entry.setCseq(lookupCSeq(lowerSym, VnlNonVnChar, VnlNonVnChar))
                return 0
            }
            VnwV, VnwCV -> {
                vs = prev.vseq()
                newVs = vs
                if (vs == Vs_uoh || vs == Vs_uho) {
                    newVs = Vs_uhoh
                }

                c1 = CsNil
                if (prev.c1Offset != -1) {
                    c1 = buffer[current - 1 - prev.c1Offset].cseq()
                }

                newCs = lookupCSeq(lowerSym, VnlNonVnChar, VnlNonVnChar)
                isValid = isValidCVC(c1, newVs, newCs)

                if (isValid) {
                    //check u+o -> u+o+
                    if (vs == Vs_uho) {
                        markChange(current - 1)
                        prev.vnSym = Vnl_oh
                        prev.setVseq(Vs_uhoh)
                        complexEvent = true
                    } else if (vs == Vs_uoh) {
                        markChange(current - 2)
                        buffer[current - 2].vnSym = Vnl_uh
                        buffer[current - 2].setVseq(Vs_uh)
                        prev.setVseq(Vs_uhoh)
                        complexEvent = true
                    }

                    if (prev.form == VnwV) {
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
                    val oldIdx = (current - 1) - (vSeqList[vs].length - 1) + getTonePosition(vs, true)
                    if (buffer[oldIdx].tone != 0) {
                        val newIdx = (current - 1) - (vSeqList[newVs].length - 1) + getTonePosition(newVs, false)
                        if (newIdx != oldIdx) {
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

                if (complexEvent) {
                    return 1
                }
                return 0
            }
            else -> { // VnwC, VnwVC, VnwCVC
                cs = prev.cseq()
                newCs = if (cs < 0 || cs >= cSeqList.size) {
                    // prev is a consonant entry with no real seq (bare h/j/w…) —
                    // C++ reads CSeqList[-1] out of bounds here; must not crash
                    CsNil
                } else if (cSeqList[cs].length == 3) {
                    CsNil
                } else if (cSeqList[cs].length == 2) {
                    lookupCSeq(cSeqList[cs].c[0], cSeqList[cs].c[1], lowerSym)
                } else {
                    lookupCSeq(cSeqList[cs].c[0], lowerSym, VnlNonVnChar)
                }

                if (newCs != CsNil && (prev.form == VnwVC || prev.form == VnwCVC)) {
                    // Check CVC combination
                    c1 = CsNil
                    if (prev.c1Offset != -1) {
                        c1 = buffer[current - 1 - prev.c1Offset].cseq()
                    }

                    val vIdx = (current - 1) - prev.vOffset
                    vs = buffer[vIdx].vseq()
                    isValid = isValidCVC(c1, vs, newCs)
                    if (!isValid) {
                        newCs = CsNil
                    }
                }

                if (newCs == CsNil) {
                    entry.form = VnwNonVn
                    entry.c1Offset = -1
                    entry.c2Offset = -1
                    entry.vOffset = -1
                } else {
                    when (prev.form) {
                        VnwC -> {
                            entry.form = VnwC
                            entry.c1Offset = 0
                            entry.c2Offset = -1
                            entry.vOffset = -1
                        }
                        VnwVC -> {
                            entry.form = VnwVC
                            entry.c1Offset = -1
                            entry.c2Offset = 0
                            entry.vOffset = prev.vOffset + 1
                        }
                        else -> { //vnw_cvc
                            entry.form = VnwCVC
                            entry.c1Offset = prev.c1Offset + 1
                            entry.c2Offset = 0
                            entry.vOffset = prev.vOffset + 1
                        }
                    }
                    entry.setCseq(newCs)
                }
                return 0
            }
        }
    }

    // ----------------------------------------------------------
    private fun processEscChar(ev: KeyEvent): Int {
        if (vietKey && current >= 0 &&
            buffer[current].form != VnwEmpty && buffer[current].form != VnwNonVn
        ) {
            toEscape = true
        }
        return processAppend(ev)
    }

    // ----------------------------------------------------------
    // This can be called only after other processing have been done.
    // The new event is supposed to be put into m_buffer already
    // ----------------------------------------------------------
    private fun processNoSpellCheck(ev: KeyEvent): Int {
        val entry = buffer[current]
        if (isVnVowel(entry.vnSym)) {
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

        if (ev.evType == VneNormal &&
            ((entry.keyCode >= 'a'.code && entry.keyCode <= 'z'.code) ||
                (entry.keyCode >= 'A'.code && entry.keyCode <= 'Z'.code))
        ) {
            return 0
        }
        markChange(current)
        return 1
    }

    // ----------------------------------------------------------
    // main entry point
    // ----------------------------------------------------------
    private fun process(keyCode: Int): Result {
        prepareBuffer()
        backs = 0
        changePos = current + 1
        out.clear()
        outputWritten = false
        reverted = false
        keyRestored = false
        keyRestoring = false
        outType = OutChar

        val ev = input.keyCodeToEvent(keyCode)

        var ret: Int
        if (!toEscape) {
            ret = dispatch(ev)
        } else {
            toEscape = false
            if (current < 0 || ev.evType == VneNormal || ev.evType == VneEscChar) {
                ret = processAppend(ev)
            } else {
                current--
                processAppend(ev)
                markChange(current) //this will assign m_backs to 1 and mark the character for output
                ret = 1
            }
        }

        if (vietKey &&
            current >= 0 && buffer[current].form == VnwNonVn &&
            ev.chType == UkcVn &&
            (opts.spellCheckEnabled == 0 || singleMode)
        ) {
            //The spell check has failed, but because we are in non-spellcheck mode,
            //we consider the new character as the beginning of a new word
            ret = processNoSpellCheck(ev)
        }

        //we add key to key buffer only if that key has not caused a reset
        if (current >= 0) {
            ev.chType = input.getCharType(ev.keyCode)
            keyCurrent++
            if (keyCurrent < MaxUkEngine) {
                keyStrokes[keyCurrent].ev.copyFrom(ev)
                keyStrokes[keyCurrent].converted = ret != 0 && !keyRestored
            }
        }
        autoCapShifted = false

        val res = Result()
        res.keyOut = outType == OutKey
        if (ret == 0) {
            return res
        }

        res.backs = backs
        if (!outputWritten) {
            writeOutput()
        }
        res.out = out.toCharArray()
        res.consumed = true
        return res
    }

    private fun dispatch(ev: KeyEvent): Int {
        return when (ev.evType) {
            VneRoofAll, VneRoofA, VneRoofE, VneRoofO -> processRoof(ev)
            VneHookAll, VneHookUO, VneHookU, VneHookO, VneBowl -> processHook(ev)
            VneDd -> processDd(ev)
            VneTone0, VneTone1, VneTone2, VneTone3, VneTone4, VneTone5 -> processTone(ev)
            VneTelexW -> processTelexW(ev)
            VneMapChar -> processMapChar(ev)
            VneEscChar -> processEscChar(ev)
            else -> processAppend(ev) //vneNormal
        }
    }

    // ----------------------------------------------------------
    private fun synchKeyStrokeBuffer() {
        //synchronize with key-stroke buffer
        if (keyCurrent >= 0) {
            keyCurrent--
        }
        if (current >= 0 && buffer[current].form == VnwEmpty) {
            //in character buffer, we have reached a word break,
            //so we also need to move key stroke pointer backward to corresponding word break
            while (keyCurrent >= 0 && keyStrokes[keyCurrent].ev.chType != UkcWordBreak) {
                keyCurrent--
            }
        }
    }

    // ----------------------------------------------------------
    private fun processBackspace(): Result {
        val res = Result()
        res.keyOut = false
        out.clear()
        if (!vietKey || current < 0) {
            return res
        }

        backs = 0
        changePos = current + 1
        markChange(current)

        if (current == 0 ||
            buffer[current].form == VnwEmpty ||
            buffer[current].form == VnwNonVn ||
            buffer[current].form == VnwC ||
            buffer[current - 1].form == VnwC ||
            buffer[current - 1].form == VnwCVC ||
            buffer[current - 1].form == VnwVC
        ) {
            current--
            res.backs = backs
            synchKeyStrokeBuffer()
            res.consumed = backs > 1
            return res
        }

        val vEnd = current - buffer[current].vOffset
        if (vEnd < 0) {
            current--
            res.backs = backs
            synchKeyStrokeBuffer()
            res.consumed = backs > 1
            return res
        }
        val vs = buffer[vEnd].vseq()
        val newVs = buffer[current - 1].vseq()
        if (vs < 0 || vs >= vSeqList.size || newVs < 0 || newVs >= vSeqList.size) {
            // cell carries no valid vowel seq (empty/nonVn) — C++ would silently
            // read out of bounds here; fall back to the plain decrement path
            current--
            res.backs = backs
            synchKeyStrokeBuffer()
            res.consumed = backs > 1
            return res
        }
        val vStart = vEnd - vSeqList[vs].length + 1
        val curTonePos = vStart + getTonePosition(vs, vEnd == current)
        val newTonePos = vStart + getTonePosition(newVs, true)
        val tone = buffer[curTonePos].tone

        if (tone == 0 || curTonePos == newTonePos ||
            (curTonePos == current && buffer[current].tone != 0)
        ) {
            current--
            res.backs = backs
            synchKeyStrokeBuffer()
            res.consumed = backs > 1
            return res
        }

        markChange(newTonePos)
        buffer[newTonePos].tone = tone
        markChange(curTonePos)
        buffer[curTonePos].tone = 0
        current--
        synchKeyStrokeBuffer()
        res.backs = backs
        writeOutput()
        res.out = out.toCharArray()
        res.consumed = true
        return res
    }

    // ----------------------------------------------------
    // make sure there are at least 10 entries available
    // ----------------------------------------------------
    private fun prepareBuffer() {
        //prepare symbol buffer
        if (current >= 0 && current + 10 >= bufSize) {
            // Get rid of at least half of the current entries
            // don't get rid from the middle of a word.
            var rid = current / 2
            while (buffer[rid].form != VnwEmpty && rid < current) {
                rid++
            }
            if (rid == current) {
                current = -1
            } else {
                rid++
                System.arraycopy(buffer, rid, buffer, 0, current + 1 - rid)
                current -= rid
            }
        }

        //prepare key stroke buffer
        if (keyCurrent > 0 && keyCurrent + 1 >= keyBufSize) {
            // Get rid of at least half of the current entries
            val rid = keyCurrent / 2
            System.arraycopy(keyStrokes, rid, keyStrokes, 0, keyCurrent + 1 - rid)
            keyCurrent -= rid
        }
    }

    // ----------------------------------------------------
    private fun macroMatch(ev: KeyEvent): Boolean {
        if (shiftPressed && (ev.keyCode == ' '.code || ev.keyCode == EnterChar)) {
            return false
        }

        var pMacText: List<StdVnChar>? = null
        val key = IntArray(MaxMacroKeyLen + 1)
        var i = current
        var j: Int

        while (i >= 0 && (current - i + 1) < MaxMacroKeyLen) {
            while (i >= 0 && buffer[i].form != VnwEmpty && (current - i + 1) < MaxMacroKeyLen) {
                i--
            }
            if (i >= 0 && buffer[i].form != VnwEmpty) {
                return false
            }

            if (i >= 0) {
                if (buffer[i].vnSym != VnlNonVnChar) {
                    key[0] = buffer[i].vnSym + VnStdCharOffset
                    if (buffer[i].caps != 0) {
                        key[0]--
                    }
                    key[0] += buffer[i].tone * 2
                } else {
                    key[0] = buffer[i].keyCode
                }
            }

            j = i + 1
            while (j <= current) {
                if (buffer[j].vnSym != VnlNonVnChar) {
                    key[j - i] = buffer[j].vnSym + VnStdCharOffset
                    if (buffer[j].caps != 0) {
                        key[j - i]--
                    }
                    key[j - i] += buffer[j].tone * 2
                } else {
                    key[j - i] = buffer[j].keyCode
                }
                j++
            }
            val keyLen = current - i + 1
            //search macro table
            pMacText = macStore.lookup(key.toList().subList(1, keyLen))
            if (pMacText != null) {
                i++ //mark the position where change is needed
                break
            }
            if (i >= 0) {
                pMacText = macStore.lookup(key.toList().subList(0, keyLen))
                if (pMacText != null) {
                    break
                }
            }
            i--
        }

        if (pMacText == null) {
            return false
        }

        markChange(i)
        out.clear()
        for (sc in pMacText) {
            stdCharToUTF16(sc)?.let { out.add(it) }
        }

        //write the last input character
        val vnChar: StdVnChar = if (ev.vnSym != VnlNonVnChar) {
            ev.vnSym + VnStdCharOffset
        } else {
            ev.keyCode
        }
        stdCharToUTF16(vnChar)?.let { out.add(it) }
        val b = backs //store backs before calling reset
        reset()
        outputWritten = true
        backs = b
        return true
    }

    // ----------------------------------------------------
    private fun doRestoreKeyStrokes(): Boolean {
        outType = OutKey
        if (!lastWordHasVnMark()) {
            return false
        }

        backs = 0
        changePos = current + 1

        var keyStart: Int
        var converted = false
        keyStart = keyCurrent
        while (keyStart >= 0 && keyStrokes[keyStart].ev.chType != UkcWordBreak) {
            if (keyStrokes[keyStart].converted) {
                converted = true
            }
            keyStart--
        }
        keyStart++

        if (!converted) {
            //no key stroke has been converted, so it doesn't make sense to restore key strokes
            return false
        }

        while (current >= 0 && buffer[current].form != VnwEmpty) {
            current--
        }
        markChange(current + 1)

        out.clear()
        keyRestoring = true
        for (i in keyStart..keyCurrent) {
            out.add(Char(keyStrokes[i].ev.keyCode and 0xFFFF))
            val ev = input.keyCodeToSymbol(keyStrokes[i].ev.keyCode)
            keyStrokes[i].converted = false
            processAppend(ev)
        }
        keyRestoring = false

        return true
    }

    // --------------------------------------------------
    private fun processWordEnd(ev: KeyEvent): Int {
        if (opts.macroEnabled != 0 && macroMatch(ev)) {
            return 1
        }

        if (opts.spellCheckEnabled == 0 || singleMode || current < 0 || keyRestoring) {
            current++
            val entry = buffer[current]
            entry.form = VnwEmpty
            entry.c1Offset = -1
            entry.c2Offset = -1
            entry.vOffset = -1
            entry.keyCode = ev.keyCode
            entry.vnSym = vnToLower(ev.vnSym)
            entry.caps = if (entry.vnSym != ev.vnSym) 1 else 0
            return 0
        }

        if (opts.autoNonVnRestore != 0 && lastWordIsNonVn()) {
            if (doRestoreKeyStrokes()) {
                keyRestored = true
                outputWritten = true
            }
        }

        current++
        val entry = buffer[current]
        entry.form = VnwEmpty
        entry.c1Offset = -1
        entry.c2Offset = -1
        entry.vOffset = -1
        entry.keyCode = ev.keyCode
        entry.vnSym = vnToLower(ev.vnSym)
        entry.caps = if (entry.vnSym != ev.vnSym) 1 else 0

        if (keyRestored) {
            out.add(Char(ev.keyCode and 0xFFFF))
            return 1
        }

        return 0
    }

    // -----------------------------------------------------------
    // Test if last word is a non-Vietnamese word
    // -----------------------------------------------------------
    private fun lastWordIsNonVn(): Boolean {
        if (current < 0) {
            return false
        }

        when (buffer[current].form) {
            VnwNonVn -> {
                // Abbreviation exception: an all-consonant word carrying a đ mark
                // (HĐ, CĐT, ĐH…) is valid Vietnamese shorthand — keep it as typed
                // instead of restoring the raw keystrokes.
                var hasDd = false
                var i = current
                while (i >= 0 && buffer[i].form != VnwEmpty) {
                    val sym = buffer[i].vnSym
                    if (isVnVowel(sym)) {
                        return true // has a vowel: a real (invalid) word — restore
                    }
                    if (sym == Vnl_dd) {
                        hasDd = true
                    }
                    i--
                }
                return !hasDd
            }
            VnwEmpty, VnwC -> return false
            VnwV, VnwCV -> return vSeqList[buffer[current].vseq()].complete == 0
            VnwVC, VnwCVC -> {
                val vIndex = current - buffer[current].vOffset
                val vs = buffer[vIndex].vseq()
                if (vSeqList[vs].complete == 0) {
                    return true
                }
                val cs = buffer[current].cseq()
                var c1 = CsNil
                if (buffer[current].c1Offset != -1) {
                    c1 = buffer[current - buffer[current].c1Offset].cseq()
                }

                if (!isValidCVC(c1, vs, cs)) {
                    return true
                }

                val tonePos = (vIndex - vSeqList[vs].length + 1) + getTonePosition(vs, false)
                val tone = buffer[tonePos].tone
                if ((cs == CsC || cs == CsCh || cs == CsP || cs == CsT) &&
                    (tone == 2 || tone == 3 || tone == 4)
                ) {
                    return true
                }
            }
        }
        return false
    }

    // -----------------------------------------------------------
    // Test if last word has a Vietnamese mark, that is tones, decorators
    // -----------------------------------------------------------
    private fun lastWordHasVnMark(): Boolean {
        var i = current
        while (i >= 0 && buffer[i].form != VnwEmpty) {
            val sym = buffer[i].vnSym
            if (sym != VnlNonVnChar) {
                if (isVnVowel(sym)) {
                    if (buffer[i].tone != 0) {
                        return true
                    }
                }
                if (sym != stdVnRootChar[sym]) {
                    return true
                }
            }
            i--
        }
        return false
    }

    // ----------------------------------------------------------
    // Public API matching the shell's Engine interface
    // ----------------------------------------------------------
    fun filter(keyCode: Int): Result = process(keyCode)

    fun backspace(): Result = processBackspace()

    fun restoreKeyStrokes(): Result {
        val res = Result()
        res.keyOut = true
        out.clear()
        if (doRestoreKeyStrokes()) {
            res.backs = backs
            res.out = out.toCharArray()
            res.consumed = true
        }
        return res
    }

    fun loadMacroFile(path: String): Boolean = loadMacroFileImpl(path)

    fun loadKeyMapFile(path: String): Boolean {
        val m = loadKeyMapFileImpl(path) ?: return false
        setKeyMap(m)
        return true
    }
}
