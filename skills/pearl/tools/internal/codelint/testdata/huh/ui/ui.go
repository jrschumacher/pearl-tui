// Package ui stands in for the kit's ui/bubbletea: a Bubbles spinner in the brand.
package ui

import (
	"charm.land/bubbles/v2/spinner"

	"example.com/huh/theme"
)

// BubblesSpinner is a Bubbles spinner in the brand's style.
func BubblesSpinner() spinner.Model { return spinner.New(spinner.WithStyle(theme.Accent())) }
