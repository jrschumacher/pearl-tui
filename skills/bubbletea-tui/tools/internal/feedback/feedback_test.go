package feedback

import (
	"bytes"
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"runtime/debug"
	"strings"
	"testing"

	"tuitools/internal/cmddoc"
)

var update = flag.Bool("update", false, "rewrite the golden files")

// ---- the forms and the table agree ------------------------------------------------

// form is what the test reads from an issue form: enough of its YAML to compare.
type form struct {
	labels []string
	fields []Field
}

func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && (s[0] == '"' && s[len(s)-1] == '"' || s[0] == '\'' && s[len(s)-1] == '\'') {
		return s[1 : len(s)-1]
	}
	return s
}

// readForm parses the subset of issue-form YAML the templates use, line by line (the
// tools take no YAML dependency): labels, and each body item's type, id, label, render,
// options and required.
func readForm(t *testing.T, path string) form {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var f form
	var cur *Field
	inOptions := false
	for _, line := range strings.Split(string(b), "\n") {
		trim := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "labels:"):
			v := strings.Trim(strings.TrimSpace(strings.TrimPrefix(line, "labels:")), "[]")
			for _, l := range strings.Split(v, ",") {
				f.labels = append(f.labels, unquote(l))
			}
		case strings.HasPrefix(trim, "- type:"):
			inOptions = false
			typ := unquote(strings.TrimPrefix(trim, "- type:"))
			if typ == "markdown" {
				cur = nil
				continue
			}
			f.fields = append(f.fields, Field{Type: typ})
			cur = &f.fields[len(f.fields)-1]
		case cur == nil:
		case strings.HasPrefix(trim, "id:"):
			cur.ID = unquote(strings.TrimPrefix(trim, "id:"))
		case strings.HasPrefix(trim, "label:"):
			cur.Label = unquote(strings.TrimPrefix(trim, "label:"))
		case strings.HasPrefix(trim, "render:"):
			cur.Render = unquote(strings.TrimPrefix(trim, "render:"))
		case strings.HasPrefix(trim, "required:"):
			cur.Required = unquote(strings.TrimPrefix(trim, "required:")) == "true"
		case trim == "options:":
			inOptions = true
		case inOptions && strings.HasPrefix(trim, "- "):
			cur.Options = append(cur.Options, unquote(strings.TrimPrefix(trim, "- ")))
		case strings.HasSuffix(trim, ":"):
			inOptions = false
		}
	}
	return f
}

// The issue forms are what a person sees; Kinds is what the tool writes. An issue filed
// through gh must read exactly like one filed from the form, and the link's query keys
// must be the form's ids, so the two can't drift.
func TestKindsMatchTheIssueForms(t *testing.T) {
	dir := filepath.Join("..", "..", "..", "..", "..", ".github", "ISSUE_TEMPLATE")
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("no issue forms at %s: a moved path would leave Kinds unchecked (%v)", dir, err)
	}
	seen := map[string]bool{}
	for _, k := range Kinds {
		seen[k.Template] = true
		f := readForm(t, filepath.Join(dir, k.Template))
		if !reflect.DeepEqual(f.labels, k.Labels) {
			t.Errorf("%s labels = %v, Kinds says %v", k.Template, f.labels, k.Labels)
		}
		var want []Field
		for _, kf := range k.Fields {
			kf.Auto = false // the form doesn't know what the tool fills in
			want = append(want, kf)
		}
		if !reflect.DeepEqual(f.fields, want) {
			t.Errorf("%s fields differ from Kinds[%s]:\n form  %+v\n kinds %+v", k.Template, k.Name, f.fields, want)
		}
	}
	ymls, _ := filepath.Glob(filepath.Join(dir, "*.yml"))
	for _, y := range ymls {
		if b := filepath.Base(y); b != "config.yml" && !seen[b] {
			t.Errorf("%s is a form with no kind in Kinds: feedback can't file it", b)
		}
	}
}

func TestKindNamed(t *testing.T) {
	for name, want := range map[string]string{
		"bug": "bug.yml", "perf": "performance.yml", "performance": "performance.yml",
		"eval": "eval-suggestion.yml", "eval-suggestion": "eval-suggestion.yml",
		"feature": "feature.yml", "enhancement": "feature.yml",
	} {
		k, err := KindNamed(name)
		if err != nil || k.Template != want {
			t.Errorf("KindNamed(%q) = %s, %v; want %s", name, k.Template, err, want)
		}
		if k.Labels[0] != "feedback" {
			t.Errorf("%s: first label %q, want feedback (the loop triages on it)", name, k.Labels[0])
		}
	}
	if _, err := KindNamed("question"); err == nil || !strings.Contains(err.Error(), "bug|perf|eval|feature") {
		t.Errorf("KindNamed(question) err = %v, want the kinds named", err)
	}
}

func TestChoose(t *testing.T) {
	k, _ := KindNamed("bug")
	mc, _ := k.Field("miss-class")
	for in, want := range map[string]string{
		"b":   "(b) the skill says it and the agent ignored it",
		"(a)": "(a) the skill never says it",
		"C":   "(c) prose can't express it",
		"not": "not sure",
		"(b) the skill says it and the agent ignored it": "(b) the skill says it and the agent ignored it",
	} {
		if got, err := mc.Choose(in); err != nil || got != want {
			t.Errorf("Choose(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	if _, err := mc.Choose("d"); err == nil || !strings.Contains(err.Error(), `"not sure"`) {
		t.Errorf("Choose(d) err = %v, want the options listed", err)
	}
	f, _ := KindNamed("feature")
	area, _ := f.Field("area")
	if got, _ := area.Choose("kit"); got != "kit (theme, ui, showcase)" {
		t.Errorf("feature area kit = %q", got)
	}
	if _, err := area.Choose("t"); err == nil {
		t.Error("feature area t: ambiguous (themes, tools) must fail")
	}
}

// ---- body, link and search ---------------------------------------------------------

func perfIssue() Issue {
	k, _ := KindNamed("perf")
	return Issue{Repo: DefaultRepo, Kind: k, Title: "cellshot --sheet takes 9 s on 40 frames", Values: map[string]string{
		"asked":   "/bubbletea-tui craft",
		"area":    "cellshot --sheet",
		"command": "cellshot frames/*.ansi --sheet sheet.png",
		"timing":  "9.1 s, 3 runs",
		"sizes":   "40 frames, 120x40",
	}}
}

func TestBody(t *testing.T) {
	is := perfIssue()
	body := is.Body()
	if !strings.Contains(body, "### Command\n\n```shell\ncellshot frames/*.ansi --sheet sheet.png\n```\n") {
		t.Errorf("a render: shell field is fenced as GitHub renders it:\n%s", body)
	}
	if !strings.Contains(body, "### Profile or repro\n\n_No response_\n") {
		t.Errorf("an empty field reads _No response_:\n%s", body)
	}
}

// The hash covers what submit sends: a changed field, title, label set or target is a
// different issue, so a --confirm for the old draft can't file it.
func TestHash(t *testing.T) {
	a := perfIssue()
	h := a.Hash()
	if len(h) != 8 || h != perfIssue().Hash() {
		t.Fatalf("Hash = %q, not 8 stable hex digits", h)
	}
	for name, change := range map[string]func(*Issue){
		"body":    func(is *Issue) { is.Values["timing"] = "9.2 s, 3 runs" },
		"title":   func(is *Issue) { is.Title += "!" },
		"repo":    func(is *Issue) { is.Repo = "me/fork" },
		"comment": func(is *Issue) { is.Comment = 3 },
	} {
		b := perfIssue()
		change(&b)
		if b.Hash() == h {
			t.Errorf("changing the %s kept the hash %s", name, h)
		}
	}
	if !strings.Contains(a.Draft(), "\nconfirm:  "+h+"\n\n### What was asked") {
		t.Errorf("the draft doesn't print the hash above the body:\n%s", a.Draft())
	}
}

func TestNewIssueURL(t *testing.T) {
	k, _ := KindNamed("bug")
	is := Issue{Repo: "me/fork", Kind: k, Title: "F-CONTRAST & the scrim: 2.1:1 #fail", Values: map[string]string{
		"asked":      "craft a list\nthen a dialog",
		"area":       "tuilint frames",
		"miss-class": "(c) prose can't express it",
		"happened":   "ü → ▌ 100%",
	}}
	link, skipped := is.NewIssueURL(URLLimit)
	if !strings.HasPrefix(link, "https://github.com/me/fork/issues/new?template=bug.yml&title=") {
		t.Errorf("link = %s", link)
	}
	u, err := url.Parse(link)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	for key, want := range map[string]string{
		"template": "bug.yml", "title": is.Title, "labels": "feedback,bug",
		"asked": "craft a list\nthen a dialog", "area": "tuilint frames", "happened": "ü → ▌ 100%",
	} {
		if q.Get(key) != want {
			t.Errorf("query %s = %q, want %q", key, q.Get(key), want)
		}
	}
	if q.Has("miss-class") || !reflect.DeepEqual(skipped, []string{"miss-class"}) {
		t.Errorf("a dropdown can't be prefilled: query has it %v, skipped %v", q.Has("miss-class"), skipped)
	}
	if strings.ContainsAny(link[strings.Index(link, "?"):], " \n#") {
		t.Errorf("unencoded character in %s", link)
	}
}

func TestNewIssueURLLengthFallback(t *testing.T) {
	is := perfIssue()
	is.Values["evidence"] = strings.Repeat("flat profile line\n", 600) // ~10 KB
	is.Values["environment"] = "go version go1.26.7 linux/amd64"
	link, skipped := is.NewIssueURL(URLLimit)
	if len(link) > URLLimit {
		t.Errorf("link is %d bytes, over %d", len(link), URLLimit)
	}
	if !reflect.DeepEqual(skipped, []string{"evidence"}) {
		t.Errorf("skipped = %v, want [evidence]", skipped)
	}
	q, _ := url.ParseQuery(link[strings.Index(link, "?")+1:])
	if q.Get("environment") == "" || q.Get("timing") == "" {
		t.Error("fields after the one that didn't fit still go in the link")
	}
}

func TestTerms(t *testing.T) {
	got := Terms("fix: tuilint F-CONTRAST fires on the ruled dialog's scrim under -profile ansi, at 80x24 too")
	// nine words; the four longest stay, in title order
	want := []string{"tuilint", "f-contrast", "dialog", "profile"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Terms = %v, want %v", got, want)
	}
	if s := SearchURL(DefaultRepo, []string{"dialog", "scrim"}); s != "https://github.com/jrschumacher/skill-bubbletea-tui/issues?q=is%3Aissue+dialog+scrim" {
		t.Errorf("SearchURL = %s", s)
	}
}

func TestRedactor(t *testing.T) {
	t.Setenv("PATH", "") // no git: the root is found by its markers
	home := t.TempDir()
	root := filepath.Join(home, "src", "acme-ops")
	os.MkdirAll(filepath.Join(root, "internal", "ui"), 0o755)
	e := Env{Cwd: root, Home: home, User: "ana"}
	r := NewRedactor(e, []string{"AcmeOps"})
	got := r.Apply("built " + root + "/cmd/acmeops; cache " + home + "/.cache; ana ran AcmeOps in /Users/ana/x, banana " + root + "2")
	want := "built <project>/cmd/<redacted>; cache ~/.cache; ana ran <redacted> in /Users/<user>/x, banana ~/src/acme-ops2"
	if got != want {
		t.Errorf("Apply =\n %s\nwant\n %s", got, want)
	}
	if rep := r.Report(); rep != "redacted 1 × <project>, 2 × ~, 1 × <user>, 2 × <redacted>" {
		t.Errorf("Report = %q", rep)
	}
}

// Go's \b is ASCII-only: a name that starts or ends in é, ë or "." never matched it, and
// leaked with no warning. A --redact word matches in any case and script, and inside a
// longer word too (an identifier like "ZoëModel" names the app as much as "Zoë" does).
func TestRedactorAnyScript(t *testing.T) {
	t.Setenv("PATH", "")
	words := []string{"Café", "Zoë", "Acme Inc.", "Ötzi", "Nowhere"}
	r := NewRedactor(Env{Cwd: t.TempDir()}, words)
	got := r.Apply("Café Zoë CAFÉ ZOË ships for Acme Inc. and ötzi; ZoëModel, cafés")
	want := "<redacted> <redacted> <redacted> <redacted> ships for <redacted> and <redacted>; <redacted>Model, <redacted>s"
	if got != want {
		t.Errorf("Apply =\n %s\nwant\n %s", got, want)
	}
	if u := r.Unmatched(words); !reflect.DeepEqual(u, []string{"Nowhere"}) {
		t.Errorf("Unmatched = %v, want [Nowhere]", u)
	}
	// ZoëModel is two words to an identifier: no note; cafés is one
	if in := r.Inside(); !reflect.DeepEqual(in, []string{`"Café" matched inside "cafés"`}) {
		t.Errorf("Inside = %q", in)
	}
	// a word inside an earlier placeholder is not a match: no <<redacted>>
	r = NewRedactor(Env{Cwd: t.TempDir()}, []string{"red", "project"})
	if got := r.Apply("<project> and <redacted> and a red project"); got != "<project> and <redacted> and a <redacted> <redacted>" {
		t.Errorf("Apply over placeholders = %q", got)
	}
}

// A word under four letters matches only on its own, or as an identifier's part: "ui"
// is redacted as a path segment and in "uiModel", never inside "build" or "guidance".
// A longer word still matches inside a word, and says so.
func TestRedactorShortWords(t *testing.T) {
	t.Setenv("PATH", "")
	const text = "F-CONTRAST in kit/ui/dialog.go: tuilint frames, the build screen, uiModel and UIList; bubbletea-tui guidance"
	for _, c := range []struct {
		word, want string
		inside     []string
	}{
		{"ui", "F-CONTRAST in kit/<redacted>/dialog.go: tuilint frames, the build screen, <redacted>Model and <redacted>List; bubbletea-tui guidance", nil},
		{"tui", "F-CONTRAST in kit/ui/dialog.go: tuilint frames, the build screen, uiModel and UIList; bubbletea-<redacted> guidance", nil},
		{"a", text, nil},
		{"Lint", "F-CONTRAST in kit/ui/dialog.go: tui<redacted> frames, the build screen, uiModel and UIList; bubbletea-tui guidance", []string{`"Lint" matched inside "tuilint"`}},
	} {
		r := NewRedactor(Env{Cwd: t.TempDir()}, []string{c.word})
		if got := r.Apply(text); got != c.want {
			t.Errorf("--redact %s: Apply =\n %s\nwant\n %s", c.word, got, c.want)
		}
		if in := r.Inside(); !reflect.DeepEqual(in, c.inside) {
			t.Errorf("--redact %s: Inside = %q, want %q", c.word, in, c.inside)
		}
	}
}

// draft says which --redact word matched nothing, so the agent redacts the spelling the
// text uses.
func TestDraftWarnsAnUnmatchedRedact(t *testing.T) {
	r := newRig(t, false)
	r.run(r.bugArgs("draft", "--redact", "Café", "--redact", "Zoë", "--redact", "Acme Inc.", "--title", "Café and Zoë break for Acme Inc.")...)
	out, errs := r.stdout.String(), r.stderr.String()
	if strings.Contains(out, "Café") || strings.Contains(out, "Zoë") || strings.Contains(out, "Acme Inc.") || !strings.Contains(errs, "3 × <redacted>") {
		t.Errorf("not redacted:\n%s\n%s", out, errs)
	}
	if strings.Contains(errs, "matched nothing") {
		t.Errorf("warned about a word that matched: %s", errs)
	}
	r.run(r.bugArgs("draft", "--redact", "Kafe")...)
	if !strings.Contains(r.stderr.String(), `--redact "Kafe" matched nothing`) {
		t.Errorf("no warning for a word that matched nothing: %s", r.stderr.String())
	}
	r.run(r.bugArgs("draft", "--redact", "Lint")...)
	if !strings.Contains(r.stderr.String(), `--redact "Lint" matched inside "tuilint"`) {
		t.Errorf("no note for a word that matched inside another: %s", r.stderr.String())
	}
}

// Agents often run as root, node or runner: the user name is redacted where it is a path
// segment, never as a word ("the root model"), and a home directory is not a project
// whose name is redacted.
func TestRedactorUserOnlyInPaths(t *testing.T) {
	t.Setenv("PATH", "")
	const text = "--root model loses focus in kit/ui/root; root model"
	r := NewRedactor(Env{Cwd: t.TempDir(), Home: "/root", User: "root"}, nil)
	got := r.Apply(text + "; /root/.cache, file:///root/x and /home/root/x.")
	want := text + "; ~/.cache, file://~/x and /home/<user>/x."
	if got != want {
		t.Errorf("Apply =\n %s\nwant\n %s", got, want)
	}
	home := filepath.Join(t.TempDir(), "root")
	os.MkdirAll(home, 0o755)
	os.WriteFile(filepath.Join(home, "go.mod"), []byte("module scratch\n"), 0o644)
	for _, cwd := range []string{home, filepath.Join(home, "scratch")} {
		os.MkdirAll(cwd, 0o755)
		r = NewRedactor(Env{Cwd: cwd, Home: home, User: "root"}, nil)
		if got := r.Apply(text + " in " + home + "/x"); got != text+" in ~/x" {
			t.Errorf("run in %s: Apply = %q", cwd, got)
		}
	}
}

// From a subdirectory, the project is still the root: its path, its name and its module.
func TestRedactorFromASubdirectory(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(home, "acme-billing")
	sub := filepath.Join(root, "internal", "ui")
	os.MkdirAll(sub, 0o755)
	os.WriteFile(filepath.Join(root, "go.mod"), []byte("module github.com/acme-corp/acme-billing\n\ngo 1.26\n"), 0o644)
	const text = "panic in %s/main.go: github.com/acme-corp/acme-billing/internal/ui.(*List).View; " +
		"github.com/acme-corp/acme-billing.Run(); see https://github.com/acme-corp/acme-billing. acme-billing crashed"
	const want = "panic in <project>/main.go: <module>/internal/ui.(*List).View; " +
		"<module>.Run(); see https://<module>. <project> crashed"
	check := func(name string) {
		t.Helper()
		r := NewRedactor(Env{Cwd: sub, Home: home}, nil)
		if got := r.Apply(fmt.Sprintf(text, root)); got != want {
			t.Errorf("%s: Apply =\n %s\nwant\n %s", name, got, want)
		}
	}
	t.Run("go.mod", func(t *testing.T) {
		t.Setenv("PATH", "") // no git: the root is found by its markers
		check("go.mod")
	})
	t.Run("git", func(t *testing.T) {
		if _, err := exec.LookPath("git"); err != nil {
			t.Skip("no git")
		}
		// the go.mod is in a subdirectory; only git knows the root
		os.Rename(filepath.Join(root, "go.mod"), filepath.Join(root, "internal", "go.mod"))
		defer os.Rename(filepath.Join(root, "internal", "go.mod"), filepath.Join(root, "go.mod"))
		if out, err := exec.Command("git", "init", "-q", root).CombinedOutput(); err != nil {
			t.Fatalf("git init: %v %s", err, out)
		}
		check("git")
	})
}

// Every module of the project is redacted, not only the one on the path up from the
// working directory: nested modules, a go.work's modules (in the project or beside it),
// and what is left of any other path under the same owner. A symlinked working directory
// finds the same project, and the symlink's own path and name are redacted too.
func TestRedactorModulesAndSymlinks(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(home, "src", "acme-billing")
	write := func(path, text string) {
		os.MkdirAll(filepath.Dir(path), 0o755)
		os.WriteFile(path, []byte(text), 0o644)
	}
	write(filepath.Join(root, "go.mod"), "module github.com/acme-corp/acme-billing\n")
	write(filepath.Join(root, "tools", "gen", "go.mod"), "module github.com/acme-corp/acme-gen\n")
	write(filepath.Join(root, "vendor", "x", "go.mod"), "module github.com/vendored/x\n")
	write(filepath.Join(root, "go.work"), "go 1.26\n\nuse (\n\t.\n\t./tools/gen\n\t../acme-lib // the shared lib\n)\n")
	write(filepath.Join(home, "src", "acme-lib", "go.mod"), "module \"github.com/acme-corp/acme-lib\"\n")
	os.MkdirAll(filepath.Join(root, "internal", "ui"), 0o755)
	link := filepath.Join(home, "acme")
	if err := os.Symlink(root, link); err != nil {
		t.Skip("no symlinks:", err)
	}
	text := "panic: github.com/acme-corp/acme-billing/internal/ui.(*List).View; github.com/acme-corp/acme-gen/cmd; " +
		"github.com/acme-corp/acme-lib/pkg; github.com/acme-corp/other; github.com/vendored/x; " +
		"file://" + link + "/main.go and " + root + "/main.go; acme crashed"
	want := "panic: <module>/internal/ui.(*List).View; <module>/cmd; <module>/pkg; <owner>/other; github.com/vendored/x; " +
		"file://<project>/main.go and <project>/main.go; <project> crashed"
	check := func(t *testing.T) {
		// from the real root nothing names the link, so only the modules are checked there
		for cwd, want := range map[string]string{
			filepath.Join(link, "internal", "ui"): want,
			root:                                  strings.Replace(strings.Replace(want, "file://<project>", "file://~/acme", 1), "<project> crashed", "acme crashed", 1),
		} {
			r := NewRedactor(Env{Cwd: cwd, Home: home}, nil)
			if got := r.Apply(text); got != want {
				t.Errorf("from %s: Apply =\n %s\nwant\n %s", cwd, got, want)
			}
		}
	}
	t.Run("go.mod", func(t *testing.T) {
		t.Setenv("PATH", "")
		check(t)
	})
	t.Run("git", func(t *testing.T) {
		if _, err := exec.LookPath("git"); err != nil {
			t.Skip("no git")
		}
		if out, err := exec.Command("git", "init", "-q", root).CombinedOutput(); err != nil {
			t.Fatalf("git init: %v %s", err, out)
		}
		check(t)
	})
}

func TestGoWorkUses(t *testing.T) {
	got := goWorkUses("go 1.26\n\nuse ./a // one\nuse \"../b\"\nuse (\n\t./c\n\n\t\"./d\" // four\n)\n")
	if want := []string{"./a", "../b", "./c", "./d"}; !reflect.DeepEqual(got, want) {
		t.Errorf("goWorkUses = %q, want %q", got, want)
	}
}

// ---- the command, against stubs on PATH ---------------------------------------------

type rig struct {
	t      *testing.T
	env    Env
	dir    string // the stubs
	log    string // every gh call, one per line
	body   string // the body file gh issue create was given
	skill  string
	cwd    string
	stdout bytes.Buffer
	stderr bytes.Buffer
}

func stub(t *testing.T, dir, name, script string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\n"+script), 0o755); err != nil {
		t.Fatal(err)
	}
}

const ghStub = `echo "[$GH_HOST] $@" >> "$STUB_LOG"
case "$1 $2" in
"auth status") [ "$STUB_AUTH" = 1 ] && exit 0; echo "You are not logged into any GitHub hosts." >&2; exit 1;;
"issue create")
  prev=; for a; do [ "$prev" = "--body-file" ] && /bin/cat "$a" > "$STUB_BODY"; prev=$a; done
  if [ -n "$STUB_LABEL_FAIL" ]; then case "$*" in *--label*) echo "could not add label: 'eval' not found" >&2; exit 1;; esac; fi
  echo "Creating issue in o/r"; echo "https://github.com/o/r/issues/99";;
"issue comment") echo "https://github.com/o/r/issues/5#issuecomment-1";;
"search issues") /bin/cat "$STUB_SEARCH";;
esac
`

// newRig puts go, git and (with gh) a gh stub alone on PATH, a project directory with a
// skills-lock.json, and a skill directory with a SKILL.md, so a draft's automatic fields
// are the same on every machine.
func newRig(t *testing.T, gh bool) *rig {
	if runtime.GOOS == "windows" {
		t.Skip("shell stubs")
	}
	r := &rig{t: t, dir: t.TempDir(), cwd: t.TempDir(), skill: t.TempDir()}
	r.log = filepath.Join(r.dir, "gh.log")
	r.body = filepath.Join(r.dir, "body.md")
	stub(t, r.dir, "go", `if [ "$2" = "-m" ]; then printf '%s: go1.26.7\n\tbuild\tvcs.revision=abcdef1234567890\n\tbuild\tvcs.modified=true\n' "$3"; else echo "go version go1.26.7 linux/amd64"; fi
`)
	stub(t, r.dir, "git", "exit 128\n")
	stub(t, r.dir, "tuilint", "exit 0\n")
	if gh {
		stub(t, r.dir, "gh", ghStub)
	}
	t.Setenv("PATH", r.dir)
	t.Setenv("STUB_LOG", r.log)
	t.Setenv("STUB_BODY", r.body)
	t.Setenv("STUB_AUTH", "1")
	os.WriteFile(filepath.Join(r.cwd, "skills-lock.json"), []byte(`{"version":1,"skills":{"bubbletea-tui":{"source":"jrschumacher/skill-bubbletea-tui","computedHash":"44f983084426b97acc609344dccbbb0339ab712329ce3b96e571ab6e1ca4df14"}}}`), 0o644)
	os.WriteFile(filepath.Join(r.skill, "SKILL.md"), []byte("---\nname: bubbletea-tui\nmetadata:\n  version: \"0.1.0\"\n---\n"), 0o644)
	r.env = Env{
		Cwd: r.cwd, Home: "/home/ana", User: "ana", Getenv: func(k string) string { return "" },
		BuildInfo: func() (*debug.BuildInfo, bool) {
			return &debug.BuildInfo{Settings: []debug.BuildSetting{{Key: "vcs.revision", Value: "03eb52f928ca74949997632dd86925afb266f433"}, {Key: "vcs.modified", Value: "false"}}}, true
		},
	}
	return r
}

func (r *rig) run(args ...string) int {
	r.stdout.Reset()
	r.stderr.Reset()
	return Main(args, "USAGE", r.env, &r.stdout, &r.stderr)
}

// submit drafts args, takes the confirm hash the draft printed, and submits with it, as
// an agent does after the user's yes.
func (r *rig) submit(args ...string) int {
	draft := append([]string{"draft"}, args[1:]...)
	r.run(draft...)
	_, after, ok := strings.Cut(r.stdout.String(), "confirm:  ")
	if !ok {
		r.t.Fatalf("the draft printed no confirm hash:\n%s", r.stdout.String())
	}
	return r.run(append(args, "--confirm", after[:8])...)
}

func (r *rig) calls() string {
	b, _ := os.ReadFile(r.log)
	return string(b)
}

func (r *rig) bugArgs(mode string, extra ...string) []string {
	return append([]string{mode, "--kind", "bug", "--skill", r.skill,
		"--title", "tuilint F-CONTRAST fires on a ruled dialog's scrim at -profile ansi",
		"--field", "asked=/bubbletea-tui craft: a queue dashboard, its retry dialog",
		"--field", "area=tuilint frames F-CONTRAST",
		"--field", "variant=dark, -profile ansi, 80x24",
		"--field", "happened=F-CONTRAST on " + filepath.Join(r.cwd, "frames/16/confirm-80x24.ansi") + " row 9: the scrim's fgSubtle on bgSubtle, 2.1:1",
		"--field", "expected=The scrim is decoration under a ruled dialog: skip it, as at truecolour",
		"--field", "miss-class=c",
		"--field", "loop=yes",
	}, extra...)
}

// The draft is the exact issue: header, then the body as GitHub renders the form, with
// the version and environment filled in and the project path redacted.
func TestDraftGolden(t *testing.T) {
	r := newRig(t, false)
	if code := r.run(r.bugArgs("draft")...); code != ExitOK {
		t.Fatalf("draft exit %d: %s", code, r.stderr.String())
	}
	golden := filepath.Join("testdata", "draft-bug.golden")
	if *update {
		os.WriteFile(golden, r.stdout.Bytes(), 0o644)
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if r.stdout.String() != string(want) {
		t.Errorf("draft differs from %s (go test -run TestDraftGolden -update):\n%s", golden, r.stdout.String())
	}
	if !strings.Contains(r.stderr.String(), "nothing was sent; redacted 1 × <project>") {
		t.Errorf("stderr = %q, want the redaction reported", r.stderr.String())
	}
	if r.calls() != "" {
		t.Errorf("draft called gh: %s", r.calls())
	}
}

func TestDraftMissingRequired(t *testing.T) {
	r := newRig(t, true)
	code := r.run("draft", "--kind", "eval", "--title", "an eval", "--field", "case=new")
	if code != ExitFailed || !strings.Contains(r.stderr.String(), "missing required: mistake, assertion") {
		t.Errorf("exit %d, stderr %q; want 1 and the missing ids", code, r.stderr.String())
	}
	if !strings.Contains(r.stdout.String(), "### Assertion\n\n_No response_") {
		t.Errorf("the draft prints anyway:\n%s", r.stdout.String())
	}
	if code := r.run("submit", "--kind", "eval", "--title", "an eval", "--field", "case=new"); code != ExitUsage {
		t.Errorf("submit with missing fields exit %d, want 2", code)
	}
	if strings.Contains(r.calls(), "issue create") {
		t.Error("submit with missing fields reached gh")
	}
}

// submit files only the draft the user saw: no hash, or a hash of another draft (a field
// changed after the yes), sends nothing.
func TestSubmitNeedsTheDraftsHash(t *testing.T) {
	r := newRig(t, true)
	r.run(r.bugArgs("draft")...)
	_, after, _ := strings.Cut(r.stdout.String(), "confirm:  ")
	hash := after[:8]
	if code := r.run(r.bugArgs("submit")...); code != ExitUsage || !strings.Contains(r.stderr.String(), "submit needs --confirm") {
		t.Errorf("no --confirm: exit %d, stderr %q", code, r.stderr.String())
	}
	changed := r.bugArgs("submit", "--field", "expected=something the user never read", "--confirm", hash)
	if code := r.run(changed...); code != ExitUsage || !strings.Contains(r.stderr.String(), "changed since the draft") {
		t.Errorf("changed body: exit %d, stderr %q", code, r.stderr.String())
	}
	refused := r.stderr.String()
	r.run(r.bugArgs("draft", "--field", "expected=something the user never read")...)
	if _, after, _ := strings.Cut(r.stdout.String(), "confirm:  "); strings.Contains(refused, after[:8]) {
		t.Errorf("the refusal gave away the changed draft's hash %s: %s", after[:8], refused)
	}
	if strings.Contains(r.calls(), "issue create") {
		t.Errorf("filed without a matching hash:\n%s", r.calls())
	}
	if code := r.run(r.bugArgs("submit", "--confirm", hash)...); code != ExitOK {
		t.Errorf("the draft's own hash: exit %d, stderr %q", code, r.stderr.String())
	}
}

func TestUsageErrors(t *testing.T) {
	r := newRig(t, false)
	for _, args := range [][]string{
		{},
		{"file"},
		{"draft", "--kind", "question", "--title", "x"},
		{"draft", "--kind", "bug", "--title", "x", "--field", "colour=red"},
		{"draft", "--kind", "bug", "--title", "x", "--field", "miss-class=d"},
		{"draft", "--kind", "bug", "--title", "x", "--field", "noequals"},
		{"draft", "--kind", "bug", "--title", "x", "--repo", "not a repo"},
		{"draft", "--kind", "bug", "--bogus"},
		{"draft", "--kind", "bug", "loose words"},
		{"similar", "--title", "a an of"},
	} {
		if code := r.run(args...); code != ExitUsage {
			t.Errorf("%v: exit %d, want 2 (stderr %q)", args, code, r.stderr.String())
		}
	}
	if code := r.run("-h"); code != ExitOK || r.stdout.String() != "USAGE\n" {
		t.Errorf("-h: exit %d, stdout %q", code, r.stdout.String())
	}
	if code := r.run("submit", "-h"); code != ExitOK || r.stdout.String() != "USAGE\n" {
		t.Errorf("submit -h: exit %d, stdout %q", code, r.stdout.String())
	}
}

func TestSubmitWithGh(t *testing.T) {
	r := newRig(t, true)
	t.Setenv("GH_HOST", "ghe.example.com")
	if code := r.submit(r.bugArgs("submit")...); code != ExitOK {
		t.Fatalf("submit exit %d: %s", code, r.stderr.String())
	}
	if got := strings.TrimSpace(r.stdout.String()); got != "https://github.com/o/r/issues/99" {
		t.Errorf("stdout = %q, want the new issue's URL alone", got)
	}
	calls := r.calls()
	if !strings.Contains(calls, "[github.com] auth status --hostname github.com\n") ||
		!strings.Contains(calls, "[github.com] issue create --repo github.com/jrschumacher/skill-bubbletea-tui --title=tuilint F-CONTRAST") ||
		!strings.Contains(calls, "--label feedback --label bug") || strings.Contains(calls, "ghe.example.com") {
		t.Errorf("gh calls, each pinned to github.com:\n%s", calls)
	}
	body, _ := os.ReadFile(r.body)
	golden, _ := os.ReadFile(filepath.Join("testdata", "draft-bug.golden"))
	if _, wantBody, _ := strings.Cut(string(golden), "\n\n"); string(body) != wantBody {
		t.Errorf("gh got a body that isn't the draft's:\n%s", body)
	}
}

func TestSubmitForkWithoutLabels(t *testing.T) {
	r := newRig(t, true)
	t.Setenv("STUB_LABEL_FAIL", "1")
	r.env.Getenv = func(k string) string {
		if k == "FEEDBACK_REPO" {
			return "me/fork"
		}
		return ""
	}
	code := r.submit("submit", "--kind", "eval", "--title", "catch a torn footer", "--field", "case=new", "--field", "mistake=m", "--field", "assertion=a")
	if code != ExitOK || !strings.Contains(r.stderr.String(), "filing without labels") {
		t.Errorf("exit %d, stderr %q", code, r.stderr.String())
	}
	if !strings.Contains(r.calls(), "issue create --repo github.com/me/fork") {
		t.Errorf("FEEDBACK_REPO not used:\n%s", r.calls())
	}
}

func TestSubmitComment(t *testing.T) {
	r := newRig(t, true)
	code := r.submit("submit", "--kind", "bug", "--comment", "12", "--field", "evidence=@@media queries don't apply; also at 120x40")
	if code != ExitOK || !strings.Contains(r.calls(), "issue comment 12 --repo github.com/jrschumacher/skill-bubbletea-tui --body-file") {
		t.Errorf("exit %d, calls:\n%s", code, r.calls())
	}
	r.run("draft", "--kind", "bug", "--comment", "12", "--field", "evidence=@@media queries don't apply")
	if !strings.Contains(r.stdout.String(), "### Frames, sheets or a minimal repro\n\n@media queries don't apply\n") {
		t.Errorf("@@ is a literal @:\n%s", r.stdout.String())
	}
	if code := r.run("submit", "--kind", "bug", "--comment", "12", "--confirm", "00000000"); code != ExitUsage {
		t.Errorf("an empty comment: exit %d, want 2", code)
	}
}

func TestSubmitWithoutGh(t *testing.T) {
	for _, c := range []struct {
		name, why string
		gh        bool
	}{
		{"absent", "gh is not installed", false},
		{"unauthenticated", "gh is not signed in", true},
	} {
		t.Run(c.name, func(t *testing.T) {
			r := newRig(t, c.gh)
			t.Setenv("STUB_AUTH", "0")
			opened := filepath.Join(r.dir, "opened")
			stub(t, r.dir, "xdg-open", `echo "$1" > "`+opened+"\"\n")
			code := r.submit(r.bugArgs("submit")...)
			if code != ExitNotFiled || !strings.Contains(r.stderr.String(), "nothing was filed: "+c.why) {
				t.Fatalf("exit %d, stderr %q; want 3 and %q", code, r.stderr.String(), c.why)
			}
			link := strings.TrimSpace(r.stdout.String())
			if !strings.HasPrefix(link, "https://github.com/jrschumacher/skill-bubbletea-tui/issues/new?template=bug.yml&title=") {
				t.Errorf("stdout = %q, want the prefilled link", link)
			}
			if !strings.Contains(r.stderr.String(), "pick in the form (GitHub can't prefill a dropdown): Kind of miss: (c) prose can't express it") {
				t.Errorf("the dropdown isn't named to pick: %s", r.stderr.String())
			}
			if strings.Contains(r.calls(), "issue create") {
				t.Error("filed without auth")
			}
			if _, err := os.Stat(opened); err == nil {
				t.Error("opened a browser without --open")
			}
			if runtime.GOOS == "linux" {
				r.submit(r.bugArgs("submit", "--open")...)
				if b, _ := os.ReadFile(opened); strings.TrimSpace(string(b)) != link {
					t.Errorf("--open opened %q, want %q", b, link)
				}
			}
		})
	}
}

func TestSubmitWithoutGhTooLong(t *testing.T) {
	r := newRig(t, false)
	out := filepath.Join(r.dir, "body.md")
	code := r.submit(r.bugArgs("submit", "--out", out, "--field", "evidence="+strings.Repeat("row 9 cell 41: fgSubtle on bgSubtle\n", 300))...)
	if code != ExitNotFiled {
		t.Fatalf("exit %d", code)
	}
	if len(strings.TrimSpace(r.stdout.String())) > URLLimit {
		t.Errorf("link is %d bytes", len(r.stdout.String()))
	}
	if !strings.Contains(r.stderr.String(), "paste these fields from "+out+": Frames, sheets or a minimal repro") {
		t.Errorf("stderr = %q", r.stderr.String())
	}
	if b, _ := os.ReadFile(out); !strings.Contains(string(b), "### Frames, sheets or a minimal repro\n\nrow 9 cell 41") {
		t.Errorf("the body file lacks the field:\n%.300s", b)
	}
}

func TestSimilar(t *testing.T) {
	r := newRig(t, true)
	results := filepath.Join(r.dir, "search.json")
	os.WriteFile(results, []byte(`[{"number":40,"title":"F-CONTRAST on scrims at 16 colours","state":"OPEN","url":"https://github.com/o/r/issues/40"}]`), 0o644)
	t.Setenv("STUB_SEARCH", results)
	if code := r.run("similar", "--title", "F-CONTRAST fires on the scrim"); code != ExitOK {
		t.Fatalf("exit %d: %s", code, r.stderr.String())
	}
	if !strings.Contains(r.stdout.String(), "#40   open   F-CONTRAST on scrims at 16 colours\n      https://github.com/o/r/issues/40") {
		t.Errorf("stdout:\n%s", r.stdout.String())
	}
	if !strings.Contains(r.calls(), "search issues --repo jrschumacher/skill-bubbletea-tui --json number,title,state,url --limit 10 -- f-contrast fires scrim") {
		t.Errorf("calls:\n%s", r.calls())
	}

	os.WriteFile(results, []byte(`[]`), 0o644)
	os.Remove(r.log)
	r.run("similar", "--title", "acmeops: wordmark tracked style drops the last letter", "--redact", "AcmeOps")
	calls := r.calls()
	if !strings.Contains(r.stdout.String(), "no issues match") || !strings.Contains(calls, "-- wordmark tracked style letter\n") ||
		strings.Count(calls, "search issues") != 1 || strings.Contains(strings.ToLower(calls+r.stdout.String()), "acmeops") {
		t.Errorf("no hits: stdout %q; one search of the four longest words, the name redacted? calls:\n%s", r.stdout.String(), calls)
	}

	t.Setenv("STUB_AUTH", "0")
	if code := r.run("similar", "--title", "F-CONTRAST fires on the scrim"); code != ExitNotFiled ||
		strings.TrimSpace(r.stdout.String()) != SearchURL(DefaultRepo, []string{"f-contrast", "fires", "scrim"}) {
		t.Errorf("without gh: exit %d, stdout %q", code, r.stdout.String())
	}
}

// The -h text is the package doc of cmd/feedback; every kind, field id and exit status
// the tool knows must be in it, or an agent reading -h can't fill a draft.
func TestUsageListsEveryField(t *testing.T) {
	src, err := os.ReadFile(filepath.Join("..", "..", "cmd", "feedback", "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	doc := cmddoc.Text(string(src))
	for _, k := range Kinds {
		// the kind's line and its continuation lines (indented past the kind names)
		block, in := "", false
		for _, l := range strings.Split(doc, "\n") {
			switch {
			case strings.HasPrefix(l, "  "+k.Name+" "):
				in = true
			case in && strings.HasPrefix(l, "   "):
			default:
				in = false
			}
			if in {
				block += l + "\n"
			}
		}
		if block == "" {
			t.Errorf("-h has no fields line for %s", k.Name)
			continue
		}
		for _, f := range k.Fields {
			mark := f.ID
			if f.Required {
				mark += "*"
			}
			if !strings.Contains(block, mark) {
				t.Errorf("-h lists %s's fields without %q", k.Name, mark)
			}
		}
	}
	for _, s := range []string{"Exit status: 0", "; 1 ", "; 2 ", "; 3 ", "FEEDBACK_REPO", "--open"} {
		if !strings.Contains(doc, s) {
			t.Errorf("-h lacks %q", s)
		}
	}
}
