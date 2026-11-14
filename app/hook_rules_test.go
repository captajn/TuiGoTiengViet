package main

import "testing"

// Terminal hosts must not auto-capitalize: the first letter after a full stop
// or Enter is an option, a path or a lowercase identifier, not a sentence start.
func TestAutoCapSkipsTerminalApps(t *testing.T) {
	old := cfg
	defer func() { cfg = old }()
	cfg.AutoCap = true

	for _, exe := range []string{
		"cmd.exe", "conhost.exe", "openconsole.exe", "powershell.exe",
		"pwsh.exe", "windowsterminal.exe", "wt.exe", "wsl.exe", "bash.exe",
		"mintty.exe", "alacritty.exe", "wezterm-gui.exe", "kitty.exe",
		"conemu.exe", "conemu64.exe", "hyper.exe",
	} {
		if autoCapEnabled(exe) {
			t.Errorf("%s: auto-capitalize must stay off in a terminal", exe)
		}
	}

	for _, exe := range []string{"winword.exe", "notepad.exe", "chrome.exe", "telegram.exe", "zalo.exe"} {
		if !autoCapEnabled(exe) {
			t.Errorf("%s: auto-capitalize must stay on for prose", exe)
		}
	}
}

func TestAutoCapFollowsSetting(t *testing.T) {
	old := cfg
	defer func() { cfg = old }()

	cfg.AutoCap = false
	if autoCapEnabled("winword.exe") {
		t.Error("a disabled setting must win over the app class")
	}
	cfg.AutoCap = true
	if isTerminalApp("winword.exe") {
		t.Error("winword.exe is not a terminal")
	}
	if !isTerminalApp("windowsterminal.exe") {
		t.Error("Windows Terminal must be recognised")
	}
	if isTerminalApp("") {
		t.Error("an unknown exe is not a terminal")
	}
}
