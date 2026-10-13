// Package config loads the shared tuigo.ini settings (same key names as
// the Windows build) plus the macro/keymap file paths for Linux.
//
// Files live in $XDG_CONFIG_HOME/tuigo/ (usually ~/.config/tuigo/):
//
//	tuigo.ini   — options, same key=value format as Windows
//	macro.txt   — UniKey macro format, auto-loaded by the engine
//	keymap.txt  — user keymap, auto-loaded by the engine
package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"tuigo/engine"
)

// Dir is the per-user config directory.
func Dir() string {
	xdg := os.Getenv("XDG_CONFIG_HOME")
	if xdg == "" {
		xdg = filepath.Join(os.Getenv("HOME"), ".config")
	}
	return filepath.Join(xdg, "tuigo")
}

func IniPath() string    { return filepath.Join(Dir(), "tuigo.ini") }
func MacroPath() string  { return filepath.Join(Dir(), "macro.txt") }
func KeymapPath() string { return filepath.Join(Dir(), "keymap.txt") }

// Settings mirrors the persisted tuigo.ini keys that apply on Linux.
type Settings struct {
	VietKey          bool
	InputMethod      int
	SpellCheck       bool
	ModernStyle      bool
	FreeMarking      bool
	MacroEnabled     bool
	AutoNonVnRestore bool
	AutoCap          bool
	// ToggleKeys lists VN/EN toggle chords: "ctrl+shift", "alt+z",
	// "ctrl+space". Empty/none disables hotkey toggling.
	ToggleKeys []string
}

// Defaults matching the Windows build (engine.New already defaults the
// engine options; these mirror loadSettings()'s fallbacks).
func Defaults() Settings {
	return Settings{
		VietKey:          true,
		InputMethod:      engine.ImTelex,
		SpellCheck:       true,
		ModernStyle:      true,
		FreeMarking:      true,
		MacroEnabled:     true,
		AutoNonVnRestore: false,
		AutoCap:          false,
		ToggleKeys:       []string{"ctrl+shift"},
	}
}

// Load reads tuigo.ini; missing file returns defaults with ok=false.
func Load() (Settings, bool) {
	s := Defaults()
	m, err := readIni(IniPath())
	if err != nil {
		return s, false
	}
	getI := func(name string) (int, bool) {
		if v, ok := m[name]; ok {
			if n, err := strconv.Atoi(v); err == nil {
				return n, true
			}
		}
		return 0, false
	}
	getB := func(name string, dflt bool) bool {
		if v, ok := m[name]; ok {
			return v == "1"
		}
		return dflt
	}
	if v, ok := getI("VietKey"); ok {
		s.VietKey = v != 0
	}
	if v, ok := getI("InputMethod"); ok {
		s.InputMethod = v
	}
	s.SpellCheck = getB("SpellCheck", s.SpellCheck)
	s.ModernStyle = getB("ModernStyle", s.ModernStyle)
	s.FreeMarking = getB("FreeMarking", s.FreeMarking)
	s.MacroEnabled = getB("MacroEnabled", s.MacroEnabled)
	s.AutoNonVnRestore = getB("AutoNonVnRestore", s.AutoNonVnRestore)
	s.AutoCap = getB("AutoCap", s.AutoCap)
	if v, ok := m["ToggleKey"]; ok {
		s.ToggleKeys = parseToggleKeys(v)
	}
	return s, true
}

func parseToggleKeys(v string) []string {
	var keys []string
	for _, k := range strings.Split(v, ",") {
		k = strings.ToLower(strings.TrimSpace(k))
		switch k {
		case "ctrl+shift", "alt+z", "ctrl+space":
			keys = append(keys, k)
		}
	}
	return keys
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// EngineOptions maps the settings onto engine.Options.
func (s Settings) EngineOptions() engine.Options {
	return engine.Options{
		SpellCheckEnabled: int32(b2i(s.SpellCheck)),
		ModernStyle:       int32(b2i(s.ModernStyle)),
		FreeMarking:       int32(b2i(s.FreeMarking)),
		MacroEnabled:      int32(b2i(s.MacroEnabled)),
		AutoNonVnRestore:  int32(b2i(s.AutoNonVnRestore)),
	}
}

// Save writes tuigo.ini atomically (temp + rename), same as Windows.
func (s Settings) Save() error {
	if err := os.MkdirAll(Dir(), 0755); err != nil {
		return err
	}
	bv := func(b bool) string {
		if b {
			return "1"
		}
		return "0"
	}
	var sb strings.Builder
	sb.WriteString("[TuiGo]\n")
	sb.WriteString("VietKey=" + bv(s.VietKey) + "\n")
	sb.WriteString("InputMethod=" + strconv.Itoa(s.InputMethod) + "\n")
	sb.WriteString("SpellCheck=" + bv(s.SpellCheck) + "\n")
	sb.WriteString("ModernStyle=" + bv(s.ModernStyle) + "\n")
	sb.WriteString("FreeMarking=" + bv(s.FreeMarking) + "\n")
	sb.WriteString("MacroEnabled=" + bv(s.MacroEnabled) + "\n")
	sb.WriteString("AutoNonVnRestore=" + bv(s.AutoNonVnRestore) + "\n")
	sb.WriteString("AutoCap=" + bv(s.AutoCap) + "\n")
	sb.WriteString("ToggleKey=" + strings.Join(s.ToggleKeys, ",") + "\n")

	tmp := IniPath() + ".tmp"
	if err := os.WriteFile(tmp, []byte(sb.String()), 0644); err != nil {
		return err
	}
	return os.Rename(tmp, IniPath())
}

// readIni parses a minimal key=value ini (one section, ; and # comments) —
// same format as the Windows shell's parser.
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
