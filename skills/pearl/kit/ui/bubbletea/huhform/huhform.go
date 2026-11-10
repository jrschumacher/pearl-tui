// Package huhform themes Huh forms and spinners from the brand: the optional path, for an
// app that already has Huh forms or needs a field the kit's own lack. A new form uses
// the kit's own fields (ui: Input, Select, Confirm), which theme a blurred input and keep
// an error under its field. An app that imports this package links Huh; one that doesn't
// never does (ui imports no Huh).
//
// Huh keeps its own mechanics and layout (typing, cursors, validation, focus order,
// sizing, accessible prompts); the kit only gives it the theme's colours, glyphs and
// button shape through Huh's public theme API. Left on its default theme, Huh draws
// Charm's palette (#F780E2, #7571F9) and a pink spinner.
//
//	huhform.NewForm(t, w, h, groups...)  huh.NewForm with the theme, the app's footer as help, ACCESSIBLE honoured
//	huhform.Spinner                      Huh's spinner in the brand's frames and colour, a subject in the title
//
// Fields are Huh's own: huh.NewInput, huh.NewSelect, huh.NewConfirm. Where Huh's
// look falls short of the theme (a blurred input draws in Bubbles' default
// styles: Huh's theme reaches only the focused one), the fix belongs in Huh,
// not in a wrapper here.
//
// It lives beside ui, not in it: theme is the dependency-free contract any renderer
// reads; Huh is one consumer of it, like every other component here, and the only one
// that costs a dependency.
package huhform

import (
	"strings"

	"charm.land/bubbles/v2/spinner"
	"charm.land/huh/v2"
	hspinner "charm.land/huh/v2/spinner"
	"charm.land/lipgloss/v2"

	"github.com/jrschumacher/pearl-tui/skills/pearl/kit/theme/bubbletea"
	"github.com/jrschumacher/pearl-tui/skills/pearl/kit/ui/bubbletea"
)

// Theme maps the resolved theme onto Huh. It ignores Huh's isDark flag:
// Huh assumes a light terminal until a tea.BackgroundColorMsg reaches the
// form, but the kit has already resolved dark or light, so the flag can only
// be wrong. The styles are built once (Huh asks for them on every View of
// every field). A theme is applied to a field only once (Huh's WithTheme
// keeps the first), so when the theme changes, rebuild the form.
func Theme(t *theme.Theme) huh.Theme {
	s := Styles(t)
	return huh.ThemeFunc(func(bool) *huh.Styles { return s })
}

// Styles is every Huh style from the theme. Start from ThemeBase so no
// field Huh adds later inherits the Charm palette.
func Styles(t *theme.Theme) *huh.Styles {
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
	f.SelectSelector = t.S.Focus.Bold(true).SetString(ui.Fit(g.Cursor, cw) + " ")
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
	f.PrevIndicator = t.S.Focus.MarginRight(1).SetString(t.G.Back)

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
	f.FocusedButton = lipgloss.NewStyle().MarginRight(gap).Transform(func(l string) string { return ui.Button(t, l, true, false, -1) })
	f.BlurredButton = lipgloss.NewStyle().MarginRight(gap).Transform(func(l string) string { return ui.Button(t, l, false, false, -1) })
	f.Next = f.FocusedButton

	// blurred: the same cells, quieter. The select's chosen option keeps a
	// mark, but the radio glyph, never the cursor: one cursor on screen.
	b := *f
	b.Base = lipgloss.NewStyle().Border(blur, false, false, false, true).BorderForeground(blurFg).PaddingLeft(1)
	b.Card = b.Base
	b.Title = t.S.Muted
	b.NoteTitle = t.S.Muted.MarginBottom(1)
	b.Description = t.S.Subtle
	b.SelectSelector = t.S.Muted.SetString(ui.Fit(g.RadioOn, cw) + " ")
	b.MultiSelectSelector = lipgloss.NewStyle().SetString(strings.Repeat(" ", cw+1))
	b.SelectedOption = t.S.Base.Padding(0, lead)
	b.UnselectedOption = t.S.Muted.Padding(0, lead)
	b.NextIndicator = lipgloss.NewStyle()
	b.PrevIndicator = lipgloss.NewStyle()
	b.TextInput.Prompt = t.S.Subtle
	b.FocusedButton = lipgloss.NewStyle().MarginRight(gap).Transform(func(l string) string { return ui.ChosenButton(t, l) })
	b.Next = b.FocusedButton
	s.Blurred = b

	s.Help.Ellipsis = t.S.Subtle
	s.Help.ShortKey, s.Help.FullKey = t.S.Key, t.S.Key
	s.Help.ShortDesc, s.Help.FullDesc = t.S.KeyDesc, t.S.KeyDesc
	s.Help.ShortSeparator, s.Help.FullSeparator = t.S.Subtle, t.S.Subtle
	return s
}

// selected is the highlighted option per expressions.selection (a private copy of
// the one ui's native Select draws: exporting it would add a name nothing else uses). Huh draws the
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

// NewForm is huh.NewForm with the theme. Huh's own help line is off: the
// app's footer, read from the same action table that dispatches keys, is the
// help (law 6). With ACCESSIBLE=1 the form runs as plain prompts (form.Run())
// and takes Huh's colourless base theme: Huh writes those prompts straight to
// the output, so a brand colour there is escape codes in a screen reader's ear.
// After a resize, call the form's own WithWidth and WithHeight.
func NewForm(t *theme.Theme, w, h int, groups ...*huh.Group) *huh.Form {
	th := Theme(t)
	if ui.Accessible() {
		th = huh.ThemeFunc(huh.ThemeBase)
	}
	return huh.NewForm(groups...).
		WithTheme(th).
		WithShowHelp(false).
		WithShowErrors(true).
		WithAccessible(ui.Accessible()).
		WithWidth(w).
		WithHeight(h)
}

// frameRate is how fast the brand's spinner frames turn: one frame per two clock
// ticks, the same pace ui.Spinner gets from the app's one clock (ui's own copy).
const frameRate = 2 * ui.FrameInterval

// Spinner is Huh's standalone spinner (spinner.New().Action(...).Run())
// in the brand: the theme's frames, the working ink (S.Working) instead of
// Huh's #F780E2, the title in fgSoft with the gap Huh forgets (it draws the frame
// and the title with no space between), and a subject instead of Huh's
// default "Loading...". It honours ACCESSIBLE=1 (one plain line, no frames).
func Spinner(t *theme.Theme, title string) *hspinner.Spinner {
	st := &hspinner.Styles{Spinner: t.S.Working, Title: t.S.Soft.PaddingLeft(1)}
	return hspinner.New().
		Type(hspinner.Type(spinner.Spinner{Frames: t.G.Spinner, FPS: frameRate})).
		WithTheme(hspinner.ThemeFunc(func(bool) *hspinner.Styles { return st })).
		Title(title).
		WithAccessible(ui.Accessible())
}
