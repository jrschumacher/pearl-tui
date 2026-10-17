package ui

import (
	"slices"
	"testing"
)

// Follow scrolls the smallest distance that shows the selection and otherwise stays put,
// so a list never jumps while the selection moves inside the window.
func TestFollow(t *testing.T) {
	for _, c := range []struct{ sel, offset, h, want int }{
		{0, 0, 5, 0}, {4, 0, 5, 0}, {5, 0, 5, 1}, {9, 0, 5, 5}, // moving down: the selection sits on the last row
		{2, 3, 5, 2}, {0, 6, 5, 0}, // moving up: the selection sits on the first row
		{7, 3, 5, 3}, {3, 3, 5, 3}, // inside the window: no move
		{3, 0, 0, 0}, {3, -2, 5, 0}, // a zero-height list and a negative offset are 0
	} {
		if got := Follow(c.sel, c.offset, c.h); got != c.want {
			t.Errorf("Follow(%d, %d, %d) = %d, want %d", c.sel, c.offset, c.h, got, c.want)
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
