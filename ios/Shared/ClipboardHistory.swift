import Foundation
import UIKit

/// Samsung-style on-device clipboard history — mirror of
/// android/app/.../ime/ClipboardHistory.kt. iOS gives keyboard extensions no
/// clipboard-change callback, so the keyboard archives the pasteboard when it
/// observes UIPasteboard.changeCount move while it is visible. Copies made in
/// secure (password) fields are never stored.
///
/// Storage: clipboard_history.json (newest first) + PNG files under clips/ in
/// the shared App Group container. Both the pasteboard and that container need
/// Full Access, so every failure is a silent no-op. Pinned entries are kept
/// above the cap; at most Prefs.clipboardHistoryMax() unpinned entries.
enum ClipboardHistory {
    struct Entry {
        let id: Int64
        let text: String?
        let image: String?
        let pinned: Bool
        let ts: Int64
    }

    private static let maxPinned = 20
    private static let maxImageEdge: CGFloat = 768
    private static let maxImageBytes = 1_500_000
    private static let lock = NSLock()
    private static var cache: [Entry]?

    private static var fileURL: URL? {
        Prefs.sharedContainer?.appendingPathComponent("clipboard_history.json")
    }
    private static var imageDir: URL? {
        Prefs.sharedContainer?.appendingPathComponent("clips")
    }

    static func entries() -> [Entry] {
        lock.lock(); defer { lock.unlock() }
        var list = cache ?? loadLocked()
        // The user may have lowered the cap since the last insert — trim now
        // so the panel never shows more than the configured limit.
        if enforceCapsLocked(&list) { saveLocked(list) }
        cache = list
        return list
    }

    static var hasEntries: Bool { !entries().isEmpty }

    /// Archive the current pasteboard contents. Call only after a changeCount
    /// change; image decoding runs on a background queue like the Android port.
    static func record(_ pb: UIPasteboard, secureField: Bool) {
        guard Prefs.clipboardHistory(), !secureField else { return }
        let id = Int64(Date().timeIntervalSince1970 * 1000)
        if let text = pb.string?.trimmingCharacters(in: .whitespacesAndNewlines), !text.isEmpty {
            insert(Entry(id: id, text: text, image: nil, pinned: false, ts: id))
        } else if let img = pb.image {
            DispatchQueue.global(qos: .utility).async {
                if let name = saveImage(img, id: id) {
                    insert(Entry(id: id, text: nil, image: name, pinned: false, ts: id))
                }
            }
        }
    }

    static func setPinned(_ id: Int64, _ pinned: Bool) {
        lock.lock(); defer { lock.unlock() }
        var list = cache ?? loadLocked()
        guard let idx = list.firstIndex(where: { $0.id == id }) else { return }
        let e = list[idx]
        list[idx] = Entry(id: e.id, text: e.text, image: e.image, pinned: pinned, ts: e.ts)
        cache = list
        saveLocked(list)
    }

    static func remove(_ id: Int64) {
        lock.lock(); defer { lock.unlock() }
        var list = cache ?? loadLocked()
        guard let idx = list.firstIndex(where: { $0.id == id }) else { return }
        if let img = list[idx].image { deleteImage(img) }
        list.remove(at: idx)
        cache = list
        saveLocked(list)
    }

    static func clear() {
        lock.lock(); defer { lock.unlock() }
        if let dir = imageDir,
           let files = try? FileManager.default.contentsOfDirectory(atPath: dir.path) {
            for f in files { try? FileManager.default.removeItem(at: dir.appendingPathComponent(f)) }
        }
        cache = []
        saveLocked([])
    }

    static func image(named name: String) -> UIImage? {
        guard let dir = imageDir else { return nil }
        return UIImage(contentsOfFile: dir.appendingPathComponent(name).path)
    }

    // ---- internals -----------------------------------------------------------

    private static func loadLocked() -> [Entry] {
        var list: [Entry] = []
        if let url = fileURL, let data = try? Data(contentsOf: url),
           let arr = try? JSONSerialization.jsonObject(with: data) as? [[String: Any]] {
            for o in arr {
                guard let id = (o["id"] as? NSNumber)?.int64Value, id > 0 else { continue }
                list.append(Entry(
                    id: id,
                    text: o["text"] as? String,
                    image: o["image"] as? String,
                    pinned: (o["pinned"] as? Bool) ?? false,
                    ts: (o["ts"] as? NSNumber)?.int64Value ?? id))
            }
        }
        // Newest first; pinned entries float to the top of the panel.
        list.sort { ($0.pinned ? 1 : 0, $0.ts) > ($1.pinned ? 1 : 0, $1.ts) }
        cache = list
        return list
    }

    /// Insert newest-first; a re-copied text bumps to the top instead of
    /// duplicating. Oldest unpinned entries are evicted beyond the cap.
    private static func insert(_ e: Entry) {
        lock.lock(); defer { lock.unlock() }
        var list = cache ?? loadLocked()
        if let t = e.text { list.removeAll { $0.text == t } }
        list.append(e)
        list.sort { ($0.pinned ? 1 : 0, $0.ts) > ($1.pinned ? 1 : 0, $1.ts) }
        enforceCapsLocked(&list)
        cache = list
        saveLocked(list)
    }

    /// Evict oldest unpinned beyond the configured cap, then oldest pinned
    /// beyond cap + maxPinned. Returns true when entries were removed.
    @discardableResult
    private static func enforceCapsLocked(_ list: inout [Entry]) -> Bool {
        let cap = Prefs.clipboardHistoryMax()
        var changed = false
        while list.filter({ !$0.pinned }).count > cap {
            guard evictLast(&list, where: { !$0.pinned }) else { break }
            changed = true
        }
        while list.count > cap + maxPinned {
            guard evictLast(&list, where: { $0.pinned }) else { break }
            changed = true
        }
        return changed
    }

    private static func evictLast(_ list: inout [Entry], where pred: (Entry) -> Bool) -> Bool {
        guard let idx = list.lastIndex(where: pred) else { return false }
        if let img = list[idx].image { deleteImage(img) }
        list.remove(at: idx)
        return true
    }

    private static func saveLocked(_ list: [Entry]) {
        guard let url = fileURL else { return }
        let arr: [[String: Any]] = list.map { e in
            [
                "id": e.id,
                "text": e.text ?? NSNull(),
                "image": e.image ?? NSNull(),
                "pinned": e.pinned,
                "ts": e.ts,
            ]
        }
        guard let data = try? JSONSerialization.data(withJSONObject: arr) else { return }
        // Atomic replace: write temp then swap so a crash mid-write keeps the
        // previous file intact (same as the Android temp+rename).
        let tmp = url.appendingPathExtension("tmp")
        do {
            try data.write(to: tmp)
            try? FileManager.default.removeItem(at: url)
            try FileManager.default.moveItem(at: tmp, to: url)
        } catch { }
    }

    private static func deleteImage(_ name: String) {
        guard let dir = imageDir else { return }
        try? FileManager.default.removeItem(at: dir.appendingPathComponent(name))
    }

    private static func saveImage(_ img: UIImage, id: Int64) -> String? {
        guard let dir = imageDir else { return nil }
        let scale = min(1, maxImageEdge / max(img.size.width, img.size.height))
        let size = CGSize(width: img.size.width * scale, height: img.size.height * scale)
        let rendered = UIGraphicsImageRenderer(size: size).image { _ in
            img.draw(in: CGRect(origin: .zero, size: size))
        }
        guard let data = rendered.pngData(), data.count <= maxImageBytes else { return nil }
        do {
            try FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)
            try data.write(to: dir.appendingPathComponent("c_\(id).png"), options: .atomic)
            return "c_\(id).png"
        } catch { return nil }
    }
}
