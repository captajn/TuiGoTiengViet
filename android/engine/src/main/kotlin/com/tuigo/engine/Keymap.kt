package com.tuigo.engine

import java.io.File

// User keymap file format (usrkeymap.cpp):
//
//	X = ActionLabel    (single char key = label)
//	; comment
private val ukEvLabels = mapOf(
    "Tone0" to VneTone0,
    "Tone1" to VneTone1,
    "Tone2" to VneTone2,
    "Tone3" to VneTone3,
    "Tone4" to VneTone4,
    "Tone5" to VneTone5,
    "Roof-All" to VneRoofAll,
    "Roof-A" to VneRoofA,
    "Roof-E" to VneRoofE,
    "Roof-O" to VneRoofO,
    "Hook-Bowl" to VneHookAll,
    "Hook-UO" to VneHookUO,
    "Hook-U" to VneHookU,
    "Hook-O" to VneHookO,
    "Bowl" to VneBowl,
    "D-Mark" to VneDd,
    "Telex-W" to VneTelexW,
    "Escape" to VneEscChar,
    "DD" to VneCount + VnlDD,
    "dd" to VneCount + Vnl_dd,
    "A^" to VneCount + VnlAr,
    "a^" to VneCount + Vnl_ar,
    "A(" to VneCount + VnlAb,
    "a(" to VneCount + Vnl_ab,
    "E^" to VneCount + VnlEr,
    "e^" to VneCount + Vnl_er,
    "O^" to VneCount + VnlOr,
    "o^" to VneCount + Vnl_or,
    "O+" to VneCount + VnlOh,
    "o+" to VneCount + Vnl_oh,
    "U+" to VneCount + VnlUh,
    "u+" to VneCount + Vnl_uh,
)

// Parses UniKey keymap.txt content into a 256-entry keymap, or null on bad input.
internal fun parseKeyMap(text: String): IntArray {
    val keyMap = IntArray(256) { VneNormal }
    val assigned = HashSet<Int>()

    for (raw in text.lineSequence()) {
        var line = raw
        val idx = line.indexOf(';')
        if (idx >= 0) line = line.substring(0, idx)
        val parts = line.split('=', limit = 2)
        if (parts.size != 2) continue
        val name = parts[0].trim()
        val value = parts[1].trim()
        if (name.length != 1) continue
        val action = ukEvLabels[value] ?: continue
        val c = name[0].code
        if (c in assigned) continue // already assigned, don't accept this map
        keyMap[c] = action
        assigned.add(c)
        if (action < VneCount) {
            keyMap[toLower(c)] = action
            keyMap[toUpper(c)] = action
        }
    }
    return keyMap
}

// Parses a UniKey keymap.txt file into a 256-entry keymap (null if unreadable).
internal fun loadKeyMapFileImpl(fname: String): IntArray? {
    val f = File(fname)
    if (!f.isFile) return null
    return parseKeyMap(f.readText(Charsets.UTF_8))
}
