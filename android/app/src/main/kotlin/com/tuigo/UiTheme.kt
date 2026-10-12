package com.tuigo

import android.content.Context
import android.content.res.Configuration

/** Shared light/dark palette (Darkroom tokens) for the keyboard + settings UI. */
class Palette(
    val bg: Int,
    val surface: Int,
    val raised: Int,
    val border: Int,
    val text: Int,
    val muted: Int,
    val accent: Int,
    val accentHover: Int,
    val onAccent: Int,
)

object UiTheme {
    val Dark = Palette(
        bg = 0xFF0A0A0E.toInt(),
        surface = 0xFF14141C.toInt(),
        raised = 0xFF1D1D28.toInt(),
        border = 0xFF272732.toInt(),
        text = 0xFFF4F4F6.toInt(),
        muted = 0xFF9C9CAB.toInt(),
        accent = 0xFF8B5CF6.toInt(),
        accentHover = 0xFFA78BFA.toInt(),
        onAccent = 0xFF0B0B10.toInt(),
    )

    val Light = Palette(
        bg = 0xFFE9ECF1.toInt(),
        surface = 0xFFDCE1E8.toInt(),
        raised = 0xFFFFFFFF.toInt(),
        border = 0xFFC7CDD6.toInt(),
        text = 0xFF1B1C1F.toInt(),
        muted = 0xFF5F6470.toInt(),
        accent = 0xFF7C3AED.toInt(),
        accentHover = 0xFF6D28D9.toInt(),
        onAccent = 0xFFFFFFFF.toInt(),
    )

    fun isDark(ctx: Context): Boolean =
        ctx.resources.configuration.uiMode and Configuration.UI_MODE_NIGHT_MASK ==
            Configuration.UI_MODE_NIGHT_YES

    fun palette(ctx: Context): Palette = if (isDark(ctx)) Dark else Light
}
