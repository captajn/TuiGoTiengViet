// fcitx5 shell for Tui Go's Sen (no-preedit) typing. The shell is thin:
// key routing, per-input-context state, config loading, and the deletion
// path choice (deleteSurroundingText vs forwarded BackSpace). All typing
// decisions live in sen.cpp + the Go engine.
#include "addon.h"

#include "quirks.h"
#include "utf8.h"

#include <cstdlib>
#include <ctime>
#include <fstream>
#include <map>
#include <string>
#include <utility>

#include <fcitx/inputcontext.h>
#include <fcitx-utils/key.h>
#include <fcitx-utils/keysym.h>

namespace fcitx {
namespace {

// X11 keysym values (same numbers fcitx KeySym uses).
constexpr std::uint32_t kSymBackSpace = 0xFF08;
constexpr std::uint32_t kSymTab = 0xFF09;
constexpr std::uint32_t kSymISOLeftTab = 0xFE20;
constexpr std::uint32_t kSymReturn = 0xFF0D;
constexpr std::uint32_t kSymEscape = 0xFF1B;
constexpr std::uint32_t kSymHome = 0xFF50;
constexpr std::uint32_t kSymEnd = 0xFF57;
constexpr std::uint32_t kSymDelete = 0xFFFF;
constexpr std::uint32_t kSymKPDelete = 0xFF9F;
constexpr std::uint32_t kSymInsert = 0xFF63;
constexpr std::uint32_t kSymMenu = 0xFF67;
constexpr std::uint32_t kSymPrint = 0xFF61;
constexpr std::uint32_t kSymScrollLock = 0xFF14;
constexpr std::uint32_t kSymPause = 0xFF13;
constexpr std::uint32_t kSymNumLock = 0xFF7F;
constexpr std::uint32_t kSymCapsLock = 0xFFE5;
constexpr std::uint32_t kSymKPBegin = 0xFF9D;
constexpr std::uint32_t kSymF1 = 0xFFBE;
constexpr std::uint32_t kSymF35 = 0xFFE0;
constexpr std::uint32_t kSymShiftL = 0xFFE1;
constexpr std::uint32_t kSymShiftR = 0xFFE2;
constexpr std::uint32_t kSymControlL = 0xFFE3;
constexpr std::uint32_t kSymControlR = 0xFFE4;
constexpr std::uint32_t kSymMetaL = 0xFFE7;
constexpr std::uint32_t kSymMetaR = 0xFFE8;
constexpr std::uint32_t kSymAltL = 0xFFE9;
constexpr std::uint32_t kSymAltR = 0xFFEA;
constexpr std::uint32_t kSymSuperL = 0xFFEB;
constexpr std::uint32_t kSymSuperR = 0xFFEC;

constexpr int kBackSpaceKeycode = 22; // X11 keycode for BackSpace, as in Ngó Sen

bool isModifierSym(std::uint32_t sym) {
    switch (sym) {
        case kSymShiftL:
        case kSymShiftR:
        case kSymControlL:
        case kSymControlR:
        case kSymMetaL:
        case kSymMetaR:
        case kSymAltL:
        case kSymAltR:
        case kSymSuperL:
        case kSymSuperR:
            return true;
        default:
            return false;
    }
}

bool isResetSym(std::uint32_t sym) {
    switch (sym) {
        case kSymReturn:
        case kSymTab:
        case kSymISOLeftTab:
        case kSymEscape:
        case kSymDelete:
        case kSymKPDelete:
        case kSymInsert:
        case kSymMenu:
        case kSymPrint:
        case kSymScrollLock:
        case kSymPause:
        case kSymNumLock:
        case kSymCapsLock:
        case kSymKPBegin:
            return true;
        default:
            break;
    }
    if (sym >= kSymHome && sym <= kSymEnd) {
        return true;
    }
    if (sym >= kSymF1 && sym <= kSymF35) {
        return true;
    }
    return false;
}

// Map a fcitx keysym to the code point the engine expects. Latin-1 keysyms
// are already code points; 0x01000000+ encodes Unicode directly; numpad
// keys map to their ASCII equivalents so VNI digits work on the keypad.
bool symToCodePoint(std::uint32_t sym, std::uint32_t* cp) {
    if (sym >= 0x20 && sym <= 0xFF) {
        *cp = sym;
        return true;
    }
    if (sym >= 0x01000000 && sym <= 0x0110FFFF) {
        *cp = sym & 0xFFFFFF;
        return true;
    }
    switch (sym) {
        case 0xFF80: // KP_Space
            *cp = ' ';
            return true;
        case 0xFFAA: // KP_Multiply
            *cp = '*';
            return true;
        case 0xFFAB: // KP_Add
            *cp = '+';
            return true;
        case 0xFFAC: // KP_Separator
            *cp = ',';
            return true;
        case 0xFFAD: // KP_Subtract
            *cp = '-';
            return true;
        case 0xFFAE: // KP_Decimal
            *cp = '.';
            return true;
        case 0xFFAF: // KP_Divide
            *cp = '/';
            return true;
        default:
            break;
    }
    if (sym >= 0xFFB0 && sym <= 0xFFB9) { // KP_0..KP_9
        *cp = '0' + (sym - 0xFFB0);
        return true;
    }
    return false;
}

std::string configDir() {
    if (const char* xdg = std::getenv("XDG_CONFIG_HOME"); xdg && *xdg) {
        return std::string(xdg) + "/tuigo";
    }
    const char* home = std::getenv("HOME");
    return (home ? std::string(home) : std::string()) + "/.config/tuigo";
}

// Minimal key=value ini reader, same format as the Go config parser:
// one section, ';' and '#' comments, '[' lines ignored.
std::map<std::string, std::string> readIni(const std::string& path) {
    std::map<std::string, std::string> out;
    std::ifstream in(path);
    std::string line;
    while (std::getline(in, line)) {
        const auto first = line.find_first_not_of(" \t\r\n");
        if (first == std::string::npos) {
            continue;
        }
        const char c = line[first];
        if (c == ';' || c == '#' || c == '[') {
            continue;
        }
        const auto eq = line.find('=', first);
        if (eq == std::string::npos) {
            continue;
        }
        auto key = line.substr(first, eq - first);
        auto value = line.substr(eq + 1);
        const auto keyEnd = key.find_last_not_of(" \t\r\n");
        const auto valStart = value.find_first_not_of(" \t\r\n");
        const auto valEnd = value.find_last_not_of(" \t\r\n");
        if (keyEnd == std::string::npos || valStart == std::string::npos) {
            continue;
        }
        key = key.substr(0, keyEnd + 1);
        value = value.substr(valStart, valEnd - valStart + 1);
        out[key] = value;
    }
    return out;
}

} // namespace

bool TuiGoEngine::FcitxSink::deleteBeforeCursor(const std::string& text) {
    const std::size_t runes = tuigo::utf8Length(text);
    if (runes == 0) {
        return false;
    }
    // Delete-path choice, via quirks (src/quirks.h, ported from Ngó Sen).
    // Behaviour first: what the field reports decides; program names only
    // where nothing reported tells the app apart.
    tuigo::Field field;
    field.frontend = ic_->frontend();
    field.program = ic_->program();
    const auto caps = ic_->capabilityFlags();
    field.surroundingText = caps.test(CapabilityFlag::SurroundingText);
    field.keyEventOrderFix = caps.test(CapabilityFlag::KeyEventOrderFix);
    field.formattedPreedit = caps.test(CapabilityFlag::FormattedPreedit);
    if (field.surroundingText && !tuigo::ignoresDeleteSurrounding(field) &&
        (tuigo::mustUseSurrounding(field) || tuigo::isBrowser(field))) {
        // GTK4 ignores forwarded keys and LibreOffice applies BackSpace
        // late, so the forwarded path cannot work there; browsers
        // implement deleteSurroundingText correctly and synchronously.
        // No snapshot gate: Chrome's snapshot is often stale (it lags
        // behind the typed chars), so the gate fails the sync path AND
        // stalls the async confirmation, making every tone edit pay the
        // full wait — the stutter on fast typing. The delete is relative
        // to the app's real cursor, which is where the typed char is,
        // so it does not need the snapshot.
        ic_->deleteSurroundingText(-static_cast<int>(runes),
                                  static_cast<unsigned int>(runes));
        return false;
    }
    if (!tuigo::ignoresDeleteSurrounding(field) &&
        !tuigo::forwardsBackspaces(field) && field.surroundingText) {
        // Trust deleteSurroundingText only when the app's reported
        // surrounding text actually ends with the text being deleted.
        // A stale/missing report, or an app that advertises the capability
        // without honouring the delete, falls back to BackSpace (which the
        // app cannot silently ignore). Cursor offsets are in characters,
        // so convert to bytes before slicing. deleteSurroundingText shares
        // the input-method channel with commitString, so it is ordered
        // with the commit: synchronous.
        const auto& surr = ic_->surroundingText();
        if (surr.isValid()) {
            const std::string before =
                surr.text().substr(0, tuigo::byteOffset(surr.text(), surr.cursor()));
            if (before.size() >= text.size() &&
                before.compare(before.size() - text.size(), text.size(), text) == 0) {
                ic_->deleteSurroundingText(-static_cast<int>(runes),
                                          static_cast<unsigned int>(runes));
                return false;
            }
        }
    }
    // Forwarded BackSpace travels a different channel (virtual keyboard)
    // than commitString (input-method), so the commit can overtake it and
    // the late BackSpace deletes the replacement instead ("chào" ->
    // "chao"). The caller must wait for the delete to land before
    // committing: returns true = async.
    for (std::size_t i = 0; i < runes; ++i) {
        const Key backSpace(FcitxKey_BackSpace, KeyStates(), kBackSpaceKeycode);
        ic_->forwardKey(backSpace, false);
        ic_->forwardKey(backSpace, true);
    }
    return true;
}

TuiGoConfig loadTuiGoConfig() {
    TuiGoConfig cfg;
    const std::string dir = configDir();
    cfg.macroPath = dir + "/macro.txt";
    cfg.keymapPath = dir + "/keymap.txt";
    const auto ini = readIni(dir + "/tuigo.ini");
    const auto getInt = [&ini](const char* name, int dflt) {
        const auto it = ini.find(name);
        if (it == ini.end()) {
            return dflt;
        }
        try {
            return std::stoi(it->second);
        } catch (...) {
            return dflt;
        }
    };
    const auto getBool = [&ini](const char* name, bool dflt) {
        const auto it = ini.find(name);
        if (it == ini.end()) {
            return dflt;
        }
        return it->second == "1";
    };
    cfg.vietKey = getBool("VietKey", cfg.vietKey);
    cfg.inputMethod = getInt("InputMethod", cfg.inputMethod);
    cfg.spellCheck = getBool("SpellCheck", cfg.spellCheck);
    cfg.modernStyle = getBool("ModernStyle", cfg.modernStyle);
    cfg.freeMarking = getBool("FreeMarking", cfg.freeMarking);
    cfg.macroEnabled = getBool("MacroEnabled", cfg.macroEnabled);
    cfg.autoNonVnRestore = getBool("AutoNonVnRestore", cfg.autoNonVnRestore);
    cfg.autoCap = getBool("AutoCap", cfg.autoCap);
    return cfg;
}

TuiGoEngine::State::State(InputContext* ic, const TuiGoConfig& cfg, Instance* instance)
    : sink(ic),
      machine(&engine, &sink, tuigo::SenOptions{cfg.autoCap}),
      ic(ic),
      instance(instance) {
    applyConfig(cfg);
}

void TuiGoEngine::State::applyConfig(const TuiGoConfig& cfg) {
    engine.setViet(cfg.vietKey);
    engine.setInputMethod(cfg.inputMethod);
    engine.setOptions(cfg.spellCheck, cfg.modernStyle, cfg.freeMarking, cfg.macroEnabled,
                      cfg.autoNonVnRestore);
    engine.loadMacroFile(cfg.macroPath);
    engine.loadKeymapFile(cfg.keymapPath);
    machine.setViet(cfg.vietKey);
}

TuiGoEngine::TuiGoEngine(Instance* instance)
    : instance_(instance), config_(loadTuiGoConfig()),
      factory_([this](InputContext& ic) {
          return new StateProperty(&ic, config_, instance_); // NOLINT
      }) {
    instance_->inputContextManager().registerProperty("TuiGoState", &factory_);
}

TuiGoEngine::State* TuiGoEngine::stateFor(InputContext* ic) {
    return &ic->propertyFor(&factory_)->state();
}

void TuiGoEngine::cancelWait(State* state) {
    // Never called from inside the timer callback: destroying the timer
    // there would destroy the source being dispatched.
    state->waitTimer_.reset();
    state->waitTimerRunning_ = false;
    state->bufferedKeys_.clear();
    // Never silently drop a pending commit: the delete already landed in
    // the app, so dropping the replacement loses the user's character.
    // E.g. typing "dd" in a Chrome chat input: the BackSpace deletes the
    // 'd', then a framework reset (input-context reset/deactivate) lands
    // before the flush, the pending "đ" is discarded, and the field is
    // left completely empty (""). Flush the replacement first, then
    // reset the machine.
    if (state->machine.deletePending()) {
        const std::string out = state->machine.takePendingOut();
        if (!out.empty()) {
            state->ic->commitString(out);
        }
    }
    state->machine.reset();
}

void TuiGoEngine::suspendWait(State* state) {
    // Never called from inside the timer callback: destroying the timer
    // there would destroy the source being dispatched.
    state->waitTimer_.reset();
    state->waitTimerRunning_ = false;
    state->bufferedKeys_.clear();
    // Deliberately NOT flushing the pending commit and NOT resetting the
    // machine: the context is deactivated, so a commit now would go to a
    // deactivated context and be dropped on Wayland (mouse-click tab
    // switch inside the ~100 ms wait window -> the tone character is
    // lost, intermittently). The pending commit is held; activate()'s
    // cancelWait flushes it to the now-active context, reliably.
}

void TuiGoEngine::activate(const InputMethodEntry& /*entry*/, InputContextEvent& event) {
    auto* state = stateFor(event.inputContext());
    cancelWait(state);
    state->applyConfig(config_);
}

void TuiGoEngine::deactivate(const InputMethodEntry& /*entry*/, InputContextEvent& event) {
    suspendWait(stateFor(event.inputContext()));
}

void TuiGoEngine::reset(const InputMethodEntry& /*entry*/, InputContextEvent& event) {
    cancelWait(stateFor(event.inputContext()));
}

void TuiGoEngine::reloadConfig() {
    config_ = loadTuiGoConfig();
}

void TuiGoEngine::keyEvent(const InputMethodEntry& /*entry*/, KeyEvent& keyEvent) {
    if (keyEvent.isRelease()) {
        return;
    }
    auto* state = stateFor(keyEvent.inputContext());
    // A delete is in flight: the machine's model assumes the pending
    // commit already landed. Buffer the key and replay it after the flush
    // instead of feeding it against a half-applied state.
    if (state->machine.deletePending()) {
        if (!state->waitTimerRunning_) {
            // Stale pending: held across a deactivate whose activate
            // never flushed it (or a timer that never ran). Buffering
            // here would swallow every key forever — the input method
            // looks dead until an E/V toggle forces a reset. Flush the
            // held commit now and handle this key normally instead.
            const std::string out = state->machine.takePendingOut();
            if (!out.empty()) {
                state->ic->commitString(out);
            }
        } else {
            state->bufferedKeys_.push_back(keyEvent.rawKey());
            keyEvent.filterAndAccept();
            return;
        }
    }
    if (handleKey(state, keyEvent.rawKey())) {
        keyEvent.filterAndAccept();
    }
    if (state->machine.deletePending()) {
        startWait(state);
    }
}

bool TuiGoEngine::handleKey(State* state, const Key& key) {
    const std::uint32_t sym = key.sym();
    const std::uint32_t states = key.states();

    // Ctrl/Alt/Super shortcuts bypass the engine and invalidate the word.
    const std::uint32_t ctrl = static_cast<std::uint32_t>(KeyState::Ctrl);
    const std::uint32_t alt = static_cast<std::uint32_t>(KeyState::Alt);
    const std::uint32_t super = static_cast<std::uint32_t>(KeyState::Super);
    if (states & (ctrl | alt | super)) {
        state->machine.reset();
        return false;
    }
    // Modifier-only keys: pass through, don't disturb the word.
    if (isModifierSym(sym)) {
        return false;
    }
    if (sym == kSymBackSpace) {
        return state->machine.backspace();
    }
    if (isResetSym(sym)) {
        state->machine.reset();
        return false;
    }
    std::uint32_t cp = 0;
    if (!symToCodePoint(sym, &cp)) {
        // Keysym we can't map (dead keys, non-Latin layouts): drop the
        // word and let the app handle the key itself.
        state->machine.reset();
        return false;
    }
    return state->machine.feed(cp);
    // Not consumed: the caller forwards the key to the app, which inserts it.
}

// Wait timing: poll every 5 ms, give up after 100 ms and flush anyway.
// Apps that never report surrounding text get a fixed 15 ms wait.
constexpr std::uint64_t kWaitPollUs = 5000;
constexpr std::uint64_t kWaitTimeoutUs = 100000;
constexpr std::uint64_t kWaitNoSurroundingUs = 15000;

void TuiGoEngine::startWait(State* state) {
    state->waitStarted_ = std::chrono::steady_clock::now();
    // A new delete needs its own pre-delete observation.
    state->waitSeenPreDelete_ = false;
    if (state->waitTimerRunning_) {
        // Called from the timer callback via replay: rearm the same source,
        // never destroy it from inside its own callback.
        state->waitTimer_->setTime(now(CLOCK_MONOTONIC) + kWaitPollUs);
        state->waitTimer_->setOneShot();
        return;
    }
    // Called from keyEvent: any old (stopped) timer can be destroyed safely.
    state->waitTimer_.reset();
    state->waitTimerRunning_ = true;
    state->waitTimer_ = state->instance->eventLoop().addTimeEvent(
        CLOCK_MONOTONIC, now(CLOCK_MONOTONIC) + kWaitPollUs, 1000,
        [this, state](EventSourceTime*, std::uint64_t) { return onWaitTimer(state); });
}

bool TuiGoEngine::deletionConfirmed(State* state) const {
    const auto& surr = state->ic->surroundingText();
    if (!surr.isValid()) {
        return false;
    }
    const std::string before =
        surr.text().substr(0, tuigo::byteOffset(surr.text(), surr.cursor()));
    // The machine's committed_ is (old word - deleted) + pending commit,
    // so base is the expected app text after the delete lands.
    const std::string& committed = state->machine.committedWord();
    const std::string& pending = state->machine.pendingOut();
    if (committed.size() < pending.size()) {
        return false;
    }
    const std::string base = committed.substr(0, committed.size() - pending.size());
    const std::string& deleted = state->machine.pendingDeleted();
    const auto endsWith = [](const std::string& s, const std::string& suffix) {
        return s.size() >= suffix.size() &&
               s.compare(s.size() - suffix.size(), suffix.size(), suffix) == 0;
    };
    if (!deleted.empty() && endsWith(before, base + deleted)) {
        // Pre-delete state observed: the text to delete is still there.
        state->waitSeenPreDelete_ = true;
        return false;
    }
    // Never confirm from a snapshot that predates the typed char. With an
    // empty base (e.g. "dd" -> "đ": the whole word is the deleted text),
    // endsWith(before, base) below is vacuously true, so a stale snapshot
    // would "confirm" the delete before the BackSpace lands, the commit
    // would flush early, and the late BackSpace would eat the replacement
    // ("đ" lost). Only trust a post-delete snapshot after the pre-delete
    // state was seen; otherwise keep waiting until the timeout flush.
    if (!state->waitSeenPreDelete_) {
        return false;
    }
    return endsWith(before, base);
}

bool TuiGoEngine::onWaitTimer(State* state) {
    if (!state->machine.deletePending()) {
        state->waitTimerRunning_ = false;
        return false; // stale fire; stop
    }
    const auto elapsed = static_cast<std::uint64_t>(
        std::chrono::duration_cast<std::chrono::microseconds>(
            std::chrono::steady_clock::now() - state->waitStarted_)
            .count());
    const auto& surr = state->ic->surroundingText();
    bool ready = false;
    if (!surr.isValid()) {
        ready = elapsed >= kWaitNoSurroundingUs;
    } else {
        ready = deletionConfirmed(state) || elapsed >= kWaitTimeoutUs;
    }
    if (!ready) {
        // Not yet: poll again. Rearm the same source; return true to stay alive.
        state->waitTimer_->setTime(now(CLOCK_MONOTONIC) + kWaitPollUs);
        state->waitTimer_->setOneShot();
        return true;
    }
    flushWait(state);
    if (state->machine.deletePending()) {
        return true; // a replayed key started a new wait; timer rearmed
    }
    state->waitTimerRunning_ = false;
    return false;
}

void TuiGoEngine::flushWait(State* state) {
    const std::string out = state->machine.takePendingOut();
    if (!out.empty()) {
        state->ic->commitString(out);
    }
    // Replay buffered keys in order. A replayed key may start a new wait;
    // the rest stay buffered for that wait.
    auto keys = std::move(state->bufferedKeys_);
    state->bufferedKeys_.clear();
    for (std::size_t i = 0; i < keys.size(); ++i) {
        const bool consumed = handleKey(state, keys[i]);
        if (!consumed) {
            // The machine didn't eat it: forward the original key so the
            // app inserts it, exactly as if it had arrived normally.
            state->ic->forwardKey(keys[i], false);
            state->ic->forwardKey(keys[i], true);
        }
        // A key with Alt/Super is a window-manager shortcut (Alt+Tab):
        // it switches focus away, so the remaining buffered keys were
        // typed for the new focus, not this input context. Drop them
        // rather than replaying them into the old window. (Ctrl is
        // excluded: Ctrl+C/V/... are app shortcuts that keep focus here,
        // and the keys after them still belong to this context.)
        const std::uint32_t alt = static_cast<std::uint32_t>(KeyState::Alt);
        const std::uint32_t super = static_cast<std::uint32_t>(KeyState::Super);
        if (keys[i].states() & (alt | super)) {
            break;
        }
        if (state->machine.deletePending()) {
            state->bufferedKeys_.assign(keys.begin() + i + 1, keys.end());
            startWait(state);
            return;
        }
    }
}

std::string TuiGoEngine::subModeLabelImpl(const InputMethodEntry& /*entry*/,
                                          InputContext& inputContext) {
    return stateFor(&inputContext)->machine.viet() ? "VI" : "EN";
}

} // namespace fcitx
