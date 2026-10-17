package ui

import (
	"image/color"
	"strings"

	"github.com/charmbracelet/colorprofile"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
)

// What Overlay does to the page before it draws a dialog, so that nothing of the page
// reads as live beside it: the page's focus marks go (the dialog has the focus), a
// phrase the dialog cuts is cleared to its column gap (no "1s" or "igh" fragment beside
// the edge), a page rule never runs on beside the dialog's edge rows, and where no
// scrim is drawn the rows the dialog spans are cleared across.

// scrims reports whether Overlay dims the page: the theme asks for it and the terminal
// can show it. There is no scrim under ASCII, at 16 colours (the page is ANSI 0 or 15
// there, and dimming toward it sends every text level to the page's own slot) or with
// no colour at all (NO_COLOR: a dimmed cell is the cell).
func (c *Canvas) scrims() bool {
	p := c.T.Profile
	return c.T.X.Scrim == "dim" && !c.T.ASCII && p != colorprofile.ANSI && p != colorprofile.Ascii && p != colorprofile.NoTTY
}

// clearPage prepares the page under a dialog drawn at r (see above). band clears the
// rows r spans across the canvas: the equivalent of a scrim where none is drawn.
func (c *Canvas) clearPage(r uv.Rectangle, dialog string, band bool) {
	c.unfocus()
	b := c.Bounds()
	for y := max(r.Min.Y, b.Min.Y); y < min(r.Max.Y, b.Max.Y); y++ {
		if band {
			for x := b.Min.X; x < b.Max.X; x++ {
				if !uv.Pos(x, y).In(r) {
					c.blank(x, y)
				}
			}
			continue
		}
		c.clearCut(r.Min.X-1, y, r.Min.X, -1)
		c.clearCut(r.Max.X, y, r.Max.X-1, 1)
	}
	for _, ey := range dialogEdgeRows(dialog) {
		c.clearRules(r, r.Min.Y+ey)
	}
}

// clearCut clears the cell beside a dialog's edge at (x, y) and, when the page's text
// runs across the edge (inside, the cell at in) or starts at the cell beside it, the
// rest of that phrase, stepping by dx until a gap of two blanks: a word the dialog cuts
// is gone whole, not left as a fragment.
func (c *Canvas) clearCut(x, y, in, dx int) {
	b := c.Bounds()
	if x < b.Min.X || x >= b.Max.X {
		return
	}
	cut := !c.isBlank(x, y)
	c.blank(x, y)
	if !cut && (in < b.Min.X || in >= b.Max.X || c.isBlank(in, y)) {
		return
	}
	for x += dx; x >= b.Min.X && x < b.Max.X; x += dx {
		if c.isBlank(x, y) && c.isBlank(x+dx, y) {
			return
		}
		c.blank(x, y)
	}
}

// clearRules clears every run of three or more rule glyphs on row y outside r: a page
// rule beside a dialog's edge row reads as the dialog's own edge, broken by a gap.
func (c *Canvas) clearRules(r uv.Rectangle, y int) {
	b := c.Bounds()
	if y < b.Min.Y || y >= b.Max.Y {
		return
	}
	isRule := func(x int) bool {
		if x < b.Min.X || x >= b.Max.X || uv.Pos(x, y).In(r) {
			return false
		}
		s := c.text(x, y)
		return s == c.T.G.Rule || (c.T.G.HeaderRule != "" && s == c.T.G.HeaderRule)
	}
	for x := b.Min.X; x < b.Max.X; {
		n := 0
		for isRule(x + n) {
			n++
		}
		if n >= 3 {
			for i := range n {
				c.blank(x+i, y)
			}
		}
		x += max(1, n)
	}
}

// unfocus clears the page's focus marks: the focus bar beside a selected block and
// the selection cursor, each with a blank after it, in the focus ink or the quieter
// one a blurred selection draws the same glyph in (with the colour gone, that is a
// focus mark too). Under a dialog the page has no focus, and a list's bar showing
// through the scrim beside the dialog's own made two.
func (c *Canvas) unfocus() {
	t := c.T
	marks := []struct {
		glyph string
		fg    color.Color
	}{
		{t.G.FocusBar, t.S.FocusBar.GetForeground()}, {t.G.FocusBar, t.S.BlurBar.GetForeground()},
		{t.G.Cursor, t.S.Focus.GetForeground()}, {t.G.Cursor, t.S.Muted.GetForeground()},
	}
	b := c.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			cell := c.CellAt(x, y)
			if cell == nil || x+1 >= b.Max.X || !c.isBlank(x+1, y) {
				continue
			}
			for _, m := range marks {
				if m.glyph != "" && cell.Content == m.glyph && sameInk(cell.Style.Fg, m.fg) {
					c.blank(x, y)
					break
				}
			}
		}
	}
}

func sameInk(a, b color.Color) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	r1, g1, b1, _ := a.RGBA()
	r2, g2, b2, _ := b.RGBA()
	return r1 == r2 && g1 == g2 && b1 == b2
}

// text is the content of the cell at x, y; a blank for an empty cell.
func (c *Canvas) text(x, y int) string {
	if cell := c.CellAt(x, y); cell != nil && cell.Content != "" {
		return cell.Content
	}
	return " "
}

func (c *Canvas) isBlank(x, y int) bool {
	if !uv.Pos(x, y).In(c.Bounds()) {
		return true
	}
	return strings.TrimSpace(c.text(x, y)) == ""
}

// blank empties the cell at x, y and keeps its background, so a cleared cell on a
// filled row stays part of the fill.
func (c *Canvas) blank(x, y int) {
	cell := c.CellAt(x, y)
	nc := uv.EmptyCell
	if cell != nil {
		nc.Style.Bg = cell.Style.Bg
	}
	c.SetCell(x, y, &nc)
}

// dialogEdgeRows are the rows of a drawn dialog that hold its top and bottom edges:
// its first and last rows with anything on them (a rules dialog's margin rows are
// blank, so its rules are the second and second-to-last).
func dialogEdgeRows(dialog string) []int {
	rows := strings.Split(dialog, "\n")
	first, last := -1, -1
	for i, r := range rows {
		if strings.TrimSpace(ansi.Strip(r)) != "" {
			if first < 0 {
				first = i
			}
			last = i
		}
	}
	if first < 0 {
		return nil
	}
	if last == first {
		return []int{first}
	}
	return []int{first, last}
}
