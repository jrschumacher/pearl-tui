package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"tuikit/theme/bubbletea"
	"tuikit/ui/bubbletea"
)

// The rollout's tests drive the root model with the messages a terminal sends (wheel
// and click events at a cell, keys, resizes) through Update, and read the panes and
// the frame. The wheel's clock is the test's: each notch arrives 100 ms after the last
// unless a test sends a burst.

// rolloutModel is state s at w×h, as the terminal would have it: sized by a
// WindowSizeMsg, drawn once, on a clock the test moves.
func rolloutModel(t *testing.T, s string, w, h int) (*app, *time.Time) {
	t.Helper()
	m := fixture(t, s)
	clock := time.Date(2026, 10, 21, 14, 0, 0, 0, time.UTC)
	m.now = func() time.Time { return clock }
	m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	m.render(w, h)
	if m.screen != scrRollout {
		t.Fatalf("%s is not the rollout screen", s)
	}
	return m, &clock
}

// at is a cell inside pane i: its view's middle row, a few cells in.
func at(m *app, i int) (int, int) {
	_, view := m.paneRects()
	return view[i].Min.X + 3, view[i].Min.Y + view[i].Dy()/2
}

// notch sends one wheel notch at x, y, 100 ms after the last.
func notch(m *app, clock *time.Time, x, y int, b tea.MouseButton, mod tea.KeyMod) {
	*clock = clock.Add(100 * time.Millisecond)
	m.Update(tea.MouseWheelMsg{X: x, Y: y, Button: b, Mod: mod})
}

func keyMsg(m *app, k string) { m.Update(press(k)) }

// The wheel scrolls the pane under the pointer and focus stays where it was: with the
// steps focused, the wheel over the log scrolls the log, and the steps, the selection
// and the focus don't move; over the steps it scrolls the steps; outside both panes (the
// header) it does nothing. The frame shows the log's new rows.
func TestWheelScrollsThePaneUnderThePointer(t *testing.T) {
	m, clock := rolloutModel(t, "rollout", 80, 24)
	before := rowsOf(m.render(80, 24), m)
	steps, log, sel := m.panes[paneSteps], m.panes[paneLog], m.panes[paneSteps].Sel
	x, y := at(m, paneLog)
	notch(m, clock, x, y, tea.MouseWheelUp, 0)
	if m.panes[paneLog].Top != log.Top-ui.WheelRows {
		t.Errorf("the wheel up over the log: its top %d, want %d", m.panes[paneLog].Top, log.Top-ui.WheelRows)
	}
	if m.pane != paneSteps || m.panes[paneSteps] != steps || m.panes[paneSteps].Sel != sel {
		t.Errorf("the wheel over the log moved focus (%d) or the steps (%+v, was %+v)", m.pane, m.panes[paneSteps], steps)
	}
	if after := rowsOf(m.render(80, 24), m); after == before {
		t.Error("the wheel scrolled the log, but the frame is the same")
	}
	m, clock = rolloutModel(t, "rollout", 60, 18) // the floor: the steps overflow by more
	x, y = at(m, paneSteps)
	top := m.panes[paneSteps].Top
	notch(m, clock, x, y, tea.MouseWheelDown, 0)
	if m.panes[paneSteps].Top == top {
		t.Error("the wheel over the steps didn't scroll them")
	}
	still := m.panes
	notch(m, clock, x, 0, tea.MouseWheelDown, 0) // the header row: no pane
	if m.panes != still {
		t.Error("the wheel over the header scrolled a pane")
	}
}

// rowsOf is the frame's body rows as text.
func rowsOf(c *ui.Canvas, m *app) string {
	var rows []string
	for y := m.body.Min.Y; y < m.body.Max.Y; y++ {
		rows = append(rows, rowText(c, y))
	}
	return strings.Join(rows, "\n")
}

// The wheel moves the view, never the selection: the steps scrolled past the selected
// step keep it selected, out of sight, and the next ↓ brings the view back to it as it
// moves to the next step, and the frame draws that step selected.
func TestWheelNeverMovesTheSelection(t *testing.T) {
	m, clock := rolloutModel(t, "rollout", 60, 18)
	sel := m.panes[paneSteps].Sel
	x, y := at(m, paneSteps)
	for range 6 {
		notch(m, clock, x, y, tea.MouseWheelUp, 0)
	}
	if m.panes[paneSteps].Sel != sel {
		t.Fatalf("the wheel changed the selection: %d, was %d", m.panes[paneSteps].Sel, sel)
	}
	if m.panes[paneSteps].Visible(sel) {
		t.Fatalf("the steps scrolled up 6 notches still show step %d: the test proves nothing", sel)
	}
	keyMsg(m, "up")
	p := m.panes[paneSteps]
	if p.Sel != sel-1 || !p.Visible(p.Sel) {
		t.Errorf("↑ after the wheel: selection %d (visible %v), want %d brought into view", p.Sel, p.Visible(p.Sel), sel-1)
	}
	c := m.render(60, 18)
	_, view := m.paneRects()
	name := rolloutSteps[p.Sel].Name
	if row := rowText(c, view[paneSteps].Min.Y+p.Sel-p.Top); !strings.Contains(row, name) {
		t.Errorf("the selected step %q isn't on its row: %q", name, row)
	}
	// the selected row is the one marked: its focus mark stands on it
	marks := m.focusMarks(c)
	if len(marks) == 0 || marks[0].Y != view[paneSteps].Min.Y+p.Sel-p.Top {
		t.Errorf("the focus mark is on rows %v, want row %d", marks, view[paneSteps].Min.Y+p.Sel-p.Top)
	}
}

// Shift+wheel and the horizontal wheel scroll the pane under the pointer sideways, the
// rows that continue past its edges marked with the theme's cut marks, and the log's
// title says the column.
func TestShiftWheelScrollsSideways(t *testing.T) {
	m, clock := rolloutModel(t, "rollout", 80, 24)
	x, y := at(m, paneLog)
	top := m.panes[paneLog].Top
	notch(m, clock, x, y, tea.MouseWheelDown, tea.ModShift)
	if l := m.panes[paneLog]; l.Left != ui.WheelCols || l.Top != top {
		t.Errorf("shift+wheel down over the log: left %d top %d, want left %d top %d", l.Left, l.Top, ui.WheelCols, top)
	}
	notch(m, clock, x, y, tea.MouseWheelRight, 0)
	if l := m.panes[paneLog]; l.Left != 2*ui.WheelCols {
		t.Errorf("the wheel right: left %d, want %d", l.Left, 2*ui.WheelCols)
	}
	if m.pane != paneSteps {
		t.Error("the sideways wheel moved focus")
	}
	c := m.render(80, 24)
	_, view := m.paneRects()
	cut := 0
	for y := view[paneLog].Min.Y; y < view[paneLog].Max.Y; y++ {
		if cellText(c, view[paneLog].Min.X, y) == m.t.G.CutLeft {
			cut++
		}
	}
	if cut != view[paneLog].Dy() {
		t.Errorf("scrolled sideways, %d of the log's %d rows carry %s at its left edge", cut, view[paneLog].Dy(), m.t.G.CutLeft)
	}
	if head := rowText(c, view[paneLog].Min.Y-1-m.t.Gap()); !strings.Contains(head, fmt.Sprintf("col %d", 2*ui.WheelCols+1)) {
		t.Errorf("the log's title doesn't say the column: %q", head)
	}
}

// ←→ scroll sideways only while the focused pane's content is wider than it, and do
// nothing else: on the steps (which fit) they are not offered and change nothing; on
// the log (wider than the pane) they are, and scroll it; on a terminal wide enough for
// the log they are gone again. tab moves focus between the panes.
func TestArrowsScrollSidewaysOnlyWhenWide(t *testing.T) {
	sideways := func(m *app) bool {
		for _, a := range m.actions() {
			if a.b.Help().Desc == "scroll sideways" {
				return true
			}
		}
		return false
	}
	m, _ := rolloutModel(t, "rollout", 80, 24)
	if sideways(m) {
		t.Error("the steps fit their pane, and ←→ are offered")
	}
	panes, pane := m.panes, m.pane
	keyMsg(m, "right")
	keyMsg(m, "left")
	if m.panes != panes || m.pane != pane || m.screen != scrRollout {
		t.Errorf("←→ on the steps changed something: panes %+v, focus %d", m.panes, m.pane)
	}
	keyMsg(m, "tab")
	if m.pane != paneLog || !sideways(m) {
		t.Fatalf("tab: focus %d, ←→ offered %v; want the log focused and ←→ offered", m.pane, sideways(m))
	}
	keyMsg(m, "right")
	if m.panes[paneLog].Left != ui.KeyCols {
		t.Errorf("→ on the log: left %d, want %d", m.panes[paneLog].Left, ui.KeyCols)
	}
	if foot := footerText(m.render(80, 24), m.footY); !strings.Contains(foot, m.t.G.LeftRight) {
		t.Errorf("the log is wide and focused, and the footer doesn't offer ←→: %q", foot)
	}
	keyMsg(m, "shift+tab")
	if m.pane != paneSteps {
		t.Errorf("shift+tab: focus %d, want the steps", m.pane)
	}
	w, _ := rolloutModel(t, "rollout-scrolled", 400, 60)
	if w.panes[paneLog].Wide() || sideways(w) || w.panes[paneLog].Left != 0 {
		t.Errorf("at 400 columns the log fits, yet wide=%v ←→ offered=%v left=%d", w.panes[paneLog].Wide(), sideways(w), w.panes[paneLog].Left)
	}
}

// A click focuses the pane under it, and on the steps selects the row clicked; a right
// click, a click outside the panes and a click under a dialog do nothing. Neither pane's
// offsets move with focus: each keeps its own.
func TestClickFocusesAndSelects(t *testing.T) {
	m, clock := rolloutModel(t, "rollout", 80, 24)
	x, y := at(m, paneLog)
	notch(m, clock, x, y, tea.MouseWheelDown, tea.ModShift) // the log scrolled sideways, unfocused
	log := m.panes[paneLog]
	m.Update(tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft})
	if m.pane != paneLog || m.panes[paneLog] != log {
		t.Errorf("a click on the log: focus %d, log %+v; want the log focused, its offsets %+v kept", m.pane, m.panes[paneLog], log)
	}
	_, view := m.paneRects()
	row := view[paneSteps].Min.Y + 2
	m.Update(tea.MouseClickMsg{X: view[paneSteps].Min.X + 4, Y: row, Button: tea.MouseLeft})
	want := m.panes[paneSteps].Top + 2
	if m.pane != paneSteps || m.panes[paneSteps].Sel != want || m.panes[paneLog] != log {
		t.Errorf("a click on step row %d: focus %d sel %d log %+v; want the steps focused on %d, the log kept", row, m.pane, m.panes[paneSteps].Sel, m.panes[paneLog], want)
	}
	m.Update(tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseRight})
	m.Update(tea.MouseClickMsg{X: 0, Y: 0, Button: tea.MouseLeft})
	if m.pane != paneSteps {
		t.Errorf("a right click or a click on the header moved focus to %d", m.pane)
	}
	keyMsg(m, "?")
	if m.ov != ovHelp {
		t.Fatal("? opened no help")
	}
	panes := m.panes
	m.Update(tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft})
	notch(m, clock, x, y, tea.MouseWheelDown, 0)
	if m.pane != paneSteps || m.panes != panes {
		t.Error("under the help dialog the page took a click or the wheel")
	}
}

// A resize clamps each pane's offsets to the new size and never resets them: the log
// read to its end and far right, made larger, ends on its last row and column, not on
// its first; made small again, it stays where the clamp left it.
func TestResizeClampsPaneOffsets(t *testing.T) {
	m, _ := rolloutModel(t, "rollout-scrolled", 80, 24)
	keyMsg(m, "pgdown")
	keyMsg(m, "pgdown")
	for range 40 {
		keyMsg(m, "right")
	}
	l := m.panes[paneLog]
	if l.Top != l.MaxTop() || l.Left != l.MaxLeft() || l.Top == 0 || l.Left == 0 {
		t.Fatalf("the log isn't at its end: %+v", l)
	}
	steps := m.panes[paneSteps]
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	l = m.panes[paneLog]
	_, view := m.paneRects()
	_, lines, _ := m.rollout()
	if l.Rows() != view[paneLog].Dy() {
		t.Fatalf("after a resize to 120x40 the log is fitted to %d rows, its view has %d: the resize didn't refit it", l.Rows(), view[paneLog].Dy())
	}
	if want := len(lines) - view[paneLog].Dy(); l.Top != want || l.Left != l.MaxLeft() || l.Top == 0 || l.Left == 0 {
		t.Errorf("after a resize to 120x40 the log is at top %d left %d, want its clamps %d %d", l.Top, l.Left, want, l.MaxLeft())
	}
	if m.panes[paneSteps].Sel != steps.Sel || m.pane != paneLog {
		t.Errorf("the resize moved the selection (%d, was %d) or focus (%d)", m.panes[paneSteps].Sel, steps.Sel, m.pane)
	}
	clamped := m.panes
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	if m.panes[paneLog].Top < clamped[paneLog].Top || m.panes[paneLog].Left < clamped[paneLog].Left {
		t.Errorf("back at 80x24 the log reset: %+v, was %+v", m.panes[paneLog], clamped[paneLog])
	}
}

// One wheel notch is one notch however many events the terminal sends for it: three
// within a millisecond scroll as far as one.
func TestWheelBurstIsOneNotch(t *testing.T) {
	m, clock := rolloutModel(t, "rollout", 80, 24)
	x, y := at(m, paneLog)
	top := m.panes[paneLog].Top
	*clock = clock.Add(time.Second)
	for range 3 {
		*clock = clock.Add(300 * time.Microsecond)
		m.Update(tea.MouseWheelMsg{X: x, Y: y, Button: tea.MouseWheelUp})
	}
	if got := top - m.panes[paneLog].Top; got != ui.WheelRows {
		t.Errorf("a burst of 3 events scrolled %d rows, want one notch (%d)", got, ui.WheelRows)
	}
}

// Help names the mouse once: the footer shows keys only, and the full help ends with a
// Mouse section (click, wheel, shift+wheel) on a state that has the mouse on, and on no
// other. The mouse is on (View.MouseMode) exactly there.
func TestHelpNamesTheMouse(t *testing.T) {
	gestures := []string{"click", "wheel", "shift+wheel"}
	for _, s := range states {
		m := fixture(t, s)
		on := m.View().MouseMode == tea.MouseModeCellMotion
		if on != m.mouseOn() {
			t.Errorf("%s: View.MouseMode is on=%v, mouseOn says %v", s, on, m.mouseOn())
		}
		if foot := footerText(m.render(80, 24), m.footY); strings.Contains(foot, "wheel") || strings.Contains(foot, "click") {
			t.Errorf("%s: the footer names the mouse: %q", s, foot)
		}
		if m.ov != ovNone || s == "toosmall" {
			continue
		}
		m.openOverlay("help") // as its help key would (the splash has none)
		body := ansi.Strip(strings.Join(m.helpBody(), "\n"))
		for _, g := range gestures {
			if has := strings.Contains(body, g+" "); has != on {
				t.Errorf("%s: mouse on=%v, and help names %q: %v\n%s", s, on, g, has, body)
			}
		}
		if on && !strings.Contains(body, m.t.Case("Mouse")) {
			t.Errorf("%s: help names the gestures under no Mouse title:\n%s", s, body)
		}
	}
	if !fixture(t, "rollout").mouseOn() {
		t.Error("the rollout, whose panes scroll, has the mouse off")
	}
}

// The panes are fitted, and the mouse hit, from the layout draw draws: rolloutBody is
// the body draw lays out, in every theme at every size, and each pane's view sits in
// the body.
func TestRolloutLayoutIsDrawn(t *testing.T) {
	for _, f := range testThemes(t) {
		tf, err := theme.Load(f)
		if err != nil {
			t.Fatal(err)
		}
		m := newApp(tf, true, false)
		for _, sz := range [][2]int{{minW, minH}, {80, 24}, {120, 40}, {200, 60}} {
			m.setState("rollout")
			m.render(sz[0], sz[1])
			if got := m.rolloutBody(); got != m.body {
				t.Errorf("%s %dx%d: rolloutBody %v, draw's body %v", filepath.Base(f), sz[0], sz[1], got, m.body)
			}
			outer, view := m.paneRects()
			for i := range outer {
				if !view[i].In(m.body) || !view[i].In(outer[i]) || view[i].Dy() < 3 {
					t.Errorf("%s %dx%d: pane %d's view %v is outside its pane %v or the body %v, or under 3 rows", filepath.Base(f), sz[0], sz[1], i, view[i], outer[i], m.body)
				}
			}
			if ui.Hit(outer[paneLog].Min.X, outer[paneLog].Min.Y, outer[:]...) != paneLog || outer[paneSteps].Overlaps(outer[paneLog]) {
				t.Errorf("%s %dx%d: the panes overlap: %v %v", filepath.Base(f), sz[0], sz[1], outer[0], outer[1])
			}
		}
	}
}
