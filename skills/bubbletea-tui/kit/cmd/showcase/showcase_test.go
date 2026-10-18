package main

import (
	"fmt"
	"image/color"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"

	"tuikit/theme"
	"tuikit/ui"
)

// testThemes finds the theme JSONs two levels up: kit/themes here, and the project's
// themes/ once the showcase is copied to cmd/<app> (kit/README.md). Finding none fails,
// so a copied test never passes by testing nothing.
func testThemes(t *testing.T) []string {
	t.Helper()
	files, _ := filepath.Glob("../../themes/*.json")
	if len(files) == 0 {
		t.Fatal("no theme JSON in ../../themes")
	}
	return files
}

// testTheme loads the named theme where it exists (the kit) and the project's own
// first theme where it doesn't.
func testTheme(t *testing.T, name string) theme.File {
	t.Helper()
	files := testThemes(t)
	path := files[0]
	for _, f := range files {
		if filepath.Base(f) == name {
			path = f
		}
	}
	f, err := theme.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

// An -out file's name is the size it holds: tuilint frames reads W×H from the name,
// so the too-small frame (always 50×14 in the kit, below its 60×18 floor) named
// toosmall-80x24 failed F-WIDTH. The too-small frame is one size whatever was asked,
// below the app's floor (minW×minH), and is named for it.
func TestFrameNameIsRenderedSize(t *testing.T) {
	m := newApp(testTheme(t, "halyard.json"), true, false)
	for _, s := range states {
		for _, sz := range [][2]int{{80, 24}, {120, 40}} {
			name := frameName(s, sz[0], sz[1])
			var w, h int
			if _, err := fmt.Sscanf(name[len(s):], "-%dx%d.ansi", &w, &h); err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			m.setState(s)
			rw, rh := frameSize(s, sz[0], sz[1])
			rows := strings.Split(ansi.Strip(m.render(rw, rh).String()), "\n")
			if len(rows) != h || lipgloss.Width(rows[0]) != w {
				t.Errorf("%s holds %dx%d", name, lipgloss.Width(rows[0]), len(rows))
			}
		}
	}
	tw, th := frameSize("toosmall", 120, 40)
	if tw >= minW && th >= minH {
		t.Errorf("the too-small frame renders at %dx%d, not below the floor %dx%d", tw, th, minW, minH)
	}
	want := fmt.Sprintf("toosmall-%dx%d.ansi", tw, th)
	for _, sz := range [][2]int{{80, 24}, {120, 40}} {
		if got := frameName("toosmall", sz[0], sz[1]); got != want {
			t.Errorf("too-small frame asked at %dx%d named %s, want %s", sz[0], sz[1], got, want)
		}
	}
}

// Every state, every theme, both modes, ASCII, at the sizes people actually
// use: each frame must be exactly W×H cells. One row at W+1 wraps and tears
// the whole screen, and it is invisible in code review.
func TestFootprint(t *testing.T) {
	files := testThemes(t)
	// the app's floor (TUI.md § Size floor; 60×18 in the kit) and the sizes above it: a
	// size under the floor draws the too-small screen, which -golden checks
	sizes := [][2]int{{minW, minH}}
	for _, sz := range [][2]int{{80, 24}, {100, 30}, {120, 40}, {200, 60}} {
		if sz[0] >= minW && sz[1] >= minH && sz != [2]int{minW, minH} {
			sizes = append(sizes, sz)
		}
	}
	for _, f := range files {
		tf, err := theme.Load(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, mode := range []struct{ dark, ascii bool }{{true, false}, {false, false}, {true, true}} {
			m := newApp(tf, mode.dark, mode.ascii)
			for _, s := range states {
				if s == "toosmall" {
					continue
				}
				for _, sz := range sizes {
					m.setState(s)
					out := ansi.Strip(m.render(sz[0], sz[1]).String())
					rows := strings.Split(out, "\n")
					if len(rows) != sz[1] {
						t.Errorf("%s %s %dx%d: %d rows", filepath.Base(f), s, sz[0], sz[1], len(rows))
						continue
					}
					for i, r := range rows {
						if w := lipgloss.Width(r); w != sz[0] {
							t.Errorf("%s %s %dx%d dark=%v ascii=%v: row %d is %d cells", filepath.Base(f), s, sz[0], sz[1], mode.dark, mode.ascii, i, w)
							break
						}
					}
				}
			}
		}
	}
}

// The ASCII set must not leak a non-ASCII glyph from the theme's vocabulary.
func TestASCIIFallback(t *testing.T) {
	files := testThemes(t)
	for _, f := range files {
		tf, _ := theme.Load(f)
		th := theme.Resolve(tf, true, true)
		g := th.G
		for _, s := range append([]string{g.HeaderRule, g.Rule, g.FocusBar, g.Cursor, g.Prompt, g.Done, g.Current, g.Todo,
			g.Fail, g.Bullet, g.Sep, g.Arrow, g.Ellipsis, g.TreeMid, g.TreeEnd, g.Thumb, g.Track}, g.Spinner...) {
			for _, r := range s {
				if r > 127 {
					t.Errorf("%s: ASCII glyph set contains %q", filepath.Base(f), s)
				}
			}
		}
	}
}

// Contrast floors on every theme's resolved palette (both modes).
func TestContrast(t *testing.T) {
	files := testThemes(t)
	for _, f := range files {
		tf, _ := theme.Load(f)
		for _, dark := range []bool{true, false} {
			th := theme.Resolve(tf, dark, false)
			p := th.P
			type check struct {
				name string
				a, b interface{ RGBA() (r, g, b, a uint32) }
				min  float64
			}
			checks := []check{
				{"fg/bg", p.Fg, p.Bg, 7}, {"fgMuted/bg", p.FgMuted, p.Bg, 4.0}, {"fgSubtle/bg", p.FgSubtle, p.Bg, 2.2},
				{"primary/bg", p.Primary, p.Bg, 3}, {"onPrimary/primary", p.OnPrimary, p.Primary, 3.5},
				{"error/bg", p.Error, p.Bg, 3},
				{"bgOverlay/bg", p.BgOverlay, p.Bg, 1.0},
			}
			// the overlay step (the palette's fill) is a step from bg toward fg, never past the
			// text or on bg's other side: a light theme's dark overlay hid the palette's text
			if lo, lb, lf := theme.Luminance(p.BgOverlay), theme.Luminance(p.Bg), theme.Luminance(p.Fg); (lo-lb)*(lf-lb) < 0 || theme.Contrast(p.BgOverlay, p.Bg) > 3 {
				t.Errorf("%s dark=%v: bgOverlay %.2f:1 from bg is not a step toward fg", filepath.Base(f), dark, theme.Contrast(p.BgOverlay, p.Bg))
			}
			// a bar footer draws the key hints (KeyDesc, fgSubtle) on bgSubtle: tuilint's 2.3 floor
			if th.X.Footer == "bar" {
				checks = append(checks, check{"fgSubtle/bgSubtle (footer bar)", p.FgSubtle, p.BgSubtle, 2.3})
			}
			for _, c := range checks {
				if r := theme.Contrast(c.a, c.b); r < c.min {
					t.Errorf("%s dark=%v: %s contrast %.2f < %.1f", filepath.Base(f), dark, c.name, r, c.min)
				}
			}
		}
	}
}

// Every visible glyph carries an explicit foreground from the theme. Unstyled
// text takes the terminal's default fg, which no palette check ever saw (on a
// painted light background it can vanish).
func TestNoUnstyledText(t *testing.T) {
	files := testThemes(t)
	for _, f := range files {
		tf, _ := theme.Load(f)
		m := newApp(tf, true, false)
		for _, s := range states {
			m.setState(s)
			c := m.render(120, 40)
			b := c.Bounds()
			for y := b.Min.Y; y < b.Max.Y; y++ {
				for x := b.Min.X; x < b.Max.X; x++ {
					cell := c.CellAt(x, y)
					if cell != nil && cell.Style.Fg == nil && strings.TrimSpace(cell.Content) != "" {
						t.Errorf("%s %s: unstyled %q at row %d col %d", filepath.Base(f), s, cell.Content, y, x)
						goto next
					}
				}
			}
		next:
		}
	}
}

// Every theme's overlay states render under 16 colours, NO_COLOR and ASCII at 16
// colours as the harness writes them (-profile ansi, NO_COLOR=1, -ascii -profile
// ansi): exactly W×H, with the theme resolved for
// that profile (a borderless dialog's edge depends on it, ui TestEveryDialog…). No scrim
// is drawn there, so the rows the dialog spans are cleared across and the page's body
// (Canvas.Body) is hidden: a help dialog sat between live list rows ("● acme/w ── keys
// · inbox ── mitri · 5h"), read as a collage, and a confirm over a composer showed the
// reply's tail under its edge at full ink. A cleared cell takes the page's background,
// not the one it had: a red 502s strip or a FAILING chip emptied by the clear stayed a
// solid block beside the dialog, and a selection band left a grey stub at its left.
func TestOverlaysInReducedProfiles(t *testing.T) {
	for _, f := range testThemes(t) {
		tf, err := theme.Load(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, pr := range []struct {
			name, flag, noColor string
			explicit            bool
			p                   colorprofile.Profile
			ascii               bool
		}{{"-profile ansi", "ansi", "", true, colorprofile.ANSI, false}, {"NO_COLOR=1", "truecolor", "1", false, colorprofile.Ascii, false},
			{"-ascii -profile ansi", "ansi", "", true, colorprofile.ANSI, true}} {
			prof, err := renderProfile(pr.flag, pr.explicit, pr.noColor)
			if err != nil || prof != pr.p {
				t.Fatalf("%s: renderProfile = %v %v, want %v", pr.name, prof, err, pr.p)
			}
			for _, dark := range []bool{true, false} {
				m := newApp(tf, dark, pr.ascii)
				m.profile = prof
				for _, s := range overlays {
					m.setState(s)
					if m.t.Profile != prof {
						t.Fatalf("%s %s: theme resolved for %v, want %v", filepath.Base(f), s, m.t.Profile, prof)
					}
					c := m.render(80, 24)
					c.PaintBackground()
					if err := checkSize(ansi.Strip(downsample(c.String(), prof)), s, 80, 24); err != nil {
						t.Errorf("%s %s dark=%v: %v", filepath.Base(f), pr.name, dark, err)
					}
					if x, y, ok := pageBeside(c, m.ovRect, 80); ok {
						t.Errorf("%s %s %s dark=%v: no scrim, and the page shows %q at %d,%d beside the dialog", filepath.Base(f), s, pr.name, dark, c.CellAt(x, y).Content, x, y)
					}
					r, body := m.ovRect, hiddenBody(c)
				rows:
					for y := range m.footY {
						for x := range 80 {
							if !(y >= r.Min.Y && y < r.Max.Y) && !uv.Pos(x, y).In(body) || uv.Pos(x, y).In(r) {
								continue
							}
							cell := c.CellAt(x, y)
							if cell != nil && strings.TrimSpace(cell.Content) != "" {
								t.Errorf("%s %s %s dark=%v: no scrim, and the page's body shows %q at %d,%d beside the dialog", filepath.Base(f), s, pr.name, dark, cell.Content, x, y)
								break rows
							}
							if cell != nil && !sameRGB(cell.Style.Bg, m.t.P.Bg) {
								t.Errorf("%s %s %s dark=%v: a cleared cell at %d,%d keeps a background %v, not the page's %v", filepath.Base(f), s, pr.name, dark, x, y, cell.Style.Bg, m.t.P.Bg)
								break rows
							}
						}
					}
				}
			}
		}
	}
}

// At 16 colours the scrim is skipped: the page is ANSI 0 or 15 there, so dimming text
// toward it put every level on the page's own slot (93 invisible cells under halyard's
// confirm). No text cell on an overlay frame (each of overlays) at -profile ansi draws in
// its background's slot, for any theme, in either mode; the dialog is judged too, since
// its own text is fg on bgPanel.
func TestNoScrimAt16Colours(t *testing.T) {
	slot := func(c color.Color, def color.Color) int {
		if c == nil {
			c = def
		}
		if b, ok := colorprofile.ANSI.Convert(c).(ansi.BasicColor); ok {
			return int(b)
		}
		return -1
	}
	for _, f := range testThemes(t) {
		tf, err := theme.Load(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, dark := range []bool{true, false} {
			for _, s := range overlays {
				m := newApp(tf, dark, false)
				m.profile = colorprofile.ANSI
				m.setState(s)
				c := m.render(80, 24)
				c.PaintBackground()
				hidden := 0
				for y := 0; y < 24; y++ {
					for x := 0; x < 80; x++ {
						cell := c.CellAt(x, y)
						if cell == nil || strings.TrimSpace(cell.Content) == "" {
							continue
						}
						if slot(cell.Style.Fg, m.t.P.Fg) == slot(cell.Style.Bg, m.t.P.Bg) {
							hidden++
						}
					}
				}
				if hidden > 0 {
					t.Errorf("%s %s dark=%v: %d text cells draw in their background's slot at 16 colours", filepath.Base(f), s, dark, hidden)
				}
			}
		}
	}
}

// -profile wins; without it NO_COLOR means no colour when strconv.ParseBool says true,
// exactly as colorprofile reads it for the live program.
func TestRenderProfile(t *testing.T) {
	for _, c := range []struct {
		name     string
		explicit bool
		noColor  string
		want     colorprofile.Profile
	}{
		{"truecolor", false, "", colorprofile.TrueColor},
		{"truecolor", false, "1", colorprofile.Ascii},
		{"truecolor", false, "true", colorprofile.Ascii},
		{"truecolor", false, "0", colorprofile.TrueColor},
		{"truecolor", false, "false", colorprofile.TrueColor},
		{"truecolor", false, "yes", colorprofile.TrueColor},
		{"truecolor", false, "x", colorprofile.TrueColor},
		{"ansi", true, "1", colorprofile.ANSI},
		{"ansi256", true, "", colorprofile.ANSI256},
	} {
		if got, err := renderProfile(c.name, c.explicit, c.noColor); err != nil || got != c.want {
			t.Errorf("renderProfile(%q, %v, %q) = %v %v, want %v", c.name, c.explicit, c.noColor, got, err, c.want)
		}
	}
	if _, err := renderProfile("sixteen", true, ""); err == nil {
		t.Error("renderProfile(sixteen) accepted")
	}
}

// dialogPos is what a dialog drawn at r says of its body's position: Clip's rows, an
// "↑ up more" above and a "… below more" below, and a "lines first–last of total" set
// into an edge (ui.DialogLines). Only cells inside r are read, and a Clip marker only as
// a row of its own: a page's cut rows ("↑ 12 more", "… (6 lines hidden)"), a narrow
// footer's ellipsis and body text that says "one more line" are none of it.
type dialogPos struct {
	up, below          int
	lines              bool
	first, last, total int
}

// hides reports rows of the body not on screen.
func (p dialogPos) hides() bool {
	return p.up > 0 || p.below > 0 || (p.lines && (p.first > 1 || p.last < p.total))
}

func (p dialogPos) String() string {
	if p.lines {
		return fmt.Sprintf("lines %d–%d of %d", p.first, p.last, p.total)
	}
	return fmt.Sprintf("↑ %d / … %d", p.up, p.below)
}

var dialogLinesRe = regexp.MustCompile(`lines (\d+)[–-](\d+) of (\d+)`)

// readPos reads a dialog's position (dialogPos) off the canvas.
func readPos(th *theme.Theme, c *ui.Canvas, r uv.Rectangle) dialogPos {
	marker := func(g string) *regexp.Regexp {
		// the row is the marker alone, between the dialog's side edges if it has any
		return regexp.MustCompile(`^\S?\s*` + regexp.QuoteMeta(g) + ` (\d+) more\s*\S?$`)
	}
	upRe, belowRe := marker(th.G.More), marker(th.G.Ellipsis)
	var p dialogPos
	for _, row := range rectRows(c, r) {
		row = strings.TrimSpace(row)
		if m := upRe.FindStringSubmatch(row); m != nil {
			p.up, _ = strconv.Atoi(m[1])
		}
		if m := belowRe.FindStringSubmatch(row); m != nil {
			p.below, _ = strconv.Atoi(m[1])
		}
		if m := dialogLinesRe.FindStringSubmatch(row); m != nil {
			p.lines = true
			p.first, _ = strconv.Atoi(m[1])
			p.last, _ = strconv.Atoi(m[2])
			p.total, _ = strconv.Atoi(m[3])
		}
	}
	return p
}

// rectRows are the plain rows of the cells inside r.
func rectRows(c *ui.Canvas, r uv.Rectangle) []string {
	var rows []string
	for y := max(r.Min.Y, 0); y < min(r.Max.Y, c.Bounds().Max.Y); y++ {
		var b strings.Builder
		for x := max(r.Min.X, 0); x < min(r.Max.X, c.Bounds().Max.X); x++ {
			if cell := c.CellAt(x, y); cell != nil && cell.Content != "" {
				b.WriteString(cell.Content)
			} else if cell == nil || cell.Width != 0 {
				b.WriteString(" ")
			}
		}
		rows = append(rows, b.String())
	}
	return rows
}

// footerText is the plain text of the footer's rows, from fy down: what the app's
// keys say. A check of the footer reads only these; a help dialog over a page with a
// scroll key names "scroll" in its own body.
func footerText(c *ui.Canvas, fy int) string {
	rows := strings.Split(ansi.Strip(c.String()), "\n")
	return strings.Join(rows[min(len(rows), max(0, fy)):], "\n")
}

// A dialog never reaches the footer: the help dialog was 20 rows, so at 80×24 its
// bottom edge landed on the footer rule (a double rule) and at 60×18 both were
// clipped. Every overlay over every page that draws a selection, every theme, at 80×24
// and at the app's floor (minW×minH): the dialog ends above the footer, a row of air
// between while it fits; a body that doesn't fit scrolls (↑↓) and draws its position,
// and only a dialog in scrolls may be cut at the floor: a rollback confirmation cut
// there hid its consequences ("… 2 more"). Only the dialog's own cells are read
// (readPos): the page under it may cut rows of its own.
func TestDialogsStayAboveTheFooter(t *testing.T) {
	for _, f := range testThemes(t) {
		tf, err := theme.Load(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, sz := range [][2]int{{max(80, minW), max(24, minH)}, {minW, minH}} {
			w, h := sz[0], sz[1]
			for _, page := range overlayPages {
				for _, s := range overlays {
					m := newApp(tf, true, false)
					m.setState(page)
					m.openOverlay(s)
					dialogFits(t, fmt.Sprintf("%s %s over %s %dx%d", filepath.Base(f), s, page, w, h), m, s, w, h)
				}
			}
			// the help body scrolls when cut: a "more" row at the bottom, then one above
			m := newApp(tf, true, false)
			m.setState("help")
			c := m.render(w, h)
			fy := m.footY
			p, foot := readPos(m.t, c, m.ovRect), footerText(c, fy)
			if len(m.helpBody()) > m.helpRows() {
				if !p.hides() {
					t.Errorf("%s help %dx%d: %d rows of keys in %d, and the dialog draws no position", filepath.Base(f), w, h, len(m.helpBody()), m.helpRows())
				}
				m.key(press("down"))
				if p2 := readPos(m.t, m.render(w, h), m.ovRect); p2.up != 1 && !(p2.lines && p2.first == 2) {
					t.Errorf("%s help %dx%d: after down the position is %v, want one row scrolled off", filepath.Base(f), w, h, p2)
				}
				if !strings.Contains(foot, "scroll") {
					t.Errorf("%s help %dx%d: the body scrolls but the footer doesn't say so", filepath.Base(f), w, h)
				}
			} else if strings.Contains(foot, "scroll") {
				t.Errorf("%s help %dx%d: the body fits but the footer offers scroll", filepath.Base(f), w, h)
			}
		}
	}
}

// dialogFits judges the overlay s that m draws at w×h (TestDialogsStayAboveTheFooter).
func dialogFits(t *testing.T, name string, m *app, s string, w, h int) {
	t.Helper()
	c := m.render(w, h)
	fy := m.footY // the app's own footer row, as render drew it
	if m.ovRect.Max.Y > fy {
		t.Errorf("%s: the dialog's last row is %d, the footer starts at %d", name, m.ovRect.Max.Y-1, fy)
	}
	if m.ovRect.Dy() <= fy-5 && m.ovRect.Max.Y > fy-1 {
		t.Errorf("%s: the dialog fits with air (%d rows) but ends on row %d, against the footer at %d", name, m.ovRect.Dy(), m.ovRect.Max.Y-1, fy)
	}
	if m.ovRect.Dy() == 0 {
		t.Errorf("%s: no dialog drawn", name)
	}
	p := readPos(m.t, c, m.ovRect)
	if p.hides() && !strings.Contains(footerText(c, fy), "scroll") {
		t.Errorf("%s: the body is cut (%v) with no scroll key in the footer", name, p)
	}
	if p.hides() && !slices.Contains(scrolls, s) {
		t.Errorf("%s: a dialog that isn't in scrolls is cut at the floor (%v): make it fit, or, if its body must be shown whole and has no length bound, list it in scrolls:\n%s", name, p, strings.Join(rectRows(c, m.ovRect), "\n"))
	}
	if slices.Contains(scrolls, s) {
		key := scrollKey(m)
		if err := scrollsToItsEnd(m.t, func() (*ui.Canvas, uv.Rectangle) { c := m.render(w, h); return c, m.ovRect }, func() { m.key(press(key)) }); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// scrollKey is the key that scrolls the open overlay's body down: a key of the action
// whose help says "scroll", down or j or pgdown first; down when none says so.
func scrollKey(m *app) string {
	for _, a := range m.actions() {
		if !strings.Contains(a.b.Help().Desc, "scroll") {
			continue
		}
		for _, k := range []string{"down", "j", "pgdown", "space"} {
			if slices.Contains(a.b.Keys(), k) {
				return k
			}
		}
	}
	return "down"
}

// scrollsToItsEnd presses down (press) on a dialog that scrolls (render draws it) until
// it stops changing, and says what is wrong: rows went by with no position drawn
// (nothing unsent is hidden without one), the position jumped past rows or went back,
// or the end of the body was never reached. A dialog that fits passes.
func scrollsToItsEnd(th *theme.Theme, render func() (*ui.Canvas, uv.Rectangle), press func()) error {
	c, r := render()
	p, text := readPos(th, c, r), strings.Join(rectRows(c, r), "\n")
	for range 2000 {
		press()
		c2, r2 := render()
		p2, text2 := readPos(th, c2, r2), strings.Join(rectRows(c2, r2), "\n")
		if text2 == text {
			break
		}
		switch {
		case !p.hides():
			return fmt.Errorf("the body scrolled, but no position was drawn while rows were hidden (Clip's rows or ui.DialogLines):\n%s", text)
		case p.lines != p2.lines:
			return fmt.Errorf("the position changed form while scrolling: %v, then %v", p, p2)
		case p.lines && (p2.total != p.total || p2.first <= p.first || p2.first > p.last+1):
			return fmt.Errorf("scrolling down went from %v to %v: every line is shown on the way, in order", p, p2)
		case !p.lines && p2.up <= p.up:
			return fmt.Errorf("scrolling down went from %v to %v: the rows above grow by each step", p, p2)
		}
		p, text = p2, text2
	}
	if p.below > 0 || (p.lines && p.last < p.total) {
		return fmt.Errorf("scrolling down stops at %v, before the body's end", p)
	}
	return nil
}

// The scroll allowance is checked, not trusted: a dialog that scrolls with
// ui.DialogLines in its edge and reaches its end passes, and one that scrolls with no
// position, or stops short, fails. A write confirm whose verbatim body has no length
// bound scrolls; what the kept tests hold is that nothing is hidden without a position.
func TestScrollAllowanceIsChecked(t *testing.T) {
	th := theme.ResolveProfile(testTheme(t, "charm.json"), true, false, colorprofile.TrueColor)
	body := make([]string, 30)
	for i := range body {
		body[i] = fmt.Sprintf("line %d of the reply, one more line to read", i+1)
	}
	const shown = 6
	for _, c := range []struct {
		name string
		draw func(off int) string
		stop int // the last offset down reaches
		ok   bool
	}{
		{"lines in the edge", func(off int) string {
			return ui.DialogStatus(th, 60, 0, "Send reply?", ui.DialogLines(th, off, shown, len(body)), body[off:off+shown], 0, "", "")
		}, len(body) - shown, true},
		{"Clip's rows", func(off int) string {
			frame := 100 - ui.DialogBodyRows(th, 100, "Send reply?", "", "")
			return ui.DialogScrolled(th, 60, shown+frame, "Send reply?", body, off, "", "")
		}, len(body), true},
		{"no position", func(off int) string {
			return ui.Dialog(th, 60, 0, "Send reply?", body[off:off+shown], "", "")
		}, len(body) - shown, false},
		{"stops short", func(off int) string {
			return ui.DialogStatus(th, 60, 0, "Send reply?", ui.DialogLines(th, off, shown, len(body)), body[off:off+shown], 0, "", "")
		}, 10, false},
	} {
		off := 0
		render := func() (*ui.Canvas, uv.Rectangle) {
			cv := ui.NewCanvas(th, 80, 40)
			return cv, cv.Overlay(c.draw(off), nil)
		}
		err := scrollsToItsEnd(th, render, func() { off = min(off+1, c.stop) })
		if (err == nil) != c.ok {
			t.Errorf("%s: scrollsToItsEnd = %v, want ok=%v", c.name, err, c.ok)
		}
	}
}

// One focus per frame in shape, not only in colour: an -ascii frame (what NO_COLOR and
// TERM=linux read) has its focus marker, the focus bar's twin under selection bar and the
// cursor's under caret, standing before a row in one column at most. A list blurred behind
// a palette kept the focus bar in fgSubtle, so the frame showed two | columns (Law 5); the
// blurred row now takes Theme.BlurBar, : under ASCII. Every showcase state is checked
// (its services list is a ui.Table), and so is a blurred ui.List beside a focused one.
func TestOneFocusMarkerColumn(t *testing.T) {
	for _, f := range testThemes(t) {
		tf, err := theme.Load(f)
		if err != nil {
			t.Fatal(err)
		}
		m := newApp(tf, true, true)
		mode := m.t.X.Selection
		if mode != "bar" && mode != "caret" {
			continue // fill and inverse mark the row, not a column
		}
		focus := m.t.G.FocusBar
		if mode == "caret" {
			focus = m.t.G.Cursor
		}
		check := func(what string, c *ui.Canvas, edge uv.Rectangle) {
			if cols := markerColumns(c, focus, edge); len(cols) > 1 {
				t.Errorf("%s %s -ascii: the focus marker %q stands in %d columns %v (column: rows), want one: a blurred selection takes Theme.BlurBar (%q)",
					filepath.Base(f), what, focus, len(cols), cols, m.t.BlurBar())
			}
		}
		for _, s := range states {
			for _, sz := range [][2]int{{80, 24}, {120, 40}} {
				m.setState(s)
				c := m.render(sz[0], sz[1])
				check(fmt.Sprintf("%s %dx%d", s, sz[0], sz[1]), c, m.ovRect)
			}
		}
		// two panes of ui.List: the sidebar blurred, the main list focused
		items := []ui.Item{{Title: "api"}, {Title: "worker"}, {Title: "queue"}}
		c := ui.NewCanvas(m.t, 60, 3)
		c.Draw(strings.Join(ui.List(m.t, items, 1, 0, 24, 3, false), "\n"), ui.Rect(0, 0, 24, 3))
		c.Draw(strings.Join(ui.List(m.t, items, 2, 0, 30, 3, true), "\n"), ui.Rect(28, 0, 30, 3))
		c.ASCIIfy()
		check("two lists", c, uv.Rectangle{})
	}
}

// markerColumns maps each column where mark stands alone before a row ("  | name") to
// its rows; a tree rail "|- build", a separator "quit | ? keys" and the edges of the
// overlay at edge are not markers.
func markerColumns(c *ui.Canvas, mark string, edge uv.Rectangle) map[int][]int {
	b := c.Bounds()
	at := func(x, y int) string {
		if x < b.Min.X || x >= b.Max.X {
			return " "
		}
		if cell := c.CellAt(x, y); cell != nil && cell.Content != "" {
			return cell.Content
		}
		return " "
	}
	cols := map[int][]int{}
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if at(x, y) != mark || at(x-1, y) != " " || at(x+1, y) != " " || at(x+2, y) == " " {
				continue
			}
			if !edge.Empty() && (x == edge.Min.X || x == edge.Max.X-1) {
				continue
			}
			cols[x] = append(cols[x], y)
		}
	}
	return cols
}

// Text is not a glyph: a reading with a unit (21.5°C), a product (3×4) and a name with
// an accent (Zoë) keep their letters and digits under -ascii (21.5 C, 3x4, Zoe); the
// sweep drew every rune outside its twin tables as "?", and a unit read as a question
// that F-ASCII could not see. No "?" may appear in an ASCII frame where the unicode
// frame has none: every state, every theme, at 80×24 and 120×40.
func TestASCIIKeepsTextRunes(t *testing.T) {
	for _, f := range testThemes(t) {
		tf, err := theme.Load(f)
		if err != nil {
			t.Fatal(err)
		}
		m, a := newApp(tf, true, false), newApp(tf, true, true)
		for _, s := range states {
			for _, sz := range [][2]int{{80, 24}, {120, 40}} {
				w, h := frameSize(s, sz[0], sz[1])
				m.setState(s)
				a.setState(s)
				uni, asc := ansi.Strip(m.render(w, h).String()), ansi.Strip(a.render(w, h).String())
				if strings.Count(asc, "?") != strings.Count(uni, "?") {
					t.Errorf("%s %s %dx%d: the ASCII frame draws a \"?\" the unicode frame has not:\n%s", filepath.Base(f), s, w, h, asc)
				}
			}
		}
	}
}
