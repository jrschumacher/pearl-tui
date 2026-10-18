package ui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"

	"tuikit/theme/bubbletea"
)

// A dialog's status (the sending dots, "3 of 12", "saved") is set into its bottom edge,
// right-aligned, as the title is set into a rules dialog's top one: a confirm's sending
// dots had nowhere to go but the button row. The status takes no row, so the dialog is
// as tall as without it (DialogBodyRows holds for it); every row keeps its width, and a
// status too long for the edge is cut, never wrapped.
func TestDialogStatusSitsInTheBottomEdge(t *testing.T) {
	for _, dialog := range []string{"rounded", "square", "rules", "none"} {
		for _, mode := range []struct {
			name  string
			ascii bool
			p     colorprofile.Profile
		}{{"truecolor", false, colorprofile.TrueColor}, {"ascii", true, colorprofile.TrueColor}, {"ansi", false, colorprofile.ANSI}} {
			f := theme.Default()
			f.Expressions.Dialog = dialog
			th := theme.ResolveProfile(f, true, mode.ascii, mode.p)
			const w = 46
			body := []string{"To: ops@example.com", "Re: checkout-api 502s"}
			btns := Buttons(th, w-th.S.Dialog.GetHorizontalFrameSize(), lipgloss.Right, Button(th, "Send", true, false, 0))
			plain := Dialog(th, w, 0, "Send reply", body, btns, "")
			d := DialogStatus(th, w, 0, "Send reply", th.S.Subtle.Render("sending"), body, 0, btns, "")
			name := dialog + " " + mode.name
			if lipgloss.Height(d) != lipgloss.Height(plain) {
				t.Errorf("%s: the status took a row: %d rows, want %d", name, lipgloss.Height(d), lipgloss.Height(plain))
			}
			rows := strings.Split(ansi.Strip(d), "\n")
			prows := strings.Split(ansi.Strip(plain), "\n")
			for i, r := range rows {
				if lipgloss.Width(r) != w {
					t.Errorf("%s: row %d is %d cells, want %d", name, i, lipgloss.Width(r), w)
				}
			}
			// the bottom edge: a border's last row (blank under a hidden one), a rules
			// dialog's rule above its row of margin
			bottom := len(rows) - 1
			if dialog == "rules" {
				bottom--
			}
			for i := range rows {
				if i != bottom && rows[i] != prows[i] {
					t.Errorf("%s: row %d changed with a status: %q, want %q", name, i, rows[i], prows[i])
				}
			}
			edge := rows[bottom]
			at := strings.LastIndex(edge, " sending ")
			if at < 0 {
				t.Errorf("%s: the bottom edge %q doesn't carry the status", name, rows[bottom])
				continue
			}
			// right-aligned: two edge cells after it, then a border's corner or a rules
			// dialog's cell of margin
			if tail := lipgloss.Width(edge[at+len(" sending "):]); tail != 3 {
				t.Errorf("%s: the status sits %d cells from the edge's end: %q", name, tail, edge)
			}
			// too long: cut on the edge, the edge keeps its width
			long := DialogStatus(th, w, 0, "Send reply", strings.Repeat("sending ", 10), body, 0, btns, "")
			for i, r := range strings.Split(ansi.Strip(long), "\n") {
				if lipgloss.Width(r) != w {
					t.Errorf("%s: with a long status row %d is %d cells, want %d", name, i, lipgloss.Width(r), w)
				}
			}
		}
	}
}

// A body that must be shown whole (a write's verbatim text, with no length bound)
// scrolls between edges that stay, and its position is set into the bottom edge:
// "lines 1–10 of 76", the last offset clamped so the last page still fills, nothing
// when the body fits. The position takes no row.
func TestDialogLinesSaysWhereTheBodyStands(t *testing.T) {
	th := theme.ResolveProfile(theme.Default(), true, false, colorprofile.TrueColor)
	for _, c := range []struct {
		off, shown, total int
		want              string
	}{
		{0, 10, 76, "lines 1–10 of 76"},
		{5, 10, 76, "lines 6–15 of 76"},
		{70, 10, 76, "lines 67–76 of 76"},
		{-3, 10, 76, "lines 1–10 of 76"},
		{0, 10, 10, ""},
		{0, 10, 4, ""},
	} {
		if got := ansi.Strip(DialogLines(th, c.off, c.shown, c.total)); got != c.want {
			t.Errorf("DialogLines(%d, %d, %d) = %q, want %q", c.off, c.shown, c.total, got, c.want)
		}
	}
	body := make([]string, 10)
	for i := range body {
		body[i] = "line " + strings.Repeat("x", i)
	}
	d := DialogStatus(th, 46, 0, "Send reply", DialogLines(th, 0, 10, 76), body, 0, "", "")
	rows := strings.Split(ansi.Strip(d), "\n")
	if !strings.Contains(rows[len(rows)-1], "lines 1–10 of 76") {
		t.Errorf("the position isn't in the bottom edge: %q", rows[len(rows)-1])
	}
	if plain := Dialog(th, 46, 0, "Send reply", body, "", ""); lipgloss.Height(d) != lipgloss.Height(plain) {
		t.Errorf("the position took a row")
	}
}
