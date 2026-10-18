package ui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"tuikit/theme/bubbletea"
)

// The theme's ellipsis resolves to its twin ("...") before a screen draws, but a cut
// made with the bare theme.Ellipsis (ui.Fit, a screen's own ansi.Truncate) reaches the
// canvas as "…", and the sweep drew it as "~": the partial mark's twin, so a cut line
// read as a row in progress ("unknown key~"). Then as one ".", which reads as a full
// stop ("unknown key."). The sweep finishes the cut as the theme's twin: the "…" and
// up to two letters or digits before it draw "...", never a border, a space or a mark
// before the cut, and a "…" with text right after it is not a cut.
func TestASCIIfyCutIsNotThePartialTwin(t *testing.T) {
	th := theme.Resolve(theme.Default(), true, true)
	c := NewCanvas(th, 16, 1)
	c.Draw(Fit("unknown key value", 12), Rect(0, 0, 12, 1))
	c.ASCIIfy()
	row := ansi.Strip(c.String())
	if !strings.HasPrefix(row, "unknown k... ") {
		t.Errorf("a Fit cut under ASCII draws %q, want \"unknown k...\"", row)
	}
	if strings.ContainsAny(row, th.G.Partial+"~") {
		t.Errorf("a Fit cut under ASCII draws %q: the partial mark's twin %q reads as in progress", row, th.G.Partial)
	}
	edge := th.S.Subtle.Render
	for _, d := range []struct{ in, want string }{
		{edge("|") + Fit("abcdef", 1) + edge("|"), "|.|"},                       // a one-cell cut beside a border: the border stays
		{edge("+") + Fit("abcdef", 2) + edge("+"), "+..+"},                      // the corner too
		{"|" + Fit("abcdef", 1) + "|", "|.|"},                                   // an unstyled border is not a letter
		{th.S.Muted.Render("> ") + Fit(th.S.Base.Render("abcdef"), 3), "> ..."}, // the prompt is not the text
		{Fit("日本語です", 5), "? ? ."},                                              // a wide cell's stand-in is not a letter
		{Fit("short", 8), "short"},                                              // no cut, no dots
		// text before the cut keeps its marks: only letters and digits turn to dots
		{"key: " + Fit("value", 1), "key: ."},
		{"v2." + Fit("4.1 rolled", 1), "v2.."},
		{th.S.Muted.Render("12") + th.S.Muted.Render(Fit(" more lines", 2)), "12 ."},
		{"[" + Fit("abcdef", 2) + "]", "[a.]"}, // the cut does not end its run: one dot
		{"a…b", "a.b"},                         // a typed ellipsis mid-text: one dot
		// the ambiguity left: a typed ellipsis at the end of a run reads as a cut
		{"Loading…", "Loadi..."},
	} {
		cv := NewCanvas(th, 12, 1)
		cv.Draw(d.in, Rect(0, 0, 12, 1))
		cv.ASCIIfy()
		if got := strings.TrimRight(ansi.Strip(cv.String()), " "); got != d.want {
			t.Errorf("ASCIIfy(%q) = %q, want %q", ansi.Strip(d.in), got, d.want)
		}
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
		// a zero-width rune rides in the cell before it, which the sweep passed over when
		// its first rune was ASCII: a soft hyphen and NFD text (e + U+0308, as macOS file
		// names and git print them) leaked into the frame; they fold into the letter
		{"soft\u00ADhyphen", "softhyphen"},
		{"Zoe\u0308 cafe\u0301 n\u0303\u0301", "Zoe cafe n"},
	} {
		cv := NewCanvas(th, 60, 1)
		cv.Draw(c.in, Rect(0, 0, 60, 1))
		cv.ASCIIfy()
		if got := strings.TrimRight(ansi.Strip(cv.String()), " "); got != c.want {
			t.Errorf("ASCIIfy(%q) = %q, want %q", c.in, got, c.want)
		}
		for _, r := range cv.String() {
			if r >= 128 {
				t.Errorf("ASCIIfy(%q) leaks %U", c.in, r)
			}
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

// ExpandTabs' doc tells apps why to expand before styling or measuring: lipgloss's
// Style.Render turns a tab into 4 spaces wherever it sits, not the 8-column stop a
// terminal advances to, and lipgloss.Width counts a raw tab as 0. This pins both claims
// to the pinned lipgloss, so the doc goes stale only with a failing test.
func TestLipglossTabIsNotAStop(t *testing.T) {
	if got := ansi.Strip(lipgloss.NewStyle().Bold(true).Render("ab\tc")); got != "ab    c" {
		t.Errorf("lipgloss renders \"ab\\tc\" as %q; ExpandTabs' doc says 4 spaces (\"ab    c\")", got)
	}
	if w := lipgloss.Width("ab\tc"); w != 3 {
		t.Errorf("lipgloss.Width(\"ab\\tc\") = %d; ExpandTabs' doc says a tab measures 0", w)
	}
	if got := lipgloss.NewStyle().Bold(true).Render(ExpandTabs("ab\tc")); ansi.Strip(got) != "ab      c" {
		t.Errorf("expanded first, the tab is %q, want its 8-column stop", ansi.Strip(got))
	}
}
