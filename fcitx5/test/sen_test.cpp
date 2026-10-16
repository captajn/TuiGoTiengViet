// End-to-end test for the Sen machine: the REAL Go typing engine
// (GoEngine over the c-archive shim) + a fake app sink. No fcitx5 needed.
//
// The fake sink models an app buffer: commitText appends, deleteBeforeCursor
// erases runes before the caret. Keys the machine does not consume are
// inserted by the app itself, like a real client would.
#include "sen.h"
#include "tuigo-engine.h"
#include "utf8.h"
#include "quirks.h"

#include <cstdio>
#include <filesystem>
#include <fstream>
#include <string>
#include <vector>

namespace {

int failures = 0;

void check(bool ok, const std::string& what, const std::string& got, const std::string& want) {
    if (!ok) {
        ++failures;
        std::fprintf(stderr, "FAIL %s: got '%s', want '%s'\n", what.c_str(), got.c_str(), want.c_str());
    }
}

class BufferSink final : public tuigo::Sink {
  public:
    void commitText(const std::string& text) override {
        buffer_ += text;
    }
    bool deleteBeforeCursor(const std::string& text) override {
        // Models a correct app: it deletes exactly the text it is asked to.
        // Synchronous: the caller may commit right away.
        const int runes = static_cast<int>(tuigo::utf8Length(text));
        if (runes <= 0) {
            return false;
        }
        const std::size_t total = tuigo::utf8Length(buffer_);
        if (static_cast<std::size_t>(runes) >= total) {
            buffer_.clear();
            return false;
        }
        const std::size_t keep = total - static_cast<std::size_t>(runes);
        std::size_t pos = 0;
        std::size_t idx = 0;
        while (pos < buffer_.size() && idx < keep) {
            const auto c = static_cast<unsigned char>(buffer_[pos]);
            std::size_t adv = 1;
            if ((c & 0xE0) == 0xC0) {
                adv = 2;
            } else if ((c & 0xF0) == 0xE0) {
                adv = 3;
            } else if ((c & 0xF8) == 0xF0) {
                adv = 4;
            }
            pos += adv;
            ++idx;
        }
        buffer_.resize(pos);
        return false;
    }

    const std::string& buffer() const {
        return buffer_;
    }

  private:
    std::string buffer_;
};

// Models an async delete (forwarded BackSpace on Wayland): the delete is
// "sent" but the replacement commit must wait, like the fcitx5 shell does.
class AsyncSink final : public tuigo::Sink {
  public:
    void commitText(const std::string& text) override {
        buffer_ += text;
    }
    bool deleteBeforeCursor(const std::string& text) override {
        deleted_.push_back(text);
        return true;
    }
    // Models the app applying the forwarded BackSpace, some time later.
    // The shell only flushes the pending commit after this lands.
    void applyDelete(const std::string& text) {
        if (buffer_.size() >= text.size() &&
            buffer_.compare(buffer_.size() - text.size(), text.size(), text) == 0) {
            buffer_.resize(buffer_.size() - text.size());
        }
    }

    const std::string& buffer() const {
        return buffer_;
    }
    const std::vector<std::string>& deleted() const {
        return deleted_;
    }

  private:
    std::string buffer_;
    std::vector<std::string> deleted_;
};

// Like BufferSink, but fails the test if the machine ever asks to delete
// text that is not the buffer's suffix. Pins the Sink contract the fcitx5
// shell relies on to verify surrounding text before deleteSurroundingText.
class StrictSink final : public tuigo::Sink {
  public:
    void commitText(const std::string& text) override {
        buffer_ += text;
    }
    bool deleteBeforeCursor(const std::string& text) override {
        if (buffer_.size() < text.size() ||
            buffer_.compare(buffer_.size() - text.size(), text.size(), text) != 0) {
            suffixOk_ = false;
            return false;
        }
        deleted_.push_back(text);
        buffer_.resize(buffer_.size() - text.size());
        return false;
    }

    const std::string& buffer() const {
        return buffer_;
    }
    bool suffixOk() const {
        return suffixOk_;
    }
    const std::vector<std::string>& deleted() const {
        return deleted_;
    }

  private:
    std::string buffer_;
    std::vector<std::string> deleted_;
    bool suffixOk_ = true;
};

struct Harness {
    tuigo::GoEngine engine;
    BufferSink sink;
    tuigo::SenMachine machine;

    Harness() : machine(&engine, &sink) {
        engine.setViet(true);
        engine.setOptions(true, true, true, true, false);
    }
    explicit Harness(tuigo::SenOptions opts) : machine(&engine, &sink, opts) {
        engine.setViet(true);
        engine.setOptions(true, true, true, true, false);
    }

    void type(const std::string& text) {
        for (unsigned char c : text) {
            if (!machine.feed(c)) {
                sink.commitText(std::string(1, static_cast<char>(c)));
            }
        }
    }
    void backspace() {
        if (!machine.backspace()) {
            // Placeholder: only its 1-rune length matters to the fake sink.
            sink.deleteBeforeCursor(" ");
        }
    }
};

} // namespace

int main() {
    {
        Harness h;
        h.type("tieengs ");
        check(h.sink.buffer() == "tiếng ", "tieengs", h.sink.buffer(), "tiếng ");
        check(h.machine.committedWord().empty(), "word cleared at boundary", h.machine.committedWord(), "");
    }
    {
        Harness h;
        h.type("vieetj ");
        check(h.sink.buffer() == "việt ", "vieetj", h.sink.buffer(), "việt ");
    }
    {
        Harness h;
        h.type("dd ");
        check(h.sink.buffer() == "đ ", "dd", h.sink.buffer(), "đ ");
    }
    {
        Harness h;
        h.type("vieet"); // no tone key -> "viêt", not "việt"
        check(h.sink.buffer() == "viêt", "vieet", h.sink.buffer(), "viêt");
        h.backspace(); // deletes 't', leaving "viê"
        check(h.sink.buffer() == "viê", "backspace", h.sink.buffer(), "viê");
    }
    {
        Harness h;
        h.type("vieet ");
        h.backspace(); // empty buffer: forwarded, app deletes the space
        check(h.sink.buffer() == "viêt", "backspace at boundary", h.sink.buffer(), "viêt");
    }
    {
        // BackSpace after a space reopens the previous word (the engine
        // treats it as live again). Re-toning it must rewrite the word,
        // not append garbage: "chào" + space + BS + 'f' -> "chaof".
        // Without the reopen restore this produced "chàoaof".
        Harness h;
        h.type("chaof ");
        h.backspace(); // deletes the space, reopens "chào"
        check(h.sink.buffer() == "chào", "backspace reopens word", h.sink.buffer(), "chào");
        h.type("f");
        check(h.sink.buffer() == "chaof", "re-tone reopened word", h.sink.buffer(), "chaof");
    }
    {
        Harness h;
        h.type("chaof ");
        h.backspace(); // reopen "chào"
        h.type("n"); // append a plain consonant to the reopened word
        check(h.sink.buffer() == "chàon", "type after reopen", h.sink.buffer(), "chàon");
        // Note: 'x' after the final consonant is literal here (strict
        // Telex: the tone goes before the final consonant), both with
        // and without the reopen — verified against the engine directly.
    }
    {
        // Macro: ko + space -> không (space included in the expansion).
        const auto dir = std::filesystem::temp_directory_path() / "tuigo-sen-test";
        std::filesystem::create_directories(dir);
        const auto macro = dir / "macro.txt";
        {
            std::ofstream out(macro);
            out << "ko:không\n";
        }
        Harness h;
        h.engine.loadMacroFile(macro.string());
        h.type("ko ");
        check(h.sink.buffer() == "không ", "macro", h.sink.buffer(), "không ");
        std::filesystem::remove_all(dir);
    }
    {
        Harness h;
        h.engine.setOptions(true, true, true, true, true); // AutoNonVnRestore
        h.type("ddasd ");
        check(h.sink.buffer() == "ddasd ", "non-vn restore", h.sink.buffer(), "ddasd ");
    }
    {
        Harness h;
        h.engine.setInputMethod(1); // VNI
        h.type("vie65t ");
        check(h.sink.buffer() == "việt ", "vni", h.sink.buffer(), "việt ");
    }
    {
        Harness h(tuigo::SenOptions{true}); // AutoCap
        h.type("ok. ban");
        check(h.sink.buffer() == "ok. Ban", "autocap", h.sink.buffer(), "ok. Ban");
    }
    {
        Harness h;
        h.machine.setViet(false);
        h.type("hello ");
        check(h.sink.buffer() == "hello ", "english passthrough", h.sink.buffer(), "hello ");
    }
    {
        // Async delete: the machine must hold the replacement commit while
        // the delete is in flight, and hand it over via takePendingOut().
        // The shell flushes it only after the delete lands; committing
        // early is what turned "chào" into "chao" on Wayland.
        tuigo::GoEngine engine;
        AsyncSink sink;
        tuigo::SenMachine machine(&engine, &sink);
        engine.setViet(true);
        engine.setOptions(true, true, true, true, false);
        for (unsigned char c : std::string("chaof")) {
            if (machine.feed(c)) {
                if (machine.deletePending()) {
                    // Shell flush: the delete landed in the app first,
                    // only then the held commit goes out. Committing
                    // before the delete lands gives "chaoào".
                    sink.applyDelete(machine.pendingDeleted());
                    sink.commitText(machine.takePendingOut());
                }
            } else {
                sink.commitText(std::string(1, static_cast<char>(c)));
            }
        }
        check(!machine.deletePending(), "pending cleared after flush",
              machine.deletePending() ? "pending" : "clear", "clear");
        check(sink.buffer() == "chào", "async delete + deferred commit", sink.buffer(), "chào");
        std::string deleted;
        for (const auto& d : sink.deleted()) {
            deleted += "[" + d + "]";
        }
        check(deleted == "[ao]", "async deleted texts", deleted, "[ao]");
    }
    {
        // Regression: "chào các con vợ" typed in Telegram came out as
        // "chaào caác con voơợ" because deleteSurroundingText was a no-op
        // there. The machine must ask to delete exactly the text it
        // committed, so the shell can verify it against surrounding text.
        tuigo::GoEngine engine;
        StrictSink sink;
        tuigo::SenMachine machine(&engine, &sink);
        engine.setViet(true);
        engine.setOptions(true, true, true, true, false);
        for (unsigned char c : std::string("chaof casc con vowj ")) {
            if (!machine.feed(c)) {
                sink.commitText(std::string(1, static_cast<char>(c)));
            }
        }
        check(sink.buffer() == "chào các con vợ ", "telegram regression", sink.buffer(),
              "chào các con vợ ");
        check(sink.suffixOk(), "deleted text matches buffer suffix", sink.suffixOk() ? "yes" : "no",
              "yes");
        std::string deleted;
        for (const auto& d : sink.deleted()) {
            deleted += "[" + d + "]";
        }
        // Note: the tone key in "chaof" deletes "ao" (backs=2), not just "a":
        // the engine rewrites the whole diphthong tail. The machine must
        // delete exactly what the engine says.
        check(deleted == "[ao][a][o][ơ]", "deleted texts", deleted, "[ao][a][o][ơ]");
    }
    // App quirks (ported from Ngó Sen): behaviour first, names only where
    // nothing reported tells the app apart.
    {
        using tuigo::Field;
        auto field = [](const char* frontend, const char* program, bool surr,
                        bool orderFix = false, bool fmtPreedit = false) {
            Field f;
            f.frontend = frontend;
            f.program = program;
            f.surroundingText = surr;
            f.keyEventOrderFix = orderFix;
            f.formattedPreedit = fmtPreedit;
            return f;
        };
        const char* yes = "yes";
        check(tuigo::forwardsBackspaces(field("wayland_v2", "x", false)), "wayland forwards",
              yes, yes);
        check(!tuigo::forwardsBackspaces(field("dbus", "SDL_game", false)),
              "SDL on dbus gets no forwarded backspace", yes, yes);
        check(tuigo::ignoresForwardedKeys(field("dbus", "x", true)),
              "gtk4 dbus without KeyEventOrderFix ignores forwarded keys", yes, yes);
        check(!tuigo::ignoresForwardedKeys(field("dbus", "x", true, true)),
              "dbus with KeyEventOrderFix takes forwarded keys", yes, yes);
        check(tuigo::mustUseSurrounding(field("wayland", "soffice", true)),
              "LibreOffice must use surrounding text", yes, yes);
        check(tuigo::mustUseSurrounding(field("wayland", "libreoffice-calc", true)),
              "Calc on Wayland must use surrounding text", yes, yes);
        check(!tuigo::mustUseSurrounding(field("wayland", "soffice", false)),
              "LibreOffice without surrounding text cannot use it", yes, yes);
        check(tuigo::ignoresDeleteSurrounding(field("wayland_v2", "microsoft-edge", true)),
              "Edge ignores deleteSurroundingText", yes, yes);
        check(tuigo::ignoresDeleteSurrounding(field("wayland_v2", "org.telegram.desktop", true)),
              "Telegram ignores deleteSurroundingText", yes, yes);
        check(!tuigo::ignoresDeleteSurrounding(field("wayland_v2", "google-chrome", true)),
              "Chrome honours deleteSurroundingText", yes, yes);
        check(tuigo::isBrowser(field("wayland_v2", "google-chrome", true)),
              "Chrome is a browser", yes, yes);
        check(tuigo::isBrowser(field("dbus", "firefox", true)), "Firefox is a browser", yes, yes);
        check(!tuigo::isBrowser(field("wayland", "soffice", true)),
              "LibreOffice is not a browser", yes, yes);
        check(!tuigo::isBrowser(field("wayland", "microsoft-edge", true)),
              "Edge is not on the browser fast path", yes, yes);
    }
    if (failures != 0) {
        std::fprintf(stderr, "sen_test: %d FAILURES\n", failures);
        return 1;
    }
    std::printf("sen_test: PASS\n");
    return 0;
}
