// Command huh holds the C-HUHTHEME cases with full types: Huh forms, fields and
// spinners, and Bubbles spinners, themed and not, chained and through variables.
package main

import (
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	huhspinner "charm.land/huh/v2/spinner"

	"example.com/huh/theme"
)

func group() *huh.Group {
	var name string
	return huh.NewGroup(huh.NewInput().Title("Name").Value(&name)) // fields in a group: the form is what is themed
}

type model struct {
	form   *huh.Form
	wizard *huh.Form
	other  *huh.Form
	spin   spinner.Model
	dots   spinner.Model
	busy   spinner.Model
}

func newModel() model {
	m := model{
		wizard: huh.NewForm(group()), // want `C-HUHTHEME: huh form without WithTheme`
		spin:   spinner.New(),
		dots:   spinner.New(spinner.WithSpinner(spinner.Dot)), // want `C-HUHTHEME: spinner.New without spinner.WithStyle or a .Style`
		busy:   spinner.New(spinner.WithSpinner(spinner.Dot), spinner.WithStyle(theme.Accent())),
	}
	m.form = huh.NewForm(group()).WithShowHelp(false)
	m.other = huh.NewForm(group()) // want `C-HUHTHEME: huh form without WithTheme`
	return m
}

// brand themes the model's form in another method, through a helper's theme.
func (m *model) brand() {
	m.form = m.form.WithTheme(theme.FormTheme())
	m.spin.Style = theme.Accent()
}

func (m model) Update(msg tea.Msg) (model, tea.Cmd) {
	f, cmd := m.form.Update(msg)
	if f, ok := f.(*huh.Form); ok {
		m.form = f
	}
	return m, cmd
}

// forms run on their own

func chained() error {
	return huh.NewForm(group()).WithShowHelp(false).Run() // want `C-HUHTHEME: huh form without WithTheme: it renders Huh's default colours, not the brand`
}

func chainedThemed() error {
	return huh.NewForm(group()).
		WithShowHelp(false).
		WithTheme(theme.FormTheme()).
		Run()
}

func variable() error {
	form := huh.NewForm(group()) // want `C-HUHTHEME: huh form without WithTheme`
	return form.Run()
}

func variableThemed() error {
	form := huh.NewForm(group())
	form.WithTheme(theme.FormTheme())
	return form.Run()
}

func stock() error {
	return huh.NewForm(group()).
		WithTheme(huh.ThemeFunc(huh.ThemeCharm)). // want `C-HUHTHEME: huh form themed with huh.ThemeCharm, a stock theme`
		Run()
}

// helpers: a result themed by the caller, a parameter themed by the helper

func plainForm() *huh.Form { return huh.NewForm(group()) }

func bareForm() *huh.Form { return huh.NewForm(group()) } // want `C-HUHTHEME: huh form without WithTheme`

func branded(f *huh.Form) *huh.Form { return f.WithTheme(theme.FormTheme()) }

func runIt(f *huh.Form) error { return f.Run() }

func helpers() {
	_ = plainForm().WithTheme(theme.FormTheme()).Run()
	_ = bareForm().Run()
	_ = branded(huh.NewForm(group())).Run()
	_ = runIt(huh.NewForm(group())) // want `C-HUHTHEME: huh form without WithTheme`
	theme.Keep(huh.NewForm(group()))
	_ = theme.Plain(group()).WithTheme(theme.FormTheme()).Run()
}

func initOnly() tea.Cmd {
	return huh.NewForm(group()).Init() // want `C-HUHTHEME: huh form without WithTheme`
}

// fields run on their own

func fields() {
	var ok bool
	_ = huh.NewConfirm().Title("Ship it?").Value(&ok).Run() // want `C-HUHTHEME: huh.NewConfirm run without WithTheme`
	_ = huh.NewConfirm().Title("Ship it?").WithTheme(theme.FormTheme()).Run()
	_ = huh.Run(huh.NewNote().Title("Done"))                            // want `C-HUHTHEME: huh.NewNote run without WithTheme`
	sel := huh.NewSelect[string]().Options(huh.NewOptions("a", "b")...) // want `C-HUHTHEME: huh.NewSelect run without WithTheme`
	_ = sel.Run()
	_ = huh.NewInput().Title("Unused") // made, never run: nothing to flag
}

// spinners

func spinners() {
	_ = huhspinner.New().Title("Deploying").Run() // want `C-HUHTHEME: huh spinner without WithTheme: it renders Huh's default fuchsia #F780E2, not the brand`
	_ = huhspinner.New().Title("Deploying").WithTheme(theme.SpinnerTheme()).Run()
	_ = huhspinner.New().WithTheme(huhspinner.ThemeFunc(huhspinner.ThemeDefault)).Run() // want `C-HUHTHEME: huh spinner themed with spinner.ThemeDefault, a stock theme`

	s := spinner.New() // want `C-HUHTHEME: spinner.New without spinner.WithStyle or a .Style: it renders unstyled, not in the brand's accent`
	_ = s.View()
	t := spinner.New()
	t.Style = theme.Accent()
	_ = t.View()
	opts := []spinner.Option{spinner.WithStyle(theme.Accent())}
	_ = spinner.New(opts...).View()
}

func main() {
	m := newModel()
	m.brand()
	_, _ = m.Update(nil)
	_ = chained()
	_ = chainedThemed()
	_ = variable()
	_ = variableThemed()
	_ = stock()
	helpers()
	_ = initOnly()
	fields()
	spinners()
	_ = theme.Confirm()
}
