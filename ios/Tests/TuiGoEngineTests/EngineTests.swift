import XCTest
@testable import TuiGoEngine

// Simulates a text document: engine output is applied as
// "delete <backs> chars, insert <out>". Text kept as UTF-16 units —
// exactly what the engine produces.
private final class DocSim {
    private(set) var text = [UInt16]()

    func feed(_ e: Engine, _ key: Character) {
        let res = e.filter(key.ord)
        if res.consumed {
            var backs = res.backs
            if backs > text.count { backs = text.count }
            text.removeLast(backs)
            text.append(contentsOf: res.out)
        } else {
            text.append(UInt16(key.ord))
        }
    }

    func backspace(_ e: Engine) {
        let res = e.backspace()
        if res.consumed {
            var backs = res.backs
            if backs > text.count { backs = text.count }
            text.removeLast(backs)
            text.append(contentsOf: res.out)
        } else if !text.isEmpty {
            text.removeLast()
        }
    }

    func typeKeys(_ e: Engine, _ keys: String) {
        for k in keys { feed(e, k) }
    }

    var string: String { String(decoding: text, as: UTF16.self) }
}

private func newEngine(_ opts: Options? = nil) -> Engine {
    let e = Engine()
    if let opts { e.setOptions(opts) }
    return e
}

private func defaultOpts() -> Options {
    let o = Options()
    o.freeMarking = 1
    o.modernStyle = 1
    o.spellCheckEnabled = 1
    return o
}

final class EngineTests: XCTestCase {

    func testTelex() {
        let cases: [(String, String)] = [
            ("tieengs", "tiếng"),
            ("aa", "â"),
            ("aw", "ă"),
            ("ow", "ơ"),
            ("uw", "ư"),
            ("oo", "ô"),
            ("ee", "ê"),
            ("dd", "đ"),
            ("vietj nam", "viẹt nam"),
            ("vieetj", "việt"),
            ("dungx", "dũng"),
            ("xin chaof", "xin chào"),
            ("nguwowif", "người"),
            ("thuow", "thuơ"),
            ("duongf", "duòng"),
            ("duowngf", "dường"),
            ("muwowng", "mương"),
            ("muwowngf", "mường"),
            ("nghiengx", "nghiẽng"),
            ("nghieengx", "nghiễng"),
            ("khoiwr", "khởi"),
            ("voiws", "với"),
            ("quanr", "quản"),
            ("trij", "trị"),
            ("hots", "hót"),
            ("langx", "lãng"),
            ("quanrz", "quan"),
            ("dduwowngf", "đường"),
            ("toanfs", "toán"), // later tone key overrides
            ("toansz", "toan"), // z strips the tone
            ("uow", "ươ"),
            ("nguoiwf", "người"),
            ("quans", "quán"),
        ]
        for (keys, want) in cases {
            let e = newEngine(defaultOpts())
            let d = DocSim()
            d.typeKeys(e, keys)
            XCTAssertEqual(d.string, want, "telex \"\(keys)\"")
        }
    }

    func testTelexOldStyle() {
        // modernStyle off -> tone on first vowel of oa/oe/uy
        let opts = Options()
        opts.freeMarking = 1
        let e = newEngine(opts)
        let d = DocSim()
        d.typeKeys(e, "hoaf")
        XCTAssertEqual(d.string, "hòa")
    }

    func testTelexModernStyle() {
        let e = newEngine(defaultOpts())
        let d = DocSim()
        d.typeKeys(e, "hoaf")
        XCTAssertEqual(d.string, "hoà")
    }

    func testToneRemoval() {
        // pressing the same tone key twice removes the tone (UniKey semantics:
        // the reverted key is consumed into the output, so "hoss" -> "hos")
        let e = newEngine(defaultOpts())
        let d = DocSim()
        d.typeKeys(e, "hoss")
        XCTAssertEqual(d.string, "hos")
        d.typeKeys(e, "s") // third s passes through as a literal letter
        XCTAssertEqual(d.string, "hoss")
    }

    func testZRemovesTone() {
        let e = newEngine(defaultOpts())
        let d = DocSim()
        d.typeKeys(e, "hasz")
        XCTAssertEqual(d.string, "ha")
    }

    func testEscapeW() {
        // standalone w maps to ư via mapChar fallback
        let e = newEngine(defaultOpts())
        let d = DocSim()
        d.typeKeys(e, "w")
        XCTAssertEqual(d.string, "ư")
    }

    func testVni() {
        let e = newEngine(defaultOpts())
        e.setInputMethod(ImVni)
        let d = DocSim()
        d.typeKeys(e, "vie6t5 nam")
        XCTAssertEqual(d.string, "việt nam")
    }

    func testBackspaceToneMove() {
        // "toanf" -> "toàn"; backspace removes 'n' -> tone moves to 'o'
        let e = newEngine(defaultOpts())
        let d = DocSim()
        d.typeKeys(e, "toanf")
        XCTAssertEqual(d.string, "toàn")
        d.backspace(e)
        XCTAssertEqual(d.string, "toà")
    }

    func testNonVnWord() {
        let e = newEngine(defaultOpts())
        let d = DocSim()
        d.typeKeys(e, "hello ")
        XCTAssertEqual(d.string, "hello ")
    }

    func testVietKeyOff() {
        let e = newEngine(defaultOpts())
        e.setVietKey(false)
        let d = DocSim()
        d.typeKeys(e, "tieengs")
        XCTAssertEqual(d.string, "tieengs")
    }

    func testAutoNonVnRestore() {
        // a word with marks but invalid spelling restores original keystrokes
        let opts = Options()
        opts.freeMarking = 1
        opts.modernStyle = 1
        opts.spellCheckEnabled = 1
        opts.autoNonVnRestore = 1
        let e = newEngine(opts)
        let d = DocSim()
        d.typeKeys(e, "ddasd ") // đá + d is invalid -> restore to "ddasd "
        XCTAssertEqual(d.string, "ddasd ")
    }

    func testDdAbbreviations() {
        // gõ tắt chữ Đ in consonant-only abbreviations: kept as typed
        let opts = Options()
        opts.freeMarking = 1
        opts.modernStyle = 1
        opts.spellCheckEnabled = 1
        opts.autoNonVnRestore = 1
        let cases: [(String, String)] = [
            ("HDD ", "HĐ "),
            ("CDDT ", "CĐT "),
            ("DDH ", "ĐH "),
            ("DDCS ", "ĐCS "),
            ("Hdd, CddT; DdH. ", "Hđ, CđT; ĐH. "),
            ("hdd ", "hđ "),
            ("cddt ", "cđt "),
            ("ddh ", "đh "),
            ("DD ", "Đ "),
            ("ddasd ", "ddasd "), // marked invalid word with vowels still restores
            ("HCM ", "HCM "),     // consonant after bare 'h' must not panic
            ("hn ", "hn "),
        ]
        for (keys, want) in cases {
            let e = newEngine(opts)
            let d = DocSim()
            d.typeKeys(e, keys)
            XCTAssertEqual(d.string, want, "dd abbrev \"\(keys)\"")
        }
    }

    func testMacro() {
        let opts = Options()
        opts.macroEnabled = 1
        let e = newEngine(opts)
        e.macStore.addItem("vn:Viet Nam")
        let d = DocSim()
        d.typeKeys(e, "vn ")
        XCTAssertEqual(d.string, "Viet Nam ")
    }

    func testMacroReloadAndUnicode() {
        let opts = Options(); opts.macroEnabled = 1
        let e = newEngine(opts)
        e.loadMacroText("\u{FEFF}; danh sách\nko:không 😀\nmail:a:b@example.com")
        let first = DocSim(); first.typeKeys(e, "ko ")
        XCTAssertEqual(first.string, "không 😀 ")
        e.resetBuf(); e.loadMacroText("ko:khác")
        let edited = DocSim(); edited.typeKeys(e, "ko ")
        XCTAssertEqual(edited.string, "khác ")
        e.resetBuf(); e.loadMacroText("")
        let deleted = DocSim(); deleted.typeKeys(e, "ko ")
        XCTAssertEqual(deleted.string, "ko ")
        e.resetBuf(); e.loadMacroText("ko:không")
        let disabledOpts = Options(); disabledOpts.macroEnabled = 0; e.setOptions(disabledOpts)
        let disabled = DocSim(); disabled.typeKeys(e, "ko ")
        XCTAssertEqual(disabled.string, "ko ")
    }

    func testMacroValidation() {
        XCTAssertNil(macroValidationError("; comment\nko:không 😀\nmail:a:b@example.com"))
        XCTAssertNil(macroValidationError(""))
        for bad in ["ko=không", ":không", "ko:", "a b:c", "ko:a\nko:b", "abcdefghijklmnop:x"] {
            XCTAssertNotNil(macroValidationError(bad), "must reject \(bad)")
        }
        XCTAssertNil(macroValidationError("abcdefghijklmno:x"))
    }

    func testViqr() {
        let e = newEngine(defaultOpts())
        e.setInputMethod(ImViqr)
        let d = DocSim(); d.typeKeys(e, "o^n? ")
        XCTAssertEqual(d.string, "ổn ")
    }

    func testTCVNConvert() {
        var out: [UInt16] = [0x74, 0x69, 0x1ebf, 0x6e, 0x67] // "tiếng" UTF-16
        convertTCVN(&out)
        let want: [UInt16] = [0x74, 0x69, 0x00d5, 0x6e, 0x67] // TCVN3: ế -> 0xD5
        XCTAssertEqual(out, want)

        // ASCII + unmapped chars pass through untouched
        var mixed: [UInt16] = [0x41, 0x7A, 0x31, 0x4E2D]
        convertTCVN(&mixed)
        XCTAssertEqual(mixed, [0x41, 0x7A, 0x31, 0x4E2D])
    }

    func testTelexVniHybrid() {
        var e = newEngine(defaultOpts())
        e.setInputMethod(ImTelexVni)
        var d = DocSim()
        d.typeKeys(e, "on63") // VNI: ô + hỏi -> ổn
        XCTAssertEqual(d.string, "ổn")

        e = newEngine(defaultOpts())
        e.setInputMethod(ImTelexVni)
        d = DocSim()
        d.typeKeys(e, "oonr") // Telex: ô + hỏi -> ổn
        XCTAssertEqual(d.string, "ổn")
    }
}
