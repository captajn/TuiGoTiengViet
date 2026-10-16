// Command tuigo-fcitx5-shim is the C bridge between the fcitx5 addon
// (C++) and Tui Go's typing engine (Go). It is built with
// `go build -buildmode=c-archive`: the addon links the archive and talks
// to the engine only through the exported functions below.
//
// One engine instance per fcitx5 input context, addressed by handle
// (the same handle pattern Ngó Sen's bamboo wrapper uses). The last
// Filter/Backspace result is kept on the instance; the addon reads it
// back with TuiGoBacks / TuiGoPullOut. Out is returned as UTF-8 text —
// the engine emits UTF-16 units, and ASCII key codes from a restore
// coincide with the characters themselves.
package main

/*
#include <stdint.h>
#include <stdlib.h>
*/
import "C"

import (
	"sync"
	"unicode/utf16"
	"unsafe"

	"tuigo/engine"
)

type instance struct {
	eng      *engine.Engine
	consumed bool
	backs    int
	out      []uint16
}

var (
	mu      sync.Mutex
	next    C.uintptr_t = 1
	insts               = map[C.uintptr_t]*instance{}
)

func get(h C.uintptr_t) *instance {
	mu.Lock()
	defer mu.Unlock()
	return insts[h]
}

func main() {}

//export TuiGoNew
func TuiGoNew() C.uintptr_t {
	mu.Lock()
	defer mu.Unlock()
	h := next
	next++
	insts[h] = &instance{eng: engine.New()}
	return h
}

//export TuiGoDelete
func TuiGoDelete(h C.uintptr_t) {
	mu.Lock()
	defer mu.Unlock()
	delete(insts, h)
}

//export TuiGoSetViet
func TuiGoSetViet(h C.uintptr_t, viet C.int) {
	if in := get(h); in != nil {
		in.eng.SetVietKey(viet != 0)
	}
}

//export TuiGoSetInputMethod
func TuiGoSetInputMethod(h C.uintptr_t, im C.int) {
	if in := get(h); in != nil {
		in.eng.SetInputMethod(int(im))
	}
}

//export TuiGoSetOptions
func TuiGoSetOptions(h C.uintptr_t, spell, modern, free, macro, restore C.int) {
	if in := get(h); in != nil {
		in.eng.SetOptions(&engine.Options{
			SpellCheckEnabled:  int32(spell),
			ModernStyle:        int32(modern),
			FreeMarking:        int32(free),
			MacroEnabled:       int32(macro),
			AutoNonVnRestore:   int32(restore),
		})
	}
}

//export TuiGoLoadMacroFile
func TuiGoLoadMacroFile(h C.uintptr_t, path *C.char) C.int {
	if in := get(h); in != nil && in.eng.LoadMacroFile(C.GoString(path)) {
		return 1
	}
	return 0
}

//export TuiGoLoadKeymapFile
func TuiGoLoadKeymapFile(h C.uintptr_t, path *C.char) C.int {
	if in := get(h); in != nil && in.eng.LoadKeyMapFile(C.GoString(path)) {
		return 1
	}
	return 0
}

//export TuiGoReset
func TuiGoReset(h C.uintptr_t) {
	if in := get(h); in != nil {
		in.eng.ResetBuf()
		in.consumed = false
		in.backs = 0
		in.out = nil
	}
}

//export TuiGoAtWordBeginning
func TuiGoAtWordBeginning(h C.uintptr_t) C.int {
	if in := get(h); in != nil && in.eng.AtWordBeginning() {
		return 1
	}
	return 0
}

func store(in *instance, res engine.Result) C.int {
	in.consumed = res.Consumed
	in.backs = res.Backs
	in.out = res.Out
	if res.Consumed {
		return 1
	}
	return 0
}

//export TuiGoProcessKey
func TuiGoProcessKey(h C.uintptr_t, key C.uint32_t) C.int {
	if in := get(h); in != nil {
		return store(in, in.eng.Filter(uint32(key)))
	}
	return 0
}

//export TuiGoBackspace
func TuiGoBackspace(h C.uintptr_t) C.int {
	if in := get(h); in != nil {
		return store(in, in.eng.Backspace())
	}
	return 0
}

//export TuiGoBacks
func TuiGoBacks(h C.uintptr_t) C.int {
	if in := get(h); in != nil {
		return C.int(in.backs)
	}
	return 0
}

//export TuiGoPullOut
func TuiGoPullOut(h C.uintptr_t, buf *C.char, cap C.int) C.int {
	in := get(h)
	if in == nil || cap <= 0 {
		return 0
	}
	text := string(utf16.Decode(in.out))
	dst := (*[1 << 30]byte)(unsafe.Pointer(buf))[:cap:cap]
	return C.int(copy(dst, text))
}
