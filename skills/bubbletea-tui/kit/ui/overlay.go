package ui

import (
	"github.com/charmbracelet/colorprofile"
	uv "github.com/charmbracelet/ultraviolet"
)

// What Overlay (and Popover) does to the page before it draws a dialog, so that nothing
// of the page reads as part of the dialog or as live beside it:
//
//   - The rows the dialog spans are its own, at every size and in every profile: the
//     page outside it on those rows is cleared across, to the page's background. Under
//     a scrim the kit once cleared only the word the dialog cut, with an ellipsis, and
//     at 120 columns a column of "…to show it" stood beside a confirm on its own rows,
//     re-cut at its margin, reading as the dialog's text.
//   - Where the page can be dimmed (a scrim, at truecolour and 256 colours, whatever
//     the glyph set), the rows the dialog doesn't span stay as the page drew them,
//     dimmed: the context of where the dialog was opened.
//   - Where it can't (16 colours, NO_COLOR, scrim: none), a modal dialog (centred
//     across: anchor nil or anchor.X < 0) hides the rows of the page's body
//     (Canvas.Body) across: at full ink, a page above and below the dialog is as live
//     as the dialog, and a confirm over a composer showed the reply's last lines under
//     its edge, a second copy of what it was about to send. The rows outside the body
//     (the app's header) are kept; with no Body set, everything outside the dialog is
//     cleared, and the app draws its footer over the dialog as it does anyway.
//   - A popover anchored to its trigger (anchor.X ≥ 0: a completion list above the line
//     being typed) is not modal, and clears only its band, so the line it completes
//     stays on screen.
//
// The page has no focus under a dialog because the app draws it unfocused, not because
// its marks are looked for afterwards (a glyph is not a focus mark: cartridge's █ bar is
// its wordmark's block, and ASCII's | bar is a header's separator).

// scrims reports whether Overlay dims the page: the theme asks for it and the terminal
// can show it. There is no scrim at 16 colours (the page is ANSI 0 or 15 there, and
// dimming toward it sends every text level to the page's own slot) or with no colour
// at all (NO_COLOR: a dimmed cell is the cell). ASCII glyphs are no reason: a terminal
// that takes --ascii at truecolour or 256 colours dims as well as any (TERM=linux, the
// console, is 16 colours and has no scrim by its profile).
func (c *Canvas) scrims() bool {
	p := c.T.Profile
	return c.T.X.Scrim == "dim" && p != colorprofile.ANSI && p != colorprofile.Ascii && p != colorprofile.NoTTY
}

// clearPage prepares the page under a dialog drawn at r (see above): the band of rows r
// spans, across the canvas, and, for a modal with no scrim, the rows of the body.
func (c *Canvas) clearPage(r uv.Rectangle, scrim, modal bool) {
	b := c.Bounds()
	c.clearOutside(uv.Rect(b.Min.X, r.Min.Y, b.Dx(), r.Dy()), r)
	if !scrim && modal {
		body := b
		if !c.Body.Empty() {
			body = uv.Rect(b.Min.X, c.Body.Min.Y, b.Dx(), c.Body.Dy())
		}
		c.clearOutside(body, r)
	}
}

// clearOutside blanks every cell of area that isn't in the dialog's rectangle r.
func (c *Canvas) clearOutside(area, r uv.Rectangle) {
	area = area.Intersect(c.Bounds())
	for y := area.Min.Y; y < area.Max.Y; y++ {
		for x := area.Min.X; x < area.Max.X; x++ {
			if !uv.Pos(x, y).In(r) {
				c.blank(x, y)
			}
		}
	}
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
