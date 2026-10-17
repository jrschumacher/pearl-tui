// Package ui draws screens. This file declares the ASCII twin table, so its glyph
// literals are the glyphs the table maps: C-GLYPH skips the file.
package ui

import "strings"

var asciiMap = map[rune]string{
	'─': "-", '│': "|", '·': ".", '→': ">",
}

// Twin returns the ASCII stand-in for a glyph.
func Twin(s string) string {
	r := []rune(s)[0]
	if strings.ContainsRune("╭╮╰╯┌┐└┘", r) {
		return "+"
	}
	if t, ok := asciiMap[r]; ok {
		return t
	}
	return "?"
}
