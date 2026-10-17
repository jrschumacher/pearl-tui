package diversity

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// testdata/<case>.txt is scripts/diversity.py's stdout for the arguments in
// testdata/cases.json, run from this directory, except that a min-distance failure now
// ends in the axes the theme shares with the baseline (TestMinDistanceNamesSharedAxes). testdata/themes is a synthetic failing
// set: two near-identical themes (twin-a, twin-b), a clone of ink with a new palette
// (inkling), a long name, and a theme with no name, a null and a number among its values.
func TestMatchesPython(t *testing.T) {
	b, err := os.ReadFile("testdata/cases.json")
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
		want, err := os.ReadFile("testdata/" + c.Name + ".txt")
		if err != nil {
			t.Fatal(err)
		}
		var out, errb bytes.Buffer
		code := Main(c.Args, "", &out, &errb)
		if code != c.Exit {
			t.Errorf("%s: exit %d, diversity.py exits %d (stderr %q)", c.Name, code, c.Exit, errb.String())
		}
		if out.String() != string(want) {
			t.Errorf("%s: output differs from diversity.py\n--- go\n%s\n--- python\n%s", c.Name, out.String(), want)
		}
	}
}

// Below --max-popular 0.5 one axis can fail on several values. diversity.py lists them
// in set (hash) order, which changes between runs; this port lists them first-seen.
func TestPopularOrderIsStable(t *testing.T) {
	args := []string{"--max-popular", "0.3", "../../../kit/themes/cartridge.json", "../../../kit/themes/ferrous.json",
		"../../../kit/themes/halyard.json", "../../../kit/themes/lumen.json", "../../../kit/themes/moss.json"}
	var first string
	for i := range 5 {
		var out, errb bytes.Buffer
		Main(args, "", &out, &errb)
		if i == 0 {
			first = out.String()
		} else if out.String() != first {
			t.Fatal("output changed between runs")
		}
	}
	filled, bold := strings.Index(first, "tag=filled used"), strings.Index(first, "tag=bold used")
	if filled < 0 || bold < 0 || filled > bold {
		t.Errorf("want tag=filled (cartridge, first) before tag=bold (lumen):\n%s", first)
	}
}

func TestArguments(t *testing.T) {
	ink := "../../../kit/themes/ink.json"
	for _, c := range []struct {
		args []string
		exit int
	}{
		{[]string{"-h"}, 0}, {[]string{"--help"}, 0}, {[]string{"--he"}, 0}, {nil, 2},
		{[]string{ink, "--max-shared"}, 2}, {[]string{ink, "--max-shared", "x"}, 2},
		{[]string{ink, "--max"}, 2}, {[]string{ink, "--colour", "x"}, 2}, {[]string{"nope.json"}, 2},
		{[]string{"--min-distance=-1", "--", ink}, 0}, {[]string{"--max-p", "0.9", ink, "--baseline", "ink"}, 0},
		// diversity.py exited 0 here, having skipped the min-distance rule: see
		// TestMissingBaselineFails.
		{[]string{"--max-p", "0.9", ink, "--baseline", "x"}, 2},
	} {
		var out, errb bytes.Buffer
		if code := Main(c.args, "USAGE", &out, &errb); code != c.exit {
			t.Errorf("diversity %v: exit %d, want %d (stderr %q)", c.args, code, c.exit, errb.String())
		}
	}
}

// A deliberate break from diversity.py, which skipped the min-distance rule without a
// word when a --baseline name matched no theme. brand.md's example passed only the
// brand's themes, so "the default with a new palette" (inkling, here) passed. An
// explicit --baseline that is missing now exits 2 with one line saying which file to
// pass; the default baseline, when missing, only adds a note on stderr.
func TestMissingBaselineFails(t *testing.T) {
	const (
		kit     = "../../../kit/themes/"
		inkling = "testdata/themes/inkling.json"
		twinA   = "testdata/themes/twin-a.json"
	)
	for _, c := range []struct {
		args   []string
		exit   int
		stderr string
	}{
		{[]string{inkling, twinA, "--baseline", "ink,charm", "--min-distance", "8"}, 2,
			`diversity: baselines "ink", "charm" are not among the themes, so the min-distance rule cannot run: ` +
				"pass their theme files too (e.g. kit/themes/ink.json kit/themes/charm.json, from the skill directory) or drop them from --baseline\n"},
		{[]string{inkling, twinA, kit + "ink.json", "--baseline=ink,charm"}, 2,
			`diversity: baseline "charm" is not among the themes, so the min-distance rule cannot run: ` +
				"pass its theme file too (e.g. kit/themes/charm.json, from the skill directory) or drop it from --baseline\n"},
		// the miss itself: without the baselines, inkling passes (exit 0), so say so
		{[]string{inkling, twinA}, 0,
			`diversity: note: baselines "ink", "charm" are not among the themes, so the min-distance rule did not run against them ` +
				"(pass kit/themes/ink.json kit/themes/charm.json, or --baseline \"\" to silence this)\n"},
		{[]string{inkling, twinA, "--baseline", ""}, 0, ""},
		// a space after the comma names the same theme
		{[]string{inkling, twinA, kit + "ink.json", "--baseline", "ink, charm"}, 2,
			`diversity: baseline "charm" is not among the themes, so the min-distance rule cannot run: ` +
				"pass its theme file too (e.g. kit/themes/charm.json, from the skill directory) or drop it from --baseline\n"},
		{[]string{inkling, twinA, kit + "ink.json", kit + "charm.json", "--baseline", "ink,charm"}, 1, ""},
	} {
		var out, errb bytes.Buffer
		code := Main(c.args, "", &out, &errb)
		if code != c.exit || errb.String() != c.stderr {
			t.Errorf("diversity %v: exit %d stderr %q; want %d %q", c.args, code, errb.String(), c.exit, c.stderr)
		}
		if code == 2 && out.Len() != 0 {
			t.Errorf("diversity %v: printed a report before failing:\n%s", c.args, out.String())
		}
	}
	// With the baseline passed, the rule runs and catches the recoloured default.
	var out, errb bytes.Buffer
	Main([]string{inkling, kit + "ink.json", "--baseline", "ink"}, "", &out, &errb)
	if !strings.Contains(out.String(), "inkling differs from baseline ink on only") {
		t.Errorf("min-distance did not catch inkling:\n%s", out.String())
	}
}

// Under dialog "rules" the title sits in the top rule, so dialogTitle renders nothing.
func TestRulesDialogKillsDialogTitle(t *testing.T) {
	p := filepath.Join(t.TempDir(), "t.json")
	if err := os.WriteFile(p, []byte(`{"name": "t", "expressions": {"dialog": "rules", "dialogTitle": "bar"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	th, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if th.Choices["dialogTitle"] != Dead || th.Choices["dialog"] != "rules" {
		t.Errorf("choices %v, want dialogTitle dead under dialog rules", th.Choices)
	}
}

// The nearest-theme line is advice under the table, never a failure: welp passed the
// baselines and matched the kit's moss on 11 of 14 axes, and nothing said so (#22).
// twin-a and twin-b are each other's nearest; a baseline never appears as a subject,
// and with fewer than two brand themes nothing is said.
func TestNearest(t *testing.T) {
	const kit = "../../../kit/themes/"
	var out, errb bytes.Buffer
	code := Main([]string{"testdata/themes/twin-a.json", "testdata/themes/twin-b.json", kit + "ink.json", kit + "charm.json", "--baseline", "ink,charm"}, "", &out, &errb)
	if code != 1 {
		t.Fatalf("exit %d (twins share too many axes)\n%s", code, out.String())
	}
	for _, want := range []string{"nearest to twin-a: twin-b (shares 16/", "nearest to twin-b: twin-a (shares 16/"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("no %q in\n%s", want, out.String())
		}
	}
	if strings.Contains(out.String(), "nearest to ink") || strings.Contains(out.String(), "nearest to charm") {
		t.Errorf("a baseline got a nearest line:\n%s", out.String())
	}
	out.Reset()
	if code := Main([]string{"testdata/themes/twin-a.json", kit + "ink.json", "--baseline", "ink"}, "", &out, &errb); code != 0 || strings.Contains(out.String(), "nearest") {
		t.Errorf("one brand theme: exit %d, output\n%s", code, out.String())
	}
	// a passing set still gets the note: it is not a failure
	out.Reset()
	if code := Main([]string{kit + "halyard.json", kit + "moss.json", kit + "ink.json", "--baseline", "ink"}, "", &out, &errb); code != 0 || !strings.Contains(out.String(), "nearest to halyard: moss (shares ") || !strings.HasSuffix(out.String(), "OK\n") {
		t.Errorf("halyard and moss: exit %d\n%s", code, out.String())
	}
}

// A min-distance failure names the axes the theme shares with the baseline, so the
// agent sees what to move and can trace each to the brand read (a recreate run moved
// three axes blind, one of them only for the lint). Each named axis is live in both and
// equal; the count is the live axes less the distance.
func TestMinDistanceNamesSharedAxes(t *testing.T) {
	ink, err := Load("../../../kit/themes/ink.json")
	if err != nil {
		t.Fatal(err)
	}
	charm, err := Load("../../../kit/themes/charm.json")
	if err != nil {
		t.Fatal(err)
	}
	var want []string
	for _, a := range Axes {
		if v := charm.Choices[a.Key]; v != Dead && v == ink.Choices[a.Key] {
			want = append(want, a.Key)
		}
	}
	if len(want) == 0 {
		t.Fatal("charm shares no axis with ink: the probe tests nothing")
	}
	fails := Failures([]Theme{charm, ink}, Options{MaxShared: 99, MaxPopular: 1, MinDistance: 99, Baseline: map[string]bool{"ink": true}})
	if len(fails) != 1 {
		t.Fatalf("failures %q, want one min-distance failure", fails)
	}
	if !strings.HasSuffix(fails[0], "; shares "+strings.Join(want, ", ")) {
		t.Errorf("failure %q, want it to end in the %d shared axes %s", fails[0], len(want), strings.Join(want, ", "))
	}
}
