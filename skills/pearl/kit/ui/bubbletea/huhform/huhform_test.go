package huhform

import (
	"bytes"
	"fmt"
	"image/color"
	"path/filepath"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"

	"github.com/jrschumacher/pearl-tui/skills/pearl/kit/theme/bubbletea"
)

// testThemes and quirePath find the kit's theme JSONs and the quire test theme: three
// levels up in the kit (ui/bubbletea/huhform/../../..), one fewer in a project that
// copied ui/bubbletea flat. Finding none fails: a glob that matches nothing would pass
// every test without running one.
func testThemes(t *testing.T) []string {
	t.Helper()
	for _, g := range []string{"../../../themes/*.json", "../../themes/*.json"} {
		if files, _ := filepath.Glob(g); len(files) > 0 {
			return files
		}
	}
	t.Fatal("no theme JSON in ../../../themes")
	return nil
}

func quirePath(t *testing.T) string {
	t.Helper()
	for _, p := range []string{
		filepath.Join("..", "..", "..", "theme", "bubbletea", "testdata", "quire.json"),
		filepath.Join("..", "..", "theme", "testdata", "quire.json"),
	} {
		if files, _ := filepath.Glob(p); len(files) > 0 {
			return p
		}
	}
	t.Fatal("no quire.json in ../../../theme/bubbletea/testdata")
	return ""
}

func sameColor(a, b color.Color) bool {
	if a == nil || b == nil {
		return a == b
	}
	r1, g1, b1, _ := a.RGBA()
	r2, g2, b2, _ := b.RGBA()
	return r1 == r2 && g1 == g2 && b1 == b2
}

// firstInk is the foreground of the first visible cell of s.
func firstInk(s string) color.Color {
	w := lipgloss.Width(s)
	buf := uv.NewScreenBuffer(max(1, w), 1)
	uv.NewStyledString(s).Draw(buf, buf.Bounds())
	for x := range w {
		if c := buf.CellAt(x, 0); c != nil && strings.TrimSpace(c.Content) != "" {
			return c.Style.Fg
		}
	}
	return nil
}

func downsample(s string, p colorprofile.Profile) string {
	if p == colorprofile.TrueColor {
		return s
	}
	var b bytes.Buffer
	w := &colorprofile.Writer{Forward: &b, Profile: p}
	_, _ = w.WriteString(s)
	return b.String()
}

// primaryCells renders s at profile p, reads it back as the terminal holds it, and
// returns the cells whose ink or fill is primary as the terminal shows it.
func primaryCells(th *theme.Theme, s string, p colorprofile.Profile) []string {
	w, h := lipgloss.Width(s), lipgloss.Height(s)
	buf := uv.NewScreenBuffer(w, h)
	uv.NewStyledString(downsample(s, p)).Draw(buf, buf.Bounds())
	prim := p.Convert(th.P.Primary)
	if prim == nil {
		return nil // no colour at all: nothing can be amber
	}
	var hits []string
	for y := range h {
		for x := range w {
			c := buf.CellAt(x, y)
			if c == nil {
				continue
			}
			if (c.Style.Fg != nil && sameColor(c.Style.Fg, prim)) || (c.Style.Bg != nil && sameColor(c.Style.Bg, prim)) {
				hits = append(hits, fmt.Sprintf("%q@%d,%d", c.Content, x, y))
			}
		}
	}
	return hits
}

// Huh's spinner turns the brand's frames in the theme's working ink (S.Working: the
// focus ink, so fg under focus "fg", or the role expressions.spinner names), never
// Huh's #F780E2 Dot or its "Loading...". Every kit theme and quire (focus fg), each as
// written and with focus forced to fg: a project whose one theme sets focus fg failed
// this test out of the box when it compared against primary.
func TestSpinnerIsThemed(t *testing.T) {
	files := append(testThemes(t), quirePath(t))
	for _, f := range files {
		tf, err := theme.Load(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, focus := range []string{tf.Expressions.Focus, "fg"} {
			tf.Expressions.Focus = focus
			th := theme.Resolve(tf, true, false)
			name := filepath.Base(f) + " focus=" + focus
			hs := Spinner(th, "Connecting to staging")
			v := hs.View()
			if !strings.HasPrefix(ansi.Strip(v), th.G.Spinner[0]+" Connecting") {
				t.Errorf("%s: huh spinner renders %q", name, ansi.Strip(v))
			}
			if got, want := firstInk(v), firstInk(th.S.Working.Render(th.G.Spinner[0])); !sameColor(got, want) {
				t.Errorf("%s: huh spinner frame is %v, want the working ink %v", name, got, want)
			}
			if strings.Contains(v, "247;128;226") || strings.Contains(ansi.Strip(v), "Loading") {
				t.Errorf("%s: huh spinner kept a default: %q", name, v)
			}
		}
	}
}

// Huh's blurred field bar is BlurBar's ink (fgSubtle), the same bar a blurred kit list
// draws.
func TestBlurredBarMatchesBlurBar(t *testing.T) {
	files := append(testThemes(t), quirePath(t))
	for _, f := range files {
		tf, err := theme.Load(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, dark := range []bool{true, false} {
			th := theme.Resolve(tf, dark, false)
			ink := th.S.BlurBar.GetForeground()
			if huh := Styles(th).Blurred.Base.GetBorderLeftForeground(); th.G.BlurBar != "" && !sameColor(huh, ink) {
				t.Errorf("%s dark=%v: Huh's blurred field bar draws %v, want BlurBar's %v", filepath.Base(f), dark, huh, ink)
			}
		}
	}
}

// Under expressions.focus "fg" Huh's selector, focus border and prompt draw in neutrals
// and weight: primary appears only where the theme or the app asks for it (the kit's own
// chrome is held to the same in ui's TestFocusFgKeepsPrimaryOffChrome).
func TestFocusFgKeepsPrimaryOffHuh(t *testing.T) {
	base, err := theme.Load(quirePath(t))
	if err != nil {
		t.Fatal(err)
	}
	modes := []struct {
		name        string
		dark, ascii bool
		p           colorprofile.Profile
	}{
		{"dark", true, false, colorprofile.TrueColor}, {"light", false, false, colorprofile.TrueColor},
		{"ascii", true, true, colorprofile.TrueColor}, {"ansi", true, false, colorprofile.ANSI},
		{"ansi256", true, false, colorprofile.ANSI256},
	}
	for _, button := range []string{"underline", "bracket", "pill", "inverse"} {
		for _, sel := range []string{"bar", "caret", "fill", "inverse"} {
			f := base
			f.Expressions.Button, f.Expressions.Selection = button, sel
			for _, m := range modes {
				th := theme.ResolveProfile(f, m.dark, m.ascii, m.p)
				huh := Styles(th).Focused
				chrome := map[string]string{
					"huh selector":     huh.SelectSelector.Render() + huh.SelectedPrefix.Render() + huh.NextIndicator.Render(),
					"huh focus border": huh.Base.Render("field"),
					"huh prompt":       huh.TextInput.Prompt.Render("›") + huh.TextInput.Cursor.Render(" "),
				}
				for what, s := range chrome {
					if hits := primaryCells(th, s, m.p); len(hits) > 0 {
						t.Errorf("button=%s selection=%s %s: %s draws primary at %v", button, sel, m.name, what, hits)
					}
				}
			}
		}
	}
}
