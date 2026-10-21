package main

import (
	"slices"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/ultraviolet/layout"

	"tuikit/ui/bubbletea"
)

// The rollout screen (enter on a deploy): the deploy's steps beside its log, two panes
// that scroll, each a ui.Pane with its own offsets. It is the kit's pattern for panes
// and the mouse (references/interaction.md § Mouse, § Panes That Scroll):
//
//   - the wheel scrolls the pane under the pointer and focus stays put; a click
//     focuses a pane, and on the steps selects the row (mouse);
//   - the wheel moves the view, never the selection; the next ↑↓ brings the view back
//     to the selection as it moves it (ui.Pane.Move);
//   - shift+wheel and the horizontal wheel scroll sideways, as ←→ do in the focused
//     pane, offered only while its content is wider than it (rolloutActions);
//   - each pane keeps its offsets across focus changes and resizes: fitPanes clamps
//     them on every layout and never resets them;
//   - mouse is on for this screen alone (View.MouseMode, mouseOn), and help names it
//     once, in a Mouse section after the keys (helpBody).

// the panes, in hit-test order
const (
	paneSteps = iota
	paneLog
)

// mouseOn reports whether the state takes the mouse: the rollout's panes. Everywhere
// else the terminal keeps its own text selection.
func (m *app) mouseOn() bool {
	return m.screen == scrRollout && m.w >= minW && m.h >= minH
}

// openRollout shows the selected deploy's rollout: the steps pane on the step that
// failed or runs (else the last), and the log on that step's lines.
func (m *app) openRollout() {
	m.screen, m.roll, m.pane = scrRollout, m.depSel, paneSteps
	m.sel = m.svcIndex(m.events[m.depSel].Svc)
	items, _, _ := m.rollout()
	sel := len(items) - 1
	for i, it := range items {
		if it.State == "error" || it.State == "busy" {
			sel = i
		}
	}
	m.panes = [2]ui.Pane{{Sel: sel}, {Sel: -1}}
	m.fitPanes()
	m.panes[paneSteps].Move(0) // the view follows the selection
	m.showInLog()
}

// showInLog scrolls the log to the selected step's first line, keeping its column.
func (m *app) showInLog() {
	_, _, first := m.rollout()
	if s := m.panes[paneSteps].Sel; s >= 0 && s < len(first) && first[s] >= 0 {
		m.panes[paneLog].ScrollTo(first[s])
	}
}

// rolloutBody is the body draw lays out on the rollout screen at the model's size:
// under the header, above the footer, with no tabs. The panes are fitted from it
// before anything is drawn, so a key or the wheel scrolls against the panes as drawn.
// (TestRolloutLayoutIsDrawn holds it to draw's.)
func (m *app) rolloutBody() uv.Rectangle {
	t := m.t
	g := t.X.Gutter
	y := len(ui.Header(t, m.w, m.h, t.AppName(), "")) + t.Gap()
	return uv.Rect(g, y, m.w-2*g, max(0, m.footerY()-y-t.Gap()))
}

// paneRects are each pane's region (its title and gutter included: what a click or the
// wheel hits) and its view (the rows ui.Pane scrolls), steps then log.
func (m *app) paneRects() (outer, view [2]uv.Rectangle) {
	body := m.rolloutBody()
	sw := min(30, max(0, body.Dx()-3)/2)
	layout.Horizontal(layout.Len(sw), layout.Len(3), layout.Fill(1)).Split(body).Assign(&outer[paneSteps], new(uv.Rectangle), &outer[paneLog])
	head := 1 + m.t.Gap() // the section title and its air
	for i, r := range outer {
		view[i] = uv.Rect(r.Min.X, r.Min.Y+head, r.Dx(), max(0, r.Dy()-head))
	}
	view[paneLog].Min.X += 2 // the log's gutter: the focus bar while it is focused
	return outer, view
}

// fitPanes tells each pane its content's size and its view's, clamping its offsets:
// on a resize, before every draw, and as the screen opens.
func (m *app) fitPanes() {
	if m.screen != scrRollout {
		return
	}
	_, view := m.paneRects()
	items, lines, first := m.rollout()
	widest := 0
	for i, l := range lines {
		if !slices.Contains(first, i) { // a step's header stays put: it never scrolls
			widest = max(widest, lipgloss.Width(l))
		}
	}
	m.panes[paneSteps].Fit(len(items), 0, view[paneSteps].Dy(), view[paneSteps].Dx())
	m.panes[paneLog].Fit(len(lines), widest, view[paneLog].Dy(), view[paneLog].Dx())
}

// mouse handles a click or a wheel event on the rollout: the pane under the pointer
// takes it. The wheel scrolls that pane (one notch per burst: ui.WheelGate) and leaves
// focus and the selection where they are; a left click focuses the pane, and on the
// steps selects the row. Under a dialog the page takes no mouse.
func (m *app) mouse(msg tea.MouseMsg) {
	if !m.mouseOn() || m.ov != ovNone {
		return
	}
	m.fitPanes()
	mo := msg.Mouse()
	outer, view := m.paneRects()
	i := ui.Hit(mo.X, mo.Y, outer[:]...)
	if i < 0 {
		return
	}
	p := &m.panes[i]
	switch msg.(type) {
	case tea.MouseWheelMsg:
		if m.wheel.Pass(mo, m.now()) && p.Wheel(mo) {
			m.last = "wheel"
		}
	case tea.MouseClickMsg:
		if mo.Button != tea.MouseLeft {
			return
		}
		m.pane, m.last = i, "click"
		if row := p.Top + mo.Y - view[i].Min.Y; p.Sel >= 0 && uv.Pos(mo.X, mo.Y).In(view[i]) && row < p.Top+p.Rows() {
			items, _, _ := m.rollout()
			if row < len(items) {
				p.Sel = row
			}
		}
	}
}

// rolloutActions is the rollout's action table: ↑↓ act in the focused pane, tab moves
// focus, ←→ scroll sideways only while the focused pane's content is wider than it.
func (m *app) rolloutActions(help, palette, quit, back action) []action {
	desc := "select"
	if m.pane == paneLog {
		desc = "scroll"
	}
	acts := []action{
		{bind([]string{"up", "k", "down", "j"}, m.t.G.UpDown, desc), true, func(m *app, msg tea.KeyPressMsg) tea.Cmd {
			d := 1
			if msg.String() == "up" || msg.String() == "k" {
				d = -1
			}
			m.panes[m.pane].Move(d)
			return nil
		}},
		{bind([]string{"tab"}, "tab", "next pane"), true, func(m *app, _ tea.KeyPressMsg) tea.Cmd { m.pane = 1 - m.pane; return nil }},
		{bind([]string{"shift+tab"}, "shift+tab", "previous pane"), false, func(m *app, _ tea.KeyPressMsg) tea.Cmd { m.pane = 1 - m.pane; return nil }},
	}
	if m.panes[m.pane].Wide() {
		acts = append(acts, action{bind([]string{"left", "right"}, m.t.G.LeftRight, "scroll sideways"), true, func(m *app, msg tea.KeyPressMsg) tea.Cmd {
			d := ui.KeyCols
			if msg.String() == "left" {
				d = -d
			}
			m.panes[m.pane].Scroll(0, d)
			return nil
		}})
	}
	// enter shows the selected step's lines, offered only for a step that wrote some
	if _, _, first := m.rollout(); m.pane == paneSteps && m.panes[paneSteps].Sel >= 0 && first[m.panes[paneSteps].Sel] >= 0 {
		acts = append(acts, action{bind([]string{"enter"}, "enter", "show in log"), true, func(m *app, _ tea.KeyPressMsg) tea.Cmd { m.showInLog(); return nil }})
	}
	return append(acts,
		action{bind([]string{"pgup"}, "pgup", "page up"), false, func(m *app, _ tea.KeyPressMsg) tea.Cmd { m.panes[m.pane].Page(-1); return nil }},
		action{bind([]string{"pgdown"}, "pgdn", "page down"), false, func(m *app, _ tea.KeyPressMsg) tea.Cmd { m.panes[m.pane].Page(1); return nil }},
		back, palette, quit, help)
}

// rollout is the shown deploy's rollout: each step as a list item (its mark, its
// time), the log its steps wrote, one styled line each, and each step's first line in
// that log (-1 for a step that wrote none). A failed deploy stops at its watch, the
// steps after it skipped; one in flight stops on the pod it is rolling.
func (m *app) rollout() (items []ui.Item, lines []string, first []int) {
	t := m.t
	e := m.events[min(m.roll, len(m.events)-1)]
	prev := m.svcs[m.svcIndex(e.Svc)].Prev
	if prev == "" || e.Svc != m.svcs[m.svcIndex(e.Svc)].Name {
		prev = "the last release"
	}
	fill := strings.NewReplacer("{svc}", e.Svc, "{ver}", e.Version, "{prev}", prev)
	stop, stopState := len(rolloutSteps), ""
	switch e.State {
	case "error":
		stop, stopState = slices.IndexFunc(rolloutSteps, func(s rstep) bool { return s.Name == "watch error rate" }), "error"
	case "busy":
		stop, stopState = slices.IndexFunc(rolloutSteps, func(s rstep) bool { return s.Name == "roll pod 2/3" }), "busy"
	}
	for i, s := range rolloutSteps {
		it := ui.Item{Title: s.Name, Right: s.Took, State: "ok"}
		log := s.Log
		switch {
		case i == stop:
			it.State = stopState
			if stopState == "busy" {
				it.Right, log = "", log[:1]
			}
		case i > stop:
			it.State, it.Right, log = "idle", "", nil
			if stopState == "error" {
				it.Right = "skipped"
			}
		case s.Name == "watch error rate": // a healthy watch
			log = []string{"info 5xx rate under 1.00% for 5m on {svc} {ver}; promoting"}
		case s.Log == nil:
			log = []string{"info " + s.Name + ": done"}
		}
		items = append(items, it)
		if len(log) == 0 {
			first = append(first, -1)
			continue
		}
		// a step's lines open on its name, as a CI log groups them; enter scrolls there, and
		// the name stays put when the log scrolls sideways (a fixed row of ui.Pane.View)
		first = append(first, len(lines))
		head := t.S.Bold.Render(s.Name)
		if it.Right != "" {
			head += t.S.Subtle.Render("  " + it.Right)
		}
		lines = append(lines, head)
		at, _ := time.Parse(time.TimeOnly, s.At)
		for j, raw := range log {
			lvl, msg, _ := strings.Cut(raw, " ")
			word, ink := t.S.Muted.Render("info"), t.S.Soft
			switch lvl {
			case "warn":
				word = t.S.Warn.Render("warn")
			case "error":
				word, ink = t.S.Err.Render("error"), t.S.Base
			}
			ts := at.Add(time.Duration(j) * time.Second).Format(time.TimeOnly)
			lines = append(lines, t.S.Subtle.Render(ts)+"  "+word+strings.Repeat(" ", 7-lipgloss.Width(lvl))+ink.Render(fill.Replace(msg)))
		}
	}
	return items, lines, first
}

// rolloutScreen draws the two panes: each a section title, then its view. The steps
// pane is a ui.List at the pane's offset (its selection marks its focus); the log is
// the pane's View behind a gutter that carries the focus bar while the log is focused.
func (m *app) rolloutScreen(c *ui.Canvas) {
	t := m.t
	outer, view := m.paneRects()
	items, lines, first := m.rollout()
	focused := m.ov == ovNone // under a dialog no pane has the focus
	done := 0
	for _, it := range items {
		if it.State == "ok" {
			done++
		}
	}
	st, lg := m.panes[paneSteps], m.panes[paneLog]
	c.Draw(ui.Section(t, "Steps", outer[paneSteps].Dx(), itoa(done)+"/"+itoa(len(items))), uv.Rect(outer[paneSteps].Min.X, outer[paneSteps].Min.Y, outer[paneSteps].Dx(), 1))
	list := ui.List(t, items, st.Sel, st.Top, view[paneSteps].Dx(), view[paneSteps].Dy(), focused && m.pane == paneSteps)
	c.Draw(strings.Join(list, "\n"), view[paneSteps])

	info := itoa(len(lines)) + " lines"
	if lg.Left > 0 {
		info += " " + t.G.Sep + " col " + itoa(lg.Left+1)
	}
	c.Draw(ui.Section(t, "Log", outer[paneLog].Dx(), info), uv.Rect(outer[paneLog].Min.X, outer[paneLog].Min.Y, outer[paneLog].Dx(), 1))
	gutter := ui.Gutter(t, focused && m.pane == paneLog, t.S.FocusBar)
	rows := lg.View(t, lines, first...) // each step's header stays put when the log scrolls sideways
	for i := range rows {
		rows[i] = gutter + rows[i]
	}
	c.Draw(strings.Join(rows, "\n"), uv.Rect(outer[paneLog].Min.X, view[paneLog].Min.Y, outer[paneLog].Dx(), view[paneLog].Dy()))
}
