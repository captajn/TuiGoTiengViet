package com.tuigo.ime

import android.inputmethodservice.InputMethodService
import android.annotation.SuppressLint
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
                keyboard.currentMethod = Prefs.inputMethod(this)
                keyboard.showToolbar = Prefs.toolbar(this)
                keyboard.setMetrics(Prefs.keyHeightDp(this), Prefs.keyGapDp(this))
                suggestionScroll?.setBackgroundColor(keyboard.theme.bg)
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

    /** Inline-autofill suggestion strip (Gboard-style chips above the keys).
     *  Content is rendered by the autofill provider — we never see the data. */
    private var suggestionScroll: HorizontalScrollView? = null
    private var suggestionStrip: LinearLayout? = null
    private var pendingInline: InlineSuggestionsResponse? = null
    private var inlineSeq = 0
    private var editorKey: String? = null

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
    }

    override fun onDestroy() {
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
        val pad = (4 * dm).toInt()
        val strip = LinearLayout(this).apply {
            orientation = LinearLayout.HORIZONTAL
            setPadding(pad, pad, pad, pad)
        }
        val scroll = HorizontalScrollView(this).apply {
            isHorizontalScrollBarEnabled = false
            isFillViewport = false
            visibility = View.GONE
            setBackgroundColor(keyboard.theme.bg)
            addView(strip)
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
        suggestionScroll?.setBackgroundColor(keyboard.theme.bg)
        if (Build.VERSION.SDK_INT >= 30) renderInlineSuggestions()
        keyboard.theme = KeyboardThemes.current(this)
        keyboard.vietnameseOn = vietOn
        keyboard.currentMethod = Prefs.inputMethod(this)
        keyboard.showToolbar = Prefs.toolbar(this)
        keyboard.showNumberRow = usesNumberRow()
        keyboard.numericLayout = numericLayoutFor(info)
        keyboard.setMetrics(Prefs.keyHeightDp(this), Prefs.keyGapDp(this))
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

    override fun onQuickMethodToggle() {
        val current = Prefs.inputMethod(this)
        val next = when (current) {
            ImTelex -> ImVni
            ImVni -> ImViqr
            else -> ImTelex
        }
        Prefs.get(this).edit().putInt(Prefs.KEY_IM, next).apply()
        loadEngineConfig()
        if (::keyboard.isInitialized) {
            keyboard.currentMethod = next
            keyboard.showNumberRow = usesNumberRow()
            keyboard.invalidate()
        }
    }

    override fun onToggleNumberRow() {
        if (!::keyboard.isInitialized) return
        val next = !keyboard.showNumberRow
        Prefs.get(this).edit().putBoolean(Prefs.KEY_NUMBER_ROW, next).apply()
        keyboard.showNumberRow = next
    }

    override fun onCursorMove(delta: Int) {
        val ic = currentInputConnection ?: return
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

    // ---- inline autofill (API 30+) ---------------------------------------------
    // Called only on API 30+ — references to the inline types resolve lazily.

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
        renderInlineSuggestions()
        return true
    }

    private fun clearInlineSuggestions() {
        pendingInline = null
        inlineSeq++
        suggestionStrip?.removeAllViews()
        suggestionScroll?.visibility = View.GONE
    }

    @SuppressLint("NewApi")
    private fun renderInlineSuggestions() {
        val scroll = suggestionScroll ?: return
        val strip = suggestionStrip ?: return
        val suggestions = pendingInline?.inlineSuggestions.orEmpty()
        inlineSeq++ // drop callbacks still in flight from a previous response
        strip.removeAllViews()
        if (suggestions.isEmpty()) {
            scroll.visibility = View.GONE
            return
        }
        scroll.visibility = View.VISIBLE
        scroll.setBackgroundColor(keyboard.theme.bg)
        val dm = resources.displayMetrics.density
        val chipH = (40 * dm).toInt()
        val gap = (6 * dm).toInt()
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

    private fun isRestrictedField(info: EditorInfo?): Boolean {
        if (info == null) return true
        if (info.inputType == InputType.TYPE_NULL) return true
        val klass = info.inputType and InputType.TYPE_MASK_CLASS
        val variation = info.inputType and InputType.TYPE_MASK_VARIATION
        if (klass == InputType.TYPE_CLASS_PHONE || klass == InputType.TYPE_CLASS_NUMBER ||
            klass == InputType.TYPE_CLASS_DATETIME) return true
        return klass == InputType.TYPE_CLASS_TEXT &&
            (variation == InputType.TYPE_TEXT_VARIATION_PASSWORD ||
             variation == InputType.TYPE_TEXT_VARIATION_VISIBLE_PASSWORD ||
             variation == InputType.TYPE_TEXT_VARIATION_WEB_PASSWORD)
    }
}
