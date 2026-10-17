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
	Match []int  // rune indices in Title to underline (fuzzy filter hits)
}

// List renders items into exactly h rows of w cells, with the selected row
// marked per expressions.selection and a scrollbar column only when the list
// overflows. focused=false keeps the selection visible but quieter, so the
// user can still see where they were when focus moves elsewhere.
func List(t *theme.Theme, items []Item, sel, offset, w, h int, focused bool) []string {
	sb := Scrollbar(t, h, len(items), offset)
	bodyW := w
	if len(items) > h {
		bodyW = w - 2
	}
	out := make([]string, 0, h)
	for row := 0; row < h; row++ {
		i := offset + row
		if i >= len(items) {
			out = append(out, Fit("", w))
			continue
		}
		line := listRow(t, items[i], i == sel, focused, bodyW)
		if len(items) > h {
			line += " " + sb[row]
		}
		out = append(out, line)
	}
	return out
}

func listRow(t *theme.Theme, it Item, selected, focused bool, w int) string {
	mode := t.X.Selection
	// the marker column is always 2 cells, focused or not, so selecting a row
	// never shifts its text (Crush's rule for its focus bar)
	marker := "  "
	switch {
	case selected && mode == "bar":
		if focused {
			marker = t.S.FocusBar.Render(t.G.FocusBar) + " "
		} else {
			marker = t.S.BlurBar.Render(t.G.FocusBar) + " "
		}
	case selected && mode == "caret":
		st := t.S.Focus
		if !focused {
			st = t.S.Muted
		}
		marker = st.Render(t.G.Cursor) + " "
	}
	plain := selected && focused && (mode == "fill" || mode == "inverse")
	if plain {
		return filledRow(t, it, w)
	}
	titleSt := t.S.Base
	if selected && focused && (mode == "bar" || mode == "caret") {
		titleSt = t.S.SelRow
	}
	title := matchUnderline(it.Title, it.Match, titleSt)
	if it.State != "" {
		title = Dot(t, it.State) + " " + title
	}
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

// filledRow is a selected row on a filled bar. Every span, the padding
// included, carries the bar's style: one bare space would punch a hole.
func filledRow(t *theme.Theme, it Item, w int) string {
	st := t.S.SelRow
	text := it.Title
	if it.State != "" {
		text = ansi.Strip(Dot(t, it.State)) + " " + text
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
	title := matchUnderline(string([]rune(full)[:nt]), it.Match, st)
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

// Table renders a borderless table: a muted header row, a rule, then rows
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
		h = append(h, t.S.Subtle.Render(cell(strings.ToUpper(c), i)))
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
			bar := t.S.FocusBar
			if !focused {
				bar = t.S.BlurBar
			}
			out = append(out, Fit(bar.Render(t.G.FocusBar)+" "+line, w))
		case selected && mode == "caret":
			out = append(out, Fit(t.S.Focus.Render(t.G.Cursor)+" "+line, w))
		case selected:
			out = append(out, lipgloss.NewStyle().Background(t.P.BgSubtle).Render(Fit("  "+line, w)))
		default:
			out = append(out, Fit("  "+line, w))
		}
	}
	return out
}
