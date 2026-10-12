package com.tuigo

import android.content.Context
import android.content.SharedPreferences
import com.tuigo.engine.Engine
import com.tuigo.engine.ImTelex
import com.tuigo.engine.ImVni
import com.tuigo.engine.ImViqr
import com.tuigo.engine.Options

object Prefs {
    const val NAME = "tuigo_prefs"
    const val KEY_IM = "input_method"
    const val KEY_FREE_MARKING = "free_marking"
    const val KEY_MODERN_STYLE = "modern_style"
    const val KEY_SPELL_CHECK = "spell_check"
    const val KEY_MACRO = "macro"
    const val KEY_AUTO_NONVN = "auto_nonvn_restore"
    const val KEY_DOUBLE_SPACE = "double_space_period"
    const val KEY_VIET_ON = "viet_on"
    const val KEY_NUMBER_ROW = "number_row"
    const val KEY_KEY_HEIGHT = "key_height"
    const val KEY_KEY_GAP = "key_gap"

    fun get(ctx: Context): SharedPreferences =
        ctx.getSharedPreferences(NAME, Context.MODE_PRIVATE)

    /** Visible desktop/mobile modes are Telex, VNI and VIQR. */
    fun inputMethod(ctx: Context): Int {
        val p = get(ctx)
        val saved = p.getInt(KEY_IM, ImTelex)
        val selected = if (saved in listOf(ImTelex, ImVni, ImViqr)) saved else ImTelex
        if (saved != selected) p.edit().putInt(KEY_IM, selected).apply()
        return selected
    }

    /** Apply persisted options onto a live engine. */
    fun applyTo(engine: Engine, ctx: Context) {
        val p = get(ctx)
        engine.setInputMethod(inputMethod(ctx))
        engine.setOptions(Options(
            freeMarking = if (p.getBoolean(KEY_FREE_MARKING, true)) 1 else 0,
            modernStyle = if (p.getBoolean(KEY_MODERN_STYLE, true)) 1 else 0,
            spellCheckEnabled = if (p.getBoolean(KEY_SPELL_CHECK, true)) 1 else 0,
            macroEnabled = if (p.getBoolean(KEY_MACRO, true)) 1 else 0,
            autoNonVnRestore = if (p.getBoolean(KEY_AUTO_NONVN, true)) 1 else 0,
        ))
        engine.setVietKey(p.getBoolean(KEY_VIET_ON, true))
    }

    fun vietOn(ctx: Context): Boolean = get(ctx).getBoolean(KEY_VIET_ON, true)
    fun doubleSpacePeriod(ctx: Context): Boolean =
        get(ctx).getBoolean(KEY_DOUBLE_SPACE, true)
    fun numberRow(ctx: Context): Boolean =
        get(ctx).getBoolean(KEY_NUMBER_ROW, false)
    fun keyHeightDp(ctx: Context): Int = get(ctx).getInt(KEY_KEY_HEIGHT, 56)
    fun keyGapDp(ctx: Context): Int = get(ctx).getInt(KEY_KEY_GAP, 4)
}
