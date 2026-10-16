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
	action string // normalised
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
	for i, raw := range s.lines {
		t := strings.TrimSpace(raw)
		if !strings.HasPrefix(t, "|") {
			inTable = false
			if m := globalRe.FindStringSubmatch(clean(listMark.ReplaceAllString(t, ""))); m != nil {
				s.globals = append(s.globals, row{line: i + 1, state: "Global", global: true, bindings: parseKeys(m[1])})
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

func parseKeys(cell string) []binding {
	var out []binding
	for _, item := range strings.FieldsFunc(clean(cell), func(r rune) bool { return r == '·' || r == '•' }) {
		f := strings.Fields(item)
		if len(f) == 0 {
			continue
		}
		action := strings.ToLower(strings.TrimRight(spaceRe.ReplaceAllString(strings.Join(f[1:], " "), " "), ".,;"))
		out = append(out, binding{token: f[0], keys: parseToken(f[0]), action: action})
	}
	return out
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
					if m := s.meaning(r, parseKey(as)); m != "" && m != b.action {
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
		if b.action != bs[at].action {
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

func (s *spec) keyDup(add func(string, int, string, ...any)) {
	type use struct {
		action string
		global bool
	}
	check := func(r row, uses map[string][]use) {
		keys := make([]string, 0, len(uses))
		for k := range uses {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			us := uses[k]
			var parts []string
			seen := map[string]bool{}
			local := false
			for _, u := range us {
				local = local || !u.global
				if seen[u.action] {
					continue
				}
				seen[u.action] = true
				p := fmt.Sprintf("%q", u.action)
				if u.global && !r.global {
					p += " (Global)"
				}
				parts = append(parts, p)
			}
			if len(parts) < 2 || (!r.global && !local) {
				continue
			}
			add("S-KEYDUP", r.line, "%s: %s means %s: the first match wins and the other hint lies. Give one of them another key",
				r.state, k, strings.Join(parts, " and "))
		}
	}
	collect := func(uses map[string][]use, bs []binding, global bool) {
		for _, b := range bs {
			done := map[key]bool{}
			for _, k := range b.keys {
				if done[k] {
					continue
				}
				done[k] = true
				uses[k.String()] = append(uses[k.String()], use{b.action, global})
			}
		}
	}
	if len(s.globals) > 0 {
		uses := map[string][]use{}
		collect(uses, s.globalBindings(), true)
		check(s.globals[0], uses)
	}
	for _, r := range s.rows {
		uses := map[string][]use{}
		collect(uses, s.globalBindings(), true)
		collect(uses, r.bindings, false)
		check(r, uses)
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
