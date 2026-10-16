// fcitx5 addon for Tui Go: one SenMachine (no preedit) per input context,
// driven by Tui Go's Go typing engine. The typing logic lives in sen.cpp
// and does not depend on fcitx5; this file is only the thin shell between
// fcitx5's InputContext/KeyEvent and the SenMachine.
#pragma once

#include "sen.h"
#include "tuigo-engine.h"

#include <chrono>
#include <memory>
#include <string>
#include <utility>
#include <vector>

#include <fcitx-config/iniparser.h>
#include <fcitx/addonfactory.h>
#include <fcitx/addonmanager.h>
#include <fcitx/event.h>
#include <fcitx-utils/event.h>
#include <fcitx-utils/key.h>
#include <fcitx/inputcontext.h>
#include <fcitx/inputmethodengine.h>
#include <fcitx/instance.h>
#include <fcitx-utils/misc.h>

namespace fcitx {

struct TuiGoConfig {
    bool vietKey = true;
    int inputMethod = 0; // ImTelex
    bool spellCheck = true;
    bool modernStyle = true;
    bool freeMarking = true;
    bool macroEnabled = true;
    bool autoNonVnRestore = false;
    bool autoCap = false;
    std::string macroPath;
    std::string keymapPath;
};

// Reads the shared tuigo.ini (same file the IBus engine and the settings
// GUI use): $XDG_CONFIG_HOME/tuigo/tuigo.ini, default ~/.config/tuigo/.
TuiGoConfig loadTuiGoConfig();

class TuiGoEngine final : public InputMethodEngineV2 {
  public:
    explicit TuiGoEngine(Instance* instance);
    ~TuiGoEngine() override = default;

    TuiGoEngine(const TuiGoEngine&) = delete;
    TuiGoEngine& operator=(const TuiGoEngine&) = delete;

    void activate(const InputMethodEntry& entry, InputContextEvent& event) override;
    void deactivate(const InputMethodEntry& entry, InputContextEvent& event) override;
    void keyEvent(const InputMethodEntry& entry, KeyEvent& keyEvent) override;
    void reset(const InputMethodEntry& entry, InputContextEvent& event) override;
    void reloadConfig() override;

    const Configuration* getConfig() const override {
        return nullptr; // no fcitx5 config UI; use tuigo.ini / tuigo-settings
    }
    const Configuration* getSubConfig(const std::string& /*path*/) const override {
        return nullptr;
    }
    void setConfig(const RawConfig& /*config*/) override {}
    void setSubConfig(const std::string& /*path*/, const RawConfig& /*config*/) override {}
    std::string subMode(const InputMethodEntry& /*entry*/, InputContext& /*inputContext*/) override {
        return "Tui Gõ";
    }
    std::string subModeIconImpl(const InputMethodEntry& /*entry*/,
                                InputContext& inputContext) override {
        // TuiGo's own E/V tray badge (app/tray_icon.go: jade badge + V
        // for Vietnamese, outlined neutral keycap + E for English),
        // ported to PNGs in data/icons/. Mode is also shown by
        // subModeLabelImpl ("VI" / "EN").
        return stateFor(&inputContext)->machine.viet() ? "fcitx-tuigo-vi" : "fcitx-tuigo-en";
    }
    std::string subModeLabelImpl(const InputMethodEntry& /*entry*/, InputContext& inputContext) override;

  private:
    // fcitx5 sink: commit + delete on a real InputContext. Deletion prefers
    // deleteSurroundingText (exact, one call) only when the app's reported
    // surrounding text ends with the text being deleted; otherwise it
    // falls back to forwarded BackSpace presses. On Wayland/XIM it always
    // forwards BackSpace: deleteSurroundingText is unreliable there (e.g.
    // Telegram silently ignores it, duplicating letters: "chào" -> "chaào").
    // Same two paths Ngó Sen's performReplacement chooses between.
    class FcitxSink final : public tuigo::Sink {
      public:
        explicit FcitxSink(InputContext* ic) : ic_(ic) {}

        void commitText(const std::string& text) override {
            ic_->commitString(text);
        }
        bool deleteBeforeCursor(const std::string& text) override;

      private:
        InputContext* ic_;
    };

    struct State {
        tuigo::GoEngine engine;
        FcitxSink sink;
        tuigo::SenMachine machine;
        InputContext* ic;
        Instance* instance;
        // Async-delete wait: while a delete is in flight, incoming keys are
        // buffered here and replayed after the pending commit flushes.
        // The timer is owned by the State: destroying the State cancels it.
        // It is never reset from inside its own callback; the callback
        // rearms the same source via setTime/setOneShot.
        std::unique_ptr<EventSourceTime> waitTimer_;
        bool waitTimerRunning_ = false;
        std::chrono::steady_clock::time_point waitStarted_;
        std::vector<Key> bufferedKeys_;
        // True once a surrounding-text snapshot showed the text to delete
        // still present (pre-delete state). deletionConfirmed only trusts
        // a post-delete snapshot after the pre-delete state was seen;
        // without this, a stale snapshot predating the typed char would
        // "confirm" the delete before the BackSpace lands.
        bool waitSeenPreDelete_ = false;

        State(InputContext* ic, const TuiGoConfig& cfg, Instance* instance);
        void applyConfig(const TuiGoConfig& cfg);
    };

    struct StateProperty final : public InputContextProperty {
        StateProperty(InputContext* ic, const TuiGoConfig& cfg, Instance* instance)
            : state_(ic, cfg, instance) {}
        State& state() {
            return state_;
        }

      private:
        State state_;
    };

    State* stateFor(InputContext* ic);
    // Route one key through the machine. Returns true if consumed.
    bool handleKey(State* state, const Key& key);
    // Async-delete wait (Ngó Sen's timer approach, minimal): poll the
    // surrounding text until the delete lands, then flush + replay.
    void startWait(State* state);
    bool onWaitTimer(State* state);
    bool deletionConfirmed(State* state) const;
    void flushWait(State* state);
    // Cancel a pending wait and reset the machine (focus change, switch).
    // Never called from inside the timer callback.
    void cancelWait(State* state);
    // Stop a pending wait WITHOUT flushing or resetting: the input
    // context lost focus (deactivate), so a commit flushed now would go
    // to a deactivated context and be dropped on Wayland. The pending
    // commit is held and flushed by cancelWait when the context
    // reactivates (activate). Never called from inside the timer callback.
    void suspendWait(State* state);

    Instance* instance_;
    TuiGoConfig config_;
    FactoryFor<StateProperty> factory_;
};

class TuiGoFactory : public AddonFactory {
  public:
    AddonInstance* create(AddonManager* manager) override {
        return new TuiGoEngine(manager->instance()); // NOLINT
    }
};

} // namespace fcitx
