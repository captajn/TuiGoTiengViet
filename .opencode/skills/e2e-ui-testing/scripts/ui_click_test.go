// Template test harness — copy vào app/ (ví dụ app/ui_click_test.go) rồi chỉnh sửa.
//
// "Nhấn nút" qua SendMessage(WM_COMMAND) để đi đúng đường xử lý thật của
// settingsProc (đồng bộ, không cần message loop). Xem thêm references/win32-ui-automation.md
// trong skill e2e-ui-testing.
//
// Lưu ý:
//   - Test thuộc package main nên dùng được hằng số cXXX, ctlHnd, swState...
//   - *_test.go được miễn lệnh cấm diagnostic logging (t.Log/fmt.Print OK).
//   - openSettings() hiện cửa sổ thật trên desktop — test tự skip nếu không tạo được.
//   - settingsCommand ghi tuigo.ini thật; uiSnapshotIni() tự restore trong Cleanup.
package main

import (
	"os"
	"testing"
)

// uiOpenSettings mở cửa sổ settings thật và skip nếu môi trường không có desktop.
func uiOpenSettings(t *testing.T) {
	t.Helper()
	openSettings()
	if gSettingsHwnd == 0 {
		t.Skip("khong tao duoc settings window (can phien desktop tuong tac)")
	}
	t.Cleanup(func() {
		if gSettingsHwnd != 0 {
			pDestroyWindow.Call(gSettingsHwnd)
			gSettingsHwnd = 0
		}
	})
}

// uiSnapshotIni lưu tuigo.ini và restore khi test kết thúc (settingsCommand ghi file thật).
func uiSnapshotIni(t *testing.T) {
	t.Helper()
	path := iniPath()
	old, err := os.ReadFile(path)
	existed := err == nil
	t.Cleanup(func() {
		if existed {
			_ = os.WriteFile(path, old, 0o644)
		} else {
			_ = os.Remove(path)
		}
	})
}

// uiClick mô phỏng click nút BUTTON: WM_COMMAND + BN_CLICKED (code = 0).
// Handler thật (settingsCommand) chạy đồng bộ ngay trong lời gọi này.
func uiClick(t *testing.T, id int) {
	t.Helper()
	if gSettingsHwnd == 0 {
		t.Fatal("chua mo settings window — goi uiOpenSettings truoc")
	}
	pSendMessage.Call(gSettingsHwnd, WM_COMMAND, uintptr(id), ctlHnd[id])
}

// uiSwitchOn đọc state checkbox từ swState (KHÔNG phải BM_GETCHECK).
func uiSwitchOn(id int) bool { return swState[id] }

// uiComboSelect chọn một mục trong COMBOBOX rồi gửi CBN_SELCHANGE như Windows thật.
func uiComboSelect(t *testing.T, id, sel int) {
	t.Helper()
	pSendMessage.Call(ctlHnd[id], CB_SETCURSEL, uintptr(sel), 0)
	pSendMessage.Call(gSettingsHwnd, WM_COMMAND, uintptr(id)|uintptr(CBN_SELCHANGE)<<16, ctlHnd[id])
}

// uiComboSel đọc mục đang chọn của COMBOBOX.
func uiComboSel(t *testing.T, id int) int {
	t.Helper()
	r, _, _ := pSendMessage.Call(ctlHnd[id], CB_GETCURSEL, 0, 0)
	return int(r)
}

// ---- Ví dụ test ----

// TestUiTemplateClickRadioVni: click radio VNI qua đường WM_COMMAND thật,
// assert engine nhận kiểu gõ mới và trạng thái được lưu.
func TestUiTemplateClickRadioVni(t *testing.T) {
	uiSnapshotIni(t)
	uiOpenSettings(t)

	uiClick(t, cRadioVni)

	if gIM != ImVni {
		t.Fatalf("sau click cRadioVni: gIM = %v, muon ImVni", gIM)
	}
}

// TestUiTemplateClickSwitch: click toggle (vd cAutoCap), assert swState đảo chiều.
func TestUiTemplateClickSwitch(t *testing.T) {
	uiSnapshotIni(t)
	uiOpenSettings(t)

	before := uiSwitchOn(cAutoCap)
	uiClick(t, cAutoCap)
	if uiSwitchOn(cAutoCap) == before {
		t.Fatalf("swState[cAutoCap] khong doi sau click (truoc=%v)", before)
	}
}
