package com.tuigo.engine

import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertTrue
import kotlin.test.fail

// Simulates a text document: engine output is applied as
// "delete <backs> chars, insert <out>".
private class DocSim {
    private val text = StringBuilder()

    fun feed(e: Engine, key: Char) {
        val res = e.filter(key.code)
        if (res.consumed) {
            var backs = res.backs
            if (backs > text.length) backs = text.length
            text.delete(text.length - backs, text.length)
            text.append(String(res.out)) // CharArray already holds UTF-16 units
        } else {
            text.append(key)
        }
    }

    fun backspace(e: Engine) {
        val res = e.backspace()
        if (res.consumed) {
            var backs = res.backs
            if (backs > text.length) backs = text.length
            text.delete(text.length - backs, text.length)
            text.append(String(res.out))
        } else if (text.isNotEmpty()) {
            text.deleteCharAt(text.length - 1)
        }
    }

    fun typeKeys(e: Engine, keys: String) {
        for (k in keys) feed(e, k)
    }

    override fun toString() = text.toString()
}

private fun newEngine(opts: Options?): Engine {
    val e = Engine()
    if (opts != null) e.setOptions(opts)
    return e
}

private val defaultOpts = Options().apply {
    freeMarking = 1
    modernStyle = 1
    spellCheckEnabled = 1
}

class EngineTest {

    @Test
    fun telex() {
        val cases = arrayOf(
            "tieengs" to "tiếng",
            "aa" to "â",
            "aw" to "ă",
            "ow" to "ơ",
            "uw" to "ư",
            "oo" to "ô",
            "ee" to "ê",
            "dd" to "đ",
            "vietj nam" to "viẹt nam",
            "vieetj" to "việt",
            "dungx" to "dũng",
            "xin chaof" to "xin chào",
            "nguwowif" to "người",
            "thuow" to "thuơ",
            "duongf" to "duòng",
            "duowngf" to "dường",
            "muwowng" to "mương",
            "muwowngf" to "mường",
            "nghiengx" to "nghiẽng",
            "nghieengx" to "nghiễng",
            "khoiwr" to "khởi",
            "voiws" to "với",
            "quanr" to "quản",
            "trij" to "trị",
            "hots" to "hót",
            "langx" to "lãng",
            "quanrz" to "quan",
            "dduwowngf" to "đường",
            "toanfs" to "toán", // later tone key overrides
            "toansz" to "toan", // z strips the tone
            "uow" to "ươ",
            "uoow" to "uơ",   // u+ô + w -> uơ (issue #3: huơ, khuơ)
            "huoow" to "huơ",
            "khuoow" to "khuơ",
            "uooiw" to "ươi", // len-3 vowel keeps double hook
            "nguooiw" to "ngươi",
            "uoowf" to "uờ",
            "nguoiwf" to "người",
            "quans" to "quán",
        )
        for ((keys, want) in cases) {
            val e = newEngine(defaultOpts)
            val d = DocSim()
            d.typeKeys(e, keys)
            assertEquals(want, d.toString(), "telex \"$keys\"")
        }
    }

    @Test
    fun telexOldStyle() {
        // modernStyle off -> tone on first vowel of oa/oe/uy
        val opts = Options().apply { freeMarking = 1 }
        val e = newEngine(opts)
        val d = DocSim()
        d.typeKeys(e, "hoaf")
        assertEquals("hòa", d.toString())
    }

    @Test
    fun telexModernStyle() {
        val e = newEngine(defaultOpts)
        val d = DocSim()
        d.typeKeys(e, "hoaf")
        assertEquals("hoà", d.toString())
    }

    @Test
    fun toneRemoval() {
        // pressing the same tone key twice removes the tone (UniKey semantics:
        // the reverted key is consumed into the output, so "hoss" -> "hos")
        val e = newEngine(defaultOpts)
        val d = DocSim()
        d.typeKeys(e, "hoss")
        assertEquals("hos", d.toString())
        d.typeKeys(e, "s") // third s passes through as a literal letter
        assertEquals("hoss", d.toString())
    }

    @Test
    fun zRemovesTone() {
        val e = newEngine(defaultOpts)
        val d = DocSim()
        d.typeKeys(e, "hasz")
        assertEquals("ha", d.toString())
    }

    @Test
    fun escapeW() {
        // standalone w maps to ư via mapChar fallback
        val e = newEngine(defaultOpts)
        val d = DocSim()
        d.typeKeys(e, "w")
        assertEquals("ư", d.toString())
    }

    @Test
    fun autoCapWStaysLatin() {
        val e = newEngine(defaultOpts)
        e.shiftPressed = true
        e.autoCapShifted = true
        val d = DocSim()
        d.feed(e, 'W')
        assertEquals("W", d.toString())
    }

    @Test
    fun vni() {
        val e = newEngine(defaultOpts)
        e.setInputMethod(ImVni)
        val d = DocSim()
        d.typeKeys(e, "vie6t5 nam")
        assertEquals("việt nam", d.toString())

        val e2 = newEngine(defaultOpts)
        e2.setInputMethod(ImVni)
        val d2 = DocSim()
        d2.typeKeys(e2, "huo67 khuo67")
        assertEquals("huơ khuơ", d2.toString())
    }

    @Test
    fun backspaceToneMove() {
        // "toanf" -> "toàn"; backspace removes 'n' -> tone moves to 'o'
        val e = newEngine(defaultOpts)
        val d = DocSim()
        d.typeKeys(e, "toanf")
        assertEquals("toàn", d.toString())
        d.backspace(e)
        assertEquals("toà", d.toString())
    }

    @Test
    fun nonVnWord() {
        val e = newEngine(defaultOpts)
        val d = DocSim()
        d.typeKeys(e, "hello ")
        assertEquals("hello ", d.toString())
    }

    @Test
    fun vietKeyOff() {
        val e = newEngine(defaultOpts)
        e.setVietKey(false)
        val d = DocSim()
        d.typeKeys(e, "tieengs")
        assertEquals("tieengs", d.toString())
    }

    @Test
    fun autoNonVnRestore() {
        // a word with marks but invalid spelling restores original keystrokes
        val opts = Options().apply {
            freeMarking = 1
            modernStyle = 1
            spellCheckEnabled = 1
            autoNonVnRestore = 1
        }
        val e = newEngine(opts)
        val d = DocSim()
        d.typeKeys(e, "ddasd ") // đá + d is invalid -> restore to "ddasd "
        assertEquals("ddasd ", d.toString())
    }

    @Test
    fun ddAbbreviations() {
        // gõ tắt chữ Đ in consonant-only abbreviations: kept as typed
        val opts = Options().apply {
            freeMarking = 1
            modernStyle = 1
            spellCheckEnabled = 1
            autoNonVnRestore = 1
        }
        val cases = arrayOf(
            "HDD " to "HĐ ",
            "CDDT " to "CĐT ",
            "DDH " to "ĐH ",
            "DDCS " to "ĐCS ",
            "Hdd, CddT; DdH. " to "Hđ, CđT; ĐH. ",
            "hdd " to "hđ ",
            "cddt " to "cđt ",
            "ddh " to "đh ",
            "DD " to "Đ ",
            "ddasd " to "ddasd ", // marked invalid word with vowels still restores
            "HCM " to "HCM ",     // consonant after bare 'h' must not panic
            "hn " to "hn ",
        )
        for ((keys, want) in cases) {
            val e = newEngine(opts)
            val d = DocSim()
            d.typeKeys(e, keys)
            assertEquals(want, d.toString(), "dd abbrev \"$keys\"")
        }
    }

    @Test
    fun macro() {
        val e = newEngine(Options().apply { macroEnabled = 1 })
        e.macStore.addItem("vn:Viet Nam")
        val d = DocSim()
        d.typeKeys(e, "vn ")
        assertEquals("Viet Nam ", d.toString())
    }

    @Test
    fun macroReloadAndUnicode() {
        val e = newEngine(Options().apply { macroEnabled = 1 })
        e.loadMacroText("\uFEFF; danh sách\nko:không 😀\nmail:a:b@example.com")
        val first = DocSim()
        first.typeKeys(e, "ko ")
        assertEquals("không 😀 ", first.toString())
        e.resetBuf()
        e.loadMacroText("ko:khác")
        val edited = DocSim()
        edited.typeKeys(e, "ko ")
        assertEquals("khác ", edited.toString())
        e.resetBuf()
        e.loadMacroText("")
        val deleted = DocSim()
        deleted.typeKeys(e, "ko ")
        assertEquals("ko ", deleted.toString())
        e.resetBuf()
        e.loadMacroText("ko:không")
        e.setOptions(Options().apply { macroEnabled = 0 })
        val disabled = DocSim()
        disabled.typeKeys(e, "ko ")
        assertEquals("ko ", disabled.toString())
    }

    @Test
    fun viqr() {
        val e = newEngine(defaultOpts)
        e.setInputMethod(ImViqr)
        val d = DocSim()
        d.typeKeys(e, "o^n? ")
        assertEquals("ổn ", d.toString())
    }

    @Test
    fun macroValidation() {
        assertEquals(null, macroValidationError("; comment\nko:không 😀\nmail:a:b@example.com"))
        assertEquals(null, macroValidationError(""))
        for (bad in listOf("ko=không", ":không", "ko:", "a b:c", "ko:a\nko:b", "abcdefghijklmnop:x")) {
            assertTrue(macroValidationError(bad) != null, "must reject $bad")
        }
        assertEquals(null, macroValidationError("abcdefghijklmno:x"))
    }

    @Test
    fun tcvnConvert() {
        val out = charArrayOf('t', 'i', 'ế', 'n', 'g') // "tiếng" UTF-16
        convertTCVN(out)
        val want = charArrayOf('t', 'i', Char(0x00d5), 'n', 'g') // TCVN3: ế -> 0xD5
        assertTrue(out.contentEquals(want), "tcvn convert: ${out.map { it.code }}")

        // ASCII + unmapped chars pass through untouched
        val mixed = charArrayOf('A', 'z', '1', Char(0x4E2D))
        convertTCVN(mixed)
        if (mixed[0] != 'A' || mixed[1] != 'z' || mixed[2] != '1' || mixed[3] != Char(0x4E2D)) {
            fail("passthrough broken: ${mixed.map { it.code }}")
        }
    }

    @Test
    fun telexVniHybrid() {
        var e = newEngine(defaultOpts)
        e.setInputMethod(ImTelexVni)
        var d = DocSim()
        d.typeKeys(e, "on63") // VNI: ô + hỏi -> ổn
        assertEquals("ổn", d.toString())

        e = newEngine(defaultOpts)
        e.setInputMethod(ImTelexVni)
        d = DocSim()
        d.typeKeys(e, "oonr") // Telex: ô + hỏi -> ổn
        assertEquals("ổn", d.toString())
    }
}
