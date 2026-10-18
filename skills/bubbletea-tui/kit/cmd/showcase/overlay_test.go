package main

import (
	"fmt"
	"image/color"
	"maps"
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
					for _, s := range overlays {
						m.setState(page)
						m.openOverlay(s)
						c := m.render(sz[0], sz[1])
						name := fmt.Sprintf("%s %s %s over %s %dx%d", filepath.Base(f), v.name, s, page, sz[0], sz[1])
						for y := range m.footY {
							if y >= m.ovRect.Min.Y && y < m.ovRect.Max.Y {
								besideTheDialog(t, name, m, base, c, y, focusCells)
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
	// the two the glyph scan broke, by name: cartridge's wordmark rows (its focus bar is
	// the banner's █) and ferrous's header under ASCII (its bar and separator are both
	// "|"): every row above the page's body (the header, none in an app without one)
	// reads as it did. In a copied project testTheme gives the project's theme.
	for _, k := range []struct {
		theme string
		ascii bool
	}{
		{"cartridge.json", false},
		{"ferrous.json", true},
	} {
		tf := testTheme(t, k.theme)
		for _, s := range overlays {
			for _, sz := range [][2]int{{80, 24}, {120, 40}} {
				m := newApp(tf, true, k.ascii)
				m.setState(overlayPages[0])
				base := m.render(sz[0], sz[1])
				head := m.body.Min.Y
				m.setState(overlayPages[0])
				m.openOverlay(s)
				c := m.render(sz[0], sz[1])
				for y := range min(head, m.ovRect.Min.Y) {
					if got, want := rowText(c, y), rowText(base, y); got != want {
						t.Errorf("%s ascii=%v %s %dx%d: row %d reads %q under the overlay, want %q", k.theme, k.ascii, s, sz[0], sz[1], y, got, want)
					}
				}
			}
		}
	}
}

// besideTheDialog judges row y of an overlay frame c outside the dialog against the
// same page with no overlay (base): where no scrim is drawn the row is cleared across
// (the band); under a scrim each side changes only in one run out from the dialog's
// edge, blanks with at most one ellipsis at its far end (the cut word), and on the
// dialog's edge rows a page rule may go too. A focus mark of the page's may change
// anywhere: the page is drawn unfocused.
func besideTheDialog(t *testing.T, name string, m *app, base, c *ui.Canvas, y int, focusCells []uv.Position) {
	t.Helper()
	r, w := m.ovRect, c.Bounds().Dx()
	th := m.t
	scrim := th.X.Scrim == "dim" && !th.ASCII && th.Profile != colorprofile.ANSI && th.Profile != colorprofile.Ascii && th.Profile != colorprofile.NoTTY
	if !scrim {
		for x := range w {
			if !uv.Pos(x, y).In(r) && cellText(c, x, y) != " " {
				t.Errorf("%s: no scrim, and the page shows %q at %d,%d beside the dialog", name, cellText(c, x, y), x, y)
				return
			}
		}
		return
	}
	edge := y <= r.Min.Y+1 || y >= r.Max.Y-2 // a rules dialog's rule sits a margin row in
	for _, side := range []struct{ halo, dx int }{{r.Min.X - 1, -1}, {r.Max.X, 1}} {
		run, ells := true, 0
		for x := side.halo; x >= 0 && x < w; x += side.dx {
			want, got := cellText(base, x, y), cellText(c, x, y)
			switch {
			case got == want:
				run = false
			case slices.Contains(focusCells, uv.Pos(x, y)):
				// the page's own focus mark, drawn unfocused
			case edge && got == " " && (want == th.G.Rule || want == th.G.HeaderRule):
				// a page rule beside the dialog's edge row
			case run && got == " " && ells == 0:
			case run && got == th.G.Ellipsis && ells == 0:
				ells++
			default:
				t.Errorf("%s: %q at %d,%d is %q beside the dialog, outside the word it cuts: %q", name, want, x, y, got, rowText(c, y))
				return
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
					for _, s := range overlays {
						m := newApp(tf, true, v.ascii)
						m.profile = v.p
						m.setState(page)
						seen += len(m.focusMarks(m.render(sz[0], sz[1])))
						m.openOverlay(s)
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
// quieter mark an unfocused page keeps, not a focus. Under a "fill" or "inverse"
// selection the focused row is the mark: its cells in the selection's reverse video
// (a blurred one keeps a bgSubtle band), or a list page in such a theme drew no mark
// at all and the focus tests judged nothing.
func (m *app) focusMarks(c *ui.Canvas) []uv.Position {
	t := m.t
	marks := []struct {
		g  string
		st lipgloss.Style
	}{{t.G.FocusBar, t.S.FocusBar}, {t.G.Cursor, t.S.Focus}, {t.G.Caret, t.S.Prompt}, {t.G.Prompt, t.S.Prompt}}
	fill := t.X.Selection == "fill" || t.X.Selection == "inverse"
	var out []uv.Position
	for y := m.body.Min.Y; y < m.body.Max.Y; y++ {
		for x := m.body.Min.X; x+1 < m.body.Max.X; x++ {
			cell, next := c.CellAt(x, y), c.CellAt(x+1, y)
			if fill && cell != nil && cell.Style.Attrs&uv.AttrReverse != 0 && sameRGB(cell.Style.Fg, t.S.SelRow.GetForeground()) {
				out = append(out, uv.Pos(x, y))
				continue
			}
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

// The tables the harness tests read are the app's own: a state opens a dialog exactly
// when overlays names it, every overlayPages entry is a page with no dialog up, and
// every page that draws a focus mark is in overlayPages, renders as one that is, or is
// in unjudgedPages with a reason. A cold build left its menu overlay out of the help
// test, and its help over the menu named the screen's keys; leaving it out of overlays
// would hide it as well, and so would moving a page into unjudgedPages: an entry needs
// a reason, is never the main screen (states[0]), and must keep a focus mark beside
// some overlay (else it blurs, and belongs in overlayPages).
func TestOverlayTablesMatchTheStates(t *testing.T) {
	for _, list := range [][]string{overlays, overlayPages, slices.Collect(maps.Keys(unjudgedPages))} {
		for _, s := range list {
			if !slices.Contains(states, s) {
				t.Errorf("%q is in the harness tables but not in states", s)
			}
		}
	}
	for s, why := range unjudgedPages {
		switch {
		case s == states[0]:
			t.Errorf("%s, the main screen, is in unjudgedPages: every overlay must be judged over it", s)
		case strings.TrimSpace(why) == "":
			t.Errorf("%s is in unjudgedPages with no reason: say why its focus can't be drawn blurred (and repeat it in TUI.md § Decisions)", s)
		case slices.Contains(overlayPages, s):
			t.Errorf("%s is in both overlayPages and unjudgedPages", s)
		case !keepsFocusUnderAnOverlay(t, s):
			t.Errorf("%s is in unjudgedPages, but every overlay over it draws it blurred: judge it in overlayPages", s)
		}
	}
	for _, s := range states {
		if s == "toosmall" {
			continue
		}
		m := fixture(t, s)
		_, unjudged := unjudgedPages[s]
		switch in := slices.Contains(overlays, s); {
		case m.ov != ovNone && !in:
			t.Errorf("%s opens a dialog but isn't in overlays: the overlay tests never judge it", s)
		case m.ov == ovNone && in:
			t.Errorf("%s is in overlays but opens no dialog", s)
		case m.ov != ovNone && slices.Contains(overlayPages, s):
			t.Errorf("%s is in overlayPages but has a dialog up", s)
		case m.ov == ovNone && !slices.Contains(overlayPages, s) && !unjudged && len(m.focusMarks(m.render(80, 24))) > 0 && judgedTwin(t, s) == "":
			t.Errorf("%s draws a focus mark but is in neither overlayPages nor unjudgedPages, and renders as no page that is: no overlay is judged over it", s)
		}
	}
}

// judgedTwin is the overlayPages state whose page body renders as state s's at 80×24
// (services with a toast over the footer is services), or "".
func judgedTwin(t *testing.T, s string) string {
	t.Helper()
	body := func(s string) string {
		m := fixture(t, s)
		c := m.render(80, 24)
		var rows []string
		for y := m.body.Min.Y; y < m.body.Max.Y; y++ {
			rows = append(rows, rowText(c, y))
		}
		return strings.Join(rows, "\n")
	}
	b := body(s)
	for _, p := range overlayPages {
		if body(p) == b {
			return p
		}
	}
	return ""
}

// keepsFocusUnderAnOverlay reports whether some overlay over state s, in some theme at
// 80×24 or 120×40, leaves a focus mark of the page beside the dialog (drawn
// unscrimmed): the limitation an unjudgedPages entry claims. A page whose focus blurs
// under every overlay is judged instead.
func keepsFocusUnderAnOverlay(t *testing.T, s string) bool {
	t.Helper()
	for _, f := range testThemes(t) {
		tf, err := theme.Load(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, sz := range [][2]int{{80, 24}, {120, 40}} {
			for _, o := range overlays {
				m := newApp(tf, true, false)
				m.setState(s)
				m.openOverlay(o)
				m.t.X.Scrim = "none"
				c := m.render(sz[0], sz[1])
				for _, p := range m.focusMarks(c) {
					if !p.In(m.ovRect) {
						return true
					}
				}
			}
		}
	}
	return false
}

// Help opened over an overlay names that overlay's keys, not the screen's under it
// (a recreation's help listed the conversation's keys over its permission dialog), and
// closing it goes back to the overlay. Every overlay's help key opens it, the body holds
// each of the overlay's bindings and none of the screen's, and esc returns. The help key
// is whichever of the overlay's keys opens help, and the screen's keys are the page's
// own action table, so the test reads the app's tables, not the showcase's words. Help
// over a page (each of overlayPages) names every key of that page's table.
func TestHelpOverAnOverlayNamesItsKeys(t *testing.T) {
	for _, s := range overlays {
		if s == "help" {
			continue
		}
		m := fixture(t, s)
		from := m.ov
		var own []string
		for _, a := range m.actions() {
			own = append(own, strings.Join(strings.Fields(a.b.Help().Desc), " "))
		}
		// the screen's keys: the page under the overlay, with the overlay closed
		page := *fixture(t, s)
		page.ov = ovNone
		var screenKeys []string
		for _, a := range page.actions() {
			if d := strings.Join(strings.Fields(a.b.Help().Desc), " "); !slices.Contains(own, d) {
				screenKeys = append(screenKeys, d)
			}
		}
		k := helpKey(t, s)
		m.key(press(k))
		if m.ov != ovHelp {
			t.Fatalf("%s: %q did not open help", s, k)
		}
		named := helpNames(m)
		for _, d := range own {
			if !slices.Contains(named, d) {
				t.Errorf("%s: help over the overlay doesn't name its %q key: %q", s, d, named)
			}
		}
		for _, d := range screenKeys {
			if slices.Contains(named, d) {
				t.Errorf("%s: help over the overlay names the screen's %q key: %q", s, d, named)
			}
		}
		m.key(press("esc"))
		if m.ov != from {
			t.Errorf("%s: esc from help went to overlay %d, want back to %d", s, m.ov, from)
		}
	}
	for _, p := range overlayPages {
		m := fixture(t, p)
		var own []string
		for _, a := range m.actions() {
			own = append(own, strings.Join(strings.Fields(a.b.Help().Desc), " "))
		}
		m.key(press(helpKey(t, p)))
		named := helpNames(m)
		for _, d := range own {
			if !slices.Contains(named, d) {
				t.Errorf("%s: help over the page doesn't name its %q key: %q", p, d, named)
			}
		}
		if m.key(press("esc")); m.ov != ovNone {
			t.Errorf("%s: esc from help left overlay %d up", p, m.ov)
		}
	}
}

// helpKey is the key of state s's action table that opens help, whatever the app calls
// it: each key is pressed on a fresh fixture until one brings ovHelp up.
func helpKey(t *testing.T, s string) string {
	t.Helper()
	for _, a := range fixture(t, s).actions() {
		for _, k := range a.b.Keys() {
			p := fixture(t, s)
			p.key(press(k))
			if p.ov == ovHelp {
				return k
			}
		}
	}
	t.Fatalf("%s: no key in the action table opens help", s)
	return ""
}

// helpNames are what each help row says a key does: the row after its key (a key name
// holds no space), however wide the app's key column is.
func helpNames(m *app) []string {
	var named []string
	for _, row := range m.helpBody() {
		if f := strings.Fields(ansi.Strip(row)); len(f) > 1 {
			named = append(named, strings.Join(f[1:], " "))
		}
	}
	return named
}
