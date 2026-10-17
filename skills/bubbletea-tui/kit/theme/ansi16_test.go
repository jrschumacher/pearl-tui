package theme

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"
)

// welp is quire with welp's own amber and error (issue #12): at 16 colours
// colorprofile maps the dark pair both to ANSI 9 and the light pair both to ANSI 1,
// so "new" and "error" are one colour, and no amber both stays amber at 256 colours
// and lands on yellow at 16.
func welp(t *testing.T, ansi16 map[string]map[string]int) File {
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
	f := welp(t, nil)
	var got []string
	for _, c := range ANSI16Clashes(f) {
		got = append(got, c.String())
	}
	for _, want := range []string{"dark: primary and error both ANSI 9", "light: primary and error both ANSI 1"} {
		if !strings.Contains(strings.Join(got, "\n"), want) {
			t.Errorf("ANSI16Clashes = %q, want %q", got, want)
		}
	}

	f = welp(t, map[string]map[string]int{"dark": {"primary": 11}, "light": {"primary": 3}})
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
			if a, b := ResolveProfile(f, m.dark, false, p).P.Primary, Resolve(welp(t, nil), m.dark, false).P.Primary; !sameRGB(a, b) {
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

// A theme under focus "fg" says its hue is a state, so at 16 colours primary must not
// land on another state role's colour: palette check fails it, and so does this test
// for every kit theme and test theme.
func TestFocusFgThemesKeepPrimaryApartAt16Colours(t *testing.T) {
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
		if f.Expressions.Focus != "fg" {
			continue
		}
		for _, c := range ANSI16Clashes(f) {
			if c.A == "primary" || c.B == "primary" {
				t.Errorf("%s: %s", filepath.Base(path), c)
			}
		}
	}
}
