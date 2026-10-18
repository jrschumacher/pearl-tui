package feedback

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Env is what the tool reads from the machine. Main fills it from the process; tests
// replace the parts that differ between machines (the build info, the directories).
type Env struct {
	Cwd       string
	Home      string
	User      string
	Getenv    func(string) string
	BuildInfo func() (*debug.BuildInfo, bool)
}

// ProcessEnv is the Env of the running process.
func ProcessEnv() Env {
	cwd, _ := os.Getwd()
	home, _ := os.UserHomeDir()
	user := os.Getenv("USER")
	if user == "" {
		user = os.Getenv("USERNAME")
	}
	return Env{Cwd: cwd, Home: home, User: user, Getenv: os.Getenv, BuildInfo: debug.ReadBuildInfo}
}

// run runs a command and returns its trimmed stdout, or "" when it fails or is missing.
func run(name string, args ...string) string {
	if _, err := exec.LookPath(name); err != nil {
		return ""
	}
	out, err := exec.Command(name, args...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// revision reads a build's VCS stamp: "03eb52f", "03eb52f-dirty", or "".
func revision(settings map[string]string) string {
	rev := settings["vcs.revision"]
	if rev == "" {
		return ""
	}
	if len(rev) > 7 {
		rev = rev[:7]
	}
	if settings["vcs.modified"] == "true" {
		rev += "-dirty"
	}
	return rev
}

// SkillDir finds the installed skill: dir if given, else the first of the project's
// .claude/skills or .agents/skills copy and the user's ~/.claude/skills copy that holds
// a SKILL.md. It returns "" when there is none.
func (e Env) SkillDir(dir string) string {
	cands := []string{dir}
	if dir == "" {
		cands = []string{
			filepath.Join(e.Cwd, ".claude", "skills", "bubbletea-tui"),
			filepath.Join(e.Cwd, ".agents", "skills", "bubbletea-tui"),
			filepath.Join(e.Cwd, "skills", "bubbletea-tui"),
		}
		if e.Home != "" {
			cands = append(cands, filepath.Join(e.Home, ".claude", "skills", "bubbletea-tui"))
		}
	}
	for _, c := range cands {
		if _, err := os.Stat(filepath.Join(c, "SKILL.md")); err == nil {
			if r, err := filepath.EvalSymlinks(c); err == nil {
				return r
			}
			return c
		}
	}
	return ""
}

var versionRe = regexp.MustCompile(`(?m)^\s+version:\s*"?([^"\s]+)"?\s*$`)

// SkillVersion says which skill the agent ran, from every source it can read: the version
// in SKILL.md's metadata; the commit, when the skill directory is the skill's own checkout
// (a plugin install or a clone), not a copy inside the user's repository, whose HEAD says
// nothing about the skill; skills-lock.json's computedHash in the project; and the commit
// these tools were built from.
func (e Env) SkillVersion(skillDir string) string {
	var parts []string
	if skillDir != "" {
		if b, err := os.ReadFile(filepath.Join(skillDir, "SKILL.md")); err == nil {
			if m := versionRe.FindSubmatch(b); m != nil {
				parts = append(parts, "v"+string(m[1]))
			}
		}
		if top := run("git", "-C", skillDir, "rev-parse", "--show-toplevel"); top != "" {
			if t, err := filepath.EvalSymlinks(top); err == nil && filepath.Join(t, "skills", "bubbletea-tui") == skillDir {
				if sha := run("git", "-C", skillDir, "rev-parse", "--short", "HEAD"); sha != "" {
					parts = append(parts, "git "+sha)
				}
			}
		}
	}
	if b, err := os.ReadFile(filepath.Join(e.Cwd, "skills-lock.json")); err == nil {
		var lock struct {
			Skills map[string]struct {
				ComputedHash string `json:"computedHash"`
			} `json:"skills"`
		}
		if json.Unmarshal(b, &lock) == nil {
			if h := lock.Skills["bubbletea-tui"].ComputedHash; h != "" {
				if len(h) > 12 {
					h = h[:12]
				}
				parts = append(parts, "computedHash "+h)
			}
		}
	}
	if rev := e.ownRevision(); rev != "" {
		parts = append(parts, "tools built from "+rev)
	}
	if len(parts) == 0 {
		return "unknown"
	}
	return strings.Join(parts, "; ")
}

func (e Env) ownRevision() string {
	if e.BuildInfo == nil {
		return ""
	}
	bi, ok := e.BuildInfo()
	if !ok {
		return ""
	}
	s := map[string]string{}
	for _, kv := range bi.Settings {
		s[kv.Key] = kv.Value
	}
	return revision(s)
}

// Environment is the go toolchain, the platform, and the build of each of the skill's
// tools the issue names (by binary name, anywhere in its fields): `go version -m` reads
// the commit a binary was built from.
func (e Env) Environment(values map[string]string) string {
	var lines []string
	if v := run("go", "version"); v != "" {
		lines = append(lines, v)
	} else {
		lines = append(lines, "go: not found; "+runtime.GOOS+"/"+runtime.GOARCH)
	}
	if rev := e.ownRevision(); rev != "" {
		lines = append(lines, "feedback built from "+rev)
	}
	var all strings.Builder
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		all.WriteString(values[k] + "\n")
	}
	for _, t := range knownTools {
		if t == "feedback" || !regexp.MustCompile(`\b`+t+`\b`).MatchString(all.String()) {
			continue
		}
		path, err := exec.LookPath(t)
		if err != nil {
			lines = append(lines, t+": not on PATH")
			continue
		}
		s := map[string]string{}
		for _, l := range strings.Split(run("go", "version", "-m", path), "\n") {
			f := strings.Fields(l)
			if len(f) == 2 && f[0] == "build" {
				if k, v, ok := strings.Cut(f[1], "="); ok {
					s[k] = v
				}
			}
		}
		if rev := revision(s); rev != "" {
			lines = append(lines, t+" built from "+rev)
		} else {
			lines = append(lines, t+": no VCS stamp (built outside a git checkout)")
		}
	}
	return strings.Join(lines, "\n")
}

// within reports whether path is dir or lies under it.
func within(path, dir string) bool {
	rel, err := filepath.Rel(dir, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

// ProjectRoot is the project the user is working in, wherever under it the tool runs:
// the git toplevel, else the nearest directory up from Cwd holding a TUI.md or a go.mod,
// else Cwd. It is "" when that is the home directory or holds it (a dotfiles repository,
// a go.mod in ~): the home is redacted as ~, and its name is the user's, not a project's.
func (e Env) ProjectRoot() string {
	if e.Cwd == "" {
		return ""
	}
	root := filepath.Clean(e.Cwd)
	if top := run("git", "-C", e.Cwd, "rev-parse", "--show-toplevel"); top != "" {
		root = filepath.Clean(top)
	} else {
	markers:
		for _, marker := range []string{"TUI.md", "go.mod"} {
			for d := filepath.Clean(e.Cwd); ; d = filepath.Dir(d) {
				if _, err := os.Stat(filepath.Join(d, marker)); err == nil {
					root = d
					break markers
				}
				if filepath.Dir(d) == d {
					break
				}
			}
		}
	}
	if len(root) <= 1 || (len(e.Home) > 1 && within(filepath.Clean(e.Home), root)) {
		return ""
	}
	return root
}

var moduleRe = regexp.MustCompile(`(?m)^module\s+"?([^"\s]+)"?`)

// modulePaths reads the module path of each go.mod from Cwd up to the project root.
func modulePaths(cwd, root string) []string {
	if root == "" || !within(filepath.Clean(cwd), root) {
		return nil
	}
	var out []string
	for d := filepath.Clean(cwd); ; d = filepath.Dir(d) {
		if b, err := os.ReadFile(filepath.Join(d, "go.mod")); err == nil {
			if m := moduleRe.FindSubmatch(b); m != nil {
				out = append(out, string(m[1]))
			}
		}
		if d == root || filepath.Dir(d) == d {
			break
		}
	}
	return out
}

// Redactor replaces what identifies the user or their project with placeholders before
// anything is shown or sent: the project root's path, its directory name and its Go
// module path; the home directory; the user name where it is a path segment
// (/home/<user>, /Users/<user>); and words the agent names (the app's name, a customer's),
// in any case and anywhere, inside a longer word too ("AcmeModel").
type Redactor struct {
	rules  []rule
	counts map[string]int
	extra  map[string]int // matches per --redact word
}

type rule struct {
	needle string
	with   string
	kind   int      // how a match is bounded: byPath, byWord or anywhere
	prefix []string // for a user name: the path segment that must precede it
	fold   bool     // match any case
	extra  string   // the --redact word this rule is for
}

const (
	anywhere = iota // a --redact word: inside a longer word too, over-redacting is the safe side
	byWord          // a project's directory name: not the middle of a longer word
	byPath          // a path or module: not the middle of a longer path
)

// isWordRune is a letter or digit in any script, or "_": what a word boundary separates.
func isWordRune(r rune) bool { return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) }

// isPathRune continues a path from the left: a word rune or . - ~ \. Not "/": a path
// after "file://" is still the path.
func isPathRune(r rune) bool { return isWordRune(r) || strings.ContainsRune(".-~\\", r) }

// NewRedactor builds the rules for e, plus one per extra word.
func NewRedactor(e Env, extra []string) *Redactor {
	r := &Redactor{counts: map[string]int{}, extra: map[string]int{}}
	root := e.ProjectRoot()
	var paths []rule
	if root != "" {
		paths = append(paths, rule{needle: root, with: "<project>", kind: byPath})
		if c := filepath.Clean(e.Cwd); !within(c, root) && len(c) > 1 && !within(filepath.Clean(e.Home), c) {
			paths = append(paths, rule{needle: c, with: "<project>", kind: byPath}) // a symlinked checkout
		}
	}
	if len(e.Home) > 1 {
		paths = append(paths, rule{needle: filepath.Clean(e.Home), with: "~", kind: byPath})
	}
	// longest first, so the project path wins over the home that holds it
	sort.SliceStable(paths, func(i, j int) bool { return len(paths[i].needle) > len(paths[j].needle) })
	r.rules = append(r.rules, paths...)
	mods := modulePaths(e.Cwd, root)
	sort.SliceStable(mods, func(i, j int) bool { return len(mods[i]) > len(mods[j]) })
	for _, m := range mods {
		if strings.Contains(m, "/") {
			r.rules = append(r.rules, rule{needle: m, with: "<module>", kind: byPath})
		} else if len([]rune(m)) >= 4 {
			r.rules = append(r.rules, rule{needle: m, with: "<module>", kind: byWord, fold: true})
		}
	}
	// the directory name, unless it is too short to be a name or is the user's
	if base := filepath.Base(root); root != "" && len([]rune(base)) >= 4 && !strings.EqualFold(base, e.User) {
		r.rules = append(r.rules, rule{needle: base, with: "<project>", kind: byWord, fold: true})
	}
	if e.User != "" {
		r.rules = append(r.rules, rule{needle: e.User, with: "<user>", kind: byPath,
			prefix: []string{"/home/", "/Users/", "\\Users\\", "/var/home/"}})
	}
	for _, w := range extra {
		if w = strings.TrimSpace(w); w != "" {
			r.extra[w] = 0
			r.rules = append(r.rules, rule{needle: w, with: "<redacted>", kind: anywhere, fold: true, extra: w})
		}
	}
	return r
}

// bounded reports whether s[i:j] stands alone. A path is not the middle of a longer path
// ("/root" in "kit/ui/root", "/home/ana" in "/home/anabel"); a "." after it is fine, as
// at the end of a sentence or before a Go symbol ("<module>.(*List).View"). A word is not
// the middle of a longer word in any script; an edge that is itself punctuation needs no
// boundary on that side.
func (ru rule) bounded(s string, i, j int) bool {
	before, _ := utf8.DecodeLastRuneInString(s[:i])
	after, _ := utf8.DecodeRuneInString(s[j:])
	switch ru.kind {
	case byPath:
		if i > 0 && ru.prefix == nil && isPathRune(before) {
			return false
		}
		return j == len(s) || !(isWordRune(after) || after == '-')
	case byWord:
		first, _ := utf8.DecodeRuneInString(ru.needle)
		last, _ := utf8.DecodeLastRuneInString(ru.needle)
		if i > 0 && isWordRune(first) && isWordRune(before) {
			return false
		}
		return j == len(s) || !isWordRune(last) || !isWordRune(after)
	}
	return true
}

// placeholderRe finds what an earlier rule wrote, so a later one can't match inside it
// (--redact project would otherwise turn <project> into <<redacted>>).
var placeholderRe = regexp.MustCompile(`<(?:project|module|user|redacted)>`)

// Apply redacts s and counts each replacement by its placeholder.
func (r *Redactor) Apply(s string) string {
	for _, ru := range r.rules {
		pat := regexp.QuoteMeta(ru.needle)
		if ru.prefix != nil {
			var ps []string
			for _, p := range ru.prefix {
				ps = append(ps, regexp.QuoteMeta(p))
			}
			pat = "(?:" + strings.Join(ps, "|") + ")" + pat
		}
		if ru.fold {
			pat = "(?i)" + pat
		}
		re := regexp.MustCompile(pat)
		held := placeholderRe.FindAllStringIndex(s, -1)
		var b strings.Builder
		last, n := 0, 0
	match:
		for _, m := range re.FindAllStringIndex(s, -1) {
			for _, h := range held {
				if m[0] < h[1] && h[0] < m[1] {
					continue match
				}
			}
			if !ru.bounded(s, m[0], m[1]) {
				continue
			}
			b.WriteString(s[last:m[0]])
			if ru.prefix != nil {
				b.WriteString(s[m[0] : m[1]-len(ru.needle)]) // keep /home/
			}
			b.WriteString(ru.with)
			last = m[1]
			n++
		}
		if n == 0 {
			continue
		}
		b.WriteString(s[last:])
		s = b.String()
		r.counts[ru.with] += n
		if ru.extra != "" {
			r.extra[ru.extra] += n
		}
	}
	return s
}

// Unmatched lists the --redact words that matched nothing, in the order given: a name
// spelled differently in the text than on the command line leaks, so say so.
func (r *Redactor) Unmatched(order []string) []string {
	var out []string
	for _, w := range order {
		w = strings.TrimSpace(w)
		if n, ok := r.extra[w]; ok && n == 0 {
			out = append(out, w)
			r.extra[w] = -1 // report each once
		}
	}
	return out
}

// Report says what was replaced, for the agent to tell the user: "redacted 3 × <project>,
// 1 × ~", or "" when nothing was.
func (r *Redactor) Report() string {
	if len(r.counts) == 0 {
		return ""
	}
	var parts []string
	for _, with := range []string{"<project>", "<module>", "~", "<user>", "<redacted>"} {
		if n := r.counts[with]; n > 0 {
			parts = append(parts, fmt.Sprintf("%d × %s", n, with))
		}
	}
	return "redacted " + strings.Join(parts, ", ")
}
