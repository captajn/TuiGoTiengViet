package com.tuigo.ime

import android.inputmethodservice.InputMethodService
import android.annotation.SuppressLint
import android.content.ClipboardManager
import android.graphics.drawable.Icon
import android.graphics.BitmapFactory
import android.graphics.Canvas
import android.graphics.Color
import android.graphics.Paint
import android.graphics.RectF
import android.graphics.Typeface
import android.graphics.drawable.GradientDrawable
import android.os.Bundle
import android.os.Handler
import android.os.Looper
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
import android.view.SurfaceControl
import android.view.SurfaceView
import android.view.View
import android.view.ViewGroup
import android.view.inputmethod.EditorInfo
import android.view.inputmethod.InputMethodManager
import android.view.inputmethod.InlineSuggestion
import android.view.inputmethod.InlineSuggestionsRequest
import android.view.inputmethod.InlineSuggestionsResponse
import android.widget.HorizontalScrollView
import android.widget.ImageView
import android.widget.LinearLayout
import android.widget.PopupWindow
import android.widget.ScrollView
import android.widget.TextView
import android.widget.inline.InlineContentView
import android.widget.inline.InlinePresentationSpec
import androidx.autofill.inline.UiVersions
import androidx.autofill.inline.common.ImageViewStyle
import androidx.autofill.inline.common.TextViewStyle
import androidx.autofill.inline.common.ViewStyle
import androidx.autofill.inline.v1.InlineSuggestionUi
import com.tuigo.KeyboardThemes
import com.tuigo.Prefs
import com.tuigo.R
import com.tuigo.engine.Engine
import com.tuigo.engine.ImTelex
import com.tuigo.engine.ImVni
import com.tuigo.engine.ImViqr
import com.tuigo.engine.Result
import com.tuigo.MacroFiles
import com.tuigo.engine.loadMacroText
import com.google.android.flexbox.FlexWrap
import com.google.android.flexbox.FlexboxLayout
import com.google.android.flexbox.JustifyContent
import java.util.concurrent.Executor

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
                keyboard.vniMode = Prefs.inputMethod(this) == ImVni
                keyboard.showNumberRow = usesNumberRow()
                keyboard.setMetrics(Prefs.keyHeightDp(this), Prefs.keyGapDp(this))
                suggestionScroll?.setBackgroundColor(keyboard.theme.bg)
                renderSuggestionStrip()
                updateAutoCapitalization()
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
    private var suggestionBar: android.widget.FrameLayout? = null
    private var suggestionScroll: HorizontalScrollView? = null
    private var suggestionSurface: SurfaceView? = null
    private var suggestionStrip: FlexboxLayout? = null
    private var pinnedSuggestionHost: android.widget.FrameLayout? = null
    private val scrollableInlineViews = mutableListOf<InlineContentView>()
    private var renderedClipKey: String? = null
    private var renderedHistoryOn = false
    private var renderedTheme: com.tuigo.KeyTheme? = null
    private var inlineGeneration = 0
    private var hasInlineSuggestions = false
    private val mainHandler = Handler(Looper.getMainLooper())

    private var clipboardManager: ClipboardManager? = null
    private var clipCopiedAtMs = 0L
    private var lastKnownClipText: String? = null
    private var dismissedClipStamp = -1L
    private var dismissedClipText: String? = null
    /** URI we just re-queued from the history panel — skip re-recording it. */
    private var replayedClipUri: String? = null
    private var activeClipCandidate: ClipCandidate? = null
    private val whitespaceRegex = Regex("\\s+")
    private val clipChangedListener = ClipboardManager.OnPrimaryClipChangedListener {
        clipCopiedAtMs = System.currentTimeMillis()
        dismissedClipStamp = -1L
        dismissedClipText = null
        recordClipToHistory()
        if (::keyboard.isInitialized) renderSuggestionStrip()
    }

    /** Samsung-style history: store each new clip while the IME is alive.
     *  Never record inside a password field — likely a secret. */
    private fun recordClipToHistory() {
        if (!Prefs.clipboardHistory(this)) return
        if (isPasswordField(currentInputEditorInfo)) return
        val clip = try { clipboardManager?.primaryClip } catch (_: Exception) { null } ?: return
        val uri = (0 until clip.itemCount)
            .mapNotNull { clip.getItemAt(it)?.uri?.toString() }.firstOrNull()
        if (uri != null && uri == replayedClipUri) { replayedClipUri = null; return }
        ClipboardHistory.record(this, clip, clip.description)
    }

    /** Cursor position we expect after our own edits — lets us distinguish
     *  external selection changes (cursor tap) from our own commits. */
    private var expectedSel = -1
    private var selStart = -1
    private var selEnd = -1

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
        syncActiveSubtype()
        vietOn = Prefs.vietForApp(this, hostPkg) ?: Prefs.vietOn(this)
        engine.setVietKey(vietOn)
        doubleSpace = Prefs.doubleSpacePeriod(this)
        // Reload on every activation, including an empty/deleted macro file.
        // Preserve the last readable list only on a genuine I/O failure.
        try {
            engine.loadMacroText(MacroFiles.read(this))
        } catch (_: java.io.IOException) { }
    }

    private fun syncActiveSubtype() {
        val subtype = Prefs.syncSystemSubtype(this) ?: return
        if (Build.VERSION.SDK_INT >= 28) {
            try {
                val imm = getSystemService(InputMethodManager::class.java) ?: return
                if (imm.currentInputMethodSubtype?.hashCode() != subtype.hashCode()) {
                    val imi = imm.inputMethodList?.firstOrNull { it.packageName == packageName } ?: return
                    switchInputMethod(imi.id, subtype)
                }
            } catch (_: Exception) { }
        }
    }

    /** VNI always gets the number row; other modes follow the user's toggle. */
    private fun usesNumberRow(): Boolean =
        Prefs.numberRow(this) || Prefs.inputMethod(this) == ImVni

    override fun onCreateInputView(): View {
        keyboard = KeyboardView(this)
        keyboard.listener = this
        keyboard.theme = KeyboardThemes.current(this)
        keyboard.vietnameseOn = vietOn
        keyboard.vniMode = Prefs.inputMethod(this) == ImVni
        keyboard.showNumberRow = usesNumberRow()
        keyboard.setMetrics(Prefs.keyHeightDp(this), Prefs.keyGapDp(this))
        val dm = resources.displayMetrics.density
        val pad = (6 * dm).toInt()
        val strip = FlexboxLayout(this).apply {
            flexWrap = FlexWrap.NOWRAP
            justifyContent = JustifyContent.FLEX_START
            alignItems = com.google.android.flexbox.AlignItems.CENTER
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
        val surface = SurfaceView(this).apply {
            setZOrderOnTop(true)
        }
        suggestionSurface = surface
        val pinnedHost = android.widget.FrameLayout(this)
        pinnedSuggestionHost = pinnedHost
        // Fixed row height (40dp) matching Fcitx inline suggestion bar.
        // The whole bar starts hidden — showing the container while the inner
        // scroll is GONE leaves a blank strip above the keys.
        val stripH = (40 * dm).toInt()
        return LinearLayout(this).apply {
            orientation = LinearLayout.VERTICAL
            addView(android.widget.FrameLayout(this@TuiGoImeService).apply {
                suggestionBar = this
                visibility = View.GONE
                addView(LinearLayout(this@TuiGoImeService).apply {
                    orientation = LinearLayout.HORIZONTAL
                    addView(android.widget.FrameLayout(this@TuiGoImeService).apply {
                        addView(scroll, android.widget.FrameLayout.LayoutParams(-1, -1))
                        addView(surface, android.widget.FrameLayout.LayoutParams(-1, -1))
                    }, LinearLayout.LayoutParams(0, -1, 1f))
                    addView(pinnedHost, LinearLayout.LayoutParams(-2, -1))
                }, android.widget.FrameLayout.LayoutParams(-1, -1))
            }, LinearLayout.LayoutParams(-1, stripH))
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
        clearInlineSuggestions()
    }

    override fun onStartInputView(info: EditorInfo?, restarting: Boolean) {
        super.onStartInputView(info, restarting)
        hostPkg = info?.packageName
        loadEngineConfig()
        keyboard.theme = KeyboardThemes.current(this)
        keyboard.vietnameseOn = vietOn
        keyboard.vniMode = Prefs.inputMethod(this) == ImVni
        keyboard.showNumberRow = usesNumberRow()
        keyboard.numericLayout = numericLayoutFor(info)
        keyboard.setMetrics(Prefs.keyHeightDp(this), Prefs.keyGapDp(this))
        suggestionScroll?.setBackgroundColor(keyboard.theme.bg)
        renderSuggestionStrip()
        engine.resetBuf()
        ownSelEchoes.clear()
        selStart = info?.initialSelStart ?: -1
        selEnd = info?.initialSelEnd ?: -1
        expectedSel = if (selStart >= 0 && selEnd >= 0) minOf(selStart, selEnd) else selEnd
        if (expectedSel >= 0) ownSelEchoes.add(expectedSel)
        spacePending = false
        keyboard.resetManualShift()
        updateAutoCapitalization()
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
        clearInlineSuggestions()
        clipPanel?.dismiss()
        spacePopup?.dismiss()
        engine.resetBuf()
        ownSelEchoes.clear()
        expectedSel = -1
        selStart = -1
        selEnd = -1
    }

    override fun onUpdateSelection(
        oldSelStart: Int, oldSelEnd: Int,
        newSelStart: Int, newSelEnd: Int,
        candidatesStart: Int, candidatesEnd: Int,
    ) {
        super.onUpdateSelection(oldSelStart, oldSelEnd, newSelStart, newSelEnd,
            candidatesStart, candidatesEnd)
        selStart = newSelStart
        selEnd = newSelEnd
        if (newSelStart != newSelEnd) {
            engine.resetBuf()
            spacePending = false
            ownSelEchoes.clear()
            expectedSel = minOf(newSelStart, newSelEnd)
            if (::keyboard.isInitialized) keyboard.resetManualShift()
            return
        }
        if (newSelStart != expectedSel) {
            if (expectedSel < 0) {
                // Never synced (app did not report an initial selection) —
                // adopt the position without discarding the input buffer.
                expectedSel = newSelStart
            } else if (ownSelEchoes.remove(newSelStart)) {
                // Intermediate echo of our own edit: keep expectedSel at the
                // final position our pending edit will produce.
                return
            } else {
                // cursor was moved by the app/user, not by our commits
                engine.resetBuf()
                spacePending = false
                ownSelEchoes.clear()
                expectedSel = newSelStart
                if (::keyboard.isInitialized) keyboard.resetManualShift()
            }
        }
        updateAutoCapitalization()
    }

    // ---- keyboard events -----------------------------------------------------

    override fun onChar(ch: Char) {
        val ic = currentInputConnection ?: return
        val ei = currentInputEditorInfo
        dismissCurrentClipSuggestion()

        val hasSelection = if (selStart >= 0 && selEnd >= 0) {
            selStart != selEnd
        } else {
            !ic.getSelectedText(0).isNullOrEmpty()
        }
        if (hasSelection) {
            engine.resetBuf()
            spacePending = false
            if (selStart >= 0 && selEnd >= 0) {
                val anchor = minOf(selStart, selEnd)
                expectedSel = anchor
                selStart = anchor
                selEnd = anchor
            }
        }

        if (isRestrictedField(ei)) {
            ic.commitText(ch.toString(), 1)
            bumpExpected(1)
            spacePending = false
            scheduleAutoCapitalizationUpdate()
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
                selStart = expectedSel
                selEnd = expectedSel
                expectEcho(expectedSel)
            }
            engine.resetBuf()
            scheduleAutoCapitalizationUpdate()
            return
        }

        if (!vietOn) {
            // engine doesn't gate vietKey inside process() — host bypasses it
            ic.commitText(ch.toString(), 1)
            bumpExpected(1)
            spacePending = ch == ' '
            if (spacePending) spaceAt = SystemClock.uptimeMillis()
            scheduleAutoCapitalizationUpdate()
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
        scheduleAutoCapitalizationUpdate()
    }

    override fun onDelete() {
        val ic = currentInputConnection ?: return
        dismissCurrentClipSuggestion()
        spacePending = false
        val hasSelection = if (selStart >= 0 && selEnd >= 0) {
            selStart != selEnd || !ic.getSelectedText(0).isNullOrEmpty()
        } else {
            !ic.getSelectedText(0).isNullOrEmpty()
        }
        if (hasSelection) {
            engine.resetBuf()
            ic.commitText("", 1)
            if (selStart >= 0 && selEnd >= 0) {
                val anchor = minOf(selStart, selEnd)
                expectedSel = anchor
                selStart = anchor
                selEnd = anchor
                expectEcho(anchor)
            }
            scheduleAutoCapitalizationUpdate()
            return
        }
        val res = engine.backspace()
        if (res.consumed) {
            applyResult(res)
        } else {
            ic.deleteSurroundingText(1, 0)
            bumpExpected(-1)
        }
        scheduleAutoCapitalizationUpdate()
    }

    override fun onEnter() {
        dismissCurrentClipSuggestion()
        spacePending = false
        engine.resetBuf()
        if (!sendDefaultEditorAction(true)) {
            sendDownUpKeyEvents(KeyEvent.KEYCODE_ENTER)
        }
        scheduleAutoCapitalizationUpdate()
    }

    private val autoCapRunnable = Runnable { updateAutoCapitalization() }

    override fun onShiftChanged(shifted: Boolean, capsLock: Boolean) {
        mainHandler.removeCallbacks(autoCapRunnable)
        engine.shiftPressed = shifted
        engine.capsLockOn = capsLock
        engine.autoCapShifted = false
    }

    private fun scheduleAutoCapitalizationUpdate() {
        mainHandler.removeCallbacks(autoCapRunnable)
        mainHandler.post(autoCapRunnable)
    }

    private fun updateAutoCapitalization() {
        if (!::keyboard.isInitialized || keyboard.capsLock) return
        val info = currentInputEditorInfo
        val enabled = Prefs.autoCapitalize(this) && isAutoCapField(info)
        val ic = currentInputConnection
        val capsMode = ic?.getCursorCapsMode(TextUtils.CAP_MODE_SENTENCES) ?: 0
        val textBeforeCursor = ic?.getTextBeforeCursor(4, 0)?.toString().orEmpty()
        val atSentenceStart = textBeforeCursor.isEmpty() ||
            textBeforeCursor.endsWith('\n') ||
            textBeforeCursor.endsWith(". ") ||
            textBeforeCursor.endsWith("? ") ||
            textBeforeCursor.endsWith("! ") ||
            textBeforeCursor.endsWith("… ") ||
            TextUtils.getCapsMode(textBeforeCursor, textBeforeCursor.length, TextUtils.CAP_MODE_SENTENCES) != 0
        val shouldShift = enabled &&
            (capsMode and TextUtils.CAP_MODE_SENTENCES != 0 || atSentenceStart)
        keyboard.setAutoShifted(shouldShift)
        engine.autoCapShifted = shouldShift
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

    // ViewStyle chip styling setters are marked @RestrictTo in autofill 1.3.0
    // although Google's own inline-suggestion docs use them — pinned dep, safe.
    @SuppressLint("NewApi", "RestrictedApi") // override is only invoked on API 30+
    override fun onCreateInlineSuggestionsRequest(uiExtras: Bundle): InlineSuggestionsRequest? {
        val versions = UiVersions.getVersions(uiExtras)
        if (!versions.contains(UiVersions.INLINE_UI_VERSION_1)) return null
        val dm = resources.displayMetrics.density
        val t = KeyboardThemes.current(this)
        val chipDrawable = if (UiTheme.isDark(this)) {
            R.drawable.bkg_inline_suggestion_dark
        } else {
            R.drawable.bkg_inline_suggestion_light
        }
        val chipBackground = Icon.createWithResource(this, chipDrawable).setTint(t.text)
        val padH = (10 * dm).toInt()
        val chip = ViewStyle.Builder()
            .setBackground(chipBackground)
            .setPadding(padH, 0, padH, 0)
            .build()
        val singleChip = ViewStyle.Builder()
            .setBackgroundColor(Color.TRANSPARENT)
            .setPadding(0, 0, 0, 0)
            .build()
        val title = TextViewStyle.Builder()
            .setLayoutMargin((4 * dm).toInt(), 0, (4 * dm).toInt(), 0)
            .setTextColor(t.text)
            .setTextSize(TypedValue.COMPLEX_UNIT_SP, 14f)
            .build()
        val subtitle = TextViewStyle.Builder()
            .setTextColor(t.spaceText)
            .setTextSize(TypedValue.COMPLEX_UNIT_SP, 12f)
            .build()
        val icon = ImageViewStyle.Builder()
            .setTintList(ColorStateList.valueOf(t.funcText))
            .build()
        val styles = UiVersions.newStylesBuilder()
            .addStyle(
                InlineSuggestionUi.newStyleBuilder()
                    .setChipStyle(chip)
                    .setSingleIconChipStyle(singleChip)
                    .setTitleStyle(title)
                    .setSubtitleStyle(subtitle)
                    .setStartIconStyle(icon)
                    .setEndIconStyle(icon)
                    .build()
            )
            .build()
        // Let each provider choose its natural chip width. Constraining the
        // presentation itself can make some providers collapse their response
        // to one item; the horizontal strip handles any total width.
        val spec = InlinePresentationSpec.Builder(
            Size(0, 0), Size(Int.MAX_VALUE, Int.MAX_VALUE))
            .setStyle(styles)
            .build()
        val request = InlineSuggestionsRequest.Builder(listOf(spec))
            .setMaxSuggestionCount(InlineSuggestionsRequest.SUGGESTION_COUNT_UNLIMITED)
        return request.build()
    }

    @SuppressLint("NewApi")
    override fun onInlineSuggestionsResponse(response: InlineSuggestionsResponse): Boolean {
        val suggestions = response.inlineSuggestions
        if (suggestions.isEmpty()) {
            clearInlineSuggestions()
            renderSuggestionStrip()
        } else {
            val generation = inlineGeneration
            showInlineSuggestions(suggestions, generation)
        }
        return true
    }

    @SuppressLint("NewApi")
    private fun clearInlineSuggestions() {
        inlineGeneration++
        hasInlineSuggestions = false
        val scroll = suggestionScroll
        scroll?.scrollTo(0, 0)
        scroll?.removeAllViews()
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.R) {
            scrollableInlineViews.forEach { view ->
                view.surfaceControl?.let { surfaceControl ->
                    SurfaceControl.Transaction().reparent(surfaceControl, null).apply()
                }
            }
        }
        scrollableInlineViews.clear()
        pinnedSuggestionHost?.removeAllViews()
        scroll?.visibility = View.GONE
        suggestionBar?.visibility = View.GONE
        suggestionStrip = null
        renderedClipKey = null
        renderedHistoryOn = false
        renderedTheme = null
    }

    /** Show the suggestion bar only when some chip row is actually visible. */
    private fun syncSuggestionBarVisibility() {
        val hasContent = suggestionScroll?.visibility == View.VISIBLE ||
            (pinnedSuggestionHost?.childCount ?: 0) > 0
        suggestionBar?.visibility = if (hasContent) View.VISIBLE else View.GONE
    }

    private data class ClipCandidate(
        val key: String,
        val rawText: String?,
        val imageUri: android.net.Uri?,
        val imageMime: String?,
        val label: String,
        val stamp: Long,
    )

    private fun currentClipCandidate(): ClipCandidate? {
        if (!Prefs.clipboardSuggest(this)) return null
        if (isPasswordField(currentInputEditorInfo)) return null
        val cm = clipboardManager ?: getSystemService(ClipboardManager::class.java) ?: return null
        val clip = try { cm.primaryClip } catch (_: Exception) { null } ?: return null
        if (clip.itemCount <= 0) return null
        val desc = clip.description
        var imageUri: android.net.Uri? = null
        var imageMime: String? = null
        var rawText: String? = null
        for (i in 0 until clip.itemCount) {
            val item = clip.getItemAt(i) ?: continue
            val u = item.uri
            if (u != null && imageUri == null) {
                val mime = (0 until (desc?.mimeTypeCount ?: 0))
                    .mapNotNull { desc?.getMimeType(it) }
                    .firstOrNull { it.startsWith("image/") }
                    ?: try { contentResolver.getType(u) } catch (_: Exception) { null }
                if (mime != null && mime.startsWith("image/")) {
                    imageUri = u
                    imageMime = mime
                }
            }
            if (rawText == null) {
                val t = item.text?.toString()
                if (!t.isNullOrBlank()) rawText = t
            }
        }
        if (imageUri == null && rawText == null) return null
        val clipKey = if (imageUri != null) "img:$imageUri" else "txt:$rawText"
        val descTs = desc?.timestamp ?: 0L
        if (clipKey != lastKnownClipText) {
            lastKnownClipText = clipKey
            clipCopiedAtMs = if (descTs > 0L) maxOf(descTs, clipCopiedAtMs) else System.currentTimeMillis()
        }
        val effectiveTs = maxOf(descTs, clipCopiedAtMs)
        if (effectiveTs <= 0L || System.currentTimeMillis() - effectiveTs > 180_000L) return null
        if (effectiveTs == dismissedClipStamp && clipKey == dismissedClipText) return null
        if (imageUri != null) {
            return ClipCandidate(
                key = clipKey,
                rawText = null,
                imageUri = imageUri,
                imageMime = imageMime ?: "image/*",
                label = "Hình ảnh",
                stamp = effectiveTs,
            )
        }
        val isSensitive = desc?.extras?.getBoolean("android.content.extra.IS_SENSITIVE", false) == true
        val singleLine = rawText!!.trim().replace(whitespaceRegex, " ")
        val label = if (isSensitive) "••••••••" else singleLine
        return ClipCandidate(
            key = clipKey,
            rawText = rawText,
            imageUri = null,
            imageMime = null,
            label = label,
            stamp = effectiveTs,
        )
    }

    private fun dismissCurrentClipSuggestion() {
        val cand = activeClipCandidate ?: return
        activeClipCandidate = null
        dismissedClipStamp = cand.stamp
        dismissedClipText = cand.key
        renderSuggestionStrip()
    }

    private fun loadThumbnail(uri: android.net.Uri, sizePx: Int): android.graphics.Bitmap? = try {
        val bounds = android.graphics.BitmapFactory.Options().apply { inJustDecodeBounds = true }
        contentResolver.openInputStream(uri)?.use { android.graphics.BitmapFactory.decodeStream(it, null, bounds) }
        if (bounds.outWidth <= 0 || bounds.outHeight <= 0) null
        else {
            var sample = 1
            while (bounds.outWidth / sample > sizePx * 2 && bounds.outHeight / sample > sizePx * 2) {
                sample *= 2
            }
            val opts = android.graphics.BitmapFactory.Options().apply { inSampleSize = sample }
            contentResolver.openInputStream(uri)?.use { android.graphics.BitmapFactory.decodeStream(it, null, opts) }
        }
    } catch (_: Exception) { null }

    private fun pasteClipCandidate(cand: ClipCandidate) {
        val ic = currentInputConnection ?: return
        activeClipCandidate = null
        dismissedClipStamp = cand.stamp
        dismissedClipText = cand.key
        engine.resetBuf()
        spacePending = false
        if (cand.imageUri != null) {
            val committed = try {
                val clipDesc = android.content.ClipDescription("Hình ảnh", arrayOf(cand.imageMime ?: "image/*"))
                val contentInfo = android.view.inputmethod.InputContentInfo(cand.imageUri, clipDesc, null)
                ic.commitContent(contentInfo, android.view.inputmethod.InputConnection.INPUT_CONTENT_GRANT_READ_URI_PERMISSION, null)
            } catch (_: Exception) { false }
            if (!committed) {
                ic.performContextMenuAction(android.R.id.paste)
            }
        } else if (cand.rawText != null) {
            if ((selStart >= 0 && selEnd >= 0 && selStart != selEnd) || !ic.getSelectedText(0).isNullOrEmpty()) {
                if (selStart >= 0 && selEnd >= 0) {
                    val anchor = minOf(selStart, selEnd)
                    expectedSel = anchor
                    selStart = anchor
                    selEnd = anchor
                }
            }
            ic.commitText(cand.rawText, 1)
            bumpExpected(cand.rawText.length)
        }
        renderSuggestionStrip()
        scheduleAutoCapitalizationUpdate()
    }

    /** Paste chip for the freshest clip. In compact mode (inline-autofill
     *  suggestions are sharing the strip) it collapses to a small icon so the
     *  provider chips keep the space and remain horizontally scrollable. */
    private fun buildClipboardChip(cand: ClipCandidate, compact: Boolean): View {
        val dm = resources.displayMetrics.density
        fun dp(v: Float) = (v * dm).toInt()
        val t = keyboard.theme
        val pill = LinearLayout(this).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.CENTER_VERTICAL
            setPadding(dp(if (compact) 10f else 12f), dp(5f), dp(if (compact) 10f else 14f), dp(5f))
            minimumHeight = dp(34f)
            background = GradientDrawable().apply {
                shape = GradientDrawable.RECTANGLE
                cornerRadius = 18f * dm
                setColor(t.key)
                if (t.border ushr 24 != 0) setStroke(maxOf(1, dp(1f)), t.border)
            }
            contentDescription = "Dán: ${cand.label}"
            setOnClickListener { pasteClipCandidate(cand) }
        }
        if (compact) {
            pill.addView(clipboardIcon(t).apply {
                layoutParams = LinearLayout.LayoutParams(dp(16f), dp(16f))
            })
            return pill
        }
        val thumb = cand.imageUri?.let { loadThumbnail(it, dp(24f)) }
        if (thumb != null) {
            val thumbView = ImageView(this).apply {
                setImageBitmap(thumb)
                scaleType = ImageView.ScaleType.CENTER_CROP
                background = GradientDrawable().apply {
                    shape = GradientDrawable.RECTANGLE
                    cornerRadius = 6f * dm
                }
                clipToOutline = true
                layoutParams = LinearLayout.LayoutParams(dp(22f), dp(22f)).apply {
                    marginEnd = dp(8f)
                }
            }
            pill.addView(thumbView)
        } else {
            pill.addView(clipboardIcon(t).apply {
                layoutParams = LinearLayout.LayoutParams(dp(16f), dp(16f)).apply {
                    marginEnd = dp(7f)
                }
            })
        }
        val labelView = TextView(this).apply {
            text = cand.label
            textSize = 14f
            setTextColor(t.text)
            isSingleLine = true
            ellipsize = TextUtils.TruncateAt.END
            maxWidth = dp(220f)
        }
        pill.addView(labelView)
        return pill
    }

    private fun clipboardIcon(t: com.tuigo.KeyTheme): View {
        val dm = resources.displayMetrics.density
        return object : View(this) {
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
        }
    }

    /** Clock-with-history glyph for the history chip — distinct from the
     *  clipboard icon used by the paste chip when both sit side by side. */
    private fun historyIcon(t: com.tuigo.KeyTheme): View {
        val dm = resources.displayMetrics.density
        return object : View(this) {
            private val paint = Paint(Paint.ANTI_ALIAS_FLAG).apply {
                style = Paint.Style.STROKE
                strokeWidth = 1.6f * dm
                strokeCap = Paint.Cap.ROUND
                strokeJoin = Paint.Join.ROUND
                color = t.funcText
            }
            override fun onDraw(canvas: Canvas) {
                val w = width.toFloat()
                val h = height.toFloat()
                val cx = w * 0.5f
                val cy = h * 0.52f
                val r = w * 0.34f
                canvas.drawCircle(cx, cy, r, paint)
                canvas.drawLine(cx, cy - r * 0.55f, cx, cy, paint)
                canvas.drawLine(cx, cy, cx + r * 0.5f, cy + r * 0.3f, paint)
            }
        }
    }

    /** Leading strip chip opening the persisted clipboard-history panel. */
    private fun buildHistoryChip(): View {
        val dm = resources.displayMetrics.density
        fun dp(v: Float) = (v * dm).toInt()
        val t = keyboard.theme
        return LinearLayout(this).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.CENTER_VERTICAL
            setPadding(dp(12f), dp(5f), dp(12f), dp(5f))
            minimumHeight = dp(34f)
            background = GradientDrawable().apply {
                shape = GradientDrawable.RECTANGLE
                cornerRadius = 18f * dm
                setColor(t.key)
                if (t.border ushr 24 != 0) setStroke(maxOf(1, dp(1f)), t.border)
            }
            contentDescription = "Lịch sử bộ nhớ tạm"
            addView(historyIcon(t).apply {
                layoutParams = LinearLayout.LayoutParams(dp(16f), dp(16f))
            })
            setOnClickListener { showClipboardPanel() }
        }
    }

    private var clipPanel: PopupWindow? = null

    /** Samsung-style clipboard panel: tap an entry to paste it; per-row buttons
     *  pin it (survives the history cap) or delete it. */
    private fun showClipboardPanel() {
        val entries = ClipboardHistory.entries(this)
        val t = keyboard.theme
        val dm = resources.displayMetrics.density
        fun dp(v: Float) = (v * dm).toInt()
        val column = LinearLayout(this).apply {
            orientation = LinearLayout.VERTICAL
            setPadding(dp(8f), dp(8f), dp(8f), dp(8f))
        }
        val header = LinearLayout(this).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.CENTER_VERTICAL
        }
        header.addView(TextView(this).apply {
            text = "Bộ nhớ tạm"
            textSize = 14f
            typeface = Typeface.DEFAULT_BOLD
            setTextColor(t.popupText)
            layoutParams = LinearLayout.LayoutParams(0, -2, 1f)
        })
        if (entries.isNotEmpty()) {
            header.addView(TextView(this).apply {
                text = "Xóa hết"
                textSize = 13f
                setTextColor(t.accent)
                setPadding(dp(12f), dp(4f), dp(4f), dp(4f))
                setOnClickListener {
                    ClipboardHistory.clear(this@TuiGoImeService)
                    clipPanel?.dismiss()
                    renderSuggestionStrip()
                }
            })
        }
        column.addView(header)
        if (entries.isEmpty()) {
            column.addView(TextView(this).apply {
                text = "Chưa có gì — mục đã sao chép sẽ hiện ở đây (tối đa ${Prefs.clipboardHistoryMax(this@TuiGoImeService)})"
                textSize = 13f
                setTextColor(t.popupText)
                alpha = 0.7f
                setPadding(0, dp(10f), 0, dp(4f))
            })
        }
        for (entry in entries) {
            column.addView(buildClipRow(entry, t, dp = { v -> dp(v) }))
        }
        val scroller = ScrollView(this).apply {
            addView(column)
            isVerticalScrollBarEnabled = entries.size > 5
        }
        val maxH = dp(320f)
        val popup = PopupWindow(scroller,
            (keyboard.width - dp(24f)).coerceAtLeast(dp(200f)),
            ViewGroup.LayoutParams.WRAP_CONTENT, true).apply {
            elevation = dp(10f).toFloat()
            isOutsideTouchable = true
            setBackgroundDrawable(GradientDrawable().apply {
                setColor(t.popup)
                cornerRadius = dp(12f).toFloat()
                if (t.popupBorder ushr 24 != 0) setStroke(dp(1f), t.popupBorder)
            })
        }
        // Cap panel height at ~5 rows by measuring lazily once attached.
        scroller.viewTreeObserver.addOnGlobalLayoutListener(object :
            android.view.ViewTreeObserver.OnGlobalLayoutListener {
            override fun onGlobalLayout() {
                if (scroller.height > maxH) {
                    scroller.layoutParams = scroller.layoutParams.apply { height = maxH }
                    popup.update((keyboard.width - dp(24f)).coerceAtLeast(dp(200f)), -2)
                }
                scroller.viewTreeObserver.removeOnGlobalLayoutListener(this)
            }
        })
        clipPanel?.dismiss()
        clipPanel = popup
        popup.showAtLocation(keyboard, Gravity.BOTTOM or Gravity.CENTER_HORIZONTAL,
            0, keyboard.height + dp(64f))
    }

    private fun buildClipRow(
        entry: ClipboardHistory.Entry,
        t: com.tuigo.KeyTheme,
        dp: (Float) -> Int,
    ): View {
        val row = LinearLayout(this).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.CENTER_VERTICAL
            setPadding(dp(8f), dp(6f), dp(4f), dp(6f))
        }
        val thumb = entry.image?.let {
            BitmapFactory.decodeFile(ClipboardHistory.imageFile(this, it).absolutePath)
        }
        if (thumb != null) {
            row.addView(ImageView(this).apply {
                setImageBitmap(thumb)
                scaleType = ImageView.ScaleType.CENTER_CROP
                background = GradientDrawable().apply {
                    cornerRadius = 6f * resources.displayMetrics.density
                }
                clipToOutline = true
                layoutParams = LinearLayout.LayoutParams(dp(30f), dp(30f)).apply {
                    marginEnd = dp(10f)
                }
            })
        } else {
            row.addView(clipboardIcon(t).apply {
                layoutParams = LinearLayout.LayoutParams(dp(18f), dp(18f)).apply {
                    marginEnd = dp(10f)
                }
            })
        }
        row.addView(TextView(this).apply {
            text = entry.text?.trim()?.replace(whitespaceRegex, " ") ?: "Hình ảnh"
            textSize = 14f
            setTextColor(t.popupText)
            maxLines = 2
            ellipsize = TextUtils.TruncateAt.END
            layoutParams = LinearLayout.LayoutParams(0, -2, 1f)
        })
        fun action(label: String, color: Int, fn: () -> Unit) =
            TextView(this).apply {
                text = label
                textSize = 12f
                setTextColor(color)
                setPadding(dp(10f), dp(8f), dp(10f), dp(8f))
                setOnClickListener { fn() }
            }
        row.addView(action(if (entry.pinned) "Bỏ ghim" else "Ghim", t.accent) {
            ClipboardHistory.setPinned(this, entry.id, !entry.pinned)
            clipPanel?.dismiss()
            showClipboardPanel()
        })
        row.addView(action("✕", t.popupText) {
            ClipboardHistory.remove(this, entry.id)
            clipPanel?.dismiss()
            showClipboardPanel()
            renderSuggestionStrip()
        })
        row.setOnClickListener { pasteClipEntry(entry) }
        return row
    }

    private fun pasteClipEntry(entry: ClipboardHistory.Entry) {
        clipPanel?.dismiss()
        engine.resetBuf()
        spacePending = false
        val ic = currentInputConnection ?: return
        if (entry.image != null) {
            val file = ClipboardHistory.imageFile(this, entry.image)
            val committed = try {
                val uri = androidx.core.content.FileProvider.getUriForFile(
                    this, "$packageName.clips", file)
                val desc = android.content.ClipDescription("Hình ảnh", arrayOf("image/png"))
                val info = android.view.inputmethod.InputContentInfo(uri, desc, null)
                ic.commitContent(info,
                    android.view.inputmethod.InputConnection.INPUT_CONTENT_GRANT_READ_URI_PERMISSION, null)
            } catch (_: Exception) { false }
            if (!committed) {
                // Host rejects rich content — put it back on the clipboard so the
                // user can paste via the app instead.
                try {
                    val uri = androidx.core.content.FileProvider.getUriForFile(
                        this, "$packageName.clips", file)
                    replayedClipUri = uri.toString()
                    clipboardManager?.setPrimaryClip(
                        android.content.ClipData.newUri(contentResolver, "Hình ảnh", uri))
                } catch (_: Exception) { }
            }
            renderSuggestionStrip()
            return
        }
        val text = entry.text ?: return
        if (selStart >= 0 && selEnd >= 0 && selStart != selEnd) {
            val anchor = minOf(selStart, selEnd)
            expectedSel = anchor
            selStart = anchor
            selEnd = anchor
        }
        ic.commitText(text, 1)
        bumpExpected(text.length)
        renderSuggestionStrip()
        scheduleAutoCapitalizationUpdate()
    }

    private val inlineExecutor = Executor { it.run() }

    @SuppressLint("NewApi")
    private fun showInlineSuggestions(
        suggestions: List<InlineSuggestion>,
        generation: Int,
    ) {
        if (generation != inlineGeneration) return
        hasInlineSuggestions = true

        if (suggestions.isEmpty()) {
            clearInlineSuggestions()
            renderSuggestionStrip()
            return
        }

        var pinned: InlineSuggestion? = null
        val scrollable = mutableListOf<InlineSuggestion>()
        var extraPinnedCount = 0
        suggestions.forEach {
            if (it.info.isPinned) {
                if (pinned == null) {
                    pinned = it
                } else {
                    scrollable.add(extraPinnedCount++, it)
                }
            } else {
                scrollable.add(it)
            }
        }
        if (pinned != null) {
            inflateInlineSuggestion(pinned) { view ->
                mainHandler.post {
                    if (generation != inlineGeneration) return@post
                    setPinnedInlineView(view, generation)
                }
            }
        } else {
            setPinnedInlineView(null, generation)
        }

        if (scrollable.isEmpty()) {
            setScrollableInlineViews(emptyList(), generation)
            return
        }

        val count = scrollable.size
        val views = MutableList<InlineContentView?>(count) { null }
        var completed = 0

        scrollable.forEachIndexed { index, suggestion ->
            inflateInlineSuggestion(suggestion) { view ->
                mainHandler.post {
                    if (generation != inlineGeneration) return@post
                    views[index] = view
                    completed++
                    if (completed == count) {
                        setScrollableInlineViews(views, generation)
                    }
                }
            }
        }
    }

    @SuppressLint("NewApi")
    private fun inflateInlineSuggestion(
        suggestion: InlineSuggestion,
        onInflated: (InlineContentView?) -> Unit,
    ) {
        try {
            suggestion.inflate(
                this,
                Size(
                    ViewGroup.LayoutParams.WRAP_CONTENT,
                    (40 * resources.displayMetrics.density).toInt(),
                ),
                inlineExecutor,
                onInflated,
            )
        } catch (_: Exception) {
            onInflated(null)
        }
    }

    @SuppressLint("NewApi")
    private fun setScrollableInlineViews(
        views: List<InlineContentView?>,
        generation: Int,
    ) {
        if (generation != inlineGeneration) return
        val scroll = suggestionScroll ?: return
        val surface = suggestionSurface ?: return

        scroll.scrollTo(0, 0)
        scroll.removeAllViews()
        scrollableInlineViews.forEach { view ->
            view.surfaceControl?.let { surfaceControl ->
                SurfaceControl.Transaction().reparent(surfaceControl, null).apply()
            }
        }
        scrollableInlineViews.clear()

        val flexbox = FlexboxLayout(this).apply {
            flexWrap = FlexWrap.NOWRAP
            justifyContent = JustifyContent.CENTER
            alignItems = com.google.android.flexbox.AlignItems.CENTER
        }
        if (Prefs.clipboardHistory(this) && ClipboardHistory.hasEntries(this)) {
            val historyChip = buildHistoryChip()
            historyChip.layoutParams = FlexboxLayout.LayoutParams(
                ViewGroup.LayoutParams.WRAP_CONTENT,
                ViewGroup.LayoutParams.WRAP_CONTENT,
            ).apply {
                flexShrink = 0f
                marginEnd = (6 * resources.displayMetrics.density).toInt()
            }
            flexbox.addView(historyChip)
        }
        views.forEach { view ->
            if (view == null) return@forEach
            scrollableInlineViews += view
            view.setSurfaceControlCallback(object : InlineContentView.SurfaceControlCallback {
                override fun onCreated(surfaceControl: SurfaceControl) {
                    val parent = surface.surfaceControl ?: return
                    SurfaceControl.Transaction()
                        .reparent(surfaceControl, parent)
                        .apply()
                }

                override fun onDestroyed(surfaceControl: SurfaceControl) = Unit
            })
            flexbox.addView(view)
            view.layoutParams = (view.layoutParams as? FlexboxLayout.LayoutParams
                ?: FlexboxLayout.LayoutParams(
                    ViewGroup.LayoutParams.WRAP_CONTENT,
                    ViewGroup.LayoutParams.MATCH_PARENT,
                )).apply { flexShrink = 0f }
        }
        suggestionStrip = flexbox
        scroll.addView(flexbox, ViewGroup.LayoutParams(
            ViewGroup.LayoutParams.WRAP_CONTENT,
            ViewGroup.LayoutParams.MATCH_PARENT,
        ))
        activeClipCandidate = null
        scroll.visibility = View.VISIBLE
        syncSuggestionBarVisibility()
    }

    @SuppressLint("NewApi")
    private fun setPinnedInlineView(view: InlineContentView?, generation: Int) {
        if (generation != inlineGeneration) return
        val host = pinnedSuggestionHost ?: return
        host.removeAllViews()
        if (view != null) {
            host.addView(view)
            view.layoutParams = (view.layoutParams as? android.widget.FrameLayout.LayoutParams
                ?: android.widget.FrameLayout.LayoutParams(
                    ViewGroup.LayoutParams.WRAP_CONTENT,
                    ViewGroup.LayoutParams.MATCH_PARENT,
                )).apply {
                val margin = (10 * resources.displayMetrics.density).toInt()
                leftMargin = margin
                rightMargin = margin
            }
        }
        syncSuggestionBarVisibility()
    }

    @SuppressLint("NewApi")
    private fun renderSuggestionStrip() {
        val scroll = suggestionScroll ?: return
        if (hasInlineSuggestions) return
        val clipCand = currentClipCandidate()
        activeClipCandidate = clipCand
        val showHistory = Prefs.clipboardHistory(this) && ClipboardHistory.hasEntries(this)
        if (clipCand == null && !showHistory) {
            renderedClipKey = null
            renderedHistoryOn = false
            suggestionStrip?.removeAllViews()
            pinnedSuggestionHost?.removeAllViews()
            scroll.visibility = View.GONE
            syncSuggestionBarVisibility()
            return
        }
        val clipKey = clipCand?.key
        val currentStrip = suggestionStrip
        if (clipKey == renderedClipKey && showHistory == renderedHistoryOn &&
                keyboard.theme == renderedTheme && (currentStrip?.childCount ?: 0) > 0) {
            scroll.visibility = View.VISIBLE
            syncSuggestionBarVisibility()
            return
        }
        renderedClipKey = clipKey
        renderedHistoryOn = showHistory
        renderedTheme = keyboard.theme
        scroll.setBackgroundColor(keyboard.theme.bg)
        val strip = FlexboxLayout(this).apply {
            flexWrap = FlexWrap.NOWRAP
            justifyContent = JustifyContent.FLEX_START
            alignItems = com.google.android.flexbox.AlignItems.CENTER
            val padding = (6 * resources.displayMetrics.density).toInt()
            setPadding(padding, padding, padding, (2 * resources.displayMetrics.density).toInt())
        }
        scroll.removeAllViews()
        scroll.addView(strip, ViewGroup.LayoutParams(
            ViewGroup.LayoutParams.WRAP_CONTENT,
            ViewGroup.LayoutParams.MATCH_PARENT,
        ))
        suggestionStrip = strip
        val gap = (6 * resources.displayMetrics.density).toInt()
        fun lp(view: View, moreAfter: Boolean) = FlexboxLayout.LayoutParams(
            view.layoutParams?.width?.takeIf { it > 0 } ?: ViewGroup.LayoutParams.WRAP_CONTENT,
            view.layoutParams?.height?.takeIf { it > 0 } ?: ViewGroup.LayoutParams.WRAP_CONTENT,
        ).apply {
            flexShrink = 0f
            if (moreAfter) marginEnd = gap
        }
        val chips = mutableListOf<View>()
        if (showHistory) chips += buildHistoryChip()
        if (clipCand != null) chips += buildClipboardChip(clipCand, compact = false)
        chips.forEachIndexed { i, chip ->
            (chip.parent as? ViewGroup)?.removeView(chip)
            chip.layoutParams = lp(chip, i < chips.lastIndex)
            strip.addView(chip)
        }
        pinnedSuggestionHost?.removeAllViews()
        scroll.visibility = View.VISIBLE
        syncSuggestionBarVisibility()
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
            selStart = expectedSel
            selEnd = expectedSel
            expectEcho(expectedSel)
        }
    }

    private fun bumpExpected(delta: Int) {
        if (expectedSel >= 0) {
            expectEcho(expectedSel)
            expectedSel += delta
            selStart = expectedSel
            selEnd = expectedSel
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

    private fun isAutoCapField(info: EditorInfo?): Boolean {
        if (isRestrictedField(info)) return false
        val inputType = info?.inputType ?: return false
        val klass = inputType and InputType.TYPE_MASK_CLASS
        if (klass != InputType.TYPE_CLASS_TEXT) return false
        val variation = inputType and InputType.TYPE_MASK_VARIATION
        return variation != InputType.TYPE_TEXT_VARIATION_URI &&
            variation != InputType.TYPE_TEXT_VARIATION_EMAIL_ADDRESS &&
            variation != InputType.TYPE_TEXT_VARIATION_WEB_EMAIL_ADDRESS
    }
}
