package session

import (
	"os"
	"path/filepath"
	"testing"
	"unicode/utf16"

	"tuigo/engine"
)

// replay drives a session through a string of ASCII chars and collects the
// document text (committed inserts) and the visible preedit, the way a host
// IME would render them.
type doc struct {
	text    []uint16 // committed text
	preedit []uint16 // currently shown marked text
	fwd     []uint16 // raw chars that were forwarded for the app to insert
}

func (d *doc) do(a Action) {
	d.text = append(d.text, a.Commit...)
	d.preedit = a.Preedit
	// a forwarded key is inserted by the app after the commit
}

func (d *doc) feed(s *Session, ch rune) Action {
	a := s.Feed(uint32(ch))
	d.do(a)
	if a.Forward {
		d.fwd = append(d.fwd, uint16(ch))
		d.text = append(d.text, uint16(ch))
	}
	return a
}

func (d *doc) visible() string {
	return string(utf16.Decode(append(append([]uint16(nil), d.text...), d.preedit...)))
}

func feedString(d *doc, s *Session, str string) {
	for _, r := range str {
		d.feed(s, r)
	}
}

func newSession(t *testing.T, viet bool) *Session {
	t.Helper()
	return New(engine.New(), viet)
}

func TestTelexWordCommitOnSpace(t *testing.T) {
	s := newSession(t, true)
	var d doc
	feedString(&d, s, "vieejt")

	if got := d.visible(); got != "việt" {
		t.Fatalf("preedit mid-word = %q, want %q", got, "việt")
	}
	a := d.feed(s, ' ')
	if !a.Forward {
		t.Fatal("space after a complete word must be forwarded")
	}
	if len(d.preedit) != 0 {
		t.Fatalf("preedit should be empty after word end, got %q", string(utf16.Decode(d.preedit)))
	}
	if got := d.visible(); got != "việt " {
		t.Fatalf("document = %q, want %q", got, "việt ")
	}
}

func TestTonePlacement(t *testing.T) {
	s := newSession(t, true)
	var d doc
	feedString(&d, s, "hoawf ") // hoă + huyền -> "hoằ"
	if got := d.visible(); got != "hoằ " {
		t.Fatalf("document = %q, want %q", got, "hoằ ")
	}
}

func TestFirstLetterSwallowedIntoPreedit(t *testing.T) {
	// The engine returns Consumed=false for the first letter of a word
	// (the app prints it on Windows). On IME APIs we must swallow it into
	// the preedit instead of forwarding — a later macro/restore edit can
	// otherwise try to delete already-committed text.
	s := newSession(t, true)
	var d doc
	a := d.feed(s, 't')
	if a.Forward {
		t.Fatal("first letter must not be forwarded — it is still editable")
	}
	feedString(&d, s, "ieengs")
	if got := d.visible(); got != "tiếng" {
		t.Fatalf("visible = %q, want %q", got, "tiếng")
	}
}

func TestNonVnRestoreDoesNotDuplicate(t *testing.T) {
	// Regression: restoring a non-VN word emits Backs covering the whole
	// word including chars that were never engine-consumed. If the first
	// char had been forwarded+committed, the restore would duplicate it.
	s := newSession(t, true)
	s.eng.GetOptions().AutoNonVnRestore = 1
	s.eng.GetOptions().SpellCheckEnabled = 1
	var d doc
	feedString(&d, s, "ddasd ") // đá+d invalid -> restores raw keys incl. 'd'
	if got := d.visible(); got != "ddasd " {
		t.Fatalf("document = %q, want %q", got, "ddasd ")
	}
}

func TestEnglishPassThrough(t *testing.T) {
	s := newSession(t, false)
	var d doc
	for _, r := range "hello " {
		a := d.feed(s, r)
		if !a.Forward {
			t.Fatalf("EN mode must forward %q", r)
		}
	}
	if got := d.visible(); got != "hello " {
		t.Fatalf("document = %q, want %q", got, "hello ")
	}
}

func TestMacroExpansion(t *testing.T) {
	dir := t.TempDir()
	mp := filepath.Join(dir, "macro.txt")
	if err := os.WriteFile(mp, []byte("ko:không\n"), 0644); err != nil {
		t.Fatal(err)
	}
	eng := engine.New()
	if !eng.LoadMacroFile(mp) {
		t.Fatal("LoadMacroFile failed")
	}
	s := New(eng, true)
	var d doc
	feedString(&d, s, "ko")
	a := d.feed(s, ' ')
	if a.Forward {
		t.Fatal("macro space must be consumed")
	}
	if got := d.visible(); got != "không " {
		t.Fatalf("document = %q, want %q", got, "không ")
	}
}

func TestBackspaceInsideWord(t *testing.T) {
	// The engine returns Consumed=false for a plain backspace (it lets the
	// app delete the char) — but on IME APIs the char lives in preedit,
	// so the session must consume it and update the model itself.
	s := newSession(t, true)
	var d doc
	feedString(&d, s, "vieet")
	b := s.Backspace()
	d.do(b)
	if b.Forward {
		t.Fatal("backspace inside a composing word must be consumed")
	}
	// deleting 't' drops the nặng tone — nặng is only valid on c/ch/p/t
	// endings, so the engine re-renders "việt" as "viê"
	if got := d.visible(); got != "viê" {
		t.Fatalf("preedit after backspace = %q, want %q", got, "viê")
	}
}

func TestBackspaceEmptyWordForwards(t *testing.T) {
	s := newSession(t, true)
	var d doc
	feedString(&d, s, "vieejt ")
	// word committed, buffer clean: next backspace must reach the app
	b := s.Backspace()
	d.do(b)
	if !b.Forward {
		t.Fatal("backspace with empty buffer must be forwarded")
	}
}

func TestResetCommitsPending(t *testing.T) {
	s := newSession(t, true)
	var d doc
	feedString(&d, s, "vieejt")
	a := s.Reset()
	d.do(a)
	if got := d.visible(); got != "việt" {
		t.Fatalf("document after reset = %q, want %q", got, "việt")
	}
}

func TestSetVietCommitsPending(t *testing.T) {
	s := newSession(t, true)
	var d doc
	feedString(&d, s, "vieejt")
	a := s.SetViet(false)
	d.do(a)
	if got := d.visible(); got != "việt" {
		t.Fatalf("document after toggle = %q, want %q", got, "việt")
	}
}

func TestVNI(t *testing.T) {
	s := newSession(t, true)
	s.eng.SetInputMethod(engine.ImVni)
	var d doc
	feedString(&d, s, "vie65t ")
	got := d.visible()
	if got != "việt " {
		t.Fatalf("VNI document = %q, want %q", got, "việt ")
	}
}

func TestAutoCap(t *testing.T) {
	s := newSession(t, true)
	s.AutoCap = true
	var d doc
	feedString(&d, s, "ok. ban")
	if got := d.visible(); got != "ok. Ban" {
		t.Fatalf("autocap document = %q, want %q", got, "ok. Ban")
	}
}

// TestMacroClearedWhenFileRemoved: the IME reloads macro.txt on every
// activation, so an emptied or removed file must drop the previous list
// instead of keeping stale expansions.
func TestMacroClearedWhenFileRemoved(t *testing.T) {
	dir := t.TempDir()
	mp := filepath.Join(dir, "macro.txt")
	if err := os.WriteFile(mp, []byte("ko:không\n"), 0644); err != nil {
		t.Fatal(err)
	}
	eng := engine.New()
	if !eng.LoadMacroFile(mp) {
		t.Fatal("initial LoadMacroFile failed")
	}

	// simulate Settings -> edit macro to an empty list, then remove it
	if err := os.Remove(mp); err != nil {
		t.Fatal(err)
	}
	if eng.LoadMacroFile(mp) {
		t.Fatal("LoadMacroFile must report failure for a missing file")
	}

	s := New(eng, true)
	var d doc
	feedString(&d, s, "ko")
	a := d.feed(s, ' ')
	if !a.Forward {
		t.Fatal("macro must be gone after the file was removed")
	}
	if got := d.visible(); got != "ko " {
		t.Fatalf("document = %q, want %q", got, "ko ")
	}
}

// TestMacroClearedWhenFileEmptied covers the same reload path with an
// existing but empty file (Settings -> clear the macro box -> Save).
func TestMacroClearedWhenFileEmptied(t *testing.T) {
	dir := t.TempDir()
	mp := filepath.Join(dir, "macro.txt")
	if err := os.WriteFile(mp, []byte("ko:không\n"), 0644); err != nil {
		t.Fatal(err)
	}
	eng := engine.New()
	if !eng.LoadMacroFile(mp) {
		t.Fatal("initial LoadMacroFile failed")
	}
	if err := os.WriteFile(mp, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	if !eng.LoadMacroFile(mp) {
		t.Fatal("LoadMacroFile failed on an empty file")
	}

	s := New(eng, true)
	var d doc
	feedString(&d, s, "ko")
	if a := d.feed(s, ' '); !a.Forward {
		t.Fatal("macro must be gone after the file was emptied")
	}
}

// TestAstralCodepointStaysInPreedit (N4): a codepoint above the BMP is
// encoded as a UTF-16 surrogate pair in the model — uint16(ch) silently
// truncated it.
func TestAstralCodepointStaysInPreedit(t *testing.T) {
	s := newSession(t, true)
	var d doc
	d.feed(s, 'a')
	a := d.feed(s, 0x1F600) // U+1F600 -> D83D DE00
	want := []uint16{'a', 0xD83D, 0xDE00}
	if len(a.Preedit) != len(want) {
		t.Fatalf("preedit = %v, want %v", a.Preedit, want)
	}
	for i := range want {
		if a.Preedit[i] != want[i] {
			t.Fatalf("preedit = %v, want %v", a.Preedit, want)
		}
	}
	d.feed(s, ' ')
	if got := d.visible(); got != "a\U0001F600 " {
		t.Fatalf("document = %q, want %q", got, "a\U0001F600 ")
	}
}

// TestAlwaysMacroExpandsInEnglish (N5): with "Gõ tắt cả khi tắt Tiếng
// Việt" on, macros expand even in English mode — the word still lives in
// preedit because the host cannot edit committed text.
func TestAlwaysMacroExpandsInEnglish(t *testing.T) {
	dir := t.TempDir()
	mp := filepath.Join(dir, "macro.txt")
	if err := os.WriteFile(mp, []byte("ko:không\n"), 0644); err != nil {
		t.Fatal(err)
	}
	eng := engine.New()
	if !eng.LoadMacroFile(mp) {
		t.Fatal("LoadMacroFile failed")
	}
	s := New(eng, false) // English mode
	s.AlwaysMacro = true

	var d doc
	feedString(&d, s, "ko")
	d.feed(s, ' ')
	if got := d.visible(); got != "không " {
		t.Fatalf("document = %q, want %q", got, "không ")
	}
}

// TestEnglishWithoutAlwaysMacroForwards (N5): the default stays pure
// pass-through in English mode — keys reach the app immediately and the
// macro does not expand.
func TestEnglishWithoutAlwaysMacroForwards(t *testing.T) {
	dir := t.TempDir()
	mp := filepath.Join(dir, "macro.txt")
	if err := os.WriteFile(mp, []byte("ko:không\n"), 0644); err != nil {
		t.Fatal(err)
	}
	eng := engine.New()
	if !eng.LoadMacroFile(mp) {
		t.Fatal("LoadMacroFile failed")
	}
	s := New(eng, false)

	var d doc
	a := d.feed(s, 'k')
	if !a.Forward {
		t.Fatal("English mode must forward keys immediately")
	}
	feedString(&d, s, "o ")
	if got := d.visible(); got != "ko " {
		t.Fatalf("document = %q, want %q", got, "ko ")
	}
}
