package ui

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"tuikit/theme/bubbletea"
)

var sgrFG = regexp.MustCompile(`\x1b\[[0-9;]*38;2;(\d+);(\d+);(\d+)`)

// In ASCII mode every button is a bracket on the terminal background, so its ink must
// read on that background: a pill's ink is its fill's on-colour, often the background
// itself, which made the focused button invisible (found by linting -ascii renders).
func TestASCIIButtonsReadOnTheBackground(t *testing.T) {
	for _, f := range testThemes(t) {
		tf, err := theme.Load(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, dark := range []bool{true, false} {
			th := theme.Resolve(tf, dark, true)
			for _, c := range []struct{ focused, danger bool }{{true, false}, {false, false}, {true, true}, {false, true}} {
				out := Button(th, "Turn on", c.focused, c.danger, -1)
				m := sgrFG.FindStringSubmatch(out)
				if m == nil {
					continue // no explicit ink: the terminal's foreground
				}
				r, _ := strconv.Atoi(m[1])
				g, _ := strconv.Atoi(m[2])
				b, _ := strconv.Atoi(m[3])
				ink := theme.Hex("#" + hex2(r) + hex2(g) + hex2(b))
				if cr := theme.Contrast(ink, th.P.Bg); cr < 2.3 {
					t.Errorf("%s dark=%v focused=%v danger=%v: ASCII button ink %.2f:1 on bg", filepath.Base(f), dark, c.focused, c.danger, cr)
				}
			}
		}
	}
}

func hex2(v int) string {
	s := strconv.FormatInt(int64(v), 16)
	if len(s) < 2 {
		s = "0" + s
	}
	return s
}

// A warning's sparkline ends in its mark, against its last bar, and the mark takes a
// column every row keeps, so the time axis stays aligned: the newest value of every
// row sits in one column, and a warning row draws the same points as a quiet one (#51
// review: it drew two fewer, two columns left).
func TestStateSparklineKeepsTheTimeAxis(t *testing.T) {
	for _, ascii := range []bool{false, true} {
		th := testTheme(t, ascii)
		vals := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14}
		quiet, warn := ansi.Strip(StateSparkline(th, "ok", vals, 12)), ansi.Strip(StateSparkline(th, "warning", vals, 12))
		if lipgloss.Width(quiet) != 12 || lipgloss.Width(warn) != 12 {
			t.Errorf("ascii=%v: widths %d and %d, want 12 for every state", ascii, lipgloss.Width(quiet), lipgloss.Width(warn))
		}
		qr, wr := []rune(quiet), []rune(warn)
		if string(qr[:10]) != string(wr[:10]) {
			t.Errorf("ascii=%v: the warning row draws %q where a quiet one draws %q: one time axis", ascii, string(wr[:10]), string(qr[:10]))
		}
		if string(wr[10:]) != th.G.Warn+" " || strings.TrimSpace(string(qr[10:])) != "" {
			t.Errorf("ascii=%v: mark column %q (warning) and %q (quiet), want %q, the mark against its last bar, and blank", ascii, string(wr[10:]), string(qr[10:]), th.G.Warn+" ")
		}
	}
}

// Under ASCII a progress bar's empty part is "-", and a 1-cell remainder is too short to
// read as a rule: tuilint frames judged it text in the border colour, 1.52:1 (#61). The
// ASCII track is drawn in an ink that clears the 2.3:1 text floor.
func TestASCIIProgressTrackReadsAsText(t *testing.T) {
	for _, f := range testThemes(t) {
		tf, err := theme.Load(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, dark := range []bool{true, false} {
			th := theme.Resolve(tf, dark, true)
			bar := Progress(th, 0.95, 20) // 19 filled, 1 empty
			c := cellAt(bar, th.G.ProgressEmpty)
			if c == nil {
				t.Fatalf("%s dark=%v: no empty cell in %q", filepath.Base(f), dark, bar)
			}
			if cr := theme.Contrast(c.Style.Fg, th.P.Bg); cr < 2.3 {
				t.Errorf("%s dark=%v: the ASCII remainder %q is %.2f:1 on the page, under the 2.3:1 text floor", filepath.Base(f), dark, c.Content, cr)
			}
		}
	}
}
