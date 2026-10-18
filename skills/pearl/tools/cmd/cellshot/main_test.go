package main

import (
	"strings"
	"testing"
)

// The -h text is the package doc comment; make sure it survives the extraction whole.
func TestUsageIsTheDocComment(t *testing.T) {
	u := usage()
	for _, want := range []string{
		"Command cellshot renders ANSI terminal frames",
		"  cellshot a.ansi b.ansi c.ansi --sheet out.png",
		"  --cols N --rows N expected size",
		"Exit status 1 if --cols/--rows mismatch.",
	} {
		if !strings.Contains(u, want) {
			t.Errorf("usage lacks %q:\n%s", want, u)
		}
	}
	if strings.Contains(u, "package main") || strings.Contains(u, "//") {
		t.Errorf("usage leaks source:\n%s", u)
	}
}
