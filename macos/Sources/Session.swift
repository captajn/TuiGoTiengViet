import Foundation
import TuiGoEngine

// Session adapts the engine's edit-stream contract (Result{backs, out}) to
// marked-text IME APIs that cannot delete already-committed text. The
// current word lives in `model`, is shown via setMarkedText, and commits
// on a word boundary. Chars the engine tracks but does not consume are
// swallowed into the preedit — see linux/internal/session for the full
// rationale.
final class Session {
    struct Action {
        var commit: [UInt16] = []
        var preedit: [UInt16] = []
        var forward = false
    }

    let eng = Engine()
    private(set) var model: [UInt16] = []
    private(set) var viet: Bool
    var autoCap = false
    private var capNext = false

    init(viet: Bool) {
        self.viet = viet
        eng.setVietKey(viet)
    }

    @discardableResult
    func setViet(_ v: Bool) -> Action {
        if v == viet { return Action() }
        viet = v
        eng.setVietKey(v)
        eng.resetBuf()
        return flush()
    }

    func feed(_ ch: UInt32) -> Action {
        if !viet { return forward() }

        var c = ch
        if autoCap && capNext && c >= 97 && c <= 122 { // 'a'..'z'
            c -= 32
        }
        let res = eng.filter(Int(c))

        if autoCap {
            switch c {
            case 46, 33, 63: capNext = true       // . ! ?
            case 32: break                        // space keeps pending
            default: capNext = false
            }
        }

        if res.consumed {
            apply(backs: res.backs, out: res.out)
            return emit()
        }
        if eng.atWordBeginning() {
            return forward()
        }
        model.append(UInt16(c))
        return Action(preedit: model)
    }

    func backspace() -> Action {
        if !viet { return forward() }
        let res = eng.backspace()
        if !res.consumed && model.isEmpty {
            return forward()
        }
        apply(backs: res.backs, out: res.out)
        return emit()
    }

    func reset() -> Action {
        capNext = false
        eng.resetBuf()
        return flush()
    }

    func enterPressed() -> Action {
        if autoCap { capNext = true }
        return reset()
    }

    private func apply(backs: Int, out: [UInt16]) {
        let cut = min(backs, model.count)
        model.removeLast(cut)
        model.append(contentsOf: out)
    }

    private func emit() -> Action {
        if eng.atWordBeginning() { return flush() }
        return Action(preedit: model)
    }

    private func flush() -> Action {
        let a = Action(commit: model)
        model = []
        return a
    }

    private func forward() -> Action {
        var a = flush()
        a.forward = true
        return a
    }
}
