// Package huhform stands in for the kit's ui/bubbletea/huhform: a form and a spinner
// built already themed, so a caller that goes through them is not flagged.
package huhform

import (
	"charm.land/huh/v2"
	huhspinner "charm.land/huh/v2/spinner"

	"example.com/huh/theme"
)

// NewForm is huh.NewForm with the brand's theme.
func NewForm(groups ...*huh.Group) *huh.Form {
	return huh.NewForm(groups...).WithTheme(theme.FormTheme())
}

// Spinner is Huh's spinner in the brand.
func Spinner(title string) *huhspinner.Spinner {
	return huhspinner.New().Title(title).WithTheme(theme.SpinnerTheme())
}
