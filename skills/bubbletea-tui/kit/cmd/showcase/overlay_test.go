package main

import (
	"fmt"
	"image/color"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"

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

// overlayVariants are the profiles the harness renders an overlay in: the scrim's
// (truecolour, 256 colours) and the band's (16 colours, NO_COLOR, ASCII).
var overlayVariants = []struct {
	name  string
	ascii bool
	p     colorprofile.Profile
}{{"truecolor", false, colorprofile.TrueColor}, {"ansi256", false, colorprofile.ANSI256},
	{"ansi", false, colorprofile.ANSI}, {"NO_COLOR", false, colorprofile.Ascii}, {"ascii", true, colorprofile.TrueColor}}

// overlayPages are the states an overlay is opened over in the tests: every screen
// that draws a selection, a focused field or a caret. (Not the setup wizard: a Huh
// form has no way to draw its focused field blurred short of Field.Blur, which
// validates it.)
var overlayPages = []string{"services", "deploys", "assistant", "deploy"}

// openOverlay opens the overlay s over whatever page m shows, as its key would.
func openOverlay(m *app, s string) {
	switch s {
	case "palette":
		m.ov, m.query, m.palSel = ovPalette, "ro", 0
	case "confirm":
		m.ov, m.btn = ovConfirm, 1
	case "help":
		m.helpFrom, m.ov = ovNone, ovHelp
	}
}

// cellText is a cell's glyph, a blank for an empty cell.
func cellText(c *ui.Canvas, x, y int) string {
	if cell := c.CellAt(x, y); cell != nil && cell.Content != "" {
		return cell.Content
	}
	return " "
}

func rowText(c *ui.Canvas, y int) string {
	var b strings.Builder
	for x := range c.Bounds().Dx() {
		b.WriteString(cellText(c, x, y))
	}
	return b.String()
}

// An overlay leaves the page's rows it doesn't span as the page drew them: the page
// under a dialog is drawn unfocused (its screens take focused=false), so nothing has
// to be scanned off it afterwards. A scan for focus marks by glyph and ink erased
// cartridge's wordmark (its focus bar is █ in the primary ink, as is the banner) and
// ferrous's header separators under ASCII (its focus bar and its separator are both
// "|"). Every theme, every profile, every overlay over every page that draws a
// selection, at 80×24 and 120×40: each cell above and below the dialog's rows and
// above the footer holds the glyph it held with no overlay, except where the page
// drew its own focus mark (the focused field's bar, the focused button's cursor, a
// caret, in the body), which drawing it unfocused takes away or turns blurred.
func TestOverlayKeepsThePageOffItsRows(t *testing.T) {
	for _, f := range testThemes(t) {
		tf, err := theme.Load(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, v := range overlayVariants {
			for _, sz := range [][2]int{{80, 24}, {120, 40}} {
				for _, page := range overlayPages {
					m := newApp(tf, true, v.ascii)
					m.profile = v.p
					m.setState(page)
					base := m.render(sz[0], sz[1])
					focusCells := m.focusMarks(base)
					for _, s := range []string{"palette", "confirm", "help"} {
						m.setState(page)
						openOverlay(m, s)
						c := m.render(sz[0], sz[1])
						name := fmt.Sprintf("%s %s %s over %s %dx%d", filepath.Base(f), v.name, s, page, sz[0], sz[1])
						for y := range m.footY {
							if y >= m.ovRect.Min.Y && y < m.ovRect.Max.Y {
								continue
							}
							for x := range sz[0] {
								want, got := cellText(base, x, y), cellText(c, x, y)
								if got == want || slices.Contains(focusCells, uv.Pos(x, y)) {
									continue
								}
								t.Errorf("%s: %q at %d,%d is %q with the overlay up; the page's row was %q", name, want, x, y, got, rowText(base, y))
								break
							}
						}
					}
				}
			}
		}
	}
	// the two the glyph scan broke, by name
	for _, k := range []struct {
		theme string
		ascii bool
		rows  []int
		want  string
	}{
		{"cartridge.json", false, []int{0, 1, 2}, ""},
		{"ferrous.json", true, []int{0}, "prod | us-east-1 | checkout-api"},
	} {
		tf := testTheme(t, k.theme)
		for _, s := range []string{"palette", "confirm", "help"} {
			for _, sz := range [][2]int{{80, 24}, {120, 40}} {
				m := newApp(tf, true, k.ascii)
				m.setState("services")
				base := m.render(sz[0], sz[1])
				m.setState(s)
				c := m.render(sz[0], sz[1])
				for _, y := range k.rows {
					if got, want := rowText(c, y), rowText(base, y); got != want {
						t.Errorf("%s ascii=%v %s %dx%d: row %d reads %q under the overlay, want %q", k.theme, k.ascii, s, sz[0], sz[1], y, got, want)
					}
					if k.want != "" && !strings.Contains(rowText(c, y), k.want) {
						t.Errorf("%s ascii=%v %s %dx%d: row %d lost %q: %q", k.theme, k.ascii, s, sz[0], sz[1], y, k.want, rowText(c, y))
					}
				}
			}
		}
	}
}

// An overlay frame has one focus, the dialog's, and nothing of the page touches the
// dialog's edge. A recreation's list drew its focus bar beside the dialog's own, and a
// pane's cut words ("1s", "igh", "00K") sat one cell right of the edge reading as
// garbage. The page under a dialog is drawn unfocused: drawn with no scrim (where a
// mark keeps its ink), no cell of the page's body outside the dialog is a focus mark
// in the focus ink, while the same page with no overlay draws one. Every theme, every
// profile, every overlay over every page that draws a selection, at 80×24 and 120×40.
func TestOverlayFrameHasOneFocus(t *testing.T) {
	seen := 0
	for _, f := range testThemes(t) {
		tf, err := theme.Load(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, v := range overlayVariants {
			for _, sz := range [][2]int{{80, 24}, {120, 40}} {
				for _, page := range overlayPages {
					for _, s := range []string{"palette", "confirm", "help"} {
						m := newApp(tf, true, v.ascii)
						m.profile = v.p
						m.setState(page)
						seen += len(m.focusMarks(m.render(sz[0], sz[1])))
						openOverlay(m, s)
						name := fmt.Sprintf("%s %s %s over %s %dx%d", filepath.Base(f), v.name, s, page, sz[0], sz[1])
						c := m.render(sz[0], sz[1])
						if x, y, ok := pageBeside(c, m.ovRect, 1); ok {
							t.Errorf("%s: the page's %q at %d,%d touches the dialog's edge", name, c.CellAt(x, y).Content, x, y)
						}
						m.t.X.Scrim = "none" // the page's own ink, undimmed
						c = m.render(sz[0], sz[1])
						for _, p := range m.focusMarks(c) {
							if !p.In(m.ovRect) {
								t.Errorf("%s: the page's focus mark %q at %d,%d shows beside the dialog", name, c.CellAt(p.X, p.Y).Content, p.X, p.Y)
							}
						}
					}
				}
			}
		}
	}
	if seen == 0 {
		t.Fatal("no page drew a focus mark: nothing was tested")
	}
}

// focusMarks are the page body's cells holding the theme's focus bar, selection cursor,
// or an input's prompt or caret, in its focus ink with a blank after. Only these
// glyphs count: a blurred selection's mark (the blur ink, or a blur glyph) is the
// quieter mark an unfocused page keeps, not a focus.
func (m *app) focusMarks(c *ui.Canvas) []uv.Position {
	t := m.t
	marks := []struct {
		g  string
		st lipgloss.Style
	}{{t.G.FocusBar, t.S.FocusBar}, {t.G.Cursor, t.S.Focus}, {t.G.Caret, t.S.Prompt}, {t.G.Prompt, t.S.Prompt}}
	var out []uv.Position
	for y := m.body.Min.Y; y < m.body.Max.Y; y++ {
		for x := m.body.Min.X; x+1 < m.body.Max.X; x++ {
			cell, next := c.CellAt(x, y), c.CellAt(x+1, y)
			if cell == nil || next == nil || strings.TrimSpace(next.Content) != "" {
				continue
			}
			for _, mk := range marks {
				if mk.g != "" && cell.Content == mk.g && sameRGB(cell.Style.Fg, mk.st.GetForeground()) {
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

// Help opened over an overlay names that overlay's keys, not the screen's under it
// (a recreation's help listed the conversation's keys over its permission dialog), and
// closing it goes back to the overlay. Every overlay's help key opens it, the body holds
// each of the overlay's bindings and none of the screen's, and esc returns.
func TestHelpOverAnOverlayNamesItsKeys(t *testing.T) {
	for _, s := range []string{"palette", "confirm"} {
		m := fixture(t, s)
		from := m.ov
		var own []string
		for _, a := range m.actions() {
			own = append(own, a.b.Help().Desc)
		}
		var helpKey string
		for _, a := range m.actions() {
			if a.b.Help().Desc == "keys" {
				helpKey = a.b.Keys()[0]
			}
		}
		if helpKey == "" {
			t.Fatalf("%s: no help key in the overlay's action table", s)
		}
		m.key(press(helpKey))
		if m.ov != ovHelp {
			t.Fatalf("%s: %q did not open help", s, helpKey)
		}
		// each help row is a 12-cell key column, then what the key does
		var named []string
		for _, row := range m.helpBody() {
			named = append(named, strings.TrimSpace(ansi.Strip(ansi.Cut(row, 12, 200))))
		}
		for _, d := range own {
			if !slices.Contains(named, d) {
				t.Errorf("%s: help over the overlay doesn't name its %q key: %q", s, d, named)
			}
		}
		for _, d := range []string{"deploy", "roll back", "next view"} {
			if !slices.Contains(own, d) && slices.Contains(named, d) {
				t.Errorf("%s: help over the overlay names the screen's %q key: %q", s, d, named)
			}
		}
		m.key(press("esc"))
		if m.ov != from {
			t.Errorf("%s: esc from help went to overlay %d, want back to %d", s, m.ov, from)
		}
	}
}
