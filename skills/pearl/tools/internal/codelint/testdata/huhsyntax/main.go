// Command huhsyntax holds C-HUHTHEME cases that must hold without the libraries' types.
package main

import (
	"charm.land/bubbles/v2/spinner"
	"charm.land/huh/v2"
	huhspinner "charm.land/huh/v2/spinner"
	"charm.land/lipgloss/v2"
)

type styles struct{ accent lipgloss.Style }

// formTheme is the brand's Huh theme, built in a helper.
func formTheme(s styles) huh.Theme {
	return huh.ThemeFunc(func(isDark bool) *huh.Styles {
		t := huh.ThemeBase(isDark)
		t.Focused.Title = s.accent
		return t
	})
}

type model struct {
	st     styles
	form   *huh.Form
	wizard *huh.Form
	spin   spinner.Model
	dots   spinner.Model
}

func newModel(st styles) model {
	m := model{st: st, spin: spinner.New(), dots: spinner.New()} // want `C-HUHTHEME: spinner.New without spinner.WithStyle`
	m.form = huh.NewForm(huh.NewGroup(huh.NewConfirm()))
	m.wizard = huh.NewForm(huh.NewGroup(huh.NewConfirm())) // want `C-HUHTHEME: huh form without WithTheme`
	return m
}

func (m *model) brand() {
	m.form = m.form.WithTheme(formTheme(m.st))
	m.spin.Style = m.st.accent
}

func run(st styles) {
	_ = huh.NewForm(huh.NewGroup(huh.NewConfirm())).Run() // want `C-HUHTHEME: huh form without WithTheme`
	form := huh.NewForm(huh.NewGroup(huh.NewConfirm()))
	form = form.WithTheme(formTheme(st))
	_ = form.Run()
	_ = huh.NewForm(huh.NewGroup(huh.NewConfirm())).WithTheme(huh.ThemeFunc(huh.ThemeDracula)).Run() // want `C-HUHTHEME: huh form themed with huh.ThemeDracula, a stock theme`
	_ = huh.NewConfirm().Title("Sure?").Run()                                                        // want `C-HUHTHEME: huh.NewConfirm run without WithTheme`
	_ = huh.NewSelect[string]().Title("Pick").WithTheme(formTheme(st)).Run()
	_ = huhspinner.New().Title("Loading").Run() // want `C-HUHTHEME: huh spinner without WithTheme`
	_ = spinner.New(spinner.WithStyle(st.accent)).View()
}

func main() {
	m := newModel(styles{})
	m.brand()
	run(m.st)
}
