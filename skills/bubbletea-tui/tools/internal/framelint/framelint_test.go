package framelint

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

// The goldens under testdata are tuilint.py's own JSON for the same frames
// and flags; the themes are copies so a kit palette change can't stale them.

func readFindings(t *testing.T, path string) []Finding {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var f []Finding
	if err := json.Unmarshal(data, &f); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return f
}

func exitFor(fs []Finding) int {
	for _, f := range fs {
		if f.Sev == "P0" || f.Sev == "P1" {
			return 1
		}
	}
	return 0
}

// run calls Main the way the CLI does and returns its findings, stdout and status.
func run(t *testing.T, args ...string) ([]Finding, string, int) {
	t.Helper()
	out := filepath.Join(t.TempDir(), "out.json")
	var stdout, stderr bytes.Buffer
	code := Main(append(args, "--json", out), &stdout, &stderr)
	if stderr.Len() > 0 {
		t.Fatalf("stderr: %s", stderr.String())
	}
	return readFindings(t, out), stdout.String(), code
}

func diffFindings(t *testing.T, label string, want, got []Finding) {
	t.Helper()
	if reflect.DeepEqual(want, got) {
		return
	}
	n := max(len(want), len(got))
	for i := 0; i < n; i++ {
		var w, g Finding
		if i < len(want) {
			w = want[i]
		}
		if i < len(got) {
			g = got[i]
		}
		if w != g {
			t.Errorf("%s #%d\n py %+v\n go %+v", label, i, w, g)
		}
	}
}

func TestSyntheticMatchesPython(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{"synthetic.sized", []string{"testdata/synthetic", "--cols", "80", "--rows", "24", "--theme", "testdata/themes/halyard.json"}},
		{"synthetic.plain", []string{"testdata/synthetic"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			want := readFindings(t, "testdata/"+c.name+".golden.json")
			got, _, code := run(t, c.args...)
			diffFindings(t, c.name, want, got)
			if code != exitFor(want) {
				t.Errorf("exit %d, want %d", code, exitFor(want))
			}
		})
	}
}

// Every frame rule fires somewhere in the synthetic set.
func TestSyntheticCoversEveryRule(t *testing.T) {
	seen := map[string]bool{}
	for _, f := range readFindings(t, "testdata/synthetic.sized.golden.json") {
		seen[f.Rule] = true
	}
	for _, r := range []string{"F-WIDTH", "F-EMOJI", "F-DEFAULT", "F-OFFPAL", "F-CONTRAST", "F-BOXES",
		"F-RAINBOW", "F-LEVELS", "F-BOLD", "F-EDGE", "F-EMPTY", "F-TOFU"} {
		if !seen[r] {
			t.Errorf("no synthetic frame trips %s", r)
		}
	}
}

// The printed report is byte-for-byte tuilint.py's.
func TestStdoutMatchesPython(t *testing.T) {
	want, err := os.ReadFile("testdata/synthetic.sized.golden.txt")
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	Main([]string{"testdata/synthetic", "--cols", "80", "--rows", "24", "--theme", "testdata/themes/halyard.json"}, &stdout, &stderr)
	if stdout.String() != string(want) {
		t.Errorf("stdout differs\n--- py\n%s\n--- go\n%s", want, stdout.String())
	}
}

func TestKitRendersMatchPython(t *testing.T) {
	goldens, _ := filepath.Glob("testdata/kit/*.golden.json")
	if len(goldens) == 0 {
		t.Fatal("no kit goldens")
	}
	for _, g := range goldens {
		dir := strings.TrimSuffix(g, ".golden.json")
		name := filepath.Base(dir)
		theme := strings.SplitN(name, "-", 2)[0]
		t.Run(name, func(t *testing.T) {
			want := readFindings(t, g)
			got, _, code := run(t, dir, "--theme", "testdata/themes/"+theme+".json")
			diffFindings(t, name, want, got)
			if code != exitFor(want) {
				t.Errorf("exit %d, want %d", code, exitFor(want))
			}
		})
	}
}

// Deliberate deviation: rows are measured with the kit's grapheme widths
// (charmbracelet/x/ansi), not tuilint.py's per-codepoint Unicode 15.1 widths.
// ZWJ sequences, a VS16 after a wide emoji, flags, format characters, nonspacing
// marks with combining class 0, C0 controls and the Unicode 16 wide Yijing
// symbols are all measured as the kit measures them, so these rows, padded to
// the kit's widths, are not torn. Everything else matches Python.
func TestDeviationsOnlyDropPythonWidthMisreads(t *testing.T) {
	for _, mode := range []struct {
		name string
		args []string
	}{
		{"deviations.sized", []string{"testdata/deviations", "--cols", "80", "--rows", "24", "--theme", "testdata/themes/halyard.json"}},
		{"deviations.plain", []string{"testdata/deviations"}},
	} {
		t.Run(mode.name, func(t *testing.T) {
			py := readFindings(t, "testdata/"+mode.name+".golden.json")
			var want []Finding
			dropped := map[string]bool{}
			for _, f := range py {
				if f.Rule == "F-WIDTH" {
					dropped[f.File] = true
					continue
				}
				want = append(want, f)
			}
			for _, f := range []string{"control.ansi", "unicode16.ansi", "zwj-emoji.ansi"} {
				if !dropped[f] {
					t.Errorf("tuilint.py no longer misreads %s: move it to synthetic/", f)
				}
			}
			got, _, _ := run(t, mode.args...)
			diffFindings(t, mode.name, want, got)
		})
	}
}

// Deliberate deviation: F-DEFAULT also knows Huh v2's signature defaults, which
// tuilint.py never listed: an agent restyled a Huh wizard, left the Charm theme and
// default spinner in place, and the frames passed. testdata/huh holds real Huh
// v2.0.3 renders (Charm theme form, dark and light, and the default spinner); each
// must trip F-DEFAULT on exactly the Huh colours. No kit theme may sit within the
// rule's ±3 of them, so the addition cannot flag a brand the kit ships.
func TestHuhDefaults(t *testing.T) {
	fs, _, code := run(t, "testdata/huh")
	var got []string
	for _, f := range fs {
		if f.Rule == "F-DEFAULT" {
			label, _, _ := strings.Cut(strings.TrimPrefix(f.Msg, "library-default colour "), " (")
			got = append(got, f.File+" "+f.Where+" "+label)
		}
	}
	want := []string{
		"huh-form-dark.ansi r0c2 #7571F9", "huh-form-dark.ansi r1c2 #F780E2",
		"huh-form-light.ansi r0c2 #5A56E0", "huh-form-light.ansi r1c2 #F780E2",
		"huh-spinner.ansi r0c0 #F780E2",
	}
	if !reflect.DeepEqual(got, want) || code != 1 {
		t.Errorf("F-DEFAULT on Huh renders (exit %d):\n got %q\nwant %q", code, got, want)
	}
	themes, _ := filepath.Glob("../../../kit/themes/*.json")
	if len(themes) == 0 {
		t.Fatal("no kit themes")
	}
	hexes := regexp.MustCompile(`#[0-9A-Fa-f]{6}\b`)
	for _, th := range themes {
		b, err := os.ReadFile(th)
		if err != nil {
			t.Fatal(err)
		}
		for _, h := range hexes.FindAllString(string(b), -1) {
			c := hexc(h)
			for _, d := range defaultColours {
				if !strings.Contains(d.label, "huh") {
					continue
				}
				dist := 0
				for k := 0; k < 3; k++ {
					dist = max(dist, abs(c[k]-d.c[k]))
				}
				if dist <= 3 {
					t.Errorf("%s: %s is within 3 of %s", filepath.Base(th), h, d.label)
				}
			}
		}
	}
}

// Deliberate deviation: tuilint.py blends each pair of roles in one direction,
// chosen by its set's iteration order, and the two directions round apart in
// a few buckets. Go blends both ways, so it holds every bucket Python does
// (it never flags a colour Python accepts) plus a handful more.
func TestPaletteCoversPython(t *testing.T) {
	golds, _ := filepath.Glob("testdata/palettes/*.bits.gz")
	if len(golds) == 0 {
		t.Fatal("no palette goldens")
	}
	for _, g := range golds {
		name := strings.TrimSuffix(filepath.Base(g), ".bits.gz")
		f, err := os.Open(g)
		if err != nil {
			t.Fatal(err)
		}
		zr, err := gzip.NewReader(f)
		if err != nil {
			t.Fatal(err)
		}
		bits, err := io.ReadAll(zr)
		f.Close()
		if err != nil {
			t.Fatal(err)
		}
		p, err := LoadPalette("testdata/themes/" + name + ".json")
		if err != nil {
			t.Fatal(err)
		}
		missing, extra, total := 0, 0, 0
		for i, on := range p.set {
			py := bits[i/8]&(1<<(i%8)) != 0
			if py {
				total++
			}
			switch {
			case py && !on:
				missing++
			case on && !py:
				extra++
			}
		}
		t.Logf("%s: %d buckets, %d beyond python's", name, total, extra)
		if missing > 0 || extra > 8 {
			t.Errorf("%s: %d python buckets missing, %d extra (of %d)", name, missing, extra, total)
		}
	}
}

// pyEmoji is cellshot.py's is_emoji over U+2600–27BF (East Asian Wide, Unicode 15.1).
const pyEmoji = "2614 2615 2648 2649 264A 264B 264C 264D 264E 264F 2650 2651 2652 2653 267F 2693 26A1 26AA 26AB " +
	"26BD 26BE 26C4 26C5 26CE 26D4 26EA 26F2 26F3 26F5 26FA 26FD 2705 270A 270B 2728 274C 274E 2753 2754 2755 " +
	"2757 2795 2796 2797 27B0 27BF"

func TestEmojiOnlyWideSymbols(t *testing.T) {
	want := map[rune]bool{}
	for _, h := range strings.Fields(pyEmoji) {
		var r rune
		fmt.Sscanf(h, "%X", &r)
		want[r] = true
	}
	for r := rune(0x2600); r <= 0x27BF; r++ {
		if isEmoji(r) != want[r] {
			t.Errorf("U+%04X %q: isEmoji %v, python %v", r, r, isEmoji(r), want[r])
		}
	}
	for _, r := range "✓✔✗★☆⚠❯➜" {
		if isEmoji(r) {
			t.Errorf("%q is a text glyph", r)
		}
	}
	if !isEmoji('🚀') {
		t.Error("🚀 is an emoji")
	}
}

func TestParseWidthsAgreeWithKit(t *testing.T) {
	for _, c := range []struct {
		s string
		w int
	}{
		{"abc", 3}, {"日本", 4}, {"👨‍👩‍👧", 2}, {"👍️", 2}, {"❤️", 2}, {"🇳🇴", 2},
		{"café", 4}, {"soft­hyphen", 10}, {"☰", 2}, {"a\x07b", 2}, {"\x1b[1;38;2;1;2;3mx\x1b[0m", 1},
		{"\x1b]8;;http://x\x07link\x1b]8;;\x1b\\", 4}, {"\tx", 9},
	} {
		f := Parse(c.s)
		if len(f.Rows) != 1 || len(f.Rows[0]) != c.w {
			n := -1
			if len(f.Rows) == 1 {
				n = len(f.Rows[0])
			}
			t.Errorf("%q: %d cells, want %d", c.s, n, c.w)
		}
	}
}

func TestParseStyles(t *testing.T) {
	f := Parse("\x1b[1;3;4;7;38;5;196;48:2::1:2:3mA\x1b[22;23;24;27;39mB\x1b[49;91mC")
	a, b, c := f.Rows[0][0], f.Rows[0][1], f.Rows[0][2]
	if !a.Bold || !a.Italic || a.Underline != 1 || !a.Reverse || *a.FG != (RGB{255, 0, 0}) || *a.BG != (RGB{1, 2, 3}) {
		t.Errorf("A: %+v", a)
	}
	if b.Bold || b.Italic || b.Underline != 0 || b.Reverse || b.FG != nil || *b.BG != (RGB{1, 2, 3}) {
		t.Errorf("B: %+v", b)
	}
	if c.BG != nil || *c.FG != (RGB{241, 76, 76}) {
		t.Errorf("C: %+v", c)
	}
}

func TestFlags(t *testing.T) {
	var out, errb bytes.Buffer
	if code := Main([]string{"--help"}, &out, &errb); code != 0 || !strings.Contains(out.String(), "usage") {
		t.Errorf("--help: %d %q", code, out.String())
	}
	for _, args := range [][]string{{"--nope", "x"}, {"--cols"}, {"--cols", "x", "a.ansi"}, {"--bg", "red"},
		{"--theme", "testdata/missing.json"}, {"testdata/missing.ansi"}} {
		out.Reset()
		errb.Reset()
		if code := Main(args, &out, &errb); code != 2 || errb.Len() == 0 {
			t.Errorf("%v: exit %d, stderr %q", args, code, errb.String())
		}
	}
	_, stdout, code := run(t, "testdata/synthetic/baseline.ansi", "--bg", "#fafafa")
	if code != 0 || !strings.HasPrefix(stdout, "linted 1 frame(s)\n") {
		t.Errorf("baseline: exit %d\n%s", code, stdout)
	}
}

// Deliberate deviation from tuilint.py: without --cols/--rows a "-<W>x<H>.ansi" name
// sets the frame's size, so an -out directory of mixed sizes lints without flags and a
// frame whose name lies about its size fails. The flags still win over the name.
func TestSizeFromName(t *testing.T) {
	dir := t.TempDir()
	cp := func(src, dst string) {
		data, err := os.ReadFile(src)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, dst), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	const kit = "testdata/kit/halyard-dark/"
	cp(kit+"services-80x24.ansi", "services-80x24.ansi") // name and size agree
	cp(kit+"toosmall-50x14.ansi", "toosmall-50x14.ansi") // a different size, same dir
	cp(kit+"services-80x24.ansi", "liar-120x40.ansi")    // 80x24 named 120x40
	cp(kit+"services-80x24.ansi", "plain.ansi")          // no suffix: widest row, any height
	width := func(args ...string) map[string]int {
		fs, _, _ := run(t, args...)
		n := map[string]int{}
		for _, f := range fs {
			if f.Rule == "F-WIDTH" {
				n[f.File]++
			}
		}
		return n
	}
	if got := width(dir); !reflect.DeepEqual(got, map[string]int{"liar-120x40.ansi": 2}) {
		t.Errorf("from names: F-WIDTH per file %v, want only liar-120x40.ansi (row and frame)", got)
	}
	if got := width(dir, "--cols", "80", "--rows", "24"); !reflect.DeepEqual(got, map[string]int{"toosmall-50x14.ansi": 2}) {
		t.Errorf("flags win: F-WIDTH per file %v, want only toosmall-50x14.ansi", got)
	}
	if w, h, ok := sizeFromName("frames/a-b-120x40.ansi"); !ok || w != 120 || h != 40 {
		t.Errorf("sizeFromName = %d %d %v", w, h, ok)
	}
	for _, p := range []string{"x.ansi", "80x24.ansi", "x-80x24.txt", "x-0x24.ansi"} {
		if _, _, ok := sizeFromName(p); ok {
			t.Errorf("sizeFromName(%q) matched", p)
		}
	}
}

// TestLiveParity renders every kit theme (dark, light, ascii; 80x24 and
// 120x40) and holds Go to tuilint.py on all of it. It needs python3 with
// Pillow, so it runs only with TUILINT_PARITY=1.
func TestLiveParity(t *testing.T) {
	if os.Getenv("TUILINT_PARITY") == "" {
		t.Skip("set TUILINT_PARITY=1 (needs python3 + Pillow)")
	}
	kit, _ := filepath.Abs("../../../kit")
	script, _ := filepath.Abs("../../../scripts/tuilint.py")
	tmp := t.TempDir()
	show := filepath.Join(tmp, "showcase")
	if out, err := exec.Command("go", "-C", kit, "build", "-o", show, "./cmd/showcase").CombinedOutput(); err != nil {
		t.Fatalf("build showcase: %v\n%s", err, out)
	}
	themes, _ := filepath.Glob(filepath.Join(kit, "themes", "*.json"))
	for _, theme := range themes {
		for _, v := range []string{"dark", "light", "ascii"} {
			name := strings.TrimSuffix(filepath.Base(theme), ".json") + "-" + v
			dir := filepath.Join(tmp, name)
			args := []string{"-theme", theme, "-out", dir, "-sizes", "80x24,120x40"}
			if v != "dark" {
				args = append(args, "-"+v)
			}
			if out, err := exec.Command(show, args...).CombinedOutput(); err != nil {
				t.Fatalf("%s: %v\n%s", name, err, out)
			}
			pyJSON := filepath.Join(tmp, name+".json")
			py := exec.Command("python3", script, "frames", dir, "--theme", theme, "--json", pyJSON)
			pyOut, _ := py.Output()
			want := readFindings(t, pyJSON)
			got, goOut, code := run(t, dir, "--theme", theme)
			diffFindings(t, name, want, got)
			if goOut != string(pyOut) || code != py.ProcessState.ExitCode() {
				t.Errorf("%s: stdout or exit differ (go %d, py %d)", name, code, py.ProcessState.ExitCode())
			}
		}
	}
}

// F-ASCII: with --ascii, any cell outside ASCII fails, once per frame; without the flag
// the rule doesn't run, so the Python-era goldens are untouched.
func TestASCIIRule(t *testing.T) {
	dir := t.TempDir()
	write := func(name, text string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	clean := write("clean-20x2.ansi", "  > inbox  * new    \n  ... more           \n")
	leaky := write("leaky-20x2.ansi", "  > inbox  ◐ read   \n  ... more  ◐ ●      \n")
	count := func(args ...string) (int, []Finding) {
		fs, _, _ := run(t, args...)
		var hits []Finding
		for _, f := range fs {
			if f.Rule == "F-ASCII" {
				hits = append(hits, f)
			}
		}
		return len(hits), hits
	}
	if n, _ := count(clean, leaky); n != 0 {
		t.Errorf("without --ascii: %d F-ASCII findings, want 0", n)
	}
	if n, _ := count(clean, "--ascii"); n != 0 {
		t.Errorf("an ASCII frame: %d F-ASCII findings, want 0", n)
	}
	n, hits := count(leaky, "--ascii")
	if n != 1 || hits[0].Sev != "P1" || hits[0].Where != "r0c11" || !strings.Contains(hits[0].Msg, "3 cells") ||
		!strings.Contains(hits[0].Msg, "◐ U+25D0, ● U+25CF") {
		t.Errorf("a leaky frame: %+v", hits)
	}
	if _, _, code := run(t, leaky, "--ascii"); code != 1 {
		t.Errorf("a leaky ASCII frame exits %d, want 1", code)
	}
}

// A 16-colour frame (-profile ansi) holds slots of the terminal's own palette: it is
// not judged for off-palette colours or for contrast under an assumed palette, only
// for text drawn in the same slot as its background, which no palette can show.
func TestBasicFrames(t *testing.T) {
	dir := t.TempDir()
	write := func(name, text string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	pad := strings.Repeat(" ", 12)
	// bright cyan on blue: 2.08:1 under cellshot's palette, readable on most terminals
	ok := write("ok-20x2.ansi", "\x1b[96;44m  header bar        \x1b[0m\n\x1b[31m  error text\x1b[0m"+pad[:8]+"\n")
	same := write("same-20x2.ansi", "\x1b[34;44m  hidden text       \x1b[0m\n\x1b[7;31;41m  reversed, hidden  \x1b[0m\n")
	rules := func(path string) map[string]int {
		fs, _, _ := run(t, path, "--theme", "testdata/themes/halyard.json")
		n := map[string]int{}
		for _, f := range fs {
			n[f.Rule]++
		}
		return n
	}
	if got := rules(ok); got["F-OFFPAL"] != 0 || got["F-CONTRAST"] != 0 {
		t.Errorf("readable 16-colour frame: %v, want no F-OFFPAL or F-CONTRAST", got)
	}
	if got := rules(same); got["F-CONTRAST"] != 1 {
		t.Errorf("text in its background's slot: %v, want one F-CONTRAST", got)
	}
}
