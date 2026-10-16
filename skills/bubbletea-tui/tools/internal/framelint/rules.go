package framelint

import (
	"fmt"
	"math"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Finding is one rule hit. Its JSON shape is tuilint.py's.
type Finding struct {
	Rule  string `json:"rule"`
	Sev   string `json:"sev"`
	File  string `json:"file"`
	Where string `json:"where"`
	Msg   string `json:"msg"`
}

// defaultColours come from Bubble Tea, Lip Gloss and Bubbles examples and
// default delegates, in tuilint.py's order (first hit order is reported), then
// Huh v2's signature defaults (not in tuilint.py): the Charm theme's indigo and
// fuchsia, which every unthemed form shows in its titles, selectors and buttons,
// and which the default spinner shows. Huh's cream, greens and reds are left
// out: a leaked theme already shows the three below, and those are neutrals and
// status hues that real brands land on (moss's light bg is within 3 of cream).
var defaultColours = []struct {
	c     RGB
	label string
}{
	{RGB{0x7D, 0x56, 0xF4}, "#7D56F4 (lipgloss README purple)"},
	{RGB{0xFF, 0x5F, 0xAF}, "#FF5FAF / 205 (bubbles default pink)"},
	{RGB{0xEE, 0x6F, 0xF8}, "#EE6FF8 (list default selected)"},
	{RGB{0x87, 0x4B, 0xFD}, "#874BFD (lipgloss example border)"},
	{RGB{0xAD, 0x58, 0xB4}, "#AD58B4 (list default selected desc)"},
	{RGB{0x5f, 0x5f, 0xd7}, "62 (example purple)"},
	{RGB{0x5f, 0x5f, 0xff}, "63 (example purple)"},
	{RGB{0xd7, 0x5f, 0xd7}, "170 (example magenta)"},
	{RGB{0xff, 0x87, 0xd7}, "212 (example pink)"},
	{RGB{0x04, 0xB5, 0x75}, "#04B575 (example green)"},
	{RGB{0xF2, 0x5D, 0x94}, "#F25D94 (lipgloss example pink)"},
	{RGB{0xFF, 0x5F, 0x87}, "204 (example pink)"},
	{RGB{0xF7, 0x80, 0xE2}, "#F780E2 (huh default fuchsia: Charm theme, spinner)"},
	{RGB{0x75, 0x71, 0xF9}, "#7571F9 (huh Charm theme indigo)"},
	{RGB{0x5A, 0x56, 0xE0}, "#5A56E0 (huh Charm theme indigo, light)"},
}

const (
	corners   = "╭╮╰╯┌┐└┘╔╗╚╝┏┓┗┛"
	topLeft   = "╭┌╔┏"
	topRight  = "╮┐╗┓"
	botLeft   = "╰└╚┗"
	ruleChars = "─│╱╲━┃═║┄┈╌╍▔▁"
)

var overlayName = regexp.MustCompile(`(dialog|palette|confirm|modal|overlay|popup|help|keys)`)

var defaultFG = RGB{216, 216, 216}

func first(s string) rune {
	if s == "" {
		return -1
	}
	r, _ := utf8.DecodeRuneInString(s)
	return r
}

// isSpace is Python's str.isspace for one rune.
func isSpace(r rune) bool {
	return unicode.IsSpace(r) || (0x1c <= r && r <= 0x1f)
}

func blank(s string) bool {
	for _, r := range s {
		if !isSpace(r) {
			return false
		}
	}
	return true
}

func lum(c RGB) float64 {
	return float64(0.2126*srgbLin(c[0])) + float64(0.7152*srgbLin(c[1])) + float64(0.0722*srgbLin(c[2]))
}

func contrast(a, b RGB) float64 {
	la, lb := lum(a), lum(b)
	return (math.Max(la, lb) + 0.05) / (math.Min(la, lb) + 0.05)
}

// hls is Python 3.13's colorsys.rgb_to_hls.
func hls(r, g, b float64) (h, l, s float64) {
	maxc := math.Max(r, math.Max(g, b))
	minc := math.Min(r, math.Min(g, b))
	sumc := maxc + minc
	rangec := maxc - minc
	l = sumc / 2.0
	if minc == maxc {
		return 0, l, 0
	}
	if l <= 0.5 {
		s = rangec / sumc
	} else {
		s = rangec / (2.0 - maxc - minc)
	}
	rc := (maxc - r) / rangec
	gc := (maxc - g) / rangec
	bc := (maxc - b) / rangec
	switch {
	case r == maxc:
		h = bc - gc
	case g == maxc:
		h = 2.0 + rc - bc
	default:
		h = 4.0 + gc - rc
	}
	h = pyMod(h/6.0, 1.0)
	return h, l, s
}

// pyMod is Python's float %: the result takes the divisor's sign.
func pyMod(x, y float64) float64 {
	m := math.Mod(x, y)
	if m != 0 {
		if (y < 0) != (m < 0) {
			m += y
		}
	} else {
		m = math.Copysign(0, y)
	}
	return m
}

func hexStr(c RGB) string { return fmt.Sprintf("#%02X%02X%02X", c[0], c[1], c[2]) }

func pyIntList(xs []int) string {
	s := make([]string, len(xs))
	for i, x := range xs {
		s[i] = strconv.Itoa(x)
	}
	return "[" + strings.Join(s, ", ") + "]"
}

type lowCell struct {
	y, x   int
	cr     float64
	fg, bg RGB
}

type offCell struct {
	col  string
	y, x int
}

type corner struct {
	y, x int
	ch   rune
}

// LintFrame applies every frame rule to one parsed frame. cols and rows are
// the expected size (0 = take the widest row, any height); pal may be nil.
func LintFrame(name string, f Frame, cols, rows int, pal *Palette, bgDefault RGB) []Finding {
	var out []Finding
	add := func(rule, sev, where, msg string) {
		out = append(out, Finding{rule, sev, name, where, msg})
	}
	grid := f.Rows

	widths := make([]int, len(grid))
	W := cols
	for i, r := range grid {
		widths[i] = len(r)
		if cols == 0 && len(r) > W {
			W = len(r)
		}
	}
	var bad []int
	for i, w := range widths {
		if w != W {
			bad = append(bad, i)
		}
	}
	if len(bad) > 0 {
		add("F-WIDTH", "P0", fmt.Sprintf("row %d", bad[0]),
			fmt.Sprintf("%d rows are not %d cells wide (row %d is %d)", len(bad), W, bad[0], widths[bad[0]]))
	}
	if rows != 0 && len(grid) != rows {
		add("F-WIDTH", "P0", "frame", fmt.Sprintf("%d rows, expected %d", len(grid), rows))
	}
	for _, w := range f.Warnings {
		if strings.HasPrefix(w, "emoji") {
			add("F-EMOJI", "P1", "frame", "emoji glyphs present: use a glyph from the brand's vocabulary (✓ ● ◆ › ╱) instead")
		}
	}

	var (
		textCells, bold     int
		hues                = map[int]int{}
		levels              = map[string]bool{}
		defaults            = map[string][2]int{}
		defaultOrder        []string
		lowc                []lowCell
		offCells            []offCell
		cornersSeen         []corner
		nonblankRows        int
		edgeLeft, edgeRight int
	)
	// a frame downsampled to 256 colours holds only xterm indexes: judge it against what
	// the theme's colours downsample to, not their exact hex
	var indexed map[RGB]bool
	if pal != nil && f.Indexed {
		indexed = pal.Indexed()
	}
	for y, row := range grid {
		rules := asciiRuleCells(row)
		var line strings.Builder
		for _, c := range row {
			line.WriteString(c.Ch)
		}
		if !blank(line.String()) {
			nonblankRows++
			if len(row) > 0 && row[0].Ch != " " && row[0].Ch != "" && row[0].BG == nil {
				edgeLeft++
			}
			if len(row) > 0 && row[len(row)-1].Ch != " " && row[len(row)-1].Ch != "" && row[len(row)-1].BG == nil {
				edgeRight++
			}
		}
		for x, c := range row {
			for _, col := range []*RGB{c.FG, c.BG} {
				if col == nil {
					continue
				}
				if indexed != nil {
					// a role the terminal can only show as 204 is still the theme's role, not
					// a pasted "204"; the role's own hex is judged in the truecolour render
					if indexed[*col] {
						continue
					}
					offCells = append(offCells, offCell{hexStr(*col), y, x})
				} else if pal != nil && !pal.Near(*col) {
					offCells = append(offCells, offCell{hexStr(*col), y, x})
				}
				for _, d := range defaultColours {
					dist := 0
					for k := 0; k < 3; k++ {
						dist = max(dist, abs(col[k]-d.c[k]))
					}
					if dist <= 3 {
						if _, ok := defaults[d.label]; !ok {
							defaults[d.label] = [2]int{y, x}
							defaultOrder = append(defaultOrder, d.label)
						}
					}
				}
			}
			ch := first(c.Ch)
			if ch >= 0 && strings.ContainsRune(corners, ch) {
				cornersSeen = append(cornersSeen, corner{y, x, ch})
			}
			if ch < 0 || isSpace(ch) || strings.ContainsRune(ruleChars, ch) {
				continue
			}
			o := ch
			if (0xE000 <= o && o <= 0xF8FF) || 0xF0000 <= o {
				add("F-TOFU", "P2", fmt.Sprintf("r%dc%d", y, x),
					fmt.Sprintf("private-use glyph U+%04X (Nerd Font icon): needs a fallback for users without the font", o))
			}
			if ch == '�' {
				add("F-TOFU", "P2", fmt.Sprintf("r%dc%d", y, x), "replacement character in output (broken encoding)")
			}
			textCells++
			if c.Bold {
				bold++
			}
			fg, bg := defaultFG, bgDefault
			if c.FG != nil {
				fg = *c.FG
			}
			if c.BG != nil {
				bg = *c.BG
			}
			if c.Reverse {
				fg, bg = bg, fg
			}
			decorative := (0x2500 <= o && o <= 0x25FF) || (0x2800 <= o && o <= 0x28FF) ||
				(0x1FB00 <= o && o <= 0x1FBFF) || ch == '•' || ch == '·' || rules[x]
			if !decorative {
				if cr := contrast(fg, bg); cr < 2.3 {
					lowc = append(lowc, lowCell{y, x, cr, fg, bg})
				}
			}
			h, l, s := hls(float64(fg[0])/255, float64(fg[1])/255, float64(fg[2])/255)
			if s > 0.25 && 0.15 < l && l < 0.92 {
				hues[int(h*24)]++
			} else {
				levels[strconv.FormatFloat(lum(fg), 'f', 2, 64)] = true
			}
		}
	}
	for i, label := range defaultOrder {
		if i == 4 {
			break
		}
		p := defaults[label]
		add("F-DEFAULT", "P1", fmt.Sprintf("r%dc%d", p[0], p[1]),
			fmt.Sprintf("library-default colour %s: the 'every Bubble Tea app is pink and purple' look", label))
	}

	var tls []corner
	for _, c := range cornersSeen {
		if strings.ContainsRune(topLeft, c.ch) {
			tls = append(tls, c)
		}
	}
	if overlayName.MatchString(name) && len(tls) != 1 && (len(lowc) > 0 || len(offCells) > 0) {
		// a borderless dialog over a scrim: we can't find its edges, so don't judge the scrim
		add("F-CONTRAST", "P2", "frame", "overlay frame without a bordered dialog: contrast and palette checked only by eye here")
		lowc, offCells = nil, nil
	}
	if len(tls) == 1 && len(lowc) > 0 {
		// one box open = a dialog over a scrim: only its inside (and the footer) must be readable
		y0, x0 := tls[0].y, tls[0].x
		y1 := len(grid) - 1
		for _, c := range cornersSeen {
			if strings.ContainsRune(botLeft, c.ch) && c.x == x0 && c.y > y0 {
				y1 = c.y
				break
			}
		}
		row := grid[y0]
		x1 := len(row) - 1
		for k := x0 + 1; k < len(row); k++ {
			if closesBox(row[k].Ch) {
				x1 = k
				break
			}
		}
		last := len(grid) - 1
		inside := func(y, x int) bool { return (y0 <= y && y <= y1 && x0 <= x && x <= x1) || y == last }
		var lc []lowCell
		for _, c := range lowc {
			if inside(c.y, c.x) {
				lc = append(lc, c)
			}
		}
		var oc []offCell
		for _, c := range offCells {
			if inside(c.y, c.x) {
				oc = append(oc, c)
			}
		}
		lowc, offCells = lc, oc
	}
	offpal := map[string][2]int{}
	var offOrder []string
	for _, c := range offCells {
		if _, ok := offpal[c.col]; !ok {
			offpal[c.col] = [2]int{c.y, c.x}
			offOrder = append(offOrder, c.col)
		}
	}
	if len(offpal) > 0 {
		type kv struct {
			col string
			p   [2]int
		}
		items := make([]kv, 0, len(offpal))
		for _, k := range offOrder {
			items = append(items, kv{k, offpal[k]})
		}
		sort.SliceStable(items, func(i, j int) bool {
			a, b := items[i].p, items[j].p
			return a[0] < b[0] || (a[0] == b[0] && a[1] < b[1])
		})
		if len(items) > 6 {
			items = items[:6]
		}
		names := make([]string, len(items))
		for i, it := range items {
			names[i] = it.col
		}
		add("F-OFFPAL", "P1", fmt.Sprintf("r%dc%d", items[0].p[0], items[0].p[1]),
			fmt.Sprintf("%d colours not in the theme roles: %s", len(offpal), strings.Join(names, ", ")))
	}
	if len(lowc) > 0 {
		seen := map[int]bool{}
		var rowsHit []int
		for _, c := range lowc {
			if !seen[c.y] {
				seen[c.y] = true
				rowsHit = append(rowsHit, c.y)
			}
		}
		sort.Ints(rowsHit)
		if len(rowsHit) > 8 {
			rowsHit = rowsHit[:8]
		}
		c := lowc[0]
		add("F-CONTRAST", "P1", fmt.Sprintf("r%dc%d", c.y, c.x),
			fmt.Sprintf("%d text cells under 2.3:1 on rows %s (first %.2f:1, fg %s on bg %s)",
				len(lowc), pyIntList(rowsHit), c.cr, hexStr(c.fg), hexStr(c.bg)))
	}
	if len(tls) > 1 {
		type box struct{ y, x, x2 int }
		var boxes []box
		for _, c := range tls {
			row := grid[c.y]
			x2 := c.x + 1
			for x2 < len(row) {
				if closesBox(row[x2].Ch) {
					break
				}
				x2++
			}
			boxes = append(boxes, box{c.y, c.x, x2})
		}
		nested, adjacent := false, false
		for _, a := range boxes {
			for _, b := range boxes {
				if a == b {
					continue
				}
				if a.y < b.y && a.x < b.x && b.x2 < a.x2 {
					nested = true
				}
				if abs(a.x2-b.x) <= 1 && a.y == b.y {
					adjacent = true
				}
			}
		}
		msg := fmt.Sprintf("%d bordered boxes", len(tls))
		if nested {
			msg += " (nested)"
		}
		if adjacent {
			msg += " (side by side)"
		}
		add("F-BOXES", "P1", fmt.Sprintf("r%dc%d", tls[0].y, tls[0].x),
			msg+": separate regions with whitespace, rules and background steps; keep borders for the one thing that floats (a dialog)")
	}
	big := 0
	for _, n := range hues {
		if n >= 4 {
			big++
		}
	}
	if big > 7 {
		add("F-RAINBOW", "P2", "frame", fmt.Sprintf("%d distinct foreground hues: hierarchy should come from lightness levels, colour from roles", big))
	}
	if textCells > 400 && len(levels) < 3 {
		add("F-LEVELS", "P2", "frame", fmt.Sprintf("only %d neutral text level(s) on a dense frame: use fg / fgMuted / fgSubtle", len(levels)))
	}
	if textCells > 0 && float64(bold)/float64(textCells) > 0.30 {
		add("F-BOLD", "P2", "frame", fmt.Sprintf("%d%% of text is bold", 100*bold/textCells))
	}
	if nonblankRows > 0 && (float64(edgeLeft) > 0.6*float64(nonblankRows) || float64(edgeRight) > 0.6*float64(nonblankRows)) {
		add("F-EDGE", "P2", "frame", "content touches the terminal edge on most rows: give it a 1–2 cell gutter")
	}
	total := 0
	for _, w := range widths {
		total += w
	}
	if total == 0 {
		total = 1
	}
	if float64(textCells)/float64(total) < 0.015 {
		add("F-EMPTY", "P2", "frame", "almost nothing on screen: if this is an empty state, say what is missing and what to do")
	}
	return out
}

// closesBox is tuilint.py's top-right test, `ch[:1] in "╮┐╗┓"`, which a
// wide glyph's empty continuation cell also passes.
func closesBox(ch string) bool {
	r := first(ch)
	return r < 0 || strings.ContainsRune(topRight, r)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// asciiRule is what the ASCII fallbacks turn ─ ━ ═ ┄ ╌ ╱ ╲ into.
const asciiRule = "-=~_/\\"

// asciiRuleCells are the columns of a row that belong to an ASCII rule: a run of 3+ of
// one rule character in one colour ("-----", "===="), or 4+ dots (a dotted rule; "..."
// is an ellipsis). A rule drawn in the border role is decoration in ASCII mode exactly
// as ─ is in Unicode, so it is not judged as text.
func asciiRuleCells(row []Cell) map[int]bool {
	out := map[int]bool{}
	head := func(c Cell) rune {
		for _, r := range c.Ch {
			return r
		}
		return -1
	}
	sameFG := func(a, b *RGB) bool { return (a == nil) == (b == nil) && (a == nil || *a == *b) }
	for x := 0; x < len(row); {
		ch := head(row[x])
		if ch < 0 || !(strings.ContainsRune(asciiRule, ch) || ch == '.') {
			x++
			continue
		}
		e := x
		for e < len(row) && head(row[e]) == ch && sameFG(row[e].FG, row[x].FG) {
			e++
		}
		min := 3
		if ch == '.' {
			min = 4
		}
		if e-x >= min {
			for i := x; i < e; i++ {
				out[i] = true
			}
		}
		x = e
	}
	return out
}

var (
	sgr256  = regexp.MustCompile(`[34]8[;:]5[;:]`)
	sgrTrue = regexp.MustCompile(`[34]8[;:]2[;:]`)
)

// lintText lints one frame's text, named by the file's base name. ascii adds F-ASCII.
func lintText(path, text string, cols, rows int, pal *Palette, bg RGB, ascii bool) []Finding {
	text = strings.ReplaceAll(text, "\r\n", "\n") // Python's universal newlines
	text = strings.ReplaceAll(text, "\r", "\n")
	text = strings.TrimSuffix(text, "\n")
	f := Parse(text)
	f.Indexed = sgr256.MatchString(text) && !sgrTrue.MatchString(text)
	out := LintFrame(filepath.Base(path), f, cols, rows, pal, bg)
	if ascii {
		out = append(out, LintASCII(filepath.Base(path), f)...)
	}
	return out
}

// LintASCII is F-ASCII: in a frame rendered for ASCII (TERM=linux, --ascii) every cell
// is ASCII. A glyph outside it is one the theme's ASCII map didn't cover (a project
// glyph with no twin) or one a screen typed as a literal; a console font draws it as
// tofu. One finding per frame: where the first is, and up to 6 of the glyphs.
func LintASCII(name string, f Frame) []Finding {
	var at [2]int
	var glyphs []string
	seen := map[string]bool{}
	n := 0
	for y, row := range f.Rows {
		for x, c := range row {
			if c.Ch == "" || !nonASCII(c.Ch) {
				continue
			}
			if n == 0 {
				at = [2]int{y, x}
			}
			n++
			if !seen[c.Ch] && len(glyphs) < 6 {
				seen[c.Ch] = true
				glyphs = append(glyphs, fmt.Sprintf("%s U+%04X", c.Ch, first(c.Ch)))
			}
		}
	}
	if n == 0 {
		return nil
	}
	return []Finding{{"F-ASCII", "P1", name, fmt.Sprintf("r%dc%d", at[0], at[1]),
		fmt.Sprintf("%d cells outside ASCII in an ASCII frame (%s): give each glyph an ASCII twin (theme glyphs.ascii, or theme.ASCII's map) instead of drawing it",
			n, strings.Join(glyphs, ", "))}}
}

func nonASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > 127 {
			return true
		}
	}
	return false
}
