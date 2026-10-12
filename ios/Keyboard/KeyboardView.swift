import UIKit

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
        func onInputModeMenu(_ sender: UIButton, event: UIEvent)
    }

    weak var listener: Listener?

    var vietnameseOn = true { didSet { updateToggleLabels() } }
    var showsGlobe = false { didSet { rebuildKeys() } }
    var returnLabel = "⏎" { didSet { updateToggleLabels() } }
    var showNumberRow = false { didSet { if oldValue != showNumberRow { rebuildKeys() } } }

    private(set) var symbolsPage = false
    private var extraSymbols = false
    private(set) var shifted = false
    private(set) var capsLock = false

    // ---- layout model -------------------------------------------------------

    private enum K {
        static let more = -9
        static let spacer = -7
        static let viet = -6
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
        init(_ code: Int, _ label: String, _ weight: CGFloat = 1) {
            self.code = code; self.label = label; self.weight = weight
        }
    }

    private final class Key {
        let spec: KeySpec
        let button: UIButton
        init(_ spec: KeySpec) {
            self.spec = spec
            button = UIButton(type: .custom)
            button.layer.cornerRadius = 10
            button.titleLabel?.textAlignment = .center
        }
    }

    private func row(_ s: String, inset: CGFloat = 0) -> [KeySpec] {
        let keys = s.map { KeySpec(Int($0.unicodeScalars.first!.value), String($0)) }
        guard inset > 0 else { return keys }
        return [KeySpec(K.spacer, "", inset)] + keys + [KeySpec(K.spacer, "", inset)]
    }

    private func lettersRows() -> [[KeySpec]] {
        [
            row("qwertyuiop"),
            row("asdfghjkl", inset: 0.5),
            [KeySpec(K.shift, "⇧", 1.5)] + row("zxcvbnm") + [KeySpec(K.del, "⌫", 1.5)],
            bottomRow(mode: "?123"),
        ]
    }

    private func symbolsRows() -> [[KeySpec]] {
        [
            row("1234567890"),
            row("@#₫_&-+()", inset: 1),
            [KeySpec(K.more, "#+=", 1.5)] + row("!?'\":;") + [KeySpec(K.del, "⌫", 1.5)],
            bottomRow(mode: "ABC"),
        ]
    }

    private func extraSymbolRows() -> [[KeySpec]] {
        [row("^~`|\\€£¥%$"), row("°={}[]<>_₫"),
         [KeySpec(K.more, "?123", 1.5)] + row("!?'\":;") + [KeySpec(K.del, "⌫", 1.5)],
         bottomRow(mode: "ABC")]
    }

    private func bottomRow(mode: String) -> [KeySpec] {
        var specs = [KeySpec(K.mode, mode, 1.4), KeySpec(K.viet, "VN", 0.9)]
        if showsGlobe { specs.insert(KeySpec(K.globe, "", 0.8), at: 0) }
        specs.append(contentsOf: [
            KeySpec(Int(",".unicodeScalars.first!.value), ",", 0.9),
            KeySpec(K.space, "Tiếng Việt", 4.5),
            KeySpec(Int(".".unicodeScalars.first!.value), ".", 0.9),
            KeySpec(K.enter, "⏎", 1.4),
        ])
        return specs
    }

    private var rows: [[KeySpec]] {
        if symbolsPage { return extraSymbols ? extraSymbolRows() : symbolsRows() }
        return (showNumberRow ? [row("1234567890")] : []) + lettersRows()
    }
    private var keys = [Key]()

    // ---- metrics ------------------------------------------------------------

    private var keyHeight: CGFloat = 52
    private let rowGap: CGFloat = 6
    private var keyGap: CGFloat = 4
    private let padH: CGFloat = 3
    private let padV: CGFloat = 8

    override var intrinsicContentSize: CGSize {
        CGSize(width: UIView.noIntrinsicMetric,
               height: padV * 2 + CGFloat(rows.count) * keyHeight + CGFloat(rows.count - 1) * rowGap)
    }

    func setMetrics(height: CGFloat, gap: CGFloat) {
        guard keyHeight != height || keyGap != gap else { return }
        keyHeight = height; keyGap = gap
        invalidateIntrinsicContentSize(); setNeedsLayout()
    }

    private let gradient = CAGradientLayer()

    override init(frame: CGRect) {
        super.init(frame: frame)
        backgroundColor = UIColor.tuiBackground
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
        switch code {
        case K.shift, K.del, K.mode, K.more, K.viet, K.enter, K.space, K.globe:
            b.backgroundColor = UIColor.tuiSurface
            b.layer.borderWidth = 1
            b.layer.borderColor = UIColor.tuiBorder.resolvedColor(with: traitCollection).cgColor
        default:
            b.backgroundColor = UIColor.tuiRaised
            b.layer.borderWidth = 1
            b.layer.borderColor = UIColor.tuiBorder.resolvedColor(with: traitCollection).cgColor
        }
        paintKey(key)
    }

    private func paintKey(_ key: Key, pressed: Bool = false) {
        let b = key.button
        let code = key.spec.code
        switch code {
        case K.shift:
            if pressed || capsLock || shifted {
                b.backgroundColor = (pressed ? UIColor.tuiPressed : UIColor.tuiAccent)
                b.setTitleColor(UIColor.tuiOnAccent, for: .normal)
            } else {
                b.backgroundColor = UIColor.tuiSurface
                b.setTitleColor(UIColor.tuiMuted, for: .normal)
            }
        case K.enter:
            b.backgroundColor = (pressed ? UIColor.tuiPressed : UIColor.tuiAccent)
            b.setTitleColor(UIColor.tuiOnAccent, for: .normal)
        case K.del, K.mode, K.more, K.viet, K.space, K.globe:
            b.backgroundColor = (pressed ? UIColor.tuiPressed : UIColor.tuiSurface)
            let color: UIColor = code == K.viet
                ? (vietnameseOn ? UIColor.tuiAccent : UIColor.tuiMuted)
                : (code == K.space ? UIColor.tuiMuted : UIColor.tuiText)
            b.setTitleColor((pressed ? UIColor.tuiOnAccent : color), for: .normal)
        default:
            b.backgroundColor = (pressed ? UIColor.tuiAccent : UIColor.tuiRaised)
            b.setTitleColor(pressed ? UIColor.tuiOnAccent : UIColor.tuiText, for: .normal)
        }
    }

    private func updateToggleLabels() {
        for key in keys {
            let b = key.button
            switch key.spec.code {
            case K.shift:
                b.setTitle(capsLock ? "⇪" : "⇧", for: .normal)
                b.titleLabel?.font = .systemFont(ofSize: 20, weight: .medium)
            case K.viet:
                b.setTitle(vietnameseOn ? "VN" : "EN", for: .normal)
                b.titleLabel?.font = .systemFont(ofSize: 14, weight: .medium)
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
                b.tintColor = UIColor.tuiText
                b.accessibilityLabel = "Chọn bàn phím"
                b.titleLabel?.font = .systemFont(ofSize: 17, weight: .regular)
            case K.space:
                b.setTitle(vietnameseOn ? "Tiếng Việt" : "English", for: .normal)
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
        gradient.colors = [UIColor.tuiBackground.resolvedColor(with: traitCollection).cgColor, UIColor.tuiGlow.resolvedColor(with: traitCollection).cgColor]
        var iter = keys.makeIterator()
        var y = padV
        for rowSpec in rows {
            let total = rowSpec.reduce(CGFloat(0)) { $0 + $1.weight }
            let unitW = (bounds.width - 2 * padH - keyGap * CGFloat(rowSpec.count - 1)) / max(total, 1)
            var x = padH
            for spec in rowSpec {
                let kw = unitW * spec.weight
                if spec.code != K.spacer, let key = iter.next() {
                    key.button.frame = CGRect(x: x, y: y, width: kw, height: keyHeight)
                }
                x += kw + keyGap
            }
            y += keyHeight + rowGap
        }
    }

    override func traitCollectionDidChange(_ previousTraitCollection: UITraitCollection?) {
        super.traitCollectionDidChange(previousTraitCollection)
        if traitCollection.hasDifferentColorAppearance(comparedTo: previousTraitCollection) {
            keys.forEach { styleKey($0) }
            hidePopup(); setNeedsLayout()
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
        let ph = keyHeight * 1.15
        let px = min(max(r.midX - pw / 2, padH), bounds.width - padH - pw)
        let py = max(r.minY - ph - rowGap, 2)
        let v = UIView(frame: CGRect(x: px, y: py, width: pw, height: ph))
        v.backgroundColor = UIColor.tuiRaised
        v.layer.cornerRadius = 8
        v.layer.borderWidth = 1
        v.layer.borderColor = UIColor.tuiBorder.cgColor
        let l = UILabel(frame: v.bounds)
        l.text = currentLabel(key.spec)
        l.textAlignment = .center
        l.textColor = UIColor.tuiText
        l.font = .systemFont(ofSize: 26, weight: .medium)
        v.addSubview(l)
        addSubview(v)
        popup = v
    }

    private func hidePopup() {
        popup?.removeFromSuperview()
        popup = nil
    }

    // ---- delete repeat ---------------------------------------------------------

    private var pressed: Key?
    private var shiftTapTime = Date.distantPast
    private var deleteTimer: Timer?

    private func startDeleteRepeat() {
        stopDeleteRepeat()
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

    // ---- touch callbacks --------------------------------------------------------

    @objc private func onDown(_ b: UIButton) {
        guard let key = keys.first(where: { $0.button === b }) else { return }
        if key.spec.code == K.globe { return }
        pressed = key
        UIImpactFeedbackGenerator(style: .light).impactOccurred()
        paintKey(key, pressed: true)
        if key.spec.code == K.del {
            startDeleteRepeat()
        } else {
            showPopup(for: key)
        }
    }

    @objc private func onGlobe(_ sender: UIButton, event: UIEvent) {
        listener?.onInputModeMenu(sender, event: event)
    }

    @objc private func onUp(_ b: UIButton) {
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
        if key.spec.code == K.del { startDeleteRepeat() }
    }

    private func clearPress() {
        if let p = pressed { paintKey(p) }
        pressed = nil
        stopDeleteRepeat()
        hidePopup()
    }

    // ---- dispatch -----------------------------------------------------------------

    private func dispatch(_ key: Key) {
        switch key.spec.code {
        case K.shift:
            let now = Date()
            if capsLock {
                capsLock = false; shifted = false
            } else if shifted, now.timeIntervalSince(shiftTapTime) < 0.35 {
                capsLock = true
            } else {
                shifted.toggle()
            }
            shiftTapTime = (shifted || capsLock) ? now : .distantPast
            listener?.onShiftChanged(shifted, capsLock)
        case K.del:
            break // handled on touch-down / repeat
        case K.mode:
            symbolsPage.toggle()
            extraSymbols = false
            rebuildKeys()
        case K.more:
            extraSymbols.toggle()
            rebuildKeys()
        case K.enter:
            listener?.onEnter()
        case K.space:
            listener?.onChar(" ")
        case K.viet:
            listener?.onVietToggle()
        case K.globe:
            listener?.onSwitchIme()
        default:
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
        guard !capsLock, shifted != on else { return }
        shifted = on
        updateToggleLabels()
        listener?.onShiftChanged(shifted, capsLock)
    }

}
