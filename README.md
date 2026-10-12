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
- HUD trạng thái nổi + icon khay V nền ngọc bích / E viền trung tính theo giao diện sáng, tối; trạng thái theo ứng dụng đang mở
- Giao diện sáng / tối, cửa sổ cài đặt dạng tab
- Chạy cùng Windows, tùy chọn quyền Admin (gõ được vào app elevated)
- Đường gửi phím: **SendInput** mặc định, **Clipboard** cho app khó tính (Metro/UWP, game)

### Cấu hình riêng theo ứng dụng

Trong **Cài đặt → Giao diện → Ứng dụng loại trừ**, bật **Tự nhớ bật/tắt tiếng Việt theo ứng dụng**. Khi chuyển V/E bằng phím tắt hoặc icon khay trong một ứng dụng, Tui Gõ lưu lựa chọn riêng cho tên `.exe` đó. Chuyển sang app khác không làm đổi lựa chọn vừa lưu; quay lại app hoặc khởi động lại Tui Gõ vẫn giữ trạng thái. Các cửa sổ cùng tên `.exe` dùng chung lựa chọn.

Tùy chọn này mặc định tắt. Khi bật, ứng dụng đang gõ giữ chế độ hiện tại; app chưa có lựa chọn riêng mặc định gõ tiếng Việt (trừ app loại trừ thủ công). Bấm icon khay hoặc menu khay sẽ đổi chế độ của app đang gõ trước khi mở khay, kể cả khi Explorer đã nhận focus. Trạng thái được lưu trong `tuigo.ini` cạnh exe; tắt tùy chọn sẽ dùng lại chế độ chung nhưng vẫn giữ các lựa chọn đã nhớ để dùng khi bật lại. App có `|lock` luôn tắt tiếng Việt.

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
| Click icon khay | Mở cài đặt · Chuột phải: menu có mục chuyển V/E |
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
android/
  engine/        port Kotlin thuần của engine (test JVM, không cần emulator)
  app/           InputMethodService + keyboard view + app cài đặt
ios/
  Sources/TuiGoEngine/  port Swift thuần của engine (SwiftPM, swift test)
  Keyboard/      Custom Keyboard Extension (UIInputViewController + UIKit)
  App/           app cài đặt SwiftUI (kiểu gõ, tuỳ chọn, sửa macro)
  Shared/        Prefs qua App Group (group.com.tuigo)
  project.yml    XcodeGen — sinh TuiGo.xcodeproj
assets/          logo + UI concept tham khảo
```

## Mobile

### Android

```bat
cd android
gradlew.bat :engine:test :app:assembleDebug
```

APK ra ở `android/app/build/outputs/apk/debug/`. Cài lên máy: `adb install app-debug.apk` → Cài đặt → bật Tui Gõ làm bàn phím. JDK toolchain 17 tự tải qua Foojay resolver, không cần cài JDK riêng.

App mobile và bàn phím dùng giao diện sáng/tối theo thiết bị, card kính nhẹ và nút Space hiển thị **Tiếng Việt** hoặc **English**. Chạm **VN/EN** để đổi chế độ; trên Android, giữ Space để mở danh sách bàn phím. Vùng điều hướng dưới cùng do hệ thống quản lý.

Màu mobile đồng bộ bản Windows: xanh ngọc #197F64 ở light mode và mint #3ED3B8 ở dark mode. Bộ chọn kiểu gõ có đúng **Telex, VNI, VIQR**; các tùy chọn simple/hybrid từ bản mobile cũ chuyển về Telex. Với VIQR, trang **?123 → #+=** có các phím `^`, dấu huyền và `~`.

**Gõ tắt trên mobile:** bật Gõ tắt → Quản lý từ gõ tắt → nhập mỗi mục trên một dòng theo định dạng `key:text`, ví dụ `ko:không` và `vn:Việt Nam` → Lưu. Khi dùng chế độ Tiếng Việt, gõ `ko` rồi chạm Space để nhận `không`. Sửa dòng để đổi nội dung, xóa dòng để xóa mục; mở lại bàn phím để nạp danh sách mới. Từ tắt phân biệt chữ hoa/thường và tối đa 15 ký tự. Dữ liệu lưu UTF-8 trên thiết bị; trình sửa báo lỗi định dạng/lưu thay vì ghi đè dữ liệu cũ khi thất bại.

Kiểm tra Android từ thư mục gốc: `powershell -File scripts/check-mobile.ps1`. Với emulator dành cho kiểm thử đã khởi động: thêm `-Device -Serial emulator-5554` để cài APK debug và chạy kiểm tra lựa chọn kiểu gõ, prefs và lưu/xóa macro. Script không tự xóa ứng dụng có chữ ký khác trên thiết bị.

### iOS

Cần macOS + Xcode + [XcodeGen](https://github.com/yonaskolb/XcodeGen) (`brew install xcodegen`):

```sh
cd ios
xcodegen          # sinh TuiGo.xcodeproj (đã gitignore)
open TuiGo.xcodeproj
```

Chọn scheme **TuiGo** → chạy trên simulator hoặc máy thật (máy thật cần chọn Development Team trong Signing — App Group `group.com.tuigo` dùng được với free provisioning). Test engine không cần Xcode: `swift test`.

Kiểm tra engine, app và keyboard extension trên macOS: `sh scripts/check-ios.sh`. App cài đặt iOS cũng có quản lý macro, hàng số và điều chỉnh kích thước phím; Space và palette bàn phím đồng bộ với chế độ gõ/theme. Build hoặc kiểm tra code trên Windows không xác nhận giao diện iOS đã chạy đúng; cần Xcode/simulator hoặc thiết bị thật để QA.

## Nguyên tắc thiết kế

- **Một file exe** — copy là chạy, không installer
- **Không DLL** — engine thuần Go, gọi Win32 trực tiếp qua syscall
- **Riêng tư tuyệt đối** — `tuigo.exe` hoàn toàn offline: không network, không telemetry, không log, keystroke chỉ nằm trong RAM (auto-update nằm ở file `tuigo-updater.exe` riêng, xoá là hết)
- **Đồng bộ** — inject phím ngay trong hook callback, không race thứ tự chữ
- **Per-app trước** — mỗi ứng dụng một profile riêng nếu cần

### Mobile release artifacts

GitHub Actions builds `tuigo-v0.0.1.apk` and `tuigo-v0.0.1.ipa`. The mobile Git tag is `mobile-v0.0.1`; the release title is **Tui Gõ Mobile v0.0.1**. Metadata and artifact checks live in `scripts/mobile-release.py`. Android uses the configured release signing key (debug fallback without secrets). The IPA is unsigned and needs Apple signing/provisioning before installation. Both engine tests and native simulator/device builds run before IPA publication.

The symbol key at the Shift position cycles through common symbols, the expanded math/currency set, and additional existing symbols. `/` and `*` are on the first page. Before publishing a mobile tag, use `powershell -File scripts/publish-mobile-tag.ps1 -Tag mobile-v0.0.1 -Commit <verified-sha> -CheckOnly` to reject stale local/remote refs.
