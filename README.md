<p align="center">
  <img src="assets/logo-diep-pham-go-phim.jpg" width="160" alt="Tui Gõ">
</p>

<h1 align="center">Tui Gõ</h1>

<p align="center">
  Bộ gõ tiếng Việt cho Windows — thuần Go, một file exe duy nhất, không DLL, không phụ thuộc.
</p>

<p align="center">
  <a href="#tính-năng">Tính năng</a> ·
  <a href="#cài-đặt--build">Cài đặt</a> ·
  <a href="#cách-dùng">Cách dùng</a> ·
  <a href="#cấu-hình-riêng-theo-ứng-dụng">Per-app</a>
</p>

---

## Giới thiệu

**Tui Gõ** là bộ gõ tiếng Việt độc lập cho Windows, viết hoàn toàn bằng Go — engine gõ được port từ UniKey/x-unikey sang Go thuần, build ra **một file `.exe` duy nhất**, không cần `ukengine.dll`, không cần cài đặt, không phụ thuộc runtime ngoài.

Engine tách riêng khỏi shell Windows (`app/engine/`) nên có thể tái sử dụng cho macOS / Android / iOS sau này.

## Tính năng

### Kiểu gõ — 7 chế độ

| Kiểu gõ | Ví dụ | Ghi chú |
|---|---|---|
| **Telex** | `oonr` → ổn | s f r x j · mũ: aa ee oo · w · dd |
| **Telex đơn giản** | | `w` chỉ thêm móc (ư ơ) |
| **VNI** | `on63` → ổn | dấu: 1-5 · mũ: 6 · móc: 7 · trăng: 8 · đ: 9 |
| **VIQR** | | dấu: ' ` ? ~ . · mũ: ^ · móc: + |
| **Telex + VNI** | `oonr` hoặc `on63` đều → ổn | hybrid, gõ lẫn hai kiểu |
| **MS-VI** | | tương thích Microsoft |
| **Tự định nghĩa** | | chỉnh `keymap.txt` |

### Bảng mã

- **Unicode** (mặc định)
- **TCVN3** — cho phần mềm legacy (CAD, …)

### Cốt lõi

- Bỏ dấu kiểu mới / kiểu cũ, đặt dấu tự do, kiểm tra chính tả + tự khôi phục
- **Gõ tắt (macro)** qua `macro.txt` — `btw` → `by the way`
- Backspace thông minh — hoàn tác đúng từng bước biến đổi
- Chuyển V/E: phím tắt tùy chỉnh (mặc định `Alt+Z`, `Ctrl+Shift`), click tray, hoặc F1/F2
- Phím nhanh: **F5** mở cài đặt · **F9** bật/tắt gõ tắt · **F12** reset bộ đệm
- HUD trạng thái nổi + icon khay đổi màu theo V/E
- Giao diện sáng / tối, cửa sổ cài đặt dạng tab
- Chạy cùng Windows, tùy chọn quyền Admin (gõ được vào app elevated)
- Đường gửi phím: **SendInput** mặc định, **Clipboard** cho app khó tính (Metro/UWP, game)

### Cấu hình riêng theo ứng dụng

File `exclude.txt` (cạnh exe) — mỗi dòng một app, kết hợp flag bằng dấu phẩy:

```text
notepad.exe            tắt gõ trong app này — Alt+Z vẫn bật tạm cho riêng app đó
game.exe|lock          tắt hẳn, phím chuyển không tác dụng
metro.exe|clip         gõ được, gửi chữ qua Clipboard
acad.exe|tcvn,vni      trong CAD: kiểu gõ VNI, bảng mã TCVN3 — app khác vẫn Telex + Unicode
```

Flag kiểu gõ: `telex` `stelex` `vni` `viqr` `msvi` `tvni` · Flag bảng mã: `tcvn` · Flag chế độ: `lock` `clip`

## Cài đặt / Build

Yêu cầu: **Go 1.21+**, Windows (x64 hoặc ARM64).

```bat
cd app
go build -ldflags "-H windowsgui -s -w" -o tuigo.exe .
```

hoặc dùng `build.bat`. Build ARM64:

```bat
set GOARCH=arm64
go build -ldflags "-H windowsgui -s -w" -o tuigo-arm64.exe .
```

Không cần cài đặt — copy `tuigo.exe` đi đâu chạy cũng được. Lần đầu chạy app tự tạo `macro.txt`, `exclude.txt` cạnh exe.

### Cập nhật tự động (tuỳ chọn)

`tuigo.exe` **không chứa code mạng** — hoàn toàn offline. Nếu muốn auto-update, để file **`tuigo-updater.exe`** (có sẵn trong bản zip/release) nằm cạnh `tuigo.exe`:

- Có file → app tự chạy updater khi khởi động (tự giới hạn 20h/lần), có bản mới sẽ hỏi rồi tự tải + thay exe + khởi động lại
- Xoá file → tắt hoàn toàn tính năng, không còn kết nối nào
- Nút **"Kiểm tra cập nhật"** trong tab Giới thiệu để check thủ công bất cứ lúc nào

## Cách dùng

| Phím | Tác dụng |
|---|---|
| `Alt+Z` (tùy chỉnh được) | Chuyển Tiếng Việt ↔ English |
| `Ctrl+Shift` | Chuyển V/E (phím phụ) |
| Click icon khay | Chuyển V/E · Double-click: mở cài đặt |
| `F1` / `F2` | Bật TV / tắt TV |
| `F5` | Mở cài đặt |
| `F9` | Bật/tắt gõ tắt |
| `F12` | Reset bộ đệm gõ |

### Gõ tắt — `macro.txt`

```text
btw=by the way
ko=không
dc=được
```

## Cấu trúc project

```text
app/
  engine/        engine gõ thuần Go (port từ UniKey) — platform-independent
  *.go           shell Windows: keyboard hook, tray, HUD, settings UI
  app.ico        icon app · rsrc.syso — resource embed
  build.bat      script build (gui/console/arm64/res)
assets/          logo + UI concept tham khảo
```

## Nguyên tắc thiết kế

- **Một file exe** — copy là chạy, không installer
- **Không DLL** — engine thuần Go, gọi Win32 trực tiếp qua syscall
- **Riêng tư tuyệt đối** — `tuigo.exe` hoàn toàn offline: không network, không telemetry, không log, keystroke chỉ nằm trong RAM (auto-update nằm ở file `tuigo-updater.exe` riêng, xoá là hết)
- **Đồng bộ** — inject phím ngay trong hook callback, không race thứ tự chữ
- **Per-app trước** — mỗi ứng dụng một profile riêng nếu cần
