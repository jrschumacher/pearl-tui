package framelint

import (
	"encoding/json"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// Every product below is wrapped in float64() so the compiler cannot fuse it
// into an FMA: the blends must round exactly as tuilint.py's do.

var hexRe = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

func hexc(s string) RGB {
	s = strings.TrimLeft(s, "#")
	var c RGB
	for k := range c {
		v, _ := strconv.ParseUint(s[2*k:2*k+2], 16, 8)
		c[k] = int(v)
	}
	return c
}

// bucketN covers channel 0..255 in 6-unit buckets.
const bucketN = 255/6 + 1

// Palette is a theme's colours quantised to 6-unit buckets: every role, every
// RGB and OKLab blend of two roles, and each of those blended toward a bg role.
type Palette struct {
	set  []bool // bucketN³
	near map[RGB]bool
}

func (p *Palette) has(r, g, b int) bool {
	if r < 0 || g < 0 || b < 0 || r >= bucketN || g >= bucketN || b >= bucketN {
		return false
	}
	return p.set[(r*bucketN+g)*bucketN+b]
}

// Near reports whether c falls in a palette bucket or a neighbouring one.
func (p *Palette) Near(c RGB) bool {
	if v, ok := p.near[c]; ok {
		return v
	}
	q := RGB{floorDiv(c[0], 6), floorDiv(c[1], 6), floorDiv(c[2], 6)}
	v := false
	for a := -1; a <= 1 && !v; a++ {
		for b := -1; b <= 1 && !v; b++ {
			for d := -1; d <= 1 && !v; d++ {
				v = p.has(q[0]+a, q[1]+b, q[2]+d)
			}
		}
	}
	p.near[c] = v
	return v
}

func floorDiv(a, b int) int {
	q := a / b
	if (a%b != 0) && ((a < 0) != (b < 0)) {
		q--
	}
	return q
}

// LoadPalette reads a theme JSON the way tuilint.py does: every #rrggbb under
// "roles" (or the whole file when there is no "roles"), with bg* keys of the
// role sets as the backgrounds that dims and scrims fade toward.
func LoadPalette(path string) (*Palette, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var t any
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, err
	}
	roles := t
	if m, ok := t.(map[string]any); ok {
		if r, ok := m["roles"]; ok {
			roles = r
		}
	}
	seen := map[RGB]bool{}
	var base []RGB
	var walk func(v any)
	walk = func(v any) {
		switch v := v.(type) {
		case string:
			if hexRe.MatchString(v) {
				if c := hexc(v); !seen[c] {
					seen[c] = true
					base = append(base, c)
				}
			}
		case map[string]any:
			for _, x := range v {
				walk(x)
			}
		case []any:
			for _, x := range v {
				walk(x)
			}
		}
	}
	walk(roles)

	var bgs []RGB
	if m, ok := roles.(map[string]any); ok {
		for _, v := range m {
			set, ok := v.(map[string]any)
			if !ok {
				continue
			}
			for k, x := range set {
				if s, ok := x.(string); ok && strings.HasPrefix(k, "bg") && hexRe.MatchString(s) {
					bgs = append(bgs, hexc(s))
				}
			}
		}
	}
	return buildPalette(base, bgs), nil
}

func buildPalette(base, bgs []RGB) *Palette {
	cols := map[RGB]struct{}{}
	for _, c := range base {
		cols[c] = struct{}{}
	}
	// gradients, dims and fades blend two roles, in RGB or in OKLab. tuilint.py
	// blends each pair one way, in whatever order its set iterates, and the
	// float rounding differs by direction; blending both ways is a superset
	// of every order.
	for i, a := range base {
		la := oklab(a)
		for _, b := range base[i+1:] {
			lb := oklab(b)
			blend(cols, a, b, la, lb)
			blend(cols, b, a, lb, la)
		}
	}
	p := &Palette{set: make([]bool, bucketN*bucketN*bucketN), near: map[RGB]bool{}}
	mark := func(c RGB) {
		r, g, b := floorDiv(c[0], 6), floorDiv(c[1], 6), floorDiv(c[2], 6)
		if r >= 0 && g >= 0 && b >= 0 && r < bucketN && g < bucketN && b < bucketN {
			p.set[(r*bucketN+g)*bucketN+b] = true
		}
	}
	// dims, fields and scrims blend a blend toward a background: one more step
	for c := range cols {
		mark(c)
		for _, bg := range bgs {
			for k := 1; k < 10; k++ {
				t := float64(k) / 10
				var d RGB
				for j := 0; j < 3; j++ {
					d[j] = int(math.RoundToEven(float64(c[j]) + float64(float64(bg[j]-c[j])*t)))
				}
				mark(d)
			}
		}
	}
	return p
}

// buckets lists the palette's buckets in (r, g, b) order.
func (p *Palette) buckets() []RGB {
	var out []RGB
	for i, on := range p.set {
		if on {
			out = append(out, RGB{i / (bucketN * bucketN), i / bucketN % bucketN, i % bucketN})
		}
	}
	return out
}

func blend(cols map[RGB]struct{}, a, b RGB, la, lb [3]float64) {
	for k := 1; k < 20; k++ {
		t := float64(k) / 20
		var c RGB
		var l [3]float64
		for j := 0; j < 3; j++ {
			c[j] = int(math.RoundToEven(float64(a[j]) + float64(float64(b[j]-a[j])*t)))
			l[j] = la[j] + float64((lb[j]-la[j])*t)
		}
		cols[c] = struct{}{}
		cols[fromOklab(l)] = struct{}{}
	}
}

func srgbLin(v int) float64 {
	x := float64(v) / 255
	if x <= 0.04045 {
		return x / 12.92
	}
	return math.Pow((x+0.055)/1.055, 2.4)
}

func oklab(c RGB) [3]float64 {
	r, g, b := srgbLin(c[0]), srgbLin(c[1]), srgbLin(c[2])
	l := math.Pow(float64(0.4122214708*r)+float64(0.5363325363*g)+float64(0.0514459929*b), 1.0/3)
	m := math.Pow(float64(0.2119034982*r)+float64(0.6806995451*g)+float64(0.1073969566*b), 1.0/3)
	s := math.Pow(float64(0.0883024619*r)+float64(0.2817188376*g)+float64(0.6299787005*b), 1.0/3)
	return [3]float64{
		float64(0.2104542553*l) + float64(0.7936177850*m) - float64(0.0040720468*s),
		float64(1.9779984951*l) - float64(2.4285922050*m) + float64(0.4505937099*s),
		float64(0.0259040371*l) + float64(0.7827717662*m) - float64(0.8086757660*s),
	}
}

// invGamma is 1/2.4 divided at run time in float64, as Python does it, not
// as an exact constant.
var invGamma = 1 / gamma

var gamma = 2.4

func fromOklab(lab [3]float64) RGB {
	L, a, b := lab[0], lab[1], lab[2]
	l := math.Pow(L+float64(0.3963377774*a)+float64(0.2158037573*b), 3)
	m := math.Pow(L-float64(0.1055613458*a)-float64(0.0638541728*b), 3)
	s := math.Pow(L-float64(0.0894841775*a)-float64(1.2914855480*b), 3)
	rgb := [3]float64{
		float64(4.0767416621*l) - float64(3.3077115913*m) + float64(0.2309699292*s),
		float64(-1.2684380046*l) + float64(2.6097574011*m) - float64(0.3413193965*s),
		float64(-0.0041960863*l) - float64(0.7034186147*m) + float64(1.7076147010*s),
	}
	var out RGB
	for k, v := range rgb {
		v = math.Max(0, math.Min(1, v))
		if v <= 0.0031308 {
			v = float64(12.92 * v)
		} else {
			v = float64(1.055*math.Pow(v, invGamma)) - 0.055
		}
		out[k] = int(math.RoundToEven(float64(v * 255)))
	}
	return out
}
