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
	// AlwaysMacro ("Gõ tắt cả khi tắt Tiếng Việt") lets macros expand
	// while Vietnamese typing is off. Same key name as Windows.
	AlwaysMacro bool
}

// ImChoice is one entry of the settings GUI input-method list.
type ImChoice struct {
	Label string
	Im    int
}

// ImChoices lists every input method the settings GUI offers. The list
// must cover all engine methods — an unknown selection would silently
// reset InputMethod to the first entry on Save (PLAN N3).
func ImChoices() []ImChoice {
	return []ImChoice{
		{"Telex", engine.ImTelex},
		{"VNI", engine.ImVni},
		{"VIQR", engine.ImViqr},
		{"MS Vi (AllCode)", engine.ImMsVi},
		{"Telex đơn giản", engine.ImTelexSimple},
		{"Telex + VNI", engine.ImTelexVni},
		{"Keymap tuỳ chỉnh (keymap.txt)", engine.ImUsrKeymap},
	}
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
	s.AlwaysMacro = getB("AlwaysMacro", s.AlwaysMacro)
	s.ToggleKeys = toggleKeysFromIni(m)
	return s, true
}

// toggleKeysFromIni resolves the VN/EN toggle chords from a shared
// tuigo.ini. ToggleKeyNames is the Linux-owned key. The legacy Linux
// ToggleKey chord list is honoured for older files, but a numeric
// ToggleKey is the Windows legacy value ("ToggleKey=%d") and must fall
// back to the defaults instead of silently disabling every hotkey (N1).
func toggleKeysFromIni(m map[string]string) []string {
	if v, ok := m["ToggleKeyNames"]; ok {
		return parseToggleKeys(v) // explicit empty = toggling disabled
	}
	if v, ok := m["ToggleKey"]; ok {
		if _, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			return Defaults().ToggleKeys // Windows legacy int — not ours
		}
		return parseToggleKeys(v) // legacy Linux chord list
	}
	return Defaults().ToggleKeys
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
		AlwaysMacro:       int32(b2i(s.AlwaysMacro)),
	}
}

// Save writes tuigo.ini atomically (temp + rename). Keys the Linux build
// does not manage are preserved verbatim so a config shared with the
// Windows build keeps its settings — Save must never wipe foreign keys
// (N2). The legacy Linux ToggleKey chord list is superseded by
// ToggleKeyNames and dropped; a numeric ToggleKey is Windows' legacy
// value and stays.
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
	managed := [][2]string{
		{"VietKey", bv(s.VietKey)},
		{"InputMethod", strconv.Itoa(s.InputMethod)},
		{"SpellCheck", bv(s.SpellCheck)},
		{"ModernStyle", bv(s.ModernStyle)},
		{"FreeMarking", bv(s.FreeMarking)},
		{"MacroEnabled", bv(s.MacroEnabled)},
		{"AutoNonVnRestore", bv(s.AutoNonVnRestore)},
		{"AlwaysMacro", bv(s.AlwaysMacro)},
		{"AutoCap", bv(s.AutoCap)},
		{"ToggleKeyNames", strings.Join(s.ToggleKeys, ",")},
	}
	existing, _ := os.ReadFile(IniPath())
	content := mergeIni(string(existing), managed)

	tmp := IniPath() + ".tmp"
	if err := os.WriteFile(tmp, []byte(content), 0644); err != nil {
		return err
	}
	return os.Rename(tmp, IniPath())
}

// mergeIni keeps every line of the existing ini the Linux settings do not
// manage (foreign keys, comments), then writes the managed keys fresh.
func mergeIni(existing string, managed [][2]string) string {
	managedSet := map[string]bool{}
	for _, kv := range managed {
		managedSet[kv[0]] = true
	}
	var sb strings.Builder
	sb.WriteString("[TuiGo]\n")
	for _, line := range strings.Split(existing, "\n") {
		t := strings.TrimSpace(line)
		if t == "" || strings.HasPrefix(t, "[") {
			continue
		}
		if t[0] == ';' || t[0] == '#' {
			sb.WriteString(line + "\n")
			continue
		}
		k, v, ok := strings.Cut(t, "=")
		if !ok {
			continue
		}
		key := strings.TrimSpace(k)
		if managedSet[key] {
			continue // rewritten below with the fresh value
		}
		if key == "ToggleKey" {
			if _, err := strconv.Atoi(strings.TrimSpace(v)); err != nil {
				continue // legacy Linux chord list — superseded by ToggleKeyNames
			}
		}
		sb.WriteString(line + "\n")
	}
	for _, kv := range managed {
		sb.WriteString(kv[0] + "=" + kv[1] + "\n")
	}
	return sb.String()
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
