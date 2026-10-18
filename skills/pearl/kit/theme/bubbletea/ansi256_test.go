package theme

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/colorprofile"
)

// themeFiles are the kit's themes and its test themes.
func themeFiles(t *testing.T) []string {
	t.Helper()
	files, _ := filepath.Glob("../../themes/*.json")
	test, _ := filepath.Glob(filepath.Join("testdata", "*.json"))
	return append(files, test...)
}

// The background ramp's steps are small by design, and at 256 colours halyard's, moss's,
// terrarium's and quire's dark bg and bgPanel both downsampled to xterm 234 (#55): every
// panel, output block and borderless dialog lost its edge. At 256 colours no two steps of
// bg › bgPanel › bgSubtle › bgOverlay share an index, but where the theme writes them as
// one colour (ferrous's bgOverlay is its bg), and the steps the kit moves keep their
// order, each further from the page than the one below.
func TestBackgroundRampApartAt256(t *testing.T) {
	roles := append([]string{"bg"}, rampRoles...)
	for _, path := range themeFiles(t) {
		f, err := Load(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, dark := range []bool{true, false} {
			tc := Resolve(f, dark, false).P
			p := ResolveProfile(f, dark, false, colorprofile.ANSI256).P
			for i, b := range roles[1:] {
				for _, a := range roles[:i+1] {
					if sameColor(paletteRole(tc, a), paletteRole(tc, b)) {
						continue
					}
					if ia, ib := Index256(paletteRole(p, a)), Index256(paletteRole(p, b)); ia == ib {
						t.Errorf("%s dark=%v: %s and %s both draw as xterm %d at 256 colours", filepath.Base(path), dark, a, b, ia)
					}
				}
			}
		}
		if shares := Ramp256Shares(f); len(shares) > 0 {
			for _, s := range shares {
				if s.Kit == s.Index {
					t.Errorf("%s: %v and the kit leaves it there", filepath.Base(path), s)
				}
			}
		}
	}
	// a stepped ramp keeps its order: moss dark's bg and bgPanel (234) apart, panel lighter
	f := File{Roles: map[string]Roles{"dark": {Bg: "#1A1C19", BgPanel: "#1A2720", BgSubtle: "#1E211D", BgOverlay: "#1F221E", Fg: "#E0E0E0"}}}
	p := ResolveProfile(f, true, false, colorprofile.ANSI256).P
	last := Luminance(p.Bg)
	for _, role := range rampRoles {
		l := Luminance(paletteRole(p, role))
		if l <= last {
			t.Errorf("stepped dark ramp: %s (xterm %d) is no lighter than the step below it", role, Index256(paletteRole(p, role)))
		}
		last = l
	}
	if s := Ramp256Shares(f); len(s) != 3 || s[0].String() != "dark: bg and bgPanel both xterm 234" {
		t.Errorf("Ramp256Shares = %v, want bgPanel, bgSubtle and bgOverlay each on a lower step's index, bg and bgPanel first", s)
	}
}

// A theme pins a role to one of the 256 colours with ansi256, as ansi16 does at 16: an
// amber brand that the kit's downsampling draws as 173, a salmon, draws as 172, palette
// derive's nearest, where it is pinned (#64), and only at 256 colours. A role derived
// from it follows it; a pinned ramp step is the theme's, and the kit leaves it.
func TestANSI256Pins(t *testing.T) {
	amber := Roles{Bg: "#17140F", Fg: "#EDE6DA", Primary: "#D4883C"} // 173 by the kit's downsampling, 172 by derive's
	f := File{Roles: map[string]Roles{"dark": amber}}
	if n := Index256(ResolveProfile(f, true, false, colorprofile.ANSI256).P.Primary); n == 172 {
		t.Fatalf("the amber already downsamples to 172: pick a brand the kit draws elsewhere")
	}
	f.ANSI256 = map[string]map[string]int{"dark": {"primary": 172}}
	th := ResolveProfile(f, true, false, colorprofile.ANSI256)
	if n := Index256(th.P.Primary); n != 172 {
		t.Errorf("primary pinned to 172 draws as %d", n)
	}
	if n := Index256(th.P.Secondary); n != 172 {
		t.Errorf("secondary, derived from a pinned primary, draws as %d, want 172", n)
	}
	if !sameColor(Resolve(f, true, false).P.Primary, Hex(amber.Primary)) {
		t.Errorf("the ansi256 pin leaked into truecolour")
	}
	if Slot16(ResolveProfile(f, true, false, colorprofile.ANSI).P.Primary) == 172 {
		t.Errorf("the ansi256 pin leaked into 16 colours")
	}
	// a pinned ramp step stays where the theme put it, on bg's index included
	f = File{Roles: map[string]Roles{"dark": {Bg: "#1A1C19", BgPanel: "#1A2720", Fg: "#E0E0E0"}}, ANSI256: map[string]map[string]int{"dark": {"bgPanel": 234}}}
	if n := Index256(ResolveProfile(f, true, false, colorprofile.ANSI256).P.BgPanel); n != 234 {
		t.Errorf("bgPanel pinned to 234 draws as %d", n)
	}
	if s := Ramp256Shares(f); len(s) == 0 || s[0].Kit != 234 {
		t.Errorf("Ramp256Shares = %v, want bg and bgPanel on 234, left there by the kit", s)
	}
	for _, c := range []struct{ json, want string }{
		{`{"ansi256": {"dark": {"primary": 256}}}`, "ansi256.dark.primary: 256 is not one of the 256 colours"},
		{`{"ansi256": {"dark": {"primay": 172}}}`, "ansi256.dark.primay: not a role"},
		{`{"ansi256": {"dim": {"primary": 172}}}`, "ansi256.dim: not a mode"},
		{`{"ansi256": {"dark": {"primary": 172, "bgPanel": 235}}}`, ""},
	} {
		_, err := Parse([]byte(c.json))
		if c.want == "" && err != nil || c.want != "" && (err == nil || !strings.Contains(err.Error(), c.want)) {
			t.Errorf("Parse(%s) = %v, want %q", c.json, err, c.want)
		}
	}
}
