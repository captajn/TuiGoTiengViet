#!/usr/bin/env python3
"""End-to-end smoke test for tuigo-ibus.

Runs inside a dbus session with a real ibus-daemon (headless via xvfb).
Feeds fake key events through an IBus.InputContext bound to engine "tuigo"
and asserts on the text the engine commits back.

Usage: e2e.py <path-to-tuigo-ibus>
"""
import sys
import time
import threading

import gi
gi.require_version("IBus", "1.0")
from gi.repository import IBus, GLib

IBUS_MOD_RELEASE = 1 << 30

committed = []   # committed text fragments
preedits = []    # last shown preedit text
failures = []


def type_str(ic, s):
    """Feed an ASCII string through the input context."""
    for i, ch in enumerate(s):
        keyval = ord(ch)
        # process_key_event(keyval, keycode, state) — keycode is the
        # hardware code; ibus only forwards it, any unique value works.
        ic.process_key_event(keyval, i, 0)
        ic.process_key_event(keyval, i, IBUS_MOD_RELEASE)
        # let the dbus round-trips land
        while GLib.MainContext.default().pending():
            GLib.MainContext.default().iteration(False)


def check(name, got, want):
    if got != want:
        failures.append(f"{name}: got {got!r}, want {want!r}")
    else:
        print(f"  ok {name}: {got!r}")


def main():
    exe = sys.argv[1]
    bus = IBus.Bus()

    ic = bus.create_input_context("tuigo-e2e")
    ic.set_capabilities(IBus.Capabilite.PREEDIT_TEXT)

    ic.connect("commit-text", lambda ic, t: committed.append(t.get_text()))
    ic.connect("update-preedit-text-with-mode",
               lambda ic, t, c, v, m: preedits.append(t.get_text() if v else ""))
    ic.connect("update-preedit-text",
               lambda ic, t, c, v: preedits.append(t.get_text() if v else ""))

    # spawn the engine component (standalone registration)
    import subprocess
    proc = subprocess.Popen([exe, "--standalone"])
    time.sleep(1.5)

    try:
        ic.focus_in()
        bus.set_global_engine("tuigo")
        time.sleep(0.5)

        # telex: vieejt -> việt
        type_str(ic, "vieejt ")
        check("telex", "".join(committed), "việt ")

        committed.clear()
        type_str(ic, "duowngf xin chaof ")
        check("phrases", "".join(committed), "đường xin chào ")
    finally:
        proc.terminate()

    if failures:
        print("\n".join(failures))
        sys.exit(1)
    print("e2e: PASS")


if __name__ == "__main__":
    main()
