package cellshot

// Geometric glyphs, drawn the way cellshot.py draws them: box drawing (light, heavy,
// double, rounded, dashed, diagonals), block elements and eighths, shades, quadrants,
// sextants, braille and powerline separators. Never font glyphs, so they join and fill
// like a modern terminal's built-in box drawing.

// boxSpec is (up, right, down, left) weights: 1 light, 2 heavy, 3 double.
type boxSpec [4]int

var boxTable = map[rune]boxSpec{}

func defBox(chars string, specs []boxSpec) {
	i := 0
	for _, r := range chars {
		boxTable[r] = specs[i]
		i++
	}
	if i != len(specs) {
		panic("cellshot: box table length mismatch for " + chars)
	}
}

func init() {
	defBox("─━│┃", []boxSpec{{0, 1, 0, 1}, {0, 2, 0, 2}, {1, 0, 1, 0}, {2, 0, 2, 0}})
	defBox("┌┍┎┏┐┑┒┓└┕┖┗┘┙┚┛", []boxSpec{
		{0, 1, 1, 0}, {0, 2, 1, 0}, {0, 1, 2, 0}, {0, 2, 2, 0}, {0, 0, 1, 1}, {0, 0, 1, 2}, {0, 0, 2, 1}, {0, 0, 2, 2},
		{1, 1, 0, 0}, {1, 2, 0, 0}, {2, 1, 0, 0}, {2, 2, 0, 0}, {1, 0, 0, 1}, {1, 0, 0, 2}, {2, 0, 0, 1}, {2, 0, 0, 2}})
	defBox("├┝┞┟┠┡┢┣┤┥┦┧┨┩┪┫", []boxSpec{
		{1, 1, 1, 0}, {1, 2, 1, 0}, {2, 1, 1, 0}, {1, 1, 2, 0}, {2, 1, 2, 0}, {2, 2, 1, 0}, {1, 2, 2, 0}, {2, 2, 2, 0},
		{1, 0, 1, 1}, {1, 0, 1, 2}, {2, 0, 1, 1}, {1, 0, 2, 1}, {2, 0, 2, 1}, {2, 0, 1, 2}, {1, 0, 2, 2}, {2, 0, 2, 2}})
	defBox("┬┭┮┯┰┱┲┳┴┵┶┷┸┹┺┻", []boxSpec{
		{0, 1, 1, 1}, {0, 1, 1, 2}, {0, 2, 1, 1}, {0, 2, 1, 2}, {0, 1, 2, 1}, {0, 1, 2, 2}, {0, 2, 2, 1}, {0, 2, 2, 2},
		{1, 1, 0, 1}, {1, 1, 0, 2}, {1, 2, 0, 1}, {1, 2, 0, 2}, {2, 1, 0, 1}, {2, 1, 0, 2}, {2, 2, 0, 1}, {2, 2, 0, 2}})
	defBox("┼┽┾┿╀╁╂╃╄╅╆╇╈╉╊╋", []boxSpec{
		{1, 1, 1, 1}, {1, 1, 1, 2}, {1, 2, 1, 1}, {1, 2, 1, 2}, {2, 1, 1, 1}, {1, 1, 2, 1}, {2, 1, 2, 1}, {2, 1, 1, 2},
		{2, 2, 1, 1}, {1, 1, 2, 2}, {1, 2, 2, 1}, {2, 2, 1, 2}, {1, 2, 2, 2}, {2, 1, 2, 2}, {2, 2, 2, 1}, {2, 2, 2, 2}})
	defBox("═║╒╓╔╕╖╗╘╙╚╛╜╝╞╟╠╡╢╣╤╥╦╧╨╩╪╫╬", []boxSpec{
		{0, 3, 0, 3}, {3, 0, 3, 0}, {0, 3, 1, 0}, {0, 1, 3, 0}, {0, 3, 3, 0}, {0, 0, 1, 3}, {0, 0, 3, 1}, {0, 0, 3, 3},
		{1, 3, 0, 0}, {3, 1, 0, 0}, {3, 3, 0, 0}, {1, 0, 0, 3}, {3, 0, 0, 1}, {3, 0, 0, 3}, {1, 3, 1, 0}, {3, 1, 3, 0},
		{3, 3, 3, 0}, {1, 0, 1, 3}, {3, 0, 3, 1}, {3, 0, 3, 3}, {0, 3, 1, 3}, {0, 1, 3, 1}, {0, 3, 3, 3}, {1, 3, 0, 3},
		{3, 1, 0, 1}, {3, 3, 0, 3}, {1, 3, 1, 3}, {3, 1, 3, 1}, {3, 3, 3, 3}})
	defBox("╴╵╶╷╸╹╺╻╼╽╾╿", []boxSpec{
		{0, 0, 0, 1}, {1, 0, 0, 0}, {0, 1, 0, 0}, {0, 0, 1, 0}, {0, 0, 0, 2}, {2, 0, 0, 0}, {0, 2, 0, 0}, {0, 0, 2, 0},
		{0, 2, 0, 1}, {1, 0, 2, 0}, {0, 1, 0, 2}, {2, 0, 1, 0}})
	for k, r := range []rune("▁▂▃▄▅▆▇") {
		blockTable[r] = []blockRect{{0, 7 - k, 8, 8}}
	}
	for k, r := range []rune("▏▎▍▌▋▊▉") {
		blockTable[r] = []blockRect{{0, 0, k + 1, 8}}
	}
}

// dashSpec is (vertical, segments, weight).
type dashSpec struct {
	vert      bool
	n, weight int
}

var dashTable = map[rune]dashSpec{
	'┄': {false, 3, 1}, '┅': {false, 3, 2}, '┆': {true, 3, 1}, '┇': {true, 3, 2},
	'┈': {false, 4, 1}, '┉': {false, 4, 2}, '┊': {true, 4, 1}, '┋': {true, 4, 2},
	'╌': {false, 2, 1}, '╍': {false, 2, 2}, '╎': {true, 2, 1}, '╏': {true, 2, 2},
}

var arcTable = map[rune]string{'╭': "dr", '╮': "dl", '╯': "ul", '╰': "ur"}

// blockRect is (x0, y0, x1, y1) in eighths of the cell.
type blockRect [4]int

var blockTable = map[rune][]blockRect{
	'█': {{0, 0, 8, 8}}, '▀': {{0, 0, 8, 4}}, '▄': {{0, 4, 8, 8}}, '▌': {{0, 0, 4, 8}}, '▐': {{4, 0, 8, 8}},
	'▔': {{0, 0, 8, 1}}, '▕': {{7, 0, 8, 8}},
	'▖': {{0, 4, 4, 8}}, '▗': {{4, 4, 8, 8}}, '▘': {{0, 0, 4, 4}}, '▝': {{4, 0, 8, 4}},
	'▙': {{0, 0, 4, 8}, {4, 4, 8, 8}}, '▚': {{0, 0, 4, 4}, {4, 4, 8, 8}}, '▛': {{0, 0, 8, 4}, {0, 4, 4, 8}},
	'▜': {{0, 0, 8, 4}, {4, 4, 8, 8}}, '▞': {{4, 0, 8, 4}, {0, 4, 4, 8}}, '▟': {{4, 0, 8, 8}, {0, 4, 4, 8}},
}

var shadeTable = map[rune]float64{'░': 0.25, '▒': 0.5, '▓': 0.75}

// isGeometric reports whether cellshot draws r itself rather than with a font.
func isGeometric(r rune) bool {
	if _, ok := boxTable[r]; ok {
		return true
	}
	if _, ok := dashTable[r]; ok {
		return true
	}
	if _, ok := arcTable[r]; ok {
		return true
	}
	if _, ok := blockTable[r]; ok {
		return true
	}
	if _, ok := shadeTable[r]; ok {
		return true
	}
	switch {
	case r == '╱' || r == '╲' || r == '╳':
		return true
	case r >= 0x2800 && r <= 0x28FF:
		return true
	case r >= 0x1FB00 && r <= 0x1FB3B:
		return true
	case r == 0xE0B0 || r == 0xE0B2 || r == 0xE0B4 || r == 0xE0B6:
		return true
	}
	return false
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (c *canvas) drawBox(x0, y0, cw, rh int, spec boxSpec, col RGB, lw int) {
	u, r, dn, l := spec[0], spec[1], spec[2], spec[3]
	mx, my := x0+cw/2, y0+rh/2
	x1, y1 := x0+cw, y0+rh
	gap := max(2, lw+1)
	seg := func(weight int, single [2]pt, double [2][2]pt) {
		switch weight {
		case 0:
		case 3:
			for _, p := range double {
				c.lines(p[:], col, lw)
			}
		default:
			w := lw
			if weight == 2 {
				w = lw * 2
			}
			c.lines(single[:], col, w)
		}
	}
	// For doubles, run the two strokes to the centre lines of the crossing direction.
	hz := gap * b2i(l != 0 || r != 0)
	vt := gap * b2i(u != 0 || dn != 0)
	seg(u, [2]pt{ipt(mx, y0), ipt(mx, my)},
		[2][2]pt{{ipt(mx-gap, y0), ipt(mx-gap, my+hz)}, {ipt(mx+gap, y0), ipt(mx+gap, my+hz)}})
	seg(dn, [2]pt{ipt(mx, my), ipt(mx, y1)},
		[2][2]pt{{ipt(mx-gap, my-hz), ipt(mx-gap, y1)}, {ipt(mx+gap, my-hz), ipt(mx+gap, y1)}})
	seg(l, [2]pt{ipt(x0, my), ipt(mx, my)},
		[2][2]pt{{ipt(x0, my-gap), ipt(mx+vt, my-gap)}, {ipt(x0, my+gap), ipt(mx+vt, my+gap)}})
	seg(r, [2]pt{ipt(mx, my), ipt(x1, my)},
		[2][2]pt{{ipt(mx-vt, my-gap), ipt(x1, my-gap)}, {ipt(mx-vt, my+gap), ipt(x1, my+gap)}})
}

// drawGeometric draws base at (x0, y0) if it is a geometric glyph and reports whether
// it did. eff is the effective background, used by shades.
func (c *canvas) drawGeometric(base rune, x0, y0, cw, rh int, fg, eff RGB) bool {
	lw := max(1, cw/8)
	if spec, ok := boxTable[base]; ok {
		c.drawBox(x0, y0, cw, rh, spec, fg, lw)
		return true
	}
	if d, ok := dashTable[base]; ok {
		L := cw
		if d.vert {
			L = rh
		}
		step := float64(L) / float64(d.n)
		for k := 0; k < d.n; k++ {
			ks := float64(float64(k) * step)
			a := ks + float64(step*0.15)
			b := ks + float64(step*0.75)
			if d.vert {
				x := float64(x0 + cw/2)
				c.lines([]pt{{x, float64(y0) + a}, {x, float64(y0) + b}}, fg, lw*d.weight)
			} else {
				y := float64(y0 + rh/2)
				c.lines([]pt{{float64(x0) + a, y}, {float64(x0) + b, y}}, fg, lw*d.weight)
			}
		}
		return true
	}
	if kind, ok := arcTable[base]; ok {
		mx, my := x0+cw/2, y0+rh/2
		rad := min(cw, rh) / 2
		var bx [4]int
		var a0, a1 float32
		// Straight parts to the cell edge, then a quarter circle around the corner.
		switch kind {
		case "dr":
			bx, a0, a1 = [4]int{mx, my, mx + 2*rad, my + 2*rad}, 180, 270
			c.lines([]pt{ipt(mx, my+rad), ipt(mx, y0+rh)}, fg, lw)
			c.lines([]pt{ipt(mx+rad, my), ipt(x0+cw, my)}, fg, lw)
		case "dl":
			bx, a0, a1 = [4]int{mx - 2*rad, my, mx, my + 2*rad}, 270, 360
			c.lines([]pt{ipt(mx, my+rad), ipt(mx, y0+rh)}, fg, lw)
			c.lines([]pt{ipt(x0, my), ipt(mx-rad, my)}, fg, lw)
		case "ul":
			bx, a0, a1 = [4]int{mx - 2*rad, my - 2*rad, mx, my}, 0, 90
			c.lines([]pt{ipt(mx, y0), ipt(mx, my-rad)}, fg, lw)
			c.lines([]pt{ipt(x0, my), ipt(mx-rad, my)}, fg, lw)
		default: // ur
			bx, a0, a1 = [4]int{mx, my - 2*rad, mx + 2*rad, my}, 90, 180
			c.lines([]pt{ipt(mx, y0), ipt(mx, my-rad)}, fg, lw)
			c.lines([]pt{ipt(mx+rad, my), ipt(x0+cw, my)}, fg, lw)
		}
		c.arc(bx[0], bx[1], bx[2], bx[3], a0, a1, fg, lw)
		return true
	}
	if base == '╱' || base == '╲' || base == '╳' {
		if base != '╲' {
			c.lines([]pt{ipt(x0, y0+rh-1), ipt(x0+cw-1, y0)}, fg, lw)
		}
		if base != '╱' {
			c.lines([]pt{ipt(x0, y0), ipt(x0+cw-1, y0+rh-1)}, fg, lw)
		}
		return true
	}
	if rects, ok := blockTable[base]; ok {
		for _, q := range rects {
			c.rect(x0+q[0]*cw/8, y0+q[1]*rh/8, x0+q[2]*cw/8-1, y0+q[3]*rh/8-1, fg)
		}
		return true
	}
	if t, ok := shadeTable[base]; ok {
		c.rect(x0, y0, x0+cw-1, y0+rh-1, mix(eff, fg, t))
		return true
	}
	if base >= 0x2800 && base <= 0x28FF {
		bits := int(base - 0x2800)
		dots := [8][3]int{{0, 0, 0x01}, {0, 1, 0x02}, {0, 2, 0x04}, {1, 0, 0x08}, {1, 1, 0x10}, {1, 2, 0x20}, {0, 3, 0x40}, {1, 3, 0x80}}
		rr := max(1, cw/6)
		for _, d := range dots {
			if bits&d[2] != 0 {
				px := x0 + (d[0]*2+1)*cw/4
				py := y0 + (d[1]*2+1)*rh/8
				c.ellipse(px-rr, py-rr, px+rr, py+rr, fg)
			}
		}
		return true
	}
	if base >= 0x1FB00 && base <= 0x1FB3B { // sextants, 2x3
		k := int(base-0x1FB00) + 1
		if k >= 21 {
			k++
		}
		if k >= 42 {
			k++
		}
		for bit := 0; bit < 6; bit++ {
			if k&(1<<bit) != 0 {
				cx, cy := bit%2, bit/2
				c.rect(x0+cx*cw/2, y0+cy*rh/3, x0+(cx+1)*cw/2-1, y0+(cy+1)*rh/3-1, fg)
			}
		}
		return true
	}
	switch base {
	case 0xE0B0, 0xE0B4:
		c.polygon([]pt{ipt(x0, y0), ipt(x0+cw-1, y0+rh/2), ipt(x0, y0+rh-1)}, fg)
		return true
	case 0xE0B2, 0xE0B6:
		c.polygon([]pt{ipt(x0+cw-1, y0), ipt(x0, y0+rh/2), ipt(x0+cw-1, y0+rh-1)}, fg)
		return true
	}
	return false
}
