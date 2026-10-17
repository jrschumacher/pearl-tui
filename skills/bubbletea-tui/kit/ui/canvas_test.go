package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"tuikit/theme"
)

// The theme's ellipsis resolves to its twin ("...") before a screen draws, but a cut
// made with the bare theme.Ellipsis (ui.Fit, a screen's own ansi.Truncate) reaches the
// canvas as "…", and the sweep drew it as "~": the partial mark's twin, so a cut line
// read as a row in progress ("unknown key~"). The sweep draws "." (one cell of the
// theme's twin) and never a status twin.
func TestASCIIfyCutIsNotThePartialTwin(t *testing.T) {
	th := theme.Resolve(theme.Default(), true, true)
	c := NewCanvas(th, 16, 1)
	c.Draw(Fit("unknown key value", 12), Rect(0, 0, 12, 1))
	c.ASCIIfy()
	row := ansi.Strip(c.String())
	if !strings.HasPrefix(row, "unknown key.") {
		t.Errorf("a Fit cut under ASCII draws %q, want \"unknown key.\"", row)
	}
	if strings.ContainsAny(row, th.G.Partial+"~") {
		t.Errorf("a Fit cut under ASCII draws %q: the partial mark's twin %q reads as in progress", row, th.G.Partial)
	}
	if tw, _ := theme.ASCIITwin(theme.Ellipsis); !strings.HasPrefix(tw, asciiRune('…')) {
		t.Errorf("the sweep's ellipsis %q is not a cell of the theme's twin %q", asciiRune('…'), tw)
	}
}

// Text is not a glyph: a reading's unit (21.5°C), a product (3×4), an accented name.
// The sweep mapped every rune outside its twin tables to "?", so a unit read as a
// question under -ascii and F-ASCII could not see it ("?" is ASCII). Latin-1 text
// takes a one-cell stand-in; a rune with no one-cell twin stays "?", which tuilint
// frames --ascii reports against the unicode frame.
func TestASCIIfyKeepsTextRunes(t *testing.T) {
	th := theme.Resolve(theme.Default(), true, true)
	for _, c := range []struct{ in, want string }{
		{"cpu 61.5°C", "cpu 61.5 C"},
		{"3×4 ± 2 ≥ 1 ≤ 9", "3x4 + 2 > 1 < 9"},
		{"Zoë Ångström — naïve café, Ø 12 µm ² ©2026 £5 €3", "Zoe Angstrom - naive cafe, O 12 um 2 c2026 L5 E3"},
		{"½ cup", "? cup"}, // no one-cell twin: loud, for the lint
	} {
		cv := NewCanvas(th, 60, 1)
		cv.Draw(c.in, Rect(0, 0, 60, 1))
		cv.ASCIIfy()
		if got := strings.TrimRight(ansi.Strip(cv.String()), " "); got != c.want {
			t.Errorf("ASCIIfy(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
