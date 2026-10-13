package com.tuigo.engine

import kotlin.test.Test
import kotlin.test.assertEquals

// Regression guard for the Android "owr" -> "ở" report: the Kotlin engine
// must keep Telex parity with the Windows port under the full app option set.
class ToneBugTest {
    private fun type(s: String): String {
        val e = Engine()
        e.setOptions(Options(freeMarking = 1, modernStyle = 1, spellCheckEnabled = 1, macroEnabled = 1, autoNonVnRestore = 1))
        var doc = ""
        for (ch in s) {
            val r = e.filter(ch.code)
            doc = if (r.consumed) doc.dropLast(r.backs) + String(r.out) else doc + ch
        }
        return doc
    }

    @Test fun owr() = assertEquals("ở ", type("owr "))
    @Test fun vowr() = assertEquals("vở ", type("vowr "))
    @Test fun uowr() = assertEquals("ưở ", type("uowr "))
    @Test fun orw() = assertEquals("ở ", type("orw "))
    @Test fun vor() = assertEquals("vỏ ", type("vor "))
}
