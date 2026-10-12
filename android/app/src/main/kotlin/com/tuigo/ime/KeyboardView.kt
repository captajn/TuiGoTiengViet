package com.tuigo.ime

import android.annotation.SuppressLint
import android.content.Context
import android.content.res.Configuration
import android.graphics.Canvas
import android.graphics.Paint
import android.graphics.RectF
import android.graphics.Typeface
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
import com.tuigo.UiTheme
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
        fun onHideKeyboard() {}
    }

    var listener: Listener? = null

    /** VN mode state shown on the VN key. */
    var vietnameseOn: Boolean = true
        set(v) { field = v; invalidate() }

    var symbolsPage = false
        private set

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
        const val IME = -9
        const val HIDE = -8
        const val SPACER = -7
        const val VIET = -6
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
                KeySpec(','.code, ",", 1f),
                KeySpec(K.SPACE, "Tui Gõ", 6f),
                KeySpec('.'.code, ".", 1f),
                KeySpec(K.ENTER, "⏎", 1.5f),
            ),
        )
    }

    private val symbolsRows: Array<Array<KeySpec>> by lazy {
        arrayOf(
            row("1234567890"),
            row("@#₫_&-+()", insetWeight = 1f),
            arrayOf(
                KeySpec(K.MODE, "ABC", 1.5f),
                *row("!?'\":;"),
                KeySpec(K.DEL, "⌫", 1.5f),
            ),
            arrayOf(
                KeySpec(K.MODE, "ABC", 1.5f),
                KeySpec(','.code, ",", 1f),
                KeySpec(K.SPACE, "Tui Gõ", 6f),
                KeySpec('.'.code, ".", 1f),
                KeySpec(K.ENTER, "⏎", 1.5f),
            ),
        )
    }

    private val numberRow = row("1234567890")

    private fun currentRows(): Array<Array<KeySpec>> =
        if (symbolsPage) symbolsRows
        else if (showNumberRow) arrayOf(numberRow, *lettersRows)
        else lettersRows

    private val keys = mutableListOf<Key>()
    private var rows: Array<Array<KeySpec>> = lettersRows

    // ---- metrics ------------------------------------------------------------

    private val dp = resources.displayMetrics.density
    private val keyHeight = 52 * dp
    private val rowGap = 6 * dp
    private val keyGap = 4 * dp
    private val padH = 3 * dp
    private val padTop = 8 * dp
    private val padBottom = 8 * dp
    private val corner = 8 * dp

    // ---- paint --------------------------------------------------------------

    private var pal = UiTheme.palette(context)

    /** Navigation-bar / gesture bottom inset. The keys sit above it; the strip
     *  below carries the hide + IME-switch buttons (Gboard-style). */
    private var navInset = 0

    private val bgPaint = Paint().apply { color = pal.bg }
    private val keyPaint = Paint(Paint.ANTI_ALIAS_FLAG)
    private val borderPaint = Paint(Paint.ANTI_ALIAS_FLAG).apply {
        style = Paint.Style.STROKE; strokeWidth = 1f; color = pal.border
    }
    private val textPaint = Paint(Paint.ANTI_ALIAS_FLAG).apply {
        color = pal.text
        textAlign = Paint.Align.CENTER
        typeface = Typeface.create("sans-serif-medium", Typeface.NORMAL)
    }
    private val popupTextPaint = Paint(Paint.ANTI_ALIAS_FLAG).apply {
        color = pal.text
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

    /** Long-press Space = IME switcher (Gboard-style). */
    private val spaceLongPress = Runnable {
        spaceLongFired = true
        pressed = null
        performHapticFeedback(HapticFeedbackConstants.LONG_PRESS)
        listener?.onSwitchIme()
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

    override fun onConfigurationChanged(newConfig: Configuration?) {
        super.onConfigurationChanged(newConfig)
        val p = UiTheme.palette(context)
        if (p !== pal) {
            pal = p
            applyPalette()
        }
    }

    private fun applyPalette() {
        bgPaint.color = pal.bg
        borderPaint.color = pal.border
        textPaint.color = pal.text
        popupTextPaint.color = pal.text
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
        if (navInset > 0) {
            val sy = (height - navInset).toFloat()
            val bw = navInset * 1.6f
            keys += Key(KeySpec(K.HIDE, "⌄"),
                RectF(0f, sy, bw, height.toFloat()))
            keys += Key(KeySpec(K.IME, "🌐"),
                RectF(width - bw, sy, width.toFloat(), height.toFloat()))
        }
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

        when (spec.code) {
            K.SHIFT -> {
                keyPaint.color = when {
                    isPressed -> pal.accentHover
                    capsLock || shifted -> pal.accent
                    else -> pal.surface
                }
                canvas.drawRoundRect(r, corner, corner, keyPaint)
                canvas.drawRoundRect(r, corner, corner, borderPaint)
                textPaint.color = if (capsLock || shifted || isPressed) pal.onAccent else pal.muted
                textPaint.textSize = 20 * dp
                drawCentered(canvas, if (capsLock) "⇪" else "⇧", r)
            }
            K.DEL, K.MODE, K.VIET, K.ENTER, K.SPACE, K.HIDE, K.IME -> {
                keyPaint.color = when {
                    isPressed -> pal.accentHover
                    spec.code == K.ENTER -> pal.accent
                    spec.code == K.HIDE || spec.code == K.IME -> pal.bg
                    else -> pal.surface
                }
                canvas.drawRoundRect(r, corner, corner, keyPaint)
                if (spec.code != K.HIDE && spec.code != K.IME)
                    canvas.drawRoundRect(r, corner, corner, borderPaint)
                textPaint.color = when {
                    isPressed -> pal.onAccent
                    spec.code == K.ENTER -> pal.onAccent
                    spec.code == K.VIET -> if (vietnameseOn) pal.accent else pal.muted
                    spec.code == K.SPACE || spec.code == K.HIDE || spec.code == K.IME -> pal.muted
                    else -> pal.text
                }
                textPaint.textSize = when (spec.code) {
                    K.SPACE -> 13 * dp
                    K.MODE, K.VIET -> 14 * dp
                    K.HIDE, K.IME -> 17 * dp
                    else -> 19 * dp
                }
                val label = if (spec.code == K.VIET && !vietnameseOn) "EN" else spec.label
                drawCentered(canvas, label, r)
            }
            else -> {
                keyPaint.color = if (isPressed) pal.accent else pal.raised
                canvas.drawRoundRect(r, corner, corner, keyPaint)
                textPaint.color = if (isPressed) pal.onAccent else pal.text
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
        keyPaint.color = pal.raised
        canvas.drawRoundRect(popup, corner, corner, keyPaint)
        canvas.drawRoundRect(popup, corner, corner, borderPaint)
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
                symbolsPage = !symbolsPage
                rows = currentRows()
                relayout()
                requestLayout()
            }
            K.ENTER -> listener?.onEnter()
            K.SPACE -> listener?.onChar(' ')
            K.VIET -> listener?.onVietToggle()
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
