package main

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/ultraviolet/layout"

	"github.com/jrschumacher/pearl-tui/skills/pearl/kit/theme/bubbletea"
	"github.com/jrschumacher/pearl-tui/skills/pearl/kit/ui/bubbletea"
)

// The first-run setup wizard: connect a cluster. Huh does the input mechanics
// (the selects, the namespace field, the confirms); the kit does everything
// around them: the brand header, a step row that keeps counting after the
// form ends, a summary that tells defaults from decisions, a sign-in wait with
// a way out, and the keys, from the same action table as every other screen.
//
// One Huh form per input step, rebuilt when the step is entered: the values
// live in setupFlow (Huh binds pointers to them), so going back never loses
// what was typed. Review and sign-in have no input, so they are kit screens.

type setupStep int

const (
	stTarget setupStep = iota // cluster (select) + namespace (input)
	stAuth                    // how to sign in (select with descriptions)
	stAlerts                  // notifications (confirm)
	stReview                  // summary + Continue / Back to edit / Discard setup
	stSignin                  // waiting for approval
	stDone                    // make it the default? Yes / No
)

// the counter runs past the review: sign-in and finishing are steps too
var setupNames = []string{"Cluster", "Sign-in", "Alerts", "Review", "Connect", "Finish"}

var clusterChoices = []huh.Option[string]{
	huh.NewOption("prod-us-east-1", "prod-us-east-1"),
	huh.NewOption("staging-eu-west-1", "staging-eu-west-1"),
	huh.NewOption("dev-local", "dev-local"),
}

var authChoices = []huh.Option[string]{
	huh.NewOption("Browser", "browser"),
	huh.NewOption("Device code", "device"),
	huh.NewOption("Access token", "token"),
}

const deviceCode = "WXKD-4PQM"

type setupFlow struct {
	step    setupStep
	reached setupStep // the furthest step entered: values of earlier steps are decisions
	form    *huh.Form

	cluster, namespace, auth string
	alerts, makeDefault      bool

	acts     ui.Actions // review: Continue / Back to edit / Discard setup. failed sign-in: Retry / Back
	since    int        // frame the sign-in started
	failed   bool
	gen      int  // sign-in attempt; a result from an older one is ignored
	escArmed bool // esc pressed once on the first step with something typed
	saved    bool // the review was accepted
}

type signedInMsg struct{ gen int }
type signinTimeoutMsg struct{ gen int }

// the defaults are what the user gets by pressing enter all the way through
func newSetup() *setupFlow {
	return &setupFlow{cluster: "staging-eu-west-1", auth: "browser", alerts: true, makeDefault: true}
}

func (f *setupFlow) steps(t *theme.Theme) ui.Steps {
	names := make([]string, len(setupNames))
	for i, n := range setupNames {
		names[i] = t.Copy(n) // the wizard's step names in the brand's case, like the kit's own copy
	}
	return ui.Steps{Names: names, Cur: int(f.step), Failed: f.failed}
}

func (m *app) openSetup() tea.Cmd {
	m.screen, m.setup = scrSetup, newSetup()
	return m.goStep(stTarget)
}

// goStep enters a step: builds its form (or its buttons) from the values so far.
func (m *app) goStep(s setupStep) tea.Cmd {
	f := m.setup
	f.step, f.failed, f.escArmed = s, false, false
	f.reached = max(f.reached, s)
	f.form = m.buildForm(s)
	switch s {
	case stReview:
		// the expected forward choice is the default; each button says what it does: a bare
		// "Cancel" on a review left the user asking whether their answers were lost
		f.acts = ui.NewActions(0, "Continue", "Back to edit", "Discard setup")
	case stSignin:
		return m.startSignin()
	}
	if f.form != nil {
		return f.form.Init()
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

// buildForm is one step's Huh form: Huh's own fields in ui.NewForm, which
// gives them the theme. Each confirm names its action first and opens on the
// value the flow already holds.
func (m *app) buildForm(s setupStep) *huh.Form {
	t, f := m.t, m.setup
	brand := t.AppName()
	var g *huh.Group
	switch s {
	case stTarget:
		// the input has a group of its own: Huh's theme reaches only a focused
		// input, so one left blurred beside another field draws in Bubbles' defaults
		return ui.NewForm(t, 60, 12,
			huh.NewGroup(huh.NewSelect[string]().Title(t.Copy("Cluster")).Description("contexts in ~/.kube/config").
				Options(clusterChoices...).Value(&f.cluster)),
			huh.NewGroup(huh.NewInput().Prompt(t.G.Prompt+" ").Title(t.Copy("Namespace")).Description("where your services run; blank means default").
				Placeholder("default").Value(&f.namespace).Validate(validNamespace)),
		)
	case stAuth:
		g = huh.NewGroup(huh.NewSelect[string]().Title(t.Copy("Sign in with")).
			Description("change it later with " + brand + " auth").
			Options(authChoices...).Value(&f.auth))
	case stAlerts:
		g = huh.NewGroup(huh.NewConfirm().Title("Tell you when a deploy fails?").Affirmative("Turn on").Negative("Not now").Value(&f.alerts).
			Description("a desktop notification and the terminal bell"))
	case stDone:
		g = huh.NewGroup(huh.NewConfirm().Title("Make " + f.cluster + " your default cluster?").Affirmative("Yes").Negative("No").Value(&f.makeDefault).
			Description(brand + " commands use it when you don't pass --cluster"))
	default:
		return nil
	}
	return ui.NewForm(t, 60, 12, g)
}

func validNamespace(s string) error {
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-') {
			return errors.New("a namespace is lowercase letters, digits and -; no spaces")
		}
	}
	return nil
}

// toForm hands a message to the step's form, and moves on when it completes.
func (m *app) toForm(msg tea.Msg) tea.Cmd {
	f := m.setup
	if f == nil || f.form == nil {
		return nil
	}
	_, cmd := f.form.Update(msg)
	if f.form.State != huh.StateCompleted {
		return cmd
	}
	switch f.step {
	case stTarget:
		return tea.Batch(cmd, m.goStep(stAuth))
	case stAuth:
		return tea.Batch(cmd, m.goStep(stAlerts))
	case stAlerts:
		return tea.Batch(cmd, m.goStep(stReview))
	case stDone:
		return tea.Batch(cmd, m.finishSetup(true))
	}
	return cmd
}

// typing reports whether the focused field takes text.
func (f *setupFlow) typing() bool {
	if f == nil || f.form == nil {
		return false
	}
	_, ok := f.form.GetFocusedField().(*huh.Input)
	return ok
}

func (f *setupFlow) focusedConfirm() *huh.Confirm {
	if f.form == nil {
		return nil
	}
	c, _ := f.form.GetFocusedField().(*huh.Confirm)
	return c
}

func (m *app) finishSetup(connected bool) tea.Cmd {
	f := m.setup
	m.screen = scrServices
	switch {
	case !connected:
		return m.say("info", "SKIPPED", "Nothing was saved "+m.t.G.Sep+" run "+m.t.AppName()+" connect any time")
	case f.makeDefault:
		return m.say("ok", "CONNECTED", f.cluster+" is your default cluster "+m.t.G.Sep+" 38 services")
	default:
		return m.say("ok", "CONNECTED", f.cluster+" "+m.t.G.Sep+" pass --cluster to use it")
	}
}

// setupActions is the action table for the wizard: one per step, and per
// focused field inside a form step. Keys reach Huh only through it, so the
// footer is the whole truth about what a key does.
func (m *app) setupActions(help action) []action {
	f := m.setup
	fwd := func(keys []string, k, desc string, show bool) action {
		return action{bind(keys, k, desc), show, func(m *app, msg tea.KeyPressMsg) tea.Cmd { return m.toForm(msg) }}
	}
	esc := func(desc string, do func(m *app) tea.Cmd) action {
		return action{bind([]string{"esc"}, "esc", desc), true, func(m *app, _ tea.KeyPressMsg) tea.Cmd { return do(m) }}
	}
	back := func(to setupStep) action {
		return esc("back", func(m *app) tea.Cmd { return m.goStep(to) })
	}

	switch f.step {
	case stReview:
		enter := strings.ToLower(f.acts.Selected())
		return []action{
			{bind([]string{"enter"}, "enter", enter), true, func(m *app, _ tea.KeyPressMsg) tea.Cmd { return m.reviewChoose() }},
			{bind([]string{"left", "right", "h", "l", "tab", "shift+tab"}, m.t.G.LeftRight, "switch"), true, func(m *app, msg tea.KeyPressMsg) tea.Cmd {
				if s := msg.String(); s == "left" || s == "h" || s == "shift+tab" {
					m.setup.acts.Move(-1)
				} else {
					m.setup.acts.Move(1)
				}
				return nil
			}},
			{bind([]string{"e"}, "e", "edit"), false, func(m *app, _ tea.KeyPressMsg) tea.Cmd { return m.goStep(stTarget) }},
			back(stAlerts), help,
		}
	case stSignin:
		if f.failed {
			return []action{
				{bind([]string{"enter"}, "enter", strings.ToLower(f.acts.Selected())), true, func(m *app, _ tea.KeyPressMsg) tea.Cmd {
					if m.setup.acts.Sel == 0 {
						return m.startSignin()
					}
					return m.goStep(stReview)
				}},
				{bind([]string{"left", "right", "h", "l", "tab", "shift+tab"}, m.t.G.LeftRight, "switch"), true, func(m *app, msg tea.KeyPressMsg) tea.Cmd {
					if s := msg.String(); s == "left" || s == "h" || s == "shift+tab" {
						m.setup.acts.Move(-1)
					} else {
						m.setup.acts.Move(1)
					}
					return nil
				}},
				{bind([]string{"r"}, "r", "retry"), false, func(m *app, _ tea.KeyPressMsg) tea.Cmd { return m.startSignin() }},
				back(stReview), help,
			}
		}
		acts := []action{esc("cancel sign-in", func(m *app) tea.Cmd {
			m.setup.gen++ // the pending result is now stale
			return tea.Batch(m.say("info", "CANCELED", "Sign-in canceled "+m.t.G.Sep+" nothing was sent to "+m.setup.cluster), m.goStep(stReview))
		})}
		if f.auth != "token" {
			acts = append(acts,
				action{bind([]string{"c"}, "c", "copy code"), true, func(m *app, _ tea.KeyPressMsg) tea.Cmd {
					return tea.Batch(m.say("ok", "COPIED", deviceCode), tea.SetClipboard(deviceCode))
				}},
				action{bind([]string{"o"}, "o", "open page"), true, func(m *app, _ tea.KeyPressMsg) tea.Cmd {
					return m.say("info", "OPENED", m.signinURL())
				}})
		}
		return append(acts, help)
	}

	// a form step: the keys depend on the focused field
	var escAct action
	switch f.step {
	case stTarget:
		escAct = esc("quit setup", func(m *app) tea.Cmd {
			if m.setup.namespace != "" && !m.setup.escArmed {
				m.setup.escArmed = true // never drop typed input on one key
				return m.say("warning", "ESC AGAIN", "to discard what you typed and leave setup")
			}
			return m.finishSetup(false)
		})
	case stDone:
		escAct = esc("skip", func(m *app) tea.Cmd { m.setup.makeDefault = false; return m.finishSetup(true) })
	default:
		escAct = back(f.step - 1)
	}
	prev := fwd([]string{"shift+tab"}, "shift+tab", "previous field", false)
	switch {
	case f.typing():
		return []action{
			fwd([]string{"enter"}, "enter", "next", true),
			fwd([]string{"tab"}, "tab", "next field", false), prev,
			fwd([]string{"backspace"}, "backspace", "delete", false),
			fwd([]string{"left", "right", "home", "end"}, m.t.G.LeftRight, "move", false),
			escAct, help,
		}
	case f.focusedConfirm() != nil:
		c := f.focusedConfirm()
		yes, _ := c.GetValue().(bool)
		enter := "confirm"
		if f.step == stAlerts {
			enter = map[bool]string{true: "turn on", false: "not now"}[yes]
		} else {
			enter = map[bool]string{true: "yes", false: "no"}[yes]
		}
		return []action{
			fwd([]string{"enter"}, "enter", enter, true),
			fwd([]string{"left", "right", "h", "l"}, m.t.G.LeftRight, "switch", true),
			fwd([]string{"y"}, "y", "yes", false),
			fwd([]string{"n"}, "n", "no", false),
			prev, escAct, help,
		}
	}
	return []action{
		fwd([]string{"enter"}, "enter", "next", true),
		fwd([]string{"up", "k", "down", "j"}, m.t.G.UpDown, "choose", true),
		fwd([]string{"tab"}, "tab", "next field", false), prev,
		escAct, help,
	}
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

func (m *app) signinURL() string { return "https://login." + m.t.AppName() + ".example/device" }

// summary is what the flow will do, telling decisions from defaults: a value
// is a decision once the user has moved past the field that sets it.
func (m *app) summary() []ui.SummaryRow {
	f := m.setup
	past := func(s setupStep) bool { return f.reached > s }
	clusterSet := past(stTarget) || (f.step == stTarget && f.typing())
	ns := f.namespace
	if ns == "" {
		ns = "default"
	}
	alerts := "off"
	if f.alerts {
		alerts = "on for failed deploys"
	}
	auth := f.auth
	for _, c := range authChoices {
		if c.Value == f.auth {
			auth = c.Key
		}
	}
	return []ui.SummaryRow{
		{Label: "Cluster", Value: f.cluster, Set: clusterSet},
		{Label: "Namespace", Value: ns, Set: past(stTarget) && f.namespace != ""},
		{Label: "Sign-in", Value: auth, Set: past(stAuth)},
		{Label: "Alerts", Value: alerts, Set: past(stAlerts)},
	}
}

// ---------------------------------------------------------------- render

// setupView lays the wizard out: the form at a measure, the summary in the
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
	switch f.step {
	case stReview:
		rows = append(rows, m.reviewRows(w)...)
	case stSignin:
		rows = append(rows, m.signinRows(w)...)
	case stDone:
		rows = append(rows, ui.Tag(t, "ok", "CONNECTED")+" "+t.S.Base.Render(f.cluster)+
			t.S.Muted.Render(ui.Trunc(t, " as mara@"+t.AppName()+".example "+t.G.Sep+" 38 services", max(0, w-lipgloss.Width(f.cluster)-12))), "")
	}
	c.Draw(strings.Join(rows, "\n"), main)
	if f.form != nil {
		top := main.Min.Y + len(rows)
		fr := uv.Rect(main.Min.X, top, w, max(0, main.Max.Y-top))
		// every draw, not only on a resize: Huh leaves room for a group's error rows
		// when WithHeight runs, so a height set before the error drew it off the bottom.
		// One row short of the rectangle: Huh v2.0.3 draws the blank row above the
		// errors past the height it was given.
		f.form.WithWidth(fr.Dx()).WithHeight(fr.Dy() - 1)
		c.Draw(f.form.View(), fr)
	}

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
		return "Sign-in", f.cluster, ""
	case stAlerts:
		return "Alerts", f.cluster, ""
	case stReview:
		return "Review", "nothing saved yet", ""
	case stSignin:
		return "Sign in to " + f.cluster, "", ""
	}
	return "Connected", "", ""
}

func (m *app) reviewRows(w int) []string {
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
	return append(rows, "", f.acts.View(t, true))
}

// nextRows is the review's side panel: what Continue will do.
func (m *app) nextRows(w int) []string {
	t, f := m.t, m.setup
	how := map[string]string{"browser": "open SSO in your browser", "device": "show a code to approve elsewhere", "token": "check your access token"}[f.auth]
	rows := []string{ui.Section(t, "What happens next", w, ""), ""}
	items := []string{
		t.S.Soft.Render(how),
		t.S.Soft.Render("save the context ") + t.S.Subtle.Render("~/.config/"+t.AppName()),
		t.S.Soft.Render("watch 38 services ") + t.S.Subtle.Render("on "+f.cluster),
	}
	for i, it := range items { // cut with the ellipsis, never clipped by the panel edge
		items[i] = ui.Trunc(t, it, max(1, w-lipgloss.Width(t.G.TreeMid)-1))
	}
	return append(rows, ui.Tree(t, items)...)
}

func (m *app) signinRows(w int) []string {
	t, f := m.t, m.setup
	secs := (m.frame - f.since) * int(ui.FrameInterval.Milliseconds()) / 1000
	if f.failed {
		return []string{
			ui.Alert(t, "error", "TIMED OUT", "No approval within 5 min; the code expired.", w), "",
			t.S.Soft.Render(ui.Trunc(t, "Nothing was saved on "+f.cluster+". Retry for a new code,", w)),
			t.S.Soft.Render(ui.Trunc(t, "or go back and choose another way to sign in.", w)), "",
			f.acts.View(t, true),
		}
	}
	wait := map[string]string{"browser": "Waiting for you to approve in the browser", "device": "Waiting for approval", "token": "Checking your token"}[f.auth]
	rows := []string{ui.Spinner(t, m.frame/2) + " " + t.S.Soft.Render(ui.Trunc(t, wait, w-8)) + t.S.Subtle.Render("  "+itoa(secs)+"s"), ""}
	if f.auth == "token" {
		return rows
	}
	lead := "If the browser didn't open, go to"
	if f.auth == "device" {
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

// runAccessibleSetup is the same flow for ACCESSIBLE=1: Huh's plain prompts
// (the same fields, built by the same buildForm), then status as new lines,
// never repainted: no spinner, no timer, no alt screen.
func runAccessibleSetup(t *theme.Theme, in io.Reader, out io.Writer, wait time.Duration) error {
	m := newApp(theme.File{}, t.Dark, t.ASCII)
	m.t, m.setup = t, newSetup()
	for _, s := range []setupStep{stTarget, stAuth, stAlerts} {
		if err := m.buildForm(s).WithInput(in).WithOutput(out).Run(); err != nil {
			return err
		}
	}
	f := m.setup
	fmt.Fprintf(out, "Signing in to %s.\n", f.cluster)
	if f.auth != "token" {
		fmt.Fprintf(out, "Go to %s and enter the code %s.\n", m.signinURL(), deviceCode)
	}
	fmt.Fprintln(out, "Waiting for approval.")
	time.Sleep(wait)
	fmt.Fprintf(out, "Signed in to %s as mara@%s.example. 38 services.\n", f.cluster, t.AppName())
	if err := m.buildForm(stDone).WithInput(in).WithOutput(out).Run(); err != nil {
		return err
	}
	if f.makeDefault {
		fmt.Fprintf(out, "Saved. %s is your default cluster.\n", f.cluster)
	} else {
		fmt.Fprintf(out, "Saved. Pass --cluster %s to use it.\n", f.cluster)
	}
	return nil
}
