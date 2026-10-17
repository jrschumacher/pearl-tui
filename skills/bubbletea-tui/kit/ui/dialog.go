package ui

import (
	"image/color"
	"strconv"
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
// and a help line, inside the theme's dialog border. w is the outer width; h is the
// height the dialog may take (0: no limit): pass the rows between the header and the
// footer, and a body that doesn't fit is cut to them with Clip (a "… n more" row at
// the bottom), so no dialog runs under the footer. A screen that scrolls the body
// draws DialogScrolled. Under expressions.dialog "rules" the box is a filled panel
// between two rules, the title set in the top one:
//
//	── Roll back checkout-api? ─────────────
//	 body…
//	────────────────────────────────────────
func Dialog(t *theme.Theme, w, h int, title string, body []string, buttons string, help string) string {
	return DialogScrolled(t, w, h, title, body, 0, buttons, help)
}

// DialogScrolled is Dialog with its body scrolled off rows when the body doesn't fit
// in h (Clip: an "↑ n more" row above, a "… n more" row below). ClipMax(len(body),
// DialogBodyRows(t, h, …)) is the last offset that still shows a row.
func DialogScrolled(t *theme.Theme, w, h int, title string, body []string, off int, buttons string, help string) string {
	return dialog(t, t.S.Dialog, t.P.BgPanel, w, h, title, body, off, buttons, help)
}

// Popover is a Dialog on bgOverlay, the ramp's step for what pops over the page: a
// command palette, completions. Draw it with Canvas.Popover.
func Popover(t *theme.Theme, w, h int, title string, body []string, buttons string, help string) string {
	return dialog(t, t.S.Popover, t.P.BgOverlay, w, h, title, body, 0, buttons, help)
}

// DialogBodyRows is how many body rows a Dialog h rows tall shows: h less the rows its
// frame takes (edge, padding, the title block, the button row and the help line when
// set). A screen with scroll state clamps its offset with ClipMax against it.
func DialogBodyRows(t *theme.Theme, h int, title, buttons, help string) int {
	return h - dialogFrame(t, t.S.Dialog, t.P.BgPanel, 20, title, buttons, help)
}

// dialogFrame is the rows a dialog takes beyond its body.
func dialogFrame(t *theme.Theme, st lipgloss.Style, fill color.Color, w int, title, buttons, help string) int {
	return lipgloss.Height(dialog(t, st, fill, w, 0, title, []string{""}, 0, buttons, help)) - 1
}

func dialog(t *theme.Theme, st lipgloss.Style, fill color.Color, w, h int, title string, body []string, off int, buttons string, help string) string {
	if h > 0 {
		if avail := h - dialogFrame(t, st, fill, w, title, buttons, help); len(body) > avail {
			body = Clip(t, body, avail, off)
		}
	}
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
	// the panel is a fill: a hint level that can't hold on it moves up a level
	block := OnPanel(t, strings.Join(rows, "\n"), fill)
	if t.X.Dialog == "rules" {
		pw := w - st.GetHorizontalMargins()
		top, bottom := dialogRules(t, title, pw, fill)
		panel := top + "\n" + st.UnsetMargins().Render(block) + "\n" + bottom
		return lipgloss.NewStyle().Margin(st.GetMargin()).MarginBackground(t.P.Bg).Render(panel)
	}
	return st.Render(block)
}

// Clip cuts rows to h: all of them when they fit, else the rows from off on, under an
// "↑ off more" row when off > 0 and over a "… n more" row when rows follow, so what is
// hidden is said in both directions. off is clamped to ClipMax, so the last page still
// shows a row. The markers are the theme's more and ellipsis glyphs in subtle; on a
// panel, OnPanel lifts them as it does any hint.
func Clip(t *theme.Theme, rows []string, h, off int) []string {
	if len(rows) <= h {
		return rows
	}
	if h <= 0 {
		return nil
	}
	off = max(0, min(off, ClipMax(len(rows), h)))
	n := h
	var out []string
	if off > 0 {
		out = append(out, t.S.Subtle.Render(t.G.More+" "+strconv.Itoa(off)+" more"))
		n--
	}
	rest := rows[off:]
	below := 0
	if len(rest) > n {
		n--
		below = len(rest) - n
		rest = rest[:n]
	}
	out = append(out, rest...)
	if below > 0 {
		out = append(out, t.S.Subtle.Render(t.G.Ellipsis+" "+strconv.Itoa(below)+" more"))
	}
	return out
}

// ClipMax is the largest offset Clip shows a row at for n rows in h: 0 when they fit.
func ClipMax(n, h int) int {
	if n <= h {
		return 0
	}
	return max(0, n-(h-1))
}

// dialogRules are a rules dialog's top and bottom edges, w cells each: the theme's
// rule glyph ("-" under ASCII) in the dialog's border colour on the panel, the title
// set into the top one after a two-glyph lead.
func dialogRules(t *theme.Theme, title string, w int, fill color.Color) (string, string) {
	g := t.G.Rule
	if t.ASCII {
		g = "-"
	}
	on := lipgloss.NewStyle().Background(fill)
	rule := func(n int) string {
		return on.Foreground(t.Ink(t.P.BorderFocus, fill)).Render(strings.Repeat(g, max(0, n)))
	}
	ti := t.S.Title.Background(fill).Render(t.Case(title))
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
// draw order is z-order. There is no scrim under ASCII, at 16 colours or with no colour:
// with the page on ANSI 0 or 15, dimming toward it sends every text level to the page's
// own slot and the scrim's text vanishes instead of fading. Where no scrim is drawn
// (those, and scrim: none) the rows the dialog spans are cleared across instead; under
// a scrim, a phrase the dialog cuts is cleared to its column gap. Either way the page's
// focus marks go and no page rule runs on beside the dialog's edges (overlay.go).
func (c *Canvas) Overlay(dialog string, anchor *uv.Position) uv.Rectangle {
	return c.overlay(dialog, anchor, c.T.P.BgPanel)
}

// Popover is Overlay for a Popover: its cells fill with bgOverlay.
func (c *Canvas) Popover(dialog string, anchor *uv.Position) uv.Rectangle {
	return c.overlay(dialog, anchor, c.T.P.BgOverlay)
}

// overlay draws dialog and fills the cells it left without a background: a dialog is a
// filled panel whatever its edge, and whether or not the app paints the terminal
// (paintBg), or the page's colour would show through it.
func (c *Canvas) overlay(dialog string, anchor *uv.Position, fill color.Color) uv.Rectangle {
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
	scrim := c.scrims()
	c.clearPage(r, dialog, !scrim)
	if scrim {
		c.Scrim(0.55)
	}
	c.Draw(dialog, r)
	// cells the dialog left without a background would show the scrim through (a rules
	// dialog's margin carries the page's bg, so it stays a clean gap)
	c.Backfill(r, fill)
	return r
}
