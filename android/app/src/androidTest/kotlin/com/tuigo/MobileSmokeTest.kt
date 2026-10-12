package com.tuigo

import android.app.Activity
import android.app.Instrumentation
import android.content.Intent
import android.os.Bundle
import android.widget.*
import android.view.ViewGroup
import com.tuigo.engine.*
import java.io.File

/** Runs on a device, exercises the real views/preferences/storage without a UI test framework. */
class MobileSmokeTest : Instrumentation() {
    private fun onUi(block: () -> Unit) {
        var failure: Throwable? = null
        runOnMainSync { try { block() } catch (error: Throwable) { failure = error } }
        failure?.let { throw it }
    }
    override fun onCreate(arguments: Bundle?) { super.onCreate(arguments); start() }
    override fun onStart() {
        val output = Bundle()
        val prefs = Prefs.get(targetContext)
        val originalPrefs = prefs.all
        val macroFile = File(targetContext.filesDir, "macro.txt")
        val originalMacro = macroFile.takeIf { it.exists() }?.readBytes()
        val activities = mutableListOf<Activity>()
        var resultCode = Activity.RESULT_CANCELED
        try {
            // Verify recovery of a legacy interrupted AtomicFile write.
            val backup = File(macroFile.path + ".bak")
            macroFile.delete()
            backup.writeText("ko:khôi phục")
            check(MacroFiles.read(targetContext) == "ko:khôi phục")
            prefs.edit().putInt(Prefs.KEY_IM, ImTelexSimple).commit()
            fun launch(type: Class<out Activity>) = startActivitySync(Intent(targetContext, type).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)).also { activities.add(it) }
            val activity = launch(MainActivity::class.java)
            onUi {
                val group = activity.window.decorView.findViewWithTag<RadioGroup>("input-methods")
                check(group.childCount == 3)
                check((group.getChildAt(0) as RadioButton).isChecked)
                check(prefs.getInt(Prefs.KEY_IM, -1) == ImTelex) { "legacy mode was not migrated" }
                val methods = listOf(ImTelex, ImVni, ImViqr)
                for ((i, method) in methods.withIndex()) {
                    (group.getChildAt(i) as RadioButton).performClick()
                    check((0 until group.childCount).count { (group.getChildAt(it) as RadioButton).isChecked } == 1)
                    check(prefs.getInt(Prefs.KEY_IM, -1) == method)
                    val engine = Engine(); Prefs.applyTo(engine, targetContext)
                    val typed = when (method) { ImVni -> "on63 "; ImViqr -> "o^n? "; else -> "oonr " }
                    var document = ""
                    for (char in typed) {
                        val result = engine.filter(char.code)
                        document = if (result.consumed) document.dropLast(result.backs) + String(result.out) else document + char
                    }
                    check(document == "ổn ") { "method $method produced $document" }
                }
            }
            onUi { activity.finish() }; waitForIdleSync()
            val reopened = launch(MainActivity::class.java)
            onUi {
                val group = reopened.window.decorView.findViewWithTag<RadioGroup>("input-methods")
                check((group.getChildAt(2) as RadioButton).isChecked)
                check((0 until group.childCount).count { (group.getChildAt(it) as RadioButton).isChecked } == 1)
            }
            val macro = launch(MacroActivity::class.java)
            onUi {
                val editor = macro.window.decorView.findViewWithTag<EditText>("macro-editor")
                fun saveButton(v: android.view.View): Button? {
                    if (v is Button && v.text.toString() == "Lưu gõ tắt") return v
                    if (v is ViewGroup) for (i in 0 until v.childCount) saveButton(v.getChildAt(i))?.let { return it }
                    return null
                }
                val save = checkNotNull(saveButton(macro.window.decorView))
                editor.setText("ko:không 😀"); save.performClick()
                check(macroFile.readText() == "ko:không 😀")
                editor.setText("ko=invalid"); save.performClick()
                check(macroFile.readText() == "ko:không 😀") { "invalid draft overwrote saved data" }
                editor.setText(""); save.performClick()
                check(macroFile.readText().isEmpty()) { "deletion was not saved" }
                // Exercise a real filesystem failure, not a mocked success.
                val blocked = File(macroFile.path + ".new")
                check(blocked.mkdir())
                File(blocked, "block-write").writeText("fixture")
                editor.setText("ko:không"); save.performClick()
                val feedback = macro.window.decorView.findViewWithTag<TextView>("macro-feedback")
                check(feedback.text.toString().startsWith("Không xác nhận")) { "write failure reported success" }
                check(macroFile.readText().isEmpty()) { "failed write replaced existing file" }
                File(blocked, "block-write").delete()
                blocked.delete()
            }
            output.putString("stream", "PASS: 3 exclusive selections (Telex, VNI, VIQR); persisted selection; engine applies methods; macro UTF-8 save, invalid draft rejection, deletion, backup recovery, write failure.\n")
            resultCode = Activity.RESULT_OK
        } catch (failure: Throwable) {
            output.putString("stream", "FAIL: ${failure.stackTraceToString()}\n")
        } finally {
            onUi { activities.forEach { it.finish() } }
            val blocked = File(macroFile.path + ".new")
            File(blocked, "block-write").delete()
            blocked.delete()
            if (originalMacro != null) macroFile.writeBytes(originalMacro) else macroFile.delete()
            val edit = prefs.edit().clear()
            for ((key, value) in originalPrefs) when (value) {
                is Boolean -> edit.putBoolean(key, value)
                is Int -> edit.putInt(key, value)
                is String -> edit.putString(key, value)
                is Float -> edit.putFloat(key, value)
                is Long -> edit.putLong(key, value)
            }
            edit.commit()
        }
        finish(resultCode, output)
    }
}
