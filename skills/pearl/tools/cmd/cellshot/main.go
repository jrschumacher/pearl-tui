package main

import (
	_ "embed"
	"os"
	"strings"

	"tuitools/internal/cellshot"
)

// The package doc comment is the -h text: embed it rather than keep a second copy.
//
//go:embed doc.go
var docSource string

// usage turns doc.go's comment block into plain text.
func usage() string {
	var b strings.Builder
	for _, line := range strings.Split(docSource, "\n") {
		if !strings.HasPrefix(line, "//") {
			break
		}
		line = strings.TrimPrefix(strings.TrimPrefix(line, "//"), " ")
		if strings.HasPrefix(line, "\t") {
			line = "  " + line[1:]
		}
		b.WriteString(line + "\n")
	}
	return b.String()
}

func main() {
	os.Exit(cellshot.Main(os.Args[1:], os.Stdin, os.Stdout, os.Stderr, usage()))
}
