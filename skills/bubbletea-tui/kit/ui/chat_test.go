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
		n := (b+2)%5 + 1 // 3, 4, 5, 1, 2, 3 rows: the last block is more than its head
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
			head := strings.HasSuffix(first, " row 0")                                           // a block's head, whole
			plain := strings.HasPrefix(first, marker) && strings.HasSuffix(first, " more  pgup") // nothing lifted
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
	if last := strings.TrimSpace(ansi.Strip(rows[len(rows)-1])); last != "block 5 row 2" {
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
	// two rows, the cut on the last block's last row: its head is lifted, not left off
	// (the last block has no air after it, so the cut looked like a block boundary)
	ten := make([]string, 10)
	for i := range ten {
		ten[i] = "  line " + strconv.Itoa(i)
	}
	for _, k := range []struct {
		name         string
		blocks       [][]string
		head, last   string
		hiddenBefore int
	}{
		{"a tool after a turn", [][]string{{"  U user"}, {"  ● Tool head", "  out 1", "  out 2", "  out 3"}}, "● Tool head", "out 3", 4},
		{"one ten-row block", [][]string{ten}, "line 0", "line 9", 8},
	} {
		rows := Transcript(th, k.blocks, 40, 2, 0, "pgup")
		first, second := ansi.Strip(rows[0]), strings.TrimSpace(ansi.Strip(rows[1]))
		if !strings.Contains(first, k.head) || !strings.Contains(first, marker) {
			t.Errorf("%s at h=2: first row %q, want the head %q lifted onto the marker", k.name, first, k.head)
		}
		var n int
		if _, err := fmtSscan(first, marker, &n); err != nil || n != k.hiddenBefore {
			t.Errorf("%s at h=2: the marker %q counts %d, want %d", k.name, first, n, k.hiddenBefore)
		}
		if second != k.last {
			t.Errorf("%s at h=2: second row %q, want %q", k.name, second, k.last)
		}
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

// A tool call carries what the spec of an agent gives it: a description under the head
// (muted, cut with the head by Transcript), a note right-aligned on the head row (how
// long it took, how it was allowed), and options that are flags (an empty value) drawn
// bare, not as "k=". When the row is short, the k=v group goes first, then the note,
// and the parameter is cut last. Two recreations added Meta by hand and one drew the
// description as output.
func TestToolCallDescMetaAndBareOpts(t *testing.T) {
	th := theme.Resolve(theme.Default(), true, false)
	tc := Tool{Name: "shell", Param: "go test ./internal/import/...", Desc: "Run import tests",
		Opts: [][2]string{{"timeout", "60s"}, {"verbose", ""}}, Meta: "allowed for session " + th.G.Sep + " 6s",
		State: ToolDone, Body: []string{"ok  ledger/internal/import  0.412s"}}
	rows := ToolCall(th, tc, 100, false)
	plain := make([]string, len(rows))
	for i, r := range rows {
		plain[i] = ansi.Strip(r)
	}
	head := strings.TrimRight(plain[0], " ")
	if !strings.HasSuffix(head, tc.Meta) {
		t.Errorf("the note is not right-aligned on the head: %q", head)
	}
	if !strings.Contains(head, "(timeout=60s, verbose)") {
		t.Errorf("a flag option is not drawn bare: %q", head)
	}
	if got := strings.TrimSpace(plain[1]); got != tc.Desc {
		t.Errorf("row 1 is %q, want the description %q under the head", got, tc.Desc)
	}
	if strings.TrimSpace(plain[2]) != "" || !strings.Contains(plain[3], "ok  ledger") {
		t.Errorf("want air then the output after the description, got %q / %q", plain[2], plain[3])
	}
	for i, r := range rows {
		if lipgloss.Width(r) != 100 {
			t.Errorf("row %d is %d cells, want 100", i, lipgloss.Width(r))
		}
	}
	// at 80 the three don't fit: the k=v group goes, the note stays
	if head := strings.TrimRight(ansi.Strip(ToolCall(th, tc, 80, false)[0]), " "); strings.Contains(head, "timeout=") || !strings.HasSuffix(head, tc.Meta) {
		t.Errorf("at 80 cells want the options dropped and the note kept: %q", head)
	}
	// narrow: the note goes before the parameter is cut for it
	narrow := ansi.Strip(ToolCall(th, tc, 46, false)[0])
	if strings.Contains(narrow, "6s") {
		t.Errorf("at 46 cells the note still draws: %q", narrow)
	}
	if !strings.Contains(narrow, "go test") {
		t.Errorf("at 46 cells the parameter is gone: %q", narrow)
	}
	// no description: the head's air row follows it, as before
	tc.Desc = ""
	if rows := ToolCall(th, tc, 100, false); strings.TrimSpace(ansi.Strip(rows[1])) != "" {
		t.Errorf("without a description row 1 is %q, want air", ansi.Strip(rows[1]))
	}
}

// A lifted head with a right-aligned note is cut at the gap before the note, not
// through it: the showcase's marker row read "✓ Edit config/timeouts.go  allowed …".
func TestTranscriptLiftsAHeadWithoutItsNote(t *testing.T) {
	th := theme.Resolve(theme.Default(), true, false)
	tc := Tool{Name: "Edit", Param: "config/timeouts.go", Meta: "allowed for session " + th.G.Sep + " 0.3s",
		State: ToolDone, Body: []string{"a", "b", "c", "d", "e", "f"}, Expand: true}
	for _, w := range []int{60, 70, 80} {
		rows := Transcript(th, [][]string{ToolCall(th, tc, w, false)}, w, 4, 0, "pgup")
		first := ansi.Strip(rows[0])
		if !strings.Contains(first, "config/timeouts.go") || !strings.Contains(first, th.G.More+" ") {
			t.Fatalf("w=%d: want the head lifted onto the marker row, got %q", w, first)
		}
		if strings.Contains(first, "allowed") && !strings.Contains(first, tc.Meta) {
			t.Errorf("w=%d: the lifted head carries a fragment of its note: %q", w, first)
		}
		if lipgloss.Width(rows[0]) != w {
			t.Errorf("w=%d: the marker row is %d cells", w, lipgloss.Width(rows[0]))
		}
	}
}

// At a width where the marker row can't hold the lifted head's icon and name, the
// marker is the plain one and counts every hidden row, the head among them: at 12 cells
// the head was cut to nothing ("↑ 4 more …" over "+ c") and its count was one short.
func TestTranscriptLiftsAHeadOnlyWhereItReads(t *testing.T) {
	th := theme.Resolve(theme.Default(), true, false)
	blocks := [][]string{{"  U user"}, {"  ● Edit config/timeouts.go", "  + a", "  + b", "  + c"}}
	const h = 2
	for w := 1; w <= 60; w++ {
		rows := Transcript(th, blocks, w, h, 0, "pgup")
		first := ansi.Strip(rows[0])
		if lipgloss.Width(rows[0]) != w {
			t.Errorf("w=%d: the marker row is %d cells", w, lipgloss.Width(rows[0]))
		}
		if !strings.Contains(first, "● Edit") {
			// plain: it counts the head too, when the count fits at all
			if strings.Contains(first, th.G.More+" ") {
				var n int
				if _, err := fmtSscan(first, th.G.More+" ", &n); err == nil && n != 5 {
					t.Errorf("w=%d: the plain marker %q counts %d, want all 5 hidden rows", w, first, n)
				}
			}
			if strings.Contains(first, "●") || strings.Contains(first, "Edi") {
				t.Errorf("w=%d: a head cut short of its icon and name: %q", w, first)
			}
			continue
		}
		var n int
		if _, err := fmtSscan(first, th.G.More+" ", &n); err != nil || n != 4 {
			t.Errorf("w=%d: the lifted marker %q counts %d, want 4 (the head is shown)", w, first, n)
		}
	}
	// wide enough, the head is lifted; at 12 cells it can't be
	if first := ansi.Strip(Transcript(th, blocks, 40, h, 0, "pgup")[0]); !strings.Contains(first, "● Edit") {
		t.Errorf("w=40: want the head lifted, got %q", first)
	}
	if first := ansi.Strip(Transcript(th, blocks, 12, h, 0, "pgup")[0]); !strings.HasPrefix(strings.TrimSpace(first), th.G.More+" 5") {
		t.Errorf("w=12: want the plain marker counting 5, got %q", first)
	}
}
