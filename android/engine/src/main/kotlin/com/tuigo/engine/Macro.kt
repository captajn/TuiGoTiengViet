package com.tuigo.engine

import java.io.File

/** Validate before saving; the reader remains compatible with existing files. */
fun macroValidationError(text: String): String? {
    val keys = mutableSetOf<String>()
    for ((index, raw) in text.lineSequence().withIndex()) {
        val line = raw.removePrefix("\uFEFF").trim()
        if (line.isEmpty() || line.startsWith(';')) continue
        val separator = line.indexOf(':')
        fun error(message: String) = "Dòng ${index + 1}: $message"
        if (separator <= 0) return error("dùng định dạng từ tắt:nội dung.")
        val key = line.substring(0, separator)
        if (key.any { it.isWhitespace() || it.isISOControl() }) return error("từ tắt không chứa khoảng trắng.")
        if (key.codePointCount(0, key.length) >= MaxMacroKeyLen) return error("từ tắt tối đa 15 ký tự.")
        if (line.substring(separator + 1).isBlank()) return error("nội dung thay thế không được trống.")
        if (!keys.add(key)) return error("từ tắt '$key' bị trùng.")
        if (keys.size > MaxMacroItems) return error("tối đa 1024 mục gõ tắt.")
    }
    return null
}

// Convert one codepoint to StdVnChar (Unicode -> VNSTANDARD conversion path).
private fun codePointToStdVn(r: Int): StdVnChar {
    if (r < 0x10000) {
        uniToStdVn[r]?.let { return it }
    }
    return r
}

internal fun textToStdVn(s: String): List<StdVnChar> {
    val out = ArrayList<StdVnChar>(s.length)
    var i = 0
    while (i < s.length) {
        val cp = s.codePointAt(i)
        // StdVnChar reserves values above BMP for Vietnamese symbols.
        // Keep astral Unicode as UTF-16 surrogate units, not colliding codepoints.
        if (cp > 0xFFFF) Character.toChars(cp).forEach { out.add(it.code) }
        else out.add(codePointToStdVn(cp))
        i += Character.charCount(cp)
    }
    return out
}

// item format "key:text"
internal fun MacroStore.addItem(item: String): Boolean {
    val pos = item.indexOf(':')
    if (pos < 0) return false
    var key = item.substring(0, pos)
    if (key.codePointCount(0, key.length) > MaxMacroKeyLen - 1) {
        key = key.substring(0, key.offsetByCodePoints(0, MaxMacroKeyLen - 1))
    }
    val text = item.substring(pos + 1)
    if (items.size >= MaxMacroItems) return false
    items[textToStdVn(key)] = textToStdVn(text)
    return true
}

// UTF-8 macro file; lines "key:text", ';' comments,
// header ";[DO NOT DELETE THIS LINE]***version=n" tolerated.
internal fun Engine.loadMacroFileImpl(fname: String): Boolean {
    val f = File(fname)
    // Clear before the early return: a removed/unreadable file must not
    // leave the previous macro list active (reloaded on activation).
    macStore.items.clear()
    if (!f.isFile) return false

    f.bufferedReader(Charsets.UTF_8).useLines { lines ->
        for (raw in lines) {
            var line = raw
            // strip UTF-8 BOM on first line if present
            line = line.removePrefix("\uFEFF").trim()
            if (line.isEmpty() || line.startsWith(";")) continue
            macStore.addItem(line)
        }
    }
    return true
}

// In-memory variant for Android assets / tests.
fun Engine.loadMacroText(text: String) {
    macStore.items.clear()
    for (raw in text.lineSequence()) {
        val line = raw.removePrefix("\uFEFF").trim()
        if (line.isEmpty() || line.startsWith(";")) continue
        macStore.addItem(line)
    }
}
