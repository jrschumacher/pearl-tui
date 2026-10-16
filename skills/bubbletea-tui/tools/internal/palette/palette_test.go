package palette

import (
	"bufio"
	"bytes"
	"encoding/json"
	"image/color"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"tuikit/theme"
	"tuitools/internal/oklch"
)

// The goldens in testdata are scripts/palette.py's output on the same inputs:
//
//	check/<name>.txt     stdout of `palette.py check` on kit/themes/<name>.json or check/<name>.json
//	derive/<case>.*      `palette.py derive <args> --swatch swatch.ansi` run in an empty
//	                     directory: .out stdout, .err stderr, .ansi the swatch, .json the --out file
//	derive/corpus.txt    roles and xterm indices for 250 random inputs (of 8,000 matched)
//
// The intended differences: the swatch hint names the Go cellshot, not cellshot.py;
// check fails role colours Lip Gloss would misread, which palette.py let through (see
// TestCheckRejectsWhatTheEngineRejects); and where neither ink cleared an on-colour's
// floor (palette.py shipped the failure), the ink is now theme.OnInk's deepened one.
// That changed derive/tint and derive/tint-zero (exit 1 → 0), 14 corpus rows'
// onPrimary/fgSelected, and check/failing's light onPrimary (the engine now draws
// #444444, not #555555, on #7FB3FF).

const kitThemes = "../../../kit/themes"

func readJSON(t *testing.T, path string, v any) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, v); err != nil {
		t.Fatal(err)
	}
}

func golden(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestCheckMatchesPython(t *testing.T) {
	var exits map[string]int
	readJSON(t, "testdata/check/exits.json", &exits)
	kit, _ := filepath.Glob(kitThemes + "/*.json")
	if len(kit) == 0 {
		t.Fatal("no kit themes")
	}
	for _, f := range kit { // every kit theme has a golden
		if _, ok := exits[strings.TrimSuffix(filepath.Base(f), ".json")]; !ok {
			t.Errorf("no golden for %s", f)
		}
	}
	for name, exit := range exits {
		path := filepath.Join(kitThemes, name+".json")
		if _, err := os.Stat(path); err != nil {
			path = filepath.Join("testdata/check", name+".json")
		}
		var out, errb bytes.Buffer
		code := Main([]string{"check", path}, "", &out, &errb)
		if code != exit {
			t.Errorf("%s: exit %d, palette.py exits %d (stderr %q)", name, code, exit, errb.String())
		}
		if want := golden(t, "testdata/check/"+name+".txt"); out.String() != want {
			t.Errorf("%s: output differs from palette.py\n--- go\n%s\n--- python\n%s", name, out.String(), want)
		}
	}
}

func TestDeriveMatchesPython(t *testing.T) {
	var cases []struct {
		Name string
		Args []string
	}
	readJSON(t, "testdata/derive/cases.json", &cases)
	var exits map[string]int
	readJSON(t, "testdata/derive/exits.json", &exits)
	td, err := filepath.Abs("testdata/derive")
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(t.TempDir())
	for _, c := range cases {
		base := filepath.Join(td, c.Name)
		var out, errb bytes.Buffer
		code := Main(append([]string{"derive"}, append(c.Args, "--swatch", "swatch.ansi")...), "", &out, &errb)
		if code != exits[c.Name] {
			t.Errorf("%s: exit %d, palette.py exits %d", c.Name, code, exits[c.Name])
		}
		if want := golden(t, base+".out"); out.String() != want {
			t.Errorf("%s: stdout differs from palette.py\n--- go\n%s\n--- python\n%s", c.Name, out.String(), want)
		}
		want := strings.Replace(golden(t, base+".err"), "(render: cellshot.py ", "(render: cellshot ", 1)
		if errb.String() != want {
			t.Errorf("%s: stderr differs from palette.py\n--- go\n%s\n--- python\n%s", c.Name, errb.String(), want)
		}
		if got := golden(t, "swatch.ansi"); got != golden(t, base+".ansi") {
			t.Errorf("%s: swatch differs from palette.py", c.Name)
		}
		if _, err := os.Stat(base + ".json"); err == nil {
			if got := golden(t, "roles.json"); got != golden(t, base+".json") {
				t.Errorf("%s: --out file differs from palette.py", c.Name)
			}
		}
	}
}

func TestDeriveCorpusMatchesPython(t *testing.T) {
	f, err := os.Open("testdata/derive/corpus.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(nil, 1<<20)
	n := 0
	for sc.Scan() {
		if strings.HasPrefix(sc.Text(), "# ") {
			continue
		}
		parts := strings.Split(sc.Text(), " | ")
		in := strings.Fields(parts[0])
		for i := range 5 {
			if in[i] == "-" {
				in[i] = ""
			}
		}
		tint, err := strconv.ParseFloat(in[5], 64)
		if err != nil {
			t.Fatal(err)
		}
		modes := Derive(in[0], in[1], in[2], in[3], in[4], tint)
		for i, m := range modes {
			keys := make([]string, 0, len(m.Roles))
			for k := range m.Roles {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			hexes, idx := make([]string, len(keys)), make([]string, len(keys))
			for j, k := range keys {
				hexes[j], idx[j] = m.Roles[k], strconv.Itoa(oklch.Xterm256(m.Roles[k]))
			}
			if got := strings.Join(hexes, " "); got != parts[1+2*i] {
				t.Errorf("derive %v %s roles:\n go %s\n py %s", in, m.Name, got, parts[1+2*i])
			}
			if got := strings.Join(idx, " "); got != parts[2+2*i] {
				t.Errorf("derive %v %s ansi256:\n go %s\n py %s", in, m.Name, got, parts[2+2*i])
			}
		}
		n++
	}
	if n < 250 {
		t.Fatalf("only %d corpus rows", n)
	}
}

// The checker's on-colours are the engine's: for every kit theme and mode, a filled
// on-colour is exactly what theme.Resolve draws.
func TestFillOnColoursIsTheEngine(t *testing.T) {
	kit, _ := filepath.Glob(kitThemes + "/*.json")
	for _, path := range kit {
		f, err := theme.Load(path)
		if err != nil {
			t.Fatal(err)
		}
		for mode, roles := range f.Roles {
			dark := mode != "light"
			b, _ := json.Marshal(roles)
			r := map[string]string{}
			_ = json.Unmarshal(b, &r)
			for k, v := range r {
				if v == "" {
					delete(r, k)
				}
			}
			if err := FillOnColours(r, dark); err != nil {
				t.Fatal(err)
			}
			p := theme.Resolve(f, dark, false).P
			for on, c := range map[string]color.Color{"onPrimary": p.OnPrimary, "onSecondary": p.OnSecondary,
				"onWarning": p.OnWarning, "onError": p.OnError, "onSuccess": p.OnSuccess, "fgSelected": p.FgSelected} {
				if !same(oklch.Color(r[on]), c) {
					t.Errorf("%s %s: checker fills %s=%s, engine draws %s", filepath.Base(path), mode, on, r[on], hexOf(c))
				}
			}
		}
	}
}

// A theme without bg gets the engine's default background as a candidate ink (palette.py
// used #000000, which the app never draws).
func TestFillOnColoursUsesEngineDefaults(t *testing.T) {
	r := map[string]string{"primary": "#8FB8DE", "fg": "#D8DEE9"}
	if err := FillOnColours(r, true); err != nil {
		t.Fatal(err)
	}
	if r["onPrimary"] != "#15171C" || r["fgSelected"] != "#15171C" {
		t.Errorf("onPrimary %s, fgSelected %s; want the engine's dark bg #15171C", r["onPrimary"], r["fgSelected"])
	}
	if _, ok := r["onSecondary"]; ok {
		t.Errorf("onSecondary filled without a secondary")
	}
}

func TestUsageAndErrors(t *testing.T) {
	for _, c := range []struct {
		args []string
		exit int
		out  string
	}{
		{nil, 0, "USAGE"}, {[]string{"-h"}, 0, "USAGE"}, {[]string{"--help"}, 0, "USAGE"},
		{[]string{"paint"}, 2, "USAGE"}, {[]string{"check"}, 2, ""}, {[]string{"check", "nope.json"}, 2, ""},
		{[]string{"derive"}, 2, ""}, {[]string{"derive", "red"}, 2, ""}, {[]string{"derive", "#123456", "--bg"}, 2, ""},
		{[]string{"derive", "#123456", "--tint", "lots"}, 2, ""}, {[]string{"derive", "#123456", "#xyz"}, 2, ""},
	} {
		var out, errb bytes.Buffer
		if code := Main(c.args, "USAGE", &out, &errb); code != c.exit || strings.TrimSpace(out.String()) != c.out {
			t.Errorf("palette %v: exit %d stdout %q; want %d %q", c.args, code, out.String(), c.exit, c.out)
		}
	}
}

func TestOptionWithEquals(t *testing.T) {
	var a, b, e bytes.Buffer
	Main([]string{"derive", "#6B50FF", "--bg=#201F26", "--name=x"}, "", &a, &e)
	Main([]string{"derive", "#6B50FF", "--bg", "#201F26", "--name", "x"}, "", &b, &e)
	if a.String() != b.String() || !strings.Contains(a.String(), `"bg": "#201F26"`) {
		t.Errorf("--opt=value and --opt value differ")
	}
}

// A deliberate break from palette.py, which read colours as loosely as its hex2rgb
// does: "6B50FF", "#6B50", "#6B50F" and "#6B50FFAA" all passed. Lip Gloss reads a
// string without "#" as an ANSI index or no colour, and any other length after "#" as
// no colour, so the app drew something else while the check said OK. check now uses
// the engine's rule (theme.CheckHex): it fails exactly the themes theme.Load fails,
// with one line on stderr and exit 2.
func TestCheckRejectsWhatTheEngineRejects(t *testing.T) {
	for _, c := range []struct {
		roles string
		err   string // the stderr line after "palette: <path>: "; "" means accepted
	}{
		{`"primary": "6B50FF"`, `dark.primary: "6B50FF" has no #: Lip Gloss reads it as an ANSI index or no colour; write "#6B50FF"`},
		{`"primary": "#6B50FFAA"`, `dark.primary: "#6B50FFAA" is not #RGB or #RRGGBB`},
		{`"primary": "#6B50"`, `dark.primary: "#6B50" is not #RGB or #RRGGBB`},
		{`"primary": "#6B50F"`, `dark.primary: "#6B50F" is not #RGB or #RRGGBB`},
		{`"primary": "##6B50FF"`, `dark.primary: "##6B50FF" is not #RGB or #RRGGBB`},
		{`"primary": " #6B50FF"`, `dark.primary: " #6B50FF" is not #RGB or #RRGGBB`},
		{`"warning": "12"`, `dark.warning: "12" has no #: Lip Gloss reads it as an ANSI index or no colour; write #RGB or #RRGGBB`},
		{`"fgSoft": "6B50FF"`, `dark.fgSoft: "6B50FF" has no #: Lip Gloss reads it as an ANSI index or no colour; write "#6B50FF"`},
		{`"primary": "#6B50FF"`, ""},
		{`"primary": "#6b50ff"`, ""},
		{`"primary": "#abc"`, ""},
	} {
		path := filepath.Join(t.TempDir(), "theme.json")
		js := `{"roles": {"dark": {"bg": "#201F26", "fg": "#DFDBDD", ` + c.roles + `}}}`
		if err := os.WriteFile(path, []byte(js), 0o644); err != nil {
			t.Fatal(err)
		}
		var out, errb bytes.Buffer
		code := Main([]string{"check", path}, "", &out, &errb)
		_, loadErr := theme.Load(path)
		if c.err == "" {
			if code == 2 || loadErr != nil {
				t.Errorf("%s: check exit %d (%q), theme.Load %v; want both to accept", c.roles, code, errb.String(), loadErr)
			}
			continue
		}
		if want := "palette: " + path + ": " + c.err + "\n"; code != 2 || errb.String() != want || out.Len() != 0 {
			t.Errorf("%s: exit %d stderr %q stdout %q; want 2 and %q", c.roles, code, errb.String(), out.String(), want)
		}
		if loadErr == nil {
			t.Errorf("%s: palette check fails it but theme.Load accepts it", c.roles)
		}
	}
}
