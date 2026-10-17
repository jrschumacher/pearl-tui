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
		{"3×4 ÷ 2", "3x4 / 2"},
		{"Zoë Ångström — naïve café, Ø 12 µm ©2026 £5 €3", "Zoe Angstrom - naive cafe, O 12 um c2026 L5 E3"},
		{"½ cup", "? cup"}, // no one-cell twin: loud, for the lint
		// a one-cell twin that changes the value is no twin: ±0.5 is not +0.5, x ≥ 1 not
		// x > 1, 5‰ not 5%, 10³ not 103, ¬a not -a; they stay "?" for the lint, as does ¿,
		// which F-ASCII --unicode would otherwise report as a "?" with no twin
		{"±0.5 x≥1 y≤9 5‰ 10¹²³ ¬a † ‡ § ¶ ¿", "?0.5 x?1 y?9 5? 10??? ?a ? ? ? ? ?"},
	} {
		cv := NewCanvas(th, 60, 1)
		cv.Draw(c.in, Rect(0, 0, 60, 1))
		cv.ASCIIfy()
		if got := strings.TrimRight(ansi.Strip(cv.String()), " "); got != c.want {
			t.Errorf("ASCIIfy(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// A tab has no width to lipgloss (it measures 0) and a terminal advances it to the next
// 8-column stop, so a go test line ("ok  \tpkg\t0.412s") fitted to a panel was padded
// short and tore the panel's right edge where it was printed. The kit's fit and draw
// path expands a tab to the next 8-column stop of its line before measuring, and no
// cell of a frame holds a raw tab (tuilint frames F-TAB).
func TestTabsExpandToStops(t *testing.T) {
	th := theme.Resolve(theme.Default(), true, false)
	for _, c := range []struct{ in, want string }{
		{"ok  \tpkg\t0.412s", "ok      pkg     0.412s"},
		{"\tx", "        x"},
		{"12345678\tx", "12345678        x"},
		{"a\tb\nc\td", "a       b\nc       d"},
		{th.S.OK.Render("ok") + "\tpkg", th.S.OK.Render("ok") + "      pkg"},
		{"no tab", "no tab"},
	} {
		if got := ExpandTabs(c.in); got != c.want {
			t.Errorf("ExpandTabs(%q) = %q, want %q", c.in, got, c.want)
		}
	}
	line := "ok  \tledgerline/internal/import\t0.412s"
	if got := Fit(line, 50); ansi.StringWidth(got) != 50 || strings.Contains(got, "\t") {
		t.Errorf("Fit(%q, 50) = %q: %d cells, want 50 and no tab", line, got, ansi.StringWidth(got))
	}
	if got := Trunc(th, line, 20); ansi.StringWidth(got) != 20 || strings.Contains(got, "\t") {
		t.Errorf("Trunc(%q, 20) = %q: %d cells, want 20 and no tab", line, got, ansi.StringWidth(got))
	}
	for _, row := range ToolCall(th, Tool{Name: "Test", Param: "./...", State: ToolDone, Body: []string{line}}, 60, false) {
		if w := ansi.StringWidth(row); w != 60 || strings.Contains(row, "\t") {
			t.Errorf("a tool body with a tab: row %q is %d cells, want 60 and no tab", ansi.Strip(row), w)
		}
	}
	cv := NewCanvas(th, 30, 1)
	cv.Draw("ok  \tpkg\t0.4s", Rect(0, 0, 30, 1))
	if got := strings.TrimRight(ansi.Strip(cv.String()), " "); got != "ok      pkg     0.4s" {
		t.Errorf("Canvas.Draw of a tab draws %q, want the tab to its 8-column stop", got)
	}
}
