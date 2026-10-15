package diversity

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// testdata/<case>.txt is scripts/diversity.py's stdout for the arguments in
// testdata/cases.json, run from this directory. testdata/themes is a synthetic failing
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
		{[]string{"--min-distance=-1", "--", ink}, 0}, {[]string{"--max-p", "0.9", ink, "--baseline", "x"}, 0},
	} {
		var out, errb bytes.Buffer
		if code := Main(c.args, "USAGE", &out, &errb); code != c.exit {
			t.Errorf("diversity %v: exit %d, want %d (stderr %q)", c.args, code, c.exit, errb.String())
		}
	}
}
