// Command quiet only talks about what it should do. TODO: handle tea.WindowSizeMsg,
// honour NO_COLOR, and move the keys to key.Binding with a help view.
package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
)

type model struct{ n int }

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch k.String() {
		case "q": // want "C-KEYS: 2 keys"
			return m, tea.Quit
		case "j":
			m.n++
		}
	}
	return m, nil
}

func (m model) View() tea.View { return tea.NewView(fmt.Sprint(m.n)) }

func main() {
	if _, err := tea.NewProgram(model{}).Run(); err != nil { // want "C-SIZE" "C-NOCOLOR"
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
