package theme

import "testing"

// titleCase lower lowercased whole titles, so both cold monitors in #61 moved SIGTERM
// out of their confirm's title rather than show "send sigterm?". A word in capitals
// keeps them.
func TestCaseKeepsAcronyms(t *testing.T) {
	th := func(tc, lc string) *Theme {
		f := Default()
		f.Expressions.TitleCase, f.Expressions.LabelCase = tc, lc
		return Resolve(f, true, false)
	}
	lower, upper := th("lower", "lower"), th("upper", "upper")
	for _, c := range []struct{ in, want string }{
		{"Send SIGTERM to 13912?", "send SIGTERM to 13912?"},
		{"CPU and Memory", "CPU and memory"},
		{"Merge 3 PRs", "merge 3 PRs"},
		{"SIGTERM's grace period", "SIGTERM's grace period"},
		{"Upgrade to HTTP/2", "upgrade to HTTP/2"},
		{"A New Session", "a new session"},
		{"Roll back checkout-api?", "roll back checkout-api?"},
		{"Terminal too small", "terminal too small"},
	} {
		if got := lower.Case(c.in); got != c.want {
			t.Errorf("lower: Case(%q) = %q, want %q", c.in, got, c.want)
		}
	}
	if got := upper.Case("send SIGTERM to pid"); got != "SEND SIGTERM TO PID" {
		t.Errorf("upper: Case = %q", got)
	}
	// labels are the brand's small words: labelCase lower lowers capitals ("CPU" header)
	if got := lower.Label("CPU", "upper"); got != "cpu" {
		t.Errorf("lower: Label = %q, want cpu", got)
	}
}

// A footer that names one arrow ("↓ select" while only down moves) had no glyph for it:
// a "↓" literal fails tuilint code's C-GLYPH, and ↑↓ said a key that did nothing. up and
// down are glyphs with twins, optional in the theme.
func TestSingleArrowsAreGlyphs(t *testing.T) {
	g := Resolve(Default(), true, false).G
	a := Resolve(Default(), true, true).G
	if g.Up != "↑" || g.Down != "↓" || a.Up != "^" || a.Down != "v" {
		t.Errorf("up/down draw %q %q, ASCII %q %q; want ↑ ↓ and ^ v", g.Up, g.Down, a.Up, a.Down)
	}
	f, err := Parse([]byte(`{"glyphs": {"up": "▲", "down": "▼", "ascii": {"▼": "v"}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := CheckKeys([]byte(`{"glyphs": {"up": "▲", "down": "▼"}}`)); err != nil {
		t.Errorf("CheckKeys refuses up/down: %v", err)
	}
	if g := Resolve(f, true, true).G; g.Up != "^" || g.Down != "v" {
		t.Errorf("a theme's own arrows twin as %q %q", g.Up, g.Down)
	}
}
