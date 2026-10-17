package ui

import (
	"image/color"
	"path/filepath"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"

	"tuikit/theme"
)

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

// Every rung of the ladder has one colour role, the same wherever the kit draws it: done
// success, current the focus ink, todo fgMuted (never dimmer than a hint), partial the
// working ink, fail error, warn warning. A Dot's idle and a Steps' todo are that rung.
func TestLadderRoles(t *testing.T) {
	files := append(testThemes(t), filepath.Join("..", "theme", "testdata", "quire.json"))
	for _, f := range files {
		tf, err := theme.Load(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, dark := range []bool{true, false} {
			th := theme.ResolveProfile(tf, dark, false, colorprofile.TrueColor)
			focus := th.P.Primary
			if th.X.Focus == "fg" {
				focus = th.P.Fg
			}
			want := map[string]color.Color{"done": th.P.Success, "current": focus, "todo": th.P.FgMuted, "partial": th.S.Working.GetForeground(),
				"pending": th.P.FgSubtle, "fail": th.P.Error, "warn": th.P.Warning}
			for slot, c := range want {
				if got := firstInk(Mark(th, slot)); !sameColor(got, c) {
					t.Errorf("%s dark=%v: %s draws %v, want %v", filepath.Base(f), dark, slot, got, c)
				}
			}
			if got := firstInk(Dot(th, "idle")); !sameColor(got, th.P.FgMuted) {
				t.Errorf("%s dark=%v: Dot idle draws %v, want fgMuted", filepath.Base(f), dark, got)
			}
			if got := firstInk(Steps{Names: []string{"A", "B"}, Cur: 0}.List(th)[1]); !sameColor(got, th.P.FgMuted) {
				t.Errorf("%s dark=%v: a todo step draws %v, want fgMuted", filepath.Base(f), dark, got)
			}
			if rows := Ladder(th, 0); len(rows) != len(Rungs)+1 {
				t.Errorf("%s: Ladder draws %d rows, want %d rungs and the spinner", filepath.Base(f), len(rows), len(Rungs)+1)
			}
		}
		// under ASCII the rungs' glyphs stay distinct in shape, or two states read as one
		g := theme.Resolve(tf, true, true)
		seen := map[string]string{}
		for _, r := range Rungs {
			glyph := strings.Fields(ansi.Strip(Mark(g, r.Slot)))[0]
			if o, dup := seen[glyph]; dup {
				t.Errorf("%s ascii: %s and %s are both %q", filepath.Base(f), o, r.Slot, glyph)
			}
			seen[glyph] = r.Slot
		}
	}
}
