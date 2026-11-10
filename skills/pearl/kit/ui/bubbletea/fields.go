package ui

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/term"

	"github.com/jrschumacher/pearl-tui/skills/pearl/kit/theme/bubbletea"
)

// Form fields, the kit's own: a text input, a select and a confirm. Each is a plain
// struct the app owns: it routes keys to the focused one from its action table, moves
// focus between them itself, and draws each into its rectangle with View(t, w,
// focused), which returns the field's rows (Height(t, w) of them, without reading
// rendered text). focused=false draws the same cells, quieter, so a field under a
// dialog or beside the focused one blurs, and focus never moves the text (law 5).
//
//	ui.NewInput(title, desc, placeholder, value)  Bubbles' textinput, themed focused and blurred; its error under it; Secret masks it
//	ui.NewSelect(title, desc, value, opts...)     one of a few options, each with a dim description column; Filterable narrows them as the user types
//	ui.NewConfirm(title, desc, yes, no, value)    two ui.Actions buttons, the action first
//
// Every row has the 2-cell gutter ui.Gutter draws, so a field, an ui.Actions row and a
// kit block line up. With ACCESSIBLE=1 (ui.Accessible) an app asks the same fields as
// plain prompts through a ui.Prompter instead of drawing them.

// Accessible reports whether the user asked for the accessible mode: plain
// line-by-line prompts, no alt screen, no spinners. ACCESSIBLE is the variable Huh
// documents; TERM=dumb is how Huh itself decides.
func Accessible() bool {
	return os.Getenv("ACCESSIBLE") != "" || os.Getenv("TERM") == "dumb"
}

// fieldHead is a field's title and description, wrapped at w: the title bold and the
// description muted when focused, a step quieter each when not.
func fieldHead(t *theme.Theme, title, desc string, w int, focused bool) []string {
	tst, dst := t.S.Muted, t.S.Subtle
	if focused {
		tst, dst = t.S.Bold, t.S.Muted
	}
	return append(wrapped(tst, title, w), wrapped(dst, desc, w)...)
}

// wrapped is s wrapped by words at w, each line in st; none for "".
func wrapped(st lipgloss.Style, s string, w int) []string {
	if s == "" {
		return nil
	}
	var out []string
	for _, l := range strings.Split(lipgloss.Wrap(s, max(1, w), ""), "\n") {
		out = append(out, st.Render(strings.TrimSpace(l)))
	}
	return out
}

// gutterW is the cells a field's gutter takes, focused or not.
func gutterW(t *theme.Theme) int { return lipgloss.Width(Gutter(t, true, t.S.FocusBar)) }

// ---------------------------------------------------------------- input

// Input is a one-line text field: Bubbles' textinput for the editing (cursor keys,
// word moves, paste, scrolling a long value), the theme for every cell of it.
//
// Secret marks a password, token or key: View draws one mask per character, focused
// and blurred, in the value's own ink (Bubbles' EchoCharacter, '*' unless the app sets
// Model.EchoCharacter: no theme glyph is a mask in every brand, a bullet is ▸ or –
// in some), and Ask reads it without echo. The kit never writes a secret's value: not
// in View, not in the plain prompt's "blank keeps" hint. Validate's error is drawn as
// written, so a secret's must not quote the value. Setting Model.EchoMode to anything
// but EchoNormal marks the field secret too.
type Input struct {
	Title, Desc string
	Validate    func(string) error // run by Submit; nil accepts anything
	Err         error              // drawn under the field until the value changes
	Model       textinput.Model    // set CharLimit on it; Prompt and styles are the kit's
	Secret      bool               // mask the value; Ask reads it without echo
}

// secret reports whether f's value is masked: Secret, or an EchoMode set by hand.
func (f *Input) secret() bool { return f.Secret || f.Model.EchoMode != textinput.EchoNormal }

// NewInput is an input holding value. Its cursor doesn't blink: a blink is a timer of
// its own, and the app has one clock.
func NewInput(title, desc, placeholder, value string) Input {
	m := textinput.New()
	m.Prompt, m.Placeholder = "", placeholder
	m.SetStyles(textinput.Styles{Cursor: textinput.CursorStyle{Blink: false}})
	m.SetValue(value)
	m.Focus()
	return Input{Title: title, Desc: desc, Model: m}
}

// Value is the text typed.
func (f Input) Value() string { return f.Model.Value() }

// Update hands the field a key (or a paste) the app routed to it. A change clears the
// error: it described a value that is gone.
func (f *Input) Update(msg tea.Msg) tea.Cmd {
	before := f.Model.Value()
	f.Model.Focus() // the app sends keys only to the focused field
	if f.Secret {
		f.Model.EchoMode = textinput.EchoPassword // word moves jump to the ends, as on any masked field
	}
	var cmd tea.Cmd
	f.Model, cmd = f.Model.Update(msg)
	if f.Model.Value() != before {
		f.Err = nil
	}
	return cmd
}

// Submit validates the value, keeps the error to draw, and reports whether it passed.
func (f *Input) Submit() bool {
	f.Err = nil
	if f.Validate != nil {
		f.Err = f.Validate(f.Value())
	}
	return f.Err == nil
}

// View draws the field in w cells: title, description, the prompt and value, and the
// error directly under it. Blurred, the prompt is blank cells and there's no cursor,
// so the value stays where it was. A pointer: the input keeps the width it was drawn
// at, which it scrolls a long value by.
func (f *Input) View(t *theme.Theme, w int, focused bool) []string {
	g := Gutter(t, focused, t.S.FocusBar)
	iw := max(1, w-gutterW(t))
	rows := fieldHead(t, f.Title, f.Desc, iw, focused)
	pr := t.G.Prompt + " "
	prompt := strings.Repeat(" ", lipgloss.Width(pr))
	if focused {
		prompt = t.S.Prompt.Bold(true).Render(pr)
	}
	f.Model.Prompt = ""
	if f.Secret {
		f.Model.EchoMode = textinput.EchoPassword
	}
	f.Model.SetStyles(inputStyles(t))
	f.Model.SetWidth(max(1, iw-lipgloss.Width(pr)-1)) // a cell for the cursor past the end
	if focused {
		f.Model.Focus()
	} else {
		f.Model.Blur()
	}
	rows = append(rows, prompt+f.Model.View())
	rows = append(rows, f.errRows(t, iw)...)
	return prefix(rows, g, w)
}

func (f *Input) errRows(t *theme.Theme, w int) []string {
	if f.Err == nil {
		return nil
	}
	return wrapped(t.S.Err, t.G.Fail+" "+f.Err.Error(), w)
}

// Height is the rows View draws at w.
func (f *Input) Height(t *theme.Theme, w int) int {
	iw := max(1, w-gutterW(t))
	return len(fieldHead(t, f.Title, f.Desc, iw, true)) + 1 + len(f.errRows(t, iw))
}

// inputStyles are the textinput's styles from the theme, focused and blurred both:
// left at Bubbles' defaults, a blurred input draws in ANSI greys.
func inputStyles(t *theme.Theme) textinput.Styles {
	return textinput.Styles{
		Focused: textinput.StyleState{Text: t.S.Base, Placeholder: t.S.Subtle, Suggestion: t.S.Subtle, Prompt: t.S.Prompt},
		Blurred: textinput.StyleState{Text: t.S.Soft, Placeholder: t.S.Subtle, Suggestion: t.S.Subtle, Prompt: t.S.Subtle},
		Cursor:  textinput.CursorStyle{Color: t.S.Prompt.GetForeground(), Shape: tea.CursorBlock},
	}
}

// ---------------------------------------------------------------- select

// Option is one choice of a Select: its name, an optional one-line description drawn
// dim in a column beside the names, and the value it stands for.
type Option[T comparable] struct {
	Name, Desc string
	Value      T
}

// Select is one of a few options. Rows caps the options shown at once (0 shows them
// all); past it the list scrolls with the highlight and draws a scrollbar.
//
// Filterable (off by default) lets typing narrow the options: the app hands the
// focused select its printable keys and backspace with Type, the options shown are
// those whose names hold the typed letters in order (ui.Fuzzy, any case), and the
// highlight stays on a shown option, which is the value. While a filter is in force
// its line (ui.Filter: the query, a caret while focused, "2 of 5") stands above the
// options with a row of air under it, as over a list, so the field grows two rows at
// the first letter typed; then it keeps its height as the options narrow, and a filter
// that leaves nothing draws the kit's empty state in their place, never a blank. Say
// that typing filters in the field's description ("…; type to filter"): the footer
// can't name every letter.
//
// The keys, one rule: on a filterable select letters type (j and k too) and only the
// arrow keys move the highlight; an unfilterable one keeps ↑↓ k j. Esc is two steps:
// while Filtered, esc clears the filter (ClearFilter) and the field keeps the focus;
// only with no filter does esc mean the app's back or cancel. With nothing matching
// (Matches is 0) no option is highlighted, so the app offers no key that submits
// until the filter changes; Value keeps the last highlighted option's meanwhile.
type Select[T comparable] struct {
	Title, Desc string
	Options     []Option[T]
	Sel         int // the highlighted option, which is the value
	Rows        int
	Top         int  // the first option shown when Rows cuts the list (of those the filter leaves)
	Filterable  bool // typing narrows the options: Type, Filtered, ClearFilter
	query       string
}

// NewSelect is a select with value highlighted (the first option when none holds it).
func NewSelect[T comparable](title, desc string, value T, opts ...Option[T]) Select[T] {
	s := Select[T]{Title: title, Desc: desc, Options: opts}
	for i, o := range opts {
		if o.Value == value {
			s.Sel = i
		}
	}
	return s
}

// visible are the indexes of the options the filter leaves, in order: all of them
// with no filter.
func (s Select[T]) visible() []int {
	out := make([]int, 0, len(s.Options))
	q := strings.ToLower(s.query)
	for i, o := range s.Options {
		if _, ok := Fuzzy(strings.ToLower(o.Name), q); ok {
			out = append(out, i)
		}
	}
	return out
}

// Move steps the highlight by d through the options shown, without wrapping.
func (s *Select[T]) Move(d int) {
	v := s.visible()
	if len(v) == 0 {
		return
	}
	i := max(0, min(len(v)-1, max(0, slices.Index(v, s.Sel))+d))
	s.Sel = v[i]
	if s.Rows > 0 {
		s.Top = Follow(i, s.Top, s.Rows, len(v))
	}
}

// Type hands a filterable select a key the app routed to it: printable text extends
// the filter, backspace takes its last character off. It reports whether the key was
// the filter's (false for any other key, a backspace with no filter, or a select that
// isn't Filterable), so the app can try its own bindings next.
func (s *Select[T]) Type(msg tea.KeyPressMsg) bool {
	if !s.Filterable {
		return false
	}
	switch {
	case msg.Code == tea.KeyBackspace && msg.Mod == 0:
		r := []rune(s.query)
		if len(r) == 0 {
			return false
		}
		s.query = string(r[:len(r)-1])
	case msg.Text != "" && msg.Mod&^tea.ModShift == 0:
		s.query += msg.Text
	default:
		return false
	}
	s.keepShown()
	return true
}

// Filtered reports whether a filter is in force: esc then clears it (ClearFilter)
// rather than leaving the field.
func (s Select[T]) Filtered() bool { return s.query != "" }

// Query is the filter as typed.
func (s Select[T]) Query() string { return s.query }

// Matches is how many options the filter leaves (all of them with none).
func (s Select[T]) Matches() int { return len(s.visible()) }

// ClearFilter drops the filter; the highlight stays on the option it was on.
func (s *Select[T]) ClearFilter() {
	s.query = ""
	s.keepShown()
}

// keepShown moves the highlight to the first option the filter leaves when the filter
// hides the highlighted one, and keeps the scroll on it.
func (s *Select[T]) keepShown() {
	v := s.visible()
	if len(v) == 0 {
		return
	}
	i := slices.Index(v, s.Sel)
	if i < 0 {
		i, s.Sel = 0, v[0]
	}
	s.Top = 0
	if s.Rows > 0 {
		s.Top = Follow(i, s.Top, s.Rows, len(v))
	}
}

// Selected is the highlighted option.
func (s Select[T]) Selected() Option[T] {
	if len(s.Options) == 0 {
		return Option[T]{}
	}
	return s.Options[s.Sel]
}

// Value is the highlighted option's value.
func (s Select[T]) Value() T { return s.Selected().Value }

// shownOf is the option rows drawn when n options are left.
func (s Select[T]) shownOf(n int) int {
	if s.Rows > 0 {
		return min(s.Rows, n)
	}
	return n
}

func (s Select[T]) shown() int { return s.shownOf(len(s.Options)) }

// noMatch is the empty state a filter that leaves nothing draws in w cells, or none.
func (s Select[T]) noMatch(t *theme.Theme, w int) []string {
	if s.query == "" || len(s.Options) == 0 || len(s.visible()) > 0 {
		return nil
	}
	return Empty(t, w, NoMark, "No option matches \""+s.query+"\"",
		"None of the "+strconv.Itoa(len(s.Options))+" names has those letters in order.", "")
}

// optionRows is the rows the options take: as many as with no filter, so the field
// keeps its height as the options narrow, or the empty state's when it needs more.
func (s Select[T]) optionRows(t *theme.Theme, bodyW int) int {
	return max(s.shown(), len(s.noMatch(t, bodyW)))
}

// bodyW is the cells the option rows take at inner width iw: two less for the
// scrollbar when Rows cuts the unfiltered list, so the columns stay put as it narrows.
func (s Select[T]) bodyW(iw int) int {
	if s.shown() < len(s.Options) {
		return iw - 2 // the scrollbar and a cell before it
	}
	return iw
}

// View draws the field in w cells. The highlighted option takes the theme's cursor and
// selection style when focused; blurred, it keeps Theme.BlurBar in the cursor's cells
// (as a blurred ui.List row does) and plain ink, so a screen has one cursor and a
// dialog over the field changes only the mark.
func (s Select[T]) View(t *theme.Theme, w int, focused bool) []string {
	g := t.G
	iw := max(1, w-gutterW(t))
	rows := fieldHead(t, s.Title, s.Desc, iw, focused)
	v := s.visible()
	if s.query != "" { // the filter's line and a row of air, as over a list
		rows = append(rows, Filter(t, s.query, focused, len(v), len(s.Options), iw), "")
	}
	n := s.shownOf(len(v))
	bodyW := s.bodyW(iw)
	cw := max(lipgloss.Width(g.Cursor), lipgloss.Width(t.BlurBar()))
	lead := 0 // a chip pads its name a cell each side; plain names keep the column
	if t.X.Selection == "fill" || t.X.Selection == "inverse" {
		lead = 1
	}
	nameW := 0
	for _, o := range s.Options {
		nameW = max(nameW, lipgloss.Width(o.Name))
	}
	nameW = min(nameW, max(1, bodyW-cw-1-2*lead))
	top := min(max(0, s.Top), max(0, len(v)-n))
	sb := Scrollbar(t, n, len(v), top)
	for r := range n {
		i := v[top+r]
		o := s.Options[i]
		mark, st := strings.Repeat(" ", cw), t.S.Soft.Padding(0, lead)
		switch {
		case i == s.Sel && focused:
			mark, st = t.S.Focus.Bold(true).Render(Fit(g.Cursor, cw)), selected(t)
		case i == s.Sel:
			mark, st = t.S.BlurBar.Render(Fit(t.BlurBar(), cw)), t.S.Base.Padding(0, lead)
		case !focused:
			st = t.S.Muted.Padding(0, lead)
		}
		name := Trunc(t, o.Name, nameW) // the chip fits the name; the column's padding is page
		row := mark + " " + st.Render(name) + strings.Repeat(" ", nameW-lipgloss.Width(name))
		if room := bodyW - lipgloss.Width(row) - 2; o.Desc != "" && room >= 8 {
			row += "  " + t.S.Subtle.Render(Trunc(t, o.Desc, room))
		}
		if bodyW < iw {
			row = Fit(row, bodyW) + " " + sb[r]
		}
		rows = append(rows, row)
	}
	empty := s.noMatch(t, bodyW)
	rows = append(rows, empty...)
	for range s.optionRows(t, bodyW) - n - len(empty) {
		rows = append(rows, "") // the rows the filter emptied: the field keeps its height
	}
	return prefix(rows, Gutter(t, focused, t.S.FocusBar), w)
}

// Height is the rows View draws at w.
func (s Select[T]) Height(t *theme.Theme, w int) int {
	iw := max(1, w-gutterW(t))
	h := len(fieldHead(t, s.Title, s.Desc, iw, true)) + s.optionRows(t, s.bodyW(iw))
	if s.query != "" {
		h += 2 // the filter's line and its row of air
	}
	return h
}

// ---------------------------------------------------------------- confirm

// Confirm is a yes/no question answered by two buttons: Acts.Labels[0] is the action
// and means yes, [1] the negative. Move between them with Acts.Move.
type Confirm struct {
	Title, Desc string
	Acts        Actions
}

// NewConfirm is a confirm with value chosen: yes names the action ("Turn on"), no the
// negative ("Not now"), never a bare Yes/No where a verb says more.
func NewConfirm(title, desc, yes, no string, value bool) Confirm {
	c := Confirm{Title: title, Desc: desc, Acts: NewActions(0, yes, no)}
	c.Set(value)
	return c
}

// Value reports whether the action is chosen.
func (c Confirm) Value() bool { return c.Acts.Sel == 0 }

// Set chooses the action (true) or the negative.
func (c *Confirm) Set(v bool) {
	c.Acts.Sel = 1
	if v {
		c.Acts.Sel = 0
	}
}

// View draws the field in w cells: title, description, a blank row, the buttons. The
// chosen button stays marked when blurred, quieter (Actions.View).
func (c Confirm) View(t *theme.Theme, w int, focused bool) []string {
	g := Gutter(t, focused, t.S.FocusBar)
	rows := prefix(append(fieldHead(t, c.Title, c.Desc, max(1, w-gutterW(t)), focused), ""), g, w)
	return append(rows, Fit(c.Acts.View(t, focused), w))
}

// Height is the rows View draws at w.
func (c Confirm) Height(t *theme.Theme, w int) int {
	return len(fieldHead(t, c.Title, c.Desc, max(1, w-gutterW(t)), true)) + 2
}

// ---------------------------------------------------------------- accessible

// Prompter asks fields as plain prompts, one question per line, for ACCESSIBLE=1 and
// TERM=dumb: no alt screen, no colour, no cursor movement, so a screen reader reads
// each line once. A blank answer keeps the field's value; a wrong one says why and asks
// again. One Prompter reads all of a flow's answers from in.
//
// A secret Input's answer is never echoed: when in is a terminal (an *os.File such as
// os.Stdin) the Prompter turns the terminal's echo off for that one line and restores
// it after, on an error too; from a pipe or a test it reads the line as given and
// never prints it back. A line typed ahead of the question was echoed as it was typed,
// before the Prompter could stop it, and is read from what is buffered. On a terminal
// the secret's line ends only at enter (ctrl+d is ignored there), and a process killed
// mid-answer (ctrl+c) ends before it can restore the echo: an app that asks a secret on
// a terminal and catches SIGINT restores it in its handler (term.Restore).
type Prompter struct {
	in  *bufio.Reader
	src io.Reader // in's source: a terminal's echo is turned off on it for a secret
	out io.Writer
}

// NewPrompter reads answers from in and writes questions to out.
func NewPrompter(in io.Reader, out io.Writer) *Prompter {
	return &Prompter{in: bufio.NewReader(in), src: in, out: out}
}

// Say writes one line of status between questions.
func (p *Prompter) Say(format string, a ...any) { fmt.Fprintf(p.out, format+"\n", a...) }

// ask writes the title, any lines under it, and the question, and reads one answer,
// trimmed; an input that ends before an answer is an error, never an endless re-ask.
func (p *Prompter) ask(title, desc string, lines []string, question string) (string, error) {
	return p.askLine(title, desc, lines, question, false)
}

// askLine is ask, reading a secret without echo when secret is set.
func (p *Prompter) askLine(title, desc string, lines []string, question string, secret bool) (string, error) {
	if desc != "" {
		title += " (" + desc + ")"
	}
	fmt.Fprintln(p.out, title)
	for _, l := range lines {
		fmt.Fprintln(p.out, l)
	}
	fmt.Fprintf(p.out, "%s: ", question)
	l, err := p.readLine(secret)
	if err != nil && (l == "" || !errors.Is(err, io.EOF)) {
		fmt.Fprintln(p.out)
		return "", fmt.Errorf("%s: no answer: %w", title, err)
	}
	return strings.TrimSpace(l), nil
}

// readLine reads one answer. A secret from a terminal is read with its echo off (the
// terminal's state restored however the read ends), and the newline the terminal no
// longer echoes is written to out; a line already buffered was echoed when typed and is
// read from the buffer, as is anything that is not a terminal.
func (p *Prompter) readLine(secret bool) (string, error) {
	if f, ok := p.src.(*os.File); ok && secret && p.in.Buffered() == 0 && term.IsTerminal(f.Fd()) {
		b, err := term.ReadPassword(f.Fd()) // restores the echo on return, an error included
		fmt.Fprintln(p.out)
		if err == nil {
			b = append(b, '\n')
		}
		return string(b), err
	}
	return p.in.ReadString('\n')
}

// Ask asks for f's value through p, until Validate passes. A Secret is read without
// echo and its current value is never written: the hint says only that one is kept.
func (f *Input) Ask(p *Prompter) error {
	q := "Type a value"
	switch v := f.Value(); {
	case v != "" && f.secret():
		q += " (blank keeps the current one)"
	case v != "":
		q += " (blank keeps " + v + ")"
	case f.Model.Placeholder != "":
		q += " (blank means " + f.Model.Placeholder + ")"
	}
	for {
		a, err := p.askLine(f.Title, f.Desc, nil, q, f.secret())
		if err != nil {
			return err
		}
		if a != "" {
			f.Model.SetValue(a)
		}
		if f.Submit() {
			return nil
		}
		p.Say("Not accepted: %s.", strings.TrimSuffix(f.Err.Error(), "."))
	}
}

// Choose asks which of opts, by number, and returns the index chosen; sel is the
// default. Select.Ask is it for a Select.
func (p *Prompter) Choose(title, desc string, names []string, sel int) (int, error) {
	lines := make([]string, len(names))
	for i, n := range names {
		lines[i] = fmt.Sprintf("%d. %s", i+1, n)
	}
	q := fmt.Sprintf("Enter 1 to %d (blank keeps %d)", len(names), sel+1)
	for {
		a, err := p.ask(title, desc, lines, q)
		if err != nil || a == "" {
			return sel, err
		}
		if n, err := strconv.Atoi(a); err == nil && n >= 1 && n <= len(names) {
			return n - 1, nil
		}
		p.Say("Not accepted: %q is not a number from 1 to %d.", a, len(names))
	}
}

// Ask asks for s's option through p.
func (s *Select[T]) Ask(p *Prompter) error {
	names := make([]string, len(s.Options))
	for i, o := range s.Options {
		names[i] = o.Name
		if o.Desc != "" {
			names[i] += ", " + o.Desc
		}
	}
	sel, err := p.Choose(s.Title, s.Desc, names, s.Sel)
	s.Sel = sel
	return err
}

// Ask asks c through p as y or n; the buttons' labels say what each answer does.
func (c *Confirm) Ask(p *Prompter) error {
	def := "n"
	if c.Value() {
		def = "y"
	}
	q := fmt.Sprintf("y for %s, n for %s (blank keeps %s)", c.Acts.Labels[0], c.Acts.Labels[1], def)
	for {
		a, err := p.ask(c.Title, c.Desc, nil, q)
		if err != nil {
			return err
		}
		switch strings.ToLower(a) {
		case "":
			return nil
		case "y", "yes":
			c.Set(true)
			return nil
		case "n", "no":
			c.Set(false)
			return nil
		}
		p.Say("Not accepted: answer y or n.")
	}
}
