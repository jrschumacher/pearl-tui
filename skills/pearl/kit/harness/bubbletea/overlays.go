package harness

import (
	"fmt"
	"image/color"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"

	"github.com/jrschumacher/pearl-tui/skills/pearl/kit/theme/bubbletea"
	"github.com/jrschumacher/pearl-tui/skills/pearl/kit/ui/bubbletea"
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
// (truecolour, 256 colours, ASCII glyphs at truecolour) and the hidden body's (16
// colours, NO_COLOR, ASCII glyphs at 16 colours).
var overlayVariants = []struct {
	name  string
	ascii bool
	p     colorprofile.Profile
}{{"truecolor", false, colorprofile.TrueColor}, {"ansi256", false, colorprofile.ANSI256},
	{"ansi", false, colorprofile.ANSI}, {"NO_COLOR", false, colorprofile.Ascii}, {"ascii", true, colorprofile.TrueColor},
	{"ascii ansi", true, colorprofile.ANSI}}

// cellText is a cell's glyph, a blank for an empty cell.
func CellText(c *ui.Canvas, x, y int) string {
	if cell := c.CellAt(x, y); cell != nil && cell.Content != "" {
		return cell.Content
	}
	return " "
}

func RowText(c *ui.Canvas, y int) string {
	var b strings.Builder
	for x := range c.Bounds().Dx() {
		b.WriteString(CellText(c, x, y))
	}
	return b.String()
}

// An overlay's rows are its own, and where the page can be dimmed the rows it doesn't
// span are the page's: the page under a dialog is drawn unfocused (its screens take
// focused=false), so nothing has to be scanned off it afterwards. A scan for focus
// marks by glyph and ink erased cartridge's wordmark (its focus bar is █ in the primary
// ink, as is the banner) and ferrous's header separators under ASCII (its focus bar
// and its separator are both "|"). Every theme, every profile, every overlay over every
// page that draws a selection, at 80×24 and 120×40, against the same page with no
// overlay, above the footer:
//
//   - on the rows the dialog spans, nothing of the page is left beside it, at every
//     size (at 120 columns a column of "…to show it" stood beside a confirm, re-cut at
//     its margin);
//   - under a scrim, each cell off those rows holds the glyph it held, except where the
//     page drew its own focus mark (the focused field's bar, the focused button's
//     cursor, a caret), which drawing it unfocused takes away or turns blurred;
//   - with no scrim (16 colours, NO_COLOR, scrim: none, whatever the glyphs), the rows
//     of the page's body (Canvas.Body, everything when unset) are blank outside the
//     dialog, and the rest, the header, holds its glyphs: at full ink a confirm over a composer showed the
//     reply's last lines under its edge, a second copy of what it would send.
func testOverlayKeepsThePageOffItsRows(t *testing.T) {
	for _, f := range Themes(t) {
		tf, err := theme.Load(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, v := range overlayVariants {
			for _, sz := range [][2]int{{80, 24}, {120, 40}} {
				for _, page := range sp.OverlayPages {
					m := sp.New(tf, true, v.ascii)
					m.SetProfile(v.p)
					m.SetState(page)
					base := m.Render(sz[0], sz[1])
					focusCells := FocusMarks(m, base)
					for _, s := range sp.Overlays {
						if sp.ChangesPage != nil && sp.ChangesPage(s, page) {
							continue // the page under this overlay isn't the page before it
						}
						m.SetState(page)
						m.OpenOverlay(s)
						c := m.Render(sz[0], sz[1])
						name := fmt.Sprintf("%s %s %s over %s %dx%d", filepath.Base(f), v.name, s, page, sz[0], sz[1])
						if why := bodyIsThePage(c, m.Body(), m.FootY()); why != "" {
							t.Errorf("%s: %s", name, why)
						}
						scrim := scrims(m.Theme())
						hidden := hiddenBody(c)
					rows:
						for y := range m.FootY() {
							band := y >= m.OverlayRect().Min.Y && y < m.OverlayRect().Max.Y
							for x := range sz[0] {
								if uv.Pos(x, y).In(m.OverlayRect()) {
									continue
								}
								want, got := CellText(base, x, y), CellText(c, x, y)
								switch {
								case band || (!scrim && uv.Pos(x, y).In(hidden)):
									if got == " " {
										continue
									}
									if band {
										t.Errorf("%s: the page shows %q at %d,%d beside the dialog, on a row it spans: %q", name, got, x, y, RowText(c, y))
									} else {
										t.Errorf("%s: no scrim, and the page's body shows %q at %d,%d at full ink beside the dialog: %q", name, got, x, y, RowText(c, y))
									}
									continue rows
								case got == want || slices.Contains(focusCells, uv.Pos(x, y)):
									continue
								}
								t.Errorf("%s: %q at %d,%d is %q with the overlay up; the page's row was %q", name, want, x, y, got, RowText(base, y))
								continue rows
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
	// reads as it did, under a scrim and where the body is hidden (NO_COLOR), which
	// keeps the header only when Canvas.Body is set. In a copied project testTheme
	// gives the project's theme, and the failure names it.
	for _, k := range []struct {
		theme string
		ascii bool
	}{
		{"cartridge.json", false},
		{"ferrous.json", true},
	} {
		tf := ThemeFile(t, k.theme)
		for _, p := range []colorprofile.Profile{colorprofile.TrueColor, colorprofile.Ascii} {
			pname := map[colorprofile.Profile]string{colorprofile.TrueColor: "truecolor", colorprofile.Ascii: "NO_COLOR"}[p]
			for _, s := range sp.Overlays {
				for _, sz := range [][2]int{{80, 24}, {120, 40}} {
					m := sp.New(tf, true, k.ascii)
					m.SetProfile(p)
					m.SetState(sp.OverlayPages[0])
					base := m.Render(sz[0], sz[1])
					head := m.Body().Min.Y
					m.SetState(sp.OverlayPages[0])
					m.OpenOverlay(s)
					c := m.Render(sz[0], sz[1])
					for y := range min(head, m.OverlayRect().Min.Y) {
						if got, want := RowText(c, y), RowText(base, y); got != want {
							t.Errorf("theme %q ascii=%v %s %s %dx%d: header row %d reads %q under the overlay, want %q (where the page can't be dimmed a dialog hides Canvas.Body, all of the page when unset: set c.Body = body, the rect you draw the body in)", tf.Name, k.ascii, pname, s, sz[0], sz[1], y, got, want)
						}
					}
				}
			}
		}
	}
}

// scrims reports whether the kit dims the page under a dialog for th (ui's Canvas
// scrims): the theme asks for it and the profile can show it.
func scrims(th *theme.Theme) bool {
	return th.X.Scrim == "dim" && th.Profile != colorprofile.ANSI && th.Profile != colorprofile.Ascii && th.Profile != colorprofile.NoTTY
}

// hiddenBody is what a centred overlay hides where it can't dim: the rows of the
// canvas's Body, across, or the whole canvas when the app set none.
func hiddenBody(c *ui.Canvas) uv.Rectangle {
	b := c.Bounds()
	if c.Body.Empty() {
		return b
	}
	return uv.Rect(b.Min.X, c.Body.Min.Y, b.Dx(), c.Body.Dy())
}

// bodyIsThePage reports what is wrong with the Canvas.Body an app set, or "": unset is
// allowed (everything but the dialog is hidden), and a set one must span every row of
// the body the app laid out (m.body) and end at or above the footer. A Body of one row
// passed every overlay test while a NO_COLOR confirm stood over the list at full ink.
func bodyIsThePage(c *ui.Canvas, body uv.Rectangle, footY int) string {
	switch b := c.Body; {
	case b.Empty():
		return ""
	case b.Min.Y > body.Min.Y || b.Max.Y < body.Max.Y:
		return fmt.Sprintf("Canvas.Body %v doesn't span the body's rows %d–%d (m.Body() %v): a dialog leaves the rest at full ink where the page can't be dimmed; set c.Body = body", b, body.Min.Y, body.Max.Y-1, body)
	case b.Max.Y > footY:
		return fmt.Sprintf("Canvas.Body %v runs into the footer at row %d", b, footY)
	}
	return ""
}

// An overlay frame has one focus, the dialog's, and nothing of the page touches the
// dialog's edge. A recreation's list drew its focus bar beside the dialog's own, and a
// pane's cut words ("1s", "igh", "00K") sat one cell right of the edge reading as
// garbage. The page under a dialog is drawn unfocused: drawn with no scrim (where a
// mark keeps its ink), no cell of the page's body outside the dialog is a focus mark
// in the focus ink, while the same page with no overlay draws one. Every theme, every
// profile, every overlay over every page that draws a selection, at 80×24 and 120×40.
func testOverlayFrameHasOneFocus(t *testing.T) {
	seen := 0
	for _, f := range Themes(t) {
		tf, err := theme.Load(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, v := range overlayVariants {
			for _, sz := range [][2]int{{80, 24}, {120, 40}} {
				for _, page := range sp.OverlayPages {
					for _, s := range sp.Overlays {
						m := sp.New(tf, true, v.ascii)
						m.SetProfile(v.p)
						m.SetState(page)
						seen += len(FocusMarks(m, m.Render(sz[0], sz[1])))
						m.OpenOverlay(s)
						name := fmt.Sprintf("%s %s %s over %s %dx%d", filepath.Base(f), v.name, s, page, sz[0], sz[1])
						c := m.Render(sz[0], sz[1])
						if x, y, ok := pageBeside(c, m.OverlayRect(), 1); ok {
							t.Errorf("%s: the page's %q at %d,%d touches the dialog's edge", name, c.CellAt(x, y).Content, x, y)
						}
						m.Theme().X.Scrim = "none" // the page's own ink, undimmed
						c = m.Render(sz[0], sz[1])
						for _, p := range FocusMarks(m, c) {
							if !p.In(m.OverlayRect()) {
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
// at all and the focus tests judged nothing. A glyph whose ASCII twin takes several
// cells (→ as "->", » as ">>") is the mark in every cell of the twin: read as one cell,
// a select's "->" cursor and an input's ">>" prompt were no focus at all.
func FocusMarks(m App, c *ui.Canvas) []uv.Position {
	t := m.Theme()
	marks := []struct {
		g  string
		st lipgloss.Style
	}{{t.G.FocusBar, t.S.FocusBar}, {t.G.Cursor, t.S.Focus}, {t.G.Caret, t.S.Prompt}, {t.G.Prompt, t.S.Prompt}}
	fill := t.X.Selection == "fill" || t.X.Selection == "inverse"
	var out []uv.Position
	for y := m.Body().Min.Y; y < m.Body().Max.Y; y++ {
		for x := m.Body().Min.X; x+1 < m.Body().Max.X; x++ {
			cell, next := c.CellAt(x, y), c.CellAt(x+1, y)
			if fill && cell != nil && cell.Style.Attrs&uv.AttrReverse != 0 && sameRGB(cell.Style.Fg, t.S.SelRow.GetForeground()) {
				out = append(out, uv.Pos(x, y))
				continue
			}
			if cell == nil || next == nil {
				continue
			}
			for _, mk := range marks {
				// a mark of several cells is an ASCII twin ("->" for →, ">>" for »): every
				// cell of it is the mark
				rs := []rune(mk.g)
				n := len(rs)
				if n < 2 || !ascii(mk.g) {
					n = 1
				}
				spelt := mk.g != "" && sameRGB(cell.Style.Fg, mk.st.GetForeground())
				for i := 0; spelt && i < n; i++ {
					want := mk.g
					if n > 1 {
						want = string(rs[i])
					}
					spelt = CellText(c, x+i, y) == want
				}
				if !spelt {
					continue
				}
				// a mark stands before a blank, or, the cursor, before a bracket button's "["
				after := CellText(c, x+n, y)
				if strings.TrimSpace(after) != "" && (mk.g != t.G.Cursor || after != "[") {
					continue
				}
				for i := range n {
					out = append(out, uv.Pos(x+i, y))
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
// when overlays names it, scrolls names overlays and help among them, every
// overlayPages entry is a page with no dialog up, and
// every page that draws a focus mark is in overlayPages, renders as one that is, or is
// in unjudgedPages with a reason. A cold build left its menu overlay out of the help
// test, and its help over the menu named the screen's keys; leaving it out of overlays
// would hide it as well, and so would moving a page into unjudgedPages: an entry needs
// a reason, is never the main screen (states[0]), is only allowed in an app that builds
// a Huh form (huhform.NewForm), and, unless its reason names Huh, must keep a focus mark
// beside some overlay (else it blurs, and belongs in overlayPages).
func testOverlayTablesMatchTheStates(t *testing.T) {
	for _, list := range [][]string{sp.Overlays, sp.OverlayPages, slices.Collect(maps.Keys(sp.UnjudgedPages))} {
		for _, s := range list {
			if !slices.Contains(sp.States, s) {
				t.Errorf("%q is in the harness tables but not in sp.States", s)
			}
		}
	}
	// scrolls names overlays: help's keys grow with the app, so help is always one
	for _, s := range sp.Scrolls {
		if !slices.Contains(sp.Overlays, s) {
			t.Errorf("%q is in sp.Scrolls but not in sp.Overlays", s)
		}
	}
	if !slices.Contains(sp.Scrolls, "help") {
		t.Errorf("sp.Scrolls has no \"help\": the keys overlay's body grows with the app and sp.Scrolls at the floor")
	}
	// unjudgedPages is for a Huh form's pages, and every Huh form goes through huhform.NewForm
	// (tuilint code, C-HUHTHEME): an app that never builds one has nothing to leave out
	if len(sp.UnjudgedPages) > 0 && !buildsAForm(t) {
		t.Errorf("sp.UnjudgedPages lists %d pages, but no .go file here calls huhform.NewForm: only a Huh form's page can't be drawn blurred; judge them in sp.OverlayPages", len(sp.UnjudgedPages))
	}
	for s, why := range sp.UnjudgedPages {
		switch {
		case s == sp.States[0]:
			t.Errorf("%s, the main screen, is in sp.UnjudgedPages: every overlay must be judged over it", s)
		case strings.TrimSpace(why) == "":
			t.Errorf("%s is in sp.UnjudgedPages with no reason: say why its focus can't be drawn blurred (and repeat it in TUI.md § Decisions)", s)
		case slices.Contains(sp.OverlayPages, s):
			t.Errorf("%s is in both sp.OverlayPages and sp.UnjudgedPages", s)
		// a Huh page needs no proof: whether a dialog happens to cover its focused field
		// depends on sizes, not on the form, which can't draw it blurred either way
		case !strings.Contains(why, "Huh") && !keepsFocusUnderAnOverlay(t, s):
			t.Errorf("%s is in sp.UnjudgedPages, but every overlay over it draws it blurred: judge it in sp.OverlayPages", s)
		}
	}
	for _, s := range sp.States {
		if s == "toosmall" {
			continue
		}
		m := fixture(t, s)
		_, unjudged := sp.UnjudgedPages[s]
		switch in := slices.Contains(sp.Overlays, s); {
		case m.Overlay() != "" && !in:
			t.Errorf("%s opens a dialog but isn't in sp.Overlays: the overlay tests never judge it", s)
		case m.Overlay() == "" && in:
			t.Errorf("%s is in sp.Overlays but opens no dialog", s)
		case m.Overlay() != "" && slices.Contains(sp.OverlayPages, s):
			t.Errorf("%s is in sp.OverlayPages but has a dialog up", s)
		case m.Overlay() == "" && !slices.Contains(sp.OverlayPages, s) && !unjudged && len(FocusMarks(m, m.Render(80, 24))) > 0 && judgedTwin(t, s) == "":
			t.Errorf("%s draws a focus mark but is in neither sp.OverlayPages nor sp.UnjudgedPages, and renders as no page that is: no overlay is judged over it", s)
		}
	}
}

// buildsAForm reports whether a non-test .go file of the app calls huhform.NewForm.
func buildsAForm(t *testing.T) bool {
	t.Helper()
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		if b, err := os.ReadFile(f); err == nil && strings.Contains(string(b), "huhform.NewForm(") {
			return true
		}
	}
	return false
}

// judgedTwin is the overlayPages state whose page body renders as state s's at 80×24
// (services with a toast over the footer is services), or "".
func judgedTwin(t *testing.T, s string) string {
	t.Helper()
	body := func(s string) string {
		m := fixture(t, s)
		c := m.Render(80, 24)
		var rows []string
		for y := m.Body().Min.Y; y < m.Body().Max.Y; y++ {
			rows = append(rows, RowText(c, y))
		}
		return strings.Join(rows, "\n")
	}
	b := body(s)
	for _, p := range sp.OverlayPages {
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
	for _, f := range Themes(t) {
		tf, err := theme.Load(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, sz := range [][2]int{{80, 24}, {120, 40}} {
			for _, o := range sp.Overlays {
				m := sp.New(tf, true, false)
				m.SetState(s)
				m.OpenOverlay(o)
				m.Theme().X.Scrim = "none"
				c := m.Render(sz[0], sz[1])
				for _, p := range FocusMarks(m, c) {
					if !p.In(m.OverlayRect()) {
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
func testHelpOverAnOverlayNamesItsKeys(t *testing.T) {
	for _, s := range sp.Overlays {
		if s == "help" {
			continue
		}
		m := fixture(t, s)
		from := m.Overlay()
		var own []string
		for _, a := range m.Actions() {
			own = append(own, strings.Join(strings.Fields(a.B.Help().Desc), " "))
		}
		// the screen's keys: the page under the overlay, with the overlay closed
		page := fixture(t, s)
		page.CloseOverlay()
		var screenKeys []string
		for _, a := range page.Actions() {
			if d := strings.Join(strings.Fields(a.B.Help().Desc), " "); !slices.Contains(own, d) {
				screenKeys = append(screenKeys, d)
			}
		}
		k := helpKey(t, s)
		m.Key(Press(k))
		if m.Overlay() != "help" {
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
		m.Key(Press("esc"))
		if m.Overlay() != from {
			t.Errorf("%s: esc from help went to overlay %q, want back to %q", s, m.Overlay(), from)
		}
	}
	for _, p := range sp.OverlayPages {
		m := fixture(t, p)
		var own []string
		for _, a := range m.Actions() {
			own = append(own, strings.Join(strings.Fields(a.B.Help().Desc), " "))
		}
		m.Key(Press(helpKey(t, p)))
		named := helpNames(m)
		for _, d := range own {
			if !slices.Contains(named, d) {
				t.Errorf("%s: help over the page doesn't name its %q key: %q", p, d, named)
			}
		}
		if m.Key(Press("esc")); m.Overlay() != "" {
			t.Errorf("%s: esc from help left overlay %q up", p, m.Overlay())
		}
	}
}

// helpKey is the key of state s's action table that opens help, whatever the app calls
// it: each key is pressed on a fresh fixture until one brings ovHelp up.
func helpKey(t *testing.T, s string) string {
	t.Helper()
	for _, a := range fixture(t, s).Actions() {
		for _, k := range a.B.Keys() {
			p := fixture(t, s)
			p.Key(Press(k))
			if p.Overlay() == "help" {
				return k
			}
		}
	}
	t.Fatalf("%s: no key in the action table opens help", s)
	return ""
}

// helpNames are what each help row says a key does: the row after its key (a key name
// holds no space), however wide the app's key column is.
func helpNames(m App) []string {
	var named []string
	for _, row := range m.HelpBody() {
		if f := strings.Fields(ansi.Strip(row)); len(f) > 1 {
			named = append(named, strings.Join(f[1:], " "))
		}
	}
	return named
}

// ascii reports whether s is all ASCII: a mark of several cells is a twin only then.
func ascii(s string) bool {
	for _, r := range s {
		if r > 0x7e {
			return false
		}
	}
	return true
}
