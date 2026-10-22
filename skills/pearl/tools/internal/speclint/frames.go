package speclint

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"tuitools/internal/framelint"
	"tuitools/internal/tuimd"
)

// frameHint is one `key action` hint read off a golden frame's footer row.
type frameHint struct {
	text    string
	keys    []key
	twin    []key          // the keys again with a lone "v" read as ↓, its ASCII twin and a key of its own
	keyCell framelint.Cell // the key's first cell, as drawn: its style is ui.Help's Key
}

// frameFooter reads the footer of a golden frame: its last non-blank row, split where
// two or more blank cells fall, as ui.Help joins hints. A chunk is a hint when its first
// word reads as keys and is drawn in another style than the word after it (ui.Help's
// Key and KeyDesc), so status text on the row ("5 variables") is not one; a row with
// no style at all (NO_COLOR) takes every chunk that reads as keys. ui.Footer's right
// status is not a hint: a count (`4 of 101`), and after the help hint a chunk that leads
// with a number (`? help  9 procs`). cut is a lone ellipsis among the chunks: Help
// dropped hints behind it.
func frameFooter(text string) (hints []frameHint, cut bool) {
	f := framelint.Parse(strings.TrimSuffix(strings.ReplaceAll(text, "\r\n", "\n"), "\n"))
	var row []framelint.Cell
	for i := len(f.Rows) - 1; i >= 0; i-- {
		for _, c := range f.Rows[i] {
			if strings.TrimSpace(c.Ch) != "" {
				row = f.Rows[i]
				break
			}
		}
		if row != nil {
			break
		}
	}
	styled := false
	for _, c := range row {
		styled = styled || c.FG != nil || c.Bold
	}
	var chunks [][]framelint.Cell
	var cur []framelint.Cell
	blanks := 0
	for _, c := range append(row, framelint.Cell{Ch: " "}, framelint.Cell{Ch: " "}) {
		if c.Ch == "" {
			continue // a wide glyph's second cell
		}
		if strings.TrimSpace(c.Ch) == "" {
			blanks++
			if blanks == 2 && len(cur) > 0 {
				chunks = append(chunks, cur)
				cur = nil
			}
			if blanks == 1 && len(cur) > 0 {
				cur = append(cur, c)
			}
			continue
		}
		blanks = 0
		cur = append(cur, c)
	}
	// every chunk's readings as a hint, in order; the first chunk may open with
	// ui.Footer's left status (a badge, a prompt, a spinner frame and what is working)
	cands := make([][]frameHint, len(chunks))
	for i, ch := range chunks {
		var sb strings.Builder
		for _, c := range ch {
			sb.WriteString(c.Ch)
		}
		if s := strings.TrimSpace(sb.String()); s == "…" || s == "..." {
			cut = true
			continue
		}
		cands[i] = chunkHints(ch, styled, i == 0)
	}
	// ui.Help draws every key in one style (Key). Past the first chunk no left status
	// stands before a hint, so the first hint there names that style, and the first
	// chunk's reading must keep it: an ASCII spinner frame ("/" in the working ink)
	// before "Deploying" is no key, the hint after it is
	var keyStyle *framelint.Cell
	for _, cs := range cands[min(1, len(cands)):] {
		if len(cs) > 0 {
			keyStyle = &cs[0].keyCell
			break
		}
	}
	afterHelp := false
	for i, cs := range cands {
		if len(cs) == 0 {
			continue
		}
		h := cs[0]
		if i == 0 && styled && keyStyle != nil {
			at := slices.IndexFunc(cs, func(c frameHint) bool { return sameStyle(c.keyCell, *keyStyle) })
			if at < 0 {
				continue // all status: ui.Footer's left
			}
			h = cs[at]
		}
		if countRe.MatchString(h.text) || (afterHelp && numberRe.MatchString(h.text)) {
			continue // "4 of 101", "1/9", and after the help hint "9 procs": ui.Footer's right status
		}
		hints = append(hints, h)
		afterHelp = afterHelp || (len(h.keys) == 1 && (h.keys[0].String() == "?" || h.keys[0].String() == "ctrl+g"))
	}
	return hints, cut
}

// chunkHints are the readings of one footer chunk as a `key action` hint, the likeliest
// first: the whole chunk, then past a prompt or badge one blank before the first hint
// ("/yes▏ enter keep", "[PROD] enter next") where a style changes, or under NO_COLOR
// past one word that is no status word. In the first chunk (first) a styled row is read
// on at every word where the style changes, past ui.Footer's left status however many
// words it has ("⣾ Deploying checkout-api ↑↓/jk move").
func chunkHints(ch []framelint.Cell, styled, first bool) []frameHint {
	var out []frameHint
	if h, ok := chunkHint(ch, styled); ok {
		out = append(out, h)
	}
	// word starts after the first word, and the cell before each
	var starts []int
	for at := 0; at < len(ch); {
		for at < len(ch) && strings.TrimSpace(ch[at].Ch) != "" {
			at++
		}
		for at < len(ch) && strings.TrimSpace(ch[at].Ch) == "" {
			at++
		}
		if at < len(ch) {
			starts = append(starts, at)
		}
	}
	if len(starts) == 0 {
		return out
	}
	switch {
	case !styled && first && !startsWord(ch) && len(out) == 0:
		// NO_COLOR: a prompt or badge one blank before the first hint, with no style to
		// tell them apart: the hint starts at the next word
		if h, ok := chunkHint(ch[starts[0]:], styled); ok {
			out = append(out, h)
		}
	case styled && !first:
		// a badge one blank before the first hint: the hint starts at the next word when
		// that is drawn in another style
		if len(out) == 0 && !sameStyle(ch[0], ch[starts[0]]) {
			if h, ok := chunkHint(ch[starts[0]:], styled); ok {
				out = append(out, h)
			}
		}
	case styled:
		for _, at := range starts {
			prev := at - 1
			for prev > 0 && strings.TrimSpace(ch[prev].Ch) == "" {
				prev--
			}
			if sameStyle(ch[prev], ch[at]) {
				continue
			}
			if h, ok := chunkHint(ch[at:], styled); ok {
				out = append(out, h)
			}
		}
	}
	return out
}

// numberRe is a chunk that leads with a number ("9 procs"): after the help hint ui.Help
// pins last, ui.Footer's right status, not a digit key.
var numberRe = regexp.MustCompile(`^\d+\s`)

// countRe is a footer chunk that counts rather than binds, the whole chunk: "4 of 101",
// "1/9", "3 / 12". "1/2 pane" is keys.
var countRe = regexp.MustCompile(`^\d+\s*(?:of|/)\s*\d+$`)

// startsWord reports a chunk whose first cell is a letter or digit: a status word, not
// a prompt (/, >, ›) or a badge ([PROD]).
func startsWord(ch []framelint.Cell) bool {
	for _, c := range ch {
		if strings.TrimSpace(c.Ch) == "" {
			continue
		}
		r := []rune(c.Ch)[0]
		return unicode.IsLetter(r) || unicode.IsDigit(r)
	}
	return false
}

// asciiArrows are the arrow keys as an ASCII frame draws them (theme.ASCIITwin): "^v"
// would read as ctrl+v. arrowPairs is an alternative made of the pairs alone ("^v",
// "<>", "^v<>"), which arrowTwins turns back.
var (
	asciiArrows = map[string]string{"^": "↑", "<-": "←", "->": "→"}
	arrowPairs  = regexp.MustCompile(`^(?:\^v|<>)+$`)
	arrowTwins  = strings.NewReplacer("^v", "↑↓", "<>", "←→")
)

// unASCII turns a key token's ASCII arrow twins back into arrows, alternative by
// alternative ("^v/jk" is ↑↓/jk).
func unASCII(tok string) string {
	alts := splitAlts(tok)
	for i, a := range alts {
		if r, ok := asciiArrows[a]; ok {
			alts[i] = r
		} else if arrowPairs.MatchString(a) {
			alts[i] = arrowTwins.Replace(a)
		}
	}
	return strings.Join(alts, "/")
}

// chunkHint reads one footer chunk as a `key action` hint: its first word reads as keys
// and, on a styled row, the action after it is drawn in another style.
func chunkHint(ch []framelint.Cell, styled bool) (frameHint, bool) {
	var sb strings.Builder
	for _, c := range ch {
		sb.WriteString(c.Ch)
	}
	s := strings.TrimSpace(sb.String())
	words := strings.Fields(s)
	if len(words) < 2 {
		return frameHint{}, false
	}
	words[0] = unASCII(words[0])
	b := readItem(words)
	if b.bad != "" || len(b.keys) == 0 || b.action == "" || slices.ContainsFunc(b.keys, unpressable) {
		return frameHint{}, false
	}
	if styled { // the action's first cell, after the key words, in another style
		at, n := 0, len(strings.Fields(b.token))
		for w := 0; w < n && at < len(ch); w++ {
			for at < len(ch) && strings.TrimSpace(ch[at].Ch) != "" {
				at++
			}
			for at < len(ch) && strings.TrimSpace(ch[at].Ch) == "" {
				at++
			}
		}
		if at >= len(ch) || sameStyle(ch[0], ch[at]) {
			return frameHint{}, false
		}
	}
	h := frameHint{text: s, keys: b.keys}
	for _, c := range ch {
		if strings.TrimSpace(c.Ch) != "" {
			h.keyCell = c
			break
		}
	}
	if alts := splitAlts(words[0]); slices.Contains(alts, "v") { // "v select": v, or ↓ under -ascii
		for i, a := range alts {
			if a == "v" {
				alts[i] = "↓"
			}
		}
		h.twin = readItem(append([]string{strings.Join(alts, "/")}, words[1:]...)).keys
	}
	return h, true
}

// unpressable reports a one-character key no keyboard sends: a spinner frame or a mark
// (braille ⣾, blocks ▖, shapes ◢ ◴, bullets • ·) drawn before a footer's status text.
// ASCII, letters, digits and Latin-1's symbols (§, £, °, a European layout's) are keys;
// the middle dot is a separator and a spinner frame, never a key.
func unpressable(k key) bool {
	r, n := utf8.DecodeRuneInString(k.base)
	if n != len(k.base) || unicode.IsLetter(r) || unicode.IsDigit(r) {
		return false
	}
	return r >= 0x100 || r == 0xB7 || !unicode.IsPrint(r)
}

// sameAs is a keys cell that borrows another state's: "as chat", "same as chat",
// "as chat; ↑ edit last request" (that state's keys and these).
var sameAs = regexp.MustCompile(`(?i)^(?:same\s+)?as\s+([\w-]+)(?:\s*[.;,]\s*(.*))?$`)

func sameStyle(a, b framelint.Cell) bool {
	sameCol := (a.FG == nil) == (b.FG == nil) && (a.FG == nil || *a.FG == *b.FG)
	return sameCol && a.Bold == b.Bold && a.Faint == b.Faint && a.Reverse == b.Reverse
}

// goldenFooters is S-GOLDEN for footers: each state's golden at 80x24 (dir/<state>-80x24.ansi,
// the harness name) shows in its footer the keys the state's row puts there. A hint whose
// keys the row and Global don't bind is a footer offering what the state lacks (row keys
// over an empty table), or a stale table; a key the row puts in the footer that the frame
// doesn't show, with nothing dropped behind …, is the other way round.
func (s *spec) goldenFooters(dir string, add func(string, int, string, ...any)) {
	var global []key
	for _, b := range s.globalBindings() {
		global = append(global, b.keys...)
	}
	byState := map[string]row{}
	for _, r := range s.rows {
		byState[r.state] = r
	}
	for _, r := range s.rows {
		name := r.state + "-80x24.ansi"
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		keysOf := r
		if m := sameAs.FindStringSubmatch(clean(r.keys)); m != nil { // "as chat; **↑ edit**": that row's keys, and these
			o, ok := byState[m[1]]
			if !ok {
				continue
			}
			keysOf.bindings = append(append([]binding{}, o.bindings...), parseKeys(m[2])...)
			keysOf.footer = append([]string{}, o.footer...)
			for _, it := range r.footer {
				if n := sameAs.FindStringSubmatch(it); n != nil {
					keysOf.footer = append(keysOf.footer, hints(n[2])...)
				} else {
					keysOf.footer = append(keysOf.footer, it)
				}
			}
		}
		hints, cut := frameFooter(string(data))
		if len(hints) == 0 {
			continue // a toast or a prompt holds the row: no footer to compare
		}
		bound, maybe := map[key]bool{}, map[key]bool{}
		for _, k := range global {
			bound[k] = true
		}
		for _, b := range keysOf.bindings {
			for _, k := range b.keys {
				bound[k] = true
				maybe[k] = maybe[k] || b.cond != ""
			}
		}
		shown := map[key]bool{}
		var extra []string
		for _, h := range hints {
			any := false
			for _, k := range append(append([]key{}, h.keys...), h.twin...) {
				shown[k] = true
				any = any || bound[k]
			}
			if !any {
				extra = append(extra, "`"+h.text+"`")
			}
		}
		var missing []string
		if !cut {
			for _, item := range keysOf.footer {
				b := readItem(strings.Fields(item))
				if b.bad != "" || len(b.keys) == 0 {
					continue
				}
				any := false
				for _, k := range b.keys {
					any = any || shown[k] || maybe[k] // shown only under a condition, or later
				}
				if !any {
					missing = append(missing, "`"+item+"`")
				}
			}
		}
		var parts []string
		if len(extra) > 0 {
			parts = append(parts, fmt.Sprintf("shows %s, which the row doesn't bind", strings.Join(extra, ", ")))
		}
		if len(missing) > 0 {
			parts = append(parts, fmt.Sprintf("lacks %s, which the row puts in the footer (end an item with `(if …)` when the footer shows it only then, or `(later)` when its screen comes later in the slice)", strings.Join(missing, ", ")))
		}
		if len(parts) > 0 {
			add("S-GOLDEN", r.line, "%s: the footer of %s %s: the footer and the table disagree, so one of them lies. Make the state's actions() and this row the same keys, then re-render",
				r.state, filepath.Join(filepath.Base(dir), name), strings.Join(parts, ", and "))
		}
	}
}

// blankRegion names a budget region that is blank by design: a gap between regions.
var blankRegion = regexp.MustCompile(`(?i)^(?:gap|gutter|margin|spacer|padding|blank)s?\b`)

// regionBudgets is S-REGION: each `<W>×<H>:` budget line of a screen block against that
// screen's golden at that size (dir/<screen>-<W>x<H>.ansi). A region whose whole
// rectangle is blank there is drawn somewhere else, or not at all: the layout moved and
// the budget went stale. A region with any text in it is not reported, even when the
// text is another region's (a status line under a transcript touches it on purpose), so
// a budget whose region the body moved into passes.
func (s *spec) regionBudgets(dir string, add func(string, int, string, ...any)) {
	for _, b := range tuimd.Budgets(strings.Join(s.lines, "\n")) {
		name := fmt.Sprintf("%s-%dx%d.ansi", b.Screen, b.W, b.H)
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		f := framelint.Parse(strings.TrimSuffix(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n"))
		for _, r := range b.Regions {
			if blankRegion.MatchString(r.Name) {
				continue
			}
			blank := true
			for y := r.Y; y < r.Y+r.H && y < len(f.Rows) && blank; y++ {
				for x := r.X; x < r.X+r.W && x < len(f.Rows[y]); x++ {
					if strings.TrimSpace(f.Rows[y][x].Ch) != "" {
						blank = false
						break
					}
				}
			}
			if blank {
				add("S-REGION", b.Line, "%s: `%s %d,%d %d×%d` is blank in %s: the region is drawn elsewhere or not at all, so the budget went stale. Measure the region in the golden and write its rectangle here (cmd-init.md: update TUI.md when a decision changes), or move the region back",
					b.Screen, r.Name, r.X, r.Y, r.W, r.H, filepath.Join(filepath.Base(dir), name))
			}
		}
	}
}
