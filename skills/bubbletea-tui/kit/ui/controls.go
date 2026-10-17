package ui

import (
	"fmt"
	"math"
	"strings"

	"charm.land/lipgloss/v2"

	"tuikit/theme"
)

// Button renders one button per expressions.button. mnemonic (0-based rune
// index, -1 for none) underlines the accelerator letter.
func Button(t *theme.Theme, label string, focused, danger bool, mnemonic int) string {
	st := t.S.Button
	if danger && !focused {
		st = st.Foreground(t.P.Error)
	}
	if focused {
		st = t.S.ButtonFocus
		if danger {
			st = t.S.ButtonDanger
		}
	}
	text := label
	if mnemonic >= 0 && mnemonic < len([]rune(label)) {
		r := []rune(label)
		text = string(r[:mnemonic]) + "\x00" + string(r[mnemonic:mnemonic+1]) + "\x01" + string(r[mnemonic+1:])
	}
	style := t.X.Button
	if t.ASCII {
		// a filled pill is invisible without colour, and its ink is the fill's on-colour
		// (often the background itself): draw it as the bracket expression does
		style = "bracket"
		switch {
		case focused && danger:
			st = t.S.Err.Bold(true)
		case focused:
			st = t.S.Focus.Bold(true)
		case danger:
			st = t.S.Err
		default:
			st = t.S.Muted
		}
	}
	switch style {
	case "bracket":
		mark := " "
		if focused {
			mark = t.G.Cursor
		}
		return st.Render(mark+"[ ") + underline(st, text) + st.Render(" ]")
	case "underline":
		return underline(st, text)
	}
	return underline(st, text)
}

// underline renders text in st, with the span between \x00 and \x01 underlined.
func underline(st lipgloss.Style, text string) string {
	a, rest, ok := strings.Cut(text, "\x00")
	if !ok {
		return st.Render(text)
	}
	m, b, _ := strings.Cut(rest, "\x01")
	// one style call so padding wraps the whole label
	pl, pr := st.GetPaddingLeft(), st.GetPaddingRight()
	inner := st.UnsetPadding()
	return inner.Render(strings.Repeat(" ", pl)+a) + inner.Underline(true).Render(m) + inner.Render(b+strings.Repeat(" ", pr))
}

// Buttons lays a row of buttons out at align within w (left|center|right).
func Buttons(t *theme.Theme, w int, align lipgloss.Position, buttons ...string) string {
	gap := "  "
	if t.X.Button == "pill" || t.X.Button == "inverse" {
		gap = " "
	}
	row := strings.Join(buttons, gap)
	return lipgloss.PlaceHorizontal(w, align, row)
}

// Tag is a state word on a chip: OK, ERROR, WARN, NEW, PLAN, v2.3.1…, in the theme's
// labelCase (as written when unset). kind: primary | ok | error | warning | info | muted
func Tag(t *theme.Theme, kind, text string) string {
	st := t.S.TagMuted
	switch kind {
	case "primary":
		st = t.S.TagPrimary
	case "ok":
		st = t.S.TagOK
	case "error":
		st = t.S.TagErr
	case "warning":
		st = t.S.TagWarn
	case "info":
		st = t.S.TagInfo
	}
	text = t.Label(text, "as-is")
	if t.X.Tag == "bracket" {
		return st.Render("[" + text + "]")
	}
	return st.Render(text)
}

// Alert is a tag plus a message bar spanning w: the terminal's banner.
func Alert(t *theme.Theme, kind, label, msg string, w int) string {
	tg := Tag(t, kind, label)
	bar := t.S.BarWarn
	switch kind {
	case "error":
		bar = t.S.BarErr
	case "ok":
		bar = t.S.BarOK
	case "info", "primary":
		bar = t.S.BarInfo
	}
	rest := max(0, w-lipgloss.Width(tg))
	if t.X.Tag != "filled" {
		return Fit(tg+"  "+t.S.Base.Render(Trunc(t, msg, rest-2)), w)
	}
	return tg + bar.Width(rest).Render(Trunc(t, msg, max(0, rest-2)))
}

// Dot is a status dot: ● in a role colour, with the label after it. Never
// colour alone: the label says the state too, and each state has its own shape
// (busy is the partial rung, ◐, not ok's ● in another colour).
func Dot(t *theme.Theme, state string) string {
	switch state {
	case "ok", "healthy", "online", "done":
		return t.S.OK.Render(t.G.Current)
	case "error", "failed", "down":
		return t.S.Err.Render(t.G.Fail)
	case "warning", "degraded":
		return t.S.Warn.Render(t.G.Warn)
	case "busy", "deploying", "running":
		return Mark(t, "busy")
	}
	return Mark(t, "todo") // idle: a rung still to come, in fgMuted, never dimmer than a hint
}

// KV renders "key   value" rows with an aligned key column. Keys are dimmer
// than values: the value is what the eye is looking for.
func KV(t *theme.Theme, rows [][2]string, keyW int) []string {
	if keyW == 0 {
		for _, r := range rows {
			keyW = max(keyW, lipgloss.Width(r[0]))
		}
	}
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = t.S.Muted.Render(Fit(r[0], keyW)) + "  " + r[1]
	}
	return out
}

// Progress is a bar of w cells: ━━━━━━╌╌╌ (or █▌ blocks in ASCII-safe themes).
// frac 0..1. The filled part takes a gradient from secondary to primary.
func Progress(t *theme.Theme, frac float64, w int) string {
	frac = math.Max(0, math.Min(1, frac))
	full := frac * float64(w)
	n := int(full)
	fill, empty := "━", "╌"
	if t.ASCII {
		fill, empty = "=", "-"
	}
	var head string
	if !t.ASCII && n < w && full-float64(n) >= 0.5 {
		head = "╸"
	}
	filled := strings.Repeat(fill, n) + head
	if !t.ASCII && n > 1 && t.X.Focus != "fg" {
		filled = theme.Grad(filled, t.P.Secondary, t.P.Primary, false)
	} else {
		filled = t.S.Mark.Render(filled)
	}
	rest := w - n - lipgloss.Width(head)
	return filled + t.S.Rule.Render(strings.Repeat(empty, max(0, rest)))
}

// Sparkline draws values as ▁▂▃▄▅▆▇█ over w cells (the last w values).
func Sparkline(t *theme.Theme, vals []float64, w int, style lipgloss.Style) string {
	if len(vals) > w {
		vals = vals[len(vals)-w:]
	}
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, v := range vals {
		lo, hi = math.Min(lo, v), math.Max(hi, v)
	}
	// a flat series stays a low flat line: ±5% noise drawn full-height reads
	// as an incident
	flat := hi <= 0 || (hi-lo)/hi < 0.25
	bars := []rune("▁▂▃▄▅▆▇█")
	if t.ASCII {
		bars = []rune("_.-=+*#@")
	}
	var b strings.Builder
	for _, v := range vals {
		k := 0
		switch {
		case flat && hi > lo:
			k = 1 + int(math.Round((v-lo)/(hi-lo)))
		case flat:
			k = 0
		default:
			k = int(math.Round((v - lo) / (hi - lo) * float64(len(bars)-1)))
		}
		b.WriteRune(bars[k])
	}
	return style.Render(Fit(b.String(), w))
}

// Scrollbar returns h cells (top to bottom) for a list of total rows with
// offset at the top. It returns blanks when everything fits: reserve the
// column only when content overflows.
func Scrollbar(t *theme.Theme, h, total, offset int) []string {
	out := make([]string, h)
	if total <= h || h <= 0 {
		for i := range out {
			out[i] = " "
		}
		return out
	}
	size := max(1, h*h/total)
	pos := int(math.Round(float64(offset) / float64(total-h) * float64(h-size)))
	for i := range out {
		if i >= pos && i < pos+size {
			out[i] = t.S.Muted.Render(t.G.Thumb)
		} else {
			out[i] = t.S.Rule.Render(t.G.Track)
		}
	}
	return out
}

// Count formats a number compactly: 1.2k, 34k, 1.1M.
func Count(n int) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1e6)
	case n >= 10_000:
		return fmt.Sprintf("%dk", n/1000)
	case n >= 1000:
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	}
	return fmt.Sprint(n)
}

// Empty is an empty state: what is missing, why, and the one thing to do.
// Never a bare "No items".
func Empty(t *theme.Theme, w int, glyph, title, why, action string) []string {
	rows := []string{t.S.Subtle.Render(glyph), "", t.S.Bold.Render(title)}
	for _, l := range strings.Split(lipgloss.NewStyle().Width(min(w, 56)).Align(lipgloss.Center).Render(why), "\n") {
		rows = append(rows, t.S.Muted.Render(strings.TrimSpace(l)))
	}
	if action != "" {
		rows = append(rows, "", action)
	}
	for i, r := range rows {
		rows[i] = lipgloss.PlaceHorizontal(w, lipgloss.Center, r)
	}
	return rows
}
