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

// Draw paints a styled string into a rectangle (clipped to it). A tab advances to the
// next 8-column stop of its line (ExpandTabs), as a terminal would.
func (c *Canvas) Draw(s string, r uv.Rectangle) {
	uv.NewStyledString(ExpandTabs(s)).Draw(c, r)
}

// ExpandTabs replaces each tab with the spaces to the next 8-column stop of its line,
// counted in cells (styling is skipped). lipgloss measures a tab as 0 cells and a
// terminal advances it to the next stop, so a line with a raw tab (go test's
// "ok  \tpkg\t0.412s") is fitted short and tears the edge it was padded to. Fit, Trunc
// and Canvas.Draw call it; call it on text you measure any other way.
func ExpandTabs(s string) string {
	if !strings.Contains(s, "\t") {
		return s
	}
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		if !strings.Contains(l, "\t") {
			continue
		}
		var b strings.Builder
		col := 0
		parts := strings.Split(l, "\t")
		for j, p := range parts {
			b.WriteString(p)
			col += ansi.StringWidth(p)
			if j < len(parts)-1 {
				n := 8 - col%8
				b.WriteString(strings.Repeat(" ", n))
				col += n
			}
		}
		lines[i] = b.String()
	}
	return strings.Join(lines, "\n")
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
// to - | +, blocks to # and =, arrows to < > ^ v, Latin-1 text to its base letter or
// unit (é → e, ° → a space, × → x), anything else to ?. Call it last when the theme is
// resolved for ASCII, so no screen can leak a glyph. A "?" is loud on purpose: tuilint
// frames --ascii reports one where the unicode frame has a glyph.
func (c *Canvas) ASCIIfy() {
	b := c.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			cell := c.CellAt(x, y)
			if cell == nil || cell.Content == "" {
				continue
			}
			rs := []rune(cell.Content)
			r := rs[0]
			if r < 128 && !nonASCII(rs[1:]) {
				continue
			}
			nc := *cell
			nc.Content, nc.Width = string(r), 1
			if r >= 128 {
				nc.Content = asciiRune(r)
			}
			// the rest of the cluster rides on the first rune: a combining mark (NFD
			// "e" + U+0308, as macOS file names and git print them) or a soft hyphen
			// (U+00AD, zero width) folds into the base letter, never leaks
			for _, m := range rs[1:] {
				if m < 128 {
					nc.Content += string(m)
				}
			}
			c.SetCell(x, y, &nc)
			if string(r) == theme.Ellipsis { // a bare cut: finish it as "..."
				c.cutDots(x, y, &nc)
			}
			if cell.Width == 2 && x+1 < b.Max.X { // a wide glyph's second cell
				sp := nc
				sp.Content = " "
				c.SetCell(x+1, y, &sp)
			}
		}
	}
}

func nonASCII(rs []rune) bool {
	for _, r := range rs {
		if r >= 128 {
			return true
		}
	}
	return false
}

// cutDots finishes a cut the sweep found as a bare "…" (ui.Fit's, a screen's own
// ansi.Truncate): one "." reads as a full stop ("unknown key."), so the two cells
// before it turn to "." too and the cut reads as the theme's twin ("unknown k...").
// Only cells of the cut text change: the same style and link as the ellipsis (a
// truncated string carries its style to the tail), never a border's | or +.
func (c *Canvas) cutDots(x, y int, dot *uv.Cell) {
	for i := 1; i <= 2 && x-i >= c.Bounds().Min.X; i++ {
		l := c.CellAt(x-i, y)
		if l == nil || l.Width != 1 || l.Content == "|" || l.Content == "+" ||
			!l.Style.Equal(&dot.Style) || l.Link != dot.Link {
			return
		}
		nc := *l
		nc.Content = "."
		c.SetCell(x-i, y, &nc)
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
	if s, ok := latin1Rune(r); ok {
		return s
	}
	return "?"
}

// latin1Rune is the one-cell ASCII stand-in for a text rune: a Latin-1 letter's base
// letter (é → e, Ø → O), a unit or mark a screen prints beside a number (° → a space,
// so 21.5°C reads 21.5 C; × → x, µ → u, © → c), currency (£ → L, € → E) and the
// punctuation of prose (« » ‚ „ ′ ″ −). One cell each, so a row keeps its columns: a
// twin that needs more (½ as 1/2, © as (c)) would tear the row. A rune whose one cell
// would change the value (± as +, ≥ as >, ‰ as %, ³ as 3, ¬ as -, † as +, § as S) has
// no stand-in: it stays "?", which the lint names, so write it in ASCII in the app's
// own text. A silent wrong value is worse than a flagged "?".
func latin1Rune(r rune) (string, bool) {
	if 0xC0 <= r && r <= 0xFF {
		return string(latin1Letters[r-0xC0]), true
	}
	s, ok := latin1Map[r]
	return s, ok
}

// latin1Letters are the base letters of U+00C0–U+00FF in order (À..ÿ): the accent
// goes, the letter stays; Æ, Ð, Ø, Þ, ß and their lowercase take one letter each.
const latin1Letters = "AAAAAAACEEEEIIIIDNOOOOOxOUUUUYTsaaaaaaaceeeeiiiidnooooo/ouuuuyty"

var latin1Map = map[rune]string{
	'\u00A0': " ", '¡': "!", '¢': "c", '£': "L", '¤': "o", '¥': "Y", '¦': "|", '¨': "\"", '©': "c",
	'ª': "a", '«': "<", '®': "R", '¯': "-", '°': " ", '´': "'", 'µ': "u", '¸': ",",
	'º': "o", '»': ">", '€': "E", '‚': ",", '„': "\"", '′': "'", '″': "\"", '−': "-", 'Œ': "O",
	'œ': "o", 'Š': "S", 'š': "s", 'Ž': "Z", 'ž': "z", 'Ÿ': "Y",
}

// asciiClasses and asciiMap are the twin tables: the one place outside the theme
// package that spells glyphs (tuilint code C-GLYPH skips their literals). A class maps
// a run of glyphs to one twin; the map, single glyphs. They agree with theme.ASCIITwin
// where a glyph carries a meaning: the caret ▏ is ', not the focus bar's |, and an
// ellipsis left for the sweep (a cut made with the bare theme.Ellipsis, ui.Fit's) is
// ".", never "~", the partial mark's twin: a cut line read as a row in progress; and
// cutDots turns the two cells of cut text before it to ".", so the cut reads "..." and
// not as a full stop. Cut with t.G.Ellipsis (ui.Trunc) to keep the text's last cells.
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
	'◝': "-", '◞': "\\", '◡': "|", '◟': "/", '◢': "/", '◣': "\\", '◤': "/", '◥': "\\",
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
// ellipsis, which the ASCII sweep draws as "..." over the cut's last cells; for the
// theme's own glyph and its ASCII twin after the text, cut with Trunc first. Tabs
// expand to 8-column stops first.
func Fit(s string, w int) string {
	if w <= 0 {
		return ""
	}
	s = ExpandTabs(s)
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

// Trunc truncates to w cells with the theme's ellipsis (no padding). Tabs expand to
// 8-column stops first.
func Trunc(t *theme.Theme, s string, w int) string {
	s = ExpandTabs(s)
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
