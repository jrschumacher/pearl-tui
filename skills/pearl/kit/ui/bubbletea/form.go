package ui

import (
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/jrschumacher/pearl-tui/skills/pearl/kit/theme/bubbletea"
)

// Form plumbing that needs no Huh: Settle, the button row and the Bubbles spinner. The
// Huh helpers (huhform.NewForm, huhform.Spinner, huhform.Theme) live in
// ui/bubbletea/huhform, so an app with no Huh form never links Huh. A new form uses the
// kit's own fields (fields.go: Input, Select, Confirm).

// Settle runs cmd and feeds every message it produces back through update,
// until nothing is left: a model driven outside a running program (a
// harness, a test) reaches the state a user would see. Huh moves between
// fields by returning a Cmd (NextField), not by moving, so Update alone
// leaves focus where it was.
//
// Every command runs at once, as a program runs them, and each has SettleWait from
// when it started to answer; one that answers later is a timer (a cursor blink, a
// spinner or clock tick) and is dropped: a frame must not wait on a timer. So a round's
// timers cost one window together, never their own time, and a command that does real
// work keeps its window under load (it had 5 ms, one after another, and a busy machine
// dropped Huh's NextField). tea.Sequence's message is private, so a sequence is
// dropped too; Huh only uses one in Form.Init, whose work is done by then.
//
//	ui.Settle(func(msg tea.Msg) tea.Cmd { _, c := form.Update(msg); return c }, cmd)
func Settle(update func(tea.Msg) tea.Cmd, cmd tea.Cmd) {
	type answer struct {
		id  int
		msg tea.Msg
		at  time.Time // when it answered, not when Settle got to it
	}
	done := make(chan struct{})
	defer close(done) // a timer still running answers nobody, and its goroutine ends
	answers := make(chan answer)
	due := map[int]time.Time{} // each running command's deadline
	next := 0
	run := func(c tea.Cmd) {
		if c == nil {
			return
		}
		id := next
		next++
		due[id] = time.Now().Add(SettleWait)
		go func() {
			m := c()
			select {
			case answers <- answer{id, m, time.Now()}:
			case <-done:
			}
		}()
	}
	run(cmd)
	for delivered := 0; len(due) > 0 && delivered < 256; {
		var last time.Time // every running command is past its deadline then
		for _, d := range due {
			if d.After(last) {
				last = d
			}
		}
		wait := time.NewTimer(time.Until(last))
		select {
		case a := <-answers:
			wait.Stop()
			late := a.at.After(due[a.id])
			delete(due, a.id)
			if late {
				continue // a timer that fired while the others ran
			}
			switch msg := a.msg.(type) {
			case nil:
			case tea.BatchMsg:
				for _, c := range msg {
					run(c)
				}
			default:
				delivered++
				run(update(msg))
			}
		case <-wait.C:
			return
		}
	}
}

// SettleWait is how long Settle gives each command to answer: past a command that does
// work on a loaded machine (tens of milliseconds), short of the kit's frame clock
// (FrameInterval, 50 ms) and every spinner and cursor blink, which Settle drops.
const SettleWait = 40 * time.Millisecond

// selected is the highlighted option per expressions.selection. Huh draws the
// selector and the option separately, so a filled selection is a chip.
func selected(t *theme.Theme) lipgloss.Style {
	switch t.X.Selection {
	case "fill", "inverse":
		return t.S.SelRow.Padding(0, 1)
	case "caret":
		return t.S.SelRow
	}
	return t.S.Bold
}

// ChosenButton is the chosen button of a confirm that doesn't have focus:
// still visibly the choice, never as loud as a focused one.
func ChosenButton(t *theme.Theme, label string) string {
	st := t.S.Button.Foreground(t.P.Fg).Bold(true)
	if Colourless(t) && t.X.Button != "underline" {
		return st.UnsetPadding().UnsetBackground().Render(" [ " + label + " ]")
	}
	return st.Render(label)
}

// ---------------------------------------------------------------- actions

// Actions is a row of buttons the app owns (a review step, a retry prompt):
// the order fixed, the action first and the negative last, the default
// explicit. It draws left-aligned in the form's text column, behind the same
// 2-cell gutter a Huh field has, never centred.
type Actions struct {
	Labels []string // the action first, the negative last, each naming what it does: "Continue", "Back to edit", "Discard setup" (never a bare "Cancel")
	Sel    int      // the focused button; NewActions starts it at the default
	Danger int      // index of a destructive button (coloured even unfocused); -1 for none
}

// NewActions makes a button row with def focused.
func NewActions(def int, labels ...string) Actions {
	return Actions{Labels: labels, Sel: def, Danger: -1}
}

// Move steps the focus by d, without wrapping (wrapping onto a negative
// action is how people cancel by accident).
func (a *Actions) Move(d int) { a.Sel = max(0, min(len(a.Labels)-1, a.Sel+d)) }

// Selected is the focused label.
func (a Actions) Selected() string { return a.Labels[a.Sel] }

// View draws the row: gutter, then the buttons left to right.
func (a Actions) View(t *theme.Theme, focused bool) string {
	bs := make([]string, len(a.Labels))
	for i, l := range a.Labels {
		bs[i] = Button(t, l, focused && i == a.Sel, i == a.Danger, -1)
		if !focused && i == a.Sel {
			bs[i] = ChosenButton(t, l)
		}
	}
	return Gutter(t, focused, t.S.FocusBar) + Buttons(t, 0, lipgloss.Left, bs...)
}

// ---------------------------------------------------------------- spinners

// frameRate is how fast the brand's spinner frames turn: one frame per two
// clock ticks, the same pace ui.Spinner gets from the app's one clock.
const frameRate = 2 * FrameInterval

// BubblesSpinner is a Bubbles spinner in the brand's frames and colour. Its
// default is Charm's Line at the terminal's default colour. Prefer
// ui.Spinner(t, frame) on the app's one clock; use this only where a
// component must own its tick.
func BubblesSpinner(t *theme.Theme) spinner.Model {
	return spinner.New(
		spinner.WithSpinner(spinner.Spinner{Frames: t.G.Spinner, FPS: frameRate}),
		spinner.WithStyle(t.S.Working),
	)
}
