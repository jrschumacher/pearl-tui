package redact

import (
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
)

func TestString(t *testing.T) {
	r := New("deploy")
	for in, want := range map[string]string{
		"Payments EU v2.4.1":                 "Xxxxxxxx XX x0.0.0",
		"enter open · esc back · q quit":     "enter xxxx · esc back · q quit",
		"ctrl+c quit · ctrl+k palette":       "ctrl+c quit · ctrl+k xxxxxxx",
		"↑/↓ move  ←/→ switch  ? help":       "↑/↓ xxxx  ←/→ xxxxxx  ? help",
		"j/k move · tab/shift+tab · y/n":     "x/x xxxx · tab/shift+tab · x/x",
		"see https://acme.test/a?b=1 now":    "xxx xxxxxxxxxxxxxxxxxxxxxx0 xxx",
		"(mail ops@acme.test)":               "(xxxx xxxxxxxxxxxxx)",
		"in ~/.kube/config, or /etc/x.conf.": "xx xxxxxxxxxxxxxx, xx xxxxxxxxxxx.",
		"kit/README.md and a/b/c":            "xxxxXXXXXXxxx xxx xxxxx",
		"Deploy ENTER Help":                  "Deploy ENTER Help",
		"ÉCLAIR 東京 42%":                      "XXXXXX xx 00%",
		"╭── Ok ──╮ ● 3/6":                   "╭── Ok ──╮ ● 0/0",
	} {
		if got := r.String(in); got != want {
			t.Errorf("String(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestLinesKeepWidthsAndStyles(t *testing.T) {
	bold := uv.Style{Attrs: uv.AttrBold}
	line := uv.Line{
		{Content: "東", Width: 2, Style: bold}, {},
		{Content: "a", Width: 1, Style: bold},
		{Content: " ", Width: 1},
		{Content: "─", Width: 1, Style: bold},
		{Content: "q", Width: 1},
	}
	New().Lines([]uv.Line{line})
	want := []string{"x", "x", "x", " ", "─", "q"}
	for i, c := range line {
		if c.Content != want[i] || c.Width != 1 && !(i == 3 || i == 4 || i == 5) {
			t.Errorf("cell %d = %q width %d, want %q", i, c.Content, c.Width, want[i])
		}
		if i <= 2 && !c.Style.Equal(&bold) {
			t.Errorf("cell %d lost its style", i)
		}
	}
}
