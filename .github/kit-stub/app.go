package main

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	uv "github.com/charmbracelet/ultraviolet"

	"example.com/brandapp/internal/theme"
	"example.com/brandapp/internal/ui"
)

// A minimal app on the kit's shape: two pages (a list and an item's detail) and one
// overlay of its own (a confirm), with the keys dialog every app has.

type screen int

const (
	scrList screen = iota
	scrDetail
)

type overlay int

const (
	ovNone overlay = iota
	ovConfirm
	ovHelp
)

type app struct {
	file    theme.File
	t       *theme.Theme
	ascii   bool
	profile colorprofile.Profile
	w, h    int

	screen   screen
	ov       overlay
	items    []string
	sel      int
	btn      int
	last     string
	helpOff  int
	helpFrom overlay
	ovRect   uv.Rectangle
	body     uv.Rectangle
	footY    int
	toasts   ui.Toasts // this app raises none; the kept toast tests then pass on nothing
}

// toastAction is the key the showing toast offers, or nil: this app's toasts offer none.
func (m *app) toastAction() *action { return nil }

func newApp(f theme.File, dark, ascii bool) *app {
	return &app{file: f, t: theme.Resolve(f, dark, ascii), ascii: ascii, profile: colorprofile.TrueColor, w: 80, h: 24,
		items: []string{"alpha", "bravo", "charlie", "delta"}, btn: 1}
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

type action struct {
	b    key.Binding
	show bool
	do   func(m *app) tea.Cmd
}

func bind(keys []string, k, desc string) key.Binding {
	return key.NewBinding(key.WithKeys(keys...), key.WithHelp(k, desc))
}

func (m *app) key(msg tea.KeyPressMsg) tea.Cmd {
	if msg.String() == "ctrl+c" {
		return tea.Quit
	}
	for _, a := range m.actions() {
		if key.Matches(msg, a.b) {
			m.last = a.b.Help().Desc
			return a.do(m)
		}
	}
	return nil
}

func (m *app) actions() []action {
	quit := action{bind([]string{"q"}, "q", "quit"), true, func(*app) tea.Cmd { return tea.Quit }}
	help := action{bind([]string{"?"}, "?", "keys"), true, func(m *app) tea.Cmd { m.helpFrom, m.ov = m.ov, ovHelp; return nil }}
	remove := action{bind([]string{"x"}, "x", "remove"), true, func(m *app) tea.Cmd { m.ov, m.btn = ovConfirm, 1; return nil }}
	if m.w < minW || m.h < minH {
		return []action{quit}
	}
	switch m.ov {
	case ovHelp:
		closeHelp := func(m *app) tea.Cmd { m.ov, m.helpFrom, m.helpOff = m.helpFrom, ovNone, 0; return nil }
		return []action{
			{bind([]string{"esc"}, "esc", "close"), true, closeHelp},
			{bind([]string{"up", "down"}, m.t.G.UpDown, "scroll"), len(m.helpBody()) > m.helpRows(), func(m *app) tea.Cmd {
				m.helpOff = min(ui.ClipMax(len(m.helpBody()), m.helpRows()), m.helpOff+1)
				return nil
			}},
			{bind([]string{"?"}, "?", "close"), false, closeHelp},
		}
	case ovConfirm:
		return []action{
			{bind([]string{"enter"}, "enter", map[int]string{0: "remove", 1: "cancel"}[m.btn]), true, func(m *app) tea.Cmd {
				if m.btn == 0 {
					m.items = append(m.items[:m.sel:m.sel], m.items[m.sel+1:]...)
					m.sel = min(m.sel, len(m.items)-1)
				}
				m.ov = ovNone
				return nil
			}},
			{bind([]string{"left", "right"}, m.t.G.LeftRight, "switch"), true, func(m *app) tea.Cmd { m.btn = 1 - m.btn; return nil }},
			{bind([]string{"esc"}, "esc", "cancel"), true, func(m *app) tea.Cmd { m.ov = ovNone; return nil }},
			help,
		}
	}
	if m.screen == scrDetail {
		return []action{{bind([]string{"esc"}, "esc", "back"), true, func(m *app) tea.Cmd { m.screen = scrList; return nil }}, remove, quit, help}
	}
	return []action{
		{bind([]string{"up", "down"}, m.t.G.UpDown, "select"), true, func(m *app) tea.Cmd { m.sel = (m.sel + 1) % len(m.items); return nil }},
		{bind([]string{"enter"}, "enter", "open"), true, func(m *app) tea.Cmd { m.screen = scrDetail; return nil }},
		remove, quit, help,
	}
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
