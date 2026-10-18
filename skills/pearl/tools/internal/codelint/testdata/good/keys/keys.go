// Package keys holds the key bindings, so help and handling agree.
package keys

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

var quit = key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit"))

// Quit reports whether msg is the quit key.
func Quit(msg tea.KeyPressMsg) bool { return key.Matches(msg, quit) }
