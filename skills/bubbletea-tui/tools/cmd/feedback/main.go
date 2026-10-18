// Feedback drafts and files an issue on the skill's repository, so a miss an agent hits
// while building a TUI (a rule that contradicts the kit, a lint that fires falsely, a
// component it had to write, a tool too slow, an eval that would have caught a mistake)
// reaches the people who fix it. Filing publishes text on the user's behalf: show them
// the draft and file only on their yes (references/cmd-feedback.md).
//
//	feedback draft   --kind bug|perf|eval|feature --title "…" [--field key=value …]
//	feedback similar --title "…" [--redact WORD …]
//	feedback submit  --kind bug|perf|eval|feature --title "…" [--field key=value …] --confirm HASH [--open]
//
// draft prints the exact issue (repo, template, labels, title, a confirm hash, then the
// body as GitHub renders the form) and sends nothing. The hash covers exactly what
// submit would send; submit refuses without --confirm set to it, so what is filed is
// the draft the user saw and said yes to. Change a field and the hash changes: draft
// again and show the user again.
//
// similar lists existing issues, open and closed, whose titles share the four longest
// words of the redacted --title (gh search issues), so a finding can go on one of them
// instead of a duplicate; without gh it prints the search link.
//
// submit files the issue with gh issue create when gh is installed and gh auth status
// passes, and prints its URL. Otherwise it files nothing: it prints the new-issue link
// with the template, title, labels and fields filled in, capped at 8000 bytes (fields
// that don't fit are written to a file to paste, and dropdowns, which GitHub can't
// prefill, are named to pick), and opens it in the browser only with --open (open,
// xdg-open, or the URL handler on Windows). Ask the user before passing --open. Every
// gh call goes to github.com, whatever GH_HOST says.
//
// Options (the same for draft and submit; similar reads --title, --repo, --redact and
// --no-redact):
//
//	--kind K          bug, perf (performance), eval (eval-suggestion) or feature (enhancement)
//	--title T         the issue title: the miss in one line, not the app
//	--field key=value a field by its id, repeatable; key=@file reads the value from a file,
//	                  key=@@text is the literal "@text". A dropdown takes its option, its
//	                  letter (miss-class=b) or a unique prefix
//	--confirm HASH    submit: the hash the draft printed, after the user's yes to it
//	--comment N       post the fields that have values as a comment on issue N (more
//	                  evidence for an open issue) instead of opening one
//	--repo owner/name file on a fork; FEEDBACK_REPO does the same (default
//	                  jrschumacher/skill-bubbletea-tui)
//	--skill DIR       the skill directory, for its version (default: the project's
//	                  .claude/skills or .agents/skills copy, then ~/.claude/skills)
//	--redact WORD     also replace WORD (the app's name, a customer's) with <redacted>,
//	                  repeatable, in any case and script. A WORD of 4+ letters matches
//	                  inside longer words too (acmeops), a shorter one only on its own or
//	                  as an identifier's part (WORDModel). stderr names a WORD that
//	                  matched nothing, and one that matched inside a longer word
//	--no-redact       keep paths, modules, owners and the user name as written
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
// Every field and the title, and similar's search words, are redacted before they are
// printed or sent, wherever under the project the tool runs: the project root's path (the
// git toplevel, else the nearest directory holding TUI.md or go.mod, symlinks resolved)
// and its directory name become <project>, as do the symlink's when the tool runs through
// one; the path of every Go module in it (vendor and testdata aside) or in its go.work
// becomes <module>, and each one's host and owner (github.com/acme-corp) <owner>; the
// home directory becomes ~, and the user name <user> where it is a path segment
// (/home/<user>, /Users/<user>), never as a bare word. A project root that is the home
// directory is not a project. draft and submit say on stderr what they replaced; tell
// the user.
//
// Exit status: 0 drafted, filed, or searched; 1 gh failed, or a draft lacks a required
// field (it prints anyway, the missing ids on stderr); 2 a usage error (bad flag, kind,
// field or value, submit missing a required field, or a --confirm that is missing or
// doesn't match the issue): nothing was sent; 3 no usable gh (not installed, or gh auth
// status fails): nothing was filed or searched, and the link to finish by hand is on
// stdout.
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
