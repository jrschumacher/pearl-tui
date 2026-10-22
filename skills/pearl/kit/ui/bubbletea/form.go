package ui

import (
	"os"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	hspinner "charm.land/huh/v2/spinner"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"tuikit/theme/bubbletea"
)

// Huh forms, themed from the brand. Keep Huh for input mechanics (typing,
// cursors, validation, accessible prompts); take every colour, glyph and
// button shape from the theme. Left alone, Huh draws its own Charm palette
// (#F780E2, #7571F9), a centred confirm, a `>` cursor and a `>` prompt that
// read as two selections, and a pink spinner. Everything here exists to stop
// that. The rules (each one a Huh v2.0.3 quirk, see the comments):
//
//	ui.NewForm(t, w, h, groups...)    never huh.NewForm: theme, size, keys, help, accessibility in the right order
//	ui.NewInput(t)                    never huh.NewInput: a blurred input keeps its value, drops its prompt glyph
//	ui.Confirm(t, …, def, &v)         never huh.NewConfirm: action first, left-aligned, the default explicit
//	ui.HuhSpinner / ui.BubblesSpinner never spinner.New(): brand frames and colour, a subject in the title
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
	lead := "" // a chip pads its name by one cell each side; plain rows match it
	if chip {
		lead = " "
	}
	f.SelectedOption = option("", selected(t), t.S.Muted)
	f.UnselectedOption = option(lead, t.S.Soft, t.S.Subtle)
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
	b.SelectedOption = option(lead, t.S.Base, t.S.Subtle)
	b.UnselectedOption = option(lead, t.S.Muted, t.S.Subtle)
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

// descSep separates an option's name from its description (see Options).
// An en space: one cell, never typed, and harmless if a theme ignores it.
const descSep = " "

// option draws an option key: the name in name (padded by pad on both sides
// when a chip marks the highlighted one, so names line up), then (for keys built
// by Options) the description in desc. Huh renders the selector and the key
// separately and has no per-option description, so the theme does it.
func option(pad string, name, desc lipgloss.Style) lipgloss.Style {
	return lipgloss.NewStyle().Transform(func(k string) string {
		n, d, ok := strings.Cut(k, descSep)
		out := pad + name.Render(n) + pad
		if ok {
			out += "  " + desc.Render(d)
		}
		return out
	})
}

// Choice is one option of a select, with a description beside it.
type Choice[T comparable] struct {
	Name, Desc string
	Value      T
}

// Options turns choices into Huh options whose descriptions line up in a
// column after the names, drawn in the description colour by HuhTheme. Huh's
// options are only a key and a value; this is the key. Keep descriptions
// short: Huh wraps a key that is wider than the field instead of cutting it.
func Options[T comparable](cs ...Choice[T]) []huh.Option[T] {
	w := 0
	for _, c := range cs {
		w = max(w, lipgloss.Width(c.Name))
	}
	out := make([]huh.Option[T], len(cs))
	for i, c := range cs {
		k := c.Name
		if c.Desc != "" {
			k = Fit(c.Name, w) + descSep + c.Desc
		}
		out[i] = huh.NewOption(k, c.Value)
	}
	return out
}

// chosenButton is the chosen button of a confirm that doesn't have focus:
// still visibly the choice, never as loud as a focused one.
func chosenButton(t *theme.Theme, label string) string {
	st := t.S.Button.Foreground(t.P.Fg).Bold(true)
	if t.ASCII || t.X.Button == "bracket" {
		return st.UnsetPadding().UnsetBackground().Render(" [ " + label + " ]")
	}
	return st.Render(label)
}

func arrowBack(t *theme.Theme) string {
	return t.G.Back
}

// FormKeys is Huh's keymap with the keys that open unthemed widgets turned
// off. "/" on a select opens a filter (a bare textinput with Bubbles' default
// colours); a wizard's lists are short enough not to need it. ctrl+c is the
// app's, never the form's: the app's key handler sees it first.
func FormKeys() *huh.KeyMap {
	k := huh.NewDefaultKeyMap()
	k.Select.Filter.SetEnabled(false)
	k.MultiSelect.Filter.SetEnabled(false)
	k.Quit = key.NewBinding(key.WithDisabled())
	return k
}

// NewForm builds a Huh form the kit's way, in the order Huh needs:
// theme first, then width (fields wrap by it), then height (Huh sized each
// group's viewport with its default theme in NewGroup and never re-measures,
// so a themed field can be clipped unless the height is set after).
//
// Huh's own help line is off: the app's footer, read from the same action
// table that dispatches keys, is the help (law 6). With ACCESSIBLE=1 the form
// runs as plain prompts (form.Run()) and takes Huh's colourless base theme:
// Huh writes those prompts straight to the output, so a brand colour there is
// escape codes in a screen reader's ear.
func NewForm(t *theme.Theme, w, h int, groups ...*huh.Group) *huh.Form {
	th := HuhTheme(t)
	if Accessible() {
		th = huh.ThemeFunc(huh.ThemeBase)
	}
	f := huh.NewForm(groups...).
		WithTheme(th).
		WithKeyMap(FormKeys()).
		WithShowHelp(false).
		WithShowErrors(true).
		WithAccessible(Accessible())
	Resize(f, w, h)
	return f
}

// Resize lays a form out at w×h: at most h rows, and no more than its
// content needs. Two Huh quirks meet here. A group draws from a viewport that
// is only refilled on Update, so after WithWidth/WithHeight the form shows
// the old layout until a message arrives (a no-op one refills it). And the
// viewport takes the whole height, with validation errors appended below it:
// sized to the screen, an error lands rows under its field, or off the
// bottom. Sized to the content, the error sits right under the field.
func Resize(f *huh.Form, w, h int) {
	f.WithWidth(max(20, w))
	f.WithHeight(max(1, h))
	f.Update(relayout{})
	rows := strings.Split(ansi.Strip(f.View()), "\n")
	errs := len(f.Errors()) // each error is a footer row, after a blank one
	if errs > 0 {
		rows = rows[:max(1, len(rows)-errs-1)]
	}
	n := len(rows)
	for n > 1 && strings.TrimSpace(rows[n-1]) == "" {
		n--
	}
	// Huh subtracts the footer from the height but not the blank row above
	// it, so with errors the form draws errs+1 rows below its content
	if n+errs < h {
		f.WithHeight(n + errs)
		f.Update(relayout{})
	}
}

type relayout struct{}

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

// ---------------------------------------------------------------- input

// Input is huh.Input with one fix: a blurred Huh input draws its prompt and
// text with Bubbles' default styles (Huh only themes the focused state of the
// textinput), so an unfocused "> prod" sat beside a select's "> " cursor and
// two things looked selected. Blurred, this Input draws its own row: the
// value in fg (or the placeholder in fgSubtle) behind blank prompt cells, so
// focusing it never shifts the text. Focused, it is Huh's input.
//
// Build it like huh.NewInput; the setters return *Input so the chain keeps
// the wrapper. Anything else is on .Input.
type Input struct {
	*huh.Input
	t                        *theme.Theme
	st                       huh.FieldStyles // the blurred styles, built once
	title, desc, placeholder string
	prompt                   string
	focused                  bool
	width                    int
}

// NewInput is a themed huh.Input with the theme's prompt glyph.
func NewInput(t *theme.Theme) *Input {
	i := &Input{Input: huh.NewInput(), t: t, st: HuhStyles(t).Blurred, prompt: t.G.Prompt + " "}
	i.Input.Prompt(i.prompt)
	return i
}

func (i *Input) Title(s string) *Input       { i.title = s; i.Input.Title(s); return i }
func (i *Input) Description(s string) *Input { i.desc = s; i.Input.Description(s); return i }
func (i *Input) Placeholder(s string) *Input {
	i.placeholder = s
	i.Input.Placeholder(s)
	return i
}
func (i *Input) Value(v *string) *Input                     { i.Input.Value(v); return i }
func (i *Input) Key(k string) *Input                        { i.Input.Key(k); return i }
func (i *Input) Validate(fn func(string) error) *Input      { i.Input.Validate(fn); return i }
func (i *Input) CharLimit(n int) *Input                     { i.Input.CharLimit(n); return i }
func (i *Input) Suggestions(s []string) *Input              { i.Input.Suggestions(s); return i }
func (i *Input) Focused() bool                              { return i.focused }
func (i *Input) Focus() tea.Cmd                             { i.focused = true; return i.Input.Focus() }
func (i *Input) Blur() tea.Cmd                              { i.focused = false; return i.Input.Blur() }
func (i *Input) WithWidth(w int) huh.Field                  { i.width = w; i.Input.WithWidth(w); return i }
func (i *Input) WithTheme(th huh.Theme) huh.Field           { i.Input.WithTheme(th); return i }
func (i *Input) WithKeyMap(k *huh.KeyMap) huh.Field         { i.Input.WithKeyMap(k); return i }
func (i *Input) WithHeight(h int) huh.Field                 { i.Input.WithHeight(h); return i }
func (i *Input) WithPosition(p huh.FieldPosition) huh.Field { i.Input.WithPosition(p); return i }

// Update keeps the wrapper in the group: Huh stores whatever Update returns.
func (i *Input) Update(msg tea.Msg) (huh.Model, tea.Cmd) {
	_, cmd := i.Input.Update(msg)
	return i, cmd
}

// View is Huh's when focused, and the kit's quiet row when not.
func (i *Input) View() string {
	if i.focused {
		return i.Input.View()
	}
	t := i.t
	st := i.st
	inner := max(1, i.width-st.Base.GetHorizontalFrameSize())
	var rows []string
	if i.title != "" {
		ti := st.Title.Render(Trunc(t, i.title, inner))
		if i.Error() != nil {
			ti += st.ErrorIndicator.String()
		}
		rows = append(rows, ti)
	}
	if i.desc != "" {
		rows = append(rows, st.Description.Render(Trunc(t, i.desc, inner)))
	}
	pad := strings.Repeat(" ", lipgloss.Width(i.prompt))
	v, _ := i.GetValue().(string)
	val := t.S.Base.Render(Trunc(t, v, inner-lipgloss.Width(pad)))
	if v == "" {
		val = t.S.Subtle.Render(Trunc(t, i.placeholder, inner-lipgloss.Width(pad)))
	}
	rows = append(rows, pad+val)
	return st.Base.Width(max(1, i.width)).Render(strings.Join(rows, "\n"))
}

// ---------------------------------------------------------------- confirm

// Confirm is huh.Confirm with the kit's convention: the action first and the
// negative last ("Yes  No", "Connect  Back"), left-aligned to the field's
// text column (Huh centres the buttons under the title by default, so they
// float mid-column), and the default stated: def is the button focused when
// the field opens. Make def the safe choice for a destructive or irreversible
// action, the expected forward choice otherwise.
func Confirm(t *theme.Theme, title, action, negative string, def bool, value *bool) *huh.Confirm {
	*value = def
	return huh.NewConfirm().
		Title(title).
		Affirmative(action).
		Negative(negative).
		WithButtonAlignment(lipgloss.Left).
		Value(value)
}

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
