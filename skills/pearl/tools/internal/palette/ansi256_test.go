package palette

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"tuitools/internal/oklch"
)

// At 256 colours a theme's background ramp can land on one xterm index: moss's dark bg
// #1A1C19 and bgPanel #1A2720 are both 234, so a panel, an output block and a borderless
// dialog lose their edge (#55). The kit draws the upper step on the next grey instead
// (theme.ramp256), so check says nothing of a step it moves, nor of a pin that parts
// them; a pin that keeps the two together is the theme's own, and check names it with
// the grey that parts them.
func TestCheckReportsASharedIndexAt256(t *testing.T) {
	roles := `{"dark": {"bg": "#1A1C19", "bgPanel": "#1A2720", "fg": "#C7D2CC", "primary": "#86C29F"}}`
	for _, c := range []struct{ name, pins, want string }{
		{"shared", ``, ``}, // the kit draws bgPanel as grey 235 (theme.ramp256): nothing for the theme to do
		{"pinned apart", `, "ansi256": {"dark": {"bgPanel": 235}}`, ``},
		{"pinned together", `, "ansi256": {"dark": {"bgPanel": 234}}`, `both draw as xterm 234 at 256 colours (-profile ansi256), so a panel has no edge on the page there; the theme pins it: "ansi256": {"dark": {"bgPanel": 235}} gives it an edge`},
	} {
		path := filepath.Join(t.TempDir(), "theme.json")
		if err := os.WriteFile(path, []byte(`{"roles": `+roles+`, "ansi16": {"dark": {"warning": 11}}`+c.pins+`}`), 0o644); err != nil {
			t.Fatal(err)
		}
		var out, errb bytes.Buffer
		if code := Main([]string{"check", path}, "", &out, &errb); code != 0 {
			t.Errorf("%s: exit %d, want 0 (a warning)\n%s", c.name, code, errb.String())
		}
		got := errb.String()
		if c.want == "" && strings.Contains(got, "ansi256") || c.want != "" && !strings.Contains(got, "palette: warning: "+path+": ") || !strings.Contains(got, c.want) {
			t.Errorf("%s: stderr %q, want %q", c.name, got, c.want)
		}
	}
}

// A theme pins a role at 256 colours with ansi256, in the shape derive writes it. check
// exits 2 on a pin the kit can't read (theme.CheckANSI256) and warns on one that draws a
// role visibly apart from its hex: a pin left behind when the hex was tuned by hand.
func TestCheckJudgesANSI256Pins(t *testing.T) {
	roles := `{"dark": {"bg": "#17140F", "fg": "#EDE6DA", "primary": "#D4883C"}}`
	for _, c := range []struct {
		name, pins string
		code       int
		want       string
	}{
		{"derive's nearest", `{"dark": {"primary": 172}}`, 0, ``},
		{"stale", `{"dark": {"primary": 33}}`, 0, `dark: primary #D4883C is pinned to xterm 33 at 256 colours (-profile ansi256)`},
		{"no colour", `{"dark": {"primary": 300}}`, 2, `ansi256.dark.primary: 300 is not one of the 256 colours (0-255)`},
		{"no role", `{"dark": {"brand": 172}}`, 2, `ansi256.dark.brand: not a role`},
	} {
		path := filepath.Join(t.TempDir(), "theme.json")
		if err := os.WriteFile(path, []byte(`{"roles": `+roles+`, "ansi16": {"dark": {"primary": 11, "warning": 3}}, "ansi256": `+c.pins+`}`), 0o644); err != nil {
			t.Fatal(err)
		}
		var out, errb bytes.Buffer
		code := Main([]string{"check", path}, "", &out, &errb)
		if code != c.code || c.want == "" && strings.Contains(errb.String(), "256") || !strings.Contains(errb.String(), c.want) {
			t.Errorf("%s: exit %d stderr %q, want %d and %q", c.name, code, errb.String(), c.code, c.want)
		}
	}
}

// derive's ansi256 block is what the kit draws at 256 colours, so its background ramp
// never shares an index: across the whole corpus, bg, bgPanel, bgSubtle and bgOverlay
// are four indices in both modes, on every page that lies on its mode's side (a near-white
// --bg leaves no grey past white for the steps, the brand's call, as
// TestDeriveCorpusMatchesPython says of its border).
func TestDeriveKeepsTheRampApartAt256(t *testing.T) {
	f, err := os.Open("testdata/derive/corpus.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(nil, 1<<20)
	for sc.Scan() {
		if strings.HasPrefix(sc.Text(), "# ") {
			continue
		}
		in := strings.Fields(strings.Split(sc.Text(), " | ")[0])
		for i := range 5 {
			if in[i] == "-" {
				in[i] = ""
			}
		}
		tint, _ := strconv.ParseFloat(in[5], 64)
		for _, m := range Derive(in[0], in[1], in[2], in[3], in[4], tint) {
			if lb, _, _ := oklch.LCH(m.Roles["bg"]); (lb < 0.5) != (m.Name == "dark") {
				continue
			}
			idx := Ansi256(m)
			seen := map[int]string{}
			for _, role := range []string{"bg", "bgPanel", "bgSubtle", "bgOverlay"} {
				if other, dup := seen[idx[role]]; dup {
					t.Errorf("derive %v %s: %s and %s both xterm %d", in, m.Name, other, role, idx[role])
				}
				seen[idx[role]] = role
			}
		}
	}
}
