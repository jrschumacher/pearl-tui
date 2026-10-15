package codelint

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/tools/go/analysis/checker"
	"golang.org/x/tools/go/packages"
)

var update = flag.Bool("update", false, "regenerate testdata/golden from scripts/tuilint.py")

// want returns the `// want "regexp" ...` expectations of the Go files under dir, keyed by
// "file:line" relative to dir, as analysistest writes them.
func want(t *testing.T, dir string) map[string][]*regexp.Regexp {
	t.Helper()
	out := map[string][]*regexp.Regexp{}
	filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".go") {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		for i, ln := range strings.Split(string(b), "\n") {
			_, rest, ok := strings.Cut(ln, "// want ")
			for ok && strings.TrimSpace(rest) != "" {
				rest = strings.TrimSpace(rest)
				q, err := strconv.QuotedPrefix(rest)
				if err != nil {
					t.Fatalf("%s:%d: bad want: %v", rel, i+1, err)
				}
				s, _ := strconv.Unquote(q)
				k := fmt.Sprintf("%s:%d", filepath.ToSlash(rel), i+1)
				out[k] = append(out[k], regexp.MustCompile(s))
				rest = rest[len(q):]
			}
		}
		return nil
	})
	return out
}

// checkWant matches diagnostics ("file:line" → "ID: message") against the expectations.
func checkWant(t *testing.T, got map[string][]string, exp map[string][]*regexp.Regexp) {
	t.Helper()
	for at, msgs := range got {
		for _, m := range msgs {
			res := exp[at]
			i := 0
			for i < len(res) && !res[i].MatchString(m) {
				i++
			}
			if i == len(res) {
				t.Errorf("%s: unexpected diagnostic %q", at, m)
				continue
			}
			exp[at] = append(res[:i], res[i+1:]...)
		}
	}
	for at, res := range exp {
		for _, re := range res {
			t.Errorf("%s: no diagnostic matched %q", at, re)
		}
	}
}

var modules = []string{"bad", "good", "quiet", "broken", "huh", "huhsyntax"}

// syntaxOnly are the modules whose dependencies cannot resolve.
var syntaxOnly = map[string]bool{"broken": true, "huhsyntax": true}

// TestWant runs tuilint's own driver over each testdata module.
func TestWant(t *testing.T) {
	for _, m := range modules {
		t.Run(m, func(t *testing.T) {
			dir := filepath.Join("testdata", m)
			var stderr bytes.Buffer
			fs, err := Lint(dir, "", &stderr)
			if err != nil {
				t.Fatal(err)
			}
			fellBack := strings.Contains(stderr.String(), "falling back to syntax-only")
			if fellBack != syntaxOnly[m] {
				t.Errorf("fell back = %v, want %v (stderr: %s)", fellBack, syntaxOnly[m], stderr.String())
			}
			got := map[string][]string{}
			for _, f := range fs {
				k := f.File + ":" + f.Where
				got[k] = append(got[k], f.Rule+": "+f.Msg)
			}
			checkWant(t, got, want(t, dir))
		})
	}
}

// TestChecker runs the analyzers under the standard go/analysis driver, with facts encoded
// and decoded between packages as go vet does, and expects the same diagnostics.
func TestChecker(t *testing.T) {
	if testing.Short() {
		t.Skip("loads the standard library from source")
	}
	for _, m := range []string{"bad", "good", "quiet", "huh"} {
		t.Run(m, func(t *testing.T) {
			dir, _ := filepath.Abs(filepath.Join("testdata", m))
			pkgs, err := packages.Load(&packages.Config{Mode: packages.LoadAllSyntax | packages.NeedModule, Dir: dir}, "./...")
			if err != nil {
				t.Fatal(err)
			}
			if n := packages.PrintErrors(pkgs); n > 0 {
				t.Fatalf("%d load errors", n)
			}
			g, err := checker.Analyze(Analyzers, pkgs, &checker.Options{SanityCheck: true})
			if err != nil {
				t.Fatal(err)
			}
			got := map[string][]string{}
			for _, act := range g.Roots {
				if act.Err != nil {
					t.Fatal(act.Err)
				}
				for _, d := range act.Diagnostics {
					p := act.Package.Fset.Position(d.Pos)
					rel, _ := filepath.Rel(dir, p.Filename)
					k := fmt.Sprintf("%s:%d", filepath.ToSlash(rel), p.Line)
					got[k] = append(got[k], d.Message)
				}
			}
			checkWant(t, got, want(t, dir))
		})
	}
}

// TestThemePkg checks that --theme-pkg exempts a package from C-HEX.
func TestThemePkg(t *testing.T) {
	fs, err := Lint("testdata/bad", "brand", io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fs {
		if strings.HasPrefix(f.File, "brand/") {
			t.Errorf("finding in the theme package: %+v", f)
		}
	}
}

// TestMain checks the command line: output format, --json, exit codes.
func TestMainExit(t *testing.T) {
	out := filepath.Join(t.TempDir(), "out.json")
	var stdout, stderr bytes.Buffer
	if code := Main([]string{"testdata/quiet", "--json", out}, &stdout, &stderr); code != 1 {
		t.Errorf("exit %d, want 1 (C-SIZE is P1)\n%s%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "[P1] C-SIZE      main.go:31  no tea.WindowSizeMsg") {
		t.Errorf("unexpected output:\n%s", stdout.String())
	}
	var fs []Finding
	b, _ := os.ReadFile(out)
	if err := json.Unmarshal(b, &fs); err != nil || len(fs) != 3 {
		t.Errorf("json: %v, %d findings", err, len(fs))
	}
	stdout.Reset()
	if code := Main([]string{"testdata/good"}, &stdout, &stderr); code != 0 || !strings.HasPrefix(stdout.String(), "OK  0 blocking") {
		t.Errorf("good: exit %d\n%s", code, stdout.String())
	}
	if code := Main([]string{"--bogus", "x"}, &stdout, &stderr); code != 2 {
		t.Errorf("unknown flag: exit %d, want 2", code)
	}
}

// A deviation is a finding only one of tuilint.py and the Go port reports.
type deviation struct {
	input, rule, at string // at is "file:line", or "-" for tuilint.py's whole-root findings
	python          bool   // true: only tuilint.py reports it
	why             string
}

// deviations lists every difference from the tuilint.py goldens, and why the port differs.
var deviations = []deviation{
	// moved: the whole-root rules now point at the tea.NewProgram whose program lacks it
	{"v1-todo", "C-SIZE", "-", true, "reported at the tea.NewProgram call"},
	{"v1-todo", "C-SIZE", "main.go:68", false, "reported at the tea.NewProgram call"},
	{"v1-todo", "C-NOCOLOR", "-", true, "reported at the tea.NewProgram call"},
	{"v1-todo", "C-NOCOLOR", "main.go:68", false, "reported at the tea.NewProgram call"},
	{"bad", "C-SIZE", "-", true, "reported at the tea.NewProgram call"},
	{"bad", "C-SIZE", "main.go:135", false, "reported at the tea.NewProgram call"},
	{"bad", "C-NOCOLOR", "-", true, "reported at the tea.NewProgram call"},
	{"bad", "C-NOCOLOR", "main.go:135", false, "reported at the tea.NewProgram call"},
	{"broken", "C-SIZE", "-", true, "reported at the tea.NewProgram call"},
	{"broken", "C-SIZE", "main.go:58", false, "reported at the tea.NewProgram call"},
	{"broken", "C-NOCOLOR", "-", true, "reported at the tea.NewProgram call"},
	{"broken", "C-NOCOLOR", "main.go:58", false, "reported at the tea.NewProgram call"},

	// C-HEX: any literal passed to lipgloss.Color is a colour outside the theme
	{"v1-todo", "C-HEX", "main.go:16", false, `lipgloss.Color("62"): an ANSI literal is still a literal`},
	{"v1-todo", "C-HEX", "main.go:17", false, `lipgloss.Color("241"): an ANSI literal is still a literal`},
	{"bad", "C-HEX", "main.go:20", false, `lipgloss.Color("212"): an ANSI literal is still a literal`},
	{"bad", "C-HEX", "legacy/legacy.go:7", false, `lipgloss.Color("62"): an ANSI literal is still a literal`},
	{"bad", "C-HEX", "main.go:22", false, `tuilint.py strips from "//" to the end of the line, so the hex after "https://" went unseen`},

	// C-V1: only import declarations are imports
	{"bad", "C-V1", "main.go:23", true, "a string constant that mentions a v1 path is not an import"},

	// C-SLEEP / C-VIEWIO: reachability through calls, Cmds excluded, I/O in Update is C-SLEEP
	{"bad", "C-VIEWIO", "main.go:62", true, "http.Get in Update is C-SLEEP [P0], as the rule table says; tuilint.py filed all I/O as C-VIEWIO"},
	{"bad", "C-SLEEP", "main.go:62", false, "http.Get in Update is C-SLEEP [P0], as the rule table says; tuilint.py filed all I/O as C-VIEWIO"},
	{"bad", "C-SLEEP", "main.go:70", true, "time.Sleep inside a tea.Cmd runs off the update goroutine: that is the fix, not the bug"},
	{"good", "C-SLEEP", "app/app.go:54", true, "time.Sleep inside a tea.Tick callback runs off the update goroutine"},
	{"broken", "C-SLEEP", "main.go:38", true, "time.Sleep inside a tea.Cmd runs off the update goroutine (also without types)"},
	{"bad", "C-SLEEP", "main.go:80", false, "os.ReadFile in load, which Update calls: tuilint.py looked only inside Update itself"},
	{"bad", "C-VIEWIO", "main.go:91", false, "exec.Cmd.Output on a Cmd built on another line: types know the receiver"},
	{"bad", "C-VIEWIO", "main.go:92", true, "cache.ReadFile is an in-memory method, not os.ReadFile"},
	{"bad", "C-VIEWIO", "main.go:123", false, "View with an unnamed receiver: tuilint.py's regex needed a receiver name"},

	// C-LEN: types say what is text; context is the expression, not the line
	{"bad", "C-LEN", "main.go:97", false, "m.status is a string; tuilint.py took names ending in s for slices"},
	{"bad", "C-LEN", "main.go:98", false, "len([]rune(s)) counts runes, not cells; tuilint.py skipped []rune"},
	{"bad", "C-LEN", "main.go:99", false, "w := len(m.name) assigns a byte count to a width; the line has no width word"},
	{"bad", "C-LEN", "main.go:101", true, "m.row is a []string, not text"},
	{"bad", "C-LEN", "main.go:104", true, "len(m.name) > 3 is a length check; the Width on the line belongs to another operand"},
	{"broken", "C-LEN", "main.go:49", true, "m.list.Items() is a []string (local types are known even without dependencies)"},

	// C-EMOJI
	{"bad", "C-EMOJI", "main.go:109", false, "rune literal: tuilint.py needed a \" on the line"},
	{"bad", "C-EMOJI", "main.go:110", false, "raw string literal: tuilint.py needed a \" on the line"},
	{"bad", "C-EMOJI", "main.go:111", false, "U+23F3 is a wide emoji outside tuilint.py's ranges"},
	{"bad", "C-EMOJI", "main.go:112", false, "U+2764 U+FE0F: the variation selector asks for emoji presentation"},

	// C-FASTTICK: durations are constants, evaluated
	{"bad", "C-FASTTICK", "main.go:47", false, "time.Second/60 is 16.7ms; tuilint.py matched only N*time.Millisecond"},
	{"bad", "C-FASTTICK", "main.go:48", false, "a named constant of 10ms"},
	{"broken", "C-FASTTICK", "main.go:26", false, "time.Second/60, evaluated from syntax when types are missing"},

	// C-PRINT: what the running model reaches, not any method in a file that says tea.
	{"bad", "C-PRINT", "main.go:85", false, "fmt.Printf in logf, a plain function Update calls"},
	{"bad", "C-PRINT", "main.go:93", false, "fmt.Fprintln(os.Stderr) writes over the frame too"},
	{"bad", "C-PRINT", "main.go:130", true, "config.usage runs before the program starts; nothing in the model reaches it"},

	// program-wide rules: a comment is not handling
	{"quiet", "C-SIZE", "main.go:31", false, "tuilint.py found \"WindowSizeMsg\" in a comment"},
	{"quiet", "C-NOCOLOR", "main.go:31", false, "tuilint.py found \"NO_COLOR\" in a comment"},
	{"quiet", "C-KEYS", "main.go:19", false, "tuilint.py found \"key.Binding\" in a comment"},
}

type input struct{ name, dir, themePkg string }

var inputs = []input{
	{"kit", "../../../kit", ""},
	{"v1-todo", "../../../evals/fixtures/v1-todo", ""},
	{"bad", "testdata/bad", ""},
	{"bad-theme-pkg", "testdata/bad", "brand"},
	{"good", "testdata/good", ""},
	{"quiet", "testdata/quiet", ""},
	{"broken", "testdata/broken", ""},
}

// TestParity holds the port to the tuilint.py goldens: every difference must be listed in
// deviations, and every listed deviation must still occur.
func TestParity(t *testing.T) {
	if *update {
		regenerate(t)
	}
	for _, in := range inputs {
		t.Run(in.name, func(t *testing.T) {
			var py []Finding
			b, err := os.ReadFile(filepath.Join("testdata", "golden", in.name+".json"))
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(b, &py); err != nil {
				t.Fatal(err)
			}
			goFs, err := Lint(in.dir, in.themePkg, io.Discard)
			if err != nil {
				t.Fatal(err)
			}
			key := func(f Finding) string {
				if f.Where == "-" {
					return f.Rule + " -"
				}
				return f.Rule + " " + f.File + ":" + f.Where
			}
			pySet, goSet := map[string]bool{}, map[string]bool{}
			for _, f := range py {
				pySet[key(f)] = true
			}
			for _, f := range goFs {
				goSet[key(f)] = true
			}
			expect := map[string]bool{} // key → python-only
			devInput := strings.TrimSuffix(in.name, "-theme-pkg")
			for _, d := range deviations {
				if d.input == devInput && !(in.themePkg != "" && strings.HasPrefix(d.at, "brand/")) {
					expect[d.rule+" "+d.at] = d.python
				}
			}
			var diffs []string
			for k := range pySet {
				if !goSet[k] {
					if only, ok := expect[k]; !ok || !only {
						diffs = append(diffs, "only tuilint.py: "+k)
					}
					delete(expect, k)
				}
			}
			for k := range goSet {
				if !pySet[k] {
					if only, ok := expect[k]; !ok || only {
						diffs = append(diffs, "only Go: "+k)
					}
					delete(expect, k)
				}
			}
			for k := range expect {
				diffs = append(diffs, "listed deviation did not occur: "+k)
			}
			sort.Strings(diffs)
			for _, d := range diffs {
				t.Error(d)
			}
			t.Logf("%s: tuilint.py %d findings, Go %d", in.name, len(py), len(goFs))
		})
	}
}

// regenerate rewrites testdata/golden with tuilint.py's output for each input.
func regenerate(t *testing.T) {
	py, err := filepath.Abs("../../../scripts/tuilint.py")
	if err != nil {
		t.Fatal(err)
	}
	for _, in := range inputs {
		out, _ := filepath.Abs(filepath.Join("testdata", "golden", in.name+".json"))
		args := []string{py, "code", filepath.Base(in.dir), "--json", out}
		if in.themePkg != "" {
			args = append(args, "--theme-pkg", in.themePkg)
		}
		cmd := exec.Command("python3", args...)
		cmd.Dir = filepath.Dir(in.dir)
		if b, err := cmd.CombinedOutput(); err != nil && cmd.ProcessState.ExitCode() != 1 {
			t.Fatalf("%s: %v\n%s", in.name, err, b)
		}
	}
}
