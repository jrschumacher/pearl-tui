package cellshot

// A port of the parts of Pillow's libImaging/Draw.c (12.x) that cellshot.py reaches
// through ImageDraw: thin and wide lines, filled rectangles, filled polygons, filled
// ellipses and arcs. Geometric cells are drawn with these so that a Go render is
// pixel-identical to a Python one at the same cell size. The C code's integer and
// float32 arithmetic is kept as written, including its rounding macros; explicit
// conversions stop the compiler fusing multiply-adds, which C does not do either.

import (
	"image"
	"math"
	"sort"
)

// canvas is an opaque RGB image (alpha is always 255).
type canvas struct {
	*image.RGBA
	w, h int
}

func newCanvas(w, h int, bg RGB) *canvas {
	c := &canvas{RGBA: image.NewRGBA(image.Rect(0, 0, w, h)), w: w, h: h}
	p := c.Pix
	for i := 0; i < len(p); i += 4 {
		p[i], p[i+1], p[i+2], p[i+3] = bg.R, bg.G, bg.B, 255
	}
	return c
}

func (c *canvas) at(x, y int) RGB {
	i := y*c.Stride + x*4
	return RGB{c.Pix[i], c.Pix[i+1], c.Pix[i+2]}
}

// point is Draw.c point32: a bounds-checked pixel write.
func (c *canvas) point(x, y int, ink RGB) {
	if x >= 0 && x < c.w && y >= 0 && y < c.h {
		i := y*c.Stride + x*4
		c.Pix[i], c.Pix[i+1], c.Pix[i+2] = ink.R, ink.G, ink.B
	}
}

// hline is Draw.c hline32: x0..x1 inclusive, clipped.
func (c *canvas) hline(x0, y0, x1 int, ink RGB) {
	if y0 < 0 || y0 >= c.h {
		return
	}
	if x0 < 0 {
		x0 = 0
	} else if x0 >= c.w {
		return
	}
	if x1 < 0 {
		return
	} else if x1 >= c.w {
		x1 = c.w - 1
	}
	i := y0*c.Stride + x0*4
	for ; x0 <= x1; x0++ {
		c.Pix[i], c.Pix[i+1], c.Pix[i+2] = ink.R, ink.G, ink.B
		i += 4
	}
}

// line is Draw.c line32: Bresenham, end point excluded.
func (c *canvas) line(x0, y0, x1, y1 int, ink RGB) {
	dx, xs := x1-x0, 1
	if dx < 0 {
		dx, xs = -dx, -1
	}
	dy, ys := y1-y0, 1
	if dy < 0 {
		dy, ys = -dy, -1
	}
	switch {
	case dx == 0:
		for i := 0; i < dy; i++ {
			c.point(x0, y0, ink)
			y0 += ys
		}
	case dy == 0:
		for i := 0; i < dx; i++ {
			c.point(x0, y0, ink)
			x0 += xs
		}
	case dx > dy:
		n := dx
		dy += dy
		e := dy - dx
		dx += dx
		for i := 0; i < n; i++ {
			c.point(x0, y0, ink)
			if e >= 0 {
				y0 += ys
				e -= dx
			}
			e += dy
			x0 += xs
		}
	default:
		n := dy
		dx += dx
		e := dx - dy
		dy += dy
		for i := 0; i < n; i++ {
			c.point(x0, y0, ink)
			if e >= 0 {
				x0 += xs
				e -= dy
			}
			e += dx
			y0 += ys
		}
	}
}

// pt is a coordinate as Python passes it: ImageDraw truncates each to int.
type pt struct{ x, y float64 }

func ipt(x, y int) pt { return pt{float64(x), float64(y)} }

// lines is ImageDraw.line (_draw_lines): a polyline of the given width.
func (c *canvas) lines(xy []pt, ink RGB, width int) {
	if width == 0 || len(xy) == 0 {
		return
	}
	if width == 1 {
		var last *pt
		for i := 0; i+1 < len(xy); i++ {
			p, q := xy[i], xy[i+1]
			c.line(int(p.x), int(p.y), int(q.x), int(q.y), ink)
			last = &xy[i+1]
		}
		if last != nil {
			c.point(int(last.x), int(last.y), ink)
		}
		return
	}
	for i := 0; i+1 < len(xy); i++ {
		p, q := xy[i], xy[i+1]
		c.wideLine(int(p.x), int(p.y), int(q.x), int(q.y), ink, width)
	}
}

// The rounding macros of Draw.c. Applied to a float they add 0.5F in float arithmetic,
// applied to a double in double arithmetic.
func roundUp64(f float64) int {
	if f >= 0 {
		return int(math.Floor(f + 0.5))
	}
	return -int(math.Floor(math.Abs(f) + 0.5))
}

func roundDown64(f float64) int {
	if f >= 0 {
		return int(math.Ceil(f - 0.5))
	}
	return -int(math.Ceil(math.Abs(f) - 0.5))
}

func roundUp32(f float32) int {
	if f >= 0 {
		return int(math.Floor(float64(f + 0.5)))
	}
	return -int(math.Floor(float64(float32(math.Abs(float64(f))) + 0.5)))
}

func roundDown32(f float32) int {
	if f >= 0 {
		return int(math.Ceil(float64(f - 0.5)))
	}
	return -int(math.Ceil(float64(float32(math.Abs(float64(f))) - 0.5)))
}

// wideLine is ImagingDrawWideLine.
func (c *canvas) wideLine(x0, y0, x1, y1 int, ink RGB, width int) {
	dx, dy := x1-x0, y1-y0
	if dx == 0 && dy == 0 {
		c.point(x0, y0, ink)
		return
	}
	big := math.Hypot(float64(dx), float64(dy))
	small := float64(width-1) / 2.0
	ratioMax := float64(roundUp64(small)) / big
	ratioMin := float64(roundDown64(small)) / big
	dxmin := roundDown64(float64(ratioMin * float64(dy)))
	dxmax := roundDown64(float64(ratioMax * float64(dy)))
	dymin := roundDown64(float64(ratioMin * float64(dx)))
	dymax := roundDown64(float64(ratioMax * float64(dx)))
	v := [4][2]int{
		{x0 - dxmin, y0 + dymax},
		{x1 - dxmin, y1 + dymax},
		{x1 + dxmax, y1 - dymin},
		{x0 + dxmax, y0 - dymin},
	}
	e := []edge{
		newEdge(v[0][0], v[0][1], v[1][0], v[1][1]),
		newEdge(v[1][0], v[1][1], v[2][0], v[2][1]),
		newEdge(v[2][0], v[2][1], v[3][0], v[3][1]),
		newEdge(v[3][0], v[3][1], v[0][0], v[0][1]),
	}
	c.polygonEdges(e, ink)
}

// rect is ImageDraw.rectangle with a fill. Pillow raises on reversed corners; this
// draws nothing instead.
func (c *canvas) rect(x0, y0, x1, y1 int, ink RGB) {
	if x1 < x0 || y1 < y0 {
		return
	}
	if y0 < 0 {
		y0 = 0
	} else if y0 >= c.h {
		return
	}
	if y1 < 0 {
		return
	} else if y1 > c.h {
		y1 = c.h
	}
	for y := y0; y <= y1; y++ {
		c.hline(x0, y, x1, ink)
	}
}

// rectOutline is ImageDraw.rectangle with an outline of width 1.
func (c *canvas) rectOutline(x0, y0, x1, y1 int, ink RGB) {
	if x1 < x0 || y1 < y0 {
		return
	}
	c.hline(x0, y0, x1, ink)
	c.hline(x0, y1, x1, ink)
	c.line(x1, y0+1, x1, y1, ink)
	c.line(x0, y0+1, x0, y1, ink)
}

type edge struct {
	d                      int
	x0, y0                 int
	xmin, ymin, xmax, ymax int
	dx                     float32
}

func newEdge(x0, y0, x1, y1 int) edge {
	var e edge
	if x0 <= x1 {
		e.xmin, e.xmax = x0, x1
	} else {
		e.xmin, e.xmax = x1, x0
	}
	if y0 <= y1 {
		e.ymin, e.ymax = y0, y1
	} else {
		e.ymin, e.ymax = y1, y0
	}
	if y0 == y1 {
		e.d = 0
		e.dx = 0
	} else {
		e.dx = float32(x1-x0) / float32(y1-y0)
		if y0 == e.ymin {
			e.d = 1
		} else {
			e.d = -1
		}
	}
	e.x0, e.y0 = x0, y0
	return e
}

func (e *edge) xAt(y int) float32 {
	return float32(float32(y-e.y0)*e.dx) + float32(e.x0)
}

func roundf(f float32) float32 { return float32(math.Round(float64(f))) }

// polygonEdges is polygon_generic without a mask or alpha: the scan-line fill.
func (c *canvas) polygonEdges(e []edge, ink RGB) {
	n := len(e)
	if n == 0 {
		return
	}
	ymin, ymax := c.h-1, 0
	table := make([]*edge, 0, n)
	for i := range e {
		if ymin > e[i].ymin {
			ymin = e[i].ymin
		}
		if ymax < e[i].ymax {
			ymax = e[i].ymax
		}
		if e[i].ymin == e[i].ymax {
			c.hline(e[i].xmin, e[i].ymin, e[i].xmax, ink)
			continue
		}
		table = append(table, &e[i])
	}
	if ymin < 0 {
		ymin = 0
	}
	if ymax > c.h {
		ymax = c.h
	}
	xx := make([]float32, 0, 2*len(table))
	for ; ymin <= ymax; ymin++ {
		xx = xx[:0]
		for i, cur := range table {
			if ymin < cur.ymin || ymin > cur.ymax {
				continue
			}
			xx = append(xx, cur.xAt(ymin))
			j := len(xx)
			if ymin == cur.ymax && ymin < ymax {
				xx = append(xx, xx[j-1])
				continue
			}
			if (ymin == cur.ymin || ymin == cur.ymax) && cur.dx != 0 {
				for k := 0; k < i; k++ {
					other := table[k]
					if (ymin != other.ymin && ymin != other.ymax) || other.dx == 0 {
						continue
					}
					if roundf(xx[j-1]) != roundf(other.xAt(ymin)) {
						continue
					}
					offset := 1
					if ymin == cur.ymax {
						offset = -1
					}
					adj := cur.xAt(ymin + offset)
					if ymin+offset >= other.ymin && ymin+offset <= other.ymax {
						adjOther := other.xAt(ymin + offset)
						if xx[j-1] > adj+1 && xx[j-1] > adjOther+1 {
							xx[j-1] = roundf(float32(math.Max(float64(adj), float64(adjOther)))) + 1
						} else if xx[j-1] < adj-1 && xx[j-1] < adjOther-1 {
							xx[j-1] = roundf(float32(math.Min(float64(adj), float64(adjOther)))) - 1
						}
						break
					}
				}
			}
		}
		sort.Slice(xx, func(a, b int) bool { return xx[a] < xx[b] })
		for i := 1; i < len(xx); i += 2 {
			c.hline(roundUp32(xx[i-1]), ymin, roundDown32(xx[i]), ink)
		}
	}
}

// polygon is ImageDraw.polygon with a fill (ImagingDrawPolygon, fill branch).
func (c *canvas) polygon(xy []pt, ink RGB) {
	count := len(xy)
	if count == 0 {
		return
	}
	p := make([]int, 2*count)
	for i, q := range xy {
		p[2*i], p[2*i+1] = int(q.x), int(q.y)
	}
	e := make([]edge, 0, count)
	i := 0
	for ; i < count-1; i++ {
		x0, y0, x1, y1 := p[i*2], p[i*2+1], p[i*2+2], p[i*2+3]
		if y0 == y1 && i != 0 && y0 == p[i*2-1] {
			last := &e[len(e)-1]
			if x1 > x0 && x0 > p[i*2-2] {
				last.xmax = x1
				continue
			} else if x1 < x0 && x0 < p[i*2-2] {
				last.xmin = x1
				continue
			}
		}
		e = append(e, newEdge(x0, y0, x1, y1))
	}
	if p[i*2] != p[0] || p[i*2+1] != p[1] {
		e = append(e, newEdge(p[i*2], p[i*2+1], p[0], p[1]))
	}
	c.polygonEdges(e, ink)
}

// ---------------------------------------------------------------- ellipses and arcs

type quarterState struct {
	a, b, cx, cy, ex, ey int32
	a2, b2, a2b2         int64
	finished             bool
}

func (s *quarterState) init(a, b int32) {
	if a < 0 || b < 0 {
		s.finished = true
		return
	}
	s.a, s.b = a, b
	s.cx, s.cy = a, b%2
	s.ex, s.ey = a%2, b
	s.a2 = int64(a) * int64(a)
	s.b2 = int64(b) * int64(b)
	s.a2b2 = s.a2 * s.b2
	s.finished = false
}

func (s *quarterState) delta(x, y int64) int64 {
	d := s.a2*y*y + s.b2*x*x - s.a2b2
	if d < 0 {
		return -d
	}
	return d
}

func (s *quarterState) next() (x, y int32, ok bool) {
	if s.finished {
		return 0, 0, false
	}
	x, y = s.cx, s.cy
	if s.cx == s.ex && s.cy == s.ey {
		s.finished = true
	} else {
		nx, ny := s.cx, s.cy+2
		nd := s.delta(int64(nx), int64(ny))
		if nx > 1 {
			d := s.delta(int64(s.cx-2), int64(s.cy+2))
			if nd > d {
				nx, ny, nd = s.cx-2, s.cy+2, d
			}
			d = s.delta(int64(s.cx-2), int64(s.cy))
			if nd > d {
				nx, ny = s.cx-2, s.cy
			}
		}
		s.cx, s.cy = nx, ny
	}
	return x, y, true
}

type ellipseState struct {
	o, i       quarterState
	py, pl, pr int32
	cy, cl, cr [4]int32
	bufcnt     int
	finished   bool
	leftmost   int32
}

func (s *ellipseState) init(a, b, w int32) {
	s.bufcnt = 0
	s.leftmost = a % 2
	s.o.init(a, b)
	var ok bool
	if w >= 1 {
		s.pr, s.py, ok = s.o.next()
	}
	if w < 1 || !ok {
		s.finished = true
		return
	}
	s.finished = false
	s.i.init(a-2*(w-1), b-2*(w-1))
	s.pl = s.leftmost
}

func (s *ellipseState) next() (x0, y, x1 int32, ok bool) {
	if s.bufcnt == 0 {
		if s.finished {
			return 0, 0, 0, false
		}
		y := s.py
		l := s.pl
		r := s.pr
		var cx, cy int32
		var nok bool
		for {
			cx, cy, nok = s.o.next()
			if !nok || cy > y {
				break
			}
		}
		if !nok {
			s.finished = true
		} else {
			s.pr, s.py = cx, cy
		}
		for {
			cx, cy, nok = s.i.next()
			if !nok || cy > y {
				break
			}
			l = cx
		}
		if !nok {
			s.pl = s.leftmost
		} else {
			s.pl = cx
		}
		push := func(cl, cy, cr int32) {
			s.cl[s.bufcnt], s.cy[s.bufcnt], s.cr[s.bufcnt] = cl, cy, cr
			s.bufcnt++
		}
		lz := l
		if l == 0 {
			lz = 2
		}
		if (l > 0 || l < r) && y > 0 {
			push(lz, y, r)
		}
		if y > 0 {
			push(-r, y, -l)
		}
		if l > 0 || l < r {
			push(lz, -y, r)
		}
		push(-r, -y, -l)
	}
	s.bufcnt--
	return s.cl[s.bufcnt], s.cy[s.bufcnt], s.cr[s.bufcnt], true
}

// ellipse is ImageDraw.ellipse with a fill (ellipseNew with fill set).
func (c *canvas) ellipse(x0, y0, x1, y1 int, ink RGB) {
	a, b := x1-x0, y1-y0
	if a < 0 || b < 0 {
		return
	}
	var st ellipseState
	st.init(int32(a), int32(b), int32(a+b))
	for {
		X0, Y, X1, ok := st.next()
		if !ok {
			break
		}
		c.hline(x0+(int(X0)+a)/2, y0+(int(Y)+b)/2, x0+(int(X1)+a)/2, ink)
	}
}

type clipKind int

const (
	ctAnd clipKind = iota
	ctOr
	ctClip
)

type clipNode struct {
	kind    clipKind
	a, b, c float64
	l, r    *clipNode
}

type event struct {
	x   int32
	typ int8
}

func (n *clipNode) transpose() {
	if n == nil {
		return
	}
	if n.kind == ctClip {
		n.a, n.b = n.b, n.a
	}
	n.l.transpose()
	n.r.transpose()
}

func clipDo(root *clipNode, x0, y, x1 int32) []event {
	if root == nil {
		return []event{{x0, 1}, {x1, -1}}
	}
	if root.kind == ctClip {
		const eps = 1e-9
		A, B, C := root.a, root.b, root.c
		if math.Abs(A) < eps {
			if float64(B*float64(y))+C < -eps {
				x0, x1 = 1, 0
			}
		} else {
			ix := -(float64(B*float64(y)) + C) / A
			if float64(A*float64(x0))+float64(B*float64(y))+C < eps {
				x0 = int32(math.Round(math.Max(float64(x0), ix)))
			}
			if float64(A*float64(x1))+float64(B*float64(y))+C < eps {
				x1 = int32(math.Round(math.Min(float64(x1), ix)))
			}
		}
		if x0 <= x1 {
			return []event{{x0, 1}, {x1, -1}}
		}
		return nil
	}
	l1 := clipDo(root.l, x0, y, x1)
	l2 := clipDo(root.r, x0, y, x1)
	var out []event
	var k1, k2 int32
	for len(l1) > 0 || len(l2) > 0 {
		var t event
		if len(l2) == 0 || (len(l1) > 0 && (l1[0].x < l2[0].x || (l1[0].x == l2[0].x && l1[0].typ > l2[0].typ))) {
			t = l1[0]
			k1 += int32(t.typ)
			l1 = l1[1:]
		} else {
			t = l2[0]
			k2 += int32(t.typ)
			l2 = l2[1:]
		}
		tailOpen := len(out) == 0 || out[len(out)-1].typ == -1
		tailStart := len(out) > 0 && out[len(out)-1].typ == 1
		if (root.kind == ctOr && ((t.typ == 1 && tailOpen) || (t.typ == -1 && k1 == 0 && k2 == 0))) ||
			(root.kind == ctAnd && ((t.typ == 1 && tailOpen && k1 > 0 && k2 > 0) || (t.typ == -1 && tailStart && (k1 == 0 || k2 == 0)))) {
			out = append(out, t)
		}
	}
	return out
}

type clipEllipse struct {
	st    ellipseState
	root  *clipNode
	nodes [7]clipNode
	count int
	head  []event
	y     int32
}

func (s *clipEllipse) node() *clipNode {
	n := &s.nodes[s.count]
	s.count++
	return n
}

// normalizeAngles is normalize_angles: float in, float out, fmod in double.
func normalizeAngles(al, ar float32) (float32, float32) {
	if ar-al >= 360 {
		return 0, 360
	}
	var a float64
	if al < 0 {
		a = 360 - math.Mod(float64(-al), 360)
	} else {
		a = float64(al)
	}
	al = float32(math.Mod(a, 360))
	var d float64
	if ar < al {
		d = 360 - math.Mod(float64(al-ar), 360)
	} else {
		d = float64(ar - al)
	}
	ar = float32(float64(al) + math.Mod(d, 360))
	return al, ar
}

func (s *clipEllipse) arcInit(a, b, w int32, al, ar float32) {
	if a < b {
		s.arcInit(b, a, w, 90-ar, 90-al)
		s.st.init(a, b, w)
		s.root.transpose()
		return
	}
	s.st.init(a, b, w)
	s.head = nil
	s.count = 0
	al, ar = normalizeAngles(al, ar)
	if ar == al+360 {
		s.root = nil
		return
	}
	fa, fb := float64(a), float64(b)
	lc, rc := s.node(), s.node()
	lc.kind, rc.kind = ctClip, ctClip
	lc.a = float64(-fa * math.Sin(float64(al)*math.Pi/180.0))
	lc.b = float64(fb * math.Cos(float64(al)*math.Pi/180.0))
	lc.c = float64(float64(a*a-b*b)*math.Sin(float64(al)*math.Pi/90.0)) / 2.0
	rc.a = float64(fa * math.Sin(float64(ar)*math.Pi/180.0))
	rc.b = float64(-fb * math.Cos(float64(ar)*math.Pi/180.0))
	rc.c = float64(float64(b*b-a*a)*math.Sin(float64(ar)*math.Pi/90.0)) / 2.0
	kind := ctOr
	if ar-al < 180 {
		kind = ctAnd
	}
	switch {
	case math.Mod(float64(al), 180) == 0 || math.Mod(float64(ar), 180) == 0:
		s.root = s.node()
		s.root.l, s.root.r, s.root.kind = lc, rc, kind
	case (int(al/180)+int(ar/180))%2 == 1:
		s.root = s.node()
		s.root.l = s.node()
		s.root.l.l = s.node()
		s.root.l.r = lc
		s.root.r = s.node()
		s.root.r.l = s.node()
		s.root.r.r = rc
		s.root.kind = ctOr
		s.root.l.kind = ctAnd
		s.root.r.kind = ctAnd
		s.root.l.l.kind = ctClip
		s.root.r.l.kind = ctClip
		s.root.l.l.b = 1
		if int(al/180)%2 != 0 {
			s.root.l.l.b = -1
		}
		s.root.r.l.b = 1
		if int(ar/180)%2 != 0 {
			s.root.r.l.b = -1
		}
	default:
		s.root = s.node()
		s.root.l = s.node()
		s.root.r = s.node()
		s.root.kind, s.root.l.kind = kind, kind
		s.root.l.l, s.root.l.r = lc, rc
		s.root.r.kind = ctClip
		s.root.r.b = -1
		if ar < 180 || ar > 540 {
			s.root.r.b = 1
		}
	}
}

func (s *clipEllipse) next() (x0, y, x1 int32, ok bool) {
	for len(s.head) == 0 {
		X0, Y, X1, ok := s.st.next()
		if !ok {
			break
		}
		s.head = clipDo(s.root, X0, Y, X1)
		s.y = Y
	}
	if len(s.head) >= 2 {
		x0, x1 = s.head[0].x, s.head[1].x
		s.head = s.head[2:]
		return x0, s.y, x1, true
	}
	return 0, 0, 0, false
}

// arc is ImageDraw.arc (ImagingDrawArc).
func (c *canvas) arc(x0, y0, x1, y1 int, start, end float32, ink RGB, width int) {
	if x1 < x0 || y1 < y0 || width == 0 {
		return
	}
	start, end = normalizeAngles(start, end)
	a, b := x1-x0, y1-y0
	if start+360 == end {
		var st ellipseState
		st.init(int32(a), int32(b), int32(width))
		for {
			X0, Y, X1, ok := st.next()
			if !ok {
				break
			}
			c.hline(x0+(int(X0)+a)/2, y0+(int(Y)+b)/2, x0+(int(X1)+a)/2, ink)
		}
		return
	}
	if start == end {
		return
	}
	var st clipEllipse
	st.arcInit(int32(a), int32(b), int32(width), start, end)
	for {
		X0, Y, X1, ok := st.next()
		if !ok {
			break
		}
		c.hline(x0+(int(X0)+a)/2, y0+(int(Y)+b)/2, x0+(int(X1)+a)/2, ink)
	}
}
