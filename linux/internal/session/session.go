// Package session adapts the engine's edit-stream contract
// (Result{Backs, Out}) to IME APIs that cannot delete already-committed
// text — IBus preedit on Linux, InputMethodKit marked text on macOS.
//
// The current word lives in a composing model; it is shown as preedit and
// only committed when the engine reaches a word boundary. Printable keys
// the engine does not consume while a word is open are swallowed into the
// preedit rather than forwarded: forwarding would commit them, and a later
// word-end restore (macro, non-Vn restore) emits Backs that assume the
// whole word is still editable.
package session

import "tuigo/engine"

// Action is the full list of things the host must do after one key event,
// in order:
//  1. insert Commit at the caret (replacing any visible preedit),
//  2. show Preedit as marked text (empty hides the preedit),
//  3. let the original key event through to the app if Forward is set.
type Action struct {
	Commit  []uint16 // UTF-16 units to commit; may be empty
	Preedit []uint16 // marked text to display after committing
	Forward bool     // forward the triggering key to the application
}

// Session wraps one engine.Engine for one client context.
type Session struct {
	eng     *engine.Engine
	model   []uint16 // rendered form of the current composing word
	viet    bool
	AutoCap bool // capitalise the first letter after ". ", "! ", "? "
	capNext bool
}

// New creates a session. viet=false means English pass-through mode.
func New(eng *engine.Engine, viet bool) *Session {
	eng.SetVietKey(viet)
	return &Session{eng: eng, viet: viet}
}

// SetViet switches Vietnamese/English mode, committing any pending word.
func (s *Session) SetViet(v bool) Action {
	if v == s.viet {
		return Action{}
	}
	s.viet = v
	s.eng.SetVietKey(v)
	s.eng.ResetBuf()
	return s.flush()
}

// Feed processes one printable character (ASCII 0x20–0x7E, or a Unicode
// codepoint for precomposed input). Returns the Action to perform.
func (s *Session) Feed(ch uint32) Action {
	if !s.viet {
		return s.forward()
	}

	forceCap := s.AutoCap && s.capNext && ch >= 'a' && ch <= 'z'
	if forceCap {
		ch -= 'a' - 'A'
	}
	res := s.eng.Filter(ch)

	if s.AutoCap {
		switch ch {
		case '.', '!', '?':
			s.capNext = true
		case ' ':
			// pending capitalisation survives whitespace
		default:
			s.capNext = false
		}
	}

	if res.Consumed {
		s.apply(res.Backs, res.Out)
		return s.emit()
	}

	if s.eng.AtWordBeginning() {
		// Word-break or reset char: commit the pending word, forward the
		// key so the app inserts the terminator itself.
		return s.forward()
	}
	// Engine recorded the char but passed it through (e.g. the first
	// letter of a word, or a non-letter mid-word). Swallow it into the
	// preedit — forwarding it would commit text the engine may still
	// rewrite at word end (macro, non-Vn restore).
	s.model = append(s.model, uint16(ch))
	return Action{Preedit: append([]uint16(nil), s.model...)}
}

// Backspace forwards one delete press through the engine.
func (s *Session) Backspace() Action {
	if !s.viet {
		return s.forward()
	}
	res := s.eng.Backspace()
	if !res.Consumed && len(s.model) == 0 {
		// Buffer empty — the app deletes a committed char itself.
		return s.forward()
	}
	// The engine shrank its buffer (res.Backs is populated even when the
	// result is not Consumed); the deleted char lives in our preedit, so
	// we must mirror the edit instead of forwarding.
	s.apply(res.Backs, res.Out)
	return s.emit()
}

// Reset commits any pending word and drops the engine buffer — call it on
// focus loss, caret movement and modifier shortcuts.
func (s *Session) Reset() Action {
	s.capNext = false
	s.eng.ResetBuf()
	return s.flush()
}

// EnterPressed marks a paragraph break for AutoCap and resets.
func (s *Session) EnterPressed() Action {
	if s.AutoCap {
		s.capNext = true
	}
	return s.Reset()
}

func (s *Session) apply(backs int, out []uint16) {
	if backs > len(s.model) {
		backs = len(s.model)
	}
	s.model = s.model[:len(s.model)-backs]
	s.model = append(s.model, out...)
}

// emit reports the visible state after a consumed edit.
func (s *Session) emit() Action {
	if s.eng.AtWordBeginning() {
		return s.flush()
	}
	return Action{Preedit: append([]uint16(nil), s.model...)}
}

// flush commits the pending word and clears the model.
func (s *Session) flush() Action {
	a := Action{Commit: append([]uint16(nil), s.model...)}
	s.model = s.model[:0]
	return a
}

func (s *Session) forward() Action {
	a := s.flush()
	a.Forward = true
	return a
}
