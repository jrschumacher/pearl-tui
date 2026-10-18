package ui

import (
	"fmt"
	"math"
	"path/filepath"
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

// An idle or near-idle history (a 0 % or 4–9 % trace, a constant or ±5 % sparkline, a
// quiet StateSparkline row) draws one glyph across the row, never the rule's glyph, and
// never the rule's texture: beside a line rule (─ ━ ═, and dashes ┄ ┈ ╌, which render
// near solid) it is dots, beside a dotted rule (· ∙ • …) it is a solid ▁. Both cold
// monitors' idle traces read as rules (▁▁▁▁ beside ─, ⣀⣀⣀⣀ beside ·), and a 4–9 % band
// that alternated floor and bar (⣀▁⣀▁) read as a dash-dot rule. Under ASCII no trace glyph
// is "-", "_", "=" or "~" (a run of any is a rule) or the rule's twin; ASCII has no solid
// low glyph that isn't a rule, so beside a "." rule the floor differs in glyph only (",").
func TestIdleHistoryIsOneTextureNotTheRules(t *testing.T) {
	var files []theme.File
	for _, path := range append(testThemes(t), filepath.Join("..", "theme", "testdata", "quire.json")) {
		f, err := theme.Load(path)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, f)
	}
	for _, rule := range []string{"─", "━", "═", "┄", "╌", "┈", "·", "∙", "•", "⋯"} {
		f := theme.Default()
		f.Name, f.Glyphs.Rule = "rule "+rule, rule
		files = append(files, f)
	}
	line := func(g string) bool {
		r := []rune(g)[0]
		return r >= 0x2500 && r <= 0x259F || strings.ContainsRune("-_=~", r)
	}
	dot := func(g string) bool {
		r := []rune(g)[0]
		return r >= 0x2800 && r <= 0x28FF || strings.ContainsRune("·∙•⋅…‥⋯.,:'", r)
	}
	low := []float64{4, 7, 5, 9, 6, 4, 8, 5, 9, 7, 4, 6}
	noise := []float64{100, 104, 98, 102, 97, 103, 100, 99, 105, 101, 98, 102}
	for _, f := range files {
		for _, ascii := range []bool{false, true} {
			th := theme.Resolve(f, true, ascii)
			name := fmt.Sprintf("%s ascii=%v (rule %q)", f.Name, ascii, th.G.Rule)
			rows := map[string]string{
				"Trace 0 %":              TraceRow(th, make([]float64, 12), 12, TraceOpts{}),
				"Trace 4-9 %":            TraceRow(th, low, 12, TraceOpts{}),
				"Trace 4-9 % three rows": Trace(th, low, 12, TraceOpts{Rows: 3})[2],
				"Sparkline constant":     Sparkline(th, []float64{40, 40, 40, 40, 40, 40}, 6, th.S.Muted),
				"Sparkline ±5 %":         Sparkline(th, noise, 12, th.S.Muted),
				"StateSparkline ok ±5 %": string([]rune(ansi.Strip(StateSparkline(th, "ok", noise, 12)))[:10]),
			}
			for what, row := range rows {
				cells := []rune(strings.TrimRight(ansi.Strip(row), " "))
				if len(cells) == 0 {
					t.Errorf("%s: %s draws nothing", name, what)
					continue
				}
				g := string(cells[0])
				if strings.Count(string(cells), g) != len(cells) {
					t.Errorf("%s: %s draws %q, more than one texture", name, what, string(cells))
					continue
				}
				switch {
				case g == th.G.Rule || g == th.G.HeaderRule:
					t.Errorf("%s: %s draws the rule's glyph %q", name, what, g)
				case ascii && line(g):
					t.Errorf("%s: %s draws %q, an ASCII rule", name, what, g)
				case !ascii && line(th.G.Rule) && !dot(g):
					t.Errorf("%s: %s draws %q beside a line rule: an idle row in dots", name, what, g)
				case !ascii && dot(th.G.Rule) && dot(g):
					t.Errorf("%s: %s draws %q beside a dotted rule: an idle row solid", name, what, g)
				}
			}
			floor, ramp := th.TraceGlyphs()
			if len(ramp) < 6 {
				t.Errorf("%s: %d heights, want 6 or more", name, len(ramp))
			}
			for _, g := range append([]string{floor}, ramp...) {
				if g == th.G.Rule || g == th.G.HeaderRule || ascii && (line(g) || !isASCIIText(g)) {
					t.Errorf("%s: a trace may draw %q", name, g)
				}
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
