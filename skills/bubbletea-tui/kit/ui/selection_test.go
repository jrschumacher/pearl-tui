package ui

import (
	"fmt"
	"image/color"
	"path/filepath"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
	uv "github.com/charmbracelet/ultraviolet"

	"tuikit/theme"
)

// A filled selection runs the whole row, through a sparkline or a trace (#60). The bars
// drew in the row's ink under reverse video, the ink a filled row's text takes, which on
// most themes is the page's own colour: at 16 colours halyard's ▆▇█ were black on a black
// page, holes punched through the bar (a cold monitor's hist column had the same). Every
// cell of a selected row carries the fill, and a bar on it never reads as the page; under
// NO_COLOR, where the only ink on the fill is the page's, the row draws no bars.
func TestSelectionFillRunsThroughATrace(t *testing.T) {
	vals := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	for _, path := range testThemes(t) {
		f, err := theme.Load(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, sel := range []string{"fill", "inverse"} {
			f.Expressions.Selection = sel
			for _, dark := range []bool{true, false} {
				for _, p := range []colorprofile.Profile{colorprofile.TrueColor, colorprofile.ANSI, colorprofile.Ascii} {
					th := theme.ResolveProfile(f, dark, false, p)
					rows := [][]string{
						{"checkout", Sparkline(th, vals, 10, th.S.Muted), Trace(th, []float64{0, 2, 40, 90, 100, 100}, 6, TraceOpts{Warn: 75, Fail: 90})[0]},
						{"ledger", Sparkline(th, vals, 10, th.S.Muted), Trace(th, []float64{2, 3}, 6, TraceOpts{})[0]},
					}
					name := fmt.Sprintf("%s %s dark=%v %v", filepath.Base(path), sel, dark, p)
					for _, focused := range []bool{true, false} {
						out := Table(th, []string{"name", "p99", "cpu"}, rows, []int{10, 10, 6}, "lll", 0, 34, focused)
						row := cellsOf(downsample(out[2], p))
						if p == colorprofile.Ascii {
							// NO_COLOR: the fill is reverse video, whose one ink is the page, so
							// every cell is reversed and no bar glyph is drawn on it (a hole); a
							// blurred row (bgSubtle, which NO_COLOR has not) draws none either, so
							// the row reads the same under an overlay
							for x, c := range row {
								if focused && c.Style.Attrs&uv.AttrReverse == 0 || strings.ContainsAny(c.Content, theme.SparkRamp+theme.TraceFloor) {
									t.Errorf("%s: cell %d %q: under NO_COLOR every cell is reversed and none is a bar", name, x, c.Content)
									break
								}
							}
							continue
						}
						fill, ink := p.Convert(th.S.SelRow.GetForeground()), color.Color(nil)
						if !focused {
							fill = p.Convert(th.P.BgSubtle)
						}
						page := p.Convert(th.P.Bg)
						for x, c := range row {
							bg, fg := c.Style.Bg, c.Style.Fg
							if c.Style.Attrs&uv.AttrReverse != 0 {
								bg, fg = fg, bg
								if fg == nil {
									fg = page // reverse of the terminal's own background
								}
							}
							if !same8(bg, fill) {
								t.Errorf("%s focused=%v: cell %d %q has no selection fill (bg %v, want %v)", name, focused, x, c.Content, bg, fill)
								break
							}
							ink = fg
							if !focused || !strings.ContainsAny(c.Content, theme.SparkRamp+theme.TraceFloor) {
								continue
							}
							cr := theme.Contrast(ink, page)
							if p == colorprofile.ANSI {
								cr = theme.Contrast16(ink, page)
							}
							if cr < 1.5 {
								t.Errorf("%s: bar %q at cell %d draws %v on the fill, %.2f:1 on the page: a hole to the page", name, c.Content, x, ink, cr)
								break
							}
						}
					}
				}
			}
		}
	}
}

// cellsOf reads one rendered row back as the terminal holds it.
func cellsOf(s string) []*uv.Cell {
	w := lipgloss.Width(s)
	buf := uv.NewScreenBuffer(w, 1)
	uv.NewStyledString(s).Draw(buf, buf.Bounds())
	out := make([]*uv.Cell, 0, w)
	for x := range w {
		if c := buf.CellAt(x, 0); c != nil && c.Width > 0 {
			out = append(out, c)
		}
	}
	return out
}
