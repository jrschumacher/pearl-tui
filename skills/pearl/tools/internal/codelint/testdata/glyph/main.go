// Command glyph holds glyph literals where they belong and where they don't: C-GLYPH
// fails a screen's, and passes the theme package's, the ASCII twin table's file, a
// comment's, a test file's, and text in another script. C-ELLIPSIS fails the bare
// theme.Ellipsis in a screen, from theme/ or one framework's theme/<framework>, and
// passes it in the ui package.
package main

import (
	"fmt"

	"example.com/glyph/theme"
	fwtheme "example.com/glyph/theme/bubbletea"
	"example.com/glyph/ui"
)

func main() {
	fmt.Println(view())
}

func view() string {
	sep := " · "              // want `C-GLYPH: glyph '·' \(U\+00B7\) in a string literal`
	arrow := '→'              // want `C-GLYPH: glyph '→' \(U\+2192\) in a rune literal`
	caret := `▏`              // want "C-GLYPH"
	more := "↑ 3 more" + " ×" // want "C-GLYPH: glyph '↑'"
	// a comment may hold ▏ · → freely: nothing draws it
	prose := "café, 日本語, naïve, ١٢٣"
	// punctuation, currency, units and maths are text a screen prints, not marks it draws
	text := "it’s «quoted» — a dash – and “curly” quotes; £5, €3, 72°F, ©2026 ™, 3×4 ÷ 2 ± 1 ≥ ½ µs ²"
	marks := "✓ done"                                    // want "C-GLYPH: glyph '✓'"
	cut := ui.Fit("a long title", true) + theme.Ellipsis // want `C-ELLIPSIS: cut with ui.Trunc\(t, s, w\)`
	fwcut := "a long title" + fwtheme.Ellipsis           // want `C-ELLIPSIS: cut with ui.Trunc\(t, s, w\)`
	spark := "⣀⣤⣿"                                       // want "C-GLYPH: glyph '⣀'"
	return sep + string(arrow) + caret + more + prose + text + marks + spark + cut + fwcut + fwtheme.Sep + theme.Caret + ui.Twin(theme.Caret)
}
