package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"

	"tuikit/theme"
)

// DialogTitle is the title row of a dialog, inner w cells, per
// expressions.dialogTitle:
//
//	rule   Commands ╱╱╱╱╱╱╱╱╱╱╱  (the header rule glyph in a primary→secondary gradient)
//	plain  Commands
//	bar    the title on a filled strip
func DialogTitle(t *theme.Theme, title string, w int, info string) string {
	ti := t.S.Title.Render(t.Case(title))
	if info != "" {
		info = " " + info
	}
	switch t.X.DialogTitle {
	case "plain":
		return Fit(ti+info, w)
	case "bar":
		st := lipgloss.NewStyle().Background(t.P.Primary).Foreground(t.P.OnPrimary).Bold(true)
		return st.Render(Fit(" "+t.Case(title)+ansi.Strip(info), w))
	}
	g := t.G.HeaderRule
	if g == "" {
		g = t.G.Rule
	}
	n := w - lipgloss.Width(ti) - 1 - lipgloss.Width(info)
	if n < 2 {
		return Fit(ti+info, w)
	}
	rule := strings.Repeat(g, n)
	if t.ASCII || t.X.Focus == "fg" {
		rule = t.S.Motif.Render(rule)
	} else {
		rule = theme.Grad(rule, t.P.Primary, t.P.Secondary, false)
	}
	return ti + " " + rule + info
}

// Dialog is the one boxed element: title, body rows, an optional button row
// and a help line, inside the theme's dialog border. w is the outer width.
// Under expressions.dialog "rules" the box is a filled panel between two rules,
// the title set in the top one:
//
//	── Roll back checkout-api? ─────────────
//	 body…
//	────────────────────────────────────────
func Dialog(t *theme.Theme, w int, title string, body []string, buttons string, help string) string {
	st := t.S.Dialog
	inner := w - st.GetHorizontalFrameSize()
	var rows []string
	if t.X.Dialog != "rules" {
		rows = append(rows, DialogTitle(t, title, inner, ""), "")
	}
	for _, b := range body {
		rows = append(rows, Fit(b, inner))
	}
	if buttons != "" {
		rows = append(rows, "", Fit(buttons, inner))
	}
	if help != "" {
		rows = append(rows, "", Fit(help, inner))
	}
	if t.X.Dialog == "rules" {
		pw := w - st.GetHorizontalMargins()
		top, bottom := dialogRules(t, title, pw)
		panel := top + "\n" + st.UnsetMargins().Render(strings.Join(rows, "\n")) + "\n" + bottom
		return lipgloss.NewStyle().Margin(st.GetMargin()).MarginBackground(t.P.Bg).Render(panel)
	}
	return st.Render(strings.Join(rows, "\n"))
}

// dialogRules are a rules dialog's top and bottom edges, w cells each: the theme's
// rule glyph ("-" under ASCII) in the dialog's border colour on the panel, the title
// set into the top one after a two-glyph lead.
func dialogRules(t *theme.Theme, title string, w int) (string, string) {
	g := t.G.Rule
	if t.ASCII {
		g = "-"
	}
	on := lipgloss.NewStyle().Background(t.P.BgPanel)
	rule := func(n int) string {
		return on.Foreground(t.Ink(t.P.BorderFocus, t.P.BgPanel)).Render(strings.Repeat(g, max(0, n)))
	}
	ti := t.S.Title.Background(t.P.BgPanel).Render(t.Case(title))
	lead := 2
	n := w - lead - 2 - lipgloss.Width(ti)
	top := rule(lead) + on.Render(" ") + ti + on.Render(" ") + rule(n)
	if n < 1 {
		top = Fit(rule(lead)+on.Render(" ")+ti, w)
	}
	return top, rule(w)
}

// Overlay draws a dialog centred on the canvas (or at anchor when non-nil),
// dimming what is under it when the theme asks for a scrim. Draw it last:
// draw order is z-order.
func (c *Canvas) Overlay(dialog string, anchor *uv.Position) uv.Rectangle {
	if c.T.X.Scrim == "dim" && !c.T.ASCII {
		c.Scrim(0.55)
	}
	dw, dh := lipgloss.Width(dialog), lipgloss.Height(dialog)
	b := c.Bounds()
	x := (b.Dx() - dw) / 2
	y := max(1, (b.Dy()-dh)/2-1)
	if anchor != nil {
		y = anchor.Y
		if anchor.X >= 0 {
			x = anchor.X
		}
	}
	r := uv.Rect(x, y, dw, dh)
	c.Draw(dialog, r)
	// cells the dialog left without a background would show the scrim through (a rules
	// dialog's margin carries the page's bg, so it stays a clean gap)
	bg := c.T.P.Bg
	if c.T.X.Dialog == "none" || c.T.X.Dialog == "rules" {
		bg = c.T.P.BgPanel
	}
	c.Backfill(r, bg)
	return r
}
