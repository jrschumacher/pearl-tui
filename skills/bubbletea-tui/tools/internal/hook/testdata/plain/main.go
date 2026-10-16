// Command plain is not a TUI: no Bubble Tea, no Huh. Its emoji would fail tuilint code
// (C-EMOJI), and the hook must stay silent anyway.
package main

import "fmt"

func main() {
	fmt.Println("✨ done")
}
