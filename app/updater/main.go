// tuigo-updater — optional companion exe for Tui Gõ.
//
// Presence = opt-in: drop this file next to tuigo.exe and the main app will
// launch it on startup (it self-throttles to once per 20h). Delete it and
// Tui Gõ is fully offline again — tuigo.exe itself contains zero network code.
//
// Flags: -now  skip the throttle and always report the result (manual check).
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const repoAPI = "https://api.github.com/repos/captajn/TuiGoTiengViet/releases/latest"
const throttleSecs = 20 * 3600

var (
	user32               = windows.NewLazySystemDLL("user32.dll")
	pMessageBox          = user32.NewProc("MessageBoxW")
	pFindWindow          = user32.NewProc("FindWindowW")
	pPostMessage         = user32.NewProc("PostMessageW")
	shell32              = windows.NewLazySystemDLL("shell32.dll")
	pShellExecute        = shell32.NewProc("ShellExecuteW")
	kernel32             = windows.NewLazySystemDLL("kernel32.dll")
	pGetModuleFileN      = kernel32.NewProc("GetModuleFileNameW")
	pOpenProcess         = kernel32.NewProc("OpenProcess")
	pWaitForSingleObject = kernel32.NewProc("WaitForSingleObject")
	pGetExitCodeProcess  = kernel32.NewProc("GetExitCodeProcess")
	pCloseHandle         = kernel32.NewProc("CloseHandle")
)

func utf16ptr(s string) *uint16 {
	p, _ := windows.UTF16PtrFromString(s)
	return p
}

func exeDir() string {
	var buf [windows.MAX_PATH]uint16
	n, _, _ := pGetModuleFileN.Call(0, uintptr(unsafe.Pointer(&buf[0])), windows.MAX_PATH)
	return filepath.Dir(windows.UTF16ToString(buf[:n]))
}

func msgBox(text, title string, style uintptr) uintptr {
	r, _, _ := pMessageBox.Call(0,
		uintptr(unsafe.Pointer(utf16ptr(text))),
		uintptr(unsafe.Pointer(utf16ptr(title))), style)
	return r
}

const (
	mbOK       = 0x0
	mbYesNo    = 0x4
	mbIconInfo = 0x40
	mbIconQ    = 0x20
	mbIconErr  = 0x10
	mbTopmost  = 0x40000
	idYes      = 6
)

func installedVersion(dir string) string {
	b, err := os.ReadFile(filepath.Join(dir, "version.txt"))
	if err != nil {
		return "0.0.0"
	}
	return strings.TrimSpace(string(b))
}

func newer(latest, cur string) bool {
	p := func(s string) [3]int {
		s = strings.TrimPrefix(strings.TrimSpace(s), "v")
		var v [3]int
		for i, part := range strings.SplitN(s, ".", 4) {
			if i >= 3 {
				break
			}
			fmt.Sscanf(part, "%d", &v[i])
		}
		return v
	}
	l, c := p(latest), p(cur)
	for i := 0; i < 3; i++ {
		if l[i] != c[i] {
			return l[i] > c[i]
		}
	}
	return false
}

type ghRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	} `json:"assets"`
}

func fetchLatest() (*ghRelease, error) {
	req, err := http.NewRequest("GET", repoAPI, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "TuiGo-Updater")
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("github api: %s", resp.Status)
	}
	var rel ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	return &rel, nil
}

func download(url, dst string) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "TuiGo-Updater")
	resp, err := (&http.Client{Timeout: 3 * time.Minute}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("download: %s", resp.Status)
	}
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

func assetURL(rel *ghRelease, base string) string {
	want := base
	if runtime.GOARCH == "arm64" {
		want = strings.TrimSuffix(base, ".exe") + "-arm64.exe"
	}
	for _, a := range rel.Assets {
		if a.Name == want {
			return a.URL
		}
	}
	return ""
}

// closeMainApp asks tuigo.exe to quit via WM_CLOSE on its hidden window and
// waits until the window is gone.
func closeMainApp() bool {
	hwnd, _, _ := pFindWindow.Call(
		uintptr(unsafe.Pointer(utf16ptr("BoGoTiengVietWnd"))), 0)
	if hwnd == 0 {
		return true
	}
	pPostMessage.Call(hwnd, 0x0010 /*WM_CLOSE*/, 0, 0)
	for i := 0; i < 120; i++ {
		time.Sleep(50 * time.Millisecond)
		hwnd, _, _ = pFindWindow.Call(
			uintptr(unsafe.Pointer(utf16ptr("BoGoTiengVietWnd"))), 0)
		if hwnd == 0 {
			time.Sleep(200 * time.Millisecond)
			return true
		}
	}
	return false
}

func swap(oldPath, newPath, backup string) error {
	os.Remove(backup)
	var err error
	for i := 0; i < 20; i++ {
		if err = os.Rename(oldPath, backup); err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err != nil {
		return err
	}
	if err = os.Rename(newPath, oldPath); err != nil {
		os.Rename(backup, oldPath) // roll back
		return err
	}
	return nil
}

func writeStamp(dir string) {
	os.WriteFile(filepath.Join(dir, "update-stamp.txt"),
		[]byte(strconv.FormatInt(time.Now().Unix(), 10)), 0644)
}

func throttled(dir string) bool {
	b, err := os.ReadFile(filepath.Join(dir, "update-stamp.txt"))
	if err != nil {
		return false
	}
	ts, err := strconv.ParseInt(strings.TrimSpace(string(b)), 10, 64)
	return err == nil && time.Now().Unix()-ts < throttleSecs
}

// wlog appends one timestamped line to crash.log next to the exe —
// shared diagnostics with the main app.
func wlog(format string, a ...any) {
	f, err := os.OpenFile(filepath.Join(exeDir(), "crash.log"),
		os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s "+format+"\n",
		append([]any{time.Now().Format("2006-01-02 15:04:05")}, a...)...)
}

// watchProcess blocks until the main app exits and records its exit code —
// the only way to tell a clean exit (0), a Go fatal (2), a crash
// (0xC0000005…), or an external TerminateProcess (killer-chosen code).
func watchProcess(pid uint32) {
	const syncAccess = 0x101000 // SYNCHRONIZE | PROCESS_QUERY_LIMITED_INFORMATION
	h, _, _ := pOpenProcess.Call(syncAccess, 0, uintptr(pid))
	if h == 0 {
		wlog("watchdog: OpenProcess(%d) failed", pid)
		return
	}
	defer pCloseHandle.Call(h)
	pWaitForSingleObject.Call(h, 0xFFFFFFFF)
	var code uint32
	pGetExitCodeProcess.Call(h, uintptr(unsafe.Pointer(&code)))
	wlog("watchdog: pid %d exited code=0x%X", pid, code)
}

func main() {
	manual := false
	var watchPID uint32
	for i, a := range os.Args[1:] {
		switch a {
		case "-now":
			manual = true
		case "-watch":
			if i+2 < len(os.Args) {
				if v, err := strconv.ParseUint(os.Args[i+2], 10, 32); err == nil {
					watchPID = uint32(v)
				}
			}
		}
	}
	dir := exeDir()
	if watchPID != 0 {
		defer watchProcess(watchPID) // runs last — after update work
	}

	if !manual && throttled(dir) {
		return
	}
	writeStamp(dir)

	rel, err := fetchLatest()
	if err != nil {
		if manual {
			msgBox("Không kết nối được GitHub.\nKiểm tra lại mạng rồi thử lại.",
				"Tui Gõ — Cập nhật", mbOK|mbIconErr|mbTopmost)
		}
		return
	}

	cur := installedVersion(dir)
	if !newer(rel.TagName, cur) {
		if manual {
			msgBox("Tui Gõ đang là bản mới nhất (v"+cur+").",
				"Tui Gõ — Cập nhật", mbOK|mbIconInfo|mbTopmost)
		}
		return
	}

	if msgBox("Đã có Tui Gõ "+rel.TagName+" (hiện tại v"+cur+").\nCập nhật ngay?",
		"Tui Gõ — Cập nhật", mbYesNo|mbIconQ|mbTopmost) != idYes {
		return
	}

	exeURL := assetURL(rel, "tuigo.exe")
	if exeURL == "" {
		msgBox("Bản phát hành "+rel.TagName+" không có file cài cho máy này.",
			"Tui Gõ — Cập nhật", mbOK|mbIconErr|mbTopmost)
		return
	}

	tmpExe := filepath.Join(dir, "tuigo.new.exe")
	if err := download(exeURL, tmpExe); err != nil {
		msgBox("Tải bản mới thất bại:\n"+err.Error(),
			"Tui Gõ — Cập nhật", mbOK|mbIconErr|mbTopmost)
		return
	}

	// Also refresh ourselves if the release ships a newer updater — the
	// rename trick works on our own running image too.
	var tmpUpd string
	if u := assetURL(rel, "tuigo-updater.exe"); u != "" {
		tmpUpd = filepath.Join(dir, "tuigo-updater.new.exe")
		if err := download(u, tmpUpd); err != nil {
			tmpUpd = ""
		}
	}

	if !closeMainApp() {
		msgBox("Không đóng được Tui Gõ đang chạy.\nThoát app thủ công rồi thử lại.",
			"Tui Gõ — Cập nhật", mbOK|mbIconErr|mbTopmost)
		return
	}

	if err := swap(filepath.Join(dir, "tuigo.exe"), tmpExe,
		filepath.Join(dir, "tuigo.old.exe")); err != nil {
		msgBox("Thay file thất bại:\n"+err.Error(),
			"Tui Gõ — Cập nhật", mbOK|mbIconErr|mbTopmost)
		return
	}
	if tmpUpd != "" {
		self := filepath.Join(dir, "tuigo-updater.exe")
		_ = swap(self, tmpUpd, filepath.Join(dir, "tuigo-updater.old.exe"))
	}

	pShellExecute.Call(0,
		uintptr(unsafe.Pointer(utf16ptr("open"))),
		uintptr(unsafe.Pointer(utf16ptr(filepath.Join(dir, "tuigo.exe")))),
		0, uintptr(unsafe.Pointer(utf16ptr(dir))), 1 /*SW_SHOWNORMAL*/)

	msgBox("Đã cập nhật lên "+rel.TagName+".",
		"Tui Gõ — Cập nhật", mbOK|mbIconInfo|mbTopmost)
}
