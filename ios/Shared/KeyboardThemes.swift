import UIKit

/// Keyboard skins modelled on Laban Key's bundled themes; mirror of
/// android/KeyboardThemes.kt. All colors sampled from its theme snapshots.
/// border alpha 0 = borderless keys. blueGlyphs draws shift/enter glyphs in
/// accent instead of filling the key (Laban default look). enterAccentFill
/// paints Enter with accent instead.
struct KeyTheme {
    let id: String
    let label: String
    let bg: UIColor
    let bgEnd: UIColor
    let key: UIColor
    let keyPressed: UIColor
    let funcKey: UIColor
    let funcPressed: UIColor
    let border: UIColor
    let corner: CGFloat
    let text: UIColor
    let funcText: UIColor
    let accent: UIColor
    let onAccent: UIColor
    let space: UIColor
    let spaceBorder: UIColor
    let spaceText: UIColor
    let blueGlyphs: Bool
    let enterAccentFill: Bool
    let pressAccent: Bool
    let popup: UIColor
    let popupText: UIColor
    let popupBorder: UIColor
}

private func rgb(_ v: UInt32, alpha: CGFloat = 1) -> UIColor {
    UIColor(red: CGFloat((v >> 16) & 255) / 255,
            green: CGFloat((v >> 8) & 255) / 255,
            blue: CGFloat(v & 255) / 255,
            alpha: alpha)
}

enum KeyboardThemes {
    static let defaultId = "tuigo"

    private static let clear = UIColor.clear

    private static let tuigoLight = KeyTheme(
        id: "tuigo", label: "Tui Gõ",
        bg: rgb(0xF8FAFC), bgEnd: rgb(0xF1F5F9),
        key: rgb(0xFFFFFF), keyPressed: rgb(0xE2E8F0),
        funcKey: rgb(0xEFF3F6), funcPressed: rgb(0xCBD5E1),
        border: rgb(0xE2E8F0), corner: 10,
        text: rgb(0x0F172A), funcText: rgb(0x0F172A),
        accent: rgb(0x197F64), onAccent: rgb(0xFFFFFF),
        space: rgb(0xFFFFFF), spaceBorder: rgb(0xE2E8F0), spaceText: rgb(0x64748B),
        blueGlyphs: false, enterAccentFill: true, pressAccent: false,
        popup: rgb(0xFFFFFF), popupText: rgb(0x0F172A), popupBorder: rgb(0xE2E8F0))
    private static let tuigoDark = KeyTheme(
        id: "tuigo", label: "Tui Gõ",
        bg: rgb(0x0F1115), bgEnd: rgb(0x161922),
        key: rgb(0x232834), keyPressed: rgb(0x353C4D),
        funcKey: rgb(0x1A1E27), funcPressed: rgb(0x2B3241),
        border: rgb(0x2E3545), corner: 10,
        text: rgb(0xF8FAFC), funcText: rgb(0xF8FAFC),
        accent: rgb(0x3ED3B8), onAccent: rgb(0x0F1115),
        space: rgb(0x232834), spaceBorder: rgb(0x2E3545), spaceText: rgb(0x94A3B8),
        blueGlyphs: false, enterAccentFill: true, pressAccent: false,
        popup: rgb(0x232834), popupText: rgb(0xF8FAFC), popupBorder: rgb(0x2E3545))

    private static let laban = KeyTheme(
        id: "laban", label: "Laban",
        bg: rgb(0xE1E2E5), bgEnd: rgb(0xE1E2E5),
        key: rgb(0xFCFCFC), keyPressed: rgb(0xB8B9BE),
        funcKey: rgb(0xE1E2E5), funcPressed: rgb(0xB8B9BE),
        border: rgb(0xB6BBC4), corner: 6,
        text: rgb(0x1B1E24), funcText: rgb(0x1B1E24),
        accent: rgb(0x3F8EEB), onAccent: rgb(0xFFFFFF),
        space: rgb(0xC7D4E6), spaceBorder: rgb(0x3F8EEB), spaceText: rgb(0x1B1E24),
        blueGlyphs: true, enterAccentFill: false, pressAccent: false,
        popup: rgb(0x758899), popupText: rgb(0xFFFFFF), popupBorder: rgb(0x5C6E7C))
    private static let white = KeyTheme(
        id: "white", label: "Trắng",
        bg: rgb(0xECEEF1), bgEnd: rgb(0xECEEF1),
        key: rgb(0xFFFFFF), keyPressed: rgb(0xC9CDD4),
        funcKey: rgb(0xECEEF1), funcPressed: rgb(0xC9CDD4),
        border: rgb(0xD5D9E0), corner: 6,
        text: rgb(0x26292E), funcText: rgb(0x26292E),
        accent: rgb(0x3F8EEB), onAccent: rgb(0xFFFFFF),
        space: rgb(0xFFFFFF), spaceBorder: rgb(0xD5D9E0), spaceText: rgb(0x26292E),
        blueGlyphs: false, enterAccentFill: false, pressAccent: false,
        popup: rgb(0xFFFFFF), popupText: rgb(0x26292E), popupBorder: rgb(0xD5D9E0))
    private static let black = KeyTheme(
        id: "black", label: "Đen",
        bg: rgb(0x000000), bgEnd: rgb(0x000000),
        key: rgb(0x444444), keyPressed: rgb(0x5E5E5E),
        funcKey: rgb(0x222222), funcPressed: rgb(0x3A3A3A),
        border: rgb(0x1A1A1A), corner: 6,
        text: rgb(0xFFFFFF), funcText: rgb(0xFFFFFF),
        accent: rgb(0x3F8EEB), onAccent: rgb(0xFFFFFF),
        space: rgb(0x444444), spaceBorder: rgb(0x1A1A1A), spaceText: rgb(0xFFFFFF),
        blueGlyphs: false, enterAccentFill: false, pressAccent: false,
        popup: rgb(0x333333), popupText: rgb(0xFFFFFF), popupBorder: rgb(0x4A4A4A))
    private static let gray = KeyTheme(
        id: "gray", label: "Xám",
        bg: rgb(0x181F2D), bgEnd: rgb(0x181F2D),
        key: rgb(0x53616C), keyPressed: rgb(0x6B7B88),
        funcKey: rgb(0x374146), funcPressed: rgb(0x4A555E),
        border: clear, corner: 6,
        text: rgb(0xFFFFFF), funcText: rgb(0xFFFFFF),
        accent: rgb(0x7FB3E8), onAccent: rgb(0xFFFFFF),
        space: rgb(0x53616C), spaceBorder: clear, spaceText: rgb(0xFFFFFF),
        blueGlyphs: false, enterAccentFill: false, pressAccent: false,
        popup: rgb(0x53616C), popupText: rgb(0xFFFFFF), popupBorder: rgb(0x374146))
    private static let blue = KeyTheme(
        id: "blue", label: "Xanh đậm",
        bg: rgb(0x1C2331), bgEnd: rgb(0x1C2331),
        key: rgb(0x26324B), keyPressed: rgb(0x3A4A6B),
        funcKey: rgb(0x30436A), funcPressed: rgb(0x425381),
        border: clear, corner: 6,
        text: rgb(0xFFFFFF), funcText: rgb(0xD7E0F4),
        accent: rgb(0x8FA9DE), onAccent: rgb(0x1C2331),
        space: rgb(0x26324B), spaceBorder: clear, spaceText: rgb(0xD7E0F4),
        blueGlyphs: true, enterAccentFill: false, pressAccent: false,
        popup: rgb(0x26324B), popupText: rgb(0xFFFFFF), popupBorder: rgb(0x30436A))
    private static let ios = KeyTheme(
        id: "ios", label: "iOS",
        bg: rgb(0xD1D5DB), bgEnd: rgb(0xD1D5DB),
        key: rgb(0xFFFFFF), keyPressed: rgb(0xC9CDD3),
        funcKey: rgb(0xA4ACB5), funcPressed: rgb(0x8A939E),
        border: clear, corner: 6,
        text: rgb(0x000000), funcText: rgb(0x26292D),
        accent: rgb(0x8A939E), onAccent: rgb(0x26292D),
        space: rgb(0xFFFFFF), spaceBorder: clear, spaceText: rgb(0x26292D),
        blueGlyphs: false, enterAccentFill: false, pressAccent: false,
        popup: rgb(0xFFFFFF), popupText: rgb(0x000000), popupBorder: rgb(0xC9CDD3))
    private static let iosOld = KeyTheme(
        id: "ios_old", label: "iOS cổ điển",
        bg: rgb(0x828A97), bgEnd: rgb(0x464F5E),
        key: rgb(0xECECEF), keyPressed: rgb(0xFFFFFF),
        funcKey: rgb(0x636B78), funcPressed: rgb(0x7A828E),
        border: rgb(0x4E5560), corner: 8,
        text: rgb(0x2B3038), funcText: rgb(0xF4F5F6),
        accent: rgb(0x2B4E8F), onAccent: rgb(0xFFFFFF),
        space: rgb(0xE7E7E9), spaceBorder: rgb(0x4E5560), spaceText: rgb(0x2B3038),
        blueGlyphs: false, enterAccentFill: false, pressAccent: false,
        popup: rgb(0xE6E8E9), popupText: rgb(0x2B3038), popupBorder: rgb(0x8A9099))
    private static let ics = KeyTheme(
        id: "ics", label: "ICS",
        bg: rgb(0x111111), bgEnd: rgb(0x000000),
        key: rgb(0x555555), keyPressed: rgb(0x777777),
        funcKey: rgb(0x222222), funcPressed: rgb(0x444444),
        border: clear, corner: 2,
        text: rgb(0xFFFFFF), funcText: rgb(0xFFFFFF),
        accent: rgb(0x33B5E6), onAccent: rgb(0x000000),
        space: rgb(0x494949), spaceBorder: clear, spaceText: rgb(0xFFFFFF),
        blueGlyphs: false, enterAccentFill: false, pressAccent: false,
        popup: rgb(0x33B5E6), popupText: rgb(0x000000), popupBorder: rgb(0x33B5E6))
    private static let kitkat = KeyTheme(
        id: "kitkat", label: "KitKat",
        bg: rgb(0x111111), bgEnd: rgb(0x000000),
        key: rgb(0x555555), keyPressed: rgb(0x777777),
        funcKey: rgb(0x222222), funcPressed: rgb(0x444444),
        border: clear, corner: 2,
        text: rgb(0xFFFFFF), funcText: rgb(0xFFFFFF),
        accent: rgb(0x33B5E6), onAccent: rgb(0x000000),
        space: rgb(0x494949), spaceBorder: clear, spaceText: rgb(0xFFFFFF),
        blueGlyphs: false, enterAccentFill: false, pressAccent: false,
        popup: rgb(0x6B6B6B), popupText: rgb(0xFFFFFF), popupBorder: rgb(0x6B6B6B))
    private static let silver = KeyTheme(
        id: "silver", label: "Bạc",
        bg: rgb(0xC8C8C8), bgEnd: rgb(0xC8C8C8),
        key: rgb(0xE7E6E4), keyPressed: rgb(0xF5F4F2),
        funcKey: rgb(0x515151), funcPressed: rgb(0x636161),
        border: rgb(0x9E9B98), corner: 6,
        text: rgb(0x3A3735), funcText: rgb(0xFFFFFF),
        accent: rgb(0x515151), onAccent: rgb(0xFFFFFF),
        space: rgb(0xE7E6E4), spaceBorder: rgb(0x9E9B98), spaceText: rgb(0x3A3735),
        blueGlyphs: false, enterAccentFill: false, pressAccent: false,
        popup: rgb(0xE7E6E4), popupText: rgb(0x3A3735), popupBorder: rgb(0x9E9B98))
    private static let wood = KeyTheme(
        id: "wood", label: "Gỗ",
        bg: rgb(0xD7B381), bgEnd: rgb(0xC09063),
        key: rgb(0xE1CDB5), keyPressed: rgb(0xEFE2D2),
        funcKey: rgb(0xA08A73), funcPressed: rgb(0x8A7358),
        border: rgb(0x96785B), corner: 5,
        text: rgb(0x4A3B2C), funcText: rgb(0xFFFFFF),
        accent: rgb(0x8A7358), onAccent: rgb(0xFFFFFF),
        space: rgb(0xDFC9B1), spaceBorder: rgb(0x96785B), spaceText: rgb(0x4A3B2C),
        blueGlyphs: false, enterAccentFill: false, pressAccent: false,
        popup: rgb(0x1A1512), popupText: rgb(0xEAD9C0), popupBorder: rgb(0x96785B))
    private static let dark = KeyTheme(
        id: "dark", label: "Tối",
        bg: rgb(0x303033), bgEnd: rgb(0x0B0B0D),
        key: rgb(0x2F2F31), keyPressed: rgb(0x4A4A4E),
        funcKey: rgb(0x181818), funcPressed: rgb(0x2A2A2C),
        border: clear, corner: 4,
        text: rgb(0xFCFCFC), funcText: rgb(0xFCFCFC),
        accent: rgb(0x5B5B5B), onAccent: rgb(0xFFFFFF),
        space: rgb(0x2D2D2F), spaceBorder: clear, spaceText: rgb(0xFCFCFC),
        blueGlyphs: false, enterAccentFill: false, pressAccent: false,
        popup: rgb(0x222222), popupText: rgb(0xFFFFFF), popupBorder: rgb(0x444444))
    private static let nLight = KeyTheme(
        id: "n_light", label: "Nougat sáng",
        bg: rgb(0xF7F6F9), bgEnd: rgb(0xF7F6F9),
        key: rgb(0xF7F6F9), keyPressed: rgb(0xE4E3E8),
        funcKey: rgb(0xF7F6F9), funcPressed: rgb(0xDDDCE1),
        border: clear, corner: 10,
        text: rgb(0x263238), funcText: rgb(0x5F6B72),
        accent: rgb(0x4DB6AC), onAccent: rgb(0xFFFFFF),
        space: rgb(0xE1E0E3), spaceBorder: clear, spaceText: rgb(0x263238),
        blueGlyphs: false, enterAccentFill: true, pressAccent: false,
        popup: rgb(0xD1D6D9), popupText: rgb(0x263238), popupBorder: rgb(0xD1D6D9))
    private static let nDark = KeyTheme(
        id: "n_dark", label: "Nougat tối",
        bg: rgb(0x263238), bgEnd: rgb(0x263238),
        key: rgb(0x263238), keyPressed: rgb(0x37474F),
        funcKey: rgb(0x263238), funcPressed: rgb(0x37474F),
        border: clear, corner: 10,
        text: rgb(0xECEFF1), funcText: rgb(0x9FB0B8),
        accent: rgb(0x4DB6AC), onAccent: rgb(0xFFFFFF),
        space: rgb(0x3C474C), spaceBorder: clear, spaceText: rgb(0xECEFF1),
        blueGlyphs: false, enterAccentFill: true, pressAccent: false,
        popup: rgb(0x3D484D), popupText: rgb(0xECEFF1), popupBorder: rgb(0x3D484D))

    private static let iosPureLight = KeyTheme(
        id: "ios_pure", label: "iOS Hiện đại",
        bg: rgb(0xD1D5DB), bgEnd: rgb(0xCBD0D8),
        key: rgb(0xFFFFFF), keyPressed: rgb(0xE2E6EB),
        funcKey: rgb(0xAEB5BF), funcPressed: rgb(0x98A0AC),
        border: clear, corner: 9,
        text: rgb(0x000000), funcText: rgb(0x1F2937),
        accent: rgb(0x007AFF), onAccent: rgb(0xFFFFFF),
        space: rgb(0xFFFFFF), spaceBorder: clear, spaceText: rgb(0x4B5563),
        blueGlyphs: false, enterAccentFill: true, pressAccent: false,
        popup: rgb(0xFFFFFF), popupText: rgb(0x000000), popupBorder: rgb(0xC5C9D0))

    private static let iosPureDark = KeyTheme(
        id: "ios_pure", label: "iOS Hiện đại",
        bg: rgb(0x1C1C1E), bgEnd: rgb(0x121214),
        key: rgb(0x3A3A3C), keyPressed: rgb(0x545458),
        funcKey: rgb(0x2C2C2E), funcPressed: rgb(0x48484A),
        border: clear, corner: 9,
        text: rgb(0xFFFFFF), funcText: rgb(0xEBEBF5),
        accent: rgb(0x0A84FF), onAccent: rgb(0xFFFFFF),
        space: rgb(0x3A3A3C), spaceBorder: clear, spaceText: rgb(0xAEAEB2),
        blueGlyphs: false, enterAccentFill: true, pressAccent: false,
        popup: rgb(0x3A3A3C), popupText: rgb(0xFFFFFF), popupBorder: rgb(0x545458))

    private static let cyberNeonDark = KeyTheme(
        id: "cyber_neon", label: "Cyber Neon",
        bg: rgb(0x04060A), bgEnd: rgb(0x0B101E),
        key: rgb(0x0B111E), keyPressed: rgb(0x16223B),
        funcKey: rgb(0x080D18), funcPressed: rgb(0x142038),
        border: rgb(0x0EA5E9), corner: 8,
        text: rgb(0x38BDF8), funcText: rgb(0x7DD3FC),
        accent: rgb(0x10B981), onAccent: rgb(0x04060A),
        space: rgb(0x0B111E), spaceBorder: rgb(0x0EA5E9), spaceText: rgb(0x38BDF8),
        blueGlyphs: false, enterAccentFill: true, pressAccent: false,
        popup: rgb(0x0B111E), popupText: rgb(0x38BDF8), popupBorder: rgb(0x0EA5E9))

    private static let cyberNeonLight = KeyTheme(
        id: "cyber_neon", label: "Cyber Neon",
        bg: rgb(0xE0F2FE), bgEnd: rgb(0xBAE6FD),
        key: rgb(0xFFFFFF), keyPressed: rgb(0xE0F2FE),
        funcKey: rgb(0xBAE6FD), funcPressed: rgb(0x7DD3FC),
        border: rgb(0x0284C7), corner: 8,
        text: rgb(0x082F49), funcText: rgb(0x0369A1),
        accent: rgb(0x10B981), onAccent: rgb(0xFFFFFF),
        space: rgb(0xFFFFFF), spaceBorder: rgb(0x0284C7), spaceText: rgb(0x0369A1),
        blueGlyphs: false, enterAccentFill: true, pressAccent: false,
        popup: rgb(0xFFFFFF), popupText: rgb(0x082F49), popupBorder: rgb(0x0284C7))

    private static let matchaLatteLight = KeyTheme(
        id: "matcha_latte", label: "Matcha Latte",
        bg: rgb(0xEDE4D8), bgEnd: rgb(0xF5ECE0),
        key: rgb(0xFFFDF9), keyPressed: rgb(0xEFE8DD),
        funcKey: rgb(0xA8D5BA), funcPressed: rgb(0x8FC4A3),
        border: rgb(0xD8C7B5), corner: 12,
        text: rgb(0x4A3B32), funcText: rgb(0x1B432C),
        accent: rgb(0xF7C59F), onAccent: rgb(0x542E13),
        space: rgb(0xFFFDF9), spaceBorder: rgb(0xD8C7B5), spaceText: rgb(0x4A3B32),
        blueGlyphs: false, enterAccentFill: true, pressAccent: false,
        popup: rgb(0xFFFDF9), popupText: rgb(0x4A3B32), popupBorder: rgb(0xD8C7B5))

    private static let matchaLatteDark = KeyTheme(
        id: "matcha_latte", label: "Matcha Latte",
        bg: rgb(0x121A16), bgEnd: rgb(0x1B2822),
        key: rgb(0x24352C), keyPressed: rgb(0x334C3F),
        funcKey: rgb(0x1B2822), funcPressed: rgb(0x2D4439),
        border: rgb(0x385446), corner: 12,
        text: rgb(0xF4ECE1), funcText: rgb(0x95B8A6),
        accent: rgb(0x74C69D), onAccent: rgb(0x121A16),
        space: rgb(0x24352C), spaceBorder: rgb(0x385446), spaceText: rgb(0xF4ECE1),
        blueGlyphs: false, enterAccentFill: true, pressAccent: false,
        popup: rgb(0x24352C), popupText: rgb(0xF4ECE1), popupBorder: rgb(0x52B788))

    private static let modernJadeLight = KeyTheme(
        id: "modern_jade", label: "Modern Jade",
        bg: rgb(0xE6F7F5), bgEnd: rgb(0xCCFBF1),
        key: rgb(0xFFFFFF), keyPressed: rgb(0xCCFBF1),
        funcKey: rgb(0xCCFBF1), funcPressed: rgb(0x99F6E4),
        border: rgb(0x99E6D8), corner: 12,
        text: rgb(0x0D2B28), funcText: rgb(0x0F766E),
        accent: rgb(0x0D9488), onAccent: rgb(0xFFFFFF),
        space: rgb(0xFFFFFF), spaceBorder: rgb(0x99E6D8), spaceText: rgb(0x0F766E),
        blueGlyphs: false, enterAccentFill: true, pressAccent: false,
        popup: rgb(0xFFFFFF), popupText: rgb(0x0D2B28), popupBorder: rgb(0x99E6D8))

    private static let modernJadeDark = KeyTheme(
        id: "modern_jade", label: "Modern Jade",
        bg: rgb(0x0D1614), bgEnd: rgb(0x142420),
        key: rgb(0x1B2E29), keyPressed: rgb(0x26403A),
        funcKey: rgb(0x14221E), funcPressed: rgb(0x1D332D),
        border: rgb(0x254039), corner: 12,
        text: rgb(0xF0FDF4), funcText: rgb(0xD1FAE5),
        accent: rgb(0x10B981), onAccent: rgb(0x064E3B),
        space: rgb(0x1B2E29), spaceBorder: rgb(0x254039), spaceText: rgb(0xD1FAE5),
        blueGlyphs: false, enterAccentFill: true, pressAccent: false,
        popup: rgb(0x1B2E29), popupText: rgb(0xF0FDF4), popupBorder: rgb(0x254039))

    private static let obsidian = KeyTheme(
        id: "obsidian", label: "Obsidian",
        bg: rgb(0x000000), bgEnd: rgb(0x090A0D),
        key: rgb(0x16181F), keyPressed: rgb(0x262A36),
        funcKey: rgb(0x0E0F14), funcPressed: rgb(0x1E212B),
        border: rgb(0x222633), corner: 10,
        text: rgb(0xF8FAFC), funcText: rgb(0x94A3B8),
        accent: rgb(0x3ED3B8), onAccent: rgb(0x000000),
        space: rgb(0x16181F), spaceBorder: rgb(0x222633), spaceText: rgb(0x94A3B8),
        blueGlyphs: false, enterAccentFill: true, pressAccent: false,
        popup: rgb(0x16181F), popupText: rgb(0xF8FAFC), popupBorder: rgb(0x222633))

    private static let pastelMint = KeyTheme(
        id: "pastel_mint", label: "Pastel Mint",
        bg: rgb(0xF0FDF4), bgEnd: rgb(0xDCFCE7),
        key: rgb(0xFFFFFF), keyPressed: rgb(0xD1FAE5),
        funcKey: rgb(0xE1F8EC), funcPressed: rgb(0xC6F2DD),
        border: rgb(0xC1EBD6), corner: 10,
        text: rgb(0x064E3B), funcText: rgb(0x065F46),
        accent: rgb(0x059669), onAccent: rgb(0xFFFFFF),
        space: rgb(0xFFFFFF), spaceBorder: rgb(0xC1EBD6), spaceText: rgb(0x065F46),
        blueGlyphs: false, enterAccentFill: true, pressAccent: false,
        popup: rgb(0xFFFFFF), popupText: rgb(0x064E3B), popupBorder: rgb(0xC1EBD6))

    static let ids = ["tuigo", "adaptive", "ios_pure", "cyber_neon", "matcha_latte", "modern_jade", "obsidian", "pastel_mint",
                      "laban", "white", "black", "gray", "blue", "ios", "ios_old", "ics", "kitkat", "silver", "wood", "dark", "n_light", "n_dark"]

    static func label(_ id: String) -> String {
        id == "adaptive" ? "Theo hệ thống" : (all(dark: false).first { $0.id == id }?.label ?? id)
    }

    static func all(dark: Bool) -> [KeyTheme] {
        [(dark ? tuigoDark : tuigoLight),
         (dark ? iosPureDark : iosPureLight),
         (dark ? cyberNeonDark : cyberNeonLight),
         (dark ? matchaLatteDark : matchaLatteLight),
         (dark ? modernJadeDark : modernJadeLight),
         obsidian, pastelMint,
         laban, white, ios, iosOld, silver, wood, nLight, ics, kitkat, KeyboardThemes.dark, black, gray, blue, nDark]
    }

    static func get(_ id: String, dark: Bool) -> KeyTheme {
        if id == "adaptive" { return dark ? gray : laban }
        return all(dark: dark).first { $0.id == id } ?? (dark ? tuigoDark : tuigoLight)
    }

    static func current(dark: Bool) -> KeyTheme {
        get(Prefs.defaults.string(forKey: Prefs.keyTheme) ?? defaultId, dark: dark)
    }
}
