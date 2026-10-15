package main

import (
	"slices"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

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
	scrSplash
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
	file  theme.File
	t     *theme.Theme
	ascii bool
	w, h  int

	screen screen
	ov     overlay
	svcs   []service // the fixtures, copied so actions can change them
	sel    int       // selected service
	frame  int       // the one animation clock
	tick   tickState
	toast  *toast
	last   string // description of the last handled action (for tests)

	// palette
	query  string
	palSel int
	// confirm
	btn int // 0 = the action, 1 = cancel (the default)
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
	// incidents
	declared bool
}

type toast struct {
	kind, label, msg string
	until            int // frame
}

// tickState is the one clock. gen kills stale tick chains: a key press can't
// start a second loop, and a loop started before a pause dies on arrival.
type tickState struct {
	gen     int
	running bool
}

type tickMsg struct{ gen int }
type connectedMsg struct{}
type workDoneMsg struct{ gen int }

func newApp(f theme.File, dark, ascii bool) *app {
	m := &app{file: f, t: theme.Resolve(f, dark, ascii), ascii: ascii, w: 80, h: 24}
	m.svcs = slices.Clone(services)
	m.version, m.checks = "v2.4.1", [2]bool{true, false}
	m.btn = 1
	return m
}

func (m *app) Init() tea.Cmd {
	m.screen = scrSplash
	connect := tea.Tick(900*time.Millisecond, func(time.Time) tea.Msg { return connectedMsg{} })
	// ask the terminal what its background is; the reply re-resolves the theme
	return tea.Batch(tea.RequestBackgroundColor, connect, m.startClock())
}

// animating reports whether anything visible moves. When nothing does, the
// clock stops: an idle screen costs zero bytes.
func (m *app) animating() bool {
	if m.screen == scrSplash || m.busy || m.toast != nil {
		return true
	}
	if m.ov == ovNone && (m.screen == scrServices || m.screen == scrDeploys) {
		for _, s := range m.svcs {
			if s.State == "deploying" {
				return true
			}
		}
	}
	return false
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

func (m *app) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
	case tea.BackgroundColorMsg:
		m.t = theme.Resolve(m.file, msg.IsDark(), m.ascii)
	case connectedMsg:
		if m.screen == scrSplash {
			m.screen = scrServices
		}
		return m, m.startClock()
	case tickMsg:
		if msg.gen != m.tick.gen {
			return m, nil // a stale chain dies here
		}
		m.frame++
		if m.toast != nil && m.frame >= m.toast.until {
			m.toast = nil
		}
		if !m.animating() {
			m.tick.running = false
			return m, nil
		}
		gen := m.tick.gen
		return m, tea.Tick(ui.FrameInterval, func(time.Time) tea.Msg { return tickMsg{gen} })
	case workDoneMsg:
		if m.busy && msg.gen == len(m.sent) {
			m.busy = false
			m.sent = append(m.sent, "\x00done")
		}
	case tea.KeyPressMsg:
		cmd := m.key(msg)
		return m, tea.Batch(cmd, m.startClock())
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
	if msg.String() == "ctrl+c" {
		return tea.Quit // always, from everywhere
	}
	for _, a := range m.actions() {
		if key.Matches(msg, a.b) {
			m.last = a.b.Help().Desc
			return a.do(m, msg)
		}
	}
	// unmatched printable keys go to the focused text input, if there is one
	if p := m.textTarget(); p != nil && msg.Text != "" {
		*p += msg.Text
		m.last = "type"
		if m.ov == ovPalette {
			m.palSel = 0
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
	}
	return nil
}

func backspace(m *app, _ tea.KeyPressMsg) tea.Cmd {
	if p := m.textTarget(); p != nil {
		if r := []rune(*p); len(r) > 0 {
			*p = string(r[:len(r)-1])
		}
	}
	return nil
}

func (m *app) say(kind, label, msg string) {
	m.toast = &toast{kind, label, msg, m.frame + 80} // ~4 s at 20 fps
}

// actions returns the current state's bindings, most important first.
func (m *app) actions() []action {
	quit := action{bind([]string{"q"}, "q", "quit"), true, func(*app, tea.KeyPressMsg) tea.Cmd { return tea.Quit }}
	help := action{bind([]string{"?"}, "?", "keys"), true, func(m *app, _ tea.KeyPressMsg) tea.Cmd { m.ov = ovHelp; return nil }}
	palette := action{bind([]string{"ctrl+p", ":"}, "ctrl+p", "commands"), true, func(m *app, _ tea.KeyPressMsg) tea.Cmd {
		m.ov, m.query, m.palSel = ovPalette, "", 0
		return nil
	}}
	tabs := action{bind([]string{"tab"}, "tab", "next view"), true, func(m *app, _ tea.KeyPressMsg) tea.Cmd {
		m.screen = (m.screen + 1) % 4
		return nil
	}}
	tabsBack := action{bind([]string{"shift+tab"}, "shift+tab", "previous view"), false, func(m *app, _ tea.KeyPressMsg) tea.Cmd {
		m.screen = (m.screen + 3) % 4
		return nil
	}}
	bs := action{bind([]string{"backspace"}, "backspace", "delete"), false, backspace}
	esc := func(desc string, f func(m *app)) action {
		return action{bind([]string{"esc"}, "esc", desc), true, func(m *app, _ tea.KeyPressMsg) tea.Cmd { f(m); return nil }}
	}

	switch m.ov {
	case ovHelp:
		return []action{esc("close", func(m *app) { m.ov = ovNone }),
			{bind([]string{"?"}, "?", "close"), false, func(m *app, _ tea.KeyPressMsg) tea.Cmd { m.ov = ovNone; return nil }}}
	case ovPalette:
		return []action{
			{bind([]string{"enter"}, "enter", "run"), true, func(m *app, _ tea.KeyPressMsg) tea.Cmd { return m.runCommand() }},
			{bind([]string{"up", "ctrl+k"}, "↑", "up"), true, func(m *app, _ tea.KeyPressMsg) tea.Cmd { m.palSel = max(0, m.palSel-1); return nil }},
			{bind([]string{"down", "ctrl+j"}, "↓", "down"), true, func(m *app, _ tea.KeyPressMsg) tea.Cmd {
				m.palSel = min(max(0, len(m.filtered())-1), m.palSel+1)
				return nil
			}},
			esc("close", func(m *app) { m.ov = ovNone }),
			bs,
		}
	case ovConfirm:
		s := m.svcs[m.sel]
		enterDesc := "cancel"
		if m.btn == 0 {
			enterDesc = "roll back"
		}
		return []action{
			{bind([]string{"enter"}, "enter", enterDesc), true, func(m *app, _ tea.KeyPressMsg) tea.Cmd {
				if m.btn == 0 {
					m.rollback()
				}
				m.ov = ovNone
				return nil
			}},
			{bind([]string{"left", "right", "tab", "h", "l"}, "←→", "switch"), true, func(m *app, _ tea.KeyPressMsg) tea.Cmd { m.btn = 1 - m.btn; return nil }},
			{bind([]string{"r"}, "r", "roll back to "+s.Prev), false, func(m *app, _ tea.KeyPressMsg) tea.Cmd { m.rollback(); m.ov = ovNone; return nil }},
			{bind([]string{"c"}, "c", "cancel"), false, func(m *app, _ tea.KeyPressMsg) tea.Cmd { m.ov = ovNone; return nil }},
			esc("cancel", func(m *app) { m.ov = ovNone }),
		}
	}

	switch m.screen {
	case scrSplash:
		return []action{quit}
	case scrDeploy:
		return []action{
			{bind([]string{"enter"}, "enter", "deploy"), true, func(m *app, _ tea.KeyPressMsg) tea.Cmd {
				if m.focus == 4 && m.formBtn == 1 {
					m.screen = scrServices
					return nil
				}
				m.deploy()
				return nil
			}},
			{bind([]string{"tab", "down"}, "tab", "next field"), true, func(m *app, _ tea.KeyPressMsg) tea.Cmd { m.focus = (m.focus + 1) % 5; return nil }},
			{bind([]string{"shift+tab", "up"}, "shift+tab", "previous"), false, func(m *app, _ tea.KeyPressMsg) tea.Cmd { m.focus = (m.focus + 4) % 5; return nil }},
			{bind([]string{"left", "right"}, "←→", "choose"), m.focus != 1, func(m *app, msg tea.KeyPressMsg) tea.Cmd {
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
			bs,
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
			action{bind([]string{"ctrl+o"}, "ctrl+o", "expand output"), true, func(m *app, _ tea.KeyPressMsg) tea.Cmd { m.expand = !m.expand; return nil }},
			action{bind([]string{"pgup"}, "pgup", "scroll"), true, func(m *app, _ tea.KeyPressMsg) tea.Cmd { m.scroll += 5; return nil }},
			action{bind([]string{"pgdown"}, "pgdn", "scroll down"), false, func(m *app, _ tea.KeyPressMsg) tea.Cmd { m.scroll = max(0, m.scroll-5); return nil }},
			tabs, tabsBack, palette, bs,
		)
	case scrIncidents:
		return []action{
			{bind([]string{"n"}, "n", "declare incident"), true, func(m *app, _ tea.KeyPressMsg) tea.Cmd {
				m.declared = true
				m.say("warning", "DECLARED", "INC-2041 checkout-api 502s · paging payments on-call")
				return nil
			}},
			{bind([]string{"h"}, "h", "history"), true, func(m *app, _ tea.KeyPressMsg) tea.Cmd {
				m.say("info", "HISTORY", "No incidents in the last 30 days")
				return nil
			}},
			tabs, tabsBack, palette, help, quit,
		}
	}
	// services and deploys: a list
	move := action{bind([]string{"up", "k", "down", "j"}, "↑↓", "select"), true, func(m *app, msg tea.KeyPressMsg) tea.Cmd {
		switch msg.String() {
		case "up", "k":
			m.sel = max(0, m.sel-1)
		default:
			m.sel = min(len(m.svcs)-1, m.sel+1)
		}
		return nil
	}}
	if m.screen == scrDeploys {
		return []action{move, tabs, tabsBack, palette, help, quit}
	}
	acts := []action{
		move,
		{bind([]string{"d"}, "d", "deploy"), true, func(m *app, _ tea.KeyPressMsg) tea.Cmd { m.openDeploy(); return nil }},
		{bind([]string{"r"}, "r", "roll back"), true, func(m *app, _ tea.KeyPressMsg) tea.Cmd { m.ov, m.btn = ovConfirm, 1; return nil }},
		palette, tabs, tabsBack, help, quit,
	}
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

func (m *app) deploy() {
	s := &m.svcs[m.sel]
	s.State, s.Version = "deploying", m.version
	m.screen = scrServices
	m.say("info", "DEPLOYING", s.Name+" "+m.version+" · "+[]string{"rolling", "canary 10%", "blue/green"}[m.strategy])
}

func (m *app) rollback() {
	s := &m.svcs[m.sel]
	from := s.Version
	s.Version, s.State, s.ErrRate, s.Deployed = s.Prev, "ok", "0.03%", "now"
	s.P99 = []float64{140, 132, 128, 125, 124, 123, 122, 121, 122, 121, 120, 121}
	m.say("ok", "DONE", "Rolled back "+s.Name+" from "+from+" to "+s.Prev)
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
		m.ov, m.btn = ovConfirm, 1
	case "Tail logs":
		m.screen = scrAssistant
	case "Open dashboard":
		m.say("info", "OPENED", "grafana.internal/d/"+s.Name)
	case "Silence alerts":
		m.say("warning", "SILENCED", s.Name+" alerts for 1h")
	case "Switch environment":
		m.say("info", "ENV", "staging is read-only in this demo")
	case "Copy service URL":
		m.say("ok", "COPIED", "https://"+s.Name+".prod.internal")
		return tea.SetClipboard("https://" + s.Name + ".prod.internal")
	case "Toggle theme":
		m.t = theme.Resolve(m.file, !m.t.Dark, m.ascii)
	}
	return nil
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
	v.WindowTitle = m.t.AppName() + " · " + m.svcs[m.sel].Name
	if m.t.X.PaintBg {
		v.BackgroundColor = m.t.P.Bg
		v.ForegroundColor = m.t.P.Fg
	}
	if m.busy {
		v.ProgressBar = tea.NewProgressBar(tea.ProgressBarIndeterminate, 0)
	}
	return v
}
