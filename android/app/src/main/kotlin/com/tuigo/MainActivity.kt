package com.tuigo

import android.app.Activity
import android.content.Intent
import android.graphics.Color
import android.graphics.Typeface
import android.graphics.drawable.GradientDrawable
import android.os.Bundle
import android.provider.Settings
import android.view.Gravity
import android.view.View
import android.view.inputmethod.InputMethodManager
import android.widget.Button
import android.widget.CheckBox
import android.widget.LinearLayout
import android.widget.RadioButton
import android.widget.RadioGroup
import android.widget.ScrollView
import android.widget.SeekBar
import android.widget.TextView
import com.tuigo.engine.ImTelex
import com.tuigo.engine.ImTelexSimple
import com.tuigo.engine.ImTelexVni
import com.tuigo.engine.ImVni

/**
 * Host app = settings screen. Darkroom tokens, programmatic layout
 * (no AppCompat dependency for a single screen).
 */
class MainActivity : Activity() {

    private val dp get() = resources.displayMetrics.density
    private fun Int.dp() = (this * dp).toInt()
    private fun Float.dp() = (this * dp).toInt()

    private lateinit var prefsEditor: android.content.SharedPreferences.Editor
    private lateinit var prefs: android.content.SharedPreferences
    private lateinit var pal: Palette

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        pal = UiTheme.palette(this)
        prefs = Prefs.get(this)
        prefsEditor = prefs.edit()
        setContentView(buildContent())
    }

    private fun buildContent(): View {
        val root = LinearLayout(this).apply {
            orientation = LinearLayout.VERTICAL
            setPadding(24.dp(), 40.dp(), 24.dp(), 32.dp())
            setBackgroundColor(pal.bg)
        }

        root += title("Tui Gõ", 28f, Typeface.BOLD)
        root += TextView(this).apply {
            text = "Bộ gõ tiếng Việt — Telex / VNI"
            setTextColor(pal.muted); textSize = 14f
            setPadding(0, 6.dp(), 0, 28.dp())
        }

        // Step card: enable + switch
        root += card().apply {
            this += label("BẬT BÀN PHÍM")
            this += body("1. Bật Tui Gõ trong cài đặt hệ thống.")
            this += button("Mở cài đặt bàn phím", primary = true) {
                startActivity(Intent(Settings.ACTION_INPUT_METHOD_SETTINGS))
            }
            this += body("2. Chọn Tui Gõ làm bàn phím đang dùng.")
            this += button("Chọn bàn phím", primary = false) {
                getSystemService(InputMethodManager::class.java)
                    ?.showInputMethodPicker()
            }
        }

        // Input method card
        root += card().apply {
            this += label("KIỂU GÕ")
            val methods = listOf(
                ImTelex to "Telex",
                ImVni to "VNI",
                ImTelexSimple to "Telex đơn giản",
                ImTelexVni to "Telex + VNI",
            )
            val group = RadioGroup(this@MainActivity).apply {
                orientation = RadioGroup.VERTICAL
            }
            val current = prefs.getInt(Prefs.KEY_IM, ImTelex)
            for ((im, name) in methods) {
                group += RadioButton(this@MainActivity).apply {
                    text = name; setTextColor(pal.text); textSize = 15f
                    isChecked = im == current
                    setOnCheckedChangeListener { _, checked ->
                        if (checked) prefsEditor.putInt(Prefs.KEY_IM, im).apply()
                    }
                }
            }
            this += group
        }

        // Options card
        root += card().apply {
            this += label("TUỲ CHỌN")
            this += check("Bỏ dấu tự do (free marking)", Prefs.KEY_FREE_MARKING, true)
            this += check("Kiểu dấu mới (oà → oà, uỳ)", Prefs.KEY_MODERN_STYLE, true)
            this += check("Kiểm tra chính tả", Prefs.KEY_SPELL_CHECK, true)
            this += check("Gõ tắt (macro)", Prefs.KEY_MACRO, true)
            this += check("Khôi phục từ không phải tiếng Việt", Prefs.KEY_AUTO_NONVN, true)
            this += check("Gõ đúp Space → dấu chấm", Prefs.KEY_DOUBLE_SPACE, true)
            this += check("Hiện hàng số trên cùng", Prefs.KEY_NUMBER_ROW, false)
            this += seek("Chiều cao phím", Prefs.KEY_KEY_HEIGHT, 44, 68, 56)
            this += seek("Khoảng cách phím (ngang)", Prefs.KEY_KEY_GAP, 0, 8, 4)
        }

        root += TextView(this).apply {
            text = "Nút VN trên bàn phím bật/tắt chế độ tiếng Việt nhanh."
            setTextColor(pal.muted); textSize = 13f
            setPadding(0, 20.dp(), 0, 0)
        }

        return ScrollView(this).apply { addView(root) }
    }

    // ---- builders ------------------------------------------------------------

    private fun card(): LinearLayout = LinearLayout(this).apply {
        orientation = LinearLayout.VERTICAL
        setPadding(20.dp(), 18.dp(), 20.dp(), 20.dp())
        background = GradientDrawable().apply {
            setColor(pal.surface); cornerRadius = 16f.dp().toFloat()
            setStroke(1, pal.border)
        }
        val lp = LinearLayout.LayoutParams(
            LinearLayout.LayoutParams.MATCH_PARENT,
            LinearLayout.LayoutParams.WRAP_CONTENT)
        lp.bottomMargin = 16.dp()
        layoutParams = lp
    }

    private fun title(s: String, sp: Float, style: Int) = TextView(this).apply {
        text = s; setTextColor(pal.text); textSize = sp
        typeface = Typeface.create("sans-serif", style)
    }

    private fun label(s: String) = TextView(this).apply {
        text = s; setTextColor(pal.accent); textSize = 12f
        typeface = Typeface.create("sans-serif-medium", Typeface.BOLD)
        letterSpacing = 0.08f
        setPadding(0, 0, 0, 12.dp())
    }

    private fun body(s: String) = TextView(this).apply {
        text = s; setTextColor(pal.text); textSize = 14f
        setPadding(0, 0, 0, 10.dp())
    }

    private fun button(s: String, primary: Boolean, onClick: () -> Unit) =
        Button(this).apply {
            text = s; textSize = 14f; isAllCaps = false
            typeface = Typeface.create("sans-serif-medium", Typeface.BOLD)
            if (primary) {
                setTextColor(pal.onAccent)
                background = GradientDrawable().apply {
                    setColor(pal.accent); cornerRadius = 10f.dp().toFloat()
                }
            } else {
                setTextColor(pal.text)
                background = GradientDrawable().apply {
                    setColor(pal.raised); cornerRadius = 10f.dp().toFloat()
                    setStroke(1, pal.border)
                }
            }
            val lp = LinearLayout.LayoutParams(
                LinearLayout.LayoutParams.MATCH_PARENT,
                LinearLayout.LayoutParams.WRAP_CONTENT)
            lp.bottomMargin = 14.dp()
            layoutParams = lp
            setOnClickListener { onClick() }
        }

    private fun check(s: String, key: String, def: Boolean) = CheckBox(this).apply {
        text = s; setTextColor(pal.text); textSize = 15f
        isChecked = prefs.getBoolean(key, def)
        setOnCheckedChangeListener { _, c -> prefsEditor.putBoolean(key, c).apply() }
    }

    private fun seek(labelText: String, key: String, min: Int, max: Int, def: Int) =
        LinearLayout(this).apply {
            orientation = LinearLayout.VERTICAL
            val tv = TextView(this@MainActivity).apply {
                setTextColor(pal.text); textSize = 15f
            }
            val sb = SeekBar(this@MainActivity).apply {
                this.max = max - min
                progress = prefs.getInt(key, def) - min
            }
            fun render() { tv.text = "$labelText: ${min + sb.progress}dp" }
            render()
            sb.setOnSeekBarChangeListener(object : SeekBar.OnSeekBarChangeListener {
                override fun onProgressChanged(s: SeekBar?, p: Int, u: Boolean) {
                    render()
                    prefsEditor.putInt(key, min + p).apply()
                }
                override fun onStartTrackingTouch(s: SeekBar?) {}
                override fun onStopTrackingTouch(s: SeekBar?) {}
            })
            addView(tv); addView(sb)
        }

    private operator fun LinearLayout.plusAssign(v: View) { addView(v) }
    private operator fun RadioGroup.plusAssign(v: View) { addView(v) }
}
