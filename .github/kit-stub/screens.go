package main

import (
	"strings"

	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"

	"example.com/brandapp/internal/ui"
)

const minW, minH = 60, 18

func (m *app) render(w, h int) *ui.Canvas {
	m.w, m.h = w, h
	t := m.t
	c := ui.NewCanvas(t, w, h)
	if w < minW || h < minH {
		// the app's own sentence, and the toosmall action table's keys under it
		c.Draw(ui.TooSmallText(t, w, h, "Make the window "+itoa(minW)+"×"+itoa(minH)+" or larger.", m.footerBindings()), c.Bounds())
	} else {
		head := ui.Header(t, w, h, t.AppName(), ui.Details(t, "items", m.items[m.sel]))
		c.Draw(strings.Join(head, "\n"), ui.Rect(0, 0, w, len(head)))
		foot := ui.Footer(t, w, "", m.footerBindings(), "")
		fy := h - len(foot)
		c.Draw(strings.Join(foot, "\n"), ui.Rect(0, fy, w, len(foot)))
		g := t.X.Gutter
		y := len(head) + 1
		body := uv.Rect(g, y, w-2*g, max(0, fy-y-1))
		m.body, m.footY = body, fy
		focused := m.ov == ovNone
		if m.screen == scrDetail {
			c.Draw(strings.Join(ui.KV(t, [][2]string{{"name", t.S.Base.Render(m.items[m.sel])}, {"state", t.S.Soft.Render("ready")}}, 8), "\n"), body)
		} else {
			var items []ui.Item
			for _, it := range m.items {
				items = append(items, ui.Item{Title: it})
			}
			c.Draw(strings.Join(ui.List(t, items, m.sel, 0, body.Dx(), body.Dy(), focused), "\n"), body)
		}
		switch m.ov {
		case ovConfirm:
			btns := ui.Buttons(t, 40, lipgloss.Right, ui.Button(t, "Remove", m.btn == 0, true, 0), ui.Button(t, "Cancel", m.btn == 1, false, 0))
			d := ui.Dialog(t, 46, fy-3, "Remove "+m.items[m.sel]+"?", []string{t.S.Soft.Render("It goes from the list.")}, btns, "")
			m.ovRect = c.Overlay(d, &uv.Position{X: -1, Y: max(3, min((h-lipgloss.Height(d))/2-1, fy-lipgloss.Height(d)))})
		case ovHelp:
			body := m.helpBody()
			m.helpOff = min(m.helpOff, ui.ClipMax(len(body), m.helpRows()))
			d := ui.DialogScrolled(t, min(50, w-6), fy-3, "Keys", body, m.helpOff, "", "")
			m.ovRect = c.Overlay(d, &uv.Position{X: -1, Y: max(3, min((h-lipgloss.Height(d))/2-1, fy-lipgloss.Height(d)))})
		}
		if m.ov != ovNone {
			c.Draw(strings.Join(foot, "\n"), ui.Rect(0, fy, w, len(foot)))
		}
	}
	if t.ASCII {
		c.ASCIIfy()
	}
	return c
}

func (m *app) helpBody() []string {
	ov := m.ov
	m.ov = m.helpFrom
	acts := m.actions()
	m.ov = ov
	var body []string
	for _, a := range acts {
		h := a.b.Help()
		body = append(body, m.t.S.Key.Render(ui.Fit(h.Key, 12))+m.t.S.Soft.Render(h.Desc))
	}
	return body
}

func (m *app) helpRows() int {
	return ui.DialogBodyRows(m.t, m.h-len(ui.Footer(m.t, m.w, "", nil, ""))-3, "Keys", "", "")
}
