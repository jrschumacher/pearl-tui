package speclint

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/width"

	"tuitools/internal/cmddoc"
)

//go:embed doc.go
var docSrc string

// Usage is the -h text: the package doc comment.
func Usage() string { return cmddoc.Text(docSrc) + "\n" }

// Severity maps each rule ID to P1 (fails the run) or P2 (warns).
var Severity = map[string]string{
	"S-TABLE": "P1", "S-KEYAMBIG": "P1", "S-KEYDUP": "P1",
	"S-KEYSHADOW": "P2", "S-GLYPHWIDTH": "P2",
}

// Finding is one result, in the shape the other tuilint modes write to --json.
type Finding struct {
	Rule string `json:"rule"`
	Sev  string `json:"sev"`
	File string `json:"file"`
	Line int    `json:"line"`
	Msg  string `json:"msg"`
}

// Main runs `tuilint spec` with its arguments and returns the exit status.
func Main(args []string, stdout, stderr io.Writer) int {
	var files []string
	jsonOut := ""
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-h" || a == "--help":
			fmt.Fprint(stdout, Usage())
			return 0
		case a == "--json" || a == "-json":
			if i+1 >= len(args) {
				fmt.Fprintf(stderr, "tuilint spec: %s needs a value\n", a)
				return 2
			}
			i++
			jsonOut = args[i]
		case strings.HasPrefix(a, "--json=") || strings.HasPrefix(a, "-json="):
			_, jsonOut, _ = strings.Cut(a, "=")
		case strings.HasPrefix(a, "-"):
			fmt.Fprintf(stderr, "tuilint spec: unknown flag %s\n%s", a, Usage())
			return 2
		default:
			files = append(files, a)
		}
	}
	if len(files) == 0 {
		files = []string{"TUI.md"}
	}
	var findings []Finding
	for _, f := range files {
		src, err := os.ReadFile(f)
		if err != nil {
			fmt.Fprintf(stderr, "tuilint spec: %v\n", err)
			return 2
		}
		findings = append(findings, Lint(f, string(src))...)
	}
	fails := 0
	for _, f := range findings {
		fmt.Fprintf(stdout, "[%s] %-12s %s:%d  %s\n", f.Sev, f.Rule, f.File, f.Line, f.Msg)
		if f.Sev != "P2" {
			fails++
		}
	}
	verdict := "OK"
	if fails > 0 {
		verdict = "FAIL"
	}
	fmt.Fprintf(stdout, "%s  %d blocking, %d warnings\n", verdict, fails, len(findings)-fails)
	if jsonOut != "" {
		if findings == nil {
			findings = []Finding{}
		}
		b, _ := json.MarshalIndent(findings, "", " ")
		if err := os.WriteFile(jsonOut, append(b, '\n'), 0o644); err != nil {
			fmt.Fprintf(stderr, "tuilint spec: %v\n", err)
			return 2
		}
	}
	if fails > 0 {
		return 1
	}
	return 0
}

// binding is one `key action` item of a keys cell.
type binding struct {
	token  string // as written: "shift+enter/ctrl+j"
	keys   []key
	action string // lowercased
	bad    string // the first alternative that names no key, as written
}

// row is one state of the table, or a Global line.
type row struct {
	line          int
	state, screen string
	keys          string // the keys cell as written
	bindings      []binding
	global        bool
}

type spec struct {
	file    string
	lines   []string
	rows    []row // states, in order
	globals []row
}

// Lint checks one TUI.md. file names it in the findings.
func Lint(file, src string) []Finding {
	s := parse(file, src)
	var out []Finding
	add := func(rule string, line int, format string, a ...any) {
		out = append(out, Finding{Rule: rule, Sev: Severity[rule], File: file, Line: line, Msg: fmt.Sprintf(format, a...)})
	}
	if len(s.rows) == 0 {
		add("S-TABLE", 1, "no States-and-keys table: write `| state (harness name) | screen / overlay | keys |` with keys as `key action · key action`, as cmd-init.md's template does; the key rules can't run without it")
	}
	for _, r := range append(append([]row{}, s.globals...), s.rows...) {
		for _, b := range r.bindings {
			if b.bad != "" {
				out = append(out, Finding{Rule: "S-TABLE", Sev: "P2", File: file, Line: r.line, Msg: fmt.Sprintf(
					"%s: can't read key %q in `%s %s`, so the key rules skip it. Write keys as Bubble Tea names (ctrl+k, shift+tab, enter, pgdown, ↑↓, j/k), each item `key action`, items separated by ·",
					r.state, b.bad, b.token, b.action)})
			}
		}
	}
	s.keyAmbig(add)
	s.keyDup(add)
	s.keyShadow(add)
	s.glyphWidth(add)
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Sev != b.Sev {
			return a.Sev < b.Sev
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.Rule < b.Rule
	})
	return out
}

var (
	globalRe   = regexp.MustCompile(`(?i)^global(?:\s+keys)?\s*(?:\([^)]*\))?\s*:\s*(.+)$`)
	globalCell = regexp.MustCompile(`(?i)^\(?(?:global|all|any|all states|\*)\)?$`)
	sepCell    = regexp.MustCompile(`^:?-+:?$`)
	listMark   = regexp.MustCompile(`^(?:[-*+]|\d+\.)\s+`)
)

func parse(file, src string) *spec {
	s := &spec{file: file, lines: strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n")}
	stateCol, screenCol, keysCol := -1, -1, -1
	inTable := false
	// A Global line counts only after a States-and-keys table and before the next
	// heading at its section's level or above, so "- Global: g is reserved…" under
	// § Decisions is prose, not a binding.
	level, globalsUntil := 0, -1
	for i, raw := range s.lines {
		t := strings.TrimSpace(raw)
		if m := headingRe.FindStringSubmatch(t); m != nil {
			n := len(m[1])
			if globalsUntil >= 0 && n <= globalsUntil {
				globalsUntil = -1
			}
			level, inTable = n, false
			continue
		}
		if !strings.HasPrefix(t, "|") {
			inTable = false
			if globalsUntil < 0 {
				continue
			}
			if m := globalRe.FindStringSubmatch(clean(listMark.ReplaceAllString(t, ""))); m != nil {
				s.globals = append(s.globals, row{line: i + 1, state: "Global", global: true, bindings: parseGlobalKeys(m[1])})
			}
			continue
		}
		cells := splitRow(t)
		if !inTable {
			// a header row followed by a |---| row
			st, sc, kc := -1, -1, -1
			for j, c := range cells {
				lc := strings.ToLower(c)
				switch {
				case st < 0 && strings.Contains(lc, "state"):
					st = j
				case kc < 0 && strings.Contains(lc, "key"):
					kc = j
				case sc < 0 && (strings.Contains(lc, "screen") || strings.Contains(lc, "overlay")):
					sc = j
				}
			}
			if st >= 0 && kc >= 0 && i+1 < len(s.lines) && isSep(splitRow(strings.TrimSpace(s.lines[i+1]))) {
				stateCol, screenCol, keysCol, inTable = st, sc, kc, true
				globalsUntil = level
				if level == 0 {
					globalsUntil = 6
				}
			}
			continue
		}
		if isSep(cells) {
			continue
		}
		get := func(j int) string {
			if j >= 0 && j < len(cells) {
				return cells[j]
			}
			return ""
		}
		r := row{line: i + 1, state: get(stateCol), screen: get(screenCol), keys: get(keysCol), bindings: parseKeys(get(keysCol))}
		if globalCell.MatchString(r.state) {
			r.global = true
			s.globals = append(s.globals, r)
			continue
		}
		s.rows = append(s.rows, r)
	}
	return s
}

func clean(s string) string {
	return strings.NewReplacer("**", "", "__", "", "`", "").Replace(strings.TrimSpace(s))
}

func isSep(cells []string) bool {
	if len(cells) == 0 {
		return false
	}
	for _, c := range cells {
		if !sepCell.MatchString(strings.ReplaceAll(c, " ", "")) {
			return false
		}
	}
	return true
}

// splitRow splits a markdown table row into trimmed cells; `\|` is a literal pipe.
func splitRow(t string) []string {
	t = strings.TrimPrefix(t, "|")
	if strings.HasSuffix(t, "|") && !strings.HasSuffix(t, `\|`) {
		t = t[:len(t)-1]
	}
	var cells []string
	var b strings.Builder
	for i := 0; i < len(t); i++ {
		switch {
		case t[i] == '\\' && i+1 < len(t) && t[i+1] == '|':
			b.WriteByte('|')
			i++
		case t[i] == '|':
			cells = append(cells, clean(b.String()))
			b.Reset()
		default:
			b.WriteByte(t[i])
		}
	}
	return append(cells, clean(b.String()))
}

var spaceRe = regexp.MustCompile(`\s+`)

var (
	emptyCell = regexp.MustCompile(`(?i)^(?:[-–—]+|n/?a|none|\(none\))?$`)
	// proseLead starts an item that names no key: "type to filter", "any key close".
	proseLead = map[string]bool{"type": true, "typing": true, "any": true, "other": true, "click": true, "mouse": true, "wheel": true}
)

func parseKeys(cell string) []binding {
	cell = clean(cell)
	if emptyCell.MatchString(cell) {
		return nil
	}
	var out []binding
	for _, item := range strings.FieldsFunc(cell, func(r rune) bool { return r == '·' || r == '•' }) {
		f := strings.Fields(item)
		if len(f) == 0 {
			continue
		}
		if proseLead[strings.ToLower(strings.Trim(f[0], "(),:"))] {
			out = append(out, binding{action: actionText(f)})
			continue
		}
		out = append(out, readItem(f))
	}
	return out
}

func actionText(f []string) string {
	return strings.ToLower(strings.TrimRight(spaceRe.ReplaceAllString(strings.Join(f, " "), " "), ".,;"))
}

// readItem reads one `key action` item. Its keys are the first field and any that
// follow it as alternatives:
//
//	j/k, ↑/↓ move      a trailing "," says another key follows
//	h/l or ←/→ switch  a lone "or", "/" or "," between two keys
//	q / ctrl+c quit
//	[ ] back/fwd       a lone one-character field after a key, with an action after it
//	enter: open        a trailing ":" ends the keys
func readItem(f []string) binding {
	var b binding
	var written []string
	i := 0
	for {
		tok, more, stop := f[i], false, false
		if n := len(tok); n > 1 && (tok[n-1] == ',' || tok[n-1] == ':') {
			more, stop = tok[n-1] == ',', tok[n-1] == ':'
			tok = tok[:n-1]
		}
		written = append(written, tok)
		ks, bad := readToken(tok)
		b.keys = append(b.keys, ks...)
		if bad != "" && b.bad == "" {
			b.bad = bad
		}
		i++
		if stop || i >= len(f) {
			break
		}
		rest := len(f) - i
		if more && looksKey(f[i]) {
			continue
		}
		if (f[i] == "/" || f[i] == "," || strings.EqualFold(f[i], "or")) && rest >= 3 && looksKey(f[i+1]) {
			i++
			continue
		}
		// "[ ] back/fwd" and "j k scroll": a lone one-character field after a key is
		// another key, not the first word of the action. A spaced "/" is never a key.
		if rest >= 2 && utf8.RuneCountInString(f[i]) == 1 && !strings.ContainsAny(f[i], "-–—=:,/") {
			continue
		}
		break
	}
	b.token = strings.Join(written, " ")
	b.action = actionText(f[i:])
	return b
}

// parseGlobalKeys reads a Global line. Older TUI.md files separate it with commas
// ("ctrl+c quit, ? help, ctrl+k switcher"), so a line without · splits on , and ; too.
func parseGlobalKeys(line string) []binding {
	if strings.ContainsAny(line, "·•") {
		return parseKeys(line)
	}
	return parseKeys(strings.NewReplacer(",", "·", ";", "·").Replace(line))
}

func (s *spec) globalBindings() []binding {
	var bs []binding
	for _, g := range s.globals {
		bs = append(bs, g.bindings...)
	}
	return bs
}

// meaning is what key k does in r (its own keys, then Global's), or "".
func (s *spec) meaning(r row, k key) string {
	for _, bs := range [][]binding{r.bindings, s.globalBindings()} {
		for _, b := range bs {
			for _, bk := range b.keys {
				if bk == k {
					return b.action
				}
			}
		}
	}
	return ""
}

func (s *spec) keyAmbig(add func(string, int, string, ...any)) {
	for _, r := range append(append([]row{}, s.globals...), s.rows...) {
		for bi, b := range r.bindings {
			for _, k := range b.keys {
				as, why, ok := legacy(k)
				if ok || hasFallback(r.bindings, bi, k) {
					continue
				}
				arrives, then := "never arrives", ""
				if as != "" {
					arrives = "arrives as " + as
					if m := s.meaning(r, parseKey(as)); m != "" && normAction(m) != normAction(b.action) {
						then = fmt.Sprintf(", and %s here means %q", as, m)
					}
				}
				fix := "a key every terminal sends (a letter, ctrl+<letter>, alt+<key>)"
				if k.base == "enter" {
					fix = "ctrl+j (LF, which every terminal sends)"
				}
				add("S-KEYAMBIG", r.line, "%s: %s (%s) %s where the terminal doesn't disambiguate keys (tmux by default, many terminals, most SSH sessions)%s; %s. Add %s for the same action in this cell: `%s/%s %s`",
					r.state, k, b.action, arrives, then, why, fix, b.token, fallbackFor(k), b.action)
			}
		}
	}
}

func fallbackFor(k key) string {
	if k.base == "enter" {
		return "ctrl+j"
	}
	return "<key>"
}

// hasFallback reports whether another key in the same cell does the same action and
// survives a legacy terminal.
func hasFallback(bs []binding, at int, k key) bool {
	for i, b := range bs {
		if normAction(b.action) != normAction(bs[at].action) {
			continue
		}
		for _, o := range b.keys {
			if i == at && o == k {
				continue
			}
			if _, _, ok := legacy(o); ok {
				return true
			}
		}
	}
	return false
}

// normAction is how two hints are compared: lowercase, with every run of spaces and
// punctuation one space, so "back/fwd" and "back fwd" match and "open" and "open in
// browser" don't.
func normAction(a string) string {
	return strings.Join(strings.FieldsFunc(strings.ToLower(a), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	}), " ")
}

// keyDup checks one cell at a time: each state's own keys, and Global's keys together.
// A state that binds a Global key to its own action overrides it, because dispatch
// runs overlay, then focused region, then globals (architecture.md), so the row's hint
// is the true one and that is not a finding.
func (s *spec) keyDup(add func(string, int, string, ...any)) {
	check := func(r row, bs []binding) {
		actions := map[string][]string{}
		seen := map[string]map[string]bool{}
		for _, b := range bs {
			done := map[key]bool{}
			for _, k := range b.keys {
				if done[k] {
					continue
				}
				done[k] = true
				ks, n := k.String(), normAction(b.action)
				if seen[ks] == nil {
					seen[ks] = map[string]bool{}
				}
				if !seen[ks][n] {
					seen[ks][n] = true
					actions[ks] = append(actions[ks], fmt.Sprintf("%q", b.action))
				}
			}
		}
		keys := make([]string, 0, len(actions))
		for k := range actions {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if len(actions[k]) < 2 {
				continue
			}
			add("S-KEYDUP", r.line, "%s: %s means %s: the first match wins and the other hint lies. Give one of them another key",
				r.state, k, strings.Join(actions[k], " and "))
		}
	}
	if len(s.globals) > 0 {
		check(s.globals[0], s.globalBindings())
	}
	for _, r := range s.rows {
		check(r, r.bindings)
	}
}

var (
	typingKeysRe  = regexp.MustCompile(`(?i)\bnew ?line\b|\btype to\b|\btyping\b`)
	typingStateRe = regexp.MustCompile(`(?i)\b(?:textarea|textinput|text[ -]?(?:input|box|field|area)|input|editor|compose|composer)\b`)
)

// textEntry is the conservative test for a state where a text component has focus.
func textEntry(r row) bool {
	return typingStateRe.MatchString(r.state) || typingStateRe.MatchString(r.screen) || typingKeysRe.MatchString(r.keys)
}

func (s *spec) keyShadow(add func(string, int, string, ...any)) {
	globals := s.globalBindings()
	if len(globals) == 0 {
		return
	}
	for _, r := range s.rows {
		if !textEntry(r) {
			continue
		}
		own := map[key]bool{}
		for _, b := range r.bindings {
			for _, k := range b.keys {
				own[k] = true
			}
		}
		for _, g := range globals {
			var taken []string
			for _, k := range g.keys {
				if own[k] {
					continue
				}
				if what := consumed(k); what != "" {
					taken = append(taken, fmt.Sprintf("%s (%s)", k, what))
				}
			}
			if len(taken) == 0 {
				continue
			}
			add("S-KEYSHADOW", r.line, "%s: Global `%s %s` never reaches this state's text box: a focused Bubbles textarea/textinput takes %s first. Pick a key the text box leaves alone (esc, ctrl+o, ctrl+s, ctrl+x, ctrl+y, f-keys), or bind it to the same action in this row, so the state takes it before the text box does",
				r.state, g.token, g.action, strings.Join(taken, ", "))
		}
	}
}

var headingRe = regexp.MustCompile(`^(#{1,6})\s+(.*)$`)
var brandHeading = regexp.MustCompile(`(?i)brand|motif|glyph|identity`)

// ambiguous reports a symbol glyph (arrows, technical, box drawing, blocks, geometric
// shapes, misc symbols, dingbats) whose East Asian Width is ambiguous. Punctuation
// such as · • … — × is left out: it is prose, and the kit maps it in ASCII mode.
func ambiguous(r rune) bool {
	in := (r >= 0x2190 && r <= 0x21FF) || (r >= 0x2300 && r <= 0x23FF) || (r >= 0x2460 && r <= 0x24FF) ||
		(r >= 0x2500 && r <= 0x27BF) || (r >= 0x2B00 && r <= 0x2BFF)
	return in && width.LookupRune(r).Kind() == width.EastAsianAmbiguous
}

func (s *spec) glyphWidth(add func(string, int, string, ...any)) {
	covered := map[rune]bool{}
	for _, l := range s.lines {
		if strings.Contains(strings.ToLower(l), "ascii") {
			for _, r := range l {
				covered[r] = true
			}
		}
	}
	type seg struct {
		line int
		text string
	}
	var segs []seg
	level := 0 // the heading level of the brand section we're in, 0 outside
	for i, l := range s.lines {
		if m := headingRe.FindStringSubmatch(strings.TrimSpace(l)); m != nil {
			n := len(m[1])
			switch {
			case brandHeading.MatchString(m[2]):
				level = n
			case level > 0 && n <= level:
				level = 0
			}
		}
		if level > 0 {
			segs = append(segs, seg{i + 1, l})
		}
	}
	for _, r := range append(append([]row{}, s.globals...), s.rows...) {
		text := r.state + " " + r.screen
		for _, b := range r.bindings {
			text += " " + b.action // key tokens (↑↓←→) are key names, not glyphs
		}
		segs = append(segs, seg{r.line, text})
	}
	sort.SliceStable(segs, func(i, j int) bool { return segs[i].line < segs[j].line })
	reported := map[rune]bool{}
	for _, sg := range segs {
		var found []string
		for _, r := range sg.text {
			if ambiguous(r) && !covered[r] && !reported[r] {
				reported[r] = true
				found = append(found, string(r))
			}
		}
		if len(found) == 0 {
			continue
		}
		g := strings.Join(found, " ")
		add("S-GLYPHWIDTH", sg.line, "%s: East Asian Width ambiguous, so one cell in most terminals and two in CJK locales, which shifts every column after it. Name each one's ASCII twin on a line with the glyph (e.g. `%s (ascii %s)`), which -ascii and TERM=linux draw instead",
			g, g, asciiHint(found))
	}
}

func asciiHint(gs []string) string {
	twins := map[string]string{"●": "*", "○": "o", "◐": "~", "◑": "~", "◆": "*", "◇": "o", "▲": "^", "△": "^", "■": "#", "□": "o", "★": "*", "─": "-", "│": "|", "▌": "|", "█": "#"}
	out := make([]string, len(gs))
	for i, g := range gs {
		if t, ok := twins[g]; ok {
			out[i] = t
		} else {
			out[i] = "<twin>"
		}
	}
	return strings.Join(out, " ")
}
