// Package hook is `tuilint hook`: a Claude Code PostToolUse hook that lints the Go
// package an edit touched and hands its blocking findings back to the model.
//
// It reads the hook input (JSON on stdin), and acts only on a .go file (not a test, not
// under testdata or vendor) in a module that requires charm.land/bubbletea/v2 or
// charm.land/huh/v2, or in a file that imports one of them. It lints the whole module,
// so the program-wide rules (C-SIZE, C-KEYS, C-NOCOLOR) see the packages the edited one
// imports, and reports the P0/P1 findings in the edited file's package directory.
//
// Exit 2 with the findings on stderr is Claude Code's PostToolUse feedback channel: the
// tool already ran, and stderr is shown to the model next to its result. Everything else
// exits 0. The hook's own failures (bad input, a module that won't load) print one line
// to stderr and exit 0, so the hook never gets in the way of an edit.
package hook

import (
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"tuitools/internal/codelint"
)

// TUIModules are the module paths that make a module a TUI worth linting.
var TUIModules = []string{"charm.land/bubbletea/v2", "charm.land/huh/v2"}

// maxShown caps the findings printed, so one bad package doesn't flood the context.
const maxShown = 12

// Input is the part of the PostToolUse hook input the hook reads.
type Input struct {
	Cwd       string `json:"cwd"`
	ToolName  string `json:"tool_name"`
	ToolInput struct {
		FilePath string `json:"file_path"`
	} `json:"tool_input"`
	ToolResponse struct {
		FilePath string `json:"filePath"`
	} `json:"tool_response"`
}

// Main runs the hook on the input read from stdin and returns the exit status.
func Main(stdin io.Reader, stderr io.Writer) int {
	var in Input
	if err := json.NewDecoder(stdin).Decode(&in); err != nil {
		fmt.Fprintf(stderr, "tuilint hook: unreadable hook input (%v); skipped\n", err)
		return 0
	}
	file := in.ToolInput.FilePath
	if file == "" {
		file = in.ToolResponse.FilePath
	}
	if file == "" {
		return 0
	}
	if !filepath.IsAbs(file) && in.Cwd != "" {
		file = filepath.Join(in.Cwd, file)
	}
	modDir, ok := Target(file)
	if !ok {
		return 0
	}
	// The hook must stay fast and offline: a module whose dependencies aren't downloaded
	// fails the typed load at once and is linted syntax-only, instead of fetching.
	os.Setenv("GOPROXY", "off")
	findings, err := codelint.Lint(modDir, "", io.Discard)
	if err != nil {
		fmt.Fprintf(stderr, "tuilint hook: %s: %v; skipped\n", modDir, firstLine(err))
		return 0
	}
	pkg, _ := filepath.Rel(modDir, filepath.Dir(file))
	pkg = filepath.ToSlash(pkg)
	var blocking []codelint.Finding
	for _, f := range findings {
		if f.Sev != "P2" && path.Dir(f.File) == pkg {
			blocking = append(blocking, f)
		}
	}
	if len(blocking) == 0 {
		return 0
	}
	codelint.Sort(blocking)
	shown := func(rel string) string { return display(in.Cwd, filepath.Join(modDir, filepath.FromSlash(rel))) }
	noun, them := "problems", "them"
	if len(blocking) == 1 {
		noun, them = "problem", "it"
	}
	fmt.Fprintf(stderr, "tuilint code: %d blocking %s in %s after this edit. Fix %s before you go on:\n",
		len(blocking), noun, shown(pkg), them)
	for i, f := range blocking {
		if i == maxShown {
			fmt.Fprintf(stderr, "  … and %d more: run `tuilint code %s`\n", len(blocking)-maxShown, shown("."))
			break
		}
		fmt.Fprintf(stderr, "  %s %s:%s  %s\n", f.Rule, shown(f.File), f.Where, f.Msg)
	}
	fmt.Fprintln(stderr, "Each rule is explained by `tuilint code -h`. If a finding is wrong, tell the user why instead of working around it.")
	return 2
}

// Target reports whether an edit to file should be linted, and the module directory to
// lint. It holds for a non-test .go file outside testdata and vendor, in a module that
// requires one of TUIModules or in a file that imports one.
func Target(file string) (string, bool) {
	if !strings.HasSuffix(file, ".go") || strings.HasSuffix(file, "_test.go") {
		return "", false
	}
	for _, el := range strings.Split(filepath.ToSlash(filepath.Dir(file)), "/") {
		if el == "testdata" || el == "vendor" {
			return "", false
		}
	}
	if _, err := os.Stat(file); err != nil {
		return "", false
	}
	modDir, gomod := findModule(filepath.Dir(file))
	if modDir == "" {
		return "", false
	}
	if requiresTUI(gomod) || importsTUI(file) {
		return modDir, true
	}
	return "", false
}

// findModule returns the directory and contents of the go.mod at or above dir.
func findModule(dir string) (string, string) {
	for d := dir; ; d = filepath.Dir(d) {
		if b, err := os.ReadFile(filepath.Join(d, "go.mod")); err == nil {
			return d, string(b)
		}
		if filepath.Dir(d) == d {
			return "", ""
		}
	}
}

// requiresTUI reports whether a go.mod requires one of TUIModules, in a require block or
// on a require line of its own.
func requiresTUI(gomod string) bool {
	for _, ln := range strings.Split(gomod, "\n") {
		ln, _, _ = strings.Cut(ln, "//")
		for _, f := range strings.Fields(ln) {
			if isTUI(strings.Trim(f, `"`)) {
				return true
			}
		}
	}
	return false
}

// importsTUI reports whether a Go file imports a package of one of TUIModules, for an
// app whose go.mod hasn't caught up with its code yet.
func importsTUI(file string) bool {
	f, _ := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly) // a partial parse still lists imports
	if f == nil {
		return false
	}
	for _, s := range f.Imports {
		if p, err := strconv.Unquote(s.Path.Value); err == nil && isTUI(p) {
			return true
		}
	}
	return false
}

func isTUI(p string) bool {
	for _, m := range TUIModules {
		if p == m || strings.HasPrefix(p, m+"/") {
			return true
		}
	}
	return false
}

// display is p relative to the session's working directory when it lies inside it, so
// the model reads the same paths it edits; otherwise p as it is.
func display(cwd, p string) string {
	if cwd != "" {
		if rel, err := filepath.Rel(cwd, p); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return filepath.ToSlash(rel)
		}
	}
	return p
}

func firstLine(err error) string {
	s, _, _ := strings.Cut(err.Error(), "\n")
	return s
}
