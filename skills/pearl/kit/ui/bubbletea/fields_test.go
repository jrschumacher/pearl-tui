package ui

import (
	"bytes"
	"errors"
	"image/color"
	"path/filepath"
	"slices"
	"strings"
	"testing"

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
