package themes

import (
	"encoding/json"
	"testing"
)

// The embedded default parses and names a brand: in the kit it is ink; in a project that
// copied one theme to themes/<brand>.json (kit/README.md), that theme.
func TestDefaultIsATheme(t *testing.T) {
	b, err := Default()
	if err != nil {
		t.Fatal(err)
	}
	var f struct{ Name, Brand string }
	if err := json.Unmarshal(b, &f); err != nil {
		t.Fatal(err)
	}
	if f.Name == "" || f.Brand == "" {
		t.Fatalf("the default theme has no name or brand: %q %q", f.Name, f.Brand)
	}
	entries, _ := FS.ReadDir(".")
	if len(entries) > 1 && f.Name != "ink" {
		t.Errorf("among %d themes the default is %q, want ink", len(entries), f.Name)
	}
}
