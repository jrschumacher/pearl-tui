package ui

import (
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"

	"tuikit/theme/bubbletea"
)

// Filter is the line a list's filter stands on while it is typed or in force: on the
// list's own first row, directly above the header it narrows (the rows below move down
// to make room), never in the footer, where a toast or the keys would cover it. It is
// the prompt and the query, then, at the right edge, how many rows it leaves of how
// many ("3 of 8", "0 of 8" with nothing left).
//
//	› er▏                                                  3 of 8   typed: the prompt's ink, a caret
//	› er                                                   3 of 8   in force: the prompt quiet, no caret
//
// typing is true while the input has the focus: the prompt takes the prompt's ink and
// the caret follows the query (or the placeholder when it is empty). In force, the
// prompt is quiet and the caret's cell is blank, so the query stays on its cells. The
// footer offers `esc clear filter` only while a filter is in force or typed, and `/`
// to type one; under a dialog draw it with typing false, as any blurred input.
func Filter(t *theme.Theme, query string, typing bool, shown, total, w int) string {
	prompt, caret := t.S.Subtle.Render(t.G.Prompt), strings.Repeat(" ", lipgloss.Width(t.G.Caret))
	if typing {
		prompt, caret = t.S.Prompt.Render(t.G.Prompt), t.S.Prompt.Render(t.G.Caret)
	}
	count := t.S.Subtle.Render(strconv.Itoa(shown) + " of " + strconv.Itoa(total))
	left := prompt + " " + t.S.Base.Render(query) + caret
	if query == "" && typing {
		left += t.S.Subtle.Render("type to filter")
	}
	room := max(0, w-lipgloss.Width(count)-2)
	return Fit(Fit(left, room)+"  "+count, w)
}
