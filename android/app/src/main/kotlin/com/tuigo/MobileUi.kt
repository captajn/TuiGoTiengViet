package com.tuigo

import android.app.Activity
import android.content.Context
import android.graphics.Typeface
import android.graphics.drawable.GradientDrawable
import android.graphics.drawable.RippleDrawable
import android.content.res.ColorStateList
import android.os.Build
import android.text.InputType
import android.text.SpannableString
import android.text.Spanned
import android.text.style.ForegroundColorSpan
import android.text.style.RelativeSizeSpan
import android.view.View
import android.view.WindowInsets
import android.widget.*

/** Lightweight translucent surfaces; no blur/render pass in the IME. */
class MobileUi(private val context: Context) {
    val pal = UiTheme.palette(context)
    fun dp(value: Int) = (value * context.resources.displayMetrics.density).toInt()
    fun column() = LinearLayout(context).apply { orientation = LinearLayout.VERTICAL }
    fun brandHeader(title: String, subtitle: String) = LinearLayout(context).apply {
        orientation = LinearLayout.HORIZONTAL; gravity = android.view.Gravity.CENTER_VERTICAL
        setPadding(0, dp(6), 0, dp(10))
        addView(ImageView(context).apply {
            setImageResource(R.drawable.tuigo_logo)
            contentDescription = "Logo Tui Gõ"
            scaleType = ImageView.ScaleType.CENTER_CROP
            background = surface(pal.raised, 16); clipToOutline = true
            layoutParams = LinearLayout.LayoutParams(dp(58), dp(58)).apply { rightMargin = dp(14) }
        })
        addView(column().apply {
            addView(text(title, 26f, bold = true))
            addView(text(subtitle, 14f, pal.muted))
        }, LinearLayout.LayoutParams(0, -2, 1f))
    }
    fun badge(value: String, active: Boolean = true) = LinearLayout(context).apply {
        orientation = LinearLayout.HORIZONTAL; gravity = android.view.Gravity.CENTER_VERTICAL
        val bgCol = if (active) ((0x26 shl 24) or (pal.accent and 0x00FFFFFF)) else ((0x18 shl 24) or (pal.muted and 0x00FFFFFF))
        val textCol = if (active) pal.accent else pal.muted
        background = GradientDrawable().apply {
            setColor(bgCol); cornerRadius = dp(14).toFloat()
            setStroke(dp(1), if (active) pal.accent else pal.border)
        }
        setPadding(dp(10), dp(5), dp(10), dp(5))
        addView(View(context).apply {
            background = GradientDrawable().apply {
                setColor(textCol); cornerRadius = dp(4).toFloat()
            }
            layoutParams = LinearLayout.LayoutParams(dp(7), dp(7)).apply { rightMargin = dp(6) }
        })
        addView(TextView(context).apply {
            text = value; textSize = 12f; setTextColor(textCol); typeface = Typeface.DEFAULT_BOLD
        })
    }
    fun twoLines(title: String, hint: String): CharSequence = SpannableString("$title\n$hint").apply {
        setSpan(ForegroundColorSpan(pal.muted), title.length + 1, length, Spanned.SPAN_EXCLUSIVE_EXCLUSIVE)
        setSpan(RelativeSizeSpan(0.84f), title.length + 1, length, Spanned.SPAN_EXCLUSIVE_EXCLUSIVE)
    }
    fun text(value: String, size: Float = 15f, color: Int = pal.text, bold: Boolean = false) = TextView(context).apply {
        text = value; textSize = size; setTextColor(color)
        typeface = Typeface.create("sans-serif", if (bold) Typeface.BOLD else Typeface.NORMAL)
        setPadding(0, dp(2), 0, dp(6))
    }
    fun surface(color: Int = pal.surface, radius: Int = 20) = GradientDrawable().apply {
        setColor(color); cornerRadius = dp(radius).toFloat(); setStroke(dp(1), pal.border)
    }
    fun glassCard(active: Boolean = false, radius: Int = 20): GradientDrawable {
        val isDark = UiTheme.isDark(context)
        val topCol = if (active) {
            if (isDark) 0x383ED3B8.toInt() else 0xEBEBF7F2.toInt()
        } else {
            if (isDark) 0xF2181C24.toInt() else 0xFAFFFFFF.toInt()
        }
        val botCol = if (active) {
            if (isDark) 0x1E3ED3B8.toInt() else 0xD8D8F0E4.toInt()
        } else {
            if (isDark) 0xF513161D.toInt() else 0xF0F4F7FB.toInt()
        }
        val borderCol = if (active) pal.accent else (if (isDark) 0x24FFFFFF.toInt() else 0x12000000.toInt())
        return GradientDrawable(GradientDrawable.Orientation.TOP_BOTTOM, intArrayOf(topCol, botCol)).apply {
            cornerRadius = dp(radius).toFloat()
            setStroke(dp(if (active) 2 else 1), borderCol)
        }
    }
    fun card(title: String, hint: String) = column().apply {
        setPadding(dp(18), dp(16), dp(18), dp(16))
        background = glassCard(false, 20)
        elevation = dp(2).toFloat()
        layoutParams = LinearLayout.LayoutParams(-1, -2).apply { topMargin = dp(16) }
        addView(text(title, 19f, bold = true)); addView(text(hint, 13f, pal.muted))
    }
    fun button(label: String, primary: Boolean = false, action: () -> Unit) = Button(context).apply {
        text = label; isAllCaps = false; textSize = 14f
        setTextColor(if (primary) pal.onAccent else pal.accent)
        val btnDrawable = if (primary) {
            GradientDrawable(GradientDrawable.Orientation.TOP_BOTTOM, intArrayOf(pal.accent, pal.accentHover)).apply {
                cornerRadius = dp(16).toFloat()
                setStroke(dp(1), 0x50FFFFFF.toInt())
            }
        } else {
            glassCard(false, 16)
        }
        background = RippleDrawable(ColorStateList.valueOf(if (primary) 0x30FFFFFF.toInt() else pal.border), btnDrawable, null)
        elevation = dp(if (primary) 2 else 1).toFloat()
        minHeight = dp(50); setPadding(dp(16), dp(12), dp(16), dp(12))
        layoutParams = LinearLayout.LayoutParams(-1, -2).apply { topMargin = dp(8) }
        setOnClickListener { action() }
    }
    fun editor(hintText: String, multiline: Boolean = true) = EditText(context).apply {
        hint = hintText; textSize = 15f; setTextColor(pal.text); setHintTextColor(pal.muted)
        inputType = InputType.TYPE_CLASS_TEXT or InputType.TYPE_TEXT_FLAG_NO_SUGGESTIONS or
            (if (multiline) InputType.TYPE_TEXT_FLAG_MULTI_LINE else 0)
        gravity = android.view.Gravity.TOP; minHeight = dp(if (multiline) 130 else 52)
        setPadding(dp(14), dp(14), dp(14), dp(14))
        background = glassCard(false, 16)
    }
    fun install(activity: Activity, content: LinearLayout) {
        content.setPadding(dp(20), dp(20), dp(20), dp(28))
        val scroll = ScrollView(context).apply {
            isFillViewport = true; clipToPadding = true
            background = GradientDrawable(GradientDrawable.Orientation.TL_BR, intArrayOf(pal.bg, pal.glow, pal.bg))
            addView(content)
            setOnApplyWindowInsetsListener { v, insets ->
                if (Build.VERSION.SDK_INT >= 30) {
                    val bars = insets.getInsets(WindowInsets.Type.systemBars() or WindowInsets.Type.displayCutout() or WindowInsets.Type.ime())
                    v.setPadding(bars.left, bars.top, bars.right, bars.bottom)
                } else {
                    @Suppress("DEPRECATION")
                    v.setPadding(insets.systemWindowInsetLeft, insets.systemWindowInsetTop, insets.systemWindowInsetRight, insets.systemWindowInsetBottom)
                }
                if (Build.VERSION.SDK_INT >= 30 && insets.isVisible(WindowInsets.Type.ime())) {
                    v.post {
                        v.findFocus()?.let { focused ->
                            focused.requestRectangleOnScreen(android.graphics.Rect(0, 0, focused.width, focused.height), false)
                        }
                    }
                }
                insets
            }
        }
        activity.setContentView(scroll)
        if (Build.VERSION.SDK_INT >= 29) activity.window.decorView.isForceDarkAllowed = false
        // Insets can arrive before the focused child has its final bounds.
        // Check after layout so the editor remains above the IME, including
        // a keyboard resize (number row, rotation, or font size changes).
        scroll.viewTreeObserver.addOnGlobalLayoutListener {
            val focused = scroll.findFocus()
            if (focused is EditText) {
                val position = IntArray(2); focused.getLocationOnScreen(position)
                val viewport = IntArray(2); scroll.getLocationOnScreen(viewport)
                val visibleBottom = viewport[1] + scroll.height - scroll.paddingBottom
                val overflow = position[1] + focused.height - visibleBottom
                if (overflow > 0) scroll.scrollBy(0, overflow + dp(16))
            }
        }
        if (Build.VERSION.SDK_INT >= 30) {
            activity.window.setDecorFitsSystemWindows(false)
            val appearance = android.view.WindowInsetsController.APPEARANCE_LIGHT_STATUS_BARS or android.view.WindowInsetsController.APPEARANCE_LIGHT_NAVIGATION_BARS
            activity.window.insetsController?.setSystemBarsAppearance(if (UiTheme.isDark(context)) 0 else appearance, appearance)
        } else {
            @Suppress("DEPRECATION")
            activity.window.decorView.systemUiVisibility = if (UiTheme.isDark(context)) 0 else View.SYSTEM_UI_FLAG_LIGHT_STATUS_BAR or View.SYSTEM_UI_FLAG_LIGHT_NAVIGATION_BAR
        }
        scroll.requestApplyInsets()
    }
}
