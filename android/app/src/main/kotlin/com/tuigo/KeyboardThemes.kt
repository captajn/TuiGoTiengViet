package com.tuigo

import android.content.Context
import android.content.res.Configuration

/** Keyboard skin modelled on Laban Key's bundled themes (colors sampled from
 *  its shipped theme snapshots/key assets). All drawn — no bitmap assets.
 *
 *  key/func are the letter and function key fills; border alpha 0 = borderless.
 *  accent is the shift-active/enter fill; onAccent its glyph color.
 *  blueGlyphs draws shift/enter glyphs in accent instead of filling the key
 *  (Laban default look). enterAccentFill paints Enter with accent instead.
 *  pressAccent paints a pressed letter in accent; otherwise keyPressed fill. */
class KeyTheme(
    val id: String,
    val label: String,
    val bg: Int,
    val bgEnd: Int,
    val key: Int,
    val keyPressed: Int,
    val func: Int,
    val funcPressed: Int,
    val border: Int,
    val cornerDp: Float,
    val text: Int,
    val funcText: Int,
    val accent: Int,
    val onAccent: Int,
    val space: Int,
    val spaceBorder: Int,
    val spaceText: Int,
    val blueGlyphs: Boolean = false,
    val enterAccentFill: Boolean = false,
    val pressAccent: Boolean = false,
    val popup: Int,
    val popupText: Int,
    val popupBorder: Int,
)

object KeyboardThemes {
    const val DEFAULT_ID = "tuigo"

    private val tuigoLight = KeyTheme(
        "tuigo", "Tui Gõ",
        bg = 0xFFF8FAFC.argb, bgEnd = 0xFFE2F0E7.argb,
        key = 0xFFF1F5F9.argb, keyPressed = 0xFF197F64.argb,
        func = 0xFFEFF3F6.argb, funcPressed = 0xFF197F64.argb,
        border = 0xFFE2E8F0.argb, cornerDp = 10f,
        text = 0xFF0F172A.argb, funcText = 0xFF0F172A.argb,
        accent = 0xFF197F64.argb, onAccent = 0xFFFFFFFF.argb,
        space = 0xFFEFF3F6.argb, spaceBorder = 0xFFE2E8F0.argb, spaceText = 0xFF64748B.argb,
        enterAccentFill = true, pressAccent = true,
        popup = 0xFFF1F5F9.argb, popupText = 0xFF0F172A.argb, popupBorder = 0xFFE2E8F0.argb,
    )
    private val tuigoDark = KeyTheme(
        "tuigo", "Tui Gõ",
        bg = 0xFF0F1115.argb, bgEnd = 0xFF243D3B.argb,
        key = 0xFF222630.argb, keyPressed = 0xFF3ED3B8.argb,
        func = 0xFF1C2027.argb, funcPressed = 0xFF3ED3B8.argb,
        border = 0xFF272B35.argb, cornerDp = 10f,
        text = 0xFFF8FAFC.argb, funcText = 0xFFF8FAFC.argb,
        accent = 0xFF3ED3B8.argb, onAccent = 0xFF0F1115.argb,
        space = 0xFF1C2027.argb, spaceBorder = 0xFF272B35.argb, spaceText = 0xFF94A3B8.argb,
        enterAccentFill = true, pressAccent = true,
        popup = 0xFF222630.argb, popupText = 0xFFF8FAFC.argb, popupBorder = 0xFF272B35.argb,
    )

    private val laban = KeyTheme(
        "laban", "Laban",
        bg = 0xFFE1E2E5.argb, bgEnd = 0xFFE1E2E5.argb,
        key = 0xFFFCFCFC.argb, keyPressed = 0xFFB8B9BE.argb,
        func = 0xFFE1E2E5.argb, funcPressed = 0xFFB8B9BE.argb,
        border = 0xFFB6BBC4.argb, cornerDp = 6f,
        text = 0xFF1B1E24.argb, funcText = 0xFF1B1E24.argb,
        accent = 0xFF3F8EEB.argb, onAccent = 0xFFFFFFFF.argb,
        space = 0xFFC7D4E6.argb, spaceBorder = 0xFF3F8EEB.argb, spaceText = 0xFF1B1E24.argb,
        blueGlyphs = true,
        popup = 0xFF758899.argb, popupText = 0xFFFFFFFF.argb, popupBorder = 0xFF5C6E7C.argb,
    )
    private val white = KeyTheme(
        "white", "Trắng",
        bg = 0xFFECEEF1.argb, bgEnd = 0xFFECEEF1.argb,
        key = 0xFFFFFFFF.argb, keyPressed = 0xFFC9CDD4.argb,
        func = 0xFFECEEF1.argb, funcPressed = 0xFFC9CDD4.argb,
        border = 0xFFD5D9E0.argb, cornerDp = 6f,
        text = 0xFF26292E.argb, funcText = 0xFF26292E.argb,
        accent = 0xFF3F8EEB.argb, onAccent = 0xFFFFFFFF.argb,
        space = 0xFFFFFFFF.argb, spaceBorder = 0xFFD5D9E0.argb, spaceText = 0xFF26292E.argb,
        popup = 0xFFFFFFFF.argb, popupText = 0xFF26292E.argb, popupBorder = 0xFFD5D9E0.argb,
    )
    private val black = KeyTheme(
        "black", "Đen",
        bg = 0xFF000000.argb, bgEnd = 0xFF000000.argb,
        key = 0xFF444444.argb, keyPressed = 0xFF5E5E5E.argb,
        func = 0xFF222222.argb, funcPressed = 0xFF3A3A3A.argb,
        border = 0xFF1A1A1A.argb, cornerDp = 6f,
        text = 0xFFFFFFFF.argb, funcText = 0xFFFFFFFF.argb,
        accent = 0xFF3F8EEB.argb, onAccent = 0xFFFFFFFF.argb,
        space = 0xFF444444.argb, spaceBorder = 0xFF1A1A1A.argb, spaceText = 0xFFFFFFFF.argb,
        popup = 0xFF333333.argb, popupText = 0xFFFFFFFF.argb, popupBorder = 0xFF4A4A4A.argb,
    )
    private val gray = KeyTheme(
        "gray", "Xám",
        bg = 0xFF181F2D.argb, bgEnd = 0xFF181F2D.argb,
        key = 0xFF53616C.argb, keyPressed = 0xFF6B7B88.argb,
        func = 0xFF374146.argb, funcPressed = 0xFF4A555E.argb,
        border = 0, cornerDp = 6f,
        text = 0xFFFFFFFF.argb, funcText = 0xFFFFFFFF.argb,
        accent = 0xFF7FB3E8.argb, onAccent = 0xFFFFFFFF.argb,
        space = 0xFF53616C.argb, spaceBorder = 0, spaceText = 0xFFFFFFFF.argb,
        popup = 0xFF53616C.argb, popupText = 0xFFFFFFFF.argb, popupBorder = 0xFF374146.argb,
    )
    private val blue = KeyTheme(
        "blue", "Xanh đậm",
        bg = 0xFF1C2331.argb, bgEnd = 0xFF1C2331.argb,
        key = 0xFF26324B.argb, keyPressed = 0xFF3A4A6B.argb,
        func = 0xFF30436A.argb, funcPressed = 0xFF425381.argb,
        border = 0, cornerDp = 6f,
        text = 0xFFFFFFFF.argb, funcText = 0xFFD7E0F4.argb,
        accent = 0xFF8FA9DE.argb, onAccent = 0xFF1C2331.argb,
        space = 0xFF26324B.argb, spaceBorder = 0, spaceText = 0xFFD7E0F4.argb,
        blueGlyphs = true,
        popup = 0xFF26324B.argb, popupText = 0xFFFFFFFF.argb, popupBorder = 0xFF30436A.argb,
    )
    private val ios = KeyTheme(
        "ios", "iOS",
        bg = 0xFFD1D5DB.argb, bgEnd = 0xFFD1D5DB.argb,
        key = 0xFFFFFFFF.argb, keyPressed = 0xFFC9CDD3.argb,
        func = 0xFFA4ACB5.argb, funcPressed = 0xFF8A939E.argb,
        border = 0, cornerDp = 6f,
        text = 0xFF000000.argb, funcText = 0xFF26292D.argb,
        accent = 0xFF8A939E.argb, onAccent = 0xFF26292D.argb,
        space = 0xFFFFFFFF.argb, spaceBorder = 0, spaceText = 0xFF26292D.argb,
        popup = 0xFFFFFFFF.argb, popupText = 0xFF000000.argb, popupBorder = 0xFFC9CDD3.argb,
    )
    private val iosOld = KeyTheme(
        "ios_old", "iOS cổ điển",
        bg = 0xFF828A97.argb, bgEnd = 0xFF464F5E.argb,
        key = 0xFFECECEF.argb, keyPressed = 0xFFFFFFFF.argb,
        func = 0xFF636B78.argb, funcPressed = 0xFF7A828E.argb,
        border = 0xFF4E5560.argb, cornerDp = 8f,
        text = 0xFF2B3038.argb, funcText = 0xFFF4F5F6.argb,
        accent = 0xFF2B4E8F.argb, onAccent = 0xFFFFFFFF.argb,
        space = 0xFFE7E7E9.argb, spaceBorder = 0xFF4E5560.argb, spaceText = 0xFF2B3038.argb,
        popup = 0xFFE6E8E9.argb, popupText = 0xFF2B3038.argb, popupBorder = 0xFF8A9099.argb,
    )
    private val ics = KeyTheme(
        "ics", "ICS",
        bg = 0xFF111111.argb, bgEnd = 0xFF000000.argb,
        key = 0xFF555555.argb, keyPressed = 0xFF777777.argb,
        func = 0xFF222222.argb, funcPressed = 0xFF444444.argb,
        border = 0, cornerDp = 2f,
        text = 0xFFFFFFFF.argb, funcText = 0xFFFFFFFF.argb,
        accent = 0xFF33B5E6.argb, onAccent = 0xFF000000.argb,
        space = 0xFF494949.argb, spaceBorder = 0, spaceText = 0xFFFFFFFF.argb,
        popup = 0xFF33B5E6.argb, popupText = 0xFF000000.argb, popupBorder = 0xFF33B5E6.argb,
    )
    private val kitkat = KeyTheme(
        "kitkat", "KitKat",
        bg = 0xFF111111.argb, bgEnd = 0xFF000000.argb,
        key = 0xFF555555.argb, keyPressed = 0xFF777777.argb,
        func = 0xFF222222.argb, funcPressed = 0xFF444444.argb,
        border = 0, cornerDp = 2f,
        text = 0xFFFFFFFF.argb, funcText = 0xFFFFFFFF.argb,
        accent = 0xFF33B5E6.argb, onAccent = 0xFF000000.argb,
        space = 0xFF494949.argb, spaceBorder = 0, spaceText = 0xFFFFFFFF.argb,
        popup = 0xFF6B6B6B.argb, popupText = 0xFFFFFFFF.argb, popupBorder = 0xFF6B6B6B.argb,
    )
    private val silver = KeyTheme(
        "silver", "Bạc",
        bg = 0xFFC8C8C8.argb, bgEnd = 0xFFC8C8C8.argb,
        key = 0xFFE7E6E4.argb, keyPressed = 0xFFF5F4F2.argb,
        func = 0xFF515151.argb, funcPressed = 0xFF636161.argb,
        border = 0xFF9E9B98.argb, cornerDp = 6f,
        text = 0xFF3A3735.argb, funcText = 0xFFFFFFFF.argb,
        accent = 0xFF515151.argb, onAccent = 0xFFFFFFFF.argb,
        space = 0xFFE7E6E4.argb, spaceBorder = 0xFF9E9B98.argb, spaceText = 0xFF3A3735.argb,
        popup = 0xFFE7E6E4.argb, popupText = 0xFF3A3735.argb, popupBorder = 0xFF9E9B98.argb,
    )
    private val wood = KeyTheme(
        "wood", "Gỗ",
        bg = 0xFFD7B381.argb, bgEnd = 0xFFC09063.argb,
        key = 0xFFE1CDB5.argb, keyPressed = 0xFFEFE2D2.argb,
        func = 0xFFA08A73.argb, funcPressed = 0xFF8A7358.argb,
        border = 0xFF96785B.argb, cornerDp = 5f,
        text = 0xFF4A3B2C.argb, funcText = 0xFFFFFFFF.argb,
        accent = 0xFF8A7358.argb, onAccent = 0xFFFFFFFF.argb,
        space = 0xFFDFC9B1.argb, spaceBorder = 0xFF96785B.argb, spaceText = 0xFF4A3B2C.argb,
        popup = 0xFF1A1512.argb, popupText = 0xFFEAD9C0.argb, popupBorder = 0xFF96785B.argb,
    )
    private val dark = KeyTheme(
        "dark", "Tối",
        bg = 0xFF303033.argb, bgEnd = 0xFF0B0B0D.argb,
        key = 0xFF2F2F31.argb, keyPressed = 0xFF4A4A4E.argb,
        func = 0xFF181818.argb, funcPressed = 0xFF2A2A2C.argb,
        border = 0, cornerDp = 4f,
        text = 0xFFFCFCFC.argb, funcText = 0xFFFCFCFC.argb,
        accent = 0xFF5B5B5B.argb, onAccent = 0xFFFFFFFF.argb,
        space = 0xFF2D2D2F.argb, spaceBorder = 0, spaceText = 0xFFFCFCFC.argb,
        popup = 0xFF222222.argb, popupText = 0xFFFFFFFF.argb, popupBorder = 0xFF444444.argb,
    )
    private val nLight = KeyTheme(
        "n_light", "Nougat sáng",
        bg = 0xFFF7F6F9.argb, bgEnd = 0xFFF7F6F9.argb,
        key = 0xFFF7F6F9.argb, keyPressed = 0xFFE4E3E8.argb,
        func = 0xFFF7F6F9.argb, funcPressed = 0xFFDDDCE1.argb,
        border = 0, cornerDp = 10f,
        text = 0xFF263238.argb, funcText = 0xFF5F6B72.argb,
        accent = 0xFF4DB6AC.argb, onAccent = 0xFFFFFFFF.argb,
        space = 0xFFE1E0E3.argb, spaceBorder = 0, spaceText = 0xFF263238.argb,
        enterAccentFill = true,
        popup = 0xFFD1D6D9.argb, popupText = 0xFF263238.argb, popupBorder = 0xFFD1D6D9.argb,
    )
    private val nDark = KeyTheme(
        "n_dark", "Nougat tối",
        bg = 0xFF263238.argb, bgEnd = 0xFF263238.argb,
        key = 0xFF263238.argb, keyPressed = 0xFF37474F.argb,
        func = 0xFF263238.argb, funcPressed = 0xFF37474F.argb,
        border = 0, cornerDp = 10f,
        text = 0xFFECEFF1.argb, funcText = 0xFF9FB0B8.argb,
        accent = 0xFF4DB6AC.argb, onAccent = 0xFFFFFFFF.argb,
        space = 0xFF3C474C.argb, spaceBorder = 0, spaceText = 0xFFECEFF1.argb,
        enterAccentFill = true,
        popup = 0xFF3D484D.argb, popupText = 0xFFECEFF1.argb, popupBorder = 0xFF3D484D.argb,
    )

    /** Order shown in the settings picker. "adaptive" resolves to
     *  laban in light mode, gray in dark mode. */
    private val light = listOf(tuigoLight, laban, white, ios, iosOld, silver, wood, nLight, ics, kitkat, dark, black, gray, blue, nDark)
    private val darkThemes = listOf(tuigoDark, laban, white, ios, iosOld, silver, wood, nLight, ics, kitkat, dark, black, gray, blue, nDark)

    val ids = listOf("tuigo", "adaptive", "laban", "white", "black", "gray", "blue", "ios", "ios_old", "ics", "kitkat", "silver", "wood", "dark", "n_light", "n_dark")

    fun label(id: String): String = when (id) {
        "adaptive" -> "Theo hệ thống"
        else -> all(false).firstOrNull { it.id == id }?.label ?: id
    }

    fun all(darkMode: Boolean): List<KeyTheme> = if (darkMode) darkThemes else light

    fun get(id: String, darkMode: Boolean): KeyTheme = when (id) {
        "adaptive" -> if (darkMode) gray else laban
        else -> all(darkMode).firstOrNull { it.id == id }
            ?: all(darkMode).first { it.id == DEFAULT_ID }
    }

    fun current(ctx: Context): KeyTheme =
        get(Prefs.get(ctx).getString(Prefs.KEY_THEME, DEFAULT_ID) ?: DEFAULT_ID, isDark(ctx))

    private fun isDark(ctx: Context): Boolean =
        ctx.resources.configuration.uiMode and Configuration.UI_MODE_NIGHT_MASK ==
            Configuration.UI_MODE_NIGHT_YES
}

private val Long.argb: Int get() = this.toInt()
private val Int.argb: Int get() = this
