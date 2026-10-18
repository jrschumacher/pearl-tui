package palette

import (
	"bytes"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"tuikit/theme/bubbletea"
	"tuitools/internal/oklch"
)

// A warm brand at 16 colours (#51), swept over the hue circle: derive's output for a
// brand at OKLCH C 0.15 and L 0.45, 0.55 and 0.62 (a deep rust drew white ink on yellow
// 11 where a mid one drew black), every 15° and every 2° from 16° to 34° (the reds,
// vermilions and corals around ANSI red), on a dark page and a light one, with derive's
// own pins. For every hue and lightness:
//
//   - check exits 0 on derive's output: derive writes no theme its own check fails;
//
//   - the selection ink and onPrimary read at 4.5:1 on primary's slot, as the kit draws
//     them at 16 colours (a white ink on yellow 11 is re-inked black);
//
//   - the warning draws on yellow (3 or 11), never on green, blue or cyan;
//
//   - a warm brand never draws on magenta, nor its steps (secondary, tertiary, accent)
//     on error's red or magenta;
//
//   - the brand shares the warning's hue only as a warm or yellow brand on a dark page,
//     on 11 beside the warning's 3, where the warning's glyph tells them apart;
//
//   - a pinned primary reads as text (3:1) and under its inks (4.5:1) in cellshot's
//     palette (Inks16), and every other pin reads as text at 3:1, but for the warning's
//     yellow 3 on a light page, which is 1.35:1 there: the warning keeps yellow and its
//     glyph (▲ !), as every terminal palette means it;
//
//   - the pins leave nothing for check to report.
//
// Before #51 the oranges and coppers (30–45°) drew their brand on magenta 13/5, and the
// ambers (60–75°) put a light warning on cyan 6, info's.
func TestWarmHueSweep(t *testing.T) {
	var hues []float64
	for h := 0.0; h < 360; h += 15 {
		hues = append(hues, h)
	}
	for h := 16.0; h <= 34; h += 2 {
		hues = append(hues, h)
	}
	text := func(n, bg int) float64 { return oklch.Contrast(basic16Hex(n), basic16Hex(bg)) }
	tables := make([][]string, 3)
	dir := t.TempDir()
	t.Run("L", func(t *testing.T) {
		for k, l := range []float64{0.45, 0.55, 0.62} {
			t.Run(fmt.Sprintf("%.2f", l), func(t *testing.T) {
				t.Parallel()
				for _, h := range hues {
					sweepHue(t, dir, l, h, text, &tables[k])
				}
			})
		}
	})
	t.Logf("brand hue, L | dark P/W/S/I | light P/W/S/I\n%s", strings.Join(slices.Concat(tables...), "\n"))
}

// sweepHue is one row of TestWarmHueSweep.
func sweepHue(t *testing.T, dir string, l, h float64, text func(n, bg int) float64, table *[]string) {
	t.Helper()
	{
		hex := oklch.FromLCH(l, 0.15, h)
		out := filepath.Join(dir, fmt.Sprintf("%03.0f-%.2f.json", h, l))
		var dOut, dErr, cOut, cErr bytes.Buffer
		if code := Main([]string{"derive", hex, "--bg", "#1C1917", "--light-bg", "#FAF7F2", "--out", out}, "", &dOut, &dErr); code != 0 {
			t.Fatalf("derive %s: exit %d\n%s", hex, code, dErr.String())
		}
		if code := Main([]string{"check", out}, "", &cOut, &cErr); code != 0 {
			t.Errorf("hue %.0f° L %.2f (%s): check exits %d on derive's output:\n%s", h, l, hex, code, cErr.String())
		}
		f, err := theme.Load(out)
		if err != nil {
			t.Fatal(err)
		}
		row := fmt.Sprintf("%3.0f° L%.2f %s", h, l, hex)
		for _, mode := range []string{"dark", "light"} {
			_, _, hp := oklch.LCH(f.Roles[mode].Primary)
			p, w := theme.ANSI16Index(f, mode, "primary"), theme.ANSI16Index(f, mode, "warning")
			s, i := theme.ANSI16Index(f, mode, "success"), theme.ANSI16Index(f, mode, "info")
			where := fmt.Sprintf("hue %.0f° L %.2f (%s, %s primary %s at %.0f°)", h, l, hex, mode, f.Roles[mode].Primary, hp)
			th := theme.Resolve16(f, mode != "light")
			for name, ink := range map[string]color.Color{"fgSelected": th.P.FgSelected, "onPrimary": th.P.OnPrimary} {
				if r := theme.Contrast16(ink, th.P.Primary); r < 4.5 {
					t.Errorf("%s: %s draws %.2f:1 on primary's ANSI %d at 16 colours, under 4.5", where, name, r, theme.Slot16(th.P.Primary))
				}
			}
			if w != 3 && w != 11 {
				t.Errorf("%s: the warning draws as ANSI %d, want yellow 3 or 11", where, w)
			}
			switch w % 8 {
			case 2, 4, 6:
				t.Errorf("%s: the warning draws as ANSI %d, a status hue (green, blue or cyan)", where, w)
			}
			if warm(hp) && (p == 5 || p == 13) {
				t.Errorf("%s: the warm brand draws as ANSI %d, magenta", where, p)
			}
			for _, role := range []string{"secondary", "tertiary", "accent"} {
				if n := theme.ANSI16Index(f, mode, role); warm(hp) && (n == 1 || n == 9 || n == 5 || n == 13) {
					t.Errorf("%s: the warm brand's %s draws as ANSI %d, red or magenta", where, role, n)
				}
			}
			shared := ""
			if p%8 == w%8 {
				shared = " shares the warning's hue"
				if !((warm(hp) || hueDiff(hp, yellowHue) <= ownHueMax) && mode == "dark" && p == 11 && w == 3) {
					t.Errorf("%s: the brand (ANSI %d) shares the warning's hue (ANSI %d) outside a warm brand on a dark page", where, p, w)
				}
			}
			bg := theme.ANSI16Index(f, mode, "bg")
			for role, n := range f.ANSI16[mode] {
				if role == "primary" || (role == "warning" && mode == "light" && n == 3) {
					continue // primary is Inks16's below; the light warning's yellow is the rule's
				}
				if r := text(n, bg); r < 3 {
					t.Errorf("%s: %s pinned to ANSI %d reads %.2f:1 on the page's ANSI %d, under 3", where, role, n, r, bg)
				}
			}
			row += fmt.Sprintf(" | %s P%-2d W%-2d S%-2d I%-2d%s", mode[:1], p, w, s, i, shared)
		}
		for _, x := range Inks16(f) {
			t.Errorf("hue %.0f° L %.2f (%s): %s", h, l, hex, x)
		}
		for _, x := range Findings16(f) {
			t.Errorf("hue %.0f° L %.2f (%s): derive's pins leave %s", h, l, hex, x.What)
		}
		*table = append(*table, row)
	}
}

// Coral #D15D51 is OKLCH 28°, 1° under ANSI red's 29°, where the first attempt's warm arc
// began: it still drew on magenta. It is warm: yellow 11 on a dark page, no hue (fg's
// grey) on a light one; a crimson at 20° is a red, and keeps the nearest free slot.
func TestCoralIsWarm(t *testing.T) {
	_, _, h := oklch.LCH("#D15D51")
	if h < 27 || h > 29 {
		t.Fatalf("coral is %.1f°, want 28°: the probe no longer sits under ANSI red", h)
	}
	f := themeOf(Derive("#D15D51", "", "", "#1C1917", "#FAF7F2", 0.012))
	pins := Pins16(f)
	if pins["dark"]["primary"] != 11 || pins["light"]["primary"] != 0 {
		t.Errorf("coral pins %v, want dark primary 11 and light primary 0", pins)
	}
	for mode, want := range map[string]bool{"dark": true, "light": true} {
		if _, ok := WarmPin16(f, mode); ok != want {
			t.Errorf("coral %s: warm %v", mode, ok)
		}
	}
	crimson := theme.File{Roles: map[string]theme.Roles{"dark": {Bg: "#1C1917", Fg: "#E8E2D6", Primary: "#D94060"}}}
	if _, ok := WarmPin16(crimson, "dark"); ok {
		t.Error("crimson #D94060 (13°) is warm: a red's nearest free slot is magenta's pink, which suits it")
	}
}

// palette check judges a primary pin at the 16-colour slot it draws on, in the palette
// the sheets show (cellshot's), not only at truecolour, and fails each case #51's table
// found with every check passing: white on bright yellow (ferrous's selection), a warning
// on green, success's slot (ferrous's light warning), and yellow 3 brand text on white.
// An ink the theme writes as a hex is the kit's to re-ink at 16 colours (white on 11
// draws black there, so check passes it); one the theme pins itself is judged as pinned.
func TestCheckFailsIssue51Table(t *testing.T) {
	dark := `"dark": {"bg": "#1A1B1D", "fg": "#D7D9DC", "primary": "#E25317", "fgSelected": "#FFFFFF", "success": "#6E9E78", "error": "#E0453A", "warning": "#CFA24E", "info": "#8C98A6"}`
	light := `"light": {"bg": "#FAF9F7", "fg": "#141516", "primary": "#B33A08", "fgSelected": "#FFFFFF", "success": "#2A6E36", "error": "#B71C1C", "warning": "#7A5200", "info": "#3C4A58"}`
	for _, c := range []struct {
		name, roles, pins, want string
	}{
		{"white on bright yellow", dark, `{"dark": {"primary": 11, "warning": 3, "fgSelected": 15}}`,
			`dark: fgSelected #FFFFFF draws as ANSI 15 on primary's ANSI 11 at 16 colours (-profile ansi), 1.17:1 in cellshot's palette, under 4.5`},
		{"warning on green", light, `{"light": {"primary": 0, "warning": 2}}`,
			`light: warning #7A5200 draws as ANSI 2 at 16 colours (-profile ansi), success's green, so a warning reads as success there; the warning keeps yellow: pin it with "ansi16": {"light": {"warning": 3}}`},
		{"yellow 3 brand text on white", light, `{"light": {"primary": 3, "warning": 6}}`,
			`light: primary #B33A08 draws as ANSI 3 on bg's ANSI 15 at 16 colours (-profile ansi), 1.35:1 in cellshot's palette, under 3.0`},
	} {
		path := filepath.Join(t.TempDir(), "theme.json")
		if err := os.WriteFile(path, []byte(`{"roles": {`+c.roles+`}, "ansi16": `+c.pins+`}`), 0o644); err != nil {
			t.Fatal(err)
		}
		var out, errb bytes.Buffer
		if code := Main([]string{"check", path}, "", &out, &errb); code != 1 {
			t.Errorf("%s: exit %d, want 1\n%s", c.name, code, errb.String())
		}
		if !strings.Contains(errb.String(), c.want) {
			t.Errorf("%s: stderr %q, want %q", c.name, errb.String(), c.want)
		}
	}
	// the same white written as a hex, unpinned: the kit draws black on the yellow
	f := theme.File{Roles: map[string]theme.Roles{"dark": {Bg: "#1A1B1D", Fg: "#D7D9DC", Primary: "#E25317", FgSelected: "#FFFFFF", OnPrimary: "#FFFFFF"}},
		ANSI16: map[string]map[string]int{"dark": {"primary": 11, "warning": 3}}}
	th := theme.Resolve16(f, true)
	if s, o := theme.Slot16(th.P.FgSelected), theme.Slot16(th.P.OnPrimary); s != 0 || o != 0 {
		t.Errorf("white inks on bright yellow draw as ANSI %d and %d at 16 colours, want black 0", s, o)
	}
	if x := Inks16(f); len(x) != 0 {
		t.Errorf("Inks16 on the re-inked theme: %v", x)
	}
}

// The kit's themes read at 16 colours with nothing for check to say: ferrous's selection
// ink is dark on its bright yellow, and neither orange brand (ferrous, halyard) draws on
// magenta, nor its warning off yellow; quire's amber, a state under focus "fg", has no
// hue on a light page either.
func TestKitThemesReadAt16Colours(t *testing.T) {
	kit, _ := filepath.Glob(kitThemes + "/*.json")
	if len(kit) == 0 {
		t.Fatal("no kit themes")
	}
	for _, path := range append(kit, kitThemes+"/../theme/bubbletea/testdata/quire.json") {
		f, err := theme.Load(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, x := range Inks16(f) {
			t.Errorf("%s: %s", filepath.Base(path), x)
		}
		for _, x := range Findings16(f) {
			t.Errorf("%s: %s; %s", filepath.Base(path), x.What, x.PinJSON())
		}
	}
}

// A brand whose own slot reads under 3:1 on a light page (cyan 6 is 2.80:1 on white,
// green 2 2.47:1 in cellshot's palette) has no hue there: cartridge's, moss's and
// terrarium's light primaries drew on cyan 6 and lumen's on green 2 with no pin, and
// Inks16 judged only pinned primaries (#60). check now fails it unpinned, naming fg's
// grey as the pin, and derive writes that pin; a blue brand on 4 (4.87:1) keeps its hue.
func TestFaintLightBrandHasNoHue(t *testing.T) {
	for _, c := range []struct {
		primary string
		want    int // the light primary pin derive writes; -1 none
	}{{"#0E937E", 0}, {"#1C7A6F", 0}, {"#009298", 0}, {"#2E8B57", 0}, {"#2472C8", -1}} {
		f := themeOf(Derive(c.primary, "", "", "", "", 0.012))
		n, ok := Pins16(f)["light"]["primary"]
		if !ok {
			n = -1
		}
		if n != c.want {
			t.Errorf("%s: derive pins light primary %d, want %d", c.primary, n, c.want)
		}
		roles := `{"light": {"bg": "#FFFFFF", "fg": "#1C1C1C", "primary": "` + f.Roles["light"].Primary + `", "warning": "#8C6800", "error": "#A13E40", "success": "#007835", "info": "#006A9D"}}`
		path := filepath.Join(t.TempDir(), "theme.json")
		if err := os.WriteFile(path, []byte(`{"roles": `+roles+`, "ansi16": {"light": {"warning": 3}}}`), 0o644); err != nil {
			t.Fatal(err)
		}
		var out, errb bytes.Buffer
		code := Main([]string{"check", path}, "", &out, &errb)
		if failed := strings.Contains(errb.String(), "under 3.0: pin it where"); failed != (c.want == 0) || (code == 1) != (c.want == 0) {
			t.Errorf("%s: check exit %d, stderr %q", c.primary, code, errb.String())
		}
	}
}

// A brand step drawn as text (accent's section titles, a secondary button) reads under
// 3:1 on a light page at 16 colours wherever the brand would: lumen greyed its teal
// primary off cyan 6 and kept its teal accent there, 2.80:1 on white, for every section
// title, and charm's accent drew them on yellow 3, 1.35:1. check fails a step under 3:1
// pinned or not, and names the pin: fg's grey, or a bright slot's normal twin when that
// reads and is no state's (pink 13, 2.94:1, to magenta 5, 4.59:1).
func TestFaintStepsHaveNoHue(t *testing.T) {
	for _, c := range []struct {
		accent, secondary string
		pins              map[string]int
	}{
		{"#1C5F58", "#2F6F68", map[string]int{"accent": 0, "secondary": 0}}, // teal steps on cyan 6
		{"#646400", "#D63AD6", map[string]int{"accent": 0, "secondary": 5}}, // yellow 3, pink 13
		{"#2E3192", "#7A1F7A", nil},                                         // blue 4, magenta 5: both read
	} {
		roles := `{"light": {"bg": "#FFFFFF", "fg": "#1C1C1C", "primary": "#2E3192", "secondary": "` + c.secondary + `", "accent": "` + c.accent + `", "warning": "#8C6800", "error": "#A13E40", "success": "#007835", "info": "#006A9D"}}`
		path := filepath.Join(t.TempDir(), "theme.json")
		if err := os.WriteFile(path, []byte(`{"roles": `+roles+`, "ansi16": {"light": {"warning": 3}}}`), 0o644); err != nil {
			t.Fatal(err)
		}
		var out, errb bytes.Buffer
		code := Main([]string{"check", path}, "", &out, &errb)
		if (code == 1) != (c.pins != nil) {
			t.Errorf("accent %s secondary %s: check exit %d\n%s", c.accent, c.secondary, code, errb.String())
		}
		f, err := theme.Load(path)
		if err != nil {
			t.Fatal(err)
		}
		got := Pins16(f)["light"]
		for role, want := range c.pins {
			if got[role] != want {
				t.Errorf("accent %s secondary %s: %s pinned %d, want %d (pins %v)", c.accent, c.secondary, role, got[role], want, got)
			}
		}
		f.ANSI16 = Pins16(f)
		if x := Inks16(f); len(x) != 0 {
			t.Errorf("accent %s secondary %s: the pins leave %v", c.accent, c.secondary, x)
		}
	}
}
