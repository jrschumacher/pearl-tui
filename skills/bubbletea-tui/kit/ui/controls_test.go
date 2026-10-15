package ui

import (
	"path/filepath"
	"regexp"
	"strconv"
	"testing"

	"tuikit/theme"
)

var sgrFG = regexp.MustCompile(`\x1b\[[0-9;]*38;2;(\d+);(\d+);(\d+)`)

// In ASCII mode every button is a bracket on the terminal background, so its ink must
// read on that background: a pill's ink is its fill's on-colour, often the background
// itself, which made the focused button invisible (found by linting -ascii renders).
func TestASCIIButtonsReadOnTheBackground(t *testing.T) {
	files, _ := filepath.Glob("../themes/*.json")
	for _, f := range files {
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
