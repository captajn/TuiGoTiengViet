import SwiftUI
import TuiGoEngine

struct SettingsView: View {
    @AppStorage(Prefs.keyIM, store: Prefs.defaults) private var inputMethod = ImTelex
    @AppStorage(Prefs.keyFreeMarking, store: Prefs.defaults) private var freeMarking = true
    @AppStorage(Prefs.keyModernStyle, store: Prefs.defaults) private var modernStyle = true
    @AppStorage(Prefs.keySpellCheck, store: Prefs.defaults) private var spellCheck = true
    @AppStorage(Prefs.keyMacro, store: Prefs.defaults) private var macro = true
    @AppStorage(Prefs.keyAutoNonVn, store: Prefs.defaults) private var autoNonVn = true
    @AppStorage(Prefs.keyDoubleSpace, store: Prefs.defaults) private var doubleSpace = true
    @AppStorage(Prefs.keyNumberRow, store: Prefs.defaults) private var numberRow = false
    @AppStorage(Prefs.keyKeyHeight, store: Prefs.defaults) private var keyHeight = 52.0
    @AppStorage(Prefs.keyKeyGap, store: Prefs.defaults) private var keyGap = 4.0
    @State private var preview = ""
    private let methods = [(ImTelex, "Telex", "s f r x j · aa, ee, oo, w, dd"), (ImVni, "VNI", "Dùng số 1–9 để thêm dấu"), (ImViqr, "VIQR", "Dùng ký hiệu ^, +, ', `, ?, ~, . để thêm dấu")]

    var body: some View {
        NavigationView {
            ZStack {
                GlassBackground()
                ScrollView {
                    VStack(alignment: .leading, spacing: 20) {
                        VStack(alignment: .leading, spacing: 6) {
                            Text("GÕ THEO CÁCH CỦA BẠN").font(.caption2.weight(.bold)).tracking(1.5).foregroundColor(GlassTheme.accent)
                            BrandHeader(title: "Tui Gõ", subtitle: "Tiếng Việt, thật tự nhiên.")
                        }.padding(.vertical, 8)
                        GlassCard("Bắt đầu", subtitle: "Hai bước để dùng bàn phím của bạn") {
                            Label("Thêm Tui Gõ trong Cài đặt → Tổng quát → Bàn phím → Thêm bàn phím mới.", systemImage: "1.circle")
                                .font(.subheadline)
                            Button { openSettings() } label: {
                                Text("Mở Cài đặt").fontWeight(.semibold).frame(maxWidth: .infinity).padding(.vertical, 8)
                            }.buttonStyle(.borderedProminent).foregroundColor(Color(uiColor: .tuiOnAccent))
                            Label("Khi gõ, giữ nút quả địa cầu để chọn Tui Gõ.", systemImage: "2.circle").font(.subheadline)
                        }
                        GlassCard("Kiểu gõ", subtitle: "Chọn một kiểu phù hợp với thói quen của bạn") {
                            ForEach(methods, id: \.0) { item in
                                Button { inputMethod = item.0 } label: {
                                    HStack(spacing: 12) {
                                        Image(systemName: inputMethod == item.0 ? "checkmark.circle.fill" : "circle")
                                            .font(.title3).foregroundColor(inputMethod == item.0 ? GlassTheme.accent : GlassTheme.muted)
                                        VStack(alignment: .leading, spacing: 4) {
                                            Text(item.1).fontWeight(.medium).foregroundColor(GlassTheme.text)
                                            Text(item.2).font(.caption).foregroundColor(GlassTheme.muted)
                                        }
                                        Spacer(minLength: 0)
                                    }.frame(maxWidth: .infinity, minHeight: 48, alignment: .leading)
                                        .padding(10)
                                        .background(inputMethod == item.0 ? GlassTheme.glow : Color.clear, in: RoundedRectangle(cornerRadius: 14))
                                        .overlay(RoundedRectangle(cornerRadius: 14).stroke(inputMethod == item.0 ? GlassTheme.accent : Color.clear, lineWidth: 1))
                                        .contentShape(Rectangle())
                                }.buttonStyle(.plain)
                                    .accessibilityAddTraits(inputMethod == item.0 ? .isSelected : [])
                            }
                        }
                        GlassCard("Cách gõ", subtitle: "Tinh chỉnh cách Tui Gõ xử lý từ và dấu") {
                            Toggle("Bỏ dấu tự do", isOn: $freeMarking)
                            Divider()
                            Toggle("Đặt dấu kiểu mới (hoà, thuỷ)", isOn: $modernStyle)
                            Divider()
                            Toggle("Kiểm tra chính tả", isOn: $spellCheck)
                            Divider()
                            Toggle("Khôi phục từ ngoại ngữ", isOn: $autoNonVn)
                            Divider()
                            Toggle("Hai lần Space → dấu chấm", isOn: $doubleSpace)
                        }
                        GlassCard("Gõ tắt", subtitle: "Một từ ngắn, cả câu dài") {
                            Toggle("Bật gõ tắt", isOn: $macro)
                            Text("Gõ ko rồi Space → không. Thêm từ của bạn trong danh sách gõ tắt.").font(.footnote).foregroundColor(GlassTheme.muted)
                            NavigationLink(destination: MacroEditView()) {
                                HStack { Text("Quản lý từ gõ tắt"); Spacer(); Image(systemName: "chevron.right") }.padding(.vertical, 8)
                            }
                        }
                        GlassCard("Bàn phím", subtitle: "Không gian gõ vừa tay bạn") {
                            Toggle("Hiện hàng số", isOn: $numberRow)
                            Text("VNI luôn có hàng số để gõ dấu.").font(.caption).foregroundColor(GlassTheme.muted)
                            Text("Chiều cao phím · \(Int(keyHeight)) pt").font(.subheadline)
                            Slider(value: $keyHeight, in: 44...68, step: 1).accessibilityLabel("Chiều cao phím")
                            Text("Khoảng cách phím · \(Int(keyGap)) pt").font(.subheadline)
                            Slider(value: $keyGap, in: 0...8, step: 1).accessibilityLabel("Khoảng cách phím")
                            Text("Sáng / tối theo thiết bị. Chạm VN/EN để đổi ngôn ngữ; Space luôn hiện chế độ đang dùng.").font(.footnote).foregroundColor(GlassTheme.muted)
                        }
                        GlassCard("Thử bàn phím", subtitle: "Gõ thử tiếng Việt hoặc từ gõ tắt") {
                            TextField("Ví dụ: tieengs Vieetj", text: $preview).autocapitalization(.none).textFieldStyle(.roundedBorder)
                        }
                        Text("Chỉ trên thiết bị của bạn. Không thu thập nội dung gõ.").font(.caption).foregroundColor(GlassTheme.muted)
                    }.padding(20).frame(maxWidth: 640).frame(maxWidth: .infinity)
                }
            }.navigationBarHidden(true)
        }.navigationViewStyle(.stack).tint(GlassTheme.accent).foregroundColor(GlassTheme.text)
            .onAppear { Prefs.migrateInputMethod() }
    }
    private func openSettings() {
        if let url = URL(string: UIApplication.openSettingsURLString) { UIApplication.shared.open(url) }
    }
}
