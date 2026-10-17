// Command glyph holds glyph literals where they belong and where they don't: C-GLYPH
// fails a screen's, and passes the theme package's, the ASCII twin table's file, a
// comment's, a test file's, and text in another script.
package main

import (
	"fmt"

	"example.com/glyph/theme"
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
	return sep + string(arrow) + caret + more + prose + theme.Caret + ui.Twin(theme.Caret)
}
