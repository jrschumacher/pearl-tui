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

	"tuikit/theme/bubbletea"
)

// A filled selection runs the whole row, through a sparkline or a trace (#60). The bars
// drew in the row's ink under reverse video, the ink a filled row's text takes, which on
// most themes is the page's own colour: at 16 colours halyard's ▆▇█ were black on a black
// page, holes punched through the bar (a cold monitor's hist column had the same). Every
// cell of a selected row carries the fill, and a bar on it never reads as the page; under
// NO_COLOR, where the only ink on the fill is the page's, every cell is reversed and the
// bars stay as marks on the fill.
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
							// NO_COLOR: the fill is reverse video, whose one ink is the page's. Every
							// cell of a focused row is reversed, so the fill is unbroken, and the bars
							// stay as marks on it: the selected row's history is the one being read
							bars := 0
							for x, c := range row {
								if focused && c.Style.Attrs&uv.AttrReverse == 0 {
									t.Errorf("%s: cell %d %q is not reversed: a hole in the NO_COLOR fill", name, x, c.Content)
									break
								}
								if strings.ContainsAny(c.Content, theme.SparkRamp+theme.TraceFloor) {
									bars++
								}
							}
							if bars < 10 {
								t.Errorf("%s focused=%v: %d bar cells under NO_COLOR, want the history kept", name, focused, bars)
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

// A filled row shades every block and braille cell, the app's own history glyphs too
// (▀ ▓ ⣿), so none is a hole; the theme's own marks among them (cartridge's ▖ spinner
// frame) are the row's text and keep its ink.
func TestSelectionShadesBarsNotMarks(t *testing.T) {
	var f theme.File
	for _, path := range testThemes(t) {
		if filepath.Base(path) == "cartridge.json" {
			var err error
			if f, err = theme.Load(path); err != nil {
				t.Fatal(err)
			}
		}
	}
	if f.Name == "" {
		t.Skip("cartridge.json is the kit's theme with a block spinner; a copied kit may not have it")
	}
	f.Expressions.Selection = "fill"
	th := theme.Resolve(f, true, false)
	spin := th.G.Spinner[0]
	rows := [][]string{{spin + " deploying", "▀▓⣿" + Sparkline(th, []float64{1, 5, 9}, 3, th.S.Muted)}}
	row := cellsOf(Table(th, []string{"name", "hist"}, rows, []int{12, 0}, "ll", 0, 30, true)[2])
	ink := func(c *uv.Cell) color.Color { return c.Style.Bg } // reverse video: the attribute bg is the glyph's ink
	rowInk := th.S.SelRow.GetBackground()
	for _, c := range row {
		switch {
		case c.Content == spin:
			if !same8(ink(c), rowInk) {
				t.Errorf("the spinner frame %q takes %v, want the row's ink %v", spin, ink(c), rowInk)
			}
		case strings.ContainsAny(c.Content, "▀▓⣿"+theme.SparkRamp):
			if same8(ink(c), rowInk) {
				t.Errorf("the bar %q is in the row's ink, a hole to the page: want the selection's shade", c.Content)
			}
		}
	}
}

// A list blurred under a fill or inverse selection keeps its selected row as a band, and
// every cell of it carries an ink that reads on the band: stripped to no ink, the row drew
// in the terminal's default (a steps pane blurred beside a log), and its subtle time on
// the band fell under 2.3:1.
func TestBlurredFilledRowIsInked(t *testing.T) {
	items := []Item{{Title: "go build", Right: "38s", State: "ok"}, {Title: "watch error rate", Right: "5m0s", State: "error"}}
	for _, f := range testThemes(t) {
		tf, err := theme.Load(f)
		if err != nil {
			t.Fatal(err)
		}
		th := theme.Resolve(tf, true, false)
		if th.X.Selection != "fill" && th.X.Selection != "inverse" {
			continue
		}
		row := List(th, items, 1, 0, 30, 2, false)[1]
		for x, c := range cellsOf(row) {
			if strings.TrimSpace(c.Content) == "" {
				continue
			}
			if c.Style.Fg == nil {
				t.Errorf("%s: the blurred row's %q at %d has no ink", filepath.Base(f), c.Content, x)
			} else if r := theme.Contrast(c.Style.Fg, th.P.BgSubtle); r < 2.3 {
				t.Errorf("%s: the blurred row's %q at %d is %.2f:1 on its band", filepath.Base(f), c.Content, x, r)
			}
		}
	}
}

// A scrollbar's track is a rail in the rule's ink, which tuilint passes as decoration
// only while it is one: an ASCII track that twins to no rail (ferrous's ░ is ".") is
// text there, and draws in an ink that reads on the page.
func TestASCIITrackReads(t *testing.T) {
	for _, f := range testThemes(t) {
		tf, err := theme.Load(f)
		if err != nil {
			t.Fatal(err)
		}
		th := theme.Resolve(tf, true, true)
		if th.G.Track == "|" || strings.TrimSpace(th.G.Track) == "" {
			continue
		}
		for i, s := range Scrollbar(th, 10, 40, 30) {
			c := cellsOf(s)[0]
			if c.Content == th.G.Track && (c.Style.Fg == nil || theme.Contrast(c.Style.Fg, th.P.Bg) < 2.3) {
				t.Errorf("%s: the ASCII track %q at row %d reads under 2.3:1 on the page", filepath.Base(f), c.Content, i)
			}
		}
	}
}
