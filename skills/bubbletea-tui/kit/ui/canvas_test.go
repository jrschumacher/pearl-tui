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
