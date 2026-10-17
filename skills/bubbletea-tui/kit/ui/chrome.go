package ui

import (
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
// a banner costs rows, so below 30 rows it collapses to the rule header.
func Header(t *theme.Theme, w, h int, brand, details string) []string {
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
		return []string{Fit(pad+Wordmark(t, brand)+"  "+details, w)}
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
	if !strings.ContainsAny(r, "▓▒") {
		return t.S.Primary.Render(r)
	}
	var sb strings.Builder
	for _, ch := range r {
		switch ch {
		case '█':
			sb.WriteString(t.S.Accent.Render("█"))
		case '▓':
			sb.WriteString(t.S.Primary.Render("▓"))
		case '▒':
			sb.WriteString(t.S.Muted.Render("▒"))
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
// The rule is the region separator; boxes are not.
func Section(t *theme.Theme, title string, w int, info string) string {
	ti := t.S.Title.Render(t.Case(title))
	if info != "" {
		info = " " + t.S.Subtle.Render(info)
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
	// the key that opens full help is pinned last: the … promises more, so the
	// way to see it must survive truncation
	pinned := ""
	if n := len(bindings); n > 1 && (bindings[n-1].Help().Key == "?" || bindings[n-1].Help().Key == "ctrl+g") {
		h := bindings[n-1].Help()
		pinned = t.S.Key.Render(h.Key) + " " + t.S.KeyDesc.Render(h.Desc)
		w -= lipgloss.Width(pinned) + 2
		bindings = bindings[:n-1]
	}
	var parts []string
	used := 0
	more := t.S.Subtle.Render(t.G.Ellipsis)
	// room for the separator and the ellipsis as the theme draws it: "  …" is 3 cells,
	// the ASCII "  ..." is 5
	reserveMore := 2 + lipgloss.Width(more)
	for i, b := range bindings {
		if !b.Enabled() {
			continue
		}
		h := b.Help()
		part := t.S.Key.Render(h.Key) + " " + t.S.KeyDesc.Render(h.Desc)
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
	tg := tag.Render(label)
	rest := w - 2*g - lipgloss.Width(tg)
	return strings.Repeat(" ", g) + tg + bar.Width(max(0, rest)).Render(Trunc(t, msg, max(0, rest-2))) + strings.Repeat(" ", g)
}

// Tabs renders text tabs per expressions.tabs. Never boxes.
//
//	underline  Services   Deploys   Incidents      (active bold, ▔ under it on a second row)
//	pill       the active tab on a filled pill
//	bracket    [Services]  Deploys   Incidents
//	slash      Services / Deploys / Incidents
func Tabs(t *theme.Theme, names []string, active int, counts []string) []string {
	var top, under []string
	for i, n := range names {
		label := n
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
			mark = "▔"
			if t.ASCII {
				mark = "^"
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

// TooSmall is the whole screen when the terminal is below the minimum.
func TooSmall(t *theme.Theme, w, h, minW, minH int) string {
	msg := Wordmark(t, t.AppName()) + "\n\n" + t.S.Soft.Render("Terminal too small") + "\n" + t.S.Muted.Render(
		strconv.Itoa(w)+"×"+strconv.Itoa(h)+", needs "+strconv.Itoa(minW)+"×"+strconv.Itoa(minH)) + "\n\n" +
		t.S.Key.Render("q") + " " + t.S.KeyDesc.Render("quit")
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, lipgloss.JoinVertical(lipgloss.Center, strings.Split(msg, "\n")...))
}
