package theme

import (
	"image/color"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
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
