package ui

import (
	"math"
	"strings"

	"charm.land/lipgloss/v2"

	"tuikit/theme"
)

// TraceOpts is how Trace draws a history. The zero value is a 0..100 scale, one sample a
// cell, one row, in fgMuted.
type TraceOpts struct {
	// Min and Max are the fixed scale: Min is the floor and Max fills the column; samples
	// outside clamp. Fixed, so an idle 3 % stays at the floor instead of filling the row
	// as a series scaled to itself does (Sparkline). Max ≤ Min, the zero value, is the
	// percent scale 0..100: set Max for anything else (bytes, a rate), or every sample
	// over 100 draws full.
	Min, Max float64
	// Per is how many samples a cell holds, drawn as their maximum, so a one-sample spike
	// survives the downsample: 60 samples at Per 4 fill 15 cells. Default 1.
	Per int
	// Rows is the height, top row first: Rows×8 heights. The bottom row holds the floor;
	// the rows above hold only what rises into them. Default 1.
	Rows int
	// Warn and Fail are levels on the scale: a cell at or over Warn draws in the warning's
	// ink, at or over Fail in the error's, so a spike keeps its colour after it has
	// scrolled left. 0 leaves a level off. Where the theme draws the warning as a fill
	// (yellow 3 on a white page at 16 colours) the bars stay quiet; the mark beside the
	// value carries it (ui.Mark).
	Warn, Fail float64
	// Ink, when set, is the ink of a cell by its value, in place of fgMuted and the levels,
	// for every cell above Min. It is never called for a cell at or below Min: that cell
	// draws the floor in fgSubtle whatever Ink would say, so an Ink that inks zeros (a red
	// "down" at 0) has no effect there; mark that state beside the trace instead.
	Ink func(v float64) lipgloss.Style
}

// Trace draws a history on a fixed scale in w cells, newest sample at the right edge, and
// returns opts.Rows rows (top first); TraceRow is the one-row case. A cell before the
// first sample is blank (a history that hasn't started is empty, not zero), and so is a
// NaN: pass NaN (or no samples) for no data, never 0. A history of zeros is real (an
// idle queue, a healthy error rate) and draws its floor. A cell under 1/8 of the scale is
// idle and draws the floor (Theme.TraceGlyphs: ⣀; ASCII . or ,), one texture for the
// whole idle band, so an idle 4–9 % never flickers between floor and bar and never reads
// as a section rule; a cell at Min itself draws it in fgSubtle, a step under a low
// reading's fgMuted. Above it the cell rises through ▁▂▃▄▅▆▇█ (ASCII .,:;%&#@),
// and the top row stops at ▇ (7/8): a full block would sit flush on the row above. On a
// selected row ui.Table draws the bars in the selection's shade (Styles.SelBars).
func Trace(t *theme.Theme, vals []float64, w int, o TraceOpts) []string {
	per, rows := max(1, o.Per), max(1, o.Rows)
	lo, hi := o.Min, o.Max
	if hi <= lo {
		lo, hi = 0, 100
	}
	_, ramp := t.TraceGlyphs()
	steps := rows * len(ramp)
	ink := o.Ink
	if ink == nil {
		ink = traceInk(t, o.Warn, o.Fail)
	}
	cells := traceCells(vals, w, per)
	levels := make([]int, len(cells))
	inks := make([]lipgloss.Style, len(cells))
	for i, v := range cells {
		if math.IsNaN(v) {
			levels[i] = -1
			continue
		}
		frac := math.Min(1, math.Max(0, (v-lo)/(hi-lo)))
		levels[i] = 0
		if frac*8 >= 1 {
			levels[i] = min(steps-1, max(1, int(math.Round(frac*float64(steps))))) // the top row stops at 7/8
		}
		if v <= lo {
			inks[i] = t.S.Subtle // nothing at all: the floor a step quieter than a low reading
		} else {
			inks[i] = ink(v) // never asked about a cell at or below Min (TraceOpts.Ink)
		}
	}
	return traceRows(t, levels, inks, rows)
}

// TraceRow is Trace one row tall: the history as one string, for a table cell.
func TraceRow(t *theme.Theme, vals []float64, w int, o TraceOpts) string {
	o.Rows = 1
	return Trace(t, vals, w, o)[0]
}

// traceInk is Trace's default ink: fgMuted, the warning's at or over warn and the error's
// at or over fail (0: off).
func traceInk(t *theme.Theme, warn, fail float64) func(float64) lipgloss.Style {
	return func(v float64) lipgloss.Style {
		switch {
		case fail > 0 && v >= fail:
			return t.S.Err
		case warn > 0 && v >= warn && !warnIsFill(t):
			return t.S.Warn
		}
		return t.S.Muted
	}
}

// traceCells groups the newest w*per samples into w cells, each the max of its group,
// NaN for a cell with no sample yet, the newest cell last.
func traceCells(vals []float64, w, per int) []float64 {
	out := make([]float64, max(0, w))
	for i := range out {
		out[i] = math.NaN()
	}
	n := len(vals)
	for c := range out {
		end := n - c*per // the group ends here (exclusive), counting back from the newest
		if end <= 0 {
			break
		}
		m := math.NaN()
		for _, v := range vals[max(0, end-per):end] {
			if !math.IsNaN(v) && (math.IsNaN(m) || v > m) {
				m = v
			}
		}
		out[len(out)-1-c] = m
	}
	return out
}

// traceRows draws levels (-1 blank, 0 the floor, 1..rows×len(ramp) a height) rows tall,
// each cell in its ink, top row first.
func traceRows(t *theme.Theme, levels []int, inks []lipgloss.Style, rows int) []string {
	floor, ramp := t.TraceGlyphs()
	n := len(ramp)
	out := make([]string, rows)
	for r := range rows {
		b := rows - 1 - r // 0 = the bottom row
		var sb, run strings.Builder
		key, ink := "", lipgloss.NewStyle()
		flush := func() {
			if run.Len() > 0 {
				sb.WriteString(ink.Render(run.String()))
				run.Reset()
			}
		}
		for i, lvl := range levels {
			g := " "
			switch part := lvl - b*n; {
			case lvl < 0:
			case lvl == 0:
				if b == 0 {
					g = floor
				}
			case part >= n:
				g = ramp[n-1]
			case part > 0:
				g = ramp[part-1]
			}
			if g == " " {
				flush()
				sb.WriteByte(' ')
				continue
			}
			// one escape per run of one ink, not per cell
			if k := inks[i].Render("x"); k != key {
				flush()
				key, ink = k, inks[i]
			}
			run.WriteString(g)
		}
		flush()
		out[r] = sb.String()
	}
	return out
}
