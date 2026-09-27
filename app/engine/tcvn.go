package engine

// unicodeToTCVN maps a Unicode codepoint to its TCVN3 codepoint, built by
// pairing unicodeTable (StdVnChar index -> Unicode) with tcvn3Table
// (StdVnChar index -> TCVN3 byte).
var unicodeToTCVN = func() map[uint16]uint16 {
	m := make(map[uint16]uint16, TotalVnChars)
	for i, u := range unicodeTable {
		if u != 0 {
			m[u] = uint16(tcvn3Table[i])
		}
	}
	return m
}()

// LookupTCVN returns the TCVN3 codepoint for a Unicode char.
// Chars without a TCVN3 mapping are returned unchanged with ok=false.
func LookupTCVN(u uint16) (uint16, bool) {
	t, ok := unicodeToTCVN[u]
	return t, ok
}

// ConvertTCVN rewrites UTF-16 output units to TCVN3 codepoints in place.
func ConvertTCVN(out []uint16) {
	for i, u := range out {
		if t, ok := unicodeToTCVN[u]; ok {
			out[i] = t
		}
	}
}
