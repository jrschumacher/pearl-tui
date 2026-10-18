// Command good is clean: its model, key bindings and colour handling live in packages of
// its own module, which the program-wide rules follow through the imports.
package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"example.com/good/app"
)

func main() {
	if _, err := tea.NewProgram(app.New(true)).Run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
