package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// Settings live in tuigo.ini next to the exe — the app is portable, so it
// must not leave traces in the registry. The Run key / scheduled task used
// by "Khởi động cùng hệ thống" is Windows infrastructure (only written when
// the user opts in), not app settings.
const regKey = `Software\BoGoTiengViet` // legacy — migrated to ini on first run

func iniPath() string {
	return filepath.Join(exeDir(), "tuigo.ini")
}

// readIni parses a minimal key=value ini (one section, ; and # comments).
func readIni(path string) (map[string]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	m := map[string]string{}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line[0] == ';' || line[0] == '#' || line[0] == '[' {
			continue
		}
		if k, v, ok := strings.Cut(line, "="); ok {
			m[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	}
	return m, nil
}

func saveSettings() {
	bv := func(b bool) string {
		if b {
			return "1"
		}
		return "0"
	}
	o := gEngine.GetOptions()
	var b strings.Builder
	b.WriteString("[TuiGo]\n")
	fmt.Fprintf(&b, "VietKey=%s\n", bv(gVietKey))
	fmt.Fprintf(&b, "InputMethod=%d\n", gIM)
	fmt.Fprintf(&b, "SpellCheck=%d\n", o.SpellCheckEnabled)
	fmt.Fprintf(&b, "ModernStyle=%d\n", o.ModernStyle)
	fmt.Fprintf(&b, "FreeMarking=%d\n", o.FreeMarking)
	fmt.Fprintf(&b, "MacroEnabled=%d\n", o.MacroEnabled)
	fmt.Fprintf(&b, "AutoNonVnRestore=%d\n", o.AutoNonVnRestore)
	fmt.Fprintf(&b, "AlwaysMacro=%d\n", o.AlwaysMacro)
	fmt.Fprintf(&b, "SkipNonUSLayout=%s\n", bv(cfg.SkipNonUSLayout))
	fmt.Fprintf(&b, "AutoCap=%s\n", bv(cfg.AutoCap))
	fmt.Fprintf(&b, "UseClipboard=%s\n", bv(cfg.UseClipboard))
	fmt.Fprintf(&b, "ShowHud=%s\n", bv(cfg.ShowHud))
	fmt.Fprintf(&b, "RunAsAdmin=%s\n", bv(cfg.RunAsAdmin))
	fmt.Fprintf(&b, "ShowOnLaunch=%s\n", bv(cfg.ShowOnLaunch))
	fmt.Fprintf(&b, "SoundOnToggle=%s\n", bv(cfg.SoundOnToggle))
	fmt.Fprintf(&b, "ToggleKey=%d\n", cfg.ToggleKey)
	fmt.Fprintf(&b, "Charset=%d\n", cfg.Charset)
	fmt.Fprintf(&b, "HotkeyMods=%d\n", cfg.HotkeyMods)
	fmt.Fprintf(&b, "HotkeyVk=%d\n", cfg.HotkeyVk)
	fmt.Fprintf(&b, "FKeys=%d\n", cfg.FKeys)
	fmt.Fprintf(&b, "UseCtrlShift=%s\n", bv(cfg.UseCtrlShift))
	fmt.Fprintf(&b, "Theme=%d\n", cfg.Theme)
	fmt.Fprintf(&b, "HudX=%d\n", cfg.HudX)
	fmt.Fprintf(&b, "HudY=%d\n", cfg.HudY)
	// Atomic-ish: write temp then rename so a crash mid-write can't leave a
	// truncated ini that silently resets every option on next start.
	tmp := iniPath() + ".tmp"
	if err := os.WriteFile(tmp, []byte(b.String()), 0644); err == nil {
		os.Rename(tmp, iniPath())
	}
}

// loadRegistrySettings reads the old HKCU\Software\BoGoTiengViet store once —
// kept only for migrating installs that predate the ini file.
func loadRegistrySettings() map[string]string {
	k, err := registry.OpenKey(registry.CURRENT_USER, regKey, registry.READ)
	if err != nil {
		return nil
	}
	defer k.Close()
	names := []string{
		"VietKey", "InputMethod", "SpellCheck", "ModernStyle", "FreeMarking",
		"MacroEnabled", "AutoNonVnRestore", "AlwaysMacro", "SkipNonUSLayout",
		"AutoCap", "UseClipboard", "ShowHud", "RunAsAdmin", "ShowOnLaunch",
		"SoundOnToggle", "ToggleKey", "Charset", "HotkeyMods", "HotkeyVk",
		"FKeys", "UseCtrlShift", "Theme", "HudX", "HudY",
	}
	m := map[string]string{}
	for _, n := range names {
		if v, _, err := k.GetIntegerValue(n); err == nil {
			m[n] = strconv.FormatUint(uint64(v), 10)
		}
	}
	if len(m) == 0 {
		return nil
	}
	return m
}

// loadSettings reads persisted config. Reports whether config already
// existed — false means first run (show the panel once for onboarding).
func loadSettings() bool {
	vals, err := readIni(iniPath())
	migrated := false
	if err != nil {
		vals = loadRegistrySettings()
		if vals == nil {
			return false
		}
		migrated = true
	}
	getI := func(name string) (int, bool) {
		s, ok := vals[name]
		if !ok {
			return 0, false
		}
		n, err := strconv.Atoi(s)
		return n, err == nil
	}
	getB := func(name string, dflt bool) bool {
		if s, ok := vals[name]; ok {
			return s == "1"
		}
		return dflt
	}
	if v, ok := getI("VietKey"); ok {
		gVietKey = v != 0
	}
	if v, ok := getI("InputMethod"); ok {
		gIM = v
		gEngine.SetInputMethod(gIM)
	}
	o := gEngine.GetOptions()
	if v, ok := getI("SpellCheck"); ok {
		o.SpellCheckEnabled = int32(v)
	}
	if v, ok := getI("ModernStyle"); ok {
		o.ModernStyle = int32(v)
	}
	if v, ok := getI("FreeMarking"); ok {
		o.FreeMarking = int32(v)
	}
	if v, ok := getI("MacroEnabled"); ok {
		o.MacroEnabled = int32(v)
	}
	if v, ok := getI("AutoNonVnRestore"); ok {
		o.AutoNonVnRestore = int32(v)
	}
	if v, ok := getI("AlwaysMacro"); ok {
		o.AlwaysMacro = int32(v)
	}
	gEngine.SetOptions(o)
	cfg.SkipNonUSLayout = getB("SkipNonUSLayout", false)
	cfg.AutoCap = getB("AutoCap", false)
	cfg.UseClipboard = getB("UseClipboard", false)
	cfg.ShowHud = getB("ShowHud", false)
	cfg.RunAsAdmin = getB("RunAsAdmin", false)
	cfg.ShowOnLaunch = getB("ShowOnLaunch", false)
	cfg.SoundOnToggle = getB("SoundOnToggle", false)
	if v, ok := getI("ToggleKey"); ok {
		cfg.ToggleKey = v
	}
	if v, ok := getI("Charset"); ok {
		cfg.Charset = v
	}
	if v, ok := getI("HotkeyMods"); ok {
		cfg.HotkeyMods = v
	}
	if v, ok := getI("HotkeyVk"); ok {
		cfg.HotkeyVk = v
	}
	if v, ok := getI("FKeys"); ok {
		cfg.FKeys = v
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
	if v, ok := getI("Theme"); ok {
		cfg.Theme = v
	}
	if v, ok := getI("HudX"); ok {
		cfg.HudX = v
	}
	if v, ok := getI("HudY"); ok {
		cfg.HudY = v
	}
	if migrated {
		saveSettings()
		registry.DeleteKey(registry.CURRENT_USER, regKey) // portable: no traces left
	}
	return true
}
