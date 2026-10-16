// Package codelint lints Go source for the mistakes that make frames wrong. Each rule is a
// go/analysis analyzer, so the same checks run from `tuilint code`, from
// `go vet -vettool=$(which tuivet)` and in editors.
//
// Rules, with their severity (P0/P1 fail the run, P2 warn):
//
//	C-LEN       [P1] len()/utf8.RuneCountInString of text used to size or pad: use lipgloss.Width
//	C-SLEEP     [P0] time.Sleep or blocking I/O reachable from Update (or Sleep from View): return a tea.Cmd
//	C-HEX       [P1] lipgloss.Color literal or "#rrggbb" outside the theme package
//	C-EMOJI     [P1] emoji in string or rune literals
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
	LenAnalyzer, BlockingAnalyzer, HexAnalyzer, EmojiAnalyzer, V1Analyzer, SizeAnalyzer,
	TickAnalyzer, PrintAnalyzer, KeysAnalyzer, NoColorAnalyzer, HuhThemeAnalyzer,
}

// Severity maps each rule ID to P0, P1 or P2.
var Severity = map[string]string{
	"C-SLEEP": "P0",
	"C-LEN":   "P1", "C-HEX": "P1", "C-EMOJI": "P1", "C-V1": "P1", "C-SIZE": "P1", "C-VIEWIO": "P1", "C-HUHTHEME": "P1",
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

Lints Go source for the mistakes that make frames wrong. A finding prints as
ID  file:line  message. P0/P1 findings fail the run (exit 1); P2 are warnings.
Packages load with full types; if that fails (unresolvable deps, a v1 app) the
files are analysed syntax-only and stderr says so.

  C-LEN       [P1] len()/utf8.RuneCountInString of text used to size or pad: use lipgloss.Width
  C-SLEEP     [P0] time.Sleep or blocking I/O reachable from Update (or Sleep from View)
  C-HEX       [P1] lipgloss.Color literal or "#rrggbb" outside the theme package
  C-EMOJI     [P1] emoji in string or rune literals
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
