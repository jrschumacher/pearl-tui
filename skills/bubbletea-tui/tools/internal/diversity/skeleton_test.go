package diversity

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite testdata/skeleton-*.txt from the current output")

const (
	appDir   = "testdata/apps/"
	dunlin   = appDir + "dunlin/TUI.md"   // a monitor: verdict on top, a summary band first, the kit's trace
	whimbrel = appDir + "whimbrel/TUI.md" // the same spec, another brand: host at the foot, the same band, verdict and trace
	godwit   = appDir + "godwit/TUI.md"   // a third brand: a sidebar, no verdict, braille
	kitInk   = "../../../kit/themes/ink.json"
	kitCharm = "../../../kit/themes/charm.json"
)

func run(args ...string) (string, string, int) {
	var out, errb bytes.Buffer
	code := Main(args, "", &out, &errb)
	return out.String(), errb.String(), code
}

// testdata/skeleton-<case>.txt is the output for the arguments in
// testdata/skeleton-cases.json. These are not diversity.py goldens: the skeleton axes and
// --verbose are new. go test -run TestSkeletonGoldens -update rewrites them.
func TestSkeletonGoldens(t *testing.T) {
	b, err := os.ReadFile("testdata/skeleton-cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Name string
		Args []string
		Exit int
	}
	if err := json.Unmarshal(b, &cases); err != nil {
		t.Fatal(err)
	}
	for _, c := range cases {
		out, errs, code := run(c.Args...)
		if code != c.Exit {
			t.Errorf("%s: exit %d, want %d (stderr %q)", c.Name, code, c.Exit, errs)
		}
		path := "testdata/skeleton-" + c.Name + ".txt"
		if *update {
			if err := os.WriteFile(path, []byte(out), 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		want, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if out != string(want) {
			t.Errorf("%s: output differs from %s\n--- got\n%s\n--- want\n%s", c.Name, path, out, want)
		}
	}
}

// The miss (#45, third pair eval): two cold monitors whose themes differ came out one
// skeleton: a verdict sentence on top, a resource band before the list, the kit's trace in
// both. Their TUI.md files declare it, and diversity fails the pair on it, naming the
// shared axes. The diverged pair (a sidebar, no verdict, braille) shares 3 and passes.
func TestConvergedSkeletonFails(t *testing.T) {
	out, _, code := run(dunlin, whimbrel, kitInk, kitCharm)
	if code != 1 {
		t.Fatalf("converged pair: exit %d, want 1\n%s", code, out)
	}
	if !strings.Contains(out, "FAIL dunlin and whimbrel share 4 of 7 skeleton axes (> 3): order, verdict, chart, chartGlyphs") {
		t.Errorf("no skeleton failure naming the four shared axes:\n%s", out)
	}
	for _, flag := range []string{"[P2] skeleton verdict=top held by every app", "[P2] skeleton order=summary-first held by every app", "[P2] skeleton chart=trace held by every app"} {
		if !strings.Contains(out, flag) {
			t.Errorf("no %q:\n%s", flag, out)
		}
	}
	out, _, code = run(dunlin, godwit, kitInk, kitCharm)
	if code != 0 || strings.Contains(out, "skeleton axes (>") {
		t.Errorf("diverged pair: exit %d, want 0 and no skeleton failure\n%s", code, out)
	}
	if !strings.Contains(out, "[P2] skeleton chart=trace held by every app (dunlin, godwit") {
		t.Errorf("a shared departure is still flagged, as a note:\n%s", out)
	}
	// --max-shared-skeleton moves the line
	if _, _, code := run(dunlin, whimbrel, kitInk, kitCharm, "--max-shared-skeleton", "4"); code != 0 {
		t.Errorf("--max-shared-skeleton 4: exit %d, want 0", code)
	}
}

// The gaming #45 warns about: in #37 an agent moved an expression axis only to pass,
// against its own brand read. Every run with a skeleton failure or a newcomer flag ends
// in one line that names the brand read as the only reason to move an axis, and no
// failure line suggests a value to move to.
func TestSkeletonFailurePointsAtTheBrandRead(t *testing.T) {
	out, _, _ := run(dunlin, whimbrel, kitInk, kitCharm)
	if strings.Count(out, FromTheRead) != 1 {
		t.Errorf("want the brand-read line exactly once:\n%s", out)
	}
	for _, want := range []string{"brand read", "move: and signature:", "moved only to pass moves back", "seating plan"} {
		if !strings.Contains(FromTheRead, want) {
			t.Errorf("FromTheRead lacks %q", want)
		}
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "FAIL") && (strings.Contains(line, " to ") || strings.Contains(line, "try ")) {
			t.Errorf("a failure suggests a target: %q", line)
		}
	}
	// a single app and a passing run without flags say nothing about it
	if out, _, _ := run(dunlin, kitInk, kitCharm); strings.Contains(out, FromTheRead) {
		t.Errorf("one app: the line printed with nothing to answer:\n%s", out)
	}
}

// A missing declaration is a P2 note, never a failure: the app's skeleton reads "?" and
// counts nowhere, and the note gives the line to add with every axis's vocabulary. A
// `skeleton:` inside a fenced sketch is not a declaration.
func TestMissingSkeletonIsP2(t *testing.T) {
	out, errs, code := run(appDir+"bare/TUI.md", godwit, kitInk, kitCharm)
	if code != 0 || errs != "" {
		t.Fatalf("exit %d stderr %q, want 0 and nothing\n%s", code, errs, out)
	}
	if !strings.Contains(out, "[P2] "+appDir+"bare/TUI.md: no skeleton line") || !strings.Contains(out, skeletonTemplate()) {
		t.Errorf("no P2 note with the template line:\n%s", out)
	}
	if strings.Contains(out, "skeleton of bare") || strings.Contains(out, "(bare, godwit") || strings.Contains(out, "(godwit, bare") {
		t.Errorf("an undeclared skeleton was counted:\n%s", out)
	}
	app, notes, err := LoadApp(appDir + "bare/TUI.md")
	if err != nil || len(notes) != 1 {
		t.Fatalf("LoadApp: %v, notes %q", err, notes)
	}
	for _, a := range SkeletonAxes {
		if app.Skeleton[a.Key] != Unknown {
			t.Errorf("bare %s = %q, want %q (the fenced line must not count)", a.Key, app.Skeleton[a.Key], Unknown)
		}
	}
}

// odd/TUI.md declares a word outside the vocabulary, an axis twice, an unknown key, a key
// with no value, and leaves axes out: each is one P2 note, none fails, and a chart of
// "none" kills the chart glyphs.
func TestSkeletonNotes(t *testing.T) {
	app, notes, err := LoadApp(appDir + "odd/TUI.md")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"head": "ribbon", "status": "footer", "order": Unknown, "foot": Unknown, "verdict": Unknown,
		"chart": "none", "chartGlyphs": Dead, "turnEnd": Unknown}
	for k, v := range want {
		if app.Skeleton[k] != v {
			t.Errorf("odd %s = %q, want %q", k, app.Skeleton[k], v)
		}
	}
	for _, n := range []string{`head "ribbon" is not in its vocabulary`, "status is declared twice; the first (footer) counts",
		`names "layout", which is no axis`, "verdict has no value", "declares no order, foot, turnEnd, so they don't count"} {
		found := false
		for _, got := range notes {
			found = found || (strings.HasPrefix(got, "[P2] ") && strings.Contains(got, n))
		}
		if !found {
			t.Errorf("no note %q in %q", n, notes)
		}
	}
	if len(notes) != 5 {
		t.Errorf("%d notes, want 5: %q", len(notes), notes)
	}
}

// An app's theme comes from its TUI.md's § Theme; passed again as a file (themes/*.json
// beside TUI.md), it is the same column, not a twin that fails max-shared. A TUI.md with
// no theme file is an input error.
func TestAppThemeOnce(t *testing.T) {
	out, _, code := run(dunlin, appDir+"dunlin/themes/dunlin.json", kitInk, kitCharm)
	if code != 0 || strings.Count(strings.SplitN(out, "\n", 2)[0], "dunlin") != 1 {
		t.Errorf("exit %d; want one dunlin column\n%s", code, out)
	}
	out, errs, code := run(appDir+"notheme/TUI.md", kitInk)
	if code != 2 || out != "" || !strings.Contains(errs, "§ Theme names no theme file") {
		t.Errorf("no theme: exit %d stderr %q out %q", code, errs, out)
	}
	// relative to the TUI.md, not to the working directory
	p := filepath.Join(t.TempDir(), "TUI.md")
	if err := os.WriteFile(p, []byte("### Theme\nthemes/gone.json\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, errs, code := run(p); code != 2 || !strings.Contains(errs, filepath.Join(filepath.Dir(p), "themes/gone.json")) {
		t.Errorf("missing theme file: exit %d stderr %q", code, errs)
	}
}

// --max-popular counts newcomers (apps) alone: with three apps a value two hold is 67%
// and flagged at 0.5, not at 0.7. A value the kit holds is never a newcomer's departure.
func TestNewcomerFlagThreshold(t *testing.T) {
	out, _, _ := run(dunlin, whimbrel, godwit, kitInk, kitCharm)
	if !strings.Contains(out, "[P2] skeleton verdict=top held by 2/3 apps (dunlin, whimbrel; > 50%)") {
		t.Errorf("2 of 3 at 0.5 not flagged:\n%s", out)
	}
	if !strings.Contains(out, "[P2] skeleton chart=trace held by every app (dunlin, whimbrel, godwit; > 50%)") {
		t.Errorf("3 of 3 not flagged:\n%s", out)
	}
	if strings.Contains(out, "head=bar held") || strings.Contains(out, "status=header held") {
		t.Errorf("the kit's own value flagged as a departure:\n%s", out)
	}
	out, _, _ = run(dunlin, whimbrel, godwit, kitInk, kitCharm, "--max-popular", "0.7")
	if strings.Contains(out, "verdict=top held") || !strings.Contains(out, "chart=trace held by every app") {
		t.Errorf("--max-popular 0.7:\n%s", out)
	}
}

// --verbose (#53) prints every axis's seating, skeleton included when an app is among the
// inputs, and every pair's shared axes; without it neither appears, so a theme-only run is
// still diversity.py's output.
func TestVerbose(t *testing.T) {
	kit := []string{"cartridge", "charm", "ferrous", "halyard", "ink", "lumen", "moss"}
	var args []string
	for _, k := range kit {
		args = append(args, "../../../kit/themes/"+k+".json")
	}
	plain, _, _ := run(args...)
	verbose, _, _ := run(append([]string{"--verbose"}, args...)...)
	short, _, _ := run(append([]string{"-v"}, args...)...)
	if verbose != short {
		t.Error("-v and --verbose differ")
	}
	if strings.Contains(plain, "seating:") || strings.Contains(plain, "pairs:") {
		t.Errorf("seating without --verbose:\n%s", plain)
	}
	lines := strings.Split(verbose, "\n")
	seat := 0
	for _, l := range lines {
		for _, a := range Axes {
			if strings.HasPrefix(l, pyfmt13(a.Key)) && strings.Contains(l, ": ") {
				seat++
			}
		}
	}
	if seat != len(Axes) {
		t.Errorf("%d seating lines, want %d\n%s", seat, len(Axes), verbose)
	}
	for i, a := range kit {
		for _, b := range kit[i+1:] {
			if !strings.Contains(verbose, "\n"+a+" · "+b+": ") {
				t.Errorf("no pair line for %s · %s", a, b)
			}
		}
	}
	withApp, _, _ := run("-v", dunlin, whimbrel, kitInk, kitCharm)
	for _, a := range SkeletonAxes {
		if !strings.Contains(withApp, "\n"+pyfmt13(a.Key)) {
			t.Errorf("no skeleton seating for %s", a.Key)
		}
	}
	if !strings.Contains(withApp, "dunlin · whimbrel: ") || !strings.Contains(withApp, "; skeleton 4/7 (order, verdict, chart, chartGlyphs)") {
		t.Errorf("app pair lacks its skeleton share:\n%s", withApp)
	}
}

func pyfmt13(s string) string { return s + strings.Repeat(" ", 13-len(s)) }

func TestSkeletonArguments(t *testing.T) {
	for _, c := range []struct {
		args []string
		exit int
	}{
		{[]string{"--verbose=yes", kitInk}, 2}, {[]string{"--max-shared-skeleton", "x", kitInk}, 2},
		{[]string{"--max-shared-s", "2", kitInk, "--baseline", ""}, 0}, {[]string{"--verb", kitInk, "--baseline", ""}, 0},
		{[]string{"--max-shared", "9", kitInk, "--baseline", ""}, 0}, // the exact name is never ambiguous
	} {
		if _, errs, code := run(c.args...); code != c.exit {
			t.Errorf("diversity %v: exit %d, want %d (stderr %q)", c.args, code, c.exit, errs)
		}
	}
}
