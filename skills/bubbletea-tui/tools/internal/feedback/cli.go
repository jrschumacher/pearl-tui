package feedback

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// Exit statuses, as the package doc of cmd/feedback documents them.
const (
	ExitOK       = 0 // drafted, filed, or searched
	ExitFailed   = 1 // gh failed; or a draft lacks a required field (printed anyway)
	ExitUsage    = 2 // a bad flag, kind, field or value; nothing was sent
	ExitNotFiled = 3 // no usable gh: nothing was sent, the link to finish by hand is printed
)

const usageLine = "usage: feedback draft|similar|submit [--kind K] [--title T] [--field key=value ...] [-h]"

type multi []string

func (m *multi) String() string     { return strings.Join(*m, ",") }
func (m *multi) Set(v string) error { *m = append(*m, v); return nil }

type opts struct {
	kind, title, repo, skill, out, confirm string
	fields, redact                         multi
	noRedact, open                         bool
	comment                                int
}

// Main runs the feedback command and returns its exit status.
func Main(args []string, usage string, env Env, stdout, stderr io.Writer) int {
	bad := func(format string, a ...any) int {
		fmt.Fprintln(stderr, usageLine)
		fmt.Fprintf(stderr, "feedback: error: "+format+"\n", a...)
		return ExitUsage
	}
	if len(args) == 0 {
		return bad("name a mode: draft, similar or submit")
	}
	mode := args[0]
	switch mode {
	case "-h", "--help", "-help", "help":
		fmt.Fprintln(stdout, usage)
		return ExitOK
	case "draft", "similar", "submit":
	default:
		return bad("unknown mode %q (draft, similar or submit)", mode)
	}
	var o opts
	fs := flag.NewFlagSet("feedback "+mode, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&o.kind, "kind", "", "")
	fs.StringVar(&o.title, "title", "", "")
	fs.Var(&o.fields, "field", "")
	fs.StringVar(&o.confirm, "confirm", "", "")
	fs.StringVar(&o.repo, "repo", "", "")
	fs.StringVar(&o.skill, "skill", "", "")
	fs.Var(&o.redact, "redact", "")
	fs.BoolVar(&o.noRedact, "no-redact", false, "")
	fs.IntVar(&o.comment, "comment", 0, "")
	fs.BoolVar(&o.open, "open", false, "")
	fs.StringVar(&o.out, "out", "", "")
	if err := fs.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprintln(stdout, usage)
			return ExitOK
		}
		return bad("%v", err)
	}
	if fs.NArg() > 0 {
		return bad("unexpected argument %q (quote the title: --title \"...\")", fs.Arg(0))
	}
	repo := o.repo
	if repo == "" {
		repo = env.Getenv("FEEDBACK_REPO")
	}
	if repo == "" {
		repo = DefaultRepo
	}
	if !regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`).MatchString(repo) {
		return bad("--repo %q is not owner/name", repo)
	}

	if mode == "similar" {
		title := o.title
		if !o.noRedact {
			red := NewRedactor(env, o.redact)
			title = red.Apply(title)
			unmatched(stderr, red, o.redact)
		}
		return similar(repo, title, stdout, stderr, bad)
	}

	is, red, err := build(o, repo, env)
	if err != nil {
		return bad("%v", err)
	}
	missing := is.Missing()
	if o.noRedact {
		if len(o.redact) > 0 {
			fmt.Fprintln(stderr, "feedback: --no-redact: the --redact words are kept as written")
		}
	} else {
		unmatched(stderr, red, o.redact)
	}
	if mode == "draft" {
		fmt.Fprint(stdout, is.Draft())
		note(stderr, red, "nothing was sent")
		fmt.Fprintf(stderr, "feedback: after the user's yes to this draft: feedback submit <the same flags> --confirm %s\n", is.Hash())
		if len(missing) > 0 {
			fmt.Fprintf(stderr, "feedback: missing required: %s\n", strings.Join(missing, ", "))
			return ExitFailed
		}
		return ExitOK
	}
	if len(missing) > 0 {
		return bad("missing required: %s (nothing was sent; see feedback draft)", strings.Join(missing, ", "))
	}
	switch {
	case o.confirm == "":
		return bad("submit needs --confirm <hash>, from the draft the user said yes to (feedback draft prints it); nothing was sent")
	case !strings.EqualFold(strings.TrimSpace(o.confirm), is.Hash()):
		// the right hash is not printed here: it comes from a draft the user saw
		return bad("--confirm %s is not this issue: it changed since the draft the user saw; draft it again and show them; nothing was sent", o.confirm)
	}
	return submit(is, o, red, stdout, stderr)
}

// build turns the flags into a redacted issue with its automatic fields filled.
func build(o opts, repo string, env Env) (Issue, *Redactor, error) {
	if o.kind == "" {
		return Issue{}, nil, errors.New("--kind is required (bug|perf|eval|feature)")
	}
	k, err := KindNamed(o.kind)
	if err != nil {
		return Issue{}, nil, err
	}
	if o.comment < 0 {
		return Issue{}, nil, fmt.Errorf("--comment %d is not an issue number", o.comment)
	}
	is := Issue{Repo: repo, Kind: k, Title: strings.TrimSpace(o.title), Values: map[string]string{}, Comment: o.comment}
	for _, kv := range o.fields {
		key, val, ok := strings.Cut(kv, "=")
		if !ok {
			return Issue{}, nil, fmt.Errorf("--field %q is not key=value", kv)
		}
		f, ok := k.Field(key)
		if !ok {
			return Issue{}, nil, fmt.Errorf("--kind %s has no field %q (it has %s)", k.Name, key, k.fieldList())
		}
		if strings.HasPrefix(val, "@@") {
			val = val[1:] // a literal leading @
		} else if strings.HasPrefix(val, "@") {
			b, err := os.ReadFile(val[1:])
			if err != nil {
				return Issue{}, nil, fmt.Errorf("--field %s: %v", key, err)
			}
			val = string(b)
		}
		v, err := f.Choose(val)
		if err != nil {
			return Issue{}, nil, err
		}
		is.Values[key] = strings.TrimSpace(v)
	}
	for _, f := range k.Fields {
		if !f.Auto || strings.TrimSpace(is.Values[f.ID]) != "" {
			continue
		}
		switch f.ID {
		case "skill-version":
			is.Values[f.ID] = env.SkillVersion(env.SkillDir(o.skill))
		case "environment":
			is.Values[f.ID] = env.Environment(is.Values)
		}
	}
	red := NewRedactor(env, o.redact)
	if !o.noRedact {
		is.Title = red.Apply(is.Title)
		for _, f := range k.Fields {
			if v, ok := is.Values[f.ID]; ok {
				is.Values[f.ID] = red.Apply(v)
			}
		}
	}
	return is, red, nil
}

// unmatched warns about each --redact word that matched nothing: spelled differently in
// the text, it is still there.
func unmatched(stderr io.Writer, red *Redactor, words []string) {
	for _, w := range red.Unmatched(words) {
		fmt.Fprintf(stderr, "feedback: --redact %q matched nothing; if the text spells it another way, redact that spelling\n", w)
	}
}

func note(stderr io.Writer, red *Redactor, what string) {
	msg := "feedback: " + what
	if r := red.Report(); r != "" {
		msg += "; " + r + " (--redact WORD for more, --no-redact for none)"
	}
	fmt.Fprintln(stderr, msg)
}

// ghReady reports whether gh can file: installed and signed in. why says what's missing.
func ghReady() (ok bool, why string) {
	if _, err := exec.LookPath("gh"); err != nil {
		return false, "gh is not installed"
	}
	if _, _, err := gh([]string{"auth", "status", "--hostname", "github.com"}); err != nil {
		return false, "gh is not signed in (gh auth status failed; gh auth login fixes it)"
	}
	return true, ""
}

func submit(is Issue, o opts, red *Redactor, stdout, stderr io.Writer) int {
	ok, why := ghReady()
	if !ok {
		return byHand(is, o, red, why, stdout, stderr)
	}
	f, err := os.CreateTemp("", "feedback-*.md")
	if err != nil {
		fmt.Fprintf(stderr, "feedback: %v\n", err)
		return ExitFailed
	}
	defer os.Remove(f.Name())
	if _, err := f.WriteString(is.Body()); err != nil {
		f.Close()
		fmt.Fprintf(stderr, "feedback: %v\n", err)
		return ExitFailed
	}
	f.Close()
	var args []string
	if is.Comment > 0 {
		args = []string{"issue", "comment", strconv.Itoa(is.Comment), "--repo", "github.com/" + is.Repo, "--body-file", f.Name()}
	} else {
		args = []string{"issue", "create", "--repo", "github.com/" + is.Repo, "--title=" + is.Title, "--body-file", f.Name()}
		for _, l := range is.Kind.Labels {
			args = append(args, "--label", l)
		}
	}
	out, errOut, err := gh(args)
	if err != nil && is.Comment == 0 && strings.Contains(strings.ToLower(errOut), "label") {
		// a fork without the labels: file it unlabelled rather than not at all
		fmt.Fprintf(stderr, "feedback: %s has no %s label; filing without labels\n", is.Repo, strings.Join(is.Kind.Labels, "/"))
		out, errOut, err = gh(args[:7]) // issue create --repo R --title=T --body-file F
	}
	if err != nil {
		fmt.Fprintf(stderr, "feedback: gh %s failed: %s\n", args[1], strings.TrimSpace(errOut))
		return ExitFailed
	}
	fmt.Fprintln(stdout, lastLine(out))
	note(stderr, red, "filed")
	return ExitOK
}

// gh runs gh against github.com whatever GH_HOST says, so an enterprise default can't
// send the user's report somewhere else.
func gh(args []string) (stdout, stderr string, err error) {
	var o, e bytes.Buffer
	cmd := exec.Command("gh", args...)
	cmd.Env = append(os.Environ(), "GH_HOST=github.com")
	cmd.Stdout, cmd.Stderr = &o, &e
	err = cmd.Run()
	return o.String(), e.String(), err
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}

// byHand prints the link that finishes the filing in a browser, writes what the link
// can't carry to a file, and opens the link only with --open.
func byHand(is Issue, o opts, red *Redactor, why string, stdout, stderr io.Writer) int {
	var link string
	var paste []string
	var pick []string
	if is.Comment > 0 {
		link = IssueURL(is.Repo, is.Comment)
	} else {
		var skipped []string
		link, skipped = is.NewIssueURL(URLLimit)
		for _, id := range skipped {
			if f, _ := is.Kind.Field(id); f.Type == "dropdown" {
				pick = append(pick, fmt.Sprintf("%s: %s", f.Label, is.Values[id]))
			} else {
				paste = append(paste, f.Label)
			}
		}
	}
	fmt.Fprintf(stderr, "feedback: nothing was filed: %s\n", why)
	if is.Comment > 0 || len(paste) > 0 {
		path, err := writeBody(is, o.out)
		if err != nil {
			fmt.Fprintf(stderr, "feedback: %v\n", err)
			return ExitFailed
		}
		if is.Comment > 0 {
			fmt.Fprintf(stderr, "feedback: paste %s as a comment on #%d\n", path, is.Comment)
		} else {
			fmt.Fprintf(stderr, "feedback: the link is capped at %d bytes; paste these fields from %s: %s\n", URLLimit, path, strings.Join(paste, ", "))
		}
	}
	for _, p := range pick {
		fmt.Fprintf(stderr, "feedback: pick in the form (GitHub can't prefill a dropdown): %s\n", p)
	}
	fmt.Fprintln(stdout, link)
	note(stderr, red, "the link holds the draft")
	if o.open {
		if err := openURL(link); err != nil {
			fmt.Fprintf(stderr, "feedback: could not open a browser: %v\n", err)
		}
	}
	return ExitNotFiled
}

func writeBody(is Issue, path string) (string, error) {
	if path == "" {
		f, err := os.CreateTemp("", "feedback-"+is.Kind.Name+"-*.md")
		if err != nil {
			return "", err
		}
		path = f.Name()
		f.Close()
	}
	if err := os.WriteFile(path, []byte(is.Body()), 0o644); err != nil {
		return "", err
	}
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	return path, nil
}

// openURL hands link to the platform's opener: open on macOS, the URL protocol handler on
// Windows (start is a cmd builtin that splits a URL at its &), xdg-open elsewhere.
func openURL(link string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", link)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", link)
	default:
		cmd = exec.Command("xdg-open", link)
	}
	return cmd.Run()
}

type hit struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	State  string `json:"state"`
	URL    string `json:"url"`
}

func similar(repo, title string, stdout, stderr io.Writer, bad func(string, ...any) int) int {
	// a placeholder is not a word the issue would hold
	terms := Terms(placeholderRe.ReplaceAllLiteralString(title, " "))
	if len(terms) == 0 {
		return bad("--title %q has no words to search for", title)
	}
	ok, why := ghReady()
	if !ok {
		fmt.Fprintf(stderr, "feedback: not searched: %s; open the link to look\n", why)
		fmt.Fprintln(stdout, SearchURL(repo, terms))
		return ExitNotFiled
	}
	args := append([]string{"search", "issues", "--repo", repo, "--json", "number,title,state,url", "--limit", "10", "--"}, terms...)
	out, errOut, err := gh(args)
	if err != nil {
		fmt.Fprintf(stderr, "feedback: gh search failed: %s\n", strings.TrimSpace(errOut))
		return ExitFailed
	}
	var hits []hit
	if err := json.Unmarshal([]byte(out), &hits); err != nil {
		fmt.Fprintf(stderr, "feedback: gh search: %v\n", err)
		return ExitFailed
	}
	if len(hits) > 0 {
		fmt.Fprintf(stdout, "issues matching %q:\n", strings.Join(terms, " "))
		for _, h := range hits {
			fmt.Fprintf(stdout, "#%-4d %-6s %s\n      %s\n", h.Number, strings.ToLower(h.State), h.Title, h.URL)
		}
		return ExitOK
	}
	fmt.Fprintf(stdout, "no issues match %q: %s\n", strings.Join(terms, " "), SearchURL(repo, terms))
	return ExitOK
}
