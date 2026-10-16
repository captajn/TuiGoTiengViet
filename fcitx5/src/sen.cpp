#include "sen.h"

#include "utf8.h"

namespace tuigo {

// Remove the trailing code point from s if it equals cp; otherwise return
// s unchanged. Used to drop the boundary char (e.g. space) from a word a
// consumed edit just committed, so the remembered word is the word itself.
std::string stripTrailingCodePoint(const std::string& s, std::uint32_t cp) {
    if (lastUnits(s, 1) == utf8FromCodePoint(cp)) {
        return eraseLastUnits(s, 1, nullptr);
    }
    return s;
}

SenMachine::SenMachine(Engine* engine, Sink* sink, SenOptions options)
    : engine_(engine), sink_(sink), options_(options) {}

bool SenMachine::feed(std::uint32_t codePoint) {
    if (!viet_) {
        return false;
    }
    std::uint32_t ch = codePoint;
    const bool forceCap = options_.autoCap && capNext_ && ch >= 'a' && ch <= 'z';
    if (forceCap) {
        ch -= 'a' - 'A';
    }
    Edit edit = engine_->processKey(ch);

    if (options_.autoCap) {
        switch (ch) {
            case '.':
            case '!':
            case '?':
                capNext_ = true;
                break;
            case ' ':
                // pending capitalisation survives whitespace
                break;
            default:
                capNext_ = false;
                break;
        }
    }

    if (edit.consumed) {
        applyEdit(edit);
        // Don't clear while a delete is in flight: committed_ holds the
        // optimistic post-flush state the pending commit belongs to.
        if (engine_->atWordBeginning() && !deletePending_) {
            // Word boundary after a consumed edit (e.g. a macro ending
            // with a space): remember the word for a later reopen, minus
            // the boundary char the edit just committed.
            lastWord_ = stripTrailingCodePoint(committed_, ch);
            committed_.clear();
        }
        return true;
    }

    if (engine_->atWordBeginning()) {
        // Word-break or reset char: the word is already committed;
        // forward the key so the app inserts the terminator itself.
        // Remember the word: BackSpace may reopen it (see backspace()).
        lastWord_ = committed_;
        committed_.clear();
        return false;
    }
    if (forceCap) {
        // The app would insert the unmodified key if forwarded; commit
        // the capitalised char ourselves instead.
        const std::string cap = utf8FromCodePoint(ch);
        committed_ += cap;
        sink_->commitText(cap);
        return true;
    }
    // Engine recorded the char but passed it through (e.g. the first
    // letter of a word). Forward it so the app commits it, and track it:
    // a later restore may still delete it from committed text.
    committed_ += utf8FromCodePoint(ch);
    return false;
}

bool SenMachine::backspace() {
    if (!viet_) {
        return false;
    }
    const bool wasAtBeginning = engine_->atWordBeginning();
    Edit edit = engine_->backspace();
    if (wasAtBeginning && !engine_->atWordBeginning()) {
        // BackSpace reopened the previous word: the engine treats it as
        // live again. The backspace itself deletes the boundary (space)
        // from the app, so forward it; do NOT applyEdit, whose backs
        // refers to the boundary, not the word. Restore the machine's
        // model of the word, or the next tone key's backs/out would be
        // applied to an empty word and produce garbage ("chàoaof").
        if (!lastWord_.empty()) {
            committed_ = lastWord_;
        }
        return false;
    }
    if (!edit.consumed && committed_.empty()) {
        // Buffer empty — the app deletes a committed char itself.
        return false;
    }
    // The engine shrank its buffer (backs is populated even when the
    // result is not consumed); mirror the edit on committed text.
    applyEdit(edit);
    if (engine_->atWordBeginning() && !deletePending_) {
        // The word was backspaced away: nothing left to reopen to.
        committed_.clear();
        lastWord_.clear();
    }
    return true;
}

void SenMachine::reset() {
    capNext_ = false;
    engine_->reset();
    committed_.clear();
    lastWord_.clear();
    deletePending_ = false;
    pendingDeleted_.clear();
    pendingOut_.clear();
}

std::string SenMachine::takePendingOut() {
    deletePending_ = false;
    pendingDeleted_.clear();
    std::string out = std::move(pendingOut_);
    pendingOut_.clear();
    return out;
}

void SenMachine::setViet(bool viet) {
    if (viet == viet_) {
        return;
    }
    viet_ = viet;
    engine_->setViet(viet);
    engine_->reset();
    committed_.clear();
    lastWord_.clear();
    capNext_ = false;
}

void SenMachine::applyEdit(const Edit& edit) {
    int backs = edit.backs;
    if (backs < 0) {
        backs = 0;
    }
    const std::string erased = lastUnits(committed_, static_cast<std::size_t>(backs));
    committed_ = eraseLastUnits(committed_, static_cast<std::size_t>(backs), nullptr);
    if (!erased.empty() && sink_->deleteBeforeCursor(erased)) {
        // Async delete (forwarded keys): the replacement commit must wait
        // until the delete lands, or the app applies the commit first and
        // the late BackSpace deletes the replacement instead ("chào" ->
        // "chao"). Hold the commit; the shell flushes via takePendingOut().
        deletePending_ = true;
        pendingDeleted_ = erased;
        pendingOut_ = edit.out;
        committed_ += edit.out; // optimistic: app state after the flush
        return;
    }
    if (!edit.out.empty()) {
        sink_->commitText(edit.out);
    }
    committed_ += edit.out;
}

} // namespace tuigo
