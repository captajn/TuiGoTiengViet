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
    private lateinit var step1Btn: View
    private lateinit var step2Btn: View
    private lateinit var macroSummary: TextView
    private var selectedTab = 0

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        selectedTab = savedInstanceState?.getInt("selected_tab", 0) ?: 0
        ui = MobileUi(this)
        val p = Prefs.get(this)
        val root = ui.column()
        root.addView(ui.text("GÕ THEO CÁCH CỦA BẠN", 11f, ui.pal.accent, true))
        root.addView(ui.brandHeader("Tui Gõ", "Tiếng Việt, thật tự nhiên."))

        // Tab 0: Bộ gõ
        val tabTyping = ui.column().apply {
            addView(ui.card("Bắt đầu", "Hai bước để dùng bàn phím của bạn").apply {
                status = ui.text("", 13.5f, ui.pal.accent, true)
                addView(status)
                step1Btn = ui.button("1. Bật Tui Gõ trong cài đặt", true) {
                    startActivity(Intent(Settings.ACTION_INPUT_METHOD_SETTINGS))
                }
                step2Btn = ui.button("2. Chọn bàn phím Tui Gõ") {
                    getSystemService(InputMethodManager::class.java)?.showInputMethodPicker()
                }
                addView(step1Btn)
                addView(step2Btn)
            })
            addView(ui.card("Kiểu gõ", "Chọn một kiểu phù hợp với thói quen của bạn").apply {
                val methods = listOf(
                    Triple(ImTelex, "Telex", "Telex · s, f, r, x, j để thêm dấu; aa, ee, oo, w, dd"),
                    Triple(ImVni, "VNI", "VNI · Dùng hàng phím số 1–9 để thêm dấu"),
                    Triple(ImViqr, "VIQR", "VIQR · Dùng ký hiệu ^, +, ', `, ?, ~, . để thêm dấu"))
                val group = RadioGroup(this@MainActivity).apply {
                    tag = "input-methods"
                    orientation = LinearLayout.HORIZONTAL
                    background = ui.surface(ui.pal.raised, 14)
                    setPadding(ui.dp(4), ui.dp(4), ui.dp(4), ui.dp(4))
                    layoutParams = ui.fullWidthLp(topMarginDp = 6)
                }
                val ids = mutableMapOf<Int, Int>()
                val hints = mutableMapOf<Int, String>()
                val current = Prefs.inputMethod(this@MainActivity)
                methods.forEach { (im, name, hint) ->
                    val radio = RadioButton(this@MainActivity).apply {
                        id = View.generateViewId(); tag = "method-$im"
                        text = name; textSize = 14.5f; isSingleLine = true
                        gravity = android.view.Gravity.CENTER
                        buttonDrawable = null // modern segmented pill
                        minHeight = ui.dp(44)
                        setPadding(ui.dp(6), ui.dp(10), ui.dp(6), ui.dp(10))
                        layoutParams = RadioGroup.LayoutParams(0, android.view.ViewGroup.LayoutParams.WRAP_CONTENT, 1f)
                    }
                    ids[radio.id] = im
                    hints[radio.id] = hint
                    group.addView(radio)
                }
                val hintText = ui.text("", 12.5f, ui.pal.muted).apply {
                    setPadding(ui.dp(2), ui.dp(10), ui.dp(2), ui.dp(4))
                }
                group.check(ids.entries.first { it.value == current }.key)
                fun renderSelection() {
                    for (i in 0 until group.childCount) {
                        val radio = group.getChildAt(i) as RadioButton
                        if (radio.isChecked) {
                            radio.background = ui.glassCard(active = true, 12)
                            radio.setTextColor(ui.pal.accent)
                            radio.typeface = android.graphics.Typeface.DEFAULT_BOLD
                            hintText.text = hints[radio.id] ?: ""
                        } else {
                            radio.background = null
                            radio.setTextColor(ui.pal.muted)
                            radio.typeface = android.graphics.Typeface.DEFAULT
                        }
                    }
                }
                renderSelection()
                Prefs.syncSystemSubtype(this@MainActivity)
                group.setOnCheckedChangeListener { _, id ->
                    ids[id]?.let {
                        p.edit().putInt(Prefs.KEY_IM, it).apply()
                        Prefs.syncSystemSubtype(this@MainActivity)
                    }
                    renderSelection()
                }
                addView(group)
                addView(hintText)
            })
            addView(ui.card("Cách gõ", "Tinh chỉnh cách Tui Gõ xử lý từ và dấu").apply {
                addView(toggle("Mặc định gõ tiếng Việt", "App chưa dùng Tui Gõ sẽ mở ở chế độ này", Prefs.KEY_VIET_ON))
                addView(toggle("Bỏ dấu tự do", "Thêm dấu ngay khi đang gõ từ", Prefs.KEY_FREE_MARKING))
                addView(toggle("Đặt dấu kiểu mới", "Ví dụ: hoà, thuỷ", Prefs.KEY_MODERN_STYLE))
                addView(toggle("Kiểm tra chính tả", "Giữ cách gõ tiếng Việt hợp lệ", Prefs.KEY_SPELL_CHECK))
                addView(toggle("Khôi phục từ ngoại ngữ", "Giữ nguyên từ không phải tiếng Việt", Prefs.KEY_AUTO_NONVN))
                addView(toggle("Viết hoa kí tự đầu tiên", "Tự bật Shift ở đầu câu", Prefs.KEY_AUTO_CAPITALIZE))
                addView(toggle("Hai lần Space → dấu chấm", "Kết thúc câu nhanh", Prefs.KEY_DOUBLE_SPACE))
            })
            addView(ui.card("Thử bàn phím", "Gõ thử tiếng Việt hoặc từ gõ tắt ngay tại đây").apply {
                addView(ui.editor("Ví dụ: tieengs Vieetj", false).apply { tag = "typing-preview" })
            })
        }

        // Tab 1: Giao diện
        val tabTheme = ui.column().apply {
            addView(ui.card("Bàn phím", "Không gian gõ vừa tay bạn").apply {
                addView(ui.text("Giao diện", 14f, ui.pal.text, true))
                addView(themePicker())
                addView(toggle("Hiện hàng số", "VNI luôn có hàng số để gõ dấu", Prefs.KEY_NUMBER_ROW, false))
                addView(toggle("Gợi ý dán từ bộ nhớ tạm", "Hiện chip dán nhanh trên bàn phím khi vừa sao chép", Prefs.KEY_CLIPBOARD_SUGGEST, true))
                addView(toggle("Lịch sử bộ nhớ tạm", "Lưu các mục đã sao chép trên thiết bị — mở từ biểu tượng đồng hồ trên thanh gợi ý", Prefs.KEY_CLIPBOARD_HISTORY, true))
                addView(slider("Số mục bộ nhớ tạm lưu tối đa", Prefs.KEY_CLIPBOARD_MAX, 10, 500, 50, "mục"))
                addView(slider("Chiều cao phím", Prefs.KEY_KEY_HEIGHT, 44, 68, 56))
                addView(slider("Khoảng cách phím", Prefs.KEY_KEY_GAP, 0, 8, 4))
                addView(ui.text("Vuốt phím Space để di chuyển con trỏ · Giữ Space để đổi ngôn ngữ và kiểu gõ.", 13f, ui.pal.muted))
            })
            addView(ui.card("Xem trước bàn phím", "Chạm vào ô bên dưới để thử giao diện và kích thước phím").apply {
                addView(ui.editor("Chạm để hiện bàn phím...", false).apply { tag = "typing-preview-theme" })
            })
        }

        // Tab 2: Gõ tắt
        val tabMacro = ui.column().apply {
            addView(ui.card("Gõ tắt", "Ít thao tác hơn, nhiều điều muốn nói hơn").apply {
                addView(toggle("Bật gõ tắt", "Gõ ko rồi Space → không", Prefs.KEY_MACRO))
                macroSummary = ui.text("", 13f, ui.pal.muted).apply {
                    setPadding(0, ui.dp(6), 0, ui.dp(6))
                }
                addView(macroSummary)
                addView(ui.button("Quản lý từ gõ tắt", true) {
                    startActivity(Intent(this@MainActivity, MacroActivity::class.java))
                })
            })
            addView(ui.card("Thử gõ tắt", "Gõ từ tắt rồi nhấn phím Space để kiểm tra").apply {
                addView(ui.editor("Ví dụ: ko + Space", false).apply { tag = "typing-preview-macro" })
            })
        }

        val tabs = listOf(
            MobileUi.TabSpec("typing", "Bộ gõ", MobileUi.ICON_KEYBOARD, tabTyping),
            MobileUi.TabSpec("theme", "Giao diện", MobileUi.ICON_THEME, tabTheme),
            MobileUi.TabSpec("macro", "Gõ tắt", MobileUi.ICON_MACRO, tabMacro),
        )
        tabs.forEach { root.addView(it.content) }
        root.addView(ui.text("Chỉ trên thiết bị của bạn. Không thu thập nội dung gõ.", 12f, ui.pal.muted).apply {
            layoutParams = ui.fullWidthLp(topMarginDp = 14)
        })
        ui.installWithBottomNav(this, root, tabs, selectedTab) { selectedTab = it }
    }

    override fun onSaveInstanceState(outState: Bundle) {
        outState.putInt("selected_tab", selectedTab)
        super.onSaveInstanceState(outState)
    }

    override fun onResume() {
        super.onResume()
        updateStatus()
        updateMacroSummary()
    }

    override fun onWindowFocusChanged(hasFocus: Boolean) {
        super.onWindowFocusChanged(hasFocus)
        if (hasFocus) updateStatus()
    }

    private fun updateMacroSummary() {
        if (!::macroSummary.isInitialized) return
        val lines = try {
            MacroFiles.read(this).lineSequence().map { it.trim() }.filter { it.isNotEmpty() && ':' in it }.toList()
        } catch (_: Exception) {
            emptyList()
        }
        macroSummary.text = if (lines.isEmpty()) {
            "Chưa có từ gõ tắt nào được lưu · Ví dụ: ko:không, vn:Việt Nam"
        } else {
            val preview = lines.take(3).joinToString(" · ") { it.replace(":", " → ") }
            "Đang lưu ${lines.size} mục · $preview"
        }
    }

    private fun updateStatus() {
        if (!::status.isInitialized) return
        val manager = getSystemService(InputMethodManager::class.java)
        val enabled = manager?.enabledInputMethodList?.any { it.packageName == packageName } == true
        val selected = Settings.Secure.getString(contentResolver, Settings.Secure.DEFAULT_INPUT_METHOD)?.startsWith("$packageName/") == true
        if (selected) {
            status.text = "✓ Sẵn sàng · Tui Gõ đang được chọn"
            if (::step1Btn.isInitialized) step1Btn.visibility = View.GONE
            if (::step2Btn.isInitialized) step2Btn.visibility = View.GONE
        } else {
            status.text = if (enabled) "Đã bật trong cài đặt · Chọn Tui Gõ để bắt đầu" else "Chưa bật bàn phím"
            if (::step1Btn.isInitialized) step1Btn.visibility = if (enabled) View.GONE else View.VISIBLE
            if (::step2Btn.isInitialized) step2Btn.visibility = View.VISIBLE
        }
    }
    private fun toggle(title: String, hint: String, key: String, default: Boolean = true): View {
        val p = Prefs.get(this)
        return Switch(this).apply {
            text = ui.twoLines(title, hint); textSize = 14f; setTextColor(ui.pal.text)
            setPadding(0, ui.dp(12), 0, ui.dp(12)); minHeight = ui.dp(60)
            switchPadding = ui.dp(12)
            layoutParams = ui.fullWidthLp()
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
                minimumWidth = ui.dp(92)
                setPadding(ui.dp(12), ui.dp(10), ui.dp(12), ui.dp(8))
                tag = "theme-$id"
                background = ui.surface(ui.pal.raised, 14)
                addView(ImageView(this@MainActivity).apply {
                    setImageDrawable(swatch(t))
                    layoutParams = LinearLayout.LayoutParams(ui.dp(72), ui.dp(32))
                })
                addView(TextView(this@MainActivity).apply {
                    text = KeyboardThemes.label(id); textSize = 12f; setTextColor(ui.pal.text)
                    isSingleLine = true
                    gravity = android.view.Gravity.CENTER
                    setPadding(0, ui.dp(6), 0, 0)
                    layoutParams = LinearLayout.LayoutParams(
                        android.view.ViewGroup.LayoutParams.WRAP_CONTENT,
                        android.view.ViewGroup.LayoutParams.WRAP_CONTENT,
                    )
                })
                setOnClickListener { select(id) }
                layoutParams = LinearLayout.LayoutParams(
                    android.view.ViewGroup.LayoutParams.WRAP_CONTENT,
                    android.view.ViewGroup.LayoutParams.WRAP_CONTENT,
                ).apply { rightMargin = ui.dp(8) }
            }
            chips[id] = chip
            row.addView(chip)
        }
        select(current)
        return HorizontalScrollView(this).apply {
            isHorizontalScrollBarEnabled = false
            isFillViewport = true
            tag = "keyboard-themes"
            layoutParams = ui.fullWidthLp(topMarginDp = 6).apply { bottomMargin = ui.dp(6) }
            addView(row, FrameLayout.LayoutParams(
                android.view.ViewGroup.LayoutParams.WRAP_CONTENT,
                android.view.ViewGroup.LayoutParams.WRAP_CONTENT,
            ))
        }
    }
    /** Mini keyboard preview used as theme card swatch. */
    private fun swatch(t: KeyTheme): android.graphics.drawable.Drawable {
        val w = ui.dp(72); val h = ui.dp(32); val r = ui.dp(6).toFloat()
        return object : android.graphics.drawable.Drawable() {
            private val p = android.graphics.Paint(android.graphics.Paint.ANTI_ALIAS_FLAG)
            override fun draw(canvas: android.graphics.Canvas) {
                p.color = t.bg; p.style = android.graphics.Paint.Style.FILL
                canvas.drawRoundRect(0f, 0f, w.toFloat(), h.toFloat(), r, r, p)
                p.color = t.key
                val pad = 3.5f
                val gap = 2f
                val kw = (w - pad * 2 - gap * 3) / 4f
                val kh = (h - pad * 2 - gap) / 2.2f
                for (i in 0..3) {
                    val kx = pad + i * (kw + gap)
                    canvas.drawRoundRect(kx, pad, kx + kw, pad + kh, 2.5f, 2.5f, p)
                }
                val spaceW = w * 0.54f; val sy = h - kh - pad
                p.color = t.space
                canvas.drawRoundRect(pad, sy, pad + spaceW, sy + kh, 2.5f, 2.5f, p)
                p.color = t.accent
                canvas.drawRoundRect(w - kw - pad, sy, w - pad, sy + kh, 2.5f, 2.5f, p)
            }
            override fun setAlpha(alpha: Int) { p.alpha = alpha }
            override fun setColorFilter(filter: android.graphics.ColorFilter?) { p.colorFilter = filter }
            @Suppress("DEPRECATION", "OVERRIDE_DEPRECATION")
            @Deprecated("Deprecated in Java")
            override fun getOpacity(): Int = android.graphics.PixelFormat.TRANSLUCENT
            override fun getIntrinsicWidth(): Int = w
            override fun getIntrinsicHeight(): Int = h
        }
    }
    private fun slider(title: String, key: String, min: Int, max: Int, default: Int, unit: String = "dp"): View {
        val p = Prefs.get(this)
        return ui.column().apply {
            layoutParams = ui.fullWidthLp(topMarginDp = 8)
            val label = ui.text("", 14f)
            val seek = SeekBar(this@MainActivity).apply {
                this.max = max - min; progress = p.getInt(key, default).coerceIn(min, max) - min
                progressTintList = ColorStateList.valueOf(ui.pal.accent)
                progressBackgroundTintList = ColorStateList.valueOf(ui.pal.border)
                thumbTintList = ColorStateList.valueOf(ui.pal.accent)
                setPadding(ui.dp(12), ui.dp(8), ui.dp(12), ui.dp(8))
                layoutParams = ui.fullWidthLp()
            }
            label.text = "$title · ${seek.progress + min} $unit"; seek.contentDescription = label.text
            seek.setOnSeekBarChangeListener(object : SeekBar.OnSeekBarChangeListener {
                override fun onProgressChanged(s: SeekBar?, value: Int, user: Boolean) {
                    label.text = "$title · ${value + min} $unit"; seek.contentDescription = label.text
                    if (user) p.edit().putInt(key, value + min).apply()
                }
                override fun onStartTrackingTouch(s: SeekBar?) {}
                override fun onStopTrackingTouch(s: SeekBar?) {}
            })
            addView(label); addView(seek)
        }
    }
}
