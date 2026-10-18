package ui

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// Neither cold monitor (#61) could use TooSmall: their specs gave the screen its own
// sentence ("Terminal is 60×18; Kestrel needs at least 80×24 for the current settings")
// and "nothing else", and TooSmall's copy and key row were fixed. TooSmallText draws the
// caller's sentence, wrapped and centred, with a key row only when it is given keys.
func TestTooSmallTextSaysTheSpec(t *testing.T) {
	th := testTheme(t, false)
	say := "Terminal is 60×18; app needs at least 80×24 for the current settings"
	words := func(s string) string { return strings.Join(strings.Fields(ansi.Strip(s)), " ") }

	bare := TooSmallText(th, 60, 18, say, nil)
	if got := words(bare); got != say {
		t.Errorf("no keys: the screen reads %q, want the sentence and nothing else, %q", got, say)
	}
	if lipgloss.Width(bare) != 60 || lipgloss.Height(bare) != 18 {
		t.Errorf("TooSmallText is %dx%d, want the whole 60x18 screen", lipgloss.Width(bare), lipgloss.Height(bare))
	}
	for _, r := range strings.Split(ansi.Strip(bare), "\n") {
		if r = strings.TrimRight(r, " "); r != "" && lipgloss.Width(r) > 60 {
			t.Errorf("row %q overflows 60 cells", r)
		}
	}
	off := key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "gone"), key.WithDisabled())
	if got := words(TooSmallText(th, 60, 18, say, []key.Binding{off})); got != say {
		t.Errorf("a disabled key drew a key row: %q", got)
	}

	quit := key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit"))
	rows := strings.Split(ansi.Strip(TooSmallText(th, 60, 18, say, []key.Binding{quit})), "\n")
	last := ""
	for _, r := range rows {
		if r = strings.Join(strings.Fields(r), " "); r != "" {
			last = r
		}
	}
	if last != "q quit" {
		t.Errorf("with a key: the last row reads %q, want the key row %q", last, "q quit")
	}
}
