// Command tuilint catches TUI slop deterministically, in rendered frames and in Go source.
//
//	tuilint frames <dir|files...> [--cols W --rows H] [--theme theme.json] [--json out.json]
//	tuilint code <dir> [--theme-pkg path/fragment] [--json out.json]
//	tuilint hook < posttooluse.json
//
// Every rule has an ID. Rules marked P0/P1 fail the run (exit 1); P2 are warnings.
//
// `hook` is the Claude Code plugin's PostToolUse hook: it reads the hook input on stdin
// and, for a .go file in a Bubble Tea or Huh module of a project that opted in to the
// skill (a TUI.md at the module or repository root, or a copy of the kit's theme
// package), runs the code rules and prints the edited file's P0/P1 findings to stderr
// with exit 2, which Claude Code shows the model. It exits 0 otherwise, and on its own
// errors after a one-line note.
package main

import (
	"fmt"
	"os"

	"tuitools/internal/codelint"
	"tuitools/internal/framelint"
	"tuitools/internal/hook"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: tuilint frames|code|hook ...")
		os.Exit(2)
	}
	switch os.Args[1] {
	case "frames":
		os.Exit(framelint.Main(os.Args[2:], os.Stdout, os.Stderr))
	case "code":
		os.Exit(codelint.Main(os.Args[2:], os.Stdout, os.Stderr))
	case "hook":
		os.Exit(hook.Main(os.Stdin, os.Stdout, os.Stderr))
	default:
		fmt.Fprintf(os.Stderr, "tuilint: unknown mode %q (frames|code|hook)\n", os.Args[1])
		os.Exit(2)
	}
}
