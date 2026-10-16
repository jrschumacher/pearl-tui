package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"

	"tuikit/theme"
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
// that profile (a borderless dialog's edge depends on it, ui TestEveryDialog…).
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
		}{{"-profile ansi", "ansi", "", true, colorprofile.ANSI}, {"NO_COLOR=1", "truecolor", "1", false, colorprofile.Ascii}} {
			prof, err := renderProfile(pr.flag, pr.explicit, pr.noColor)
			if err != nil || prof != pr.p {
				t.Fatalf("%s: renderProfile = %v %v, want %v", pr.name, prof, err, pr.p)
			}
			for _, dark := range []bool{true, false} {
				m := newApp(tf, dark, false)
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
				}
			}
		}
	}
}

// -profile wins; without it NO_COLOR (any non-empty value) means no colour.
func TestRenderProfile(t *testing.T) {
	for _, c := range []struct {
		name     string
		explicit bool
		noColor  string
		want     colorprofile.Profile
	}{
		{"truecolor", false, "", colorprofile.TrueColor},
		{"truecolor", false, "1", colorprofile.Ascii},
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
