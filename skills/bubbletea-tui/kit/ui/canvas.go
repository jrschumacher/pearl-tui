// Package ui is the component vocabulary: one function per idiom element,
// each reading the theme and returning exactly the cells it was given. Screens
// are compositions of these, drawn into rectangles on a canvas.
package ui

import (
	"image/color"
	"regexp"
	"strings"

	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/ultraviolet/screen"
	"github.com/charmbracelet/x/ansi"

	"tuikit/theme"
)

// Canvas is a cell buffer the size of the terminal. Regions draw into
// rectangles; anything outside a rectangle is clipped, so a long line can
// never wrap and tear the frame. Draw order is z-order: dialogs last.
type Canvas struct {
	uv.ScreenBuffer
	T *theme.Theme
}

// NewCanvas makes a cleared canvas.
func NewCanvas(t *theme.Theme, w, h int) *Canvas {
	c := &Canvas{ScreenBuffer: uv.NewScreenBuffer(w, h), T: t}
	screen.Clear(c)
	return c
}

// Draw paints a styled string into a rectangle (clipped to it).
func (c *Canvas) Draw(s string, r uv.Rectangle) {
	uv.NewStyledString(s).Draw(c, r)
}

// Fill paints a rectangle's background (for panels and bars).
func (c *Canvas) Fill(r uv.Rectangle, bg color.Color) {
	cell := uv.EmptyCell
	cell.Style.Bg = bg
	screen.FillArea(c, &cell, r)
}

// Scrim dims everything already drawn toward the background, so a dialog
// drawn next reads as in front. amount 0..1 (0.55 is a good default).
func (c *Canvas) Scrim(amount float64) {
	p := c.T.P
	b := c.Bounds()
	cache := map[color.Color]color.Color{}
	dim := func(col color.Color, def color.Color) color.Color {
		if col == nil {
			col = def
		}
		if v, ok := cache[col]; ok {
			return v
		}
		v := theme.Mix(col, p.Bg, amount)
		cache[col] = v
		return v
	}
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			cell := c.CellAt(x, y)
			if cell == nil {
				continue
			}
			nc := *cell
			nc.Style.Fg = dim(cell.Style.Fg, p.Fg)
			if cell.Style.Bg != nil {
				nc.Style.Bg = dim(cell.Style.Bg, p.Bg)
			}
			c.SetCell(x, y, &nc)
		}
	}
}

// Backfill gives every cell in r that has no background the colour bg. Draw
// a panel's content first, then backfill: inner style resets can't punch
// holes in the panel that way.
func (c *Canvas) Backfill(r uv.Rectangle, bg color.Color) {
	r = r.Intersect(c.Bounds())
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			cell := c.CellAt(x, y)
			if cell == nil {
				nc := uv.EmptyCell
				nc.Style.Bg = bg
				c.SetCell(x, y, &nc)
				continue
			}
			if cell.Style.Bg != nil {
				continue
			}
			nc := *cell
			nc.Style.Bg = bg
			c.SetCell(x, y, &nc)
		}
	}
}

// PaintBackground fills every cell that has no background with the theme bg,
// and every cell with no foreground with the theme fg: what a terminal set to
// the brand's colours shows. The live program gets the same from
// tea.View.BackgroundColor/ForegroundColor; the harness uses this so
// screenshots are honest. Unstyled text is a bug either way: on a light
// terminal it takes the user's default fg, which your palette never checked.
func (c *Canvas) PaintBackground() {
	c.Backfill(c.Bounds(), c.T.P.Bg)
	b := c.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if cell := c.CellAt(x, y); cell != nil && cell.Style.Fg == nil && cell.Content != " " && cell.Content != "" {
				nc := *cell
				nc.Style.Fg = c.T.P.Fg
				c.SetCell(x, y, &nc)
			}
		}
	}
}

// ASCIIfy maps every cell outside ASCII onto a tier-0 stand-in: box drawing
// to - | +, blocks to # and =, arrows to < > ^ v, anything else to ?. Call it
// last when the theme is resolved for ASCII, so no screen can leak a glyph.
func (c *Canvas) ASCIIfy() {
	b := c.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			cell := c.CellAt(x, y)
			if cell == nil || cell.Content == "" {
				continue
			}
			r := []rune(cell.Content)[0]
			if r < 128 {
				continue
			}
			nc := *cell
			nc.Content, nc.Width = asciiRune(r), 1
			c.SetCell(x, y, &nc)
			if cell.Width == 2 && x+1 < b.Max.X { // a wide glyph's second cell
				sp := nc
				sp.Content = " "
				c.SetCell(x+1, y, &sp)
			}
		}
	}
}

func asciiRune(r rune) string {
	for _, c := range asciiClasses {
		if strings.ContainsRune(c.glyphs, r) {
			return c.twin
		}
	}
	switch {
	case r >= 0x2500 && r <= 0x257F: // other box drawing: corners, tees, crosses
		return "+"
	case r >= 0x2800 && r <= 0x28FF: // braille
		return "."
	}
	if s, ok := asciiMap[r]; ok {
		return s
	}
	return "?"
}

// asciiClasses and asciiMap are the twin tables: the one place outside the theme
// package that spells glyphs (tuilint code C-GLYPH skips their literals). A class maps
// a run of glyphs to one twin; the map, single glyphs. They agree with theme.ASCIITwin
// where a glyph carries a meaning: the caret ▏ is ', not the focus bar's |, and an
// ellipsis left for the sweep (a cut made with the bare theme.Ellipsis, ui.Fit's) is
// ".", one cell of the theme's "...", never "~", the partial mark's twin: a cut line
// read as a row in progress. Cut with t.G.Ellipsis (ui.Trunc) to get the full twin.
var asciiClasses = []struct{ glyphs, twin string }{
	{"─━═╌┄┈╍┅┉▔▁╴╶╸╺", "-"},
	{"│┃║╎┆┊╏┇┋▎▍▌▐▕", "|"},
	{"╱", "/"}, {"╲", "\\"},
	{"█▓▒▆▇▅▄▀▙▛▜▟", "#"},
	{"░▂▃▖▗▘▝▚▞", "."},
}

var asciiMap = map[rune]string{
	'↑': "^", '↓': "v", '←': "<", '→': ">", '·': ".", '•': "*", '…': ".", '—': "-", '–': "-",
	'×': "x", '✕': "x", '✗': "x", '✓': "+", '✔': "+", '●': "*", '○': "o", '◆': "*", '◇': "o",
	'▲': "^", '△': "^", '▶': ">", '▸': ">", '›': ">", '❯': ">", '‹': "<", '“': "\"", '”': "\"",
	'‘': "'", '’': "'", '▏': "'", '◉': "@", '■': "#", '□': "o", '★': "*", '✦': "*", '◜': "|", '◠': "/",
	'◝': "-", '◞': "\\", '◡': "|", '◟': "/", '◢': "/", '◣': "\\", '◤': "/", '◥': "\\", '¬': "-",
}

// String renders the canvas as rows joined by "\n" (every row exactly W cells).
func (c *Canvas) String() string {
	return strings.ReplaceAll(c.Render(), "\r\n", "\n")
}

// OnBg puts a whole pre-styled line on a background: the colour is re-applied
// after every reset inside it, so styled spans don't punch holes in a strip.
// (On a canvas, prefer Draw + Backfill.)
func OnBg(s string, bg color.Color) string {
	open := lipgloss.NewStyle().Background(bg).Render(" ")
	open = open[:strings.Index(open, " ")] // just the SGR that sets bg
	s = resetRe.ReplaceAllString(s, "${0}"+open)
	return open + s + "\x1b[0m"
}

var resetRe = regexp.MustCompile(`\x1b\[0?m`)

// Fit pads or truncates a single line to exactly w cells. A cut ends in the kit's
// ellipsis, which ASCII sweeps to "."; for the theme's own glyph and its full ASCII twin
// cut with Trunc first.
func Fit(s string, w int) string {
	if w <= 0 {
		return ""
	}
	n := lipgloss.Width(s)
	if n > w {
		return ansi.Truncate(s, w, theme.Ellipsis)
	}
	return s + strings.Repeat(" ", w-n)
}

// FitLines fits every line of a block to w cells.
func FitLines(block string, w int) string {
	lines := strings.Split(block, "\n")
	for i, l := range lines {
		lines[i] = Fit(l, w)
	}
	return strings.Join(lines, "\n")
}

// Trunc truncates to w cells with the theme's ellipsis (no padding).
func Trunc(t *theme.Theme, s string, w int) string {
	if lipgloss.Width(s) <= w {
		return s
	}
	return ansi.Truncate(s, w, t.G.Ellipsis)
}

// Rect is uv.Rect re-exported for screens.
func Rect(x, y, w, h int) uv.Rectangle { return uv.Rect(x, y, w, h) }

// Inset shrinks a rectangle by dx columns on each side and dy rows top/bottom.
func Inset(r uv.Rectangle, dx, dy int) uv.Rectangle {
	r.Min.X += dx
	r.Max.X -= dx
	r.Min.Y += dy
	r.Max.Y -= dy
	if r.Max.X < r.Min.X {
		r.Max.X = r.Min.X
	}
	if r.Max.Y < r.Min.Y {
		r.Max.Y = r.Min.Y
	}
	return r
}
