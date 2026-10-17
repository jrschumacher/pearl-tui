package main

import (
	"fmt"
	"image/color"
	"path/filepath"
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
// so the too-small frame (always 50×14) named toosmall-80x24 failed F-WIDTH.
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
	if got := frameName("toosmall", 120, 40); got != "toosmall-50x14.ansi" {
		t.Errorf("too-small frame named %s", got)
	}
}

// Every state, every theme, both modes, ASCII, at the sizes people actually
// use: each frame must be exactly W×H cells. One row at W+1 wraps and tears
// the whole screen, and it is invisible in code review.
func TestFootprint(t *testing.T) {
	files := testThemes(t)
	sizes := [][2]int{{80, 24}, {100, 30}, {120, 40}, {200, 60}, {60, 18}}
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

// Every theme's overlay states render under 16 colours and NO_COLOR as the harness
// writes them (-profile ansi, NO_COLOR=1): exactly W×H, with the theme resolved for
// that profile (a borderless dialog's edge depends on it, ui TestEveryDialog…). No scrim
// is drawn there, so the rows the dialog spans are cleared across: a help dialog sat
// between live list rows ("● acme/w ── keys · inbox ── mitri · 5h"), read as a collage.
// A cleared cell takes the page's background, not the one it had: a red 502s strip or
// a FAILING chip emptied by the clear stayed a solid block beside the dialog, and a
// selection band left a grey stub at its left.
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
			{"-ascii", "truecolor", "", false, colorprofile.TrueColor, true}} {
			prof, err := renderProfile(pr.flag, pr.explicit, pr.noColor)
			if err != nil || prof != pr.p {
				t.Fatalf("%s: renderProfile = %v %v, want %v", pr.name, prof, err, pr.p)
			}
			for _, dark := range []bool{true, false} {
				m := newApp(tf, dark, pr.ascii)
				m.profile = prof
				for _, s := range []string{"palette", "confirm", "help"} {
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
					r := m.ovRect
				rows:
					for y := r.Min.Y; y < r.Max.Y; y++ {
						for x := range 80 {
							if cell := c.CellAt(x, y); cell != nil && !uv.Pos(x, y).In(r) && !sameRGB(cell.Style.Bg, m.t.P.Bg) {
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
// confirm). No text cell on a confirm frame at -profile ansi draws in its background's
// slot, for any theme, in either mode; the dialog is judged too, since its own text is
// fg on bgPanel.
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
			m := newApp(tf, dark, false)
			m.profile = colorprofile.ANSI
			m.setState("confirm")
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
				t.Errorf("%s dark=%v: %d text cells draw in their background's slot at 16 colours", filepath.Base(f), dark, hidden)
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

// dialogText is the plain text of the rows a dialog was drawn on (m.ovRect) and the
// footer: what a test about the dialog judges. The page under it has cut rows of its
// own ("↑ 12 more", "… (6 lines hidden)" in a transcript), and a copied test that
// scanned the whole frame failed under every overlay of an app whose page has one.
func dialogText(c *ui.Canvas, r uv.Rectangle, fy int) string {
	rows := strings.Split(ansi.Strip(c.String()), "\n")
	keep := append([]string{}, rows[max(0, r.Min.Y):min(len(rows), r.Max.Y)]...)
	keep = append(keep, rows[min(len(rows), fy):]...)
	return strings.Join(keep, "\n")
}

// A dialog never reaches the footer: the help dialog was 20 rows, so at 80×24 its
// bottom edge landed on the footer rule (a double rule) and at 60×18 both were
// clipped. Every overlay state, every theme, both sizes: the dialog ends above the
// footer, a row of air between while it fits; a body that doesn't fit scrolls (↑↓),
// and a dialog whose keys don't scroll (confirm, palette) is never cut at the kit's own
// minimum size, where a rollback confirmation hid its consequences ("… 2 more"). Each
// overlay is judged over the services list and over the assistant's transcript, whose
// tail is cut ("↑ n more", "… (n lines hidden)"): only the dialog's rows count.
func TestDialogsStayAboveTheFooter(t *testing.T) {
	for _, f := range testThemes(t) {
		tf, err := theme.Load(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, sz := range [][2]int{{80, 24}, {60, 18}} {
			w, h := sz[0], sz[1]
			for _, page := range []screen{scrServices, scrAssistant} {
				for _, s := range []string{"help", "confirm", "palette"} {
					m := newApp(tf, true, false)
					m.setState(s)
					if page == scrAssistant {
						// the page a dialog is judged over: the assistant's cut transcript
						m.screen, m.sent = scrAssistant, []string{"ship it as v2.4.1"}
					}
					if page == scrAssistant {
						ov := m.ov
						m.ov = ovNone
						if full := ansi.Strip(m.render(w, h).String()); !strings.Contains(full, m.t.G.More+" ") || !strings.Contains(full, " more") {
							t.Fatalf("%s %s %dx%d: the transcript under the dialog has no cut row to be fooled by", filepath.Base(f), s, w, h)
						}
						m.ov = ov
					}
					c := m.render(w, h)
					fy := h - len(ui.Footer(m.t, w, "", nil, ""))
					frame := dialogText(c, m.ovRect, fy)
					if m.ovRect.Max.Y > fy {
						t.Errorf("%s %s %dx%d: the dialog's last row is %d, the footer starts at %d", filepath.Base(f), s, w, h, m.ovRect.Max.Y-1, fy)
					}
					if m.ovRect.Dy() <= fy-5 && m.ovRect.Max.Y > fy-1 {
						t.Errorf("%s %s %dx%d: the dialog fits with air (%d rows) but ends on row %d, against the footer at %d", filepath.Base(f), s, w, h, m.ovRect.Dy(), m.ovRect.Max.Y-1, fy)
					}
					if m.ovRect.Dy() == 0 {
						t.Errorf("%s %s %dx%d: no dialog drawn", filepath.Base(f), s, w, h)
					}
					if cut := strings.Contains(frame, m.t.G.Ellipsis+" ") && strings.Contains(frame, " more"); cut && !strings.Contains(frame, "scroll") {
						t.Errorf("%s %s %dx%d: the body is cut (\"%s n more\") with no scroll key in the footer", filepath.Base(f), s, w, h, m.t.G.Ellipsis)
					}
					if s != "help" && strings.Contains(frame, m.t.G.Ellipsis+" ") && strings.Contains(frame, " more") {
						t.Errorf("%s %s %dx%d: a dialog that fits the kit's minimum size is cut:\n%s", filepath.Base(f), s, w, h, frame)
					}
				}
			}
			// the help body scrolls when cut: a "more" row at the bottom, then one above
			m := newApp(tf, true, false)
			m.setState("help")
			fy := h - len(ui.Footer(m.t, w, "", nil, ""))
			frame := dialogText(m.render(w, h), m.ovRect, fy)
			if more := m.t.G.Ellipsis + " "; len(m.helpBody()) > m.helpRows() && !strings.Contains(frame, more) {
				t.Errorf("%s help %dx%d: %d rows of keys in %d: want a %q row", filepath.Base(f), w, h, len(m.helpBody()), m.helpRows(), more)
			}
			if len(m.helpBody()) > m.helpRows() {
				m.key(press("down"))
				if f2 := dialogText(m.render(w, h), m.ovRect, fy); !strings.Contains(f2, m.t.G.More+" 1 more") {
					t.Errorf("%s help %dx%d: after down, no %q row", filepath.Base(f), w, h, m.t.G.More+" 1 more")
				}
				if !strings.Contains(frame, "scroll") {
					t.Errorf("%s help %dx%d: the body scrolls but the footer doesn't say so", filepath.Base(f), w, h)
				}
			} else if strings.Contains(frame, "scroll") {
				t.Errorf("%s help %dx%d: the body fits but the footer offers scroll", filepath.Base(f), w, h)
			}
		}
	}
}
