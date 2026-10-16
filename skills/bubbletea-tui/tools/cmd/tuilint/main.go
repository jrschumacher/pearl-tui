// Command tuilint catches TUI slop deterministically, in rendered frames and in Go source.
//
//	tuilint frames <dir|files...> [--cols W --rows H] [--theme theme.json] [--bg #rrggbb] [--json out.json]
//	tuilint code <dir...> [--theme-pkg path/fragment] [--json out.json]
//
// Every rule has an ID. Rules marked P0/P1 fail the run (exit 1); P2 are warnings.
// tuilint frames -h and tuilint code -h list the rules.
package main

import (
	_ "embed"
	"fmt"
	"os"

	"tuitools/internal/cmddoc"
	"tuitools/internal/codelint"
	"tuitools/internal/framelint"
)

//go:embed main.go
var src string

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, cmddoc.Text(src))
		os.Exit(2)
	}
	switch os.Args[1] {
	case "-h", "--help", "help":
		fmt.Println(cmddoc.Text(src))
	case "frames":
		os.Exit(framelint.Main(os.Args[2:], os.Stdout, os.Stderr))
	case "code":
		os.Exit(codelint.Main(os.Args[2:], os.Stdout, os.Stderr))
	default:
		fmt.Fprintf(os.Stderr, "tuilint: unknown mode %q (frames|code)\n", os.Args[1])
		os.Exit(2)
	}
}
