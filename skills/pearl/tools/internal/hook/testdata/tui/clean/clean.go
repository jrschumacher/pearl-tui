// Package clean pads a label by its display width: nothing for tuilint to find.
package clean

import "charm.land/lipgloss/v2"

// Pad right-pads s with spaces to w cells.
func Pad(s string, w int) string {
	if n := lipgloss.Width(s); n < w {
		return s + lipgloss.NewStyle().Width(w-n).Render("")
	}
	return s
}
