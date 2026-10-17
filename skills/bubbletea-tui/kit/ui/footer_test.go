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
// footer to the same dropped sets at 80 columns on the default theme, resolved
// for ASCII where a case says "ascii".
func TestFooterCases(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("testdata", "footer-cases.json"))
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Name           string
		ASCII          bool
		Hints, Dropped []string
	}
	if err := json.Unmarshal(b, &cases); err != nil {
		t.Fatal(err)
	}
	for _, c := range cases {
		th := theme.Resolve(theme.Default(), true, c.ASCII)
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

// Under -ascii the ellipsis is "...", drawn as "  ..." (5 cells), not "  …" (3): Help
// kept room for 3, so a cut footer overflowed and Fit cut it to a non-ASCII ".…".
func TestFooterASCIIEllipsisFits(t *testing.T) {
	th := theme.Resolve(theme.Default(), true, true)
	var bs []key.Binding
	for _, h := range []string{"j/k move", "enter open", "e edit", "s snooze", "x select", "E done selected", "/ filter", "r refresh", "R reload all", "d delete"} {
		k, desc, _ := strings.Cut(h, " ")
		bs = append(bs, key.NewBinding(key.WithKeys(k), key.WithHelp(k, desc)))
	}
	for w := 40; w <= 90; w++ {
		rows := Footer(th, w, "", bs, "")
		row := ansi.Strip(rows[len(rows)-1])
		if n := ansi.StringWidth(row); n != w {
			t.Errorf("w=%d: footer row is %d cells: %q", w, n, row)
		}
		for _, r := range row {
			if r > 127 {
				t.Errorf("w=%d: footer row has non-ASCII %q: %q", w, r, row)
				break
			}
		}
		if body := strings.TrimRight(row, " "); !strings.HasSuffix(body, "  ...") || strings.HasSuffix(body, "....") {
			t.Errorf("w=%d: a cut ASCII footer ends in %q, want the full \"  ...\"", w, body[max(0, len(body)-8):])
		}
	}
}
