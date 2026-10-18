package ui

import (
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"tuikit/theme"
)

// Gutter is the 2-cell left column every block in a stream gets. Focused: a
// focus bar + 1 space. Blurred: the blur bar (or 2 spaces). Both exactly 2
// cells, so moving focus never shifts content (Crush's rule).
func Gutter(t *theme.Theme, focused bool, bar lipgloss.Style) string {
	if focused {
		return bar.Render(t.G.FocusBar) + " "
	}
	if t.G.BlurBar != "" {
		return t.S.BlurBar.Render(t.G.BlurBar) + " "
	}
	return "  "
}

// prefix puts a gutter in front of every row.
func prefix(rows []string, g string, w int) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = Fit(g+r, w)
	}
	return out
}

// UserMessage: the user's turn carries a bar (it is the user's voice), plain text,
// wrapped to the measure. Focused, the bar is the focus bar in the focus ink; blurred,
// the turn bar (or the blur bar when the theme has one) in muted: a bold or coloured
// bar means focus, and a turn the user isn't editing has none.
func UserMessage(t *theme.Theme, text string, w int, focused bool) []string {
	m := min(w-2, t.X.Measure)
	body := lipgloss.NewStyle().Width(m).Render(t.S.Base.Render(text))
	g := t.S.FocusBar.Render(t.G.FocusBar) + " "
	if !focused {
		bar := t.G.TurnBar
		if t.G.BlurBar != "" {
			bar = t.G.BlurBar
		}
		g = t.S.Muted.Render(bar) + " "
	}
	return prefix(strings.Split(body, "\n"), g, w)
}

// AssistantMessage: prose in the soft text level, wrapped to the measure,
// indented 2 (a focus bar in success when focused).
func AssistantMessage(t *theme.Theme, text string, w int, focused bool) []string {
	m := min(w-2, t.X.Measure)
	var rows []string
	for _, para := range strings.Split(text, "\n") {
		if para == "" {
			rows = append(rows, "")
			continue
		}
		rows = append(rows, strings.Split(lipgloss.NewStyle().Width(m).Render(styleInline(t, para)), "\n")...)
	}
	return prefix(rows, Gutter(t, focused, t.S.FocusBar), w)
}

// styleInline styles `code` spans in prose: tertiary on the panel colour.
func styleInline(t *theme.Theme, s string) string {
	parts := strings.Split(s, "`")
	for i := range parts {
		if i%2 == 1 {
			parts[i] = lipgloss.NewStyle().Foreground(t.P.Tertiary).Render(parts[i])
		} else {
			parts[i] = t.S.Soft.Render(parts[i])
		}
	}
	return strings.Join(parts, "")
}

// ToolState is the lifecycle of a tool call.
type ToolState int

const (
	ToolPending ToolState = iota
	ToolRunning
	ToolDone
	ToolError
	ToolCanceled
)

// Tool is one tool call in a stream.
type Tool struct {
	Name   string
	Param  string      // the main parameter (a command, a path, a pattern)
	Opts   [][2]string // secondary k=v params, dropped first when narrow; a flag has an empty value and draws bare
	Desc   string      // what the call is for ("Run import tests"), muted under the head
	Meta   string      // a note right-aligned on the head (elapsed, "allowed for session"); Opts go before it, it goes before Param is cut
	State  ToolState
	Body   []string // output lines (pre-styled or plain)
	Err    string
	Spin   string // the spinner frame while running (see Anim)
	Shown  int    // collapsed height (default 6); 0 = show all
	Expand bool
}

// ToolCall renders the Crush grammar (its fixed copy, "Canceled." and "Waiting for
// permission", in the theme's case):
//
//	● Name main-param (k=v, flag)              allowed for session · 6s
//	  what the call is for
//
//	  output line on the panel colour, full width
//	  … (12 lines hidden) [space to expand]
func ToolCall(t *theme.Theme, tc Tool, w int, focused bool) []string {
	inner := w - 2
	var icon string
	switch tc.State {
	case ToolPending:
		icon = t.S.Subtle.Render(t.G.Pending)
	case ToolRunning: // in progress: the partial rung, so it isn't pending in another colour
		icon = t.S.Working.Render(t.G.Partial)
	case ToolDone:
		icon = t.S.OK.Render(t.G.Done)
	case ToolError:
		icon = t.S.Err.Render(t.G.Fail)
	case ToolCanceled:
		icon = t.S.Subtle.Render(t.G.Pending)
	}
	name := t.S.Info.Bold(true).Render(tc.Name)
	head := icon + " " + name
	avail := inner - lipgloss.Width(head) - 1
	param := tc.Param
	opts := ""
	if len(tc.Opts) > 0 {
		var kv []string
		for _, o := range tc.Opts {
			if o[1] == "" {
				kv = append(kv, o[0]) // a flag
			} else {
				kv = append(kv, o[0]+"="+o[1])
			}
		}
		opts = " (" + strings.Join(kv, ", ") + ")"
	}
	// the note keeps its place while the whole parameter and two cells of air fit
	// beside it, and goes before the parameter is cut for it; the k=v group goes first
	meta := ""
	if tc.Meta != "" && lipgloss.Width(param)+2+lipgloss.Width(tc.Meta) <= avail {
		meta = tc.Meta
		avail -= 2 + lipgloss.Width(meta)
	}
	// drop the k=v group before the main param gets cramped (< 30 cells)
	if lipgloss.Width(param)+lipgloss.Width(opts) > avail && avail-lipgloss.Width(opts) < 30 {
		opts = ""
	}
	head += " " + t.S.Muted.Render(Trunc(t, param, max(0, avail-lipgloss.Width(opts)))) + t.S.Subtle.Render(opts)
	if tc.State == ToolRunning && tc.Spin != "" {
		head += " " + tc.Spin
	}
	if meta != "" {
		head += strings.Repeat(" ", max(2, inner-lipgloss.Width(head)-lipgloss.Width(meta))) + t.S.Subtle.Render(meta)
	}
	rows := []string{head}
	if tc.Desc != "" {
		rows = append(rows, t.S.Muted.Render(Trunc(t, tc.Desc, inner)))
	}
	switch {
	case tc.State == ToolError:
		rows = append(rows, "", Tag(t, "error", "ERROR")+" "+t.S.Muted.Render(Trunc(t, tc.Err, inner-8)))
	case tc.State == ToolCanceled:
		rows = append(rows, "", t.S.Subtle.Render(t.Sentence("Canceled.")))
	case tc.State == ToolPending:
		rows = append(rows, "", t.S.Subtle.Render(t.Sentence("Waiting for permission")+t.G.Ellipsis))
	case len(tc.Body) > 0:
		rows = append(rows, "")
		shown := tc.Shown
		if shown == 0 {
			shown = 6
		}
		body := tc.Body
		hidden := 0
		// never hide exactly one line: show it instead
		if !tc.Expand && len(body) > shown+1 {
			hidden = len(body) - shown
			body = body[:shown]
		}
		panel := lipgloss.NewStyle().Background(t.P.BgPanel).Foreground(t.P.FgMuted)
		for _, l := range body {
			rows = append(rows, panel.Render(Fit("  "+Trunc(t, l, inner-4)+"  ", inner)))
		}
		if hidden > 0 {
			// on the panel colour the subtle level drops under 2.3:1: hints use muted here
			rows = append(rows, panel.Render(Fit("  "+t.S.Muted.Render(t.G.Ellipsis+" ("+strconv.Itoa(hidden)+" lines hidden) ")+
				t.S.Soft.Render("[space to expand]"), inner)))
		}
	}
	return prefix(rows, Gutter(t, focused, t.S.FocusBar), w)
}

// Thinking is demoted reasoning: subtle text on the panel colour, only the
// last n lines, with a footer of how long it took ("Thought for" in the theme's case).
func Thinking(t *theme.Theme, lines []string, n int, took string, w int) []string {
	inner := w - 2
	st := lipgloss.NewStyle().Background(t.P.BgPanel).Foreground(t.P.FgMuted).Italic(true)
	var rows []string
	if len(lines) > n {
		rows = append(rows, st.Render(Fit("  "+t.G.Ellipsis+" ("+strconv.Itoa(len(lines)-n)+" lines hidden)", inner)))
		lines = lines[len(lines)-n:]
	}
	for _, l := range lines {
		rows = append(rows, st.Render(Fit("  "+Trunc(t, l, inner-4), inner)))
	}
	if took != "" {
		rows = append(rows, t.S.Subtle.Render(t.Sentence("Thought for")+" "+took))
	}
	return prefix(rows, "  ", w)
}

// TurnFooter closes a turn: "◇ model via provider in 12s ──────".
func TurnFooter(t *theme.Theme, model, via, took string, w int) string {
	label := t.S.OK.Render(t.G.Done) + t.S.Muted.Render(" "+model) + t.S.Subtle.Render(" via "+via+" in "+took)
	n := w - 2 - lipgloss.Width(label) - 1
	return "  " + label + " " + t.S.Rule.Render(strings.Repeat(t.G.Rule, max(0, n)))
}

// Tree renders children under a rounded rail (sub-agent steps, nested items). An item
// with more than one line continues under the rail (glyphs.treeRail) while items
// follow, and under air after the last one:
//
//	├─ run smoke tests
//	│  ~40s
//	╰─ watch error rate
//	   5 min, auto-rollback >1%
func Tree(t *theme.Theme, items []string) []string {
	var out []string
	for i, it := range items {
		e, rail := t.G.TreeMid, t.G.TreeRail
		if i == len(items)-1 {
			e, rail = t.G.TreeEnd, ""
		}
		lines := strings.Split(it, "\n")
		out = append(out, t.S.Subtle.Render(e)+" "+lines[0])
		for _, l := range lines[1:] {
			out = append(out, t.S.Subtle.Render(Fit(rail, lipgloss.Width(e)))+" "+l)
		}
	}
	return out
}

// Transcript is a stream's visible rows: blocks (each a component's rows: a turn, a
// tool call, a thinking panel) joined by a row of air, following the tail, scrolled up
// by scroll rows (clamped, so the first block can reach the top and no further). The
// tail is cut at the top under a marker row that says how much is above; the cut never
// falls inside a block: when it would, the marker row carries the cut block's head,
// truncated to make room, so the first visible row is always a block head or the
// marker, and a tool's output is never shown with its head off-screen. hint is a key
// after the marker (the one that scrolls up). A page draws the result into its stream
// rectangle; a block cut at the bottom (scrolled up) is cut on a row:
//
//	● Logs checkout-api (since=14:00)              ↑ 24 more  pgup
//	  14:02:40  WARN   upstream timeout
func Transcript(t *theme.Theme, blocks [][]string, w, h, scroll int, hint string) []string {
	if h <= 0 {
		return nil
	}
	var rows []string
	var heads []int // the row each block starts on
	for _, b := range blocks {
		if len(b) == 0 {
			continue
		}
		if len(rows) > 0 {
			rows = append(rows, Fit("", w)) // air between blocks
		}
		heads = append(heads, len(rows))
		rows = append(rows, b...)
	}
	end := len(rows) - min(max(0, scroll), max(0, len(rows)-h))
	if end <= h {
		return rows[:end]
	}
	hidden := end - h + 1 // the marker takes a row
	// the block the cut falls in: its head is the last one at or before the cut, and
	// the cut is inside it when it is past the head and short of the air after it (the
	// last block has none: its "air" is one past the end)
	head, next := -1, len(rows)+1
	for _, hd := range heads {
		if hd <= hidden {
			head = hd
		} else {
			next = hd
			break
		}
	}
	more := func(n int) string { return t.S.Subtle.Render(t.G.More + " " + strconv.Itoa(n) + " more") }
	tail := ""
	if hint != "" {
		tail = "  " + hint
	}
	if head >= 0 && hidden > head && hidden < next-1 {
		right := more(hidden-1) + tail
		// lifted only where the head keeps its icon and name; narrower, the plain
		// marker counts the head with the rest
		if room := w - lipgloss.Width(right) - 2; room >= headWidth(rows[head]) {
			return append([]string{Fit(cutRow(t, rows[head], room)+"  "+right, w)}, rows[hidden:end]...)
		}
	}
	marker := "  " + more(hidden) + tail
	return append([]string{Fit(marker, w)}, rows[hidden:end]...)
}

// headWidth is the cells a block's head needs to say what it is: through its second
// word (the gutter, then an icon and a name: "  ● Edit") and a cell for the ellipsis
// after it, or all of it when that is shorter.
func headWidth(row string) int {
	plain := []rune(ansi.Strip(row))
	all := ansi.StringWidth(strings.TrimRight(string(plain), " "))
	words, in := 0, false
	for i, r := range plain {
		switch {
		case r != ' ' && !in:
			in = true
		case r == ' ' && in:
			in = false
			if words++; words == 2 {
				return min(all, ansi.StringWidth(string(plain[:i]))+1)
			}
		}
	}
	return all
}

// cutRow truncates a padded row to w cells: plainly when only its padding is lost;
// at the gap before a right-aligned note (two blanks or more) when the cut falls in
// the note, so no "allowed …" fragment is left; else with the theme's ellipsis.
func cutRow(t *theme.Theme, s string, w int) string {
	w = max(0, w)
	if lipgloss.Width(s) <= w {
		return Fit(s, w)
	}
	if strings.TrimSpace(ansi.Strip(ansi.TruncateLeft(s, w, ""))) == "" {
		return ansi.Truncate(s, w, "")
	}
	kept := []rune(ansi.Strip(ansi.Truncate(s, w, "")))
	for x := len(kept) - 1; x > 0; x-- {
		if kept[x] == ' ' && kept[x-1] == ' ' {
			if strings.TrimSpace(string(kept[:x])) != "" {
				return Fit(ansi.Truncate(s, lipgloss.Width(string(kept[:x])), ""), w)
			}
			break
		}
	}
	// the row is pre-styled: the ellipsis gets its own ink, or it would be unstyled
	ell := t.S.Subtle.Render(t.G.Ellipsis)
	return Fit(ansi.Truncate(s, w-lipgloss.Width(ell), "")+ell, w)
}
