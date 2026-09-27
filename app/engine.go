package main

import (
	"bogovn/engine"
)

// Engine abstracts the Vietnamese input engine.
// Implemented by the pure-Go port in package engine.
type Engine interface {
	Setup()
	Filter(ch uint32) Result // feed one ASCII char
	Backspace() Result
	Reset()
	SetCapsState(shift, caps bool)
	SetVietKey(on bool) // off = pass-through, macros still expand
	SetInputMethod(im int)
	SetOptions(o Options)
	GetOptions() Options
	LoadMacro(path string) bool
	LoadKeymap(path string) bool
	ConvertTCVN(out []uint16) // rewrite UTF-16 units to TCVN3 codepoints
}

// Result of processing one key.
type Result = engine.Result

type Options = engine.Options

// Input methods (UkInputMethod).
const (
	ImTelex       = engine.ImTelex
	ImVni         = engine.ImVni
	ImViqr        = engine.ImViqr
	ImMsVi        = engine.ImMsVi
	ImUsrKeymap   = engine.ImUsrKeymap
	ImTelexSimple = engine.ImTelexSimple
	ImTelexVni    = engine.ImTelexVni
)

// ---- pure-Go engine -------------------------------------------------------

type goEngine struct {
	e *engine.Engine
}

func newEngine() (*goEngine, error) {
	return &goEngine{e: engine.New()}, nil
}

func (g *goEngine) Setup() {}

func (g *goEngine) Filter(ch uint32) Result { return g.e.Filter(ch) }
func (g *goEngine) Backspace() Result       { return g.e.Backspace() }
func (g *goEngine) Reset()                  { g.e.ResetBuf() }

func (g *goEngine) SetCapsState(shift, caps bool) {
	g.e.ShiftPressed = shift
	g.e.CapsLockOn = caps
}

func (g *goEngine) SetVietKey(on bool)     { g.e.SetVietKey(on) }
func (g *goEngine) ConvertTCVN(o []uint16) { engine.ConvertTCVN(o) }
func (g *goEngine) SetInputMethod(im int)  { g.e.SetInputMethod(im) }

func (g *goEngine) SetOptions(o Options) { g.e.SetOptions(&o) }
func (g *goEngine) GetOptions() Options  { return *g.e.GetOptions() }

func (g *goEngine) LoadMacro(p string) bool  { return g.e.LoadMacroFile(p) }
func (g *goEngine) LoadKeymap(p string) bool { return g.e.LoadKeyMapFile(p) }
