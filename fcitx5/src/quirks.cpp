// SPDX-FileCopyrightText: 2026 TuiGo contributors
// SPDX-License-Identifier: GPL-3.0-or-later
//
// Ported from Ngó Sen (src/core/ngosen-app-quirks.cpp). Behaviour first:
// frontend and capability flags decide where possible; program names are
// matched only when nothing the app reports tells it apart.
#include "quirks.h"

#include <cctype>

namespace tuigo {
namespace {

// Strip the ".desktop" suffix GNOME adds to program names: under GNOME's
// IBus path a window reports "firefox.desktop" where KDE reports "firefox".
std::string stripDesktopSuffix(const std::string& program) {
    const std::string suffix = ".desktop";
    if (program.size() >= suffix.size() &&
        program.compare(program.size() - suffix.size(), suffix.size(), suffix) == 0) {
        return program.substr(0, program.size() - suffix.size());
    }
    return program;
}

bool isStartsWith(const std::string& str, const std::string& prefix) {
    return str.size() >= prefix.size() && str.compare(0, prefix.size(), prefix) == 0;
}

// Lowercased substring match: program names arrive in various forms
// ("google-chrome", "com.google.Chrome", "org.telegram.desktop"), and the
// match must survive all of them.
bool containsName(const std::string& program, const char* name) {
    std::string lower = program;
    for (char& c : lower) {
        c = static_cast<char>(std::tolower(static_cast<unsigned char>(c)));
    }
    return lower.find(name) != std::string::npos;
}

// Bare program name, lowercased, without the ".desktop" suffix: for exact
// matches ("code" must not match "barcode" or "codeblocks").
std::string bareName(const std::string& program) {
    std::string lower = stripDesktopSuffix(program);
    for (char& c : lower) {
        c = static_cast<char>(std::tolower(static_cast<unsigned char>(c)));
    }
    return lower;
}

} // namespace

bool forwardsBackspaces(const Field& field) {
    // On wayland_v2 (input-method-v2: Sway, Hyprland) fcitx5 sends forwarded
    // keys through a virtual keyboard.
    if (field.frontend == "wayland" || field.frontend == "wayland_v2" ||
        field.frontend == "xim") {
        return true;
    }
    if (field.frontend == "ibus") {
        // SDL takes only commits and preedit from the IM.
        return !isStartsWith(field.program, "SDL");
    }
    if (field.frontend == "dbus") {
        // fcitx5-gtk and fcitx5-qt set one of these; SDL sets neither and
        // handles only commits and preedit.
        return field.keyEventOrderFix || field.surroundingText;
    }
    if (field.frontend == "fcitx4") {
        // Snap apps bundle the fcitx4 GTK module, which sets one of these;
        // SDL 2.0.12 and older speak this protocol too and set at most Preedit.
        return field.formattedPreedit || field.surroundingText;
    }
    return false;
}

bool ignoresForwardedKeys(const Field& field) {
    if (field.frontend == "ibus") {
        // IBus for GTK4 feeds forwarded keys back into the IM instead of
        // the widget.
        return isStartsWith(field.program, "gtk4-im:");
    }
    // fcitx5-gtk4 discards forwarded keys, and is the D-Bus client that
    // leaves this flag unset.
    return field.frontend == "dbus" && !field.keyEventOrderFix && field.surroundingText;
}

bool appliesBackspacesLate(const Field& field) {
    return stripDesktopSuffix(field.program) == "soffice";
}

bool reportsFieldOnlyOnKey(const Field& field) {
    // Nothing Calc reports sets it apart from a text field that keeps its text.
    return field.frontend == "wayland" &&
           stripDesktopSuffix(field.program) == "libreoffice-calc";
}

bool ignoresDeleteSurrounding(const Field& field) {
    // TuiGo rule (see quirks.h): Edge and Telegram report the capability
    // but ignore the delete.
    return containsName(field.program, "msedge") ||
           containsName(field.program, "microsoft-edge") ||
           containsName(field.program, "telegram");
}

bool mustUseSurrounding(const Field& field) {
    return field.surroundingText &&
           (ignoresForwardedKeys(field) || appliesBackspacesLate(field) ||
            reportsFieldOnlyOnKey(field));
}

bool isBrowser(const Field& field) {
    static const char* browsers[] = {"chromium",
                                     "chrome",
                                     "helium", // Chromium fork; was missed, fell back to async
                                     "thorium",
                                     "ungoogled-chromium",
                                     "firefox",
                                     "brave",
                                     "vivaldi",
                                     "opera",
                                     "librewolf",
                                     "floorp",
                                     "zen"};
    for (const char* b : browsers) {
        if (containsName(field.program, b)) {
            return true;
        }
    }
    // Chromium-based Electron apps share Chrome's text-input stack.
    // "code" is an exact match: a substring would also match "barcode".
    const std::string bare = bareName(field.program);
    if (bare == "code" || bare == "code-oss") {
        return true;
    }
    return containsName(field.program, "discord") ||
           containsName(field.program, "slack") ||
           containsName(field.program, "signal") ||
           containsName(field.program, "obsidian");
}

} // namespace tuigo
