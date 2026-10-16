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
import android.view.Gravity
import android.view.View
import android.view.ViewGroup
import android.view.WindowInsets
import androidx.core.view.WindowCompat
import android.widget.*

/** Lightweight translucent surfaces; no blur/render pass in the IME. */
class MobileUi(private val context: Context) {
    val pal = UiTheme.palette(context)
    private val swDp get() = context.resources.configuration.screenWidthDp
    fun dp(value: Int) = (value * context.resources.displayMetrics.density).toInt()
    fun fullWidthLp(topMarginDp: Int = 0) = LinearLayout.LayoutParams(
        ViewGroup.LayoutParams.MATCH_PARENT,
        ViewGroup.LayoutParams.WRAP_CONTENT,
    ).apply { if (topMarginDp > 0) topMargin = dp(topMarginDp) }

    fun column(): LinearLayout = object : LinearLayout(context) {
        override fun generateDefaultLayoutParams(): LayoutParams =
            LayoutParams(LayoutParams.MATCH_PARENT, LayoutParams.WRAP_CONTENT)
    }.apply {
        orientation = LinearLayout.VERTICAL
        layoutParams = fullWidthLp()
    }

    fun brandHeader(title: String, subtitle: String) = LinearLayout(context).apply {
        orientation = LinearLayout.HORIZONTAL; gravity = Gravity.CENTER_VERTICAL
        setPadding(0, dp(6), 0, dp(10))
        layoutParams = fullWidthLp()
        val logoSize = if (swDp < 360) dp(48) else dp(58)
        addView(ImageView(context).apply {
            setImageResource(R.drawable.tuigo_logo)
            contentDescription = "Logo Tui Gõ"
            scaleType = ImageView.ScaleType.CENTER_CROP
            background = surface(pal.raised, 16); clipToOutline = true
            layoutParams = LinearLayout.LayoutParams(logoSize, logoSize).apply { rightMargin = dp(14) }
        })
        addView(column().apply {
            addView(text(title, if (swDp < 360) 22f else 26f, bold = true))
            addView(text(subtitle, 14f, pal.muted))
        }, LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f))
    }

    fun badge(value: String, active: Boolean = true) = LinearLayout(context).apply {
        orientation = LinearLayout.HORIZONTAL; gravity = Gravity.CENTER_VERTICAL
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
        layoutParams = fullWidthLp()
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
        val padH = if (swDp < 360) dp(14) else dp(18)
        val padV = if (swDp < 360) dp(14) else dp(16)
        setPadding(padH, padV, padH, padV)
        background = glassCard(false, 20)
        elevation = dp(2).toFloat()
        layoutParams = fullWidthLp(topMarginDp = 16)
        addView(text(title, if (swDp < 360) 17.5f else 19f, bold = true))
        addView(text(hint, 13f, pal.muted))
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
        layoutParams = fullWidthLp(topMarginDp = 8)
        setOnClickListener { action() }
    }

    fun editor(hintText: String, multiline: Boolean = true) = EditText(context).apply {
        hint = hintText; textSize = 15f; setTextColor(pal.text); setHintTextColor(pal.muted)
        inputType = InputType.TYPE_CLASS_TEXT or InputType.TYPE_TEXT_FLAG_NO_SUGGESTIONS or
            (if (multiline) InputType.TYPE_TEXT_FLAG_MULTI_LINE else 0)
        gravity = if (multiline) Gravity.TOP or Gravity.START else Gravity.CENTER_VERTICAL or Gravity.START
        minHeight = dp(if (multiline) 130 else 52)
        setPadding(dp(14), dp(14), dp(14), dp(14))
        background = glassCard(false, 16)
        layoutParams = fullWidthLp(topMarginDp = 6)
    }

    data class TabSpec(
        val id: String,
        val label: String,
        val iconType: Int,
        val content: LinearLayout,
    )

    companion object {
        const val ICON_KEYBOARD = 0
        const val ICON_THEME = 1
        const val ICON_MACRO = 2
    }

    private fun navIcon(type: Int, color: Int): View = object : View(context) {
        private val dm = context.resources.displayMetrics.density
        private val strokePaint = android.graphics.Paint(android.graphics.Paint.ANTI_ALIAS_FLAG).apply {
            style = android.graphics.Paint.Style.STROKE
            strokeWidth = 1.8f * dm
            strokeCap = android.graphics.Paint.Cap.ROUND
            strokeJoin = android.graphics.Paint.Join.ROUND
            this.color = color
        }
        private val fillPaint = android.graphics.Paint(android.graphics.Paint.ANTI_ALIAS_FLAG).apply {
            style = android.graphics.Paint.Style.FILL
            this.color = color
        }
        private val rect = android.graphics.RectF()
        private val path = android.graphics.Path()

        override fun onDraw(canvas: android.graphics.Canvas) {
            val w = width.toFloat()
            val h = height.toFloat()
            when (type) {
                ICON_KEYBOARD -> {
                    rect.set(w * 0.10f, h * 0.22f, w * 0.90f, h * 0.80f)
                    canvas.drawRoundRect(rect, 3.5f * dm, 3.5f * dm, strokePaint)
                    val dotR = 1.2f * dm
                    for (i in 0..3) {
                        val cx = w * (0.26f + i * 0.16f)
                        canvas.drawCircle(cx, h * 0.42f, dotR, fillPaint)
                    }
                    for (i in 0..2) {
                        val cx = w * (0.34f + i * 0.16f)
                        canvas.drawCircle(cx, h * 0.56f, dotR, fillPaint)
                    }
                    canvas.drawLine(w * 0.32f, h * 0.68f, w * 0.68f, h * 0.68f, strokePaint)
                }
                ICON_THEME -> {
                    val gap = w * 0.08f
                    val box = (w * 0.72f - gap) / 2f
                    val startX = w * 0.14f
                    val startY = h * 0.14f
                    val r = 3f * dm
                    rect.set(startX, startY, startX + box, startY + box)
                    canvas.drawRoundRect(rect, r, r, strokePaint)
                    rect.set(startX + box + gap, startY, startX + box * 2 + gap, startY + box)
                    canvas.drawRoundRect(rect, r, r, fillPaint)
                    rect.set(startX, startY + box + gap, startX + box, startY + box * 2 + gap)
                    canvas.drawRoundRect(rect, r, r, strokePaint)
                    rect.set(startX + box + gap, startY + box + gap, startX + box * 2 + gap, startY + box * 2 + gap)
                    canvas.drawRoundRect(rect, r, r, strokePaint)
                }
                else -> {
                    path.reset()
                    path.moveTo(w * 0.56f, h * 0.12f)
                    path.lineTo(w * 0.24f, h * 0.54f)
                    path.lineTo(w * 0.48f, h * 0.54f)
                    path.lineTo(w * 0.42f, h * 0.88f)
                    path.lineTo(w * 0.76f, h * 0.44f)
                    path.lineTo(w * 0.52f, h * 0.44f)
                    path.close()
                    canvas.drawPath(path, strokePaint)
                }
            }
        }
    }.apply {
        layoutParams = LinearLayout.LayoutParams(dp(22), dp(22))
    }

    private fun applyResponsiveContentPadding(content: LinearLayout, bottomExtraDp: Int = 28) {
        val dm = context.resources.displayMetrics
        val baseH = if (swDp < 360) dp(14) else dp(18)
        val maxContentW = dp(640)
        val extraH = if (dm.widthPixels > maxContentW + baseH * 2) (dm.widthPixels - maxContentW) / 2 else baseH
        content.setPadding(extraH, dp(18), extraH, dp(bottomExtraDp))
    }

    fun install(activity: Activity, content: LinearLayout) {
        applyResponsiveContentPadding(content)
        val scroll = ScrollView(context).apply {
            isFillViewport = true; clipToPadding = true
            background = GradientDrawable(GradientDrawable.Orientation.TL_BR, intArrayOf(pal.bg, pal.glow, pal.bg))
            addView(content, FrameLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT))
            setOnApplyWindowInsetsListener { v, insets ->
                applyResponsiveContentPadding(content)
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
        configureWindowAndScrollFocus(activity, scroll)
    }

    @Suppress("ClickableViewAccessibility")
    fun installWithBottomNav(
        activity: Activity,
        content: LinearLayout,
        tabs: List<TabSpec>,
        initialTab: Int = 0,
        onTabSelected: (Int) -> Unit = {},
    ) {
        applyResponsiveContentPadding(content, bottomExtraDp = 104)
        val container = FrameLayout(context).apply {
            background = GradientDrawable(GradientDrawable.Orientation.TL_BR, intArrayOf(pal.bg, pal.glow, pal.bg))
        }
        val scroll = ScrollView(context).apply {
            isFillViewport = true; clipToPadding = true
            addView(content, FrameLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT))
        }
        val navHost = FrameLayout(context).apply {
            val dm = context.resources.displayMetrics
            val baseH = if (swDp < 360) dp(14) else dp(18)
            val maxW = dp(520)
            val hMargin = if (dm.widthPixels > maxW + baseH * 2) (dm.widthPixels - maxW) / 2 else baseH
            setPadding(hMargin, dp(6), hMargin, dp(12))
        }
        val navBar = LinearLayout(context).apply {
            tag = "bottom-nav"
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.CENTER_VERTICAL
            background = glassCard(false, 24)
            elevation = dp(10).toFloat()
            setPadding(dp(6), dp(6), dp(6), dp(6))
        }
        var activeIndex = initialTab.coerceIn(0, (tabs.size - 1).coerceAtLeast(0))
        val tabViews = mutableListOf<LinearLayout>()

        fun selectTab(index: Int, animate: Boolean) {
            activeIndex = index
            onTabSelected(index)
            tabs.forEachIndexed { i, tab ->
                val active = i == index
                tab.content.animate().cancel()
                if (active) {
                    tab.content.visibility = View.VISIBLE
                    if (animate) {
                        tab.content.alpha = 0.35f
                        tab.content.translationY = dp(8).toFloat()
                        tab.content.animate()
                            .alpha(1f)
                            .translationY(0f)
                            .setDuration(180)
                            .setInterpolator(android.view.animation.DecelerateInterpolator(1.6f))
                            .start()
                    } else {
                        tab.content.alpha = 1f
                        tab.content.translationY = 0f
                    }
                } else {
                    tab.content.visibility = View.GONE
                }
            }
            tabViews.forEachIndexed { i, item ->
                val active = i == index
                val spec = tabs[i]
                item.isSelected = active
                item.background = if (active) glassCard(active = true, 18) else null
                item.removeAllViews()
                val col = if (active) pal.accent else pal.muted
                item.addView(navIcon(spec.iconType, col))
                item.addView(TextView(context).apply {
                    text = spec.label
                    textSize = 12f
                    setTextColor(col)
                    typeface = Typeface.create("sans-serif", if (active) Typeface.BOLD else Typeface.NORMAL)
                    gravity = Gravity.CENTER
                    setPadding(0, dp(3), 0, 0)
                })
            }
            if (animate) {
                scroll.post { scroll.smoothScrollTo(0, 0) }
            }
        }

        tabs.forEachIndexed { index, spec ->
            val item = LinearLayout(context).apply {
                tag = "nav-tab-${spec.id}"
                orientation = LinearLayout.VERTICAL
                gravity = Gravity.CENTER
                minimumHeight = dp(54)
                setPadding(dp(8), dp(6), dp(8), dp(6))
                layoutParams = LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f)
                setOnTouchListener { v, ev ->
                    when (ev.actionMasked) {
                        android.view.MotionEvent.ACTION_DOWN -> {
                            v.animate().cancel()
                            v.animate().scaleX(0.95f).scaleY(0.95f).setDuration(80).start()
                        }
                        android.view.MotionEvent.ACTION_UP, android.view.MotionEvent.ACTION_CANCEL -> {
                            v.animate().cancel()
                            v.animate().scaleX(1f).scaleY(1f).setDuration(140)
                                .setInterpolator(android.view.animation.OvershootInterpolator(1.4f))
                                .start()
                        }
                    }
                    false
                }
                setOnClickListener {
                    if (activeIndex != index) selectTab(index, animate = true)
                }
            }
            tabViews.add(item)
            navBar.addView(item)
        }
        selectTab(activeIndex, animate = false)

        navHost.addView(navBar, FrameLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT))
        container.addView(scroll, FrameLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.MATCH_PARENT))
        container.addView(navHost, FrameLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT, Gravity.BOTTOM))

        container.setOnApplyWindowInsetsListener { _, insets ->
            applyResponsiveContentPadding(content, bottomExtraDp = 104)
            val dm = context.resources.displayMetrics
            val baseH = if (swDp < 360) dp(14) else dp(18)
            val maxW = dp(520)
            val hMargin = if (dm.widthPixels > maxW + baseH * 2) (dm.widthPixels - maxW) / 2 else baseH
            if (Build.VERSION.SDK_INT >= 30) {
                val bars = insets.getInsets(WindowInsets.Type.systemBars() or WindowInsets.Type.displayCutout())
                val ime = insets.getInsets(WindowInsets.Type.ime())
                val imeVisible = insets.isVisible(WindowInsets.Type.ime())
                navHost.visibility = if (imeVisible) View.GONE else View.VISIBLE
                scroll.setPadding(bars.left, bars.top, bars.right, maxOf(bars.bottom, ime.bottom))
                navHost.setPadding(bars.left + hMargin, dp(6), bars.right + hMargin, bars.bottom + dp(12))
                if (imeVisible) {
                    scroll.post {
                        scroll.findFocus()?.let { focused ->
                            focused.requestRectangleOnScreen(android.graphics.Rect(0, 0, focused.width, focused.height), false)
                        }
                    }
                }
            } else {
                @Suppress("DEPRECATION")
                val top = insets.systemWindowInsetTop
                @Suppress("DEPRECATION")
                val bottom = insets.systemWindowInsetBottom
                @Suppress("DEPRECATION")
                val left = insets.systemWindowInsetLeft
                @Suppress("DEPRECATION")
                val right = insets.systemWindowInsetRight
                val imeVisible = bottom > dp(120)
                navHost.visibility = if (imeVisible) View.GONE else View.VISIBLE
                scroll.setPadding(left, top, right, bottom)
                navHost.setPadding(left + hMargin, dp(6), right + hMargin, bottom + dp(12))
            }
            insets
        }

        activity.setContentView(container)
        configureWindowAndScrollFocus(activity, scroll)
        container.requestApplyInsets()
    }

    private fun configureWindowAndScrollFocus(activity: Activity, scroll: ScrollView) {
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
            WindowCompat.setDecorFitsSystemWindows(activity.window, false)
            val appearance = android.view.WindowInsetsController.APPEARANCE_LIGHT_STATUS_BARS or android.view.WindowInsetsController.APPEARANCE_LIGHT_NAVIGATION_BARS
            activity.window.insetsController?.setSystemBarsAppearance(if (UiTheme.isDark(context)) 0 else appearance, appearance)
        } else {
            @Suppress("DEPRECATION")
            activity.window.decorView.systemUiVisibility = if (UiTheme.isDark(context)) 0 else View.SYSTEM_UI_FLAG_LIGHT_STATUS_BAR or View.SYSTEM_UI_FLAG_LIGHT_NAVIGATION_BAR
        }
        scroll.requestApplyInsets()
    }
}
