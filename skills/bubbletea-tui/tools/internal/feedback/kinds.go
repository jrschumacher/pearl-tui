// Package feedback drafts and files issues on the skill's repository: the agent's half of
// the loop in AGENTS.md § How the skill improves. The kinds and their fields mirror the
// issue forms in .github/ISSUE_TEMPLATE (a test holds them together), so an issue filed
// through gh reads exactly like one filed from the form.
package feedback

import (
	"fmt"
	"sort"
	"strings"
)

// DefaultRepo is where feedback goes unless --repo or FEEDBACK_REPO names a fork.
const DefaultRepo = "jrschumacher/skill-bubbletea-tui"

// Field is one field of an issue form, keyed by its id (the URL query parameter that
// prefills it, and the --field key).
type Field struct {
	ID       string
	Label    string
	Type     string   // input, textarea or dropdown
	Options  []string // a dropdown's choices
	Render   string   // a textarea's code language: its value is fenced in the body
	Required bool
	Auto     bool // filled by the tool when not given (skill-version, environment)
}

// Kind is one issue form.
type Kind struct {
	Name     string   // the --kind value
	Aliases  []string // other --kind values
	Template string   // the form's file name in .github/ISSUE_TEMPLATE
	Labels   []string
	Fields   []Field
}

var (
	asked = Field{ID: "asked", Label: "What was asked", Type: "textarea", Required: true}
	skill = Field{ID: "skill-version", Label: "Skill version", Type: "input", Required: true, Auto: true}
	env   = Field{ID: "environment", Label: "Environment", Type: "textarea", Auto: true}
)

// Kinds are the four forms, in the order -h lists them.
var Kinds = []Kind{
	{
		Name: "bug", Template: "bug.yml", Labels: []string{"feedback", "bug"},
		Fields: []Field{
			asked, skill,
			{ID: "area", Label: "Where", Type: "input", Required: true},
			{ID: "variant", Label: "Variant", Type: "input"},
			{ID: "happened", Label: "What happened", Type: "textarea", Required: true},
			{ID: "expected", Label: "What should have happened", Type: "textarea", Required: true},
			{ID: "miss-class", Label: "Kind of miss", Type: "dropdown", Required: true, Options: []string{
				"(a) the skill never says it",
				"(b) the skill says it and the agent ignored it",
				"(c) prose can't express it",
				"not sure",
			}},
			{ID: "evidence", Label: "Frames, sheets or a minimal repro", Type: "textarea"},
			{ID: "loop", Label: "Did the agent run the loop?", Type: "dropdown", Options: []string{"yes", "partly", "no"}},
			env,
		},
	},
	{
		Name: "perf", Aliases: []string{"performance"}, Template: "performance.yml", Labels: []string{"feedback", "performance"},
		Fields: []Field{
			asked, skill,
			{ID: "area", Label: "Tool or code path", Type: "input", Required: true},
			{ID: "command", Label: "Command", Type: "textarea", Render: "shell", Required: true},
			{ID: "timing", Label: "Timing", Type: "textarea", Required: true},
			{ID: "sizes", Label: "Sizes", Type: "input", Required: true},
			{ID: "expected", Label: "Fast enough would be", Type: "textarea"},
			{ID: "evidence", Label: "Profile or repro", Type: "textarea"},
			env,
		},
	},
	{
		Name: "eval", Aliases: []string{"eval-suggestion"}, Template: "eval-suggestion.yml", Labels: []string{"feedback", "eval"},
		Fields: []Field{
			{ID: "case", Label: "Case", Type: "input", Required: true},
			{ID: "prompt", Label: "Prompt", Type: "textarea"},
			{ID: "mistake", Label: "The mistake it catches", Type: "textarea", Required: true},
			{ID: "assertion", Label: "Assertion", Type: "textarea", Required: true},
			{ID: "fixtures", Label: "Fixtures", Type: "textarea"},
			{ID: "skill-version", Label: "Skill version", Type: "input", Auto: true},
		},
	},
	{
		Name: "feature", Aliases: []string{"enhancement"}, Template: "feature.yml", Labels: []string{"feedback", "enhancement"},
		Fields: []Field{
			asked,
			{ID: "area", Label: "Area", Type: "dropdown", Required: true, Options: []string{
				"skill (SKILL.md, references)",
				"kit (theme, ui, showcase)",
				"themes",
				"tools",
				"scripts",
				"evals",
			}},
			{ID: "workaround", Label: "What the agent did instead", Type: "textarea", Required: true},
			{ID: "proposal", Label: "Proposal", Type: "textarea", Required: true},
			{ID: "evidence", Label: "Example or repro", Type: "textarea"},
			skill, env,
		},
	},
}

// KindNamed finds a kind by its name or an alias.
func KindNamed(name string) (Kind, error) {
	var names []string
	for _, k := range Kinds {
		if k.Name == name {
			return k, nil
		}
		for _, a := range k.Aliases {
			if a == name {
				return k, nil
			}
		}
		names = append(names, k.Name)
	}
	return Kind{}, fmt.Errorf("unknown kind %q (one of %s)", name, strings.Join(names, "|"))
}

// Field returns the kind's field with id, and whether it has one.
func (k Kind) Field(id string) (Field, bool) {
	for _, f := range k.Fields {
		if f.ID == id {
			return f, true
		}
	}
	return Field{}, false
}

// IDs lists the kind's field ids in form order.
func (k Kind) IDs() []string {
	out := make([]string, len(k.Fields))
	for i, f := range k.Fields {
		out[i] = f.ID
	}
	return out
}

// Choose maps a value onto one of a dropdown's options: the option itself, its leading
// "(a)" tag or the bare letter ("b" for "(b) the skill says it…"), or a unique
// case-insensitive prefix. Anything else is an error naming the options.
func (f Field) Choose(v string) (string, error) {
	if f.Type != "dropdown" {
		return v, nil
	}
	lv := strings.ToLower(strings.TrimSpace(v))
	var prefix []string
	for _, o := range f.Options {
		lo := strings.ToLower(o)
		tag, _, _ := strings.Cut(lo, " ")
		if lo == lv || (strings.HasPrefix(tag, "(") && (tag == lv || tag == "("+lv+")")) {
			return o, nil
		}
		if lv != "" && strings.HasPrefix(lo, lv) {
			prefix = append(prefix, o)
		}
	}
	if len(prefix) == 1 {
		return prefix[0], nil
	}
	quoted := make([]string, len(f.Options))
	for i, o := range f.Options {
		quoted[i] = fmt.Sprintf("%q", o)
	}
	return "", fmt.Errorf("--field %s=%q is not one of %s", f.ID, v, strings.Join(quoted, ", "))
}

// fieldList is the field summary -h and errors print: "asked*, skill-version*, area*, …",
// a star on each required field.
func (k Kind) fieldList() string {
	var out []string
	for _, f := range k.Fields {
		s := f.ID
		if f.Required {
			s += "*"
		}
		out = append(out, s)
	}
	return strings.Join(out, ", ")
}

// knownTools are the binaries whose build the environment field reports when a field
// names them.
var knownTools = func() []string {
	t := []string{"cellshot", "tuilint", "tuivet", "palette", "diversity", "feedback"}
	sort.Strings(t)
	return t
}()
