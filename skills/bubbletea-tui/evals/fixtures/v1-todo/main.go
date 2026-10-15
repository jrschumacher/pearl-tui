// A deliberately typical tutorial-style Bubble Tea v1 app: the "before" for the
// restyle and critique evals. It works. It looks like every other one.
package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFDF5")).Background(lipgloss.Color("#7D56F4")).Padding(0, 1)
	boxStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62")).Padding(1, 2)
	helpStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

type item struct{ title, desc string }

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

type model struct {
	list list.Model
	done map[int]bool
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case " ":
			m.done[m.list.Index()] = !m.done[m.list.Index()]
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	done := 0
	for _, v := range m.done {
		if v {
			done++
		}
	}
	header := titleStyle.Render("✨ My Todos ✨")
	stats := boxStyle.Render(fmt.Sprintf("✅ Done: %d\n📋 Total: %d", done, len(m.list.Items())))
	return lipgloss.JoinVertical(lipgloss.Left, header, boxStyle.Render(m.list.View()), stats,
		helpStyle.Render("Press space to toggle, / to filter, q to quit. Have a great day! 🚀"))
}

func main() {
	items := []list.Item{
		item{"Write the README", "docs"}, item{"Fix the flaky test", "ci"}, item{"Ship v1.2", "release"},
		item{"Reply to issue #42", "community"}, item{"Update dependencies", "chore"},
	}
	l := list.New(items, list.NewDefaultDelegate(), 60, 14)
	l.Title = "Todos"
	if _, err := tea.NewProgram(model{list: l, done: map[int]bool{}}, tea.WithAltScreen()).Run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
