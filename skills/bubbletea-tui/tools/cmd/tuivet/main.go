// Command tuivet runs the tuilint code rules as go/analysis analyzers, so they work under
// go vet and in editors:
//
//	go vet -vettool=$(which tuivet) ./...
//	tuivet ./...                          # standalone, same analyzers
//	tuivet -tuihex.theme-pkg=brand ./...  # an extra theme package for C-HEX
//
// Each diagnostic starts with its rule ID (C-HEX, C-SLEEP, ...). The analyzers are the
// -tui* flags below; -<analyzer>=false switches one off. It exits 3 when a diagnostic
// is reported (go vet's convention), 0 when none, 1 or 2 on a load error or usage.
// The same rules, with tuilint's ID-first output and P0/P1 exit codes, are
// `tuilint code`.
package main

import (
	_ "embed"
	"fmt"
	"os"

	"golang.org/x/tools/go/analysis/multichecker"

	"tuitools/internal/cmddoc"
	"tuitools/internal/codelint"
)

//go:embed main.go
var src string

func main() {
	// The package doc is the -h text, as for the other tools: print it before
	// multichecker's flag list (which it prints itself, then exits).
	for _, a := range os.Args[1:] {
		if a == "-h" || a == "-help" || a == "--help" {
			fmt.Println(cmddoc.Text(src))
			break
		}
	}
	multichecker.Main(codelint.Analyzers...)
}
