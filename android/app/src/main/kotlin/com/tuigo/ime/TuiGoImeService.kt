package com.tuigo.ime

import android.inputmethodservice.InputMethodService
import android.os.SystemClock
import android.text.InputType
import android.view.KeyEvent
import android.view.View
import android.view.inputmethod.EditorInfo
import android.view.inputmethod.InputMethodManager
import com.tuigo.Prefs
import com.tuigo.engine.Engine
import com.tuigo.engine.ImTelex
import com.tuigo.engine.ImTelexVni
import com.tuigo.engine.ImVni
import com.tuigo.engine.Result
import java.io.File

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

    private lateinit var engine: Engine
    private lateinit var keyboard: KeyboardView

    private var vietOn = true
    private var doubleSpace = true

    /** Cursor position we expect after our own edits — lets us distinguish
     *  external selection changes (cursor tap) from our own commits. */
    private var expectedSel = -1

    /** Pending "double space -> ." state. */
    private var spaceAt = 0L
    private var spacePending = false

    override fun onCreate() {
        super.onCreate()
        engine = Engine()
        loadEngineConfig()
        // macro.txt next to app files (same convention as desktop macro.txt)
        File(filesDir, "macro.txt").takeIf { it.exists() }
            ?.let { engine.loadMacroFile(it.absolutePath) }
    }

    private fun loadEngineConfig() {
        Prefs.applyTo(engine, this)
        vietOn = Prefs.vietOn(this)
        doubleSpace = Prefs.doubleSpacePeriod(this)
    }

    /** VNI-style methods type tone marks with digits — they always get the
     *  number row; other methods follow the settings toggle. */
    private fun usesNumberRow(): Boolean =
        Prefs.numberRow(this) || when (Prefs.get(this).getInt(Prefs.KEY_IM, ImTelex)) {
            ImVni, ImTelexVni -> true
            else -> false
        }

    override fun onCreateInputView(): View {
        keyboard = KeyboardView(this)
        keyboard.listener = this
        keyboard.vietnameseOn = vietOn
        keyboard.showNumberRow = usesNumberRow()
        keyboard.setMetrics(Prefs.keyHeightDp(this), Prefs.keyGapDp(this))
        return keyboard
    }

    // Always show the soft keyboard even when a hardware keyboard is attached
    // (emulator, Chromebook) — same behavior as Gboard; the default impl hides
    // us whenever config.keyboard != NOKEYS.
    override fun onEvaluateInputViewShown(): Boolean = true

    override fun onStartInputView(info: EditorInfo?, restarting: Boolean) {
        super.onStartInputView(info, restarting)
        loadEngineConfig()
        keyboard.vietnameseOn = vietOn
        keyboard.showNumberRow = usesNumberRow()
        keyboard.setMetrics(Prefs.keyHeightDp(this), Prefs.keyGapDp(this))
        engine.resetBuf()
        expectedSel = info?.initialSelEnd ?: -1
        spacePending = false
    }

    override fun onFinishInputView(finishingInput: Boolean) {
        super.onFinishInputView(finishingInput)
        engine.resetBuf()
    }

    override fun onUpdateSelection(
        oldSelStart: Int, oldSelEnd: Int,
        newSelStart: Int, newSelEnd: Int,
        candidatesStart: Int, candidatesEnd: Int,
    ) {
        super.onUpdateSelection(oldSelStart, oldSelEnd, newSelStart, newSelEnd,
            candidatesStart, candidatesEnd)
        if (expectedSel != newSelStart) {
            // cursor was moved by the app/user, not by our commits
            engine.resetBuf()
            spacePending = false
        }
        expectedSel = newSelStart
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
            expectedSel += 1 // -1 + 2
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
        vietOn = !vietOn
        Prefs.get(this).edit().putBoolean(Prefs.KEY_VIET_ON, vietOn).apply()
        keyboard.vietnameseOn = vietOn
        engine.setVietKey(vietOn)
        engine.resetBuf()
    }

    override fun onSwitchIme() {
        getSystemService(InputMethodManager::class.java)?.showInputMethodPicker()
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
        if (expectedSel >= 0) expectedSel = expectedSel - res.backs + res.out.size
    }

    private fun bumpExpected(delta: Int) {
        if (expectedSel >= 0) expectedSel += delta
    }

    private fun isRestrictedField(info: EditorInfo?): Boolean {
        if (info == null) return true
        if (info.inputType == InputType.TYPE_NULL) return true
        val klass = info.inputType and InputType.TYPE_MASK_CLASS
        val variation = info.inputType and InputType.TYPE_MASK_VARIATION
        if (klass == InputType.TYPE_CLASS_PHONE) return true
        if (klass == InputType.TYPE_CLASS_NUMBER &&
            variation == InputType.TYPE_NUMBER_VARIATION_PASSWORD) return true
        return klass == InputType.TYPE_CLASS_TEXT &&
            (variation == InputType.TYPE_TEXT_VARIATION_PASSWORD ||
             variation == InputType.TYPE_TEXT_VARIATION_VISIBLE_PASSWORD ||
             variation == InputType.TYPE_TEXT_VARIATION_WEB_PASSWORD)
    }
}
