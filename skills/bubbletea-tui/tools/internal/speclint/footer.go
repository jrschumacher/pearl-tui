package speclint

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// footerCut in a keys cell ends what the footer shows: the items before it are the
// footer, the items after it are help-only (`show: false` in the action table, listed
// by the ? overlay).
const footerCut = '‖'

// How the kit draws a footer (kit/ui/chrome.go Help and Footer, kit/theme/derive.go),
// at the 80-column design width. kit/ui/testdata/footer-cases.json holds packFooter and
// the kit's ui.Footer to the same dropped sets (TestFooterCases in both modules).
const (
	footerWidth  = 80
	kitGutter    = 2           // theme default side margin, on both sides; a theme may set 1
	hintSep      = 2           // Help joins hints with two spaces
	moreReserve  = hintSep + 1 // room Help keeps for "  …" while hints remain
	asciiReserve = hintSep + 3 // and for "  ..." when the theme is resolved for ASCII
	footerBudget = footerWidth - 2*kitGutter
)

// badgeItem is a keys cell's first item when it is a footer badge, a bracketed word
// alone (`[plan]`, **[PROD]**): ui.Footer's left, drawn before the keys. "[ ] back" is
// keys, not a badge.
var badgeItem = regexp.MustCompile("^\\s*((?:\\*\\*|`)?\\[[^\\]\\s][^\\]]*\\](?:\\*\\*|`)?)\\s*(?:[·•]|$)")

// cutBadge splits a keys cell into the badge it names first (as the footer draws it,
// markup removed) and the rest of the cell.
func cutBadge(cell string) (badge, rest string) {
	m := badgeItem.FindStringSubmatchIndex(cell)
	if m == nil {
		return "", cell
	}
	return clean(cell[m[2]:m[3]]), cell[m[1]:]
}

// footerItems splits a keys cell, as written, into the items the footer shows and the
// help-only rest, each as the kit renders a hint: `key desc`, single-spaced. A cell
// marks its footer one of two ways: its **bold** items (in table order), or the items
// before a ‖. With neither, every item is footer. mixed is a cell that uses both; the
// ‖ wins.
func footerItems(raw string) (footer, helpOnly []string, marker string, mixed bool) {
	if before, after, ok := strings.Cut(raw, string(footerCut)); ok {
		return hints(before), keyHints(after), string(footerCut), len(boldItems(raw)) > 0
	}
	bold := boldItems(raw)
	if len(bold) == 0 {
		return hints(raw), nil, "", false
	}
	for i, it := range splitItems(raw) {
		if strings.TrimSpace(it) == "" {
			continue
		}
		if bold[i] {
			footer = append(footer, hints(it)...)
		} else {
			helpOnly = append(helpOnly, keyHints(it)...)
		}
	}
	return footer, helpOnly, "bold", false
}

// boldItems indexes the items of a raw cell (as splitItems returns them) that are
// written bold: `**? help**`, or `**?** help` with the key alone bold.
func boldItems(raw string) map[int]bool {
	m := map[int]bool{}
	for i, it := range splitItems(raw) {
		if strings.HasPrefix(strings.TrimSpace(it), "**") {
			m[i] = true
		}
	}
	return m
}

func hints(part string) []string {
	part = clean(part)
	if emptyCell.MatchString(part) {
		return nil
	}
	var out []string
	for _, it := range splitItems(part) {
		it, _ = cutCond(it)
		if f := strings.Fields(it); len(f) > 0 {
			out = append(out, strings.Join(f, " "))
		}
	}
	return out
}

// keyHints is hints without prose items ("type to filter"): a line that describes the
// text box names no key, so leaving it out of the footer hides nothing.
func keyHints(part string) []string {
	var out []string
	for _, h := range hints(part) {
		if f := strings.Fields(h); !proseLead[strings.ToLower(strings.Trim(f[0], "(),:"))] {
			out = append(out, h)
		}
	}
	return out
}

var helpAction = regexp.MustCompile(`(?i)\bhelp\b|^(?:full |all )?keys$`)

// opensHelp reports an item that opens full help: Help pins `?` and ctrl+g last.
func opensHelp(item string) bool {
	b := readItem(strings.Fields(item))
	for _, k := range b.keys {
		if k.String() == "?" || k.String() == "ctrl+g" {
			return true
		}
	}
	return helpAction.MatchString(b.action)
}

// helpState is the help overlay itself, full help: it needs no key that opens it.
var helpState = regexp.MustCompile(`(?i)^(?:help|keys|shortcuts)(?:\b|$)`)

// drawnHint is an item as the footer draws it. A help item that lists aliases for the
// dispatcher (`?/h/f1 help`, `ctrl+g/f1 help`) is drawn as its first key, `? help`,
// the hint ui.Help pins last; any other item is drawn as written.
func drawnHint(item string) string {
	tok, desc, ok := strings.Cut(item, " ")
	alts := splitAlts(tok)
	if !ok || len(alts) < 2 || (alts[0] != "?" && alts[0] != "ctrl+g") {
		return item
	}
	return alts[0] + " " + desc
}

func drawnHints(items []string) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = drawnHint(it)
	}
	return out
}

// packFooter lays items out as ui.Help does in w columns and returns the ones it
// drops behind "…". A first item wider than w is cut, so it counts as dropped too.
func packFooter(items []string, w int) (shown, dropped []string) {
	return packFooterReserving(items, w, moreReserve)
}

// packFooterReserving is packFooter keeping reserve columns for the separator and
// ellipsis while hints remain: moreReserve, or asciiReserve for an ASCII theme.
func packFooterReserving(items []string, w, reserveMore int) (shown, dropped []string) {
	var pinned string
	for i, it := range items {
		if len(items) > 1 && pins(it) {
			pinned = it
			w -= hintWidth(pinned) + hintSep
			items = append(items[:i:i], items[i+1:]...)
			break
		}
	}
	used := 0
	for i, it := range items {
		need := hintWidth(it)
		if len(shown) > 0 {
			need += hintSep
		}
		reserve := 0
		if i < len(items)-1 {
			reserve = reserveMore
		}
		if used+need+reserve > w && len(shown) > 0 {
			dropped = items[i:]
			break
		}
		shown = append(shown, it)
		used += need
	}
	if used > w && len(dropped) == 0 && len(shown) > 0 {
		dropped = shown // the only hint, cut by Fit
		shown = nil
	}
	if pinned != "" {
		shown = append(shown, pinned)
	}
	return shown, dropped
}

// pins is Help's own test: it pins last the first hint whose key is exactly ? or
// ctrl+g, wherever the cell lists it.
func pins(item string) bool {
	f := strings.Fields(item)
	return len(f) > 0 && (f[0] == "?" || f[0] == "ctrl+g")
}

func hintsWidth(items []string) int {
	n := 0
	for i, it := range items {
		if i > 0 {
			n += hintSep
		}
		n += hintWidth(it)
	}
	return n
}

// hintWidth is the width ui.Help draws a hint in. A key token that pairs a key some
// terminals can't tell apart (legacy) with its fallback, `shift+enter/ctrl+j`, is drawn
// as one alternative (ui.HelpKey: ctrl+j, or shift+enter where the terminal
// disambiguates keys), so it counts as the widest single one; any other token is drawn
// whole.
func hintWidth(item string) int {
	tok, desc, _ := strings.Cut(item, " ")
	alts := splitAlts(tok)
	widest, ambiguous := 0, false
	for _, a := range alts {
		for _, k := range expandAlt(a) {
			_, _, ok := legacy(k)
			ambiguous = ambiguous || !ok
		}
		widest = max(widest, ansi.StringWidth(a))
	}
	if len(alts) < 2 || !ambiguous {
		return ansi.StringWidth(item)
	}
	return widest + 1 + ansi.StringWidth(desc)
}

// footerFit checks each state's footer against the 80-column footer the kit draws,
// and that a state which hides keys behind ‖ still shows how to reach them.
func (s *spec) footerFit(add func(string, int, string, ...any)) {
	globalHelp := false
	for _, g := range s.globals {
		for _, b := range g.bindings {
			globalHelp = globalHelp || opensHelp(b.token+" "+b.action)
		}
	}
	for _, r := range s.rows {
		budget, beside := footerBudget, ""
		if r.badge != "" { // ui.Footer draws the badge, then a space, then the keys
			budget -= ansi.StringWidth(r.badge) + 1
			beside = fmt.Sprintf(", %d beside its `%s` badge", budget, r.badge)
		}
		if len(r.footer) == 0 {
			continue // no footer row (toosmall through ui.TooSmallText): nothing to fit or find
		}
		_, dropped := packFooter(drawnHints(r.footer), budget)
		if len(dropped) > 0 {
			what, fix := "its footer keys are", "Bold the keys the footer shows (or put ‖ after them)"
			switch r.marker {
			case "bold":
				what, fix = "its bold footer keys are still", "Bold fewer keys"
			case string(footerCut):
				what, fix = "its footer keys before ‖ are still", "Move ‖ left"
			}
			front, back := suggestCut(r.footer, budget)
			eg := strings.Join(front, " · ") + " ‖ " + strings.Join(append(back, r.helpOnly...), " · ")
			if r.marker != string(footerCut) {
				eg = boldCell(front, append(back, r.helpOnly...))
			}
			add("S-FOOTER", r.line, "%s: %s %d columns, and an %d-column footer has %d inside the kit's default %d-cell gutters (a theme with a 1-cell gutter has 2 more)%s, so ui.Help drops `%s` behind …. %s, the state's main actions and its way out first; the other keys are help-only (show: false in the action table; ? help lists them), e.g. `%s`",
				r.state, what, hintsWidth(drawnHints(r.footer)), footerWidth, footerBudget, kitGutter, beside,
				strings.Join(dropped, " · "), fix, eg)
			continue
		}
		if r.marker == "" || len(r.helpOnly) == 0 || globalHelp || helpState.MatchString(r.state) {
			continue
		}
		help := false
		for _, it := range r.footer {
			help = help || opensHelp(it)
		}
		if !help {
			add("S-FOOTER", r.line, "%s: its help-only keys are reachable only from full help, but its footer shows no key that opens help, so nobody finds them. End the footer with `? help` (ctrl+g while a text box has focus)",
				r.state)
		}
	}
}

var (
	wayOutAction = regexp.MustCompile(`(?i)^(?:quit|exit|back|close|cancel|stop)\b`)
	navAction    = regexp.MustCompile(`(?i)\b(?:move|scroll|page|top|bottom|next|previous|prev|jump|choose)\b`)
	navKeys      = map[string]bool{"up": true, "down": true, "left": true, "right": true, "j": true, "k": true, "h": true, "l": true,
		"pgup": true, "pgdown": true, "home": true, "end": true, "tab": true, "shift+tab": true, "g": true, "G": true, "ctrl+d": true, "ctrl+u": true}
)

// wayOut reports an item that leaves the state: esc, or an action that starts with
// quit, exit, back, close, cancel or stop ("r roll back" is not one).
func wayOut(item string) bool {
	b := readItem(strings.Fields(item))
	for _, k := range b.keys {
		if k.base == "esc" {
			return true
		}
	}
	return wayOutAction.MatchString(b.action)
}

// navigation reports an item that only moves around the state.
func navigation(item string) bool {
	b := readItem(strings.Fields(item))
	if navAction.MatchString(b.action) {
		return true
	}
	for _, k := range b.keys {
		if !navKeys[k.String()] {
			return false
		}
	}
	return len(b.keys) > 0
}

// suggestCut picks a footer that fits: the help key last (or `? help` added), the ways
// out and the state's only action other than navigation kept, and the other items
// moved behind ‖ from the end of the cell, so the cut hides secondary keys first.
func suggestCut(items []string, budget int) (front, back []string) {
	help := ""
	var rest []string
	for _, it := range items {
		if help == "" && opensHelp(it) {
			help = it
			continue
		}
		rest = append(rest, it)
	}
	if help == "" {
		help = "? help"
	}
	keep := make([]bool, len(rest))
	main := -1
	for i, it := range rest {
		switch {
		case wayOut(it):
			keep[i] = true
		case !navigation(it):
			if main == -1 {
				main = i
			} else {
				main = -2 // more than one: none of them is the state's purpose by itself
			}
		}
	}
	if main >= 0 {
		keep[main] = true
	}
	in := make([]bool, len(rest))
	for i := range in {
		in[i] = true
	}
	fits := func() bool {
		var f []string
		for i, it := range rest {
			if in[i] {
				f = append(f, it)
			}
		}
		_, d := packFooter(drawnHints(append(f, help)), budget)
		return len(d) == 0
	}
	// drop unkept items from the end first, then kept ones if that is still too wide
	for _, pass := range []bool{false, true} {
		for i := len(rest) - 1; i >= 0 && !fits(); i-- {
			if in[i] && keep[i] == pass {
				in[i] = false
			}
		}
	}
	for i, it := range rest {
		if in[i] {
			front = append(front, it)
		} else {
			back = append(back, it)
		}
	}
	return append(front, help), back
}

// boldCell writes a suggestion in the bold form: footer items bold, the rest plain.
func boldCell(front, back []string) string {
	var out []string
	for _, it := range front {
		out = append(out, "**"+it+"**")
	}
	return strings.Join(append(out, back...), " · ")
}
