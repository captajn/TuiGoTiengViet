import SwiftUI
import TuiGoEngine

// Host app = settings screen (mirrors android/MainActivity.kt).
struct SettingsView: View {
    @AppStorage(Prefs.keyIM, store: Prefs.defaults) private var inputMethod = ImTelex
    @AppStorage(Prefs.keyFreeMarking, store: Prefs.defaults) private var freeMarking = true
    @AppStorage(Prefs.keyModernStyle, store: Prefs.defaults) private var modernStyle = true
    @AppStorage(Prefs.keySpellCheck, store: Prefs.defaults) private var spellCheck = true
    @AppStorage(Prefs.keyMacro, store: Prefs.defaults) private var macro = true
    @AppStorage(Prefs.keyAutoNonVn, store: Prefs.defaults) private var autoNonVn = true
    @AppStorage(Prefs.keyDoubleSpace, store: Prefs.defaults) private var doubleSpace = true

    var body: some View {
        NavigationView {
            Form {
                Section {
                    Text("1. Mở Cài đặt → Tổng quát → Bàn phím → Bàn phím → Thêm bàn phím mới → **Tui Gõ**.")
                    Text("2. Khi gõ, chạm nút 🌐 để chuyển sang Tui Gõ.")
                    Button("Mở Cài đặt") {
                        if let url = URL(string: UIApplication.openSettingsURLString) {
                            UIApplication.shared.open(url)
                        }
                    }
                } header: {
                    Text("Bật bàn phím")
                }

                Section {
                    Picker("Kiểu gõ", selection: $inputMethod) {
                        Text("Telex").tag(ImTelex)
                        Text("VNI").tag(ImVni)
                        Text("Telex đơn giản").tag(ImTelexSimple)
                        Text("Telex + VNI").tag(ImTelexVni)
                    }
                } header: {
                    Text("Kiểu gõ")
                }

                Section {
                    Toggle("Bỏ dấu tự do (free marking)", isOn: $freeMarking)
                    Toggle("Kiểu dấu mới (oà, uỳ)", isOn: $modernStyle)
                    Toggle("Kiểm tra chính tả", isOn: $spellCheck)
                    Toggle("Gõ tắt (macro)", isOn: $macro)
                    Toggle("Khôi phục từ không phải tiếng Việt", isOn: $autoNonVn)
                    Toggle("Gõ đúp Space → dấu chấm", isOn: $doubleSpace)
                } header: {
                    Text("Tuỳ chọn")
                }

                Section {
                    NavigationLink("Sửa gõ tắt…") {
                        MacroEditView()
                    }
                } header: {
                    Text("Gõ tắt")
                }

                Section {
                    Text("Nút VN trên bàn phím bật/tắt chế độ tiếng Việt nhanh.")
                        .font(.footnote)
                        .foregroundColor(.secondary)
                }
            }
            .navigationTitle("Tui Gõ")
        }
    }
}
