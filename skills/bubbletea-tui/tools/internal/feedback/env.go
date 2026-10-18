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
	"slices"
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

// realPath resolves p's symlinks, or returns p cleaned when it can't.
func realPath(p string) string {
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return r
	}
	return filepath.Clean(p)
}

// ProjectRoot is the project the user is working in, wherever under it the tool runs:
// the git toplevel, else the nearest directory up from Cwd holding a TUI.md or a go.mod,
// else Cwd, with symlinks resolved. It is "" when that is the home directory or holds it
// (a dotfiles repository, a go.mod in ~): the home is redacted as ~, and its name is the
// user's, not a project's.
func (e Env) ProjectRoot() string {
	root, _ := e.projectRoots()
	return root
}

// projectRoots is ProjectRoot, and alias, the root as Cwd reaches it when that is through
// a symlink (~/acme/internal/ui, with ~/acme a link to ~/src/acme-billing), else "".
func (e Env) projectRoots() (root, alias string) {
	if e.Cwd == "" {
		return "", ""
	}
	cwd := filepath.Clean(e.Cwd)
	real := realPath(cwd)
	root = real
	if top := run("git", "-C", real, "rev-parse", "--show-toplevel"); top != "" {
		root = realPath(top)
	} else {
	markers:
		for _, marker := range []string{"TUI.md", "go.mod"} {
			for d := real; ; d = filepath.Dir(d) {
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
	home := ""
	if len(e.Home) > 1 {
		home = filepath.Clean(e.Home)
	}
	holdsHome := func(dir string) bool {
		return home != "" && (within(home, dir) || within(realPath(home), dir))
	}
	if len(root) <= 1 || holdsHome(root) {
		return "", ""
	}
	if real != cwd && within(real, root) {
		rel, _ := filepath.Rel(root, real)
		alias = cwd
		for _, part := range strings.Split(rel, string(filepath.Separator)) {
			if part != "." {
				alias = filepath.Dir(alias)
			}
		}
		if alias == root || len(alias) <= 1 || holdsHome(alias) {
			alias = ""
		}
	}
	return root, alias
}

var moduleRe = regexp.MustCompile(`(?m)^module\s+"?([^"\s]+)"?`)

// modulePaths reads the module path of every go.mod under root (vendor, testdata and
// dot-directories aside) and of every module a go.work there uses, in or out of root.
func modulePaths(root string) []string {
	if root == "" {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	read := func(dir string) {
		b, err := os.ReadFile(filepath.Join(dir, "go.mod"))
		if err != nil {
			return
		}
		if m := moduleRe.FindSubmatch(b); m != nil && !seen[string(m[1])] {
			seen[string(m[1])] = true
			out = append(out, string(m[1]))
		}
	}
	visited := 0
	filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if visited++; err != nil || visited > 20000 {
			return filepath.SkipDir // unreadable, or a tree too big to walk on every draft
		}
		if d.IsDir() {
			if n := d.Name(); p != root && (strings.HasPrefix(n, ".") || n == "vendor" || n == "testdata" || n == "node_modules") {
				return filepath.SkipDir
			}
			return nil
		}
		switch d.Name() {
		case "go.mod":
			read(filepath.Dir(p))
		case "go.work":
			b, _ := os.ReadFile(p)
			for _, u := range goWorkUses(string(b)) {
				read(filepath.Join(filepath.Dir(p), u))
			}
		}
		return nil
	})
	return out
}

// goWorkUses lists the directories a go.work's use directives name, in either form
// ("use ./a" or a "use ( … )" block).
func goWorkUses(work string) []string {
	var out []string
	block := false
	for _, line := range strings.Split(work, "\n") {
		line, _, _ = strings.Cut(line, "//")
		f := strings.Fields(line)
		switch {
		case len(f) == 0:
		case block && f[0] == ")":
			block = false
		case block:
			out = append(out, strings.Trim(f[0], `"`))
		case f[0] == "use" && len(f) > 1 && f[1] == "(":
			block = true
		case f[0] == "use" && len(f) > 1:
			out = append(out, strings.Trim(f[1], `"`))
		}
	}
	return out
}

// moduleOwner is a module path's host and owner ("github.com/acme-corp" for
// github.com/acme-corp/acme-billing), or "" when it has no host.
func moduleOwner(m string) string {
	parts := strings.Split(m, "/")
	if len(parts) < 3 || !strings.Contains(parts[0], ".") {
		return ""
	}
	return parts[0] + "/" + parts[1]
}

// Redactor replaces what identifies the user or their project with placeholders before
// anything is shown or sent: the project root's path and directory name (and the
// symlink's, when the tool runs through one); the path of every Go module in it or in
// its go.work, and each one's host and owner (github.com/acme-corp); the home directory;
// the user name where it is a path segment (/home/<user>, /Users/<user>); and words the
// agent names (the app's name, a customer's), in any case and script. A word of four or
// more letters matches inside a longer word too ("AcmeModel", "acmeops"), and Inside
// says where; a shorter one only as a word of its own, where an identifier's case change
// counts as a word's edge ("Zoë" in "ZoëModel", never "ui" in "build").
type Redactor struct {
	rules  []rule
	counts map[string]int
	extra  map[string]int // matches per --redact word
	inside []string       // "Lint" matched inside "tuilint", once each
}

type rule struct {
	needle string
	with   string
	kind   int      // how a match is bounded: byPath, byWord, byIdent or anywhere
	prefix []string // for a user name: the path segment that must precede it
	fold   bool     // match any case
	extra  string   // the --redact word this rule is for
}

const (
	anywhere = iota // a --redact word of 4+ letters: inside a longer word too, the safe side
	byIdent         // a shorter --redact word: a word or an identifier's part of its own
	byWord          // a project's directory name: not the middle of a longer word
	byPath          // a path or module: not the middle of a longer path
)

// minInside is the fewest letters a --redact word needs to match inside a longer word:
// "ui" inside "build" and "guidance" would wreck the report and hide nothing.
const minInside = 4

// edgeAt reports whether a word can begin or end at byte k of s: at either end of s, next
// to a rune that isn't a letter or digit, or at an identifier's case change ("zoë|Model",
// "UI|List").
func edgeAt(s string, k int) bool {
	if k <= 0 || k >= len(s) {
		return true
	}
	a, _ := utf8.DecodeLastRuneInString(s[:k])
	b, n := utf8.DecodeRuneInString(s[k:])
	if !isWordRune(a) || !isWordRune(b) {
		return true
	}
	if unicode.IsLower(a) && unicode.IsUpper(b) {
		return true
	}
	c, _ := utf8.DecodeRuneInString(s[k+n:])
	return unicode.IsUpper(a) && unicode.IsUpper(b) && unicode.IsLower(c)
}

// wordAround is the whole word holding s[i:j], for saying where a match landed.
func wordAround(s string, i, j int) string {
	for i > 0 {
		r, n := utf8.DecodeLastRuneInString(s[:i])
		if !isWordRune(r) {
			break
		}
		i -= n
	}
	for j < len(s) {
		r, n := utf8.DecodeRuneInString(s[j:])
		if !isWordRune(r) {
			break
		}
		j += n
	}
	return s[i:j]
}

// isWordRune is a letter or digit in any script, or "_": what a word boundary separates.
func isWordRune(r rune) bool { return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) }

// isPathRune continues a path from the left: a word rune or . - ~ \. Not "/": a path
// after "file://" is still the path.
func isPathRune(r rune) bool { return isWordRune(r) || strings.ContainsRune(".-~\\", r) }

// NewRedactor builds the rules for e, plus one per extra word.
func NewRedactor(e Env, extra []string) *Redactor {
	r := &Redactor{counts: map[string]int{}, extra: map[string]int{}}
	root, alias := e.projectRoots()
	var paths []rule
	for _, p := range []string{root, alias} {
		if p != "" {
			paths = append(paths, rule{needle: p, with: "<project>", kind: byPath})
		}
	}
	if len(e.Home) > 1 {
		paths = append(paths, rule{needle: filepath.Clean(e.Home), with: "~", kind: byPath})
	}
	// longest first, so the project path wins over the home that holds it
	sort.SliceStable(paths, func(i, j int) bool { return len(paths[i].needle) > len(paths[j].needle) })
	r.rules = append(r.rules, paths...)
	// every module path, longest first, then what is left of each: its host and owner
	mods := modulePaths(root)
	sort.SliceStable(mods, func(i, j int) bool { return len(mods[i]) > len(mods[j]) })
	var owners []string
	for _, m := range mods {
		if strings.Contains(m, "/") {
			r.rules = append(r.rules, rule{needle: m, with: "<module>", kind: byPath})
		} else if len([]rune(m)) >= 4 {
			r.rules = append(r.rules, rule{needle: m, with: "<module>", kind: byWord, fold: true})
		}
		if o := moduleOwner(m); o != "" && !slices.Contains(owners, o) {
			owners = append(owners, o)
		}
	}
	for _, o := range owners {
		r.rules = append(r.rules, rule{needle: o, with: "<owner>", kind: byPath})
	}
	// the directory names, whole words only, unless too short to be a name or the user's
	for _, p := range []string{root, alias} {
		if base := filepath.Base(p); p != "" && len([]rune(base)) >= 4 && !strings.EqualFold(base, e.User) {
			r.rules = append(r.rules, rule{needle: base, with: "<project>", kind: byWord, fold: true})
		}
	}
	if e.User != "" {
		r.rules = append(r.rules, rule{needle: e.User, with: "<user>", kind: byPath,
			prefix: []string{"/home/", "/Users/", "\\Users\\", "/var/home/"}})
	}
	for _, w := range extra {
		if w = strings.TrimSpace(w); w != "" {
			r.extra[w] = 0
			kind := anywhere
			if len([]rune(w)) < minInside {
				kind = byIdent
			}
			r.rules = append(r.rules, rule{needle: w, with: "<redacted>", kind: kind, fold: true, extra: w})
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
	case byIdent:
		return edgeAt(s, i) && edgeAt(s, j)
	}
	return true
}

// placeholderRe finds what an earlier rule wrote, so a later one can't match inside it
// (--redact project would otherwise turn <project> into <<redacted>>).
var placeholderRe = regexp.MustCompile(`<(?:project|module|owner|user|redacted)>`)

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
			if ru.kind == anywhere && !(edgeAt(s, m[0]) && edgeAt(s, m[1])) {
				note := fmt.Sprintf("%q matched inside %q", ru.extra, wordAround(s, m[0], m[1]))
				if !slices.Contains(r.inside, note) {
					r.inside = append(r.inside, note)
				}
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

// Inside lists each --redact word that matched inside a longer word, with that word:
// `"Lint" matched inside "tuilint"`. The match is redacted; the agent checks the draft
// still reads, or passes a longer word.
func (r *Redactor) Inside() []string { return r.inside }

// Report says what was replaced, for the agent to tell the user: "redacted 3 × <project>,
// 1 × ~", or "" when nothing was.
func (r *Redactor) Report() string {
	if len(r.counts) == 0 {
		return ""
	}
	var parts []string
	for _, with := range []string{"<project>", "<module>", "<owner>", "~", "<user>", "<redacted>"} {
		if n := r.counts[with]; n > 0 {
			parts = append(parts, fmt.Sprintf("%d × %s", n, with))
		}
	}
	return "redacted " + strings.Join(parts, ", ")
}
