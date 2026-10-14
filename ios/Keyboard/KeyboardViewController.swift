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
    private var clipBar: UIView!
    private var clipButton: UIButton!
    private var clipBarHeightConstraint: NSLayoutConstraint!
    private var heightConstraint: NSLayoutConstraint!

    private var vietOn = true
    private var doubleSpace = true
    private var lastClipChangeCount = -1
    private var clipCopiedAt = Date.distantPast
    private var dismissedClipChangeCount = -1
    private var currentClipRaw: String?
    private var currentClipImage: UIImage?
    private var clipHintWorkItem: DispatchWorkItem?

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
        clipBar = UIView()
        clipBar.translatesAutoresizingMaskIntoConstraints = false
        clipBar.isHidden = true

        clipButton = UIButton(type: .system)
        clipButton.translatesAutoresizingMaskIntoConstraints = false
        clipButton.layer.cornerRadius = 17
        clipButton.contentEdgeInsets = UIEdgeInsets(top: 5, left: 12, bottom: 5, right: 14)
        clipButton.titleLabel?.font = .systemFont(ofSize: 14, weight: .regular)
        clipButton.titleLabel?.lineBreakMode = .byTruncatingTail
        clipButton.addTarget(self, action: #selector(onTapClipSuggestion), for: .touchUpInside)
        clipBar.addSubview(clipButton)

        keyboard = KeyboardView()
        keyboard.listener = self
        keyboard.translatesAutoresizingMaskIntoConstraints = false
        view.addSubview(clipBar)
        view.addSubview(keyboard)

        clipBarHeightConstraint = clipBar.heightAnchor.constraint(equalToConstant: 0)
        NSLayoutConstraint.activate([
            clipBar.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            clipBar.trailingAnchor.constraint(equalTo: view.trailingAnchor),
            clipBar.topAnchor.constraint(equalTo: view.topAnchor),
            clipBarHeightConstraint,
            clipButton.centerXAnchor.constraint(equalTo: clipBar.centerXAnchor),
            clipButton.centerYAnchor.constraint(equalTo: clipBar.centerYAnchor, constant: 2),
            clipButton.widthAnchor.constraint(lessThanOrEqualToConstant: 260),
            clipButton.heightAnchor.constraint(equalToConstant: 34),
            keyboard.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            keyboard.trailingAnchor.constraint(equalTo: view.trailingAnchor),
            keyboard.topAnchor.constraint(equalTo: clipBar.bottomAnchor),
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
        heightConstraint.constant = keyboard.intrinsicContentSize.height + clipBarHeightConstraint.constant + view.safeAreaInsets.bottom
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
        refreshClipSuggestion()
    }

    override func textDidChange(_ textInput: UITextInput?) {
        super.textDidChange(textInput)
        updateAutoShift()
    }

    /// documentIdentifier is the per-host memory key. Without Full Access it
    /// is the zero UUID — treat that as unknown so a toggle stays session-only
    /// instead of leaking to a key shared by every restricted host.
    private var hostKey: String? {
        let id = textDocumentProxy.documentIdentifier
        return id == UUID(uuidString: "00000000-0000-0000-0000-000000000000") ? nil : id.uuidString
    }

    private func applyPrefs() {
        Prefs.apply(to: engine)
        let globalViet = hasFullAccess ? Prefs.vietOn() : (UserDefaults.standard.object(forKey: Prefs.keyVietOn) as? Bool ?? Prefs.vietOn())
        vietOn = Prefs.vietForApp(hostKey) ?? globalViet
        keyboard.theme = KeyboardThemes.current(dark: traitCollection.userInterfaceStyle == .dark)
        doubleSpace = Prefs.doubleSpacePeriod()
        keyboard.vietnameseOn = vietOn
        engine.setVietKey(vietOn)
        let isVni = Prefs.inputMethod() == ImVni
        keyboard.vniMode = isVni
        keyboard.showNumberRow = Prefs.numberRow() || isVni
        // Read-only access to the containing app's shared container is sufficient.
        if let url = Prefs.macroURL {
            if !FileManager.default.fileExists(atPath: url.path) { engine.loadMacroText("") }
            else if let text = try? String(contentsOf: url, encoding: .utf8) { engine.loadMacroText(text) }
        }
        refreshClipSuggestion()
        if heightConstraint != nil {
            heightConstraint.constant = keyboard.intrinsicContentSize.height + clipBarHeightConstraint.constant + view.safeAreaInsets.bottom
        }
        view.setNeedsLayout()
    }

    private func refreshClipSuggestion() {
        guard clipBar != nil else { return }
        clipHintWorkItem?.cancel()
        clipHintWorkItem = nil
        let t = keyboard.theme
        clipBar.backgroundColor = t.bg
        clipButton.backgroundColor = t.key
        clipButton.setTitleColor(t.text, for: .normal)
        clipButton.tintColor = t.funcText
        clipButton.layer.borderWidth = t.border.cgColor.alpha > 0 ? 1 : 0
        clipButton.layer.borderColor = t.border.resolvedColor(with: traitCollection).cgColor

        let enabled = hasFullAccess ? Prefs.clipboardSuggest() : (UserDefaults.standard.object(forKey: Prefs.keyClipboardSuggest) as? Bool ?? Prefs.clipboardSuggest())
        let isPassword = textDocumentProxy.isSecureTextEntry == true
        guard enabled, !isPassword else {
            hideClipBar()
            return
        }
        let pb = UIPasteboard.general
        let hasText = pb.hasStrings
        let hasImage = pb.hasImages
        guard hasText || hasImage else {
            hideClipBar()
            return
        }
        let changeCount = pb.changeCount
        if changeCount != lastClipChangeCount {
            lastClipChangeCount = changeCount
            clipCopiedAt = Date()
        }
        guard changeCount != dismissedClipChangeCount,
              Date().timeIntervalSince(clipCopiedAt) <= 180 else {
            hideClipBar()
            return
        }

        if hasImage, let img = pb.image {
            currentClipImage = img
            currentClipRaw = pb.string
            let thumb = Self.roundedThumbnail(from: img, size: CGSize(width: 22, height: 22), radius: 5)
            clipButton.setImage(thumb?.withRenderingMode(.alwaysOriginal), for: .normal)
            clipButton.setTitle("  Hình ảnh", for: .normal)
            clipBar.isHidden = false
            clipBarHeightConstraint.constant = 42
            return
        }

        guard let raw = pb.string?.trimmingCharacters(in: .whitespacesAndNewlines),
              !raw.isEmpty else {
            hideClipBar()
            return
        }
        currentClipImage = nil
        currentClipRaw = pb.string
        let singleLine = raw.replacingOccurrences(of: "\\s+", with: " ", options: .regularExpression)
        let display = singleLine.count > 26 ? String(singleLine.prefix(26)) + "…" : singleLine
        clipButton.setImage(UIImage(systemName: "doc.on.clipboard"), for: .normal)
        clipButton.setTitle("  " + display, for: .normal)
        clipBar.isHidden = false
        clipBarHeightConstraint.constant = 42
    }

    private static func roundedThumbnail(from image: UIImage, size: CGSize, radius: CGFloat) -> UIImage? {
        let renderer = UIGraphicsImageRenderer(size: size)
        return renderer.image { _ in
            UIBezierPath(roundedRect: CGRect(origin: .zero, size: size), cornerRadius: radius).addClip()
            let aspect = max(size.width / max(image.size.width, 1), size.height / max(image.size.height, 1))
            let drawW = image.size.width * aspect
            let drawH = image.size.height * aspect
            let rect = CGRect(x: (size.width - drawW) / 2, y: (size.height - drawH) / 2, width: drawW, height: drawH)
            image.draw(in: rect)
        }
    }

    private func hideClipBar() {
        clipHintWorkItem?.cancel()
        clipHintWorkItem = nil
        currentClipRaw = nil
        currentClipImage = nil
        clipBar?.isHidden = true
        clipBarHeightConstraint?.constant = 0
    }

    private func dismissClipSuggestion() {
        guard currentClipRaw != nil || currentClipImage != nil else { return }
        dismissedClipChangeCount = lastClipChangeCount
        hideClipBar()
        if heightConstraint != nil {
            heightConstraint.constant = keyboard.intrinsicContentSize.height + view.safeAreaInsets.bottom
        }
    }

    @objc private func onTapClipSuggestion() {
        if let img = currentClipImage {
            if let raw = currentClipRaw, !raw.isEmpty {
                dismissedClipChangeCount = lastClipChangeCount
                hideClipBar()
                engine.resetBuf()
                spacePending = false
                textDocumentProxy.insertText(raw)
                rememberTail()
                updateAutoShift()
                if heightConstraint != nil {
                    heightConstraint.constant = keyboard.intrinsicContentSize.height + view.safeAreaInsets.bottom
                }
                return
            }
            // iOS UITextDocumentProxy does not expose an image-commit API to keyboard extensions;
            // keep the image on UIPasteboard.general and prompt the user to paste in the host field.
            UIPasteboard.general.image = img
            clipButton.setTitle("  Chạm ô nhập → Dán ảnh", for: .normal)
            clipHintWorkItem?.cancel()
            let work = DispatchWorkItem { [weak self] in
                self?.dismissClipSuggestion()
            }
            clipHintWorkItem = work
            DispatchQueue.main.asyncAfter(deadline: .now() + 1.8, execute: work)
            return
        }
        guard let raw = currentClipRaw else { return }
        dismissedClipChangeCount = lastClipChangeCount
        hideClipBar()
        engine.resetBuf()
        spacePending = false
        textDocumentProxy.insertText(raw)
        rememberTail()
        updateAutoShift()
        if heightConstraint != nil {
            heightConstraint.constant = keyboard.intrinsicContentSize.height + view.safeAreaInsets.bottom
        }
    }

    // ---- KeyboardView.Listener ------------------------------------------------

    func onChar(_ ch: Character) {
        let proxy = textDocumentProxy
        dismissClipSuggestion()
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
        dismissClipSuggestion()
        if let sel = textDocumentProxy.selectedText, !sel.isEmpty {
            engine.resetBuf()
            spacePending = false
            textDocumentProxy.deleteBackward()
            rememberTail()
            updateAutoShift()
            return
        }
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
        dismissClipSuggestion()
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
        setViet(!vietOn)
    }

    private func setViet(_ on: Bool) {
        vietOn = on
        // Without Full Access an extension cannot write to the shared container.
        // Always change the current session; persist only when writing is allowed.
        if hasFullAccess { Prefs.setViet(hostKey, on: on) }
        UserDefaults.standard.set(on, forKey: Prefs.keyVietOn)
        keyboard.vietnameseOn = on
        engine.setVietKey(on)
        engine.resetBuf()
        spacePending = false
    }

    /// Long-press Space: quick settings for language and input method.
    func onSpaceMenu() {
        let alert = UIAlertController(title: nil, message: nil, preferredStyle: .actionSheet)
        let im = Prefs.inputMethod()
        func check(_ on: Bool) -> String { on ? "✓ " : "" }
        alert.addAction(UIAlertAction(title: check(vietOn) + "Tiếng Việt", style: .default) { [weak self] _ in self?.setViet(true) })
        alert.addAction(UIAlertAction(title: check(!vietOn) + "English", style: .default) { [weak self] _ in self?.setViet(false) })
        alert.addAction(UIAlertAction(title: check(im == ImTelex) + "Telex", style: .default) { [weak self] _ in
            Prefs.defaults.set(ImTelex, forKey: Prefs.keyIM)
            self?.applyPrefs()
        })
        alert.addAction(UIAlertAction(title: check(im == ImVni) + "VNI", style: .default) { [weak self] _ in
            Prefs.defaults.set(ImVni, forKey: Prefs.keyIM)
            self?.applyPrefs()
        })
        alert.addAction(UIAlertAction(title: check(im == ImViqr) + "VIQR", style: .default) { [weak self] _ in
            Prefs.defaults.set(ImViqr, forKey: Prefs.keyIM)
            self?.applyPrefs()
        })
        alert.addAction(UIAlertAction(title: "Bàn phím khác…", style: .default) { [weak self] _ in self?.advanceToNextInputMode() })
        alert.addAction(UIAlertAction(title: "Huỷ", style: .cancel))
        if let pop = alert.popoverPresentationController {
            pop.sourceView = keyboard
            pop.sourceRect = CGRect(x: keyboard.bounds.midX, y: keyboard.bounds.midY, width: 0, height: 0)
            pop.permittedArrowDirections = []
        }
        present(alert, animated: true)
    }

    func onSwitchIme() {
        advanceToNextInputMode()
    }

    func onInputModeMenu(_ sender: UIButton, event: UIEvent) {
        handleInputModeList(from: sender, with: event)
    }

    func onCursorMove(_ delta: Int) {
        engine.resetBuf()
        spacePending = false
        textDocumentProxy.adjustTextPosition(byCharacterOffset: delta)
        rememberTail()
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
        let ctx = textDocumentProxy.documentContextBeforeInput ?? ""
        expectedTail = String(ctx.suffix(64))
    }

    private func checkExternalChange() {
        if let sel = textDocumentProxy.selectedText, !sel.isEmpty {
            engine.resetBuf()
            spacePending = false
            return
        }
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
        let ctx = expectedTail.isEmpty ? (proxy.documentContextBeforeInput ?? "") : expectedTail
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
