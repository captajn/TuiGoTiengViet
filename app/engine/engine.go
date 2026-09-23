package engine

// Pure Go port of UkEngine (ukengine.cpp), specialized to Unicode (UTF-16) output.
// The original supports multiple output charsets; this port keeps only the
// Unicode path (charsetId == CONV_CHARSET_UNICODE), which makes getSeqSteps a
// simple character count and lets writeOutput emit UTF-16 code units directly.

type Options struct {
	FreeMarking         int32
	ModernStyle         int32
	MacroEnabled        int32
	UseUnicodeClipboard int32
	AlwaysMacro         int32
	StrictSpellCheck    int32
	UseIME              int32
	SpellCheckEnabled   int32
	AutoNonVnRestore    int32
}

const (
	OutChar = iota // UkCharOutput
	OutKey         // UkKeyOutput
)

// Result of processing one key.
type Result struct {
	Consumed bool     // swallow the key (engine produced output)
	Backs    int      // number of UTF-16 units to backspace
	Out      []uint16 // UTF-16 output to insert
	KeyOut   bool     // Out contains raw key codes, not characters
}

type keyBufEntry struct {
	ev        keyEvent
	converted bool
}

type wordInfo struct {
	// info for word ending at this position
	form                        int // vnw_*
	c1Offset, vOffset, c2Offset int

	// union { VowelSeq vseq; ConSeq cseq } — single storage, cast on read
	seq int

	// info for current symbol
	caps, tone int
	vnSym      VnLexiName // canonical symbol, after caps & tone removed; -1 for non-Vn
	keyCode    uint32
}

func (w *wordInfo) vseq() VowelSeq     { return VowelSeq(w.seq) }
func (w *wordInfo) cseq() ConSeq       { return ConSeq(w.seq) }
func (w *wordInfo) setVseq(v VowelSeq) { w.seq = int(v) }
func (w *wordInfo) setCseq(c ConSeq)   { w.seq = int(c) }

type macroStore struct {
	items map[string][]StdVnChar // key: stdChars encoded as string of runes
}

func stdKeyString(key []StdVnChar) string {
	rs := make([]rune, len(key))
	for i, c := range key {
		rs[i] = rune(c)
	}
	return string(rs)
}

func (m *macroStore) lookup(key []StdVnChar) []StdVnChar {
	if m == nil || len(m.items) == 0 {
		return nil
	}
	return m.items[stdKeyString(key)]
}

// ------------------------------------------------------------------

type Engine struct {
	buffer     [MaxUkEngine]wordInfo
	keyStrokes [MaxUkEngine]keyBufEntry

	changePos     int
	backs         int
	bufSize       int
	current       int
	singleMode    bool
	keyBufSize    int
	keyCurrent    int
	toEscape      bool
	outputWritten bool
	reverted      bool
	keyRestored   bool
	keyRestoring  bool
	outType       int

	// keyboard state (replaces m_keyCheckFunc)
	ShiftPressed bool
	CapsLockOn   bool

	out []uint16

	opts     Options
	vietKey  bool
	input    *inputProcessor
	macStore *macroStore

	usedAsMapChar bool // static flag in processTelexW
}

func New() *Engine {
	e := &Engine{
		vietKey: true,
		// UniKey defaults: strict Telex (FreeMarking=0) can't reach back
		// through 'i' — "khoiwr" would stay literal instead of "khởi".
		opts:       Options{FreeMarking: 1, ModernStyle: 1, SpellCheckEnabled: 1, MacroEnabled: 1},
		input:      newInputProcessor(),
		macStore:   &macroStore{items: map[string][]StdVnChar{}},
		bufSize:    MaxUkEngine,
		keyBufSize: MaxUkEngine,
		current:    -1,
		keyCurrent: -1,
	}
	return e
}

func (e *Engine) SetInputMethod(im int) { e.input.setIM(im) }
func (e *Engine) SetKeyMap(m [256]int)  { e.input.setIMKeyMap(m) }
func (e *Engine) SetVietKey(v bool)     { e.vietKey = v }
func (e *Engine) SetOptions(o *Options) { e.opts = *o }
func (e *Engine) GetOptions() *Options  { return &e.opts }
func (e *Engine) SetSingleMode()        { e.singleMode = true }
func (e *Engine) AtWordBeginning() bool {
	return e.current < 0 || e.buffer[e.current].form == VnwEmpty
}

func (e *Engine) reset() {
	e.current = -1
	e.keyCurrent = -1
	e.singleMode = false
	e.toEscape = false
}

func (e *Engine) ResetBuf() { e.reset() }

func (e *Engine) Pass(keyCode uint32) {
	ev := e.input.keyCodeToEvent(keyCode)
	e.processAppend(&ev)
}

// ----------------------------------------------------------
func (e *Engine) getSeqSteps(first, last int) int {
	if last < first {
		return 0
	}
	return last - first + 1 // Unicode output: 1 entry = 1 UTF-16 unit
}

func (e *Engine) markChange(pos int) {
	if pos < e.changePos {
		e.backs += e.getSeqSteps(pos, e.changePos-1)
		e.changePos = pos
	}
}

func stdCharToUTF16(sc StdVnChar) (uint16, bool) {
	if sc == InvalidStdChar {
		return 0, false
	}
	if sc < VnStdCharOffset {
		return uint16(sc), true
	}
	idx := sc - VnStdCharOffset
	if int(idx) < TotalVnChars {
		return unicodeTable[idx], true
	}
	return 0, false
}

// ----------------------------------------------------------
func (e *Engine) writeOutput() {
	e.out = e.out[:0]
	for i := e.changePos; i <= e.current; i++ {
		var stdChar StdVnChar
		if e.buffer[i].vnSym != VnlNonVnChar {
			stdChar = StdVnChar(e.buffer[i].vnSym) + VnStdCharOffset
			if e.buffer[i].caps != 0 {
				stdChar--
			}
			if e.buffer[i].tone != 0 {
				stdChar += StdVnChar(e.buffer[i].tone * 2)
			}
		} else {
			stdChar = isoToStdVnChar(e.buffer[i].keyCode)
		}
		if c, ok := stdCharToUTF16(stdChar); ok {
			e.out = append(e.out, c)
		}
	}
	e.outputWritten = true
}

// ----------------------------------------------------------
// isValidCV / isValidVC / isValidCVC
// ----------------------------------------------------------
func isValidCV(c ConSeq, v VowelSeq) bool {
	if c == CsNil || v == VsNil {
		return true
	}
	vInfo := vSeqList[v]

	if (c == CsGi && vInfo.v[0] == Vnl_i) ||
		(c == CsQu && vInfo.v[0] == Vnl_u) {
		return false // gi doesn't go with i, qu doesn't go with u
	}

	if c == CsK {
		// k can only go with the following vowel sequences
		kVseq := []VowelSeq{Vs_e, Vs_i, Vs_y, Vs_er, Vs_eo, Vs_eu,
			Vs_eru, Vs_ia, Vs_ie, Vs_ier, Vs_ieu, Vs_ieru}
		for _, kv := range kVseq {
			if kv == v {
				return true
			}
		}
		return false
	}
	return true
}

func isValidVC(v VowelSeq, c ConSeq) bool {
	if v == VsNil || c == CsNil {
		return true
	}
	if vSeqList[v].conSuffix == 0 {
		return false
	}
	if !cSeqList[c].suffix {
		return false
	}
	return vcPairSet[vcPair{v, c}]
}

func isValidCVC(c1 ConSeq, v VowelSeq, c2 ConSeq) bool {
	if v == VsNil {
		return c1 == CsNil || c2 != CsNil
	}
	if c1 == CsNil {
		return isValidVC(v, c2)
	}
	if c2 == CsNil {
		return isValidCV(c1, v)
	}
	okCV := isValidCV(c1, v)
	okVC := isValidVC(v, c2)
	if okCV && okVC {
		return true
	}
	if !okVC {
		// exceptions: vc fails but cvc passes
		// quyn, quynh
		if c1 == CsQu && v == Vs_y && (c2 == CsN || c2 == CsNh) {
			return true
		}
		// gieng, gie^ng
		if c1 == CsGi && (v == Vs_e || v == Vs_er) && (c2 == CsN || c2 == CsNg) {
			return true
		}
	}
	return false
}

// ----------------------------------------------------------
func (e *Engine) getTonePosition(vs VowelSeq, terminated bool) int {
	info := vSeqList[vs]
	if info.length == 1 {
		return 0
	}
	if info.roofPos != -1 {
		return info.roofPos
	}
	if info.hookPos != -1 {
		if vs == Vs_uhoh || vs == Vs_uhohi || vs == Vs_uhohu { //u+o+, u+o+u, u+o+i
			return 1
		}
		return info.hookPos
	}
	if info.length == 3 {
		return 1
	}
	if e.opts.ModernStyle != 0 &&
		(vs == Vs_oa || vs == Vs_oe || vs == Vs_uy) {
		return 1
	}
	if terminated {
		return 0
	}
	return 1
}

// ----------------------------------------------------------
func (e *Engine) processRoof(ev *keyEvent) int {
	if !e.vietKey || e.current < 0 || e.buffer[e.current].vOffset < 0 {
		return e.processAppend(ev)
	}

	var target VnLexiName
	switch ev.evType {
	case VneRoofA:
		target = Vnl_ar
	case VneRoofE:
		target = Vnl_er
	case VneRoofO:
		target = Vnl_or
	default:
		target = VnlNonVnChar
	}

	var vs, newVs VowelSeq
	var i, vStart, vEnd int
	var curTonePos, newTonePos, tone int
	var changePos int
	roofRemoved := false

	vEnd = e.current - e.buffer[e.current].vOffset
	vs = e.buffer[vEnd].vseq()
	vStart = vEnd - (vSeqList[vs].length - 1)
	curTonePos = vStart + e.getTonePosition(vs, vEnd == e.current)
	tone = e.buffer[curTonePos].tone

	doubleChangeUO := false
	if vs == Vs_uho || vs == Vs_uhoh || vs == Vs_uhoi || vs == Vs_uhohi {
		//special cases: u+o+ -> uo^, u+o -> uo^, u+o+i -> uo^i, u+oi -> uo^i
		newVs = lookupVSeq(Vnl_u, Vnl_or, vSeqList[vs].v[2])
		doubleChangeUO = true
	} else {
		newVs = vSeqList[vs].withRoof
	}

	var pInfo *vowelSeqInfo

	if newVs == VsNil {
		if vSeqList[vs].roofPos == -1 {
			return e.processAppend(ev) //roof is not applicable
		}

		//a roof already exists -> undo roof
		curCh := e.buffer[vStart+vSeqList[vs].roofPos].vnSym
		if target != VnlNonVnChar && curCh != target {
			return e.processAppend(ev) //specific roof and the roof character don't match
		}

		var newCh VnLexiName
		if curCh == Vnl_ar {
			newCh = Vnl_a
		} else if curCh == Vnl_er {
			newCh = Vnl_e
		} else {
			newCh = Vnl_o
		}
		changePos = vStart + vSeqList[vs].roofPos

		if e.opts.FreeMarking == 0 && changePos != e.current {
			return e.processAppend(ev)
		}

		e.markChange(changePos)
		e.buffer[changePos].vnSym = newCh

		if vSeqList[vs].length == 3 {
			newVs = lookupVSeq(e.buffer[vStart].vnSym, e.buffer[vStart+1].vnSym, e.buffer[vStart+2].vnSym)
		} else if vSeqList[vs].length == 2 {
			newVs = lookupVSeq(e.buffer[vStart].vnSym, e.buffer[vStart+1].vnSym, VnlNonVnChar)
		} else {
			newVs = lookupVSeq(e.buffer[vStart].vnSym, VnlNonVnChar, VnlNonVnChar)
		}

		pInfo = &vSeqList[newVs]
		roofRemoved = true
	} else {
		pInfo = &vSeqList[newVs]
		if target != VnlNonVnChar && pInfo.v[pInfo.roofPos] != target {
			return e.processAppend(ev)
		}

		//check validity of new VC and CV
		c1 := CsNil
		c2 := CsNil
		if e.buffer[e.current].c1Offset != -1 {
			c1 = e.buffer[e.current-e.buffer[e.current].c1Offset].cseq()
		}
		if e.buffer[e.current].c2Offset != -1 {
			c2 = e.buffer[e.current-e.buffer[e.current].c2Offset].cseq()
		}
		if !isValidCVC(c1, newVs, c2) {
			return e.processAppend(ev)
		}

		if doubleChangeUO {
			changePos = vStart
		} else {
			changePos = vStart + pInfo.roofPos
		}
		if e.opts.FreeMarking == 0 && changePos != e.current {
			return e.processAppend(ev)
		}
		e.markChange(changePos)
		if doubleChangeUO {
			e.buffer[vStart].vnSym = Vnl_u
			e.buffer[vStart+1].vnSym = Vnl_or
		} else {
			e.buffer[changePos].vnSym = pInfo.v[pInfo.roofPos]
		}
	}

	for i = 0; i < pInfo.length; i++ { //update sub-sequences
		e.buffer[vStart+i].setVseq(pInfo.sub[i])
	}

	//check if tone re-position is needed
	newTonePos = vStart + e.getTonePosition(newVs, vEnd == e.current)
	if curTonePos != newTonePos && tone != 0 {
		e.markChange(newTonePos)
		e.buffer[newTonePos].tone = tone
		e.markChange(curTonePos)
		e.buffer[curTonePos].tone = 0
	}

	if roofRemoved {
		e.singleMode = false
		e.processAppend(ev)
		e.reverted = true
	}

	return 1
}

// ----------------------------------------------------------
// can only be called from processHook
// ----------------------------------------------------------
func (e *Engine) processHookWithUO(ev *keyEvent) int {
	var vs, newVs VowelSeq
	var i, vStart, vEnd int
	var curTonePos, newTonePos, tone int
	hookRemoved := false
	removeWithUndo := true

	if e.opts.FreeMarking == 0 && e.buffer[e.current].vOffset != 0 {
		return e.processAppend(ev)
	}

	vEnd = e.current - e.buffer[e.current].vOffset
	vs = e.buffer[vEnd].vseq()
	vStart = vEnd - (vSeqList[vs].length - 1)
	v := vSeqList[vs].v
	curTonePos = vStart + e.getTonePosition(vs, vEnd == e.current)
	tone = e.buffer[curTonePos].tone

	switch ev.evType {
	case VneHookU:
		if v[0] == Vnl_u {
			newVs = vSeqList[vs].withHook
			e.markChange(vStart)
			e.buffer[vStart].vnSym = Vnl_uh
		} else { // v[0] = vnl_uh, -> uo
			newVs = lookupVSeq(Vnl_u, Vnl_o, v[2])
			e.markChange(vStart)
			e.buffer[vStart].vnSym = Vnl_u
			e.buffer[vStart+1].vnSym = Vnl_o
			hookRemoved = true
		}
	case VneHookO:
		if v[1] == Vnl_o || v[1] == Vnl_or {
			if vEnd == e.current && vSeqList[vs].length == 2 &&
				e.buffer[e.current].form == VnwCV && e.buffer[e.current-2].cseq() == CsTh {
				// o|o^ -> o+
				newVs = vSeqList[vs].withHook
				e.markChange(vStart + 1)
				e.buffer[vStart+1].vnSym = Vnl_oh
			} else {
				newVs = lookupVSeq(Vnl_uh, Vnl_oh, v[2])
				if v[0] == Vnl_u {
					e.markChange(vStart)
					e.buffer[vStart].vnSym = Vnl_uh
					e.buffer[vStart+1].vnSym = Vnl_oh
				} else {
					e.markChange(vStart + 1)
					e.buffer[vStart+1].vnSym = Vnl_oh
				}
			}
		} else { // v[1] = vnl_oh, -> uo
			newVs = lookupVSeq(Vnl_u, Vnl_o, v[2])
			if v[0] == Vnl_uh {
				e.markChange(vStart)
				e.buffer[vStart].vnSym = Vnl_u
				e.buffer[vStart+1].vnSym = Vnl_o
			} else {
				e.markChange(vStart + 1)
				e.buffer[vStart+1].vnSym = Vnl_o
			}
			hookRemoved = true
		}
	default: //vneHookAll, vneHookUO:
		if v[0] == Vnl_u {
			if v[1] == Vnl_o || v[1] == Vnl_or {
				//uo -> uo+ if prefixed by "th"
				if (vs == Vs_uo || vs == Vs_uor) && vEnd == e.current &&
					e.buffer[e.current].form == VnwCV && e.buffer[e.current-2].cseq() == CsTh {
					newVs = Vs_uoh
					e.markChange(vStart + 1)
					e.buffer[vStart+1].vnSym = Vnl_oh
				} else {
					//uo -> u+o+
					newVs = vSeqList[vs].withHook
					e.markChange(vStart)
					e.buffer[vStart].vnSym = Vnl_uh
					newVs = vSeqList[newVs].withHook
					e.buffer[vStart+1].vnSym = Vnl_oh
				}
			} else { //uo+ -> u+o+
				newVs = vSeqList[vs].withHook
				e.markChange(vStart)
				e.buffer[vStart].vnSym = Vnl_uh
			}
		} else { //v[0] == vnl_uh
			if v[1] == Vnl_o { // u+o -> u+o+
				newVs = vSeqList[vs].withHook
				e.markChange(vStart + 1)
				e.buffer[vStart+1].vnSym = Vnl_oh
			} else { //v[1] == vnl_oh, u+o+ -> uo
				newVs = lookupVSeq(Vnl_u, Vnl_o, v[2])
				e.markChange(vStart)
				e.buffer[vStart].vnSym = Vnl_u
				e.buffer[vStart+1].vnSym = Vnl_o
				hookRemoved = true
			}
		}
	}

	p := &vSeqList[newVs]
	for i = 0; i < p.length; i++ { //update sub-sequences
		e.buffer[vStart+i].setVseq(p.sub[i])
	}

	//check if tone re-position is needed
	newTonePos = vStart + e.getTonePosition(newVs, vEnd == e.current)
	if curTonePos != newTonePos && tone != 0 {
		e.markChange(newTonePos)
		e.buffer[newTonePos].tone = tone
		e.markChange(curTonePos)
		e.buffer[curTonePos].tone = 0
	}

	if hookRemoved && removeWithUndo {
		e.singleMode = false
		e.processAppend(ev)
		e.reverted = true
	}

	return 1
}

// ----------------------------------------------------------
func (e *Engine) processHook(ev *keyEvent) int {
	if !e.vietKey || e.current < 0 || e.buffer[e.current].vOffset < 0 {
		return e.processAppend(ev)
	}

	var vs, newVs VowelSeq
	var i, vStart, vEnd int
	var curTonePos, newTonePos, tone int
	var changePos int
	hookRemoved := false
	var pInfo *vowelSeqInfo

	vEnd = e.current - e.buffer[e.current].vOffset
	vs = e.buffer[vEnd].vseq()

	v := vSeqList[vs].v

	if vSeqList[vs].length > 1 &&
		ev.evType != VneBowl &&
		(v[0] == Vnl_u || v[0] == Vnl_uh) &&
		(v[1] == Vnl_o || v[1] == Vnl_oh || v[1] == Vnl_or) {
		return e.processHookWithUO(ev)
	}

	vStart = vEnd - (vSeqList[vs].length - 1)
	curTonePos = vStart + e.getTonePosition(vs, vEnd == e.current)
	tone = e.buffer[curTonePos].tone

	newVs = vSeqList[vs].withHook
	if newVs == VsNil {
		if vSeqList[vs].hookPos == -1 {
			return e.processAppend(ev) //hook is not applicable
		}

		//a hook already exists -> undo hook
		curCh := e.buffer[vStart+vSeqList[vs].hookPos].vnSym
		var newCh VnLexiName
		if curCh == Vnl_ab {
			newCh = Vnl_a
		} else if curCh == Vnl_uh {
			newCh = Vnl_u
		} else {
			newCh = Vnl_o
		}
		changePos = vStart + vSeqList[vs].hookPos
		if e.opts.FreeMarking == 0 && changePos != e.current {
			return e.processAppend(ev)
		}

		switch ev.evType {
		case VneHookU:
			if curCh != Vnl_uh {
				return e.processAppend(ev)
			}
		case VneHookO:
			if curCh != Vnl_oh {
				return e.processAppend(ev)
			}
		case VneBowl:
			if curCh != Vnl_ab {
				return e.processAppend(ev)
			}
		default:
			if ev.evType == VneHookUO && curCh == Vnl_ab {
				return e.processAppend(ev)
			}
		}

		e.markChange(changePos)
		e.buffer[changePos].vnSym = newCh

		if vSeqList[vs].length == 3 {
			newVs = lookupVSeq(e.buffer[vStart].vnSym, e.buffer[vStart+1].vnSym, e.buffer[vStart+2].vnSym)
		} else if vSeqList[vs].length == 2 {
			newVs = lookupVSeq(e.buffer[vStart].vnSym, e.buffer[vStart+1].vnSym, VnlNonVnChar)
		} else {
			newVs = lookupVSeq(e.buffer[vStart].vnSym, VnlNonVnChar, VnlNonVnChar)
		}

		pInfo = &vSeqList[newVs]
		hookRemoved = true
	} else {
		pInfo = &vSeqList[newVs]

		switch ev.evType {
		case VneHookU:
			if pInfo.v[pInfo.hookPos] != Vnl_uh {
				return e.processAppend(ev)
			}
		case VneHookO:
			if pInfo.v[pInfo.hookPos] != Vnl_oh {
				return e.processAppend(ev)
			}
		case VneBowl:
			if pInfo.v[pInfo.hookPos] != Vnl_ab {
				return e.processAppend(ev)
			}
		default: //vneHook_uo, vneHookAll
			if ev.evType == VneHookUO && pInfo.v[pInfo.hookPos] == Vnl_ab {
				return e.processAppend(ev)
			}
		}

		//check validity of new VC and CV
		c1 := CsNil
		c2 := CsNil
		if e.buffer[e.current].c1Offset != -1 {
			c1 = e.buffer[e.current-e.buffer[e.current].c1Offset].cseq()
		}
		if e.buffer[e.current].c2Offset != -1 {
			c2 = e.buffer[e.current-e.buffer[e.current].c2Offset].cseq()
		}
		if !isValidCVC(c1, newVs, c2) {
			return e.processAppend(ev)
		}

		changePos = vStart + pInfo.hookPos
		if e.opts.FreeMarking == 0 && changePos != e.current {
			return e.processAppend(ev)
		}

		e.markChange(changePos)
		e.buffer[changePos].vnSym = pInfo.v[pInfo.hookPos]
	}

	for i = 0; i < pInfo.length; i++ { //update sub-sequences
		e.buffer[vStart+i].setVseq(pInfo.sub[i])
	}

	//check if tone re-position is needed
	newTonePos = vStart + e.getTonePosition(newVs, vEnd == e.current)
	if curTonePos != newTonePos && tone != 0 {
		e.markChange(newTonePos)
		e.buffer[newTonePos].tone = tone
		e.markChange(curTonePos)
		e.buffer[curTonePos].tone = 0
	}

	if hookRemoved {
		e.singleMode = false
		e.processAppend(ev)
		e.reverted = true
	}

	return 1
}

// ----------------------------------------------------------
func (e *Engine) processTone(ev *keyEvent) int {
	if e.current < 0 || !e.vietKey {
		return e.processAppend(ev)
	}

	if e.buffer[e.current].form == VnwC &&
		(e.buffer[e.current].cseq() == CsGi || e.buffer[e.current].cseq() == CsGin) {
		var p int
		if e.buffer[e.current].cseq() == CsGi {
			p = e.current
		} else {
			p = e.current - 1
		}
		if e.buffer[p].tone == 0 && ev.tone == 0 {
			return e.processAppend(ev)
		}
		e.markChange(p)
		if e.buffer[p].tone == ev.tone {
			e.buffer[p].tone = 0
			e.singleMode = false
			e.processAppend(ev)
			e.reverted = true
			return 1
		}
		e.buffer[p].tone = ev.tone
		return 1
	}

	if e.buffer[e.current].vOffset < 0 {
		return e.processAppend(ev)
	}

	var vEnd int
	var vs VowelSeq

	vEnd = e.current - e.buffer[e.current].vOffset
	vs = e.buffer[vEnd].vseq()
	info := vSeqList[vs]
	if e.opts.SpellCheckEnabled != 0 && e.opts.FreeMarking == 0 && info.complete == 0 {
		return e.processAppend(ev)
	}

	if e.buffer[e.current].form == VnwVC || e.buffer[e.current].form == VnwCVC {
		cs := e.buffer[e.current].cseq()
		if (cs == CsC || cs == CsCh || cs == CsP || cs == CsT) &&
			(ev.tone == 2 || ev.tone == 3 || ev.tone == 4) {
			return e.processAppend(ev) // c, ch, p, t suffixes don't allow ` ? ~
		}
	}

	toneOffset := e.getTonePosition(vs, vEnd == e.current)
	tonePos := vEnd - (info.length - 1) + toneOffset
	if e.buffer[tonePos].tone == 0 && ev.tone == 0 {
		return e.processAppend(ev)
	}

	if e.buffer[tonePos].tone == ev.tone {
		e.markChange(tonePos)
		e.buffer[tonePos].tone = 0
		e.singleMode = false
		e.processAppend(ev)
		e.reverted = true
		return 1
	}

	e.markChange(tonePos)
	e.buffer[tonePos].tone = ev.tone
	return 1
}

// ----------------------------------------------------------
func (e *Engine) processDd(ev *keyEvent) int {
	if !e.vietKey || e.current < 0 {
		return e.processAppend(ev)
	}

	var pos int

	// we want to allow dd even in non-vn sequence, because dd is used a lot in abbreviation
	// we allow dd only if preceding character is not a vowel
	if e.buffer[e.current].form == VnwNonVn &&
		e.buffer[e.current].vnSym == Vnl_d &&
		(e.current-1 < 0 || e.buffer[e.current-1].vnSym == VnlNonVnChar || !isVnVowel(e.buffer[e.current-1].vnSym)) {
		e.singleMode = true
		pos = e.current
		e.markChange(pos)
		e.buffer[pos].setCseq(CsDd)
		e.buffer[pos].vnSym = Vnl_dd
		e.buffer[pos].form = VnwC
		e.buffer[pos].c1Offset = 0
		e.buffer[pos].c2Offset = -1
		e.buffer[pos].vOffset = -1
		return 1
	}

	if e.buffer[e.current].c1Offset < 0 {
		return e.processAppend(ev)
	}

	pos = e.current - e.buffer[e.current].c1Offset
	if e.opts.FreeMarking == 0 && pos != e.current {
		return e.processAppend(ev)
	}

	if e.buffer[pos].cseq() == CsD {
		e.markChange(pos)
		e.buffer[pos].setCseq(CsDd)
		e.buffer[pos].vnSym = Vnl_dd
		return 1
	}

	if e.buffer[pos].cseq() == CsDd {
		//undo dd
		e.markChange(pos)
		e.buffer[pos].setCseq(CsD)
		e.buffer[pos].vnSym = Vnl_d
		e.singleMode = false
		e.processAppend(ev)
		e.reverted = true
		return 1
	}

	return e.processAppend(ev)
}

// ----------------------------------------------------------
func (e *Engine) processMapChar(ev *keyEvent) int {
	if e.CapsLockOn {
		ev.vnSym = changeCase(ev.vnSym)
	}

	ret := e.processAppend(ev)
	if !e.vietKey {
		return ret
	}

	if e.current >= 0 && e.buffer[e.current].form != VnwEmpty &&
		e.buffer[e.current].form != VnwNonVn {
		return 1
	}

	if e.current < 0 {
		return 0
	}

	// mapChar doesn't apply
	e.current--
	entry := &e.buffer[e.current]

	undo := false
	// test if undo is needed
	if entry.form != VnwEmpty && entry.form != VnwNonVn {
		prevSym := entry.vnSym
		if entry.caps != 0 {
			prevSym = prevSym - 1
		}
		if prevSym == ev.vnSym {
			if entry.form != VnwC {
				var vStart, vEnd, curTonePos, newTonePos, tone int
				var vs, newVs VowelSeq

				vEnd = e.current - entry.vOffset
				vs = e.buffer[vEnd].vseq()
				vStart = vEnd - vSeqList[vs].length + 1
				curTonePos = vStart + e.getTonePosition(vs, vEnd == e.current)
				tone = e.buffer[curTonePos].tone
				e.markChange(e.current)
				e.current--

				//check if tone position is needed
				if tone != 0 && e.current >= 0 &&
					(e.buffer[e.current].form == VnwV || e.buffer[e.current].form == VnwCV) {
					newVs = e.buffer[e.current].vseq()
					newTonePos = vStart + e.getTonePosition(newVs, true)
					if newTonePos != curTonePos {
						e.markChange(newTonePos)
						e.buffer[newTonePos].tone = tone
						e.markChange(curTonePos)
						e.buffer[curTonePos].tone = 0
					}
				}
			} else {
				e.markChange(e.current)
				e.current--
			}
			undo = true
		}
	}

	ev.evType = VneNormal
	ev.chType = e.input.getCharType(ev.keyCode)
	ev.vnSym = isoToVnLexi(ev.keyCode)
	ret = e.processAppend(ev)
	if undo {
		e.singleMode = false
		e.reverted = true
		return 1
	}
	return ret
}

// ----------------------------------------------------------
func (e *Engine) processTelexW(ev *keyEvent) int {
	if !e.vietKey {
		return e.processAppend(ev)
	}

	var ret int

	if e.usedAsMapChar {
		ev.evType = VneMapChar
		if isUpper(byte(ev.keyCode)) {
			ev.vnSym = VnlUh
		} else {
			ev.vnSym = Vnl_uh
		}
		if e.CapsLockOn {
			ev.vnSym = changeCase(ev.vnSym)
		}
		ev.chType = UkcVn
		ret = e.processMapChar(ev)
		if ret == 0 {
			if e.current >= 0 {
				e.current--
			}
			e.usedAsMapChar = false
			ev.evType = VneHookAll
			return e.processHook(ev)
		}
		return ret
	}

	ev.evType = VneHookAll
	e.usedAsMapChar = false
	ret = e.processHook(ev)
	if ret == 0 {
		if e.current >= 0 {
			e.current--
		}
		ev.evType = VneMapChar
		if isUpper(byte(ev.keyCode)) {
			ev.vnSym = VnlUh
		} else {
			ev.vnSym = Vnl_uh
		}
		if e.CapsLockOn {
			ev.vnSym = changeCase(ev.vnSym)
		}
		ev.chType = UkcVn
		e.usedAsMapChar = true
		return e.processMapChar(ev)
	}
	return ret
}

// ----------------------------------------------------------
func (e *Engine) processAppend(ev *keyEvent) int {
	switch ev.chType {
	case UkcReset:
		if ev.keyCode == EnterChar {
			if e.opts.MacroEnabled != 0 && e.macroMatch(ev) {
				return 1
			}
		}
		e.reset()
		return 0
	case UkcWordBreak:
		e.singleMode = false
		return e.processWordEnd(ev)
	case UkcNonVn:
		e.current++
		entry := &e.buffer[e.current]
		if ev.chType == UkcWordBreak {
			entry.form = VnwEmpty
		} else {
			entry.form = VnwNonVn
		}
		entry.c1Offset = -1
		entry.c2Offset = -1
		entry.vOffset = -1
		entry.keyCode = ev.keyCode
		entry.vnSym = vnToLower(ev.vnSym)
		entry.tone = 0
		if entry.vnSym != ev.vnSym {
			entry.caps = 1
		} else {
			entry.caps = 0
		}
		return 0
	case UkcVn:
		if isVnVowel(ev.vnSym) {
			v := VnLexiName(stdVnNoTone[vnToLower(ev.vnSym)])
			if e.current >= 0 && e.buffer[e.current].form == VnwC &&
				((e.buffer[e.current].cseq() == CsQ && v == Vnl_u) ||
					(e.buffer[e.current].cseq() == CsG && v == Vnl_i)) {
				return e.appendConsonnant(ev) //process u after q, i after g as consonnants
			}
			return e.appendVowel(ev)
		}
		return e.appendConsonnant(ev)
	}
	return 0
}

// ----------------------------------------------------------
func (e *Engine) appendVowel(ev *keyEvent) int {
	autoCompleted := false
	_ = autoCompleted

	e.current++
	entry := &e.buffer[e.current]

	lowerSym := vnToLower(ev.vnSym)
	canSym := VnLexiName(stdVnNoTone[lowerSym])

	entry.vnSym = canSym
	if lowerSym != ev.vnSym {
		entry.caps = 1
	} else {
		entry.caps = 0
	}
	entry.tone = int(lowerSym-canSym) / 2
	entry.keyCode = ev.keyCode

	if e.current == 0 || !e.vietKey {
		entry.form = VnwV
		entry.c1Offset = -1
		entry.c2Offset = -1
		entry.vOffset = 0
		entry.setVseq(lookupVSeq(canSym, VnlNonVnChar, VnlNonVnChar))

		if !e.vietKey || isAlpha(entry.keyCode) {
			return 0
		}
		e.markChange(e.current)
		return 1
	}

	prev := &e.buffer[e.current-1]
	var vs, newVs VowelSeq
	var cs ConSeq
	var prevTonePos int
	var tone, newTone, tonePos, newTonePos int

	switch prev.form {
	case VnwEmpty:
		entry.form = VnwV
		entry.c1Offset = -1
		entry.c2Offset = -1
		entry.vOffset = 0
		newVs = lookupVSeq(canSym, VnlNonVnChar, VnlNonVnChar)
		entry.setVseq(newVs)

	case VnwNonVn, VnwCVC, VnwVC:
		entry.form = VnwNonVn
		entry.c1Offset = -1
		entry.c2Offset = -1
		entry.vOffset = -1

	case VnwV, VnwCV:
		vs = prev.vseq()
		prevTonePos = (e.current - 1) - (vSeqList[vs].length - 1) + e.getTonePosition(vs, true)
		tone = e.buffer[prevTonePos].tone

		if lowerSym != canSym && tone != 0 { //new sym has a tone, but there's already a preceding tone
			newVs = VsNil
		} else {
			if vSeqList[vs].length == 3 {
				newVs = VsNil
			} else if vSeqList[vs].length == 2 {
				newVs = lookupVSeq(vSeqList[vs].v[0], vSeqList[vs].v[1], canSym)
			} else {
				newVs = lookupVSeq(vSeqList[vs].v[0], canSym, VnlNonVnChar)
			}
		}

		if newVs != VsNil && prev.form == VnwCV {
			cs = e.buffer[e.current-1-prev.c1Offset].cseq()
			if !isValidCV(cs, newVs) {
				newVs = VsNil
			}
		}

		if newVs == VsNil {
			entry.form = VnwNonVn
			entry.c1Offset = -1
			entry.c2Offset = -1
			entry.vOffset = -1
			break
		}

		entry.form = prev.form
		if prev.form == VnwCV {
			entry.c1Offset = prev.c1Offset + 1
		} else {
			entry.c1Offset = -1
		}
		entry.c2Offset = -1
		entry.vOffset = 0
		entry.setVseq(newVs)
		entry.tone = 0

		newTone = int(lowerSym-canSym) / 2
		if tone == 0 {
			if newTone != 0 {
				tone = newTone
				tonePos = e.getTonePosition(newVs, true) + ((e.current - 1) - vSeqList[vs].length + 1)
				e.markChange(tonePos)
				e.buffer[tonePos].tone = tone
				return 1
			}
		} else {
			newTonePos = e.getTonePosition(newVs, true) + ((e.current - 1) - vSeqList[vs].length + 1)
			if newTonePos != prevTonePos {
				e.markChange(prevTonePos)
				e.buffer[prevTonePos].tone = 0
				e.markChange(newTonePos)
				if newTone != 0 {
					tone = newTone
				}
				e.buffer[newTonePos].tone = tone
				return 1
			}
			if newTone != 0 && newTone != tone {
				tone = newTone
				e.markChange(prevTonePos)
				e.buffer[prevTonePos].tone = tone
				return 1
			}
		}

	case VnwC:
		newVs = lookupVSeq(canSym, VnlNonVnChar, VnlNonVnChar)
		cs = prev.cseq()
		if !isValidCV(cs, newVs) {
			entry.form = VnwNonVn
			entry.c1Offset = -1
			entry.c2Offset = -1
			entry.vOffset = -1
			break
		}

		entry.form = VnwCV
		entry.c1Offset = 1
		entry.c2Offset = -1
		entry.vOffset = 0
		entry.setVseq(newVs)

		if cs == CsGi && prev.tone != 0 {
			if entry.tone == 0 {
				entry.tone = prev.tone
			}
			e.markChange(e.current - 1)
			prev.tone = 0
			return 1
		}
	}

	if isAlpha(entry.keyCode) {
		return 0
	}
	e.markChange(e.current)
	return 1
}

// ----------------------------------------------------------
func (e *Engine) appendConsonnant(ev *keyEvent) int {
	complexEvent := false
	e.current++
	entry := &e.buffer[e.current]

	lowerSym := vnToLower(ev.vnSym)

	entry.vnSym = lowerSym
	if lowerSym != ev.vnSym {
		entry.caps = 1
	} else {
		entry.caps = 0
	}
	entry.keyCode = ev.keyCode
	entry.tone = 0

	if e.current == 0 || !e.vietKey {
		entry.form = VnwC
		entry.c1Offset = 0
		entry.c2Offset = -1
		entry.vOffset = -1
		entry.setCseq(lookupCSeq(lowerSym, VnlNonVnChar, VnlNonVnChar))
		return 0
	}

	var cs, newCs, c1 ConSeq
	var vs, newVs VowelSeq
	var isValid bool

	prev := &e.buffer[e.current-1]

	switch prev.form {
	case VnwNonVn:
		entry.form = VnwNonVn
		entry.c1Offset = -1
		entry.c2Offset = -1
		entry.vOffset = -1
		return 0
	case VnwEmpty:
		entry.form = VnwC
		entry.c1Offset = 0
		entry.c2Offset = -1
		entry.vOffset = -1
		entry.setCseq(lookupCSeq(lowerSym, VnlNonVnChar, VnlNonVnChar))
		return 0
	case VnwV, VnwCV:
		vs = prev.vseq()
		newVs = vs
		if vs == Vs_uoh || vs == Vs_uho {
			newVs = Vs_uhoh
		}

		c1 = CsNil
		if prev.c1Offset != -1 {
			c1 = e.buffer[e.current-1-prev.c1Offset].cseq()
		}

		newCs = lookupCSeq(lowerSym, VnlNonVnChar, VnlNonVnChar)
		isValid = isValidCVC(c1, newVs, newCs)

		if isValid {
			//check u+o -> u+o+
			if vs == Vs_uho {
				e.markChange(e.current - 1)
				prev.vnSym = Vnl_oh
				prev.setVseq(Vs_uhoh)
				complexEvent = true
			} else if vs == Vs_uoh {
				e.markChange(e.current - 2)
				e.buffer[e.current-2].vnSym = Vnl_uh
				e.buffer[e.current-2].setVseq(Vs_uh)
				prev.setVseq(Vs_uhoh)
				complexEvent = true
			}

			if prev.form == VnwV {
				entry.form = VnwVC
				entry.c1Offset = -1
				entry.c2Offset = 0
				entry.vOffset = 1
			} else { //prev == vnw_cv
				entry.form = VnwCVC
				entry.c1Offset = prev.c1Offset + 1
				entry.c2Offset = 0
				entry.vOffset = 1
			}
			entry.setCseq(newCs)

			//reposition tone if needed
			oldIdx := (e.current - 1) - (vSeqList[vs].length - 1) + e.getTonePosition(vs, true)
			if e.buffer[oldIdx].tone != 0 {
				newIdx := (e.current - 1) - (vSeqList[newVs].length - 1) + e.getTonePosition(newVs, false)
				if newIdx != oldIdx {
					e.markChange(newIdx)
					e.buffer[newIdx].tone = e.buffer[oldIdx].tone
					e.markChange(oldIdx)
					e.buffer[oldIdx].tone = 0
					return 1
				}
			}
		} else {
			entry.form = VnwNonVn
			entry.c1Offset = -1
			entry.c2Offset = -1
			entry.vOffset = -1
		}

		if complexEvent {
			return 1
		}
		return 0
	case VnwC, VnwVC, VnwCVC:
		cs = prev.cseq()
		if cSeqList[cs].length == 3 {
			newCs = CsNil
		} else if cSeqList[cs].length == 2 {
			newCs = lookupCSeq(cSeqList[cs].c[0], cSeqList[cs].c[1], lowerSym)
		} else {
			newCs = lookupCSeq(cSeqList[cs].c[0], lowerSym, VnlNonVnChar)
		}

		if newCs != CsNil && (prev.form == VnwVC || prev.form == VnwCVC) {
			// Check CVC combination
			c1 = CsNil
			if prev.c1Offset != -1 {
				c1 = e.buffer[e.current-1-prev.c1Offset].cseq()
			}

			vIdx := (e.current - 1) - prev.vOffset
			vs = e.buffer[vIdx].vseq()
			isValid = isValidCVC(c1, vs, newCs)
			if !isValid {
				newCs = CsNil
			}
		}

		if newCs == CsNil {
			entry.form = VnwNonVn
			entry.c1Offset = -1
			entry.c2Offset = -1
			entry.vOffset = -1
		} else {
			if prev.form == VnwC {
				entry.form = VnwC
				entry.c1Offset = 0
				entry.c2Offset = -1
				entry.vOffset = -1
			} else if prev.form == VnwVC {
				entry.form = VnwVC
				entry.c1Offset = -1
				entry.c2Offset = 0
				entry.vOffset = prev.vOffset + 1
			} else { //vnw_cvc
				entry.form = VnwCVC
				entry.c1Offset = prev.c1Offset + 1
				entry.c2Offset = 0
				entry.vOffset = prev.vOffset + 1
			}
			entry.setCseq(newCs)
		}
		return 0
	}
	return 0
}

// ----------------------------------------------------------
func (e *Engine) processEscChar(ev *keyEvent) int {
	if e.vietKey &&
		e.current >= 0 && e.buffer[e.current].form != VnwEmpty && e.buffer[e.current].form != VnwNonVn {
		e.toEscape = true
	}
	return e.processAppend(ev)
}

// ----------------------------------------------------------
// This can be called only after other processing have been done.
// The new event is supposed to be put into m_buffer already
// ----------------------------------------------------------
func (e *Engine) processNoSpellCheck(ev *keyEvent) int {
	entry := &e.buffer[e.current]
	if isVnVowel(entry.vnSym) {
		entry.form = VnwV
		entry.vOffset = 0
		entry.setVseq(lookupVSeq(entry.vnSym, VnlNonVnChar, VnlNonVnChar))
		entry.c1Offset = -1
		entry.c2Offset = -1
	} else {
		entry.form = VnwC
		entry.c1Offset = 0
		entry.c2Offset = -1
		entry.vOffset = -1
		entry.setCseq(lookupCSeq(entry.vnSym, VnlNonVnChar, VnlNonVnChar))
	}

	if ev.evType == VneNormal &&
		((entry.keyCode >= 'a' && entry.keyCode <= 'z') ||
			(entry.keyCode >= 'A' && entry.keyCode <= 'Z')) {
		return 0
	}
	e.markChange(e.current)
	return 1
}

// ----------------------------------------------------------
// main entry point
// ----------------------------------------------------------
func (e *Engine) process(keyCode uint32) Result {
	var ev keyEvent
	e.prepareBuffer()
	e.backs = 0
	e.changePos = e.current + 1
	e.out = e.out[:0]
	e.outputWritten = false
	e.reverted = false
	e.keyRestored = false
	e.keyRestoring = false
	e.outType = OutChar

	ev = e.input.keyCodeToEvent(keyCode)

	var ret int
	if !e.toEscape {
		ret = e.dispatch(&ev)
	} else {
		e.toEscape = false
		if e.current < 0 || ev.evType == VneNormal || ev.evType == VneEscChar {
			ret = e.processAppend(&ev)
		} else {
			e.current--
			e.processAppend(&ev)
			e.markChange(e.current) //this will assign m_backs to 1 and mark the character for output
			ret = 1
		}
	}

	if e.vietKey &&
		e.current >= 0 && e.buffer[e.current].form == VnwNonVn &&
		ev.chType == UkcVn &&
		(e.opts.SpellCheckEnabled == 0 || e.singleMode) {
		//The spell check has failed, but because we are in non-spellcheck mode,
		//we consider the new character as the beginning of a new word
		ret = e.processNoSpellCheck(&ev)
	}

	//we add key to key buffer only if that key has not caused a reset
	if e.current >= 0 {
		ev.chType = e.input.getCharType(ev.keyCode)
		e.keyCurrent++
		if e.keyCurrent < MaxUkEngine {
			e.keyStrokes[e.keyCurrent].ev = ev
			e.keyStrokes[e.keyCurrent].converted = ret != 0 && !e.keyRestored
		}
	}

	res := Result{KeyOut: e.outType == OutKey}
	if ret == 0 {
		return res
	}

	res.Backs = e.backs
	if !e.outputWritten {
		e.writeOutput()
	}
	res.Out = append([]uint16(nil), e.out...)
	res.Consumed = true
	return res
}

func (e *Engine) dispatch(ev *keyEvent) int {
	switch ev.evType {
	case VneRoofAll, VneRoofA, VneRoofE, VneRoofO:
		return e.processRoof(ev)
	case VneHookAll, VneHookUO, VneHookU, VneHookO, VneBowl:
		return e.processHook(ev)
	case VneDd:
		return e.processDd(ev)
	case VneTone0, VneTone1, VneTone2, VneTone3, VneTone4, VneTone5:
		return e.processTone(ev)
	case VneTelexW:
		return e.processTelexW(ev)
	case VneMapChar:
		return e.processMapChar(ev)
	case VneEscChar:
		return e.processEscChar(ev)
	default: //vneNormal
		return e.processAppend(ev)
	}
}

// ----------------------------------------------------------
func (e *Engine) synchKeyStrokeBuffer() {
	//synchronize with key-stroke buffer
	if e.keyCurrent >= 0 {
		e.keyCurrent--
	}
	if e.current >= 0 && e.buffer[e.current].form == VnwEmpty {
		//in character buffer, we have reached a word break,
		//so we also need to move key stroke pointer backward to corresponding word break
		for e.keyCurrent >= 0 && e.keyStrokes[e.keyCurrent].ev.chType != UkcWordBreak {
			e.keyCurrent--
		}
	}
}

// ----------------------------------------------------------
func (e *Engine) processBackspace() Result {
	res := Result{KeyOut: false}
	e.out = e.out[:0]
	if !e.vietKey || e.current < 0 {
		return res
	}

	e.backs = 0
	e.changePos = e.current + 1
	e.markChange(e.current)

	if e.current == 0 ||
		e.buffer[e.current].form == VnwEmpty ||
		e.buffer[e.current].form == VnwNonVn ||
		e.buffer[e.current].form == VnwC ||
		e.buffer[e.current-1].form == VnwC ||
		e.buffer[e.current-1].form == VnwCVC ||
		e.buffer[e.current-1].form == VnwVC {
		e.current--
		res.Backs = e.backs
		e.synchKeyStrokeBuffer()
		res.Consumed = e.backs > 1
		return res
	}

	var vs, newVs VowelSeq
	var curTonePos, newTonePos, tone, vStart, vEnd int

	vEnd = e.current - e.buffer[e.current].vOffset
	vs = e.buffer[vEnd].vseq()
	vStart = vEnd - vSeqList[vs].length + 1
	newVs = e.buffer[e.current-1].vseq()
	curTonePos = vStart + e.getTonePosition(vs, vEnd == e.current)
	newTonePos = vStart + e.getTonePosition(newVs, true)
	tone = e.buffer[curTonePos].tone

	if tone == 0 || curTonePos == newTonePos ||
		(curTonePos == e.current && e.buffer[e.current].tone != 0) {
		e.current--
		res.Backs = e.backs
		e.synchKeyStrokeBuffer()
		res.Consumed = e.backs > 1
		return res
	}

	e.markChange(newTonePos)
	e.buffer[newTonePos].tone = tone
	e.markChange(curTonePos)
	e.buffer[curTonePos].tone = 0
	e.current--
	e.synchKeyStrokeBuffer()
	res.Backs = e.backs
	e.writeOutput()
	res.Out = append([]uint16(nil), e.out...)
	res.Consumed = true
	return res
}

// ----------------------------------------------------
// make sure there are at least 10 entries available
// ----------------------------------------------------
func (e *Engine) prepareBuffer() {
	var rid int
	//prepare symbol buffer
	if e.current >= 0 && e.current+10 >= e.bufSize {
		// Get rid of at least half of the current entries
		// don't get rid from the middle of a word.
		for rid = e.current / 2; e.buffer[rid].form != VnwEmpty && rid < e.current; rid++ {
		}
		if rid == e.current {
			e.current = -1
		} else {
			rid++
			copy(e.buffer[:], e.buffer[rid:e.current+1])
			e.current -= rid
		}
	}

	//prepare key stroke buffer
	if e.keyCurrent > 0 && e.keyCurrent+1 >= e.keyBufSize {
		// Get rid of at least half of the current entries
		rid = e.keyCurrent / 2
		copy(e.keyStrokes[:], e.keyStrokes[rid:e.keyCurrent+1])
		e.keyCurrent -= rid
	}
}

// ----------------------------------------------------
func (e *Engine) macroMatch(ev *keyEvent) bool {
	if e.ShiftPressed && (ev.keyCode == ' ' || ev.keyCode == EnterChar) {
		return false
	}

	var pMacText []StdVnChar
	var key [MaxMacroKeyLen + 1]StdVnChar
	var i, j int

	i = e.current
	for i >= 0 && (e.current-i+1) < MaxMacroKeyLen {
		for i >= 0 && e.buffer[i].form != VnwEmpty && (e.current-i+1) < MaxMacroKeyLen {
			i--
		}
		if i >= 0 && e.buffer[i].form != VnwEmpty {
			return false
		}

		if i >= 0 {
			if e.buffer[i].vnSym != VnlNonVnChar {
				key[0] = StdVnChar(e.buffer[i].vnSym) + VnStdCharOffset
				if e.buffer[i].caps != 0 {
					key[0]--
				}
				key[0] += StdVnChar(e.buffer[i].tone * 2)
			} else {
				key[0] = StdVnChar(e.buffer[i].keyCode)
			}
		}

		for j = i + 1; j <= e.current; j++ {
			if e.buffer[j].vnSym != VnlNonVnChar {
				key[j-i] = StdVnChar(e.buffer[j].vnSym) + VnStdCharOffset
				if e.buffer[j].caps != 0 {
					key[j-i]--
				}
				key[j-i] += StdVnChar(e.buffer[j].tone * 2)
			} else {
				key[j-i] = StdVnChar(e.buffer[j].keyCode)
			}
		}
		keyLen := e.current - i + 1
		//search macro table
		if pMacText = e.macStore.lookup(key[1:keyLen]); pMacText != nil {
			i++ //mark the position where change is needed
			break
		}
		if i >= 0 {
			if pMacText = e.macStore.lookup(key[0:keyLen]); pMacText != nil {
				break
			}
		}
		i--
	}

	if pMacText == nil {
		return false
	}

	e.markChange(i)
	e.out = e.out[:0]
	for _, sc := range pMacText {
		if c, ok := stdCharToUTF16(sc); ok {
			e.out = append(e.out, c)
		}
	}

	//write the last input character
	var vnChar StdVnChar
	if ev.vnSym != VnlNonVnChar {
		vnChar = StdVnChar(ev.vnSym) + VnStdCharOffset
	} else {
		vnChar = StdVnChar(ev.keyCode)
	}
	if c, ok := stdCharToUTF16(vnChar); ok {
		e.out = append(e.out, c)
	}
	backs := e.backs //store backs before calling reset
	e.reset()
	e.outputWritten = true
	e.backs = backs
	return true
}

// ----------------------------------------------------
func (e *Engine) restoreKeyStrokes() bool {
	e.outType = OutKey
	if !e.lastWordHasVnMark() {
		return false
	}

	e.backs = 0
	e.changePos = e.current + 1

	var keyStart int
	converted := false
	for keyStart = e.keyCurrent; keyStart >= 0 && e.keyStrokes[keyStart].ev.chType != UkcWordBreak; keyStart-- {
		if e.keyStrokes[keyStart].converted {
			converted = true
		}
	}
	keyStart++

	if !converted {
		//no key stroke has been converted, so it doesn't make sense to restore key strokes
		return false
	}

	for e.current >= 0 && e.buffer[e.current].form != VnwEmpty {
		e.current--
	}
	e.markChange(e.current + 1)

	e.out = e.out[:0]
	var ev keyEvent
	e.keyRestoring = true
	for i := keyStart; i <= e.keyCurrent; i++ {
		e.out = append(e.out, uint16(e.keyStrokes[i].ev.keyCode))
		ev = e.input.keyCodeToSymbol(e.keyStrokes[i].ev.keyCode)
		e.keyStrokes[i].converted = false
		e.processAppend(&ev)
	}
	e.keyRestoring = false

	return true
}

// --------------------------------------------------
func (e *Engine) processWordEnd(ev *keyEvent) int {
	if e.opts.MacroEnabled != 0 && e.macroMatch(ev) {
		return 1
	}

	if e.opts.SpellCheckEnabled == 0 || e.singleMode || e.current < 0 || e.keyRestoring {
		e.current++
		entry := &e.buffer[e.current]
		entry.form = VnwEmpty
		entry.c1Offset = -1
		entry.c2Offset = -1
		entry.vOffset = -1
		entry.keyCode = ev.keyCode
		entry.vnSym = vnToLower(ev.vnSym)
		if entry.vnSym != ev.vnSym {
			entry.caps = 1
		} else {
			entry.caps = 0
		}
		return 0
	}

	if e.opts.AutoNonVnRestore != 0 && e.lastWordIsNonVn() {
		if e.restoreKeyStrokes() {
			e.keyRestored = true
			e.outputWritten = true
		}
	}

	e.current++
	entry := &e.buffer[e.current]
	entry.form = VnwEmpty
	entry.c1Offset = -1
	entry.c2Offset = -1
	entry.vOffset = -1
	entry.keyCode = ev.keyCode
	entry.vnSym = vnToLower(ev.vnSym)
	if entry.vnSym != ev.vnSym {
		entry.caps = 1
	} else {
		entry.caps = 0
	}

	if e.keyRestored {
		e.out = append(e.out, uint16(ev.keyCode))
		return 1
	}

	return 0
}

// -----------------------------------------------------------
// Test if last word is a non-Vietnamese word
// -----------------------------------------------------------
func (e *Engine) lastWordIsNonVn() bool {
	if e.current < 0 {
		return false
	}

	switch e.buffer[e.current].form {
	case VnwNonVn:
		return true
	case VnwEmpty, VnwC:
		return false
	case VnwV, VnwCV:
		return vSeqList[e.buffer[e.current].vseq()].complete == 0
	case VnwVC, VnwCVC:
		vIndex := e.current - e.buffer[e.current].vOffset
		vs := e.buffer[vIndex].vseq()
		if vSeqList[vs].complete == 0 {
			return true
		}
		cs := e.buffer[e.current].cseq()
		c1 := CsNil
		if e.buffer[e.current].c1Offset != -1 {
			c1 = e.buffer[e.current-e.buffer[e.current].c1Offset].cseq()
		}

		if !isValidCVC(c1, vs, cs) {
			return true
		}

		tonePos := (vIndex - vSeqList[vs].length + 1) + e.getTonePosition(vs, false)
		tone := e.buffer[tonePos].tone
		if (cs == CsC || cs == CsCh || cs == CsP || cs == CsT) &&
			(tone == 2 || tone == 3 || tone == 4) {
			return true
		}
	}
	return false
}

// -----------------------------------------------------------
// Test if last word has a Vietnamese mark, that is tones, decorators
// -----------------------------------------------------------
func (e *Engine) lastWordHasVnMark() bool {
	for i := e.current; i >= 0 && e.buffer[i].form != VnwEmpty; i-- {
		sym := e.buffer[i].vnSym
		if sym != VnlNonVnChar {
			if isVnVowel(sym) {
				if e.buffer[i].tone != 0 {
					return true
				}
			}
			if int(sym) != stdVnRootChar[sym] {
				return true
			}
		}
	}
	return false
}

// ----------------------------------------------------------
// Public API matching the shell's Engine interface
// ----------------------------------------------------------
func (e *Engine) Filter(keyCode uint32) Result {
	return e.process(keyCode)
}

func (e *Engine) Backspace() Result {
	return e.processBackspace()
}

func (e *Engine) RestoreKeyStrokes() Result {
	res := Result{KeyOut: true}
	e.out = e.out[:0]
	if e.restoreKeyStrokes() {
		res.Backs = e.backs
		res.Out = append([]uint16(nil), e.out...)
		res.Consumed = true
	}
	return res
}

func (e *Engine) LoadMacroFile(path string) bool {
	return e.loadMacroFile(path)
}

func (e *Engine) LoadKeyMapFile(path string) bool {
	m, ok := loadKeyMapFile(path)
	if !ok {
		return false
	}
	e.SetKeyMap(m)
	return true
}

func isAlpha(c uint32) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}
