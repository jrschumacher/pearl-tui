// Command broken has dependencies that cannot be resolved: tuilint analyses it from syntax,
// with types only for its own declarations.
package main

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type tickMsg struct{}

type model struct {
	width int
	name  string
	tabs  []string
	list  menu
}

type menu interface{ Items() []string }

func (m model) Init() tea.Cmd {
	return tea.Tick(time.Second/60, func(time.Time) tea.Msg { return tickMsg{} }) // want "C-FASTTICK: 16.7ms"
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q": // want "C-KEYS: 2 keys"
			return m, tea.Quit
		case "s":
			time.Sleep(time.Second) // want "C-SLEEP"
			return m, func() tea.Msg {
				time.Sleep(time.Second)
				return tickMsg{}
			}
		}
	}
	return m, nil
}

func (m model) View() tea.View {
	pad := strings.Repeat(" ", m.width-len(m.name))          // want `C-LEN: len\(m.name\)`
	bar := strings.Repeat("─", m.width-len(m.tabs))          // a slice
	more := strings.Repeat(" ", m.width-len(m.list.Items())) // a slice: types are known for local declarations
	rendered := lipgloss.NewStyle().Render(m.name)
	gap := strings.Repeat(" ", m.width-len(rendered))                                        // want `C-LEN: len\(rendered\)`
	title := lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Render("🚀 " + m.name) // want "C-HEX" "C-EMOJI"
	fmt.Println(title)                                                                       // want "C-PRINT"
	return tea.NewView(title + pad + bar + more + gap)
}

func main() {
	if _, err := tea.NewProgram(model{}).Run(); err != nil { // want "C-SIZE" "C-NOCOLOR"
		fmt.Println(err)
	}
}
