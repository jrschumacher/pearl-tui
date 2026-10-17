package theme

import (
	"image/color"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"
)

// amberIsAState is quire with the amber and the error of a brand whose amber means "new":
// at 16 colours colorprofile maps the dark pair both to ANSI 9 and the light pair both
// to ANSI 1, so "new" and "error" are one colour, and no amber both stays amber at 256
// colours and lands on yellow at 16.
func amberIsAState(t *testing.T, ansi16 map[string]map[string]int) File {
	t.Helper()
	f, err := Load(filepath.Join("testdata", "quire.json"))
	if err != nil {
		t.Fatal(err)
	}
	d, l := f.Roles["dark"], f.Roles["light"]
	d.Primary, d.Error = "#D99A3B", "#EF8482"
	l.Primary, l.Error = "#AF7400", "#A13E40"
	f.Roles = map[string]Roles{"dark": d, "light": l}
	f.ANSI16 = ansi16
	return f
}

func index16(t *testing.T, th *Theme, role string) int {
	t.Helper()
	c, ok := colorprofile.ANSI.Convert(paletteRole(th.P, role)).(ansi.BasicColor)
	if !ok {
		t.Fatalf("%s does not convert to one of the 16 colours", role)
	}
	return int(c)
}

func TestANSI16Overrides(t *testing.T) {
	f := amberIsAState(t, nil)
	var got []string
	for _, c := range ANSI16Clashes(f) {
		got = append(got, c.String())
	}
	for _, want := range []string{"dark: primary and error both ANSI 9", "light: primary and error both ANSI 1"} {
		if !strings.Contains(strings.Join(got, "\n"), want) {
			t.Errorf("ANSI16Clashes = %q, want %q", got, want)
		}
	}

	f = amberIsAState(t, map[string]map[string]int{"dark": {"primary": 11}, "light": {"primary": 3}})
	for _, c := range ANSI16Clashes(f) {
		if c.A == "primary" || c.B == "primary" {
			t.Errorf("after the override: %s", c)
		}
	}
	for _, m := range []struct {
		dark          bool
		primary, errs int
	}{{true, 11, 9}, {false, 3, 1}} {
		th := ResolveProfile(f, m.dark, false, colorprofile.ANSI)
		if p, e := index16(t, th, "primary"), index16(t, th, "error"); p != m.primary || e != m.errs {
			t.Errorf("dark=%v at 16 colours: primary %d error %d, want %d and %d", m.dark, p, e, m.primary, m.errs)
		}
		if th.Distinct(th.P.Primary, th.P.Error) == false {
			t.Errorf("dark=%v: primary and error still draw as one colour", m.dark)
		}
		// the override is for 16 colours only: truecolour and 256 keep the brand's amber
		for _, p := range []colorprofile.Profile{colorprofile.TrueColor, colorprofile.ANSI256} {
			if a, b := ResolveProfile(f, m.dark, false, p).P.Primary, Resolve(amberIsAState(t, nil), m.dark, false).P.Primary; !sameRGB(a, b) {
				t.Errorf("dark=%v %v: primary %v, want the theme's own %v", m.dark, p, a, b)
			}
		}
	}
	// a role derived from the overridden one follows it (secondary defaults to primary)
	g := File{Roles: map[string]Roles{"dark": {Primary: "#D99A3B", Error: "#EF8482"}}, ANSI16: map[string]map[string]int{"dark": {"primary": 11}}}
	if th := ResolveProfile(g, true, false, colorprofile.ANSI); index16(t, th, "secondary") != 11 {
		t.Errorf("secondary, derived from primary, is ANSI %d at 16 colours, want 11", index16(t, th, "secondary"))
	}
	// any role can be pinned, an on-colour too
	g.ANSI16["dark"]["onPrimary"] = 0
	if th := ResolveProfile(g, true, false, colorprofile.ANSI); index16(t, th, "onPrimary") != 0 {
		t.Errorf("a pinned onPrimary is ANSI %d, want 0", index16(t, th, "onPrimary"))
	}
}

// ansi16 holds colours, so Load rejects one that would draw wrong, as it rejects a bad hex.
func TestLoadRejectsBadANSI16(t *testing.T) {
	for _, c := range []struct{ json, want string }{
		{`{"ansi16": {"dark": {"primry": 11}}}`, `ansi16.dark.primry: not a role`},
		{`{"ansi16": {"dark": {"primary": 16}}}`, `ansi16.dark.primary: 16 is not one of the 16 colours (0-15)`},
		{`{"ansi16": {"dim": {"primary": 3}}}`, `ansi16.dim: not a mode (dark, light)`},
	} {
		path := writeTheme(t, c.json)
		if _, err := Load(path); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("Load(%s) = %v, want %q", c.json, err, c.want)
		}
	}
}

// What palette check fails at 16 colours, every kit theme and test theme passes: error
// on red (1 or 9), and under focus "fg", where the hue is a state, primary neither on
// red nor on another state role's colour.
func TestThemesReadRightAt16Colours(t *testing.T) {
	var files []string
	for _, dir := range []string{"../themes", "../../themes"} {
		if files, _ = filepath.Glob(filepath.Join(dir, "*.json")); len(files) > 0 {
			break
		}
	}
	test, _ := filepath.Glob(filepath.Join("testdata", "*.json"))
	for _, path := range append(files, test...) {
		f, err := Load(path)
		if err != nil {
			t.Fatal(err)
		}
		fg := f.Expressions.Focus == "fg"
		for _, r := range ANSI16Red(f) {
			if r.Role == "error" || (fg && r.Role == "primary") {
				t.Errorf("%s: %s", filepath.Base(path), r)
			}
		}
		for _, c := range ANSI16Clashes(f) {
			if fg && (c.A == "primary" || c.B == "primary") {
				t.Errorf("%s: %s", filepath.Base(path), c)
			}
		}
	}
}

// charm's, halyard's and quire's light error downsampled to ANSI 8 and their warning to
// 1: at 16 colours the warning was red and the error grey, the two reversed. quire
// without its pins shows it, and its amber on red as well.
func TestANSI16Red(t *testing.T) {
	f, err := Load(filepath.Join("testdata", "quire.json"))
	if err != nil {
		t.Fatal(err)
	}
	f.ANSI16 = nil
	var got []string
	for _, r := range ANSI16Red(f) {
		got = append(got, r.String())
	}
	for _, want := range []string{"dark: primary draws as ANSI 9, error's red", "light: error draws as ANSI 8, not red (1 or 9)", "light: warning draws as ANSI 1, error's red"} {
		if !slices.Contains(got, want) {
			t.Errorf("ANSI16Red(quire without pins) = %q, want %q", got, want)
		}
	}
}

// The border role is "barely lighter than bg": at 16 colours it downsamples onto the
// page's own colour (ANSI 0 on dark, 15 on light) and every rule, section rule and table
// rule vanishes. ResolveProfile moves a rule or border colour that isn't Distinct from
// bg to the nearest one that is, at 16 and 256 colours; truecolour keeps the theme's.
func TestRulesStayVisibleInReducedProfiles(t *testing.T) {
	var files []string
	for _, dir := range []string{"../themes", "../../themes"} {
		if files, _ = filepath.Glob(filepath.Join(dir, "*.json")); len(files) > 0 {
			break
		}
	}
	test, _ := filepath.Glob(filepath.Join("testdata", "*.json"))
	for _, path := range append(files, test...) {
		f, err := Load(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, dark := range []bool{true, false} {
			for _, p := range []colorprofile.Profile{colorprofile.ANSI, colorprofile.ANSI256} {
				th := ResolveProfile(f, dark, false, p)
				for what, c := range map[string]color.Color{"border": th.P.Border, "borderFocus": th.P.BorderFocus, "S.Rule": th.S.Rule.GetForeground()} {
					if !th.Distinct(c, th.P.Bg) {
						t.Errorf("%s dark=%v %v: %s draws in the page's colour", filepath.Base(path), dark, p, what)
					}
				}
			}
			mode := "dark"
			if !dark {
				mode = "light"
			}
			if tc, base := Resolve(f, dark, false).P, derive(f.Roles[mode], dark, nil); !sameRGB(tc.Border, base.Border) {
				t.Errorf("%s dark=%v: truecolour border %v, want the theme's %v", filepath.Base(path), dark, tc.Border, base.Border)
			}
		}
	}
}
