package ui

import "tuikit/theme"

// Mark is a status slot's glyph in its role: the one colour each rung of the ladder has
// wherever the kit draws it. done is success, current the focus ink (primary, or fg
// under focus "fg"), todo fgMuted (a rung that is still to come, not a hint), partial
// and busy the working ink, pending fgSubtle, fail error, warn warning.
func Mark(t *theme.Theme, slot string) string {
	g := t.G
	switch slot {
	case "done":
		done := g.Done
		if t.ASCII && done == g.Current { // done and current must differ in shape, not only in colour
			done = "+"
		}
		return t.S.OK.Render(done)
	case "current":
		return t.S.Focus.Bold(true).Render(g.Current)
	case "partial", "busy":
		return t.S.Working.Render(g.Partial)
	case "pending":
		return t.S.Subtle.Render(g.Pending)
	case "fail":
		return t.S.Err.Render(g.Fail)
	case "warn":
		return t.S.Warn.Render(g.Warn)
	}
	return t.S.Muted.Render(g.Todo)
}

// Rung is one row of the status ladder: the slot, what it means, and the role it draws in.
type Rung struct{ Slot, Meaning, Role string }

// Rungs are the ladder's slots in the order a thing moves through them, with the role
// each draws in (focus and working name the ink expressions.focus and expressions.spinner
// resolve to).
var Rungs = []Rung{
	{"todo", "not started", "fgMuted"}, {"pending", "waiting", "fgSubtle"}, {"current", "now", "focus"},
	{"partial", "in progress", "working"}, {"done", "finished", "success"}, {"warn", "degraded", "warning"},
	{"fail", "failed", "error"},
}

// Ladder is the status ladder as a legend: every rung's glyph in its role in a two-cell
// column, its meaning, and the role's name, then the spinner at frame n on a row of its
// own. A theme is judged here: each rung must read as its own state in every profile.
func Ladder(t *theme.Theme, n int) []string {
	rows := make([]string, 0, len(Rungs)+1)
	for _, r := range Rungs {
		rows = append(rows, Mark(t, r.Slot)+" "+t.S.Base.Render(Fit(r.Meaning, 13))+t.S.Muted.Render(r.Slot)+t.S.Subtle.Render(" "+t.G.Sep+" "+r.Role))
	}
	return append(rows, Busy(t, n)+" "+t.S.Base.Render(Fit("working", 13))+t.S.Muted.Render("spinner")+t.S.Subtle.Render(" "+t.G.Sep+" "+t.X.Spinner))
}
