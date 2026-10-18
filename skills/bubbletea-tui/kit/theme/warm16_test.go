package theme

import (
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
)

// A primary pinned to a grey at 16 colours has no hue there (a warm brand on a light
// page, #51: yellow 3 is 1.35:1 on white, magenta is the pink an orange is not). The
// kit's emphasis then falls back as under focus "fg": the focus, prompt and focus bar
// are fg and bold, a fill selection is fg reversed, and brand text is fg and bold, so
// it still stands apart from body text by weight. Truecolour and 256 colours keep the
// hue and the theme's own focus.
func TestGreyPrimaryPinDropsTheHue(t *testing.T) {
	f := File{
		Roles:       map[string]Roles{"light": {Bg: "#FAF9F7", Fg: "#141516", Primary: "#B33A08", FgSelected: "#FFFFFF"}},
		Expressions: Expressions{Selection: "fill"},
		ANSI16:      map[string]map[string]int{"light": {"primary": 0}},
	}
	th := ResolveProfile(f, false, false, colorprofile.ANSI)
	if th.X.Focus != "fg" {
		t.Errorf("light at 16 colours: focus %q, want fg (primary has no hue there)", th.X.Focus)
	}
	for name, st := range map[string]lipgloss.Style{"Focus": th.S.Focus, "Prompt": th.S.Prompt, "FocusBar": th.S.FocusBar, "Primary": th.S.Primary} {
		if !st.GetBold() {
			t.Errorf("S.%s is not bold: with no hue, weight is the brand's cue", name)
		}
		if n := Slot16(st.GetForeground()); n != 0 {
			t.Errorf("S.%s draws as ANSI %d, want fg's 0", name, n)
		}
	}
	if fg := Slot16(th.S.SelRow.GetForeground()); fg != 0 || !th.S.SelRow.GetReverse() {
		t.Errorf("fill selection draws ANSI %d (reverse %v), want fg reversed", fg, th.S.SelRow.GetReverse())
	}
	for _, p := range []colorprofile.Profile{colorprofile.TrueColor, colorprofile.ANSI256} {
		if th := ResolveProfile(f, false, false, p); th.X.Focus != "primary" || th.S.Primary.GetBold() {
			t.Errorf("%v: focus %q, bold brand %v: the hue returns past 16 colours", p, th.X.Focus, th.S.Primary.GetBold())
		}
	}
}
