package ui

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"tuikit/theme"
)

// Follow scrolls the smallest distance that shows the selection and otherwise stays put,
// so a list never jumps while the selection moves inside the window, and never past the
// end, so a list that shrinks under a filter shows items, not blank rows.
func TestFollow(t *testing.T) {
	for _, c := range []struct{ sel, offset, h, n, want int }{
		{0, 0, 5, 10, 0}, {4, 0, 5, 10, 0}, {5, 0, 5, 10, 1}, {9, 0, 5, 10, 5}, // moving down: the selection sits on the last row
		{2, 3, 5, 10, 2}, {0, 6, 5, 10, 0}, // moving up: the selection sits on the first row
		{7, 3, 5, 10, 3}, {3, 3, 5, 10, 3}, // inside the window: no move
		{3, 0, 0, 10, 0}, {3, -2, 5, 10, 0}, // a zero-height list and a negative offset are 0
		{2, 5, 5, 6, 1}, {0, 5, 5, 3, 0}, {1, 1, 5, 4, 0}, // the list shrank under a filter: the offset comes back to n-h
	} {
		if got := Follow(c.sel, c.offset, c.h, c.n); got != c.want {
			t.Errorf("Follow(%d, %d, %d, %d) = %d, want %d", c.sel, c.offset, c.h, c.n, got, c.want)
		}
	}
}

// Fuzzy is a subsequence match with the indices a palette underlines.
func TestFuzzy(t *testing.T) {
	for _, c := range []struct {
		s, q string
		idx  []int
		ok   bool
	}{
		{"roll back", "ro", []int{0, 1}, true},
		{"roll back", "rb", []int{0, 5}, true},
		{"roll back", "", nil, true},
		{"roll back", "br", nil, false}, // out of order
		{"roll back", "rolx", nil, false},
		{"déploy", "dp", []int{0, 2}, true}, // rune indices, not bytes
	} {
		idx, ok := Fuzzy(c.s, c.q)
		if ok != c.ok || !slices.Equal(idx, c.idx) {
			t.Errorf("Fuzzy(%q, %q) = %v %v, want %v %v", c.s, c.q, idx, ok, c.idx, c.ok)
		}
	}
}

// A blurred selection is its own glyph, not the focus marker in a quieter colour: with
// colour stripped (NO_COLOR, ASCII) the blurred row in one pane and the focused row in
// another were both │, two focuses. Under bar and caret, List and Table mark the blurred
// row with Theme.BlurBar (glyphs.blurBar, or ╎ / ASCII :), which is never the focus
// bar's glyph, the cursor's, or either one's twin.
func TestBlurredSelectionIsItsOwnGlyph(t *testing.T) {
	files := append(testThemes(t), filepath.Join("..", "theme", "testdata", "quire.json"))
	items := []Item{{Title: "api"}, {Title: "worker"}}
	for _, f := range files {
		tf, err := theme.Load(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, sel := range []string{"bar", "caret"} {
			tf.Expressions.Selection = sel
			for _, ascii := range []bool{false, true} {
				th := theme.Resolve(tf, true, ascii)
				focus := th.G.FocusBar
				if sel == "caret" {
					focus = th.G.Cursor
				}
				blur := th.BlurBar()
				if blur == "" || blur == focus || blur == th.G.Cursor || blur == th.G.FocusBar {
					t.Errorf("%s %s ascii=%v: BlurBar %q is no glyph of its own beside focus %q", filepath.Base(f), sel, ascii, blur, focus)
				}
				for _, focused := range []bool{true, false} {
					want := focus
					if !focused {
						want = blur
					}
					list := ansi.Strip(List(th, items, 1, 0, 30, 2, focused)[1])
					table := ansi.Strip(Table(th, []string{"name"}, [][]string{{"api"}, {"worker"}}, []int{0}, "l", 1, 30, focused)[3])
					for what, row := range map[string]string{"List": list, "Table": table} {
						if !strings.HasPrefix(row, want+" ") {
							t.Errorf("%s %s ascii=%v focused=%v: %s marks the selected row %q, want %q", filepath.Base(f), sel, ascii, focused, what, row, want)
						}
					}
				}
			}
		}
	}
}
