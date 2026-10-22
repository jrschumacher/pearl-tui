package ui

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"tuikit/theme/bubbletea"
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
	files := append(testThemes(t), quirePath(t))
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

// A row's status column takes a rung of the theme's ladder (Item.Mark, drawn with
// ui.Mark) beside Dot's states, and a Lead column (a repo#n, an id) as wide as the
// widest lead, so titles line up: an inbox drew its own rows because List had neither.
func TestItemMarkAndLead(t *testing.T) {
	for _, f := range testThemes(t) {
		tf, err := theme.Load(f)
		if err != nil {
			t.Fatal(err)
		}
		tf.Ladder = append(tf.Ladder, theme.LadderRung{Glyph: "current", Meaning: "new", Role: "accent"})
		for _, ascii := range []bool{false, true} {
			th := theme.Resolve(tf, true, ascii)
			items := []Item{{Mark: "new", Lead: "acme/web#128", Title: "Fix the login", Right: "5h"}, {Lead: "ops#7", Title: "Bump deps"}}
			for _, focused := range []bool{true, false} {
				for _, sel := range []int{0, 1} {
					rows := List(th, items, sel, 0, 60, 2, focused)
					var col []int
					for i, r := range rows {
						plain := ansi.Strip(r)
						if n := ansi.StringWidth(plain); n != 60 {
							t.Errorf("%s sel=%d focused=%v: row %d is %d cells: %q", filepath.Base(f), sel, focused, i, n, plain)
						}
						col = append(col, ansi.StringWidth(plain[:strings.Index(plain, items[i].Title)]))
						lead := strings.Index(plain, items[i].Lead)
						if lead < 0 {
							t.Errorf("%s sel=%d focused=%v: row %d drops its lead: %q", filepath.Base(f), sel, focused, i, plain)
						}
					}
					if col[0] != col[1] {
						t.Errorf("%s sel=%d focused=%v: titles start at %v, want one column", filepath.Base(f), sel, focused, col)
					}
					if mark := ansi.Strip(Mark(th, "new")); !strings.Contains(ansi.Strip(rows[0]), mark+" acme/web#128") {
						t.Errorf("%s sel=%d focused=%v: the new rung %q is not before the lead: %q", filepath.Base(f), sel, focused, mark, ansi.Strip(rows[0]))
					}
				}
			}
		}
	}
}

// The status column is as wide as its widest mark: under ASCII cartridge's done and todo
// are [x] and [ ], and a one-cell column cut them to …, which the canvas sweep drew as
// ~, the partial twin: done read as busy.
func TestListStatusColumnKeepsItsMarks(t *testing.T) {
	for _, f := range testThemes(t) {
		tf, err := theme.Load(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, ascii := range []bool{false, true} {
			th := theme.Resolve(tf, true, ascii)
			items := []Item{{State: "ok", Title: "api"}, {State: "idle", Title: "db"}, {State: "error", Title: "queue"}, {Title: "web"}}
			rows := List(th, items, 3, 0, 40, len(items), false)
			col := -1
			for i, r := range rows {
				plain := ansi.Strip(r)
				x := strings.Index(plain, items[i].Title)
				if col >= 0 && ansi.StringWidth(plain[:x]) != col {
					t.Errorf("%s ascii=%v: %q starts at %d, the titles above at %d", filepath.Base(f), ascii, items[i].Title, ansi.StringWidth(plain[:x]), col)
				}
				col = ansi.StringWidth(plain[:x])
				if items[i].State == "" {
					continue
				}
				if mark := ansi.Strip(Dot(th, items[i].State)); !strings.Contains(plain, mark+" ") {
					t.Errorf("%s ascii=%v: %s row %q, want its whole mark %q", filepath.Base(f), ascii, items[i].State, plain, mark)
				}
			}
		}
	}
}

// ui.Table's gap was fixed at 2 cells, which a 10-column process list at 80 columns
// can't afford: Thrift (#61) hand-wrote its rows for 1-cell gaps. The gap is an argument,
// 2 by default.
func TestTableGapIsTheCallers(t *testing.T) {
	th := testTheme(t, false)
	header := []string{"pid", "s", "program", "user", "mem", "cpu", "hist", "bal", "thr", "command"}
	widths := []int{6, 1, 12, 6, 5, 5, 8, 4, 3, 0}
	row := []string{"13912", "R", "yes", "root", "1.4M", "24.6", "abcdefgh", "100", "1", "yes --verbose"}
	starts := func(line string, gap int) []int {
		var out []int
		x := 2
		for _, w := range widths[:len(widths)-1] {
			out = append(out, x)
			x += w + gap
		}
		return append(out, x)
	}
	for _, gap := range []int{1, 2} {
		var out []string
		if gap == 2 {
			out = Table(th, header, [][]string{row}, widths, "rllllrrlll", -1, 80, true)
		} else {
			out = Table(th, header, [][]string{row}, widths, "rllllrrlll", -1, 80, true, gap)
		}
		line := []rune(ansi.Strip(out[2]))
		if len(line) > 80 || len([]rune(ansi.Strip(out[0]))) > 80 {
			t.Fatalf("gap %d: rows of %d cells at 80", gap, len(line))
		}
		for i, x := range starts(string(line), gap) {
			end := len(line)
			if widths[i] > 0 {
				end = x + widths[i]
			}
			cell := strings.TrimSuffix(strings.TrimSpace(string(line[x:end])), th.G.Ellipsis)
			if !strings.HasPrefix(row[i], cell) || cell == "" {
				t.Errorf("gap %d: column %d at cell %d reads %q, want %q", gap, i, x, cell, row[i])
			}
		}
	}
	// at 80 the command column has 19 cells on 1-cell gaps and 10 on 2: the whole command fits only on 1
	wide := ansi.Strip(Table(th, header, [][]string{row}, widths, "rllllrrlll", -1, 80, true, 1)[2])
	if !strings.HasSuffix(strings.TrimRight(wide, " "), "yes --verbose") {
		t.Errorf("1-cell gaps at 80: %q, want the whole command", wide)
	}
	if narrow := ansi.Strip(Table(th, header, [][]string{row}, widths, "rllllrrlll", -1, 80, true)[2]); strings.Contains(narrow, "yes --verbose") {
		t.Errorf("the default gap is not 2: %q", narrow)
	}
}

// The rule under a table's header is the caller's: an app that draws its own divider
// there (a section rule, a filter line) or none sliced Table's output to drop it.
// TableWith{NoRule} leaves it out, and the rows follow the header directly; the rest
// (header, gap, selection) is Table's, row for row.
func TestTableRuleIsOptional(t *testing.T) {
	for _, ascii := range []bool{false, true} {
		th := testTheme(t, ascii)
		header := []string{"service", "when"}
		rows := [][]string{{"ledger", "2d"}, {"auth-gateway", "8d"}}
		ruled := Table(th, header, rows, []int{0, 6}, "lr", 1, 40, true, 1)
		bare := TableWith(th, header, rows, []int{0, 6}, "lr", 1, 40, true, TableOpts{Gap: 1, NoRule: true})
		if len(ruled) != len(rows)+2 || len(bare) != len(rows)+1 {
			t.Fatalf("ascii=%v: %d rows with the rule and %d without, want %d and %d", ascii, len(ruled), len(bare), len(rows)+2, len(rows)+1)
		}
		if !strings.Contains(ansi.Strip(ruled[1]), th.G.Rule) {
			t.Errorf("ascii=%v: Table's second row %q is not the rule", ascii, ansi.Strip(ruled[1]))
		}
		want := append([]string{ruled[0]}, ruled[2:]...)
		for i := range want {
			if bare[i] != want[i] {
				t.Errorf("ascii=%v: row %d without the rule reads %q, want Table's %q", ascii, i, ansi.Strip(bare[i]), ansi.Strip(want[i]))
			}
		}
		if got := TableWith(th, header, rows, []int{0, 6}, "lr", 1, 40, true, TableOpts{}); !slices.Equal(got, Table(th, header, rows, []int{0, 6}, "lr", 1, 40, true)) {
			t.Errorf("ascii=%v: TableWith's zero options are not Table's defaults (gap 2, the rule)", ascii)
		}
		// a negative Gap is none, as Table(…, 0) is: an app moving to TableWith for NoRule
		// keeps its 0-cell gap
		if got, want := TableWith(th, header, rows, []int{0, 6}, "lr", 1, 40, true, TableOpts{Gap: -1, NoRule: true}),
			Table(th, header, rows, []int{0, 6}, "lr", 1, 40, true, 0); !slices.Equal(got, append([]string{want[0]}, want[2:]...)) {
			t.Errorf("ascii=%v: TableWith Gap -1 is not Table's gap 0:\n%q\n%q", ascii, got, want)
		}
	}
}

// A zero width stretches its column over the room the others leave, wherever it stands:
// Table stretched a zero-width last column, and drew a zero-width middle (or first)
// column as no cells at all, its text gone and the next column moved up. Several zero
// columns share the room, the last taking any odd cell, and every row keeps w cells.
func TestTableStretchesAZeroColumn(t *testing.T) {
	th := testTheme(t, false)
	header := []string{"service", "summary", "age"}
	rows := [][]string{{"ledger", "p99 latency over budget", "2d"}, {"auth-gateway", "all healthy", "8d"}}
	for _, c := range []struct {
		widths []int
		want   []int // each column's cells at w 60, gap 2, the 2-cell marker column
	}{
		{[]int{14, 0, 4}, []int{14, 36, 4}},
		{[]int{0, 30, 4}, []int{20, 30, 4}},
		{[]int{14, 30, 0}, []int{14, 30, 10}}, // the last, as before
		{[]int{14, 0, 0}, []int{14, 20, 20}},
		{[]int{0, 0, 4}, []int{25, 25, 4}},
	} {
		out := Table(th, header, rows, c.widths, "llr", -1, 60, true)
		for i, l := range out {
			if got := len([]rune(ansi.Strip(l))); got != 60 {
				t.Errorf("widths %v: row %d is %d cells, want 60: %q", c.widths, i, got, ansi.Strip(l))
			}
		}
		line := []rune(ansi.Strip(out[2]))
		x := 2
		for i, w := range c.want {
			cell := strings.TrimSpace(string(line[x : x+w]))
			if !strings.HasPrefix(rows[0][i], strings.TrimSuffix(cell, th.G.Ellipsis)) || cell == "" {
				t.Errorf("widths %v: column %d at %d+%d reads %q, want %q: %q", c.widths, i, x, w, cell, rows[0][i], string(line))
			}
			x += w + 2
		}
	}
}

// The marker column is as wide as the widest mark it holds, measured: an app whose
// theme twinned its → cursor to "->" under ASCII drew a selected row a cell wider than
// the rest, its title a cell right of every other title and its last cell cut, and a
// kit test failed. List and Table keep every row w cells and every title in one column,
// whatever the cursor's, focus bar's or blurred bar's width.
func TestMarkerColumnFitsAWideCursor(t *testing.T) {
	path := filepath.Join("..", "..", "themes", "moss.json")
	for _, f := range testThemes(t) {
		if filepath.Base(f) == "moss.json" {
			path = f
		}
	}
	tf, err := theme.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	tf.Glyphs.Cursor = "→"
	tf.Glyphs.ASCII = map[string]string{"→": "->"}
	items := []Item{{Title: "ledger", Right: "2d"}, {Title: "auth-gateway", Right: "8d"}, {Title: "checkout-api", Right: "1h"}}
	rows := [][]string{{"ledger", "2d"}, {"auth-gateway", "8d"}, {"checkout-api", "1h"}}
	for _, sel := range []string{"caret", "bar", "fill", "inverse"} {
		for _, ascii := range []bool{false, true} {
			tf.Expressions.Selection = sel
			th := theme.Resolve(tf, true, ascii)
			name := fmt.Sprintf("%s ascii=%v cursor=%q", sel, ascii, th.G.Cursor)
			for _, focused := range []bool{true, false} {
				check := func(what string, out []string, selRow int, title func(i int) string) {
					col := -1
					for i, l := range out {
						plain := ansi.Strip(l)
						if got := lipgloss.Width(l); got != 40 {
							t.Errorf("%s focused=%v %s row %d: %d cells, want 40: %q", name, focused, what, i, got, plain)
						}
						x := strings.Index(plain, title(i))
						if x < 0 {
							t.Errorf("%s focused=%v %s row %d: %q lost %q", name, focused, what, i, plain, title(i))
							continue
						}
						x = lipgloss.Width(plain[:x])
						if col < 0 {
							col = x
						}
						if x != col {
							t.Errorf("%s focused=%v %s row %d: %q starts at %d, the other rows at %d", name, focused, what, i, plain, x, col)
						}
						if i == selRow && focused && sel == "caret" && !strings.Contains(plain, th.G.Cursor+" ") {
							t.Errorf("%s %s: the selected row %q doesn't draw the cursor %q whole", name, what, plain, th.G.Cursor)
						}
					}
				}
				check("list", List(th, items, 1, 0, 40, 3, focused), 1, func(i int) string { return items[i].Title })
				tbl := Table(th, []string{"service", "age"}, rows, []int{0, 4}, "lr", 1, 40, focused)
				check("table", append([]string{tbl[0]}, tbl[2:]...), 2, func(i int) string {
					if i == 0 {
						return th.Label("service", "upper")
					}
					return rows[i-1][0]
				})
			}
		}
	}
}
