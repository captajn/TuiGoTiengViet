#include "tuigo-engine.h"

#include "tuigo-shim.h"

#include <vector>

namespace tuigo {

GoEngine::GoEngine() : handle_(static_cast<std::uintptr_t>(TuiGoNew())) {}

GoEngine::~GoEngine() {
    TuiGoDelete(static_cast<uintptr_t>(handle_));
}

Edit GoEngine::processKey(std::uint32_t codePoint) {
    const bool consumed = TuiGoProcessKey(static_cast<uintptr_t>(handle_), codePoint) != 0;
    Edit edit = readResult();
    edit.consumed = consumed;
    return edit;
}

Edit GoEngine::backspace() {
    const bool consumed = TuiGoBackspace(static_cast<uintptr_t>(handle_)) != 0;
    Edit edit = readResult();
    edit.consumed = consumed;
    return edit;
}

bool GoEngine::atWordBeginning() const {
    return TuiGoAtWordBeginning(static_cast<uintptr_t>(handle_)) != 0;
}

void GoEngine::reset() {
    TuiGoReset(static_cast<uintptr_t>(handle_));
}

void GoEngine::setViet(bool viet) {
    TuiGoSetViet(static_cast<uintptr_t>(handle_), viet ? 1 : 0);
}

void GoEngine::setInputMethod(int im) {
    TuiGoSetInputMethod(static_cast<uintptr_t>(handle_), im);
}

void GoEngine::setOptions(bool spell, bool modern, bool free, bool macro, bool restore) {
    TuiGoSetOptions(static_cast<uintptr_t>(handle_), spell ? 1 : 0, modern ? 1 : 0, free ? 1 : 0,
                    macro ? 1 : 0, restore ? 1 : 0);
}

void GoEngine::loadMacroFile(const std::string& path) {
    // The shim copies the path (C.GoString); the cast is const-only.
    TuiGoLoadMacroFile(static_cast<uintptr_t>(handle_), const_cast<char*>(path.c_str()));
}

void GoEngine::loadKeymapFile(const std::string& path) {
    TuiGoLoadKeymapFile(static_cast<uintptr_t>(handle_), const_cast<char*>(path.c_str()));
}

Edit GoEngine::readResult() const {
    Edit edit;
    edit.backs = static_cast<int>(TuiGoBacks(static_cast<uintptr_t>(handle_)));
    // PullOut re-reads the stored Out, so grow the buffer until it fits.
    std::vector<char> buf(256);
    for (;;) {
        const int n = static_cast<int>(
            TuiGoPullOut(static_cast<uintptr_t>(handle_), buf.data(), static_cast<int>(buf.size())));
        if (n < static_cast<int>(buf.size()) || buf.size() >= 65536) {
            edit.out.assign(buf.data(), static_cast<std::size_t>(n));
            return edit;
        }
        buf.resize(buf.size() * 2);
    }
}

} // namespace tuigo
