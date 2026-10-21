package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"

	"tuikit/theme/bubbletea"
)

// How far one gesture moves a pane: a wheel notch scrolls WheelRows rows or, sideways,
// WheelCols columns; one ← or → scrolls KeyCols columns.
const (
	WheelRows = 3
	WheelCols = 8
	KeyCols   = 4
)

// WheelBurst is how close two wheel events of one direction must be to count as one
// notch (WheelGate): some terminals send 3 events per notch, within a millisecond.
const WheelBurst = 15 * time.Millisecond

// Pane is the state of one region that scrolls both ways: its own offsets, kept across
// focus changes and resizes, and its selection, if it has one. The wheel moves the
// view and never the selection (Wheel); ↑↓ move the selection and bring the view back
// to it first (Move); ←→ scroll sideways only while the content is wider than the pane
// (Wide). Fit tells it the content's size and the view's on every layout and clamps
// the offsets to them, so a resize clamps them and never resets them.
//
// One Pane per region, in the root model; View draws a pane of lines with its cut
// edges marked, and a list pane passes Top and Sel to List instead.
type Pane struct {
	Top  int // the first content row shown
	Left int // the first content column shown
	Sel  int // the selected row, or -1 in a pane without a selection (a log)

	rows, width, h, w int // the content (rows, widest row in cells) and the view, as Fit set them
}

// Fit records the content's size (rows, and its widest row in cells) and the view's
// (h rows of w cells) and clamps the offsets to them. Call it on every layout: from the
// WindowSizeMsg handler and before drawing, so keys and the wheel scroll against the
// pane as it is drawn.
func (p *Pane) Fit(rows, width, h, w int) {
	p.rows, p.width, p.h, p.w = max(0, rows), max(0, width), max(0, h), max(0, w)
	p.Scroll(0, 0)
	if p.Sel >= p.rows && p.rows > 0 {
		p.Sel = p.rows - 1 // the content shrank under it: its last row
	}
}

// Tall reports content with more rows than the view: a scrollbar column is drawn.
func (p Pane) Tall() bool { return p.rows > p.h }

// Wide reports content wider than the view: ←→ and the sideways wheel scroll it, and
// its cut edges are marked.
func (p Pane) Wide() bool {
	w := p.w
	if p.Tall() {
		w -= 2
	}
	return p.width > w
}

// TextWidth is the cells of content a row shows: the view, less the scrollbar column
// and its gap while Tall, less a cut-edge cell on each side while Wide.
func (p Pane) TextWidth() int {
	switch wide, tall := p.Wide(), p.Tall(); {
	case wide && tall:
		return max(0, p.w-3) // a mark, the text, a mark, then the bar: the right mark stands against it
	case wide, tall:
		return max(0, p.w-2)
	}
	return p.w
}

// Rows is the view's height, as Fit last set it.
func (p Pane) Rows() int { return p.h }

// MaxTop and MaxLeft are the largest offsets: the last row on the view's last row, the
// widest row's end on its last column.
func (p Pane) MaxTop() int  { return max(0, p.rows-p.h) }
func (p Pane) MaxLeft() int { return max(0, p.width-p.TextWidth()) }

// Scroll moves the view dy rows and dx columns, clamped; never the selection. It
// reports whether the view moved.
func (p *Pane) Scroll(dy, dx int) bool {
	top, left := p.Top, p.Left
	p.Top = max(0, min(p.Top+dy, p.MaxTop()))
	p.Left = max(0, min(p.Left+dx, p.MaxLeft()))
	return p.Top != top || p.Left != left
}

// Wheel scrolls the pane for a wheel event: up and down by WheelRows, sideways by
// WheelCols for shift+wheel and a trackpad's horizontal wheel. It moves the view only:
// the selection stays on its row, which may scroll out of sight. Route each event to
// the pane under the pointer (Hit), not the focused one, and through a WheelGate.
func (p *Pane) Wheel(m tea.Mouse) bool {
	shift := m.Mod&tea.ModShift != 0
	switch {
	case m.Button == tea.MouseWheelLeft, m.Button == tea.MouseWheelUp && shift:
		return p.Scroll(0, -WheelCols)
	case m.Button == tea.MouseWheelRight, m.Button == tea.MouseWheelDown && shift:
		return p.Scroll(0, WheelCols)
	case m.Button == tea.MouseWheelUp:
		return p.Scroll(-WheelRows, 0)
	case m.Button == tea.MouseWheelDown:
		return p.Scroll(WheelRows, 0)
	}
	return false
}

// Move is ↑ (d < 0) or ↓: the selection moves d rows and the view follows it (Follow),
// so a selection the wheel scrolled out of sight is brought back into view by the same
// press that moves it. A pane without a selection scrolls d rows instead.
func (p *Pane) Move(d int) {
	if p.Sel < 0 {
		p.Scroll(d, 0)
		return
	}
	p.Sel = max(0, min(p.Sel+d, p.rows-1))
	p.Top = Follow(p.Sel, p.Top, p.h, p.rows)
}

// Page is pgup (d = -1) or pgdown: Move by a view less one row.
func (p *Pane) Page(d int) { p.Move(d * max(1, p.h-1)) }

// ScrollTo puts row at the view's top (clamped), keeping the column.
func (p *Pane) ScrollTo(row int) { p.Scroll(row-p.Top, 0) }

// Visible reports whether row is on screen.
func (p Pane) Visible(row int) bool { return row >= p.Top && row < p.Top+p.h }

// View draws lines into the pane: exactly Rows() rows of the view's width, from Top,
// each cut at Left (styles kept), with the theme's cut marks (CutLeft, CutRight in
// subtle) on the rows that continue past an edge while the pane is Wide, and the
// scrollbar while it is Tall. Fit it with these lines first.
func (p Pane) View(t *theme.Theme, lines []string) []string {
	tall, wide, tw := p.Tall(), p.Wide(), p.TextWidth()
	sb := Scrollbar(t, p.h, p.rows, p.Top)
	out := make([]string, 0, p.h)
	for r := range p.h {
		row := ""
		if i := p.Top + r; i < len(lines) {
			l := ExpandTabs(lines[i])
			if wide {
				lw := lipgloss.Width(l)
				left, right := " ", " "
				if p.Left > 0 && lw > 0 {
					left = t.S.Subtle.Render(t.G.CutLeft)
				}
				if lw > p.Left+tw {
					right = t.S.Subtle.Render(t.G.CutRight)
				}
				row = left + Fit(ansi.Cut(l, p.Left, p.Left+tw), tw) + right
			} else {
				row = Fit(l, tw)
			}
		} else {
			row = Fit("", tw)
			if wide {
				row = "  " + row
			}
		}
		if tall {
			if !wide {
				row += " "
			}
			row += sb[r]
		}
		out = append(out, Fit(row, p.w))
	}
	return out
}

// Hit is the index of the first region holding the cell x, y, or -1: the pane a click
// or the wheel lands on, from the rectangles the screen laid its panes out in.
func Hit(x, y int, regions ...uv.Rectangle) int {
	for i, r := range regions {
		if uv.Pos(x, y).In(r) {
			return i
		}
	}
	return -1
}

// WheelGate coalesces a burst of wheel events into one notch: an event of the same
// button and modifiers within WheelBurst of the last one let through is dropped, so a
// terminal that sends 3 events per notch scrolls as far as one that sends 1, and a
// trackpad's stream is held to one step per WheelBurst. A change of direction passes
// at once.
type WheelGate struct {
	last time.Time
	btn  tea.MouseButton
	mod  tea.KeyMod
}

// Pass reports whether the wheel event m, arriving at now, is a notch of its own.
func (g *WheelGate) Pass(m tea.Mouse, now time.Time) bool {
	if m.Button == g.btn && m.Mod == g.mod && !g.last.IsZero() && now.Sub(g.last) < WheelBurst {
		return false
	}
	g.last, g.btn, g.mod = now, m.Button, m.Mod
	return true
}
