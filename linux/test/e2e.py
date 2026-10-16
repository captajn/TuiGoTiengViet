#!/usr/bin/env python3
"""End-to-end smoke test for tuigo-ibus.

Runs inside a dbus session with a real ibus-daemon (headless via xvfb).
Feeds fake key events through an IBus.InputContext bound to engine "tuigo"
and asserts on the text the engine commits back.

Scenarios (PLAN G6): Telex + phrases, Backspace inside a word, macro
expansion, VNI, VIQR, VI/EN toggle via the panel property, and the
invariant that the preedit is empty after a word commit.

Usage: e2e.py  (run inside a dbus session with ibus-daemon already up
and the tuigo component installed in ~/.local/share/ibus/component/)
"""
import os
import shutil
import sys
import time

import gi
gi.require_version("IBus", "1.0")
from gi.repository import IBus, GLib

IBUS_MOD_RELEASE = 1 << 30
KEY_BACKSPACE = 0xFF08

PROP_UNCHECKED = 0
PROP_CHECKED = 1

document = []  # what the app would show: commit-text + forwarded chars
preedits = []  # last shown preedit text
failures = []

CONFIG_DIR = os.path.join(
    os.environ.get("XDG_CONFIG_HOME", os.path.expanduser("~/.config")), "tuigo")
MACRO_PATH = os.path.join(CONFIG_DIR, "macro.txt")


def pump():
    while GLib.MainContext.default().pending():
        GLib.MainContext.default().iteration(False)


def feed_key(keyval, state=0, keycode=0):
    """Feed one raw key event to the active input context."""
    return IC.process_key_event(keyval, keycode, state)


def type_str(s, verbose=False):
    """Feed an ASCII string through the input context. Keys the engine
    does not handle are forwarded — a real app inserts them itself, so
    we record them into the document."""
    for i, ch in enumerate(s):
        keyval = ord(ch)
        handled = feed_key(keyval, 0, i)
        feed_key(keyval, IBUS_MOD_RELEASE, i)
        if verbose:
            print(f"    key {ch!r}: handled={handled}")
        # deliver pending commit-text signals before treating an
        # unhandled key as app-inserted text — the engine commits the
        # word just before letting the terminator key through.
        pump()
        if not handled and 0x20 <= keyval <= 0x7E:
            document.append(ch)


def backspace(n=1):
    for _ in range(n):
        feed_key(KEY_BACKSPACE)
        feed_key(KEY_BACKSPACE, IBUS_MOD_RELEASE)
        pump()


def activate(prop, state):
    """Click a panel property (input method radio, VI/EN toggle)."""
    IC.property_activate(prop, state)
    pump()
    time.sleep(0.2)  # let setIM/toggleViet settle
    pump()


def last_preedit():
    return preedits[-1] if preedits else None


def doc_text():
    return "".join(document)


def doc_clear():
    document.clear()


def check(name, got, want):
    if got != want:
        failures.append(f"{name}: got {got!r}, want {want!r}")
    else:
        print(f"  ok {name}: {got!r}")


def scenario_telex():
    doc_clear()
    type_str("vieejt ", verbose=True)
    check("telex", doc_text(), "việt ")
    check("telex preedit empty after commit", last_preedit(), "")


def scenario_phrases():
    doc_clear()
    type_str("dduowngf xin chaof ")
    check("phrases", doc_text(), "đường xin chào ")


def scenario_backspace():
    doc_clear()
    type_str("vieejt")
    check("backspace preedit word", last_preedit(), "việt")
    backspace(1)
    check("backspace once", last_preedit(), "việ")
    backspace(1)
    check("backspace twice", last_preedit(), "vi")
    type_str("ejt ")
    check("backspace retyped commit", doc_text(), "việt ")


def scenario_macro():
    doc_clear()
    type_str("ko ")
    check("macro", doc_text(), "không ")


def scenario_vni():
    doc_clear()
    activate("im_vni", PROP_CHECKED)
    # e6 = roof on e, 5 = dot-below tone
    type_str("vie65t ")
    check("vni", doc_text(), "việt ")


def scenario_viqr():
    doc_clear()
    activate("im_viqr", PROP_CHECKED)
    # ^ = roof, . = dot-below tone
    type_str("vie^.t ")
    check("viqr", doc_text(), "việt ")
    activate("im_telex", PROP_CHECKED)


def scenario_toggle():
    doc_clear()
    activate("vietkey", PROP_UNCHECKED)
    type_str("vieejt ")
    check("toggle off passthrough", doc_text(), "vieejt ")
    doc_clear()
    activate("vietkey", PROP_CHECKED)
    type_str("vieejt ")
    check("toggle on", doc_text(), "việt ")


def prepare_macro_file():
    """Install a macro list before the engine is enabled (it reloads on
    activation). Backs up any existing file for restore at the end."""
    backup = None
    if os.path.exists(MACRO_PATH):
        backup = MACRO_PATH + ".e2e.bak"
        shutil.copy2(MACRO_PATH, backup)
    os.makedirs(CONFIG_DIR, exist_ok=True)
    with open(MACRO_PATH, "w", encoding="utf-8") as f:
        f.write("ko:không\n")
    return backup


def restore_macro_file(backup):
    if backup and os.path.exists(backup):
        shutil.move(backup, MACRO_PATH)
    elif os.path.exists(MACRO_PATH):
        os.remove(MACRO_PATH)


def main():
    global IC

    bus = IBus.Bus()

    IC = bus.create_input_context("tuigo-e2e")
    # CAP_FOCUS is required for focus_in/focus_out to do anything.
    IC.set_capabilities(IBus.Capabilite.PREEDIT_TEXT | IBus.Capabilite.FOCUS)

    IC.connect("commit-text", lambda ic, t: document.append(t.get_text()))
    IC.connect("update-preedit-text-with-mode",
               lambda ic, t, c, v, m: preedits.append(t.get_text() if v else ""))
    IC.connect("update-preedit-text",
               lambda ic, t, c, v: preedits.append(t.get_text() if v else ""))

    backup = prepare_macro_file()
    try:
        # Set the global engine BEFORE focusing the context — a second
        # focus_in on an already-focused context is a no-op. The engine
        # reads macro.txt on Enable, so the file is in place by now.
        bus.set_global_engine("tuigo")
        IC.focus_in()
        time.sleep(1.0)
        try:
            eng = bus.get_global_engine()
            print("global engine:", eng.get_name() if eng else None)
        except Exception as ex:
            print("get_global_engine:", ex)
        names = [d.get_name() for d in bus.list_engines()]
        print("known engines:", [n for n in names if "tuigo" in n])

        scenario_telex()
        scenario_phrases()
        scenario_backspace()
        scenario_macro()
        scenario_vni()
        scenario_viqr()
        scenario_toggle()
    finally:
        IC.focus_out()
        restore_macro_file(backup)

    if failures:
        print("\n".join(failures))
        sys.exit(1)
    print("e2e: PASS")


if __name__ == "__main__":
    main()
