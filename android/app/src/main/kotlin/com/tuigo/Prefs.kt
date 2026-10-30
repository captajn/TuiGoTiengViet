package com.tuigo

import android.content.Context
import android.content.SharedPreferences
import android.os.Build
import android.view.inputmethod.InputMethodManager
import android.view.inputmethod.InputMethodSubtype
import org.json.JSONObject
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
    const val KEY_AUTO_CAPITALIZE = "auto_capitalize"
    const val KEY_VIET_ON = "viet_on"
    const val KEY_NUMBER_ROW = "number_row"
    const val KEY_KEY_HEIGHT = "key_height"
    const val KEY_LAND_KEY_HEIGHT = "land_key_height"
    const val KEY_LAND_SPLIT = "land_split"
    const val KEY_LAND_SPLIT_GAP = "land_split_gap"
    const val KEY_HOLD_DEL_SELECT = "hold_del_select"
    const val KEY_HAPTIC = "haptic_feedback"
    const val KEY_KEY_POPUP = "key_popup"
    const val KEY_KEY_GAP = "key_gap"
    const val KEY_THEME = "key_theme"
    const val KEY_CLIPBOARD_SUGGEST = "clipboard_suggest"
    const val KEY_CLIPBOARD_HISTORY = "clipboard_history"
    const val KEY_CLIPBOARD_MAX = "clipboard_history_max"
    const val KEY_VIET_PER_APP = "viet_per_app"

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

    fun subtypeForMethod(im: Int): InputMethodSubtype {
        val labelRes = when (im) {
            ImVni -> R.string.subtype_vi_vni
            ImViqr -> R.string.subtype_vi_viqr
            else -> R.string.subtype_vi_telex
        }
        @Suppress("DEPRECATION")
        val builder = InputMethodSubtype.InputMethodSubtypeBuilder()
            .setSubtypeNameResId(labelRes)
            .setSubtypeLocale("vi_VN")
            .setSubtypeMode("keyboard")
            .setIsAsciiCapable(true)
            .setOverridesImplicitlyEnabledSubtype(true)
            .setSubtypeId(im + 100)
        if (Build.VERSION.SDK_INT >= 24) {
            builder.setLanguageTag("vi")
        }
        return builder.build()
    }

    @Volatile
    private var lastSyncedIm: Int = -1

    /** Sync the active input method label (Telex / VNI / VIQR) shown in Android System Settings. */
    @Suppress("DEPRECATION")
    fun syncSystemSubtype(ctx: Context, force: Boolean = false): InputMethodSubtype? {
        val im = inputMethod(ctx)
        if (!force && im == lastSyncedIm) return null
        return try {
            val imm = ctx.getSystemService(InputMethodManager::class.java) ?: return null
            val imi = imm.inputMethodList?.firstOrNull { it.packageName == ctx.packageName } ?: return null
            val subtype = subtypeForMethod(im)
            imm.setAdditionalInputMethodSubtypes(imi.id, arrayOf(subtype))
            lastSyncedIm = im
            subtype
        } catch (_: Exception) {
            null
        }
    }

    /** Apply persisted options onto a live engine. */
    fun applyTo(engine: Engine, ctx: Context) {
        val p = get(ctx)
        engine.setInputMethod(inputMethod(ctx))
        syncSystemSubtype(ctx)
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
    fun autoCapitalize(ctx: Context): Boolean =
        get(ctx).getBoolean(KEY_AUTO_CAPITALIZE, true)
    fun numberRow(ctx: Context): Boolean =
        get(ctx).getBoolean(KEY_NUMBER_ROW, false)
    fun clipboardSuggest(ctx: Context): Boolean =
        get(ctx).getBoolean(KEY_CLIPBOARD_SUGGEST, true)
    fun clipboardHistory(ctx: Context): Boolean =
        get(ctx).getBoolean(KEY_CLIPBOARD_HISTORY, true)
    fun clipboardHistoryMax(ctx: Context): Int =
        get(ctx).getInt(KEY_CLIPBOARD_MAX, 50).coerceIn(5, 2000)
    fun keyHeightDp(ctx: Context): Int = get(ctx).getInt(KEY_KEY_HEIGHT, 56)
    /** Landscape key height; -1 = auto (compact default). */
    fun landKeyHeightDp(ctx: Context): Int = get(ctx).getInt(KEY_LAND_KEY_HEIGHT, -1)
    fun landSplit(ctx: Context): Boolean = get(ctx).getBoolean(KEY_LAND_SPLIT, true)
    fun landSplitGapDp(ctx: Context): Int = get(ctx).getInt(KEY_LAND_SPLIT_GAP, 40)
    /** Hold Backspace → extend selection leftward until release (Gboard-style). */
    fun holdDelSelect(ctx: Context): Boolean = get(ctx).getBoolean(KEY_HOLD_DEL_SELECT, false)
    fun haptic(ctx: Context): Boolean = get(ctx).getBoolean(KEY_HAPTIC, true)
    fun keyPopup(ctx: Context): Boolean = get(ctx).getBoolean(KEY_KEY_POPUP, true)
    fun keyGapDp(ctx: Context): Int = get(ctx).getInt(KEY_KEY_GAP, 4)

    // ---- per-app language memory -------------------------------------------

    /** packageName -> vietnameseOn. Unknown packages fall back to KEY_VIET_ON. */
    fun vietForApp(ctx: Context, pkg: String?): Boolean? {
        if (pkg == null) return null
        val json = get(ctx).getString(KEY_VIET_PER_APP, null) ?: return null
        return try {
            val obj = JSONObject(json)
            if (obj.has(pkg)) obj.getBoolean(pkg) else null
        } catch (_: Exception) { null }
    }

    /** Remember the toggle per host app (Windows-style per-app input state).
     *  The global KEY_VIET_ON is only the default for apps without a recorded
     *  choice — toggling inside one app must not change other apps. */
    fun setViet(ctx: Context, pkg: String?, on: Boolean) {
        if (pkg == null) return
        val p = get(ctx)
        val obj = try { JSONObject(p.getString(KEY_VIET_PER_APP, "{}") ?: "{}") }
            catch (_: Exception) { JSONObject() }
        obj.put(pkg, on)
        // Bound the map: drop everything once it grows silly-large.
        if (obj.length() > 300) {
            p.edit().putString(KEY_VIET_PER_APP,
                JSONObject().put(pkg, on).toString()).apply()
            return
        }
        p.edit().putString(KEY_VIET_PER_APP, obj.toString()).apply()
    }
}
