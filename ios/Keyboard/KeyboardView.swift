import UIKit
import TuiGoEngine

// Canvas-free keyboard built on UIButtons, mirroring android/KeyboardView.kt:
// key popup preview, shift one-shot / caps-lock on double tap,
// backspace auto-repeat, letters/symbols pages, VN toggle, globe key.
final class KeyboardView: UIView {

    protocol Listener: AnyObject {
        func onChar(_ ch: Character)
        func onDelete()
        func onEnter()
        func onShiftChanged(_ shifted: Bool, _ capsLock: Bool)
        func onVietToggle()
        func onSwitchIme()
        func onSpaceMenu()
        func onInputModeMenu(_ sender: UIButton, event: UIEvent)
        func onCursorMove(_ delta: Int)
    }

    weak var listener: Listener?

    /// Keyboard skin; themes follow the Laban Key catalog (Shared/KeyboardThemes.swift).
    var theme = KeyboardThemes.current(dark: false) {
        didSet { keys.forEach { styleKey($0) }; setNeedsLayout() }
    }

    var vietnameseOn = true { didSet { if oldValue != vietnameseOn && vniMode && showNumberRow { rebuildKeys() } else { updateToggleLabels() } } }
    var vniMode = false { didSet { if oldValue != vniMode { rebuildKeys() } } }
    var showsGlobe = false { didSet { rebuildKeys() } }
    var returnLabel = "⏎" { didSet { updateToggleLabels() } }
    var showNumberRow = false { didSet { if oldValue != showNumberRow { rebuildKeys() } } }

    private(set) var symbolsPage = false
    private var symbolLevel = 0
    private(set) var shifted = false
    private(set) var capsLock = false
    private var manualShift = false
    private var spaceSliding = false
    private var spaceLastStepX: CGFloat = 0
    private weak var spaceHoldGesture: UILongPressGestureRecognizer?

    func resetManualShift() {
        manualShift = false
    }

    // ---- layout model -------------------------------------------------------

    private enum K {
        static let more = -9
        static let spacer = -7
        static let space = -5
        static let enter = -4
        static let mode = -3
        static let del = -2
        static let shift = -1
        static let globe = -8
    }

    private struct KeySpec {
        let code: Int
        let label: String
        let weight: CGFloat
        let hint: String
        let longPressChar: Character?
        init(_ code: Int, _ label: String, _ weight: CGFloat = 1, hint: String = "", longPressChar: Character? = nil) {
            self.code = code
            self.label = label
            self.weight = weight
            self.hint = hint
            self.longPressChar = longPressChar ?? hint.first
        }
    }

    private final class Key {
        let spec: KeySpec
        let button: UIButton
        let hintLabel: UILabel?
        init(_ spec: KeySpec) {
            self.spec = spec
            button = UIButton(type: .custom)
            button.layer.cornerRadius = 10
            button.titleLabel?.textAlignment = .center
            if !spec.hint.isEmpty {
                let lbl = UILabel()
                lbl.text = spec.hint
                lbl.font = .systemFont(ofSize: 10.5, weight: .medium)
                lbl.textAlignment = .right
                lbl.isUserInteractionEnabled = false
                button.addSubview(lbl)
                hintLabel = lbl
            } else {
                hintLabel = nil
            }
        }
    }

    private static let topRowNumberHints: [Character: Character] = [
        "q": "1", "w": "2", "e": "3", "r": "4", "t": "5",
        "y": "6", "u": "7", "i": "8", "o": "9", "p": "0",
    ]

    private static let letterHints: [Character: Character] = [
        "a": "@", "s": "#", "d": "$", "f": "%", "g": "&",
        "h": "-", "j": "+", "k": "(", "l": ")",
        "z": "*", "x": "\"", "c": "'", "v": ":",
        "b": ";", "n": "!", "m": "?",
    ]

    private static let vniNumberHints: [Character: String] = [
        "1": "´", "2": "`", "3": "?", "4": "~", "5": ".",
        "6": "^", "7": "ʼ", "8": "˘", "9": "Đ",
    ]

    private func row(_ s: String, inset: CGFloat = 0, withHints: Bool = false) -> [KeySpec] {
        let keys = s.map { ch -> KeySpec in
            let hintChar: Character? = !withHints ? nil : (Self.letterHints[ch] ?? (!showNumberRow ? Self.topRowNumberHints[ch] : nil))
            return KeySpec(
                Int(ch.unicodeScalars.first!.value),
                String(ch),
                hint: hintChar.map { String($0) } ?? "",
                longPressChar: hintChar
            )
        }
        guard inset > 0 else { return keys }
        return [KeySpec(K.spacer, "", inset)] + keys + [KeySpec(K.spacer, "", inset)]
    }

    private func numberRowSpecs() -> [KeySpec] {
        "1234567890".map { ch in
            let vniHint = (vniMode && vietnameseOn) ? (Self.vniNumberHints[ch] ?? "") : ""
            return KeySpec(Int(ch.unicodeScalars.first!.value), String(ch), hint: vniHint, longPressChar: nil)
        }
    }

    private func lettersRows() -> [[KeySpec]] {
        [
            row("qwertyuiop", withHints: true),
            row("asdfghjkl", inset: 0.5, withHints: true),
            [KeySpec(K.shift, "⇧", 1.5)] + row("zxcvbnm", withHints: true) + [KeySpec(K.del, "⌫", 1.5)],
            bottomRow(mode: "?123"),
        ]
    }

    private func symbolsRows() -> [[KeySpec]] {
        [
            row("1234567890"),
            row("@#₫_&-+()/"),
            [KeySpec(K.more, "#+=", 1.5)] + row("*\"':;!?") + [KeySpec(K.del, "⌫", 1.5)],
            bottomRow(mode: "ABC"),
        ]
    }

    private func legacySymbolRows() -> [[KeySpec]] {
        [row("^~`|\\€£¥%$"), row("°={}[]<>_₫"),
         [KeySpec(K.more, "?123", 1.5)] + row("!?'\":;") + [KeySpec(K.del, "⌫", 1.5)],
         bottomRow(mode: "ABC")]
    }

    private func extraSymbolRows() -> [[KeySpec]] {
        [row("~`|•√π÷×§Δ"), row("£€$¢^°={}\\"),
         [KeySpec(K.more, "¥<>", 1.5)] + row("%©®™✓[]") + [KeySpec(K.del, "⌫", 1.5)],
         bottomRow(mode: "ABC")]
    }

    private func bottomRow(mode: String) -> [KeySpec] {
        var specs = [KeySpec(K.mode, mode, 1.4)]
        if showsGlobe { specs.insert(KeySpec(K.globe, "", 0.8), at: 0) }
        specs.append(contentsOf: [
            KeySpec(Int(",".unicodeScalars.first!.value), ",", 0.9),
            KeySpec(K.space, "Tiếng Việt", 5.4),
            KeySpec(Int(".".unicodeScalars.first!.value), ".", 0.9),
            KeySpec(K.enter, "⏎", 1.4),
        ])
        return specs
    }

    private var rows: [[KeySpec]] {
        if symbolsPage {
            switch symbolLevel {
            case 1: return extraSymbolRows()
            case 2: return legacySymbolRows()
            default: return symbolsRows()
            }
        }
        if showNumberRow {
            return [numberRowSpecs()] + lettersRows()
        }
        return lettersRows()
    }
    private var keys = [Key]()

    // ---- metrics ------------------------------------------------------------

    private var keyHeight: CGFloat = 52
    private let rowGap: CGFloat = 6
    private var keyGap: CGFloat = 4
    private let padH: CGFloat = 3
    private let padV: CGFloat = 8

    private var effectiveKeyHeight: CGFloat {
        let base = min(max(keyHeight, 44), 68)
        return traitCollection.verticalSizeClass == .compact ? min(base, 40) : base
    }

    override var intrinsicContentSize: CGSize {
        let n = CGFloat(rows.count)
        let kh = effectiveKeyHeight
        let h = padV * 2 + n * kh + (n - 1) * rowGap
        return CGSize(width: UIView.noIntrinsicMetric, height: h)
    }

    func setMetrics(height: CGFloat, gap: CGFloat) {
        let clampedH = min(max(height, 44), 68)
        let clampedG = min(max(gap, 0), 8)
        guard keyHeight != clampedH || keyGap != clampedG else { return }
        keyHeight = clampedH; keyGap = clampedG
        invalidateIntrinsicContentSize(); setNeedsLayout()
    }

    private let gradient = CAGradientLayer()

    override init(frame: CGRect) {
        super.init(frame: frame)
        theme = KeyboardThemes.current(dark: traitCollection.userInterfaceStyle == .dark)
        layer.insertSublayer(gradient, at: 0)
        rebuildKeys()
    }

    required init?(coder: NSCoder) { fatalError("init(coder:) is not used") }

    // ---- key construction ---------------------------------------------------

    private func rebuildKeys() {
        clearPress()
        keys.forEach { $0.button.removeFromSuperview() }
        keys = rows.flatMap { $0 }.filter { $0.code != K.spacer }.map { spec in
            let key = Key(spec)
            let b = key.button
            if spec.code == K.globe {
                b.addTarget(self, action: #selector(onGlobe(_:event:)), for: .allTouchEvents)
            }
            if spec.code == K.space {
                let hold = UILongPressGestureRecognizer(target: self, action: #selector(onSpaceHold(_:)))
                hold.minimumPressDuration = 0.55
                hold.allowableMovement = 8
                b.addGestureRecognizer(hold)
                spaceHoldGesture = hold
                let pan = UIPanGestureRecognizer(target: self, action: #selector(onSpacePan(_:)))
                b.addGestureRecognizer(pan)
            }
            b.addTarget(self, action: #selector(onDown(_:)), for: .touchDown)
            b.addTarget(self, action: #selector(onUp(_:)), for: .touchUpInside)
            b.addTarget(self, action: #selector(onCancel(_:)), for: [.touchUpOutside, .touchCancel, .touchDragExit])
            b.addTarget(self, action: #selector(onDragEnter(_:)), for: .touchDragEnter)
            styleKey(key)
            addSubview(b)
            return key
        }
        updateToggleLabels()
        invalidateIntrinsicContentSize()
        setNeedsLayout()
    }

    private func styleKey(_ key: Key) {
        let b = key.button
        let code = key.spec.code
        if (code == K.enter || code == K.space) && theme.enterAccentFill {
            b.layer.cornerRadius = effectiveKeyHeight / 2
        } else {
            b.layer.cornerRadius = theme.corner
        }
        let border = theme.border
        b.layer.borderWidth = border.cgColor.alpha > 0 ? 1 : 0
        b.layer.borderColor = border.resolvedColor(with: traitCollection).cgColor
        paintKey(key)
    }

    private func paintKey(_ key: Key, pressed: Bool = false) {
        let b = key.button
        let code = key.spec.code
        let t = theme
        switch code {
        case K.shift:
            let active = capsLock || shifted
            if pressed {
                b.backgroundColor = t.pressAccent ? t.accent : t.funcPressed
                b.setTitleColor(t.pressAccent ? t.onAccent : t.funcText, for: .normal)
            } else if active {
                b.backgroundColor = t.blueGlyphs ? t.space : t.accent
                b.setTitleColor(t.blueGlyphs ? t.accent : t.onAccent, for: .normal)
            } else {
                b.backgroundColor = t.funcKey
                b.setTitleColor(t.funcText, for: .normal)
            }
        case K.enter:
            if pressed {
                b.backgroundColor = t.pressAccent ? t.accent : t.funcPressed
                b.setTitleColor(t.pressAccent ? t.onAccent : t.funcText, for: .normal)
            } else {
                b.backgroundColor = t.enterAccentFill ? t.accent : t.funcKey
                b.setTitleColor(t.enterAccentFill ? t.onAccent : (t.blueGlyphs ? t.accent : t.funcText), for: .normal)
            }
        case K.space:
            b.backgroundColor = (pressed && !t.pressAccent) ? t.funcPressed : t.space
            b.setTitleColor(pressed ? (t.pressAccent ? t.onAccent : t.spaceText) : t.spaceText, for: .normal)
            let sb = t.spaceBorder
            b.layer.borderWidth = sb.cgColor.alpha > 0 ? 1 : 0
            b.layer.borderColor = sb.resolvedColor(with: traitCollection).cgColor
        case K.del, K.mode, K.more, K.globe:
            b.backgroundColor = pressed ? (t.pressAccent ? t.accent : t.funcPressed) : t.funcKey
            b.setTitleColor(pressed ? (t.pressAccent ? t.onAccent : t.funcText) : t.funcText, for: .normal)
        default:
            b.backgroundColor = pressed ? (t.pressAccent ? t.accent : t.keyPressed) : t.key
            b.setTitleColor(pressed ? (t.pressAccent ? t.onAccent : t.text) : t.text, for: .normal)
            if let hl = key.hintLabel {
                hl.textColor = (pressed ? (t.pressAccent ? t.onAccent : t.text) : t.funcText).withAlphaComponent(0.67)
            }
        }
    }

    private func updateToggleLabels() {
        for key in keys {
            let b = key.button
            switch key.spec.code {
            case K.shift:
                b.setTitle(capsLock ? "⇪" : "⇧", for: .normal)
                b.titleLabel?.font = .systemFont(ofSize: 20, weight: .medium)
            case K.enter:
                b.setTitle(returnLabel, for: .normal)
                b.titleLabel?.font = .systemFont(ofSize: returnLabel.count > 2 ? 14 : 19, weight: .medium)
            case K.mode, K.more:
                b.setTitle(key.spec.label, for: .normal)
                b.titleLabel?.font = .systemFont(ofSize: 14, weight: .medium)
            case K.del:
                b.setTitle("⌫", for: .normal)
                b.titleLabel?.font = .systemFont(ofSize: 19, weight: .regular)
            case K.globe:
                b.setTitle(nil, for: .normal)
                b.setImage(UIImage(systemName: "globe"), for: .normal)
                b.tintColor = theme.funcText
                b.accessibilityLabel = "Chọn bàn phím"
                b.titleLabel?.font = .systemFont(ofSize: 17, weight: .regular)
            case K.space:
                let base = vietnameseOn ? "Tiếng Việt" : "English"
                b.setTitle(spaceSliding ? "◀   \(base)   ▶" : base, for: .normal)
                b.titleLabel?.font = .systemFont(ofSize: 13, weight: .regular)
            default:
                b.setTitle(currentLabel(key.spec), for: .normal)
                b.titleLabel?.font = .systemFont(ofSize: 21, weight: .regular)
            }
            paintKey(key)
        }
    }

    private func currentLabel(_ spec: KeySpec) -> String {
        if shifted, spec.label.count == 1, spec.label.first!.isLetter {
            return spec.label.uppercased()
        }
        return spec.label
    }

    // ---- layout --------------------------------------------------------------

    override func layoutSubviews() {
        super.layoutSubviews()
        gradient.frame = bounds
        gradient.colors = [theme.bg.resolvedColor(with: traitCollection).cgColor, theme.bgEnd.resolvedColor(with: traitCollection).cgColor]
        let kh = effectiveKeyHeight
        var iter = keys.makeIterator()
        var y = padV
        for rowSpec in rows {
            let total = rowSpec.reduce(CGFloat(0)) { $0 + $1.weight }
            let unitW = (bounds.width - 2 * padH - keyGap * CGFloat(rowSpec.count - 1)) / max(total, 1)
            var x = padH
            for spec in rowSpec {
                let kw = unitW * spec.weight
                if spec.code != K.spacer, let key = iter.next() {
                    key.button.frame = CGRect(x: x, y: y, width: kw, height: kh)
                    key.hintLabel?.frame = CGRect(x: max(0, kw - 18), y: 2, width: 14, height: 13)
                    if (spec.code == K.enter || spec.code == K.space) && theme.enterAccentFill {
                        key.button.layer.cornerRadius = kh / 2
                    }
                }
                x += kw + keyGap
            }
            y += kh + rowGap
        }
    }

    override func traitCollectionDidChange(_ previousTraitCollection: UITraitCollection?) {
        super.traitCollectionDidChange(previousTraitCollection)
        if traitCollection.hasDifferentColorAppearance(comparedTo: previousTraitCollection) {
            theme = KeyboardThemes.current(dark: traitCollection.userInterfaceStyle == .dark)
            keys.forEach { styleKey($0) }
            hidePopup(); setNeedsLayout()
        }
        if previousTraitCollection?.verticalSizeClass != traitCollection.verticalSizeClass {
            invalidateIntrinsicContentSize()
            setNeedsLayout()
        }
    }

    override func didMoveToWindow() {
        super.didMoveToWindow()
        if window == nil { clearPress() }
    }

    // ---- popup preview --------------------------------------------------------

    private var popup: UIView?

    private func showPopup(for key: Key) {
        guard key.spec.code > 0 else { return }
        hidePopup()
        let r = key.button.frame
        let pw = r.width * 1.5
        let ph = effectiveKeyHeight * 1.15
        let px = min(max(r.midX - pw / 2, padH), bounds.width - padH - pw)
        let py = max(r.minY - ph - rowGap, 2)
        let v = UIView(frame: CGRect(x: px, y: py, width: pw, height: ph))
        v.backgroundColor = theme.popup
        v.layer.cornerRadius = 8
        v.layer.borderWidth = theme.popupBorder.cgColor.alpha > 0 ? 1 : 0
        v.layer.borderColor = theme.popupBorder.resolvedColor(with: traitCollection).cgColor
        let l = UILabel(frame: v.bounds)
        l.text = currentLabel(key.spec)
        l.textAlignment = .center
        l.textColor = theme.popupText
        l.font = .systemFont(ofSize: 26, weight: .medium)
        v.addSubview(l)
        addSubview(v)
        popup = v
    }

    private func hidePopup() {
        popup?.removeFromSuperview()
        popup = nil
    }

    // ---- delete repeat & long-press hint ---------------------------------------

    private var pressed: Key?
    private var shiftTapTime = Date.distantPast
    private var deleteTimer: Timer?
    private var hintTimer: Timer?
    private let tapFeedback = UIImpactFeedbackGenerator(style: .light)
    private let longPressFeedback = UIImpactFeedbackGenerator(style: .medium)
    private let selectionFeedback = UISelectionFeedbackGenerator()

    private func startDeleteRepeat() {
        stopDeleteRepeat()
        manualShift = false
        listener?.onDelete()
        deleteTimer = Timer.scheduledTimer(withTimeInterval: 0.4, repeats: false) { [weak self] _ in
            self?.deleteTimer = Timer.scheduledTimer(withTimeInterval: 0.05, repeats: true) { [weak self] _ in
                self?.listener?.onDelete()
            }
        }
    }

    private func stopDeleteRepeat() {
        deleteTimer?.invalidate()
        deleteTimer = nil
    }

    private func startHintTimer(for key: Key) {
        stopHintTimer()
        guard let alt = key.spec.longPressChar else { return }
        hintTimer = Timer.scheduledTimer(withTimeInterval: 0.45, repeats: false) { [weak self] _ in
            guard let self = self, self.pressed === key else { return }
            self.clearPress()
            self.manualShift = false
            self.longPressFeedback.impactOccurred()
            self.listener?.onChar(alt)
        }
    }

    private func stopHintTimer() {
        hintTimer?.invalidate()
        hintTimer = nil
    }

    // ---- touch callbacks --------------------------------------------------------

    @objc private func onDown(_ b: UIButton) {
        guard let key = keys.first(where: { $0.button === b }) else { return }
        if key.spec.code == K.globe { return }
        pressed = key
        paintKey(key, pressed: true)
        if key.spec.code == K.del {
            startDeleteRepeat()
        } else {
            showPopup(for: key)
            startHintTimer(for: key)
        }
        tapFeedback.impactOccurred()
        tapFeedback.prepare()
    }

    @objc private func onGlobe(_ sender: UIButton, event: UIEvent) {
        listener?.onInputModeMenu(sender, event: event)
    }

    /// Long-press Space = language/input-method menu (Gboard-style).
    @objc private func onSpaceHold(_ g: UILongPressGestureRecognizer) {
        guard g.state == .began, !spaceSliding else { return }
        clearPress()
        longPressFeedback.impactOccurred()
        listener?.onSpaceMenu()
    }

    /// Horizontal swipe on Space = move cursor left/right (Gboard-style).
    @objc private func onSpacePan(_ g: UIPanGestureRecognizer) {
        switch g.state {
        case .began:
            spaceSliding = true
            manualShift = false
            spaceLastStepX = 0
            spaceHoldGesture?.isEnabled = false
            spaceHoldGesture?.isEnabled = true
            clearPress()
            updateToggleLabels()
        case .changed:
            if !spaceSliding {
                spaceSliding = true
                manualShift = false
                spaceLastStepX = 0
                spaceHoldGesture?.isEnabled = false
                spaceHoldGesture?.isEnabled = true
                clearPress()
                updateToggleLabels()
            }
            let x = g.translation(in: self).x
            let stepPt: CGFloat = 10
            let steps = Int((x - spaceLastStepX) / stepPt)
            if steps != 0 {
                spaceLastStepX += CGFloat(steps) * stepPt
                let dir = steps > 0 ? 1 : -1
                for _ in 0..<abs(steps) { listener?.onCursorMove(dir) }
                selectionFeedback.selectionChanged()
            }
        case .ended, .cancelled, .failed:
            spaceSliding = false
            clearPress()
            updateToggleLabels()
        default:
            break
        }
    }

    @objc private func onUp(_ b: UIButton) {
        if spaceSliding { spaceSliding = false; clearPress(); updateToggleLabels(); return }
        guard let key = pressed, key.button === b else { clearPress(); return }
        clearPress()
        if key.spec.code != K.del { dispatch(key) }
    }

    @objc private func onCancel(_ b: UIButton) {
        if pressed?.button === b { clearPress() }
    }

    @objc private func onDragEnter(_ b: UIButton) {
        guard let key = keys.first(where: { $0.button === b }) else { return }
        clearPress()
        pressed = key
        paintKey(key, pressed: true)
        if key.spec.code == K.del {
            startDeleteRepeat()
        } else {
            startHintTimer(for: key)
        }
    }

    private func clearPress() {
        if let p = pressed { paintKey(p) }
        pressed = nil
        stopDeleteRepeat()
        stopHintTimer()
        hidePopup()
    }

    // ---- dispatch -----------------------------------------------------------------

    private func dispatch(_ key: Key) {
        switch key.spec.code {
        case K.shift:
            let now = Date()
            if capsLock {
                capsLock = false; shifted = false; manualShift = false
            } else if shifted, now.timeIntervalSince(shiftTapTime) < 0.35 {
                capsLock = true; manualShift = false
            } else {
                shifted.toggle(); manualShift = true
            }
            shiftTapTime = (shifted || capsLock) ? now : .distantPast
            listener?.onShiftChanged(shifted, capsLock)
        case K.del:
            break // handled on touch-down / repeat
        case K.mode:
            symbolsPage.toggle()
            symbolLevel = 0
            rebuildKeys()
        case K.more:
            symbolLevel = (symbolLevel + 1) % 3
            rebuildKeys()
        case K.enter:
            manualShift = false
            listener?.onEnter()
        case K.space:
            manualShift = false
            listener?.onChar(" ")
        case K.globe:
            listener?.onSwitchIme()
        default:
            manualShift = false
            listener?.onChar(currentLabel(key.spec).first!)
            if shifted, !capsLock {
                shifted = false
                listener?.onShiftChanged(false, false)
            }
        }
        updateToggleLabels()
    }

    /// One-shot shift from the host (autocapitalization after ". " etc).
    func setAutoShift(_ on: Bool) {
        guard !capsLock, !manualShift, shifted != on else { return }
        shifted = on
        updateToggleLabels()
        listener?.onShiftChanged(shifted, capsLock)
    }

}
