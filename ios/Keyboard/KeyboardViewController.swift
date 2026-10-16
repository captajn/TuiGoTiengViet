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
    private var clipScroll: UIScrollView!
    private var clipStack: UIStackView!
    private var clipButton: UIButton!
    private var historyButton: UIButton!
    private var clipBarHeightConstraint: NSLayoutConstraint!
    private var heightConstraint: NSLayoutConstraint!
    private var clipPanelShade: UIButton?
    private var clipPanel: UIView?

    private var vietOn = true
    private var doubleSpace = true
    private var lastClipChangeCount = -1
    private var clipCopiedAt = Date.distantPast
    private var dismissedClipChangeCount = -1
    private var currentClipRaw: String?
    private var currentClipImage: UIImage?
    private var clipHintWorkItem: DispatchWorkItem?
    /// changeCount of a pasteboard write we made ourselves (image re-queued
    /// for the host app) — never archived back into history.
    private var selfWrittenChangeCount = -1

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

        // Horizontally scrollable chip strip — same role as the Android
        // suggestionScroll/strip: leading history icon, then the paste chip.
        clipScroll = UIScrollView()
        clipScroll.translatesAutoresizingMaskIntoConstraints = false
        clipScroll.showsHorizontalScrollIndicator = false
        clipStack = UIStackView()
        clipStack.translatesAutoresizingMaskIntoConstraints = false
        clipStack.axis = .horizontal
        clipStack.spacing = 6
        clipStack.alignment = .center
        clipStack.isLayoutMarginsRelativeArrangement = true
        clipStack.directionalLayoutMargins = NSDirectionalEdgeInsets(top: 6, leading: 12, bottom: 2, trailing: 12)
        clipScroll.addSubview(clipStack)
        clipBar.addSubview(clipScroll)

        historyButton = UIButton(type: .system)
        historyButton.translatesAutoresizingMaskIntoConstraints = false
        historyButton.layer.cornerRadius = 17
        historyButton.contentEdgeInsets = UIEdgeInsets(top: 5, left: 11, bottom: 5, right: 11)
        historyButton.setImage(UIImage(systemName: "clock.arrow.circlepath"), for: .normal)
        historyButton.accessibilityLabel = "Lịch sử bộ nhớ tạm"
        historyButton.isHidden = true
        historyButton.addTarget(self, action: #selector(onTapHistory), for: .touchUpInside)

        clipButton = UIButton(type: .system)
        clipButton.translatesAutoresizingMaskIntoConstraints = false
        clipButton.layer.cornerRadius = 17
        clipButton.contentEdgeInsets = UIEdgeInsets(top: 5, left: 12, bottom: 5, right: 14)
        clipButton.titleLabel?.font = .systemFont(ofSize: 14, weight: .regular)
        clipButton.titleLabel?.lineBreakMode = .byTruncatingTail
        clipButton.addTarget(self, action: #selector(onTapClipSuggestion), for: .touchUpInside)

        clipStack.addArrangedSubview(historyButton)
        clipStack.addArrangedSubview(clipButton)

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
            clipScroll.leadingAnchor.constraint(equalTo: clipBar.leadingAnchor),
            clipScroll.trailingAnchor.constraint(equalTo: clipBar.trailingAnchor),
            clipScroll.topAnchor.constraint(equalTo: clipBar.topAnchor),
            clipScroll.bottomAnchor.constraint(equalTo: clipBar.bottomAnchor),
            clipStack.leadingAnchor.constraint(equalTo: clipScroll.contentLayoutGuide.leadingAnchor),
            clipStack.trailingAnchor.constraint(equalTo: clipScroll.contentLayoutGuide.trailingAnchor),
            clipStack.topAnchor.constraint(equalTo: clipScroll.contentLayoutGuide.topAnchor),
            clipStack.bottomAnchor.constraint(equalTo: clipScroll.contentLayoutGuide.bottomAnchor),
            clipStack.heightAnchor.constraint(equalTo: clipScroll.frameLayoutGuide.heightAnchor),
            clipButton.widthAnchor.constraint(lessThanOrEqualToConstant: 260),
            clipButton.heightAnchor.constraint(equalToConstant: 34),
            historyButton.heightAnchor.constraint(equalToConstant: 34),
            keyboard.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            keyboard.trailingAnchor.constraint(equalTo: view.trailingAnchor),
            keyboard.topAnchor.constraint(equalTo: clipBar.bottomAnchor),
            keyboard.bottomAnchor.constraint(equalTo: view.bottomAnchor),
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
        updateHeight()
    }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        applyPrefs()
        keyboard.showsGlobe = needsInputModeSwitchKey
        keyboard.returnLabel = Self.returnLabel(for: textDocumentProxy.returnKeyType)
        engine.resetBuf()
        spacePending = false
        rememberTail()
        keyboard.resetManualShift()
        updateAutoShift()
        recordClipboardIfChanged()
        refreshClipSuggestion()
    }

    override func viewWillDisappear(_ animated: Bool) {
        super.viewWillDisappear(animated)
        dismissClipPanel()
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
        updateHeight()
        view.setNeedsLayout()
    }

    /// Archive the current clip if the pasteboard moved since our last look —
    /// iOS has no clipboard-change callback for extensions, so this is polled
    /// on keyboard appear and on every keystroke. Returns true when a new clip
    /// was observed (caller should refresh the strip rather than dismiss it).
    @discardableResult
    private func recordClipboardIfChanged() -> Bool {
        let pb = UIPasteboard.general
        let cc = pb.changeCount
        guard cc != lastClipChangeCount else { return false }
        lastClipChangeCount = cc
        clipCopiedAt = Date()
        if cc == selfWrittenChangeCount {
            selfWrittenChangeCount = -1
            return false
        }
        if hasFullAccess {
            ClipboardHistory.record(pb, secureField: textDocumentProxy.isSecureTextEntry == true)
        }
        return true
    }

    private func refreshClipSuggestion() {
        guard clipBar != nil else { return }
        clipHintWorkItem?.cancel()
        clipHintWorkItem = nil
        let t = keyboard.theme
        clipBar.backgroundColor = t.bg
        for chip in [clipButton!, historyButton!] {
            chip.backgroundColor = t.key
            chip.setTitleColor(t.text, for: .normal)
            chip.tintColor = t.funcText
            chip.layer.borderWidth = t.border.cgColor.alpha > 0 ? 1 : 0
            chip.layer.borderColor = t.border.resolvedColor(with: traitCollection).cgColor
        }

        let showHistory = Prefs.clipboardHistory() && ClipboardHistory.hasEntries
        historyButton.isHidden = !showHistory

        var showClip = false
        defer {
            clipButton.isHidden = !showClip
            clipBar.isHidden = !showClip && !showHistory
            clipBarHeightConstraint.constant = (showClip || showHistory) ? 42 : 0
            updateHeight()
        }

        let enabled = hasFullAccess ? Prefs.clipboardSuggest() : (UserDefaults.standard.object(forKey: Prefs.keyClipboardSuggest) as? Bool ?? Prefs.clipboardSuggest())
        let isPassword = textDocumentProxy.isSecureTextEntry == true
        guard enabled, !isPassword else { return }
        let pb = UIPasteboard.general
        let hasText = pb.hasStrings
        let hasImage = pb.hasImages
        guard hasText || hasImage else { return }
        let changeCount = pb.changeCount
        if changeCount != lastClipChangeCount {
            lastClipChangeCount = changeCount
            clipCopiedAt = Date()
        }
        guard changeCount != dismissedClipChangeCount,
              Date().timeIntervalSince(clipCopiedAt) <= 180 else { return }

        if hasImage, let img = pb.image {
            currentClipImage = img
            currentClipRaw = pb.string
            let thumb = Self.roundedThumbnail(from: img, size: CGSize(width: 22, height: 22), radius: 5)
            clipButton.setImage(thumb?.withRenderingMode(.alwaysOriginal), for: .normal)
            clipButton.setTitle("  Hình ảnh", for: .normal)
            showClip = true
            return
        }

        guard let raw = pb.string?.trimmingCharacters(in: .whitespacesAndNewlines),
              !raw.isEmpty else { return }
        currentClipImage = nil
        currentClipRaw = pb.string
        let singleLine = raw.replacingOccurrences(of: "\\s+", with: " ", options: .regularExpression)
        let display = singleLine.count > 26 ? String(singleLine.prefix(26)) + "…" : singleLine
        clipButton.setImage(UIImage(systemName: "doc.on.clipboard"), for: .normal)
        clipButton.setTitle("  " + display, for: .normal)
        showClip = true
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

    /// Hide only the paste chip — the history icon keeps the strip alive.
    private func hideClipBar() {
        clipHintWorkItem?.cancel()
        clipHintWorkItem = nil
        currentClipRaw = nil
        currentClipImage = nil
        clipButton?.isHidden = true
        let historyShown = historyButton?.isHidden == false
        clipBar?.isHidden = !historyShown
        clipBarHeightConstraint?.constant = historyShown ? 42 : 0
        updateHeight()
    }

    private func updateHeight() {
        heightConstraint?.constant = keyboard.intrinsicContentSize.height + clipBarHeightConstraint.constant + view.safeAreaInsets.bottom
    }

    private func dismissClipSuggestion() {
        guard currentClipRaw != nil || currentClipImage != nil else { return }
        dismissedClipChangeCount = lastClipChangeCount
        hideClipBar()
        updateHeight()
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
                updateHeight()
                return
            }
            // iOS UITextDocumentProxy does not expose an image-commit API to keyboard extensions;
            // keep the image on UIPasteboard.general and prompt the user to paste in the host field.
            UIPasteboard.general.image = img
            selfWrittenChangeCount = UIPasteboard.general.changeCount
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
        updateHeight()
    }

    // ---- clipboard history panel ------------------------------------------------

    /// Button carrying the history-entry id it acts on.
    private final class ClipEntryButton: UIButton {
        var entryId: Int64 = 0
    }

    @objc private func onTapHistory() {
        if clipPanel != nil { dismissClipPanel() } else { showClipPanel() }
    }

    @objc private func onTapClipShade() { dismissClipPanel() }

    private func dismissClipPanel() {
        clipPanel?.removeFromSuperview()
        clipPanel = nil
        clipPanelShade?.removeFromSuperview()
        clipPanelShade = nil
    }

    /// Panel floating over the bottom of the keyboard — same role as the
    /// Android PopupWindow in showClipboardPanel(): tap a row to paste it,
    /// pin it (survives the history cap) or delete it.
    private func showClipPanel() {
        dismissClipPanel()
        let t = keyboard.theme
        let entries = ClipboardHistory.entries()

        let shade = UIButton(type: .custom)
        shade.translatesAutoresizingMaskIntoConstraints = false
        shade.addTarget(self, action: #selector(onTapClipShade), for: .touchUpInside)
        view.addSubview(shade)
        NSLayoutConstraint.activate([
            shade.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            shade.trailingAnchor.constraint(equalTo: view.trailingAnchor),
            shade.topAnchor.constraint(equalTo: view.topAnchor),
            shade.bottomAnchor.constraint(equalTo: view.bottomAnchor),
        ])
        clipPanelShade = shade

        let column = UIStackView()
        column.axis = .vertical
        column.spacing = 2
        column.translatesAutoresizingMaskIntoConstraints = false
        column.isLayoutMarginsRelativeArrangement = true
        column.directionalLayoutMargins = NSDirectionalEdgeInsets(top: 8, leading: 10, bottom: 8, trailing: 6)

        let header = UIStackView()
        header.axis = .horizontal
        header.alignment = .center
        let title = UILabel()
        title.text = "Bộ nhớ tạm"
        title.font = .systemFont(ofSize: 14, weight: .bold)
        title.textColor = t.popupText
        header.addArrangedSubview(title)
        if !entries.isEmpty {
            let clear = UIButton(type: .system)
            clear.setTitle("Xóa hết", for: .normal)
            clear.titleLabel?.font = .systemFont(ofSize: 13)
            clear.setTitleColor(t.accent, for: .normal)
            clear.contentEdgeInsets = UIEdgeInsets(top: 4, left: 12, bottom: 4, right: 4)
            clear.addTarget(self, action: #selector(onClearClipHistory), for: .touchUpInside)
            header.addArrangedSubview(clear)
        }
        column.addArrangedSubview(header)

        if entries.isEmpty {
            let empty = UILabel()
            empty.text = "Chưa có gì — mục đã sao chép sẽ hiện ở đây (tối đa \(Prefs.clipboardHistoryMax()))"
            empty.font = .systemFont(ofSize: 13)
            empty.textColor = t.popupText
            empty.alpha = 0.7
            empty.numberOfLines = 0
            column.addArrangedSubview(empty)
        }
        for e in entries { column.addArrangedSubview(clipRow(e, theme: t)) }

        let scroll = UIScrollView()
        scroll.translatesAutoresizingMaskIntoConstraints = false
        scroll.showsVerticalScrollIndicator = entries.count > 5
        scroll.addSubview(column)
        NSLayoutConstraint.activate([
            column.leadingAnchor.constraint(equalTo: scroll.contentLayoutGuide.leadingAnchor),
            column.trailingAnchor.constraint(equalTo: scroll.contentLayoutGuide.trailingAnchor),
            column.topAnchor.constraint(equalTo: scroll.contentLayoutGuide.topAnchor),
            column.bottomAnchor.constraint(equalTo: scroll.contentLayoutGuide.bottomAnchor),
            column.widthAnchor.constraint(equalTo: scroll.frameLayoutGuide.widthAnchor),
        ])

        let panel = UIView()
        panel.translatesAutoresizingMaskIntoConstraints = false
        panel.backgroundColor = t.popup
        panel.layer.cornerRadius = 12
        panel.clipsToBounds = true
        if t.popupBorder.cgColor.alpha > 0 {
            panel.layer.borderWidth = 1
            panel.layer.borderColor = t.popupBorder.resolvedColor(with: traitCollection).cgColor
        }
        panel.addSubview(scroll)
        view.addSubview(panel)
        // Fit the panel to its rows, capped at 260pt like the Android ~320dp.
        let fitRows = scroll.heightAnchor.constraint(equalTo: column.heightAnchor)
        fitRows.priority = .defaultHigh
        NSLayoutConstraint.activate([
            scroll.leadingAnchor.constraint(equalTo: panel.leadingAnchor),
            scroll.trailingAnchor.constraint(equalTo: panel.trailingAnchor),
            scroll.topAnchor.constraint(equalTo: panel.topAnchor),
            scroll.bottomAnchor.constraint(equalTo: panel.bottomAnchor),
            fitRows,
            scroll.heightAnchor.constraint(lessThanOrEqualToConstant: 260),
            panel.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 12),
            panel.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -12),
            panel.bottomAnchor.constraint(equalTo: view.bottomAnchor, constant: -(view.safeAreaInsets.bottom + 8)),
        ])
        clipPanel = panel
    }

    private func clipRow(_ e: ClipboardHistory.Entry, theme t: KeyTheme) -> UIView {
        let row = UIStackView()
        row.axis = .horizontal
        row.alignment = .center
        row.spacing = 6
        row.isLayoutMarginsRelativeArrangement = true
        row.directionalLayoutMargins = NSDirectionalEdgeInsets(top: 4, leading: 4, bottom: 4, trailing: 4)

        if let name = e.image, let img = ClipboardHistory.image(named: name) {
            let thumb = UIImageView(image: Self.roundedThumbnail(from: img, size: CGSize(width: 30, height: 30), radius: 6)?.withRenderingMode(.alwaysOriginal))
            thumb.contentMode = .scaleAspectFill
            thumb.widthAnchor.constraint(equalToConstant: 30).isActive = true
            thumb.heightAnchor.constraint(equalToConstant: 30).isActive = true
            row.addArrangedSubview(thumb)
        } else {
            let icon = UIImageView(image: UIImage(systemName: "doc.on.clipboard"))
            icon.tintColor = t.funcText
            icon.contentMode = .scaleAspectFit
            icon.widthAnchor.constraint(equalToConstant: 18).isActive = true
            row.addArrangedSubview(icon)
        }

        let label = ClipEntryButton(type: .system)
        label.entryId = e.id
        let preview = e.text?.trimmingCharacters(in: .whitespacesAndNewlines)
            .replacingOccurrences(of: "\\s+", with: " ", options: .regularExpression)
        label.setTitle(preview?.isEmpty == false ? preview : "Hình ảnh", for: .normal)
        label.setTitleColor(t.popupText, for: .normal)
        label.titleLabel?.font = .systemFont(ofSize: 14)
        label.titleLabel?.numberOfLines = 2
        label.titleLabel?.lineBreakMode = .byTruncatingTail
        label.contentHorizontalAlignment = .leading
        label.addTarget(self, action: #selector(onPasteClipEntry(_:)), for: .touchUpInside)
        label.setContentHuggingPriority(.defaultLow, for: .horizontal)
        row.addArrangedSubview(label)

        func action(_ title: String, _ color: UIColor, _ sel: Selector) -> ClipEntryButton {
            let b = ClipEntryButton(type: .system)
            b.entryId = e.id
            b.setTitle(title, for: .normal)
            b.titleLabel?.font = .systemFont(ofSize: 12)
            b.setTitleColor(color, for: .normal)
            b.contentEdgeInsets = UIEdgeInsets(top: 8, left: 8, bottom: 8, right: 8)
            b.setContentCompressionResistancePriority(.required, for: .horizontal)
            b.addTarget(self, action: sel, for: .touchUpInside)
            return b
        }
        row.addArrangedSubview(action(e.pinned ? "Bỏ ghim" : "Ghim", t.accent, #selector(onPinClipEntry(_:))))
        row.addArrangedSubview(action("✕", t.popupText, #selector(onDeleteClipEntry(_:))))
        return row
    }

    @objc private func onPasteClipEntry(_ sender: ClipEntryButton) {
        guard let e = ClipboardHistory.entries().first(where: { $0.id == sender.entryId }) else { return }
        pasteClipEntry(e)
    }

    @objc private func onPinClipEntry(_ sender: ClipEntryButton) {
        guard let e = ClipboardHistory.entries().first(where: { $0.id == sender.entryId }) else { return }
        ClipboardHistory.setPinned(e.id, !e.pinned)
        showClipPanel()
        refreshClipSuggestion()
    }

    @objc private func onDeleteClipEntry(_ sender: ClipEntryButton) {
        ClipboardHistory.remove(sender.entryId)
        showClipPanel()
        refreshClipSuggestion()
    }

    @objc private func onClearClipHistory() {
        ClipboardHistory.clear()
        dismissClipPanel()
        refreshClipSuggestion()
    }

    private func pasteClipEntry(_ e: ClipboardHistory.Entry) {
        dismissClipPanel()
        engine.resetBuf()
        spacePending = false
        if let name = e.image {
            // UITextDocumentProxy has no image-commit API — put the image back
            // on the pasteboard so the user can paste in the host field.
            guard let img = ClipboardHistory.image(named: name) else { return }
            UIPasteboard.general.image = img
            selfWrittenChangeCount = UIPasteboard.general.changeCount
            clipButton.setImage(UIImage(systemName: "doc.on.clipboard"), for: .normal)
            clipButton.setTitle("  Chạm ô nhập → Dán ảnh", for: .normal)
            clipButton.isHidden = false
            clipBar.isHidden = false
            clipBarHeightConstraint.constant = 42
            updateHeight()
            clipHintWorkItem?.cancel()
            let work = DispatchWorkItem { [weak self] in self?.hideClipBar() }
            clipHintWorkItem = work
            DispatchQueue.main.asyncAfter(deadline: .now() + 1.8, execute: work)
            return
        }
        guard let text = e.text else { return }
        textDocumentProxy.insertText(text)
        rememberTail()
        updateAutoShift()
    }

    // ---- KeyboardView.Listener ------------------------------------------------

    func onChar(_ ch: Character) {
        let proxy = textDocumentProxy
        if recordClipboardIfChanged() { refreshClipSuggestion() } else { dismissClipSuggestion() }
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
        if recordClipboardIfChanged() { refreshClipSuggestion() } else { dismissClipSuggestion() }
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
        if recordClipboardIfChanged() { refreshClipSuggestion() } else { dismissClipSuggestion() }
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
            keyboard.resetManualShift()
            return
        }
        let ctx = textDocumentProxy.documentContextBeforeInput ?? ""
        if !ctx.hasSuffix(expectedTail) {
            engine.resetBuf()
            spacePending = false
            keyboard.resetManualShift()
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
        let enabled = hasFullAccess ? Prefs.autoCapitalize() : (UserDefaults.standard.object(forKey: Prefs.keyAutoCapitalize) as? Bool ?? Prefs.autoCapitalize())
        guard enabled, !isRestrictedField,
              proxy.isSecureTextEntry != true,
              proxy.keyboardType != .URL,
              proxy.keyboardType != .emailAddress,
              proxy.autocapitalizationType != UITextAutocapitalizationType.none else {
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
