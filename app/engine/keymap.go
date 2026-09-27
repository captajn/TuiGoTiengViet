package engine

import (
	"bufio"
	"os"
	"strings"
)

// User keymap file format (usrkeymap.cpp):
//
//	X = ActionLabel    (single char key = label)
//	; comment
var ukEvLabels = map[string]int{
	"Tone0":     VneTone0,
	"Tone1":     VneTone1,
	"Tone2":     VneTone2,
	"Tone3":     VneTone3,
	"Tone4":     VneTone4,
	"Tone5":     VneTone5,
	"Roof-All":  VneRoofAll,
	"Roof-A":    VneRoofA,
	"Roof-E":    VneRoofE,
	"Roof-O":    VneRoofO,
	"Hook-Bowl": VneHookAll,
	"Hook-UO":   VneHookUO,
	"Hook-U":    VneHookU,
	"Hook-O":    VneHookO,
	"Bowl":      VneBowl,
	"D-Mark":    VneDd,
	"Telex-W":   VneTelexW,
	"Escape":    VneEscChar,
	"DD":        VneCount + int(VnlDD),
	"dd":        VneCount + int(Vnl_dd),
	"A^":        VneCount + int(VnlAr),
	"a^":        VneCount + int(Vnl_ar),
	"A(":        VneCount + int(VnlAb),
	"a(":        VneCount + int(Vnl_ab),
	"E^":        VneCount + int(VnlEr),
	"e^":        VneCount + int(Vnl_er),
	"O^":        VneCount + int(VnlOr),
	"o^":        VneCount + int(Vnl_or),
	"O+":        VneCount + int(VnlOh),
	"o+":        VneCount + int(Vnl_oh),
	"U+":        VneCount + int(VnlUh),
	"u+":        VneCount + int(Vnl_uh),
}

// loadKeyMapFile parses a UniKey keymap.txt into a [256]int keymap.
func loadKeyMapFile(fname string) ([256]int, bool) {
	var keyMap [256]int
	for i := range keyMap {
		keyMap[i] = VneNormal
	}

	f, err := os.Open(fname)
	if err != nil {
		return keyMap, false
	}
	defer f.Close()

	assigned := map[byte]bool{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if idx := strings.IndexByte(line, ';'); idx >= 0 {
			line = line[:idx]
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if len(name) != 1 {
			continue
		}
		action, ok := ukEvLabels[value]
		if !ok {
			continue
		}
		c := name[0]
		if assigned[c] {
			continue // already assigned, don't accept this map
		}
		keyMap[c] = action
		assigned[c] = true
		if action < VneCount {
			keyMap[toLower(c)] = action
			keyMap[toUpper(c)] = action
		}
	}
	return keyMap, true
}
