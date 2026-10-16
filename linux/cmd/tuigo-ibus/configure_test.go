package main

import (
	"os"
	"path/filepath"
	"testing"
	"unicode/utf16"

	"tuigo-linux/config"
	"tuigo/engine"
)

// keymapDoc types one key through the engine the way a text document
// would see it: "delete Backs, insert Out", pass-through keys append.
func keymapDoc(e *engine.Engine, keys string) string {
	var doc []uint16
	for _, k := range keys {
		res := e.Filter(uint32(k))
		if res.Consumed {
			if res.Backs > len(doc) {
				res.Backs = len(doc)
			}
			doc = doc[:len(doc)-res.Backs]
			doc = append(doc, res.Out...)
		} else {
			doc = append(doc, uint16(k))
		}
	}
	return string(utf16.Decode(doc))
}

// writeKeymap writes a one-key UniKey keymap: 'q' emits 'đ' directly.
// 'q' is a plain letter in every built-in method (Telex/VNI/VIQR/…), so
// the custom mapping is detectable either way.
func writeKeymap(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "keymap.txt")
	if err := os.WriteFile(p, []byte("q = dd\n"), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

// TestKeymapDoesNotOverrideSelectedIM (PLAN G2): a stray keymap.txt must
// not hijack the method selected in tuigo.ini — only ImUsrKeymap loads it.
func TestKeymapDoesNotOverrideSelectedIM(t *testing.T) {
	km := writeKeymap(t)
	for _, im := range []int{engine.ImTelex, engine.ImVni, engine.ImViqr} {
		cfg := config.Defaults()
		cfg.InputMethod = im
		e := engine.New()
		configureEngine(e, cfg, km, filepath.Join(t.TempDir(), "macro.txt"))
		if got := keymapDoc(e, "q"); got != "q" {
			t.Fatalf("im=%d: keymap.txt hijacked the selected method, doc=%q, want %q", im, got, "q")
		}
	}
}

// TestKeymapUsedWhenSelected: with ImUsrKeymap selected the keymap file is
// loaded and its mapping applies ('q' -> 'đ').
func TestKeymapUsedWhenSelected(t *testing.T) {
	km := writeKeymap(t)
	cfg := config.Defaults()
	cfg.InputMethod = engine.ImUsrKeymap
	e := engine.New()
	configureEngine(e, cfg, km, filepath.Join(t.TempDir(), "macro.txt"))
	if got := keymapDoc(e, "q"); got != "đ" {
		t.Fatalf("ImUsrKeymap: keymap not applied, doc=%q, want %q", got, "đ")
	}
}

// TestMacroLoadedByConfigure: the macro file still loads through the same
// wiring (guards the reload path used on every Enable).
func TestMacroLoadedByConfigure(t *testing.T) {
	dir := t.TempDir()
	mp := filepath.Join(dir, "macro.txt")
	if err := os.WriteFile(mp, []byte("ko:không\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg := config.Defaults()
	e := engine.New()
	configureEngine(e, cfg, filepath.Join(dir, "keymap.txt"), mp)
	if got := keymapDoc(e, "ko "); got != "không " {
		t.Fatalf("macro not applied, doc=%q, want %q", got, "không ")
	}
}
