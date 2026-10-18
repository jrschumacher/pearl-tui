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
	"strconv"
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
		"F-RAINBOW", "F-LEVELS", "F-BOLD", "F-EDGE", "F-EMPTY", "F-TOFU", "F-TAB"} {
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

// A "?" is ASCII, so F-ASCII alone could not see the sweep's fallback for a rune with no
// twin: 21.5°C drew 21.5?C. With --unicode (the unicode render of the same states) a
// "?" where the unicode frame draws something else is F-ASCII; a "?" the unicode frame
// also has (a help key, a question) is not.
func TestASCIIQuestionAgainstUnicode(t *testing.T) {
	dir := t.TempDir()
	uniDir := filepath.Join(dir, "uni")
	ascDir := filepath.Join(dir, "asc")
	for _, d := range []string{uniDir, ascDir} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write := func(d, name, text string) string {
		p := filepath.Join(d, name)
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	write(uniDir, "temp-20x2.ansi", "  cpu 61.5°C  ½ cup \n  ? keys  Zoë       \n")
	asc := write(ascDir, "temp-20x2.ansi", "  cpu 61.5?C  ? cup \n  ? keys  Zoe       \n")
	write(uniDir, "fine-20x2.ansi", "  cpu 61.5°C        \n  ? keys            \n")
	fine := write(ascDir, "fine-20x2.ansi", "  cpu 61.5 C        \n  ? keys            \n")
	lone := write(ascDir, "lone-20x2.ansi", "  what?             \n                    \n")
	hits := func(args ...string) []Finding {
		fs, _, _ := run(t, args...)
		var out []Finding
		for _, f := range fs {
			if f.Rule == "F-ASCII" {
				out = append(out, f)
			}
		}
		return out
	}
	if h := hits(asc, "--ascii"); len(h) != 0 {
		t.Errorf("without --unicode a \"?\" is ASCII: %+v", h)
	}
	h := hits(asc, "--ascii", "--unicode", uniDir)
	if len(h) != 1 || h[0].Sev != "P1" || h[0].Where != "r0c10" || !strings.Contains(h[0].Msg, "2 cells") ||
		!strings.Contains(h[0].Msg, "° U+00B0, ½ U+00BD") {
		t.Errorf("a \"?\" where the unicode frame has a unit: %+v", h)
	}
	if h := hits(asc, "--ascii", "--unicode", filepath.Join(uniDir, "temp-20x2.ansi")); len(h) != 1 {
		t.Errorf("--unicode naming one frame: %+v", h)
	}
	if h := hits(fine, lone, "--ascii", "--unicode", uniDir); len(h) != 0 {
		t.Errorf("a \"?\" the unicode frame has too, and a state it lacks: %+v", h)
	}
	var out, errb bytes.Buffer
	if code := Main([]string{asc, "--unicode", uniDir}, &out, &errb); code != 2 || !strings.Contains(errb.String(), "--ascii") {
		t.Errorf("--unicode without --ascii: exit %d, stderr %q", code, errb.String())
	}
	if code := Main([]string{asc, "--ascii", "--unicode", filepath.Join(dir, "missing")}, &out, &errb); code != 2 {
		t.Errorf("--unicode on a missing path: exit %d", code)
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

// F-BOLD is a ratio, so it needs a floor: a loading frame of a few labels (the recreate
// run's overview-loading: ~120 text cells, the wordmark, six bold labels and two footer
// keys) is 35% bold by arithmetic, not a wall of bold. Under boldFloor text cells it is
// not judged, as a splash isn't; at the floor the same ratio still fails.
func TestBoldSkipsSparseFrames(t *testing.T) {
	dir := t.TempDir()
	frame := func(name string, cells int) string {
		var b strings.Builder
		row, n := 0, 0
		for row < 24 {
			line := ""
			for x := 0; x < 80; x += 10 {
				if n < cells {
					word := "abcdefgh"
					if n%5 < 2 { // 40% bold
						word = "\x1b[1m" + word + "\x1b[22m"
					}
					line += word + "  "
					n += 8
				} else {
					line += "          "
				}
			}
			b.WriteString(line + "\n")
			row++
		}
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(b.String()), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	bolds := func(path string) int {
		fs, _, _ := run(t, path)
		n := 0
		for _, f := range fs {
			if f.Rule == "F-BOLD" {
				n++
			}
		}
		return n
	}
	if n := bolds(frame("loading-80x24.ansi", boldFloor-40)); n != 0 {
		t.Errorf("a sparse frame under the floor: %d F-BOLD, want 0", n)
	}
	if n := bolds(frame("list-80x24.ansi", boldFloor+40)); n != 1 {
		t.Errorf("a frame over the floor at 40%% bold: %d F-BOLD, want 1", n)
	}
}

// F-TRUNC on an overlay frame: the kit's Overlay clears the word a dialog cuts and puts
// an ellipsis where the kept text ends, so rows the dialog cuts end near its left edge,
// at one column when their words break alike. Those are the dialog's cut, not a column
// budgeted short, and don't count: beside a bordered dialog, and beside a dialog found
// by its fill (no edge the lint can find). The dialog's own rows still count: a
// palette's description column cut on 4 rows is reported, with no dialog edge at all
// and inside a bordered dialog beside a bordered sidebar. Where neither the dialog's
// edges nor its fill can be found and the cut rows run on past the ellipsis, the rule
// says it was not run rather than nothing.
func TestTruncatedBesideADialog(t *testing.T) {
	dir := t.TempDir()
	write := func(name string, rows []string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(strings.Join(rows, "\n")+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	page := func(edge func(y int) string, gap string) []string {
		var rows []string
		for y := range 8 {
			rows = append(rows, "  14:0"+strconv.Itoa(y)+"  circuit opened on…"+gap+edge(y)+" body text     ")
		}
		return rows
	}
	box := func(y int) string {
		switch y {
		case 0:
			return "╭──────────╮"
		case 7:
			return "╰──────────╯"
		}
		return "│  dialog  │"
	}
	bare := func(int) string { return "   dialog    " }
	filled := func(int) string { return "\x1b[48;2;60;60;70m   dialog    \x1b[0m" }
	truncs := func(p string) []Finding {
		fs, _, _ := run(t, p)
		var out []Finding
		for _, f := range fs {
			if f.Rule == "F-TRUNC" {
				out = append(out, f)
			}
		}
		return out
	}
	cut := func(fs []Finding) bool { return len(fs) == 1 && fs[0].Where != "frame" }
	notRun := func(fs []Finding) bool {
		return len(fs) == 1 && fs[0].Where == "frame" && strings.Contains(fs[0].Msg, "not run on this frame (no dialog found)")
	}
	if got := truncs(write("confirm-60x8.ansi", page(box, "  "))); len(got) != 0 {
		t.Errorf("an overlay frame, ellipses beside its bordered dialog: %+v, want none", got)
	}
	if got := truncs(write("services-60x8.ansi", page(box, "  "))); !cut(got) {
		t.Errorf("the same rows on a page with a box: %+v, want one F-TRUNC", got)
	}
	if got := truncs(write("help-61x8.ansi", page(box, " x "))); !cut(got) {
		t.Errorf("an overlay frame, text between the ellipses and the dialog: %+v, want one F-TRUNC", got)
	}
	if got := truncs(write("help-60x8.ansi", page(filled, "  "))); len(got) != 0 {
		t.Errorf("an overlay frame, ellipses beside a filled dialog with no edge: %+v, want none", got)
	}
	if got := truncs(write("palette-61x8.ansi", page(bare, "  "))); !notRun(got) {
		t.Errorf("no dialog found, cut rows running on: %+v, want the not-run line", got)
	}
	// the dialog's own column, with no dialog edge or fill to find
	pal := []string{"  header", "     Commands", "     > ro"}
	for _, r := range [][2]string{{"Roll back", "return to the previous healthy version"},
		{"Switch env", "prod, staging, dev or preview"}, {"Deploy", "ship the selected version to the env"},
		{"Pause", "stop the auto-refresh loop of the list"}} {
		pal = append(pal, fmt.Sprintf("     %-16s%s…", r[0], r[1][:25]))
	}
	pal = append(pal, "")
	for i := range pal {
		pal[i] = fmt.Sprintf("%-60s", pal[i])
	}
	if got := truncs(write("palette-60x8.ansi", pal)); !cut(got) || !strings.Contains(got[0].Msg, "4 rows") {
		t.Errorf("a palette's own description column cut on 4 rows: %+v, want one F-TRUNC naming 4 rows", got)
	}
	// the same column inside a bordered dialog beside a bordered sidebar
	boxed := []string{
		"┌──────────┐                                                ",
		"│ sidebar  │   ┌──────────────────────────────────┐         ",
		"└──────────┘   │ Roll back  return to the previou…│         ",
		"               │ Switch     prod, staging or dev,…│         ",
		"               │ Deploy     ship the selected ver…│         ",
		"               │ Pause      stop the auto-refresh…│         ",
		"               │ Tail logs  follow the service's …│         ",
		"               └──────────────────────────────────┘         ",
	}
	if got := truncs(write("palette-boxed-60x8.ansi", boxed)); !cut(got) {
		t.Errorf("a cut column inside a bordered dialog beside a bordered sidebar: %+v, want one F-TRUNC", got)
	}
}

// F-TRUNC: more than truncMax rows that end in an ellipsis at one column is a column
// budgeted narrower than its content (the recreate run's 32-cell sidebar beside a
// 34-character fixture title, 5 rows cut). Three cuts are long names; a row of dots
// under ASCII is a rule, not a cut; the theme's own glyphs.ellipsis counts too.
func TestTruncatedColumn(t *testing.T) {
	dir := t.TempDir()
	frame := func(name string, cut int, ellipsis string) string {
		var b strings.Builder
		for y := range 8 {
			side := "  short title         "
			if y < cut {
				side = "  a title that is lon" + ellipsis
			}
			b.WriteString(side + "  │ body text\n")
		}
		b.WriteString("  " + strings.Repeat(".", 20) + "  │ body text\n")
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(b.String()), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	theme := filepath.Join(dir, "theme.json")
	if err := os.WriteFile(theme, []byte(`{"roles": {"dark": {"bg": "#101010", "fg": "#E0E0E0"}}, "glyphs": {"ellipsis": "⋯"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	truncs := func(args ...string) []Finding {
		fs, _, _ := run(t, args...)
		var out []Finding
		for _, f := range fs {
			if f.Rule == "F-TRUNC" {
				out = append(out, f)
			}
		}
		return out
	}
	if got := truncs(frame("sidebar-34x9.ansi", 5, "…")); len(got) != 1 || got[0].Sev != "P2" || got[0].Where != "r0c21" || !strings.Contains(got[0].Msg, "5 rows") {
		t.Errorf("five cut rows: %+v, want one P2 F-TRUNC at r0c21 naming 5 rows", got)
	}
	if got := truncs(frame("few-34x9.ansi", truncMax, "…")); len(got) != 0 {
		t.Errorf("%d cut rows: %+v, want none", truncMax, got)
	}
	if got := truncs(frame("ascii-34x9.ansi", 0, "…"), "--ascii"); len(got) != 0 {
		t.Errorf("a dotted rule under ASCII: %+v, want none", got)
	}
	if got := truncs(frame("ascii-cut-36x9.ansi", 5, "..."), "--ascii"); len(got) != 1 {
		t.Errorf("five rows cut with ... under ASCII: %+v, want one", got)
	}
	if got := truncs(frame("own-34x9.ansi", 5, "⋯"), "--theme", theme); len(got) != 1 {
		t.Errorf("five rows cut with the theme's ⋯: %+v, want one", got)
	}
	// a table's last column headed command, path, file or url cuts by design (a process
	// list's command line); the message says to widen the column or say so in TUI.md
	table := func(name, header, after string) string {
		var b strings.Builder
		b.WriteString("  pid    " + header + strings.Repeat(" ", 16-len(header)) + after + "\n")
		b.WriteString("  " + strings.Repeat("─", 24) + "\n")
		for y := range 6 {
			b.WriteString(fmt.Sprintf("  %-5d  /usr/local/bin/s…%s\n", 100+y, after))
		}
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(b.String()), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	if got := truncs(table("procs-34x9.ansi", "command", "")); len(got) != 0 {
		t.Errorf("a last column headed command: %+v, want none", got)
	}
	if got := truncs(table("paths-34x9.ansi", "Paths", "")); len(got) != 0 {
		t.Errorf("a last column headed Paths: %+v, want none", got)
	}
	if got := truncs(table("titles-34x9.ansi", "title", "")); len(got) != 1 || !strings.Contains(got[0].Msg, "widen the column") || !strings.Contains(got[0].Msg, "§ Decisions") {
		t.Errorf("a last column headed title: %+v, want one that says widen it or say so in § Decisions", got)
	}
	asciiOf := func(p string) string { // the same frame as -ascii draws it: - rules, ... cuts
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		b = []byte(strings.NewReplacer("─", "-", "…", "...").Replace(string(b)))
		if err := os.WriteFile(p, b, 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	if got := truncs(asciiOf(table("ascii-procs-36x9.ansi", "command", "")), "--ascii"); len(got) != 0 {
		t.Errorf("a last column headed command under --ascii: %+v, want none", got)
	}
	if got := truncs(table("mid-40x9.ansi", "command", "  ok")); len(got) != 1 {
		t.Errorf("a command column with a column after it: %+v, want one", got)
	}
	// the header heads the column: the cell where the cut rows' text starts, on the row
	// above them. A sidebar heading or a hint that ends in "files" or "path" is not one.
	sidebar := func(name string, top ...string) string {
		var b strings.Builder
		for _, l := range top {
			b.WriteString(fmt.Sprintf("%-80s\n", l))
		}
		for _, title := range []string{"Add CSV import for bank stat", "Fix rounding in ledger expor", "Refactor the OFX parser into", "Write tests for the importer", "Port the CLI flags to cobra "} {
			b.WriteString(fmt.Sprintf("%-80s\n", fmt.Sprintf("  %-46s%s…", "body text here", title)))
		}
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(b.String()), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	if got := truncs(sidebar("recent-80x8.ansi", fmt.Sprintf("  %-46s%s", "chat", "Recent files"), "")); len(got) != 1 {
		t.Errorf("titles under a \"Recent files\" heading: %+v, want one", got)
	}
	if got := truncs(sidebar("hint-80x9.ansi", "  press y to Copy path", fmt.Sprintf("  %-46s%s", "chat", "Sessions"), "")); len(got) != 1 {
		t.Errorf("titles under \"Sessions\" with a \"Copy path\" hint above: %+v, want one", got)
	}
	if got := truncs(sidebar("section-80x9.ansi", fmt.Sprintf("  %-46s%s", "chat", "Files"), fmt.Sprintf("  %-46s%s", "body text here", "csv.go"), "", fmt.Sprintf("  %-46s%s", "body text here", "Sessions"))); len(got) != 1 {
		t.Errorf("titles under \"Sessions\" below a \"Files\" section and a blank: %+v, want one", got)
	}
	if got := truncs(sidebar("files-80x7.ansi", fmt.Sprintf("  %-46s%s", "chat", "Files (5)"))); len(got) != 0 {
		t.Errorf("a column headed \"Files (5)\": %+v, want none", got)
	}
}

// F-LEVELS isn't judged on a frame with no colour (NO_COLOR): it has one text level by
// construction, its levels are weight and glyphs. The same frame in colour still is.
func TestLevelsSkipsColourlessFrames(t *testing.T) {
	dir := t.TempDir()
	var plain, grey strings.Builder
	for range 12 {
		row := strings.Repeat("lorem ipsum ", 6) + "dolor si"
		plain.WriteString(row + "\n")
		grey.WriteString("\x1b[38;2;200;200;200m" + row + "\x1b[0m\n")
	}
	levels := func(name, text string) int {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		fs, _, _ := run(t, p)
		n := 0
		for _, f := range fs {
			if f.Rule == "F-LEVELS" {
				n++
			}
		}
		return n
	}
	if n := levels("plain-80x12.ansi", plain.String()); n != 0 {
		t.Errorf("no-colour frame: %d F-LEVELS, want 0", n)
	}
	if n := levels("basic-80x12.ansi", strings.ReplaceAll(grey.String(), "\x1b[38;2;200;200;200m", "\x1b[37m")); n != 0 {
		t.Errorf("16-colour frame: %d F-LEVELS, want 0 (palette check judges the theme's greys)", n)
	}
	if n := levels("grey-80x12.ansi", grey.String()); n != 1 {
		t.Errorf("one-grey colour frame: %d F-LEVELS, want 1", n)
	}
}

// contrastHits returns the F-CONTRAST findings of a run.
func contrastHits(t *testing.T, args ...string) []Finding {
	t.Helper()
	fs, _, _ := run(t, args...)
	var out []Finding
	for _, f := range fs {
		if f.Rule == "F-CONTRAST" {
			out = append(out, f)
		}
	}
	return out
}

// A "dialog": "rules" panel (two rule rows of one span around a filled panel, a title
// or a status set into either) and an ASCII +--+ box are dialogs: their inside is judged
// for contrast as a bordered box's is, and the scrim around them is not. testdata/overlays holds the kit's quire confirm
// (rules), quire help under --ascii (rules in -) and halyard confirm under --ascii
// (+--+); a "dialog": "none" palette (moss) still gets the P2 note, since it has no edge.
func TestOverlayDialogsAreFound(t *testing.T) {
	dir := t.TempDir()
	mutate := func(src, dst, old, new string) string {
		data, err := os.ReadFile(src)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), old) {
			t.Fatalf("%s: no %q to replace", src, old)
		}
		p := filepath.Join(dir, dst)
		if err := os.WriteFile(p, []byte(strings.Replace(string(data), old, new, 1)), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	// the kit's frames pass as they are, with no "checked only by eye" note
	for _, c := range []struct {
		file  string
		ascii bool
	}{{"rules-confirm-80x24.ansi", false}, {"rules-ascii-help-80x24.ansi", true}, {"box-ascii-confirm-80x24.ansi", true}} {
		args := []string{"testdata/overlays/" + c.file, "--theme", "testdata/themes/quire.json"}
		if c.ascii {
			args = append(args, "--ascii")
		}
		if hits := contrastHits(t, args...); len(hits) != 0 {
			t.Errorf("%s: %+v, want no F-CONTRAST", c.file, hits)
		}
	}
	if hits := contrastHits(t, "testdata/overlays/none-palette-80x24.ansi"); len(hits) != 1 || hits[0].Sev != "P2" || !strings.Contains(hits[0].Msg, "checked only by eye") {
		t.Errorf("a dialog: none palette: %+v, want the P2 note", hits)
	}
	// a low-contrast cell inside the rules panel fails; the same cell in the scrim doesn't
	// (quire: panel bg #22201C, page bg #17181B; #2A2A2E is 1.3:1 on either)
	inside := mutate("testdata/overlays/rules-confirm-80x24.ansi", "confirm-in-80x24.ansi",
		"\x1b[38;2;178;175;171mprod returns", "\x1b[38;2;42;42;46mprod returns")
	if hits := contrastHits(t, inside); len(hits) != 1 || hits[0].Sev != "P1" || hits[0].Where != "r7c13" {
		t.Errorf("low contrast inside a rules dialog: %+v, want one P1 at r7c13", hits)
	}
	outside := mutate("testdata/overlays/rules-confirm-80x24.ansi", "confirm-out-80x24.ansi",
		"\x1b[38;2;54;53;52mnow", "\x1b[38;2;20;21;24mnow")
	if hits := contrastHits(t, outside); len(hits) != 0 {
		t.Errorf("low contrast in the scrim beside a rules dialog: %+v, want none", hits)
	}
	// a rules panel whose rows carry a second background (terrarium's boxed keys on
	// bgSubtle inside the help panel) is still found: a planted 1.1:1 label inside fails
	if hits := contrastHits(t, "testdata/overlays/rules-boxed-help-80x24.ansi", "--theme", "testdata/themes/terrarium.json"); len(hits) != 0 {
		t.Errorf("terrarium help: %+v, want no F-CONTRAST", hits)
	}
	boxedKeys := mutate("testdata/overlays/rules-boxed-help-80x24.ansi", "help-boxed-80x24.ansi",
		"\x1b[38;2;167;177;174;48;2;25;35;32mdeploy", "\x1b[38;2;28;38;35;48;2;25;35;32mdeploy")
	if hits := contrastHits(t, boxedKeys, "--theme", "testdata/themes/terrarium.json"); len(hits) != 1 || hits[0].Sev != "P1" || !strings.Contains(hits[0].Msg, "6 text cells") {
		t.Errorf("low contrast inside a rules dialog with boxed keys: %+v, want one P1 on the 6 letters", hits)
	}
	// a rules help with a status set into its bottom edge (ui.DialogStatus, "1 of 2"):
	// the bottom edge is two runs around the status, and the one-run match missed it, so
	// a cold app's scrolled help got the note and its panel went unjudged. The frame is
	// the kit's (quire, DialogStatus over a page); a planted 1.3:1 label inside fails
	if hits := contrastHits(t, "testdata/overlays/rules-status-help-80x24.ansi", "--theme", "testdata/themes/quire.json"); len(hits) != 0 {
		t.Errorf("quire help with a status in its bottom edge: %+v, want no F-CONTRAST", hits)
	}
	status := mutate("testdata/overlays/rules-status-help-80x24.ansi", "help-status-80x24.ansi",
		"\x1b[38;2;135;130;125;22mquit", "\x1b[38;2;42;42;46;22mquit")
	if hits := contrastHits(t, status, "--theme", "testdata/themes/quire.json"); len(hits) != 1 || hits[0].Sev != "P1" {
		t.Errorf("low contrast inside a rules dialog with a status edge: %+v, want one P1", hits)
	}
	// at 256 colours the panel's fill isn't required (it can land on the page's colour),
	// and a page of two sections side by side over a full-width one reads as a titled top
	// rule over a bottom rule: the lint took those page rows for the dialog and judged the
	// scrim on them (a cold monitor's confirm: 97 cells at 1.92:1). The kit's quire
	// confirm over such a page, at -profile ansi256: the filled pair is the dialog, the
	// scrim isn't judged, and a planted faint label inside the real dialog still fails
	if hits := contrastHits(t, "testdata/overlays/rules-sections-256-confirm-80x24.ansi", "--theme", "testdata/themes/quire.json"); len(hits) != 0 {
		t.Errorf("quire 256-colour confirm over sectioned rows: %+v, want no F-CONTRAST", hits)
	}
	sections := mutate("testdata/overlays/rules-sections-256-confirm-80x24.ansi", "confirm-sections-80x24.ansi",
		"\x1b[48;5;234m \x1b[38;5;244mone row", "\x1b[48;5;234m \x1b[38;5;235mone row")
	if hits := contrastHits(t, sections, "--theme", "testdata/themes/quire.json"); len(hits) != 1 || hits[0].Sev != "P1" || !strings.HasPrefix(hits[0].Where, "r10") {
		t.Errorf("low contrast inside the 256-colour confirm: %+v, want one P1 on row 10", hits)
	}
	// a 2×2 grid of sections at 256 colours ("disk ···· 52%  network ····" under "cpu
	// ···· 52%  memory ····") is not a dialog with a status in its edge: the status sits
	// at the edge's right end, the run after it 3 cells at most. Read as one, the grid
	// paired as the dialog and hid moss's "dialog": "none" help note behind a P1 on the page
	if hits := contrastHits(t, "testdata/overlays/none-grid-256-help-120x40.ansi", "--theme", "testdata/themes/moss.json"); len(hits) != 1 || hits[0].Sev != "P2" || !strings.Contains(hits[0].Msg, "checked only by eye") {
		t.Errorf("moss help over a grid of sections at 256 colours: %+v, want the P2 note", hits)
	}
	// the same inside an ASCII +--+ box (halyard ascii confirm: box bg #142030)
	boxed := mutate("testdata/overlays/box-ascii-confirm-80x24.ansi", "confirm-box-80x24.ansi",
		"prod returns to the last healthy version.", "\x1b[38;2;24;34;48mprod returns to the last healthy version.")
	if hits := contrastHits(t, boxed, "--ascii"); len(hits) != 1 || hits[0].Sev != "P1" {
		t.Errorf("low contrast inside an ASCII box: %+v, want one P1", hits)
	}
}

// A synthetic rules dialog, built here so the shape is explicit: a title in the top rule,
// a bottom rule over the same columns, a filled panel between, a scrim around.
func TestRulesDialogShape(t *testing.T) {
	const (
		page  = "\x1b[48;2;22;22;26m"
		panel = "\x1b[48;2;36;34;30m"
		dim   = "\x1b[38;2;40;40;44m" // 1.3:1 on either background
		ink   = "\x1b[38;2;200;200;200m"
		reset = "\x1b[0m"
	)
	scrim := page + dim + "scrim text scrim text   " + reset
	var rows []string
	rows = append(rows, scrim+page+strings.Repeat(" ", 16)+reset) // 0
	rows = append(rows, scrim+panel+ink+"── Roll back? ──"+reset) // 1: top rule, title
	rows = append(rows, scrim+panel+ink+"  are you sure  "+reset) // 2
	rows = append(rows, scrim+panel+dim+"  faint hint    "+reset) // 3: low contrast inside
	rows = append(rows, scrim+panel+ink+"────────────────"+reset) // 4: bottom rule
	rows = append(rows, page+ink+strings.Repeat(" ", 40)+reset)   // 5: footer
	write := func(name string, rows []string) string {
		p := filepath.Join(t.TempDir(), name)
		if err := os.WriteFile(p, []byte(strings.Join(rows, "\n")+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	hits := contrastHits(t, write("confirm-40x6.ansi", rows))
	if len(hits) != 1 || hits[0].Sev != "P1" || hits[0].Where != "r3c26" {
		t.Errorf("rules dialog: %+v, want one P1 at r3c26 (the hint inside), none for the scrim", hits)
	}
	// without the panel fill it is two rules on a page, not a dialog: the note stays
	flat := make([]string, len(rows))
	for i, r := range rows {
		flat[i] = strings.ReplaceAll(r, panel, page)
	}
	if hits := contrastHits(t, write("confirm-flat-40x6.ansi", flat)); len(hits) != 1 || hits[0].Sev != "P2" {
		t.Errorf("two rules without a fill: %+v, want the P2 note", hits)
	}
	// a table header row (columns, 3+ spaces apart) and a footer rule are not a dialog
	rows[1] = scrim + panel + ink + "──  name    size" + reset
	if hits := contrastHits(t, write("confirm-table-40x6.ansi", rows)); len(hits) != 1 || hits[0].Sev != "P2" {
		t.Errorf("a header rule and a footer rule: %+v, want the P2 note", hits)
	}
}

// Status marks (● ◆ ▶ ○ ◐ •) carry meaning, so they are judged for contrast as text;
// box drawing, blocks, braille (a sparkline) and rule runs (────, ····) are decoration.
func TestContrastJudgesStatusMarks(t *testing.T) {
	dir := t.TempDir()
	const faint = "\x1b[38;2;42;42;46m" // 1.26:1 on #16161A
	write := func(name, text string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	for _, c := range []struct {
		name, text string
		hits       int
	}{
		{"marks-20x1.ansi", faint + "● ◆ ▶ ○ ◐ • ■       \x1b[0m", 1},
		{"bullet-20x1.ansi", "  " + faint + "•\x1b[0m item            ", 1},
		{"spark-20x1.ansi", faint + "⣀⣠⣤⣶⣿⣿⣶⣤⣀⣀⣀⣀⣀⣀      \x1b[0m", 0},
		{"blocks-20x1.ansi", faint + "▁▂▃▄▅▆▇█ ░▒▓ ─━│┃  \x1b[0m", 0},
		{"dots-20x1.ansi", faint + "moss ·········· prod\x1b[0m", 1},       // the words, not the dotted rule
		{"bar-20x1.ansi", faint + "| \x1b[0mselected row       ", 0},       // a lone | is a bar (what ▌ became)
		{"caret-20x1.ansi", "  > ro" + faint + "|\x1b[0m             ", 1}, // a | against a letter is a caret: text
		{"dotrule-20x1.ansi", faint + "····················\x1b[0m", 0},
	} {
		hits := contrastHits(t, write(c.name, c.text))
		if len(hits) != c.hits {
			t.Errorf("%s: %d F-CONTRAST, want %d: %+v", c.name, len(hits), c.hits, hits)
		}
	}
	// the words in dots-20x1 are the only cells counted: 8 letters, not the 10 dots
	if hits := contrastHits(t, filepath.Join(dir, "dots-20x1.ansi")); len(hits) == 1 && !strings.HasPrefix(hits[0].Msg, "8 text cells") {
		t.Errorf("dotted rule: %s", hits[0].Msg)
	}
}

// F-EMPTY and F-BOLD are not judged on a splash: the kit's ink splash at 120x40 (a
// wordmark on an empty page) lints with no warnings, and the same frame under another
// name still trips F-EMPTY.
func TestSplashIsNotEmpty(t *testing.T) {
	fs, stdout, code := run(t, "testdata/kit/ink-dark/splash-120x40.ansi", "--theme", "testdata/themes/ink.json")
	if code != 0 || len(fs) != 0 || !strings.Contains(stdout, "OK  0 blocking, 0 warnings") {
		t.Errorf("ink splash: exit %d, %d findings\n%s", code, len(fs), stdout)
	}
	data, err := os.ReadFile("testdata/kit/ink-dark/splash-120x40.ansi")
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "home-120x40.ansi")
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}
	fs, _, _ = run(t, p, "--theme", "testdata/themes/ink.json")
	if len(fs) != 1 || fs[0].Rule != "F-EMPTY" {
		t.Errorf("the same frame as home: %+v, want F-EMPTY", fs)
	}
}

// --max-warnings N fails the run (exit 1) when more than N warnings remain, names the
// limit on the verdict line, and rejects a value that is not a count (exit 2).
func TestMaxWarnings(t *testing.T) {
	const frame = "testdata/synthetic/all-bold.ansi" // F-BOLD and F-LEVELS: 2 warnings, 0 blocking
	_, stdout, code := run(t, frame, "--max-warnings", "2")
	if code != 0 || !strings.HasSuffix(stdout, "OK  0 blocking, 2 warnings\n") {
		t.Errorf("at the limit: exit %d\n%s", code, stdout)
	}
	_, stdout, code = run(t, frame, "--max-warnings", "1")
	if code != 1 || !strings.HasSuffix(stdout, "FAIL  0 blocking, 2 warnings (more than --max-warnings 1)\n") {
		t.Errorf("over the limit: exit %d\n%s", code, stdout)
	}
	var out, errb bytes.Buffer
	if code := Main([]string{frame, "--max-warnings", "two"}, &out, &errb); code != 2 || !strings.Contains(errb.String(), "--max-warnings") {
		t.Errorf("a bad count: exit %d, stderr %q", code, errb.String())
	}
}

// F-RAINBOW isn't judged on a 256-colour frame: downsampling scatters a gradient's steps
// over hues the design never chose, and the truecolour render is where hues are judged.
// The synthetic rainbow trips it in truecolour and not when its colours are indexed.
func TestRainbowSkipsIndexedFrames(t *testing.T) {
	data, err := os.ReadFile("testdata/synthetic/rainbow.ansi")
	if err != nil {
		t.Fatal(err)
	}
	// re-express each truecolour fg as the nearest xterm-256 index (a coarse map is enough:
	// the point is the SGR kind, 38;5;N, which marks the frame as indexed)
	re := regexp.MustCompile(`([34])8;2;(\d+);(\d+);(\d+)`)
	indexed := re.ReplaceAllStringFunc(string(data), func(m string) string {
		var k, r, g, b int
		fmt.Sscanf(m, "%d8;2;%d;%d;%d", &k, &r, &g, &b)
		return fmt.Sprintf("%d8;5;%d", k, 16+36*(r/51)+6*(g/51)+b/51)
	})
	rainbow := func(name, text string) int {
		p := filepath.Join(t.TempDir(), name)
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		fs, _, _ := run(t, p)
		n := 0
		for _, f := range fs {
			if f.Rule == "F-RAINBOW" {
				n++
			}
		}
		return n
	}
	if n := rainbow("rainbow-80x24.ansi", string(data)); n != 1 {
		t.Errorf("truecolour rainbow: %d F-RAINBOW, want 1", n)
	}
	if n := rainbow("rainbow-256-80x24.ansi", indexed); n != 0 {
		t.Errorf("the same frame in 256 colours: %d F-RAINBOW, want 0", n)
	}
}

// A rule drawn in ASCII twins is decoration the way its glyph is: a rules dialog's
// titled top edge "-- Commands -----" (the two-cell lead before the title, which a
// 3+ run rule missed), a "======-----" progress bar and a tab underline, in the
// dialog's edge colour at 2.29:1 on the panel, are no F-CONTRAST; the same cells as
// letters are. A lone "--" in prose, with no rule run of the same ink in the row, is
// still text. welp darkened its overlay to get the lead past the 2.3 text floor.
func TestContrastExemptsASCIIRuleTwins(t *testing.T) {
	dir := t.TempDir()
	const edge = "\x1b[48;2;46;39;32m\x1b[38;2;102;93;80m" // #665D50 on #2E2720: 2.27:1
	const ink = "\x1b[48;2;46;39;32m\x1b[38;2;207;205;202m"
	write := func(name, text string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	for _, c := range []struct {
		name, text string
		hits       int
	}{
		{"lead-30x1.ansi", edge + "--" + ink + " Commands " + edge + "------------------\x1b[0m", 0},
		{"letters-30x1.ansi", edge + "ab" + ink + " Commands " + edge + "cdefghijklmnopqrst\x1b[0m", 1},
		{"progress-30x1.ansi", edge + "=============-----------------\x1b[0m", 0},
		{"underline-30x1.ansi", edge + "  ---------                   \x1b[0m", 0},
		{"dash-30x1.ansi", edge + "--" + ink + " one, then " + edge + "--" + ink + " two      \x1b[0m", 1},
	} {
		hits := contrastHits(t, write(c.name, c.text), "--ascii")
		if len(hits) != c.hits {
			t.Errorf("%s: %d F-CONTRAST, want %d: %+v", c.name, len(hits), c.hits, hits)
		}
	}
	if hits := contrastHits(t, filepath.Join(dir, "dash-30x1.ansi"), "--ascii"); len(hits) == 1 && !strings.HasPrefix(hits[0].Msg, "4 text cells") {
		t.Errorf("prose dashes: %s", hits[0].Msg)
	}
}
