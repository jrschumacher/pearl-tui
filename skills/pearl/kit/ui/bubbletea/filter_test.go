package ui

import (
	"path/filepath"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"tuikit/theme/bubbletea"
)

// Both cold monitors (#70) invented a filter of their own, one in a footer badge, one
// elsewhere. Filter is the line the kit puts it on, directly above the list it narrows:
// the prompt and the query, a caret while it is typed, and how many rows it leaves of
// how many on the right. Typed and in force draw the query on the same cells, so
// leaving the input moves nothing; only the prompt's ink and the caret change.
func TestFilterLine(t *testing.T) {
	files := append(testThemes(t), quirePath(t))
	for _, f := range files {
		tf, err := theme.Load(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, ascii := range []bool{false, true} {
			th := theme.Resolve(tf, true, ascii)
			name := filepath.Base(f) + map[bool]string{false: "", true: " ascii"}[ascii]
			for _, w := range []int{40, 76, 116} {
				typed := Filter(th, "er", true, 3, 8, w)
				held := Filter(th, "er", false, 3, 8, w)
				for _, row := range []string{typed, held} {
					if lipgloss.Width(row) != w {
						t.Errorf("%s w=%d: filter line is %d cells: %q", name, w, lipgloss.Width(row), ansi.Strip(row))
					}
					if !strings.HasSuffix(strings.TrimRight(ansi.Strip(row), " "), "3 of 8") {
						t.Errorf("%s w=%d: %q doesn't end with the count", name, w, ansi.Strip(row))
					}
				}
				pt, ph := ansi.Strip(typed), ansi.Strip(held)
				if strings.Index(pt, "er") != strings.Index(ph, "er") {
					t.Errorf("%s w=%d: the query moves when the input is left:\n%q\n%q", name, w, pt, ph)
				}
				if !strings.Contains(pt, "er"+th.G.Caret) || strings.Contains(ph, th.G.Caret) {
					t.Errorf("%s w=%d: the caret is the typed line's only: %q / %q", name, w, pt, ph)
				}
				if c := cellAt(typed, th.G.Prompt); c == nil || !same8(c.Style.Fg, th.S.Prompt.GetForeground()) {
					t.Errorf("%s w=%d: the typed filter's prompt is not in the prompt's ink", name, w)
				}
				if c := cellAt(held, th.G.Prompt); c == nil || same8(c.Style.Fg, th.S.Prompt.GetForeground()) {
					t.Errorf("%s w=%d: a filter in force keeps the prompt's (focus) ink with the input left", name, w)
				}
			}
			if got := ansi.Strip(Filter(th, "", true, 8, 8, 60)); !strings.Contains(got, "type to filter") {
				t.Errorf("%s: an empty typed filter reads %q, want its placeholder", name, got)
			}
		}
	}
}
