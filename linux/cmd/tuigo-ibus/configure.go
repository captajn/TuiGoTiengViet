package main

import (
	"tuigo-linux/config"
	"tuigo/engine"
)

// configureEngine wires parsed settings onto the shared engine.
//
// keymap.txt is loaded only for ImUsrKeymap. LoadKeyMapFile calls
// SetKeyMap which forces ImUsrKeymap, so loading it unconditionally
// hijacked every other selected method as soon as a keymap.txt existed
// in the config dir (PLAN G2). Keep this order: SetInputMethod first,
// then the keymap only when the custom method is selected.
func configureEngine(eng *engine.Engine, cfg config.Settings, keymapPath, macroPath string) {
	opts := cfg.EngineOptions()
	eng.SetOptions(&opts)
	eng.SetInputMethod(cfg.InputMethod)
	if cfg.InputMethod == engine.ImUsrKeymap {
		eng.LoadKeyMapFile(keymapPath)
	}
	eng.LoadMacroFile(macroPath)
}
