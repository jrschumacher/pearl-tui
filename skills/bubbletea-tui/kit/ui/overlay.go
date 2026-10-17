package ui

import (
	"strings"

	"github.com/charmbracelet/colorprofile"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
)

// What Overlay (and Popover) does to the page before it draws a dialog, so that nothing
// of the page reads as live beside it: a phrase the dialog cuts is cleared to its column
// gap (no "1s" or "igh" fragment beside the edge), a page rule never runs on beside the
// dialog's edge rows, and where no scrim is drawn the rows the dialog spans are cleared
// across. Rows the dialog doesn't span are left as the page drew them: the page has no
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
