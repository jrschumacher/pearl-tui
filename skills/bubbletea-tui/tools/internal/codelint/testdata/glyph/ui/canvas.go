// Package ui draws screens. This file declares the ASCII twin tables: the literals
// inside them are the glyphs the tables map, so C-GLYPH skips those and nothing else
// in the file.
package ui

import "strings"

var asciiMap = map[rune]string{
	'─': "-", '│': "|", '·': ".", '→': ">",
}

// asciiClasses map runs of glyphs to one twin each.
var asciiClasses = []struct{ glyphs, twin string }{
	{"╭╮╰╯┌┐└┘", "+"}, {"█▓▒", "#"},
}

// Twin returns the ASCII stand-in for a glyph.
func Twin(s string) string {
	r := []rune(s)[0]
	for _, c := range asciiClasses {
		if strings.ContainsRune(c.glyphs, r) {
			return c.twin
		}
	}
	if strings.ContainsRune("╱╲", r) { // want "C-GLYPH: glyph '╱'"
		return "/"
	}
	if t, ok := asciiMap[r]; ok {
		return t
	}
	return "?"
}
