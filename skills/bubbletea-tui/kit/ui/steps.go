package ui

import (
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"tuikit/theme"
)

// Steps is a wizard's progress, drawn with the theme's done / current / todo
// glyphs. Count it every screen of the flow, including the ones after the
// form (signing in, finishing): a counter that sits on "5/5" for four screens
// tells the user nothing is moving.
type Steps struct {
	Names  []string
	Cur    int  // the current step (0-based); len(Names) once all are done
	Failed bool // the current step failed: the fail glyph in the error colour
}

// Count is the compact form for a header: "2/6 Sign-in". Plain text, so a
// filled header can carry it. No separator glyph: a brand whose separator is
// "/" would make it "2/6 / Sign-in".
func (s Steps) Count() string {
	n := len(s.Names)
	if s.Cur >= n {
		return strconv.Itoa(n) + "/" + strconv.Itoa(n) + " done"
	}
	return strconv.Itoa(s.Cur+1) + "/" + strconv.Itoa(n) + " " + s.Names[s.Cur]
}

// mark is one step's glyph and label, styled by its state.
func (s Steps) mark(t *theme.Theme, i int) (glyph, label string) {
	switch {
	case i < s.Cur:
		return Mark(t, "done"), t.S.Muted.Render(s.Names[i])
	case i == s.Cur && s.Failed:
		return t.S.Err.Bold(true).Render(t.G.Fail), t.S.Err.Bold(true).Render(s.Names[i])
	case i == s.Cur:
		return t.S.Focus.Bold(true).Render(t.G.Current), t.S.Bold.Render(s.Names[i])
	}
	return Mark(t, "todo"), t.S.Subtle.Render(s.Names[i])
}

// Row is the full progress on one line, at most w cells:
//
//	◆ Cluster ┄ ◆ Sign-in ┄ ▲ Alerts ┄ △ Review ┄ △ Connect ┄ △ Finish
//
// When that doesn't fit it drops the connectors, then the names of every step
// but the current one, then falls back to Count. It never wraps.
func (s Steps) Row(t *theme.Theme, w int) string {
	join := func(names bool, sep string) string {
		parts := make([]string, len(s.Names))
		for i := range s.Names {
			g, l := s.mark(t, i)
			if names || i == s.Cur {
				parts[i] = g + " " + l
			} else {
				parts[i] = g
			}
		}
		return strings.Join(parts, sep)
	}
	// longer in ASCII: "--" reads as a flag and "..." as an ellipsis; "---" and "...."
	// read as rules (and tuilint agrees)
	n := 2
	if t.ASCII {
		n = 3
		if t.G.Rule == "." {
			n = 4
		}
	}
	conn := " " + t.S.Rule.Render(strings.Repeat(t.G.Rule, n)) + " "
	for _, try := range []string{join(true, conn), join(true, "   "), join(true, "  "), join(false, " ")} {
		if lipgloss.Width(try) <= w {
			return try
		}
	}
	return Trunc(t, t.S.Soft.Render(s.Count()), w)
}

// List is the full progress as rows, one per step: for a side rail or a
// review screen.
func (s Steps) List(t *theme.Theme) []string {
	out := make([]string, len(s.Names))
	for i := range s.Names {
		g, l := s.mark(t, i)
		out[i] = g + " " + l
	}
	return out
}

// SummaryRow is one decision in a summary: a label over its value.
type SummaryRow struct {
	Label string
	Value string
	Set   bool // the user chose it. false = still the default: drawn subtle and marked "default"
}

// Summary is the panel beside a form that shows what the form will do:
//
//	Summary ┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄ DRAFT
//	Cluster
//	staging-eu-west-1
//	Namespace                       default
//	default
//
// Labels are muted, values fg; a value the user hasn't chosen is subtle and
// says "default", so a default never reads as a decision. Values truncate by
// cells with the theme's ellipsis: a summary never wraps a value mid-word.
// tag is a pre-rendered ui.Tag (DRAFT, SAVED) or "".
func Summary(t *theme.Theme, title, tag string, rows []SummaryRow, w int) []string {
	head := Section(t, title, w, "")
	if tag != "" {
		head = Section(t, title, max(0, w-lipgloss.Width(tag)-1), "") + " " + tag
	}
	out := []string{head, ""}
	def := t.S.Subtle.Render("default")
	for i, r := range rows {
		label := t.S.Muted.Render(Trunc(t, r.Label, w))
		val := t.S.Base.Render(Trunc(t, r.Value, w))
		if !r.Set {
			label = t.S.Muted.Render(Trunc(t, r.Label, max(1, w-9)))
			label += strings.Repeat(" ", max(1, w-lipgloss.Width(label)-7)) + def
			val = t.S.Subtle.Render(Trunc(t, r.Value, w))
		}
		out = append(out, label, val)
		if i < len(rows)-1 {
			out = append(out, "")
		}
	}
	return out
}

// SummaryStrip is the summary on one line, for terminals too narrow for the
// panel: "cluster staging-eu-west-1 · auth browser · …", defaults subtle,
// cut by cells with the ellipsis.
func SummaryStrip(t *theme.Theme, rows []SummaryRow, w int) string {
	parts := make([]string, len(rows))
	for i, r := range rows {
		v := t.S.Base.Render(r.Value)
		if !r.Set {
			v = t.S.Subtle.Render(r.Value)
		}
		parts[i] = t.S.Muted.Render(t.Label(r.Label, "lower")) + " " + v
	}
	line := strings.Join(parts, t.S.Subtle.Render(" "+t.G.Sep+" "))
	if lipgloss.Width(line) > w {
		e := t.G.Ellipsis
		line = ansi.Truncate(line, max(0, w-lipgloss.Width(e)), "") + t.S.Subtle.Render(e)
	}
	return line
}
