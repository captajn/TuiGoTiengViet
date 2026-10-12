import SwiftUI
import TuiGoEngine

struct MacroEditView: View {
    @Environment(\.dismiss) private var dismiss
    @State private var text = ""
    @State private var original = ""
    @State private var loaded = false
    @State private var loadFailed = false
    @State private var message = ""
    @State private var showMessage = false
    @State private var confirmDiscard = false
    private var validation: String? { macroValidationError(text) }
    var body: some View {
        ZStack {
            GlassBackground()
            ScrollView {
                VStack(spacing: 20) {
                    BrandHeader(title: "Gõ tắt", subtitle: "Một từ ngắn, cả câu dài.")
                    GlassCard("Cách dùng", subtitle: "Bật gõ tắt và dùng chế độ Tiếng Việt") {
                        Text("Mỗi dòng là một mục, ngăn bằng dấu hai chấm. Gõ từ tắt rồi chạm Space để thay thế.")
                        Text("ko:không\nvn:Việt Nam\nmail:hello@example.com").font(.system(.body, design: .monospaced)).foregroundColor(GlassTheme.accent)
                        Text("Sửa nội dung để đổi một mục, xóa dòng để xóa. Từ tắt tối đa 15 ký tự; phân biệt chữ hoa/thường.").font(.footnote).foregroundColor(GlassTheme.muted)
                    }
                    GlassCard("Từ của bạn", subtitle: "Mỗi dòng: từ tắt:nội dung thay thế") {
                        TextEditor(text: $text)
                            .modifier(PlainEditorBackground())
                            .font(.system(.body, design: .monospaced))
                            .autocapitalization(.none).disableAutocorrection(true)
                            .frame(minHeight: 240).padding(8)
                            .background(Color(uiColor: .tuiRaised), in: RoundedRectangle(cornerRadius: 14))
                            .accessibilityLabel("Danh sách gõ tắt")
                        if let error = validation { Text(error).font(.footnote).foregroundColor(.red) }
                        if loadFailed { Text(message).font(.footnote).foregroundColor(.red) }
                        Button(action: save) {
                            Text("Lưu gõ tắt").fontWeight(.semibold).frame(maxWidth: .infinity).padding(.vertical, 8)
                        }.buttonStyle(.borderedProminent).foregroundColor(Color(uiColor: .tuiOnAccent)).disabled(loadFailed || validation != nil || text == original)
                        Text("Dữ liệu chỉ được lưu trên thiết bị. Mở lại bàn phím để dùng danh sách mới.").font(.footnote).foregroundColor(GlassTheme.muted)
                    }
                }.padding(20).frame(maxWidth: 640).frame(maxWidth: .infinity)
            }
        }.foregroundColor(GlassTheme.text).navigationTitle("Gõ tắt").navigationBarTitleDisplayMode(.inline).navigationBarHidden(false)
            .navigationBarBackButtonHidden(true)
            .toolbar {
                ToolbarItem(placement: .navigationBarLeading) {
                    Button {
                        if text == original { dismiss() } else { confirmDiscard = true }
                    } label: { Label("Cài đặt", systemImage: "chevron.left") }
                }
            }
            .confirmationDialog("Chưa lưu thay đổi", isPresented: $confirmDiscard, titleVisibility: .visible) {
                Button("Bỏ thay đổi", role: .destructive) { dismiss() }
                Button("Sửa tiếp", role: .cancel) {}
            }
            .onAppear(perform: load)
            .alert(message, isPresented: $showMessage) { Button("OK", role: .cancel) {} }
    }
    private func load() {
        guard !loaded else { return }; loaded = true
        do {
            guard let url = Prefs.macroURL else { throw MacroError.container }
            original = FileManager.default.fileExists(atPath: url.path) ? try String(contentsOf: url, encoding: .utf8) : ""
            text = original
        } catch {
            loadFailed = true; message = "Không đọc được gõ tắt. Kiểm tra cấu hình App Group và mở lại màn hình; dữ liệu cũ chưa bị ghi đè."
        }
    }
    private func save() {
        guard validation == nil, !loadFailed else { return }
        do {
            guard let url = Prefs.macroURL else { throw MacroError.container }
            try text.write(to: url, atomically: true, encoding: .utf8)
            original = text; message = "Đã lưu gõ tắt"
        } catch { message = "Không lưu được gõ tắt. Dữ liệu cũ được giữ lại; hãy thử lại." }
        showMessage = true
    }
    private enum MacroError: Error { case container }
}

private struct PlainEditorBackground: ViewModifier {
    @ViewBuilder func body(content: Content) -> some View {
        if #available(iOS 16.0, *) { content.scrollContentBackground(.hidden) }
        else { content }
    }
}
