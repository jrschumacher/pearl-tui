package palette

import (
	"testing"

	"tuikit/theme/bubbletea"
)

// The warning never leaves yellow, even with both yellows held: it shares 3, never the
// brand's 11 on a dark page nor a status hue (#59 review: the fallback returned 11 and no
// test reached it).
func TestWarningPinStaysYellow(t *testing.T) {
	both := map[int]bool{1: true, 9: true, 3: true, 11: true}
	for _, dark := range []bool{true, false} {
		if n := Pin16("warning", "#CFA24E", dark, both); n != 3 {
			t.Errorf("dark=%v, both yellows held: warning pinned to %d, want 3", dark, n)
		}
	}
	free := map[int]bool{1: true, 9: true}
	if n := Pin16("warning", "#CFA24E", true, free); n != 11 {
		t.Errorf("dark, yellows free: warning pinned to %d, want its own 11", n)
	}
	// through Findings16: a warm brand on 11 and success pinned to 3 on a dark page, the
	// warning on green: it moves to 3 beside success (whose clash then moves success), never
	// to the brand's 11 or another hue
	f := theme.File{Roles: map[string]theme.Roles{"dark": {Bg: "#1C1917", Fg: "#E8E2D6", Primary: "#C2622B", Warning: "#F4C352", Success: "#4EB068", Error: "#E27170", Info: "#2EA2E5"}},
		ANSI16: map[string]map[string]int{"dark": {"primary": 11, "success": 3, "warning": 10}}}
	seen := false
	for _, x := range Findings16(f) {
		if x.Role == "warning" {
			seen = true
			if x.Pin != 3 {
				t.Errorf("warning's pin %d, want 3: %s", x.Pin, x.What)
			}
		}
	}
	if !seen {
		t.Errorf("no finding for the warning on green: %+v", Findings16(f))
	}
}
