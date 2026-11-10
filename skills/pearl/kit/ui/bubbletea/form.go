package ui

import (
	"os"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	hspinner "charm.land/huh/v2/spinner"
	"charm.land/lipgloss/v2"

	"github.com/jrschumacher/pearl-tui/skills/pearl/kit/theme/bubbletea"
)

// Huh forms, themed from the brand. Huh keeps its own mechanics and layout
// (typing, cursors, validation, focus order, sizing, accessible prompts); the
// kit only gives it the theme's colours, glyphs and button shape through
// Huh's public theme API. Left on its default theme, Huh draws Charm's palette
// (#F780E2, #7571F9) and a pink spinner.
//
//	ui.NewForm(t, w, h, groups...)    huh.NewForm with the theme, the app's footer as help, ACCESSIBLE honoured
//	ui.HuhSpinner / ui.BubblesSpinner the brand's frames and colour, a subject in the title
//
// Fields are Huh's own: huh.NewInput, huh.NewSelect, huh.NewConfirm. Where Huh's
// look falls short of the theme (a blurred input draws in Bubbles' default
// styles: Huh's theme reaches only the focused one), the fix belongs in Huh,
// not in a wrapper here.
//
// It lives in ui, not theme: theme is the dependency-free contract any renderer
// reads; Huh is one consumer of it, like every other component here.

// Accessible reports whether the user asked for the accessible mode: plain
// line-by-line prompts, no alt screen, no spinners. ACCESSIBLE is the
// variable Huh documents; TERM=dumb is how Huh itself decides.
func Accessible() bool {
	return os.Getenv("ACCESSIBLE") != "" || os.Getenv("TERM") == "dumb"
}

// HuhTheme maps the resolved theme onto Huh. It ignores Huh's isDark flag:
// Huh assumes a light terminal until a tea.BackgroundColorMsg reaches the
// form, but the kit has already resolved dark or light, so the flag can only
// be wrong. The styles are built once (Huh asks for them on every View of
// every field). A theme is applied to a field only once (Huh's WithTheme
// keeps the first), so when the theme changes, rebuild the form.
func HuhTheme(t *theme.Theme) huh.Theme {
	s := HuhStyles(t)
	return huh.ThemeFunc(func(bool) *huh.Styles { return s })
}

// HuhStyles is every Huh style from the theme. Start from ThemeBase so no
// field Huh adds later inherits the Charm palette.
func HuhStyles(t *theme.Theme) *huh.Styles {
	p, g := t.P, t.G
	s := huh.ThemeBase(t.Dark)
	s.Form.Base = lipgloss.NewStyle()
	s.Group.Base = lipgloss.NewStyle()
	s.Group.Title = t.S.Title
	s.Group.Description = t.S.Muted
	s.FieldSeparator = lipgloss.NewStyle().SetString("\n\n") // one blank row between fields, even when compact

	// the gutter: a focus bar plus one cell, exactly as ui.Gutter draws it, so
	// a Huh field and a kit block line up and focus never moves the text
	bar := lipgloss.Border{Left: g.FocusBar}
	blur := lipgloss.Border{Left: " "}
	blurFg := p.Bg
	if g.BlurBar != "" {
		blur, blurFg = lipgloss.Border{Left: g.BlurBar}, t.S.BlurBar.GetForeground()
	}

	f := &s.Focused
	f.Base = lipgloss.NewStyle().Border(bar, false, false, false, true).BorderForeground(t.S.FocusBar.GetForeground()).PaddingLeft(1)
	f.Card = f.Base
	f.Title = t.S.Bold
	f.NoteTitle = t.S.Title.MarginBottom(1)
	f.Description = t.S.Muted
	f.ErrorIndicator = t.S.Err.SetString(" " + g.Fail)
	// Huh prints errors under the group, at its left edge: indent them to the
	// fields' text column and mark them with the fail glyph, not colour alone
	f.ErrorMessage = t.S.Err.PaddingLeft(2).Transform(func(e string) string { return g.Fail + " " + e })
	f.Directory = t.S.Mark
	f.File = t.S.Base

	// selects: the theme's cursor, then the option. Unselected options are
	// fgSoft: readable, never the "disabled" grey of Huh's default.
	cw := max(lipgloss.Width(g.Cursor), lipgloss.Width(g.RadioOn))
	f.SelectSelector = t.S.Focus.Bold(true).SetString(Fit(g.Cursor, cw) + " ")
	f.MultiSelectSelector = f.SelectSelector
	chip := t.X.Selection == "fill" || t.X.Selection == "inverse"
	lead := 0 // a chip pads its name by one cell each side; plain rows match it
	if chip {
		lead = 1
	}
	f.SelectedOption = selected(t)
	f.UnselectedOption = t.S.Soft.Padding(0, lead)
	f.SelectedPrefix = t.S.Focus.SetString(g.CheckOn + " ")
	f.UnselectedPrefix = t.S.Subtle.SetString(g.CheckOff + " ")
	f.NextIndicator = t.S.Focus.MarginLeft(1).SetString(g.Arrow)
	f.PrevIndicator = t.S.Focus.MarginRight(1).SetString(arrowBack(t))

	f.TextInput = huh.TextInputStyles{
		Cursor:      t.S.Prompt,
		CursorText:  t.S.Base,
		Placeholder: t.S.Subtle,
		Prompt:      t.S.Prompt.Bold(true),
		Text:        t.S.Base,
	}

	// buttons are ui.Button, so a Huh confirm and a kit dialog are one shape
	gap := 2
	if t.X.Button == "pill" || t.X.Button == "inverse" {
		gap = 1
	}
	f.FocusedButton = lipgloss.NewStyle().MarginRight(gap).Transform(func(l string) string { return Button(t, l, true, false, -1) })
	f.BlurredButton = lipgloss.NewStyle().MarginRight(gap).Transform(func(l string) string { return Button(t, l, false, false, -1) })
	f.Next = f.FocusedButton

	// blurred: the same cells, quieter. The select's chosen option keeps a
	// mark, but the radio glyph, never the cursor: one cursor on screen.
	b := *f
	b.Base = lipgloss.NewStyle().Border(blur, false, false, false, true).BorderForeground(blurFg).PaddingLeft(1)
	b.Card = b.Base
	b.Title = t.S.Muted
	b.NoteTitle = t.S.Muted.MarginBottom(1)
	b.Description = t.S.Subtle
	b.SelectSelector = t.S.Muted.SetString(Fit(g.RadioOn, cw) + " ")
	b.MultiSelectSelector = lipgloss.NewStyle().SetString(strings.Repeat(" ", cw+1))
	b.SelectedOption = t.S.Base.Padding(0, lead)
	b.UnselectedOption = t.S.Muted.Padding(0, lead)
	b.NextIndicator = lipgloss.NewStyle()
	b.PrevIndicator = lipgloss.NewStyle()
	b.TextInput.Prompt = t.S.Subtle
	b.FocusedButton = lipgloss.NewStyle().MarginRight(gap).Transform(func(l string) string { return chosenButton(t, l) })
	b.Next = b.FocusedButton
	s.Blurred = b

	s.Help.Ellipsis = t.S.Subtle
	s.Help.ShortKey, s.Help.FullKey = t.S.Key, t.S.Key
	s.Help.ShortDesc, s.Help.FullDesc = t.S.KeyDesc, t.S.KeyDesc
	s.Help.ShortSeparator, s.Help.FullSeparator = t.S.Subtle, t.S.Subtle
	return s
}

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

// chosenButton is the chosen button of a confirm that doesn't have focus:
// still visibly the choice, never as loud as a focused one.
func chosenButton(t *theme.Theme, label string) string {
	st := t.S.Button.Foreground(t.P.Fg).Bold(true)
	if Colourless(t) && t.X.Button != "underline" {
		return st.UnsetPadding().UnsetBackground().Render(" [ " + label + " ]")
	}
	return st.Render(label)
}

func arrowBack(t *theme.Theme) string {
	return t.G.Back
}

// NewForm is huh.NewForm with the theme. Huh's own help line is off: the
// app's footer, read from the same action table that dispatches keys, is the
// help (law 6). With ACCESSIBLE=1 the form runs as plain prompts (form.Run())
// and takes Huh's colourless base theme: Huh writes those prompts straight to
// the output, so a brand colour there is escape codes in a screen reader's ear.
// After a resize, call the form's own WithWidth and WithHeight.
func NewForm(t *theme.Theme, w, h int, groups ...*huh.Group) *huh.Form {
	th := HuhTheme(t)
	if Accessible() {
		th = huh.ThemeFunc(huh.ThemeBase)
	}
	return huh.NewForm(groups...).
		WithTheme(th).
		WithShowHelp(false).
		WithShowErrors(true).
		WithAccessible(Accessible()).
		WithWidth(w).
		WithHeight(h)
}

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
			bs[i] = chosenButton(t, l)
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

// HuhSpinner is Huh's standalone spinner (spinner.New().Action(...).Run())
// in the brand: the theme's frames, the working ink (S.Working) instead of
// Huh's #F780E2, the title in fgSoft with the gap Huh forgets (it draws the frame
// and the title with no space between), and a subject instead of Huh's
// default "Loading...". It honours ACCESSIBLE=1 (one plain line, no frames).
func HuhSpinner(t *theme.Theme, title string) *hspinner.Spinner {
	st := &hspinner.Styles{Spinner: t.S.Working, Title: t.S.Soft.PaddingLeft(1)}
	return hspinner.New().
		Type(hspinner.Type(spinner.Spinner{Frames: t.G.Spinner, FPS: frameRate})).
		WithTheme(hspinner.ThemeFunc(func(bool) *hspinner.Styles { return st })).
		Title(title).
		WithAccessible(Accessible())
}
