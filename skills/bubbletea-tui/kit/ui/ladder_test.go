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
			if rows := Ladder(th, 0); len(rows) != len(Rungs(th))+1 {
				t.Errorf("%s: Ladder draws %d rows, want %d rungs and the spinner", filepath.Base(f), len(rows), len(Rungs(th))+1)
			}
			// ok is the done rung: ● would otherwise mean now (a step) and healthy (a row)
			if got := firstInk(Dot(th, "ok")); !sameColor(got, th.P.Success) {
				t.Errorf("%s dark=%v: Dot ok draws %v, want success", filepath.Base(f), dark, got)
			}
		}
		// under ASCII the rungs' glyphs stay distinct in shape, or two states read as one
		g := theme.Resolve(tf, true, true)
		seen := map[string]string{}
		for _, r := range Rungs(g) {
			glyph := strings.Fields(ansi.Strip(r.Mark(g)))[0]
			if o, dup := seen[glyph]; dup {
				t.Errorf("%s ascii: %s and %s are both %q", filepath.Base(f), o, r.Slot, glyph)
			}
			seen[glyph] = r.Slot
		}
		if ok, cur := ansi.Strip(Dot(g, "ok")), ansi.Strip(Mark(g, "current")); ok == cur {
			t.Errorf("%s ascii: Dot ok and the current rung are both %q", filepath.Base(f), ok)
		}
	}
}

// A theme may declare the product's own ladder (File.Ladder): rungs named for what
// they mean to the product, each a status slot or a literal, in any colour role. The
// ladder state draws them in those inks, so a sheet can judge the product's states,
// and ui.Mark draws a rung by its meaning. quire's: ● new in amber, ◐ reading in the
// working ink, ○ read in fgMuted, ✔ reviewed in fgSubtle. The kit's own slots keep
// their inks beside it, and a theme without a ladder draws the kit's default rungs.
func TestThemeLadder(t *testing.T) {
	f := quire(t)
	if len(f.Ladder) != 4 {
		t.Fatalf("quire declares %d rungs, want 4", len(f.Ladder))
	}
	for _, m := range []struct {
		name        string
		dark, ascii bool
		p           colorprofile.Profile
	}{{"dark", true, false, colorprofile.TrueColor}, {"light", false, false, colorprofile.TrueColor}, {"ascii", true, true, colorprofile.TrueColor}, {"ansi", true, false, colorprofile.ANSI}} {
		th := theme.ResolveProfile(f, m.dark, m.ascii, m.p)
		want := map[string]struct {
			ink   color.Color
			glyph string
		}{
			"new": {th.P.Primary, th.G.Current}, "reading": {th.S.Working.GetForeground(), th.G.Partial},
			"read": {th.P.FgMuted, th.G.Todo}, "reviewed": {th.P.FgSubtle, th.G.Done},
		}
		rows := Ladder(th, 0)
		if len(rows) != 5 {
			t.Fatalf("%s: the ladder draws %d rows, want quire's 4 rungs and the spinner", m.name, len(rows))
		}
		for i, r := range Rungs(th) {
			w := want[r.Meaning]
			if got := firstInk(rows[i]); !sameColor(got, w.ink) {
				t.Errorf("%s: ladder row %q draws %v, want %s %v", m.name, r.Meaning, got, r.Role, w.ink)
			}
			if got := strings.Fields(ansi.Strip(rows[i]))[0]; got != w.glyph {
				t.Errorf("%s: ladder row %q is %q, want the %s slot %q", m.name, r.Meaning, got, r.Slot, w.glyph)
			}
			mark := Mark(th, r.Meaning)
			if got := ansi.Strip(mark); got != w.glyph || !sameColor(firstInk(mark), w.ink) {
				t.Errorf("%s: Mark(%q) = %q in %v, want %q in %v", m.name, r.Meaning, got, firstInk(mark), w.glyph, w.ink)
			}
		}
		// the kit's slots are untouched: current stays the focus ink, done success
		if got := firstInk(Mark(th, "current")); !sameColor(got, th.P.Fg) {
			t.Errorf("%s: the current slot draws %v under a ladder, want fg (focus fg)", m.name, got)
		}
		if got := firstInk(Mark(th, "done")); !sameColor(got, th.P.Success) {
			t.Errorf("%s: the done slot draws %v under a ladder, want success", m.name, got)
		}
	}
	// a literal glyph takes its twin under ASCII; without a ladder, the default rungs
	f.Ladder = []theme.LadderRung{{Glyph: "◔", Meaning: "queued", Role: "info"}}
	f.Glyphs.ASCII = map[string]string{"◔": "%"}
	if got := ansi.Strip(Mark(theme.Resolve(f, true, true), "queued")); got != "%" {
		t.Errorf("a literal rung draws %q under ASCII, want its twin %%", got)
	}
	if got := ansi.Strip(Mark(theme.Resolve(f, true, false), "queued")); got != "◔" {
		t.Errorf("a literal rung draws %q, want ◔", got)
	}
	f.Ladder = nil
	if rungs := Rungs(theme.Resolve(f, true, false)); len(rungs) != len(DefaultRungs) || rungs[0].Slot != "todo" {
		t.Errorf("a theme without a ladder draws %v, want the default rungs", rungs)
	}
}
