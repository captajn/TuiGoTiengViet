package com.tuigo.engine

internal class KeyEvent(
    var keyCode: Int = 0,
    var evType: Int = VneNormal,
    var chType: Int = 0,
    var vnSym: VnLexiName = VnlNonVnChar,
    var tone: Int = -1,
) {
    fun copyFrom(o: KeyEvent) {
        keyCode = o.keyCode; evType = o.evType; chType = o.chType; vnSym = o.vnSym; tone = o.tone
    }
}

internal class InputProcessor {
    var im: Int = ImTelex
        private set
    val keyMap = IntArray(256)

    init {
        setIM(ImTelex)
    }

    fun setIM(im: Int) {
        this.im = im
        when (im) {
            ImVni -> useBuiltIn(vniMethodMapping)
            ImViqr -> useBuiltIn(viqrMethodMapping)
            ImMsVi -> useBuiltIn(msViMethodMapping)
            ImTelexSimple -> useBuiltIn(simpleTelexMethodMapping)
            ImTelexVni -> useBuiltIn(telexMethodMapping + vniMethodMapping)
            else -> {
                this.im = ImTelex
                useBuiltIn(telexMethodMapping)
            }
        }
    }

    fun setIMKeyMap(keyMap: IntArray) {
        im = ImUsrKeymap
        keyMap.copyInto(this.keyMap, 0, 0, 256)
    }

    private fun useBuiltIn(map: Array<KeyMapping>) {
        keyMap.fill(VneNormal)
        for (m in map) {
            keyMap[m.key] = m.action
            if (m.action < VneCount) {
                if (isLower(m.key)) {
                    keyMap[toUpper(m.key)] = m.action
                } else if (isUpper(m.key)) {
                    keyMap[toLower(m.key)] = m.action
                }
            }
        }
    }

    fun keyCodeToEvent(keyCode: Int): KeyEvent {
        val ev = KeyEvent(keyCode = keyCode, evType = VneNormal, tone = -1)
        if (keyCode > 255) {
            ev.vnSym = isoToVnLexi(keyCode)
            ev.chType = if (ev.vnSym == VnlNonVnChar) UkcNonVn else UkcVn
        } else {
            ev.chType = ukcMap[keyCode]
            ev.evType = keyMap[keyCode]
            if (ev.evType in VneTone0..VneTone5) {
                ev.tone = ev.evType - VneTone0
            }
            if (ev.evType >= VneCount) {
                ev.chType = UkcVn
                ev.vnSym = ev.evType - VneCount
                ev.evType = VneMapChar
            } else {
                ev.vnSym = isoToVnLexi(keyCode)
            }
        }
        return ev
    }

    fun keyCodeToSymbol(keyCode: Int): KeyEvent {
        val ev = KeyEvent(keyCode = keyCode, evType = VneNormal, tone = -1)
        ev.vnSym = isoToVnLexi(keyCode)
        if (keyCode > 255) {
            ev.chType = if (ev.vnSym == VnlNonVnChar) UkcNonVn else UkcVn
        } else {
            ev.chType = ukcMap[keyCode]
        }
        return ev
    }

    fun getCharType(keyCode: Int): Int =
        if (keyCode > 255) {
            if (isoToVnLexi(keyCode) == VnlNonVnChar) UkcNonVn else UkcVn
        } else {
            ukcMap[keyCode]
        }
}

internal fun isLower(c: Int) = c in 'a'.code..'z'.code
internal fun isUpper(c: Int) = c in 'A'.code..'Z'.code
internal fun toLower(c: Int) = if (isUpper(c)) c + 32 else c
internal fun toUpper(c: Int) = if (isLower(c)) c - 32 else c
internal fun isAlpha(c: Int) = isLower(c) || isUpper(c)
