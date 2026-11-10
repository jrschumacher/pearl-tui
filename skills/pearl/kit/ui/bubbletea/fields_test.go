package ui

import (
	"bytes"
	"errors"
	"image/color"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/jrschumacher/pearl-tui/skills/pearl/kit/theme/bubbletea"
)

func sampleFields() (Input, Select[string], Confirm) {
	in := NewInput("Namespace", "where your services run; blank means default", "default", "pay")
	in.Validate = func(s string) error {
		if strings.Contains(s, " ") {
			return errors.New("no spaces in a namespace, which the cluster would refuse at the first deploy")
		}
		return nil
	}
	sel := NewSelect("Cluster", "contexts in ~/.kube/config", "b",
		Option[string]{Name: "prod-us-east-1", Desc: "production", Value: "a"},
		Option[string]{Name: "staging", Desc: "current context", Value: "b"},
		Option[string]{Name: "dev", Value: "c"})
	return in, sel, NewConfirm("Tell you when a deploy fails?", "a desktop notification", "Turn on", "Not now", true)
}

// eachTheme runs fn for every kit theme, as Unicode and as ASCII.
func eachTheme(t *testing.T, fn func(name string, th *theme.Theme)) {
	for _, f := range append(testThemes(t), quirePath(t)) {
		tf, err := theme.Load(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, ascii := range []bool{false, true} {
			fn(filepath.Base(f)+map[bool]string{true: " ascii"}[ascii], theme.Resolve(tf, true, ascii))
		}
	}
}

// A field's Height is the rows its View draws, at every width, focused or not, with an
// error showing: the app lays fields out from Height, never from rendered text.
func TestFieldHeightIsItsView(t *testing.T) {
	eachTheme(t, func(name string, th *theme.Theme) {
		in, sel, cf := sampleFields()
		in.Model.SetValue("pay ments")
		in.Submit()
		for _, w := range []int{24, 40, 80} {
			for _, focused := range []bool{true, false} {
				for what, got := range map[string][2]int{
					"input":   {in.Height(th, w), len(in.View(th, w, focused))},
					"select":  {sel.Height(th, w), len(sel.View(th, w, focused))},
					"confirm": {cf.Height(th, w), len(cf.View(th, w, focused))},
				} {
					if got[0] != got[1] {
						t.Errorf("%s %s w=%d focused=%v: Height %d, View draws %d rows", name, what, w, focused, got[0], got[1])
					}
				}
				for _, rows := range [][]string{in.View(th, w, focused), sel.View(th, w, focused), cf.View(th, w, focused)} {
					for _, r := range rows {
						if ansi.StringWidth(r) != w {
							t.Errorf("%s w=%d focused=%v: a row is %d cells: %q", name, w, focused, ansi.StringWidth(r), ansi.Strip(r))
						}
					}
				}
			}
		}
	})
}

// Focus never moves the text (law 5): a field's value, option names and descriptions
// stand in the same cells focused or not; only the marks before them change.
func TestFieldFocusKeepsTheText(t *testing.T) {
	eachTheme(t, func(name string, th *theme.Theme) {
		in, sel, _ := sampleFields()
		col := func(rows []string, s string) []int {
			var out []int
			for _, r := range rows {
				out = append(out, cellIndex(r, s))
			}
			return out
		}
		for _, s := range []string{"pay", "Namespace"} {
			if a, b := col(in.View(th, 60, true), s), col(in.View(th, 60, false), s); !slices.Equal(a, b) {
				t.Errorf("%s input: %q moves on blur: %v → %v", name, s, a, b)
			}
		}
		for _, s := range []string{"prod-us-east-1", "staging", "current context", "production"} {
			if a, b := col(sel.View(th, 60, true), s), col(sel.View(th, 60, false), s); !slices.Equal(a, b) {
				t.Errorf("%s select: %q moves on blur: %v → %v", name, s, a, b)
			}
		}
	})
}

// cellIndex is the cell where s starts in the styled row r, or -1.
func cellIndex(r, s string) int {
	p := ansi.Strip(r)
	i := strings.Index(p, s)
	if i < 0 {
		return -1
	}
	return ansi.StringWidth(p[:i])
}

// The option descriptions are one column, dim, beside the widest name: Huh drew the
// name alone, and an app faked the column by padding names by hand.
func TestSelectDescriptionColumn(t *testing.T) {
	eachTheme(t, func(name string, th *theme.Theme) {
		_, sel, _ := sampleFields()
		rows := sel.View(th, 60, true)
		a, b := cellIndex(rows[2], "production"), cellIndex(rows[3], "current context")
		if a < 0 || a != b {
			t.Errorf("%s: descriptions at columns %d and %d, want one column:\n%s", name, a, b, ansi.Strip(strings.Join(rows, "\n")))
		}
		// too narrow for a description: the names stay, the column goes
		for _, r := range sel.View(th, 22, true) {
			if strings.Contains(r, "production") {
				t.Errorf("%s: a description squeezed into 22 cells: %q", name, ansi.Strip(r))
			}
		}
	})
}

// One cursor: the focused select draws the theme's cursor once, a blurred one none, and
// a blurred input no prompt glyph (Huh drew it in Bubbles' default grey).
func TestFieldsOneCursor(t *testing.T) {
	eachTheme(t, func(name string, th *theme.Theme) {
		in, sel, _ := sampleFields()
		g := th.G
		count := func(rows []string, mark string) int {
			n := 0
			for _, r := range rows {
				if strings.HasPrefix(strings.TrimLeft(ansi.Strip(ansi.Cut(r, gutterW(th), 60)), " "), mark+" ") {
					n++
				}
			}
			return n
		}
		if n := count(sel.View(th, 60, true), g.Cursor); n != 1 {
			t.Errorf("%s: focused select draws %d cursors", name, n)
		}
		if n := count(sel.View(th, 60, false), g.Cursor); n != 0 {
			t.Errorf("%s: blurred select draws %d cursors", name, n)
		}
		if n := count(in.View(th, 60, false), g.Prompt); n != 0 {
			t.Errorf("%s: blurred input draws its prompt", name)
		}
		if n := count(in.View(th, 60, true), g.Prompt); n != 1 {
			t.Errorf("%s: focused input draws %d prompts", name, n)
		}
	})
}

// Every colour of every field, focused and blurred, is a theme role: never an ANSI
// index from a Bubbles default style. Huh's theme reached only a focused input.
func TestFieldsAreThemed(t *testing.T) {
	eachTheme(t, func(name string, th *theme.Theme) {
		in, sel, cf := sampleFields()
		empty := NewInput("Empty", "", "default", "")
		for _, focused := range []bool{true, false} {
			rows := append(append(append(in.View(th, 60, focused), sel.View(th, 60, focused)...), cf.View(th, 60, focused)...), empty.View(th, 60, focused)...)
			c := NewCanvas(th, 60, len(rows))
			c.Draw(strings.Join(rows, "\n"), Rect(0, 0, 60, len(rows)))
			b := c.Bounds()
			for y := b.Min.Y; y < b.Max.Y; y++ {
				for x := b.Min.X; x < b.Max.X; x++ {
					cell := c.CellAt(x, y)
					if cell == nil {
						continue
					}
					for _, col := range []color.Color{cell.Style.Fg, cell.Style.Bg} {
						switch col.(type) {
						case ansi.BasicColor, ansi.IndexedColor:
							t.Fatalf("%s focused=%v: an ANSI default colour at %d,%d (%q)", name, focused, x, y, cell.Content)
						}
					}
				}
			}
		}
	})
}

// The error stands on the row under the input, marked with the fail glyph, and goes
// when the value changes: Huh printed it under the whole form, rows below the field.
func TestInputErrorUnderTheField(t *testing.T) {
	th := testTheme(t, false)
	in, _, _ := sampleFields()
	in.Model.CursorEnd()
	for _, r := range " x" {
		in.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	if in.Submit() {
		t.Fatal("a value with a space passed")
	}
	rows := in.View(th, 60, true)
	at := -1
	for i, r := range rows {
		if strings.Contains(ansi.Strip(r), "pay x") {
			at = i
		}
	}
	if at < 0 || at+1 >= len(rows) || !strings.Contains(ansi.Strip(rows[at+1]), th.G.Fail+" no spaces") {
		t.Fatalf("the error is not on the row under the value:\n%s", ansi.Strip(strings.Join(rows, "\n")))
	}
	in.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	if in.Err != nil {
		t.Fatal("editing the value kept the old error")
	}
}

// Select and Confirm keep their values as the app moves them.
func TestSelectAndConfirmValues(t *testing.T) {
	_, sel, cf := sampleFields()
	if sel.Value() != "b" {
		t.Fatalf("NewSelect highlighted %q, want b", sel.Value())
	}
	sel.Move(5)
	if sel.Value() != "c" {
		t.Fatalf("Move past the end gave %q, want the last, c", sel.Value())
	}
	sel.Rows = 2
	sel.Move(-2)
	if sel.Top != 0 || sel.Value() != "a" {
		t.Fatalf("scrolled select: top %d value %q", sel.Top, sel.Value())
	}
	if !cf.Value() || cf.Acts.Selected() != "Turn on" {
		t.Fatal("NewConfirm(true) did not choose the action")
	}
	cf.Acts.Move(1)
	if cf.Value() {
		t.Fatal("moving to the negative kept yes")
	}
}

// The plain prompts: a numbered choice, a blank keeping the value, a wrong answer
// re-asked with the reason, and an input that ends before an answer is an error.
func TestPrompter(t *testing.T) {
	in, sel, cf := sampleFields()
	var out bytes.Buffer
	p := NewPrompter(strings.NewReader("7\n1\nbad name\nprod\nmaybe\nn\n"), &out)
	for _, ask := range []func(*Prompter) error{sel.Ask, in.Ask, cf.Ask} {
		if err := ask(p); err != nil {
			t.Fatal(err)
		}
	}
	if sel.Value() != "a" || in.Value() != "prod" || cf.Value() {
		t.Fatalf("answers lost: %q %q %v\n%s", sel.Value(), in.Value(), cf.Value(), out.String())
	}
	txt := out.String()
	for _, want := range []string{"1. prod-us-east-1, production", "not a number from 1 to 3", "no spaces in a namespace", "answer y or n", "y for Turn on, n for Not now"} {
		if !strings.Contains(txt, want) {
			t.Errorf("transcript lacks %q:\n%s", want, txt)
		}
	}
	if strings.Contains(txt, "\x1b[") {
		t.Errorf("escape codes in the plain prompts:\n%q", txt)
	}
	kept := NewInput("Namespace", "", "default", "pay")
	if err := kept.Ask(NewPrompter(strings.NewReader("\n"), &out)); err != nil || kept.Value() != "pay" {
		t.Errorf("a blank answer did not keep the value: %q %v", kept.Value(), err)
	}
	if err := kept.Ask(NewPrompter(strings.NewReader(""), &out)); err == nil {
		t.Error("an input with no answer left asked forever or passed")
	}
}

func typeInto[T comparable](s *Select[T], text string) {
	for _, r := range text {
		s.Type(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
}

func clusters() Select[string] {
	s := NewSelect("Cluster", "contexts in ~/.kube/config", "staging-eu-west-1",
		Option[string]{Name: "prod-us-east-1", Desc: "production", Value: "prod-us-east-1"},
		Option[string]{Name: "prod-eu-west-1", Desc: "production", Value: "prod-eu-west-1"},
		Option[string]{Name: "staging-eu-west-1", Desc: "current context", Value: "staging-eu-west-1"},
		Option[string]{Name: "staging-us-east-1", Value: "staging-us-east-1"},
		Option[string]{Name: "dev-local", Desc: "kind on this machine", Value: "dev-local"})
	s.Filterable = true
	return s
}

// shownNames are the option names a select's rows draw, in order.
func shownNames(th *theme.Theme, s Select[string]) []string {
	var out []string
	txt := ansi.Strip(strings.Join(s.View(th, 60, true), "\n"))
	for _, o := range s.Options {
		if strings.Contains(txt, o.Name+" ") {
			out = append(out, o.Name)
		}
	}
	return out
}

// Typing narrows a filterable select to the names holding the letters in order, the
// highlight stays on a shown option as the filter changes (and is the value), the
// arrows move through the shown options only, and clearing the filter keeps the option
// chosen. Backspace widens it; with no filter it is not the select's.
func TestSelectFilterNarrows(t *testing.T) {
	th := testTheme(t, false)
	s := clusters()
	typeInto(&s, "PRD")
	if !s.Filtered() || s.Query() != "PRD" {
		t.Fatalf("typed PRD, the filter is %q", s.Query())
	}
	if got, want := shownNames(th, s), []string{"prod-us-east-1", "prod-eu-west-1"}; !slices.Equal(got, want) {
		t.Fatalf("PRD shows %v, want %v", got, want)
	}
	if s.Value() != "prod-us-east-1" {
		t.Fatalf("the highlight %q is an option the filter hides", s.Value())
	}
	s.Move(1)
	s.Move(1)
	if s.Value() != "prod-eu-west-1" {
		t.Fatalf("down twice reached %q, want the last shown, prod-eu-west-1", s.Value())
	}
	typeInto(&s, "eu")
	if s.Value() != "prod-eu-west-1" || s.Matches() != 1 {
		t.Fatalf("PRDeu: %d matches, highlight %q", s.Matches(), s.Value())
	}
	if !s.Type(tea.KeyPressMsg{Code: tea.KeyBackspace}) || !s.Type(tea.KeyPressMsg{Code: tea.KeyBackspace}) || s.Query() != "PRD" {
		t.Fatalf("backspace left %q", s.Query())
	}
	if s.Type(tea.KeyPressMsg{Code: tea.KeyDown}) {
		t.Fatal("an arrow key went to the filter")
	}
	s.ClearFilter()
	if s.Filtered() || s.Matches() != 5 || s.Value() != "prod-eu-west-1" {
		t.Fatalf("cleared: filtered %v, %d shown, value %q", s.Filtered(), s.Matches(), s.Value())
	}
	if s.Type(tea.KeyPressMsg{Code: tea.KeyBackspace}) {
		t.Fatal("backspace with no filter claimed the key")
	}
	// scrolled: the highlight stays in the window as the filter narrows the list
	s.Rows = 2
	typeInto(&s, "st")
	if got := shownNames(th, s); len(got) != 2 || !slices.Contains(got, s.Selected().Name) {
		t.Fatalf("filtered and scrolled: shows %v, highlight %q", got, s.Selected().Name)
	}
}

// A filter that leaves nothing draws the kit's empty state, not a blank, and the field
// keeps its height as the options narrow (the filter's line added at the first letter),
// at every width, focused or not, in every theme. With no filter a filterable select
// draws exactly as any select.
func TestSelectFilterEmptyAndHeight(t *testing.T) {
	eachTheme(t, func(name string, th *theme.Theme) {
		for _, q := range []string{"", "dev", "zzz"} {
			s := clusters()
			typeInto(&s, q)
			for _, w := range []int{24, 40, 80} {
				for _, focused := range []bool{true, false} {
					rows := s.View(th, w, focused)
					if h := s.Height(th, w); h != len(rows) {
						t.Errorf("%s %q w=%d focused=%v: Height %d, View draws %d rows", name, q, w, focused, h, len(rows))
					}
					for _, r := range rows {
						if ansi.StringWidth(r) != w {
							t.Errorf("%s %q w=%d: a row is %d cells: %q", name, q, w, ansi.StringWidth(r), ansi.Strip(r))
						}
					}
				}
			}
			// the filter's line and its air come with the first letter; past it the
			// height holds as the options narrow, so what's under the field stays put
			want := clusters().Height(th, 60)
			if q != "" {
				want += 2
			}
			if h := s.Height(th, 60); h != want {
				t.Errorf("%s %q: the field is %d rows, want %d", name, q, h, want)
			}
		}
		s := clusters()
		typeInto(&s, "zzz")
		txt := ansi.Strip(strings.Join(s.View(th, 60, true), "\n"))
		if s.Matches() != 0 || !strings.Contains(txt, `No option matches "zzz"`) || !strings.Contains(txt, "0 of 5") {
			t.Errorf("%s: a filter that leaves nothing draws:\n%s", name, txt)
		}
		// focus moves no text: the blurred filter line keeps the query's cells
		for _, str := range []string{"zzz", "No option matches"} {
			if a, b := cellIndex(strings.Join(s.View(th, 60, true), ""), str), cellIndex(strings.Join(s.View(th, 60, false), ""), str); a != b {
				t.Errorf("%s: %q moves on blur: %d → %d", name, str, a, b)
			}
		}
	})
}

// Filtering is opt-in: a select without it draws no filter line, takes no typed key,
// and moves as it always did.
func TestSelectUnfilteredUnchanged(t *testing.T) {
	th := testTheme(t, false)
	_, sel, _ := sampleFields()
	before := sel.View(th, 60, true)
	if sel.Type(tea.KeyPressMsg{Code: 'j', Text: "j"}) || sel.Filtered() {
		t.Fatal("a select that isn't Filterable took a typed key")
	}
	if got := sel.View(th, 60, true); !slices.Equal(got, before) {
		t.Fatal("a typed key changed an unfilterable select")
	}
	if len(before) != 2+len(sel.Options) || strings.Contains(ansi.Strip(strings.Join(before, "")), "type to filter") {
		t.Fatalf("an unfilterable select drew a filter line:\n%s", ansi.Strip(strings.Join(before, "\n")))
	}
}

// A secret input draws a mask for each character, focused and blurred, in every theme
// and under ASCII, and never the value; the plain prompt never writes it either. The
// messages here name lengths, never the value, so a failure can't print it.
func TestInputSecretIsMasked(t *testing.T) {
	const secret = "hunter2-tok"
	eachTheme(t, func(name string, th *theme.Theme) {
		in := NewInput("Token", "from your profile", "paste a token", "")
		in.Secret = true
		for _, r := range secret {
			in.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
		}
		if in.Value() != secret {
			t.Fatalf("%s: typed %d characters, the value holds %d", name, len(secret), len(in.Value()))
		}
		mask := strings.Repeat(string(in.Model.EchoCharacter), len(secret))
		for _, focused := range []bool{true, false} {
			txt := ansi.Strip(strings.Join(in.View(th, 60, focused), "\n"))
			if strings.Contains(txt, "hunter") || strings.Contains(txt, "tok") {
				t.Errorf("%s focused=%v: the secret's text is on screen", name, focused)
			}
			if !strings.Contains(txt, mask) {
				t.Errorf("%s focused=%v: no %d-cell mask on screen", name, focused, len(secret))
			}
		}
	})
	// an EchoMode set by hand is a secret too: the plain prompt doesn't name the value
	in := NewInput("Token", "", "", secret)
	in.Model.EchoMode = textinput.EchoPassword
	var out bytes.Buffer
	if err := in.Ask(NewPrompter(strings.NewReader("\n"), &out)); err != nil || in.Value() != secret {
		t.Fatalf("a blank answer did not keep the secret (err %v)", err)
	}
	if strings.Contains(out.String(), "hunter") || !strings.Contains(out.String(), "blank keeps the current one") {
		t.Errorf("the prompt for a kept secret wrote its value or no hint (%d bytes)", out.Len())
	}
	// a typed secret from a pipe is read as given and never written back
	out.Reset()
	typed := NewInput("Token", "", "", "")
	typed.Secret = true
	if err := typed.Ask(NewPrompter(strings.NewReader(secret+"\n"), &out)); err != nil || typed.Value() != secret {
		t.Fatalf("the piped secret was not read (err %v, %d characters)", err, len(typed.Value()))
	}
	if strings.Contains(out.String(), "hunter") {
		t.Error("the plain prompt wrote the secret to its output")
	}
}
