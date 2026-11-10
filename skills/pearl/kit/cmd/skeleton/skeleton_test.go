package main

import (
	"testing"

	"github.com/jrschumacher/pearl-tui/skills/pearl/kit/harness/bubbletea"
)

// A test of your app's own behaviour, beside the harness: drive the model with the
// keys a user presses (harness.Press) and check the state they leave. Write one for
// each law of your app the harness can't know.
func TestOpenBackAndSafeConfirm(t *testing.T) {
	f, err := embeddedTheme()
	if err != nil {
		t.Fatal(err)
	}
	m := newApp(f, true, false)
	n := len(m.items)
	for _, k := range []string{"down", "enter"} {
		m.key(harness.Press(k))
	}
	if m.screen != scrDetail || m.sel != 1 {
		t.Fatalf("down, enter: screen %d sel %d, want the second item open", m.screen, m.sel)
	}
	m.key(harness.Press("x"))
	if m.ov != ovConfirm || m.btn != btnCancel {
		t.Fatalf("x: overlay %d button %d, want the confirm on Cancel", m.ov, m.btn)
	}
	m.key(harness.Press("enter")) // the default button keeps the item
	if m.ov != ovNone || len(m.items) != n {
		t.Fatalf("enter on the default: %d items, want %d kept", len(m.items), n)
	}
	m.key(harness.Press("esc"))
	if m.screen != scrList {
		t.Fatalf("esc: screen %d, want the list", m.screen)
	}
	for _, k := range []string{"x", "left", "enter"} {
		m.key(harness.Press(k))
	}
	if len(m.items) != n-1 || m.sel != 1 {
		t.Fatalf("remove: %d items, sel %d; want %d, the next item selected", len(m.items), m.sel, n-1)
	}
}
