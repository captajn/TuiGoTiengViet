package com.tuigo

import android.app.Activity
import android.app.AlertDialog
import android.annotation.SuppressLint
import android.os.Build
import android.os.Bundle
import android.text.Editable
import android.text.TextWatcher
import android.widget.EditText
import android.widget.TextView
import android.util.AtomicFile
import com.tuigo.engine.macroValidationError
import java.io.File

/** Edit the existing UTF-8 format, without silently dropping unknown lines. */
class MacroActivity : Activity() {
    private lateinit var editor: EditText
    private lateinit var feedback: TextView
    private var original = ""
    private var loadFailed = false
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        if (Build.VERSION.SDK_INT >= 33) {
            onBackInvokedDispatcher.registerOnBackInvokedCallback(android.window.OnBackInvokedDispatcher.PRIORITY_DEFAULT) { confirmBack() }
        }
        val ui = MobileUi(this)
        val root = ui.column()
        root.addView(ui.button("‹ Cài đặt") { confirmBack() })
        root.addView(ui.brandHeader("Gõ tắt", "Một từ ngắn, cả câu dài."))
        root.addView(ui.card("Cách dùng", "Bật gõ tắt trong Cài đặt và dùng chế độ Tiếng Việt.").apply {
            addView(ui.text("Thêm mỗi mục trên một dòng, ngăn bằng dấu hai chấm. Gõ từ tắt rồi chạm Space để thay thế.", 15f))
            addView(ui.text("ko:không\nvn:Việt Nam\nmail:hello@example.com", 15f, ui.pal.accent, true))
            addView(ui.text("Sửa nội dung để đổi một mục, xóa dòng để xóa. Từ tắt tối đa 15 ký tự; phân biệt chữ hoa/thường.", 13f, ui.pal.muted))
        })
        root.addView(ui.card("Từ của bạn", "Mỗi dòng: từ tắt:nội dung thay thế").apply {
            editor = ui.editor("ko:không\nvn:Việt Nam").apply {
                tag = "macro-editor"
                typeface = android.graphics.Typeface.MONOSPACE
                minLines = 6
                maxLines = 12
            }
            addView(editor)
            feedback = ui.text("", 13f, ui.pal.muted).apply { tag = "macro-feedback" }
            addView(feedback)
            addView(ui.button("Lưu gõ tắt", true) { save() })
        })
        ui.install(this, root)
        try {
            original = MacroFiles.read(this)
            editor.setText(savedInstanceState?.getString("draft") ?: original)
            feedback.text = "Dữ liệu chỉ được lưu trên thiết bị."
        } catch (e: Exception) {
            loadFailed = true
            feedback.text = "Không đọc được gõ tắt. Quay lại và thử mở lại; chưa ghi đè dữ liệu."
        }
        editor.addTextChangedListener(object : TextWatcher {
            override fun beforeTextChanged(s: CharSequence?, start: Int, count: Int, after: Int) {}
            override fun onTextChanged(s: CharSequence?, start: Int, before: Int, count: Int) {
                if (!loadFailed) feedback.text = macroValidationError(s.toString()) ?: "Chưa lưu · Chạm Lưu để áp dụng."
            }
            override fun afterTextChanged(s: Editable?) {}
        })
    }
    override fun onSaveInstanceState(outState: Bundle) {
        outState.putString("draft", editor.text.toString()); super.onSaveInstanceState(outState)
    }
    private fun save() {
        if (loadFailed) return
        val value = editor.text.toString()
        macroValidationError(value)?.let { feedback.text = it; return }
        val file = AtomicFile(File(filesDir, "macro.txt"))
        var stream: java.io.FileOutputStream? = null
        try {
            stream = file.startWrite()
            stream.write(value.toByteArray(Charsets.UTF_8))
            file.finishWrite(stream)
            stream = null
            // AtomicFile can log a failed rename instead of throwing; confirm it.
            if (MacroFiles.read(this) != value) throw java.io.IOException("Saved contents could not be confirmed")
            original = value
            feedback.text = "Đã lưu · Mở lại bàn phím để dùng danh sách mới."
        } catch (e: Exception) {
            file.failWrite(stream)
            feedback.text = "Không xác nhận được việc lưu. Hãy giữ bản nháp và thử lại."
        }
    }
    @Deprecated("Deprecated in Java")
    @SuppressLint("GestureBackNavigation") // API 33+ uses the dispatcher above; this is the API 26–32 fallback.
    override fun onBackPressed() { confirmBack() }
    private fun confirmBack() {
        if (editor.text.toString() == original) { finish(); return }
        AlertDialog.Builder(this).setTitle("Chưa lưu thay đổi")
            .setMessage("Quay lại sẽ bỏ nội dung vừa sửa.")
            .setNegativeButton("Sửa tiếp", null)
            .setPositiveButton("Bỏ thay đổi") { _, _ -> finish() }.show()
    }
}
