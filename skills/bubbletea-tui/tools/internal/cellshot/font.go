package cellshot

import (
	_ "embed"
	"image"
	"image/draw"
	"math"
	"sync"
	"unicode"

	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
	"golang.org/x/image/vector"
	"golang.org/x/text/unicode/norm"
)

// The fonts are embedded so a render is the same on every machine. DejaVu Sans Mono
// (regular, bold) draws text; DejaVuSans-Symbols.ttf is DejaVu Sans cut down to the
// U+2000-U+2BFF symbols that the mono face lacks (⋮ ☰ ⟳ ⬤ …), built with
//
//	pyftsubset DejaVuSans.ttf --unicodes=<codepoints in Sans, not in Mono, U+2000-2BFF> \
//	  --no-hinting --layout-features='' --name-IDs='*' --notdef-outline
//
// All three are DejaVu 2.37 under the Bitstream Vera / Arev licence in fonts/LICENSE,
// which permits redistribution and, for renamed-free subsets, modification.
var (
	//go:embed fonts/DejaVuSansMono.ttf
	monoRegularTTF []byte
	//go:embed fonts/DejaVuSansMono-Bold.ttf
	monoBoldTTF []byte
	//go:embed fonts/DejaVuSans-Symbols.ttf
	symbolsTTF []byte
)

// obliqueShear is tan(11°), the slant of DejaVu Sans Mono Oblique, used to synthesise
// italics from the regular face.
const obliqueShear = 0.19438

var (
	parseOnce                          sync.Once
	fontRegular, fontBold, fontSymbols *sfnt.Font
)

func parsedFonts() {
	parseOnce.Do(func() {
		must := func(b []byte) *sfnt.Font {
			f, err := sfnt.Parse(b)
			if err != nil {
				panic("cellshot: embedded font: " + err.Error())
			}
			return f
		}
		fontRegular, fontBold, fontSymbols = must(monoRegularTTF), must(monoBoldTTF), must(symbolsTTF)
	})
}

// face is one font at one pixel size.
type face struct {
	f      *sfnt.Font
	ppem   fixed.Int26_6
	asc    int // FreeType's hinted ascender: ceil(hhea.ascender * size / upem)
	buf    sfnt.Buffer
	glyphs map[glyphKey]*glyphMask
	has    map[rune]bool
}

type glyphKey struct {
	idx     sfnt.GlyphIndex
	oblique bool
}

// glyphMask is a rasterised glyph: coverage, its offset from the pen position on the
// baseline, the advance, and the control-box extent in pixels.
type glyphMask struct {
	mask       *image.Alpha
	ox, oy     int
	adv        fixed.Int26_6
	inkL, inkR int
	empty      bool
}

func newFace(f *sfnt.Font, size int) *face {
	fc := &face{f: f, ppem: fixed.I(size), glyphs: map[glyphKey]*glyphMask{}, has: map[rune]bool{}}
	upem := f.UnitsPerEm()
	m, err := f.Metrics(&fc.buf, fixed.I(int(upem)), font.HintingNone)
	if err == nil {
		units := float64(m.Ascent) / 64
		fc.asc = int(math.Ceil(units*float64(size)/float64(upem) - 1e-9))
	}
	return fc
}

func (fc *face) index(r rune) sfnt.GlyphIndex {
	i, err := fc.f.GlyphIndex(&fc.buf, r)
	if err != nil {
		return 0
	}
	return i
}

// covers mirrors cellshot.py's Fonts._missing: the glyph must exist, and a glyph with no
// ink counts as missing unless the character is a space.
func (fc *face) covers(r rune) bool {
	if v, ok := fc.has[r]; ok {
		return v
	}
	idx := fc.index(r)
	ok := idx != 0
	if ok && !unicode.IsSpace(r) {
		ok = !fc.glyph(idx, false).empty
	}
	fc.has[r] = ok
	return ok
}

func (fc *face) glyph(idx sfnt.GlyphIndex, oblique bool) *glyphMask {
	k := glyphKey{idx, oblique}
	if g, ok := fc.glyphs[k]; ok {
		return g
	}
	g := &glyphMask{}
	fc.glyphs[k] = g
	if adv, err := fc.f.GlyphAdvance(&fc.buf, idx, fc.ppem, font.HintingNone); err == nil {
		g.adv = adv
	}
	segs, err := fc.f.LoadGlyph(&fc.buf, idx, fc.ppem, nil)
	if err != nil || len(segs) == 0 {
		g.empty = true
		return g
	}
	shear := float32(0)
	if oblique {
		shear = obliqueShear
	}
	tx := func(p fixed.Point26_6) (float32, float32) {
		x, y := float32(p.X)/64, float32(p.Y)/64
		return x - y*shear, y
	}
	minX, minY := float32(math.Inf(1)), float32(math.Inf(1))
	maxX, maxY := float32(math.Inf(-1)), float32(math.Inf(-1))
	rawMin, rawMax := float32(math.Inf(1)), float32(math.Inf(-1))
	for _, s := range segs {
		n := 1
		switch s.Op {
		case sfnt.SegmentOpQuadTo:
			n = 2
		case sfnt.SegmentOpCubeTo:
			n = 3
		}
		for _, p := range s.Args[:n] {
			x, y := tx(p)
			minX, maxX = min(minX, x), max(maxX, x)
			minY, maxY = min(minY, y), max(maxY, y)
			rx := float32(p.X) / 64
			rawMin, rawMax = min(rawMin, rx), max(rawMax, rx)
		}
	}
	g.inkL = int(math.Floor(float64(rawMin)))
	g.inkR = int(math.Ceil(float64(rawMax)))
	g.ox = int(math.Floor(float64(minX)))
	g.oy = int(math.Floor(float64(minY)))
	w := int(math.Ceil(float64(maxX))) - g.ox
	h := int(math.Ceil(float64(maxY))) - g.oy
	if w <= 0 || h <= 0 {
		g.empty = true
		return g
	}
	var z vector.Rasterizer
	z.Reset(w, h)
	z.DrawOp = draw.Src
	fx, fy := float32(g.ox), float32(g.oy)
	for i, s := range segs {
		switch s.Op {
		case sfnt.SegmentOpMoveTo:
			if i > 0 {
				z.ClosePath()
			}
			x, y := tx(s.Args[0])
			z.MoveTo(x-fx, y-fy)
		case sfnt.SegmentOpLineTo:
			x, y := tx(s.Args[0])
			z.LineTo(x-fx, y-fy)
		case sfnt.SegmentOpQuadTo:
			x1, y1 := tx(s.Args[0])
			x2, y2 := tx(s.Args[1])
			z.QuadTo(x1-fx, y1-fy, x2-fx, y2-fy)
		case sfnt.SegmentOpCubeTo:
			x1, y1 := tx(s.Args[0])
			x2, y2 := tx(s.Args[1])
			x3, y3 := tx(s.Args[2])
			z.CubeTo(x1-fx, y1-fy, x2-fx, y2-fy, x3-fx, y3-fy)
		}
	}
	z.ClosePath()
	g.mask = image.NewAlpha(image.Rect(0, 0, w, h))
	z.Draw(g.mask, g.mask.Bounds(), image.Opaque, image.Point{})
	return g
}

// fontSet is cellshot.py's Fonts: the faces at one cell height, and the per-character
// choice of face.
type fontSet struct {
	reg, bold *face
	fallbacks []*face
	picks     map[pickKey]*face
}

type pickKey struct {
	r          rune
	bold, ital bool
}

// newFontSet sizes the faces like cellshot.py: 80% of the cell height.
func newFontSet(px int) *fontSet {
	parsedFonts()
	size := int(float64(px) * 0.80)
	return &fontSet{
		reg:       newFace(fontRegular, size),
		bold:      newFace(fontBold, size),
		fallbacks: []*face{newFace(fontSymbols, size)},
		picks:     map[pickKey]*face{},
	}
}

// pick returns the face that draws r, or nil when none can. Bold wins over italic,
// as in cellshot.py; italic uses the regular face, slanted at draw time.
func (fs *fontSet) pick(r rune, bold, italic bool) *face {
	k := pickKey{r, bold, italic}
	if f, ok := fs.picks[k]; ok {
		return f
	}
	base := fs.reg
	if bold {
		base = fs.bold
	}
	var f *face
	if base.covers(r) {
		f = base
	} else {
		for _, fb := range fs.fallbacks {
			if fb.covers(r) {
				f = fb
				break
			}
		}
	}
	fs.picks[k] = f
	return f
}

// pixel is Pillow's PIXEL macro: 26.6 to whole pixels, rounding half up.
func pixel(x fixed.Int26_6) int { return int((x + 32) &^ 63 >> 6) }

// layout positions the runes of s in face fc: per-glyph pen offsets in whole pixels,
// and the Pillow-style text box (x_min ≤ 0, x_max ≥ advance). Like HarfBuzz under
// Pillow, it composes the cluster first (e + U+0301 draws é); a mark left over sits on
// its base's origin with no advance of its own.
func (fc *face) layout(s string, oblique bool) (gs []*glyphMask, xs []int, xmin, xmax int) {
	var pos, last fixed.Int26_6
	for i, r := range norm.NFC.String(s) {
		g := fc.glyph(fc.index(r), oblique)
		if i > 0 && unicode.Is(unicode.Mn, r) {
			if !g.empty {
				xmin = min(xmin, g.inkL+pixel(last))
				xmax = max(xmax, g.inkR+pixel(last))
			}
			gs = append(gs, g)
			xs = append(xs, pixel(last))
			continue
		}
		last = pos
		px := pixel(pos)
		pos += g.adv
		xmax = max(xmax, pixel(pos))
		if !g.empty {
			xmin = min(xmin, g.inkL+px)
			xmax = max(xmax, g.inkR+px)
		}
		gs = append(gs, g)
		xs = append(xs, px)
	}
	return gs, xs, xmin, xmax
}

// text draws s with its ascender line at y, pen starting at x.
func (c *canvas) text(fc *face, s string, x, y int, ink RGB, oblique bool) {
	gs, xs, _, _ := fc.layout(s, oblique)
	base := y + fc.asc
	for i, g := range gs {
		if g.empty {
			continue
		}
		c.blendMask(g.mask, x+xs[i]+g.ox, base+g.oy, ink)
	}
}

// blendMask composites ink through an 8-bit coverage mask with Pillow's BLEND/DIV255
// arithmetic (fill_mask_L).
func (c *canvas) blendMask(m *image.Alpha, x, y int, ink RGB) {
	b := m.Bounds()
	for my := 0; my < b.Dy(); my++ {
		py := y + my
		if py < 0 || py >= c.h {
			continue
		}
		row := m.Pix[my*m.Stride : my*m.Stride+b.Dx()]
		for mx, a := range row {
			px := x + mx
			if a == 0 || px < 0 || px >= c.w {
				continue
			}
			i := py*c.Stride + px*4
			c.Pix[i] = blend8(a, c.Pix[i], ink.R)
			c.Pix[i+1] = blend8(a, c.Pix[i+1], ink.G)
			c.Pix[i+2] = blend8(a, c.Pix[i+2], ink.B)
		}
	}
}

func blend8(mask, in1, in2 uint8) uint8 {
	t := uint32(in1)*uint32(255-mask) + uint32(in2)*uint32(mask) + 128
	return uint8(((t >> 8) + t) >> 8)
}
