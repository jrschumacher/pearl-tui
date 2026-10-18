// Feedback drafts and files an issue on the skill's repository, so a miss an agent hits
// while building a TUI (a rule that contradicts the kit, a lint that fires falsely, a
// component it had to write, a tool too slow, an eval that would have caught a mistake)
// reaches the people who fix it. Filing publishes text on the user's behalf: show them
// the draft and file only on their yes (references/cmd-feedback.md).
//
//	feedback draft   --kind bug|perf|eval|feature --title "…" [--field key=value …]
//	feedback similar --title "…"
//	feedback submit  --kind bug|perf|eval|feature --title "…" [--field key=value …] [--open]
//
// draft prints the exact issue (repo, template, labels, title, then the body as GitHub
// renders the form) and sends nothing. similar lists existing issues, open and closed,
// whose titles share the words of --title (gh search issues), so a finding can go on
// one of them instead of a duplicate; without gh it prints the search link. submit files
// the issue with gh issue create when gh is installed and gh auth status passes, and
// prints its URL. Otherwise it files nothing: it prints the new-issue link with the
// template, title, labels and fields filled in, capped at 8000 bytes (fields that don't
// fit are written to a file to paste, and dropdowns, which GitHub can't prefill, are
// named to pick), and opens it in the browser only with --open (open, xdg-open, or the
// URL handler on Windows). Ask the user before passing --open.
//
// Options (the same for draft and submit; similar reads --title and --repo):
//
//	--kind K          bug, perf (performance), eval (eval-suggestion) or feature (enhancement)
//	--title T         the issue title: the miss in one line, not the app
//	--field key=value a field by its id, repeatable; key=@file reads the value from a file.
//	                  A dropdown takes its option, its letter (miss-class=b) or a unique prefix
//	--comment N       post the fields that have values as a comment on issue N (more
//	                  evidence for an open issue) instead of opening one
//	--repo owner/name file on a fork; FEEDBACK_REPO does the same (default
//	                  jrschumacher/skill-bubbletea-tui)
//	--skill DIR       the skill directory, for its version (default: the project's
//	                  .claude/skills or .agents/skills copy, then ~/.claude/skills)
//	--redact WORD     also replace WORD (the app's name, a customer's) with <redacted>,
//	                  repeatable, whole words, any case
//	--no-redact       keep the project path, home directory and user name as written
//	--out FILE        where submit writes the body when the link can't carry it
//	--open            submit without gh: open the link in the browser
//
// Fields by kind (* required; skill-version and environment are filled in when not given:
// SKILL.md's version, the skill checkout's commit, skills-lock.json's computedHash, the
// commit the tools were built from, go version, and the build of each tool a field
// names, read with go version -m):
//
//	bug       asked*, skill-version*, area*, variant, happened*, expected*,
//	          miss-class* (a|b|c|not sure), evidence, loop (yes|partly|no), environment
//	perf      asked*, skill-version*, area*, command*, timing*, sizes*, expected,
//	          evidence, environment
//	eval      case*, prompt, mistake*, assertion*, fixtures, skill-version
//	feature   asked*, area* (skill|kit|themes|tools|scripts|evals), workaround*,
//	          proposal*, evidence, skill-version*, environment
//
// Every field and the title are redacted before they are printed or sent: the project's
// absolute path becomes <project>, the home directory ~, the user name <user>. draft and
// submit say on stderr what they replaced; tell the user.
//
// Exit status: 0 drafted, filed, or searched; 1 gh failed, or a draft lacks a required
// field (it prints anyway, the missing ids on stderr); 2 a usage error (bad flag, kind,
// field or value, or submit missing a required field): nothing was sent; 3 no usable gh
// (not installed, or gh auth status fails): nothing was filed or searched, and the link
// to finish by hand is on stdout.
package main

import (
	_ "embed"
	"os"

	"tuitools/internal/cmddoc"
	"tuitools/internal/feedback"
)

//go:embed main.go
var src string

func main() {
	os.Exit(feedback.Main(os.Args[1:], cmddoc.Text(src), feedback.ProcessEnv(), os.Stdout, os.Stderr))
}
