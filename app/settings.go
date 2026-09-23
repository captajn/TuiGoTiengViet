package main

import (
	"golang.org/x/sys/windows/registry"
)

const regKey = `Software\BoGoTiengViet`

func saveSettings() {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, regKey, registry.WRITE)
	if err != nil {
		return
	}
	defer k.Close()
	v := uint32(0)
	if gVietKey {
		v = 1
	}
	k.SetDWordValue("VietKey", v)
	k.SetDWordValue("InputMethod", uint32(gIM))
	o := gEngine.GetOptions()
	k.SetDWordValue("SpellCheck", uint32(o.SpellCheckEnabled))
	k.SetDWordValue("ModernStyle", uint32(o.ModernStyle))
	k.SetDWordValue("FreeMarking", uint32(o.FreeMarking))
	k.SetDWordValue("MacroEnabled", uint32(o.MacroEnabled))
	k.SetDWordValue("AutoNonVnRestore", uint32(o.AutoNonVnRestore))
	k.SetDWordValue("AlwaysMacro", uint32(o.AlwaysMacro))
	setB := func(name string, v bool) {
		var x uint32
		if v {
			x = 1
		}
		k.SetDWordValue(name, x)
	}
	setB("SkipNonUSLayout", cfg.SkipNonUSLayout)
	setB("AutoCap", cfg.AutoCap)
	setB("UseClipboard", cfg.UseClipboard)
	setB("ShowHud", cfg.ShowHud)
	setB("RunAsAdmin", cfg.RunAsAdmin)
	setB("ShowOnLaunch", cfg.ShowOnLaunch)
	setB("SoundOnToggle", cfg.SoundOnToggle)
	k.SetDWordValue("ToggleKey", uint32(cfg.ToggleKey))
	k.SetDWordValue("Charset", uint32(cfg.Charset))
	k.SetDWordValue("HotkeyMods", uint32(cfg.HotkeyMods))
	k.SetDWordValue("HotkeyVk", uint32(cfg.HotkeyVk))
	k.SetDWordValue("FKeys", uint32(cfg.FKeys))
	setB("UseCtrlShift", cfg.UseCtrlShift)
	k.SetDWordValue("Theme", uint32(cfg.Theme))
	k.SetDWordValue("HudX", uint32(int32(cfg.HudX)))
	k.SetDWordValue("HudY", uint32(int32(cfg.HudY)))
}

func loadSettings() {
	k, err := registry.OpenKey(registry.CURRENT_USER, regKey, registry.READ)
	if err != nil {
		return
	}
	defer k.Close()
	if v, _, err := k.GetIntegerValue("VietKey"); err == nil {
		gVietKey = v != 0
	}
	if v, _, err := k.GetIntegerValue("InputMethod"); err == nil {
		gIM = int(v)
		gEngine.SetInputMethod(gIM)
	}
	o := gEngine.GetOptions()
	if v, _, err := k.GetIntegerValue("SpellCheck"); err == nil {
		o.SpellCheckEnabled = int32(v)
	}
	if v, _, err := k.GetIntegerValue("ModernStyle"); err == nil {
		o.ModernStyle = int32(v)
	}
	if v, _, err := k.GetIntegerValue("FreeMarking"); err == nil {
		o.FreeMarking = int32(v)
	}
	if v, _, err := k.GetIntegerValue("MacroEnabled"); err == nil {
		o.MacroEnabled = int32(v)
	}
	if v, _, err := k.GetIntegerValue("AutoNonVnRestore"); err == nil {
		o.AutoNonVnRestore = int32(v)
	}
	if v, _, err := k.GetIntegerValue("AlwaysMacro"); err == nil {
		o.AlwaysMacro = int32(v)
	}
	gEngine.SetOptions(o)
	getB := func(name string, dflt bool) bool {
		if v, _, err := k.GetIntegerValue(name); err == nil {
			return v != 0
		}
		return dflt
	}
	cfg.SkipNonUSLayout = getB("SkipNonUSLayout", false)
	cfg.AutoCap = getB("AutoCap", false)
	cfg.UseClipboard = getB("UseClipboard", false)
	cfg.ShowHud = getB("ShowHud", false)
	cfg.RunAsAdmin = getB("RunAsAdmin", false)
	cfg.ShowOnLaunch = getB("ShowOnLaunch", false)
	cfg.SoundOnToggle = getB("SoundOnToggle", false)
	if v, _, err := k.GetIntegerValue("ToggleKey"); err == nil {
		cfg.ToggleKey = int(v)
	}
	if v, _, err := k.GetIntegerValue("Charset"); err == nil {
		cfg.Charset = int(v)
	}
	if v, _, err := k.GetIntegerValue("HotkeyMods"); err == nil {
		cfg.HotkeyMods = int(v)
	}
	if v, _, err := k.GetIntegerValue("HotkeyVk"); err == nil {
		cfg.HotkeyVk = int(v)
	}
	if v, _, err := k.GetIntegerValue("FKeys"); err == nil {
		cfg.FKeys = int(v)
	}
	// migrate legacy ToggleKey -> new hotkey model
	if cfg.HotkeyVk == 0 && cfg.HotkeyMods == 0 {
		switch cfg.ToggleKey {
		case 2: // Ctrl+Shift only
			cfg.HotkeyMods, cfg.HotkeyVk = 3, 0
		default: // Alt+Z (+ optional Ctrl+Shift)
			cfg.HotkeyMods, cfg.HotkeyVk = 4, 'Z'
		}
	}
	cfg.UseCtrlShift = getB("UseCtrlShift", cfg.ToggleKey != 1)
	if v, _, err := k.GetIntegerValue("Theme"); err == nil {
		cfg.Theme = int(v)
	}
	if v, _, err := k.GetIntegerValue("HudX"); err == nil {
		cfg.HudX = int(int32(v))
	}
	if v, _, err := k.GetIntegerValue("HudY"); err == nil {
		cfg.HudY = int(int32(v))
	}
}
