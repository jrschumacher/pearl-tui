// Command tuilint catches TUI slop deterministically: in rendered frames, in Bubble Tea Go
// source and in TUI.md. frames and spec judge any framework's output; code lints Bubble
// Tea v2 code, the one framework the kit supports today.
//
//	tuilint frames <dir|files...> [--cols W --rows H] [--theme theme.json] [--bg #rrggbb] [--spec TUI.md] [--json out.json]
//	tuilint code <dir...> [--theme-pkg path/fragment] [--json out.json]
//	tuilint spec [TUI.md...] [--frames DIR] [--json out.json]
//	tuilint hook < posttooluse.json
//
// Every rule has an ID. Rules marked P0/P1 fail the run (exit 1); P2 are warnings, and
// frames --max-warnings N fails on more than N of them. Exit 2 on usage errors.
// tuilint frames -h, tuilint code -h and tuilint spec -h list the rules; tuilint hook -h
// says when the hook runs.
//
// `hook` is the Claude Code plugin's PostToolUse hook: it reads the hook input on stdin
// and, for a .go file in a Bubble Tea or Huh module of a project that opted in to the
// skill (a TUI.md at the module or repository root, or a copy of the kit's theme
// package), runs the code rules and prints the edited file's P0/P1 findings to stderr
// with exit 2, which Claude Code shows the model. It exits 0 otherwise, and on its own
// errors after a one-line note.
package main

import (
	_ "embed"
	"fmt"
	"os"

	"tuitools/internal/cmddoc"
	"tuitools/internal/codelint"
	"tuitools/internal/framelint"
	"tuitools/internal/hook"
	"tuitools/internal/speclint"
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
	case "spec":
		os.Exit(speclint.Main(os.Args[2:], os.Stdout, os.Stderr))
	case "hook":
		if len(os.Args) > 2 && (os.Args[2] == "-h" || os.Args[2] == "--help") {
			fmt.Print(hook.Usage())
			os.Exit(0)
		}
		os.Exit(hook.Main(os.Stdin, os.Stdout, os.Stderr))
	default:
		fmt.Fprintf(os.Stderr, "tuilint: unknown mode %q (frames|code|spec|hook)\n", os.Args[1])
		os.Exit(2)
	}
}
