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
import kotlin.math.abs
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
        /** Extend the selection one char leftward (hold-Backspace select mode). */
        fun onSelectLeft() {}
        fun onHideKeyboard() {}
        fun onCursorMove(delta: Int) {}
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
        set(v) {
            if (field == v) return
            field = v
            rows = currentRows()
            relayout()
            invalidate()
        }

    /** True when the active input method is VNI — shows VNI tone/mark hints on the number row. */
    var vniMode: Boolean = false
        set(v) {
            if (field == v) return
            field = v
            rows = currentRows()
            relayout()
            invalidate()
        }

    var symbolsPage = false
        private set
    private var symbolLevel = 0

    /** Back to the letter page — called when the keyboard opens so a new
     *  field never starts on the symbols page. */
    fun resetPage() {
        symbolsPage = false
        symbolLevel = 0
        rows = currentRows()
        relayout()
    }

    /** Split the letter rows into left/right halves in landscape. */
    var landscapeSplit = true
        set(v) {
            if (field == v) return
            field = v
            relayout()
            invalidate()
        }

    /** Hold Backspace → highlight text leftward until release (Gboard-style). */
    var holdDeleteSelect = false

    /** Vibrate on key press. */
    var hapticEnabled = true
        set(v) {
            field = v
            isHapticFeedbackEnabled = v
        }

    /** Show the key popup preview on press. */
    var popupEnabled = true


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

    /** Gboard-style 4x4 keypad for number/phone/date-time fields (PIN, OTP…). */
    enum class NumericLayout { NONE, NUMBER, PHONE, DATETIME }

    var numericLayout = NumericLayout.NONE
        set(v) {
            if (field == v) return
            field = v
            symbolsPage = false
            symbolLevel = 0
            rows = currentRows()
            relayout()
            requestLayout()
        }

    var shifted = false
        private set
    var capsLock = false
        private set
    private var manualShift = false

    fun resetManualShift() {
        manualShift = false
    }

    fun setAutoShifted(enabled: Boolean) {
        if (capsLock || manualShift || shifted == enabled) return
        shifted = enabled
        listener?.onShiftChanged(shifted, false)
        invalidate()
    }

    // ---- layout model ------------------------------------------------------

    private object K {
        const val NUM_SPACE = -11
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

    private data class KeySpec(
        val code: Int,
        val label: String,
        val weight: Float = 1f,
        val hint: String = "",
        val longPressChar: Char? = hint.firstOrNull(),
    )

    private class Key(val spec: KeySpec, val rect: RectF = RectF())

    /** Top-row digit hints when the dedicated number row is hidden. */
    private val topRowNumberHints = mapOf(
        'q' to '1', 'w' to '2', 'e' to '3', 'r' to '4', 't' to '5',
        'y' to '6', 'u' to '7', 'i' to '8', 'o' to '9', 'p' to '0',
    )

    /** Standard Gboard / Laban Key secondary symbols on rows 2 and 3. */
    private val letterHints = mapOf(
        'a' to '@', 's' to '#', 'd' to '$', 'f' to '%', 'g' to '&',
        'h' to '-', 'j' to '+', 'k' to '(', 'l' to ')',
        'z' to '*', 'x' to '"', 'c' to '\'', 'v' to ':',
        'b' to ';', 'n' to '!', 'm' to '?',
    )

    /** Standard Laban Key VNI diacritic/tone hints on the number row (1..9). */
    private val vniNumberHints = mapOf(
        '1' to "´", '2' to "`", '3' to "?", '4' to "~", '5' to ".",
        '6' to "^", '7' to "ʼ", '8' to "˘", '9' to "Đ",
    )

    private fun row(s: String, insetWeight: Float = 0f, withLetterHints: Boolean = false): Array<KeySpec> {
        val keys = s.map { ch ->
            val hintChar = if (!withLetterHints) null
            else letterHints[ch] ?: if (!showNumberRow) topRowNumberHints[ch] else null
            KeySpec(
                code = ch.code,
                label = ch.toString(),
                hint = hintChar?.toString() ?: "",
                longPressChar = hintChar,
            )
        }
        return if (insetWeight > 0f) {
            (listOf(KeySpec(K.SPACER, "", insetWeight)) + keys +
                KeySpec(K.SPACER, "", insetWeight)).toTypedArray()
        } else keys.toTypedArray()
    }

    /** Bottom row shared by every page. */
    private fun bottomRow(modeLabel: String): Array<KeySpec> = arrayOf(
        KeySpec(K.MODE, modeLabel, 1.7f),
        KeySpec(','.code, ",", 1.1f),
        KeySpec(K.SPACE, "Tiếng Việt", 5.7f),
        KeySpec('.'.code, ".", 1.1f),
        KeySpec(K.ENTER, "⏎", 1.5f),
    )

    private fun buildLettersRows(): Array<Array<KeySpec>> = arrayOf(
        row("qwertyuiop", withLetterHints = true),
        row("asdfghjkl", insetWeight = 0.5f, withLetterHints = true),
        arrayOf(
            KeySpec(K.SHIFT, "⇧", 1.5f),
            *row("zxcvbnm", withLetterHints = true),
            KeySpec(K.DEL, "⌫", 1.5f),
        ),
        bottomRow("?123"),
    )

    private val symbolsRows: Array<Array<KeySpec>> by lazy {
        arrayOf(
            row("1234567890"),
            row("@#₫_&-+()/"),
            arrayOf(
                KeySpec(K.MORE, "#+=", 1.5f),
                *row("*\"':;!?"),
                KeySpec(K.DEL, "⌫", 1.5f),
            ),
        )
    }

    private val legacySymbolRows: Array<Array<KeySpec>> by lazy {
        arrayOf(
            row("^~`|\\€£¥%$"),
            row("°={}[]<>_₫"),
            arrayOf(KeySpec(K.MORE, "?123", 1.5f), *row("!?'\":;"), KeySpec(K.DEL, "⌫", 1.5f)),
        )
    }

    private val extraSymbolRows: Array<Array<KeySpec>> by lazy {
        arrayOf(
            row("~`|•√π÷×§Δ"),
            row("£€$¢^°={}\\"),
            arrayOf(KeySpec(K.MORE, "¥<>", 1.5f), *row("%©®™✓[]"), KeySpec(K.DEL, "⌫", 1.5f)),
        )
    }

    private fun buildNumberRow(): Array<KeySpec> =
        "1234567890".map { ch ->
            val vniHint = if (vniMode && vietnameseOn) vniNumberHints[ch] ?: "" else ""
            KeySpec(code = ch.code, label = ch.toString(), hint = vniHint, longPressChar = null)
        }.toTypedArray()

    private fun numPadRows(layout: NumericLayout): Array<Array<KeySpec>> {
        val phone = layout == NumericLayout.PHONE
        val bottom = when (layout) {
            NumericLayout.PHONE -> "*0#"
            NumericLayout.DATETIME -> "/0:"
            else -> ",0."
        }
        return arrayOf(
            arrayOf(*row("123"), if (phone) KeySpec('+'.code, "+") else KeySpec('-'.code, "-")),
            arrayOf(*row("456"), if (phone) KeySpec('-'.code, "-") else KeySpec(K.NUM_SPACE, "␣")),
            arrayOf(*row("789"), KeySpec(K.DEL, "⌫")),
            arrayOf(*row(bottom), KeySpec(K.ENTER, "✓")),
        )
    }

    private fun currentRows(): Array<Array<KeySpec>> =
        if (numericLayout != NumericLayout.NONE) numPadRows(numericLayout)
        else if (symbolsPage) arrayOf(*when (symbolLevel) { 1 -> extraSymbolRows; 2 -> legacySymbolRows; else -> symbolsRows }, bottomRow("ABC"))
        else if (showNumberRow) arrayOf(buildNumberRow(), *buildLettersRows())
        else buildLettersRows()

    private val keys = mutableListOf<Key>()
    private var rows: Array<Array<KeySpec>> = buildLettersRows()

    /** Split each letter row into left/right halves so landscape thumbs reach
     *  every key — same idea as the iPad/Laban Key landscape layout. The
     *  bottom row (mostly functional keys) stays one continuous row. */
    private fun computeVisualRows(): List<List<Array<KeySpec>>> {
        val landscape = resources.configuration.orientation == Configuration.ORIENTATION_LANDSCAPE
        if (!landscape || !landscapeSplit || numericLayout != NumericLayout.NONE) {
            return rows.map { listOf(it) }
        }
        val out = ArrayList<List<Array<KeySpec>>>(rows.size)
        for (rowSpec in rows) {
            val spaceIdx = rowSpec.indexOfFirst { it.code == K.SPACE }
            if (spaceIdx >= 0) {
                // Bottom row: give each thumb its own Space half, like the
                // Laban Key landscape layout.
                val space = rowSpec[spaceIdx]
                val halfSpace = space.copy(weight = space.weight * 0.48f)
                out += listOf(
                    arrayOf(*rowSpec.sliceArray(0 until spaceIdx), halfSpace),
                    arrayOf(halfSpace, *rowSpec.sliceArray(spaceIdx + 1 until rowSpec.size)),
                )
                continue
            }
            var letters = 0
            for (spec in rowSpec) if (spec.code >= 0) letters++
            if (letters == 0) {
                out += listOf(rowSpec)
                continue
            }
            var seen = 0
            var splitAt = -1
            for (i in rowSpec.indices) {
                if (rowSpec[i].code >= 0) {
                    seen++
                    if (seen == (letters + 1) / 2) { splitAt = i + 1; break }
                }
            }
            if (splitAt <= 0 || splitAt >= rowSpec.size) {
                out += listOf(rowSpec)
                continue
            }
            out += listOf(
                rowSpec.sliceArray(0 until splitAt),
                rowSpec.sliceArray(splitAt until rowSpec.size),
            )
        }
        return out
    }

    // ---- metrics ------------------------------------------------------------

    private val dp = resources.displayMetrics.density
    private var keyHeight = 56 * dp
    private val rowGap = 6 * dp
    private var keyGap = 4 * dp
    /** Horizontal gap between the left/right halves in landscape split mode. */
    private var splitGap = 40 * dp
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
    private val sheenPaint = Paint(Paint.ANTI_ALIAS_FLAG).apply {
        style = Paint.Style.STROKE; strokeWidth = 1f; color = 0x40FFFFFF.toInt()
    }
    private val iconPaint = Paint(Paint.ANTI_ALIAS_FLAG).apply {
        style = Paint.Style.STROKE
        strokeCap = Paint.Cap.ROUND
        strokeJoin = Paint.Join.ROUND
    }
    private val shadowPaint = Paint(Paint.ANTI_ALIAS_FLAG).apply {
        style = Paint.Style.FILL; color = 0x1A000000.toInt()
    }
    private val dotPaint = Paint(Paint.ANTI_ALIAS_FLAG)
    private val shadowRect = RectF()
    private val popupRect = RectF()

    private var pressed: Key? = null
    /** Last touched key kept briefly after release so the popup lingers. */
    private var popupKey: Key? = null
    private var popupHoldUntil = 0L
    private val popupHoldMs = 120L
    private var shiftTapTime = 0L
    private var downPointerId = -1
    private var downX = 0f
    private var downY = 0f
    private var spaceLongFired = false
    private var spaceSliding = false
    private var spaceDownX = 0f
    private var spaceDownY = 0f
    private var spaceLastStepX = 0f
    private val spaceSwipeSlop = 10 * dp
    private val spaceStepPx = 10 * dp
    private val keyHysteresis = 8 * dp
    private val tapLockSlop = 10 * dp

    /** True after a long-press already produced output — further moves in
     *  the same gesture must not re-arm the touched key (finger jitter used
     *  to type "4r" instead of just the hint char). */
    private var pressConsumed = false

    private val handler = Handler(Looper.getMainLooper())
    private val repeatDelete = object : Runnable {
        override fun run() {
            listener?.onDelete()
            handler.postDelayed(this, 50)
        }
    }

    /** After the long-press delay in hold-select mode: switch from deleting to
     *  extending the selection leftward until the finger lifts. */
    private val repeatSelect = object : Runnable {
        override fun run() {
            listener?.onSelectLeft()
            handler.postDelayed(this, 50)
        }
    }

    private val delLongPress = Runnable {
        if (!holdDeleteSelect || pressed?.spec?.code != K.DEL) return@Runnable
        pressConsumed = true
        handler.removeCallbacks(repeatDelete)
        performHapticFeedback(HapticFeedbackConstants.LONG_PRESS)
        handler.post(repeatSelect)
        invalidate()
    }

    /** Long-press a hinted key types the hint char directly (Unikey-style). */
    private val hintLongPress = Runnable {
        val k = pressed ?: return@Runnable
        val alt = k.spec.longPressChar ?: return@Runnable
        pressed = null
        pressConsumed = true
        manualShift = false
        performHapticFeedback(HapticFeedbackConstants.LONG_PRESS)
        listener?.onChar(alt)
        invalidate()
    }

    /** Long-press Space = language/input-method menu (Gboard-style). */
    private val spaceLongPress = Runnable {
        if (spaceSliding) return@Runnable
        spaceLongFired = true
        pressed = null
        pressConsumed = true
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

    /** Key metrics in dp — from the settings screen. `landHeightDp` lets the
     *  user pick a separate key height for landscape; when it is unset the
     *  landscape default of 36dp applies. `landSplitGapDp` is the space
     *  between the two halves in landscape split mode. */
    fun setMetrics(heightDp: Int, gapDp: Int, landHeightDp: Int = -1, landSplitGapDp: Int = 40) {
        val h = (if (resources.configuration.orientation == Configuration.ORIENTATION_LANDSCAPE)
            (if (landHeightDp > 0) landHeightDp else 36).coerceIn(32, 56)
            else heightDp.coerceIn(44, 68)) * dp
        val g = gapDp.coerceIn(0, 8) * dp
        val s = landSplitGapDp.coerceIn(8, 200) * dp
        if (h == keyHeight && g == keyGap && s == splitGap) return
        keyHeight = h
        keyGap = g
        splitGap = s
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
        setMetrics(Prefs.keyHeightDp(context), Prefs.keyGapDp(context), Prefs.landKeyHeightDp(context), Prefs.landSplitGapDp(context))
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
        val h = (padTop + maxOf(padBottom, navInset.toFloat()) +
            n * keyHeight + (n - 1) * rowGap).roundToInt()
        setMeasuredDimension(w, h)
    }

    override fun onSizeChanged(w: Int, h: Int, oldw: Int, oldh: Int) {
        relayout()
    }


    private fun relayout() {
        keys.clear()
        val visualRows = computeVisualRows()
        val contentW = width - 2 * padH
        var y = padTop
        for (halves in visualRows) {
            val halfW = if (halves.size > 1) (contentW - splitGap) / 2f else contentW
            for (half in halves.indices) {
                val rowSpec = halves[half]
                var x = padH + half * (halfW + splitGap)
                val totalWeight = rowSpec.sumOf { it.weight.toDouble() }.toFloat()
                val unitW = (halfW - keyGap * (rowSpec.size - 1)) / totalWeight
                for (spec in rowSpec) {
                    val kw = unitW * spec.weight
                    keys += Key(spec, RectF(x, y, x + kw, y + keyHeight))
                    x += kw + keyGap
                }
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

        fun cap(fill: Int, stroke: Int = t.border, keyCorner: Float = corner) {
            shadowRect.set(r.left, r.top + 1.5f * dp, r.right, r.bottom + 2.0f * dp)
            shadowPaint.color = if (t.bg ushr 24 != 0 && (t.bg and 0xFF) > 128) 0x1F000000.toInt() else 0x3D000000.toInt()
            canvas.drawRoundRect(shadowRect, keyCorner, keyCorner, shadowPaint)

            keyPaint.color = fill
            canvas.drawRoundRect(r, keyCorner, keyCorner, keyPaint)
            if (stroke ushr 24 != 0) {
                borderPaint.color = stroke
                canvas.drawRoundRect(r, keyCorner, keyCorner, borderPaint)
            }
            val sheenAlpha = if ((t.key and 0xFF) > 128) 0x55FFFFFF.toInt() else 0x1AFFFFFF.toInt()
            sheenPaint.color = sheenAlpha
            canvas.drawLine(r.left + keyCorner * 0.8f, r.top + 1f, r.right - keyCorner * 0.8f, r.top + 1f, sheenPaint)
            if (isPressed) {
                val bloomColor = (0x35 shl 24) or (t.accent and 0x00FFFFFF)
                borderPaint.color = bloomColor
                borderPaint.strokeWidth = 2f * dp
                canvas.drawRoundRect(r, keyCorner, keyCorner, borderPaint)
                borderPaint.strokeWidth = 1f
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
                iconPaint.color = when {
                    isPressed -> pressedGlyph(t.funcText)
                    active -> if (t.blueGlyphs) t.accent else t.onAccent
                    else -> t.funcText
                }
                drawShiftIcon(canvas, r, active, capsLock)
            }
            K.ENTER -> {
                val enterCorner = if (t.enterAccentFill) r.height() / 2f else corner
                cap(when {
                    isPressed -> if (t.pressAccent) t.accent else t.funcPressed
                    t.enterAccentFill -> t.accent
                    else -> t.func
                }, t.border, enterCorner)
                iconPaint.color = when {
                    isPressed -> pressedGlyph(t.funcText)
                    t.enterAccentFill -> t.onAccent
                    t.blueGlyphs -> t.accent
                    else -> t.funcText
                }
                if (numericLayout != NumericLayout.NONE) drawCheckIcon(canvas, r)
                else drawEnterIcon(canvas, r)
            }
            K.SPACE -> {
                val spaceCorner = r.height() / 2f
                cap(if (isPressed && !t.pressAccent) t.funcPressed else t.space, t.spaceBorder, spaceCorner)
                textPaint.color = if (isPressed) pressedGlyph(t.spaceText) else t.spaceText
                textPaint.textSize = 13 * dp
                val baseLabel = if (vietnameseOn) "Tiếng Việt" else "English"
                val spaceLabel = if (spaceSliding) "◀   $baseLabel   ▶" else baseLabel
                drawCentered(canvas, spaceLabel, r)
                if (vietnameseOn && !spaceSliding) {
                    dotPaint.color = t.accent
                    val labelWidth = textPaint.measureText(spaceLabel)
                    val dotX = r.centerX() - labelWidth / 2f - 7 * dp
                    val dotY = r.centerY()
                    canvas.drawCircle(dotX, dotY, 2.5f * dp, dotPaint)
                }
            }
            K.DEL -> {
                cap(if (isPressed) (if (t.pressAccent) t.accent else t.funcPressed) else t.func)
                iconPaint.color = if (isPressed) pressedGlyph(t.funcText) else t.funcText
                drawDeleteIcon(canvas, r)
            }
            K.MODE, K.MORE, K.NUM_SPACE -> {
                cap(if (isPressed) (if (t.pressAccent) t.accent else t.funcPressed) else t.func)
                textPaint.color = if (isPressed) pressedGlyph(t.funcText) else t.funcText
                textPaint.textSize = if (spec.code == K.NUM_SPACE) 19 * dp else 14 * dp
                drawCentered(canvas, spec.label, r)
            }
            else -> {
                cap(if (isPressed) (if (t.pressAccent) t.accent else t.keyPressed) else t.key)
                textPaint.color = if (isPressed) pressedGlyph(t.text) else t.text
                textPaint.textSize = 21 * dp
                drawCentered(canvas, currentLabel(spec), r)
                if (spec.hint.isNotEmpty()) {
                    textPaint.textSize = 10.5f * dp
                    textPaint.color = if (isPressed) pressedGlyph(t.text) else t.funcText
                    textPaint.alpha = 0xAA
                    val hx = r.right - 7 * dp - textPaint.measureText(spec.hint) / 2
                    canvas.drawText(spec.hint, hx, r.top + 13 * dp, textPaint)
                    textPaint.alpha = 0xFF
                }
            }
        }
    }

    private fun drawCentered(canvas: Canvas, label: String, r: RectF) {
        if (label.isEmpty()) return
        val y = r.centerY() - (textPaint.descent() + textPaint.ascent()) / 2
        canvas.drawText(label, r.centerX(), y, textPaint)
    }

    private val iconPath = android.graphics.Path()

    /** Gboard-style arrow: outline when off, filled when shifted, filled +
     *  underline when caps-lock. */
    private fun drawShiftIcon(canvas: Canvas, r: RectF, active: Boolean, caps: Boolean) {
        val cx = r.centerX(); val cy = r.centerY(); val s = 6.5f * dp
        iconPath.reset()
        iconPath.moveTo(cx, cy - s)
        iconPath.lineTo(cx + s, cy - 0.05f * s)
        iconPath.lineTo(cx + 0.42f * s, cy - 0.05f * s)
        iconPath.lineTo(cx + 0.42f * s, cy + 0.95f * s)
        iconPath.lineTo(cx - 0.42f * s, cy + 0.95f * s)
        iconPath.lineTo(cx - 0.42f * s, cy - 0.05f * s)
        iconPath.lineTo(cx - s, cy - 0.05f * s)
        iconPath.close()
        iconPaint.style = if (active) Paint.Style.FILL else Paint.Style.STROKE
        iconPaint.strokeWidth = 1.7f * dp
        canvas.drawPath(iconPath, iconPaint)
        if (caps) {
            iconPaint.style = Paint.Style.STROKE
            iconPaint.strokeWidth = 1.6f * dp
            canvas.drawLine(cx - 0.42f * s, cy + 1.35f * s, cx + 0.42f * s, cy + 1.35f * s, iconPaint)
        }
        iconPaint.style = Paint.Style.STROKE
    }

    /** Return-arrow icon (letter page). */
    private fun drawEnterIcon(canvas: Canvas, r: RectF) {
        val cx = r.centerX(); val cy = r.centerY(); val s = 6.5f * dp
        iconPaint.style = Paint.Style.STROKE
        iconPaint.strokeWidth = 1.8f * dp
        iconPath.reset()
        iconPath.moveTo(cx + 0.65f * s, cy - 1.0f * s)
        iconPath.lineTo(cx + 0.65f * s, cy + 0.3f * s)
        iconPath.lineTo(cx - 0.8f * s, cy + 0.3f * s)
        canvas.drawPath(iconPath, iconPaint)
        iconPath.reset()
        iconPath.moveTo(cx - 0.25f * s, cy - 0.35f * s)
        iconPath.lineTo(cx - 0.8f * s, cy + 0.3f * s)
        iconPath.lineTo(cx - 0.25f * s, cy + 0.95f * s)
        canvas.drawPath(iconPath, iconPaint)
    }

    /** Check icon (numeric keypad "done" key). */
    private fun drawCheckIcon(canvas: Canvas, r: RectF) {
        val cx = r.centerX(); val cy = r.centerY(); val s = 6f * dp
        iconPaint.style = Paint.Style.STROKE
        iconPaint.strokeWidth = 2f * dp
        iconPath.reset()
        iconPath.moveTo(cx - 0.9f * s, cy + 0.1f * s)
        iconPath.lineTo(cx - 0.25f * s, cy + 0.75f * s)
        iconPath.lineTo(cx + s, cy - 0.6f * s)
        canvas.drawPath(iconPath, iconPaint)
    }

    /** Backspace icon: pentagon with an x inside. */
    private fun drawDeleteIcon(canvas: Canvas, r: RectF) {
        val cx = r.centerX(); val cy = r.centerY(); val s = 7.5f * dp
        iconPaint.style = Paint.Style.STROKE
        iconPaint.strokeWidth = 1.6f * dp
        iconPath.reset()
        iconPath.moveTo(cx - s, cy)
        iconPath.lineTo(cx - 0.15f * s, cy - 0.72f * s)
        iconPath.lineTo(cx + 0.85f * s, cy - 0.72f * s)
        iconPath.quadTo(cx + s, cy - 0.72f * s, cx + s, cy - 0.5f * s)
        iconPath.lineTo(cx + s, cy + 0.5f * s)
        iconPath.quadTo(cx + s, cy + 0.72f * s, cx + 0.85f * s, cy + 0.72f * s)
        iconPath.lineTo(cx - 0.15f * s, cy + 0.72f * s)
        iconPath.close()
        canvas.drawPath(iconPath, iconPaint)
        val x = 0.22f * s
        canvas.drawLine(cx - x, cy - x, cx + 0.5f * s, cy + x, iconPaint)
        canvas.drawLine(cx + 0.5f * s, cy - x, cx - x, cy + x, iconPaint)
    }

    private fun drawPreview(canvas: Canvas) {
        if (!popupEnabled) return
        val key = pressed ?: popupKey.takeIf {
            SystemClock.uptimeMillis() < popupHoldUntil
        } ?: return
        if (key.spec.code <= 0) return // only literal-char keys get a popup
        val r = key.rect
        val pw = r.width() * 1.5f
        val ph = keyHeight * 1.15f
        val px = (r.centerX() - pw / 2).coerceIn(padH, width - padH - pw)
        val py = (r.top - ph - rowGap).coerceAtLeast(2 * dp)
        popupRect.set(px, py, px + pw, py + ph)
        keyPaint.color = theme.popup
        canvas.drawRoundRect(popupRect, corner, corner, keyPaint)
        if (theme.popupBorder ushr 24 != 0) {
            borderPaint.color = theme.popupBorder
            canvas.drawRoundRect(popupRect, corner, corner, borderPaint)
        }
        popupTextPaint.textSize = 26 * dp
        val y = popupRect.centerY() - (popupTextPaint.descent() + popupTextPaint.ascent()) / 2
        canvas.drawText(currentLabel(key.spec), popupRect.centerX(), y, popupTextPaint)
    }

    // ---- touch --------------------------------------------------------------

    @SuppressLint("ClickableViewAccessibility")
    override fun onTouchEvent(ev: MotionEvent): Boolean {
        when (ev.actionMasked) {
            MotionEvent.ACTION_DOWN -> {
                downPointerId = ev.getPointerId(0)
                onDown(ev.x, ev.y)
            }
            MotionEvent.ACTION_POINTER_DOWN -> {
                val idx = ev.actionIndex
                // Multi-touch rollover: when a second thumb lands before the first
                // thumb lifts, commit the currently pressed key in order immediately.
                val prev = pressed
                if (!pressConsumed && !spaceSliding && prev != null &&
                    prev.spec.code != K.DEL && prev.spec.code != K.SPACER
                ) {
                    cancelPress()
                    dispatch(prev)
                } else {
                    cancelPress()
                }
                downPointerId = ev.getPointerId(idx)
                onDown(ev.getX(idx), ev.getY(idx))
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
        pressConsumed = false
        pressed = key
        downX = x
        downY = y
        performHapticFeedback(HapticFeedbackConstants.KEYBOARD_TAP)
        playSoundEffect(SoundEffectConstants.CLICK)
        if (key.spec.code == K.DEL) {
            manualShift = false
            if (holdDeleteSelect) {
                // Select-on-hold mode: nothing deletes on down — a quick tap
                // deletes once on release, a hold only highlights.
                handler.postDelayed(delLongPress, 400)
            } else {
                listener?.onDelete()
                handler.postDelayed(repeatDelete, 400)
            }
        }
        if (key.spec.code == K.SPACE) {
            spaceLongFired = false
            spaceSliding = false
            spaceDownX = x
            spaceDownY = y
            spaceLastStepX = x
            handler.postDelayed(spaceLongPress, 550)
        } else if (key.spec.longPressChar != null) {
            handler.postDelayed(hintLongPress, 450)
        }
        invalidate()
    }

    private fun withinHysteresis(key: Key, x: Float, y: Float): Boolean {
        if (abs(x - downX) <= tapLockSlop && abs(y - downY) <= tapLockSlop) return true
        val r = key.rect
        return x >= r.left - keyHysteresis && x <= r.right + keyHysteresis &&
            y >= r.top - keyHysteresis && y <= r.bottom + keyHysteresis
    }

    private fun onMove(x: Float, y: Float) {
        if (pressConsumed) return
        if (spaceSliding || pressed?.spec?.code == K.SPACE) {
            val dx = x - spaceDownX
            val dy = y - spaceDownY
            if (!spaceSliding && abs(dx) > spaceSwipeSlop && abs(dx) >= abs(dy)) {
                spaceSliding = true
                manualShift = false
                handler.removeCallbacks(spaceLongPress)
                invalidate()
            }
            if (spaceSliding) {
                val steps = ((x - spaceLastStepX) / spaceStepPx).toInt()
                if (steps != 0) {
                    spaceLastStepX += steps * spaceStepPx
                    val dir = if (steps > 0) 1 else -1
                    repeat(abs(steps)) { listener?.onCursorMove(dir) }
                    performHapticFeedback(HapticFeedbackConstants.CLOCK_TICK)
                }
                return
            }
        }
        // Keep the current key locked while the finger is still inside its hysteresis
        // zone so slight vertical thumb-roll between the letter row and number row
        // never jumps to the adjacent row mid-tap.
        val current = pressed
        if (current != null && withinHysteresis(current, x, y)) return
        val key = hit(x, y)
        if (key !== current) {
            pressed = key
            downX = x
            downY = y
            handler.removeCallbacks(repeatDelete)
            handler.removeCallbacks(repeatSelect)
            handler.removeCallbacks(delLongPress)
            handler.removeCallbacks(spaceLongPress)
            handler.removeCallbacks(hintLongPress)
            if (key?.spec?.code == K.DEL) {
                if (holdDeleteSelect) handler.postDelayed(delLongPress, 400)
                else handler.postDelayed(repeatDelete, 400)
            }
            else if (key?.spec?.longPressChar != null) handler.postDelayed(hintLongPress, 450)
            invalidate()
        }
    }

    private fun onUp(x: Float, y: Float) {
        val p = pressed ?: return
        val wasSliding = spaceSliding
        val wasConsumed = pressConsumed
        val shouldDispatch = !wasSliding && p.spec.code != K.DEL && p.spec.code != K.SPACER &&
            (withinHysteresis(p, x, y) || hit(x, y) === p)
        cancelPress()
        if (holdDeleteSelect && !wasConsumed && p.spec.code == K.DEL &&
            (withinHysteresis(p, x, y) || hit(x, y) === p)) {
            // Quick tap in select-on-hold mode: delete once on release.
            listener?.onDelete()
            return
        }
        if (shouldDispatch) {
            dispatch(p)
        }
    }

    private fun cancelPress() {
        val released = pressed
        if (popupEnabled && released != null && released.spec.code > 0) {
            popupKey = released
            popupHoldUntil = SystemClock.uptimeMillis() + popupHoldMs
            postInvalidateDelayed(popupHoldMs + 10)
        } else {
            popupKey = null
        }
        pressed = null
        spaceSliding = false
        handler.removeCallbacks(repeatDelete)
        handler.removeCallbacks(repeatSelect)
        handler.removeCallbacks(delLongPress)
        handler.removeCallbacks(spaceLongPress)
        handler.removeCallbacks(hintLongPress)
        invalidate()
    }

    override fun onDetachedFromWindow() {
        cancelPress()
        super.onDetachedFromWindow()
    }

    private fun hit(x: Float, y: Float): Key? {
        for (i in 0 until keys.size) {
            val k = keys[i]
            if (k.spec.code != K.SPACER && k.rect.contains(x, y)) return k
        }
        // Snap touches landing in the narrow gap between keys/rows to the closest key.
        val padX = keyGap * 0.6f
        val padY = rowGap * 0.6f
        var best: Key? = null
        var bestDistSq = Float.MAX_VALUE
        for (i in 0 until keys.size) {
            val k = keys[i]
            if (k.spec.code == K.SPACER) continue
            val r = k.rect
            if (x >= r.left - padX && x <= r.right + padX &&
                y >= r.top - padY && y <= r.bottom + padY
            ) {
                val dx = x - r.centerX()
                val dy = y - r.centerY()
                val distSq = dx * dx + dy * dy
                if (distSq < bestDistSq) {
                    bestDistSq = distSq
                    best = k
                }
            }
        }
        return best
    }

    private fun dispatch(key: Key) {
        when (key.spec.code) {
            K.SHIFT -> {
                val now = SystemClock.uptimeMillis()
                if (capsLock) {
                    capsLock = false; shifted = false; manualShift = false
                } else if (shifted && now - shiftTapTime < 350) {
                    capsLock = true; manualShift = false
                } else {
                    shifted = !shifted; manualShift = true
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
            K.ENTER -> {
                manualShift = false
                listener?.onEnter()
            }
            K.SPACE, K.NUM_SPACE -> {
                manualShift = false
                listener?.onChar(' ')
            }
            K.HIDE -> listener?.onHideKeyboard()
            K.IME -> listener?.onSwitchIme()
            else -> {
                manualShift = false
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
