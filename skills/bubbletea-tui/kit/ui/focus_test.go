package ui

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
	uv "github.com/charmbracelet/ultraviolet"

	"tuikit/theme"
)

// quire is the kit's focus "fg" test theme: amber is its only hue and means "new", so
// no focus, selection, cursor, prompt or wordmark may be drawn in it.
func quire(t *testing.T) theme.File {
	t.Helper()
	f, err := theme.Load(filepath.Join("..", "theme", "testdata", "quire.json"))
	if err != nil {
		t.Fatal(err)
	}
	return f
}

// primaryCells renders s at profile p, reads it back as the terminal holds it, and
// returns the cells whose ink or fill is primary as the terminal shows it.
func primaryCells(th *theme.Theme, s string, p colorprofile.Profile) []string {
	w, h := lipgloss.Width(s), lipgloss.Height(s)
	buf := uv.NewScreenBuffer(w, h)
	uv.NewStyledString(downsample(s, p)).Draw(buf, buf.Bounds())
	prim := p.Convert(th.P.Primary)
	if prim == nil {
		return nil // no colour at all: nothing can be amber
	}
	var hits []string
	for y := range h {
		for x := range w {
			c := buf.CellAt(x, y)
			if c == nil {
				continue
			}
			if (c.Style.Fg != nil && sameColor(c.Style.Fg, prim)) || (c.Style.Bg != nil && sameColor(c.Style.Bg, prim)) {
				hits = append(hits, fmt.Sprintf("%q@%d,%d", c.Content, x, y))
			}
		}
	}
	return hits
}

// Under expressions.focus "fg" the kit draws focus, the current step, selection, the
// cursor, the prompt, the focused button in every button style, the wordmark "plain"
// and its own marks (spinner, progress, busy dot, header and dialog-title rules) in
// neutrals and weight: primary appears only where the theme or the app asks for it.
func TestFocusFgKeepsPrimaryOffChrome(t *testing.T) {
	base := quire(t)
	modes := []struct {
		name        string
		dark, ascii bool
		p           colorprofile.Profile
	}{
		{"dark", true, false, colorprofile.TrueColor}, {"light", false, false, colorprofile.TrueColor},
		{"ascii", true, true, colorprofile.TrueColor}, {"ansi", true, false, colorprofile.ANSI},
		{"ansi256", true, false, colorprofile.ANSI256},
	}
	for _, button := range []string{"underline", "bracket", "pill", "inverse"} {
		for _, sel := range []string{"bar", "caret", "fill", "inverse"} {
			f := base
			f.Expressions.Button, f.Expressions.Selection = button, sel
			for _, m := range modes {
				th := theme.ResolveProfile(f, m.dark, m.ascii, m.p)
				// the kit's own rungs: a rung the theme's ladder puts in primary is content
				// the brand asked for (TestThemeLadder), not chrome
				kit := f
				kit.Ladder = nil
				items := []Item{{Title: "The long read", State: "busy"}, {Title: "Weekly notes"}}
				steps := Steps{Names: []string{"Account", "Feeds", "Done"}, Cur: 1}
				huh := HuhStyles(th).Focused
				chrome := map[string]string{
					"focus bar":        Gutter(th, true, th.S.FocusBar),
					"list cursor":      strings.Join(List(th, items, 0, 0, 40, 2, true), "\n"),
					"focused button":   Button(th, "Subscribe", true, false, 0),
					"steps":            steps.Row(th, 60),
					"steps list":       strings.Join(steps.List(th), "\n"),
					"ladder":           strings.Join(Ladder(theme.ResolveProfile(kit, m.dark, m.ascii, m.p), 0), "\n"),
					"marks":            Mark(th, "current") + Mark(th, "partial") + Mark(th, "todo") + Mark(th, "done"),
					"tabs":             strings.Join(Tabs(th, []string{"Inbox", "Saved"}, 0, nil), "\n"),
					"wordmark":         Wordmark(th, "quire"),
					"header":           strings.Join(Header(th, 60, 24, "quire", "inbox"), "\n"),
					"spinner":          Spinner(th, 0),
					"progress":         Progress(th, 0.5, 20),
					"user turn":        strings.Join(UserMessage(th, "summarise this", 40, false), "\n"),
					"dialog":           Dialog(th, 40, 0, "Unsubscribe?", []string{"You can come back."}, "", ""),
					"huh selector":     huh.SelectSelector.Render() + huh.SelectedPrefix.Render() + huh.NextIndicator.Render(),
					"huh focus border": huh.Base.Render("field"),
					"huh prompt":       huh.TextInput.Prompt.Render("›") + huh.TextInput.Cursor.Render(" "),
					"footer":           strings.Join(Footer(th, 60, "", []key.Binding{key.NewBinding(key.WithKeys("j"), key.WithHelp("j", "next"))}, ""), "\n"),
				}
				for what, s := range chrome {
					if hits := primaryCells(th, s, m.p); len(hits) > 0 {
						t.Errorf("button=%s selection=%s %s: %s draws primary at %v", button, sel, m.name, what, hits)
					}
				}
			}
		}
	}
}
