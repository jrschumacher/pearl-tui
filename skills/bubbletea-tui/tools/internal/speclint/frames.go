package speclint

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"tuitools/internal/framelint"
)

// frameHint is one `key action` hint read off a golden frame's footer row.
type frameHint struct {
	text string
	keys []key
}

// frameFooter reads the footer of a golden frame: its last non-blank row, split where
// two or more blank cells fall, as ui.Help joins hints. A chunk is a hint when its first
// word reads as keys and is drawn in another style than the word after it (ui.Help's
// Key and KeyDesc), so status text on the row ("5 variables") is not one; a row with
// no style at all (NO_COLOR) takes every chunk that reads as keys. cut is a lone
// ellipsis among the chunks: Help dropped hints behind it.
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
	for _, ch := range chunks {
		var sb strings.Builder
		for _, c := range ch {
			sb.WriteString(c.Ch)
		}
		s := strings.TrimSpace(sb.String())
		if s == "…" || s == "..." {
			cut = true
			continue
		}
		words := strings.Fields(s)
		if len(words) < 2 {
			continue
		}
		words[0] = unASCII(words[0])
		b := readItem(words)
		if b.bad != "" || len(b.keys) == 0 || b.action == "" {
			continue
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
				continue
			}
		}
		hints = append(hints, frameHint{text: s, keys: b.keys})
	}
	return hints, cut
}

// asciiArrows are the arrow keys as an ASCII frame draws them (theme.ASCIITwin): "^v"
// would read as ctrl+v.
var asciiArrows = map[string]string{"^v": "↑↓", "<>": "←→", "^": "↑", "<-": "←", "->": "→"}

// unASCII turns a key token's ASCII arrow twins back into arrows, alternative by
// alternative ("^v/jk" is ↑↓/jk).
func unASCII(tok string) string {
	alts := splitAlts(tok)
	for i, a := range alts {
		if r, ok := asciiArrows[a]; ok {
			alts[i] = r
		}
	}
	return strings.Join(alts, "/")
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
		bound := map[key]bool{}
		for _, k := range global {
			bound[k] = true
		}
		for _, b := range keysOf.bindings {
			for _, k := range b.keys {
				bound[k] = true
			}
		}
		shown := map[key]bool{}
		var extra []string
		for _, h := range hints {
			any := false
			for _, k := range h.keys {
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
					any = any || shown[k]
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
			parts = append(parts, fmt.Sprintf("lacks %s, which the row puts in the footer", strings.Join(missing, ", ")))
		}
		if len(parts) > 0 {
			add("S-GOLDEN", r.line, "%s: the footer of %s %s: the footer and the table disagree, so one of them lies. Make the state's actions() and this row the same keys, then re-render",
				r.state, filepath.Join(filepath.Base(dir), name), strings.Join(parts, ", and "))
		}
	}
}
