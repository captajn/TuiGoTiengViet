// SPDX-FileCopyrightText: 2026 TuiGo contributors
// SPDX-License-Identifier: GPL-3.0-or-later
//
// How apps differ in taking text and deletions from the input method.
// Ported from Ngó Sen (src/core/ngosen-app-quirks.cpp): decided from what
// the field reports where possible; a program name is matched only when
// nothing reported tells the app apart. TuiGo-specific rules (Edge and
// Telegram ignoring deleteSurroundingText) are marked as such.
//
// This file is fcitx5-free: the delete decision in addon.cpp builds a Field
// from the input context and asks these questions, so the rules are
// unit-testable in sen_test without fcitx5 headers.
#pragma once

#include <string>

namespace tuigo {

// What the app reports about the field: how it talks to the input method,
// its program name, and the capability flags it sets.
struct Field {
    std::string frontend; // wayland, wayland_v2, xim, ibus, dbus, fcitx4
    std::string program;
    bool surroundingText = false;  // app reports the text around the cursor
    bool keyEventOrderFix = false; // KeyEventOrderFix capability
    bool formattedPreedit = false; // FormattedPreedit capability
};

// True when the frontend can deliver backspaces itself, so no extra
// mechanism is needed: the Wayland input-method frontend, XIM clients,
// and IBus, D-Bus and fcitx4 clients other than SDL.
bool forwardsBackspaces(const Field& field);

// True for GTK4 clients, whose IM modules drop forwarded keys; delete
// through surrounding text there.
bool ignoresForwardedKeys(const Field& field);

// True for LibreOffice: it runs Backspace as an async shortcut, so
// committed text overtakes it. Matched by name because nothing Calc or
// Writer reports sets them apart from an app whose BackSpace is prompt.
bool appliesBackspacesLate(const Field& field);

// True for LibreOffice Calc over Wayland text-input: it tells the input
// method nothing when another cell is selected, and reports the field
// only when a key reaches it. Matched by name for the same reason.
bool reportsFieldOnlyOnKey(const Field& field);

// True for apps that report the SurroundingText capability but silently
// ignore deleteSurroundingText (observed: Edge and Telegram duplicate
// letters, "chào" -> "chaào"). Matched by name because nothing they
// report tells them apart from apps that honour the delete. TuiGo rule,
// not in Ngó Sen: Ngó Sen only meets these apps on Wayland, where its
// forwardsBackspaces rule already routes them to forwarded keys.
bool ignoresDeleteSurrounding(const Field& field);

// True when the app must delete through surrounding text: it ignores
// forwarded keys (GTK4) or applies BackSpace late (LibreOffice), and it
// reports surrounding text. The alternative (forwarded BackSpace) cannot
// work there at all.
bool mustUseSurrounding(const Field& field);

// True for browsers (Chromium/Firefox family + Chromium-based Electron
// apps): they implement deleteSurroundingText correctly and
// synchronously. Matched by name because the capability flags do not
// distinguish a browser from any other app that reports surrounding text.
bool isBrowser(const Field& field);

} // namespace tuigo
