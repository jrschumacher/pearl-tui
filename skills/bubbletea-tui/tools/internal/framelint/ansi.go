package framelint

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"
)

// RGB is a colour as parsed. Channels are ints, not bytes: an out-of-range
// SGR value is kept as written, the way cellshot.py keeps it.
type RGB [3]int

// Cell is one terminal cell. A wide grapheme fills its first cell (Width 2)
// and leaves a continuation cell with an empty Ch.
type Cell struct {
	Ch        string // the grapheme cluster; "" for a continuation cell
	Width     int    // display width of Ch: 0 (continuation), 1 or 2
	FG, BG    *RGB   // nil is the terminal default
	Bold      bool
	Faint     bool
	Italic    bool
	Underline int // 0 none, 1 single, 2 double, 3+ styled (4:n)
	Strike    bool
	Reverse   bool
}

// Frame is a parsed flat render: rows of cells plus what the parser noticed.
type Frame struct {
	Rows     [][]Cell
	Indexed  bool     // only 256-colour SGR (38;5 / 48;5), no truecolour: -profile ansi256
	Basic    bool     // only 16-colour SGR (30–37, 90–97, 40–47, 100–107): -profile ansi
	Emoji    bool     // an emoji codepoint was drawn
	Warnings []string // sorted, as cellshot.py reports them
}

type style struct {
	fg, bg                           *RGB
	bold, faint, italic, strike, rev bool
	under                            int
}

func (s style) cell(ch string, w int) Cell {
	return Cell{Ch: ch, Width: w, FG: s.fg, BG: s.bg, Bold: s.bold, Faint: s.faint, Italic: s.italic,
		Underline: s.under, Strike: s.strike, Reverse: s.rev}
}

// token matches the escape sequences cellshot.py recognises, in its order:
// CSI (no intermediates), OSC, DCS/SOS/PM/APC, then any two-byte ESC.
var token = regexp.MustCompile(`^(?:\x1b\[([0-9;:?<>=]*)([@-~])|\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)|\x1b[PX^_][^\x1b]*\x1b\\|\x1b[@-Z\\-_])`)

// Parse reads a flat truecolor (or 256/16-colour) ANSI render into cells.
// Widths come from charmbracelet/x/ansi grapheme widths, the same maths
// lipgloss.Width uses, so a row here is as wide as the kit measures it.
// C0 controls other than \n, \r and \t are dropped; \t pads to the next
// multiple of 8.
func Parse(text string) Frame {
	text = strings.ToValidUTF8(text, "�")
	var (
		rows  [][]Cell
		row   []Cell
		st    style
		warns = map[string]bool{}
		emoji bool
	)
	i := 0
	for i < len(text) {
		c := text[i]
		if c == 0x1b {
			m := token.FindStringSubmatchIndex(text[i:])
			if m == nil {
				i++
				continue
			}
			if m[4] >= 0 {
				final := text[i+m[4]]
				if final == 'm' {
					sgr(text[i+m[2]:i+m[3]], &st)
				} else if final != 'h' && final != 'l' {
					warns["non-SGR CSI '"+string(final)+"' ignored (cursor movement? frame may not be a flat render)"] = true
				}
			}
			i += m[1]
			continue
		}
		switch {
		case c == '\n':
			rows = append(rows, row)
			row = nil
			i++
			continue
		case c == '\r':
			i++
			continue
		case c == '\t':
			for n := 8 - len(row)%8; n > 0; n-- {
				row = append(row, st.cell(" ", 1))
			}
			i++
			continue
		case c < 0x20 || c == 0x7f:
			i++
			continue
		}
		// a printable run ends at the next ESC or C0 control
		end := i
		for end < len(text) && text[end] >= 0x20 && text[end] != 0x7f {
			end++
		}
		run := text[i:end]
		for len(run) > 0 {
			cl, w := ansi.FirstGraphemeCluster(run, ansi.GraphemeWidth)
			run = run[len(cl):]
			for _, r := range cl {
				if isEmoji(r) {
					emoji = true
				}
			}
			if w == 0 {
				if len(row) > 0 {
					row[len(row)-1].Ch += cl
				}
				continue
			}
			row = append(row, st.cell(cl, w))
			for k := 1; k < w; k++ {
				row = append(row, st.cell("", 0))
			}
		}
		i = end
	}
	if len(row) > 0 {
		rows = append(rows, row)
	}
	if emoji {
		warns["emoji present (rendering varies by terminal and font; usually a design smell)"] = true
	}
	f := Frame{Rows: rows, Emoji: emoji}
	for w := range warns {
		f.Warnings = append(f.Warnings, w)
	}
	sort.Strings(f.Warnings)
	return f
}

// isEmoji is cellshot.py's test: the pictograph planes, or a wide symbol in
// U+2600–27BF (a narrow ✓ or ★ is a text glyph). The Yijing trigrams and
// digrams (☰ ⚊) turned wide in Unicode 16 but are not emoji.
func isEmoji(r rune) bool {
	if 0x1F300 <= r && r <= 0x1FAFF {
		return true
	}
	if (0x2630 <= r && r <= 0x2637) || (0x268A <= r && r <= 0x268F) {
		return false
	}
	return 0x2600 <= r && r <= 0x27BF && runeWidth(r) == 2
}

func runeWidth(r rune) int {
	var b [utf8.UTFMax]byte
	n := utf8.EncodeRune(b[:], r)
	_, w := ansi.FirstGraphemeCluster(string(b[:n]), ansi.GraphemeWidth)
	return w
}

// basic is the 16-colour palette cellshot.py draws with.
var basic = [16]RGB{{0, 0, 0}, {205, 49, 49}, {13, 188, 121}, {229, 229, 16}, {36, 114, 200}, {188, 63, 188},
	{17, 168, 205}, {229, 229, 229}, {102, 102, 102}, {241, 76, 76}, {35, 209, 139}, {245, 245, 67},
	{59, 142, 234}, {214, 112, 214}, {41, 184, 219}, {255, 255, 255}}

func xterm(n int) RGB {
	if n < 16 {
		if n < 0 {
			n = 0
		}
		return basic[n]
	}
	if n < 232 {
		n -= 16
		c := RGB{n / 36, (n / 6) % 6, n % 6}
		for k, v := range c {
			if v != 0 {
				c[k] = 55 + v*40
			}
		}
		return c
	}
	v := 8 + (n-232)*10
	return RGB{v, v, v}
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

// sgr applies one SGR parameter string to the pen, as cellshot.py does.
func sgr(params string, st *style) {
	codes := []string{"0"}
	if params != "" {
		codes = strings.Split(params, ";")
	}
	for j := 0; j < len(codes); j++ {
		part := codes[j]
		if strings.Contains(part, ":") { // 38:2::r:g:b, 4:3
			sub := strings.Split(part, ":")
			switch head := atoi(sub[0]); head {
			case 38, 48, 58:
				var vals []int
				for _, s := range sub[1:] {
					if s != "" {
						vals = append(vals, atoi(s))
					}
				}
				var col *RGB
				if len(vals) >= 2 && vals[0] == 5 {
					c := xterm(vals[1])
					col = &c
				} else if len(vals) >= 4 && vals[0] == 2 {
					c := RGB{vals[len(vals)-3], vals[len(vals)-2], vals[len(vals)-1]}
					col = &c
				}
				if head == 38 {
					st.fg = col
				} else if head == 48 {
					st.bg = col
				}
			case 4:
				st.under = 1
				if sub[1] != "" {
					st.under = atoi(sub[1])
				}
			}
			continue
		}
		c := atoi(part)
		switch {
		case c == 0:
			*st = style{}
		case c == 1:
			st.bold = true
		case c == 2:
			st.faint = true
		case c == 3:
			st.italic = true
		case c == 4:
			st.under = 1
		case c == 21:
			st.under = 2
		case c == 7:
			st.rev = true
		case c == 9:
			st.strike = true
		case c == 22:
			st.bold, st.faint = false, false
		case c == 23:
			st.italic = false
		case c == 24:
			st.under = 0
		case c == 27:
			st.rev = false
		case c == 29:
			st.strike = false
		case 30 <= c && c <= 37:
			col := basic[c-30]
			st.fg = &col
		case 90 <= c && c <= 97:
			col := basic[c-90+8]
			st.fg = &col
		case 40 <= c && c <= 47:
			col := basic[c-40]
			st.bg = &col
		case 100 <= c && c <= 107:
			col := basic[c-100+8]
			st.bg = &col
		case c == 39:
			st.fg = nil
		case c == 49:
			st.bg = nil
		case (c == 38 || c == 48 || c == 58) && j+1 < len(codes):
			var col *RGB
			switch atoi(codes[j+1]) {
			case 5:
				if j+2 < len(codes) {
					x := xterm(atoi(codes[j+2]))
					col = &x
					j += 2
				}
			case 2:
				if j+4 < len(codes) {
					x := RGB{atoi(codes[j+2]), atoi(codes[j+3]), atoi(codes[j+4])}
					col = &x
					j += 4
				}
			}
			if c == 38 {
				st.fg = col
			} else if c == 48 {
				st.bg = col
			}
		}
	}
}
