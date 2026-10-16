// tuigo::Engine over the Go shim (../go/shim.go, built as a C archive).
// One instance per fcitx5 input context.
#pragma once

#include "sen.h"

#include <cstdint>
#include <string>

namespace tuigo {

class GoEngine final : public Engine {
  public:
    GoEngine();
    ~GoEngine() override;

    GoEngine(const GoEngine&) = delete;
    GoEngine& operator=(const GoEngine&) = delete;

    Edit processKey(std::uint32_t codePoint) override;
    Edit backspace() override;
    bool atWordBeginning() const override;
    void reset() override;
    void setViet(bool viet) override;

    void setInputMethod(int im);
    void setOptions(bool spell, bool modern, bool free, bool macro, bool restore);
    void loadMacroFile(const std::string& path);
    void loadKeymapFile(const std::string& path);

  private:
    Edit readResult() const;

    std::uintptr_t handle_;
};

} // namespace tuigo
