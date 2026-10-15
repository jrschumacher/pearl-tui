// Package oklch is the colour maths behind palette derive: sRGB hex <-> OKLab/OKLCH,
// gamut mapping by chroma reduction, and the nearest xterm-256 index.
//
// It mirrors scripts/palette.py operation for operation so the two produce the same hex
// digits: the same operand order, Python's round-half-even, Python's float modulo, and
// explicit float64 conversions around every product that is summed, so no platform may
// fuse them into an FMA. The transcendental functions (math.Pow, Sin, Cos, Atan2,
// Hypot) are Go's, which can differ from the C library's by an ulp; at 8-bit hex
// precision that has not changed an output on any input we compared (see the tests).
//
// Contrast is the kit's own (tuikit/theme.Contrast), so the checker and the engine
// share one definition.
package oklch

import (
	"errors"
	"fmt"
	"image/color"
	"math"
	"strings"

	"tuikit/theme"
)

// Runtime copies of constants, so divisions round the way Python's do (a Go constant
// expression such as 1/2.4 is computed exactly and rounded once, Python's is not).
var (
	pi       = math.Pi
	degToRad = pi / 180
	radToDeg = 180 / pi
	invGamma = one / 2.4
	third    = one / 3
	one      = 1.0
)

// RGB is a colour as three sRGB channels in 0..1 (unclamped when out of gamut).
type RGB [3]float64

// ErrHex reports a string that is not a hex colour.
var ErrHex = errors.New("not a hex colour")

// Parse reads a hex colour the way palette.py's hex2rgb does: surrounding space and
// leading '#'s are ignored, three digits are doubled, and the first six digits are
// read as three channels (so "#RRGGBBAA" ignores its alpha).
func Parse(s string) ([3]uint8, error) {
	h := strings.TrimLeft(strings.TrimSpace(s), "#")
	if len(h) == 3 {
		h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})
	}
	var out [3]uint8
	for i := range 3 {
		lo, hi := 2*i, 2*i+2
		if lo >= len(h) {
			return out, fmt.Errorf("%q: %w", s, ErrHex)
		}
		hi = min(hi, len(h))
		var v uint8
		for _, c := range []byte(h[lo:hi]) {
			d := strings.IndexByte("0123456789abcdef", lower(c))
			if d < 0 {
				return out, fmt.Errorf("%q: %w", s, ErrHex)
			}
			v = v*16 + uint8(d)
		}
		out[i] = v
	}
	return out, nil
}

func lower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + 'a' - 'A'
	}
	return c
}

// Valid reports whether s parses as a hex colour.
func Valid(s string) bool {
	_, err := Parse(s)
	return err == nil
}

// MustRGB returns the channels of a hex colour in 0..1. It panics on an invalid string;
// callers validate their inputs first.
func MustRGB(s string) RGB {
	b, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return RGB{float64(b[0]) / 255, float64(b[1]) / 255, float64(b[2]) / 255}
}

// Color returns a hex string as a color.Color for the kit's theme functions.
func Color(s string) color.Color {
	b, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return color.RGBA{R: b[0], G: b[1], B: b[2], A: 0xff}
}

// Round is Python's round() on a float: half to even.
func Round(v float64) int {
	return int(math.RoundToEven(v))
}

// Hex formats channels as "#RRGGBB", clamping each to 0..255 after rounding.
func Hex(c RGB) string {
	var sb strings.Builder
	sb.WriteByte('#')
	for _, v := range c {
		n := max(0, min(255, Round(v*255)))
		fmt.Fprintf(&sb, "%02X", n)
	}
	return sb.String()
}

// Mod is Python's float modulo: the result takes the sign of the divisor.
func Mod(x, y float64) float64 {
	m := math.Mod(x, y)
	if m != 0 {
		if (y < 0) != (m < 0) {
			m += y
		}
	} else {
		m = math.Copysign(0, y)
	}
	return m
}

func lin(v float64) float64 {
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

func delin(v float64) float64 {
	if v <= 0.0031308 {
		return 12.92 * v
	}
	return float64(1.055*math.Pow(v, invGamma)) - 0.055
}

// cube is x**3 correctly rounded (as the C library's pow returns it): the square is
// split exactly with FMA, then multiplied out in double-double.
func cube(x float64) float64 {
	hi := x * x
	lo := math.FMA(x, x, -hi)
	p := hi * x
	e := math.FMA(hi, x, -p)
	return p + (e + float64(lo*x))
}

// cbrt is palette.py's math.copysign(abs(x) ** (1 / 3), x). Python's pow is the C
// library's, which is correctly rounded in nearly every case; Go's math.Pow is not
// (it differs on about a third of inputs), and that difference decides real ties: #010101
// sits exactly halfway between xterm 16 and 232. So: the cube root refined by one Newton
// step in double-double, times x**(fl(1/3) - 1/3) = exp(-ln(x) / (3 * 2**54)), the
// tiny correction for 1/3 not being exact in binary.
func cbrt(x float64) float64 {
	a := math.Abs(x)
	if a == 0 || math.IsInf(a, 0) || math.IsNaN(a) {
		return math.Copysign(math.Pow(a, third), x)
	}
	c := math.Cbrt(a)
	s := c * c
	se := math.FMA(c, c, -s)
	p := s * c
	pe := math.FMA(s, c, -p)
	r := (a - p) - pe - float64(se*c)
	dc := r / float64(3*s)
	eps := float64(-1.0/(3*(1<<54))) * math.Log(a)
	return math.Copysign(c+(dc+float64(c*eps)), x)
}

// ToOKLab converts sRGB channels to OKLab.
func ToOKLab(c RGB) (L, a, b float64) {
	r, g, bl := lin(c[0]), lin(c[1]), lin(c[2])
	l := float64(0.4122214708*r) + float64(0.5363325363*g) + float64(0.0514459929*bl)
	m := float64(0.2119034982*r) + float64(0.6806995451*g) + float64(0.1073969566*bl)
	s := float64(0.0883024619*r) + float64(0.2817188376*g) + float64(0.6299787005*bl)
	l, m, s = cbrt(l), cbrt(m), cbrt(s)
	return float64(0.2104542553*l) + float64(0.7936177850*m) - float64(0.0040720468*s),
		float64(1.9779984951*l) - float64(2.4285922050*m) + float64(0.4505937099*s),
		float64(0.0259040371*l) + float64(0.7827717662*m) - float64(0.8086757660*s)
}

// FromOKLab converts OKLab to (possibly out-of-gamut) sRGB channels.
func FromOKLab(L, a, b float64) RGB {
	l := cube(L + float64(0.3963377774*a) + float64(0.2158037573*b))
	m := cube(L - float64(0.1055613458*a) - float64(0.0638541728*b))
	s := cube(L - float64(0.0894841775*a) - float64(1.2914855480*b))
	r := float64(4.0767416621*l) - float64(3.3077115913*m) + float64(0.2309699292*s)
	g := float64(-1.2684380046*l) + float64(2.6097574011*m) - float64(0.3413193965*s)
	bb := float64(-0.0041960863*l) - float64(0.7034186147*m) + float64(1.7076147010*s)
	return RGB{delin(r), delin(g), delin(bb)}
}

// LCH returns a hex colour's OKLCH lightness, chroma and hue (degrees, 0..360).
func LCH(hex string) (L, C, H float64) {
	L, a, b := ToOKLab(MustRGB(hex))
	return L, math.Hypot(a, b), Mod(math.Atan2(b, a)*radToDeg, 360)
}

// InGamut reports whether every channel is within sRGB, with palette.py's 1e-4 slack.
func InGamut(c RGB) bool {
	for _, v := range c {
		if v < -1e-4 || v > 1+1e-4 {
			return false
		}
	}
	return true
}

func lchRGB(L, C, H float64) RGB {
	return FromOKLab(L, C*math.Cos(H*degToRad), C*math.Sin(H*degToRad))
}

// FromLCH converts OKLCH to hex, reducing chroma until the colour fits sRGB (keeping
// hue and lightness): 24 steps of bisection, as palette.py's from_lch.
func FromLCH(L, C, H float64) string {
	rgb := lchRGB(L, C, H)
	if InGamut(rgb) {
		return Hex(rgb)
	}
	lo, hi := 0.0, C
	for range 24 {
		mid := (lo + hi) / 2
		if InGamut(lchRGB(L, mid, H)) {
			lo = mid
		} else {
			hi = mid
		}
	}
	return Hex(lchRGB(L, lo, H))
}

// Contrast is the WCAG 2 contrast ratio between two hex colours, computed by the kit.
func Contrast(a, b string) float64 {
	return theme.Contrast(Color(a), Color(b))
}

var xtermLab = func() (t [240][3]float64) {
	steps := [6]int{0, 95, 135, 175, 215, 255}
	for i := 16; i < 256; i++ {
		var c [3]int
		if i < 232 {
			n := i - 16
			c = [3]int{steps[n/36], steps[(n/6)%6], steps[n%6]}
		} else {
			v := 8 + (i-232)*10
			c = [3]int{v, v, v}
		}
		L, a, b := ToOKLab(RGB{float64(c[0]) / 255, float64(c[1]) / 255, float64(c[2]) / 255})
		t[i-16] = [3]float64{L, a, b}
	}
	return t
}()

// Xterm256 is the nearest xterm-256 index (16..255) to a hex colour, by squared OKLab
// distance; the first of equals wins.
func Xterm256(hex string) int {
	c := MustRGB(hex)
	var q RGB
	for i, v := range c {
		q[i] = float64(Round(v*255)) / 255
	}
	L2, a2, b2 := ToOKLab(q)
	best, bi := 1e9, 0
	for i, p := range xtermLab {
		dl, da, db := p[0]-L2, p[1]-a2, p[2]-b2
		d := float64(dl*dl) + float64(da*da) + float64(db*db)
		if d < best {
			best, bi = d, i+16
		}
	}
	return bi
}
