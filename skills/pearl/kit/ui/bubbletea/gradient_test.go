package ui

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/charmbracelet/colorprofile"
	uv "github.com/charmbracelet/ultraviolet"

	"tuikit/theme/bubbletea"
)

// inkSlots16 are the 16-colour slots of the inks s draws its visible cells in, read
// back as a 16-colour terminal holds them.
func inkSlots16(s string) []int {
	out := downsample(s, colorprofile.ANSI)
	var w int
	for _, l := range strings.Split(out, "\n") {
		w = max(w, uv.NewStyledString(l).Bounds().Dx())
	}
	rows := strings.Count(out, "\n") + 1
	buf := uv.NewScreenBuffer(max(1, w), rows)
	uv.NewStyledString(out).Draw(buf, buf.Bounds())
	var slots []int
	for y := range rows {
		for x := range w {
			if c := buf.CellAt(x, y); c != nil && strings.TrimSpace(c.Content) != "" && c.Style.Fg != nil {
				slots = append(slots, theme.Slot16(c.Style.Fg))
			}
		}
	}
	return slots
}

// At 16 colours a gradient has no steps to draw: each cell snaps to the nearest of the
// 16, and the blend between two brand hues lands on whatever slot is closest. halyard's
// progress bar ran blue, grey, yellow and one cell of error's red into its yellow, and
// where a brand pins secondary to red (cartridge, lumen, moss, terrarium dark) a full,
// healthy bar opened in error red: it read as failing. No kit gradient (the progress
// fill, the wordmark, a dialog's title rule, the scramble) draws a cell on red (1 or 9)
// or on error's own slot, and each draws only its two ends' slots.
func TestGradientsAvoidErrorAt16Colours(t *testing.T) {
	for _, f := range append(testThemes(t), quirePath(t)) {
		tf, err := theme.Load(f)
		if err != nil {
			t.Fatal(err)
		}
		tf.Expressions.Wordmark, tf.Expressions.DialogTitle, tf.Expressions.Working = "gradient", "rule", "scramble"
		for _, dark := range []bool{true, false} {
			th := theme.ResolveProfile(tf, dark, false, colorprofile.ANSI)
			bad := func(s int) bool { return s == 1 || s == 9 || s == theme.Slot16(th.P.Error) }
			ends := []int{theme.Slot16(th.P.Primary), theme.Slot16(th.P.Secondary)}
			a := NewAnim(th, 12, "", "x") // no label: the frame is its scramble alone
			for name, s := range map[string]string{
				"progress":     Progress(th, 1, 40),
				"wordmark":     Wordmark(th, "checkout-api-deployer"),
				"dialog title": DialogTitle(th, "", 40, ""), // no title: the row is its rule
				"scramble":     a.Frame(40, ""),
			} {
				where := fmt.Sprintf("%s dark=%v %s", filepath.Base(f), dark, name)
				slots := inkSlots16(s)
				for _, s := range slots {
					if bad(s) {
						t.Errorf("%s: a cell on ANSI %d, error's red (primary %d, secondary %d, error %d): %v",
							where, s, ends[0], ends[1], theme.Slot16(th.P.Error), slots)
						break
					}
				}
				for _, s := range slots {
					if !slices.Contains(ends, s) && th.X.Focus != "fg" {
						t.Errorf("%s: a cell on ANSI %d, neither end of the gradient (%v): %v", where, s, ends, slots)
						break
					}
				}
			}
			// the error role's own gradient keeps its red
			if got := th.GradRamp(4, th.P.Error, th.P.Error); theme.Slot16(got[0]) != theme.Slot16(th.P.Error) {
				t.Errorf("%s dark=%v: error's own gradient moved off red to ANSI %d", filepath.Base(f), dark, theme.Slot16(got[0]))
			}
		}
	}
}
