package theme

import (
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
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
// ●'s twin: a ◔ "reading" state beside a ● "new" one drew as the same mark. Now a
// theme gives its glyphs twins in glyphs.ascii, and Load fails on one without.
func TestLoadFailsOnAGlyphWithNoASCIITwin(t *testing.T) {
	for _, c := range []struct{ json, want string }{
		{`{"glyphs": {"current": "●", "pending": "◔"}}`, `glyphs.pending "◔" has no ASCII twin`},
		{`{"glyphs": {"focusBar": "🮇"}}`, `glyphs.focusBar "🮇" has no ASCII twin`},
		{`{"glyphs": {"pending": "◔", "ascii": {"◔": "◐"}}}`, `glyphs.ascii: "◔" → "◐" is not ASCII`},
		{`{"glyphs": {"ascii": {"◔": ""}}}`, `glyphs.ascii: "◔" → "" is not ASCII`},
	} {
		path := writeTheme(t, c.json)
		_, err := Load(path)
		if err == nil || !strings.Contains(err.Error(), c.want) || !strings.HasPrefix(err.Error(), path+": ") {
			t.Errorf("Load(%s) = %v, want %q", c.json, err, c.want)
		}
	}
	// warn and fail are always "!" and "x" under ASCII, so they need no twin
	if _, err := Load(writeTheme(t, `{"glyphs": {"warn": "◬", "fail": "✘"}}`)); err != nil {
		t.Errorf("Load: %v", err)
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
