// Package legacy still imports Lip Gloss v1.
package legacy

import "github.com/charmbracelet/lipgloss" // want "C-V1"

// Border is the library example's border colour.
var Border = lipgloss.Color("62") // want "C-HEX"

// Pink is the bubbles default.
var Pink = lipgloss.Color("#FF5FAF") // want "C-HEX"
