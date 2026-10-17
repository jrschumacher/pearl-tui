package ui

import (
	"strconv"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"tuikit/theme"
)

// A stream that follows its tail is cut at the top. The cut never falls inside a
// block: when it would, the marker row carries the cut block's head, so the first
// visible row is always a block head or the marker, never a tool's output with its
// head off-screen (a recreation showed "Run import tests · done in 6s" with no "✓
// shell …" above it). Every height and every scroll, for blocks of 1 to 5 rows.
func TestTranscriptCutsOnBlockHeads(t *testing.T) {
	th := theme.Resolve(theme.Default(), true, false)
	const w = 60
	var blocks [][]string
	for b := range 6 {
		n := b%5 + 1
		block := make([]string, n)
		for i := range block {
			block[i] = Fit("  block "+strconv.Itoa(b)+" row "+strconv.Itoa(i), w)
		}
		blocks = append(blocks, block)
	}
	total := 0
	for _, b := range blocks {
		total += len(b) + 1
	}
	total-- // no air after the last block
	marker := th.G.More + " "
	for h := 1; h <= total+2; h++ {
		for scroll := 0; scroll <= total; scroll++ {
			rows := Transcript(th, blocks, w, h, scroll, "pgup")
			if want := min(h, total); len(rows) != want {
				t.Fatalf("h=%d scroll=%d: %d rows, want %d", h, scroll, len(rows), want)
			}
			if len(rows) == 0 {
				continue
			}
			for i, r := range rows {
				if lipgloss.Width(r) != w {
					t.Errorf("h=%d scroll=%d: row %d is %d cells, want %d", h, scroll, i, lipgloss.Width(r), w)
				}
			}
			first := strings.TrimSpace(ansi.Strip(rows[0]))
			head := strings.HasSuffix(first, " row 0")                                                 // a block's head, whole
			plain := strings.HasPrefix(first, marker) && strings.HasSuffix(first, " more lines  pgup") // nothing lifted
			lifted := strings.Contains(first, " row 0") && strings.Contains(first, marker) && strings.HasSuffix(first, " more  pgup")
			if !head && !plain && !lifted {
				t.Errorf("h=%d scroll=%d: the first visible row %q is neither a block head nor the marker", h, scroll, first)
			}
			// after a plain marker the next row is air or a head; after a lifted head, a
			// later row of the same block follows (the rows between are what "n more" counts)
			if len(rows) > 1 {
				second := strings.TrimSpace(ansi.Strip(rows[1]))
				switch {
				case plain && second != "" && !strings.HasSuffix(second, " row 0"):
					t.Errorf("h=%d scroll=%d: after the marker comes %q, want air or a block head", h, scroll, second)
				case lifted && (blockOf(second) != blockOf(first[strings.Index(first, "block"):]) || strings.HasSuffix(second, " row 0")):
					t.Errorf("h=%d scroll=%d: the marker lifts a head but %q follows, want a later row of that block", h, scroll, second)
				}
			}
			// what the marker says is hidden is what is above the first row it shows
			if plain || lifted {
				var n int
				if _, err := fmtSscan(first, marker, &n); err != nil {
					t.Errorf("h=%d scroll=%d: can't read the count in %q: %v", h, scroll, first, err)
				} else if n < 1 {
					t.Errorf("h=%d scroll=%d: the marker hides %d rows", h, scroll, n)
				}
			}
		}
	}
	// the tail is followed: at scroll 0 the last visible row is the last row
	rows := Transcript(th, blocks, w, 8, 0, "")
	if last := strings.TrimSpace(ansi.Strip(rows[len(rows)-1])); last != "block 5 row 0" {
		t.Errorf("scroll 0 ends on %q, want the last row", last)
	}
	// scrolled to the top there is nothing to mark
	if rows := Transcript(th, blocks, w, 8, total, ""); strings.Contains(ansi.Strip(rows[0]), marker) {
		t.Errorf("scrolled to the top, the first row is still a marker: %q", ansi.Strip(rows[0]))
	}
	// a block taller than the window: its head on the marker row, then its tail
	tall := [][]string{{"  short"}, {"  head", "  a", "  b", "  c", "  d", "  e", "  f"}}
	rows = Transcript(th, tall, w, 3, 0, "")
	if first := ansi.Strip(rows[0]); !strings.Contains(first, "head") || !strings.Contains(first, marker) {
		t.Errorf("a block taller than the window: first row %q, want its head on the marker row", first)
	}
	if last := strings.TrimSpace(ansi.Strip(rows[2])); last != "f" {
		t.Errorf("a block taller than the window: last row %q, want its last row", last)
	}
}

// blockOf is the "block N" of a fixture row.
func blockOf(row string) string {
	f := strings.Fields(row)
	if len(f) < 2 {
		return ""
	}
	return f[0] + " " + f[1]
}

// fmtSscan reads the "n" of "<marker>n more" out of a marker row, wherever it sits.
func fmtSscan(row, marker string, n *int) (int, error) {
	_, after, ok := strings.Cut(row, marker)
	if !ok {
		return 0, strconv.ErrSyntax
	}
	v, err := strconv.Atoi(strings.Fields(after)[0])
	*n = v
	return 1, err
}
