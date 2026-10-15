// tuigo-settings is the settings GUI for the Linux port. IBus launches it
// via the component's <setup> line, or the engine's "Cài đặt Tui Gõ…"
// panel menu item. It edits the shared tuigo.ini + macro.txt; running
// engine instances reload both on next activation (Enable).
package main

import (
	_ "embed"
	"fmt"
	"os"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"tuigo-linux/config"
)

//go:embed logo.jpg
var logoJPG []byte

var imNames = config.ImChoices()

func main() {
	a := app.NewWithID("com.tuigo.settings")
	a.SetIcon(fyne.NewStaticResource("logo.jpg", logoJPG))
	w := a.NewWindow("Cài đặt Tui Gõ")
	w.Resize(fyne.NewSize(560, 560))

	cfg, _ := config.Load()

	// --- tab Cài đặt ---------------------------------------------------
	vietCheck := widget.NewCheck("Bật bộ gõ tiếng Việt", func(bool) {})
	vietCheck.Checked = cfg.VietKey

	imLabels := make([]string, len(imNames))
	imSel := 0
	for i, n := range imNames {
		imLabels[i] = n.Label
		if n.Im == cfg.InputMethod {
			imSel = i
		}
	}
	imRadio := widget.NewRadioGroup(imLabels, func(string) {})
	imRadio.Selected = imLabels[imSel]

	spellCheck := widget.NewCheck("Kiểm tra chính tả khi gõ", func(bool) {})
	spellCheck.Checked = cfg.SpellCheck
	modernCheck := widget.NewCheck("Kiểu mới (oà, uý thay vì òa, uý)", func(bool) {})
	modernCheck.Checked = cfg.ModernStyle
	freeMarking := widget.NewCheck("Cho phép đặt dấu tự do", func(bool) {})
	freeMarking.Checked = cfg.FreeMarking
	macroCheck := widget.NewCheck("Bật gõ tắt (macro)", func(bool) {})
	macroCheck.Checked = cfg.MacroEnabled
	alwaysMacro := widget.NewCheck("Gõ tắt cả khi tắt Tiếng Việt", func(bool) {})
	alwaysMacro.Checked = cfg.AlwaysMacro
	restoreCheck := widget.NewCheck("Tự khôi phục từ không đúng tiếng Việt", func(bool) {})
	restoreCheck.Checked = cfg.AutoNonVnRestore
	autoCap := widget.NewCheck("Tự viết hoa sau dấu chấm câu", func(bool) {})
	autoCap.Checked = cfg.AutoCap

	has := func(k string) bool {
		for _, t := range cfg.ToggleKeys {
			if t == k {
				return true
			}
		}
		return false
	}
	ctrlShift := widget.NewCheck("Ctrl + Shift", func(bool) {})
	ctrlShift.Checked = has("ctrl+shift")
	altZ := widget.NewCheck("Alt + Z", func(bool) {})
	altZ.Checked = has("alt+z")
	ctrlSpace := widget.NewCheck("Ctrl + Space", func(bool) {})
	ctrlSpace.Checked = has("ctrl+space")

	// --- tab Gõ tắt ----------------------------------------------------
	macroData, _ := os.ReadFile(config.MacroPath())
	macroEdit := widget.NewMultiLineEntry()
	macroEdit.SetText(string(macroData))
	macroEdit.SetPlaceHolder("ko:không\nhn:hôm nay")
	macroHint := widget.NewLabel(
		"Mỗi dòng: từ-tắt:kết-quả — gõ từ tắt rồi Space/Enter để bung.\n" +
			"Ví dụ: ko:không — gõ \"ko\" + Space -> \"không\".")

	// --- save ----------------------------------------------------------
	status := widget.NewLabel("")
	save := func() bool {
		for _, n := range imNames {
			if imRadio.Selected == n.Label {
				cfg.InputMethod = n.Im
			}
		}
		cfg.VietKey = vietCheck.Checked
		cfg.SpellCheck = spellCheck.Checked
		cfg.ModernStyle = modernCheck.Checked
		cfg.FreeMarking = freeMarking.Checked
		cfg.MacroEnabled = macroCheck.Checked
		cfg.AlwaysMacro = alwaysMacro.Checked
		cfg.AutoNonVnRestore = restoreCheck.Checked
		cfg.AutoCap = autoCap.Checked
		cfg.ToggleKeys = cfg.ToggleKeys[:0]
		if ctrlShift.Checked {
			cfg.ToggleKeys = append(cfg.ToggleKeys, "ctrl+shift")
		}
		if altZ.Checked {
			cfg.ToggleKeys = append(cfg.ToggleKeys, "alt+z")
		}
		if ctrlSpace.Checked {
			cfg.ToggleKeys = append(cfg.ToggleKeys, "ctrl+space")
		}

		// validate macro before touching any file
		if bad := badMacroLine(macroEdit.Text); bad != "" {
			status.SetText("Lỗi gõ tắt: " + bad)
			dialog.ShowError(fmt.Errorf("dòng gõ tắt không hợp lệ: %s", bad), w)
			return false
		}
		if err := os.MkdirAll(config.Dir(), 0755); err != nil {
			status.SetText("Lỗi: " + err.Error())
			return false
		}
		if err := writeAtomic(config.MacroPath(), []byte(macroEdit.Text)); err != nil {
			status.SetText("Lỗi ghi macro: " + err.Error())
			return false
		}
		if err := cfg.Save(); err != nil {
			status.SetText("Lỗi ghi cài đặt: " + err.Error())
			return false
		}
		status.SetText("Đã lưu — có hiệu lực khi bàn phím kích hoạt lại")
		return true
	}

	saveBtn := widget.NewButton("Lưu", func() { save() })

	general := container.NewVBox(
		widget.NewLabelWithStyle("Tui Gõ — Bộ gõ tiếng Việt",
			fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		vietCheck,
		widget.NewSeparator(),
		widget.NewLabel("Kiểu gõ"), imRadio,
		widget.NewSeparator(),
		widget.NewLabel("Tùy chọn"),
		spellCheck, modernCheck, freeMarking, restoreCheck, autoCap,
		widget.NewSeparator(),
		widget.NewLabel("Phím tắt chuyển Việt/Anh"),
		container.NewHBox(ctrlShift, altZ, ctrlSpace),
	)
	macro := container.NewBorder(
		container.NewVBox(macroCheck, alwaysMacro, widget.NewSeparator(), macroHint),
		nil, nil, nil, container.NewScroll(macroEdit))

	tabs := container.NewAppTabs(
		container.NewTabItem("Cài đặt", container.NewScroll(general)),
		container.NewTabItem("Gõ tắt", macro),
	)
	w.SetContent(container.NewBorder(nil,
		container.NewVBox(status, saveBtn), nil, nil, tabs))
	w.ShowAndRun()
}

// badMacroLine returns the first macro line that has no ':' separator.
func badMacroLine(text string) string {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if !strings.Contains(line, ":") {
			return line
		}
	}
	return ""
}

func writeAtomic(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
