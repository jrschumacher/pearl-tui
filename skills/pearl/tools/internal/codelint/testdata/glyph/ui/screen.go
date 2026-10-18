package ui

// Row draws one row; a glyph typed here is a literal in a screen.
func Row(label string) string {
	return "▌ " + label // want "C-GLYPH: glyph '▌'"
}
