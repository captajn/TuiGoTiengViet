package com.tuigo.ime

import android.inputmethodservice.InputMethodService
import android.annotation.SuppressLint
import android.content.ClipboardManager
import android.graphics.Canvas
import android.graphics.Paint
import android.graphics.RectF
import android.graphics.Typeface
import android.graphics.drawable.GradientDrawable
import android.os.Bundle
import android.os.SystemClock
import android.os.Build
import android.content.res.ColorStateList
import android.content.res.Configuration
import android.content.SharedPreferences
import android.util.Size
import android.util.TypedValue
import android.view.WindowInsetsController
import com.tuigo.UiTheme
import android.text.InputType
import android.text.TextUtils
import android.view.Gravity
import android.view.KeyEvent
import android.view.View
import android.view.ViewGroup
import android.view.inputmethod.EditorInfo
import android.view.inputmethod.InputMethodManager
import android.view.inputmethod.InlineSuggestionsRequest
import android.view.inputmethod.InlineSuggestionsResponse
import android.widget.HorizontalScrollView
import android.widget.ImageView
import android.widget.LinearLayout
import android.widget.PopupWindow
import android.widget.TextView
import android.widget.inline.InlinePresentationSpec
import androidx.autofill.inline.UiVersions
import androidx.autofill.inline.common.ImageViewStyle
import androidx.autofill.inline.common.TextViewStyle
import androidx.autofill.inline.common.ViewStyle
import androidx.autofill.inline.v1.InlineSuggestionUi
import com.tuigo.KeyboardThemes
import com.tuigo.Prefs
import com.tuigo.engine.Engine
import com.tuigo.engine.ImTelex
import com.tuigo.engine.ImVni
import com.tuigo.engine.ImViqr
import com.tuigo.engine.Result
import com.tuigo.MacroFiles
import com.tuigo.engine.loadMacroText

/**
 * InputMethodService shell around the pure-Kotlin engine port.
 *
 * Engine contract (same as the Windows host):
 *   filter(code) -> Result(consumed, backs, out, keyOut)
 *   consumed   : engine handled the key; apply backs+out
 *   !consumed  : pass-through; we commit the raw char ourselves
 *   backs/out  : UTF-16 units — matches InputConnection.deleteSurroundingText
 */
class TuiGoImeService : InputMethodService(), KeyboardView.Listener {
    private val preferenceListener = SharedPreferences.OnSharedPreferenceChangeListener { _, key ->
        if (key != Prefs.KEY_VIET_ON) {
            loadEngineConfig()
            engine.resetBuf(); spacePending = false
            if (::keyboard.isInitialized) {
                keyboard.theme = KeyboardThemes.current(this)
                keyboard.showNumberRow = usesNumberRow()
                keyboard.setMetrics(Prefs.keyHeightDp(this), Prefs.keyGapDp(this))
                suggestionScroll?.setBackgroundColor(keyboard.theme.bg)
                renderSuggestionStrip()
                applySystemBarTheme()
            }
        }
    }

    private lateinit var engine: Engine
    private lateinit var keyboard: KeyboardView

    private var vietOn = true
    private var doubleSpace = true

    /** Host app's package — per-app VN/EN memory (Windows-style). */
    private var hostPkg: String? = null
    private var spacePopup: PopupWindow? = null

    /** Inline-autofill & clipboard suggestion strip (Gboard-style chips above the keys). */
    private var suggestionScroll: HorizontalScrollView? = null
    private var suggestionStrip: LinearLayout? = null
    private var pendingInline: InlineSuggestionsResponse? = null
    private var inlineSeq = 0
    private var editorKey: String? = null

    private var clipboardManager: ClipboardManager? = null
    private var clipCopiedAtMs = 0L
    private var lastKnownClipText: String? = null
    private var dismissedClipStamp = -1L
    private var dismissedClipText: String? = null
    private val clipChangedListener = ClipboardManager.OnPrimaryClipChangedListener {
        clipCopiedAtMs = System.currentTimeMillis()
        dismissedClipStamp = -1L
        dismissedClipText = null
        if (::keyboard.isInitialized) renderSuggestionStrip()
    }

    /** Cursor position we expect after our own edits — lets us distinguish
     *  external selection changes (cursor tap) from our own commits. */
    private var expectedSel = -1

    /** Positions the cursor may legitimately report while echoing our own
     *  edits. Apps deliver onUpdateSelection asynchronously; some report an
     *  intermediate position (after deleteSurroundingText, before commitText)
     *  instead of one update per batch. Treating those echoes as external
     *  cursor moves would reset the engine mid-word (e.g. "owr" -> "ơr"). */
    private val ownSelEchoes = HashSet<Int>()

    /** Pending "double space -> ." state. */
    private var spaceAt = 0L
    private var spacePending = false

    override fun onCreate() {
        super.onCreate()
        engine = Engine()
        loadEngineConfig()
        Prefs.get(this).registerOnSharedPreferenceChangeListener(preferenceListener)
        clipboardManager = getSystemService(ClipboardManager::class.java)
        clipboardManager?.addPrimaryClipChangedListener(clipChangedListener)
    }

    override fun onDestroy() {
        clipboardManager?.removePrimaryClipChangedListener(clipChangedListener)
        Prefs.get(this).unregisterOnSharedPreferenceChangeListener(preferenceListener)
        super.onDestroy()
    }

    private fun loadEngineConfig() {
        Prefs.applyTo(engine, this)
        vietOn = Prefs.vietForApp(this, hostPkg) ?: Prefs.vietOn(this)
        engine.setVietKey(vietOn)
        doubleSpace = Prefs.doubleSpacePeriod(this)
        // Reload on every activation, including an empty/deleted macro file.
        // Preserve the last readable list only on a genuine I/O failure.
        try {
            engine.loadMacroText(MacroFiles.read(this))
        } catch (_: java.io.IOException) { }
    }

    /** VNI always gets the number row; other modes follow the user's toggle. */
    private fun usesNumberRow(): Boolean =
        Prefs.numberRow(this) || Prefs.inputMethod(this) == ImVni

    override fun onCreateInputView(): View {
        keyboard = KeyboardView(this)
        keyboard.listener = this
        keyboard.theme = KeyboardThemes.current(this)
        keyboard.vietnameseOn = vietOn
        keyboard.showNumberRow = usesNumberRow()
        keyboard.setMetrics(Prefs.keyHeightDp(this), Prefs.keyGapDp(this))
        val dm = resources.displayMetrics.density
        val pad = (6 * dm).toInt()
        val strip = LinearLayout(this).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.CENTER
            setPadding(pad, (6 * dm).toInt(), pad, (2 * dm).toInt())
        }
        val scroll = HorizontalScrollView(this).apply {
            isHorizontalScrollBarEnabled = false
            isFillViewport = true
            visibility = View.GONE
            setBackgroundColor(keyboard.theme.bg)
            addView(strip, ViewGroup.LayoutParams(ViewGroup.LayoutParams.WRAP_CONTENT, ViewGroup.LayoutParams.WRAP_CONTENT))
        }
        suggestionStrip = strip
        suggestionScroll = scroll
        return LinearLayout(this).apply {
            orientation = LinearLayout.VERTICAL
            addView(scroll, LinearLayout.LayoutParams(-1, -2))
            addView(keyboard, LinearLayout.LayoutParams(-1, -2))
        }
    }

    // Always show the soft keyboard even when a hardware keyboard is attached
    // (emulator, Chromebook) — same behavior as Gboard; the default impl hides
    // us whenever config.keyboard != NOKEYS.
    override fun onEvaluateInputViewShown(): Boolean {
        super.onEvaluateInputViewShown()
        return true
    }

    // Keep the host app visible in landscape. The framework's fullscreen
    // extract editor otherwise introduces an unrelated light panel in dark mode.
    override fun onEvaluateFullscreenMode(): Boolean {
        super.onEvaluateFullscreenMode()
        return false
    }

    override fun onStartInput(info: EditorInfo?, restarting: Boolean) {
        super.onStartInput(info, restarting)
        // A different editor means any pending autofill suggestions are stale —
        // the platform issues a fresh request/response for the new field.
        val key = "${info?.packageName}/${info?.fieldId}"
        if (key != editorKey) {
            editorKey = key
            clearInlineSuggestions()
        }
    }

    override fun onStartInputView(info: EditorInfo?, restarting: Boolean) {
        super.onStartInputView(info, restarting)
        hostPkg = info?.packageName
        loadEngineConfig()
        keyboard.theme = KeyboardThemes.current(this)
        keyboard.vietnameseOn = vietOn
        keyboard.showNumberRow = usesNumberRow()
        keyboard.numericLayout = numericLayoutFor(info)
        keyboard.setMetrics(Prefs.keyHeightDp(this), Prefs.keyGapDp(this))
        suggestionScroll?.setBackgroundColor(keyboard.theme.bg)
        renderSuggestionStrip()
        engine.resetBuf()
        ownSelEchoes.clear()
        expectedSel = info?.initialSelEnd ?: -1
        if (expectedSel >= 0) ownSelEchoes.add(expectedSel)
        spacePending = false
        applySystemBarTheme()
    }

    @Suppress("DEPRECATION")
    private fun applySystemBarTheme() {
        val imeWindow = window?.window ?: return
        val light = !UiTheme.isDark(this)
        imeWindow.navigationBarColor = if (::keyboard.isInitialized) keyboard.theme.bg else UiTheme.palette(this).bg
        if (Build.VERSION.SDK_INT >= 30) {
            imeWindow.insetsController?.setSystemBarsAppearance(
                if (light) WindowInsetsController.APPEARANCE_LIGHT_NAVIGATION_BARS else 0,
                WindowInsetsController.APPEARANCE_LIGHT_NAVIGATION_BARS)
        } else {
            val flags = imeWindow.decorView.systemUiVisibility
            imeWindow.decorView.systemUiVisibility = if (light) flags or View.SYSTEM_UI_FLAG_LIGHT_NAVIGATION_BAR else flags and View.SYSTEM_UI_FLAG_LIGHT_NAVIGATION_BAR.inv()
        }
    }

    override fun onConfigurationChanged(newConfig: Configuration) {
        super.onConfigurationChanged(newConfig)
        applySystemBarTheme()
    }

    override fun onFinishInputView(finishingInput: Boolean) {
        super.onFinishInputView(finishingInput)
        engine.resetBuf()
        ownSelEchoes.clear()
        expectedSel = -1
    }

    override fun onUpdateSelection(
        oldSelStart: Int, oldSelEnd: Int,
        newSelStart: Int, newSelEnd: Int,
        candidatesStart: Int, candidatesEnd: Int,
    ) {
        super.onUpdateSelection(oldSelStart, oldSelEnd, newSelStart, newSelEnd,
            candidatesStart, candidatesEnd)
        if (newSelStart != expectedSel || newSelStart != newSelEnd) {
            if (expectedSel < 0) {
                // Never synced (app did not report an initial selection) —
                // adopt the position without discarding the input buffer.
                expectedSel = newSelStart
            } else if (newSelStart == newSelEnd && ownSelEchoes.remove(newSelStart)) {
                // Intermediate echo of our own edit: keep expectedSel at the
                // final position our pending edit will produce.
            } else {
                // cursor was moved by the app/user, not by our commits
                engine.resetBuf()
                spacePending = false
                ownSelEchoes.clear()
                expectedSel = newSelStart
            }
        }
    }

    // ---- keyboard events -----------------------------------------------------

    override fun onChar(ch: Char) {
        val ic = currentInputConnection ?: return
        val ei = currentInputEditorInfo
        dismissCurrentClipSuggestion()

        if (isRestrictedField(ei)) {
            ic.commitText(ch.toString(), 1)
            bumpExpected(1)
            spacePending = false
            return
        }

        if (ch == ' ' && doubleSpace && spacePending &&
            SystemClock.uptimeMillis() - spaceAt < 700
        ) {
            spacePending = false
            ic.beginBatchEdit()
            ic.deleteSurroundingText(1, 0)
            ic.commitText(". ", 1)
            ic.endBatchEdit()
            if (expectedSel >= 0) {
                expectEcho(expectedSel - 1)
                expectedSel += 1 // -1 + 2
                expectEcho(expectedSel)
            }
            engine.resetBuf()
            return
        }

        if (!vietOn) {
            // engine doesn't gate vietKey inside process() — host bypasses it
            ic.commitText(ch.toString(), 1)
            bumpExpected(1)
            spacePending = ch == ' '
            if (spacePending) spaceAt = SystemClock.uptimeMillis()
            return
        }

        engine.shiftPressed = keyboard.shifted
        engine.capsLockOn = keyboard.capsLock
        val res = engine.filter(ch.code)
        if (res.consumed) {
            applyResult(res)
        } else {
            ic.commitText(ch.toString(), 1)
            bumpExpected(1)
        }
        spacePending = ch == ' '
        if (spacePending) spaceAt = SystemClock.uptimeMillis()
    }

    override fun onDelete() {
        val ic = currentInputConnection ?: return
        dismissCurrentClipSuggestion()
        spacePending = false
        val res = engine.backspace()
        if (res.consumed) {
            applyResult(res)
        } else {
            ic.deleteSurroundingText(1, 0)
            bumpExpected(-1)
        }
    }

    override fun onEnter() {
        dismissCurrentClipSuggestion()
        spacePending = false
        engine.resetBuf()
        if (!sendDefaultEditorAction(true)) {
            sendDownUpKeyEvents(KeyEvent.KEYCODE_ENTER)
        }
    }

    override fun onShiftChanged(shifted: Boolean, capsLock: Boolean) {
        engine.shiftPressed = shifted
        engine.capsLockOn = capsLock
    }

    override fun onVietToggle() {
        setVietOn(!vietOn)
    }

    private fun setVietOn(on: Boolean) {
        vietOn = on
        Prefs.setViet(this, hostPkg, on)
        keyboard.vietnameseOn = on
        engine.setVietKey(on)
        engine.resetBuf()
        spacePending = false
    }

    override fun onCursorMove(delta: Int) {
        currentInputConnection ?: return
        engine.resetBuf()
        spacePending = false
        sendDownUpKeyEvents(if (delta < 0) KeyEvent.KEYCODE_DPAD_LEFT else KeyEvent.KEYCODE_DPAD_RIGHT)
    }

    /** Long-press Space: quick settings for language and input method. */
    override fun onSpaceMenu() {
        val t = keyboard.theme
        val dm = resources.displayMetrics.density
        fun dp(v: Int) = (v * dm).toInt()
        val panel = LinearLayout(this).apply {
            orientation = LinearLayout.VERTICAL
            setPadding(dp(6), dp(6), dp(6), dp(6))
            background = GradientDrawable().apply {
                setColor(t.popup)
                cornerRadius = dp(12).toFloat()
                if (t.popupBorder ushr 24 != 0) setStroke(dp(1), t.popupBorder)
            }
        }
        fun section(label: String) {
            panel.addView(TextView(this).apply {
                text = label; textSize = 11f; setTextColor(t.popupText); alpha = 0.65f
                typeface = Typeface.DEFAULT_BOLD
                setPadding(dp(12), dp(6), dp(12), dp(2))
            })
        }
        fun divider() {
            panel.addView(View(this).apply {
                setBackgroundColor(t.popupBorder); alpha = 0.7f
                layoutParams = LinearLayout.LayoutParams(-1, maxOf(1, dp(1))).apply {
                    topMargin = dp(4); bottomMargin = dp(4)
                }
            })
        }
        fun item(label: String, checked: Boolean, action: () -> Unit) {
            panel.addView(TextView(this).apply {
                text = (if (checked) "✓ " else "    ") + label
                textSize = 15f; setTextColor(t.popupText)
                if (checked) typeface = Typeface.DEFAULT_BOLD
                setPadding(dp(12), dp(10), dp(28), dp(10))
                setOnClickListener { action(); spacePopup?.dismiss() }
            })
        }
        section("Ngôn ngữ")
        item("Tiếng Việt", vietOn) { setVietOn(true) }
        item("English", !vietOn) { setVietOn(false) }
        divider()
        section("Kiểu gõ")
        val current = Prefs.inputMethod(this)
        item("Telex", current == ImTelex) { Prefs.get(this).edit().putInt(Prefs.KEY_IM, ImTelex).apply() }
        item("VNI", current == ImVni) { Prefs.get(this).edit().putInt(Prefs.KEY_IM, ImVni).apply() }
        item("VIQR", current == ImViqr) { Prefs.get(this).edit().putInt(Prefs.KEY_IM, ImViqr).apply() }
        divider()
        item("Chọn bàn phím khác…", false) { onSwitchIme() }
        spacePopup?.dismiss()
        spacePopup = PopupWindow(panel, ViewGroup.LayoutParams.WRAP_CONTENT,
            ViewGroup.LayoutParams.WRAP_CONTENT, true).apply {
            elevation = dp(10).toFloat()
            isOutsideTouchable = true
            showAtLocation(keyboard, Gravity.BOTTOM or Gravity.CENTER_HORIZONTAL,
                0, keyboard.height + dp(64))
        }
    }

    override fun onSwitchIme() {
        getSystemService(InputMethodManager::class.java)?.showInputMethodPicker()
    }

    // ---- inline autofill & clipboard suggestion strip --------------------------

    @SuppressLint("NewApi") // override is only invoked on API 30+
    override fun onCreateInlineSuggestionsRequest(uiExtras: Bundle): InlineSuggestionsRequest? {
        if (!UiVersions.getVersions(uiExtras).contains(UiVersions.INLINE_UI_VERSION_1)) return null
        val dm = resources.displayMetrics.density
        val chipH = (40 * dm).toInt()
        val t = KeyboardThemes.current(this)
        val padH = (14 * dm).toInt()
        val chip = ViewStyle.Builder()
            .setBackgroundColor(t.key)
            .setPadding(padH, 0, padH, 0)
            .build()
        val singleChip = ViewStyle.Builder()
            .setBackgroundColor(t.key)
            .build()
        val title = TextViewStyle.Builder()
            .setTextColor(t.text)
            .setTextSize(TypedValue.COMPLEX_UNIT_SP, 14f)
            .build()
        val subtitle = TextViewStyle.Builder()
            .setTextColor(t.spaceText)
            .setTextSize(TypedValue.COMPLEX_UNIT_SP, 12f)
            .build()
        // Action icons get our glyph tint; provider app icons stay untinted.
        val actionIcon = ImageViewStyle.Builder()
            .setTintList(ColorStateList.valueOf(t.funcText))
            .build()
        val appIcon = ImageViewStyle.Builder()
            .setScaleType(ImageView.ScaleType.FIT_CENTER)
            .build()
        val styles = UiVersions.newStylesBuilder()
            .addStyle(
                InlineSuggestionUi.newStyleBuilder()
                    .setChipStyle(chip)
                    .setSingleIconChipStyle(singleChip)
                    .setTitleStyle(title)
                    .setSubtitleStyle(subtitle)
                    .setStartIconStyle(appIcon)
                    .setEndIconStyle(actionIcon)
                    .setSingleIconChipIconStyle(actionIcon)
                    .build()
            )
            .build()
        val maxW = (resources.displayMetrics.widthPixels * 0.8f).toInt()
        val spec = InlinePresentationSpec.Builder(Size(chipH, chipH), Size(maxW, chipH))
            .setStyle(styles)
            .build()
        val request = InlineSuggestionsRequest.Builder(mutableListOf(spec))
            .setMaxSuggestionCount(InlineSuggestionsRequest.SUGGESTION_COUNT_UNLIMITED)
        if (Build.VERSION.SDK_INT >= 31) {
            val tooltipSpec = InlinePresentationSpec.Builder(
                Size(chipH, chipH), Size((200 * dm).toInt(), chipH))
                .setStyle(styles)
                .build()
            request.setInlineTooltipPresentationSpec(tooltipSpec)
        }
        return request.build()
    }

    @SuppressLint("NewApi")
    override fun onInlineSuggestionsResponse(response: InlineSuggestionsResponse): Boolean {
        pendingInline = response
        renderSuggestionStrip()
        return true
    }

    private fun clearInlineSuggestions() {
        pendingInline = null
        inlineSeq++
        renderSuggestionStrip()
    }

    private data class ClipCandidate(val raw: String, val label: String, val stamp: Long)

    private fun currentClipCandidate(): ClipCandidate? {
        if (!Prefs.clipboardSuggest(this)) return null
        if (isPasswordField(currentInputEditorInfo)) return null
        val cm = clipboardManager ?: getSystemService(ClipboardManager::class.java) ?: return null
        val clip = try { cm.primaryClip } catch (_: Exception) { null } ?: return null
        if (clip.itemCount <= 0) return null
        val raw = clip.getItemAt(0)?.text?.toString() ?: return null
        val trimmed = raw.trim()
        if (trimmed.isEmpty()) return null
        val desc = clip.description
        val descTs = desc?.timestamp ?: 0L
        if (raw != lastKnownClipText) {
            lastKnownClipText = raw
            clipCopiedAtMs = if (descTs > 0L) maxOf(descTs, clipCopiedAtMs) else System.currentTimeMillis()
        }
        val effectiveTs = maxOf(descTs, clipCopiedAtMs)
        if (effectiveTs <= 0L || System.currentTimeMillis() - effectiveTs > 180_000L) return null
        if (effectiveTs == dismissedClipStamp && raw == dismissedClipText) return null
        val isSensitive = desc?.extras?.getBoolean("android.content.extra.IS_SENSITIVE", false) == true
        val singleLine = trimmed.replace(Regex("\\s+"), " ")
        val label = if (isSensitive) "••••••••" else singleLine
        return ClipCandidate(raw, label, effectiveTs)
    }

    private fun dismissCurrentClipSuggestion() {
        val cand = currentClipCandidate() ?: return
        dismissedClipStamp = cand.stamp
        dismissedClipText = cand.raw
        renderSuggestionStrip()
    }

    private fun buildClipboardChip(cand: ClipCandidate): View {
        val dm = resources.displayMetrics.density
        fun dp(v: Float) = (v * dm).toInt()
        val t = keyboard.theme
        val pill = LinearLayout(this).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.CENTER_VERTICAL
            setPadding(dp(14f), dp(6f), dp(14f), dp(6f))
            minimumHeight = dp(34f)
            background = GradientDrawable().apply {
                shape = GradientDrawable.RECTANGLE
                cornerRadius = 18f * dm
                setColor(t.key)
                if (t.border ushr 24 != 0) setStroke(maxOf(1, dp(1f)), t.border)
            }
            contentDescription = "Dán từ bộ nhớ tạm"
            setOnClickListener {
                val ic = currentInputConnection ?: return@setOnClickListener
                dismissedClipStamp = cand.stamp
                dismissedClipText = cand.raw
                engine.resetBuf()
                spacePending = false
                ic.commitText(cand.raw, 1)
                bumpExpected(cand.raw.length)
                renderSuggestionStrip()
            }
        }
        val icon = object : View(this) {
            private val paint = Paint(Paint.ANTI_ALIAS_FLAG).apply {
                style = Paint.Style.STROKE
                strokeWidth = 1.6f * dm
                strokeCap = Paint.Cap.ROUND
                strokeJoin = Paint.Join.ROUND
                color = t.funcText
            }
            private val rect = RectF()
            override fun onDraw(canvas: Canvas) {
                val w = width.toFloat()
                val h = height.toFloat()
                rect.set(w * 0.18f, h * 0.20f, w * 0.82f, h * 0.90f)
                canvas.drawRoundRect(rect, 2f * dm, 2f * dm, paint)
                rect.set(w * 0.35f, h * 0.10f, w * 0.65f, h * 0.27f)
                canvas.drawRoundRect(rect, 1.5f * dm, 1.5f * dm, paint)
                canvas.drawLine(w * 0.32f, h * 0.48f, w * 0.68f, h * 0.48f, paint)
                canvas.drawLine(w * 0.32f, h * 0.67f, w * 0.58f, h * 0.67f, paint)
            }
        }.apply {
            layoutParams = LinearLayout.LayoutParams(dp(16f), dp(16f)).apply {
                marginEnd = dp(7f)
            }
        }
        val labelView = TextView(this).apply {
            text = cand.label
            textSize = 14f
            setTextColor(t.text)
            isSingleLine = true
            ellipsize = TextUtils.TruncateAt.END
            maxWidth = dp(220f)
        }
        pill.addView(icon)
        pill.addView(labelView)
        return pill
    }

    @SuppressLint("NewApi")
    private fun renderSuggestionStrip() {
        val scroll = suggestionScroll ?: return
        val strip = suggestionStrip ?: return
        val suggestions = if (Build.VERSION.SDK_INT >= 30) pendingInline?.inlineSuggestions.orEmpty() else emptyList()
        val clipCand = currentClipCandidate()
        inlineSeq++ // drop callbacks still in flight from a previous response
        strip.removeAllViews()
        if (suggestions.isEmpty() && clipCand == null) {
            scroll.visibility = View.GONE
            return
        }
        scroll.visibility = View.VISIBLE
        scroll.setBackgroundColor(keyboard.theme.bg)
        val dm = resources.displayMetrics.density
        val chipH = (36 * dm).toInt()
        val gap = (6 * dm).toInt()
        if (clipCand != null) {
            val chip = buildClipboardChip(clipCand)
            chip.layoutParams = LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.WRAP_CONTENT,
                ViewGroup.LayoutParams.WRAP_CONTENT
            ).apply {
                if (suggestions.isNotEmpty()) marginEnd = gap
            }
            strip.addView(chip)
        }
        if (Build.VERSION.SDK_INT >= 30 && suggestions.isNotEmpty()) {
            val seq = inlineSeq
            for (suggestion in suggestions) {
                suggestion.inflate(this,
                    Size(ViewGroup.LayoutParams.WRAP_CONTENT, chipH), mainExecutor) { view ->
                    if (seq != inlineSeq || view == null) return@inflate
                    view.layoutParams = LinearLayout.LayoutParams(view.layoutParams)
                        .apply { marginEnd = gap }
                    strip.addView(view)
                }
            }
        }
    }

    override fun onHideKeyboard() {
        requestHideSelf(0)
    }

    // ---- output --------------------------------------------------------------

    private fun applyResult(res: Result) {
        val ic = currentInputConnection ?: return
        ic.beginBatchEdit()
        if (res.backs > 0) ic.deleteSurroundingText(res.backs, 0)
        if (res.out.isNotEmpty()) ic.commitText(String(res.out), 1)
        ic.endBatchEdit()
        if (expectedSel >= 0) {
            expectEcho(expectedSel)
            if (res.backs > 0) expectEcho(expectedSel - res.backs)
            expectedSel = expectedSel - res.backs + res.out.size
            expectEcho(expectedSel)
        }
    }

    private fun bumpExpected(delta: Int) {
        if (expectedSel >= 0) {
            expectEcho(expectedSel)
            expectedSel += delta
            expectEcho(expectedSel)
        }
    }

    private fun expectEcho(pos: Int) {
        if (pos < 0) return
        ownSelEchoes.add(pos)
        if (ownSelEchoes.size > 128) ownSelEchoes.clear()
    }

    private fun numericLayoutFor(info: EditorInfo?): KeyboardView.NumericLayout =
        when ((info?.inputType ?: 0) and InputType.TYPE_MASK_CLASS) {
            InputType.TYPE_CLASS_NUMBER -> KeyboardView.NumericLayout.NUMBER
            InputType.TYPE_CLASS_PHONE -> KeyboardView.NumericLayout.PHONE
            InputType.TYPE_CLASS_DATETIME -> KeyboardView.NumericLayout.DATETIME
            else -> KeyboardView.NumericLayout.NONE
        }

    private fun isPasswordField(info: EditorInfo?): Boolean {
        if (info == null) return false
        val klass = info.inputType and InputType.TYPE_MASK_CLASS
        val variation = info.inputType and InputType.TYPE_MASK_VARIATION
        return klass == InputType.TYPE_CLASS_TEXT &&
            (variation == InputType.TYPE_TEXT_VARIATION_PASSWORD ||
             variation == InputType.TYPE_TEXT_VARIATION_VISIBLE_PASSWORD ||
             variation == InputType.TYPE_TEXT_VARIATION_WEB_PASSWORD)
    }

    private fun isRestrictedField(info: EditorInfo?): Boolean {
        if (info == null) return true
        if (info.inputType == InputType.TYPE_NULL) return true
        val klass = info.inputType and InputType.TYPE_MASK_CLASS
        if (klass == InputType.TYPE_CLASS_PHONE || klass == InputType.TYPE_CLASS_NUMBER ||
            klass == InputType.TYPE_CLASS_DATETIME) return true
        return isPasswordField(info)
    }
}
