package com.tuigo

import android.content.Context
import android.content.res.Configuration

/** Registry + lookup for keyboard skins. Theme definitions live in
 *  KeyboardThemePresets.kt — this file only maps ids and system dark mode. */
object KeyboardThemes {
    const val DEFAULT_ID = "tuigo"

    /** Order shown in the settings picker. "adaptive" resolves to
     *  laban in light mode, gray in dark mode. */
    private val light = listOf(
        ThemePresets.tuigoLight, ThemePresets.iosPureLight, ThemePresets.cyberNeonLight,
        ThemePresets.matchaLatteLight, ThemePresets.modernJadeLight, ThemePresets.obsidian,
        ThemePresets.pastelMint, ThemePresets.materialLight, ThemePresets.deepSeaLight,
        ThemePresets.sunflower, ThemePresets.snowfall, ThemePresets.cottonCandy,
        ThemePresets.laban, ThemePresets.white, ThemePresets.ios, ThemePresets.iosOld,
        ThemePresets.silver, ThemePresets.wood, ThemePresets.nLight, ThemePresets.ics,
        ThemePresets.kitkat, ThemePresets.dark, ThemePresets.black, ThemePresets.gray,
        ThemePresets.blue, ThemePresets.nDark, ThemePresets.catppuccin, ThemePresets.emerald,
        ThemePresets.amoledPurple, ThemePresets.steelGrey, ThemePresets.highContrast,
        ThemePresets.materialDark, ThemePresets.deepSeaDark,
    )
    private val darkThemes = listOf(
        ThemePresets.tuigoDark, ThemePresets.iosPureDark, ThemePresets.cyberNeonDark,
        ThemePresets.matchaLatteDark, ThemePresets.modernJadeDark, ThemePresets.obsidian,
        ThemePresets.pastelMint, ThemePresets.materialDark, ThemePresets.deepSeaDark,
        ThemePresets.catppuccin, ThemePresets.emerald, ThemePresets.amoledPurple,
        ThemePresets.steelGrey, ThemePresets.highContrast,
        ThemePresets.laban, ThemePresets.white, ThemePresets.ios, ThemePresets.iosOld,
        ThemePresets.silver, ThemePresets.wood, ThemePresets.nLight, ThemePresets.ics,
        ThemePresets.kitkat, ThemePresets.dark, ThemePresets.black, ThemePresets.gray,
        ThemePresets.blue, ThemePresets.nDark, ThemePresets.materialLight,
        ThemePresets.deepSeaLight, ThemePresets.sunflower, ThemePresets.snowfall,
        ThemePresets.cottonCandy,
    )

    val ids = listOf(
        "tuigo", "adaptive", "ios_pure", "cyber_neon", "matcha_latte", "modern_jade",
        "obsidian", "pastel_mint", "material", "deep_sea", "sunflower", "snowfall",
        "cotton_candy", "catppuccin", "emerald", "amoled_purple", "steel_grey",
        "high_contrast", "laban", "white", "black", "gray", "blue", "ios", "ios_old",
        "ics", "kitkat", "silver", "wood", "dark", "n_light", "n_dark",
    )

    fun label(id: String): String = when (id) {
        "adaptive" -> "Theo hệ thống"
        else -> all(false).firstOrNull { it.id == id }?.label ?: id
    }

    fun all(darkMode: Boolean): List<KeyTheme> = if (darkMode) darkThemes else light

    fun get(id: String, darkMode: Boolean): KeyTheme = when (id) {
        "adaptive" -> if (darkMode) ThemePresets.gray else ThemePresets.laban
        else -> all(darkMode).firstOrNull { it.id == id }
            ?: all(darkMode).first { it.id == DEFAULT_ID }
    }

    fun current(ctx: Context): KeyTheme =
        get(Prefs.get(ctx).getString(Prefs.KEY_THEME, DEFAULT_ID) ?: DEFAULT_ID, isDark(ctx))

    private fun isDark(ctx: Context): Boolean =
        ctx.resources.configuration.uiMode and Configuration.UI_MODE_NIGHT_MASK ==
            Configuration.UI_MODE_NIGHT_YES
}
