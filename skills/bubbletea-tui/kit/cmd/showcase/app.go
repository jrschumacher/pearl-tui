package main

import (
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"tuikit/theme"
	"tuikit/ui"
)

// One root model. Regions are plain structs drawn into rectangles by View;
// there are no nested tea.Models to keep in sync (Crush's architecture).

type screen int

const (
	scrServices screen = iota
	scrDeploys
	scrIncidents
	scrAssistant
	scrDeploy // the deploy form (from services)
	scrSplash
)

type overlay int

const (
	ovNone overlay = iota
	ovPalette
	ovConfirm
)

type app struct {
	file   theme.File
	t      *theme.Theme
	ascii  bool
	w, h   int
	screen screen
	ov     overlay
	sel    int // selected service
	palSel int // selected command in the palette
	query  string
	focus  int // deploy form: focused field
	btn    int // focused button in a dialog / form
	frame  int // the one animation clock
	toast  *toast
	busy   bool // assistant is working (drives the anim)
	keys   keymap
}

type toast struct {
	kind, label, msg string
	until            time.Time
}

type tickMsg struct{ gen int }

var farFuture = time.Now().Add(24 * time.Hour)

type keymap struct {
	Up, Down, Tab, Palette, Deploy, Rollback, Back, Enter, Quit, Help, Next key.Binding
}

func newKeys() keymap {
	return keymap{
		Up:       key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:     key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Tab:      key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next view")),
		Palette:  key.NewBinding(key.WithKeys("ctrl+p", ":"), key.WithHelp("ctrl+p", "commands")),
		Deploy:   key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "deploy")),
		Rollback: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "roll back")),
		Back:     key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		Enter:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
		Quit:     key.NewBinding(key.WithKeys("ctrl+c", "q"), key.WithHelp("q", "quit")),
		Help:     key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Next:     key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next field")),
	}
}

func newApp(f theme.File, dark, ascii bool) *app {
	return &app{file: f, t: theme.Resolve(f, dark, ascii), ascii: ascii, w: 80, h: 24, keys: newKeys()}
}

func (m *app) Init() tea.Cmd {
	// ask the terminal what its background is; the reply re-resolves the theme
	return tea.Batch(tea.RequestBackgroundColor, m.tick())
}

// animating reports whether anything on screen moves. When nothing does, the
// clock stops: an idle screen costs zero bytes.
func (m *app) animating() bool {
	return m.screen == scrSplash || m.busy || m.screen == scrServices && m.ov == ovNone
}

func (m *app) tick() tea.Cmd {
	if !m.animating() {
		return nil
	}
	return tea.Tick(ui.FrameInterval, func(time.Time) tea.Msg { return tickMsg{} })
}

func (m *app) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
	case tea.BackgroundColorMsg:
		m.t = theme.Resolve(m.file, msg.IsDark(), m.ascii)
	case tickMsg:
		m.frame++
		if m.toast != nil && time.Now().After(m.toast.until) {
			m.toast = nil
		}
		return m, m.tick()
	case tea.KeyPressMsg:
		return m.key(msg)
	}
	return m, nil
}

func (m *app) key(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	k := m.keys
	if msg.String() == "ctrl+c" {
		return m, tea.Quit
	}
	// overlays take input first; only the front one
	switch m.ov {
	case ovPalette:
		switch {
		case key.Matches(msg, k.Back):
			m.ov = ovNone
		case key.Matches(msg, k.Up):
			m.palSel = max(0, m.palSel-1)
		case key.Matches(msg, k.Down):
			m.palSel = min(len(m.filtered())-1, m.palSel+1)
		case key.Matches(msg, k.Enter):
			m.ov = ovNone
			if cs := m.filtered(); len(cs) > 0 && cs[m.palSel].c.Title == "Roll back" {
				m.ov, m.btn = ovConfirm, 1
			}
		case msg.String() == "backspace":
			if r := []rune(m.query); len(r) > 0 {
				m.query = string(r[:len(r)-1])
				m.palSel = 0
			}
		default:
			if msg.Text != "" {
				m.query += msg.Text
				m.palSel = 0
			}
		}
		return m, m.tick()
	case ovConfirm:
		switch msg.String() {
		case "esc", "n":
			m.ov = ovNone
		case "left", "right", "tab", "h", "l":
			m.btn = 1 - m.btn
		case "enter", "y":
			m.ov = ovNone
			if m.btn == 0 || msg.String() == "y" {
				m.toast = &toast{"ok", "DONE", "Rolled back checkout-api to v2.3.9 · 3/3 pods healthy", time.Now().Add(4 * time.Second)}
			}
		}
		return m, m.tick()
	}
	switch {
	case msg.String() == "q" && m.screen != scrDeploy:
		return m, tea.Quit
	case key.Matches(msg, k.Palette):
		m.ov, m.query, m.palSel = ovPalette, "", 0
	case m.screen == scrDeploy:
		switch msg.String() {
		case "esc":
			m.screen = scrServices
		case "tab", "down":
			m.focus = (m.focus + 1) % 4
		case "shift+tab", "up":
			m.focus = (m.focus + 3) % 4
		}
	case key.Matches(msg, k.Tab):
		m.screen = (m.screen + 1) % 4
	case key.Matches(msg, k.Up):
		m.sel = max(0, m.sel-1)
	case key.Matches(msg, k.Down):
		m.sel = min(len(services)-1, m.sel+1)
	case key.Matches(msg, k.Deploy):
		m.screen, m.focus = scrDeploy, 1
	case key.Matches(msg, k.Rollback):
		m.ov, m.btn = ovConfirm, 1
	}
	return m, m.tick()
}

func (m *app) View() tea.View {
	c := m.render(m.w, m.h)
	v := tea.NewView(c.String())
	v.AltScreen = true
	v.WindowTitle = m.t.AppName() + " · " + services[m.sel].Name
	if m.t.X.PaintBg {
		v.BackgroundColor = m.t.P.Bg
		v.ForegroundColor = m.t.P.Fg
	}
	return v
}
