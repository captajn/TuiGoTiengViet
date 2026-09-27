package engine

import (
	"bufio"
	"os"
	"strings"
)

// Convert one rune to StdVnChar (Unicode -> VNSTANDARD conversion path).
func runeToStdVn(r rune) StdVnChar {
	if r < 0x10000 {
		if sc, ok := uniToStdVn[uint16(r)]; ok {
			return sc
		}
	}
	return StdVnChar(r)
}

func textToStdVn(s string) []StdVnChar {
	out := make([]StdVnChar, 0, len(s))
	for _, r := range s {
		out = append(out, runeToStdVn(r))
	}
	return out
}

// addItem: item format "key:text"
func (m *macroStore) addItem(item string) bool {
	pos := strings.IndexByte(item, ':')
	if pos < 0 {
		return false
	}
	key := item[:pos]
	if len([]rune(key)) > MaxMacroKeyLen-1 {
		key = string([]rune(key)[:MaxMacroKeyLen-1])
	}
	text := item[pos+1:]
	if len(m.items) >= MaxMacroItems {
		return false
	}
	m.items[stdKeyString(textToStdVn(key))] = textToStdVn(text)
	return true
}

// loadFromFile: UTF-8 macro file; lines "key:text", ';' comments,
// header ";[DO NOT DELETE THIS LINE]***version=n" tolerated.
func (e *Engine) loadMacroFile(fname string) bool {
	f, err := os.Open(fname)
	if err != nil {
		return false
	}
	defer f.Close()

	e.macStore.items = make(map[string][]StdVnChar)

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 64*1024)
	for scanner.Scan() {
		line := scanner.Text()
		// strip UTF-8 BOM on first line if present
		line = strings.TrimPrefix(line, "\ufeff")
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, ";") {
			continue
		}
		e.macStore.addItem(line)
	}
	return true
}
