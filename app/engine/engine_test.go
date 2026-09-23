package engine

import (
	"testing"
	"unicode/utf16"
)

// Simulates a text document: engine output is applied as
// "delete <backs> chars, insert <out>".
type docSim struct {
	text []rune
}

func (d *docSim) feed(e *Engine, key rune) {
	res := e.Filter(uint32(key))
	if res.Consumed {
		if res.Backs > len(d.text) {
			res.Backs = len(d.text)
		}
		d.text = d.text[:len(d.text)-res.Backs]
		if res.KeyOut {
			for _, c := range res.Out {
				d.text = append(d.text, rune(c))
			}
		} else {
			d.text = append(d.text, utf16.Decode(res.Out)...)
		}
	} else {
		d.text = append(d.text, key)
	}
}

func (d *docSim) backspace(e *Engine) {
	res := e.Backspace()
	if res.Consumed {
		if res.Backs > len(d.text) {
			res.Backs = len(d.text)
		}
		d.text = d.text[:len(d.text)-res.Backs]
		d.text = append(d.text, utf16.Decode(res.Out)...)
	} else if len(d.text) > 0 {
		d.text = d.text[:len(d.text)-1]
	}
}

func (d *docSim) typeKeys(e *Engine, keys string) {
	for _, k := range keys {
		d.feed(e, k)
	}
}

func (d *docSim) String() string { return string(d.text) }

func newEngine(opts *Options) *Engine {
	e := New()
	if opts != nil {
		e.SetOptions(opts)
	}
	return e
}

var defaultOpts = &Options{
	FreeMarking:       1,
	ModernStyle:       1,
	SpellCheckEnabled: 1,
}

func TestTelex(t *testing.T) {
	cases := []struct {
		keys string
		want string
	}{
		{"tieengs", "tiếng"},
		{"aa", "â"},
		{"aw", "ă"},
		{"ow", "ơ"},
		{"uw", "ư"},
		{"oo", "ô"},
		{"ee", "ê"},
		{"dd", "đ"},
		{"vietj nam", "viẹt nam"},
		{"vieetj", "việt"},
		{"dungx", "dũng"},
		{"xin chaof", "xin chào"},
		{"nguwowif", "người"},
		{"thuow", "thuơ"},
		{"duongf", "duòng"},
		{"duowngf", "dường"},
		{"muwowng", "mương"},
		{"muwowngf", "mường"},
		{"nghiengx", "nghiẽng"},
		{"nghieengx", "nghiễng"},
		{"quans", "quán"},
	}
	for _, c := range cases {
		e := newEngine(defaultOpts)
		d := &docSim{}
		d.typeKeys(e, c.keys)
		if got := d.String(); got != c.want {
			t.Errorf("telex %q: got %q, want %q", c.keys, got, c.want)
		}
	}
}

func TestTelexOldStyle(t *testing.T) {
	// modernStyle off -> tone on first vowel of oa/oe/uy
	opts := &Options{FreeMarking: 1}
	e := newEngine(opts)
	d := &docSim{}
	d.typeKeys(e, "hoaf")
	if got := d.String(); got != "hòa" {
		t.Errorf("old style: got %q, want %q", got, "hòa")
	}
}

func TestTelexModernStyle(t *testing.T) {
	e := newEngine(defaultOpts)
	d := &docSim{}
	d.typeKeys(e, "hoaf")
	if got := d.String(); got != "hoà" {
		t.Errorf("modern style: got %q, want %q", got, "hoà")
	}
}

func TestToneRemoval(t *testing.T) {
	// pressing the same tone key twice removes the tone (UniKey semantics:
	// the reverted key is consumed into the output, so "hoss" -> "hos")
	e := newEngine(defaultOpts)
	d := &docSim{}
	d.typeKeys(e, "hoss")
	if got := d.String(); got != "hos" {
		t.Errorf("hoss: got %q, want %q", got, "hos")
	}
	d.typeKeys(e, "s") // third s passes through as a literal letter
	if got := d.String(); got != "hoss" {
		t.Errorf("hosss: got %q, want %q", got, "hoss")
	}
}

func TestZRemovesTone(t *testing.T) {
	e := newEngine(defaultOpts)
	d := &docSim{}
	d.typeKeys(e, "hasz")
	if got := d.String(); got != "ha" {
		t.Errorf("hasz: got %q, want %q", got, "ha")
	}
}

func TestEscapeW(t *testing.T) {
	// standalone w maps to ư via mapChar fallback
	e := newEngine(defaultOpts)
	d := &docSim{}
	d.typeKeys(e, "w")
	if got := d.String(); got != "ư" {
		t.Errorf("w: got %q, want %q", got, "ư")
	}
}

func TestVni(t *testing.T) {
	e := newEngine(defaultOpts)
	e.SetInputMethod(ImVni)
	d := &docSim{}
	d.typeKeys(e, "vie6t5 nam")
	if got := d.String(); got != "việt nam" {
		t.Errorf("vni: got %q, want %q", got, "việt nam")
	}
}

func TestBackspaceToneMove(t *testing.T) {
	// "toanf" -> "toàn"; backspace removes 'n' -> tone moves to 'o'
	e := newEngine(defaultOpts)
	d := &docSim{}
	d.typeKeys(e, "toanf")
	if d.String() != "toàn" {
		t.Fatalf("setup: got %q", d.String())
	}
	d.backspace(e)
	if got := d.String(); got != "toà" {
		t.Errorf("backspace tone move: got %q, want %q", got, "toà")
	}
}

func TestNonVnWord(t *testing.T) {
	e := newEngine(defaultOpts)
	d := &docSim{}
	d.typeKeys(e, "hello ")
	if got := d.String(); got != "hello " {
		t.Errorf("hello: got %q, want %q", got, "hello ")
	}
}

func TestVietKeyOff(t *testing.T) {
	e := newEngine(defaultOpts)
	e.SetVietKey(false)
	d := &docSim{}
	d.typeKeys(e, "tieengs")
	if got := d.String(); got != "tieengs" {
		t.Errorf("vietKey off: got %q, want %q", got, "tieengs")
	}
}

func TestAutoNonVnRestore(t *testing.T) {
	// a word with marks but invalid spelling restores original keystrokes
	opts := &Options{
		FreeMarking:       1,
		ModernStyle:       1,
		SpellCheckEnabled: 1,
		AutoNonVnRestore:  1,
	}
	e := newEngine(opts)
	d := &docSim{}
	d.typeKeys(e, "ddasd ") // đá + d is invalid -> restore to "ddasd "
	if got := d.String(); got != "ddasd " {
		t.Errorf("auto restore: got %q, want %q", got, "ddasd ")
	}
}

func TestMacro(t *testing.T) {
	e := newEngine(&Options{MacroEnabled: 1})
	e.macStore.addItem("vn:Viet Nam")
	d := &docSim{}
	d.typeKeys(e, "vn ")
	if got := d.String(); got != "Viet Nam " {
		t.Errorf("macro: got %q, want %q", got, "Viet Nam ")
	}
}

func TestTCVNConvert(t *testing.T) {
	out := []uint16{'t', 'i', 0x1ebf, 'n', 'g'} // "tiếng" UTF-16
	ConvertTCVN(out)
	want := []uint16{'t', 'i', 0x00d5, 'n', 'g'} // TCVN3: ế -> 0xD5
	for i := range want {
		if out[i] != want[i] {
			t.Fatalf("unit %d: got 0x%04X, want 0x%04X", i, out[i], want[i])
		}
	}
	// ASCII + unmapped chars pass through untouched
	mixed := []uint16{'A', 'z', '1', 0x4E2D}
	ConvertTCVN(mixed)
	if mixed[0] != 'A' || mixed[1] != 'z' || mixed[2] != '1' || mixed[3] != 0x4E2D {
		t.Fatalf("passthrough broken: %v", mixed)
	}
}

func TestTelexVniHybrid(t *testing.T) {
	e := newEngine(defaultOpts)
	e.SetInputMethod(ImTelexVni)
	d := &docSim{}
	d.typeKeys(e, "on63") // VNI: ô + hỏi -> ổn
	if got := d.String(); got != "ổn" {
		t.Errorf("hybrid VNI side: got %q, want %q", got, "ổn")
	}
	e = newEngine(defaultOpts)
	e.SetInputMethod(ImTelexVni)
	d = &docSim{}
	d.typeKeys(e, "oonr") // Telex: ô + hỏi -> ổn
	if got := d.String(); got != "ổn" {
		t.Errorf("hybrid Telex side: got %q, want %q", got, "ổn")
	}
}
