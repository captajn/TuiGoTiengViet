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
    @AppStorage(Prefs.keyVietOn, store: Prefs.defaults) private var vietOnDefault = true
    @AppStorage(Prefs.keyNumberRow, store: Prefs.defaults) private var numberRow = false
    @AppStorage(Prefs.keyKeyHeight, store: Prefs.defaults) private var keyHeight = 52.0
    @AppStorage(Prefs.keyKeyGap, store: Prefs.defaults) private var keyGap = 4.0
    @AppStorage(Prefs.keyTheme, store: Prefs.defaults) private var keyTheme = KeyboardThemes.defaultId
    @State private var preview = ""
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.horizontalSizeClass) private var sizeClass
    private let methods = [(ImTelex, "Telex", "Telex · s, f, r, x, j để thêm dấu; aa, ee, oo, w, dd"), (ImVni, "VNI", "VNI · Dùng hàng phím số 1–9 để thêm dấu"), (ImViqr, "VIQR", "VIQR · Dùng ký hiệu ^, +, ', `, ?, ~, . để thêm dấu")]

    var body: some View {
        let outerHPad: CGFloat = sizeClass == .regular ? 24 : 16
        NavigationView {
            ZStack {
                GlassBackground()
                ScrollView {
                    VStack(alignment: .leading, spacing: 18) {
                        VStack(alignment: .leading, spacing: 6) {
                            Text("GÕ THEO CÁCH CỦA BẠN").font(.caption2.weight(.bold)).tracking(1.5).foregroundColor(GlassTheme.accent)
                            BrandHeader(title: "Tui Gõ", subtitle: "Tiếng Việt, thật tự nhiên.")
                        }.padding(.vertical, 6)
                        GlassCard("Bắt đầu", subtitle: "Hai bước để dùng bàn phím của bạn") {
                            Label("Thêm Tui Gõ trong Cài đặt → Tổng quát → Bàn phím → Thêm bàn phím mới.", systemImage: "1.circle")
                                .font(.subheadline)
                                .fixedSize(horizontal: false, vertical: true)
                            Button { openSettings() } label: {
                                Text("Mở Cài đặt").fontWeight(.semibold).frame(maxWidth: .infinity).padding(.vertical, 8)
                            }.buttonStyle(.borderedProminent).foregroundColor(Color(uiColor: .tuiOnAccent))
                            Label("Khi gõ, giữ nút quả địa cầu để chọn Tui Gõ.", systemImage: "2.circle")
                                .font(.subheadline)
                                .fixedSize(horizontal: false, vertical: true)
                        }
                        GlassCard("Kiểu gõ", subtitle: "Chọn một kiểu phù hợp với thói quen của bạn") {
                            Picker("Kiểu gõ", selection: $inputMethod) {
                                ForEach(methods, id: \.0) { item in
                                    Text(item.1).tag(item.0)
                                }
                            }.pickerStyle(.segmented)
                                .frame(maxWidth: .infinity)
                            Text(methods.first { $0.0 == inputMethod }?.2 ?? "")
                                .font(.caption).foregroundColor(GlassTheme.muted).padding(.top, 4)
                                .fixedSize(horizontal: false, vertical: true)
                        }
                        GlassCard("Cách gõ", subtitle: "Tinh chỉnh cách Tui Gõ xử lý từ và dấu") {
                            Toggle("Mặc định gõ tiếng Việt", isOn: $vietOnDefault)
                            Text("App chưa dùng Tui Gõ sẽ mở ở chế độ này.").font(.caption).foregroundColor(GlassTheme.muted)
                            Divider()
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
                                .fixedSize(horizontal: false, vertical: true)
                            NavigationLink(destination: MacroEditView()) {
                                HStack { Text("Quản lý từ gõ tắt"); Spacer(); Image(systemName: "chevron.right") }
                                    .frame(maxWidth: .infinity)
                                    .padding(.vertical, 8)
                            }
                        }
                        GlassCard("Bàn phím", subtitle: "Không gian gõ vừa tay bạn") {
                            Text("Giao diện").font(.subheadline.weight(.semibold))
                            ScrollView(.horizontal, showsIndicators: false) {
                                HStack(spacing: 10) {
                                    ForEach(KeyboardThemes.ids, id: \.self) { id in
                                        let t = KeyboardThemes.get(id, dark: colorScheme == .dark)
                                        let selected = keyTheme == id
                                        Button { keyTheme = id } label: {
                                            VStack(spacing: 6) {
                                                swatch(t)
                                                Text(KeyboardThemes.label(id))
                                                    .font(.caption.weight(selected ? .semibold : .regular))
                                                    .foregroundColor(GlassTheme.text)
                                                    .lineLimit(1)
                                                    .fixedSize(horizontal: true, vertical: false)
                                            }
                                            .frame(minWidth: 92)
                                            .padding(.horizontal, 12).padding(.vertical, 10)
                                            .background(selected ? GlassTheme.glow : Color(uiColor: .tuiRaised), in: RoundedRectangle(cornerRadius: 16, style: .continuous))
                                            .overlay(RoundedRectangle(cornerRadius: 16, style: .continuous).stroke(selected ? GlassTheme.accent : Color(uiColor: .tuiBorder), lineWidth: selected ? 1.5 : 1))
                                            .contentShape(Rectangle())
                                        }.buttonStyle(.plain)
                                            .accessibilityAddTraits(selected ? .isSelected : [])
                                    }
                                }.padding(.vertical, 2)
                            }.frame(maxWidth: .infinity)
                            Toggle("Hiện hàng số", isOn: $numberRow)
                            Text("VNI luôn có hàng số để gõ dấu.").font(.caption).foregroundColor(GlassTheme.muted)
                            VStack(alignment: .leading, spacing: 6) {
                                Text("Chiều cao phím · \(Int(keyHeight)) pt").font(.subheadline)
                                Slider(value: $keyHeight, in: 44...68, step: 1)
                                    .frame(maxWidth: .infinity)
                                    .accessibilityLabel("Chiều cao phím")
                            }
                            VStack(alignment: .leading, spacing: 6) {
                                Text("Khoảng cách phím · \(Int(keyGap)) pt").font(.subheadline)
                                Slider(value: $keyGap, in: 0...8, step: 1)
                                    .frame(maxWidth: .infinity)
                                    .accessibilityLabel("Khoảng cách phím")
                            }
                            Text("Vuốt phím Space để di chuyển con trỏ · Giữ Space để đổi ngôn ngữ và kiểu gõ.").font(.footnote).foregroundColor(GlassTheme.muted)
                                .fixedSize(horizontal: false, vertical: true)
                        }
                        GlassCard("Thử bàn phím", subtitle: "Gõ thử tiếng Việt hoặc từ gõ tắt") {
                            TextField("Ví dụ: tieengs Vieetj", text: $preview)
                                .autocapitalization(.none)
                                .padding(.horizontal, 14)
                                .padding(.vertical, 12)
                                .frame(maxWidth: .infinity)
                                .background(Color(uiColor: .tuiRaised), in: RoundedRectangle(cornerRadius: 14, style: .continuous))
                                .overlay(RoundedRectangle(cornerRadius: 14, style: .continuous).stroke(Color(uiColor: .tuiBorder), lineWidth: 1))
                        }
                        Text("Chỉ trên thiết bị của bạn. Không thu thập nội dung gõ.").font(.caption).foregroundColor(GlassTheme.muted)
                    }
                    .padding(.horizontal, outerHPad)
                    .padding(.vertical, 20)
                    .frame(maxWidth: 640)
                    .frame(maxWidth: .infinity)
                }
            }.navigationBarHidden(true)
        }.navigationViewStyle(.stack).tint(GlassTheme.accent).foregroundColor(GlassTheme.text)
            .onAppear { Prefs.migrateInputMethod() }
    }
    private func openSettings() {
        if let url = URL(string: UIApplication.openSettingsURLString) { UIApplication.shared.open(url) }
    }

    /// Mini keyboard preview swatch.
    private func swatch(_ t: KeyTheme) -> some View {
        ZStack {
            RoundedRectangle(cornerRadius: 5, style: .continuous).fill(Color(uiColor: t.bg))
            VStack(spacing: 3) {
                HStack(spacing: 2) {
                    ForEach(0..<4) { _ in
                        RoundedRectangle(cornerRadius: 2, style: .continuous).fill(Color(uiColor: t.key))
                    }
                }.frame(height: 9)
                HStack(spacing: 2) {
                    RoundedRectangle(cornerRadius: 2, style: .continuous).fill(Color(uiColor: t.space)).frame(width: 40)
                    RoundedRectangle(cornerRadius: 2, style: .continuous).fill(Color(uiColor: t.accent))
                }.frame(height: 9)
            }.padding(4)
        }.frame(width: 72, height: 32)
    }
}
