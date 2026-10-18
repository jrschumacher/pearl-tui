// Package codelint lints Bubble Tea code: Go source built on Bubble Tea v2, Lip Gloss,
// Bubbles and Huh, the one framework the kit supports today, for the mistakes that make
// frames wrong. Each rule is a go/analysis analyzer, so the same checks run from
// `tuilint code`, from `go vet -vettool=$(which tuivet)` and in editors. The theme
// package is one whose path ends in theme (a project's internal/theme) or in
// theme/<framework> (the kit's theme/bubbletea).
//
// Rules, with their severity (P0/P1 fail the run, P2 warn):
//
//	C-LEN       [P1] len()/utf8.RuneCountInString of text used to size or pad: use lipgloss.Width
//	C-SLEEP     [P0] time.Sleep or blocking I/O reachable from Update (or Sleep from View): return a tea.Cmd
//	C-HEX       [P1] lipgloss.Color literal or "#rrggbb" outside the theme package
//	C-EMOJI     [P1] emoji in string or rune literals
//	C-GLYPH     [P1] a drawn mark in a string or rune literal outside the theme package and the
//	                 ASCII twin table (a composite literal initializing a package-level var named
//	                 *ascii*, the kit's ui/bubbletea/canvas.go): arrows, box drawing, blocks,
//	                 geometric shapes, misc symbols and technical, dingbats, braille, the
//	                 marks · • …, and any rune the kit's twin table maps. Screens take those
//	                 from the theme, so a brand can change them and ASCII gets a twin. Text is
//	                 not a glyph: letters and digits of any script, quotation marks and dashes
//	                 (it’s, «q», a — b, a range's 1–9), currency (£5) and units, marks and
//	                 maths (72°F © ® ™ × ÷ ± ≥ ≤ ≠ ¹ ² ³ ½ µ, 80×24); the canvas's ASCII sweep
//	                 maps × and the dashes itself
//	C-ELLIPSIS  [P1] the bare theme.Ellipsis outside the ui package: cut with ui.Trunc(t, s, w), which
//	                 fits the theme's glyph t.G.Ellipsis and its ASCII twin inside w
//	C-V1        [P1] Bubble Tea / Lip Gloss / Bubbles / Huh v1 import paths
//	C-SIZE      [P1] a program whose code never handles tea.WindowSizeMsg
//	C-VIEWIO    [P1] os, net/http or os/exec I/O reachable from View
//	C-HUHTHEME  [P1] a huh.NewForm (or a Huh field run on its own) never given WithTheme, or given a
//	                 stock Huh theme; a Huh spinner without WithTheme; a Bubbles spinner.New without
//	                 spinner.WithStyle or a .Style: each renders the library's colours, not the brand
//	C-FASTTICK  [P2] tea.Tick/Every faster than 33ms
//	C-PRINT     [P2] fmt.Print* reachable from a running model (Init, Update, View or their Cmds)
//	C-KEYS      [P2] key strings matched inline and no key.Binding anywhere in the program
//	C-NOCOLOR   [P2] a program with no colour-profile / dark-background / NO_COLOR handling
package codelint

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// Analyzers are the rules, in the order tuivet registers them.
var Analyzers = []*analysis.Analyzer{
	LenAnalyzer, BlockingAnalyzer, HexAnalyzer, EmojiAnalyzer, GlyphAnalyzer, EllipsisAnalyzer, V1Analyzer, SizeAnalyzer,
	TickAnalyzer, PrintAnalyzer, KeysAnalyzer, NoColorAnalyzer, HuhThemeAnalyzer,
}

// Severity maps each rule ID to P0, P1 or P2.
var Severity = map[string]string{
	"C-SLEEP": "P0",
	"C-LEN":   "P1", "C-HEX": "P1", "C-EMOJI": "P1", "C-GLYPH": "P1", "C-ELLIPSIS": "P1", "C-V1": "P1", "C-SIZE": "P1", "C-VIEWIO": "P1", "C-HUHTHEME": "P1",
	"C-FASTTICK": "P2", "C-PRINT": "P2", "C-KEYS": "P2", "C-NOCOLOR": "P2",
}

// Finding is one result, in the shape tuilint.py wrote to --json.
type Finding struct {
	Rule  string `json:"rule"`
	Sev   string `json:"sev"`
	File  string `json:"file"`
	Where string `json:"where"`
	Msg   string `json:"msg"`
}

const usage = `usage: tuilint code <dir...> [--theme-pkg path/fragment] [--json out.json]

Lints Bubble Tea code (Go on Bubble Tea v2, Lip Gloss, Bubbles and Huh, the one
framework the kit supports today) for the mistakes that make frames wrong. The theme
package is the one whose path ends in theme or theme/<framework>. A finding prints as
[SEV] ID  file:line  message. P0/P1 findings fail the run (exit 1); P2 are
warnings to fix or justify in one line. Exit 2 on usage errors or a directory
that won't load. Test files (_test.go) are not linted.
Packages load with full types; if that fails (unresolvable deps, a v1 app) the
files are analysed syntax-only and stderr says so.

  C-LEN       [P1] len()/utf8.RuneCountInString of text used to size or pad: use lipgloss.Width
  C-SLEEP     [P0] time.Sleep or blocking I/O reachable from Update (or Sleep from View)
  C-HEX       [P1] lipgloss.Color literal or "#rrggbb" outside the theme package
  C-EMOJI     [P1] emoji in string or rune literals
  C-GLYPH     [P1] a drawn mark (arrows, box drawing, blocks, shapes, symbols, dingbats,
                   braille, · • …, or any rune the kit's twin table maps) in a string or rune
                   literal outside the theme package and the ASCII twin table (a composite
                   literal of a package-level var named *ascii*): screens take glyphs from
                   the theme. Text is not a glyph: other scripts, quotes and dashes (it’s,
                   «q», a — b, 1–9), currency (£5), units and maths (72°F © ™ × ± ≥ ½ µ,
                   80×24); the ASCII sweep maps × and the dashes itself
  C-ELLIPSIS  [P1] the bare theme.Ellipsis outside the ui package: cut with
                   ui.Trunc(t, s, w), which fits the theme's glyph and its twin inside w
  C-V1        [P1] Bubble Tea / Lip Gloss / Bubbles / Huh v1 import paths
  C-SIZE      [P1] a program whose code never handles tea.WindowSizeMsg
  C-VIEWIO    [P1] os, net/http or os/exec I/O reachable from View
  C-HUHTHEME  [P1] huh form or run field without WithTheme (or with a stock Huh theme), Huh
                   spinner without WithTheme, Bubbles spinner without WithStyle or .Style
  C-FASTTICK  [P2] tea.Tick/Every faster than 33ms
  C-PRINT     [P2] fmt.Print* reachable from a running model
  C-KEYS      [P2] key strings matched inline and no key.Binding anywhere in the program
  C-NOCOLOR   [P2] no colour-profile / dark-background / NO_COLOR handling in the program
`

// Main runs the mode with its arguments and returns the exit status.
func Main(args []string, stdout, stderr io.Writer) int {
	opts := map[string]string{}
	var roots []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-h" || a == "--help":
			fmt.Fprint(stdout, usage)
			return 0
		case strings.HasPrefix(a, "-"):
			name, val, ok := strings.Cut(strings.TrimLeft(a, "-"), "=")
			if name != "theme-pkg" && name != "json" {
				fmt.Fprintf(stderr, "tuilint code: unknown flag %s\n%s", a, usage)
				return 2
			}
			if !ok {
				if i+1 >= len(args) {
					fmt.Fprintf(stderr, "tuilint code: %s needs a value\n", a)
					return 2
				}
				i++
				val = args[i]
			}
			opts[name] = val
		default:
			roots = append(roots, a)
		}
	}
	if len(roots) == 0 {
		roots = []string{"."}
	}
	var findings []Finding
	for _, root := range roots {
		fs, err := Lint(root, opts["theme-pkg"], stderr)
		if err != nil {
			fmt.Fprintf(stderr, "tuilint code: %s: %v\n", root, err)
			return 2
		}
		findings = append(findings, fs...)
	}
	Sort(findings)
	fails := 0
	for _, f := range findings {
		fmt.Fprintf(stdout, "[%s] %-11s %s:%s  %s\n", f.Sev, f.Rule, f.File, f.Where, f.Msg)
		if f.Sev != "P2" {
			fails++
		}
	}
	verdict := "OK"
	if fails > 0 {
		verdict = "FAIL"
	}
	fmt.Fprintf(stdout, "%s  %d blocking, %d warnings\n", verdict, fails, len(findings)-fails)
	if path := opts["json"]; path != "" {
		if findings == nil {
			findings = []Finding{}
		}
		b, _ := json.MarshalIndent(findings, "", " ")
		if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
			fmt.Fprintf(stderr, "tuilint code: %v\n", err)
			return 2
		}
	}
	if fails > 0 {
		return 1
	}
	return 0
}

// Lint runs every analyzer over the packages under root. File paths in the findings are
// relative to root. A note goes to stderr when it falls back to syntax-only analysis.
func Lint(root, themePkg string, stderr io.Writer) ([]Finding, error) {
	return LintWith(root, themePkg, stderr, Load{})
}

// Load is how LintWith runs the go command to load packages with types. The zero value
// inherits the process environment and adds no flags.
type Load struct {
	Env        []string // the go command's environment; nil means os.Environ()
	BuildFlags []string // e.g. -mod=readonly, which overrides a -mod in GOFLAGS
}

// LintWith is Lint with control over how the go command loads the packages, for a
// caller such as the edit hook that must stay offline and never write go.mod or go.sum.
func LintWith(root, themePkg string, stderr io.Writer, load Load) ([]Finding, error) {
	if err := HexAnalyzer.Flags.Set("theme-pkg", themePkg); err != nil {
		return nil, err
	}
	defer HexAnalyzer.Flags.Set("theme-pkg", "")
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	units, fset, err := loadTyped(abs, load)
	if err != nil {
		fmt.Fprintf(stderr, "tuilint code: %s: loading with types failed (%v); falling back to syntax-only analysis\n", root, firstLine(err))
		units, fset, err = loadSyntax(abs)
		if err != nil {
			return nil, err
		}
	}
	diags, err := run(Analyzers, units, fset)
	if err != nil {
		return nil, err
	}
	var out []Finding
	for _, d := range diags {
		p := fset.Position(d.Pos)
		rel, err := filepath.Rel(abs, p.Filename)
		if err != nil {
			rel = p.Filename
		}
		rule, msg := d.Category, strings.TrimPrefix(d.Message, d.Category+": ")
		out = append(out, Finding{Rule: rule, Sev: Severity[rule], File: filepath.ToSlash(rel), Where: strconv.Itoa(p.Line), Msg: msg})
	}
	return out, nil
}

// Sort orders findings as tuilint.py printed them: severity, rule, file, then line.
func Sort(fs []Finding) {
	sort.SliceStable(fs, func(i, j int) bool {
		a, b := fs[i], fs[j]
		if a.Sev != b.Sev {
			return a.Sev < b.Sev
		}
		if a.Rule != b.Rule {
			return a.Rule < b.Rule
		}
		if a.File != b.File {
			return a.File < b.File
		}
		la, _ := strconv.Atoi(a.Where)
		lb, _ := strconv.Atoi(b.Where)
		if la != lb {
			return la < lb
		}
		return a.Msg < b.Msg
	})
}

func firstLine(err error) string {
	s, _, _ := strings.Cut(err.Error(), "\n")
	return s
}
