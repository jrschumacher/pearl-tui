package cmddoc

import (
	"regexp"
	"sort"
	"strings"
)

// Invocation is one tool command found in prose: a backticked span, or a line of a
// fenced code block, whose first word is a tool's name.
type Invocation struct {
	File string
	Line int
	Tool string   // the tool's name, without a directory (wordmark.py, not scripts/wordmark.py)
	Args []string // the words after it, brackets and trailing punctuation stripped
	Text string   // the span as written
}

// Spec is a command's documented interface: its mode words (none for a command without
// modes), the flags of the bare command, and the flags of each mode. A flag maps to its
// choices, nil when it takes anything or nothing.
type Spec struct {
	Modes     []string
	Flags     map[string][]string
	ModeFlags map[string]map[string][]string
}

var (
	fenceRe    = regexp.MustCompile("(?s)```.*?```")
	spanRe     = regexp.MustCompile("`([^`\n]*(?:\n[^`\n]*)?)`")
	flagRe     = regexp.MustCompile(`(^|[\s\[(|/])(--?[a-zA-Z][a-zA-Z0-9.-]*)`)
	argparseRe = regexp.MustCompile(`add_argument\(\s*"(--[a-z0-9-]+)"([^)]*)\)`)
	choicesRe  = regexp.MustCompile(`choices=\[([^\]]*)\]`)
	quotedRe   = regexp.MustCompile(`"([^"]*)"`)
)

// Invocations finds every invocation of one of tools in a Markdown text: backticked
// spans (which may wrap one line) outside fenced blocks whose first word names a tool,
// and each line inside a fenced block whose first word does, after any step label
// ("3. LINT     tuilint …", as verify.md writes the loop). A tool's name elsewhere in
// a span or a line (a tree listing, a template's "palette / tone") is prose, not a
// command.
func Invocations(file, text string, tools []string) []Invocation {
	isTool := map[string]bool{}
	for _, t := range tools {
		isTool[t] = true
	}
	var out []Invocation
	add := func(line int, span string) {
		words := strings.Fields(span)
		for i, w := range words {
			name := w[strings.LastIndex(w, "/")+1:]
			if !isTool[name] {
				if isLabel(w) {
					continue
				}
				return
			}
			inv := Invocation{File: file, Line: line, Tool: name, Text: span}
			for _, a := range words[i+1:] {
				if a = cleanArg(a); a != "" {
					inv.Args = append(inv.Args, a)
				}
			}
			out = append(out, inv)
			return
		}
	}
	lineAt := func(off int) int { return 1 + strings.Count(text[:off], "\n") }
	// fenced blocks: one invocation per line, from the tool's name
	prose := []byte(text)
	for _, m := range fenceRe.FindAllStringIndex(text, -1) {
		block := text[m[0]:m[1]]
		for i, l := range strings.Split(block, "\n") {
			add(lineAt(m[0])+i, l)
		}
		for i := m[0]; i < m[1]; i++ {
			if prose[i] != '\n' {
				prose[i] = ' '
			}
		}
	}
	for _, m := range spanRe.FindAllStringSubmatchIndex(string(prose), -1) {
		add(lineAt(m[0]), text[m[2]:m[3]])
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Line < out[j].Line })
	return out
}

// isLabel reports a word that may precede the tool's name in a command: a step label a
// fenced loop line starts with ("3.", "LINT"), or the interpreter a script runs under
// ("python3 scripts/wordmark.py …" is still an invocation of wordmark.py).
func isLabel(w string) bool {
	if w == "python" || w == "python3" || w == "-I" {
		return true
	}
	if strings.HasSuffix(w, ".") && len(w) > 1 {
		w = w[:len(w)-1]
		for _, r := range w {
			if r < '0' || r > '9' {
				return false
			}
		}
		return true
	}
	for _, r := range w {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return w != ""
}

// cleanArg strips the brackets and punctuation prose wraps an argument in ("[--cols",
// "H]", "out.png)," "N."), keeping "." and "./..." as the paths they are.
func cleanArg(a string) string {
	a = strings.TrimLeft(a, "[(")
	a = strings.TrimRight(a, "])>,;:")
	if strings.HasSuffix(a, ".") && !strings.HasSuffix(a, "..") && a != "." {
		a = a[:len(a)-1]
	}
	return a
}

// FlagsIn lists the flags a usage or doc text names: every --word, and every -word for
// single-dash tools, each once.
func FlagsIn(doc string) []string {
	seen := map[string]bool{}
	var out []string
	for _, m := range flagRe.FindAllStringSubmatch(doc, -1) {
		if f := m[2]; !seen[f] {
			seen[f] = true
			out = append(out, f)
		}
	}
	return out
}

// ArgparseFlags reads the flags of a Python script from its add_argument calls, with
// the choices of each flag that has them. That is what its --help prints, without
// needing Python to run.
func ArgparseFlags(src string) map[string][]string {
	out := map[string][]string{}
	for _, m := range argparseRe.FindAllStringSubmatch(src, -1) {
		var choices []string
		if c := choicesRe.FindStringSubmatch(m[2]); c != nil {
			for _, q := range quotedRe.FindAllStringSubmatch(c[1], -1) {
				choices = append(choices, q[1])
			}
		}
		out[m[1]] = choices
	}
	return out
}

// Free turns a flag list into a Spec.Flags map with no choices.
func Free(flags []string) map[string][]string {
	out := map[string][]string{}
	for _, f := range flags {
		out[f] = nil
	}
	return out
}

// Check reports what in inv the spec doesn't document: a mode word the command lacks, a
// flag it (or its mode) lacks, or a choice a flag doesn't take. A value written as
// `a|b|c` is each of its words; placeholders (`<…>`, `…`, `N`, a quoted string) pass.
func (s Spec) Check(inv Invocation) []string {
	var bad []string
	flags := s.Flags
	args := inv.Args
	if len(s.Modes) > 0 && len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		mode := args[0]
		mf, ok := s.ModeFlags[mode]
		if !ok {
			// "palette <the hues…>", "palette warm, tinted neutrals": prose that opens
			// with the tool's name is a command only when it also carries a flag or a
			// path-like argument; a plain word followed by plain words is a sentence
			if !isWord(mode) || !commandLike(args[1:]) {
				return nil
			}
			return []string{"unknown mode " + mode + " (one of " + strings.Join(s.Modes, "|") + ")"}
		}
		flags = mf
		args = args[1:]
	}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "-") || a == "-" || a == "--" {
			continue
		}
		name, val, hasVal := strings.Cut(a, "=")
		choices, ok := flags[name]
		if !ok {
			bad = append(bad, "unknown flag "+name)
			continue
		}
		if choices == nil {
			continue
		}
		if !hasVal {
			if i+1 >= len(args) {
				continue
			}
			i++
			val = args[i]
		}
		for _, v := range strings.Split(val, "|") {
			if v == "" || !isWord(v) {
				continue // a placeholder
			}
			found := false
			for _, c := range choices {
				found = found || c == v
			}
			if !found {
				bad = append(bad, name+" "+v+" is not one of "+strings.Join(choices, "|"))
			}
		}
	}
	return bad
}

// commandLike reports arguments that read as a command line rather than a sentence: a
// flag, or something with a path separator, a dot or a glob in it.
func commandLike(args []string) bool {
	for _, a := range args {
		if strings.HasPrefix(a, "-") || strings.ContainsAny(a, "/.*") {
			return true
		}
	}
	return false
}

// isWord reports a plain lowercase word, the only thing a choice can be.
func isWord(s string) bool {
	for _, r := range s {
		if r < 'a' || r > 'z' {
			return false
		}
	}
	return s != ""
}
