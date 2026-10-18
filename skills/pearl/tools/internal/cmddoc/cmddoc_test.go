package cmddoc

import "testing"

func TestText(t *testing.T) {
	src := "// Copyright header, not the doc.\n\n// Tool does a thing.\n//\n//\ttool run x\n//   - a list item\npackage main\n\n// not doc\n"
	want := "Tool does a thing.\n\n  tool run x\n  - a list item"
	if got := Text(src); got != want {
		t.Errorf("Text = %q, want %q", got, want)
	}
}
