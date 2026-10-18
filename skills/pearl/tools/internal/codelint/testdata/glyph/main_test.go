package main

import "testing"

// A test file is not linted: its literals are expectations, not screens.
const want = "▏ · →"

func TestView(t *testing.T) {
	if view() == want {
		t.Fatal("unexpected")
	}
}
