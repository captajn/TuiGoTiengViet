package engine

type keyEvent struct {
	keyCode uint32
	evType  int
	chType  int
	vnSym   VnLexiName
	tone    int
}

type inputProcessor struct {
	im     int
	keyMap [256]int
}

func newInputProcessor() *inputProcessor {
	p := &inputProcessor{}
	p.setIM(ImTelex)
	return p
}

func (p *inputProcessor) setIM(im int) {
	p.im = im
	switch im {
	case ImVni:
		p.useBuiltIn(vniMethodMapping)
	case ImViqr:
		p.useBuiltIn(viqrMethodMapping)
	case ImMsVi:
		p.useBuiltIn(msViMethodMapping)
	case ImTelexSimple:
		p.useBuiltIn(simpleTelexMethodMapping)
	case ImTelexVni:
		merged := make([]keyMapping, 0, len(telexMethodMapping)+len(vniMethodMapping))
		merged = append(merged, telexMethodMapping...)
		merged = append(merged, vniMethodMapping...)
		p.useBuiltIn(merged)
	default:
		p.im = ImTelex
		p.useBuiltIn(telexMethodMapping)
	}
}

func (p *inputProcessor) setIMKeyMap(keyMap [256]int) {
	p.im = ImUsrKeymap
	p.keyMap = keyMap
}

func isLower(c byte) bool { return c >= 'a' && c <= 'z' }
func isUpper(c byte) bool { return c >= 'A' && c <= 'Z' }
func toLower(c byte) byte {
	if isUpper(c) {
		return c + 32
	}
	return c
}
func toUpper(c byte) byte {
	if isLower(c) {
		return c - 32
	}
	return c
}

func (p *inputProcessor) useBuiltIn(map_ []keyMapping) {
	for i := range p.keyMap {
		p.keyMap[i] = VneNormal
	}
	for _, m := range map_ {
		p.keyMap[m.key] = m.action
		if m.action < VneCount {
			if isLower(m.key) {
				p.keyMap[toUpper(m.key)] = m.action
			} else if isUpper(m.key) {
				p.keyMap[toLower(m.key)] = m.action
			}
		}
	}
}

func (p *inputProcessor) keyCodeToEvent(keyCode uint32) keyEvent {
	ev := keyEvent{keyCode: keyCode, evType: VneNormal, tone: -1}
	if keyCode > 255 {
		ev.vnSym = isoToVnLexi(keyCode)
		if ev.vnSym == VnlNonVnChar {
			ev.chType = UkcNonVn
		} else {
			ev.chType = UkcVn
		}
	} else {
		ev.chType = ukcMap[keyCode]
		ev.evType = p.keyMap[keyCode]
		if ev.evType >= VneTone0 && ev.evType <= VneTone5 {
			ev.tone = ev.evType - VneTone0
		}
		if ev.evType >= VneCount {
			ev.chType = UkcVn
			ev.vnSym = VnLexiName(ev.evType - VneCount)
			ev.evType = VneMapChar
		} else {
			ev.vnSym = isoToVnLexi(keyCode)
		}
	}
	return ev
}

func (p *inputProcessor) keyCodeToSymbol(keyCode uint32) keyEvent {
	ev := keyEvent{keyCode: keyCode, evType: VneNormal, tone: -1}
	ev.vnSym = isoToVnLexi(keyCode)
	if keyCode > 255 {
		if ev.vnSym == VnlNonVnChar {
			ev.chType = UkcNonVn
		} else {
			ev.chType = UkcVn
		}
	} else {
		ev.chType = ukcMap[keyCode]
	}
	return ev
}

func (p *inputProcessor) getCharType(keyCode uint32) int {
	if keyCode > 255 {
		if isoToVnLexi(keyCode) == VnlNonVnChar {
			return UkcNonVn
		}
		return UkcVn
	}
	return ukcMap[keyCode]
}

func (p *inputProcessor) getKeyMap() [256]int {
	return p.keyMap
}
