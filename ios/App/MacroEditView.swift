import SwiftUI

// Edits macro.txt in the App Group container — same format as desktop
// (one "key:text" per line, ';' for comments).
struct MacroEditView: View {
    @State private var text = ""
    @State private var saved = false
    @Environment(\.presentationMode) private var presentationMode

    var body: some View {
        VStack(spacing: 0) {
            TextEditor(text: $text)
                .font(.system(.body, design: .monospaced))
                .autocapitalization(.none)
                .disableAutocorrection(true)
                .padding(8)
            HStack {
                Text("Mỗi dòng: key:text — vd `ko:không`")
                    .font(.footnote)
                    .foregroundColor(.secondary)
                Spacer()
                Button("Lưu") { save() }
                    .disabled(!dirty())
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 8)
        }
        .navigationTitle("Gõ tắt")
        .navigationBarTitleDisplayMode(.inline)
        .onAppear(perform: load)
        .alert("Đã lưu", isPresented: $saved) {
            Button("OK") { presentationMode.wrappedValue.dismiss() }
        }
    }

    private func load() {
        guard let url = Prefs.macroURL,
              let s = try? String(contentsOf: url, encoding: .utf8) else { return }
        text = s
    }

    private func save() {
        guard let url = Prefs.macroURL else { return }
        try? text.write(to: url, atomically: true, encoding: .utf8)
        saved = true
    }

    private func dirty() -> Bool {
        guard let url = Prefs.macroURL else { return !text.isEmpty }
        let old = (try? String(contentsOf: url, encoding: .utf8)) ?? ""
        return text != old
    }
}
