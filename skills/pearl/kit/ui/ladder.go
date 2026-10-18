package ui

import (
	"charm.land/lipgloss/v2"

	"tuikit/theme"
)

// Mark is a status slot's glyph in its role: the one colour each rung of the ladder has
// wherever the kit draws it. done is success, current the focus ink (primary, or fg
// under focus "fg"), todo fgMuted (a rung that is still to come, not a hint), partial
// and busy the working ink, pending fgSubtle, fail error, warn warning. A rung the
// theme's ladder names (Mark(t, "new") for {"glyph": "current", "meaning": "new",
// "role": "primary"}) draws that glyph in that role; the kit's slots keep their inks.
func Mark(t *theme.Theme, slot string) string {
	for _, r := range t.Ladder {
		if r.Meaning == slot {
			return RoleInk(t, r.Role).Render(t.LadderGlyph(r))
		}
	}
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

// RoleInk is the style a ladder rung's role draws in (theme.RoleStyle): a state role is
// the kit's state style, so a warning rung is a fill where the warning's slot is faint,
// as the warn slot is; focus and working are the inks expressions.focus and
// expressions.spinner resolve to; any other colour role is that colour; a name the theme
// doesn't know is fg.
func RoleInk(t *theme.Theme, role string) lipgloss.Style {
	return t.RoleStyle(role)
}

// Rung is one row of the status ladder: the slot (a kit slot, or for a theme's own rung
// the slot its glyph names, "" for a literal), what it means, the role it draws in, and
// for a theme's own rung the glyph resolved for this theme ("" for a kit slot).
type Rung struct{ Slot, Meaning, Role, Glyph string }

// Mark draws the rung's glyph in its role.
func (r Rung) Mark(t *theme.Theme) string {
	if r.Glyph != "" {
		return RoleInk(t, r.Role).Render(r.Glyph)
	}
	return Mark(t, r.Slot)
}

// DefaultRungs are the kit's slots in the order a thing moves through them, with the
// role each draws in (focus and working name the ink expressions.focus and
// expressions.spinner resolve to): the ladder of a theme that declares none.
var DefaultRungs = []Rung{
	{Slot: "todo", Meaning: "not started", Role: "fgMuted"}, {Slot: "pending", Meaning: "waiting", Role: "fgSubtle"},
	{Slot: "current", Meaning: "now", Role: "focus"}, {Slot: "partial", Meaning: "in progress", Role: "working"},
	{Slot: "done", Meaning: "finished", Role: "success"}, {Slot: "warn", Meaning: "degraded", Role: "warning"},
	{Slot: "fail", Meaning: "failed", Role: "error"},
}

// Rungs are the ladder's rungs for t: the theme's own (File.Ladder, each glyph resolved
// for this theme and profile) when it declares one, else DefaultRungs.
func Rungs(t *theme.Theme) []Rung {
	if len(t.Ladder) == 0 {
		return DefaultRungs
	}
	out := make([]Rung, len(t.Ladder))
	for i, r := range t.Ladder {
		slot := ""
		if _, ok := theme.GlyphSlot(t.G, r.Glyph); ok {
			slot = r.Glyph
		}
		out[i] = Rung{Slot: slot, Meaning: r.Meaning, Role: r.Role, Glyph: t.LadderGlyph(r)}
	}
	return out
}

// Ladder is the status ladder as a legend: every rung's glyph in its role in a two-cell
// column, its meaning, the slot it takes, and the role's name, then the spinner at
// frame n on a row of its own. A theme is judged here: each rung must read as its own
// state in every profile.
func Ladder(t *theme.Theme, n int) []string {
	rungs := Rungs(t)
	rows := make([]string, 0, len(rungs)+1)
	for _, r := range rungs {
		rows = append(rows, r.Mark(t)+" "+t.S.Base.Render(Fit(r.Meaning, 13))+t.S.Muted.Render(r.Slot)+t.S.Subtle.Render(" "+t.G.Sep+" "+r.Role))
	}
	return append(rows, Busy(t, n)+" "+t.S.Base.Render(Fit("working", 13))+t.S.Muted.Render("spinner")+t.S.Subtle.Render(" "+t.G.Sep+" "+t.X.Spinner))
}
