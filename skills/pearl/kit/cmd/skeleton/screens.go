package main

import (
	"strings"

	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"

	"github.com/jrschumacher/pearl-tui/skills/pearl/kit/ui/bubbletea"
)

// minW×minH is your app's floor: below it the too-small screen says what to do.
const minW, minH = 60, 18

// render draws the whole screen at w×h; View and the harness share it, so what a frame
// shows is what the terminal shows. Each region is drawn into its own rectangle:
// header and footer first, the body gets what is left, overlays last.
func (m *app) render(w, h int) *ui.Canvas {
	m.w, m.h = w, h
	t := m.t
	c := ui.NewCanvas(t, w, h)
	if w < minW || h < minH {
		c.Draw(ui.TooSmall(t, w, h, minW, minH, m.footerBindings()), c.Bounds())
	} else {
		m.draw(c, w, h)
	}
	if t.ASCII {
		c.ASCIIfy() // nothing outside ASCII survives, whatever a region drew
	}
	return c
}

func (m *app) draw(c *ui.Canvas, w, h int) {
	t := m.t
	details := ui.Details(t, "items", "none")
	if len(m.items) > 0 {
		details = ui.Details(t, "items", m.items[m.sel].Name)
	}
	head := ui.Header(t, w, h, t.AppName(), details)
	c.Draw(strings.Join(head, "\n"), ui.Rect(0, 0, w, len(head)))
	foot := ui.Footer(t, w, "", m.footerBindings(), "")
	fy := h - len(foot)
	c.Draw(strings.Join(foot, "\n"), ui.Rect(0, fy, w, len(foot)))

	g, y := t.X.Gutter, len(head)+t.Gap()
	body := uv.Rect(g, y, w-2*g, max(0, fy-y-t.Gap()))
	m.body, m.footY = body, fy
	c.Body = body // the rows a dialog hides where the page can't be dimmed
	// your regions: one method per screen, each drawing into the body it is given
	switch {
	case len(m.items) == 0:
		c.Draw(strings.Join(ui.Empty(t, body.Dx(), "", "Nothing here", "Every item was removed.", t.S.Key.Render("q")+t.S.Muted.Render(" to quit")), "\n"), body)
	case m.screen == scrDetail:
		m.detail(c, body)
	default:
		m.list(c, body)
	}
	switch m.ov {
	case ovConfirm:
		m.confirm(c, fy)
	case ovHelp:
		m.keysOverlay(c, fy)
	}
	if m.ov != ovNone {
		// the footer now names the dialog's keys: it sits above the scrim
		c.Draw(strings.Join(foot, "\n"), ui.Rect(0, fy, w, len(foot)))
	}
}

func (m *app) list(c *ui.Canvas, r uv.Rectangle) {
	t := m.t
	c.Draw(ui.Section(t, "Items", r.Dx(), ui.Count(len(m.items))), ui.Rect(r.Min.X, r.Min.Y, r.Dx(), 1))
	var items []ui.Item
	for _, it := range m.items {
		items = append(items, ui.Item{Title: it.Name, Desc: it.Desc, Right: it.Updated, State: it.State})
	}
	rows := r.Dy() - 2
	off := ui.Follow(m.sel, 0, rows, len(items))
	c.Draw(strings.Join(ui.List(t, items, m.sel, off, r.Dx(), rows, m.ov == ovNone), "\n"), ui.Rect(r.Min.X, r.Min.Y+2, r.Dx(), rows))
}

func (m *app) detail(c *ui.Canvas, r uv.Rectangle) {
	t := m.t
	it := m.items[m.sel]
	c.Draw(ui.Section(t, it.Name, r.Dx(), ui.Dot(t, it.State)+" "+it.State), ui.Rect(r.Min.X, r.Min.Y, r.Dx(), 1))
	rows := append([]string{t.S.Base.Render(it.Desc), ""}, ui.KV(t, [][2]string{
		{"owner", t.S.Base.Render(it.Owner)},
		{"updated", t.S.Soft.Render(it.Updated)},
	}, 8)...)
	c.Draw(strings.Join(rows, "\n"), ui.Rect(r.Min.X, r.Min.Y+2, r.Dx(), r.Dy()-2))
}

// dialogAt keeps a dialog of height dh below the header and above the footer at fy.
func dialogAt(c *ui.Canvas, dh, fy int) *uv.Position {
	return &uv.Position{X: -1, Y: min(max(3, (c.Bounds().Dy()-dh)/2-1), max(0, fy-dh))}
}

// confirm asks before a destructive action; the safe button has the focus on open.
func (m *app) confirm(c *ui.Canvas, fy int) {
	t := m.t
	w := min(52, c.Bounds().Dx()-6)
	inner := w - t.S.Dialog.GetHorizontalFrameSize()
	btns := ui.Buttons(t, inner, lipgloss.Right,
		ui.Button(t, "Remove", m.btn == btnRemove, true, -1), ui.Button(t, "Cancel", m.btn == btnCancel, false, -1))
	it := m.items[m.sel]
	body := []string{t.S.Soft.Render(it.Name + " leaves the list. This can't be undone.")}
	d := ui.Dialog(t, w, fy-3, "Remove "+it.Name+"?", body, btns, "")
	m.ovRect = c.Overlay(d, dialogAt(c, lipgloss.Height(d), fy))
}

// helpBody is every binding of the state help was opened over, then ctrl+c.
func (m *app) helpBody() []string {
	t := m.t
	ov := m.ov
	m.ov = m.helpFrom
	acts := m.actions()
	m.ov = ov
	var body []string
	row := func(k, desc string) string { return t.S.Key.Render(ui.Fit(k, 12)) + t.S.Soft.Render(desc) }
	for _, a := range acts {
		body = append(body, row(a.b.Help().Key, a.b.Help().Desc))
	}
	return append(body, row("ctrl+c", "quit from anywhere"))
}

func (m *app) helpRows() int {
	return ui.DialogBodyRows(m.t, m.h-len(ui.Footer(m.t, m.w, "", nil, ""))-3, "Keys", "", "")
}

func (m *app) keysOverlay(c *ui.Canvas, fy int) {
	body := m.helpBody()
	m.helpOff = min(m.helpOff, ui.ClipMax(len(body), m.helpRows()))
	d := ui.DialogScrolled(m.t, min(50, c.Bounds().Dx()-6), fy-3, "Keys", body, m.helpOff, "", "")
	m.ovRect = c.Overlay(d, dialogAt(c, lipgloss.Height(d), fy))
}
