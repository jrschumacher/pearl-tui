package main

import (
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/ultraviolet/layout"

	"tuikit/theme"
	"tuikit/ui"
)

const (
	minW, minH = 60, 18
	wideW      = 110 // detail pane / sidebar appear at this width
)

// render draws the whole screen for the current state at w×h. The live
// View and the golden harness share it, so what the PNG shows is what the
// terminal shows.
func (m *app) render(w, h int) *ui.Canvas {
	t := m.t
	c := ui.NewCanvas(t, w, h)
	if w < minW || h < minH {
		c.Draw(ui.TooSmall(t, w, h, minW, minH), c.Bounds())
		return c
	}
	if m.screen == scrSplash {
		m.splash(c)
		return c
	}
	g := t.X.Gutter
	// header
	details := ui.Details(t, "prod", "us-east-1", services[m.sel].Name)
	head := ui.Header(t, w, h, t.AppName(), details)
	y := 0
	c.Draw(strings.Join(head, "\n"), ui.Rect(0, y, w, len(head)))
	y += len(head) + t.Gap()

	// footer (drawn first so the body knows how much room is left)
	foot := m.footer(w)
	fy := h - len(foot)
	c.Draw(strings.Join(foot, "\n"), ui.Rect(0, fy, w, len(foot)))
	if m.toast != nil {
		c.Draw(ui.Toast(t, m.toast.kind, m.toast.label, m.toast.msg, w), ui.Rect(0, h-1, w, 1))
	}

	// tabs
	if m.screen != scrDeploy {
		counts := []string{t.S.Subtle.Render(itoa(len(services))), "", "", ""}
		tabs := ui.Tabs(t, []string{"Services", "Deploys", "Incidents", "Assistant"}, int(m.screen), counts)
		c.Draw(strings.Join(tabs, "\n"), ui.Rect(g, y, w-2*g, len(tabs)))
		y += len(tabs) + t.Gap()
	}
	body := uv.Rect(g, y, w-2*g, max(0, fy-y-t.Gap()))
	switch m.screen {
	case scrServices, scrDeploys:
		m.services(c, body)
	case scrIncidents:
		m.incidents(c, body)
	case scrAssistant:
		m.assistant(c, body)
	case scrDeploy:
		m.deployForm(c, body)
	}
	switch m.ov {
	case ovPalette:
		m.palette(c)
	case ovConfirm:
		m.confirm(c)
	}
	if m.ov != ovNone {
		// the footer describes the dialog's keys: it sits above the scrim
		c.Draw(strings.Join(foot, "\n"), ui.Rect(0, fy, w, len(foot)))
	}
	return c
}

func (m *app) footer(w int) []string {
	k := m.keys
	var b []key.Binding
	left := ""
	switch {
	case m.ov == ovPalette:
		b = []key.Binding{k.Enter, k.Up, k.Down, k.Back}
	case m.ov == ovConfirm:
		b = []key.Binding{key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm")), key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "switch")), k.Back}
	case m.screen == scrDeploy:
		b = []key.Binding{key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "deploy")), k.Next, key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "back")), k.Back}
		left = ui.Tag(m.t, "warning", "PROD")
	case m.screen == scrAssistant:
		b = []key.Binding{key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "send")), k.Tab, k.Palette, key.NewBinding(key.WithKeys("ctrl+o"), key.WithHelp("ctrl+o", "expand")), k.Quit}
	default:
		b = []key.Binding{k.Down, k.Up, k.Deploy, k.Rollback, k.Palette, k.Tab, k.Quit}
	}
	right := ""
	if m.screen == scrServices && m.ov == ovNone {
		right = m.t.S.Subtle.Render("synced 4s ago")
	}
	return ui.Footer(m.t, w, left, b, right)
}

// ---------------------------------------------------------------- services

func (m *app) services(c *ui.Canvas, r uv.Rectangle) {
	t := m.t
	var main, side uv.Rectangle
	if r.Dx() >= wideW-4 {
		layout.Horizontal(layout.Fill(1), layout.Len(3), layout.Len(40)).Split(r).Assign(&main, new(uv.Rectangle), &side)
	} else {
		main = r
	}
	wide := main.Dx() >= 70
	header := []string{"service", "version", "p99", "errors"}
	widths := []int{0, 8, 12, 6}
	align := "llll"
	if wide {
		header = append(header, "deployed")
		widths = []int{0, 8, 12, 6, 8}
		align = "llllr"
	}
	// first column takes the rest: compute it so the table is exactly main wide
	fixed := 2
	for _, x := range widths[1:] {
		fixed += x + 2
	}
	widths[0] = max(12, main.Dx()-fixed)
	// the name column comes first; reorder widths/headers accordingly
	var rows [][]string
	for i, s := range services {
		name := ui.Dot(t, s.State) + " " + t.S.Base.Render(s.Name)
		if s.State == "deploying" {
			name = ui.Spinner(t, m.frame/2) + " " + t.S.Base.Render(s.Name)
		}
		spark := t.S.Muted
		switch s.State {
		case "error":
			spark = t.S.Err
		case "warning":
			spark = t.S.Warn
		}
		errs := t.S.Soft.Render(s.ErrRate)
		if s.State == "error" {
			errs = t.S.Err.Bold(true).Render(s.ErrRate)
		}
		ver := t.S.Muted.Render(s.Version)
		if i == m.sel {
			ver = t.S.Soft.Render(s.Version)
		}
		row := []string{name, ver, ui.Sparkline(t, s.P99, 12, spark), errs}
		if wide {
			row = append(row, t.S.Subtle.Render(s.Deployed))
		}
		rows = append(rows, row)
	}
	tbl := ui.Table(t, header, rows, widths, align, m.sel, main.Dx(), m.ov == ovNone)
	c.Draw(strings.Join(tbl, "\n"), main)
	if side.Dx() > 0 {
		m.serviceDetail(c, side)
		// tall terminals: use the room for the activity feed, not blank rows
		if top := main.Min.Y + len(tbl) + 2; main.Max.Y-top >= 6 {
			act := []string{ui.Section(t, "Activity", main.Dx(), "last hour"), ""}
			for _, e := range activity {
				who := t.S.Muted.Render(e[2])
				act = append(act, "  "+t.S.Subtle.Render(e[0])+"  "+ui.Dot(t, e[3])+" "+t.S.Soft.Render(e[1])+"  "+who)
			}
			n := min(len(act), main.Max.Y-top)
			c.Draw(strings.Join(act[:n], "\n"), uv.Rect(main.Min.X, top, main.Dx(), n))
		}
	} else if y := main.Min.Y + len(tbl) + 1; y < main.Max.Y {
		s := services[m.sel]
		line := t.S.Muted.Render(s.Name) + t.S.Subtle.Render("  owner ") + t.S.Soft.Render(s.Owner) +
			t.S.Subtle.Render("  deployed ") + t.S.Soft.Render(s.Deployed)
		c.Draw(line, uv.Rect(main.Min.X+2, y, main.Dx()-2, 1))
		if s.State == "error" && y+2 < main.Max.Y {
			c.Draw(ui.Alert(t, "error", "502s", "4.8% of checkout requests failing since 14:02", main.Dx()-2), uv.Rect(main.Min.X+2, y+2, main.Dx()-2, 1))
		}
	}
}

func (m *app) serviceDetail(c *ui.Canvas, r uv.Rectangle) {
	t := m.t
	s := services[m.sel]
	w := r.Dx()
	var rows []string
	rows = append(rows, ui.Section(t, s.Name, w, s.Env))
	state := map[string]string{"error": "error", "ok": "ok", "warning": "warning", "deploying": "primary", "idle": "muted"}[s.State]
	label := map[string]string{"error": "FAILING", "ok": "HEALTHY", "warning": "DEGRADED", "deploying": "DEPLOYING", "idle": "IDLE"}[s.State]
	rows = append(rows, "")
	rows = append(rows, ui.KV(t, [][2]string{
		{"status", ui.Tag(t, state, label)},
		{"version", t.S.Base.Render(s.Version)},
		{"owner", t.S.Base.Render(s.Owner)},
		{"deployed", t.S.Base.Render(s.Deployed)},
	}, 9)...)
	rows = append(rows, "")
	last := s.P99[len(s.P99)-1]
	rows = append(rows, ui.Section(t, "p99 latency", w, ""))
	rows = append(rows, "")
	sp := t.S.Primary
	if s.State == "error" {
		sp = t.S.Err
	}
	rows = append(rows, ui.Sparkline(t, stretch(s.P99, w), w, sp))
	trend := t.S.Base.Render(itoa(int(last))+"ms") + "  " + t.S.Subtle.Render("12 min")
	if s.P99[0] > 0 && last/s.P99[0] > 2 {
		trend += "  " + t.S.Err.Render("▲ "+ftoa(last/s.P99[0])+"×")
	}
	rows = append(rows, trend)
	if s.State == "error" {
		rows = append(rows, "", ui.Section(t, "Recent", w, ""), "")
		ev := [][2]string{{"14:03", "circuit open · payments"}, {"14:02", "502 rate crossed 1%"}, {"14:00", "deployed v2.4.0 by @mara"}}
		for i, e := range ev {
			txt := t.S.Soft.Render(e[1])
			if i == 0 {
				txt = t.S.Err.Render(e[1])
			}
			rows = append(rows, t.S.Subtle.Render(e[0])+"  "+txt)
		}
		rows = append(rows, "", t.S.Muted.Render("r ")+t.S.Subtle.Render("roll back to v2.3.9"))
	}
	c.Draw(strings.Join(rows, "\n"), r)
}

// stretch resamples vals to n points (nearest), so a sparkline fills its column.
func stretch(vals []float64, n int) []float64 {
	out := make([]float64, n)
	for i := range out {
		out[i] = vals[i*len(vals)/n]
	}
	return out
}

// ---------------------------------------------------------------- incidents

func (m *app) incidents(c *ui.Canvas, r uv.Rectangle) {
	t := m.t
	act := t.S.Key.Render("n") + " " + t.S.KeyDesc.Render("declare incident") + "   " + t.S.Key.Render("h") + " " + t.S.KeyDesc.Render("history")
	rows := ui.Empty(t, r.Dx(), t.G.FieldB, "No incidents declared", "checkout-api has been failing for 12 min. Declare one to page payments on-call.", act)
	top := r.Min.Y + max(0, (r.Dy()-len(rows))/2-1)
	c.Draw(strings.Join(rows, "\n"), uv.Rect(r.Min.X, top, r.Dx(), len(rows)))
}

// ---------------------------------------------------------------- assistant

func (m *app) assistant(c *ui.Canvas, r uv.Rectangle) {
	t := m.t
	var main, side uv.Rectangle
	if r.Dx() >= wideW-4 {
		layout.Horizontal(layout.Fill(1), layout.Len(3), layout.Len(30)).Split(r).Assign(&main, new(uv.Rectangle), &side)
		m.sessionSidebar(c, side)
	} else {
		main = r
	}
	w := main.Dx()
	diff := make([]string, len(diffLines))
	for i, l := range diffLines {
		switch {
		case strings.HasPrefix(l, "+"):
			diff[i] = lipgloss.NewStyle().Foreground(t.P.Success).Render(l)
		case strings.HasPrefix(l, "-"):
			diff[i] = lipgloss.NewStyle().Foreground(t.P.Error).Render(l)
		default: // hunk headers sit on the panel colour: muted, not subtle
			diff[i] = t.S.Muted.Render(l)
		}
	}
	var s []string
	s = append(s, ui.UserMessage(t, chatUser, w, false)...)
	s = append(s, "")
	s = append(s, ui.Thinking(t, thinkingLines, 2, "6.1s", w)...)
	s = append(s, "")
	s = append(s, ui.ToolCall(t, ui.Tool{Name: "Logs", Param: "checkout-api", Opts: [][2]string{{"since", "14:00"}, {"tail", "200"}},
		State: ui.ToolDone, Body: logLines, Shown: 4}, w, false)...)
	s = append(s, "")
	s = append(s, ui.ToolCall(t, ui.Tool{Name: "Grep", Param: "PAYMENTS_TIMEOUT", Opts: [][2]string{{"path", "deploy/"}},
		State: ui.ToolError, Err: "deploy/: no such directory (did you mean config/?)"}, w, false)...)
	s = append(s, "")
	s = append(s, ui.ToolCall(t, ui.Tool{Name: "Edit", Param: "config/timeouts.go", State: ui.ToolDone, Body: diff, Expand: true}, w, true)...)
	s = append(s, "")
	s = append(s, ui.AssistantMessage(t, chatAnswer, w, false)...)
	s = append(s, "")
	s = append(s, ui.TurnFooter(t, "opus", "anthropic", "41s", w))
	// editor: 3 rows at the bottom, prompt in tertiary
	editorH := 3
	streamH := main.Dy() - editorH - 1
	if m.busy {
		s = append(s, "", "  "+ui.NewAnim(t, 10, "Deploying v2.4.1", "deploy").Frame(m.frame, "12s"))
	}
	// follow mode: show the tail of the stream
	if len(s) > streamH {
		hidden := len(s) - streamH + 1
		s = append([]string{"  " + t.S.Subtle.Render("↑ "+itoa(hidden)+" more lines") + t.S.Subtle.Render("  "+t.G.Sep+" ") + t.S.Muted.Render("pgup")}, s[hidden:]...)
	}
	c.Draw(strings.Join(s, "\n"), uv.Rect(main.Min.X, main.Min.Y, w, streamH))
	ey := main.Max.Y - editorH
	prompt := t.S.Tertiary.Bold(true).Render(t.G.Prompt) + " "
	cont := t.S.Subtle.Render(":") + " "
	ed := []string{
		ui.Rule(t, w),
		prompt + t.S.Base.Render("ship it as v2.4.1, and add a test for the unit parsing") + t.S.Tertiary.Render("▏"),
		cont + t.S.Subtle.Render(""),
	}
	c.Draw(strings.Join(ed, "\n"), uv.Rect(main.Min.X, ey, w, editorH))
}

func (m *app) sessionSidebar(c *ui.Canvas, r uv.Rectangle) {
	t := m.t
	w := r.Dx()
	rows := []string{
		t.S.Bold.Render("checkout 502s"),
		t.S.Subtle.Render("started 14:06 · 3 turns"),
		"",
		ui.Section(t, "Model", w, ""),
		t.S.Muted.Render(t.G.FieldB+" ") + t.S.Base.Render("opus") + t.S.Subtle.Render(" via anthropic"),
		t.S.Subtle.Render("context ") + ui.Progress(t, 0.42, 10) + t.S.Subtle.Render(" 42%"),
		"",
		ui.Section(t, "Changed", w, "1"),
		t.S.Soft.Render("config/timeouts.go ") + t.S.OK.Render("+4") + " " + t.S.Err.Render("-1"),
		"",
		ui.Section(t, "Touched", w, ""),
	}
	for _, s := range services[:4] {
		rows = append(rows, ui.Dot(t, s.State)+" "+t.S.Soft.Render(ui.Trunc(t, s.Name, w-2)))
	}
	c.Draw(strings.Join(rows, "\n"), r)
}

// ---------------------------------------------------------------- deploy form

func (m *app) deployForm(c *ui.Canvas, r uv.Rectangle) {
	t := m.t
	var main, side uv.Rectangle
	if r.Dx() >= wideW-4 {
		layout.Horizontal(layout.Len(min(64, r.Dx()-44)), layout.Len(4), layout.Fill(1)).Split(r).Assign(&main, new(uv.Rectangle), &side)
	} else {
		main = r
	}
	w := main.Dx()
	var rows []string
	rows = append(rows, ui.Section(t, "Deploy checkout-api", w, "prod"))
	rows = append(rows, "")
	field := func(i int, label, hint string, body ...string) {
		focused := m.focus == i
		bar := ui.Gutter(t, focused, t.S.FocusBar)
		lab := t.S.Muted.Render(t.Case(label))
		if focused {
			lab = t.S.Bold.Render(t.Case(label))
		}
		rows = append(rows, bar+lab)
		if hint != "" {
			rows = append(rows, bar+t.S.Subtle.Render(hint))
		}
		for _, b := range body {
			rows = append(rows, bar+b)
		}
		for range t.Gap() {
			rows = append(rows, "")
		}
	}
	radio := func(opts []string, on int) string {
		var out []string
		for i, o := range opts {
			if i == on {
				out = append(out, t.S.Primary.Render(t.G.RadioOn)+" "+t.S.Base.Render(o))
			} else {
				out = append(out, t.S.Subtle.Render(t.G.RadioOff)+" "+t.S.Muted.Render(o))
			}
		}
		return strings.Join(out, "   ")
	}
	field(0, "Environment", "", radio([]string{"prod", "staging", "dev"}, 0))
	input := t.S.Tertiary.Render(t.G.Prompt) + " " + t.S.Base.Render("v2.4.1")
	if m.focus == 1 {
		input += t.S.Tertiary.Render("▏")
	}
	field(1, "Version", "tag or commit; latest is v2.4.1 (2 min ago)", input)
	opt := func(name, desc string, on bool) string {
		if on {
			return t.S.Primary.Render(t.G.Cursor) + " " + t.S.Bold.Render(name) + "  " + t.S.Subtle.Render(desc)
		}
		return "  " + t.S.Muted.Render(name) + "  " + t.S.Subtle.Render(desc)
	}
	field(2, "Strategy", "", opt("Rolling", "one pod at a time", true), opt("Canary", "10% for 15 min, then all", false), opt("Blue/green", "switch traffic at once", false))
	check := func(on bool, s string) string {
		if on {
			return t.S.Primary.Render(t.G.CheckOn) + " " + t.S.Base.Render(s)
		}
		return t.S.Subtle.Render(t.G.CheckOff) + " " + t.S.Muted.Render(s)
	}
	field(3, "Checks", "", check(true, "run smoke tests")+"   "+check(false, "skip approval"))
	btns := ui.Buttons(t, w-2, lipgloss.Left,
		ui.Button(t, "Deploy", true, false, 0),
		ui.Button(t, "Cancel", false, false, -1))
	rows = append(rows, "  "+btns)
	c.Draw(strings.Join(rows, "\n"), main)

	if side.Dx() > 0 {
		sw := side.Dx()
		steps := ui.Tree(t, []string{
			t.S.Soft.Render("build image ") + t.S.Subtle.Render("cached"),
			t.S.Soft.Render("run smoke tests ") + t.S.Subtle.Render("~40s"),
			t.S.Soft.Render("roll 3 pods ") + t.S.Subtle.Render("one at a time"),
			t.S.Soft.Render("watch error rate ") + t.S.Subtle.Render("5 min, auto-rollback >1%"),
		})
		pr := []string{ui.Section(t, "What happens", sw, ""), ""}
		pr = append(pr, t.S.Base.Render("checkout-api ")+t.S.Muted.Render("v2.4.0 "+t.G.Arrow+" ")+t.S.Primary.Bold(true).Render("v2.4.1"))
		pr = append(pr, "")
		pr = append(pr, steps...)
		pr = append(pr, "", ui.Alert(t, "warning", "PROD", "customer traffic, 3 regions", sw))
		c.Draw(strings.Join(pr, "\n"), side)
	}
}

// ---------------------------------------------------------------- overlays

type hit struct {
	c   command
	idx []int
}

func (m *app) filtered() []hit {
	var out []hit
	q := strings.ToLower(m.query)
	for _, c := range commands {
		if idx, ok := fuzzy(strings.ToLower(c.Title), q); ok {
			out = append(out, hit{c, idx})
		}
	}
	return out
}

// fuzzy matches q as a subsequence of s, returning the matched rune indices.
func fuzzy(s, q string) ([]int, bool) {
	var idx []int
	rs := []rune(s)
	j := 0
	for _, qr := range q {
		for j < len(rs) && rs[j] != qr {
			j++
		}
		if j == len(rs) {
			return nil, false
		}
		idx = append(idx, j)
		j++
	}
	return idx, true
}

func (m *app) palette(c *ui.Canvas) {
	t := m.t
	w := min(64, c.Bounds().Dx()-6)
	inner := w - t.S.Dialog.GetHorizontalFrameSize()
	hits := m.filtered()
	var items []ui.Item
	for _, h := range hits {
		items = append(items, ui.Item{Title: h.c.Title, Desc: h.c.Desc, Right: h.c.Key, Match: h.idx})
	}
	input := t.S.Tertiary.Render(t.G.Prompt) + " " + t.S.Base.Render(m.query) + t.S.Tertiary.Render("▏")
	if m.query == "" {
		input = t.S.Tertiary.Render(t.G.Prompt) + " " + t.S.Tertiary.Render("▏") + t.S.Subtle.Render("type to filter")
	}
	listH := min(6, max(3, c.Bounds().Dy()-14))
	body := append([]string{input, ""}, ui.List(t, items, m.palSel, 0, inner, listH, true)...)
	help := ui.Help(t, []key.Binding{m.keys.Enter, m.keys.Up, m.keys.Down, m.keys.Back}, inner)
	c.Overlay(ui.Dialog(t, w, "Commands", body, "", help), nil)
}

func (m *app) confirm(c *ui.Canvas) {
	t := m.t
	w := min(58, c.Bounds().Dx()-6)
	inner := w - t.S.Dialog.GetHorizontalFrameSize()
	body := []string{
		t.S.Soft.Render("prod returns to the last healthy version."),
		"",
	}
	body = append(body, ui.KV(t, [][2]string{
		{"now", t.S.Err.Render("v2.4.0") + t.S.Subtle.Render("  failing 12 min")},
		{"after", t.S.OK.Render("v2.3.9") + t.S.Subtle.Render("  healthy for 3 days")},
		{"impact", t.S.Base.Render("3 pods restart, one at a time (~90s)")},
	}, 7)...)
	btns := ui.Buttons(t, inner, lipgloss.Right,
		ui.Button(t, "Roll back", m.btn == 0, true, 0),
		ui.Button(t, "Cancel", m.btn == 1, false, -1))
	c.Overlay(ui.Dialog(t, w, "Roll back checkout-api?", body, btns, ""), nil)
}

// ---------------------------------------------------------------- splash

func (m *app) splash(c *ui.Canvas) {
	t := m.t
	b := c.Bounds()
	f := ui.DefaultField()
	f.ClearR, f.Dim, f.Drift = 0.42, 0.25, m.frame/3
	f.Draw(c, b)
	var block []string
	if rows := t.Wordmark.Rows; len(rows) > 0 && lipgloss.Width(rows[0]) < b.Dx()-4 {
		for _, r := range rows {
			block = append(block, ui.Fit(ui.Wordmark(t, ""), 0)+bannerTone(t, r))
		}
	} else {
		block = append(block, ui.Wordmark(t, t.AppName()))
	}
	block = append(block, "", ui.NewAnim(t, 8, "connecting to prod", "splash").Frame(m.frame, ""))
	bw := 0
	for _, r := range block {
		bw = max(bw, lipgloss.Width(r))
	}
	for i := range block {
		block[i] = lipgloss.PlaceHorizontal(bw, lipgloss.Center, block[i])
	}
	top := (b.Dy()-len(block))/2 - 1
	left := (b.Dx() - bw) / 2
	r := uv.Rect(left, top, bw, len(block))
	c.Draw(strings.Join(block, "\n"), r)
	c.Draw(t.S.Subtle.Render(t.AppName()+" v0.9.0"), uv.Rect(2, b.Dy()-1, 30, 1))
	q := t.S.Muted.Render("q") + t.S.Subtle.Render(" quit")
	c.Draw(q, uv.Rect(b.Dx()-2-lipgloss.Width(q), b.Dy()-1, lipgloss.Width(q), 1))
}

func bannerTone(t *theme.Theme, r string) string {
	var sb strings.Builder
	for _, ch := range r {
		switch ch {
		case '▓':
			sb.WriteString(t.S.Primary.Render("▓"))
		case '▒':
			sb.WriteString(t.S.Muted.Render("▒"))
		case ' ':
			sb.WriteRune(' ')
		default:
			sb.WriteString(t.S.Accent.Render(string(ch)))
		}
	}
	return sb.String()
}
