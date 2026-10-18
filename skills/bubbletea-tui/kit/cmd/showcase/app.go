package main

import (
	"slices"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	uv "github.com/charmbracelet/ultraviolet"

	"tuikit/theme"
	"tuikit/ui"
)

// One root model. Regions are plain structs drawn into rectangles by View;
// there are no nested tea.Models to keep in sync (Crush's architecture).
//
// Keys live in ONE place per state: actions() returns the bindings for the
// current state, in help order, each with the function that handles it. The
// footer renders that list and key() dispatches on it, so help cannot
// advertise a key that does nothing (and TestHelpIsHandled proves it).

type screen int

const (
	scrServices screen = iota
	scrDeploys
	scrIncidents
	scrAssistant
	scrDeploy // the deploy form (from services)
	scrLadder // the status legend: every mark in its role (from the palette)
	scrSplash
	scrSetup // the first-run wizard (setup.go)
)

var tabNames = []string{"Services", "Deploys", "Incidents", "Assistant"}

type overlay int

const (
	ovNone overlay = iota
	ovPalette
	ovConfirm
	ovHelp
)

type app struct {
	file    theme.File
	t       *theme.Theme
	ascii   bool
	profile colorprofile.Profile // the terminal's colour profile (tea.ColorProfileMsg)
	keys    bool                 // the terminal disambiguates keys (tea.KeyboardEnhancementsMsg)
	w, h    int

	screen screen
	ov     overlay
	svcs   []service // the fixtures, copied so actions can change them
	sel    int       // selected service
	frame  int       // the one animation clock
	tick   tickState
	// toasts is the toast row: each toast has its own length (ui.ToastTTL, UndoTTL,
	// GraceTTL) on the frame clock and may offer one key (u undo), which joins the
	// state's action table while it shows (toastAction); a grace window holds a send
	toasts ui.Toasts
	last   string // description of the last handled action (for tests)

	// palette
	query  string
	palSel int
	palOff int // the list's scroll offset (ui.Follow keeps palSel on screen)
	// confirm
	btn int // 0 = the action, 1 = cancel (the default)
	// help and confirm: a dialog's body scrolls when it doesn't fit (ui.Clip)
	helpOff  int
	confOff  int
	helpFrom overlay      // the overlay help was opened over: its keys are the ones help names
	ovRect   uv.Rectangle // where the last overlay was drawn (tests hold it above the footer)
	body     uv.Rectangle // the page's body, between the tabs and the footer, as last drawn
	footY    int          // the footer's first row, as last drawn
	// deploy form
	focus    int // 0 env, 1 version, 2 strategy, 3 checks, 4 buttons
	env      int
	version  string
	strategy int
	checks   [2]bool
	checkSel int
	formBtn  int
	// assistant
	input     string
	sent      []string
	busy      bool
	busySince int // frame the work started
	expand    bool
	scroll    int
	// incidents: declaring pages on-call through a grace window (paging while it runs)
	declared bool
	paging   bool
	pages    int // pages sent (pagedMsg): the grace window's send ran, once
	// alerts silenced per service (the palette's Silence alerts, undone by its toast)
	silenced map[string]bool
	// services: a filter narrows the list (typed while filtering, in force while set)
	filter    string
	filtering bool
	// services: the list refreshes itself unless paused (the product clock)
	paused  bool
	refresh refreshState
	// deploys log: newest first; depSel is a row, resolved to a service by name
	events []event
	depSel int
	feed   [][4]string // the activity feed (time, what, who, state): live, newest first
	// first-run setup (setup.go)
	setup    *setupFlow
	firstRun bool
}

// tickState is the one frame clock. gen kills stale tick chains: a key press can't
// start a second loop, and a loop started before a pause dies on arrival.
type tickState struct {
	gen     int
	running bool
}

type tickMsg struct{ gen int }

// refreshState is the product clock: the services list syncs every refreshRates[rate]
// while it is live. It differs from the frame clock in one rule: a stale refresh tick
// must not die with nothing in its place, or the list never syncs again. So every
// setting that touches it (pause, resume, a new rate) returns armRefresh(), which
// retires the pending tick and arms a fresh one (TestClockSurvivesSettingChange).
type refreshState struct {
	gen   int
	rate  int // index into refreshRates
	syncs int // syncs so far (for tests)
}

// refreshRate is one interval with its label as the palette and the toast print it:
// "1m", where time.Duration's String gives "1m0s".
type refreshRate struct {
	every time.Duration
	label string
}

var refreshRates = []refreshRate{{time.Second, "1s"}, {2 * time.Second, "2s"}, {5 * time.Second, "5s"}, {15 * time.Second, "15s"}, {time.Minute, "1m"}}

type refreshMsg struct{ gen int }
type connectedMsg struct{}
type workDoneMsg struct{ gen int }
type deployDoneMsg struct{ svc string }

// pagedMsg is what a page's command delivers once on-call is paged: the showcase's page
// is local (page), a real one calls the paging API, and either way the grace window's
// send returns it (hold), so a dropped command is a page that never went.
type pagedMsg struct{ incident string }

func page(incident string) tea.Cmd { return func() tea.Msg { return pagedMsg{incident} } }

func newApp(f theme.File, dark, ascii bool) *app {
	m := &app{file: f, t: theme.Resolve(f, dark, ascii), ascii: ascii, profile: colorprofile.TrueColor, w: 80, h: 24}
	m.svcs = slices.Clone(services)
	m.events = slices.Clone(initialEvents)
	m.feed = slices.Clone(activity)
	m.version, m.checks = "v2.4.1", [2]bool{true, false}
	m.btn = 1
	m.silenced = map[string]bool{}
	m.refresh.rate = 2 // every 5 s
	m.setup = newSetup()
	return m
}

func (m *app) Init() tea.Cmd {
	m.screen = scrSplash
	connect := tea.Tick(900*time.Millisecond, func(time.Time) tea.Msg { return connectedMsg{} })
	// ask the terminal what its background is; the reply re-resolves the theme
	// the fixture's in-flight deploy finishes like any other: by a message
	done := tea.Tick(6*time.Second, func(time.Time) tea.Msg { return deployDoneMsg{"search-indexer"} })
	return tea.Batch(tea.RequestBackgroundColor, connect, done, m.startClock(), m.armRefresh())
}

// animating reports whether anything visible moves. When nothing does, the
// clock stops: an idle screen costs zero bytes.
func (m *app) animating() bool {
	if _, toast := m.toasts.Current(); m.screen == scrSplash || m.busy || toast {
		return true
	}
	if m.screen == scrSetup && m.setup.step == stSignin && !m.setup.failed {
		return true // the spinner and the elapsed time
	}
	// a deploying row moves only where Busy does: a theme whose spinner is wider than
	// one cell draws the still partial mark, and re-rendering it would be the same frame
	if m.ov == ovNone && (m.screen == scrServices || m.screen == scrDeploys) && ui.BusyAnimates(m.t) {
		return m.hasDeploying()
	}
	return false
}

// hasDeploying reports whether a service is mid-deploy (a busy row in the list).
func (m *app) hasDeploying() bool {
	for _, s := range m.svcs {
		if s.State == "deploying" {
			return true
		}
	}
	return false
}

// step is one frame of the clock, what a tick does before it re-arms: the frame moves,
// and the toast row counts down, returning a grace window's send when it ends. Its clock
// waits whenever the toast's key isn't offered (toastAction: under a dialog, while an
// input takes letters, below the floor, or where the state binds the key), so a window
// never sends, and an undo never lapses, out of reach.
func (m *app) step() tea.Cmd {
	m.frame++
	return m.toasts.Tick(m.toastAction() == nil)
}

// startClock arms the clock if something animates and it isn't already running.
func (m *app) startClock() tea.Cmd {
	if m.tick.running || !m.animating() {
		return nil
	}
	m.tick.running = true
	m.tick.gen++
	gen := m.tick.gen
	return tea.Tick(ui.FrameInterval, func(time.Time) tea.Msg { return tickMsg{gen} })
}

// armRefresh retires the pending refresh tick and arms one at the current rate, or
// none while paused. Every setting change returns it: a gen bump alone would orphan
// the pending tick and stop the clock for good.
// This sync is in place; a real poll is async (tick → fetch cmd → result msg), and its
// result handler re-arms only if the result's gen is current, or a setting change
// mid-fetch, which armed its own tick, starts a second chain. A tick that finds a
// fetch in flight re-arms instead of skipping, so a stale result can't strand the clock.
func (m *app) armRefresh() tea.Cmd {
	m.refresh.gen++
	if m.paused {
		return nil
	}
	gen := m.refresh.gen
	return tea.Tick(refreshRates[m.refresh.rate].every, func(time.Time) tea.Msg { return refreshMsg{gen} })
}

// sync is one refresh of the live list: each sparkline takes its next sample.
func (m *app) sync() {
	m.refresh.syncs++
	for i := range m.svcs {
		if p := m.svcs[i].P99; len(p) > 1 {
			m.svcs[i].P99 = append(p[1:], p[0])
		}
	}
}

func (m *app) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
	case tea.ColorProfileMsg:
		// 16 colours or NO_COLOR can merge a background step into the page: the
		// theme adds edges where that happens, so it must know the profile
		m.profile = msg.Profile
		m.resolve(m.t.Dark)
		if m.screen == scrSetup && m.setup.form != nil {
			return m, m.goStep(m.setup.step)
		}
	case tea.KeyboardEnhancementsMsg:
		// shift+enter is told apart from enter: help draws it rather than its fallback
		m.keys = msg.SupportsKeyDisambiguation()
		m.resolve(m.t.Dark)
	case tea.BackgroundColorMsg:
		m.resolve(msg.IsDark())
		if m.screen == scrSetup && m.setup.form != nil {
			// Huh keeps the first theme a field is given: rebuild the step
			return m, m.goStep(m.setup.step)
		}
	case connectedMsg:
		var cmd tea.Cmd
		if m.screen == scrSplash {
			m.screen = scrServices
			if m.firstRun {
				cmd = m.openSetup()
			}
		}
		return m, tea.Batch(cmd, m.startClock())
	case signedInMsg:
		if f := m.setup; m.screen == scrSetup && f.step == stSignin && !f.failed && msg.gen == f.gen {
			return m, tea.Batch(m.goStep(stDone), m.startClock())
		}
	case signinTimeoutMsg:
		if f := m.setup; m.screen == scrSetup && f.step == stSignin && !f.failed && msg.gen == f.gen {
			f.failed, f.acts = true, ui.NewActions(0, "Retry", "Back")
		}
	case tickMsg:
		if msg.gen != m.tick.gen {
			return m, nil // a stale chain dies here
		}
		sent := m.step()
		if !m.animating() {
			m.tick.running = false
			return m, sent
		}
		gen := m.tick.gen
		return m, tea.Batch(sent, tea.Tick(ui.FrameInterval, func(time.Time) tea.Msg { return tickMsg{gen} }))
	case refreshMsg:
		if msg.gen != m.refresh.gen {
			return m, nil // retired by a setting change, which armed the new tick
		}
		m.sync()
		return m, m.armRefresh()
	case pagedMsg:
		m.pages++
		m.feed = append([][4]string{{"now", "paged payments on-call for " + msg.incident, "@you", "warning"}}, m.feed...)
	case deployDoneMsg:
		return m, tea.Batch(m.finishDeploy(msg.svc), m.startClock())
	case workDoneMsg:
		if m.busy && msg.gen == len(m.sent) {
			m.busy = false
			m.sent = append(m.sent, "\x00done")
		}
	case tea.KeyPressMsg:
		cmd := m.key(msg)
		return m, tea.Batch(cmd, m.startClock())
	default:
		// everything else a Huh form asked for (its next-field messages,
		// the cursor blink) goes back to the form
		if m.screen == scrSetup {
			return m, tea.Batch(m.toForm(msg), m.startClock())
		}
	}
	return m, nil
}

// action is one key binding and what it does in the current state.
type action struct {
	b    key.Binding
	show bool // in the footer (all actions appear in the ? overlay)
	do   func(m *app, msg tea.KeyPressMsg) tea.Cmd
}

func bind(keys []string, k, desc string) key.Binding {
	return key.NewBinding(key.WithKeys(keys...), key.WithHelp(k, desc))
}

func (m *app) key(msg tea.KeyPressMsg) tea.Cmd {
	defer m.keepSelShown() // whatever the key did (a tab back, a typed letter), the selection is a shown row
	if msg.String() == "ctrl+c" {
		return tea.Sequence(m.toasts.Flush(), tea.Quit) // always, from everywhere; a held send goes first
	}
	// a focused input takes every printable key first: no letter, colon or
	// question mark can fire a command while the user is typing
	if m.screen == scrSetup && m.ov == ovNone && m.setup.typing() && msg.Text != "" && msg.Mod&^tea.ModShift == 0 {
		m.last, m.setup.escArmed = "type", false
		return m.toForm(msg) // the Huh input does the editing
	}
	if p := m.textTarget(); p != nil && msg.Text != "" && msg.Mod == 0 {
		*p += msg.Text
		m.last = "type"
		if m.ov == ovPalette {
			m.palSel = 0
		}
		return nil
	}
	for _, a := range m.actions() {
		if key.Matches(msg, a.b) {
			m.last = a.b.Help().Desc
			return a.do(m, msg)
		}
	}
	return nil
}

// textTarget is the string the focused input edits, or nil.
func (m *app) textTarget() *string {
	switch {
	case m.ov == ovPalette:
		return &m.query
	case m.ov != ovNone:
		return nil
	case m.screen == scrAssistant && !m.busy:
		return &m.input
	case m.screen == scrDeploy && m.focus == 1:
		return &m.version
	case m.screen == scrServices && m.filtering:
		return &m.filter
	}
	return nil
}

// shown are the indexes of the services the filter leaves, in order: all of them with
// no filter. The selection stays an index into m.svcs (identity), never into this.
func (m *app) shown() []int {
	var out []int
	q := strings.ToLower(m.filter)
	for i, s := range m.svcs {
		if strings.Contains(strings.ToLower(s.Name), q) {
			out = append(out, i)
		}
	}
	return out
}

// keepSelShown moves the selection on Services to the first row the filter leaves when
// the filter hides it (a typed letter, a tab back from Deploys, which selects a deploy's
// service), so the row the actions target is always one on screen. key runs it after
// every key. With no row left, noRow says so and nothing names or acts on the selection.
func (m *app) keepSelShown() {
	if v := m.shown(); m.screen == scrServices && len(v) > 0 && !slices.Contains(v, m.sel) {
		m.sel = v[0]
	}
}

// noRow reports whether Services shows no row (a filter that leaves none): then there is
// no selection, so the header, the side pane and the window title name no service, and
// no key (d, r, the palette's commands) acts on the one the filter hides.
func (m *app) noRow() bool {
	return m.screen == scrServices && len(m.shown()) == 0
}

func backspace(m *app, _ tea.KeyPressMsg) tea.Cmd {
	if p := m.textTarget(); p != nil {
		if r := []rune(*p); len(r) > 0 {
			*p = string(r[:len(r)-1])
		}
	}
	m.keepSelShown()
	return nil
}

// say raises a plain toast: what happened, for ui.ToastTTL. Raising runs nothing
// (ui.Toasts.Raise), so its command is nil; callers return it beside their own anyway,
// so a toast that one day starts work can't be dropped on the way out.
func (m *app) say(kind, label, msg string) tea.Cmd {
	m.toasts.Raise(ui.Note{Kind: kind, Label: label, Msg: msg})
	return nil
}

// sayUndo raises an undo toast, rung 1 of the confirm ladder: what happened and the key
// that takes it back (b, bound while the toast shows), for ui.UndoTTL, long enough to
// reach the key.
func (m *app) sayUndo(kind, label, msg string, b key.Binding, undo func(m *app) tea.Cmd) tea.Cmd {
	m.toasts.Raise(ui.Note{Kind: kind, Label: label, Msg: msg, For: ui.UndoTTL, Key: b,
		Act: func() tea.Cmd { return undo(m) }})
	return nil
}

// hold raises a grace window, rung 2's default for a send that can't be recalled: msg
// says what will happen ("Paging payments on-call"), the row counts down to it ("in 5s")
// and offers u undo for ui.GraceTTL. send runs once, when the window ends or on quit,
// and returns the command that sends; undo runs instead of it. No other toast cuts the
// window short: one raised meanwhile waits (ui.Toasts). It reports whether the window
// opened: one holds at a time, so offer the key that starts one only while
// m.toasts.Holding() is false (TestGraceWindowsHold), and start nothing on false.
func (m *app) hold(kind, label, msg string, send, undo func(m *app) tea.Cmd) bool {
	return m.toasts.Raise(ui.Note{Kind: kind, Label: label, Msg: msg, For: ui.GraceTTL,
		Key:  bind([]string{"u"}, "u", "undo"),
		Act:  func() tea.Cmd { return undo(m) },
		Send: func() tea.Cmd { return send(m) }})
}

// toastAction is the key the toast offers in the current state, or nil, and the row
// names a key only when this returns one. None under a dialog (the dialog's keys hold
// the footer row, the toast isn't drawn, and its clock waits: ui.Toasts.Tick), none
// while an input takes printable keys (u is a letter typed), and none where the state
// binds the key itself: the state's own key wins a clash, and the toast then offers
// nothing rather than a key that does something else.
func (m *app) toastAction() *action {
	n, ok := m.toasts.Current()
	if !ok || n.Key.Help().Key == "" || n.Act == nil || m.ov != ovNone || m.w < minW || m.h < minH {
		return nil
	}
	if m.textTarget() != nil || (m.screen == scrSetup && m.setup.typing()) {
		return nil
	}
	for _, a := range m.stateActions() {
		for _, k := range n.Key.Keys() {
			if slices.Contains(a.b.Keys(), k) {
				return nil
			}
		}
	}
	return &action{n.Key, true, func(m *app, _ tea.KeyPressMsg) tea.Cmd { return m.toasts.Act() }}
}

// actions returns the current state's bindings, most important first, and the toast's
// key while it offers one (toastAction: never a key the state binds).
func (m *app) actions() []action {
	if a := m.toastAction(); a != nil {
		return append([]action{*a}, m.stateActions()...)
	}
	return m.stateActions()
}

// stateActions are the current state's own bindings, the toast's aside.
func (m *app) stateActions() []action {
	quit := action{bind([]string{"q"}, "q", "quit"), true, func(m *app, _ tea.KeyPressMsg) tea.Cmd { return tea.Sequence(m.toasts.Flush(), tea.Quit) }}
	// with a text input focused, help and the palette move off printable keys
	typing := m.textTarget() != nil || (m.screen == scrSetup && m.setup.typing())
	helpKeys, paletteKeys := []string{"?"}, []string{"ctrl+p", ":"}
	if typing {
		helpKeys, paletteKeys = []string{"ctrl+g"}, []string{"ctrl+p"}
	}
	// help names the keys of the state it was opened from, an overlay's when one is up,
	// and closing it goes back there
	help := action{bind(helpKeys, helpKeys[0], "keys"), true, func(m *app, _ tea.KeyPressMsg) tea.Cmd {
		m.helpFrom, m.ov = m.ov, ovHelp
		return nil
	}}
	palette := action{bind(paletteKeys, "ctrl+p", "commands"), true, func(m *app, _ tea.KeyPressMsg) tea.Cmd {
		m.ov, m.query, m.palSel = ovPalette, "", 0
		return nil
	}}
	tabs := action{bind([]string{"tab"}, "tab", "next view"), true, func(m *app, _ tea.KeyPressMsg) tea.Cmd {
		m.screen = (m.screen + 1) % screen(len(tabNames))
		return nil
	}}
	tabsBack := action{bind([]string{"shift+tab"}, "shift+tab", "previous view"), false, func(m *app, _ tea.KeyPressMsg) tea.Cmd {
		m.screen = (m.screen + screen(len(tabNames)) - 1) % screen(len(tabNames))
		return nil
	}}
	bs := action{bind([]string{"backspace"}, "backspace", "delete"), false, backspace}
	esc := func(desc string, f func(m *app)) action {
		return action{bind([]string{"esc"}, "esc", desc), true, func(m *app, _ tea.KeyPressMsg) tea.Cmd { f(m); return nil }}
	}

	// toosmall: the screen draws no layout, so it takes only what ui.TooSmall shows
	if m.w < minW || m.h < minH {
		return []action{quit}
	}

	// a dialog's body scrolls only when it overflows, and the footer says so only then
	scroll := func(rows, shown int, off *int) action {
		return action{bind([]string{"up", "k", "down", "j"}, m.t.G.UpDown, "scroll"), rows > shown, func(m *app, msg tea.KeyPressMsg) tea.Cmd {
			if msg.String() == "up" || msg.String() == "k" {
				*off = max(0, *off-1)
			} else {
				*off = min(ui.ClipMax(rows, shown), *off+1)
			}
			return nil
		}}
	}

	switch m.ov {
	case ovHelp:
		closeHelp := func(m *app) { m.ov, m.helpFrom, m.helpOff = m.helpFrom, ovNone, 0 }
		return []action{esc("close", closeHelp),
			scroll(len(m.helpBody()), m.helpRows(), &m.helpOff),
			{bind([]string{"?", "ctrl+g"}, "?", "close"), false, func(m *app, _ tea.KeyPressMsg) tea.Cmd { closeHelp(m); return nil }}}
	case ovPalette:
		return []action{
			{bind([]string{"enter"}, "enter", "run"), true, func(m *app, _ tea.KeyPressMsg) tea.Cmd { return m.runCommand() }},
			{bind([]string{"up", "ctrl+k", "down", "ctrl+j"}, m.t.G.UpDown, "move"), true, func(m *app, msg tea.KeyPressMsg) tea.Cmd {
				if msg.String() == "up" || msg.String() == "ctrl+k" {
					m.palSel = max(0, m.palSel-1)
				} else {
					m.palSel = min(max(0, len(m.filtered())-1), m.palSel+1)
				}
				return nil
			}},
			esc("close", func(m *app) { m.ov = ovNone }),
			help, bs,
		}
	case ovConfirm:
		s := m.svcs[m.sel]
		_, confBody, _ := m.confirmBody()
		enterDesc := "cancel"
		if m.btn == 0 {
			enterDesc = "roll back"
		}
		return []action{
			{bind([]string{"enter"}, "enter", enterDesc), true, func(m *app, _ tea.KeyPressMsg) tea.Cmd {
				m.ov = ovNone
				if m.btn == 0 {
					return m.rollback()
				}
				return nil
			}},
			{bind([]string{"left", "right", "tab", "h", "l"}, m.t.G.LeftRight, "switch"), true, func(m *app, _ tea.KeyPressMsg) tea.Cmd { m.btn = 1 - m.btn; return nil }},
			{bind([]string{"r"}, "r", "roll back to "+s.Prev), false, func(m *app, _ tea.KeyPressMsg) tea.Cmd { m.ov = ovNone; return m.rollback() }},
			{bind([]string{"c"}, "c", "cancel"), false, func(m *app, _ tea.KeyPressMsg) tea.Cmd { m.ov = ovNone; return nil }},
			esc("cancel", func(m *app) { m.ov = ovNone; m.confOff = 0 }),
			// a fallback: the dialog takes the rows it needs down to the footer first
			scroll(len(confBody), m.confirmRows(), &m.confOff),
			help,
		}
	}

	switch m.screen {
	case scrSetup:
		return m.setupActions(help)
	case scrSplash:
		return []action{quit}
	case scrLadder:
		return []action{esc("back", func(m *app) { m.screen = scrServices }), palette, quit, help}
	case scrDeploy:
		enterDesc := "next field" // enter in a field never ships to prod
		if m.focus == 4 {
			enterDesc = []string{"deploy", "cancel"}[m.formBtn]
		}
		return []action{
			{bind([]string{"enter"}, "enter", enterDesc), true, func(m *app, _ tea.KeyPressMsg) tea.Cmd {
				switch {
				case m.focus < 4:
					m.focus++
					m.armDeploy()
				case m.formBtn == 1:
					m.screen = scrServices
				default:
					return m.deploy()
				}
				return nil
			}},
			// in a field, enter already says "next field": tab joins the footer only on the buttons
			{bind([]string{"tab", "down"}, "tab", "next field"), m.focus == 4, func(m *app, _ tea.KeyPressMsg) tea.Cmd { m.focus = (m.focus + 1) % 5; m.armDeploy(); return nil }},
			{bind([]string{"shift+tab", "up"}, "shift+tab", "previous"), false, func(m *app, _ tea.KeyPressMsg) tea.Cmd { m.focus = (m.focus + 4) % 5; m.armDeploy(); return nil }},
			{bind([]string{"left", "right"}, m.t.G.LeftRight, "choose"), m.focus != 1, func(m *app, msg tea.KeyPressMsg) tea.Cmd {
				d := 1
				if msg.String() == "left" {
					d = -1
				}
				switch m.focus {
				case 0:
					m.env = (m.env + 3 + d) % 3
				case 2:
					m.strategy = (m.strategy + 3 + d) % 3
				case 3:
					m.checkSel = 1 - m.checkSel
				case 4:
					m.formBtn = 1 - m.formBtn
				}
				return nil
			}},
			{bind([]string{"space"}, "space", "toggle"), m.focus == 3, func(m *app, _ tea.KeyPressMsg) tea.Cmd {
				if m.focus == 3 {
					m.checks[m.checkSel] = !m.checks[m.checkSel]
				} else if p := m.textTarget(); p != nil {
					*p += " "
				}
				return nil
			}},
			esc("back", func(m *app) { m.screen = scrServices }),
			help, bs,
		}
	case scrAssistant:
		acts := []action{}
		if m.busy {
			acts = append(acts, esc("stop", func(m *app) {
				m.busy = false
				m.sent = append(m.sent, "\x00canceled")
			}))
		} else {
			acts = append(acts, action{bind([]string{"enter"}, "enter", "send"), true, func(m *app, _ tea.KeyPressMsg) tea.Cmd {
				if m.input == "" {
					return nil
				}
				m.sent = append(m.sent, m.input)
				m.input, m.busy, m.busySince, m.scroll = "", true, m.frame, 0
				gen := len(m.sent)
				return tea.Tick(3*time.Second, func(time.Time) tea.Msg { return workDoneMsg{gen} })
			}})
		}
		return append(acts,
			// a toggle's help names what the key does now: "collapse" while expanded
			action{bind([]string{"ctrl+o"}, "ctrl+o", map[bool]string{false: "expand output", true: "collapse output"}[m.expand]), true,
				func(m *app, _ tea.KeyPressMsg) tea.Cmd { m.expand = !m.expand; return nil }},
			action{bind([]string{"pgup"}, "pgup", "scroll"), true, func(m *app, _ tea.KeyPressMsg) tea.Cmd { m.scroll += 5; return nil }},
			action{bind([]string{"pgdown"}, "pgdn", "scroll down"), false, func(m *app, _ tea.KeyPressMsg) tea.Cmd { m.scroll = max(0, m.scroll-5); return nil }},
			tabs, tabsBack, palette, help, bs,
		)
	case scrIncidents:
		acts := []action{}
		if !m.declared && !m.paging && !m.toasts.Holding() { // one window at a time
			// a page is a send to people one labelled key starts: the label is the judgement
			// (rung 2), so no confirm, but it waits out a grace window under an undo toast
			acts = append(acts, action{bind([]string{"n"}, "n", "declare incident"), true, func(m *app, _ tea.KeyPressMsg) tea.Cmd {
				m.paging = m.hold("warning", "HELD", "Paging payments on-call", func(m *app) tea.Cmd {
					m.paging, m.declared = false, true
					// the send is a command: dropped, nobody is paged whatever the row says
					return tea.Batch(m.say("ok", "SENT", "Declared INC-2041 "+m.t.G.Sep+" paged payments on-call"), page("INC-2041"))
				}, func(m *app) tea.Cmd {
					m.paging = false
					return m.say("info", "CANCELED", "Nobody was paged "+m.t.G.Sep+" INC-2041 not declared")
				})
				return nil
			}})
		}
		return append(acts,
			action{bind([]string{"h"}, "h", "history"), true, func(m *app, _ tea.KeyPressMsg) tea.Cmd {
				return m.say("info", "HISTORY", "No incidents in the last 30 days")
			}},
			tabs, tabsBack, palette, quit, help,
		)
	}
	// services and deploys: a list
	move := action{bind([]string{"up", "k", "down", "j"}, m.t.G.UpDown, "select"), true, func(m *app, msg tea.KeyPressMsg) tea.Cmd {
		up := msg.String() == "up" || msg.String() == "k"
		if m.screen == scrDeploys {
			if up {
				m.depSel = max(0, m.depSel-1)
			} else {
				m.depSel = min(len(m.events)-1, m.depSel+1)
			}
			m.sel = m.svcIndex(m.events[m.depSel].Svc) // a row selects its service
			return nil
		}
		// services: through the rows the filter leaves
		v := m.shown()
		if len(v) == 0 {
			return nil
		}
		i := max(0, slices.Index(v, m.sel))
		if up {
			i = max(0, i-1)
		} else {
			i = min(len(v)-1, i+1)
		}
		m.sel = v[i]
		return nil
	}}
	if m.screen == scrDeploys {
		return []action{move, palette, tabs, tabsBack, quit, help}
	}
	// the filter: / types it, esc clears it, offered only while one is typed or in force
	clearFilter := esc("clear filter", func(m *app) { m.filter, m.filtering = "", false })
	if m.filtering {
		// typing: letters are the query, so the list moves on the arrows alone
		move.b = bind([]string{"up", "down"}, m.t.G.UpDown, "select")
		acts := []action{
			{bind([]string{"enter"}, "enter", "keep filter"), true, func(m *app, _ tea.KeyPressMsg) tea.Cmd {
				m.filtering = false
				return nil
			}},
			clearFilter,
		}
		if !m.noRow() { // the palette's commands act on the selection, which the filter hides
			acts = append(acts, move, palette)
		}
		return append(acts, help, bs)
	}
	filterDesc := "filter"
	if m.filter != "" {
		filterDesc = "edit filter"
	}
	typeFilter := action{bind([]string{"/"}, "/", filterDesc), m.filter == "", func(m *app, _ tea.KeyPressMsg) tea.Cmd {
		m.filtering = true
		return nil
	}}
	if m.filter != "" && len(m.shown()) == 0 {
		// nothing left to act on: no key may reach the service the filter hides
		return []action{clearFilter, typeFilter, tabs, tabsBack, quit, help}
	}
	// a toggle's help names what the key does now (TestToggleHelpIsTruthful): "pause"
	// while the list is live, "resume" while it is paused. Pause changes the product
	// clock, so it returns the new arm cmd ("Refresh rate" in command() does too)
	pause := action{bind([]string{"p"}, "p", map[bool]string{false: "pause", true: "resume"}[m.paused]), true,
		func(m *app, _ tea.KeyPressMsg) tea.Cmd { m.paused = !m.paused; return m.armRefresh() }}
	acts := []action{move}
	if m.filter != "" {
		acts = append(acts, clearFilter)
	}
	acts = append(acts,
		action{bind([]string{"d"}, "d", "deploy"), true, func(m *app, _ tea.KeyPressMsg) tea.Cmd { m.openDeploy(); return nil }},
		action{bind([]string{"r"}, "r", "roll back"), true, func(m *app, _ tea.KeyPressMsg) tea.Cmd { return m.askRollback() }},
		palette, typeFilter, tabs, tabsBack, pause, quit, help,
	)
	// the palette's shortcuts work here too, so the hints it shows are true
	for _, c := range commands {
		if c.Key == "" || c.Key == "d" || c.Key == "r" {
			continue
		}
		c := c
		acts = append(acts, action{bind([]string{c.Key}, c.Key, strings.ToLower(c.Title)), false, func(m *app, _ tea.KeyPressMsg) tea.Cmd { return m.command(c.Title) }})
	}
	return acts
}

func (m *app) openDeploy() {
	m.screen, m.focus, m.formBtn = scrDeploy, 1, 0
}

// armDeploy runs as focus moves: arriving on the buttons, a prod deploy starts on Cancel,
// so the review step is the confirmation and enter alone never ships to prod; the
// other environments start on the act, the forward choice.
func (m *app) armDeploy() {
	if m.focus != 4 {
		return
	}
	m.formBtn = 0
	if m.env == 0 {
		m.formBtn = 1
	}
}

func (m *app) svcIndex(name string) int {
	for i, s := range m.svcs {
		if s.Name == name {
			return i
		}
	}
	return m.sel
}

// logEvent records a deploy or rollback everywhere it is shown: the deploy
// log, the activity feed and (via the feed) the detail pane's Recent list.
func (m *app) logEvent(e event, what string) {
	m.events = append([]event{e}, m.events...)
	m.depSel = 0
	m.feed = append([][4]string{{"now", what, e.By, e.State}}, m.feed...)
}

// deploy ships the form's version. Prod changes the fleet and finishes by a
// message; other environments only log, since this view shows prod.
func (m *app) deploy() tea.Cmd {
	s := &m.svcs[m.sel]
	env := []string{"prod", "staging", "dev"}[m.env]
	how := []string{"rolling", "canary 10%", "blue/green"}[m.strategy]
	m.screen = scrServices
	if env != "prod" {
		m.logEvent(event{"now", s.Name, env, m.version, "@you", "ok"}, "deployed "+s.Name+" "+m.version+" to "+env)
		return m.say("ok", "DEPLOYED", s.Name+" "+m.version+" to "+env)
	}
	s.Prev, s.State, s.Version = s.Version, "deploying", m.version
	m.logEvent(event{"now", s.Name, "prod", m.version, "@you", "busy"}, "deploying "+s.Name+" "+m.version)
	name := s.Name
	return tea.Batch(m.say("info", "DEPLOYING", s.Name+" "+m.version+" to prod "+m.t.G.Sep+" "+how),
		tea.Tick(4*time.Second, func(time.Time) tea.Msg { return deployDoneMsg{name} }))
}

func (m *app) finishDeploy(name string) tea.Cmd {
	i := m.svcIndex(name)
	if m.svcs[i].Name != name || m.svcs[i].State != "deploying" {
		return nil
	}
	s := &m.svcs[i]
	s.State, s.ErrRate, s.Deployed = "ok", "0.02%", "now"
	for k := range m.events {
		if m.events[k].Svc == name && m.events[k].State == "busy" {
			m.events[k].State = "ok"
		}
	}
	m.feed = append([][4]string{{"now", "deployed " + name + " " + s.Version + " " + m.t.G.Sep + " 3/3 pods healthy", "@you", "ok"}}, m.feed...)
	return m.say("ok", "DEPLOYED", name+" "+s.Version+" "+m.t.G.Sep+" 3/3 pods healthy")
}

// askRollback opens the confirm, or says why there is nothing to roll back to.
func (m *app) askRollback() tea.Cmd {
	if m.svcs[m.sel].Prev == "" {
		return m.say("warning", "NOTHING TO ROLL BACK", m.svcs[m.sel].Name+" has no earlier release recorded")
	}
	m.ov, m.btn = ovConfirm, 1
	return nil
}

func (m *app) rollback() tea.Cmd {
	s := &m.svcs[m.sel]
	if s.Prev == "" {
		return nil
	}
	from := s.Version
	s.Version, s.Prev, s.State, s.ErrRate, s.Deployed = s.Prev, "", "ok", "0.03%", "now"
	s.P99 = []float64{140, 132, 128, 125, 124, 123, 122, 121, 122, 121, 120, 121}
	m.logEvent(event{"now", s.Name, "prod", s.Version, "@you", "ok"}, "rolled back "+s.Name+" "+from+" "+m.t.G.Arrow+" "+s.Version)
	return m.say("ok", "DONE", "Rolled back "+s.Name+" from "+from+" to "+s.Version)
}

func (m *app) runCommand() tea.Cmd {
	hits := m.filtered()
	m.ov = ovNone
	if len(hits) == 0 {
		return nil
	}
	return m.command(hits[min(m.palSel, len(hits)-1)].c.Title)
}

// command runs a palette command by name; shortcuts and the palette share it.
func (m *app) command(title string) tea.Cmd {
	s := m.svcs[m.sel]
	switch title {
	case "Deploy service":
		m.openDeploy()
	case "Roll back":
		return m.askRollback()
	case "Tail logs":
		m.screen = scrAssistant
	case "Open dashboard":
		return m.say("info", "OPENED", "grafana.internal/d/"+s.Name)
	case "Silence alerts":
		// rung 1: reversible in one gesture, so no confirm; the toast offers the undo
		name := s.Name
		m.silenced[name] = true
		return m.sayUndo("ok", "DONE", "Silenced "+name+" alerts for 1h", bind([]string{"u"}, "u", "unsilence"),
			func(m *app) tea.Cmd { return m.unsilence(name) })
	case "Unsilence alerts":
		return m.unsilence(s.Name)
	case "Switch environment":
		return m.say("info", "ENV", "staging is read-only in this demo")
	case "Copy service URL":
		return tea.Batch(m.say("ok", "COPIED", "https://"+s.Name+".prod.internal"), tea.SetClipboard("https://"+s.Name+".prod.internal"))
	case "Refresh rate":
		m.refresh.rate = (m.refresh.rate + 1) % len(refreshRates)
		// a changed interval re-arms the product clock
		return tea.Batch(m.say("info", "REFRESH", "every "+refreshRates[m.refresh.rate].label), m.armRefresh())
	case "Toggle theme":
		m.resolve(!m.t.Dark)
	case "Status legend":
		m.screen = scrLadder
	}
	return nil
}

// unsilence turns a service's alerts back on.
func (m *app) unsilence(name string) tea.Cmd {
	delete(m.silenced, name)
	return m.say("ok", "DONE", "Alerts back on for "+name)
}

// resolve rebuilds the theme for a mode, keeping the ASCII choice and the profile.
func (m *app) resolve(dark bool) {
	m.t = theme.ResolveProfile(m.file, dark, m.ascii, m.profile)
	m.t.KeyDisambiguation = m.keys
}

func (m *app) footerBindings() []key.Binding {
	var b []key.Binding
	for _, a := range m.actions() {
		if a.show {
			b = append(b, a.b)
		}
	}
	return b
}

func (m *app) View() tea.View {
	c := m.render(m.w, m.h)
	v := tea.NewView(c.String())
	v.AltScreen = true
	v.WindowTitle = m.t.AppName() + " " + m.t.G.Sep + " " + m.svcs[m.sel].Name
	if m.noRow() {
		v.WindowTitle = m.t.AppName()
	}
	if m.screen == scrSetup {
		v.WindowTitle = m.t.AppName() + " " + m.t.G.Sep + " setup"
	}
	if m.t.X.PaintBg {
		v.BackgroundColor = m.t.P.Bg
		v.ForegroundColor = m.t.P.Fg
	}
	if m.busy {
		v.ProgressBar = tea.NewProgressBar(tea.ProgressBarIndeterminate, 0)
	}
	return v
}
