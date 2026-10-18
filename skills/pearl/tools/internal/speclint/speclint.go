package speclint

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/width"

	"tuikit/theme/bubbletea"
	"tuitools/internal/cmddoc"
)

//go:embed doc.go
var docSrc string

// Usage is the -h text: the package doc comment.
func Usage() string { return cmddoc.Text(docSrc) + "\n" }

// Severity maps each rule ID to P1 (fails the run) or P2 (warns).
var Severity = map[string]string{
	"S-TABLE": "P1", "S-KEYAMBIG": "P1", "S-KEYDUP": "P1",
	"S-KEYSHADOW": "P2", "S-GLYPHWIDTH": "P2", "S-FOOTER": "P2", "S-ACCENT": "P2", "S-SLOTS": "P2", "S-GOLDEN": "P2", "S-REGION": "P2",
}

// Finding is one result, in the shape the other tuilint modes write to --json: Where
// is the line number, as a string like frames' "r3c4" and code's line.
type Finding struct {
	Rule  string `json:"rule"`
	Sev   string `json:"sev"`
	File  string `json:"file"`
	Where string `json:"where"`
	Msg   string `json:"msg"`
}

// Line is the finding's line number.
func (f Finding) Line() int {
	n, _ := strconv.Atoi(f.Where)
	return n
}

// Main runs `tuilint spec` with its arguments and returns the exit status.
func Main(args []string, stdout, stderr io.Writer) int {
	var files []string
	jsonOut, frames := "", ""
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
		case a == "--frames" || a == "-frames":
			if i+1 >= len(args) {
				fmt.Fprintf(stderr, "tuilint spec: %s needs a value\n", a)
				return 2
			}
			i++
			frames = args[i]
		case strings.HasPrefix(a, "--frames=") || strings.HasPrefix(a, "-frames="):
			_, frames, _ = strings.Cut(a, "=")
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
	if frames != "" {
		if st, err := os.Stat(frames); err != nil || !st.IsDir() {
			fmt.Fprintf(stderr, "tuilint spec: --frames %s is not a directory\n", frames)
			return 2
		}
	}
	var findings []Finding
	for _, f := range files {
		src, err := os.ReadFile(f)
		if err != nil {
			fmt.Fprintf(stderr, "tuilint spec: %v\n", err)
			return 2
		}
		findings = append(findings, LintFrames(f, string(src), frames)...)
	}
	fails := 0
	for _, f := range findings {
		fmt.Fprintf(stdout, "[%s] %-12s %s:%s  %s\n", f.Sev, f.Rule, f.File, f.Where, f.Msg)
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
	cond   string // "if …", "when …", "while …" or "later": the footer shows it only then
}

// condMark ends an item the footer shows only under a condition, `(if a filter is
// applied)`, `(when …)`, `(while the body overflows)`, or once a screen later in the
// slice lands, `(later)`, `(later: palette)`. The rules read the item without it, and
// S-GOLDEN doesn't ask a golden to show it.
var condMark = regexp.MustCompile(`(?i)\s*\(\s*((?:if|when|while|later)\b[^)]*)\)\s*$`)

// cutCond splits an item's condition mark off: the item as the other rules read it, and
// the condition ("" for none).
func cutCond(item string) (string, string) {
	m := condMark.FindStringSubmatchIndex(item)
	if m == nil {
		return item, ""
	}
	return item[:m[0]], strings.ToLower(strings.TrimSpace(item[m[2]:m[3]]))
}

// row is one state of the table, or a Global line.
type row struct {
	line          int
	state, screen string
	keys          string // the keys cell as written
	bindings      []binding
	global        bool
	footer        []string // the items the footer shows: all, or those before ‖
	helpOnly      []string // the other items: in ? help, not in the footer
	marker        string   // how the cell marks its footer: "", "‖" or "bold"
	mixed         bool     // the cell uses both markers
	badge         string   // the footer badge the cell names first (`[plan]`), or ""
}

type spec struct {
	file    string
	lines   []string
	rows    []row // states, in order
	globals []row
}

// Lint checks one TUI.md. file names it in the findings.
func Lint(file, src string) []Finding { return LintFrames(file, src, "") }

// LintFrames is Lint, and with frames (a directory of golden frames named by harness
// state, <state>-80x24.ansi) S-GOLDEN too: each state's footer there against its row.
func LintFrames(file, src, frames string) []Finding {
	s := parse(file, src)
	var out []Finding
	add := func(rule string, line int, format string, a ...any) {
		out = append(out, Finding{Rule: rule, Sev: Severity[rule], File: file, Where: strconv.Itoa(line), Msg: fmt.Sprintf(format, a...)})
	}
	warn := func(rule string, line int, format string, a ...any) { // P2 under a rule that is P1 by default
		out = append(out, Finding{Rule: rule, Sev: "P2", File: file, Where: strconv.Itoa(line), Msg: fmt.Sprintf(format, a...)})
	}
	if len(s.rows) == 0 {
		add("S-TABLE", 1, "no States-and-keys table: write `| state (harness name) | screen / overlay | keys |` with keys as `key action · key action`, as cmd-init.md's template does; the key rules can't run without it")
	}
	for _, r := range append(append([]row{}, s.globals...), s.rows...) {
		for _, b := range r.bindings {
			if b.bad != "" {
				warn("S-TABLE", r.line, "%s: can't read key %q in `%s %s`, so the key rules skip it. Write keys as Bubble Tea names (ctrl+k, shift+tab, enter, pgdown, ↑↓, j/k), each item `key action`, items separated by ·",
					r.state, b.bad, b.token, b.action)
			}
		}
	}
	for _, r := range s.rows {
		if r.mixed {
			warn("S-TABLE", r.line, "%s: the keys cell marks its footer both with **bold** and with ‖. Pick one footer marker: bold the footer keys, or put ‖ after the last of them; this run reads the ‖", r.state)
		}
	}
	s.keyAmbig(add, warn)
	s.keyDup(add)
	s.keyShadow(add)
	s.glyphWidth(add)
	s.footerFit(add)
	s.accentDrift(add)
	s.brandSlots(add)
	s.screenBrand(add)
	if frames != "" {
		s.goldenFooters(frames, add)
		s.regionBudgets(frames, add)
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Sev != b.Sev {
			return a.Sev < b.Sev
		}
		if a.Line() != b.Line() {
			return a.Line() < b.Line()
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
		badge, keys := cutBadge(get(keysCol))
		r := row{line: i + 1, state: get(stateCol), screen: get(screenCol), keys: keys, bindings: parseKeys(keys), badge: badge}
		raw := splitRaw(t)
		if keysCol < len(raw) {
			_, cell := cutBadge(raw[keysCol])
			r.footer, r.helpOnly, r.marker, r.mixed = footerItems(cell)
		}
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

// splitRow splits a markdown table row into trimmed cells with their markup removed.
func splitRow(t string) []string {
	cells := splitRaw(t)
	for i, c := range cells {
		cells[i] = clean(c)
	}
	return cells
}

// splitRaw splits a markdown table row into trimmed cells as written, markup and all;
// `\|` is a literal pipe.
func splitRaw(t string) []string {
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
			cells = append(cells, strings.TrimSpace(b.String()))
			b.Reset()
		default:
			b.WriteByte(t[i])
		}
	}
	return append(cells, strings.TrimSpace(b.String()))
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
	for _, item := range splitItems(cell) {
		item, cond := cutCond(item)
		f := strings.Fields(item)
		if len(f) == 0 {
			continue
		}
		if proseLead[strings.ToLower(strings.Trim(f[0], "(),:"))] {
			out = append(out, binding{action: actionText(f), cond: cond})
			continue
		}
		b := readItem(f)
		b.cond = cond
		out = append(out, b)
	}
	return out
}

// splitItems splits a keys cell into its `key action` items. ‖ (the footer cut) ends
// an item as · does.
func splitItems(cell string) []string {
	return strings.FieldsFunc(cell, func(r rune) bool { return r == '·' || r == '•' || r == footerCut })
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
	if strings.ContainsAny(line, "·•‖") {
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

func (s *spec) keyAmbig(add, warn func(string, int, string, ...any)) {
	for _, r := range append(append([]row{}, s.globals...), s.rows...) {
		for bi, b := range r.bindings {
			for _, k := range b.keys {
				if bsClash(k) {
					warn("S-KEYAMBIG", r.line, "%s: %s (%s) arrives as itself, but BS (0x08) decodes as ctrl+h too, and xterm-like terminals send BS for ctrl+backspace (a few for backspace), so there that key does %q as well. Keep it if that's harmless here; otherwise pick another key",
						r.state, k, b.action, b.action)
					continue
				}
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
// runs overlay, then focused region, then globals (bubbletea-architecture.md), so the
// row's hint is the true one and that is not a finding.
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
		line  int
		text  string
		table bool
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
			segs = append(segs, seg{i + 1, l, false})
		}
	}
	for _, r := range append(append([]row{}, s.globals...), s.rows...) {
		text := r.state + " " + r.screen
		for _, b := range r.bindings {
			text += " " + b.action // key tokens (↑↓←→) are key names, not glyphs
		}
		segs = append(segs, seg{r.line, text, true})
	}
	sort.SliceStable(segs, func(i, j int) bool { return segs[i].line < segs[j].line })
	reported := map[rune]bool{}
	for _, sg := range segs {
		var found []string
		for _, r := range sg.text {
			if !ambiguous(r) || covered[r] || reported[r] {
				continue
			}
			if _, ok := theme.ASCIITwin(string(r)); ok {
				continue // the kit's ASCII map draws its twin (─ ▌ │ ● ○ ◆ ▲ ■ …)
			}
			if r >= 0x2190 && r <= 0x21FF && !sg.table {
				continue // an arrow in prose ("queued → running"), not a column glyph
			}
			reported[r] = true
			found = append(found, string(r))
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
	// twins for common glyphs the kit's map (theme.ASCIITwin) leaves out
	twins := map[string]string{"◑": "~", "◒": "~", "◓": "~", "★": "*", "☆": "o", "▼": "v", "▽": "v", "◀": "<", "◁": "<"}
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

var (
	accentsLine = regexp.MustCompile(`(?i)^\s*(?:[-*+]\s+)?accents?\s*:\s*(.+)$`)
	accentPair  = regexp.MustCompile(`^\s*([^=]+?)\s*=\s*(.+)$`)
	// "the accent", "primary hue/colour/accent": the hue. "primary pane", "primary key"
	// and "primary button" are not.
	accentWord = regexp.MustCompile(`(?i)\baccent\b|\bprimary\s+(?:hue|colou?r|accent)\b`)
	wordRe     = regexp.MustCompile(`[\p{L}\p{N}]+`)
)

// accentDrift is S-ACCENT: once the `accents:` lines give each hue one meaning
// ("amber = new, nothing else"), a later line that puts "the accent" or "the primary
// hue" on something else (the wordmark, the spinner's dots, a progress ramp) breaks the
// promise; only the critic used to catch it. Every `hue = meaning` pair of every
// accents line (pairs separated by ;) joins one set of words; a line passes when it
// holds a hue or a word of a meaning. Lines of the States-and-keys table, fenced
// sketches and the accents lines themselves are not read.
func (s *spec) accentDrift(add func(string, int, string, ...any)) {
	same, isAccents := map[string]bool{}, map[int]bool{}
	firstLine := 0
	for i, l := range s.lines {
		m := accentsLine.FindStringSubmatch(clean(l))
		if m == nil {
			continue
		}
		for _, seg := range strings.Split(m[1], ";") {
			p := accentPair.FindStringSubmatch(seg)
			if p == nil || strings.HasPrefix(p[1], "<") || strings.HasPrefix(p[2], "<") {
				continue // the template's placeholders, or no "hue = meaning"
			}
			if firstLine == 0 {
				firstLine = i + 1
			}
			isAccents[i+1] = true
			for _, w := range wordRe.FindAllString(strings.ToLower(p[1]+" "+p[2]), -1) {
				if len(w) >= 3 && !accentStop[w] {
					same[w] = true
				}
			}
		}
	}
	if firstLine == 0 {
		return
	}
	fenced := false
	for i, l := range s.lines {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "```") {
			fenced = !fenced
			continue
		}
		if fenced || isAccents[i+1] || strings.HasPrefix(t, "|") || !accentWord.MatchString(t) {
			continue
		}
		drift := true
		for _, w := range wordRe.FindAllString(strings.ToLower(t), -1) {
			if same[w] {
				drift = false
				break
			}
		}
		if !drift {
			continue
		}
		add("S-ACCENT", i+1, "puts the accent on something else: `%s`, but line %d says `%s`. One hue, one meaning: give this the role you mean (fg, fgMuted, secondary) or the hue's own meaning",
			t, firstLine, strings.TrimSpace(s.lines[firstLine-1]))
	}
}

// accentStop are words of an accents line that carry no meaning of their own.
var accentStop = map[string]bool{"the": true, "and": true, "for": true, "only": true, "nothing": true, "else": true, "one": true, "its": true, "hue": true, "never": true, "not": true, "any": true, "all": true, "that": true, "this": true, "with": true}

var slotLine = regexp.MustCompile(`(?i)^(?:[-*+]\s+)?(remember|move|signature)\s*:`)

// brandSlots is S-SLOTS: in a brand section (brandHeading), a `remember:` line and no
// `move:` or `signature:` line is a brand read from before those two replaced it. A
// slot word in backticks is prose about the slot, and a fenced line is a sketch.
func (s *spec) brandSlots(add func(string, int, string, ...any)) {
	remember, slots, fenced, level := 0, false, false, 0
	for i, l := range s.lines {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "```") {
			fenced = !fenced
			continue
		}
		if m := headingRe.FindStringSubmatch(t); m != nil && !fenced {
			n := len(m[1])
			switch {
			case brandHeading.MatchString(m[2]):
				level = n
			case level > 0 && n <= level:
				level = 0
			}
			continue
		}
		if fenced || level == 0 {
			continue
		}
		m := slotLine.FindStringSubmatch(strings.NewReplacer("**", "", "__", "").Replace(t))
		if m == nil {
			continue
		}
		if !strings.EqualFold(m[1], "remember") {
			slots = true
		} else if remember == 0 {
			remember = i + 1
		}
	}
	if remember > 0 && !slots {
		add("S-SLOTS", remember, "`remember:` with no `move:` or `signature:`: the brand read now names a move (one region elsewhere in the screen's order than the kit puts it, or gone, or added) and a signature (one element the kit doesn't draw), both on the main screen's default frame, and brand, craft and the critic look for those. Write both from this line and the brand's three words (references/brand.md § Move A Region)")
	}
}

var (
	// § Screens, numbered or not; "Screens as models" under Architecture is not one
	screensHeading = regexp.MustCompile(`(?i)^(?:\d+[.)]?\s+)?screens\s*(?:\([^)]*\))?\s*:?$`)
	// a sibling section that continues it ("Secondary screens", "Overlays"), read only
	// right after § Screens, so "Dialogs" under Architecture is not a screen list
	screensSibling = regexp.MustCompile(`(?i)^(?:\d+[.)]?\s+)?(?:(?:main|secondary|other|more)\s+)?(?:screens?|overlays?|dialogs?)(?:\s+and\s+(?:screens|overlays|dialogs|states))?\s*(?:\([^)]*\))?\s*:?$`)
	brandLine      = regexp.MustCompile(`(?i)^(?:[-*+]\s+)?brand\s*:\s*(.*)$`)
	flowLine       = regexp.MustCompile(`(?i)^(?:[-*+]\s+)?flow\s*:\s*(.*)$`)
	// a value that skips the question: "nothing", "none", "n/a", "-" with no reason after
	// it, or a placeholder: "TBD", "todo", "?", "nothing yet", the template's "<…>"
	brandSkip = regexp.MustCompile(`(?i)^(?:nothing|none|n/?a|-+|—|–|tbd|todo|\?+|nothing\s+yet|<[^>]*>)[.!]?$`)
	// a block's heading names its job after a dash or a colon: "detail — one job"
	jobRe = regexp.MustCompile(`\s[—–-]\s|:`)
)

// fenceToggle tracks ``` and ~~~ fences: a fence closes only on its own marker.
func fenceToggle(fence *string, t string) bool {
	for _, m := range []string{"```", "~~~"} {
		if strings.HasPrefix(t, m) {
			switch *fence {
			case "":
				*fence = m
			case m:
				*fence = ""
			default:
				return false // the other marker inside a fence is content
			}
			return true
		}
	}
	return false
}

// screenBrand is S-SLOTS for § Screens: each block (a heading under the Screens
// heading) after the first, which is the main screen and carries the read's move and
// signature, is a secondary screen or an overlay, and names a `brand:` line: what the
// brand's three words change there, or "nothing, because …". Blocks sit at the level of
// the section's first heading with a job (a dash or a colon); a heading above that level
// whose next heading is deeper groups blocks ("Main", "Overlays — dialogs and help") and
// is no block itself, and one below it ("regions", "layout: two panes") is part of its
// block. Presence only, but a value that skips the question
// ("nothing", "none", "n/a", "-" with no reason) doesn't count, nor does an empty
// value, a fenced line (the sketch) or a slot word in backticks; the critic judges
// what the line says. A `flow:` line is optional in any block, the main screen's too,
// but one that is there and holds only such a value is flagged where it stands: the
// template ships one in every block it copies. A block still named `<screen>` is the
// template itself, and is skipped.
func (s *spec) screenBrand(add func(string, int, string, ...any)) {
	type heading struct {
		line, level int
		text        string
	}
	var hs []heading
	fence := ""
	for i, l := range s.lines {
		t := strings.TrimSpace(l)
		if fenceToggle(&fence, t) || fence != "" {
			continue
		}
		if m := headingRe.FindStringSubmatch(t); m != nil {
			hs = append(hs, heading{i, len(m[1]), clean(m[2])})
		}
	}
	screens, block, blockLevel := 0, 0, 0 // the Screens heading's level; the current block's; the blocks
	blocks, start, name, has, skip := 0, -1, "", false, -1
	flush := func() {
		switch {
		case start < 0 || blocks <= 1 || has:
		case skip >= 0:
			add("S-SLOTS", skip+1, "%s: `%s` is a placeholder or gives no reason. Say what the brand's three words change on this screen (its copy voice, its dialog form, its footer phrasing), or why nothing: `brand: nothing, because help lists the action tables as written` (references/cmd-shape.md § Behind The Main Screen)", name, strings.TrimSpace(s.lines[skip]))
		default:
			add("S-SLOTS", start+1, "%s: a secondary screen or overlay with no `brand:` line (the first block under § Screens is read as the main screen, so put the main screen's block first). Say what the brand's three words change on it (its copy voice, its dialog form, its footer phrasing), or `brand: nothing, because …`: two brands built from one spec differed on the main screen and matched behind it word for word, footers, confirm copy and all (references/cmd-shape.md § Behind The Main Screen)", name)
		}
		start, name, has, skip = -1, "", false, -1
	}
	next := 0 // index into hs of the next heading
	fence = ""
	for i, l := range s.lines {
		t := strings.TrimSpace(l)
		if fenceToggle(&fence, t) || fence != "" {
			continue
		}
		if next < len(hs) && hs[next].line == i {
			h := hs[next]
			next++
			switch {
			case screensHeading.MatchString(h.text) || (screens > 0 && h.level <= screens && screensSibling.MatchString(h.text)):
				flush()
				screens, block = h.level, 0
				// the blocks' level: the first heading of the section with a job, else its
				// first heading
				blockLevel = 0
				for _, g := range hs[next:] {
					if g.level <= screens {
						break
					}
					if blockLevel == 0 {
						blockLevel = g.level
					}
					if jobRe.MatchString(g.text) {
						blockLevel = g.level
						break
					}
				}
			case screens > 0 && h.level <= screens:
				flush()
				screens = 0
			case screens == 0 || (start >= 0 && h.level > block):
				// outside § Screens, or a sub-heading inside a block
			case h.level < blockLevel && next < len(hs) && hs[next].level > h.level:
				flush() // a group heading, above the blocks' level: its blocks follow
				block = 0
			default:
				flush()
				block, blocks, start = h.level, blocks+1, i
				name = strings.TrimSpace(jobRe.Split(h.text, 2)[0])
			}
			continue
		}
		if start < 0 {
			continue
		}
		bare := strings.NewReplacer("**", "", "__", "").Replace(t)
		if f := flowLine.FindStringSubmatch(bare); f != nil {
			// a block still named as the template names it (`<screen>`) is the template
			if v := strings.TrimSpace(f[1]); (v == "" || brandSkip.MatchString(v)) && !(strings.HasPrefix(name, "<") && strings.HasSuffix(name, ">")) {
				add("S-SLOTS", i+1, "%s: `%s` is a placeholder. Name the domain fact behind this screen's depth (who acts, on how many objects, whether the act can be taken back) and any brand word that moves it, or, on a block after the main screen's, delete the line where its flow is the spec's (references/cmd-shape.md § Depth Follows The Domain, Then The Brand)", name, strings.TrimSpace(s.lines[i]))
			}
			continue
		}
		m := brandLine.FindStringSubmatch(bare)
		if m == nil {
			continue
		}
		switch v := strings.TrimSpace(m[1]); {
		case v == "":
		case brandSkip.MatchString(v):
			if skip < 0 {
				skip = i
			}
		default:
			has = true
		}
	}
	flush()
}
