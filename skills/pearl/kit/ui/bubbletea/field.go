package ui

import (
	"math"

	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"

	"tuikit/theme/bubbletea"
)

// Field is the brand backdrop: two glyphs on a sheared lattice with a hue
// ramp (references/cell-art.md "Fields"). Draw it first, then content over
// it; give content a clearing (ClearR) or keep the field out of the content
// column with Avoid.
type Field struct {
	PX, PY    int     // lattice period in cols and rows (10, 2)
	Shear     int     // cells each lattice row shifts: the angle (3–5)
	SubtleMix float64 // subtle glyph pulled toward bg by 1-this (0.5)
	Dim       float64 // global dim toward bg (0 none … 1 invisible)
	ClearR    float64 // radial clearing radius, normalised (0 = none)
	ClearY    float64 // clearing centre, 0..1 of height (0.45)
	Drift     int     // columns of drift (animate by bumping this)
}

// DefaultField is a calm backdrop.
func DefaultField() Field {
	return Field{PX: 10, PY: 2, Shear: 4, SubtleMix: 0.5, Dim: 0.35, ClearY: 0.45}
}

// Draw paints the field into r. Cells the field leaves empty are untouched.
func (f Field) Draw(c *Canvas, r uv.Rectangle) {
	t := c.T
	if t.ASCII || t.X.Backdrop != "field" {
		return
	}
	w, h := r.Dx(), r.Dy()
	cache := map[[2]int]lipgloss.Style{}
	for y := 0; y < h; y++ {
		if y%f.PY != 0 {
			continue
		}
		lr := y / f.PY
		for x := 0; x < w; x++ {
			k := x + f.Drift + lr*f.Shear
			if k%f.PX != 0 && (k+f.PX/2)%f.PX != 0 {
				continue
			}
			bright := k%f.PX == 0
			// diagonal hue ramp primary → secondary
			d := float64(x)/float64(max(1, w)) + float64(y)/float64(max(1, h))
			tone := int(math.Round(d / 2 * 10))
			dim := f.Dim
			if f.ClearR > 0 {
				nx := (float64(x)/float64(w) - 0.5)
				ny := (float64(y)/float64(h) - f.ClearY) * 0.5 // squash: cells are 1:2
				dist := math.Sqrt(nx*nx+ny*ny) / f.ClearR
				if dist < 1 {
					dim = 1 - (1-dim)*dist*dist
				}
			}
			dk := int(math.Round(dim * 10))
			key := [2]int{tone*2 + boolInt(bright), dk}
			st, ok := cache[key]
			if !ok {
				col := theme.Mix(t.P.Primary, t.P.Secondary, float64(tone)/10)
				if !bright {
					col = theme.Mix(col, t.P.Bg, 1-f.SubtleMix)
				}
				col = theme.Mix(col, t.P.Bg, float64(dk)/10)
				st = lipgloss.NewStyle().Foreground(col)
				cache[key] = st
			}
			if dk >= 10 {
				continue
			}
			g := t.G.FieldB
			if bright {
				g = t.G.FieldA
			}
			c.Draw(st.Render(g), uv.Rect(r.Min.X+x, r.Min.Y+y, 1, 1))
		}
	}
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
