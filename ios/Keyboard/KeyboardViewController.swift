import UIKit
import TuiGoEngine

// UIInputViewController shell around the pure-Swift engine port.
// Same contract as the Android host (ime/TuiGoImeService.kt):
//   filter(code) -> Result(consumed, backs, out, keyOut)
//   consumed   : engine handled the key; apply backs+out
//   !consumed  : pass-through; commit the raw char ourselves
//   backs/out  : UTF-16 units -> deleteBackward()/insertText()
final class KeyboardViewController: UIInputViewController, KeyboardView.Listener {

    private let engine = Engine()
    private var keyboard: KeyboardView!
    private var heightConstraint: NSLayoutConstraint!

    private var vietOn = true
    private var doubleSpace = true

    /// Pending "double space -> ." state (700ms window, same as Android).
    private var spaceAt = Date.distantPast
    private var spacePending = false

    /// Tail of documentContextBeforeInput after our last edit. On the next
    /// keypress, if the context no longer ends with this, the user/app moved
    /// the caret or edited externally -> drop engine state. Checking lazily
    /// like this (instead of textDidChange) is safe because iOS can deliver
    /// textDidChange for our own commits too.
    private var expectedTail = ""

    override func viewDidLoad() {
        super.viewDidLoad()
        keyboard = KeyboardView()
        keyboard.listener = self
        keyboard.translatesAutoresizingMaskIntoConstraints = false
        view.addSubview(keyboard)
        NSLayoutConstraint.activate([
            keyboard.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            keyboard.trailingAnchor.constraint(equalTo: view.trailingAnchor),
            keyboard.topAnchor.constraint(equalTo: view.topAnchor),
            keyboard.bottomAnchor.constraint(equalTo: view.safeAreaLayoutGuide.bottomAnchor),
        ])
        heightConstraint = view.heightAnchor.constraint(equalToConstant: keyboard.intrinsicContentSize.height)
        heightConstraint.priority = .defaultHigh
        heightConstraint.isActive = true
        applyPrefs()
    }

    override func viewWillLayoutSubviews() {
        super.viewWillLayoutSubviews()
        let landscape = view.window?.windowScene?.interfaceOrientation.isLandscape == true
        keyboard.setMetrics(height: CGFloat(landscape ? min(40, Prefs.keyHeight()) : Prefs.keyHeight()), gap: CGFloat(Prefs.keyGap()))
        heightConstraint.constant = keyboard.intrinsicContentSize.height + view.safeAreaInsets.bottom
    }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        applyPrefs()
        keyboard.showsGlobe = needsInputModeSwitchKey
        keyboard.returnLabel = Self.returnLabel(for: textDocumentProxy.returnKeyType)
        engine.resetBuf()
        spacePending = false
        rememberTail()
        updateAutoShift()
    }

    override func textDidChange(_ textInput: UITextInput?) {
        super.textDidChange(textInput)
        updateAutoShift()
    }

    private func applyPrefs() {
        Prefs.apply(to: engine)
        vietOn = hasFullAccess ? Prefs.vietOn() : (UserDefaults.standard.object(forKey: Prefs.keyVietOn) as? Bool ?? Prefs.vietOn())
        doubleSpace = Prefs.doubleSpacePeriod()
        keyboard.vietnameseOn = vietOn
        engine.setVietKey(vietOn)
        keyboard.showNumberRow = Prefs.numberRow()
        // Read-only access to the containing app's shared container is sufficient.
        if let url = Prefs.macroURL {
            if !FileManager.default.fileExists(atPath: url.path) { engine.loadMacroText("") }
            else if let text = try? String(contentsOf: url, encoding: .utf8) { engine.loadMacroText(text) }
        }
        view.setNeedsLayout()
    }

    // ---- KeyboardView.Listener ------------------------------------------------

    func onChar(_ ch: Character) {
        let proxy = textDocumentProxy
        checkExternalChange()

        if isRestrictedField {
            proxy.insertText(String(ch))
            spacePending = false
            rememberTail()
            return
        }

        if ch == " ", doubleSpace, spacePending,
           Date().timeIntervalSince(spaceAt) < 0.7 {
            spacePending = false
            proxy.deleteBackward()
            proxy.insertText(". ")
            engine.resetBuf()
            rememberTail()
            updateAutoShift()
            return
        }

        guard vietOn else {
            proxy.insertText(String(ch))
            trackSpace(ch)
            rememberTail()
            updateAutoShift()
            return
        }

        engine.shiftPressed = keyboard.shifted
        engine.capsLockOn = keyboard.capsLock
        let res = engine.filter(Self.keyCode(ch))
        if res.consumed {
            apply(res)
        } else {
            proxy.insertText(String(ch))
        }
        trackSpace(ch)
        rememberTail()
        updateAutoShift()
    }

    func onDelete() {
        checkExternalChange()
        spacePending = false
        let res = engine.backspace()
        if res.consumed {
            apply(res)
        } else {
            textDocumentProxy.deleteBackward()
        }
        rememberTail()
        updateAutoShift()
    }

    func onEnter() {
        checkExternalChange()
        spacePending = false
        engine.resetBuf()
        textDocumentProxy.insertText("\n")
        rememberTail()
        updateAutoShift()
    }

    func onShiftChanged(_ shifted: Bool, _ capsLock: Bool) {
        engine.shiftPressed = shifted
        engine.capsLockOn = capsLock
    }

    func onVietToggle() {
        vietOn.toggle()
        // Without Full Access an extension cannot write to the shared container.
        // Always change the current session; persist only when writing is allowed.
        if hasFullAccess { Prefs.defaults.set(vietOn, forKey: Prefs.keyVietOn) }
        UserDefaults.standard.set(vietOn, forKey: Prefs.keyVietOn)
        keyboard.vietnameseOn = vietOn
        engine.setVietKey(vietOn)
        engine.resetBuf()
        spacePending = false
    }

    func onSwitchIme() {
        advanceToNextInputMode()
    }

    func onInputModeMenu(_ sender: UIButton, event: UIEvent) {
        handleInputModeList(from: sender, with: event)
    }

    // ---- output -------------------------------------------------------------------

    private func apply(_ res: TuiGoEngine.Result) {
        let proxy = textDocumentProxy
        for _ in 0..<res.backs { proxy.deleteBackward() }
        if !res.out.isEmpty {
            proxy.insertText(String(decoding: res.out, as: UTF16.self))
        }
    }

    // ---- helpers --------------------------------------------------------------------

    private func rememberTail() {
        expectedTail = String((textDocumentProxy.documentContextBeforeInput ?? "").suffix(64))
    }

    private func checkExternalChange() {
        let ctx = textDocumentProxy.documentContextBeforeInput ?? ""
        if !ctx.hasSuffix(expectedTail) {
            engine.resetBuf()
            spacePending = false
        }
    }

    private func trackSpace(_ ch: Character) {
        spacePending = ch == " "
        if spacePending { spaceAt = Date() }
    }

    private var isRestrictedField: Bool {
        switch textDocumentProxy.keyboardType {
        case .phonePad, .numberPad, .decimalPad:
            return true
        default:
            return false
        }
    }

    /// Sentence-start detection for one-shot shift (autocapitalization).
    private func updateAutoShift() {
        let proxy = textDocumentProxy
        guard proxy.autocapitalizationType == .sentences ||
              proxy.autocapitalizationType == .allCharacters ||
              proxy.autocapitalizationType == .words else {
            keyboard.setAutoShift(false)
            return
        }
        let ctx = proxy.documentContextBeforeInput ?? ""
        let on = ctx.isEmpty
            || ctx.hasSuffix("\n")
            || ctx.hasSuffix(". ") || ctx.hasSuffix("? ") || ctx.hasSuffix("! ")
            || ctx.hasSuffix("… ")
        keyboard.setAutoShift(on)
    }

    private static func keyCode(_ ch: Character) -> Int {
        Int(ch.unicodeScalars.first?.value ?? 0)
    }

    private static func returnLabel(for type: UIReturnKeyType?) -> String {
        switch type {
        case .go: return "Đi"
        case .search: return "Tìm"
        case .send: return "Gửi"
        case .next: return "Tiếp"
        case .done: return "Xong"
        case .join: return "Vào"
        default: return "⏎"
        }
    }
}
