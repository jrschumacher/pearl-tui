package ui

import (
	"image/color"
	"strings"

	"github.com/charmbracelet/colorprofile"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
)

// What Overlay (and Popover) does to the page before it draws a dialog, so that nothing
// of the page reads as live beside it: the cell beside each edge is blank and a word the
// dialog cuts is cleared whole, with an ellipsis where its phrase was cut (no "1s" or
// "igh" fragment beside the edge), a page rule never runs on beside the dialog's edge
// rows, and where no scrim is drawn the rows the dialog spans are cleared across. Rows the dialog doesn't span are left as the page drew them: the page has no
// focus under a dialog because the app draws it unfocused, not because its marks are
// looked for afterwards (a glyph is not a focus mark: cartridge's █ bar is its
// wordmark's block, and ASCII's | bar is a header's separator).

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
		c.clearCut(r.Min.X-1, y, -1)
		c.clearCut(r.Max.X, y, 1)
	}
	for _, ey := range dialogEdgeRows(dialog) {
		c.clearRules(r, r.Min.Y+ey)
	}
}

// clearCut keeps a one-cell halo beside a dialog's edge at (h, y): the cell is
// blanked, and when it holds part of a word, the rest of that word outside the dialog
// goes too (stepping by dx, away from the dialog), with the theme's ellipsis where the
// kept text was cut, the kit's own overflow idiom: "▲ notifications-…", "opened on…",
// "…latency". No "1s" or "igh" fragment is left at the edge, and every row keeps what
// it had up to the cut word. A word ends at a blank or after a - or /; a run of rules,
// blocks or braille (a sparkline, a section rule) has no words, so only the halo cell
// goes (shape). When the cut word follows a column gap, its column's phrase is gone
// whole and no ellipsis is drawn.
func (c *Canvas) clearCut(h, y, dx int) {
	b := c.Bounds()
	if h < b.Min.X || h >= b.Max.X {
		return
	}
	if c.isBlank(h, y) {
		// a chip's padding at the halo: the rest of the padding goes with it
		for x := h; x >= b.Min.X && x < b.Max.X && c.isBlank(x, y) && !c.onPage(x, y); x += dx {
			c.blank(x, y)
		}
		c.blank(h, y)
		return
	}
	if c.shape(h, y) {
		c.blank(h, y)
		return
	}
	in := func(x int) bool { return x >= b.Min.X && x < b.Max.X }
	word := func(x int) bool { return in(x) && !c.isBlank(x, y) && !c.shape(x, y) }
	joint := func(x int) bool { s := c.text(x, y); return s == "-" || s == "/" }
	// far is the word's last cell away from the dialog: going left, the cell after a
	// joint or a gap; going right, a joint itself or the cell before a gap
	far := func(x int) int {
		if dx < 0 {
			for word(x-1) && !joint(x-1) {
				x--
			}
			return x
		}
		for !joint(x) && word(x+1) {
			x++
		}
		return x
	}
	for x := h; ; {
		end := far(x)
		for i := min(x, end); i <= max(x, end); i++ {
			c.blank(i, y)
		}
		// the ellipsis takes the cell between the cleared word and the kept text: a
		// joint's place going right, the cell after it going left, or a single blank
		next := end + dx
		var at int
		switch {
		case dx < 0 && joint(next):
			at = end
		case dx > 0 && joint(end) && word(next):
			at = end
		case !c.isBlank(next, y) || !in(next) || !word(next+dx):
			// a column gap, the row's edge or a graphic: the phrase is gone whole, and so
			// is a chip's padding after it (an empty FAILING chip read as a bug)
			for x := next; in(x) && c.isBlank(x, y) && !c.onPage(x, y); x += dx {
				c.blank(x, y)
			}
			return
		default:
			at = next
		}
		if at == h {
			// the ellipsis would touch the dialog (the halo held a joint): the next word
			// out goes too
			x = next
			continue
		}
		c.ellipsis(at, y)
		return
	}
}

// onPage reports a cell on the page's own background, not a chip's or a band's.
func (c *Canvas) onPage(x, y int) bool {
	cell := c.CellAt(x, y)
	if cell == nil || cell.Style.Bg == nil {
		return true
	}
	r1, g1, b1, _ := cell.Style.Bg.RGBA()
	r2, g2, b2, _ := c.T.P.Bg.RGBA()
	return c.T.X.PaintBg && r1 == r2 && g1 == g2 && b1 == b2
}

// ellipsis draws the theme's ellipsis at x, y in the ink of the kept text beside it,
// on the page's background; nothing when the theme's ellipsis is wider than a cell.
func (c *Canvas) ellipsis(x, y int) {
	g := c.T.G.Ellipsis
	if ansi.StringWidth(g) != 1 {
		return
	}
	var fg color.Color
	for _, k := range []int{x - 1, x + 1} {
		if cell := c.CellAt(k, y); cell != nil && !c.isBlank(k, y) {
			fg = cell.Style.Fg
			break
		}
	}
	nc := uv.EmptyCell
	nc.Content, nc.Width = g, 1
	nc.Style.Fg = fg
	if c.T.X.PaintBg {
		nc.Style.Bg = c.T.P.Bg
	}
	c.SetCell(x, y, &nc)
}

// shape reports a cell that draws a shape, not a letter: the theme's rule glyphs
// (moss rules in ·), box drawing, block elements (bars, sparklines) and braille
// (charts). A cut through them is no fragment.
func (c *Canvas) shape(x, y int) bool {
	s := c.text(x, y)
	return s == c.T.G.Rule || (c.T.G.HeaderRule != "" && s == c.T.G.HeaderRule) || graphic(s)
}

func graphic(s string) bool {
	for _, r := range s {
		if (r >= 0x2500 && r <= 0x259F) || (r >= 0x2800 && r <= 0x28FF) {
			return true
		}
	}
	return false
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

// blank empties the cell at x, y to the page's background: bg when the app paints the
// terminal, else none (the terminal's own). Not the background the cell had: a chip,
// a status strip or a selection band emptied by the clear would stay a solid block
// with nothing in it. Under a scrim the cleared cell is dimmed with the page after it.
func (c *Canvas) blank(x, y int) {
	nc := uv.EmptyCell
	if c.T.X.PaintBg {
		nc.Style.Bg = c.T.P.Bg
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
