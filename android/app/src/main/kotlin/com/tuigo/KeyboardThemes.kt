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
        bg = 0xFFF8FAFC.argb, bgEnd = 0xFFF1F5F9.argb,
        key = 0xFFFFFFFF.argb, keyPressed = 0xFFE2E8F0.argb,
        func = 0xFFEFF3F6.argb, funcPressed = 0xFFCBD5E1.argb,
        border = 0xFFE2E8F0.argb, cornerDp = 10f,
        text = 0xFF0F172A.argb, funcText = 0xFF0F172A.argb,
        accent = 0xFF197F64.argb, onAccent = 0xFFFFFFFF.argb,
        space = 0xFFFFFFFF.argb, spaceBorder = 0xFFE2E8F0.argb, spaceText = 0xFF64748B.argb,
        enterAccentFill = true, pressAccent = false,
        popup = 0xFFFFFFFF.argb, popupText = 0xFF0F172A.argb, popupBorder = 0xFFE2E8F0.argb,
    )
    private val tuigoDark = KeyTheme(
        "tuigo", "Tui Gõ",
        bg = 0xFF0F1115.argb, bgEnd = 0xFF161922.argb,
        key = 0xFF232834.argb, keyPressed = 0xFF353C4D.argb,
        func = 0xFF1A1E27.argb, funcPressed = 0xFF2B3241.argb,
        border = 0xFF2E3545.argb, cornerDp = 10f,
        text = 0xFFF8FAFC.argb, funcText = 0xFFF8FAFC.argb,
        accent = 0xFF3ED3B8.argb, onAccent = 0xFF0F1115.argb,
        space = 0xFF232834.argb, spaceBorder = 0xFF2E3545.argb, spaceText = 0xFF94A3B8.argb,
        enterAccentFill = true, pressAccent = false,
        popup = 0xFF232834.argb, popupText = 0xFFF8FAFC.argb, popupBorder = 0xFF2E3545.argb,
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

    private val modernJadeLight = KeyTheme(
        "modern_jade", "Modern Jade",
        bg = 0xFFE6F7F5.argb, bgEnd = 0xFFCCFBF1.argb,
        key = 0xFFFFFFFF.argb, keyPressed = 0xFFCCFBF1.argb,
        func = 0xFFCCFBF1.argb, funcPressed = 0xFF99F6E4.argb,
        border = 0xFF99E6D8.argb, cornerDp = 12f,
        text = 0xFF0D2B28.argb, funcText = 0xFF0F766E.argb,
        accent = 0xFF0D9488.argb, onAccent = 0xFFFFFFFF.argb,
        space = 0xFFFFFFFF.argb, spaceBorder = 0xFF99E6D8.argb, spaceText = 0xFF0F766E.argb,
        enterAccentFill = true, pressAccent = false,
        popup = 0xFFFFFFFF.argb, popupText = 0xFF0D2B28.argb, popupBorder = 0xFF99E6D8.argb,
    )
    private val modernJadeDark = KeyTheme(
        "modern_jade", "Modern Jade",
        bg = 0xFF0D1614.argb, bgEnd = 0xFF142420.argb,
        key = 0xFF1B2E29.argb, keyPressed = 0xFF26403A.argb,
        func = 0xFF14221E.argb, funcPressed = 0xFF1D332D.argb,
        border = 0xFF254039.argb, cornerDp = 12f,
        text = 0xFFF0FDF4.argb, funcText = 0xFFD1FAE5.argb,
        accent = 0xFF10B981.argb, onAccent = 0xFF064E3B.argb,
        space = 0xFF1B2E29.argb, spaceBorder = 0xFF254039.argb, spaceText = 0xFFD1FAE5.argb,
        enterAccentFill = true, pressAccent = false,
        popup = 0xFF1B2E29.argb, popupText = 0xFFF0FDF4.argb, popupBorder = 0xFF254039.argb,
    )
    private val iosPureLight = KeyTheme(
        "ios_pure", "iOS Hiện đại",
        bg = 0xFFD1D5DB.argb, bgEnd = 0xFFCBD0D8.argb,
        key = 0xFFFFFFFF.argb, keyPressed = 0xFFE2E6EB.argb,
        func = 0xFFAEB5BF.argb, funcPressed = 0xFF98A0AC.argb,
        border = 0, cornerDp = 9f,
        text = 0xFF000000.argb, funcText = 0xFF1F2937.argb,
        accent = 0xFF007AFF.argb, onAccent = 0xFFFFFFFF.argb,
        space = 0xFFFFFFFF.argb, spaceBorder = 0, spaceText = 0xFF4B5563.argb,
        enterAccentFill = true, pressAccent = false,
        popup = 0xFFFFFFFF.argb, popupText = 0xFF000000.argb, popupBorder = 0xFFC5C9D0.argb,
    )
    private val iosPureDark = KeyTheme(
        "ios_pure", "iOS Hiện đại",
        bg = 0xFF1C1C1E.argb, bgEnd = 0xFF121214.argb,
        key = 0xFF3A3A3C.argb, keyPressed = 0xFF545458.argb,
        func = 0xFF2C2C2E.argb, funcPressed = 0xFF48484A.argb,
        border = 0, cornerDp = 9f,
        text = 0xFFFFFFFF.argb, funcText = 0xFFEBEBF5.argb,
        accent = 0xFF0A84FF.argb, onAccent = 0xFFFFFFFF.argb,
        space = 0xFF3A3A3C.argb, spaceBorder = 0, spaceText = 0xFFAEAEB2.argb,
        enterAccentFill = true, pressAccent = false,
        popup = 0xFF3A3A3C.argb, popupText = 0xFFFFFFFF.argb, popupBorder = 0xFF545458.argb,
    )
    private val obsidian = KeyTheme(
        "obsidian", "Obsidian",
        bg = 0xFF000000.argb, bgEnd = 0xFF090A0D.argb,
        key = 0xFF16181F.argb, keyPressed = 0xFF262A36.argb,
        func = 0xFF0E0F14.argb, funcPressed = 0xFF1E212B.argb,
        border = 0xFF222633.argb, cornerDp = 10f,
        text = 0xFFF8FAFC.argb, funcText = 0xFF94A3B8.argb,
        accent = 0xFF3ED3B8.argb, onAccent = 0xFF000000.argb,
        space = 0xFF16181F.argb, spaceBorder = 0xFF222633.argb, spaceText = 0xFF94A3B8.argb,
        enterAccentFill = true, pressAccent = false,
        popup = 0xFF16181F.argb, popupText = 0xFFF8FAFC.argb, popupBorder = 0xFF222633.argb,
    )
    private val pastelMint = KeyTheme(
        "pastel_mint", "Pastel Mint",
        bg = 0xFFF0FDF4.argb, bgEnd = 0xFFDCFCE7.argb,
        key = 0xFFFFFFFF.argb, keyPressed = 0xFFD1FAE5.argb,
        func = 0xFFE1F8EC.argb, funcPressed = 0xFFC6F2DD.argb,
        border = 0xFFC1EBD6.argb, cornerDp = 10f,
        text = 0xFF064E3B.argb, funcText = 0xFF065F46.argb,
        accent = 0xFF059669.argb, onAccent = 0xFFFFFFFF.argb,
        space = 0xFFFFFFFF.argb, spaceBorder = 0xFFC1EBD6.argb, spaceText = 0xFF065F46.argb,
        enterAccentFill = true, pressAccent = false,
        popup = 0xFFFFFFFF.argb, popupText = 0xFF064E3B.argb, popupBorder = 0xFFC1EBD6.argb,
    )

    private val cyberNeonDark = KeyTheme(
        "cyber_neon", "Cyber Neon",
        bg = 0xFF04060A.argb, bgEnd = 0xFF0B101E.argb,
        key = 0xFF0B111E.argb, keyPressed = 0xFF16223B.argb,
        func = 0xFF080D18.argb, funcPressed = 0xFF142038.argb,
        border = 0xFF0EA5E9.argb, cornerDp = 8f,
        text = 0xFF38BDF8.argb, funcText = 0xFF7DD3FC.argb,
        accent = 0xFF10B981.argb, onAccent = 0xFF04060A.argb,
        space = 0xFF0B111E.argb, spaceBorder = 0xFF0EA5E9.argb, spaceText = 0xFF38BDF8.argb,
        enterAccentFill = true, pressAccent = false,
        popup = 0xFF0B111E.argb, popupText = 0xFF38BDF8.argb, popupBorder = 0xFF0EA5E9.argb,
    )
    private val cyberNeonLight = KeyTheme(
        "cyber_neon", "Cyber Neon",
        bg = 0xFFE0F2FE.argb, bgEnd = 0xFFBAE6FD.argb,
        key = 0xFFFFFFFF.argb, keyPressed = 0xFFE0F2FE.argb,
        func = 0xFFBAE6FD.argb, funcPressed = 0xFF7DD3FC.argb,
        border = 0xFF0284C7.argb, cornerDp = 8f,
        text = 0xFF082F49.argb, funcText = 0xFF0369A1.argb,
        accent = 0xFF10B981.argb, onAccent = 0xFFFFFFFF.argb,
        space = 0xFFFFFFFF.argb, spaceBorder = 0xFF0284C7.argb, spaceText = 0xFF0369A1.argb,
        enterAccentFill = true, pressAccent = false,
        popup = 0xFFFFFFFF.argb, popupText = 0xFF082F49.argb, popupBorder = 0xFF0284C7.argb,
    )
    private val matchaLatteLight = KeyTheme(
        "matcha_latte", "Matcha Latte",
        bg = 0xFFEDE4D8.argb, bgEnd = 0xFFF5ECE0.argb,
        key = 0xFFFFFDF9.argb, keyPressed = 0xFFEFE8DD.argb,
        func = 0xFFA8D5BA.argb, funcPressed = 0xFF8FC4A3.argb,
        border = 0xFFD8C7B5.argb, cornerDp = 12f,
        text = 0xFF4A3B32.argb, funcText = 0xFF1B432C.argb,
        accent = 0xFFF7C59F.argb, onAccent = 0xFF542E13.argb,
        space = 0xFFFFFDF9.argb, spaceBorder = 0xFFD8C7B5.argb, spaceText = 0xFF4A3B32.argb,
        enterAccentFill = true, pressAccent = false,
        popup = 0xFFFFFDF9.argb, popupText = 0xFF4A3B32.argb, popupBorder = 0xFFD8C7B5.argb,
    )
    private val matchaLatteDark = KeyTheme(
        "matcha_latte", "Matcha Latte",
        bg = 0xFF121A16.argb, bgEnd = 0xFF1B2822.argb,
        key = 0xFF24352C.argb, keyPressed = 0xFF334C3F.argb,
        func = 0xFF1B2822.argb, funcPressed = 0xFF2D4439.argb,
        border = 0xFF385446.argb, cornerDp = 12f,
        text = 0xFFF4ECE1.argb, funcText = 0xFF95B8A6.argb,
        accent = 0xFF74C69D.argb, onAccent = 0xFF121A16.argb,
        space = 0xFF24352C.argb, spaceBorder = 0xFF385446.argb, spaceText = 0xFFF4ECE1.argb,
        enterAccentFill = true, pressAccent = false,
        popup = 0xFF24352C.argb, popupText = 0xFFF4ECE1.argb, popupBorder = 0xFF52B788.argb,
    )

    /** Order shown in the settings picker. "adaptive" resolves to
     *  laban in light mode, gray in dark mode. */
    private val light = listOf(tuigoLight, iosPureLight, cyberNeonLight, matchaLatteLight, modernJadeLight, obsidian, pastelMint, laban, white, ios, iosOld, silver, wood, nLight, ics, kitkat, dark, black, gray, blue, nDark)
    private val darkThemes = listOf(tuigoDark, iosPureDark, cyberNeonDark, matchaLatteDark, modernJadeDark, obsidian, pastelMint, laban, white, ios, iosOld, silver, wood, nLight, ics, kitkat, dark, black, gray, blue, nDark)

    val ids = listOf("tuigo", "adaptive", "ios_pure", "cyber_neon", "matcha_latte", "modern_jade", "obsidian", "pastel_mint", "laban", "white", "black", "gray", "blue", "ios", "ios_old", "ics", "kitkat", "silver", "wood", "dark", "n_light", "n_dark")

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
