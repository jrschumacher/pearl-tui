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

// The hero animation appended an animated "..." to its label: "connecting to prod...",
// the "Loading..." the slop test removes. The spinner is the motion; the label is still.
func TestAnimHasNoEllipsis(t *testing.T) {
	for _, working := range []string{"spinner", "scramble"} {
		f := theme.Default()
		f.Expressions.Working = working
		for _, ascii := range []bool{false, true} {
			th := theme.Resolve(f, true, ascii)
			a := NewAnim(th, 8, "connecting to prod", "splash")
			for n := range 40 {
				if s := ansi.Strip(a.Frame(n, "")); strings.Contains(s, "connecting to prod.") || strings.HasSuffix(strings.TrimSpace(s), ".") {
					t.Fatalf("working=%s ascii=%v frame %d: %q trails dots", working, ascii, n, s)
				}
			}
			if s := ansi.Strip(a.Frame(3, "12s")); !strings.HasSuffix(s, "connecting to prod 12s") {
				t.Errorf("working=%s ascii=%v: elapsed not after the label: %q", working, ascii, s)
			}
		}
	}
}

// Under ASCII no spinner frame is the focus bar, the blur bar or a status mark, in any
// kit theme: a working row beside a focus bar read as focused when the frame was "|".
func TestASCIISpinnerFramesAreNoMark(t *testing.T) {
	files := append(testThemes(t), filepath.Join("..", "theme", "testdata", "quire.json"))
	for _, f := range files {
		tf, err := theme.Load(f)
		if err != nil {
			t.Fatal(err)
		}
		g := theme.Resolve(tf, true, true).G
		marks := map[string]string{"focusBar": g.FocusBar, "blurBar": g.BlurBar, "done": g.Done, "current": g.Current, "todo": g.Todo,
			"fail": g.Fail, "warn": g.Warn, "pending": g.Pending, "partial": g.Partial}
		if len(g.Spinner) < 2 {
			t.Errorf("%s: %d ASCII spinner frames, want at least 2", filepath.Base(f), len(g.Spinner))
		}
		for _, fr := range g.Spinner {
			for name, m := range marks {
				if m != "" && strings.TrimSpace(fr) == m {
					t.Errorf("%s: ASCII spinner frame %q is the %s mark", filepath.Base(f), fr, name)
				}
			}
		}
	}
}

// expressions.spinner names the role the working marks draw in: the spinner, Busy, a
// busy Dot and a running tool call. Unset, they follow the focus ink (primary, or fg
// under focus "fg"), as before the key existed.
func TestSpinnerColour(t *testing.T) {
	ink := func(s string) color.Color {
		buf := uv.NewScreenBuffer(lipgloss.Width(s), 1)
		uv.NewStyledString(s).Draw(buf, buf.Bounds())
		for x := range lipgloss.Width(s) {
			if c := buf.CellAt(x, 0); c != nil && strings.TrimSpace(c.Content) != "" {
				return c.Style.Fg
			}
		}
		return nil
	}
	for _, c := range []struct{ spinner, focus, want string }{
		{"", "primary", "primary"}, {"", "fg", "fg"}, {"tertiary", "primary", "tertiary"}, {"muted", "fg", "fgMuted"}, {"info", "primary", "info"},
	} {
		f := theme.Default()
		f.Expressions.Spinner, f.Expressions.Focus = c.spinner, c.focus
		th := theme.ResolveProfile(f, true, false, colorprofile.TrueColor)
		want := map[string]color.Color{"primary": th.P.Primary, "fg": th.P.Fg, "tertiary": th.P.Tertiary, "fgMuted": th.P.FgMuted, "info": th.P.Info}[c.want]
		for what, s := range map[string]string{
			"spinner": Spinner(th, 0), "busy": Busy(th, 0), "dot": Dot(th, "busy"),
			"tool": ToolCall(th, Tool{Name: "Edit", State: ToolRunning}, 40, false)[0],
			"anim": NewAnim(th, 8, "deploying", "x").Frame(0, ""),
		} {
			if got := ink(s); !sameColor(got, want) {
				t.Errorf("spinner=%q focus=%s: %s draws %v, want %s %v", c.spinner, c.focus, what, got, c.want, want)
			}
		}
	}
}
