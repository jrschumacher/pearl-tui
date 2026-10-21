package ui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"

	"tuikit/theme/bubbletea"
)

// wheel is the tea.Mouse of a wheel event.
func wheel(b tea.MouseButton, mod tea.KeyMod) tea.Mouse { return tea.Mouse{Button: b, Mod: mod} }

// The wheel moves the view and never the selection: a list scrolled past its selected
// row keeps that row selected, out of sight, so an action never lands on a row the
// user only scrolled past. The next ↑↓ brings the view back to the selection as it
// moves it, wherever the view was.
func TestPaneWheelMovesTheViewNotTheSelection(t *testing.T) {
	p := Pane{Sel: 2}
	p.Fit(40, 0, 10, 30)
	for range 3 {
		if !p.Wheel(wheel(tea.MouseWheelDown, 0)) {
			t.Fatal("the wheel didn't scroll a pane of 40 rows in 10")
		}
	}
	if p.Top != 3*WheelRows || p.Sel != 2 {
		t.Fatalf("after three notches down: top %d sel %d, want top %d sel 2", p.Top, p.Sel, 3*WheelRows)
	}
	if p.Visible(p.Sel) {
		t.Fatal("the selection is still on screen: the test scrolled nothing past it")
	}
	p.Move(1) // ↓
	if p.Sel != 3 || !p.Visible(3) {
		t.Errorf("↓ after the wheel: sel %d top %d, want the selection on row 3 and in view", p.Sel, p.Top)
	}
	// and from above: the view far below the selection, ↑ brings it back
	p.Sel, p.Top = 30, 0
	p.Move(-1)
	if p.Sel != 29 || !p.Visible(29) {
		t.Errorf("↑ with the selection below the view: sel %d top %d, want 29 in view", p.Sel, p.Top)
	}
	// a pane without a selection (a log): ↑↓ scroll it by a row
	l := Pane{Sel: -1}
	l.Fit(40, 0, 10, 30)
	l.Move(1)
	if l.Top != 1 || l.Sel != -1 {
		t.Errorf("↓ in a log: top %d sel %d, want top 1 and no selection", l.Top, l.Sel)
	}
}

// Shift+wheel and the horizontal wheel scroll sideways, and only a pane wider than its
// view moves: a pane that fits takes no sideways scroll at all.
func TestPaneScrollsSideways(t *testing.T) {
	p := Pane{Sel: -1}
	p.Fit(40, 120, 10, 30)
	if !p.Wide() {
		t.Fatal("120 cells in 30 is not Wide")
	}
	p.Wheel(wheel(tea.MouseWheelDown, tea.ModShift))
	if p.Left != WheelCols || p.Top != 0 {
		t.Errorf("shift+wheel down: left %d top %d, want left %d top 0", p.Left, p.Top, WheelCols)
	}
	p.Wheel(wheel(tea.MouseWheelRight, 0))
	if p.Left != 2*WheelCols {
		t.Errorf("wheel right: left %d, want %d", p.Left, 2*WheelCols)
	}
	p.Wheel(wheel(tea.MouseWheelLeft, 0))
	p.Wheel(wheel(tea.MouseWheelUp, tea.ModShift))
	if p.Left != 0 || p.Top != 0 {
		t.Errorf("wheel left, then shift+wheel up: left %d top %d, want 0 0", p.Left, p.Top)
	}
	for range 100 {
		p.Scroll(0, KeyCols)
	}
	if p.Left != p.MaxLeft() || p.MaxLeft() != 120-p.TextWidth() {
		t.Errorf("scrolled right to the end: left %d, want %d (the widest row's end on the last column)", p.Left, 120-p.TextWidth())
	}
	n := Pane{Sel: -1}
	n.Fit(40, 20, 10, 30)
	if n.Wide() || n.Wheel(wheel(tea.MouseWheelRight, 0)) || n.Scroll(0, KeyCols) || n.Left != 0 {
		t.Errorf("a pane whose rows fit scrolled sideways: left %d", n.Left)
	}
}

// Each pane keeps its offsets across resizes: Fit clamps them to the new size and
// never resets them, so a pane read halfway down is still halfway down after the
// terminal is resized, and back at the same rows when it returns.
func TestPaneFitClampsNeverResets(t *testing.T) {
	p := Pane{Sel: 5}
	p.Fit(50, 200, 10, 40)
	p.Scroll(25, 60)
	top, left := p.Top, p.Left
	p.Fit(50, 200, 12, 50) // a little larger: room for both offsets as they were
	if p.Top != top || p.Left != left || p.Sel != 5 {
		t.Errorf("a resize that leaves room moved the pane: top %d left %d sel %d, want %d %d 5", p.Top, p.Left, p.Sel, top, left)
	}
	p.Fit(50, 200, 40, 180) // much larger: clamped to the new ends, not to 0
	if p.Top != p.MaxTop() || p.Left != p.MaxLeft() || p.Top == 0 || p.Left == 0 {
		t.Errorf("a resize past the offsets: top %d left %d, want the clamps %d %d (never 0)", p.Top, p.Left, p.MaxTop(), p.MaxLeft())
	}
	p.Fit(50, 200, 10, 40) // back: the clamped offsets stay
	if p.Top == 0 || p.Left == 0 {
		t.Errorf("shrinking back reset the pane: top %d left %d", p.Top, p.Left)
	}
}

// A pane scrolled sideways marks its cut edges: a row that continues past the left edge
// starts with the theme's CutLeft, one that continues past the right ends with
// CutRight (before the scrollbar while the pane is tall), and an empty row has neither.
// Every row is exactly the pane's width, styles kept through the cut, in every theme,
// with the ASCII twins (< >) under ASCII.
func TestPaneViewMarksCutEdges(t *testing.T) {
	for _, f := range testThemes(t) {
		tf, err := theme.Load(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, ascii := range []bool{false, true} {
			th := theme.Resolve(tf, true, ascii)
			long := th.S.Muted.Render("14:09:36") + "  " + th.S.Soft.Render(strings.Repeat("upstream payments timed out ", 5))
			lines := []string{long, "short", "", long}
			for i := range 20 { // tall: a scrollbar
				lines = append(lines, th.S.Soft.Render("row "+string(rune('a'+i))))
			}
			p := Pane{Sel: -1}
			p.Fit(len(lines), lipgloss.Width(long), 6, 30)
			p.Scroll(0, 12)
			rows := p.View(th, lines)
			if len(rows) != 6 {
				t.Fatalf("%s ascii=%v: %d rows, want 6", tf.Name, ascii, len(rows))
			}
			plain := make([]string, len(rows))
			for i, r := range rows {
				if w := lipgloss.Width(r); w != 30 {
					t.Errorf("%s ascii=%v: row %d is %d cells, want 30: %q", tf.Name, ascii, i, w, ansi.Strip(r))
				}
				plain[i] = []string{ansi.Strip(r)}[0]
			}
			body := func(r string) string { return strings.TrimRight(r, " "+th.G.Thumb+th.G.Track) }
			l, rt := th.G.CutLeft, th.G.CutRight
			if ascii && (l != "<" || rt != ">") {
				t.Errorf("%s: ASCII cut marks %q %q, want < >", tf.Name, l, rt)
			}
			if r := plain[0]; !strings.HasPrefix(r, l) || !strings.HasSuffix(body(r), rt) {
				t.Errorf("%s ascii=%v: a row cut at both edges reads %q, want %s…%s", tf.Name, ascii, r, l, rt)
			}
			if r := plain[1]; !strings.HasPrefix(r, l) || strings.Contains(r, rt) {
				t.Errorf("%s ascii=%v: a row wholly left of the view reads %q, want only %s", tf.Name, ascii, r, l)
			}
			if r := plain[2]; strings.Contains(r, l) || strings.Contains(r, rt) {
				t.Errorf("%s ascii=%v: an empty row is marked: %q", tf.Name, ascii, r)
			}
			// unscrolled, no left mark on any row
			p.Scroll(0, -100)
			if r := ansi.Strip(p.View(th, lines)[0]); strings.HasPrefix(r, l) || !strings.HasSuffix(body(r), rt) {
				t.Errorf("%s ascii=%v: unscrolled, the long row reads %q, want no %s and a %s", tf.Name, ascii, r, l, rt)
			}
		}
	}
	// a pane that fits draws its lines as they are: no mark, no bar, no reserved cell
	th := theme.Resolve(theme.Default(), true, false)
	p := Pane{Sel: -1}
	p.Fit(2, 5, 4, 10)
	if got := ansi.Strip(strings.Join(p.View(th, []string{"hello", "world"}), "\n")); got != "hello     \nworld     \n          \n          " {
		t.Errorf("a pane that fits drew %q", got)
	}
}

// Hit maps a cell to the region under it: the pane a click or the wheel lands on.
func TestHit(t *testing.T) {
	regions := []uv.Rectangle{uv.Rect(2, 4, 30, 18), uv.Rect(35, 4, 43, 18)}
	for _, c := range []struct{ x, y, want int }{
		{2, 4, 0}, {31, 21, 0}, {32, 10, -1}, {35, 10, 1}, {77, 21, 1}, {78, 10, -1}, {10, 3, -1}, {10, 22, -1},
	} {
		if got := Hit(c.x, c.y, regions...); got != c.want {
			t.Errorf("Hit(%d, %d) = %d, want %d", c.x, c.y, got, c.want)
		}
	}
}

// A burst of wheel events is one notch: a terminal that sends 3 per notch, within a
// millisecond, scrolls as far as one that sends 1. The next notch, a change of
// direction and a sideways turn each pass at once.
func TestWheelGateCoalescesBursts(t *testing.T) {
	var g WheelGate
	t0 := time.Date(2026, 10, 21, 14, 0, 0, 0, time.UTC)
	down := wheel(tea.MouseWheelDown, 0)
	passed := 0
	for i := range 3 {
		if g.Pass(down, t0.Add(time.Duration(i)*400*time.Microsecond)) {
			passed++
		}
	}
	if passed != 1 {
		t.Errorf("a burst of 3 events passed %d notches, want 1", passed)
	}
	if !g.Pass(down, t0.Add(120*time.Millisecond)) {
		t.Error("the next notch, 120 ms later, was dropped")
	}
	if !g.Pass(wheel(tea.MouseWheelUp, 0), t0.Add(121*time.Millisecond)) {
		t.Error("a change of direction inside the burst window was dropped")
	}
	if !g.Pass(wheel(tea.MouseWheelUp, tea.ModShift), t0.Add(122*time.Millisecond)) {
		t.Error("shift+wheel right after the wheel was dropped")
	}
	p := Pane{Sel: -1}
	p.Fit(100, 0, 10, 30)
	var g2 WheelGate
	for i := range 3 {
		if g2.Pass(down, t0.Add(time.Duration(i)*time.Millisecond)) {
			p.Wheel(down)
		}
	}
	if p.Top != WheelRows {
		t.Errorf("one notch sent as 3 events scrolled %d rows, want %d", p.Top, WheelRows)
	}
}
