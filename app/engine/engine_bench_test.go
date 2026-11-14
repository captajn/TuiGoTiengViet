package engine

import (
	"os"
	"path/filepath"
	"testing"
)

const benchTelex = "Tieengs Vieetj laf ngoon ngwx cuar nguwowif Vieetj Nam, " +
	"chungs toi ddang gox thuwr booj gox tieengs Vieetj tuwj host. "

const benchVni = "Tie61ng Vie65t la2 ngo6n ngu74 cu3a ngu7o7i2 Vie65t Nam, " +
	"chu1ng to6i d9ang go4 thu73 bo65 go4 tie61ng Vie65t tu75 host. "

func benchTyping(b *testing.B, im int, text string, macros bool) {
	e := New()
	e.SetInputMethod(im)
	if macros {
		path := filepath.Join(b.TempDir(), "macro.txt")
		data := "ko:không\nvn:Việt Nam\ntg:Tui Gõ\nhn:Hà Nội\nsg:Sài Gòn\n"
		if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
			b.Fatal(err)
		}
		if !e.LoadMacroFile(path) {
			b.Fatal("macro file not loaded")
		}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		for _, k := range text {
			e.Filter(uint32(k))
		}
		e.ResetBuf()
	}
}

func BenchmarkTelexSentence(b *testing.B)      { benchTyping(b, ImTelex, benchTelex, false) }
func BenchmarkTelexSentenceMacro(b *testing.B) { benchTyping(b, ImTelex, benchTelex, true) }
func BenchmarkVniSentence(b *testing.B)        { benchTyping(b, ImVni, benchVni, false) }
func BenchmarkTelexWordBackspace(b *testing.B) {
	e := New()
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		for _, k := range "nguwowif" {
			e.Filter(uint32(k))
		}
		for range 8 {
			e.Backspace()
		}
		e.ResetBuf()
	}
}
