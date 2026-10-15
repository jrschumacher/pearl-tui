// Package app is the root model.
package app

import (
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"example.com/good/keys"
	"example.com/good/look"
)

type (
	tickMsg   struct{}
	loadedMsg []byte
)

// Model is the root model.
type Model struct {
	width int
	items []string
	label string
	dark  bool
	data  []byte
}

// New returns the root model.
func New(dark bool) *Model { return &Model{label: "ready", dark: dark} }

// Init starts the first load.
func (m *Model) Init() tea.Cmd { return load }

// load reads the state file inside a Cmd, off the update goroutine.
func load() tea.Msg {
	b, _ := os.ReadFile("state.json")
	return loadedMsg(b)
}

// Update handles messages.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
	case loadedMsg:
		m.data = msg
	case tea.KeyPressMsg:
		if keys.Quit(msg) || msg.String() == "q" {
			return m, tea.Quit
		}
		return m, tea.Tick(time.Second, func(time.Time) tea.Msg {
			time.Sleep(10 * time.Millisecond) // a Tick callback runs in its own goroutine
			return tickMsg{}
		})
	}
	return m, nil
}

// View renders the model.
func (m *Model) View() tea.View {
	if len(m.items) > 0 && lipgloss.Width(m.items[0]) < m.width {
		return tea.NewView(m.items[0])
	}
	pad := strings.Repeat(" ", max(0, m.width-lipgloss.Width(m.label)))
	return tea.NewView(look.Title(m.dark).Render(m.label) + pad)
}
