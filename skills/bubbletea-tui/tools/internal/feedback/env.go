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

// Redactor replaces what identifies the user or their project with placeholders before
// anything is shown or sent: the project's absolute path, the home directory, the user
// name, and words the agent names (the app's name, a customer's).
type Redactor struct {
	rules  []rule
	counts map[string]int
}

type rule struct {
	re   *regexp.Regexp
	with string
}

// NewRedactor builds the rules for e, plus one per extra word.
func NewRedactor(e Env, extra []string) *Redactor {
	r := &Redactor{counts: map[string]int{}}
	add := func(re, with string) { r.rules = append(r.rules, rule{regexp.MustCompile(re), with}) }
	// paths first, longest first, so the project path wins over the home that holds it
	var paths []string
	for _, p := range []string{e.Cwd, e.Home} {
		if len(p) > 1 {
			paths = append(paths, filepath.Clean(p))
		}
	}
	sort.Slice(paths, func(i, j int) bool { return len(paths[i]) > len(paths[j]) })
	for _, p := range paths {
		with := "~"
		if p == filepath.Clean(e.Cwd) {
			with = "<project>"
		}
		add(regexp.QuoteMeta(p)+`\b`, with)
	}
	if len(e.User) >= 3 {
		add(`\b`+regexp.QuoteMeta(e.User)+`\b`, "<user>")
	}
	for _, w := range extra {
		if w = strings.TrimSpace(w); w != "" {
			add(`(?i)\b`+regexp.QuoteMeta(w)+`\b`, "<redacted>")
		}
	}
	return r
}

// Apply redacts s and counts each replacement by its placeholder.
func (r *Redactor) Apply(s string) string {
	for _, ru := range r.rules {
		n := len(ru.re.FindAllStringIndex(s, -1))
		if n > 0 {
			r.counts[ru.with] += n
			s = ru.re.ReplaceAllLiteralString(s, ru.with)
		}
	}
	return s
}

// Report says what was replaced, for the agent to tell the user: "redacted 3 × <project>,
// 1 × ~", or "" when nothing was.
func (r *Redactor) Report() string {
	if len(r.counts) == 0 {
		return ""
	}
	var parts []string
	for _, with := range []string{"<project>", "~", "<user>", "<redacted>"} {
		if n := r.counts[with]; n > 0 {
			parts = append(parts, fmt.Sprintf("%d × %s", n, with))
		}
	}
	return "redacted " + strings.Join(parts, ", ")
}
