package oklch

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"testing"
)

// The goldens are palette.py's own output (cut from larger sweeps: 120,000 colours for
// xterm256 and 400,000 OKLCH triples for from_lch matched with no difference).

func lines(t *testing.T, path string) [][]string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var out [][]string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if l := sc.Text(); l != "" && !strings.HasPrefix(l, "# ") { // "# " is a comment, "#0A0B0C" data
			out = append(out, strings.Fields(l))
		}
	}
	return out
}

func TestXterm256MatchesPython(t *testing.T) {
	rows := lines(t, "testdata/xterm256.txt")
	if len(rows) < 4000 {
		t.Fatalf("only %d rows", len(rows))
	}
	for _, r := range rows {
		want, _ := strconv.Atoi(r[1])
		if got := Xterm256(r[0]); got != want {
			t.Errorf("Xterm256(%s) = %d, palette.py says %d", r[0], got, want)
		}
	}
}

func TestFromLCHMatchesPython(t *testing.T) {
	rows := lines(t, "testdata/fromlch.txt")
	if len(rows) < 2000 {
		t.Fatalf("only %d rows", len(rows))
	}
	for _, r := range rows {
		var v [3]float64
		for i := range v {
			f, err := strconv.ParseFloat(r[i], 64)
			if err != nil {
				t.Fatal(err)
			}
			v[i] = f
		}
		if got := FromLCH(v[0], v[1], v[2]); got != r[3] {
			t.Errorf("FromLCH(%v) = %s, palette.py says %s", v, got, r[3])
		}
	}
}

// #010101 is exactly halfway (in OKLab) between xterm 16 and 232; only a cube root as
// accurate as the C library's breaks the tie the way palette.py does.
func TestXterm256Tie(t *testing.T) {
	if got := Xterm256("#010101"); got != 232 {
		t.Errorf("Xterm256(#010101) = %d, want 232", got)
	}
}

func TestParse(t *testing.T) {
	for in, want := range map[string][3]uint8{
		"#6B50FF": {0x6B, 0x50, 0xFF}, "6b50ff": {0x6B, 0x50, 0xFF}, " #abc ": {0xAA, 0xBB, 0xCC},
		"##123456": {0x12, 0x34, 0x56}, "#11223344": {0x11, 0x22, 0x33}, "#12345": {0x12, 0x34, 0x05},
	} {
		got, err := Parse(in)
		if err != nil || got != want {
			t.Errorf("Parse(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	for _, in := range []string{"", "#", "#12", "#1234", "#GG0000", "red"} {
		if Valid(in) {
			t.Errorf("Parse(%q) accepted", in)
		}
	}
}

func TestPythonArithmetic(t *testing.T) {
	for in, want := range map[float64]int{0.5: 0, 1.5: 2, 2.5: 2, 254.5: 254, -0.5: 0} {
		if got := Round(in); got != want {
			t.Errorf("Round(%v) = %d, want %d", in, got, want)
		}
	}
	for _, c := range [][3]float64{{-20, 360, 340}, {400, 360, 40}, {0, 360, 0}, {-360, 360, 0}} {
		if got := Mod(c[0], c[1]); got != c[2] {
			t.Errorf("Mod(%v, %v) = %v, want %v", c[0], c[1], got, c[2])
		}
	}
}
