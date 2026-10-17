package ui

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
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
		case y%8 == 0:
			row = "  " + th.S.Soft.Render("search-indexer  v1.9.3  busy  p99 88ms  error 0.00%  deployed 1d ago")
		case y%8 == 4:
			row = "  " + th.S.Warn.Render("▲") + " " + th.S.Base.Render("notifications-worker-eu-west  14:03 circuit opened on payments/api for checkout-web")
		case y%8 == 1:
			row = th.S.Soft.Render("lorem ipsum dolor sit amet consectetur adipiscing elit sed do eiusmod tempor")
		case y%8 == 5:
			row = "  " + th.S.Muted.Render("p99 "+strings.Repeat("▁▂▃▄▅▆▇█", 8))
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

// words are a row's words as [start, end) columns: runs of letters split at blanks,
// at the theme's rule glyph, box drawing, blocks and braille (shapes, which have no
// words), and after a - or /.
func words(row []rune, rule string) [][2]int {
	var out [][2]int
	start := -1
	for x, r := range row {
		if r == ' ' || graphic(string(r)) || string(r) == rule {
			if start >= 0 {
				out = append(out, [2]int{start, x})
			}
			start = -1
			continue
		}
		if start < 0 {
			start = x
		}
		if r == '-' || r == '/' {
			out = append(out, [2]int{start, x + 1})
			start = -1
		}
	}
	if start >= 0 {
		out = append(out, [2]int{start, len(row)})
	}
	return out
}

// checkCut judges one side of a dialog's row under a scrim: the cells from the halo
// (x = halo) outward by dx. The halo is blank; what changed is one run out from the
// halo, blanks and at most one ellipsis at its far end, so the row keeps everything
// past the cut word, and a run of shapes loses only the halo cell; no word is left in part (a word the dialog or the halo cuts is
// gone whole); and where the cut word's phrase goes on (a single blank or a joint
// before the kept text), the ellipsis marks the cut.
func checkCut(t *testing.T, name string, orig []rune, got []string, halo, dx int, th *theme.Theme, skip func(x int) bool) {
	t.Helper()
	if got[halo] != " " {
		t.Errorf("%s: %q at %d touches the dialog's edge", name, got[halo], halo)
	}
	in := func(x int) bool { return x >= 0 && x < len(orig) }
	changed := func(x int) bool { return got[x] != string(orig[x]) && !skip(x) }
	shape := func(r rune) bool { return graphic(string(r)) || string(r) == th.G.Rule }
	x := halo
	for in(x) && changed(x) && got[x] == " " {
		if x != halo && shape(orig[x]) {
			t.Errorf("%s: the shape %q at %d is cleared past the halo (a rule or a sparkline has no words): %q", name, string(orig[x]), x, strings.Join(got, ""))
			return
		}
		x += dx
	}
	ellAt := -1
	if in(x) && changed(x) && got[x] == th.G.Ellipsis {
		ellAt = x
		x += dx
	}
	for ; in(x); x += dx {
		if changed(x) {
			t.Errorf("%s: %q at %d is changed (now %q) past the cut: %q", name, string(orig[x]), x, got[x], strings.Join(got, ""))
			return
		}
	}
	for _, wd := range words(orig, th.G.Rule) {
		lo, hi := wd[0], wd[1]
		var out, kept int
		cut := false
		for i := lo; i < hi; i++ {
			if (dx < 0 && i > halo) || (dx > 0 && i < halo) {
				cut = true // inside the dialog
				continue
			}
			if i == halo {
				cut = true
			}
			out++
			if got[i] == string(orig[i]) {
				kept++
			}
		}
		if out == 0 {
			continue
		}
		if kept != 0 && (kept != out || cut) {
			t.Errorf("%s: %q is left in part: %q", name, string(orig[lo:hi]), strings.Join(got, ""))
		}
	}
	// the phrase goes on past the cleared run: the ellipsis says so
	end := halo
	for in(end+dx) && changed(end+dx) {
		end += dx
	}
	if end == halo || ellAt >= 0 || !changed(halo) && got[halo] == " " && orig[halo] == ' ' {
		return
	}
	k := end + dx
	if in(k) && (orig[k] == '-' || orig[k] == '/' || (orig[k] != ' ' && !shape(orig[k]))) {
		t.Errorf("%s: a cut word's phrase goes on at %d but no ellipsis marks it: %q", name, k, strings.Join(got, ""))
		return
	}
	if in(k) && orig[k] == ' ' && in(k+dx) && orig[k+dx] != ' ' && !shape(orig[k+dx]) {
		t.Errorf("%s: a cut word's phrase goes on at %d but no ellipsis marks it: %q", name, k+dx, strings.Join(got, ""))
	}
}

func cellText(c *Canvas, x, y int) string {
	if cell := c.CellAt(x, y); cell != nil && cell.Content != "" {
		return cell.Content
	}
	return " "
}

// A dialog leaves nothing of the page that reads as live beside it. Under a scrim, the
// cell beside each edge is blank and a word the dialog cuts is cleared whole, so no
// fragment ("1s", "igh", "00K") sits beside the edge, with the theme's ellipsis where
// its phrase was cut ("opened on…"); the rest of the row stays (clearing a phrase to its
// column gap took whole list rows, section headings and sparklines). Where there is
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
			{"rules in ·", false, colorprofile.TrueColor, "dim"}, // a rule glyph that is no shape: a section rule has no words either
		} {
			f := theme.Default()
			f.Expressions.Dialog, f.Expressions.Scrim = dialog, mode.scrim
			if mode.name == "rules in ·" {
				f.Glyphs.Rule = "·"
			}
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
					got := make([]string, W)
					for x := range W {
						got[x] = cellText(c, x, y)
					}
					if !scrimmed {
						for x := range W {
							if !uv.Pos(x, y).In(r) && got[x] != " " {
								t.Errorf("%s y0=%d: no scrim, and the page shows %q at %d,%d on a row the dialog spans", name, y0, got[x], x, y)
								break
							}
						}
						continue
					}
					// on the dialog's edge rows a page rule is cleared too (below)
					edge := slices.Contains(edges, y-r.Min.Y)
					skip := func(x int) bool { return edge && string(orig[x]) == th.G.Rule }
					row := fmt.Sprintf("%s y0=%d row %d", name, y0, y)
					checkCut(t, row+" left", orig, got, r.Min.X-1, -1, th, skip)
					checkCut(t, row+" right", orig, got, r.Max.X, 1, th, skip)
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

// A chip the dialog cuts goes whole, its padding too: "FAILING" was cleared and its
// filled padding stayed beside the dialog, an empty block of the chip's colour. Under a
// scrim, at every column the chip can sit across either edge, a side of the chip's row
// with no text left keeps no background but the page's.
func TestOverlayLeavesNoEmptyChip(t *testing.T) {
	const W, H = 70, 9
	th := theme.ResolveProfile(theme.Default(), true, false, colorprofile.TrueColor)
	d := Dialog(th, 44, 0, "Roll back checkout-api?", []string{"It goes back to v2.3.0."}, "", "")
	chip := Tag(th, "error", "FAILING")
	for x0 := 0; x0 < W-lipgloss.Width(chip); x0++ {
		c := NewCanvas(th, W, H)
		c.Draw(strings.Repeat(" ", x0)+chip, uv.Rect(0, 4, W, 1))
		r := c.Overlay(d, &uv.Position{X: -1, Y: 2})
		for _, side := range [][2]int{{0, r.Min.X}, {r.Max.X, W}} {
			text, filled := false, -1
			for x := side[0]; x < side[1]; x++ {
				cell := c.CellAt(x, 4)
				if cell == nil {
					continue
				}
				if strings.TrimSpace(cell.Content) != "" {
					text = true
				} else if cell.Style.Bg != nil && !sameColor(cell.Style.Bg, th.P.Bg) && filled < 0 {
					filled = x
				}
			}
			if !text && filled >= 0 {
				t.Errorf("chip at %d: an empty chip at %d beside the dialog", x0, filled)
			}
		}
	}
}
