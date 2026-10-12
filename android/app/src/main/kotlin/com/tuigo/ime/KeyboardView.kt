package com.tuigo.ime

import android.annotation.SuppressLint
import android.content.Context
import android.content.res.Configuration
import android.graphics.Canvas
import android.graphics.Paint
import android.graphics.RectF
import android.graphics.Typeface
import android.graphics.LinearGradient
import android.graphics.Shader
import android.os.Build
import android.os.Handler
import android.os.Looper
import android.os.SystemClock
import android.util.AttributeSet
import android.view.HapticFeedbackConstants
import android.view.MotionEvent
import android.view.SoundEffectConstants
import android.view.View
import android.view.WindowInsets
import com.tuigo.KeyTheme
import com.tuigo.KeyboardThemes
import com.tuigo.Prefs
import kotlin.math.roundToInt

/**
 * Canvas-drawn software keyboard (framework KeyboardView is deprecated).
 * Gboard-style: key popup preview, shift one-shot / caps-lock on double tap,
 * backspace auto-repeat, letters/symbols pages.
 * Colors follow the Darkroom tokens.
 */
class KeyboardView @JvmOverloads constructor(
    context: Context,
    attrs: AttributeSet? = null,
) : View(context, attrs) {

    interface Listener {
        fun onChar(ch: Char)
        fun onDelete()
        fun onEnter()
        fun onShiftChanged(shifted: Boolean, capsLock: Boolean)
        fun onVietToggle() {}
        fun onSwitchIme() {}
        fun onSpaceMenu() {}
        fun onHideKeyboard() {}
    }

    var listener: Listener? = null

    /** Keyboard skin; themes follow the Laban Key catalog (KeyboardThemes.kt). */
    var theme: KeyTheme = KeyboardThemes.current(context)
        set(v) {
            field = v
            applyTheme()
            invalidate()
        }

    /** VN mode state shown on the space bar. */
    var vietnameseOn: Boolean = true
        set(v) { field = v; invalidate() }

    var symbolsPage = false
        private set
    private var symbolLevel = 0

    /** Number row above letters — needed by VNI-style input methods
     *  (digits double as tone marks). */
    var showNumberRow = false
        set(v) {
            if (field == v) return
            field = v
            rows = currentRows()
            relayout()
            requestLayout()
        }

    var shifted = false
        private set
    var capsLock = false
        private set

    // ---- layout model ------------------------------------------------------

    private object K {
        const val MORE = -10
        const val IME = -9
        const val HIDE = -8
        const val SPACER = -7
        const val SPACE = -5
        const val ENTER = -4
        const val MODE = -3
        const val DEL = -2
        const val SHIFT = -1
    }

    private data class KeySpec(val code: Int, val label: String, val weight: Float = 1f)

    private class Key(val spec: KeySpec, val rect: RectF = RectF())

    private fun row(s: String, insetWeight: Float = 0f): Array<KeySpec> {
        val keys = s.map { KeySpec(it.code, it.toString()) }
        return if (insetWeight > 0f) {
            (listOf(KeySpec(K.SPACER, "", insetWeight)) + keys +
                KeySpec(K.SPACER, "", insetWeight)).toTypedArray()
        } else keys.toTypedArray()
    }

    private val lettersRows: Array<Array<KeySpec>> by lazy {
        arrayOf(
            row("qwertyuiop"),
            row("asdfghjkl", insetWeight = 0.5f),
            arrayOf(
                KeySpec(K.SHIFT, "⇧", 1.5f),
                *row("zxcvbnm"),
                KeySpec(K.DEL, "⌫", 1.5f),
            ),
            arrayOf(
                KeySpec(K.MODE, "?123", 1.5f),
                KeySpec(','.code, ",", 0.8f),
                KeySpec(K.SPACE, "Tiếng Việt", 6.2f),
                KeySpec('.'.code, ".", 1f),
                KeySpec(K.ENTER, "⏎", 1.5f),
            ),
        )
    }

    private val symbolsRows: Array<Array<KeySpec>> by lazy {
        arrayOf(
            row("1234567890"),
            row("@#₫_&-+()/"),
            arrayOf(
                KeySpec(K.MORE, "#+=", 1.5f),
                *row("*\"':;!?"),
                KeySpec(K.DEL, "⌫", 1.5f),
            ),
            arrayOf(
                KeySpec(K.MODE, "ABC", 1.5f),
                KeySpec(','.code, ",", 0.8f),
                KeySpec(K.SPACE, "Tiếng Việt", 6.2f),
                KeySpec('.'.code, ".", 1f),
                KeySpec(K.ENTER, "⏎", 1.5f),
            ),
        )
    }

    private val legacySymbolRows: Array<Array<KeySpec>> by lazy {
        arrayOf(
            row("^~`|\\€£¥%$"),
            row("°={}[]<>_₫"),
            arrayOf(KeySpec(K.MORE, "?123", 1.5f), *row("!?'\":;"), KeySpec(K.DEL, "⌫", 1.5f)),
            symbolsRows.last(),
        )
    }

    private val extraSymbolRows: Array<Array<KeySpec>> by lazy {
        arrayOf(
            row("~`|•√π÷×§Δ"),
            row("£€$¢^°={}\\"),
            arrayOf(KeySpec(K.MORE, "¥<>", 1.5f), *row("%©®™✓[]"), KeySpec(K.DEL, "⌫", 1.5f)),
            symbolsRows.last(),
        )
    }

    private val numberRow = row("1234567890")

    private fun currentRows(): Array<Array<KeySpec>> =
        if (symbolsPage) { when (symbolLevel) { 1 -> extraSymbolRows; 2 -> legacySymbolRows; else -> symbolsRows } }
        else if (showNumberRow) arrayOf(numberRow, *lettersRows)
        else lettersRows

    private val keys = mutableListOf<Key>()
    private var rows: Array<Array<KeySpec>> = lettersRows

    // ---- metrics ------------------------------------------------------------

    private val dp = resources.displayMetrics.density
    private var keyHeight = 56 * dp
    private val rowGap = 6 * dp
    private var keyGap = 4 * dp
    private val padH = 3 * dp
    private val padTop = 8 * dp
    private val padBottom = 8 * dp
    private var corner = 10 * dp

    // ---- paint --------------------------------------------------------------

    /** Reserve any inset dispatched by the framework; never put controls in it.
     * Android owns the navigation bar and its IME hide/switch buttons. */
    private var navInset = 0

    private val bgPaint = Paint().apply { color = theme.bg }
    private val keyPaint = Paint(Paint.ANTI_ALIAS_FLAG)
    private val borderPaint = Paint(Paint.ANTI_ALIAS_FLAG).apply {
        style = Paint.Style.STROKE; strokeWidth = 1f; color = theme.border
    }
    private val textPaint = Paint(Paint.ANTI_ALIAS_FLAG).apply {
        color = theme.text
        textAlign = Paint.Align.CENTER
        typeface = Typeface.create("sans-serif-medium", Typeface.NORMAL)
    }
    private val popupTextPaint = Paint(Paint.ANTI_ALIAS_FLAG).apply {
        color = theme.popupText
        textAlign = Paint.Align.CENTER
        typeface = Typeface.create("sans-serif-medium", Typeface.NORMAL)
    }

    private var pressed: Key? = null
    private var shiftTapTime = 0L
    private var downPointerId = -1
    private var spaceLongFired = false

    private val handler = Handler(Looper.getMainLooper())
    private val repeatDelete = object : Runnable {
        override fun run() {
            listener?.onDelete()
            handler.postDelayed(this, 50)
        }
    }

    /** Long-press Space = language/input-method menu (Gboard-style). */
    private val spaceLongPress = Runnable {
        spaceLongFired = true
        pressed = null
        performHapticFeedback(HapticFeedbackConstants.LONG_PRESS)
        listener?.onSpaceMenu()
        invalidate()
    }

    init {
        isHapticFeedbackEnabled = true
        setOnApplyWindowInsetsListener { _, insets ->
            val b = when {
                Build.VERSION.SDK_INT >= 30 ->
                    insets.getInsets(
                        WindowInsets.Type.navigationBars() or
                            WindowInsets.Type.mandatorySystemGestures() or
                            WindowInsets.Type.systemGestures()
                    ).bottom
                Build.VERSION.SDK_INT == 29 ->
                    @Suppress("DEPRECATION")
                    maxOf(insets.mandatorySystemGestureInsets.bottom,
                        insets.systemGestureInsets.bottom)
                else -> @Suppress("DEPRECATION") insets.systemWindowInsetBottom
            }
            if (b != navInset) {
                navInset = b
                requestLayout()
            }
            insets
        }
    }

    override fun onAttachedToWindow() {
        super.onAttachedToWindow()
        requestApplyInsets()
    }

    /** Key metrics in dp — from the settings screen. */
    fun setMetrics(heightDp: Int, gapDp: Int) {
        val h = (if (resources.configuration.orientation == Configuration.ORIENTATION_LANDSCAPE) heightDp.coerceAtMost(40) else heightDp.coerceIn(44, 68)) * dp
        val g = gapDp.coerceIn(0, 8) * dp
        if (h == keyHeight && g == keyGap) return
        keyHeight = h
        keyGap = g
        relayout()
        requestLayout()
    }

    override fun onConfigurationChanged(newConfig: Configuration?) {
        super.onConfigurationChanged(newConfig)
        val t = KeyboardThemes.current(context)
        if (t != theme) {
            theme = t
        } else {
            applyTheme()
        }
        setMetrics(Prefs.keyHeightDp(context), Prefs.keyGapDp(context))
    }

    private fun applyTheme() {
        corner = theme.cornerDp * dp
        bgPaint.color = theme.bg
        bgPaint.shader = LinearGradient(0f, 0f, 0f, height.toFloat().coerceAtLeast(1f), theme.bg, theme.bgEnd, Shader.TileMode.CLAMP)
        borderPaint.color = theme.border
        textPaint.color = theme.text
        popupTextPaint.color = theme.popupText
        invalidate()
    }

    // ---- sizing -------------------------------------------------------------

    override fun onMeasure(widthMeasureSpec: Int, heightMeasureSpec: Int) {
        val w = MeasureSpec.getSize(widthMeasureSpec)
        val n = rows.size
        val h = (padTop + padBottom + navInset +
            n * keyHeight + (n - 1) * rowGap).roundToInt()
        setMeasuredDimension(w, h)
    }

    override fun onSizeChanged(w: Int, h: Int, oldw: Int, oldh: Int) {
        relayout()
    }

    private fun relayout() {
        keys.clear()
        val contentW = width - 2 * padH
        var y = padTop
        for (rowSpec in rows) {
            var x = padH
            val totalWeight = rowSpec.sumOf { it.weight.toDouble() }.toFloat()
            val unitW = (contentW - keyGap * (rowSpec.size - 1)) / totalWeight
            for (spec in rowSpec) {
                val kw = unitW * spec.weight
                keys += Key(spec, RectF(x, y, x + kw, y + keyHeight))
                x += kw + keyGap
            }
            y += keyHeight + rowGap
        }
        bgPaint.shader = LinearGradient(0f, 0f, 0f, height.toFloat().coerceAtLeast(1f), theme.bg, theme.bgEnd, Shader.TileMode.CLAMP)
        invalidate()
    }

    // ---- drawing ------------------------------------------------------------

    override fun onDraw(canvas: Canvas) {
        canvas.drawRect(0f, 0f, width.toFloat(), height.toFloat(), bgPaint)
        for (key in keys) drawKey(canvas, key)
        drawPreview(canvas)
    }

    private fun drawKey(canvas: Canvas, key: Key) {
        val spec = key.spec
        if (spec.code == K.SPACER) return
        val r = key.rect
        val isPressed = pressed === key
        val t = theme

        fun cap(fill: Int, stroke: Int = t.border) {
            keyPaint.color = fill
            canvas.drawRoundRect(r, corner, corner, keyPaint)
            if (stroke ushr 24 != 0) {
                borderPaint.color = stroke
                canvas.drawRoundRect(r, corner, corner, borderPaint)
            }
        }
        fun pressedGlyph(normal: Int) = if (t.pressAccent) t.onAccent else normal

        when (spec.code) {
            K.SHIFT -> {
                val active = capsLock || shifted
                cap(when {
                    isPressed -> if (t.pressAccent) t.accent else t.funcPressed
                    active -> if (t.blueGlyphs) t.space else t.accent
                    else -> t.func
                })
                textPaint.color = when {
                    isPressed -> pressedGlyph(t.funcText)
                    active -> if (t.blueGlyphs) t.accent else t.onAccent
                    else -> t.funcText
                }
                textPaint.textSize = 20 * dp
                drawCentered(canvas, if (capsLock) "⇪" else "⇧", r)
            }
            K.ENTER -> {
                cap(when {
                    isPressed -> if (t.pressAccent) t.accent else t.funcPressed
                    t.enterAccentFill -> t.accent
                    else -> t.func
                })
                textPaint.color = when {
                    isPressed -> pressedGlyph(t.funcText)
                    t.enterAccentFill -> t.onAccent
                    t.blueGlyphs -> t.accent
                    else -> t.funcText
                }
                textPaint.textSize = 19 * dp
                drawCentered(canvas, spec.label, r)
            }
            K.SPACE -> {
                cap(if (isPressed && !t.pressAccent) t.funcPressed else t.space, t.spaceBorder)
                textPaint.color = if (isPressed) pressedGlyph(t.spaceText) else t.spaceText
                textPaint.textSize = 13 * dp
                drawCentered(canvas, if (vietnameseOn) "Tiếng Việt" else "English", r)
            }
            K.DEL, K.MODE, K.MORE -> {
                cap(if (isPressed) (if (t.pressAccent) t.accent else t.funcPressed) else t.func)
                textPaint.color = if (isPressed) pressedGlyph(t.funcText) else t.funcText
                textPaint.textSize = if (spec.code == K.DEL) 19 * dp else 14 * dp
                drawCentered(canvas, spec.label, r)
            }
            else -> {
                cap(if (isPressed) (if (t.pressAccent) t.accent else t.keyPressed) else t.key)
                textPaint.color = if (isPressed) pressedGlyph(t.text) else t.text
                textPaint.textSize = 21 * dp
                drawCentered(canvas, currentLabel(spec), r)
            }
        }
    }

    private fun drawCentered(canvas: Canvas, label: String, r: RectF) {
        if (label.isEmpty()) return
        val y = r.centerY() - (textPaint.descent() + textPaint.ascent()) / 2
        canvas.drawText(label, r.centerX(), y, textPaint)
    }

    private fun drawPreview(canvas: Canvas) {
        val key = pressed ?: return
        if (key.spec.code <= 0) return // only literal-char keys get a popup
        val r = key.rect
        val pw = r.width() * 1.5f
        val ph = keyHeight * 1.15f
        val px = (r.centerX() - pw / 2).coerceIn(padH, width - padH - pw)
        val py = (r.top - ph - rowGap).coerceAtLeast(2 * dp)
        val popup = RectF(px, py, px + pw, py + ph)
        keyPaint.color = theme.popup
        canvas.drawRoundRect(popup, corner, corner, keyPaint)
        if (theme.popupBorder ushr 24 != 0) {
            borderPaint.color = theme.popupBorder
            canvas.drawRoundRect(popup, corner, corner, borderPaint)
        }
        popupTextPaint.textSize = 26 * dp
        val y = popup.centerY() - (popupTextPaint.descent() + popupTextPaint.ascent()) / 2
        canvas.drawText(currentLabel(key.spec), popup.centerX(), y, popupTextPaint)
    }

    // ---- touch --------------------------------------------------------------

    @SuppressLint("ClickableViewAccessibility")
    override fun onTouchEvent(ev: MotionEvent): Boolean {
        when (ev.actionMasked) {
            MotionEvent.ACTION_DOWN -> {
                downPointerId = ev.getPointerId(0)
                onDown(ev.x, ev.y)
            }
            MotionEvent.ACTION_MOVE -> {
                val idx = ev.findPointerIndex(downPointerId)
                if (idx >= 0) onMove(ev.getX(idx), ev.getY(idx))
            }
            MotionEvent.ACTION_UP, MotionEvent.ACTION_POINTER_UP -> {
                val idx = ev.actionIndex
                if (idx >= 0 && ev.getPointerId(idx) == downPointerId) {
                    onUp(ev.getX(idx), ev.getY(idx))
                    downPointerId = -1
                }
            }
            MotionEvent.ACTION_CANCEL -> cancelPress()
        }
        return true
    }

    private fun onDown(x: Float, y: Float) {
        val key = hit(x, y) ?: return
        pressed = key
        performHapticFeedback(HapticFeedbackConstants.KEYBOARD_TAP)
        playSoundEffect(SoundEffectConstants.CLICK)
        if (key.spec.code == K.DEL) {
            listener?.onDelete()
            handler.postDelayed(repeatDelete, 400)
        }
        if (key.spec.code == K.SPACE) {
            spaceLongFired = false
            handler.postDelayed(spaceLongPress, 500)
        }
        invalidate()
    }

    private fun onMove(x: Float, y: Float) {
        val key = hit(x, y)
        if (key !== pressed) {
            pressed = key
            handler.removeCallbacks(repeatDelete)
            handler.removeCallbacks(spaceLongPress)
            if (key?.spec?.code == K.DEL) handler.postDelayed(repeatDelete, 400)
            invalidate()
        }
    }

    private fun onUp(x: Float, y: Float) {
        val p = pressed ?: return
        cancelPress()
        if (hit(x, y) === p && p.spec.code != K.DEL && p.spec.code != K.SPACER) {
            dispatch(p)
        }
    }

    private fun cancelPress() {
        pressed = null
        handler.removeCallbacks(repeatDelete)
        handler.removeCallbacks(spaceLongPress)
        invalidate()
    }

    override fun onDetachedFromWindow() {
        cancelPress()
        super.onDetachedFromWindow()
    }

    private fun hit(x: Float, y: Float): Key? =
        keys.firstOrNull { it.rect.contains(x, y) }

    private fun dispatch(key: Key) {
        when (key.spec.code) {
            K.SHIFT -> {
                val now = SystemClock.uptimeMillis()
                if (capsLock) {
                    capsLock = false; shifted = false
                } else if (shifted && now - shiftTapTime < 350) {
                    capsLock = true
                } else {
                    shifted = !shifted
                }
                shiftTapTime = if (shifted || capsLock) now else 0
                listener?.onShiftChanged(shifted, capsLock)
            }
            K.DEL, K.SPACER -> {} // delete fires on down/repeat
            K.MODE -> {
                val previousCount = rows.size
                symbolsPage = !symbolsPage
                symbolLevel = 0
                rows = currentRows()
                relayout()
                if (previousCount != rows.size) requestLayout()
            }
            K.MORE -> {
                symbolLevel = (symbolLevel + 1) % 3
                rows = currentRows()
                relayout()
            }
            K.ENTER -> listener?.onEnter()
            K.SPACE -> listener?.onChar(' ')
            K.HIDE -> listener?.onHideKeyboard()
            K.IME -> listener?.onSwitchIme()
            else -> {
                listener?.onChar(currentLabel(key.spec).first())
                if (shifted && !capsLock) {
                    shifted = false
                    listener?.onShiftChanged(false, false)
                }
            }
        }
        invalidate()
    }

    private fun currentLabel(spec: KeySpec): String =
        if (shifted && spec.label.length == 1 && spec.label[0].isLetter())
            spec.label.uppercase()
        else spec.label
}
