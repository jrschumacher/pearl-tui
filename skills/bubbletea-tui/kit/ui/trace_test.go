package ui

import (
	"fmt"
	"math"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"unicode"

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
		full, top := ramp[len(ramp)-1], ramp[len(ramp)-2] // a bar's top row stops at 7/8
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
		if up[0] != top || down[0] != full || up[1] != " " || down[1] != full || up[2] != " " || down[2] != floor {
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
	// TraceOpts.Ink's doc: a cell at or below Min is never asked about and draws the
	// floor in fgSubtle, whatever the caller's Ink would have said
	var asked []float64
	low := Trace(th, []float64{0, 50}, 2, TraceOpts{Ink: func(v float64) lipgloss.Style { asked = append(asked, v); return th.S.Err }})[0]
	if slices.Contains(asked, 0) {
		t.Errorf("Ink was asked about a cell at Min: %v", asked)
	}
	floor, _ := th.TraceGlyphs()
	if c := cellAt(low, floor); c == nil || !same8(c.Style.Fg, th.P.FgSubtle) {
		t.Errorf("a cell at Min with an Ink set is not the floor in fgSubtle: %v in %q", c, ansi.Strip(low))
	}
}

// An idle or near-idle history (a 0 % or 4–9 % trace, a constant or ±5 % sparkline, a
// quiet StateSparkline row) draws one glyph across the row, never the rule's glyph, and
// never a solid line: a run of ▁ is a hairline, and beside any rule (─ beside Kestrel's
// ▁▁▁▁, the · leaders beside a ▁ floor) or above the next row's bars a hairline reads as
// one more rule. It is dots: braille, or under ASCII . or ,. A 4–9 % band that alternated
// floor and bar (⣀▁⣀▁) read as a dash-dot rule. Under ASCII no trace glyph is "-", "_",
// "=" or "~" (a run of any is a rule) or the rule's twin.
func TestIdleHistoryIsDotsNeverALine(t *testing.T) {
	var files []theme.File
	for _, path := range append(testThemes(t), filepath.Join("..", "theme", "testdata", "quire.json")) {
		f, err := theme.Load(path)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, f)
	}
	for _, rule := range []string{"─", "━", "═", "┄", "╌", "┈", "·", "∙", "•", "⋯", "⣀", "⠤"} {
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
				"Trace 0 % between busy": string([]rune(ansi.Strip(TraceRow(th, append(make([]float64, 12), 50), 13, TraceOpts{})))[:12]),
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
				case line(g) || !dot(g):
					t.Errorf("%s: %s draws %q: an idle row is dots, never a line", name, what, g)
				}
			}
			floor, ramp := th.TraceGlyphs()
			if len(ramp) < 6 {
				t.Errorf("%s: %d heights, want 6 or more", name, len(ramp))
			}
			for _, g := range append([]string{floor}, ramp...) {
				// under ASCII, punctuation only: a letter in a run (il) reads as a word
				if g == th.G.Rule || g == th.G.HeaderRule || ascii && (line(g) || !isASCIIText(g) || !unicode.IsPunct([]rune(g)[0]) && !unicode.IsSymbol([]rune(g)[0])) {
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

// No data is not low data, in a Sparkline: a series of zeros there (a p99 with no
// requests) drew the same dotted floor as one at 0.01 %, a step of ink apart, so under
// NO_COLOR and in ASCII (..........) the two were one row. A Sparkline of zeros draws
// blank. A Trace is a fixed-scale reading, where a window of zeros is real (an idle
// queue, a healthy error rate): it draws its floor in fgSubtle, a step under a low
// reading's fgMuted, and a Trace's no data is NaN or no samples, which draw blank. A
// zero inside an active window draws the floor too (the time axis runs on).
func TestNoDataIsNotLowData(t *testing.T) {
	nan := math.NaN()
	for _, ascii := range []bool{false, true} {
		th := testTheme(t, ascii)
		floor, _ := th.TraceGlyphs()
		low := TraceRow(th, []float64{4, 7, 5, 9, 6, 4}, 6, TraceOpts{})
		for what, none := range map[string]string{
			"Sparkline of zeros":  Sparkline(th, make([]float64, 6), 6, th.S.Muted),
			"StateSparkline ok":   StateSparkline(th, "ok", make([]float64, 6), 6),
			"Trace of NaN":        TraceRow(th, []float64{nan, nan, nan, nan, nan, nan}, 6, TraceOpts{}),
			"Trace of no samples": TraceRow(th, nil, 6, TraceOpts{}),
		} {
			if got := ansi.Strip(none); strings.TrimSpace(got) != "" || lipgloss.Width(none) != 6 {
				t.Errorf("ascii=%v: %s draws %q, want 6 blank cells", ascii, what, got)
			}
		}
		for what, zeros := range map[string]string{
			"Trace of zeros":  TraceRow(th, make([]float64, 6), 6, TraceOpts{}),
			"Trace on 10..90": TraceRow(th, []float64{10, 3, 10, 10, 10, 10}, 6, TraceOpts{Min: 10, Max: 90}),
		} {
			if got := ansi.Strip(zeros); got != strings.Repeat(floor, 6) {
				t.Errorf("ascii=%v: %s draws %q, want its floor in every cell: a window of zeros is a reading", ascii, what, got)
			} else if c := cellAt(zeros, floor); c == nil || !same8(c.Style.Fg, th.P.FgSubtle) {
				t.Errorf("ascii=%v: %s is not in fgSubtle: %v", ascii, what, c)
			}
		}
		if strings.TrimSpace(ansi.Strip(low)) == "" {
			t.Errorf("ascii=%v: a low trace draws nothing", ascii)
		}
		if c := cellAt(low, string([]rune(ansi.Strip(low))[0])); c == nil || !same8(c.Style.Fg, th.P.FgMuted) {
			t.Errorf("ascii=%v: a low trace is not in fgMuted: %v", ascii, c)
		}
		mixed := TraceRow(th, []float64{0, 0, 50}, 3, TraceOpts{})
		if got := []rune(ansi.Strip(mixed)); string(got[0]) != floor {
			t.Errorf("ascii=%v: a zero beside activity draws %q, want the floor", ascii, string(got[0]))
		}
	}
}

// A full-height bar filled its cell to the top, so the next row's floor or bar sat
// flush on it, and under NO_COLOR the two rows fused into one shape. A bar's top row
// stops at 7/8 (▇): a one-row Trace and a Sparkline never draw the ramp's top glyph,
// and a taller Trace draws it only below its top row, where the bar runs on.
func TestBarsStopShortOfTheRowAbove(t *testing.T) {
	for _, ascii := range []bool{false, true} {
		th := testTheme(t, ascii)
		_, ramp := th.TraceGlyphs()
		full := ramp[len(ramp)-1]
		for what, row := range map[string]string{
			"TraceRow 100":     TraceRow(th, []float64{100, 50, 100}, 3, TraceOpts{}),
			"Trace 2 rows top": Trace(th, []float64{100, 50, 100}, 3, TraceOpts{Rows: 2})[0],
			"Sparkline":        Sparkline(th, []float64{1, 5, 10, 2}, 4, th.S.Muted),
			"StateSparkline":   StateSparkline(th, "warning", []float64{1, 5, 10, 2}, 6),
		} {
			if strings.Contains(ansi.Strip(row), full) {
				t.Errorf("ascii=%v: %s draws %q: a top row stops at 7/8", ascii, what, ansi.Strip(row))
			}
		}
		if got := ansi.Strip(Trace(th, []float64{100}, 1, TraceOpts{Rows: 2})[1]); got != full {
			t.Errorf("ascii=%v: below its top row a full bar draws %q, want %q", ascii, got, full)
		}
	}
}

// Sparkline's height grows with the series' spread one level at a time: it capped a
// spread under a third at ▄, then jumped to full height at exactly a third, so a 33 %
// change drew twice as tall as a 32 % one.
func TestSparklineRisesWithoutACliff(t *testing.T) {
	for _, ascii := range []bool{false, true} {
		th := testTheme(t, ascii)
		_, ramp := th.TraceGlyphs()
		level := func(s string) int {
			hi := -1
			for _, r := range ansi.Strip(s) {
				for i, g := range ramp {
					if string(r) == g && i > hi {
						hi = i
					}
				}
			}
			return hi
		}
		prev := -1
		for pct := 10; pct <= 100; pct++ {
			lo := 100 - float64(pct) // spread (hi-lo)/hi = pct %
			got := level(Sparkline(th, []float64{lo, lo, 100, 100}, 4, th.S.Muted))
			if prev >= 0 && got > prev+1 {
				t.Errorf("ascii=%v: a spread of %d %% draws level %d, %d %% drew %d: a cliff", ascii, pct, got, pct-1, prev)
			}
			if got < prev {
				t.Errorf("ascii=%v: a spread of %d %% draws lower (%d) than %d %% (%d)", ascii, pct, got, pct-1, prev)
			}
			prev = got
		}
		if prev != len(ramp)-2 {
			t.Errorf("ascii=%v: a 100 %% spread reaches level %d, want %d (7/8)", ascii, prev, len(ramp)-2)
		}
	}
}

// A Sparkline drew every series within 25 % of its peak as the idle floor, so a +30 %
// p99 regression read the same as a constant one (#63 review). Under 10 % is noise and
// draws the floor; a +20 % step draws a step; a 30 % one never jumps to full height.
func TestSparklineShowsASmallStep(t *testing.T) {
	for _, ascii := range []bool{false, true} {
		th := testTheme(t, ascii)
		floor, ramp := th.TraceGlyphs()
		distinct := func(s string) int {
			seen := map[rune]bool{}
			for _, r := range ansi.Strip(s) {
				seen[r] = true
			}
			return len(seen)
		}
		for _, c := range []struct {
			name string
			vals []float64
		}{
			{"p99 100→120 ms", []float64{100, 100, 100, 100, 120, 120, 120, 120}},
			{"rps 1000→800", []float64{1000, 1000, 1000, 1000, 800, 800, 800, 800}},
			{"p99 400→520 ms", []float64{400, 400, 400, 400, 520, 520, 520, 520}},
		} {
			got := Sparkline(th, c.vals, 8, th.S.Muted)
			if distinct(got) < 2 {
				t.Errorf("ascii=%v: %s draws %q, one glyph: a step of 20-30 %% must show", ascii, c.name, ansi.Strip(got))
			}
			if strings.Contains(ansi.Strip(got), ramp[len(ramp)-1]) {
				t.Errorf("ascii=%v: %s draws %q: a 20-30 %% step is not full height", ascii, c.name, ansi.Strip(got))
			}
		}
		if got := ansi.Strip(Sparkline(th, []float64{100, 104, 98, 102, 97, 103}, 6, th.S.Muted)); got != strings.Repeat(floor, 6) {
			t.Errorf("ascii=%v: ±5 %% noise draws %q, want the floor", ascii, got)
		}
	}
}
