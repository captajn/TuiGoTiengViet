package com.tuigo

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

internal object ThemePresets {

    val tuigoLight = KeyTheme(
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
    val tuigoDark = KeyTheme(
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
    val laban = KeyTheme(
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
    val white = KeyTheme(
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
    val black = KeyTheme(
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
    val gray = KeyTheme(
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
    val blue = KeyTheme(
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
    val ios = KeyTheme(
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
    val iosOld = KeyTheme(
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
    val ics = KeyTheme(
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
    val kitkat = KeyTheme(
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
    val silver = KeyTheme(
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
    val wood = KeyTheme(
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
    val dark = KeyTheme(
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
    val nLight = KeyTheme(
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
    val nDark = KeyTheme(
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
    val modernJadeLight = KeyTheme(
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
    val modernJadeDark = KeyTheme(
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
    val iosPureLight = KeyTheme(
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
    val iosPureDark = KeyTheme(
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
    val obsidian = KeyTheme(
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
    val pastelMint = KeyTheme(
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
    val cyberNeonDark = KeyTheme(
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
    val cyberNeonLight = KeyTheme(
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
    val matchaLatteLight = KeyTheme(
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
    val matchaLatteDark = KeyTheme(
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

    val materialLight = KeyTheme(
        "material", "Material",
        bg = 0xFFEBECF0.argb, bgEnd = 0xFFE1E2E5.argb,
        key = 0xFFFFFFFF.argb, keyPressed = 0xFFD4D5D9.argb,
        func = 0xFFF7F7F7.argb, funcPressed = 0xFFD4D5D9.argb,
        border = 0xFFBCC2D6.argb, cornerDp = 10f,
        text = 0xFF191B21.argb, funcText = 0xFF484B54.argb,
        accent = 0xFF0A1633.argb, onAccent = 0xFFF2F6FF.argb,
        space = 0xFFFFFFFF.argb, spaceBorder = 0xFFBCC2D6.argb, spaceText = 0xFF484B54.argb,
        enterAccentFill = true,
        popup = 0xFFFFFFFF.argb, popupText = 0xFF191B21.argb, popupBorder = 0xFFBCC2D6.argb,
    )
    val materialDark = KeyTheme(
        "material", "Material",
        bg = 0xFF121316.argb, bgEnd = 0xFF0D0D0F.argb,
        key = 0xFF1E2024.argb, keyPressed = 0xFF3C3F47.argb,
        func = 0xFF17181C.argb, funcPressed = 0xFF3C3F47.argb,
        border = 0xFF1D1E24.argb, cornerDp = 10f,
        text = 0xFFE6E6EE.argb, funcText = 0xFFBBBDBF.argb,
        accent = 0xFFB2C8FF.argb, onAccent = 0xFF131D36.argb,
        space = 0xFF1E2024.argb, spaceBorder = 0xFF1D1E24.argb, spaceText = 0xFFBBBDBF.argb,
        enterAccentFill = true,
        popup = 0xFF1E2024.argb, popupText = 0xFFE6E6EE.argb, popupBorder = 0xFF3C3F47.argb,
    )

    val catppuccin = KeyTheme(
        "catppuccin", "Catppuccin Mocha",
        bg = 0xFF11111B.argb, bgEnd = 0xFF181825.argb,
        key = 0xFF1E1E2E.argb, keyPressed = 0xFF313244.argb,
        func = 0xFF181825.argb, funcPressed = 0xFF313244.argb,
        border = 0xFF45475A.argb, cornerDp = 10f,
        text = 0xFFCDD6F4.argb, funcText = 0xFFBAC2DE.argb,
        accent = 0xFFF4B8E4.argb, onAccent = 0xFF2A2430.argb,
        space = 0xFF1E1E2E.argb, spaceBorder = 0xFF45475A.argb, spaceText = 0xFFBAC2DE.argb,
        enterAccentFill = true,
        popup = 0xFF1E1E2E.argb, popupText = 0xFFCDD6F4.argb, popupBorder = 0xFF45475A.argb,
    )

    val emerald = KeyTheme(
        "emerald", "Emerald",
        bg = 0xFF0D1A0B.argb, bgEnd = 0xFF0A1409.argb,
        key = 0xFF162D13.argb, keyPressed = 0xFF24521E.argb,
        func = 0xFF0F210D.argb, funcPressed = 0xFF24521E.argb,
        border = 0xFF344A2A.argb, cornerDp = 10f,
        text = 0xFFDCE5D5.argb, funcText = 0xFFA8B3A2.argb,
        accent = 0xFF67FB59.argb, onAccent = 0xFF112E10.argb,
        space = 0xFF162D13.argb, spaceBorder = 0xFF344A2A.argb, spaceText = 0xFFA8B3A2.argb,
        enterAccentFill = true,
        popup = 0xFF162D13.argb, popupText = 0xFFDCE5D5.argb, popupBorder = 0xFF344A2A.argb,
    )

    val amoledPurple = KeyTheme(
        "amoled_purple", "AMOLED Purple",
        bg = 0xFF000000.argb, bgEnd = 0xFF000000.argb,
        key = 0xFF1E192B.argb, keyPressed = 0xFF31264F.argb,
        func = 0xFF181324.argb, funcPressed = 0xFF31264F.argb,
        border = 0xFF3B2D4F.argb, cornerDp = 10f,
        text = 0xFFE6E1E5.argb, funcText = 0xFFCCC1D6.argb,
        accent = 0xFFD0BCFF.argb, onAccent = 0xFF381E72.argb,
        space = 0xFF1E192B.argb, spaceBorder = 0xFF3B2D4F.argb, spaceText = 0xFFCCC1D6.argb,
        enterAccentFill = true,
        popup = 0xFF1E192B.argb, popupText = 0xFFE6E1E5.argb, popupBorder = 0xFF3B2D4F.argb,
    )

    val deepSeaLight = KeyTheme(
        "deep_sea", "Deep Sea",
        bg = 0xFFBACAFF.argb, bgEnd = 0xFFAFBEF0.argb,
        key = 0xFFE8EDFF.argb, keyPressed = 0xFFA6B9FF.argb,
        func = 0xFFD8E1FE.argb, funcPressed = 0xFFA6B9FF.argb,
        border = 0xFFABB3CC.argb, cornerDp = 10f,
        text = 0xFF1B1D22.argb, funcText = 0xFF363840.argb,
        accent = 0xFF314BA1.argb, onAccent = 0xFFFFFFFF.argb,
        space = 0xFFE8EDFF.argb, spaceBorder = 0xFFABB3CC.argb, spaceText = 0xFF363840.argb,
        enterAccentFill = true,
        popup = 0xFFE8EDFF.argb, popupText = 0xFF1B1D22.argb, popupBorder = 0xFFABB3CC.argb,
    )
    val deepSeaDark = KeyTheme(
        "deep_sea", "Deep Sea",
        bg = 0xFF0F1026.argb, bgEnd = 0xFF0E0E1C.argb,
        key = 0xFF1B1D42.argb, keyPressed = 0xFF2A2D66.argb,
        func = 0xFF16163B.argb, funcPressed = 0xFF2A2D66.argb,
        border = 0xFF1B1B29.argb, cornerDp = 10f,
        text = 0xFFE3E1EC.argb, funcText = 0xFFAEAEBF.argb,
        accent = 0xFFBEC2FF.argb, onAccent = 0xFF131636.argb,
        space = 0xFF1B1D42.argb, spaceBorder = 0xFF1B1B29.argb, spaceText = 0xFFAEAEBF.argb,
        enterAccentFill = true,
        popup = 0xFF1B1D42.argb, popupText = 0xFFE3E1EC.argb, popupBorder = 0xFF2A2D66.argb,
    )

    val steelGrey = KeyTheme(
        "steel_grey", "Steel Grey",
        bg = 0xFF2E2E2E.argb, bgEnd = 0xFF262626.argb,
        key = 0xFF454545.argb, keyPressed = 0xFF585B5E.argb,
        func = 0xFF383838.argb, funcPressed = 0xFF585B5E.argb,
        border = 0xFF545859.argb, cornerDp = 10f,
        text = 0xFFE5F2FF.argb, funcText = 0xFFC7C7C7.argb,
        accent = 0xFFBBD5F0.argb, onAccent = 0xFF393E47.argb,
        space = 0xFF454545.argb, spaceBorder = 0xFF545859.argb, spaceText = 0xFFC7C7C7.argb,
        enterAccentFill = true,
        popup = 0xFF454545.argb, popupText = 0xFFE5F2FF.argb, popupBorder = 0xFF545859.argb,
    )

    val sunflower = KeyTheme(
        "sunflower", "Sunflower",
        bg = 0xFFEDDCBB.argb, bgEnd = 0xFFE0D3B8.argb,
        key = 0xFFFFF8DE.argb, keyPressed = 0xFFFFEDAB.argb,
        func = 0xFFFFF4C9.argb, funcPressed = 0xFFFFEDAB.argb,
        border = 0xFFCDC09E.argb, cornerDp = 10f,
        text = 0xFF1E1C13.argb, funcText = 0xFF4A4840.argb,
        accent = 0xFF6D5E0F.argb, onAccent = 0xFFFFFFFF.argb,
        space = 0xFFFFF8DE.argb, spaceBorder = 0xFFCDC09E.argb, spaceText = 0xFF4A4840.argb,
        enterAccentFill = true,
        popup = 0xFFFFF8DE.argb, popupText = 0xFF1E1C13.argb, popupBorder = 0xFFCDC09E.argb,
    )

    val snowfall = KeyTheme(
        "snowfall", "Snowfall",
        bg = 0xFFF8F8F8.argb, bgEnd = 0xFFF0F0F0.argb,
        key = 0xFFE5E5E5.argb, keyPressed = 0xFFBABCBF.argb,
        func = 0xFFF0F0F0.argb, funcPressed = 0xFFBABCBF.argb,
        border = 0xFFB8BDBF.argb, cornerDp = 10f,
        text = 0xFF17181A.argb, funcText = 0xFF404344.argb,
        accent = 0xFF1D2023.argb, onAccent = 0xFFFFFFFF.argb,
        space = 0xFFE5E5E5.argb, spaceBorder = 0xFFB8BDBF.argb, spaceText = 0xFF404344.argb,
        enterAccentFill = true,
        popup = 0xFFE5E5E5.argb, popupText = 0xFF17181A.argb, popupBorder = 0xFFB8BDBF.argb,
    )

    val cottonCandy = KeyTheme(
        "cotton_candy", "Cotton Candy",
        bg = 0xFFFFBAE9.argb, bgEnd = 0xFFF0B6DD.argb,
        key = 0xFFFFD9F8.argb, keyPressed = 0xFFF29DE3.argb,
        func = 0xFFFFCCF6.argb, funcPressed = 0xFFF29DE3.argb,
        border = 0xFFD5A8C5.argb, cornerDp = 10f,
        text = 0xFF21161E.argb, funcText = 0xFF40383D.argb,
        accent = 0xFFB02378.argb, onAccent = 0xFFFFFFFF.argb,
        space = 0xFFFFD9F8.argb, spaceBorder = 0xFFD5A8C5.argb, spaceText = 0xFF40383D.argb,
        enterAccentFill = true,
        popup = 0xFFFFD9F8.argb, popupText = 0xFF21161E.argb, popupBorder = 0xFFD5A8C5.argb,
    )

    val highContrast = KeyTheme(
        "high_contrast", "High Contrast",
        bg = 0xFF000000.argb, bgEnd = 0xFF000000.argb,
        key = 0xFFFFFF00.argb, keyPressed = 0xFFB9B9B9.argb,
        func = 0xFFFFFFFF.argb, funcPressed = 0xFFB9B9B9.argb,
        border = 0xFFFFFFFF.argb, cornerDp = 6f,
        text = 0xFF000000.argb, funcText = 0xFF000000.argb,
        accent = 0xFFFFFF00.argb, onAccent = 0xFF000000.argb,
        space = 0xFFFFFF00.argb, spaceBorder = 0xFFFFFFFF.argb, spaceText = 0xFF000000.argb,
        popup = 0xFF000000.argb, popupText = 0xFFFFFF00.argb, popupBorder = 0xFFFFFFFF.argb,
    )
}

private val Long.argb: Int get() = this.toInt()
private val Int.argb: Int get() = this
