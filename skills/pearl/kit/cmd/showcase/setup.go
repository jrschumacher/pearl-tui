package main

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/ultraviolet/layout"

	"github.com/jrschumacher/pearl-tui/skills/pearl/kit/theme/bubbletea"
	"github.com/jrschumacher/pearl-tui/skills/pearl/kit/ui/bubbletea"
)

// The first-run setup wizard: connect a cluster. The kit's fields do the input
// (ui.Select, ui.Input, ui.Confirm); the app does everything around them: the brand
// header, a step row that keeps counting after the inputs end, a summary that tells
// defaults from decisions, a sign-in wait with a way out, and the keys, from the same
// action table as every other screen.
//
// The fields live in setupFlow for the whole flow, so going back never loses what was
// typed. Review and sign-in have no input; their buttons are ui.Actions.

type setupStep int

const (
	stTarget setupStep = iota // cluster (select) + namespace (input), one screen
	stAuth                    // how to sign in (select with descriptions)
	stAlerts                  // notifications (confirm)
	stReview                  // summary + Continue / Back to edit / Discard setup
	stSignin                  // waiting for approval
	stDone                    // make it the default? Yes / No
)

// the counter runs past the review: sign-in and finishing are steps too
var setupNames = []string{"Cluster", "Sign-in", "Alerts", "Review", "Connect", "Finish"}

var clusterChoices = []ui.Option[string]{
	{Name: "prod-us-east-1", Desc: "production", Value: "prod-us-east-1"},
	{Name: "staging-eu-west-1", Desc: "current context", Value: "staging-eu-west-1"},
	{Name: "dev-local", Desc: "kind on this machine", Value: "dev-local"},
}

var authChoices = []ui.Option[string]{
	{Name: "Browser", Desc: "single sign-on in a tab", Value: "browser"},
	{Name: "Device code", Desc: "approve on another device", Value: "device"},
	{Name: "Access token", Desc: "paste one from your profile", Value: "token"},
}

const deviceCode = "WXKD-4PQM"

type setupFlow struct {
	step    setupStep
	reached setupStep // the furthest step entered: values of earlier steps are decisions

	cluster, auth       ui.Select[string]
	namespace           ui.Input
	alerts, makeDefault ui.Confirm
	field               int // stTarget's focused field: 0 the cluster, 1 the namespace

	acts     ui.Actions // review: Continue / Back to edit / Discard setup. failed sign-in: Retry / Back
	since    int        // frame the sign-in started
	failed   bool
	gen      int  // sign-in attempt; a result from an older one is ignored
	escArmed bool // esc pressed once on the first step with something typed
	saved    bool // the review was accepted
}

type signedInMsg struct{ gen int }
type signinTimeoutMsg struct{ gen int }

// the defaults are what the user gets by pressing enter all the way through. Field
// titles are the app's copy, in the brand's case (t.Copy) like the kit's own labels.
func newSetup(t *theme.Theme) *setupFlow {
	brand := t.AppName()
	ns := ui.NewInput(t.Copy("Namespace"), "where your services run; blank means default", "default", "")
	ns.Validate = validNamespace
	return &setupFlow{
		cluster:     ui.NewSelect(t.Copy("Cluster"), "contexts in ~/.kube/config", "staging-eu-west-1", clusterChoices...),
		namespace:   ns,
		auth:        ui.NewSelect(t.Copy("Sign in with"), "change it later with "+brand+" auth", "browser", authChoices...),
		alerts:      ui.NewConfirm("Tell you when a deploy fails?", "a desktop notification and the terminal bell", "Turn on", "Not now", true),
		makeDefault: ui.NewConfirm("", brand+" commands use it when you don't pass --cluster", "Yes", "No", true),
	}
}

func (f *setupFlow) steps(t *theme.Theme) ui.Steps {
	names := make([]string, len(setupNames))
	for i, n := range setupNames {
		names[i] = t.Copy(n) // the wizard's step names in the brand's case, like the kit's own copy
	}
	return ui.Steps{Names: names, Cur: int(f.step), Failed: f.failed}
}

func (m *app) openSetup() tea.Cmd {
	m.screen, m.setup = scrSetup, newSetup(m.t)
	return m.goStep(stTarget)
}

// goStep enters a step with its first field focused, or its buttons.
func (m *app) goStep(s setupStep) tea.Cmd {
	f := m.setup
	f.step, f.failed, f.escArmed, f.field = s, false, false, 0
	f.reached = max(f.reached, s)
	switch s {
	case stReview:
		// the expected forward choice is the default; each button says what it does: a bare
		// "Cancel" on a review left the user asking whether their answers were lost
		f.acts = ui.NewActions(0, "Continue", "Back to edit", "Discard setup")
	case stSignin:
		return m.startSignin()
	case stDone:
		f.makeDefault.Title = "Make " + f.cluster.Value() + " your default cluster?"
	}
	return nil
}

func (m *app) startSignin() tea.Cmd {
	f := m.setup
	f.step, f.failed, f.saved = stSignin, false, true
	f.gen++
	f.since = m.frame
	gen := f.gen
	// the fixture's identity provider approves after a few seconds; a real
	// one answers a poll. Either way the result is a message, never a wait.
	return tea.Batch(
		tea.Tick(5*time.Second, func(time.Time) tea.Msg { return signedInMsg{gen} }),
		tea.Tick(5*time.Minute, func(time.Time) tea.Msg { return signinTimeoutMsg{gen} }),
	)
}

func validNamespace(s string) error {
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-') {
			return errors.New("a namespace is lowercase letters, digits and -; no spaces")
		}
	}
	return nil
}

// next submits the focused field: the cluster moves on to the namespace, the namespace
// must pass its check, and a step's last field enters the next step.
func (m *app) next() tea.Cmd {
	f := m.setup
	switch f.step {
	case stTarget:
		if f.field == 0 {
			f.field = 1
			return nil
		}
		if !f.namespace.Submit() {
			return nil // the error shows under the field, which keeps the focus
		}
		return m.goStep(stAuth)
	case stAuth:
		return m.goStep(stAlerts)
	case stAlerts:
		return m.goStep(stReview)
	case stDone:
		return m.finishSetup(true)
	}
	return nil
}

// typing reports whether the focused field takes text.
func (f *setupFlow) typing() bool {
	return f != nil && f.step == stTarget && f.field == 1
}

// confirm is the step's confirm, or nil.
func (f *setupFlow) confirm() *ui.Confirm {
	switch f.step {
	case stAlerts:
		return &f.alerts
	case stDone:
		return &f.makeDefault
	}
	return nil
}

// choice is the focused select, or nil.
func (f *setupFlow) choice() *ui.Select[string] {
	switch {
	case f.step == stTarget && f.field == 0:
		return &f.cluster
	case f.step == stAuth:
		return &f.auth
	}
	return nil
}

func (m *app) finishSetup(connected bool) tea.Cmd {
	f := m.setup
	m.screen = scrServices
	switch {
	case !connected:
		return m.say("info", "SKIPPED", "Nothing was saved "+m.t.G.Sep+" run "+m.t.AppName()+" connect any time")
	case f.makeDefault.Value():
		return m.say("ok", "CONNECTED", f.cluster.Value()+" is your default cluster "+m.t.G.Sep+" 38 services")
	default:
		return m.say("ok", "CONNECTED", f.cluster.Value()+" "+m.t.G.Sep+" pass --cluster to use it")
	}
}

// setupActions is the action table for the wizard: one per step, and per focused
// field. Keys reach a field only through it, so the footer is the whole truth about
// what a key does.
func (m *app) setupActions(help action) []action {
	f := m.setup
	do := func(keys []string, k, desc string, show bool, fn func(m *app, msg tea.KeyPressMsg) tea.Cmd) action {
		return action{bind(keys, k, desc), show, fn}
	}
	next := func(m *app, _ tea.KeyPressMsg) tea.Cmd { return m.next() }
	esc := func(desc string, fn func(m *app) tea.Cmd) action {
		return do([]string{"esc"}, "esc", desc, true, func(m *app, _ tea.KeyPressMsg) tea.Cmd { return fn(m) })
	}
	back := func(to setupStep) action {
		return esc("back", func(m *app) tea.Cmd { return m.goStep(to) })
	}
	// left and right move between buttons: a review's, a failed sign-in's, a confirm's
	switchKeys := []string{"left", "right", "h", "l", "tab", "shift+tab"}
	switchBtn := func(acts func(m *app) *ui.Actions) action {
		return do(switchKeys, m.t.G.LeftRight, "switch", true, func(m *app, msg tea.KeyPressMsg) tea.Cmd {
			if s := msg.String(); s == "left" || s == "h" || s == "shift+tab" {
				acts(m).Move(-1)
			} else {
				acts(m).Move(1)
			}
			return nil
		})
	}
	setupActs := func(m *app) *ui.Actions { return &m.setup.acts }

	switch f.step {
	case stReview:
		return []action{
			do([]string{"enter"}, "enter", strings.ToLower(f.acts.Selected()), true, func(m *app, _ tea.KeyPressMsg) tea.Cmd { return m.reviewChoose() }),
			switchBtn(setupActs),
			do([]string{"e"}, "e", "edit", false, func(m *app, _ tea.KeyPressMsg) tea.Cmd { return m.goStep(stTarget) }),
			back(stAlerts), help,
		}
	case stSignin:
		if f.failed {
			return []action{
				do([]string{"enter"}, "enter", strings.ToLower(f.acts.Selected()), true, func(m *app, _ tea.KeyPressMsg) tea.Cmd {
					if m.setup.acts.Sel == 0 {
						return m.startSignin()
					}
					return m.goStep(stReview)
				}),
				switchBtn(setupActs),
				do([]string{"r"}, "r", "retry", false, func(m *app, _ tea.KeyPressMsg) tea.Cmd { return m.startSignin() }),
				back(stReview), help,
			}
		}
		acts := []action{esc("cancel sign-in", func(m *app) tea.Cmd {
			m.setup.gen++ // the pending result is now stale
			return tea.Batch(m.say("info", "CANCELED", "Sign-in canceled "+m.t.G.Sep+" nothing was sent to "+m.setup.cluster.Value()), m.goStep(stReview))
		})}
		if f.auth.Value() != "token" {
			acts = append(acts,
				do([]string{"c"}, "c", "copy code", true, func(m *app, _ tea.KeyPressMsg) tea.Cmd {
					return tea.Batch(m.say("ok", "COPIED", deviceCode), tea.SetClipboard(deviceCode))
				}),
				do([]string{"o"}, "o", "open page", true, func(m *app, _ tea.KeyPressMsg) tea.Cmd {
					return m.say("info", "OPENED", m.signinURL())
				}))
		}
		return append(acts, help)
	}

	// a step of fields: the keys depend on the focused one
	var escAct action
	switch f.step {
	case stTarget:
		escAct = esc("quit setup", func(m *app) tea.Cmd {
			if m.setup.namespace.Value() != "" && !m.setup.escArmed {
				m.setup.escArmed = true // never drop typed input on one key
				return m.say("warning", "ESC AGAIN", "to discard what you typed and leave setup")
			}
			return m.finishSetup(false)
		})
	case stDone:
		escAct = esc("skip", func(m *app) tea.Cmd { m.setup.makeDefault.Set(false); return m.finishSetup(true) })
	default:
		escAct = back(f.step - 1)
	}
	switch c := f.confirm(); {
	case f.typing():
		// printable keys type (app.key sends them here before any action)
		edit := func(m *app, msg tea.KeyPressMsg) tea.Cmd { return m.setup.namespace.Update(msg) }
		return []action{
			do([]string{"enter"}, "enter", "next", true, next),
			do([]string{"shift+tab"}, "shift+tab", "previous field", false, func(m *app, _ tea.KeyPressMsg) tea.Cmd { m.setup.field = 0; return nil }),
			do([]string{"backspace"}, "backspace", "delete", false, edit),
			do([]string{"left", "right", "home", "end"}, m.t.G.LeftRight, "move", false, edit),
			escAct, help,
		}
	case c != nil:
		answer := func(v bool) func(m *app, _ tea.KeyPressMsg) tea.Cmd {
			return func(m *app, _ tea.KeyPressMsg) tea.Cmd { m.setup.confirm().Set(v); return m.next() }
		}
		return []action{
			do([]string{"enter"}, "enter", strings.ToLower(c.Acts.Selected()), true, next),
			do([]string{"left", "right", "h", "l"}, m.t.G.LeftRight, "switch", true, func(m *app, msg tea.KeyPressMsg) tea.Cmd {
				if s := msg.String(); s == "left" || s == "h" {
					m.setup.confirm().Acts.Move(-1)
				} else {
					m.setup.confirm().Acts.Move(1)
				}
				return nil
			}),
			do([]string{"y"}, "y", "yes", false, answer(true)),
			do([]string{"n"}, "n", "no", false, answer(false)),
			escAct, help,
		}
	}
	acts := []action{
		do([]string{"enter"}, "enter", "next", true, next),
		do([]string{"up", "k", "down", "j"}, m.t.G.UpDown, "choose", true, func(m *app, msg tea.KeyPressMsg) tea.Cmd {
			if s := msg.String(); s == "up" || s == "k" {
				m.setup.choice().Move(-1)
			} else {
				m.setup.choice().Move(1)
			}
			return nil
		}),
	}
	if f.step == stTarget {
		acts = append(acts, do([]string{"tab"}, "tab", "next field", false, func(m *app, _ tea.KeyPressMsg) tea.Cmd { m.setup.field = 1; return nil }))
	}
	return append(acts, escAct, help)
}

func (m *app) reviewChoose() tea.Cmd {
	switch m.setup.acts.Sel {
	case 0:
		return m.goStep(stSignin)
	case 1:
		return m.goStep(stTarget)
	}
	return m.finishSetup(false)
}

func (m *app) signinURL() string { return signinURL(m.t) }

func signinURL(t *theme.Theme) string { return "https://login." + t.AppName() + ".example/device" }

// summary is what the flow will do, telling decisions from defaults: a value
// is a decision once the user has moved past the field that sets it.
func (m *app) summary() []ui.SummaryRow {
	f := m.setup
	past := func(s setupStep) bool { return f.reached > s }
	clusterSet := past(stTarget) || f.typing()
	ns := f.namespace.Value()
	if ns == "" {
		ns = "default"
	}
	alerts := "off"
	if f.alerts.Value() {
		alerts = "on for failed deploys"
	}
	return []ui.SummaryRow{
		{Label: "Cluster", Value: f.cluster.Value(), Set: clusterSet},
		{Label: "Namespace", Value: ns, Set: past(stTarget) && f.namespace.Value() != ""},
		{Label: "Sign-in", Value: f.auth.Selected().Name, Set: past(stAuth)},
		{Label: "Alerts", Value: alerts, Set: past(stAlerts)},
	}
}

// ---------------------------------------------------------------- render

// setupView lays the wizard out: the fields at a measure, the summary in the
// rest of the width up to a cap. Below 100 columns the summary is a strip.
func (m *app) setupView(c *ui.Canvas, r uv.Rectangle) {
	t, f := m.t, m.setup
	var main, side, strip uv.Rectangle
	if r.Dx() >= 96 {
		fw := min(84, r.Dx()*3/5)
		sw := min(64, r.Dx()-fw-4)
		layout.Horizontal(layout.Len(fw), layout.Len(4), layout.Len(sw), layout.Fill(1)).Split(r).
			Assign(&main, new(uv.Rectangle), &side, new(uv.Rectangle))
	} else {
		main = r
		if f.step != stReview && r.Dy() >= 14 {
			main.Max.Y -= 2
			strip = uv.Rect(r.Min.X, r.Max.Y-1, r.Dx(), 1)
		}
	}
	w := main.Dx()
	title, info, intro := m.setupCopy()
	rows := []string{ui.Section(t, title, w, info), ""}
	if intro != "" { // wrapped at the measure by words, never cut mid-word
		for _, l := range strings.Split(lipgloss.Wrap(intro, min(w, t.X.Measure), ""), "\n") {
			rows = append(rows, t.S.Soft.Render(strings.TrimSpace(l)))
		}
		rows = append(rows, "")
	}
	focused := m.ov == ovNone // under a dialog no field has the focus
	switch f.step {
	case stTarget:
		rows = append(rows, f.cluster.View(t, w, focused && f.field == 0)...)
		rows = append(append(rows, ""), f.namespace.View(t, w, focused && f.field == 1)...)
	case stAuth:
		rows = append(rows, f.auth.View(t, w, focused)...)
	case stAlerts:
		rows = append(rows, f.alerts.View(t, w, focused)...)
	case stReview:
		rows = append(rows, m.reviewRows(w, focused)...)
	case stSignin:
		rows = append(rows, m.signinRows(w, focused)...)
	case stDone:
		rows = append(rows, ui.Tag(t, "ok", "CONNECTED")+" "+t.S.Base.Render(f.cluster.Value())+
			t.S.Muted.Render(ui.Trunc(t, " as mara@"+t.AppName()+".example "+t.G.Sep+" 38 services", max(0, w-lipgloss.Width(f.cluster.Value())-12))), "")
		rows = append(rows, f.makeDefault.View(t, w, focused)...)
	}
	c.Draw(strings.Join(rows, "\n"), main)

	tag := ui.Tag(t, "muted", "DRAFT")
	if f.saved {
		tag = ui.Tag(t, "ok", "SAVED")
	}
	if side.Dx() > 0 {
		var srows []string
		if f.step == stReview {
			srows = m.nextRows(side.Dx())
		} else {
			srows = ui.Summary(t, "Summary", tag, m.summary(), side.Dx())
		}
		c.Draw(strings.Join(srows, "\n"), side)
	}
	if strip.Dx() > 0 {
		c.Draw(ui.SummaryStrip(t, m.summary(), strip.Dx()), strip)
	}
}

// setupCopy is each step's title, right-hand info and one-line intro.
func (m *app) setupCopy() (title, info, intro string) {
	f, brand := m.setup, m.t.AppName()
	switch f.step {
	case stTarget:
		return "Connect a cluster", "", "Pick the cluster " + brand + " watches. Add more later with " + brand + " connect."
	case stAuth:
		return "Sign-in", f.cluster.Value(), ""
	case stAlerts:
		return "Alerts", f.cluster.Value(), ""
	case stReview:
		return "Review", "nothing saved yet", ""
	case stSignin:
		return "Sign in to " + f.cluster.Value(), "", ""
	}
	return "Connected", "", ""
}

func (m *app) reviewRows(w int, focused bool) []string {
	t, f := m.t, m.setup
	var kv [][2]string
	for _, r := range m.summary() {
		v := t.S.Base.Render(r.Value)
		if !r.Set {
			v = t.S.Subtle.Render(r.Value + "  (default)")
		}
		kv = append(kv, [2]string{r.Label, v})
	}
	rows := ui.KV(t, kv, 11)
	for i := range rows {
		rows[i] = "  " + ui.Trunc(t, rows[i], w-2)
	}
	return append(rows, "", f.acts.View(t, focused))
}

// nextRows is the review's side panel: what Continue will do.
func (m *app) nextRows(w int) []string {
	t, f := m.t, m.setup
	how := map[string]string{"browser": "open SSO in your browser", "device": "show a code to approve elsewhere", "token": "check your access token"}[f.auth.Value()]
	rows := []string{ui.Section(t, "What happens next", w, ""), ""}
	items := []string{
		t.S.Soft.Render(how),
		t.S.Soft.Render("save the context ") + t.S.Subtle.Render("~/.config/"+t.AppName()),
		t.S.Soft.Render("watch 38 services ") + t.S.Subtle.Render("on "+f.cluster.Value()),
	}
	for i, it := range items { // cut with the ellipsis, never clipped by the panel edge
		items[i] = ui.Trunc(t, it, max(1, w-lipgloss.Width(t.G.TreeMid)-1))
	}
	return append(rows, ui.Tree(t, items)...)
}

func (m *app) signinRows(w int, focused bool) []string {
	t, f := m.t, m.setup
	secs := (m.frame - f.since) * int(ui.FrameInterval.Milliseconds()) / 1000
	cluster, auth := f.cluster.Value(), f.auth.Value()
	if f.failed {
		return []string{
			ui.Alert(t, "error", "TIMED OUT", "No approval within 5 min; the code expired.", w), "",
			t.S.Soft.Render(ui.Trunc(t, "Nothing was saved on "+cluster+". Retry for a new code,", w)),
			t.S.Soft.Render(ui.Trunc(t, "or go back and choose another way to sign in.", w)), "",
			f.acts.View(t, focused),
		}
	}
	wait := map[string]string{"browser": "Waiting for you to approve in the browser", "device": "Waiting for approval", "token": "Checking your token"}[auth]
	rows := []string{ui.Spinner(t, m.frame/2) + " " + t.S.Soft.Render(ui.Trunc(t, wait, w-8)) + t.S.Subtle.Render("  "+itoa(secs)+"s"), ""}
	if auth == "token" {
		return rows
	}
	lead := "If the browser didn't open, go to"
	if auth == "device" {
		lead = "On any device, go to"
	}
	url := m.signinURL()
	left := max(0, 300-secs)
	return append(rows,
		t.S.Muted.Render(lead),
		"  "+t.S.Tertiary.Underline(true).Hyperlink(url).Render(ui.Trunc(t, url, w-2)), "",
		t.S.Muted.Render("and enter the code"),
		"  "+t.S.Panel.Foreground(t.P.Fg).Bold(true).Padding(0, 2).Render(deviceCode), "",
		t.S.Subtle.Render(fmt.Sprintf("The code expires in %d:%02d.", left/60, left%60)),
	)
}

// ---------------------------------------------------------------- accessible

// runAccessibleSetup is the same flow for ACCESSIBLE=1: the same fields asked as plain
// prompts (ui.Prompter), then status as new lines, never repainted: no spinner, no
// timer, no alt screen, no colour.
func runAccessibleSetup(t *theme.Theme, in io.Reader, out io.Writer, wait time.Duration) error {
	f, p := newSetup(t), ui.NewPrompter(in, out)
	for _, ask := range []func(*ui.Prompter) error{f.cluster.Ask, f.namespace.Ask, f.auth.Ask, f.alerts.Ask} {
		if err := ask(p); err != nil {
			return err
		}
	}
	cluster := f.cluster.Value()
	p.Say("Signing in to %s.", cluster)
	if f.auth.Value() != "token" {
		p.Say("Go to %s and enter the code %s.", signinURL(t), deviceCode)
	}
	p.Say("Waiting for approval.")
	time.Sleep(wait)
	p.Say("Signed in to %s as mara@%s.example. 38 services.", cluster, t.AppName())
	f.makeDefault.Title = "Make " + cluster + " your default cluster?"
	if err := f.makeDefault.Ask(p); err != nil {
		return err
	}
	if f.makeDefault.Value() {
		p.Say("Saved. %s is your default cluster.", cluster)
	} else {
		p.Say("Saved. Pass --cluster %s to use it.", cluster)
	}
	return nil
}
