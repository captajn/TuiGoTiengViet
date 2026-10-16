// Sen (no-preedit) state machine for the fcitx5 addon.
//
// This is the Gõ Sen mechanism from Ngó Sen, reimplemented for Tui Go's
// engine: commit immediately, and rewrite already-committed text with
// delete + commit when a tone/mark edit changes the word. It never shows
// preedit. It does not depend on fcitx5, so it is unit-tested with a fake
// engine and a fake sink (see ../test/sen_test.cpp).
#pragma once

#include <cstdint>
#include <string>

namespace tuigo {

// One edit from the typing engine. backs counts UTF-16 units (the Tui Go
// engine contract); out is UTF-8 text.
struct Edit {
    bool consumed = false;
    int backs = 0;
    std::string out;
};

// Typing engine behind the Sen machine. Implemented by GoEngine (Go shim
// over tuigo/engine); tests use a fake.
class Engine {
  public:
    virtual ~Engine() = default;
    virtual Edit processKey(std::uint32_t codePoint) = 0;
    virtual Edit backspace() = 0;
    virtual bool atWordBeginning() const = 0;
    virtual void reset() = 0;
    virtual void setViet(bool viet) = 0;
};

// Output side of the Sen machine. The fcitx5 addon implements this on an
// InputContext; tests use a buffer sink.
class Sink {
  public:
    virtual ~Sink() = default;
    // Insert text at the caret.
    virtual void commitText(const std::string& text) = 0;
    // Delete `text` immediately before the caret. The sink gets the exact
    // text (not just a count) so a real input-method shell can verify the
    // app's surrounding text ends with it before trusting
    // deleteSurroundingText: some apps advertise the capability but
    // silently ignore the delete, which would duplicate letters.
    // Returns true if the delete was issued asynchronously (forwarded
    // keys: the caller must wait for the delete to land before committing
    // the replacement); false if it applied synchronously.
    virtual bool deleteBeforeCursor(const std::string& text) = 0;
};

struct SenOptions {
    bool autoCap = false; // capitalise first letter after ". " "! " "? "
};

class SenMachine {
  public:
    // engine and sink are borrowed, not owned.
    SenMachine(Engine* engine, Sink* sink, SenOptions options = {});

    // Feed one printable code point. Returns true if the key was consumed
    // (do NOT forward it to the app); false means forward it.
    bool feed(std::uint32_t codePoint);
    // Returns true if consumed (do NOT forward the BackSpace).
    bool backspace();
    void reset();
    void setViet(bool viet);

    bool viet() const {
        return viet_;
    }
    // True while a delete is in flight and its replacement commit is held.
    // The caller must not feed keys while this is true; the shell buffers
    // them and replays after takePendingOut().
    bool deletePending() const {
        return deletePending_;
    }
    // Text of the in-flight delete, for surrounding-text confirmation.
    const std::string& pendingDeleted() const {
        return pendingDeleted_;
    }
    // Held replacement commit. Empty when nothing is pending.
    const std::string& pendingOut() const {
        return pendingOut_;
    }
    // Take the held commit. Clears the pending state.
    std::string takePendingOut();
    // Rendered form of the current word as committed in the app. Used by
    // tests; the addon never needs it.
    const std::string& committedWord() const {
        return committed_;
    }

  private:
    void applyEdit(const Edit& edit);

    Engine* engine_;
    Sink* sink_;
    SenOptions options_;
    std::string committed_;
    // The last committed word before a word boundary (space/enter). The
    // engine reopens that word when BackSpace deletes the boundary, so
    // the machine restores this as committed_ instead of treating the
    // word as gone; otherwise the next tone key's backs/out would be
    // applied to an empty word and produce garbage ("chàoaof").
    std::string lastWord_;
    bool viet_ = true;
    bool capNext_ = false;
    bool deletePending_ = false;
    std::string pendingDeleted_;
    std::string pendingOut_;
};

} // namespace tuigo
