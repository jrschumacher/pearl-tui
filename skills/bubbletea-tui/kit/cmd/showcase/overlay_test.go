package main

import (
	"image/color"
	"path/filepath"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
	uv "github.com/charmbracelet/ultraviolet"

	"tuikit/theme"
	"tuikit/ui"
)

// pageBeside finds a page cell left on the rows a dialog spans, outside it, within
// reach cells of its edges: the first one with text, if any.
func pageBeside(c *ui.Canvas, r uv.Rectangle, reach int) (int, int, bool) {
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X - reach; x < r.Max.X+reach; x++ {
			if uv.Pos(x, y).In(r) || !uv.Pos(x, y).In(c.Bounds()) {
				continue
			}
			if cell := c.CellAt(x, y); cell != nil && strings.TrimSpace(cell.Content) != "" {
				return x, y, true
			}
		}
	}
	return 0, 0, false
}

// An overlay frame has one focus, the dialog's, and nothing of the page touches the
// dialog's edge. The services list's focus bar showed through the scrim beside the
// dialog's own (two marks, one dimmed), and a pane's cut words ("1s", "igh", "00K")
// sat one cell right of the edge reading as garbage. Every theme, every profile the
// harness renders, the three overlays over the services list and the assistant's
// transcript, at 80×24 and 120×40: each focus mark the page draws alone is gone.
func TestOverlayFrameHasOneFocus(t *testing.T) {
	seen := 0
	for _, f := range testThemes(t) {
		tf, err := theme.Load(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, v := range []struct {
			name  string
			ascii bool
			p     colorprofile.Profile
		}{{"truecolor", false, colorprofile.TrueColor}, {"ansi256", false, colorprofile.ANSI256},
			{"ansi", false, colorprofile.ANSI}, {"NO_COLOR", false, colorprofile.Ascii}, {"ascii", true, colorprofile.TrueColor}} {
			for _, sz := range [][2]int{{80, 24}, {120, 40}} {
				for _, page := range []screen{scrServices, scrAssistant} {
					for _, s := range []string{"palette", "confirm", "help"} {
						m := newApp(tf, true, v.ascii)
						m.profile = v.p
						m.setState(s)
						m.screen, m.w, m.h = page, sz[0], sz[1]
						name := filepath.Base(f) + " " + v.name + " " + s
						// draw, not render: the marks are found before ASCIIfy, in their ink
						ov := m.ov
						m.ov = ovNone
						marks := focusMarks(m.draw(sz[0], sz[1]), m.t)
						m.ov = ov
						c := m.draw(sz[0], sz[1])
						for _, p := range marks {
							if p.In(m.ovRect) {
								continue
							}
							seen++
							if cell := c.CellAt(p.X, p.Y); cell != nil && strings.TrimSpace(cell.Content) != "" {
								t.Errorf("%s %dx%d: the page's focus mark %q at %d,%d shows beside the dialog", name, sz[0], sz[1], cell.Content, p.X, p.Y)
							}
						}
						if x, y, ok := pageBeside(c, m.ovRect, 1); ok {
							t.Errorf("%s %dx%d: the page's %q at %d,%d touches the dialog's edge", name, sz[0], sz[1], c.CellAt(x, y).Content, x, y)
						}
					}
				}
			}
		}
	}
	if seen == 0 {
		t.Fatal("no page under a dialog drew a focus mark: nothing was tested")
	}
}

// focusMarks are the cells holding the theme's focus bar or selection cursor in its
// focus ink with a blank after: a gutter mark.
func focusMarks(c *ui.Canvas, t *theme.Theme) []uv.Position {
	marks := []struct {
		g  string
		st lipgloss.Style
	}{{t.G.FocusBar, t.S.FocusBar}, {t.G.Cursor, t.S.Focus}}
	b := c.Bounds()
	var out []uv.Position
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x+1 < b.Max.X; x++ {
			cell, next := c.CellAt(x, y), c.CellAt(x+1, y)
			if cell == nil || next == nil || strings.TrimSpace(next.Content) != "" {
				continue
			}
			for _, m := range marks {
				if m.g != "" && cell.Content == m.g && sameRGB(cell.Style.Fg, m.st.GetForeground()) {
					out = append(out, uv.Pos(x, y))
				}
			}
		}
	}
	return out
}

func sameRGB(a, b color.Color) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	r1, g1, b1, _ := a.RGBA()
	r2, g2, b2, _ := b.RGBA()
	return r1 == r2 && g1 == g2 && b1 == b2
}
