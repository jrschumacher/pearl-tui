// Command bare is someone's Bubble Tea app, not built with the skill: no TUI.md, no kit.
// It breaks three tuilint code rules (C-HEX, C-EMOJI, C-SIZE), and the hook must stay
// silent anyway.
package main

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var title = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5F87"))

type model struct{}

func (model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(tea.KeyPressMsg); ok {
		return m, tea.Quit
	}
	return m, nil
}

func (model) View() tea.View { return tea.NewView(title.Render("✅ done")) }

func main() {
	tea.NewProgram(model{}).Run()
}
