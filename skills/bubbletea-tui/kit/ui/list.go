package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"tuikit/theme"
)

// Item is one list row.
type Item struct {
	Title string
	Desc  string // shown muted after the title when there is room
	Right string // right-aligned, subtle (a shortcut, a count, a time)
	State string // optional status dot (see Dot); "" for none
	Mark  string // a rung of the theme's ladder by its meaning ("new"), drawn with ui.Mark in place of State
	Lead  string // a muted column before the title (a repo#n, an id), as wide as the widest lead in the list
	Match []int  // rune indices in Title to underline (fuzzy filter hits)
}

// status is the row's status column: its ladder rung, else its Dot, else "".
func (it Item) status(t *theme.Theme) string {
	switch {
	case it.Mark != "":
		return Mark(t, it.Mark)
	case it.State != "":
		return Dot(t, it.State)
	}
	return ""
}

// cols are the columns a list's rows share, from all its items, so titles line up
// whatever the scroll offset: a status column when any item has one, and a Lead column
// as wide as the widest lead, at most a third of the row.
type cols struct {
	status bool
	lead   int
}

func listCols(items []Item, w int) cols {
	var c cols
	for _, it := range items {
		c.status = c.status || it.Mark != "" || it.State != ""
		c.lead = max(c.lead, lipgloss.Width(it.Lead))
	}
	c.lead = min(c.lead, w/3)
	return c
}

// prefix is the status and lead columns before a title; plain drops their styles
// (a filled row carries its own).
func (c cols) prefix(t *theme.Theme, it Item, plain bool) string {
	out := ""
	if c.status {
		st := it.status(t)
		if plain {
			st = ansi.Strip(st)
		}
		out = Fit(st, 1) + " "
	}
	if c.lead > 0 {
		lead := Fit(Trunc(t, it.Lead, c.lead), c.lead)
		if !plain {
			lead = t.S.Muted.Render(lead)
		}
		out += lead + "  "
	}
	return out
}

// List renders items into exactly h rows of w cells, with the selected row
// marked per expressions.selection and a scrollbar column only when the list
// overflows. focused=false keeps the selection visible but quieter, so the
// user can still see where they were when focus moves elsewhere: under bar and
// caret it is marked with Theme.BlurBar, its own glyph in fgSubtle, never the
// focus bar or the cursor in a quieter colour (one focus under NO_COLOR too).
func List(t *theme.Theme, items []Item, sel, offset, w, h int, focused bool) []string {
	sb := Scrollbar(t, h, len(items), offset)
	bodyW := w
	if len(items) > h {
		bodyW = w - 2
	}
	out := make([]string, 0, h)
	c := listCols(items, bodyW)
	for row := 0; row < h; row++ {
		i := offset + row
		if i >= len(items) {
			out = append(out, Fit("", w))
			continue
		}
		line := listRow(t, items[i], i == sel, focused, bodyW, c)
		if len(items) > h {
			line += " " + sb[row]
		}
		out = append(out, line)
	}
	return out
}

// Follow is the offset that keeps sel on screen in a list of n items, h rows tall:
// offset when sel is already visible, else the smallest move that shows it (sel at the
// top when it was above the window, at the bottom when it was below), never past n-h,
// so a list that shrinks under a filter shows its last items and no blank rows. Call it
// before List on every render, so a selection moved by a key or a filter scrolls into
// view and a list that fits never scrolls.
func Follow(sel, offset, h, n int) int {
	if h <= 0 {
		return 0
	}
	offset = min(max(0, offset), max(0, n-h))
	switch {
	case sel < offset:
		return max(0, sel)
	case sel >= offset+h:
		return sel - h + 1
	}
	return offset
}

// Fuzzy matches q as a subsequence of s, the way a command palette filters: the matched
// rune indices (Item.Match underlines them) and whether every rune of q was found, in
// order. Lower-case both for a case-insensitive filter.
func Fuzzy(s, q string) ([]int, bool) {
	var idx []int
	rs := []rune(s)
	j := 0
	for _, qr := range q {
		for j < len(rs) && rs[j] != qr {
			j++
		}
		if j == len(rs) {
			return nil, false
		}
		idx = append(idx, j)
		j++
	}
	return idx, true
}

func listRow(t *theme.Theme, it Item, selected, focused bool, w int, c cols) string {
	mode := t.X.Selection
	// the marker column is always 2 cells, focused or not, so selecting a row
	// never shifts its text (Crush's rule for its focus bar)
	marker := "  "
	switch {
	case selected && mode == "bar":
		if focused {
			marker = t.S.FocusBar.Render(t.G.FocusBar) + " "
		} else {
			marker = t.S.BlurBar.Render(t.BlurBar()) + " " // its own shape: colour alone is no cue
		}
	case selected && mode == "caret" && focused:
		marker = t.S.Focus.Render(t.G.Cursor) + " "
	case selected && mode == "caret":
		marker = t.S.BlurBar.Render(t.BlurBar()) + " " // the cursor is the focus: a blurred row keeps the quiet bar
	}
	plain := selected && focused && (mode == "fill" || mode == "inverse")
	if plain {
		return filledRow(t, it, w, c)
	}
	titleSt := t.S.Base
	if selected && focused && (mode == "bar" || mode == "caret") {
		titleSt = t.S.SelRow
	}
	title := matchUnderline(it.Title, it.Match, titleSt)
	title = c.prefix(t, it, false) + title
	right := ""
	if it.Right != "" {
		right = t.S.Subtle.Render(it.Right)
	}
	avail := w - 2 - lipgloss.Width(right)
	if right != "" {
		avail--
	}
	text := title
	if it.Desc != "" && lipgloss.Width(title)+2+8 < avail {
		text = title + "  " + t.S.Muted.Render(it.Desc)
	}
	line := Fit(marker+Fit(Trunc(t, text, max(0, avail)), max(0, avail))+pre(right != "", " ")+right, w)
	if selected && !focused && (mode == "fill" || mode == "inverse") {
		return lipgloss.NewStyle().Background(t.P.BgSubtle).Render(ansi.Strip(line))
	}
	return line
}

// bolden draws pre-styled cells in bold throughout, keeping each one's colour: every
// span's style ends in a reset, so bold is set again after each SGR sequence.
func bolden(s string) string {
	on := ansi.Style{}.Bold().String()
	var b strings.Builder
	b.WriteString(on)
	var state byte
	for len(s) > 0 {
		seq, _, n, next := ansi.DecodeSequence(s, state, nil)
		b.WriteString(seq)
		if ansi.HasCsiPrefix(seq) && strings.HasSuffix(seq, "m") {
			b.WriteString(on)
		}
		s, state = s[n:], next
	}
	b.WriteString(ansi.Style{}.Normal().String())
	return b.String()
}

// filledRow is a selected row on a filled bar. Every span, the padding
// included, carries the bar's style: one bare space would punch a hole.
func filledRow(t *theme.Theme, it Item, w int, c cols) string {
	st := t.S.SelRow
	lead := ansi.Strip(c.prefix(t, it, true))
	text := lead + it.Title
	match := make([]int, len(it.Match)) // Match indexes the title, after the columns
	for i, m := range it.Match {
		match[i] = m + len([]rune(lead))
	}
	avail := w - 2
	if it.Right != "" {
		avail -= lipgloss.Width(it.Right) + 1
	}
	desc := ""
	if it.Desc != "" && lipgloss.Width(text)+2+8 < avail {
		desc = "  " + it.Desc
	}
	full := Trunc(t, text+desc, avail)
	nt := min(len([]rune(full)), len([]rune(text)))
	title := matchUnderline(string([]rune(full)[:nt]), match, st)
	rest := st.Render(string([]rune(full)[nt:]))
	gap := avail - lipgloss.Width(full)
	out := st.Render("  ") + title + rest + st.Render(strings.Repeat(" ", max(0, gap)))
	if it.Right != "" {
		out += st.Render(" " + it.Right)
	}
	return out
}

func pre(ok bool, s string) string {
	if ok {
		return s
	}
	return ""
}

func matchUnderline(s string, idx []int, st lipgloss.Style) string {
	if len(idx) == 0 {
		return st.Render(s)
	}
	hit := map[int]bool{}
	for _, i := range idx {
		hit[i] = true
	}
	var b strings.Builder
	for i, r := range []rune(s) {
		if hit[i] {
			b.WriteString(st.Underline(true).Render(string(r)))
		} else {
			b.WriteString(st.Render(string(r)))
		}
	}
	return b.String()
}

// Table renders a borderless table: a muted header row (upper case unless the theme's
// labelCase says otherwise), a rule, then rows
// with aligned columns. widths are cells per column (the last may be 0 =
// take the rest). align: 'l' or 'r' per column. Selected row per selection.
func Table(t *theme.Theme, header []string, rows [][]string, widths []int, align string, sel, w int, focused bool) []string {
	ws := append([]int(nil), widths...)
	used := 0
	for i, x := range ws {
		if i < len(ws)-1 || x > 0 {
			used += x + 2
		}
	}
	if ws[len(ws)-1] == 0 {
		ws[len(ws)-1] = max(4, w-2-used)
	}
	cell := func(s string, i int) string {
		s = Trunc(t, s, ws[i])
		if i < len(align) && align[i] == 'r' {
			return lipgloss.PlaceHorizontal(ws[i], lipgloss.Right, s)
		}
		return Fit(s, ws[i])
	}
	var h []string
	for i, c := range header {
		h = append(h, t.S.Subtle.Render(cell(t.Label(c, "upper"), i)))
	}
	out := []string{Fit("  "+strings.Join(h, "  "), w), "  " + Rule(t, w-2)}
	for r, row := range rows {
		var cs []string
		for i, c := range row {
			cs = append(cs, cell(c, i))
		}
		line := strings.Join(cs, "  ")
		selected := r == sel
		mode := t.X.Selection
		switch {
		case selected && focused && (mode == "fill" || mode == "inverse"):
			out = append(out, t.S.SelRow.Render(Fit("  "+ansi.Strip(line), w)))
		case selected && (mode == "bar"):
			bar, glyph := t.S.FocusBar, t.G.FocusBar
			if !focused {
				bar, glyph = t.S.BlurBar, t.BlurBar() // its own shape, as List draws it
			} else {
				line = bolden(line) // SelRow: bold text beside the bar, as List draws it
			}
			out = append(out, Fit(bar.Render(glyph)+" "+line, w))
		case selected && mode == "caret" && focused:
			out = append(out, Fit(t.S.Focus.Render(t.G.Cursor)+" "+bolden(line), w))
		case selected && mode == "caret":
			out = append(out, Fit(t.S.BlurBar.Render(t.BlurBar())+" "+line, w))
		case selected:
			out = append(out, lipgloss.NewStyle().Background(t.P.BgSubtle).Render(Fit("  "+line, w)))
		default:
			out = append(out, Fit("  "+line, w))
		}
	}
	return out
}
