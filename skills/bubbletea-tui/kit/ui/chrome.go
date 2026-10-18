package ui

import (
	"slices"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"tuikit/theme"
)

// Wordmark renders the brand name per expressions.wordmark (one row).
func Wordmark(t *theme.Theme, name string) string {
	p := t.P
	switch t.X.Wordmark {
	case "solid":
		return t.S.Primary.Bold(true).Render(name)
	case "inverse":
		return lipgloss.NewStyle().Background(p.Primary).Foreground(p.OnPrimary).Bold(true).Padding(0, 1).Render(name)
	case "tracked":
		return t.S.Bold.Foreground(p.Primary).Render(strings.Join(strings.Split(strings.ToUpper(name), ""), " "))
	case "bracket":
		return t.S.Subtle.Render("[") + t.S.Bold.Render(name) + t.S.Subtle.Render("]")
	case "upper":
		return t.S.Primary.Bold(true).Render(strings.ToUpper(name))
	case "plain": // the name in fg, bold, no hue
		return t.S.Bold.Render(name)
	}
	if t.ASCII {
		return t.S.Bold.Render(name)
	}
	return theme.Grad(name, p.Secondary, p.Primary, true)
}

// Header is the app's top row(s), exactly w cells wide, per expressions.header:
//
//	rule     brand ╱╱╱╱╱╱╱╱╱╱╱╱╱ details     (Crush's compact header)
//	bar      a filled primary strip with the brand and details
//	minimal  brand  details
//	stamp    details at the left, the brand as an inverse stamp at the right
//	banner   the multi-row wordmark (theme.wordmark.rows) then the details row
//
// details is pre-styled (use Details to build it). h is the terminal height:
// a banner costs rows, so below 30 rows it collapses to the rule header. right is an
// optional pre-styled part at the right edge (a count, a status) that minimal draws
// after "brand · details"; the other headers carry details at the right already.
func Header(t *theme.Theme, w, h int, brand, details string, right ...string) []string {
	g := t.X.Gutter
	inner := w - 2*g
	pad := strings.Repeat(" ", g)
	switch t.X.Header {
	case "bar":
		// every span carries the bar's background: an inner reset would punch a
		// hole in the strip
		bar := lipgloss.NewStyle().Background(t.P.Primary).Foreground(t.P.OnPrimary)
		name := t.Case(brand)
		if t.X.Wordmark == "tracked" {
			name = strings.Join(strings.Split(strings.ToUpper(brand), ""), " ")
		}
		d := ansi.Strip(details)
		gap := max(1, inner-lipgloss.Width(name)-lipgloss.Width(d))
		if lipgloss.Width(name)+lipgloss.Width(d)+1 > inner {
			d = ""
			gap = max(0, inner-lipgloss.Width(name))
		}
		return []string{bar.Render(pad) + bar.Bold(true).Render(name) + bar.Render(strings.Repeat(" ", gap)+d+pad)}
	case "minimal":
		left := pad + Wordmark(t, brand)
		if details != "" {
			left += t.S.Subtle.Render(" "+t.G.Sep+" ") + details
		}
		r := strings.Join(right, " ")
		if r == "" {
			return []string{Fit(left, w)}
		}
		room := inner - lipgloss.Width(r) - 2 // the right part keeps 2 cells from the details
		left = Trunc(t, left, max(0, room+g))
		return []string{Fit(left+strings.Repeat(" ", max(0, w-g-lipgloss.Width(left)-lipgloss.Width(r)))+r, w-g) + pad}
	case "stamp":
		st := lipgloss.NewStyle().Background(t.P.Primary).Foreground(t.P.OnPrimary).Bold(true).Padding(0, 1).Render(strings.ToUpper(brand))
		gap := max(1, inner-lipgloss.Width(details)-lipgloss.Width(st))
		return []string{Fit(pad+details+strings.Repeat(" ", gap)+st, w)}
	case "banner":
		if rows := t.Wordmark.Rows; len(rows) > 0 && lipgloss.Width(rows[0]) <= inner && h >= 30 {
			var out []string
			for _, r := range rows {
				out = append(out, Fit(pad+bannerRow(t, r), w))
			}
			return append(out, Fit(pad+details, w))
		}
	}
	// rule (default, and banner's fallback when too narrow)
	b := Wordmark(t, brand)
	used := lipgloss.Width(b) + 1 + lipgloss.Width(details) + 1
	n := inner - used
	fill := strings.Repeat(" ", max(0, n))
	if t.G.HeaderRule != "" && n > 0 {
		fill = t.S.Motif.Render(strings.Repeat(t.G.HeaderRule, n))
	}
	if n < 3 { // too narrow for the rule: drop details first, keep the brand
		return []string{Fit(pad+b+" "+details, w)}
	}
	return []string{Fit(pad+b+" "+fill+" "+details, w)}
}

func bannerRow(t *theme.Theme, r string) string {
	if !strings.ContainsAny(r, theme.ShadeDark+theme.ShadeMid) {
		return t.S.Primary.Render(r)
	}
	var sb strings.Builder
	for _, ch := range r {
		switch string(ch) {
		case theme.ShadeFull:
			sb.WriteString(t.S.Accent.Render(theme.ShadeFull))
		case theme.ShadeDark:
			sb.WriteString(t.S.Primary.Render(theme.ShadeDark))
		case theme.ShadeMid:
			sb.WriteString(t.S.Muted.Render(theme.ShadeMid))
		default:
			sb.WriteRune(ch)
		}
	}
	return sb.String()
}

// Details joins header details with the theme's separator: the first part in
// fg, the rest muted ("checkout · main · 3 alerts").
func Details(t *theme.Theme, parts ...string) string {
	var out []string
	for i, p := range parts {
		if p == "" {
			continue
		}
		if i == 0 {
			out = append(out, t.S.Soft.Render(p))
		} else {
			out = append(out, t.S.Muted.Render(p))
		}
	}
	return strings.Join(out, t.S.Subtle.Render(" "+t.G.Sep+" "))
}

// Section is "Title ───────── info": a title then a rule to the width.
// The rule is the region separator; boxes are not. Info is fgSubtle, and a span of it
// the caller styles keeps its own ink (t.S.Err.Render(t.G.Fail)+" 4 of 4 pinned", a
// t.S.Base figure, a bold filter), every unstyled span after it subtle again, so a
// section never has to be rebuilt to colour its figure.
func Section(t *theme.Theme, title string, w int, info string) string {
	ti := t.S.Title.Render(t.Case(title))
	if info != "" {
		info = " " + reapply(info, ansi.Style{}.ForegroundColor(t.P.FgSubtle).String(), "39") + ansi.Style{}.DefaultForegroundColor().String()
	}
	n := w - lipgloss.Width(ti) - 1 - lipgloss.Width(info)
	if n < 1 {
		return Fit(ti+info, w)
	}
	return ti + " " + t.S.Rule.Render(strings.Repeat(t.G.Rule, n)) + info
}

// Rule is a full-width rule in the border colour.
func Rule(t *theme.Theme, w int) string {
	return t.S.Rule.Render(strings.Repeat(t.G.Rule, max(0, w)))
}

// Help renders key hints that fit w, packed greedily from the left (most
// important first). It never ends on a dangling separator, and says … when
// it had to drop hints, so the user knows there is more (full help).
func Help(t *theme.Theme, bindings []key.Binding, w int) string {
	sep := t.S.Subtle.Render("  ")
	// the key that opens full help is pinned last, wherever the table lists it: the …
	// promises more, so the way to see it must survive truncation
	pinned := ""
	if i := helpBinding(bindings); i >= 0 && len(bindings) > 1 {
		h := bindings[i].Help()
		pinned = t.S.Key.Render(HelpKey(t, h.Key)) + " " + t.S.KeyDesc.Render(h.Desc)
		w -= lipgloss.Width(pinned) + 2
		bindings = append(bindings[:i:i], bindings[i+1:]...)
	}
	var parts []string
	used := 0
	more := t.S.Subtle.Render(t.G.Ellipsis)
	hint := func(h key.Help) string { return t.S.Key.Render(HelpKey(t, h.Key)) + " " + t.S.KeyDesc.Render(h.Desc) }
	// room for the separator and the ellipsis as the theme draws it: "  …" is 3 cells,
	// the ASCII "  ..." is 5
	reserveMore := 2 + lipgloss.Width(more)
	for i, b := range bindings {
		if !b.Enabled() {
			continue
		}
		part := hint(b.Help())
		pw := lipgloss.Width(part)
		need := pw
		if len(parts) > 0 {
			need += 2
		}
		reserve := 0
		if i < len(bindings)-1 {
			reserve = reserveMore
		}
		if used+need+reserve > w && len(parts) > 0 {
			parts = append(parts, more)
			break
		}
		parts = append(parts, part)
		used += need
	}
	line := strings.Join(parts, sep)
	// a single hint wider than w is cut with the theme's ellipsis, so ASCII stays ASCII
	line = Fit(Trunc(t, line, max(0, w)), w)
	if pinned != "" {
		return line + sep + pinned
	}
	return line
}

// HelpKey is a help key as this terminal can press it. A token that pairs a key some
// terminals can't tell apart from another (shift+enter arrives as enter where the
// terminal doesn't disambiguate keys: tmux, most SSH sessions) with its fallback,
// "shift+enter/ctrl+j", is drawn as one alternative: the first, once the terminal
// reported key disambiguation (t.KeyDisambiguation), else the first every terminal
// sends. Any other token ("↑↓/jk", "?/f1") is drawn whole.
func HelpKey(t *theme.Theme, k string) string {
	alts := strings.Split(k, "/")
	if len(alts) < 2 || !slices.ContainsFunc(alts, ambiguousKey) {
		return k
	}
	if t.KeyDisambiguation {
		return alts[0]
	}
	if i := slices.IndexFunc(alts, func(a string) bool { return a != "" && !ambiguousKey(a) }); i >= 0 {
		return alts[i]
	}
	return k
}

// ambiguousKey reports a key a terminal without key disambiguation delivers as another
// key or not at all: ctrl or shift on enter, backspace or esc (one byte each), ctrl+tab,
// shift+space, ctrl+m and ctrl+i (enter's and tab's bytes), ctrl+shift+<letter>, ctrl
// on a digit or punctuation other than \ ] ^ _, and super, hyper or meta on anything.
// tuilint spec's S-KEYAMBIG holds the same model.
func ambiguousKey(k string) bool {
	parts := strings.Split(k, "+")
	base, mods := parts[len(parts)-1], parts[:len(parts)-1]
	ctrl, shift := slices.Contains(mods, "ctrl"), slices.Contains(mods, "shift")
	if slices.ContainsFunc(mods, func(m string) bool { return m == "super" || m == "hyper" || m == "meta" }) {
		return true
	}
	switch base {
	case "enter", "backspace", "esc":
		return ctrl || shift
	case "tab":
		return ctrl
	case "space":
		return shift
	}
	if !ctrl || len([]rune(base)) != 1 {
		return false
	}
	r := []rune(base)[0]
	switch {
	case r >= 'a' && r <= 'z':
		return shift || r == 'm' || r == 'i'
	case r >= 'A' && r <= 'Z':
		return true
	}
	return !strings.ContainsRune(`\]^_`, r)
}

// helpBinding is the index of the first enabled binding whose help key is ? or ctrl+g
// (the key that opens full help), or -1.
func helpBinding(bindings []key.Binding) int {
	for i, b := range bindings {
		if k := b.Help().Key; b.Enabled() && (k == "?" || k == "ctrl+g") {
			return i
		}
	}
	return -1
}

// Footer is the bottom row(s) per expressions.footer: bare keys, keys on a
// filled status strip, or a faint rule then the keys. left is an optional
// badge (mode, PLAN, YOLO) and right optional status text.
func Footer(t *theme.Theme, w int, left string, bindings []key.Binding, right string) []string {
	g := t.X.Gutter
	inner := w - 2*g
	pad := strings.Repeat(" ", g)
	lw := lipgloss.Width(left)
	if lw > 0 {
		lw++
	}
	rw := lipgloss.Width(right)
	if rw > 0 {
		rw += 2
	}
	help := Help(t, bindings, max(0, inner-lw-rw))
	row := pad
	if left != "" {
		row += left + " "
	}
	row += help
	if right != "" {
		row += "  " + right
	}
	row = Fit(row+pad, w)
	switch t.X.Footer {
	case "bar":
		// the keys are fgMuted and fgSubtle on the strip; where either lands on the
		// strip's colour (16 colours), the strip would hide them: draw the keys bare
		if !t.Distinct(t.P.FgMuted, t.P.BgSubtle) || !t.Distinct(t.P.FgSubtle, t.P.BgSubtle) {
			return []string{row}
		}
		return []string{OnBg(row, t.P.BgSubtle)}
	case "rule":
		return []string{pad + Rule(t, inner) + pad, row}
	}
	return []string{row}
}

// Toast paints over the footer row: a tag then the message on a tinted bar.
// It replaces the help line for a few seconds instead of taking a new row.
func Toast(t *theme.Theme, kind, label, msg string, w int) string {
	g := t.X.Gutter
	tag, bar := t.S.TagOK, t.S.BarOK
	switch kind {
	case "error":
		tag, bar = t.S.TagErr, t.S.BarErr
	case "warning":
		tag, bar = t.S.TagWarn, t.S.BarWarn
	case "info":
		tag, bar = t.S.TagInfo, t.S.BarInfo
	}
	tg := tag.Render(t.Label(label, "as-is"))
	rest := w - 2*g - lipgloss.Width(tg)
	return strings.Repeat(" ", g) + tg + bar.Width(max(0, rest)).Render(Trunc(t, msg, max(0, rest-2))) + strings.Repeat(" ", g)
}

// Tabs renders text tabs per expressions.tabs, the names in the theme's case (Copy).
// Never boxes.
//
//	underline  Services   Deploys   Incidents      (active bold, ▔ under it on a second row)
//	pill       the active tab on a filled pill
//	bracket    [Services]  Deploys   Incidents
//	slash      Services / Deploys / Incidents
func Tabs(t *theme.Theme, names []string, active int, counts []string) []string {
	var top, under []string
	for i, n := range names {
		label := t.Copy(n)
		if i < len(counts) && counts[i] != "" {
			c := counts[i]
			if t.X.Tabs == "pill" && i == active {
				c = ansi.Strip(c) // on the filled pill the count takes the pill's ink
			}
			label += " " + c
		}
		var s string
		switch {
		case t.X.Tabs == "pill" && i == active:
			fill, on := t.P.Primary, t.P.OnPrimary
			if t.X.Focus == "fg" { // the active tab is focus chrome
				fill, on = t.P.Fg, t.P.Bg
			}
			s = lipgloss.NewStyle().Background(fill).Foreground(on).Bold(true).Padding(0, 1).Render(label)
		case t.X.Tabs == "pill":
			s = t.S.Muted.Padding(0, 1).Render(label)
		case t.X.Tabs == "bracket" && i == active:
			s = t.S.Subtle.Render("[") + t.S.Bold.Render(label) + t.S.Subtle.Render("]")
		case t.X.Tabs == "bracket":
			s = " " + t.S.Muted.Render(label) + " "
		case i == active:
			s = t.S.Bold.Render(label)
		default:
			s = t.S.Muted.Render(label)
		}
		top = append(top, s)
		mark := " "
		if i == active {
			mark = t.G.TabUnderline
			if t.ASCII { // the theme's rule, as ASCII draws it: an underline, not a run of carets
				mark = t.G.Rule
			}
		}
		u := strings.Repeat(" ", lipgloss.Width(s))
		if i == active {
			u = t.S.Focus.Render(strings.Repeat(mark, lipgloss.Width(s)))
		}
		under = append(under, u)
	}
	gap := "   "
	if t.X.Tabs == "slash" {
		gap = t.S.Subtle.Render(" / ")
	}
	if t.X.Tabs == "pill" || t.X.Tabs == "bracket" {
		gap = " "
	}
	rows := []string{strings.Join(top, gap)}
	if t.X.Tabs == "underline" {
		ug := strings.Repeat(" ", lipgloss.Width(gap))
		rows = append(rows, strings.Join(under, ug))
	}
	return rows
}

// TooSmall is the whole screen when the terminal is below the minimum: the wordmark,
// "Terminal too small" in the theme's case, the sizes, and the keys the too-small
// state binds (its action table's footer bindings, usually one: q quit), so the hint
// is never a key the app doesn't handle. A spec with its own sentence uses TooSmallText.
func TooSmall(t *theme.Theme, w, h, minW, minH int, bindings []key.Binding) string {
	msg := append(strings.Split(Wordmark(t, t.AppName()), "\n"), "", t.S.Soft.Render(t.Sentence("Terminal too small")), t.S.Muted.Render(
		strconv.Itoa(w)+"x"+strconv.Itoa(h)+", needs "+strconv.Itoa(minW)+"x"+strconv.Itoa(minH)))
	return tooSmall(t, w, h, msg, bindings)
}

// TooSmallText is TooSmall in the product's own words: say (the spec's sentence, sizes
// and all: "Terminal is 60×18; needs 80×24"), wrapped to the width and centred, then the
// keys of bindings. No wordmark, and no key row when bindings has no enabled key: for a
// spec whose too-small screen is its sentence and nothing else.
func TooSmallText(t *theme.Theme, w, h int, say string, bindings []key.Binding) string {
	var msg []string
	for _, l := range strings.Split(lipgloss.NewStyle().Width(max(1, min(w-2, 56))).Render(say), "\n") {
		msg = append(msg, t.S.Soft.Render(strings.TrimSpace(l)))
	}
	return tooSmall(t, w, h, msg, bindings)
}

// tooSmall centres msg in w×h, with the enabled bindings' keys under it after a blank row.
func tooSmall(t *theme.Theme, w, h int, msg []string, bindings []key.Binding) string {
	var keys []string
	for _, b := range bindings {
		if b.Enabled() {
			keys = append(keys, t.S.Key.Render(HelpKey(t, b.Help().Key))+" "+t.S.KeyDesc.Render(b.Help().Desc))
		}
	}
	if len(keys) > 0 {
		msg = append(msg, "", Trunc(t, strings.Join(keys, "  "), w))
	}
	out := lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, lipgloss.JoinVertical(lipgloss.Center, msg...))
	// Place never clips: on a terminal shorter than the message, keep the first h rows
	if rows := strings.Split(out, "\n"); len(rows) > h {
		out = strings.Join(rows[:max(0, h)], "\n")
	}
	return out
}
