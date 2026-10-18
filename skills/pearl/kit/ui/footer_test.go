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

// A key token that pairs a key some terminals can't tell apart (shift+enter arrives as
// enter in tmux) with its fallback is drawn as the one that works: ctrl+j, or
// shift+enter once the terminal reports key disambiguation. Drawn whole, the 18-cell
// "shift+enter/ctrl+j" cost the footer a key at 80 columns. Alternatives every terminal
// sends ("↑↓/jk") are all drawn.
func TestHelpDrawsOneKeyAlternative(t *testing.T) {
	bs := []key.Binding{
		key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑↓/jk", "move")),
		key.NewBinding(key.WithKeys("shift+enter", "ctrl+j"), key.WithHelp("shift+enter/ctrl+j", "newline")),
		key.NewBinding(key.WithKeys("ctrl+enter", "ctrl+s"), key.WithHelp("ctrl+enter/ctrl+s", "send")),
	}
	for _, c := range []struct {
		disambiguates bool
		want, not     []string
	}{
		{false, []string{"↑↓/jk move", "ctrl+j newline", "ctrl+s send"}, []string{"shift+enter", "ctrl+enter"}},
		{true, []string{"↑↓/jk move", "shift+enter newline", "ctrl+enter send"}, []string{"ctrl+j", "ctrl+s"}},
	} {
		th := theme.Resolve(theme.Default(), true, false)
		th.KeyDisambiguation = c.disambiguates
		row := ansi.Strip(Help(th, bs, 76))
		for _, w := range c.want {
			if !strings.Contains(row, w) {
				t.Errorf("disambiguates=%v: Help draws %q, want %q", c.disambiguates, row, w)
			}
		}
		for _, n := range c.not {
			if strings.Contains(row, n) {
				t.Errorf("disambiguates=%v: Help draws %q, which names %q", c.disambiguates, row, n)
			}
		}
	}
}

// ambiguousKey is S-KEYAMBIG's legacy model (tools/internal/speclint/keys.go): the keys a
// terminal without key disambiguation delivers as another key or not at all.
func TestAmbiguousKey(t *testing.T) {
	for k, want := range map[string]bool{
		"shift+enter": true, "ctrl+enter": true, "alt+enter": false, "enter": false,
		"ctrl+backspace": true, "shift+esc": true, "ctrl+tab": true, "shift+tab": false,
		"shift+space": true, "ctrl+space": false, "ctrl+m": true, "ctrl+i": true,
		"ctrl+shift+a": true, "ctrl+j": false, "ctrl+_": false, "ctrl+/": true, "ctrl+2": true,
		"super+k": true, "alt+k": false, "j": false, "?": false,
	} {
		if got := ambiguousKey(k); got != want {
			t.Errorf("ambiguousKey(%q) = %v, want %v", k, got, want)
		}
	}
}
