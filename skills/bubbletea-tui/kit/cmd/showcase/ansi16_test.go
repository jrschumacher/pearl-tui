package main

import (
	"fmt"
	"image/color"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"unicode"

	"github.com/charmbracelet/colorprofile"
	uv "github.com/charmbracelet/ultraviolet"

	"tuikit/theme"
	"tuikit/ui"
)

// ansi16States are the states #51's sheets judge: every page that draws a selection
// (a warning beside a success and the selection, a form) and every overlay (a filled
// selection in a dialog), from the app's own lists. The showcase adds its status ladder
// and its wizard's brand title and filled header (screens_test.go).
var ansi16States = append(slices.Clone(overlayPages), overlays...)

// renderState draws state at w×h for theme f.
func renderState(f theme.File, dark bool, prof colorprofile.Profile, state string, w, h int) (*app, *ui.Canvas) {
	m := newApp(f, dark, false)
	m.profile = prof
	m.setState(state)
	c := m.render(w, h)
	c.PaintBackground()
	return m, c
}

// At 16 colours a warm brand draws in bright yellow on a dark page beside the warning in
// the other yellow (#51), and only the warning's glyph tells the two apart. So every run
// of marks in the warning's ink carries its glyph (▲, !), or sits one blank from it: the
// services table's p99 sparkline of a degraded service was a run of yellow bars with no
// mark, the yellow of the selection bar beside it. A run with a word in it (DEGRADED,
// PROD) says what it is. The warning is set to a colour no other role has, so the test
// finds the warning's own cells at any profile.
func TestWarningInkCarriesItsGlyph(t *testing.T) {
	const sentinel = "#010203"
	for _, path := range testThemes(t) {
		tf, err := theme.Load(path)
		if err != nil {
			t.Fatal(err)
		}
		roles := map[string]theme.Roles{}
		for mode, r := range tf.Roles {
			r.Warning = sentinel
			roles[mode] = r
		}
		tf.Roles = roles
		for _, dark := range []bool{true, false} {
			for _, state := range ansi16States {
				for _, size := range [][2]int{{80, 24}, {120, 40}} {
					m, c := renderState(tf, dark, colorprofile.TrueColor, state, size[0], size[1])
					where := fmt.Sprintf("%s dark=%v %s %dx%d", filepath.Base(path), dark, state, size[0], size[1])
					isWarn := func(cell *uv.Cell) bool {
						return cell != nil && cell.Style.Fg != nil && cell.Style.Attrs&uv.AttrReverse == 0 && sameHex(cell.Style.Fg, sentinel)
					}
					for y := range size[1] {
						var run []int
						glyph, word, gap := false, false, 0
						flush := func() {
							if len(run) > 0 && !glyph && !word {
								t.Errorf("%s: row %d, columns %d-%d draw in the warning's ink with no %q beside them: %q",
									where, y, run[0], run[len(run)-1], m.t.G.Warn, strings.TrimRight(rowText(c, y), " "))
							}
							run, glyph, word, gap = nil, false, false, 0
						}
						for x := range size[0] {
							cell := c.CellAt(x, y)
							text := ""
							if cell != nil {
								text = strings.TrimSpace(cell.Content)
							}
							switch {
							case text != "" && isWarn(cell):
								run = append(run, x)
								glyph = glyph || text == m.t.G.Warn
								word = word || strings.IndexFunc(text, unicode.IsLetter) >= 0
								gap = 0
							case text == "" && len(run) > 0 && gap == 0:
								gap++ // one blank between the bars and their mark
							default:
								flush()
							}
						}
						flush()
					}
				}
			}
		}
	}
}

// sameHex reports whether c is the colour hex names.
func sameHex(c color.Color, hex string) bool {
	r1, g1, b1, _ := c.RGBA()
	r2, g2, b2, _ := theme.Hex(hex).RGBA()
	return r1 == r2 && g1 == g2 && b1 == b2
}

// A pin moves a fill to a slot its ink was never chosen for: a warm brand pinned to fg's
// black on a light page (#51) drew halyard's header bar, whose ink is a near-black, black
// on black. No text cell, in any of #51's states, draws in its background's slot at 16
// colours.
func TestNoTextOnItsFillAt16Colours(t *testing.T) {
	for _, path := range testThemes(t) {
		tf, err := theme.Load(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, dark := range []bool{true, false} {
			for _, state := range ansi16States {
				m, c := renderState(tf, dark, colorprofile.ANSI, state, 80, 24)
				hidden := 0
				for y := range 24 {
					for x := range 80 {
						cell := c.CellAt(x, y)
						if cell == nil || strings.TrimSpace(cell.Content) == "" {
							continue
						}
						if slot16(cell.Style.Fg, m.t.P.Fg) == slot16(cell.Style.Bg, m.t.P.Bg) {
							hidden++
							if hidden == 1 {
								t.Errorf("%s dark=%v %s: %q at %d,%d draws in its background's slot: %q", filepath.Base(path), dark, state, cell.Content, x, y, strings.TrimRight(rowText(c, y), " "))
							}
						}
					}
				}
			}
		}
	}
}

// slot16 is the 16-colour slot a cell's colour draws as, def when the cell sets none.
func slot16(c, def color.Color) int {
	if c == nil {
		c = def
	}
	return theme.Slot16(c)
}

// The warning keeps yellow at 16 colours (#51), and yellow 3 on a white page is 1.35:1:
// halyard's thin "!", a [DEGRADED] tag and a [PROD] badge drawn in it vanished. Where the
// warning's slot reads under 3:1 on the page, no cell draws in the warning as text
// alone: the warning is a fill there (theme.Styles.Warn), a yellow cell with a dark ink.
// The warning's own cells are found by a second render with the warning pinned to
// magenta: other roles that land on yellow (an accent) stay put.
func TestWarningIsAFillWhereItsSlotIsFaint(t *testing.T) {
	for _, path := range testThemes(t) {
		tf, err := theme.Load(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, dark := range []bool{true, false} {
			mode := map[bool]string{true: "dark", false: "light"}[dark]
			moved := tf
			moved.ANSI16 = map[string]map[string]int{}
			for m, pins := range tf.ANSI16 {
				moved.ANSI16[m] = map[string]int{}
				for r, n := range pins {
					moved.ANSI16[m][r] = n
				}
			}
			if moved.ANSI16[mode] == nil {
				moved.ANSI16[mode] = map[string]int{}
			}
			probe := map[bool]int{true: 13, false: 5}[dark]
			moved.ANSI16[mode]["warning"] = probe
			for _, state := range ansi16States {
				m, c := renderState(tf, dark, colorprofile.ANSI, state, 80, 24)
				if theme.Contrast16(m.t.P.Warning, m.t.P.Bg) >= 3 {
					continue
				}
				_, c2 := renderState(moved, dark, colorprofile.ANSI, state, 80, 24)
				warn := theme.Slot16(m.t.P.Warning)
				found := 0
				for y := range 24 {
					for x := range 80 {
						cell, probed := c.CellAt(x, y), c2.CellAt(x, y)
						if cell == nil || probed == nil || strings.TrimSpace(cell.Content) == "" || cell.Style.Attrs&uv.AttrReverse != 0 {
							continue
						}
						if slot16(probed.Style.Fg, m.t.P.Fg) != probe || slot16(cell.Style.Fg, m.t.P.Fg) != warn {
							continue // not the warning's ink
						}
						found++
						if slot16(cell.Style.Bg, m.t.P.Bg) != warn && found < 3 {
							t.Errorf("%s dark=%v %s: %q at %d,%d draws in the warning's ANSI %d as text on ANSI %d, %.2f:1: %q", filepath.Base(path), dark, state,
								cell.Content, x, y, warn, slot16(cell.Style.Bg, m.t.P.Bg), theme.Contrast16(m.t.P.Warning, m.t.P.Bg), strings.TrimRight(rowText(c, y), " "))
						}
					}
				}
			}
		}
	}
}
