package com.tuigo

import android.app.Activity
import android.content.Intent
import android.content.res.ColorStateList
import android.os.Bundle
import android.provider.Settings
import android.view.View
import android.view.inputmethod.InputMethodManager
import android.widget.*
import com.tuigo.engine.*

class MainActivity : Activity() {
    private lateinit var ui: MobileUi
    private lateinit var status: TextView
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        ui = MobileUi(this)
        val p = Prefs.get(this)
        val root = ui.column()
        root.addView(ui.text("GÕ THEO CÁCH CỦA BẠN", 11f, ui.pal.accent, true))
        root.addView(ui.brandHeader("Tui Gõ", "Tiếng Việt, thật tự nhiên."))
        root.addView(ui.card("Bắt đầu", "Hai bước để dùng bàn phím của bạn").apply {
            status = ui.text("", 13f, ui.pal.accent, true)
            addView(status)
            addView(ui.button("1. Bật Tui Gõ trong cài đặt", true) {
                startActivity(Intent(Settings.ACTION_INPUT_METHOD_SETTINGS))
            })
            addView(ui.button("2. Chọn bàn phím Tui Gõ") {
                getSystemService(InputMethodManager::class.java)?.showInputMethodPicker()
            })
        })
        root.addView(ui.card("Kiểu gõ", "Chọn một kiểu phù hợp với thói quen của bạn").apply {
            val methods = listOf(
                Triple(ImTelex, "Telex", "s f r x j · aa, ee, oo, w, dd"),
                Triple(ImVni, "VNI", "Dùng số 1–9 để thêm dấu"),
                Triple(ImViqr, "VIQR", "Dùng ký hiệu ^, +, ', `, ?, ~, . để thêm dấu"))
            val group = RadioGroup(this@MainActivity).apply { tag = "input-methods" }
            val ids = mutableMapOf<Int, Int>()
            val current = Prefs.inputMethod(this@MainActivity)
            methods.forEach { (im, name, hint) ->
                val radio = RadioButton(this@MainActivity).apply {
                    id = View.generateViewId(); tag = "method-$im"
                    text = ui.twoLines(name, hint); textSize = 15f; setTextColor(ui.pal.text)
                    buttonTintList = ColorStateList.valueOf(ui.pal.accent)
                    setPadding(ui.dp(4), ui.dp(10), ui.dp(4), ui.dp(10)); minHeight = ui.dp(64)
                    layoutParams = RadioGroup.LayoutParams(-1, -2).apply { bottomMargin = ui.dp(6) }
                }
                ids[radio.id] = im
                group.addView(radio)
            }
            group.check(ids.entries.first { it.value == current }.key)
            fun renderSelection() {
                for (i in 0 until group.childCount) {
                    val radio = group.getChildAt(i) as RadioButton
                    radio.background = if (radio.isChecked) ui.surface(ui.pal.glow, 14).apply { setStroke(ui.dp(1), ui.pal.accent) } else null
                }
            }
            renderSelection()
            group.setOnCheckedChangeListener { _, id ->
                ids[id]?.let { p.edit().putInt(Prefs.KEY_IM, it).apply() }
                renderSelection()
            }
            addView(group)
        })
        root.addView(ui.card("Cách gõ", "Tinh chỉnh cách Tui Gõ xử lý từ và dấu").apply {
            addView(toggle("Mặc định gõ tiếng Việt", "App chưa dùng Tui Gõ sẽ mở ở chế độ này", Prefs.KEY_VIET_ON))
            addView(toggle("Bỏ dấu tự do", "Thêm dấu ngay khi đang gõ từ", Prefs.KEY_FREE_MARKING))
            addView(toggle("Đặt dấu kiểu mới", "Ví dụ: hoà, thuỷ", Prefs.KEY_MODERN_STYLE))
            addView(toggle("Kiểm tra chính tả", "Giữ cách gõ tiếng Việt hợp lệ", Prefs.KEY_SPELL_CHECK))
            addView(toggle("Khôi phục từ ngoại ngữ", "Giữ nguyên từ không phải tiếng Việt", Prefs.KEY_AUTO_NONVN))
            addView(toggle("Hai lần Space → dấu chấm", "Kết thúc câu nhanh", Prefs.KEY_DOUBLE_SPACE))
        })
        root.addView(ui.card("Gõ tắt", "Ít thao tác hơn, nhiều điều muốn nói hơn").apply {
            addView(toggle("Bật gõ tắt", "Gõ ko rồi Space → không", Prefs.KEY_MACRO))
            addView(ui.button("Quản lý từ gõ tắt") { startActivity(Intent(this@MainActivity, MacroActivity::class.java)) })
        })
        root.addView(ui.card("Bàn phím", "Không gian gõ vừa tay bạn").apply {
            addView(ui.text("Giao diện", 14f, ui.pal.text, true))
            addView(themePicker())
            addView(toggle("Hiện hàng số", "VNI luôn có hàng số để gõ dấu", Prefs.KEY_NUMBER_ROW, false))
            addView(slider("Chiều cao phím", Prefs.KEY_KEY_HEIGHT, 44, 68, 56))
            addView(slider("Khoảng cách phím", Prefs.KEY_KEY_GAP, 0, 8, 4))
            addView(ui.text("Giữ phím Space để đổi ngôn ngữ và kiểu gõ.", 13f, ui.pal.muted))
        })
        root.addView(ui.card("Thử bàn phím", "Gõ thử tiếng Việt hoặc từ gõ tắt ngay tại đây").apply {
            addView(ui.editor("Ví dụ: tieengs Vieetj", false).apply { tag = "typing-preview" })
        })
        root.addView(ui.text("Chỉ trên thiết bị của bạn. Không thu thập nội dung gõ.", 12f, ui.pal.muted))
        ui.install(this, root)
    }
    override fun onResume() {
        super.onResume()
        updateStatus()
    }
    override fun onWindowFocusChanged(hasFocus: Boolean) {
        super.onWindowFocusChanged(hasFocus)
        if (hasFocus) updateStatus()
    }
    private fun updateStatus() {
        if (!::status.isInitialized) return
        val manager = getSystemService(InputMethodManager::class.java)
        val enabled = manager?.enabledInputMethodList?.any { it.packageName == packageName } == true
        val selected = Settings.Secure.getString(contentResolver, Settings.Secure.DEFAULT_INPUT_METHOD)?.startsWith("$packageName/") == true
        status.text = when { selected -> "Sẵn sàng · Tui Gõ đang được chọn"; enabled -> "Đã bật · Chọn Tui Gõ để bắt đầu"; else -> "Chưa bật bàn phím" }
    }
    private fun toggle(title: String, hint: String, key: String, default: Boolean = true): View {
        val p = Prefs.get(this)
        return Switch(this).apply {
            text = ui.twoLines(title, hint); textSize = 14f; setTextColor(ui.pal.text)
            setPadding(0, ui.dp(12), 0, ui.dp(12)); minHeight = ui.dp(60)
            thumbTintList = ColorStateList(arrayOf(intArrayOf(android.R.attr.state_checked), intArrayOf()), intArrayOf(ui.pal.accent, ui.pal.muted))
            trackTintList = ColorStateList(arrayOf(intArrayOf(android.R.attr.state_checked), intArrayOf()), intArrayOf((ui.pal.accent and 0x00FFFFFF) or 0x33000000, ui.pal.border))
            isChecked = p.getBoolean(key, default)
            setOnCheckedChangeListener { _, checked -> p.edit().putBoolean(key, checked).apply() }
        }
    }
    /** Horizontal strip of skin chips with live color previews. */
    private fun themePicker(): View {
        val p = Prefs.get(this)
        val dark = UiTheme.isDark(this)
        val current = p.getString(Prefs.KEY_THEME, KeyboardThemes.DEFAULT_ID) ?: KeyboardThemes.DEFAULT_ID
        val row = LinearLayout(this).apply { orientation = LinearLayout.HORIZONTAL }
        val chips = mutableMapOf<String, View>()
        fun select(id: String) {
            p.edit().putString(Prefs.KEY_THEME, id).apply()
            for ((chipId, chip) in chips) {
                chip.background = if (chipId == id)
                    ui.surface(ui.pal.glow, 14).apply { setStroke(ui.dp(1), ui.pal.accent) }
                else ui.surface(ui.pal.raised, 14)
            }
        }
        KeyboardThemes.ids.forEach { id ->
            val t = KeyboardThemes.get(id, dark)
            val chip = LinearLayout(this).apply {
                orientation = LinearLayout.VERTICAL; gravity = android.view.Gravity.CENTER_HORIZONTAL
                setPadding(ui.dp(10), ui.dp(8), ui.dp(10), ui.dp(6))
                tag = "theme-$id"
                background = ui.surface(ui.pal.raised, 14)
                addView(ImageView(this@MainActivity).apply {
                    setImageDrawable(swatch(t))
                    layoutParams = LinearLayout.LayoutParams(ui.dp(48), ui.dp(18))
                })
                addView(TextView(this@MainActivity).apply {
                    text = KeyboardThemes.label(id); textSize = 11f; setTextColor(ui.pal.text)
                    setPadding(0, ui.dp(5), 0, 0)
                })
                setOnClickListener { select(id) }
                layoutParams = LinearLayout.LayoutParams(-2, -2).apply { rightMargin = ui.dp(8) }
            }
            chips[id] = chip
            row.addView(chip)
        }
        select(current)
        return HorizontalScrollView(this).apply {
            isHorizontalScrollBarEnabled = false
            tag = "keyboard-themes"
            addView(row)
        }
    }
    /** Horizontal strip [bg | key | accent] ~48x18dp used as theme preview. */
    private fun swatch(t: KeyTheme): android.graphics.drawable.Drawable {
        val d = ui.dp(4); val w = ui.dp(48); val h = ui.dp(18)
        return android.graphics.drawable.LayerDrawable(arrayOf(
            android.graphics.drawable.GradientDrawable().apply {
                shape = android.graphics.drawable.GradientDrawable.RECTANGLE
                cornerRadius = d * 2f; setColor(t.bg)
            },
            android.graphics.drawable.GradientDrawable().apply {
                shape = android.graphics.drawable.GradientDrawable.RECTANGLE
                cornerRadius = d * 2f; setColor(t.key); setStroke(1, t.border)
            },
            android.graphics.drawable.GradientDrawable().apply {
                shape = android.graphics.drawable.GradientDrawable.RECTANGLE
                cornerRadius = d * 2f; setColor(t.accent)
            },
        )).apply {
            setLayerInset(0, 0, 0, 0, 0)
            setLayerInset(1, w / 3, 0, w / 3, 0)
            setLayerInset(2, w * 2 / 3, 0, 0, 0)
            setBounds(0, 0, w, h)
        }
    }
    private fun slider(title: String, key: String, min: Int, max: Int, default: Int): View {
        val p = Prefs.get(this)
        return ui.column().apply {
            val label = ui.text("", 14f)
            val seek = SeekBar(this@MainActivity).apply {
                this.max = max - min; progress = p.getInt(key, default).coerceIn(min, max) - min
                progressTintList = ColorStateList.valueOf(ui.pal.accent); thumbTintList = ColorStateList.valueOf(ui.pal.accent)
            }
            label.text = "$title · ${seek.progress + min} dp"; seek.contentDescription = label.text
            seek.setOnSeekBarChangeListener(object : SeekBar.OnSeekBarChangeListener {
                override fun onProgressChanged(s: SeekBar?, value: Int, user: Boolean) {
                    label.text = "$title · ${value + min} dp"; seek.contentDescription = label.text
                    if (user) p.edit().putInt(key, value + min).apply()
                }
                override fun onStartTrackingTouch(s: SeekBar?) {}
                override fun onStopTrackingTouch(s: SeekBar?) {}
            })
            addView(label); addView(seek)
        }
    }
}
