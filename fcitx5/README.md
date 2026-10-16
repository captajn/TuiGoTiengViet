# Tui Gõ cho fcitx5 (Linux) — chế độ Sen, không preedit

Addon fcitx5 cho Tui Gõ, lấy cơ chế **Gõ Sen** từ Ngó Sen: chữ commit ngay
khi gõ, không gạch chân, không preedit. Khi sửa dấu, addon xóa phần đã
commit rồi commit phần mới. Đường xóa chọn theo app (quirks, `src/quirks.h`,
port từ Ngó Sen — behavior trước, tên app chỉ khi behavior không phân
biệt được):

- App bỏ qua `deleteSurroundingText` (Edge, Telegram — đã quan sát thấy
  nhân đôi `chaào`) → luôn forward BackSpace + chờ xác nhận.
- App bắt buộc dùng surrounding text (GTK4 bỏ qua phím forward;
  LibreOffice áp dụng BackSpace muộn) → `deleteSurroundingText` đồng bộ.
- Trình duyệt (Chromium/Firefox/Electron, gồm các fork Chromium như
  Helium/Thorium/Ungoogled) → `deleteSurroundingText` đồng bộ, KHÔNG
  kiểm chứng snapshot (snapshot của Chrome thường cũ, kiểm chứng thất
  bại → rơi xuống đường chờ → giật khi gõ nhanh). Helium từng bị sót
  khỏi danh sách → rơi xuống đường async → gõ `hoafng` ra `ho` rồi
  mất luôn hệ gõ.
- Còn lại: kiểm chứng snapshot khớp rồi mới `deleteSurroundingText`;
  không khớp → forward BackSpace + chờ.

Đường forward đi kênh khác với commit (virtual keyboard vs input-method)
nên commit có thể tới trước — addon **chờ xác nhận xóa đã áp dụng
(surrounding text, timeout 100 ms) rồi mới commit**; phím gõ trong lúc
chờ được buffer rồi replay theo thứ tự. Chỉ tin snapshot sau-xóa khi đã
thấy snapshot trước-xóa (chứa đúng đoạn cần xóa) — snapshot cũ hơn phím
vừa gõ mà xác nhận ngay thì BackSpace tới sau sẽ ăn mất chữ mới
(lỗi gõ `dd` → mất `đ` trên Chrome). Reset/deactivate/activate giữa lúc
chờ xóa không được vứt pending commit — delete đã áp dụng, vứt commit
là mất chữ (`dd` → trống trơn) — phải flush commit rồi mới reset máy.
BackSpace sau space **mở lại từ trước**: engine coi từ là live lại
(`atWordBeginning` 1→0). Machine nhớ từ cuối (`lastWord_`) để khôi
phục `committed_`, và để app xóa space (không applyEdit — `backs` của
engine lúc đó chỉ boundary, không phải từ). Không làm vậy, phím dấu
sau đó xóa sai lượng → rác (`chào` + space + BackSpace + `f` →
`chàoaof`).
Pending commit mà không còn timer nào chạy (giữ qua deactivate mà
activate không flush) thì flush ngay ở phím tiếp theo — buffer mãi
sẽ nuốt mọi phím, hệ gõ chết hẳn cho tới khi E/V lại.
Switch giữa lúc chờ xóa (2 lỗ, đều gây gõ lúc được lúc không):
click chuột đổi tab/cửa sổ → `deactivate()` giữ pending commit lại
(`suspendWait`), không flush vào context đã mất focus (Wayland bỏ
commit đó → mất ký tự); khi quay lại, `activate()` flush vào context
đang active. Alt+Tab bị buffer trong lúc chờ → khi replay, sau khi
forward phím Alt/Super thì dừng, các phím sau nó gõ cho cửa sổ mới
chứ không phải cửa sổ cũ — replay tiếp sẽ đưa chữ sang nhầm cửa sổ.

Khác với engine IBus (`linux/`, giữ từ trong preedit): addon này gõ thẳng
vào app, nên chạy được cả những app không hiển thị preedit.

## Cấu trúc

| Phần | File | Ghi chú |
|---|---|---|
| Máy Sen | `src/sen.cpp` (+ `sen.h`) | Không phụ thuộc fcitx5, test được độc lập |
| Engine Go | `src/tuigo-engine.cpp` + `go/shim.go` | Cầu nối C tới engine Go của Tui Gõ (`app/engine`), build thành C archive |
| Vỏ fcitx5 | `src/addon.cpp`, `src/addon-factory.cpp` | `InputMethodEngineV2`, 1 SenMachine mỗi input context |
| Test | `test/sen_test.cpp` | Máy Sen + engine Go thật + sink giả, không cần fcitx5 |
| Cấu hình | `tuigo.conf.in`, `tuigo-addon.conf.in.in` | Cài vào fcitx5 |

Cấu hình dùng chung file `tuigo.ini` với bản IBus và tuigo-settings
(`$XDG_CONFIG_HOME/tuigo/tuigo.ini`): `VietKey`, `InputMethod`,
`SpellCheck`, `ModernStyle`, `FreeMarking`, `MacroEnabled`,
`AutoNonVnRestore`, `AutoCap`, cùng `macro.txt` / `keymap.txt`.
Không có giao diện cấu hình riêng của fcitx5 (`getConfig()` trả null).

## Build và test

Yêu cầu: Go 1.23+, CMake, C++17. Addon fcitx5 cần thêm gói dev của
fcitx5 (`libfcitx5core-dev`, `libfcitx5config-dev` trên Debian/Ubuntu);
không có thì chỉ build máy Sen + test.

```bash
cmake -B build -DCMAKE_BUILD_TYPE=Release
cmake --build build -j
ctest --test-dir build --output-on-failure
```

## Cài đặt

**Nhanh nhất — gói có sẵn:** workflow `.github/workflows/fcitx5-packages.yml`
(CPack) build `.deb` (Ubuntu/Debian) + `.rpm` (Fedora) cho addon. Chạy tay
từ tab Actions (upstream chỉ chạy manual để tiết kiệm phút Actions; fork
thì push vào `fcitx5/` cũng build). Tải gói về rồi cài:

```bash
# Ubuntu/Debian
sudo apt install ./fcitx5-tuigo_<version>_amd64.deb
# Fedora
sudo dnf install ./fcitx5-tuigo-<version>-1.x86_64.rpm
fcitx5 -rd   # hoặc đăng xuất/đăng nhập lại
```

Chạy workflow với `release: true` + tag `vX.Y.Z` để đính gói vào GitHub
release. Gói cài vào `/usr` (đúng chỗ fcitx5 tìm addon) và phụ thuộc
`fcitx5 (>= 5.0)`, nên không cần build tay.

**Build tay:** thư mục `fcitx5/` phải nằm trong repo TuiGoTiengViet:
`go/go.mod` trỏ `replace tuigo => ../../app` (engine Go). Đừng tách
riêng nó ra ngoài.

fcitx5 cài bằng apt nằm ở `/usr`, nên phải cài addon với prefix `/usr`
(prefix mặc định `/usr/local` làm fcitx5 không tìm thấy `.so`):

```bash
cmake -B build -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX=/usr
cmake --build build -j
sudo cmake --install build
fcitx5 -rd   # hoặc đăng xuất/đăng nhập lại
```

Sau đó thêm **Tui Gõ** trong cấu hình fcitx5 (`fcitx5-configtool`) và
chuyển sang nó bằng phím tắt của fcitx5. Gỡ: xóa `libtuigo.so` khỏi
`$libdir/fcitx5`, `tuigo.conf` khỏi `.../share/fcitx5/inputmethod` và
`.../share/fcitx5/addon`.

## Giới hạn (nói thẳng)

- Addon đã compile + chạy được với header fcitx5 thật (đã build trên máy
  có `libfcitx5core-dev`). Phần `addon.cpp` dùng các API call đã có trong
  code Ngó Sen: `commitString`, `deleteSurroundingText`, `forwardKey`,
  `surroundingText`, `frontend`, `eventLoop().addTimeEvent`.
- Icon khay là icon E/V thật của TuiGo (`app/tray_icon.go`: badge ngọc
  bích + chữ V khi VI, keycap trắng viền xám + chữ E khi EN), port sang
  PNG 22/24/48/256 cài vào `hicolor/<size>x<size>/apps` **và**
  `hicolor/<size>x<size>/status` — khay hệ thống tìm icon ở `status`.
  PNG, không phải SVG: icon SVG không hiện lên khay (trình render khay
  thiếu SVG loader). Chế độ VI/EN vẫn hiện bằng chữ `VI`/`EN` qua
  `subModeLabelImpl`. Nếu khay vẫn không hiện icon sau khi cài, chạy
  `sudo gtk-update-icon-cache /usr/share/icons/hicolor` rồi
  `fcitx5 -rd`.
- Game SDL trên Wayland: fcitx5 không forward được BackSpace đáng tin cậy
  — giới hạn này Ngó Sen cũng gặp.
- Ô địa chỉ Chromium (autofill): chưa xử lý selection gợi ý khi sửa dấu
  trong omnibox — Ngó Sen xử lý bằng 1 BackSpace thêm (#190), addon này
  chưa có.
