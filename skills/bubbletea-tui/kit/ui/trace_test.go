package ui

import (
	"math"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"tuikit/theme"
)

// Both cold monitors (#61) wrote their own trace because Sparkline scales a series to
// itself: an idle 3–6 % filled the row and read as an incident. Trace holds a fixed
// scale, keeps a spike through the downsample, leaves the time before the first sample
// blank, inks a cell by its level and grows rows tall.
func TestTraceHoldsItsScale(t *testing.T) {
	for _, ascii := range []bool{false, true} {
		th := testTheme(t, ascii)
		floor, ramp := th.TraceGlyphs()
		top := ramp[len(ramp)-1]
		cells := func(s string) []string {
			var out []string
			for _, r := range ansi.Strip(s) {
				out = append(out, string(r))
			}
			return out
		}

		// idle on 0..100: the floor, never a height
		idle := cells(Trace(th, []float64{3, 5, 2, 4, 6, 3}, 6, TraceOpts{})[0])
		for i, c := range idle {
			if c != floor {
				t.Errorf("ascii=%v: idle cell %d draws %q, want the floor %q: a fixed scale keeps 3-6 %% low", ascii, i, c, floor)
			}
		}
		// the same series scaled to itself is what Sparkline draws: a full-height bar
		if got := ansi.Strip(Sparkline(th, []float64{3, 5, 2, 4, 6, 3}, 6, th.S.Muted)); !strings.Contains(got, top) {
			t.Errorf("ascii=%v: Sparkline %q should autoscale (the reason Trace exists)", ascii, got)
		}

		// a one-sample spike survives four samples a cell; cells before the first sample are blank
		vals := []float64{0, 0, 100, 0, 0, 0, 0, 0}
		got := cells(Trace(th, vals, 5, TraceOpts{Per: 4})[0])
		want := []string{" ", " ", " ", top, floor}
		if strings.Join(got, "") != strings.Join(want, "") {
			t.Errorf("ascii=%v: Per 4 draws %q, want %q: blank before the first sample, the spike kept as the max", ascii, strings.Join(got, ""), strings.Join(want, ""))
		}

		// a range other than 0..100: 50 of 0..50 is full
		if c := cells(Trace(th, []float64{50}, 1, TraceOpts{Max: 50})[0]); c[0] != top {
			t.Errorf("ascii=%v: 50 on 0..50 draws %q, want %q", ascii, c[0], top)
		}
		// NaN is a missing sample: blank
		if c := cells(Trace(th, []float64{math.NaN()}, 1, TraceOpts{})[0]); c[0] != " " {
			t.Errorf("ascii=%v: a NaN sample draws %q, want blank", ascii, c[0])
		}

		// two rows: 100 fills both, 50 fills the bottom, idle draws the floor only at the bottom
		rows := Trace(th, []float64{100, 50, 1}, 3, TraceOpts{Rows: 2})
		if len(rows) != 2 {
			t.Fatalf("ascii=%v: Rows 2 returned %d rows", ascii, len(rows))
		}
		up, down := cells(rows[0]), cells(rows[1])
		if up[0] != top || down[0] != top || up[1] != " " || down[1] != top || up[2] != " " || down[2] != floor {
			t.Errorf("ascii=%v: two rows of 100, 50, 1 draw\n%q\n%q", ascii, strings.Join(up, ""), strings.Join(down, ""))
		}
		for _, r := range rows {
			if lipgloss.Width(r) != 3 {
				t.Errorf("ascii=%v: row %q is %d cells, want 3", ascii, ansi.Strip(r), lipgloss.Width(r))
			}
		}
	}
}

// A cell at or over Warn takes the warning's ink and at or over Fail the error's, so a
// spike keeps its colour after it scrolls left.
func TestTraceInksByLevel(t *testing.T) {
	th := testTheme(t, false)
	out := Trace(th, []float64{10, 80, 95}, 3, TraceOpts{Warn: 75, Fail: 90})[0]
	for i, want := range []lipgloss.Style{th.S.Muted, th.S.Warn, th.S.Err} {
		if !strings.Contains(out, want.Render(string([]rune(ansi.Strip(out))[i]))) {
			t.Errorf("cell %d of %q is not drawn in its level's ink", i, out)
		}
	}
	ink := Trace(th, []float64{10}, 1, TraceOpts{Ink: func(float64) lipgloss.Style { return th.S.Accent }})[0]
	if !strings.Contains(ink, th.S.Accent.Render(ansi.Strip(ink))) {
		t.Errorf("Ink %q is not the caller's", ink)
	}
}

// Both cold monitors' idle traces read as rules: Kestrel's ▁▁▁▁ beside a ─ rule, Thrift's
// ⣀⣀⣀⣀ beside a · leader. No glyph a trace draws is the rule's (or its twin), and the floor
// takes the other texture: dots beside a solid rule, solid beside a dotted one.
func TestTraceNeverDrawsTheRule(t *testing.T) {
	var files []theme.File
	for _, path := range testThemes(t) {
		f, err := theme.Load(path)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, f)
	}
	dotted := theme.Default()
	dotted.Name, dotted.Glyphs.Rule = "dotted", "·"
	heavy := theme.Default()
	heavy.Name, heavy.Glyphs.Rule = "heavy", "━"
	files = append(files, dotted, heavy)
	for _, f := range files {
		for _, ascii := range []bool{false, true} {
			th := theme.Resolve(f, true, ascii)
			floor, ramp := th.TraceGlyphs()
			if len(ramp) < 6 {
				t.Errorf("%s ascii=%v: %d heights, want 6 or more", f.Name, ascii, len(ramp))
			}
			for _, g := range append([]string{floor}, ramp...) {
				if g == th.G.Rule || g == th.G.HeaderRule {
					t.Errorf("%s ascii=%v: a trace draws %q, the rule's glyph", f.Name, ascii, g)
				}
				if ascii && !isASCIIText(g) {
					t.Errorf("%s ascii=%v: a trace draws %q, not ASCII", f.Name, ascii, g)
				}
			}
			solid := floor == theme.TraceFloorSolid || floor == theme.TraceFloorSolidASCII
			if dottedRule := strings.ContainsAny(th.G.Rule, "·┄┈╌┅┉╍⋯…‥.:"); solid != dottedRule {
				t.Errorf("%s ascii=%v: floor %q beside rule %q: an idle trace takes the rule's texture", f.Name, ascii, floor, th.G.Rule)
			}
			// a constant series (an idle service) draws the floor in Sparkline too
			if got := ansi.Strip(Sparkline(th, []float64{4, 4, 4}, 3, th.S.Muted)); got != strings.Repeat(floor, 3) {
				t.Errorf("%s ascii=%v: a constant sparkline draws %q, want the floor %q", f.Name, ascii, got, floor)
			}
		}
	}
}

func isASCIIText(s string) bool {
	for _, r := range s {
		if r > 126 || r < 32 {
			return false
		}
	}
	return true
}
