<p align="center">
  <img src="assets/logo-diep-pham-go-phim.jpg" width="160" alt="Tui Gõ">
</p>

<h1 align="center">Tui Gõ</h1>

<p align="center">
  Bộ gõ tiếng Việt cho Windows, Android, iOS, macOS và Linux.<br>
  Xử lý trên thiết bị · Không lưu lịch sử gõ · Không telemetry.
</p>

<p align="center">
  <a href="#nền-tảng">Nền tảng</a> ·
  <a href="#tính-năng">Tính năng</a> ·
  <a href="#quyền-riêng-tư-và-dữ-liệu">Quyền riêng tư</a> ·
  <a href="#cài-đặt-và-build">Cài đặt / Build</a> ·
  <a href="#kiểm-chứng">Kiểm chứng</a>
</p>

## Giới thiệu

**Tui Gõ** xử lý tiếng Việt ngay trên máy hoặc điện thoại của bạn. Engine được port từ UniKey/x-unikey sang Go, Kotlin và Swift; mỗi nền tảng dùng giao diện và cơ chế bàn phím native.

Windows có app portable; Android và iOS có app cài đặt cùng bàn phím; macOS tích hợp InputMethodKit; Linux tích hợp IBus. Không cần tài khoản để gõ.

## Nền tảng

| Nền tảng | Thành phần | Yêu cầu từ source |
| --- | --- | --- |
| Windows | App portable, khay hệ thống, HUD và cài đặt | Windows x64 / ARM64; Go 1.23+ khi build |
| Android | App cài đặt + InputMethodService | Android 8.0+; JDK 17, Android SDK khi build |
| iOS / iPadOS | App cài đặt + Keyboard Extension | iOS 15+; macOS, Xcode và XcodeGen khi build |
| macOS | InputMethodKit + cài đặt SwiftUI | macOS 12+; Xcode và XcodeGen khi build |
| Linux | Engine IBus hoặc addon fcitx5 (Sen, không preedit) + cài đặt Fyne | IBus hoặc fcitx5; Go 1.23+ và thư viện GL/X11 khi build GUI |

[Bản phát hành](https://github.com/captajn/TuiGoTiengViet/releases) · [Build trên GitHub Actions](https://github.com/captajn/TuiGoTiengViet/actions)

Source có đủ năm nền tảng. Bản tải sẵn, chữ ký và kết quả kiểm thử phụ thuộc từng release/run; có source hoặc build thành công không đồng nghĩa đã kiểm tra giao diện trên mọi thiết bị.

## Tính năng

- **Telex, VNI, VIQR**; hỗ trợ đặt dấu tự do, kiểu dấu mới/cũ, kiểm tra chính tả và khôi phục từ ngoại ngữ.
- **Gõ tắt (macro)** lưu cục bộ; Backspace hoàn tác theo engine.
- **Windows:** thêm Telex đơn giản, Telex + VNI, MS-VI và keymap tùy chỉnh; Unicode/TCVN3, SendInput/Clipboard, ứng dụng loại trừ, phím tắt, khay hệ thống, HUD và giao diện sáng/tối.
- **Android/iOS:** ba kiểu gõ Telex/VNI/VIQR, theme, hàng số, tùy chọn gợi ý dán nhanh từ bộ nhớ tạm, điều chỉnh kích thước/khoảng cách phím và màn hình quản lý macro. Vuốt ngang Space để di chuyển con trỏ; giữ Space để mở menu ngôn ngữ/kiểu gõ/chuyển bàn phím.
- **macOS/Linux:** chọn kiểu gõ trong cài đặt, bật/tắt tiếng Việt bằng menu hoặc phím tắt. Từ đang soạn được giữ trong marked text/preedit cho đến ranh giới từ. Linux có thêm addon **fcitx5** (`fcitx5/`, chế độ Sen kiểu Gõ Sen của Ngó Sen): chữ commit ngay khi gõ, không preedit, sửa dấu bằng xóa chữ đã commit rồi commit lại.

### Gõ tắt

File UTF-8 dùng định dạng **từ-tắt:nội-dung**, mỗi mục một dòng:

~~~text
ko:không
dc:được
vn:Việt Nam
~~~

Bật gõ tắt, gõ từ tắt rồi Space để bung nội dung. Android/iOS có trình quản lý trong app; desktop dùng macro.txt tại thư mục dữ liệu bên dưới.

### Windows: nhớ ngôn ngữ và loại trừ theo app

Trong **Cài đặt → Giao diện & Hệ thống → Ứng dụng loại trừ**, bật **Tự nhớ bật/tắt tiếng Việt theo ứng dụng** nếu muốn lưu VI/EN theo tên .exe. Mặc định tùy chọn này tắt; việc lưu trạng thái không lưu nội dung đã gõ.

File exclude.txt cạnh exe hỗ trợ:

~~~text
notepad.exe
game.exe|lock
metro.exe|clip
acad.exe|tcvn,vni
~~~

Các flag gồm lock, clip, safedel, tcvn và kiểu gõ telex/stelex/vni/viqr/msvi/tvni. Alt+Z hoặc Ctrl+Shift chuyển VI/EN; F5 mở cài đặt, F9 bật/tắt macro, F12 reset bộ đệm.

## Quyền riêng tư và dữ liệu

### Nội dung gõ và chẩn đoán

- App xử lý chuỗi đang soạn trong bộ nhớ để biến đổi tiếng Việt, không lưu lịch sử gõ và không gửi nội dung gõ tới server.
- Source hiện tại không chủ động ghi file log chẩn đoán, heartbeat, thông số sửa chữ hay theo dõi mã thoát app. Không có công tắc debug-log.
- Không tích hợp SDK analytics/telemetry/crash-upload trong source sản phẩm.
- Bản Windows cũ có thể còn ghi crash.log. Cần dùng binary được build từ source mới để áp dụng thay đổi; app không tự xóa file log cũ của bạn.
- Báo cáo lỗi do hệ điều hành, runtime hoặc framework tạo ra thuộc cơ chế của nền tảng; đây không phải cam kết tắt mọi log của hệ điều hành.

### Dữ liệu nào được lưu?

| Nền tảng | Vị trí | Dữ liệu |
| --- | --- | --- |
| Windows | Cạnh tuigo.exe | tuigo.ini, macro.txt, keymap.txt, exclude.txt; version.txt và update-stamp.txt là trạng thái phiên bản/cập nhật |
| Android | Bộ nhớ riêng của app | SharedPreferences, macro.txt, lựa chọn giao diện/kiểu gõ và VI/EN theo package ứng dụng; nếu bật **Lịch sử bộ nhớ tạm**: clipboard_history.json + clips/ (số mục lưu tùy chỉnh, kể cả ảnh thu nhỏ) |
| iOS / iPadOS | App Group group.com.tuigo và preferences của extension | Tùy chọn, macro.txt; VI/EN theo documentIdentifier khi extension có quyền ghi; nếu bật **Lịch sử bộ nhớ tạm**: clipboard_history.json + clips/ (chỉ ghi được khi extension được cấp Full Access) |
| macOS | ~/Library/Application Support/TuiGo/ | tuigo.ini, macro.txt, keymap.txt |
| Linux | $XDG_CONFIG_HOME/tuigo/, mặc định ~/.config/tuigo/ | tuigo.ini, macro.txt, keymap.txt |

Macro do bạn nhập được lưu dưới dạng văn bản cục bộ, không phải lịch sử gõ. Các file này không được app mã hóa riêng. Android đặt allowBackup=false; phạm vi sao lưu của hệ điều hành trên các nền tảng khác tùy cấu hình thiết bị.

Windows chỉ tạo Run key/task khởi động khi bạn bật chức năng tương ứng. macOS có thể tạo LaunchAgent khi bạn chọn khởi động cùng hệ thống.

### Quyền truy cập và clipboard

- **Windows:** hook bàn phím/chuột phục vụ xử lý gõ, con trỏ và phím tắt. Chế độ Clipboard và nhánh tương thích lựa chọn có thể đọc/thay tạm clipboard để gửi/sửa chữ, với cơ chế khôi phục; dữ liệu này không được ghi thành log hoặc tải lên.
- **Android:** đăng ký dịch vụ bàn phím với quyền BIND_INPUT_METHOD. Khi bật tùy chọn **Gợi ý dán từ bộ nhớ tạm**, bàn phím chỉ đọc mục văn bản vừa sao chép trong bộ nhớ để hiện chip dán nhanh trên thanh gợi ý. Khi bật **Lịch sử bộ nhớ tạm** (mặc định bật), bàn phím ghi lại các mục đã sao chép trong lúc bàn phím chạy — số mục lưu tối đa tùy chỉnh trong Cài đặt (mặc định 50), lưu cục bộ trong bộ nhớ riêng của app, không đồng bộ; clip đánh dấu nhạy cảm và nội dung sao chép trong ô mật khẩu không được ghi. Mở danh sách từ biểu tượng đồng hồ trên thanh gợi ý; xóa từng mục, ghim hoặc xóa hết trong panel đó. Manifest không xin INTERNET; bạn tự bật/chọn bàn phím trong Cài đặt hệ thống.
- **iOS:** extension không yêu cầu Full Access để gõ. Ghi lựa chọn VI/EN vào App Group hoặc đọc pasteboard hệ thống từ extension phụ thuộc quyền hệ điều hành cấp; thiếu quyền thì vẫn đổi VI/EN trong phiên. Với Full Access, **Lịch sử bộ nhớ tạm** lưu các mục đã sao chép trong khi bàn phím mở (số mục tùy chỉnh, mặc định 50; bỏ qua ô mật khẩu) vào App Group, mở panel từ biểu tượng đồng hồ trên thanh gợi ý. App không dùng Full Access để kết nối mạng.
- **macOS/Linux:** dùng API input method của hệ điều hành. Source xử lý việc gõ không có HTTP client hay chức năng đồng bộ cloud.

### Kết nối mạng và cập nhật

Chỉ **tuigo-updater.exe**, companion tùy chọn trên Windows, có HTTP client. Khi đặt nó cạnh app, app khởi chạy updater; kiểm tra tự động được giới hạn khoảng 20 giờ/lần. Nút **Kiểm tra cập nhật** chạy kiểm tra thủ công.

Updater truy vấn GitHub Releases của repo này và tải asset qua URL do GitHub trả về; GitHub có thể nhận IP, thời điểm và User-Agent của request. Updater không gửi macro, cấu hình hay nội dung gõ. Nó hỏi trước khi thay bản đang dùng.

Không đặt updater cạnh app nếu muốn tắt cơ chế này. Android, iOS, macOS và Linux hiện không có updater mạng tích hợp. Công cụ build có thể tải dependencies; điều đó khác với kết nối mạng lúc dùng bàn phím.

## Cài đặt và build

Các lệnh dưới đây chạy từ thư mục repo, trừ khi có cd riêng.

### Windows

Không cần cài đặt: đặt tuigo.exe trong một thư mục có quyền ghi để lưu tùy chọn/macro.

~~~powershell
cd app
go build -ldflags "-H windowsgui -s -w" -o ../tuigo.exe .
go build -ldflags "-H windowsgui -s -w" -o ../tuigo-updater.exe ./updater
~~~

Updater là tùy chọn. Binary app chuẩn nằm ở root repo, tránh tạo thêm app/tuigo.exe.

### Android

~~~powershell
cd android
./gradlew.bat :engine:test :app:assembleDebug
~~~

APK debug nằm ở `android/app/build/outputs/apk/debug/app-debug.apk`. Cài APK rồi mở **Tui Gõ → Bật bàn phím → Chọn bàn phím**. Bản phát hành chính thức (`com.tuigo.tiengviet`, build bằng `:app:assembleRelease`) được ký với chứng chỉ SHA-256: `4e5f43f62a3c5442ddb85663fa1fe812fc1d655620a70693659d0feb14b67f1b` (V2 + V3 Signature Scheme); debug APK không thay được app release có chữ ký khác.

### iOS / iPadOS

Trên macOS, chuẩn bị Xcode và XcodeGen:

~~~sh
cd ios
xcodegen
open TuiGo.xcodeproj
~~~

Chọn scheme TuiGo; simulator không cần ký cho kiểm thử. Cài trên iPhone/iPad cần ký/provisioning phù hợp cho app và keyboard extension.

Thêm Tui Gõ trong **Cài đặt → Tổng quát → Bàn phím → Thêm bàn phím mới**, rồi dùng nút đổi bàn phím của hệ thống.

Mobile artifact dùng tên tuigo-vX.Y.Z.apk / tuigo-vX.Y.Z.ipa và tag mobile-vX.Y.Z theo scripts/mobile-release.py. IPA unsigned cần được ký trước khi cài; đọc mô tả release thay vì mặc định cho rằng mọi IPA đã cài được.

### macOS

~~~sh
sh scripts/check-macos.sh
~~~

App được build ở build/macos/Build/Products/Release/Tui Go.app. Copy vào ~/Library/Input Methods/, rồi thêm **Tui Gõ** trong cài đặt nguồn nhập; có thể cần đăng xuất/đăng nhập để macOS nhận nguồn mới.

Bật/tắt VI/EN trong input menu hoặc bằng Ctrl+Shift / Alt+Z theo tùy chọn. Cài đặt và macro editor nằm trong cùng app.

### Linux

Trên Ubuntu/Debian, cần IBus; khi build GUI cần thêm thư viện GL/X11, pkg-config. scripts/check-linux.sh liệt kê dependencies cho build và kiểm thử.

~~~sh
cd linux
./install.sh
ibus restart
~~~

Mặc định cài cho user vào ~/.local/lib/tuigo/; sau đó thêm Tui Gõ trong nguồn nhập của hệ thống. ./install.sh --system là cài toàn hệ thống, cần quyền quản trị. ./uninstall.sh gỡ phần cài đặt.

## Kiểm chứng

~~~powershell
python -P scripts/check-no-diagnostics.py
powershell -File scripts/check-diagnostics.ps1
powershell -File scripts/check-mobile.ps1
~~~

Trên host tương ứng:

~~~sh
sh scripts/check-ios.sh
sh scripts/check-macos.sh
sh scripts/check-linux.sh
~~~

Source guard kiểm tra năm nền tảng: không có đường log chẩn đoán/debug-log, telemetry SDK hoặc HTTP client trong app xử lý gõ; kiểm tra manifest Android và yêu cầu quyền của keyboard iOS.

Build/test engine không chứng minh bàn phím đã render hoặc tương tác đúng trên mọi thiết bị. iOS/macOS cần kiểm chứng native trên macOS; Linux cần IBus thực; Android cần emulator/thiết bị phù hợp cho kiểm thử tương tác.

## Cấu trúc project

~~~text
app/          Windows app + engine Go + updater tùy chọn
android/      App cài đặt, bàn phím Android và engine Kotlin
ios/          App iOS, Keyboard Extension, engine Swift và shared preferences
macos/        InputMethodKit, cài đặt và macro editor
linux/        IBus engine, cài đặt Fyne và script cài/gỡ
scripts/      Kiểm tra, metadata và đóng gói
assets/       Logo và hình tham khảo
~~~
