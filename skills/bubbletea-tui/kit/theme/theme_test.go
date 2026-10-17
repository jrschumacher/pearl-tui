package theme

import (
	"image/color"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"unicode"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"
)

// CheckHex accepts exactly the strings Lip Gloss draws as the colour written: for
// each accepted string Lip Gloss returns that RGB, and for each rejected one it
// returns no colour or a different one.
func TestCheckHexMatchesLipGloss(t *testing.T) {
	want := color.RGBA{0x6B, 0x50, 0xFF, 0xff}
	for _, s := range []string{"#6B50FF", "#6b50ff"} {
		if err := CheckHex(s); err != nil {
			t.Errorf("CheckHex(%q) = %v, want ok", s, err)
		}
		if !sameRGB(lipgloss.Color(s), want) {
			t.Errorf("lipgloss draws %q as %v", s, lipgloss.Color(s))
		}
	}
	if err := CheckHex("#abc"); err != nil {
		t.Errorf("CheckHex(#abc) = %v, want ok", err)
	}
	if !sameRGB(lipgloss.Color("#abc"), color.RGBA{0xaa, 0xbb, 0xcc, 0xff}) {
		t.Errorf("lipgloss draws #abc as %v", lipgloss.Color("#abc"))
	}
	for _, s := range []string{"6B50FF", "123456", "12", "6B5", "#6B50", "#6B50F", "#6B50FFAA",
		"6B50FFAA", " #6B50FF", "#6B50FF ", "##6B50FF", "#GGGGGG", "red", "#", ""} {
		if CheckHex(s) == nil {
			t.Errorf("CheckHex(%q) = ok, want an error", s)
		}
		if s != "" && sameRGB(lipgloss.Color(s), want) {
			t.Errorf("%q is rejected but lipgloss draws it as #6B50FF", s)
		}
	}
}

func TestCheckHexSaysWhatToWrite(t *testing.T) {
	err := CheckHex("6B50FF")
	if err == nil || !strings.Contains(err.Error(), `write "#6B50FF"`) {
		t.Errorf("CheckHex(6B50FF) = %v, want it to suggest \"#6B50FF\"", err)
	}
}

func TestLoadRejectsBadRoleColours(t *testing.T) {
	for _, c := range []struct{ json, want string }{
		{`{"roles": {"dark": {"primary": "6B50FF"}}}`, `roles.dark.primary: "6B50FF" has no #`},
		{`{"roles": {"dark": {"bg": "#201F26"}, "light": {"fgMuted": "#6B50FFAA"}}}`, `roles.light.fgMuted: "#6B50FFAA" is not #RGB or #RRGGBB`},
		{`{"roles": {"dark": {"onPrimary": "12"}}}`, `roles.dark.onPrimary: "12" has no #`},
	} {
		path := filepath.Join(t.TempDir(), "bad.json")
		if err := os.WriteFile(path, []byte(c.json), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := Load(path)
		if err == nil || !strings.Contains(err.Error(), c.want) || !strings.HasPrefix(err.Error(), path+": ") {
			t.Errorf("Load(%s) = %v, want %q", c.json, err, c.want)
		}
	}
}

// Empty roles are derived, not errors.
func TestLoadAcceptsEmptyAndShortRoles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ok.json")
	if err := os.WriteFile(path, []byte(`{"roles": {"dark": {"primary": "#abc", "bg": ""}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err != nil {
		t.Errorf("Load: %v", err)
	}
}

// TestKitThemesLoad loads every theme beside the package: kit/themes in the kit,
// <project>/themes once theme/ is copied to internal/theme.
func TestKitThemesLoad(t *testing.T) {
	var files []string
	for _, dir := range []string{"../themes", "../../themes"} {
		if files, _ = filepath.Glob(filepath.Join(dir, "*.json")); len(files) > 0 {
			break
		}
	}
	if len(files) == 0 {
		t.Fatal("no theme JSON in ../themes or ../../themes")
	}
	for _, f := range files {
		if _, err := Load(f); err != nil {
			t.Error(err)
		}
	}
}

// A key the kit doesn't read is dropped by Load, and the role it meant falls back to the
// derived colour with nothing to say why; CheckKeys (palette check) names it. Every kit
// theme and test theme passes it.
func TestCheckKeys(t *testing.T) {
	for _, c := range []struct{ json, want string }{
		{`{"roles": {"dark": {"primay": "#6B50FF"}}}`, `roles.dark.primay is not a key the kit reads (primary, secondary, tertiary, accent, bg, `},
		{`{"dark": {"bg": "#201F26"}, "light": {"fgMutted": "#777"}}`, `roles.light.fgMutted is not a key the kit reads`},
		{`{"glyphs": {"focusbar": "▌"}}`, `glyphs.focusbar is not a key the kit reads (headerRule, rule, focusBar, `},
		{`{"wordmark": {"rows": [], "style": "bogus"}}`, `wordmark.style "bogus" is not a style wordmark.py writes: half | shade | ascii | tracked`},
		{`{"wordmark": {"rows": [], "styl": "half"}}`, `wordmark.styl is not a key the kit reads (rows, style)`},
		{`{"roles": {"dark": {"primary": "#6B50FF", "white": "#fff"}}, "glyphs": {"focusBar": "▌", "ascii": {"▌": "|"}}, "wordmark": {"rows": ["x"], "style": "shade"}, "contrastAccept": {}, "unknownTop": 1}`, ""},
		{`{"roles": {"dark": {"primary": 12}}}`, ""}, // not a key problem: Load reports the colour
		// the product's own ladder: keys the kit reads, a role it can draw, a glyph it can twin
		{`{"ladder": [{"glyph": "current", "meaning": "new", "colour": "primary"}]}`, `ladder[0].colour is not a key the kit reads (glyph, meaning, role)`},
		{`{"ladder": [{"glyph": "current", "meaning": "new", "role": "amber"}]}`, `ladder[0].role "amber" is not a colour role`},
		{`{"ladder": [{"glyph": "◔", "meaning": "queued", "role": "info"}]}`, `ladder[0].glyph "◔" is neither a status slot`},
		{`{"ladder": [{"glyph": "●", "meaning": "new", "role": "primary"}, {"glyph": "◆", "meaning": "finished", "role": "success"}]}`, `ladder[1].glyph "◆" and ladder[0].glyph "●" both draw as "*" under ASCII`},
		{`{"ladder": [{"glyph": "current", "meaning": "new", "role": "primary"}, {"glyph": "done", "meaning": "new", "role": "success"}]}`, `ladder[1].meaning "new" is also ladder[0]'s`},
		{`{"ladder": [{"glyph": "current", "meaning": "", "role": "primary"}]}`, `ladder[0].meaning is empty`},
		{`{"ladder": {"glyph": "current"}}`, `ladder: not an array of rungs`},
		// a meaning that is a kit slot would recolour the kit's own mark (Dot ok, a step)
		{`{"ladder": [{"glyph": "✔", "meaning": "done", "role": "fgSubtle"}]}`, `ladder[0].meaning "done" is a kit slot`},
		{`{"ladder": [{"glyph": "partial", "meaning": "busy", "role": "info"}]}`, `ladder[0].meaning "busy" is a kit slot`},
		// only the status slots are slot names: a chrome glyph's key is a literal, and no literal is a word
		{`{"ladder": [{"glyph": "focusBar", "meaning": "new", "role": "primary"}]}`, `ladder[0].glyph "focusBar" is neither a status slot`},
		{`{"ladder": [{"glyph": "▌", "meaning": "new", "role": "primary"}, {"glyph": "caret", "meaning": "typing", "role": "info"}]}`, `ladder[1].glyph "caret" is neither a status slot`},
		{`{"ladder": [{"glyph": "current", "meaning": "new", "role": "primary"}, {"glyph": "◔", "meaning": "queued", "role": "working"}, {"glyph": "✔", "meaning": "reviewed", "role": "fgSubtle"}], "glyphs": {"ascii": {"◔": "%"}}}`, ""},
	} {
		err := CheckKeys([]byte(c.json))
		if c.want == "" && err != nil || c.want != "" && (err == nil || !strings.Contains(err.Error(), c.want)) {
			t.Errorf("CheckKeys(%s) = %v, want %q", c.json, err, c.want)
		}
	}
	var files []string
	for _, dir := range []string{"../themes", "../../themes"} {
		if files, _ = filepath.Glob(filepath.Join(dir, "*.json")); len(files) > 0 {
			break
		}
	}
	test, _ := filepath.Glob(filepath.Join("testdata", "*.json"))
	for _, path := range append(files, test...) {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := CheckKeys(b); err != nil {
			t.Errorf("%s: %v", filepath.Base(path), err)
		}
	}
}

func sameRGB(a, b color.Color) bool {
	if a == nil || b == nil {
		return false
	}
	r1, g1, b1, a1 := a.RGBA()
	r2, g2, b2, a2 := b.RGBA()
	return r1 == r2 && g1 == g2 && b1 == b2 && a1 == a2
}

// writeTheme writes a theme JSON to a temp file and returns its path.
func writeTheme(t *testing.T, json string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "theme.json")
	if err := os.WriteFile(path, []byte(json), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// A glyph the kit's ASCII map doesn't know used to become "*" under -ascii, which is
// ●'s twin: a ◔ "reading" state beside a ● "new" one drew as the same mark. A theme
// gives its glyphs twins in glyphs.ascii; CheckGlyphs (palette check, the kit's tests)
// fails one without, and two marks of one meaning that share a twin. Load still loads
// the theme, so no app refuses to start over a glyph, and ASCII draws it as "?", which
// tuilint frames --ascii fails.
func TestCheckGlyphs(t *testing.T) {
	for _, c := range []struct{ json, want, ascii string }{
		{`{"glyphs": {"current": "●", "pending": "◔"}}`, `glyphs.pending "◔" has no ASCII twin`, "?"},
		{`{"glyphs": {"focusBar": "🮇"}}`, `glyphs.focusBar "🮇" has no ASCII twin`, ""},
		{`{"glyphs": {"pending": "◔", "ascii": {"◔": "◐"}}}`, `glyphs.ascii: "◔" → "◐" is not ASCII`, ""},
		{`{"glyphs": {"ascii": {"◔": ""}}}`, `glyphs.ascii: "◔" → "" is not ASCII`, ""},
		// a declared twin that is another state's twin
		{`{"glyphs": {"current": "●", "pending": "◔", "ascii": {"◔": "*"}}}`, `glyphs.current "●" and glyphs.pending "◔" both draw as "*"`, "*"},
		// halyard and lumen before their glyphs.ascii: done and pending, done and current
		{`{"glyphs": {"done": "◆", "current": "▲", "todo": "△"}}`, `glyphs.done "◆" and glyphs.pending "●" both draw as "*"`, ""},
		{`{"glyphs": {"done": "•", "current": "●", "todo": "○"}}`, `glyphs.done "•" and glyphs.current "●" both draw as "*"`, ""},
		{`{"glyphs": {"checkOn": "◼", "checkOff": "◻", "ascii": {"◼": "#", "◻": "#"}}}`, `glyphs.checkOn "◼" and glyphs.checkOff "◻" both draw as "#"`, ""},
		// the caret sits after typed text: as the focus bar, or its twin, "> ro|" reads as focus
		{`{"glyphs": {"focusBar": "▌", "caret": "▌"}}`, `glyphs.caret "▌" draws as "|", the focusBar's`, ""},
		{`{"glyphs": {"focusBar": "┃", "caret": "│"}}`, `glyphs.caret "│" draws as "|", the focusBar's`, ""},
		{`{"glyphs": {"focusBar": "▌", "blurBar": "┆", "caret": "┆", "ascii": {"┆": ":"}}}`, `glyphs.caret "┆" draws as ":", the blurBar's`, ""},
	} {
		path := writeTheme(t, c.json)
		f, err := Load(path)
		if err != nil {
			t.Errorf("Load(%s) = %v: a glyph must never stop an app from starting", c.json, err)
			continue
		}
		if err := CheckGlyphs(f.Glyphs); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("CheckGlyphs(%s) = %v, want %q", c.json, err, c.want)
		}
		if c.ascii != "" {
			if g := Resolve(f, true, true).G; g.Pending != c.ascii {
				t.Errorf("%s: pending draws %q under ASCII, want %q", c.json, g.Pending, c.ascii)
			}
		}
	}
	// warn and fail are always "!" and "x" under ASCII, so they need no twin; the same
	// glyph in two fields is one mark on purpose
	for _, js := range []string{`{"glyphs": {"warn": "◬", "fail": "✘"}}`, `{"glyphs": {"done": "■", "current": "■"}}`} {
		f, err := Load(writeTheme(t, js))
		if err != nil {
			t.Fatal(err)
		}
		if err := CheckGlyphs(f.Glyphs); err != nil {
			t.Errorf("CheckGlyphs(%s) = %v", js, err)
		}
	}
}

// Every theme beside the package (kit/themes, or the project's themes/ after a copy)
// and every test theme under testdata passes CheckGlyphs: what palette check asks.
func TestThemesPassCheckGlyphs(t *testing.T) {
	var files []string
	for _, dir := range []string{"../themes", "../../themes"} {
		if files, _ = filepath.Glob(filepath.Join(dir, "*.json")); len(files) > 0 {
			break
		}
	}
	test, _ := filepath.Glob(filepath.Join("testdata", "*.json"))
	if len(files) == 0 {
		t.Fatal("no theme JSON in ../themes or ../../themes")
	}
	for _, path := range append(files, test...) {
		f, err := Load(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := CheckGlyphs(f.Glyphs); err != nil {
			t.Errorf("%s: %v", path, err)
		}
	}
}

func TestThemeASCIITwins(t *testing.T) {
	f, err := Load(writeTheme(t, `{"glyphs": {"current": "●", "pending": "◔", "todo": "◐", "ascii": {"◔": "~", "●": "@", "◐": "%"}}}`))
	if err != nil {
		t.Fatal(err)
	}
	g := Resolve(f, true, true).G
	if g.Pending != "~" || g.Current != "@" {
		t.Errorf("the theme's twins: pending %q current %q, want ~ and @ (the theme's map wins over the kit's)", g.Pending, g.Current)
	}
	if g.Todo != "%" {
		t.Errorf("◐ draws %q under ASCII, want the theme's %%", g.Todo)
	}
	// a glyph with no twin anywhere (a File built in code, not loaded) is loud, not ●'s twin
	g = ASCII(Glyphs{Pending: "◔", Current: "●"})
	if g.Pending == g.Current || g.Pending != "?" {
		t.Errorf("unmapped ◔ draws %q beside ● as %q, want ?", g.Pending, g.Current)
	}
}

// Text on a fill must stay apart from it at every colour profile: under 16 colours
// ferrous's muted keys and ink's pill labels landed in their fill's slot and vanished.
// Every ready-made style that sets both an ink and a fill is held to t.Distinct.
func TestStylesKeepInkOffTheirFill(t *testing.T) {
	var files []string
	for _, dir := range []string{"../themes", "../../themes"} {
		if files, _ = filepath.Glob(filepath.Join(dir, "*.json")); len(files) > 0 {
			break
		}
	}
	for _, f := range files {
		tf, err := Load(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, dark := range []bool{true, false} {
			for _, p := range []colorprofile.Profile{colorprofile.TrueColor, colorprofile.ANSI256, colorprofile.ANSI} {
				th := ResolveProfile(tf, dark, false, p)
				v, ty := reflect.ValueOf(th.S), reflect.TypeOf(th.S)
				for i := range ty.NumField() {
					st := v.Field(i).Interface().(lipgloss.Style)
					fg, bg := st.GetForeground(), st.GetBackground()
					if _, none := fg.(lipgloss.NoColor); none || fg == nil {
						continue
					}
					if _, none := bg.(lipgloss.NoColor); none || bg == nil {
						continue
					}
					if !th.Distinct(fg, bg) {
						t.Errorf("%s dark=%v %v: S.%s draws its text in its fill's colour", filepath.Base(f), dark, p, ty.Field(i).Name)
					}
				}
			}
		}
	}
}

// A state ladder has a rung between current and done: "◐ reading" between "● new" and
// "✓ reviewed". Glyphs had no slot for it, so a theme's "reading" key was dropped by
// the struct and the rung drew as whatever the app hard-coded. partial is that slot:
// ◐ by default, "~" under ASCII from the kit's map, and one of the status marks
// CheckGlyphs keeps apart.
func TestPartialSlot(t *testing.T) {
	f, err := Load(writeTheme(t, `{"glyphs": {"partial": "◑", "ascii": {"◑": "%"}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if g := Resolve(f, true, false).G; g.Partial != "◑" {
		t.Errorf("glyphs.partial is %q, want the theme's ◑", g.Partial)
	}
	if g := Resolve(f, true, true).G; g.Partial != "%" {
		t.Errorf("glyphs.partial draws %q under ASCII, want the theme's twin %%", g.Partial)
	}
	d := Resolve(File{}, true, false).G
	a := Resolve(File{}, true, true).G
	if d.Partial != "◐" || a.Partial != "~" {
		t.Errorf("default partial %q (ASCII %q), want ◐ (~)", d.Partial, a.Partial)
	}
	if tw, ok := ASCIITwin("◐"); !ok || tw != "~" {
		t.Errorf("the kit's map twins ◐ as %q (%v), want ~", tw, ok)
	}
	// a partial whose twin is another state's is caught like any other status mark
	for _, c := range []struct{ json, want string }{
		{`{"glyphs": {"partial": "◑", "ascii": {"◑": "*"}}}`, `glyphs.current "●" and glyphs.partial "◑" both draw as "*"`},
		{`{"glyphs": {"partial": "◑", "ascii": {"◑": "o"}}}`, `glyphs.todo "○" and glyphs.partial "◑" both draw as "o"`},
	} {
		f, err := Load(writeTheme(t, c.json))
		if err != nil {
			t.Fatal(err)
		}
		if err := CheckGlyphs(f.Glyphs); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("CheckGlyphs(%s) = %v, want %q", c.json, err, c.want)
		}
	}
}

// A theme written before the partial slot existed, with its own "in progress" mark
// twinned "~" by brand.md's convention ({"◔": "~"}), still passes CheckGlyphs: the
// default ◐ takes the next free twin instead of making two states one mark.
func TestDefaultPartialKeepsOutOfTheThemesWay(t *testing.T) {
	f, err := Load(writeTheme(t, `{"glyphs": {"pending": "◔", "ascii": {"◔": "~"}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := CheckGlyphs(f.Glyphs); err != nil {
		t.Errorf("CheckGlyphs = %v: a default must not collide with the theme's own twins", err)
	}
	g := Resolve(f, true, true).G
	if g.Pending != "~" || g.Partial == "~" || g.Partial == "?" {
		t.Errorf("ASCII pending %q partial %q, want ~ and another twin", g.Pending, g.Partial)
	}
	if f.Glyphs.ASCII["◐"] != "" {
		t.Errorf("the default wrote into the theme's own map: %v", f.Glyphs.ASCII)
	}
}

// A working spinner sits in a status column beside the state marks, and beside focus
// bars. Under ASCII its frames were | / - \, and "|" is the focus bar's twin: a row with
// a spinner read as focused. ASCII's frames leave out any frame a focus bar or a state
// mark draws as, and CheckGlyphs fails a theme spinner frame that is one of those marks.
func TestSpinnerNeverDrawsAnotherMark(t *testing.T) {
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
		for _, ascii := range []bool{false, true} {
			g := Resolve(f, true, ascii).G
			marks := map[string]string{"focusBar": g.FocusBar, "blurBar": g.BlurBar, "done": g.Done, "current": g.Current,
				"todo": g.Todo, "pending": g.Pending, "partial": g.Partial, "warn": g.Warn, "fail": g.Fail}
			if len(g.Spinner) < 2 {
				t.Errorf("%s ascii=%v: spinner %q has no motion", filepath.Base(path), ascii, g.Spinner)
			}
			marks["caret"] = g.Caret
			for _, fr := range g.Spinner {
				for name, m := range marks {
					if m != "" && strings.TrimSpace(fr) == m {
						t.Errorf("%s ascii=%v: spinner frame %q is glyphs.%s", filepath.Base(path), ascii, fr, name)
					}
					for _, r := range fr { // a multi-cell frame's cells, each beside a status column or a bar
						if m != "" && !unicode.IsSpace(r) && string(r) == m {
							t.Errorf("%s ascii=%v: spinner frame %q contains glyphs.%s %q", filepath.Base(path), ascii, fr, name, m)
						}
					}
				}
			}
		}
	}
	// a frame that is a mark the theme set is an error; the default partial steps aside
	f, err := Load(writeTheme(t, `{"glyphs": {"partial": "◐", "spinner": ["◐", "◓", "◑", "◒"]}}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := CheckGlyphs(f.Glyphs); err == nil || !strings.Contains(err.Error(), `glyphs.spinner frame "◐" is also glyphs.partial`) {
		t.Errorf("CheckGlyphs(spinner with ◐, partial ◐) = %v", err)
	}
	f, err = Load(writeTheme(t, `{"glyphs": {"spinner": ["◐", "◓", "◑", "◒"]}}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := CheckGlyphs(f.Glyphs); err != nil {
		t.Errorf("CheckGlyphs(spinner with ◐, partial unset) = %v: the theme never wrote a partial", err)
	}
	if g := Resolve(f, true, false).G; slices.Contains(g.Spinner, g.Partial) {
		t.Errorf("the default partial %q is a spinner frame %q", g.Partial, g.Spinner)
	}
	// every rune of a multi-cell frame counts, not only a one-cell frame as a whole: a
	// dots spinner beside a "·" pending (welp's) draws the pending mark three times in a
	// working row, and a frame holding the caret or the focus bar draws a focus
	for _, c := range []struct{ json, want string }{
		{`{"glyphs": {"pending": "·", "spinner": ["···", "•··", "••·", "•••"]}}`, `glyphs.spinner frame "···" contains glyphs.pending "·"`},
		{`{"glyphs": {"spinner": ["▏·", "·▏"]}}`, `glyphs.spinner frame "▏·" contains glyphs.caret "▏"`},
		{`{"glyphs": {"focusBar": "┃", "spinner": ["┃·", "·┃"]}}`, `glyphs.spinner frame "┃·" contains glyphs.focusBar "┃"`},
	} {
		f, err := Load(writeTheme(t, c.json))
		if err != nil {
			t.Fatal(err)
		}
		if err := CheckGlyphs(f.Glyphs); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("CheckGlyphs(%s) = %v, want %q", c.json, err, c.want)
		}
	}
	// the same dots beside a "◌" pending are fine: no cell is a mark
	f, err = Load(writeTheme(t, `{"glyphs": {"pending": "◌", "spinner": ["···", "•··", "••·", "•••"]}}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := CheckGlyphs(f.Glyphs); err != nil {
		t.Errorf("CheckGlyphs(dots spinner, pending ◌) = %v", err)
	}
}

// The default partial is a glyph no status mark or spinner frame of the theme already
// is: a theme whose pending is ◐ got ◐ for partial too, so a running and a pending tool
// call were one mark.
func TestDefaultPartialIsItsOwnGlyph(t *testing.T) {
	for _, js := range []string{
		`{"glyphs": {"pending": "◐"}}`,
		`{"glyphs": {"pending": "◐", "todo": "◑", "ascii": {"◑": "-"}}}`,
		`{"glyphs": {"spinner": ["◐", "◓"]}}`,
	} {
		f, err := Load(writeTheme(t, js))
		if err != nil {
			t.Fatal(err)
		}
		if err := CheckGlyphs(f.Glyphs); err != nil {
			t.Errorf("%s: CheckGlyphs %v", js, err)
		}
		for _, ascii := range []bool{false, true} {
			g := Resolve(f, true, ascii).G
			for name, m := range map[string]string{"done": g.Done, "current": g.Current, "todo": g.Todo, "pending": g.Pending, "warn": g.Warn, "fail": g.Fail} {
				if m == g.Partial {
					t.Errorf("%s ascii=%v: the default partial %q is glyphs.%s", js, ascii, g.Partial, name)
				}
			}
		}
	}
}

// The ASCII spinner turns: at least two distinct frames, none a mark the theme draws,
// even when the marks take the line spinner's frames and the pulse's.
func TestASCIISpinnerTurns(t *testing.T) {
	for _, js := range []string{
		`{}`,
		`{"glyphs": {"focusBar": "|", "blurBar": "/", "todo": "-", "done": "\\", "pending": ".", "partial": ":"}}`,
		`{"glyphs": {"focusBar": "|", "blurBar": "/", "todo": "-", "done": "\\", "pending": ".", "partial": ":", "current": "'"}}`,
	} {
		f, err := Load(writeTheme(t, js))
		if err != nil {
			t.Fatal(err)
		}
		g := Resolve(f, true, true).G
		distinct := map[string]bool{}
		for _, fr := range g.Spinner {
			distinct[fr] = true
			for _, m := range []string{g.FocusBar, g.BlurBar, g.Done, g.Current, g.Todo, g.Pending, g.Partial, g.Warn, g.Fail} {
				if fr == m {
					t.Errorf("%s: ASCII spinner frame %q is a mark", js, fr)
				}
			}
		}
		if len(distinct) < 2 {
			t.Errorf("%s: ASCII spinner %q doesn't turn", js, g.Spinner)
		}
	}
}

// themes/ink.json and Default are one fallback in two forms (the embedded theme the
// showcase draws with no flag, and what a partial theme's gaps resolve to): they
// resolve to the same palette, glyphs and expressions in both modes, at truecolour and
// at 16 colours, where ink's pins apply. A project that copied the kit has its own
// themes/, so the file is looked for and the test passes without it.
func TestInkIsTheDefault(t *testing.T) {
	var ink string
	for _, p := range []string{"../themes/ink.json", "../../themes/ink.json"} {
		if _, err := os.Stat(p); err == nil {
			ink = p
		}
	}
	if ink == "" {
		t.Skip("no themes/ink.json: a copied project carries its own theme")
	}
	f, err := Load(ink)
	if err != nil {
		t.Fatal(err)
	}
	for _, dark := range []bool{true, false} {
		for _, p := range []colorprofile.Profile{colorprofile.TrueColor, colorprofile.ANSI} {
			a, b := ResolveProfile(f, dark, false, p), ResolveProfile(Default(), dark, false, p)
			if !reflect.DeepEqual(a.P, b.P) {
				t.Errorf("dark=%v %v: ink.json and Default() resolve to different palettes", dark, p)
			}
			if !reflect.DeepEqual(a.G, b.G) {
				t.Errorf("dark=%v %v: glyphs differ:\n%+v\n%+v", dark, p, a.G, b.G)
			}
			if !reflect.DeepEqual(a.X, b.X) {
				t.Errorf("dark=%v %v: expressions differ:\n%+v\n%+v", dark, p, a.X, b.X)
			}
		}
	}
}

// Twins never collide across meanings: the caret ▏ drew as "|", the focus bar's twin, so
// an ASCII input ("> ro|") read as a focused block; and SparkRampASCII "_.-=+*#@" held
// "*" and "+", the current and done twins, so a sparkline's peaks read as status marks.
// Every theme, under ASCII: the caret is neither bar, and the ramp holds no twin of a
// mark that means a state at a glance (done, current, warn, fail); the quiet marks' "."
// (pending, a todo) is a low bar in a run, which reads as what it is.
func TestTwinsStayApartAcrossMeanings(t *testing.T) {
	var files []string
	for _, dir := range []string{"../themes", "../../themes"} {
		if files, _ = filepath.Glob(filepath.Join(dir, "*.json")); len(files) > 0 {
			break
		}
	}
	test, _ := filepath.Glob(filepath.Join("testdata", "*.json"))
	themes := []File{Default()}
	for _, path := range append(files, test...) {
		f, err := Load(path)
		if err != nil {
			t.Fatal(err)
		}
		themes = append(themes, f)
	}
	for _, f := range themes {
		g := Resolve(f, true, true).G
		if g.Caret == g.FocusBar || g.Caret == g.BlurBar || g.Caret == "|" {
			t.Errorf("%s: the ASCII caret %q is a bar (focus %q, blur %q)", f.Name, g.Caret, g.FocusBar, g.BlurBar)
		}
		for name, m := range map[string]string{"done": g.Done, "current": g.Current, "warn": g.Warn, "fail": g.Fail, "caret": g.Caret} {
			if m != "" && strings.Contains(SparkRampASCII, m) {
				t.Errorf("%s: SparkRampASCII %q holds the %s twin %q", f.Name, SparkRampASCII, name, m)
			}
		}
	}
	if len([]rune(SparkRampASCII)) < len([]rune(SparkRamp)) {
		t.Errorf("SparkRampASCII has %d steps, fewer than SparkRamp's %d", len([]rune(SparkRampASCII)), len([]rune(SparkRamp)))
	}
	if tw, _ := ASCIITwin("▏"); tw != "'" {
		t.Errorf("▏ twins as %q, want ' (not the focus bar's |)", tw)
	}
}

// A rule is half the idiom (Law 2), and a border at 1.18:1 on its page is a rule nobody
// sees: welp's #2A231A on #17140F passed palette check's 1.14 floor with its section,
// table and footer rules invisible on both sheets. Every kit theme's border, set or
// derived (ink's), now reads at 1.5:1 or more in both modes and stays under 1.9, where
// palette check warns and fails respectively; the engine's own derived border lands in
// the same band on any page.
func TestBordersAreVisibleButQuiet(t *testing.T) {
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
			p := Resolve(f, dark, false).P
			if cr := Contrast(p.Border, p.Bg); cr < 1.5 || cr > 1.9 {
				t.Errorf("%s dark=%v: border %v on bg %v is %.2f:1, want 1.5–1.9", filepath.Base(path), dark, p.Border, p.Bg, cr)
			}
		}
	}
	for _, c := range []struct {
		bg, fg string
	}{{"#17140F", "#E8E2D6"}, {"#F6F1E6", "#2B2620"}, {"#000000", "#FFFFFF"}, {"#808080", "#FFFFFF"}, {"#FAFAF9", "#1D232D"}} {
		dark := Luminance(Hex(c.bg)) < 0.5
		p := derive(Roles{Bg: c.bg, Fg: c.fg}, dark, nil)
		if cr := Contrast(p.Border, p.Bg); cr < 1.5 || cr > 1.9 {
			t.Errorf("derived border on %s is %.2f:1, want 1.5–1.9", c.bg, cr)
		}
	}
}

// A multi-cell spinner's ASCII twin was the one-cell line spinner: a three-cell dots
// spinner (··· •·· ••· •••) turned into a \ one cell wide under -ascii. The twin keeps
// each frame's width and shape, cell for cell, in twins no status, bar or caret mark
// draws (a cell that would be one takes the next of . : = # …), and two glyphs of the
// frames never share a twin, so the motion survives.
func TestASCIIMultiCellSpinnerKeepsItsShape(t *testing.T) {
	for _, c := range []struct {
		json string
		want []string
	}{
		// · keeps its twin "."; • (twin *, current's) takes the next free cell
		{`{"glyphs": {"spinner": ["···", "•··", "••·", "•••"]}}`, []string{"...", ":..", "::.", ":::"}},
		{`{"glyphs": {"spinner": ["·  ", "·· ", "···", " ··", "  ·", "   "]}}`, []string{".  ", ".. ", "...", " ..", "  .", "   "}},
		// a pending ◌ twins to ".", so the dots take ":" and • the next free cell
		{`{"glyphs": {"pending": "◌", "spinner": ["···", "•··", "••·", "•••"]}}`, []string{":::", "=::", "==:", "==="}},
	} {
		f, err := Load(writeTheme(t, c.json))
		if err != nil {
			t.Fatal(err)
		}
		if err := CheckGlyphs(f.Glyphs); err != nil {
			t.Errorf("%s: CheckGlyphs %v", c.json, err)
		}
		if g := Resolve(f, true, true).G; !slices.Equal(g.Spinner, c.want) {
			t.Errorf("%s: ASCII spinner %q, want %q", c.json, g.Spinner, c.want)
		}
	}
	// every shipped theme: each ASCII frame as wide as its unicode frame
	files, _ := filepath.Glob(filepath.Join("..", "themes", "*.json"))
	test, _ := filepath.Glob(filepath.Join("testdata", "*.json"))
	for _, path := range append(files, test...) {
		f, err := Load(path)
		if err != nil {
			t.Fatal(err)
		}
		uni, asc := Resolve(f, true, false).G.Spinner, Resolve(f, true, true).G.Spinner
		wide := false
		for _, fr := range uni {
			wide = wide || ansi.StringWidth(fr) > 1
		}
		if !wide {
			continue
		}
		if len(asc) != len(uni) {
			t.Errorf("%s: ASCII spinner %q has %d frames, unicode %q %d", filepath.Base(path), asc, len(asc), uni, len(uni))
			continue
		}
		for i := range uni {
			if ansi.StringWidth(asc[i]) != ansi.StringWidth(uni[i]) {
				t.Errorf("%s: ASCII frame %q is %d cells, unicode %q %d", filepath.Base(path), asc[i], ansi.StringWidth(asc[i]), uni[i], ansi.StringWidth(uni[i]))
			}
		}
	}
}
