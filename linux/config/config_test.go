package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tuigo/engine"
)

// setCfgDir points the config dir at a fresh temp dir for the test.
func setCfgDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	return filepath.Join(dir, "tuigo")
}

func writeIni(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tuigo.ini"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func rawIni(t *testing.T, dir string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, "tuigo.ini"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// TestSavePreservesForeignKeys (N2): Save must not wipe keys it does not
// manage — a tuigo.ini shared with Windows carries many Windows-only
// settings (Theme, HotkeyVk, AppMode.*, …) that Linux has no business
// deleting.
func TestSavePreservesForeignKeys(t *testing.T) {
	dir := setCfgDir(t)
	writeIni(t, dir, strings.Join([]string{
		"[TuiGo]",
		"VietKey=1",
		"ToggleKey=2",
		"Theme=3",
		"HotkeyVk=90",
		"AppMode.chrome.exe=1",
		"; user comment",
		"",
	}, "\n"))

	s := Defaults()
	s.SpellCheck = false // a managed key we can verify got rewritten
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}

	raw := rawIni(t, dir)
	for _, want := range []string{"ToggleKey=2", "Theme=3", "HotkeyVk=90", "AppMode.chrome.exe=1", "; user comment"} {
		if !strings.Contains(raw, want) {
			t.Errorf("Save dropped %q from the shared ini:\n%s", want, raw)
		}
	}
	if strings.Contains(raw, "SpellCheck=1") {
		t.Errorf("managed key SpellCheck not rewritten to 0:\n%s", raw)
	}
	if strings.Contains(raw, "SpellCheck=0") == false {
		t.Errorf("managed key SpellCheck=0 missing:\n%s", raw)
	}
}

// TestToggleKeyNumericFallsBackToDefaults (N1): ToggleKey is a legacy int
// on Windows. A shared ini must not silently kill the Linux toggle
// hotkeys — fall back to the defaults instead of parsing "2" as chords.
func TestToggleKeyNumericFallsBackToDefaults(t *testing.T) {
	dir := setCfgDir(t)
	writeIni(t, dir, "ToggleKey=2\n")

	s, ok := Load()
	if !ok {
		t.Fatal("Load failed")
	}
	if len(s.ToggleKeys) != 1 || s.ToggleKeys[0] != "ctrl+shift" {
		t.Fatalf("ToggleKeys = %v, want [ctrl+shift] (Windows legacy int must fall back to defaults)", s.ToggleKeys)
	}
}

// TestToggleKeyNamesPreferred (N1): the Linux-owned ToggleKeyNames key
// wins over the shared/legacy ToggleKey key.
func TestToggleKeyNamesPreferred(t *testing.T) {
	dir := setCfgDir(t)
	writeIni(t, dir, "ToggleKeyNames=alt+z,ctrl+space\nToggleKey=2\n")

	s, ok := Load()
	if !ok {
		t.Fatal("Load failed")
	}
	if len(s.ToggleKeys) != 2 || s.ToggleKeys[0] != "alt+z" || s.ToggleKeys[1] != "ctrl+space" {
		t.Fatalf("ToggleKeys = %v, want [alt+z ctrl+space]", s.ToggleKeys)
	}
}

// TestToggleKeyNamesEmptyDisables (N1): an explicit empty ToggleKeyNames
// means "no hotkey toggle" and must survive the round trip.
func TestToggleKeyNamesEmptyDisables(t *testing.T) {
	dir := setCfgDir(t)
	writeIni(t, dir, "ToggleKeyNames=\n")

	s, ok := Load()
	if !ok {
		t.Fatal("Load failed")
	}
	if len(s.ToggleKeys) != 0 {
		t.Fatalf("ToggleKeys = %v, want disabled", s.ToggleKeys)
	}
}

// TestLegacyLinuxToggleKeyNamesParsed: older Linux files stored the chord
// list in ToggleKey itself — keep reading those.
func TestLegacyLinuxToggleKeyNamesParsed(t *testing.T) {
	dir := setCfgDir(t)
	writeIni(t, dir, "ToggleKey=ctrl+shift,alt+z\n")

	s, ok := Load()
	if !ok {
		t.Fatal("Load failed")
	}
	if len(s.ToggleKeys) != 2 || s.ToggleKeys[0] != "ctrl+shift" || s.ToggleKeys[1] != "alt+z" {
		t.Fatalf("ToggleKeys = %v, want [ctrl+shift alt+z]", s.ToggleKeys)
	}
}

// TestAlwaysMacroRoundTrip (N5): AlwaysMacro ("Gõ tắt cả khi tắt Tiếng
// Việt") must survive save/load like every other engine option.
func TestAlwaysMacroRoundTrip(t *testing.T) {
	setCfgDir(t)
	s := Defaults()
	s.AlwaysMacro = true
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}
	got, ok := Load()
	if !ok {
		t.Fatal("Load failed")
	}
	if !got.AlwaysMacro {
		t.Fatal("AlwaysMacro lost in save/load")
	}
	if got.EngineOptions().AlwaysMacro != 1 {
		t.Fatal("EngineOptions does not map AlwaysMacro")
	}
}

// TestImChoicesCoverAllMethods (N3): the settings GUI combo must know
// every input method, otherwise Save silently rewrites an unknown
// selection (ImUsrKeymap) to Telex.
func TestImChoicesCoverAllMethods(t *testing.T) {
	all := map[int]string{
		engine.ImTelex:      "ImTelex",
		engine.ImTelexSimple: "ImTelexSimple",
		engine.ImVni:        "ImVni",
		engine.ImViqr:       "ImViqr",
		engine.ImMsVi:       "ImMsVi",
		engine.ImTelexVni:   "ImTelexVni",
		engine.ImUsrKeymap:  "ImUsrKeymap",
	}
	seen := map[int]bool{}
	for _, c := range ImChoices() {
		seen[c.Im] = true
		if c.Label == "" {
			t.Errorf("im=%d has an empty label", c.Im)
		}
	}
	for im, name := range all {
		if !seen[im] {
			t.Errorf("%s (%d) missing from ImChoices — Save would silently reset it", name, im)
		}
	}
}
