package ui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"charm.land/bubbles/v2/key"
	"github.com/charmbracelet/x/ansi"

	"tuikit/theme"
)

// testdata/footer-cases.json is shared with tuilint spec's S-FOOTER
// (tools/internal/speclint), which models this footer: both tests hold their
// footer to the same dropped sets at 80 columns on the default theme.
func TestFooterCases(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("testdata", "footer-cases.json"))
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Name           string
		Hints, Dropped []string
	}
	if err := json.Unmarshal(b, &cases); err != nil {
		t.Fatal(err)
	}
	th := theme.Resolve(theme.Default(), true, false)
	for _, c := range cases {
		var bs []key.Binding
		for _, h := range c.Hints {
			k, desc, _ := strings.Cut(h, " ")
			bs = append(bs, key.NewBinding(key.WithKeys(k), key.WithHelp(k, desc)))
		}
		rows := Footer(th, 80, "", bs, "")
		row := ansi.Strip(rows[len(rows)-1])
		var dropped []string
		for _, h := range c.Hints {
			if !strings.Contains(row, h) {
				dropped = append(dropped, h)
			}
		}
		if strings.Join(dropped, "|") != strings.Join(c.Dropped, "|") {
			t.Errorf("%s: Footer drops %q, footer-cases.json says %q\n%q", c.Name, dropped, c.Dropped, row)
		}
	}
}
