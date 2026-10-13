#!/usr/bin/env python3
"""End-to-end smoke test for tuigo-ibus.

Runs inside a dbus session with a real ibus-daemon (headless via xvfb).
Feeds fake key events through an IBus.InputContext bound to engine "tuigo"
and asserts on the text the engine commits back.

Usage: e2e.py  (run inside a dbus session with ibus-daemon already up
and the tuigo component installed in ~/.local/share/ibus/component/)
"""
import sys
import time

import gi
gi.require_version("IBus", "1.0")
from gi.repository import IBus, GLib

IBUS_MOD_RELEASE = 1 << 30

document = []  # what the app would show: commit-text + forwarded chars
preedits = []  # last shown preedit text
failures = []


def pump():
    while GLib.MainContext.default().pending():
        GLib.MainContext.default().iteration(False)


def type_str(ic, s, verbose=False):
    """Feed an ASCII string through the input context. Keys the engine
    does not handle are forwarded — a real app inserts them itself, so
    we record them into the document."""
    for i, ch in enumerate(s):
        keyval = ord(ch)
        handled = ic.process_key_event(keyval, i, 0)
        ic.process_key_event(keyval, i, IBUS_MOD_RELEASE)
        if verbose:
            print(f"    key {ch!r}: handled={handled}")
        # deliver pending commit-text signals before treating an
        # unhandled key as app-inserted text — the engine commits the
        # word just before letting the terminator key through.
        pump()
        if not handled and 0x20 <= keyval <= 0x7E:
            document.append(ch)


def doc_text():
    return "".join(document)


def doc_clear():
    document.clear()


def check(name, got, want):
    if got != want:
        failures.append(f"{name}: got {got!r}, want {want!r}")
    else:
        print(f"  ok {name}: {got!r}")


def main():
    bus = IBus.Bus()

    ic = bus.create_input_context("tuigo-e2e")
    # CAP_FOCUS is required for focus_in/focus_out to do anything.
    ic.set_capabilities(IBus.Capabilite.PREEDIT_TEXT | IBus.Capabilite.FOCUS)

    ic.connect("commit-text", lambda ic, t: document.append(t.get_text()))
    ic.connect("update-preedit-text-with-mode",
               lambda ic, t, c, v, m: preedits.append(t.get_text() if v else ""))
    ic.connect("update-preedit-text",
               lambda ic, t, c, v: preedits.append(t.get_text() if v else ""))

    try:
        # Set the global engine BEFORE focusing the context — a second
        # focus_in on an already-focused context is a no-op.
        bus.set_global_engine("tuigo")
        ic.focus_in()
        time.sleep(1.0)
        try:
            eng = bus.get_global_engine()
            print("global engine:", eng.get_name() if eng else None)
        except Exception as ex:
            print("get_global_engine:", ex)
        names = [d.get_name() for d in bus.list_engines()]
        print("known engines:", [n for n in names if "tuigo" in n])

        # telex: vieejt + space -> "việt " (space forwarded to the app)
        type_str(ic, "vieejt ", verbose=True)
        check("telex", doc_text(), "việt ")

        doc_clear()
        type_str(ic, "dduowngf xin chaof ")
        check("phrases", doc_text(), "đường xin chào ")
    finally:
        ic.focus_out()

    if failures:
        print("\n".join(failures))
        sys.exit(1)
    print("e2e: PASS")


if __name__ == "__main__":
    main()
