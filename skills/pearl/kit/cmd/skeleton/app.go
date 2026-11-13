package main

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	uv "github.com/charmbracelet/ultraviolet"

	"github.com/jrschumacher/pearl-tui/skills/pearl/kit/theme/bubbletea"
	"github.com/jrschumacher/pearl-tui/skills/pearl/kit/ui/bubbletea"
)

// screen is a page of your app; overlay a dialog drawn over one. Add yours here, a
// state for each in main.go, and a branch in actions and render.
type (
	screen  int
	overlay int
)

const (
	scrList screen = iota
	scrDetail
)

const (
	ovNone overlay = iota
	ovConfirm
	ovHelp
)

const btnRemove, btnCancel = 0, 1 // the confirm's buttons; Cancel, the safe one, has the focus on open

// app is the one root model. Selection is by index into items here; when your items
// can reorder or refresh under the user, select by identity (an ID) instead.
type app struct {
	file    theme.File
	t       *theme.Theme
	ascii   bool
	profile colorprofile.Profile
	w, h    int

	screen   screen
	ov       overlay
	helpFrom overlay // what help was opened over: esc returns there
	helpOff  int
	items    []item
	sel      int
	btn      int
	last     string // the last action's description, which the harness reads

	// where the last render put things, for the harness and for mouse hits
	body   uv.Rectangle
	footY  int
	ovRect uv.Rectangle
	toasts ui.Toasts // none raised yet; the showcase's app.go shows toasts and undo
}

func newApp(f theme.File, dark, ascii bool) *app {
	return &app{file: f, t: theme.Resolve(f, dark, ascii), ascii: ascii, profile: colorprofile.TrueColor,
		w: 80, h: 24, items: fixtures()}
}

func (m *app) resolve(dark bool) { m.t = theme.ResolveProfile(m.file, dark, m.ascii, m.profile) }

func (m *app) Init() tea.Cmd { return nil }

func (m *app) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
	case tea.ColorProfileMsg:
		m.profile = msg.Profile
		m.resolve(m.t.Dark)
	case tea.KeyPressMsg:
		return m, m.key(msg)
	}
	return m, nil
}

func (m *app) View() tea.View {
	v := tea.NewView(m.render(m.w, m.h).String())
	v.AltScreen = true
	return v
}

// action is one key binding and what it does in the current state. One table per
// state both dispatches keys and draws the footer and help, so every key shown works.
type action struct {
	b    key.Binding
	show bool // in the footer; help lists every action
	do   func(m *app, msg tea.KeyPressMsg) tea.Cmd
}

func bind(keys []string, k, desc string) key.Binding {
	return key.NewBinding(key.WithKeys(keys...), key.WithHelp(k, desc))
}

func (m *app) key(msg tea.KeyPressMsg) tea.Cmd {
	if msg.String() == "ctrl+c" {
		return tea.Quit // from everywhere
	}
	for _, a := range m.actions() {
		if key.Matches(msg, a.b) {
			m.last = a.b.Help().Desc
			return a.do(m, msg)
		}
	}
	return nil
}

// toastAction is the key the showing toast offers, or nil: this app's toasts offer none.
func (m *app) toastAction() *action { return nil }

// actions is the current state's table, most important key first.
func (m *app) actions() []action {
	quit := action{bind([]string{"q"}, "q", "quit"), true, func(*app, tea.KeyPressMsg) tea.Cmd { return tea.Quit }}
	help := action{bind([]string{"?"}, "?", "keys"), true, func(m *app, _ tea.KeyPressMsg) tea.Cmd {
		m.helpFrom, m.ov, m.helpOff = m.ov, ovHelp, 0
		return nil
	}}
	if m.w < minW || m.h < minH {
		return []action{quit}
	}
	switch m.ov {
	case ovHelp:
		closeHelp := func(m *app, _ tea.KeyPressMsg) tea.Cmd { m.ov, m.helpFrom = m.helpFrom, ovNone; return nil }
		return []action{
			{bind([]string{"up", "k", "down", "j"}, m.t.G.UpDown, "scroll"), len(m.helpBody()) > m.helpRows(), func(m *app, msg tea.KeyPressMsg) tea.Cmd {
				m.helpOff = max(0, min(ui.ClipMax(len(m.helpBody()), m.helpRows()), m.helpOff+step(msg)))
				return nil
			}},
			{bind([]string{"esc"}, "esc", "close"), true, closeHelp},
			{bind([]string{"?"}, "?", "close"), false, closeHelp},
		}
	case ovConfirm:
		return []action{
			{bind([]string{"enter"}, "enter", map[int]string{btnRemove: "remove", btnCancel: "cancel"}[m.btn]), true, func(m *app, _ tea.KeyPressMsg) tea.Cmd {
				if m.btn == btnRemove {
					m.items = append(m.items[:m.sel:m.sel], m.items[m.sel+1:]...)
					m.sel, m.screen = max(0, min(m.sel, len(m.items)-1)), scrList
				}
				m.ov = ovNone
				return nil
			}},
			{bind([]string{"left", "right", "tab"}, m.t.G.LeftRight, "switch"), true, func(m *app, _ tea.KeyPressMsg) tea.Cmd { m.btn = 1 - m.btn; return nil }},
			{bind([]string{"esc"}, "esc", "cancel"), true, func(m *app, _ tea.KeyPressMsg) tea.Cmd { m.ov = ovNone; return nil }},
			help,
		}
	}
	remove := action{bind([]string{"x"}, "x", "remove"), true, func(m *app, _ tea.KeyPressMsg) tea.Cmd { m.ov, m.btn = ovConfirm, btnCancel; return nil }}
	if len(m.items) == 0 {
		return []action{quit, help} // offer only what works on an empty list
	}
	if m.screen == scrDetail {
		back := action{bind([]string{"esc"}, "esc", "back"), true, func(m *app, _ tea.KeyPressMsg) tea.Cmd { m.screen = scrList; return nil }}
		return []action{back, remove, quit, help}
	}
	return []action{
		{bind([]string{"up", "k", "down", "j"}, m.t.G.UpDown, "select"), true, func(m *app, msg tea.KeyPressMsg) tea.Cmd {
			m.sel = max(0, min(len(m.items)-1, m.sel+step(msg)))
			return nil
		}},
		{bind([]string{"enter"}, "enter", "open"), true, func(m *app, _ tea.KeyPressMsg) tea.Cmd { m.screen = scrDetail; return nil }},
		remove, quit, help,
	}
}

// step is -1 for up and k, +1 for down and j.
func step(msg tea.KeyPressMsg) int {
	if s := msg.String(); s == "up" || s == "k" {
		return -1
	}
	return 1
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
