// Package look picks styles for the terminal's background.
package look

import "charm.land/lipgloss/v2"

// Title is the title style for a dark or light background.
func Title(dark bool) lipgloss.Style {
	pick := lipgloss.LightDark(dark)
	return lipgloss.NewStyle().Bold(true).Foreground(pick(lipgloss.Black, lipgloss.White))
}
