package framelint

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode"
)

// F-FOOTWRAP and F-SPREAD read a frame's layout, not its colours: where its rows of
// text sit. Both came from one restyle of a setup wizard whose frames passed every
// other rule: its key footer ran to a second row at 80 columns, and its step column
// and form were distributed over the terminal's height instead of packed from the top.

// word is one run of non-space cells on a row: its text and its columns [x, e).
type word struct {
	text string
	x, e int
}

// rowWords splits a row into words. A wide glyph's continuation cell belongs to it.
func rowWords(row []Cell) []word {
	var out []word
	var b strings.Builder
	start := -1
	for x := 0; x <= len(row); x++ {
		sp := x == len(row)
		if !sp {
			c := row[x]
			if c.Ch == "" {
				sp = x == 0 || row[x-1].Width != 2
			} else {
				sp = blank(c.Ch)
			}
		}
		if sp {
			if start >= 0 {
				out = append(out, word{b.String(), start, x})
				b.Reset()
				start = -1
			}
			continue
		}
		if start < 0 {
			start = x
		}
		b.WriteString(row[x].Ch)
	}
	return out
}

// phrase is a run of words one space apart: two or more spaces end it.
type phrase struct {
	words []word
	x, e  int
}

func phrases(ws []word) []phrase {
	var out []phrase
	for _, w := range ws {
		if n := len(out); n > 0 && w.x-out[n-1].e == 1 {
			out[n-1].words = append(out[n-1].words, w)
			out[n-1].e = w.e
			continue
		}
		out = append(out, phrase{[]word{w}, w.x, w.e})
	}
	return out
}

var (
	keyNames = map[string]bool{
		"enter": true, "return": true, "ret": true, "esc": true, "escape": true, "tab": true,
		"space": true, "spc": true, "backspace": true, "bksp": true, "del": true, "delete": true,
		"ins": true, "insert": true, "home": true, "end": true, "pgup": true, "pgdn": true,
		"pgdown": true, "pageup": true, "pagedown": true, "up": true, "down": true, "left": true,
		"right": true, "arrows": true, "arrow": true,
	}
	fnKey    = regexp.MustCompile(`^f([1-9]|1[0-9]|2[0-4])$`)
	modKey   = regexp.MustCompile(`^(ctrl|control|alt|opt|option|meta|cmd|command|super|shift|hyper|c|m)[+-](.+)$`)
	caretKey = regexp.MustCompile(`^\^[a-z@\[\]\\^_]$`)
)

// keyPunct are the one-cell keys that are not letters or digits. A separator (· • │ |
// … - –) is not one: it sits between a status row's fields; nor is a bracket, which
// opens a [ button ].
const keyPunct = `?/:;!@#$%&*_+=~,.'"<>\`

// isKey says whether a word reads as a key in a hint: enter, esc, tab, space,
// pgup, home, an arrow or arrows (↑↓ ←→), f1–f24, ctrl+x / alt+x / shift+tab / C-x / ^c,
// one ASCII letter, digit or punctuation key (q ? /), or keys joined by / (j/k, ↑/k).
func isKey(s string) bool {
	if n := len(s); n > 2 && ((s[0] == '[' && s[n-1] == ']') || (s[0] == '<' && s[n-1] == '>')) {
		s = s[1 : n-1]
	}
	l := strings.ToLower(s)
	rs := []rune(l)
	switch {
	case len(rs) == 0:
		return false
	case len(rs) == 1:
		r := rs[0]
		return r < unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r)) || strings.ContainsRune(keyPunct+"↑↓←→", r)
	case keyNames[l], fnKey.MatchString(l), caretKey.MatchString(l):
		return true
	case strings.Trim(l, "↑↓←→") == "":
		return true
	}
	if m := modKey.FindStringSubmatch(l); m != nil {
		return isKey(m[2])
	}
	if strings.Contains(l, "/") {
		for _, p := range strings.Split(l, "/") {
			if p == "" || !isKey(p) {
				return false
			}
		}
		return true
	}
	return false
}

// hint is one `<key> <action>` pair on a row, columns [x, e).
type hint struct{ x, e int }

// keyHints reads a row as a key footer: its `<key> <action>` hints, and whether the
// row is one (a hint, and hints in at least half its phrases). A hint is a phrase that
// opens with a key and has 1–4 words of action after it ("enter next", "ctrl+b Back
// screen"), or a key alone and its action two spaces on ("enter  next"), the spacing
// some brands give the pair. A help listing's key column ("↑↓          select") is
// further from its action than that, and a status row ("cluster eu-west · namespace
// payments") opens with a word, so neither reads as one.
func keyHints(row []Cell) ([]hint, bool) {
	ps := phrases(rowWords(row))
	var hs []hint
	covered := 0
	for i := 0; i < len(ps); i++ {
		p := ps[i]
		if !isKey(p.words[0].text) {
			continue
		}
		if n := len(p.words); n >= 2 && n <= 5 {
			hs = append(hs, hint{p.x, p.e})
			covered++
			continue
		}
		if len(p.words) == 1 && i+1 < len(ps) && ps[i+1].x-p.e == 2 && len(ps[i+1].words) <= 4 && !isKey(ps[i+1].words[0].text) {
			hs = append(hs, hint{p.x, ps[i+1].e})
			covered += 2
			i++
		}
	}
	return hs, len(hs) > 0 && 2*covered >= len(ps)
}

// extent is the first and one past the last non-blank column of a row, or -1, -1.
func extent(row []Cell) (int, int) {
	ws := rowWords(row)
	if len(ws) == 0 {
		return -1, -1
	}
	return ws[0].x, ws[len(ws)-1].e
}

// lastRow is the index of the last non-blank row, or -1.
func lastRow(rows [][]Cell) int {
	for y := len(rows) - 1; y >= 0; y-- {
		if x, _ := extent(rows[y]); x >= 0 {
			return y
		}
	}
	return -1
}

// LintFootWrap is F-FOOTWRAP: the key footer runs to a second row. The last non-blank
// row is in the frame's last two, it and the row above it both read as key hints
// (the upper with 2 or more), and the first hint of the last row would not have fit
// on the row above: its end, two spaces and that hint pass the width less the row's
// left margin (the footer's measure, its margins taken as equal). That is what a wrap
// is. Two key rows whose hints share 2 or more columns are a grid, a help view
// listing keys in columns, not a wrap; a status row above the keys opens with a word,
// and a rule under footer: rule is no hint, so neither is the upper row.
func LintFootWrap(name string, f Frame) []Finding {
	rows := f.Rows
	last := lastRow(rows)
	if last < 1 || last < len(rows)-2 {
		return nil
	}
	low, ok := keyHints(rows[last])
	if !ok {
		return nil
	}
	up, ok := keyHints(rows[last-1])
	if !ok || len(up) < 2 {
		return nil
	}
	// a grid: any two adjacent key rows of the block over the bottom share 2 hint columns
	prev := low
	for y := last - 1; y >= 0; y-- {
		hs, ok := keyHints(rows[y])
		if !ok {
			break
		}
		shared := 0
		for _, h := range hs {
			if slices.ContainsFunc(prev, func(p hint) bool { return p.x == h.x }) {
				shared++
			}
		}
		if shared >= 2 {
			return nil
		}
		prev = hs
	}
	left, end := extent(rows[last-1])
	w := len(rows[last-1])
	if end+2+(low[0].e-low[0].x) <= w-left {
		return nil
	}
	return []Finding{{"F-FOOTWRAP", "P1", name, fmt.Sprintf("r%dc%d", last, low[0].x),
		fmt.Sprintf("the key footer runs to two rows (%d and %d): its hints did not fit in %d columns; keep it one row at every width: the keys this state needs most, packed from the left and cut with … and ? help when the rest don't fit (ui.Footer does)",
			last-1, last, w)}}
}

// spreadMinItems, spreadMinGap and spreadSlack are F-SPREAD's even-spacing thresholds:
// 4 or more rows starting at one column, every gap between them 3 or more blank rows,
// and no two gaps more than a row apart. A packed list's gaps are 0 or 1 blank row and
// a summary's label, value, blank is 1; a layout that divides the height among its
// items gives 5–6 at 40 rows and 3–4 at 24, all the same to within the remainder.
const (
	spreadMinItems = 4
	spreadMinGap   = 3
	spreadSlack    = 1
)

// hasWord is what makes a phrase an item: two letters or digits in a row. A rule, a
// mark, a bar or one glyph of a backdrop field (ʃ ▭) is not one.
var hasWord = regexp.MustCompile(`[\p{L}\p{N}]{2}`)

// LintSpread is F-SPREAD: content distributed over the frame's height instead of
// packed from the top. Two shapes, a finding for each a frame shows:
//
//   - evenly spaced items: 4 or more phrases starting at one column, between the
//     frame's first and last rows (its header and footer), each from the next 3 or
//     more rows that are blank across both (rows with text there, a step's sub-line,
//     are not counted), all those gaps within a row of each other;
//   - a stranded action row: a row of buttons (filled spans, or [ bracketed ]) in
//     the 4 rows over the bottom, with a quarter of the frame's height or more blank
//     above it, across its own columns, up to the content it belongs to.
func LintSpread(name string, f Frame) []Finding {
	rows := f.Rows
	first, last := -1, lastRow(rows)
	for y := range rows {
		if x, _ := extent(rows[y]); x >= 0 {
			first = y
			break
		}
	}
	if last-first < 2*spreadMinItems || splashName.MatchString(name) {
		return nil // a splash centres its mark in the room on purpose
	}
	return append(evenSpread(name, rows, first, last), strandedAction(name, rows, first, last)...)
}

func bandBlank(row []Cell, x0, x1 int) bool {
	for x := max(x0, 0); x < x1 && x < len(row); x++ {
		if c := row[x].Ch; c != "" && !blank(c) {
			return false
		}
	}
	return true
}

func evenSpread(name string, rows [][]Cell, first, last int) []Finding {
	c, ys, lo, hi := spreadRun(rows, first, last, spreadMinItems, spreadMinGap, spreadSlack)
	if ys == nil {
		return nil
	}
	span := fmt.Sprint(lo)
	if hi != lo {
		span = fmt.Sprintf("%d–%d", lo, hi)
	}
	return []Finding{{"F-SPREAD", "P2", name, fmt.Sprintf("r%dc%d", ys[0], c),
		fmt.Sprintf("%d rows at column %d (rows %s) sit %s blank rows apart, evenly: the layout spread them over the height; pack them from the top (one row each, a blank row between groups) and leave the room at the bottom",
			len(ys), c, pyIntList(ys), span)}}
}

// spreadRun is the first run, by column, of minItems or more phrases starting at one
// column whose gaps are all minGap or more blank rows and within slack of each other:
// the column, the rows, and the smallest and largest gap; nil rows when there is none.
func spreadRun(rows [][]Cell, first, last, minItems, minGap, slack int) (int, []int, int, int) {
	// an item's band runs from the start of the phrase left of it on its row (0 when
	// there is none) to its own end: a tag right-aligned beside a label is judged with
	// the label's value rows under it, not as a column of its own
	type item struct{ y, l, e int }
	at := map[int][]item{}
	var cols []int
	for y := first + 1; y < last; y++ {
		l := 0
		ps := phrases(rowWords(rows[y]))
		texts := make([]string, len(ps))
		seen := map[string]int{}
		for i, p := range ps {
			var b strings.Builder
			for _, w := range p.words {
				b.WriteString(w.text + " ")
			}
			texts[i] = b.String()
			seen[texts[i]]++
		}
		for i, p := range ps {
			// a phrase repeated 3 times on its row is a pattern (a backdrop field), not an item
			if hasWord.MatchString(texts[i]) && seen[texts[i]] < 3 {
				if _, ok := at[p.x]; !ok {
					cols = append(cols, p.x)
				}
				at[p.x] = append(at[p.x], item{y, l, p.e})
			}
			l = p.x
		}
	}
	slices.Sort(cols)
	for _, c := range cols {
		its := at[c]
		if len(its) < minItems {
			continue
		}
		gaps := make([]int, len(its)-1)
		for i := range gaps {
			a, b := its[i], its[i+1]
			l, e := min(a.l, b.l), max(a.e, b.e)
			for y := a.y + 1; y < b.y; y++ {
				if bandBlank(rows[y], l, e) {
					gaps[i]++
				}
			}
		}
		for i := range gaps {
			lo, hi, j := gaps[i], gaps[i], i
			for j < len(gaps) && gaps[j] >= minGap && max(hi, gaps[j])-min(lo, gaps[j]) <= slack {
				lo, hi = min(lo, gaps[j]), max(hi, gaps[j])
				j++
			}
			if n := j - i + 1; j > i && n >= minItems {
				ys := make([]int, n)
				for k := range ys {
					ys[k] = its[i+k].y
				}
				return c, ys, lo, hi
			}
		}
	}
	return 0, nil, 0, 0
}

var bracketButton = regexp.MustCompile(`\[[^\[\]]+\]|<[^<>]+>`)

// actionRow says whether a row is buttons and little else: its non-blank cells, but
// for 3 (a focus mark, a cursor), are inside filled spans no wider than half the row
// (a background other than the page's, or reverse video) or [ bracketed ] labels. It
// returns the columns [x0, x1) the buttons and their marks cover.
func actionRow(row []Cell, page string) (int, int, bool) {
	filled := func(c Cell) bool { return c.Reverse || bgKey(c) != page }
	in := make([]bool, len(row))
	spans := 0
	for x := 0; x < len(row); {
		if !filled(row[x]) {
			x++
			continue
		}
		e := x
		for e < len(row) && filled(row[e]) {
			e++
		}
		if 2*(e-x) > len(row) {
			return 0, 0, false // a bar, not a button
		}
		for k := x; k < e; k++ {
			in[k] = true
		}
		spans++
		x = e
	}
	var line strings.Builder
	starts := make([]int, 0, len(row))
	for _, c := range row {
		starts = append(starts, line.Len())
		ch := c.Ch
		if ch == "" {
			ch = "\x00" // a wide glyph's second cell
		}
		line.WriteString(ch)
	}
	text := line.String()
	brackets := 0
	for _, m := range bracketButton.FindAllStringIndex(text, -1) {
		brackets++
		for x := range row {
			if starts[x] >= m[0] && starts[x] < m[1] {
				in[x] = true
			}
		}
	}
	if spans == 0 && brackets == 0 {
		return 0, 0, false
	}
	text0, outside, x0, x1 := 0, 0, -1, -1
	for x, c := range row {
		if in[x] || (c.Ch != "" && !blank(c.Ch)) {
			if x0 < 0 {
				x0 = x
			}
			x1 = x + 1
		}
		if c.Ch == "" || blank(c.Ch) {
			continue
		}
		text0++
		if !in[x] {
			outside++
		}
	}
	return x0, x1, text0 > outside && outside <= 3
}

func strandedAction(name string, rows [][]Cell, first, last int) []Finding {
	page, _ := pageBG(Frame{Rows: rows})
	for y := max(last-3, first+1); y <= last; y++ {
		row := rows[y]
		x0, x1, ok := actionRow(row, page)
		if _, key := keyHints(row); key || !ok {
			continue
		}
		above := y - 1
		for above > first && bandBlank(rows[above], x0, x1) {
			above--
		}
		gap := y - above - 1
		if above <= first || 4*gap < len(rows) {
			continue
		}
		return []Finding{{"F-SPREAD", "P2", name, fmt.Sprintf("r%dc%d", y, x0),
			fmt.Sprintf("the action row at row %d sits %d blank rows below the content it belongs to (row %d), pushed to the bottom: pack it under that content with one blank row and leave the room at the bottom",
				y, gap, above)}}
	}
	return nil
}
