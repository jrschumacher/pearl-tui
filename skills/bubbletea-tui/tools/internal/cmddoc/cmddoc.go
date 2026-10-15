// Package cmddoc turns a command's package doc comment into its -h text, so the comment
// is the one place a tool is documented (as the Python tools' docstrings were).
package cmddoc

import "strings"

// Text extracts the doc comment above the package clause of a Go source file: "// "
// prefixes are dropped and "//\t" code lines are indented by two spaces.
func Text(src string) string {
	var out []string
	for line := range strings.SplitSeq(src, "\n") {
		if strings.HasPrefix(line, "package ") {
			break
		}
		switch {
		case strings.HasPrefix(line, "//\t"):
			out = append(out, "  "+line[3:])
		case strings.HasPrefix(line, "// "):
			out = append(out, line[3:])
		case line == "//":
			out = append(out, "")
		default:
			out = out[:0] // a blank line or code ends a comment that was not the doc
		}
	}
	return strings.Join(out, "\n")
}
