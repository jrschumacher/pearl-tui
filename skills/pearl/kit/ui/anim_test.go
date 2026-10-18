package ui

import (
	"image/color"
	"path/filepath"
	"slices"
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

// Busy decided by the resolved frames' width, so a theme whose spinner is wider than
// one cell (lumen's "···") drew a still ◐ in unicode and a turning / under ASCII, where
// the kit's line frames are one cell. BusyAnimates judges the theme's own frames, so a
// theme animates in both profiles or holds the partial mark in both, and the showcase
// clock (TestOneClock) ticks only where Busy changes between frames.
func TestBusyAgreesAcrossProfiles(t *testing.T) {
	files := append(testThemes(t), filepath.Join("..", "theme", "testdata", "quire.json"))
	moves := func(th *theme.Theme) bool {
		seen := map[string]bool{}
		for n := range 8 {
			seen[ansi.Strip(Busy(th, n))] = true
		}
		return len(seen) > 1
	}
	wide := 0
	for _, f := range files {
		tf, err := theme.Load(f)
		if err != nil {
			t.Fatal(err)
		}
		uni, asc := theme.Resolve(tf, true, false), theme.Resolve(tf, true, true)
		if BusyAnimates(uni) != BusyAnimates(asc) {
			t.Errorf("%s: BusyAnimates %v in unicode, %v under ASCII", filepath.Base(f), BusyAnimates(uni), BusyAnimates(asc))
		}
		for _, th := range []*theme.Theme{uni, asc} {
			if got := moves(th); got != BusyAnimates(th) {
				t.Errorf("%s ascii=%v: Busy changes between frames: %v, BusyAnimates says %v", filepath.Base(f), th.ASCII, got, BusyAnimates(th))
			}
			if !BusyAnimates(th) {
				wide++
				if got := ansi.Strip(Busy(th, 3)); got != th.G.Partial {
					t.Errorf("%s ascii=%v: a still Busy is %q, want the partial mark %q", filepath.Base(f), th.ASCII, got, th.G.Partial)
				}
			}
		}
	}
	// the kit's themes include a multi-cell spinner (lumen); a copied project may not
	if wide == 0 {
		t.Log("no theme here has a spinner wider than one cell")
	}
	f := theme.Default()
	f.Glyphs.Spinner = []string{"·  ", " · ", "  ·"}
	for _, ascii := range []bool{false, true} {
		if th := theme.Resolve(f, true, ascii); BusyAnimates(th) || moves(th) {
			t.Errorf("ascii=%v: a three-cell spinner animates Busy", ascii)
		}
	}
	// one frame is no motion in unicode, so it is none under ASCII either, where the
	// kit's line spinner would otherwise turn
	f.Glyphs.Spinner = []string{"◍"}
	f.Glyphs.ASCII = map[string]string{"◍": "@"}
	for _, ascii := range []bool{false, true} {
		if th := theme.Resolve(f, true, ascii); BusyAnimates(th) || moves(th) {
			t.Errorf("ascii=%v: a one-frame spinner animates Busy", ascii)
		}
	}
}

// A theme whose ASCII spinner falls back to the one-cell pulse . : ' : draws its blurred
// selection's twin, :, as one of Busy's frames, and a blurred busy row read ": : name",
// the gutter mark twice. Busy leaves the blurred-selection mark out of the one-cell
// frames it turns through; the theme's spinner, which a multi-cell
// Anim and the labelled Spinner draw away from the gutter, keeps it.
func TestBusyIsNeverTheBlurMark(t *testing.T) {
	f := theme.Default()
	f.Glyphs.FocusBar, f.Glyphs.Todo, f.Glyphs.Done, f.Glyphs.Caret = "|", "/", "-", "_"
	th := theme.Resolve(f, true, true)
	if !slices.Contains(th.G.Spinner, th.BlurBar()) {
		t.Skipf("the fallback pulse %q no longer draws the blur twin %q", th.G.Spinner, th.BlurBar())
	}
	seen := map[string]bool{}
	for n := range 2 * len(th.G.Spinner) {
		g := ansi.Strip(Busy(th, n))
		seen[g] = true
		if g == th.BlurBar() {
			t.Errorf("Busy frame %d is %q, the blurred selection's mark", n, g)
		}
	}
	// without : the pulse may be one frame (' here, the . a mark's): then Busy holds the
	// partial mark, still, and says so, as for any one-frame spinner
	if moves := len(seen) > 1; moves != BusyAnimates(th) {
		t.Errorf("Busy turns through %v, and BusyAnimates says %v", seen, BusyAnimates(th))
	}
	if got := ansi.Strip(Spinner(th, 1)); got != th.G.Spinner[1] {
		t.Errorf("Spinner frame 1 is %q, want the theme's %q: only Busy leaves the mark out", got, th.G.Spinner[1])
	}
}
