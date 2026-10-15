// Package cellshot renders ANSI terminal frames to images the way a modern terminal
// draws them: geometric box drawing, blocks, shades, quadrants, sextants, braille and
// powerline separators, embedded DejaVu Sans Mono for text, two cells for wide
// characters. It is the Go port of scripts/cellshot.py and draws geometric glyphs
// pixel-identically to it at the same cell size.
package cellshot

import (
	"fmt"
	"image"
	"image/draw"
	"math"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Options configure a render.
type Options struct {
	CellW, CellH int // pixel size of one cell
	FG, BG       RGB // terminal default foreground and background
	Cols, Rows   int // expected size; the image is padded to at least this (0: none)
	Grid         bool
}

// mix is cellshot.py's mix: a + (b-a)*t per channel, rounded half to even.
func mix(a, b RGB, t float64) RGB {
	m := func(x, y uint8) uint8 {
		v := float64(x) + float64(float64(int(y)-int(x))*t)
		return uint8(math.RoundToEven(v))
	}
	return RGB{m(a.R, b.R), m(a.G, b.G), m(a.B, b.B)}
}

// Render draws rows of cells. It returns the image and the sorted warnings about
// glyphs it could not draw.
func Render(rows [][]Cell, o Options) (*image.RGBA, []string) {
	cw, rh := o.CellW, o.CellH
	width := max(o.Cols, 1)
	for _, r := range rows {
		width = max(width, len(r))
	}
	height := max(len(rows), o.Rows, 1)
	c := newCanvas(width*cw, height*rh, o.BG)
	fonts := newFontSet(rh)
	warn := map[string]bool{}
	for y, row := range rows {
		for x, cell := range row {
			if cell.Cont {
				continue
			}
			c.drawCell(fonts, cell, x*cw, y*rh, cw, rh, o.FG, o.BG, warn)
		}
	}
	if o.Grid {
		g := mix(o.BG, RGB{128, 128, 128}, 0.18)
		for x := 0; x <= width; x++ {
			c.lines([]pt{ipt(x*cw, 0), ipt(x*cw, height*rh)}, g, 1)
		}
		for y := 0; y <= height; y++ {
			c.lines([]pt{ipt(0, y*rh), ipt(width*cw, y*rh)}, g, 1)
		}
	}
	return c.RGBA, sortedKeys(warn)
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func (c *canvas) drawCell(fonts *fontSet, cell Cell, x0, y0, cw, rh int, dfg, dbg RGB, warn map[string]bool) {
	fg := dfg
	if cell.FG.Set {
		fg = cell.FG.RGB
	}
	bg := cell.BG
	if cell.Reverse {
		nbg := Color{fg, true}
		fg = dbg
		if cell.BG.Set {
			fg = cell.BG.RGB
		}
		bg = nbg
	}
	w := cw
	if cell.Wide {
		w = 2 * cw
	}
	if bg.Set {
		c.rect(x0, y0, x0+w-1, y0+rh-1, bg.RGB)
	}
	eff := dbg
	if bg.Set {
		eff = bg.RGB
	}
	if cell.Faint {
		fg = mix(eff, fg, 0.55)
	}
	lw := max(1, cw/8)
	base, _ := utf8.DecodeRuneInString(cell.Text)
	switch {
	case cell.Text == "" || base == ' ':
	case c.drawGeometric(base, x0, y0, cw, rh, fg, eff):
	default:
		f := fonts.pick(base, cell.Bold, cell.Italic)
		if f == nil {
			warn[fmt.Sprintf("no font has %s U+%04X: drawn as tofu (it may be tofu for users too)", pyRepr(base), base)] = true
			c.rectOutline(x0+1, y0+2, x0+w-2, y0+rh-3, RGB{255, 0, 255})
			break
		}
		oblique := cell.Italic && !cell.Bold && f == fonts.reg
		_, _, xmin, xmax := f.layout(cell.Text, oblique)
		ox := x0
		if gw := xmax - xmin; gw > w {
			ox = x0 + max(0, floorDiv(w-gw, 2)-xmin)
		}
		c.text(f, cell.Text, ox, y0+int(float64(rh)*0.08), fg, oblique)
	}
	if cell.Under != 0 {
		uy := y0 + rh - max(2, rh/10)
		if cell.Under == 3 { // curly
			pts := make([]pt, 0, w)
			for k := 0; k < w; k++ {
				d := -1
				if (k/2)%2 != 0 {
					d = 1
				}
				pts = append(pts, ipt(x0+k, uy+d))
			}
			c.lines(pts, fg, 1)
		} else {
			c.lines([]pt{ipt(x0, uy), ipt(x0+w-1, uy)}, fg, max(1, lw))
			if cell.Under == 2 {
				c.lines([]pt{ipt(x0, uy-3), ipt(x0+w-1, uy-3)}, fg, 1)
			}
		}
	}
	if cell.Strike {
		c.lines([]pt{ipt(x0, y0+rh/2), ipt(x0+w-1, y0+rh/2)}, fg, max(1, lw))
	}
}

func floorDiv(a, b int) int {
	q := a / b
	if (a%b != 0) && ((a < 0) != (b < 0)) {
		q--
	}
	return q
}

// pyRepr formats a character the way Python's repr() does, for identical reports.
func pyRepr(r rune) string {
	switch {
	case r == '\'':
		return `"'"`
	case r == '\\':
		return `'\\'`
	case unicode.IsPrint(r) || r == ' ':
		return "'" + string(r) + "'"
	case r < 0x100:
		return fmt.Sprintf(`'\x%02x'`, r)
	case r < 0x10000:
		return fmt.Sprintf(`'\u%04x'`, r)
	}
	return fmt.Sprintf(`'\U%08x'`, r)
}

// Scale is an integer nearest-neighbour upscale.
func Scale(img *image.RGBA, n int) *image.RGBA {
	if n <= 1 {
		return img
	}
	b := img.Bounds()
	out := image.NewRGBA(image.Rect(0, 0, b.Dx()*n, b.Dy()*n))
	for y := 0; y < out.Rect.Dy(); y++ {
		src := img.Pix[(y/n)*img.Stride:]
		dst := out.Pix[y*out.Stride:]
		for x := 0; x < out.Rect.Dx(); x++ {
			copy(dst[x*4:x*4+4], src[(x/n)*4:(x/n)*4+4])
		}
	}
	return out
}

// Label puts a caption bar above img, as cellshot.py's label_bar does.
func Label(img *image.RGBA, text string, bg RGB) *image.RGBA {
	w, h := img.Rect.Dx(), img.Rect.Dy()
	c := newCanvas(w, h+30, bg)
	c.rect(0, 0, w-1, 29, mix(bg, RGB{255, 255, 255}, 0.12))
	fonts := newFontSet(22)
	c.text(fonts.reg, text, 8, 5, RGB{235, 235, 235}, false)
	draw.Draw(c.RGBA, image.Rect(0, 30, w, h+30), img, img.Rect.Min, draw.Src)
	return c.RGBA
}

// Sheet lays images out as a contact sheet: two per row, three when there are five or
// more narrow ones, 8px gutters on #28282c.
func Sheet(images []*image.RGBA) *image.RGBA {
	per := 1
	if len(images) > 1 {
		per = 2
	}
	if len(images) >= 5 && images[0].Rect.Dx() < 1100 {
		per = 3
	}
	gw := 0
	for _, im := range images {
		gw = max(gw, im.Rect.Dx())
	}
	var rowsOf [][]*image.RGBA
	for k := 0; k < len(images); k += per {
		rowsOf = append(rowsOf, images[k:min(k+per, len(images))])
	}
	rowH := func(r []*image.RGBA) int {
		h := 0
		for _, im := range r {
			h = max(h, im.Rect.Dy())
		}
		return h
	}
	H := 8 * (len(rowsOf) + 1)
	for _, r := range rowsOf {
		H += rowH(r)
	}
	sheet := newCanvas(per*gw+8*(per+1), H, RGB{40, 40, 44})
	y := 8
	for _, r := range rowsOf {
		x := 8
		for _, im := range r {
			draw.Draw(sheet.RGBA, image.Rect(x, y, x+im.Rect.Dx(), y+im.Rect.Dy()), im, im.Rect.Min, draw.Src)
			x += gw + 8
		}
		y += rowH(r) + 8
	}
	return sheet.RGBA
}

// ParseHex parses #rrggbb.
func ParseHex(s string) (RGB, error) {
	s = strings.TrimPrefix(s, "#")
	if len(s) != 6 {
		return RGB{}, fmt.Errorf("bad colour %q (want #rrggbb)", s)
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return RGB{}, fmt.Errorf("bad colour %q (want #rrggbb)", s)
	}
	return RGB{uint8(v >> 16), uint8(v >> 8), uint8(v)}, nil
}
