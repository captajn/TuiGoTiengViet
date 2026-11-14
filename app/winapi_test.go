package main

import "testing"

// Windows 10 ignores DWMWA_WINDOW_CORNER_PREFERENCE, but DwmSetWindowAttribute
// can still return S_OK for it. Rounding the client there left a square, blurry
// frame corner behind the painted round one, so the build number gates it too.
func TestRoundedCornersRequireWindows11(t *testing.T) {
	cases := []struct {
		name  string
		hr    uintptr
		build uint32
		want  bool
	}{
		{"win11 accepted", 0, 22631, true},
		{"win11 22000 boundary", 0, 22000, true},
		{"win10 rejects the attribute", 0x80070057, 19045, false},
		{"win10 21h2 accepted but ignored", 0, 19045, false},
		{"win11 attribute rejected", 1, 22631, false},
		{"unknown build", 0, 0, false},
	}
	for _, c := range cases {
		if got := roundedCornersSupported(c.hr, c.build); got != c.want {
			t.Errorf("%s: roundedCornersSupported(%#x, %d) = %v, want %v",
				c.name, c.hr, c.build, got, c.want)
		}
	}
}
