package ui

import (
	"bytes"
	"fmt"
	"image/color"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"

	"tuikit/theme"
)

// borderless is a dialog: none theme whose panel step is one ANSI colour away from
// nothing: moss's dark bg and bgPanel both downsample to ANSI 0.
func borderless() theme.File {
	f := theme.Default()
	r := f.Roles["dark"]
	r.Bg, r.BgPanel = "#1A1C19", "#1A2720"
	f.Roles = map[string]theme.Roles{"dark": r, "light": f.Roles["light"]}
	f.Expressions.Dialog = "none"
	return f
}

// downsample writes s through colorprofile as Bubble Tea does for a terminal at p.
func downsample(s string, p colorprofile.Profile) string {
	if p == colorprofile.TrueColor {
		return s
	}
	var b bytes.Buffer
	w := &colorprofile.Writer{Forward: &b, Profile: p}
	_, _ = w.WriteString(s)
	return b.String()
}

// dialogStandsOut overlays a dialog on a page of body text, sends the frame through
// the profile and reads it back as the terminal holds it. The dialog stands out when
// its top and bottom rows carry an edge (a glyph at both ends) or every cell inside
// it has a background the page doesn't.
func dialogStandsOut(t *testing.T, th *theme.Theme, p colorprofile.Profile) (bool, string) {
	t.Helper()
	const W, H = 70, 20
	c := NewCanvas(th, W, H)
	page := strings.Repeat(th.S.Soft.Render(strings.Repeat("lorem ipsum ", 6))+"\n", H)
	c.Draw(page, c.Bounds())
	r := c.Overlay(Dialog(th, 44, "Roll back checkout-api?", []string{"It goes back to v2.3.0."}, "", ""), nil)
	c.PaintBackground()
	out := downsample(c.String(), p)
	read := uv.NewScreenBuffer(W, H)
	uv.NewStyledString(out).Draw(read, read.Bounds())

	// a rules dialog keeps a cell of page around its panel: judge the panel
	r = Inset(r, th.S.Dialog.GetMarginLeft(), th.S.Dialog.GetMarginTop())
	// one cell in: a rule edge leaves its corners blank, a gap to the page's text
	edge := func(y int) bool {
		l, rt := read.CellAt(r.Min.X+1, y), read.CellAt(r.Max.X-2, y)
		return l != nil && rt != nil && strings.TrimSpace(l.Content) != "" && strings.TrimSpace(rt.Content) != ""
	}
	if edge(r.Min.Y) && edge(r.Max.Y-1) {
		return true, "edge"
	}
	pageBg := read.CellAt(0, 0).Style.Bg
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if bg := read.CellAt(x, y).Style.Bg; bg == nil || sameColor(bg, pageBg) {
				return false, fmt.Sprintf("cell %d,%d has the page's background %v and the dialog has no edge", x, y, bg)
			}
		}
	}
	return true, "background"
}

func sameColor(a, b color.Color) bool {
	if a == nil || b == nil {
		return a == b
	}
	r1, g1, b1, _ := a.RGBA()
	r2, g2, b2, _ := b.RGBA()
	return r1 == r2 && g1 == g2 && b1 == b2
}

var profiles = []struct {
	name string
	p    colorprofile.Profile
}{{"truecolor", colorprofile.TrueColor}, {"ansi256", colorprofile.ANSI256}, {"ansi", colorprofile.ANSI}, {"NO_COLOR", colorprofile.Ascii}}

// A dialog: none panel is one background step up with no border. Under 16 colours
// (moss dark: bg and bgPanel both ANSI 0) and NO_COLOR (no backgrounds at all) the
// step is gone, so the dialog must carry an edge there, decided by the profile.
func TestBorderlessDialogStandsOutInEveryProfile(t *testing.T) {
	for _, pr := range profiles {
		th := theme.ResolveProfile(borderless(), true, false, pr.p)
		if ok, why := dialogStandsOut(t, th, pr.p); !ok {
			t.Errorf("%s: the dialog blends into the page: %s", pr.name, why)
		}
	}
}

// Every theme's dialog, both modes, ASCII or not, stands out from the page in every
// profile the terminal can report.
func TestEveryDialogStandsOutInEveryProfile(t *testing.T) {
	for _, f := range testThemes(t) {
		tf, err := theme.Load(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, mode := range []struct{ dark, ascii bool }{{true, false}, {false, false}, {true, true}} {
			for _, pr := range profiles {
				th := theme.ResolveProfile(tf, mode.dark, mode.ascii, pr.p)
				if ok, why := dialogStandsOut(t, th, pr.p); !ok {
					t.Errorf("%s dark=%v ascii=%v %s: the dialog blends into the page: %s", f, mode.dark, mode.ascii, pr.name, why)
				}
			}
		}
	}
}

// dialog: rules is a filled panel between two hairline rules, the title in the top
// one, no side borders. Its edge is glyphs, so it holds in every profile and in ASCII.
func TestRulesDialog(t *testing.T) {
	f := theme.Default()
	f.Glyphs.Rule = "╌"
	f.Expressions.Dialog = "rules"
	for _, mode := range []struct {
		name        string
		dark, ascii bool
		p           colorprofile.Profile
		rule        string
	}{
		{"dark", true, false, colorprofile.TrueColor, "╌"},
		{"light", false, false, colorprofile.TrueColor, "╌"},
		{"ascii", true, true, colorprofile.TrueColor, "-"},
		{"ansi", true, false, colorprofile.ANSI, "╌"},
		{"NO_COLOR", true, false, colorprofile.Ascii, "╌"},
	} {
		th := theme.ResolveProfile(f, mode.dark, mode.ascii, mode.p)
		if ok, why := dialogStandsOut(t, th, mode.p); !ok {
			t.Errorf("%s: the rules dialog blends into the page: %s", mode.name, why)
		}
		const w = 46 // a 44-cell panel and a cell of page on each side
		d := Dialog(th, w, "Send reply", []string{"To: ops@example.com"}, "", "")
		rows := strings.Split(ansi.Strip(d), "\n")
		blank := strings.Repeat(" ", w)
		if rows[0] != blank || rows[len(rows)-1] != blank {
			t.Errorf("%s: want a row of page above and below, got %q / %q", mode.name, rows[0], rows[len(rows)-1])
		}
		top, bottom := rows[1], rows[len(rows)-2]
		r := mode.rule
		if want := " " + r + r + " Send reply " + strings.Repeat(r, 30) + " "; top != want {
			t.Errorf("%s: top rule %q, want %q", mode.name, top, want)
		}
		if want := " " + strings.Repeat(r, 44) + " "; bottom != want {
			t.Errorf("%s: bottom rule %q, want %q", mode.name, bottom, want)
		}
		for i, row := range rows {
			if lipgloss.Width(row) != w {
				t.Errorf("%s: row %d is %d cells, want %d", mode.name, i, lipgloss.Width(row), w)
			}
			if i > 1 && i < len(rows)-2 && (row[:2] != "  " || row[len(row)-2:] != "  ") {
				t.Errorf("%s: row %d has a side border or no margin: %q", mode.name, i, row)
			}
		}
		if rows[2] != blank || !strings.HasPrefix(rows[3], "  To: ops") {
			t.Errorf("%s: want a cell of padding inside the rules, got %q / %q", mode.name, rows[2], rows[3])
		}
	}
}

// theme.Load doesn't validate expressions: an unknown dialog value ("ruled") falls back
// to the default rounded border, as before "rules" existed.
func TestUnknownDialogFallsBackToRounded(t *testing.T) {
	f := theme.Default()
	f.Expressions.Dialog = "ruled"
	d := ansi.Strip(Dialog(theme.Resolve(f, true, false), 30, "Title", []string{"body"}, "", ""))
	if !strings.HasPrefix(d, "╭") {
		t.Errorf("unknown dialog value draws %q, want the rounded default", strings.Split(d, "\n")[0])
	}
}
