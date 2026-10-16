package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSaveSettingsToPreservesForeignKeys (N2 mirror on Windows): saving
// settings must not wipe keys this build does not manage — a tuigo.ini
// shared with the Linux build carries Linux-owned keys (ToggleKeyNames)
// and user comments that must survive.
func TestSaveSettingsToPreservesForeignKeys(t *testing.T) {
	oldEngine, oldCfg, oldGlobal := gEngine, cfg, gVietKey
	oldModes := gRememberedAppViet
	t.Cleanup(func() {
		gEngine, cfg, gVietKey = oldEngine, oldCfg, oldGlobal
		gRememberedAppViet = oldModes
	})
	var err error
	gEngine, err = newEngine()
	if err != nil {
		t.Fatal(err)
	}
	cfg = config{RememberAppMode: true}
	gVietKey = true
	gRememberedAppViet = map[string]bool{"chrome.exe": true}

	path := filepath.Join(t.TempDir(), "tuigo.ini")
	if err := os.WriteFile(path, []byte(strings.Join([]string{
		"[TuiGo]",
		"VietKey=1",
		"ToggleKeyNames=alt+z",
		"FutureKey=7",
		"AppMode.edge.exe=1",
		"; user comment",
		"",
	}, "\n")), 0644); err != nil {
		t.Fatal(err)
	}

	if err := saveSettingsTo(path); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	raw := string(b)
	for _, want := range []string{"ToggleKeyNames=alt+z", "FutureKey=7", "; user comment", "AppMode.chrome.exe=1"} {
		if !strings.Contains(raw, want) {
			t.Errorf("save dropped %q from the shared ini:\n%s", want, raw)
		}
	}
	if strings.Contains(raw, "AppMode.edge.exe") {
		t.Errorf("stale AppMode entry survived a managed rewrite:\n%s", raw)
	}
}

// TestMergeIniPreserveIsIdempotent: running the merge on its own output
// must not duplicate preserved lines.
func TestMergeIniPreserveIsIdempotent(t *testing.T) {
	generated := "[TuiGo]\nVietKey=1\n"
	existing := "[TuiGo]\nToggleKeyNames=alt+z\n; note\n"
	once := mergeIniPreserve(existing, generated)
	twice := mergeIniPreserve(once, generated)
	if once != twice {
		t.Fatalf("merge not idempotent:\nonce:  %q\ntwice: %q", once, twice)
	}
	if strings.Count(twice, "ToggleKeyNames=alt+z") != 1 {
		t.Fatalf("preserved key duplicated:\n%s", twice)
	}
}
