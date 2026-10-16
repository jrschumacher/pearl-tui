// Package hook is `tuilint hook`: a Claude Code PostToolUse hook that lints the Go file
// an edit touched and hands its blocking findings back to the model.
//
// It reads the hook input (JSON on stdin) and acts only on a .go file (not a test, not
// under testdata or vendor) in a module that requires charm.land/bubbletea/v2 or
// charm.land/huh/v2, or in a file that imports one of them, and only in a project that
// opted in to the bubbletea-tui skill. The plugin installs at user scope, so without the
// opt-in it would hold every Bubble Tea app on the machine to this kit's rules. A
// project opts in with either marker:
//
//   - a TUI.md (the file the skill's init command writes) in the module root or in any
//     directory above it, up to the repository root (the first directory holding .git);
//     outside a repository, only the module root counts
//   - a copy of the kit's theme package at theme/ or internal/theme/ in the module: a
//     package that declares type Roles and func Load or func Resolve
//
// It lints the whole module, so the program-wide rules (C-SIZE, C-NOCOLOR) see every
// package, and reports only the P0/P1 findings located in the edited file: problems
// elsewhere in the package or module were there before this edit and are not its doing.
//
// The go command runs offline and read-only: GOPROXY=off and -mod=readonly (-mod=vendor
// when the module vendors), which override a GOFLAGS=-mod=mod in the environment, so the
// hook never downloads a module or writes go.mod or go.sum. A module whose dependencies
// aren't in the module cache fails the typed load at once and is linted syntax-only.
//
// Exit 2 with the findings on stderr is Claude Code's PostToolUse feedback channel: the
// tool already ran, and stderr is shown to the model next to its result. Findings are
// the only exit 2, and their first line always starts "tuilint code: ", which the
// plugin's wrapper (hooks/tuilint-edit.sh) checks, so that a Go runtime crash, which
// also exits 2, never reaches the model as a finding. Everything else exits 0: a
// one-line note on stderr when the input is unreadable or the module won't load (Claude
// Code keeps it in the debug log), and on a panic in the hook a systemMessage on stdout,
// which Claude Code shows the user.
package hook

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"tuitools/internal/codelint"
)

// TUIModules are the module paths that make a module a TUI worth linting.
var TUIModules = []string{"charm.land/bubbletea/v2", "charm.land/huh/v2"}

// Marker is the file that opts a project in: the skill's init command writes it.
const Marker = "TUI.md"

// Header starts the first line of every findings report, and only of those.
const Header = "tuilint code: "

// maxShown caps the findings printed, so one bad file doesn't flood the context.
const maxShown = 12

// lint is codelint.LintWith, swapped out by tests.
var lint = codelint.LintWith

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

// Main runs the hook on the input read from stdin and returns the exit status: 2 with
// findings on stderr, 0 otherwise.
func Main(stdin io.Reader, stdout, stderr io.Writer) (code int) {
	defer func() {
		if r := recover(); r != nil {
			why, _, _ := strings.Cut(fmt.Sprint(r), "\n")
			msg := fmt.Sprintf("tuilint hook: crashed (%s); this edit was not linted. Please report it at https://github.com/jrschumacher/skill-bubbletea-tui/issues", why)
			json.NewEncoder(stdout).Encode(map[string]string{"systemMessage": msg})
			fmt.Fprintln(stderr, msg)
			code = 0
		}
	}()
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
	findings, err := lint(modDir, "", io.Discard, LoadConfig(modDir))
	if err != nil {
		fmt.Fprintf(stderr, "tuilint hook: %s: %v; skipped\n", modDir, firstLine(err))
		return 0
	}
	rel, _ := filepath.Rel(modDir, file)
	rel = filepath.ToSlash(rel)
	var blocking []codelint.Finding
	for _, f := range findings {
		if f.Sev != "P2" && f.File == rel {
			blocking = append(blocking, f)
		}
	}
	if len(blocking) == 0 {
		return 0
	}
	codelint.Sort(blocking)
	shown := display(in.Cwd, file)
	noun, them := "problems", "them"
	if len(blocking) == 1 {
		noun, them = "problem", "it"
	}
	exe := executable()
	fmt.Fprintf(stderr, "%s%d blocking %s in %s after this edit. Fix %s before you go on:\n",
		Header, len(blocking), noun, shown, them)
	for i, f := range blocking {
		if i == maxShown {
			fmt.Fprintf(stderr, "  … and %d more: run `%s code %s`\n", len(blocking)-maxShown, exe, display(in.Cwd, modDir))
			break
		}
		fmt.Fprintf(stderr, "  %s %s:%s  %s\n", f.Rule, shown, f.Where, f.Msg)
	}
	fmt.Fprintf(stderr, "Each rule is explained by `%s code -h`. If a finding is wrong, tell the user why instead of working around it.\n", exe)
	return 2
}

// LoadConfig is how the hook runs the go command in modDir: offline, and never writing
// go.mod or go.sum, whatever GOFLAGS says. A command-line -mod overrides one in GOFLAGS.
func LoadConfig(modDir string) codelint.Load {
	mod := "-mod=readonly"
	if _, err := os.Stat(filepath.Join(modDir, "vendor", "modules.txt")); err == nil {
		mod = "-mod=vendor"
	}
	return codelint.Load{
		Env:        append(os.Environ(), "GOPROXY=off"), // the last value of a key wins
		BuildFlags: []string{mod},
	}
}

// Target reports whether an edit to file should be linted, and the module directory to
// lint. It holds for a non-test .go file outside testdata and vendor, in a module that
// requires one of TUIModules or in a file that imports one, in a project that opted in
// (OptedIn).
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
	if (requiresTUI(gomod) || importsTUI(file)) && OptedIn(modDir) {
		return modDir, true
	}
	return "", false
}

// OptedIn reports whether the module at modDir belongs to a project built with the
// skill: a TUI.md in the module root or above it up to the repository root, or a copy
// of the kit's theme package in the module.
func OptedIn(modDir string) bool {
	return hasMarker(modDir) || hasKitTheme(modDir)
}

// hasMarker looks for Marker in modDir and each parent up to the first directory that
// holds .git (a directory, or a file in a worktree). Outside a repository it checks
// modDir alone, so a stray TUI.md in a home directory doesn't opt in everything below.
func hasMarker(modDir string) bool {
	var dirs []string
	for d := modDir; ; d = filepath.Dir(d) {
		dirs = append(dirs, d)
		if exists(filepath.Join(d, ".git")) {
			break
		}
		if filepath.Dir(d) == d {
			dirs = dirs[:1]
			break
		}
	}
	for _, d := range dirs {
		if fi, err := os.Stat(filepath.Join(d, Marker)); err == nil && fi.Mode().IsRegular() {
			return true
		}
	}
	return false
}

// hasKitTheme reports whether the module holds a copy of the kit's theme package where
// the skill puts it (theme/ or internal/theme/): a package declaring type Roles and func
// Load or func Resolve. It parses declarations only, without bodies.
func hasKitTheme(modDir string) bool {
	for _, dir := range []string{"theme", filepath.Join("internal", "theme")} {
		ents, err := os.ReadDir(filepath.Join(modDir, dir))
		if err != nil {
			continue
		}
		roles, api := false, false
		for _, e := range ents {
			n := e.Name()
			if e.IsDir() || !strings.HasSuffix(n, ".go") || strings.HasSuffix(n, "_test.go") {
				continue
			}
			f, _ := parser.ParseFile(token.NewFileSet(), filepath.Join(modDir, dir, n), nil, parser.SkipObjectResolution)
			if f == nil {
				continue
			}
			for _, d := range f.Decls {
				switch d := d.(type) {
				case *ast.FuncDecl:
					if d.Recv == nil && (d.Name.Name == "Load" || d.Name.Name == "Resolve") {
						api = true
					}
				case *ast.GenDecl:
					for _, s := range d.Specs {
						if ts, ok := s.(*ast.TypeSpec); ok && ts.Name.Name == "Roles" {
							roles = true
						}
					}
				}
			}
		}
		if roles && api {
			return true
		}
	}
	return false
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
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

// executable is the absolute path of this binary, which the plugin keeps off PATH, so
// the model can run the commands the report names.
func executable() string {
	if p, err := os.Executable(); err == nil {
		return p
	}
	return "tuilint"
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
