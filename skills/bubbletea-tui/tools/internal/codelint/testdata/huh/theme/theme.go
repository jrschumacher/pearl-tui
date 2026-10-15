// Package theme is the brand: Huh and spinner styles built from its roles.
package theme

import (
	"charm.land/huh/v2"
	huhspinner "charm.land/huh/v2/spinner"
	"charm.land/lipgloss/v2"
)

var accent = lipgloss.Color("#3FA7A0")

// Accent is the spinner style.
func Accent() lipgloss.Style { return lipgloss.NewStyle().Foreground(accent) }

// FormTheme starts from Huh's base (stock, but not handed to WithTheme) and brands it.
func FormTheme() huh.Theme {
	return huh.ThemeFunc(func(isDark bool) *huh.Styles {
		s := huh.ThemeBase(isDark)
		s.Focused.Title = s.Focused.Title.Foreground(accent)
		return s
	})
}

// SpinnerTheme is the brand's Huh spinner theme.
func SpinnerTheme() huhspinner.Theme {
	return huhspinner.ThemeFunc(func(bool) *huhspinner.Styles {
		return &huhspinner.Styles{Spinner: Accent(), Title: lipgloss.NewStyle()}
	})
}

// Plain is a library's exported constructor: its callers, in other packages, theme it.
func Plain(g *huh.Group) *huh.Form { return huh.NewForm(g) }

// Keep takes a form somewhere this package's callers cannot see.
func Keep(*huh.Form) {}

// Confirm runs a yes/no question.
func Confirm() error {
	return bare().Run()
}

func bare() *huh.Form {
	return huh.NewForm(huh.NewGroup(huh.NewConfirm())) // want `C-HUHTHEME: huh form without WithTheme: it renders Huh's default colours, not the brand`
}
