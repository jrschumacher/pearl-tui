package ui

import (
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"

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

// UserMessage: the user's turn carries a bar in the primary colour even when
// blurred (it is the user's voice), plain text, wrapped to the measure.
func UserMessage(t *theme.Theme, text string, w int, focused bool) []string {
	m := min(w-2, t.X.Measure)
	body := lipgloss.NewStyle().Width(m).Render(t.S.Base.Render(text))
	bar := t.G.FocusBar
	if !focused && t.G.BlurBar != "" {
		bar = t.G.BlurBar
	} else if !focused {
		bar = "│"
		if t.ASCII {
			bar = "|"
		}
	}
	g := t.S.Focus.Render(bar) + " "
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
	Opts   [][2]string // secondary k=v params, dropped first when narrow
	State  ToolState
	Body   []string // output lines (pre-styled or plain)
	Err    string
	Spin   string // the spinner frame while running (see Anim)
	Shown  int    // collapsed height (default 6); 0 = show all
	Expand bool
}

// ToolCall renders the Crush grammar:
//
//	● Name main-param (k=v, k=v)
//
//	  output line on the panel colour, full width
//	  … (12 lines hidden) [space to expand]
func ToolCall(t *theme.Theme, tc Tool, w int, focused bool) []string {
	inner := w - 2
	var icon string
	switch tc.State {
	case ToolPending:
		icon = t.S.Subtle.Render(t.G.Pending)
	case ToolRunning:
		icon = t.S.Mark.Render(t.G.Pending)
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
			kv = append(kv, o[0]+"="+o[1])
		}
		opts = " (" + strings.Join(kv, ", ") + ")"
	}
	// drop the k=v group before the main param gets cramped (< 30 cells)
	if lipgloss.Width(param)+lipgloss.Width(opts) > avail && avail-lipgloss.Width(opts) < 30 {
		opts = ""
	}
	head += " " + t.S.Muted.Render(Trunc(t, param, max(0, avail-lipgloss.Width(opts)))) + t.S.Subtle.Render(opts)
	if tc.State == ToolRunning && tc.Spin != "" {
		head += " " + tc.Spin
	}
	rows := []string{head}
	switch {
	case tc.State == ToolError:
		rows = append(rows, "", Tag(t, "error", "ERROR")+" "+t.S.Muted.Render(Trunc(t, tc.Err, inner-8)))
	case tc.State == ToolCanceled:
		rows = append(rows, "", t.S.Subtle.Render("Canceled."))
	case tc.State == ToolPending:
		rows = append(rows, "", t.S.Subtle.Render("Waiting for permission…"))
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
// last n lines, with a footer of how long it took.
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
		rows = append(rows, t.S.Subtle.Render("Thought for "+took))
	}
	return prefix(rows, "  ", w)
}

// TurnFooter closes a turn: "◇ model via provider in 12s ──────".
func TurnFooter(t *theme.Theme, model, via, took string, w int) string {
	label := t.S.OK.Render(t.G.Done) + t.S.Muted.Render(" "+model) + t.S.Subtle.Render(" via "+via+" in "+took)
	n := w - 2 - lipgloss.Width(label) - 1
	return "  " + label + " " + t.S.Rule.Render(strings.Repeat(t.G.Rule, max(0, n)))
}

// Tree renders children under a rounded rail (sub-agent steps, nested items).
func Tree(t *theme.Theme, items []string) []string {
	out := make([]string, len(items))
	for i, it := range items {
		e := t.G.TreeMid
		if i == len(items)-1 {
			e = t.G.TreeEnd
		}
		out[i] = t.S.Subtle.Render(e) + " " + it
	}
	return out
}
