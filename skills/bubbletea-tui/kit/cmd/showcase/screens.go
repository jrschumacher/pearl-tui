package main

import (
	"strings"

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
	c := m.draw(w, h)
	if m.t.ASCII {
		c.ASCIIfy() // nothing outside tier 0 survives, whatever a screen typed
	}
	return c
}

func (m *app) draw(w, h int) *ui.Canvas {
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
	s := m.svcs[m.sel]
	head := ui.Header(t, w, h, t.AppName(), ui.Details(t, "prod", "us-east-1", s.Name))
	y := 0
	c.Draw(strings.Join(head, "\n"), ui.Rect(0, y, w, len(head)))
	y += len(head) + t.Gap()

	// footer first: the body gets what is left
	foot := ui.Footer(t, w, m.footerBadge(), m.footerBindings(), m.footerRight())
	fy := h - len(foot)
	c.Draw(strings.Join(foot, "\n"), ui.Rect(0, fy, w, len(foot)))

	if m.screen != scrDeploy {
		counts := []string{t.S.Subtle.Render(itoa(len(m.svcs))), "", "", ""}
		if m.declared {
			counts[2] = t.S.Warn.Render("1")
		}
		tabs := ui.Tabs(t, tabNames, int(m.screen), counts)
		c.Draw(strings.Join(tabs, "\n"), ui.Rect(g, y, w-2*g, len(tabs)))
		y += len(tabs) + t.Gap()
	}
	body := uv.Rect(g, y, w-2*g, max(0, fy-y-t.Gap()))
	switch m.screen {
	case scrServices:
		m.services(c, body)
	case scrDeploys:
		m.deploys(c, body)
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
	case ovHelp:
		m.keysOverlay(c)
	}
	if m.ov != ovNone {
		// the footer now describes the dialog's keys: it sits above the scrim
		c.Draw(strings.Join(foot, "\n"), ui.Rect(0, fy, w, len(foot)))
	}
	if m.toast != nil {
		c.Draw(ui.Toast(t, m.toast.kind, m.toast.label, m.toast.msg, w), ui.Rect(0, h-1, w, 1))
	}
	return c
}

func (m *app) footerBadge() string {
	if m.screen == scrDeploy && m.ov == ovNone {
		return ui.Tag(m.t, "warning", "PROD")
	}
	return ""
}

func (m *app) footerRight() string {
	if m.screen == scrServices && m.ov == ovNone {
		return m.t.S.Subtle.Render("synced 4s ago")
	}
	return ""
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
	fixed := 2
	for _, x := range widths[1:] {
		fixed += x + 2
	}
	widths[0] = max(12, main.Dx()-fixed)
	var rows [][]string
	for i, s := range m.svcs {
		name := ui.Dot(t, s.State) + " " + t.S.Base.Render(s.Name)
		if s.State == "deploying" {
			name = ui.Spinner(t, m.frame/2) + " " + t.S.Base.Render(s.Name)
		}
		// red lives on the dot and the number; the sparkline stays quiet
		spark := t.S.Muted
		if s.State == "warning" {
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
	s := m.svcs[m.sel]
	if side.Dx() > 0 {
		m.serviceDetail(c, side)
		// tall terminals: use the room for the activity feed, not blank rows
		if top := main.Min.Y + len(tbl) + 2; main.Max.Y-top >= 6 {
			act := []string{ui.Section(t, "Activity", main.Dx(), "last hour"), ""}
			for _, e := range activity {
				act = append(act, "  "+t.S.Subtle.Render(e[0])+"  "+ui.Dot(t, e[3])+" "+t.S.Soft.Render(e[1])+"  "+t.S.Muted.Render(e[2]))
			}
			n := min(len(act), main.Max.Y-top)
			c.Draw(strings.Join(act[:n], "\n"), uv.Rect(main.Min.X, top, main.Dx(), n))
		}
	} else if y := main.Min.Y + len(tbl) + 1; y < main.Max.Y {
		line := t.S.Muted.Render(s.Name) + t.S.Subtle.Render("  owner ") + t.S.Soft.Render(s.Owner) +
			t.S.Subtle.Render("  deployed ") + t.S.Soft.Render(s.Deployed)
		c.Draw(line, uv.Rect(main.Min.X+2, y, main.Dx()-2, 1))
		if s.State == "error" && y+2 < main.Max.Y {
			c.Draw(ui.Alert(t, "error", "502s", "4.8% of checkout requests failing since 14:02 · r roll back", main.Dx()-2), uv.Rect(main.Min.X+2, y+2, main.Dx()-2, 1))
		}
	}
}

var stateTag = map[string][2]string{
	"error": {"error", "FAILING"}, "ok": {"ok", "HEALTHY"}, "warning": {"warning", "DEGRADED"},
	"deploying": {"primary", "DEPLOYING"}, "idle": {"muted", "IDLE"},
}

func (m *app) serviceDetail(c *ui.Canvas, r uv.Rectangle) {
	t := m.t
	s := m.svcs[m.sel]
	w := r.Dx()
	tag := stateTag[s.State]
	rows := []string{ui.Section(t, s.Name, w, s.Env), ""}
	rows = append(rows, ui.KV(t, [][2]string{
		{"status", ui.Tag(t, tag[0], tag[1])},
		{"version", t.S.Base.Render(s.Version)},
		{"owner", t.S.Base.Render(s.Owner)},
		{"deployed", t.S.Base.Render(s.Deployed)},
	}, 9)...)
	rows = append(rows, "", ui.Section(t, "p99 latency", w, "12 min"), "")
	rows = append(rows, ui.Sparkline(t, stretch(s.P99, w), w, t.S.Muted))
	last := s.P99[len(s.P99)-1]
	trend := t.S.Base.Render(itoa(int(last)) + "ms")
	if s.P99[0] > 0 && last/s.P99[0] > 2 {
		trend += "  " + t.S.Err.Render(t.G.Arrow+" "+ftoa(last/s.P99[0])+"× since 14:00")
	}
	rows = append(rows, trend)
	if s.State == "error" {
		rows = append(rows, "", ui.Section(t, "Recent", w, ""), "")
		ev := [][2]string{{"14:03", "circuit open · payments"}, {"14:02", "502 rate crossed 1%"}, {"14:00", "deployed " + s.Version + " by @mara"}}
		for _, e := range ev {
			rows = append(rows, t.S.Subtle.Render(e[0])+"  "+t.S.Soft.Render(e[1]))
		}
		rows = append(rows, "", t.S.Key.Render("r")+" "+t.S.KeyDesc.Render("roll back to "+s.Prev))
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

// ---------------------------------------------------------------- deploys

func (m *app) deploys(c *ui.Canvas, r uv.Rectangle) {
	t := m.t
	rows := [][]string{}
	for _, d := range deployLog {
		st := ui.Dot(t, d[4])
		if d[4] == "busy" {
			st = ui.Spinner(t, m.frame/2)
		}
		rows = append(rows, []string{st + " " + t.S.Base.Render(d[1]), t.S.Soft.Render(d[2]), t.S.Muted.Render(d[3]), t.S.Subtle.Render(d[0])})
	}
	widths := []int{0, 16, 10, 8}
	fixed := 2 + (16 + 2) + (10 + 2) + (8 + 2)
	widths[0] = max(12, r.Dx()-fixed)
	tbl := ui.Table(t, []string{"service", "version", "by", "when"}, rows, widths, "lllr", m.sel%len(deployLog), r.Dx(), m.ov == ovNone)
	c.Draw(strings.Join(tbl, "\n"), r)
}

// ---------------------------------------------------------------- incidents

func (m *app) incidents(c *ui.Canvas, r uv.Rectangle) {
	t := m.t
	act := t.S.Key.Render("n") + " " + t.S.KeyDesc.Render("declare incident") + "   " + t.S.Key.Render("h") + " " + t.S.KeyDesc.Render("history")
	title, why := "No incidents declared", "checkout-api has been failing for 12 min. Declare one to page payments on-call."
	if m.declared {
		title, why, act = "INC-2041 declared", "Paging payments on-call. Updates will appear here as responders join.", ""
	}
	rows := ui.Empty(t, r.Dx(), t.G.FieldB, title, why, act)
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
			diff[i] = t.S.OK.Render(l)
		case strings.HasPrefix(l, "-"):
			diff[i] = t.S.Err.Render(l)
		default: // hunk headers sit on the panel colour: muted, not subtle
			diff[i] = t.S.Muted.Render(l)
		}
	}
	shown := 4
	if m.expand {
		shown = len(logLines)
	}
	var s []string
	s = append(s, ui.UserMessage(t, chatUser, w, false)...)
	s = append(s, "")
	s = append(s, ui.Thinking(t, thinkingLines, 2, "6.1s", w)...)
	s = append(s, "")
	s = append(s, ui.ToolCall(t, ui.Tool{Name: "Logs", Param: "checkout-api", Opts: [][2]string{{"since", "14:00"}, {"tail", "200"}},
		State: ui.ToolDone, Body: logLines, Shown: shown, Expand: m.expand}, w, false)...)
	s = append(s, "")
	s = append(s, ui.ToolCall(t, ui.Tool{Name: "Grep", Param: "PAYMENTS_TIMEOUT", Opts: [][2]string{{"path", "deploy/"}},
		State: ui.ToolError, Err: "deploy/: no such directory (did you mean config/?)"}, w, false)...)
	s = append(s, "")
	s = append(s, ui.ToolCall(t, ui.Tool{Name: "Edit", Param: "config/timeouts.go", State: ui.ToolDone, Body: diff, Expand: true}, w, false)...)
	s = append(s, "")
	s = append(s, ui.AssistantMessage(t, chatAnswer, w, false)...)
	s = append(s, "")
	s = append(s, ui.TurnFooter(t, "opus", "anthropic", "41s", w))
	for _, msg := range m.sent {
		switch msg {
		case "\x00done":
			s = append(s, "")
			s = append(s, ui.ToolCall(t, ui.Tool{Name: "Deploy", Param: "checkout-api v2.4.1", Opts: [][2]string{{"strategy", "rolling"}},
				State: ui.ToolDone, Body: []string{"3/3 pods healthy · error rate 0.02% · p99 124ms"}}, w, false)...)
		case "\x00canceled":
			s = append(s, "", "  "+t.S.Subtle.Render("Canceled."))
		default:
			s = append(s, "")
			s = append(s, ui.UserMessage(t, msg, w, false)...)
		}
	}
	editorH := 3
	streamH := main.Dy() - editorH - 1
	if m.busy {
		s = append(s, "", "  "+ui.NewAnim(t, 10, "Deploying checkout-api v2.4.1", "deploy").Frame(m.frame, elapsed(m.frame-m.busySince)))
	}
	// follow mode: the tail of the stream, minus how far the user scrolled up
	end := max(0, len(s)-m.scroll)
	if end > streamH {
		hidden := end - streamH + 1
		s = append([]string{"  " + t.S.Subtle.Render("↑ "+itoa(hidden)+" more lines  ") + t.S.Key.Render("pgup")}, s[hidden:end]...)
	} else {
		s = s[:end]
	}
	c.Draw(strings.Join(s, "\n"), uv.Rect(main.Min.X, main.Min.Y, w, streamH))
	ey := main.Max.Y - editorH
	prompt := t.S.Tertiary.Bold(true).Render(t.G.Prompt) + " "
	text := m.input
	if m.busy {
		prompt = t.S.Subtle.Render(t.G.Prompt) + " "
		text = ""
	}
	line := prompt + t.S.Base.Render(text)
	switch {
	case m.busy:
		line += t.S.Subtle.Render("working… esc to stop")
	case text == "":
		line += t.S.Tertiary.Render("▏") + t.S.Subtle.Render("ask, or tell it what to do")
	default:
		line += t.S.Tertiary.Render("▏")
	}
	c.Draw(strings.Join([]string{ui.Rule(t, w), line}, "\n"), uv.Rect(main.Min.X, ey, w, editorH))
}

func elapsed(frames int) string {
	s := frames * int(ui.FrameInterval.Milliseconds()) / 1000
	return itoa(s) + "s"
}

func (m *app) sessionSidebar(c *ui.Canvas, r uv.Rectangle) {
	t := m.t
	w := r.Dx()
	rows := []string{
		t.S.Bold.Render("checkout 502s"),
		t.S.Subtle.Render("started 14:06 · " + itoa(3+len(m.sent)) + " turns"),
		"",
		ui.Section(t, "Model", w, ""),
		t.S.Muted.Render(t.G.Todo+" ") + t.S.Base.Render("opus") + t.S.Subtle.Render(" via anthropic"),
		t.S.Subtle.Render("context ") + ui.Progress(t, 0.42, 10) + t.S.Subtle.Render(" 42%"),
		"",
		ui.Section(t, "Changed", w, "1"),
		t.S.Soft.Render("config/timeouts.go ") + t.S.OK.Render("+4") + " " + t.S.Err.Render("-1"),
		"",
		ui.Section(t, "Touched", w, ""),
	}
	for _, s := range m.svcs[:4] {
		rows = append(rows, ui.Dot(t, s.State)+" "+t.S.Soft.Render(ui.Trunc(t, s.Name, w-2)))
	}
	c.Draw(strings.Join(rows, "\n"), r)
}

// ---------------------------------------------------------------- deploy form

func (m *app) deployForm(c *ui.Canvas, r uv.Rectangle) {
	t := m.t
	s := m.svcs[m.sel]
	var main, side uv.Rectangle
	if r.Dx() >= wideW-4 {
		layout.Horizontal(layout.Len(min(64, r.Dx()-44)), layout.Len(4), layout.Fill(1)).Split(r).Assign(&main, new(uv.Rectangle), &side)
	} else {
		main = r
	}
	w := main.Dx()
	rows := []string{ui.Section(t, "Deploy "+s.Name, w, s.Env), ""}
	// a field is a group: one blank row between fields even in compact brands
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
		rows = append(rows, "")
	}
	choice := func(opts []string, on int, focused bool) string {
		var out []string
		for i, o := range opts {
			switch {
			case i == on && focused:
				out = append(out, t.S.Primary.Render(t.G.RadioOn)+" "+t.S.Bold.Render(o))
			case i == on:
				out = append(out, t.S.Muted.Render(t.G.RadioOn)+" "+t.S.Base.Render(o))
			default:
				out = append(out, t.S.Subtle.Render(t.G.RadioOff)+" "+t.S.Muted.Render(o))
			}
		}
		return strings.Join(out, "   ")
	}
	field(0, "Environment", "", choice([]string{"prod", "staging", "dev"}, m.env, m.focus == 0))
	input := t.S.Tertiary.Render(t.G.Prompt) + " " + t.S.Base.Render(m.version)
	if m.focus == 1 {
		input += t.S.Tertiary.Render("▏")
	}
	field(1, "Version", "tag or commit; latest is v2.4.1 (2 min ago)", input)
	strategies := [][2]string{{"Rolling", "one pod at a time"}, {"Canary", "10% for 15 min, then all"}, {"Blue/green", "switch traffic at once"}}
	field(2, "Strategy", strategies[m.strategy][1], choice([]string{"Rolling", "Canary", "Blue/green"}, m.strategy, m.focus == 2))
	check := func(i int, label string) string {
		g, st := t.G.CheckOff, t.S.Subtle
		if m.checks[i] {
			g, st = t.G.CheckOn, t.S.Primary
		}
		lab := t.S.Muted.Render(label)
		if m.focus == 3 && m.checkSel == i {
			lab = t.S.Bold.Render(label)
		}
		return st.Render(g) + " " + lab
	}
	field(3, "Checks", "", check(0, "run smoke tests")+"   "+check(1, "skip approval"))
	btns := ui.Buttons(t, w-2, lipgloss.Left,
		ui.Button(t, "Deploy", m.focus == 4 && m.formBtn == 0, false, -1),
		ui.Button(t, "Cancel", m.focus == 4 && m.formBtn == 1, false, -1))
	rows = append(rows, ui.Gutter(t, m.focus == 4, t.S.FocusBar)+btns)
	c.Draw(strings.Join(rows, "\n"), main)

	if side.Dx() > 0 {
		sw := side.Dx()
		steps := []string{t.S.Soft.Render("build image ") + t.S.Subtle.Render("cached")}
		if m.checks[0] {
			steps = append(steps, t.S.Soft.Render("run smoke tests ")+t.S.Subtle.Render("~40s"))
		}
		steps = append(steps, t.S.Soft.Render([]string{"roll 3 pods ", "canary 1 pod ", "start 3 new pods "}[m.strategy])+t.S.Subtle.Render(strategies[m.strategy][1]),
			t.S.Soft.Render("watch error rate ")+t.S.Subtle.Render("5 min, auto-rollback >1%"))
		pr := []string{ui.Section(t, "What happens", sw, ""), ""}
		pr = append(pr, t.S.Base.Render(s.Name+" ")+t.S.Muted.Render(s.Version+" "+t.G.Arrow+" ")+t.S.Primary.Bold(true).Render(m.version))
		pr = append(pr, "")
		pr = append(pr, ui.Tree(t, steps)...)
		if m.env == 0 {
			pr = append(pr, "", ui.Alert(t, "warning", "PROD", "customer traffic, 3 regions", sw))
		}
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

// dialogTop keeps dialogs below the header and tabs.
func dialogTop(c *ui.Canvas, dh int) *uv.Position {
	h := c.Bounds().Dy()
	y := max(4, (h-dh)/2-1)
	y = min(y, max(0, h-dh-1))
	return &uv.Position{X: -1, Y: y} // X < 0: centre horizontally
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
	listH := min(len(commands), max(3, c.Bounds().Dy()-14))
	body := []string{input, ""}
	if len(items) == 0 {
		body = append(body, t.S.Muted.Render("  no command matches “"+m.query+"”"))
		for range listH - 1 {
			body = append(body, "")
		}
	} else {
		body = append(body, ui.List(t, items, min(m.palSel, len(items)-1), 0, inner, listH, true)...)
	}
	// no help line inside: the footer above the scrim carries the dialog's keys
	d := ui.Dialog(t, w, "Commands", body, "", "")
	c.Overlay(d, dialogTop(c, lipgloss.Height(d)))
}

func (m *app) confirm(c *ui.Canvas) {
	t := m.t
	s := m.svcs[m.sel]
	w := min(58, c.Bounds().Dx()-6)
	inner := w - t.S.Dialog.GetHorizontalFrameSize()
	body := []string{t.S.Soft.Render(s.Env + " returns to the last healthy version."), ""}
	now := t.S.Base.Render(s.Version) + t.S.Subtle.Render("  deployed "+s.Deployed)
	if s.State == "error" {
		now = t.S.Err.Render(s.Version) + t.S.Subtle.Render("  failing 12 min")
	}
	body = append(body, ui.KV(t, [][2]string{
		{"now", now},
		{"after", t.S.OK.Render(s.Prev) + t.S.Subtle.Render("  the previous release")},
		{"impact", t.S.Base.Render("3 pods restart, one at a time (~90s)")},
	}, 7)...)
	btns := ui.Buttons(t, inner, lipgloss.Right,
		ui.Button(t, "Roll back", m.btn == 0, true, 0),
		ui.Button(t, "Cancel", m.btn == 1, false, 0))
	d := ui.Dialog(t, w, "Roll back "+s.Name+"?", body, btns, "")
	c.Overlay(d, dialogTop(c, lipgloss.Height(d)))
}

// keysOverlay is the full help: every binding of the screen underneath.
func (m *app) keysOverlay(c *ui.Canvas) {
	t := m.t
	ov := m.ov
	m.ov = ovNone
	acts := m.actions()
	m.ov = ov
	w := min(56, c.Bounds().Dx()-6)
	var body []string
	for _, a := range acts {
		h := a.b.Help()
		body = append(body, t.S.Key.Render(ui.Fit(h.Key, 12))+t.S.Soft.Render(h.Desc))
	}
	body = append(body, t.S.Key.Render(ui.Fit("ctrl+c", 12))+t.S.Soft.Render("quit from anywhere"))
	d := ui.Dialog(t, w, "Keys · "+tabNames[min(int(m.screen), 3)], body, "", "")
	c.Overlay(d, dialogTop(c, lipgloss.Height(d)))
}

// ---------------------------------------------------------------- splash

func (m *app) splash(c *ui.Canvas) {
	t := m.t
	b := c.Bounds()
	f := ui.DefaultField()
	f.ClearR, f.Dim, f.Drift = 0.42, 0.25, m.frame/3
	f.Draw(c, b)
	// a brand whose signature is its header (a flag strip, a stamp) shows it here too
	top := 0
	if t.X.Header == "bar" || t.X.Header == "stamp" {
		head := ui.Header(t, b.Dx(), b.Dy(), t.AppName(), t.S.Muted.Render("v0.9.0"))
		c.Draw(strings.Join(head, "\n"), uv.Rect(0, 0, b.Dx(), len(head)))
		top = len(head)
	}
	var block []string
	if rows := t.Wordmark.Rows; len(rows) > 0 && lipgloss.Width(rows[0]) < b.Dx()-4 {
		for _, r := range rows {
			block = append(block, bannerTone(t, r))
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
	y := max(top+1, (b.Dy()-len(block))/2-1)
	r := uv.Rect((b.Dx()-bw)/2, y, bw, len(block))
	c.Draw(strings.Join(block, "\n"), r)
	if top == 0 {
		c.Draw(t.S.Subtle.Render(t.AppName()+" v0.9.0"), uv.Rect(t.X.Gutter, b.Dy()-1, 30, 1))
	}
	q := t.S.Key.Render("q") + t.S.KeyDesc.Render(" quit")
	c.Draw(q, uv.Rect(b.Dx()-t.X.Gutter-lipgloss.Width(q), b.Dy()-1, lipgloss.Width(q), 1))
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
