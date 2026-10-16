// Small UTF-8 helpers for the Sen machine. No dependency on fcitx5 or ICU:
// the machine only needs code-point / UTF-16-unit counts to translate the
// engine's Backs (UTF-16 units) into the exact text to delete.
#pragma once

#include <cstddef>
#include <cstdint>
#include <string>

namespace tuigo {

// UTF-16 units needed to encode s (astral code points count as 2).
std::size_t utf16Units(const std::string& s);

// Number of Unicode code points in s.
std::size_t utf8Length(const std::string& s);

// Erase the last `units` UTF-16 units from s. Returns the shortened string
// and, in runesErased, how many code points were removed. units larger
// than the string's unit count erases everything.
std::string eraseLastUnits(const std::string& s, std::size_t units, std::size_t* runesErased);

// The last `units` UTF-16 units of s, whole code points only. units larger
// than the string's unit count returns the whole string.
std::string lastUnits(const std::string& s, std::size_t units);

// Byte offset of the `chars`-th code point in s (chars == 0 -> 0).
// Clamped to s.size() when s has fewer code points.
std::size_t byteOffset(const std::string& s, std::size_t chars);

// Encode one code point as UTF-8. Invalid values yield "?".
std::string utf8FromCodePoint(std::uint32_t cp);

} // namespace tuigo
