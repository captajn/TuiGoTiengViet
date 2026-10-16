#include "utf8.h"

namespace tuigo {
namespace {

// Decode one code point at s[i]; advances i past it. Invalid bytes are
// consumed one at a time and reported as U+FFFD.
std::uint32_t decodeOne(const std::string& s, std::size_t& i) {
    const auto c = static_cast<unsigned char>(s[i]);
    if (c < 0x80) {
        ++i;
        return c;
    }
    int extra = 0;
    std::uint32_t cp = 0;
    if ((c & 0xE0) == 0xC0) {
        extra = 1;
        cp = c & 0x1F;
    } else if ((c & 0xF0) == 0xE0) {
        extra = 2;
        cp = c & 0x0F;
    } else if ((c & 0xF8) == 0xF0) {
        extra = 3;
        cp = c & 0x07;
    } else {
        ++i;
        return 0xFFFD;
    }
    if (i + static_cast<std::size_t>(extra) >= s.size()) {
        i = s.size();
        return 0xFFFD;
    }
    for (int k = 1; k <= extra; ++k) {
        const auto cc = static_cast<unsigned char>(s[i + static_cast<std::size_t>(k)]);
        if ((cc & 0xC0) != 0x80) {
            i += static_cast<std::size_t>(k);
            return 0xFFFD;
        }
        cp = (cp << 6) | (cc & 0x3F);
    }
    i += static_cast<std::size_t>(extra) + 1;
    return cp;
}

} // namespace

std::size_t utf16Units(const std::string& s) {
    std::size_t units = 0;
    std::size_t i = 0;
    while (i < s.size()) {
        units += decodeOne(s, i) >= 0x10000 ? 2 : 1;
    }
    return units;
}

std::size_t utf8Length(const std::string& s) {
    std::size_t n = 0;
    std::size_t i = 0;
    while (i < s.size()) {
        (void)decodeOne(s, i);
        ++n;
    }
    return n;
}

std::string eraseLastUnits(const std::string& s, std::size_t units, std::size_t* runesErased) {
    if (runesErased) {
        *runesErased = 0;
    }
    // Walk forward: `cut` is the byte offset where the tail of `units`
    // UTF-16 units starts (s.size() when units == 0).
    std::size_t cut = s.size();
    std::size_t tailRunes = 0;
    const std::size_t total = utf16Units(s);
    if (units >= total) {
        if (runesErased) {
            *runesErased = utf8Length(s);
        }
        return {};
    }
    const std::size_t keepUnits = total - units;
    std::size_t seen = 0;
    std::size_t i = 0;
    while (i < s.size()) {
        const std::size_t start = i;
        const std::uint32_t cp = decodeOne(s, i);
        const std::size_t u = cp >= 0x10000 ? 2 : 1;
        if (seen + u > keepUnits) {
            cut = start;
            break;
        }
        seen += u;
    }
    // Count runes in the erased tail.
    std::size_t j = cut;
    while (j < s.size()) {
        (void)decodeOne(s, j);
        ++tailRunes;
    }
    if (runesErased) {
        *runesErased = tailRunes;
    }
    return s.substr(0, cut);
}

std::string lastUnits(const std::string& s, std::size_t units) {
    const std::size_t total = utf16Units(s);
    if (units >= total) {
        return s;
    }
    const std::size_t keepUnits = total - units;
    std::size_t seen = 0;
    std::size_t i = 0;
    while (i < s.size()) {
        const std::size_t start = i;
        const std::uint32_t cp = decodeOne(s, i);
        const std::size_t u = cp >= 0x10000 ? 2 : 1;
        if (seen + u > keepUnits) {
            return s.substr(start);
        }
        seen += u;
    }
    return {};
}

std::size_t byteOffset(const std::string& s, std::size_t chars) {
    std::size_t i = 0;
    std::size_t n = 0;
    while (i < s.size() && n < chars) {
        (void)decodeOne(s, i);
        ++n;
    }
    return i;
}

std::string utf8FromCodePoint(std::uint32_t cp) {
    if (cp < 0x80) {
        return std::string(1, static_cast<char>(cp));
    }
    if (cp < 0x800) {
        return {static_cast<char>(0xC0 | (cp >> 6)), static_cast<char>(0x80 | (cp & 0x3F))};
    }
    if (cp < 0x10000) {
        return {static_cast<char>(0xE0 | (cp >> 12)), static_cast<char>(0x80 | ((cp >> 6) & 0x3F)),
                static_cast<char>(0x80 | (cp & 0x3F))};
    }
    if (cp <= 0x10FFFF) {
        return {static_cast<char>(0xF0 | (cp >> 18)), static_cast<char>(0x80 | ((cp >> 12) & 0x3F)),
                static_cast<char>(0x80 | ((cp >> 6) & 0x3F)), static_cast<char>(0x80 | (cp & 0x3F))};
    }
    return "?";
}

} // namespace tuigo
