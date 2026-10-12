package com.tuigo

import android.content.Context
import android.content.res.Configuration

/** Shared light/dark palette (Windows jade tokens (app/theme.go)) for the keyboard + settings UI. */
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
    val glow: Int,
)

object UiTheme {
    val Dark = Palette(
        bg = 0xFF0F1115.toInt(),
        surface = 0xE6181A20.toInt(),
        raised = 0xFF222630.toInt(),
        border = 0xFF272B35.toInt(),
        text = 0xFFF8FAFC.toInt(),
        muted = 0xFF94A3B8.toInt(),
        accent = 0xFF3ED3B8.toInt(),
        accentHover = 0xFF65DFC9.toInt(),
        onAccent = 0xFF0F1115.toInt(),
        glow = 0xFF243D3B.toInt(),
    )

    val Light = Palette(
        bg = 0xFFF8FAFC.toInt(),
        surface = 0xE6FFFFFF.toInt(),
        raised = 0xFFF1F5F9.toInt(),
        border = 0xFFE2E8F0.toInt(),
        text = 0xFF0F172A.toInt(),
        muted = 0xFF64748B.toInt(),
        accent = 0xFF197F64.toInt(),
        accentHover = 0xFF146B54.toInt(),
        onAccent = 0xFFFFFFFF.toInt(),
        glow = 0xFFE2F0E7.toInt(),
    )

    fun isDark(ctx: Context): Boolean =
        ctx.resources.configuration.uiMode and Configuration.UI_MODE_NIGHT_MASK ==
            Configuration.UI_MODE_NIGHT_YES

    fun palette(ctx: Context): Palette = if (isDark(ctx)) Dark else Light
}
