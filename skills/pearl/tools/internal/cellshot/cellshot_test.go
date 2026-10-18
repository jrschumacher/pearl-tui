package cellshot

import (
	"bytes"
	"flag"
	"image"
	"image/draw"
	"image/gif"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the Go golden PNGs in testdata")

func render(t *testing.T, path string, o Options) *image.RGBA {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	rows, _ := Parse(strings.TrimSuffix(string(data), "\n"))
	img, _ := Render(rows, o)
	return img
}

func readPNG(t *testing.T, path string) image.Image {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	return img
}

// diff compares got with want and returns the number of pixels whose channels differ
// by more than tol, and the largest channel difference.
func diff(got *image.RGBA, want image.Image, tol int) (bad, worst int) {
	b := want.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := want.At(x, y).RGBA()
			p := got.At(x-b.Min.X, y-b.Min.Y)
			r2, g2, b2, _ := p.RGBA()
			d := max(absd(r>>8, r2>>8), absd(g>>8, g2>>8), absd(bl>>8, b2>>8))
			worst = max(worst, d)
			if d > tol {
				bad++
			}
		}
	}
	return bad, worst
}

func absd(a, b uint32) int {
	if a > b {
		return int(a - b)
	}
	return int(b - a)
}

var dark = Options{FG: RGB{0xd8, 0xd8, 0xd8}, BG: RGB{0x16, 0x16, 0x1a}}
var light = Options{FG: RGB{0x22, 0x22, 0x22}, BG: RGB{0xfa, 0xfa, 0xfa}}

func with(o Options, w, h int, grid bool) Options {
	o.CellW, o.CellH, o.Grid = w, h, grid
	return o
}

// TestGeometryMatchesPython holds the port to its contract: every geometric glyph is
// pixel-identical to cellshot.py. The references were rendered by cellshot.py (Pillow
// 12.3) from testdata/geometry.ansi, which has no font glyphs at all.
func TestGeometryMatchesPython(t *testing.T) {
	for _, tc := range []struct {
		ref string
		o   Options
	}{
		{"geometry-py-12x24.png", with(dark, 12, 24, false)},
		{"geometry-py-9x18.png", with(light, 9, 18, false)},
		{"geometry-py-16x32.png", with(dark, 16, 32, true)},
	} {
		got := render(t, "testdata/geometry.ansi", tc.o)
		want := readPNG(t, filepath.Join("testdata", tc.ref))
		if got.Bounds().Size() != want.Bounds().Size() {
			t.Fatalf("%s: size %v, Python %v", tc.ref, got.Bounds().Size(), want.Bounds().Size())
		}
		if bad, worst := diff(got, want, 0); bad > 0 {
			t.Errorf("%s: %d pixels differ from Python (max channel diff %d)", tc.ref, bad, worst)
		}
	}
}

// TestGoldens re-renders whole frames, text included, against the Go goldens. Text is
// rasterised in float32, so a different CPU may move an anti-aliased edge by a level
// or two; anything beyond that is a regression. Run with -update to accept a change.
func TestGoldens(t *testing.T) {
	for _, tc := range []struct {
		in, golden string
		o          Options
	}{
		{"synthetic.ansi", "synthetic-12x24.png", with(dark, 12, 24, false)},
		{"halyard-confirm.ansi", "halyard-confirm-8x16.png", with(dark, 8, 16, false)},
		{"lumen-light-services.ansi", "lumen-light-services-8x16.png", with(light, 8, 16, false)},
	} {
		got := render(t, filepath.Join("testdata", tc.in), tc.o)
		path := filepath.Join("testdata", tc.golden)
		if *update {
			var buf bytes.Buffer
			if err := (&png.Encoder{CompressionLevel: png.BestCompression}).Encode(&buf, got); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		want := readPNG(t, path)
		if got.Bounds().Size() != want.Bounds().Size() {
			t.Fatalf("%s: size %v, golden %v", tc.golden, got.Bounds().Size(), want.Bounds().Size())
		}
		if bad, worst := diff(got, want, 2); bad > 0 {
			t.Errorf("%s: %d pixels differ from the golden (max channel diff %d); -update to accept", tc.golden, bad, worst)
		}
	}
}

func TestParseWidthsFollowTheKit(t *testing.T) {
	rows, warns := Parse("日本\x1b[1mab\x1b[0mé🚀\tx\n\x1b[2Jz")
	if len(rows) != 2 {
		t.Fatalf("rows = %d", len(rows))
	}
	// 日 日' 本 本' a b é 🚀 🚀' then a tab to column 16, x
	if got := len(rows[0]); got != 17 {
		t.Errorf("row 0 width = %d, want 17", got)
	}
	if !rows[0][0].Wide || !rows[0][1].Cont || rows[0][6].Text != "é" || !rows[0][4].Bold || rows[0][6].Bold {
		t.Errorf("cells: %+v", rows[0][:8])
	}
	want := []string{
		"emoji present (rendering varies by terminal and font; usually a design smell)",
		"non-SGR CSI 'J' ignored (cursor movement? frame may not be a flat render)",
	}
	if strings.Join(warns, "|") != strings.Join(want, "|") {
		t.Errorf("warns = %q", warns)
	}
}

func TestSGR(t *testing.T) {
	cases := []struct {
		seq  string
		want Style
	}{
		{"38;2;1;2;3;48;5;196", Style{FG: rgb(1, 2, 3), BG: rgb(255, 0, 0)}},
		{"38:2::10:20:30;4:3", Style{FG: rgb(10, 20, 30), Under: 3}},
		{"91;104;1;2;3;7;9;21", Style{FG: Color{basic16[9], true}, BG: Color{basic16[12], true}, Bold: true, Faint: true, Italic: true, Reverse: true, Strike: true, Under: 2}},
		{"1;2;22", Style{}},
		{"38;5;244", Style{FG: rgb(128, 128, 128)}},
		{"38;5;16;39", Style{}},
		{"", Style{}},
	}
	for _, c := range cases {
		var st Style
		st.Bold = c.seq == "" // a bare CSI m resets
		applySGR(c.seq, &st)
		if st != c.want {
			t.Errorf("SGR %q = %+v, want %+v", c.seq, st, c.want)
		}
	}
}

func TestEscapes(t *testing.T) {
	for _, c := range []struct {
		in   string
		text string
		warn int
	}{
		{"a\x1b]8;;http://x\x07b\x1b]8;;\x1b\\c", "abc", 0}, // OSC 8 hyperlinks
		{"a\x1b[?25lb\x1b[?25hc", "abc", 0},                 // cursor show/hide
		{"a\x1b[ qb", "ab", 1},                              // DECSCUSR, an intermediate byte
		{"a\x1b[>4;2mb", "ab", 1},                           // XTMODKEYS is not SGR
		{"a\x1bPq#0\x1b\\b", "ab", 0},                       // DCS
		{"a\x07b", "ab", 1},                                 // stray control character
	} {
		rows, warns := Parse(c.in)
		var b strings.Builder
		for _, cell := range rows[0] {
			b.WriteString(cell.Text)
		}
		if b.String() != c.text || len(warns) != c.warn {
			t.Errorf("%q: text %q warns %q", c.in, b.String(), warns)
		}
	}
}

func TestReport(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "frame.v2.ansi")
	if err := os.WriteFile(in, []byte("ab\nc\n日\U0001F680\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errb bytes.Buffer
	code := Main([]string{in, "--cols", "5", "--sheet", filepath.Join(dir, "s.png")}, nil, &out, &errb, "usage")
	want := in + `: 4x3 cells
  - emoji present (rendering varies by terminal and font; usually a design smell)
  - rows of unequal width: 2 rows short of 4 (first: row 0 = 2)
  - width 4 != expected 5
  - no font has '日' U+65E5: drawn as tofu (it may be tofu for users too)
  - no font has '🚀' U+1F680: drawn as tofu (it may be tofu for users too)
wrote ` + filepath.Join(dir, "s.png") + "\n"
	if code != 1 || out.String() != want {
		t.Errorf("exit %d, stderr %q, report:\n%s\nwant:\n%s", code, errb.String(), out.String(), want)
	}
	if stem(in) != "frame.v2" || stem("-") != "-" || stem("/x/.hidden") != ".hidden" {
		t.Errorf("stem: %q %q %q", stem(in), stem("-"), stem("/x/.hidden"))
	}
}

func TestGIFKeepsFrameColours(t *testing.T) {
	a := render(t, "testdata/geometry.ansi", with(dark, 8, 16, false))
	b := render(t, "testdata/geometry.ansi", with(light, 8, 16, false))
	var buf bytes.Buffer
	if err := WriteGIF(&buf, []*image.RGBA{a, b}, 80); err != nil {
		t.Fatal(err)
	}
	out, err := gifDecode(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Image) != 2 || out.Delay[0] != 8 {
		t.Fatalf("frames %d delay %v", len(out.Image), out.Delay)
	}
	// Geometry has few enough colours for an exact palette: nothing may change.
	for i, want := range []*image.RGBA{a, b} {
		if bad, worst := diff(rgbaOf(out.Image[i]), want, 0); bad > 0 {
			t.Errorf("frame %d: %d pixels changed (max %d)", i, bad, worst)
		}
	}
}

func gifDecode(b []byte) (*gif.GIF, error) { return gif.DecodeAll(bytes.NewReader(b)) }

func rgbaOf(m image.Image) *image.RGBA {
	out := image.NewRGBA(m.Bounds())
	draw.Draw(out, out.Rect, m, m.Bounds().Min, draw.Src)
	return out
}
