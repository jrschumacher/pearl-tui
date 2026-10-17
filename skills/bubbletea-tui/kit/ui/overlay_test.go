package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/colorprofile"
	uv "github.com/charmbracelet/ultraviolet"

	"tuikit/theme"
)

// overlayPage is a page a dialog is drawn over: table rows whose columns are two blanks
// apart, prose, full-width rules, a rule that starts after a gap, and one row carrying
// the theme's focus bar glyph. It returns the rows as plain text, cell for cell.
func overlayPage(th *theme.Theme, c *Canvas) []string {
	b := c.Bounds()
	w := b.Dx()
	var plain []string
	for y := range b.Dy() {
		var row string
		switch {
		case y == 5:
			row = "  " + Gutter(th, true, th.S.FocusBar) + th.S.Bold.Render("checkout-api  v2.4.0  ok  p99 124ms  error 0.02%  deployed 3h")
		case y%4 == 0:
			row = "  " + th.S.Soft.Render("search-indexer  v1.9.3  busy  p99 88ms  error 0.00%  deployed 1d ago")
		case y%4 == 1:
			row = th.S.Soft.Render("lorem ipsum dolor sit amet consectetur adipiscing elit sed do eiusmod tempor")
		case y%4 == 2:
			row = th.S.Rule.Render(strings.Repeat(th.G.Rule, w))
		default:
			row = "  " + th.S.Subtle.Render("SERVICE") + strings.Repeat(" ", w-20) + th.S.Rule.Render(strings.Repeat(th.G.Rule, 11)) // starts two cells past the dialog
		}
		c.Draw(row, uv.Rect(0, y, w, 1))
		var sb strings.Builder
		for x := range w {
			if cell := c.CellAt(x, y); cell != nil && cell.Content != "" {
				sb.WriteString(cell.Content)
			} else {
				sb.WriteString(" ")
			}
		}
		plain = append(plain, sb.String())
	}
	return plain
}

// runs are a row's phrases: the cells between two-blank gaps, as [start, end) columns.
func runs(row []rune) [][2]int {
	var out [][2]int
	start, gap := -1, 0
	for x, r := range row {
		if r == ' ' {
			gap++
			if gap == 2 && start >= 0 {
				out = append(out, [2]int{start, x - 1})
				start = -1
			}
			continue
		}
		if start < 0 {
			start = x
		}
		gap = 0
	}
	if start >= 0 {
		end := len(row)
		for end > start && row[end-1] == ' ' {
			end--
		}
		out = append(out, [2]int{start, end})
	}
	return out
}

func cellText(c *Canvas, x, y int) string {
	if cell := c.CellAt(x, y); cell != nil && cell.Content != "" {
		return cell.Content
	}
	return " "
}

// A dialog leaves nothing of the page that reads as live beside it. Under a scrim, a
// phrase the dialog cuts is cleared to its column gap, so no fragment ("1s", "igh",
// "00K") sits beside the edge, and the cell beside each edge is blank; where there is
// no scrim (16 colours, NO_COLOR, ASCII, scrim: none) the rows the dialog spans are
// cleared across, an equivalent of the scrim rather than nothing. A page rule never
// runs on beside the dialog's own edge rows (a rules dialog's top rule met a table rule
// with one gap cell). The rows the dialog doesn't span are the page's, cell for cell,
// a focus bar's glyph among them: Overlay doesn't look for focus marks (the app draws
// the page unfocused), so a wordmark or a separator drawn in the same glyph survives.
func TestOverlayLeavesNoPageBesideIt(t *testing.T) {
	const W, H = 70, 20
	for _, dialog := range []string{"rounded", "rules", "none"} {
		for _, mode := range []struct {
			name  string
			ascii bool
			p     colorprofile.Profile
			scrim string
		}{
			{"truecolor", false, colorprofile.TrueColor, "dim"},
			{"ansi256", false, colorprofile.ANSI256, "dim"},
			{"ansi", false, colorprofile.ANSI, "dim"},
			{"NO_COLOR", false, colorprofile.Ascii, "dim"},
			{"ascii", true, colorprofile.TrueColor, "dim"},
			{"scrim none", false, colorprofile.TrueColor, "none"},
		} {
			f := theme.Default()
			f.Expressions.Dialog, f.Expressions.Scrim = dialog, mode.scrim
			th := theme.ResolveProfile(f, true, mode.ascii, mode.p)
			scrimmed := mode.scrim == "dim" && !mode.ascii && mode.p != colorprofile.ANSI && mode.p != colorprofile.Ascii
			d := Dialog(th, 44, 0, "Roll back checkout-api?", []string{"It goes back to v2.3.0.", "", "3 pods restart."}, "", "")
			edges := dialogEdgeRows(d)
			for y0 := 1; y0+len(strings.Split(d, "\n")) < H; y0++ {
				c := NewCanvas(th, W, H)
				page := overlayPage(th, c)
				r := c.Overlay(d, &uv.Position{X: -1, Y: y0})
				name := dialog + " " + mode.name
				for y := range H {
					if y >= r.Min.Y && y < r.Max.Y {
						continue
					}
					var got strings.Builder
					for x := range W {
						got.WriteString(cellText(c, x, y))
					}
					if got.String() != page[y] {
						t.Errorf("%s y0=%d: row %d, which the dialog doesn't span, reads %q, want the page's %q", name, y0, y, got.String(), page[y])
					}
				}
				for y := r.Min.Y; y < r.Max.Y; y++ {
					orig := []rune(page[y])
					for x := range W {
						if uv.Pos(x, y).In(r) {
							continue
						}
						got := cellText(c, x, y)
						if !scrimmed && got != " " {
							t.Errorf("%s y0=%d: no scrim, and the page shows %q at %d,%d on a row the dialog spans", name, y0, got, x, y)
							break
						}
						if got == " " {
							continue
						}
						if x == r.Min.X-1 || x == r.Max.X {
							t.Errorf("%s y0=%d: %q at %d,%d touches the dialog's edge", name, y0, got, x, y)
						}
						for _, run := range runs(orig) {
							if x >= run[0] && x < run[1] && run[0] <= r.Max.X && run[1] >= r.Min.X {
								t.Errorf("%s y0=%d: %q at %d,%d is a fragment of %q, which the dialog cuts", name, y0, got, x, y, string(orig[run[0]:run[1]]))
								x = W
								break
							}
						}
					}
				}
				for _, ey := range edges {
					y := r.Min.Y + ey
					for x := range W {
						if !uv.Pos(x, y).In(r) && cellText(c, x, y) == th.G.Rule {
							t.Errorf("%s y0=%d: a page rule at %d,%d runs on beside the dialog's edge row %d", name, y0, x, y, y)
							break
						}
					}
				}
			}
		}
	}
}
