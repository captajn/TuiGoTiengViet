import SwiftUI

struct SettingsView: View {
    @State private var cfg = TuiGoConfig.load()
    @State private var macroText = ""
    @State private var status = ""

    private let inputMethods: [(String, Int)] = [
        ("Telex", 0), ("VNI", 1), ("VIQR", 2),
        ("MS Vi (AllCode)", 3), ("Telex đơn giản", 5), ("Telex + VNI", 6),
    ]

    var body: some View {
        TabView {
            generalTab.tabItem { Text("Cài đặt") }
            macroTab.tabItem { Text("Gõ tắt") }
        }
        .padding()
    }

    private var generalTab: some View {
        Form {
            Toggle("Gõ tiếng Việt", isOn: $cfg.vietKey)

            Picker("Kiểu gõ", selection: $cfg.inputMethod) {
                ForEach(inputMethods, id: \.1) { Text($0.0).tag($0.1) }
            }

            Section("Tùy chọn") {
                Toggle("Kiểm tra chính tả khi gõ", isOn: $cfg.spellCheck)
                Toggle("Kiểu mới (oà, uý thay vì òa, uý)", isOn: $cfg.modernStyle)
                Toggle("Cho phép đặt dấu tự do", isOn: $cfg.freeMarking)
                Toggle("Bật gõ tắt (macro)", isOn: $cfg.macroEnabled)
                Toggle("Tự khôi phục từ không đúng tiếng Việt",
                       isOn: $cfg.autoNonVnRestore)
                Toggle("Tự viết hoa sau dấu chấm câu", isOn: $cfg.autoCap)
            }

            Section("Phím tắt chuyển Việt/Anh") {
                Toggle("Ctrl + Shift", isOn: toggleBinding("ctrl+shift"))
                Toggle("Alt + Z (⌥Z)", isOn: toggleBinding("alt+z"))
            }

            Toggle("Khởi động cùng hệ thống", isOn: $cfg.launchAtLogin)

            HStack {
                Button("Lưu") { save() }
                Text(status).foregroundStyle(.secondary)
            }
        }
    }

    private var macroTab: some View {
        VStack(alignment: .leading) {
            Text("Mỗi dòng: từ-tắt:kết-quả — gõ từ tắt rồi Space/Enter để bung.\n"
                 + "Ví dụ: ko:không — gõ \"ko\" + Space → \"không\".")
                .font(.callout).foregroundStyle(.secondary)
            TextEditor(text: $macroText)
                .font(.system(.body, design: .monospaced))
                .border(.quaternary)
        }
        .onAppear {
            macroText = (try? String(contentsOf: TuiGoConfig.macroURL,
                                     encoding: .utf8)) ?? ""
        }
    }

    private func toggleBinding(_ key: String) -> Binding<Bool> {
        Binding(
            get: { cfg.toggles.contains(key) },
            set: { on in
                if on && !cfg.toggles.contains(key) { cfg.toggles.append(key) }
                if !on { cfg.toggles.removeAll { $0 == key } }
            })
    }

    private func save() {
        if let bad = badMacroLine(macroText) {
            status = "Lỗi gõ tắt: \(bad)"
            return
        }
        do {
            try FileManager.default.createDirectory(
                at: TuiGoConfig.configDir, withIntermediateDirectories: true)
            try macroText.write(
                to: TuiGoConfig.macroURL, atomically: true, encoding: .utf8)
            try cfg.setLaunchAtLogin(cfg.launchAtLogin)
            try cfg.save()
            status = "Đã lưu — có hiệu lực khi bàn phím kích hoạt lại"
        } catch {
            status = "Lỗi ghi: \(error.localizedDescription)"
        }
    }
}

// First macro line without a ':' separator is invalid.
private func badMacroLine(_ text: String) -> String? {
    for raw in text.split(separator: "\n", omittingEmptySubsequences: false) {
        let line = raw.trimmingCharacters(in: .whitespaces)
        if line.isEmpty || line.hasPrefix("#") || line.hasPrefix(";") {
            continue
        }
        if !line.contains(":") { return line }
    }
    return nil
}
