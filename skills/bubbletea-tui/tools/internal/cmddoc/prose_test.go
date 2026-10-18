package cmddoc_test

import (
	"flag"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"tuitools/internal/cmddoc"
	"tuitools/internal/codelint"
)

// skill is the skill directory, from this package's.
const skill = "../../.."

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(skill, path))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// constString returns the raw-string constant NAME of a Go source: the text between
// "const NAME = `" and the next backtick.
func constString(t *testing.T, src, name string) string {
	t.Helper()
	_, rest, ok := strings.Cut(src, "const "+name+" = `")
	if !ok {
		t.Fatalf("no const %s", name)
	}
	body, _, _ := strings.Cut(rest, "`")
	return body
}

// specs builds each tool's Spec from the text its -h prints: the package doc of its
// command (cmddoc.Text), the usage constant of each tuilint mode, the analyzers tuivet
// registers, and the argparse block of each Python script. Nothing here is a second
// copy of a flag list: when a flag is renamed in a tool, this changes with it.
func specs(t *testing.T) map[string]cmddoc.Spec {
	t.Helper()
	// every tool and mode takes -h/--help, whether or not its doc spells it out
	free := func(doc string) map[string][]string {
		return cmddoc.Free(append(cmddoc.FlagsIn(doc), "-h", "--help"))
	}
	lint := cmddoc.Text(read(t, "tools/cmd/tuilint/main.go"))
	frames := constString(t, read(t, "tools/internal/framelint/framelint.go"), "usage")
	code := constString(t, read(t, "tools/internal/codelint/codelint.go"), "usage")
	spec := cmddoc.Text(read(t, "tools/internal/speclint/doc.go"))
	vet := map[string][]string{"-h": nil, "-help": nil, "--help": nil}
	for _, a := range codelint.Analyzers {
		vet["-"+a.Name] = nil
		a.Flags.VisitAll(func(f *flag.Flag) { vet["-"+a.Name+"."+f.Name] = nil })
	}
	return map[string]cmddoc.Spec{
		"cellshot":  {Flags: free(cmddoc.Text(read(t, "tools/cmd/cellshot/doc.go")))},
		"diversity": {Flags: free(cmddoc.Text(read(t, "tools/cmd/diversity/main.go")))},
		"palette": {
			Modes: []string{"derive", "check"},
			Flags: free(cmddoc.Text(read(t, "tools/cmd/palette/main.go"))),
			ModeFlags: map[string]map[string][]string{
				"derive": free(cmddoc.Text(read(t, "tools/cmd/palette/main.go"))),
				"check":  {"-h": nil, "--help": nil},
			},
		},
		"tuilint": {
			Modes: []string{"frames", "code", "spec", "hook", "help"},
			Flags: free(lint),
			ModeFlags: map[string]map[string][]string{
				"frames": free(frames), "code": free(code), "spec": free(spec), "hook": free(""), "help": free(""),
			},
		},
		"feedback": func() cmddoc.Spec {
			fb := free(cmddoc.Text(read(t, "tools/cmd/feedback/main.go")))
			fb["--kind"] = []string{"bug", "perf", "performance", "eval", "eval-suggestion", "feature", "enhancement"}
			return cmddoc.Spec{
				Modes:     []string{"draft", "similar", "submit"},
				Flags:     fb,
				ModeFlags: map[string]map[string][]string{"draft": fb, "submit": fb, "similar": {"--title": nil, "--repo": nil, "-h": nil, "--help": nil}},
			}
		}(),
		"tuivet":        {Flags: vet},
		"wordmark.py":   {Flags: cmddoc.ArgparseFlags(read(t, "scripts/wordmark.py"))},
		"logo2cells.py": {Flags: cmddoc.ArgparseFlags(read(t, "scripts/logo2cells.py"))},
	}
}

func tools(s map[string]cmddoc.Spec) []string {
	var out []string
	for name := range s {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// known lists invocations the prose gets wrong on purpose, or that another change in
// flight fixes: file, then substrings of the spans. Remove an entry when its line is
// fixed; empty means every invocation must pass.
var known = map[string][]string{}

// TestProseInvocations holds every tool command written in SKILL.md, references/, the
// README and AGENTS.md to the tools as they are: a mode word the tool has, flags its -h
// documents, and choices (--style, --mode) the script accepts. An agent copies these
// lines verbatim, so a renamed flag that survives in prose fails on its first use.
func TestProseInvocations(t *testing.T) {
	s := specs(t)
	names := tools(s)
	files := []string{"SKILL.md", "../../README.md", "../../AGENTS.md"}
	refs, err := filepath.Glob(filepath.Join(skill, "references", "*.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range refs {
		files = append(files, filepath.Join("references", filepath.Base(r)))
	}
	seen := 0
	for _, f := range files {
		for _, inv := range cmddoc.Invocations(f, read(t, f), names) {
			seen++
			for _, p := range s[inv.Tool].Check(inv) {
				if allowed(f, inv.Text) {
					t.Logf("%s:%d: %s (known: %s)", f, inv.Line, p, strings.TrimSpace(inv.Text))
					continue
				}
				t.Errorf("%s:%d: %s %s: %s", f, inv.Line, inv.Tool, p, strings.TrimSpace(inv.Text))
			}
		}
	}
	t.Logf("%d tool invocations checked", seen)
	if seen < 40 {
		t.Errorf("found %d tool invocations in the prose; the extractor used to find more", seen)
	}
}

// TestSpecsAreReal checks the specs hold what the tools actually take, so a passing
// prose test means something: the flags each tool parses are documented, and so found.
func TestSpecsAreReal(t *testing.T) {
	s := specs(t)
	want := map[string][]string{
		"cellshot":  {"--sheet", "--gif", "--ms", "--cell", "--bg", "--light", "--fg", "--cols", "--rows", "--grid", "--scale", "--label"},
		"diversity": {"--max-shared", "--max-popular", "--baseline", "--min-distance"},
		"palette":   {"--accent", "--bg", "--light-bg", "--tint", "--name", "--out", "--swatch"},
		"tuivet":    {"-tuihex", "-tuihex.theme-pkg", "-tuiblocking", "-tuihuhtheme"},
	}
	for tool, flags := range want {
		for _, f := range flags {
			if _, ok := s[tool].Flags[f]; !ok {
				t.Errorf("%s: %s is not in its documented flags %v", tool, f, keys(s[tool].Flags))
			}
		}
	}
	modes := map[string][]string{"frames": {"--cols", "--rows", "--theme", "--bg", "--ascii", "--json"}, "code": {"--theme-pkg", "--json"}, "spec": {"--json"}}
	for mode, flags := range modes {
		for _, f := range flags {
			if _, ok := s["tuilint"].ModeFlags[mode][f]; !ok {
				t.Errorf("tuilint %s: %s is not in its documented flags", mode, f)
			}
		}
	}
	if got := s["wordmark.py"].Flags["--style"]; !reflect.DeepEqual(got, []string{"half", "shade", "ascii", "tracked"}) {
		t.Errorf("wordmark.py --style choices = %v", got)
	}
	if got := s["logo2cells.py"].Flags["--mode"]; !reflect.DeepEqual(got, []string{"half", "shade", "quad"}) {
		t.Errorf("logo2cells.py --mode choices = %v", got)
	}
}

// TestCheckCatches plants the mistakes the prose test exists for.
func TestCheckCatches(t *testing.T) {
	s := specs(t)
	cases := []struct {
		text string
		want string // "" = passes
	}{
		{"`tuilint frames frames --theme t.json --bg #000`", ""},
		{"`tuilint frames frames --themes t.json`", "unknown flag --themes"},
		{"`tuilint lint .`", "unknown mode lint"},
		{"`palette derive \"#000\" --bg \"#111\" --out r.json`", ""},
		{"`palette derive \"#000\" --background \"#111\"`", "unknown flag --background"},
		{"`palette check theme.json`", ""},
		{"`scripts/wordmark.py \"name\" --style blocks|shade --rows N`", "--style blocks is not one of half|shade|ascii|tracked"},
		{"`scripts/wordmark.py \"name\" --style shade|half|ascii`", ""},
		{"`scripts/logo2cells.py logo.png --cols 40 --mode half|shade|quad --colors 3`", ""},
		{"`logo2cells.py logo.png --mode quads`", "--mode quads is not one of half|shade|quad"},
		{"`cellshot name.ansi name.png --grid --cell 16x32`", ""},
		{"`cellshot f*.ansi --gif out.gif --fps 20`", "unknown flag --fps"},
		{"`diversity themes/*.json --baseline ink,charm --min-distance 8`", ""},
		{"`tuivet -tuihex.theme-pkg=brand ./...`", ""},
		{"`tuivet -tuicolour ./...`", "unknown flag -tuicolour"},
		{"```\n3. LINT     tuilint frames frames --theme t.json\n            tuilint code --json out.json .\n```", ""},
		{"```\n4. LOOK     cellshot a.ansi --sheet s.png --cells 8x16\n```", "unknown flag --cells"},
		{"the value (`--style tracked`) when the mark is the hero", ""},                       // no tool named: not an invocation
		{"```" + "\n│   ├── tuilint   slop + correctness lint\n" + "```", ""},                 // a tree listing, not a command
		{"```" + "\npalette   <the hues, measured>\npalette / tone / geometry\n" + "```", ""}, // a template, not a command
		{"`tuivet ./...` and `tuilint code .`", ""},
		{"`python3 scripts/wordmark.py \"name\" --style blocks --rows 4`", "--style blocks is not one of half|shade|ascii|tracked"},
		{"`python3 -I scripts/logo2cells.py logo.png --mode quads`", "--mode quads is not one of half|shade|quad"},
		{"```" + "\npalette warm, tinted neutrals and one accent\n" + "```", ""},            // fenced prose opening with a tool's name
		{"```" + "\npalette warm --tint 0.02 themes/x.json\n" + "```", "unknown mode warm"}, // a flag makes it a command
		{"`palette tune themes/x.json`", "unknown mode tune"},                               // so does a path
	}
	for _, c := range cases {
		invs := cmddoc.Invocations("x.md", c.text, tools(s))
		var got []string
		for _, inv := range invs {
			got = append(got, s[inv.Tool].Check(inv)...)
		}
		switch {
		case c.want == "" && len(got) > 0:
			t.Errorf("%q: unexpected %v", c.text, got)
		case c.want != "" && !strings.Contains(strings.Join(got, "; "), c.want):
			t.Errorf("%q: got %v, want %q", c.text, got, c.want)
		}
	}
}

func TestInvocationsLines(t *testing.T) {
	text := "intro\n\n`tuilint code .` and\n`scripts/wordmark.py \"a b\" --style half\n   --rows 4` then\n```\ngo run ./cmd/app\ncellshot a.ansi --grid\n```\n"
	got := cmddoc.Invocations("f.md", text, []string{"tuilint", "wordmark.py", "cellshot"})
	var lines []string
	for _, inv := range got {
		lines = append(lines, inv.Tool+"@"+strings.Join(inv.Args, ","))
	}
	want := []string{"tuilint@code,.", "wordmark.py@\"a,b\",--style,half,--rows,4", "cellshot@a.ansi,--grid"}
	if !reflect.DeepEqual(lines, want) {
		t.Errorf("got %v, want %v", lines, want)
	}
	if got[2].Line != 8 {
		t.Errorf("fenced line = %d, want 8", got[2].Line)
	}
}

// allowed reports a span that known lists for its file.
func allowed(file, text string) bool {
	for _, sub := range known[file] {
		if strings.Contains(text, sub) {
			return true
		}
	}
	return false
}

func keys(m map[string][]string) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
