package main

import (
	"fmt"
	"path/filepath"
	"testing"
)

// Unset color overrides must be -1, never 0: 0 is black, so a zero-valued
// default paints the tray letter/badge and the accent solid black.
func TestDefaultConfigShipsUnsetColorsAndHotkey(t *testing.T) {
	d := defaultConfig()
	if d.TrayV != -1 || d.TrayVBg != -1 || d.TrayE != -1 || d.TrayEBg != -1 || d.Accent != -1 || d.AccentText != -1 {
		t.Fatalf("defaultConfig must use -1 for unset color overrides, got %+v", d)
	}
	if d.HotkeyMods != 4 || d.HotkeyVk != 'Z' || !d.UseCtrlShift {
		t.Fatalf("defaultConfig must set the default Alt+Z + Ctrl+Shift hotkey, got %+v", d)
	}
}

// loadConfigForTest resets the globals the way loadSettings does, then applies
// one ini snapshot (the first-run path keeps defaultConfig untouched).
func loadConfigForTest(t *testing.T, vals map[string]string) config {
	t.Helper()
	setupAppModeTest(t)
	cfg = defaultConfig()
	applyLoadedSettings(vals)
	return cfg
}

func TestColorOverridesFromIni(t *testing.T) {
	cases := []struct {
		name string
		vals map[string]string
		want [6]int // TrayV, TrayVBg, TrayE, TrayEBg, Accent, AccentText
	}{
		{
			name: "absent keys follow the palette",
			vals: map[string]string{},
			want: [6]int{-1, -1, -1, -1, -1, -1},
		},
		{
			// The build that persisted zeroed color fields left exactly this
			// snapshot behind; it must not render as black.
			name: "legacy zeroed snapshot loads as unset",
			vals: map[string]string{
				"TrayV": "0", "TrayVBg": "0", "TrayE": "0", "TrayEBg": "0",
				"Accent": "0", "AccentText": "0",
			},
			want: [6]int{-1, -1, -1, -1, -1, -1},
		},
		{
			name: "picked colors survive a reload",
			vals: map[string]string{
				"TrayV":      fmt.Sprint(0x123456),
				"TrayVBg":    fmt.Sprint(0x445566),
				"TrayE":      fmt.Sprint(0x778899),
				"TrayEBg":    fmt.Sprint(0xAABBCC),
				"Accent":     fmt.Sprint(0x197F64),
				"AccentText": fmt.Sprint(0xFFFFFF),
			},
			want: [6]int{0x123456, 0x445566, 0x778899, 0xAABBCC, 0x197F64, 0xFFFFFF},
		},
		{
			// Documented trade-off: only the exact all-zero group is treated as
			// the legacy bug, so a group the user did customize keeps its
			// values verbatim (including a deliberately picked black).
			name: "partially picked tray group is kept verbatim",
			vals: map[string]string{
				"TrayV": fmt.Sprint(0x123456), "TrayVBg": "0", "TrayE": "0", "TrayEBg": "0",
			},
			want: [6]int{0x123456, 0, 0, 0, -1, -1},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := loadConfigForTest(t, tc.vals)
			actual := [6]int{got.TrayV, got.TrayVBg, got.TrayE, got.TrayEBg, got.Accent, got.AccentText}
			if actual != tc.want {
				t.Fatalf("colors = %v, want %v", actual, tc.want)
			}
		})
	}
}

func TestHotkeyMigration(t *testing.T) {
	cases := []struct {
		name     string
		vals     map[string]string
		wantMods int
		wantVk   int
	}{
		{
			name:     "legacy ini gets the shipped Alt+Z",
			vals:     map[string]string{"ToggleKey": "0"},
			wantMods: 4,
			wantVk:   'Z',
		},
		{
			name:     "legacy Ctrl+Shift-only keeps the chord, no key",
			vals:     map[string]string{"ToggleKey": "2"},
			wantMods: 3,
			wantVk:   0,
		},
		{
			name:     "modern pair wins over the legacy ToggleKey",
			vals:     map[string]string{"ToggleKey": "2", "HotkeyMods": "1", "HotkeyVk": "75"},
			wantMods: 1,
			wantVk:   75,
		},
		{
			name:     "a hotkey the user cleared stays cleared",
			vals:     map[string]string{"HotkeyMods": "0", "HotkeyVk": "0"},
			wantMods: 0,
			wantVk:   0,
		},
		{
			name:     "separate keys are read independently",
			vals:     map[string]string{"HotkeyMods": "4"},
			wantMods: 4,
			wantVk:   'Z',
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := loadConfigForTest(t, tc.vals)
			if got.HotkeyMods != tc.wantMods || got.HotkeyVk != tc.wantVk {
				t.Fatalf("hotkey = mods %#x / vk %#x, want mods %#x / vk %#x",
					got.HotkeyMods, got.HotkeyVk, tc.wantMods, tc.wantVk)
			}
		})
	}
}

// The save side and the load side must agree on the -1 sentinel, otherwise a
// config that was never customized comes back as black on the next launch.
func TestSavedIniRoundTripsUnsetColors(t *testing.T) {
	setupAppModeTest(t)
	cfg = defaultConfig()

	path := filepath.Join(t.TempDir(), "tuigo.ini")
	if err := saveSettingsTo(path); err != nil {
		t.Fatalf("saveSettingsTo: %v", err)
	}
	vals, err := readIni(path)
	if err != nil {
		t.Fatalf("readIni: %v", err)
	}
	for _, key := range []string{"TrayV", "TrayVBg", "TrayE", "TrayEBg", "Accent", "AccentText"} {
		if vals[key] != "-1" {
			t.Fatalf("%s persisted as %q, want -1 (unset)", key, vals[key])
		}
	}

	cfg = defaultConfig()
	applyLoadedSettings(vals)
	if cfg.TrayV != -1 || cfg.TrayVBg != -1 || cfg.TrayE != -1 || cfg.TrayEBg != -1 || cfg.Accent != -1 || cfg.AccentText != -1 {
		t.Fatalf("reload turned unset colors into %+v", cfg)
	}
}
