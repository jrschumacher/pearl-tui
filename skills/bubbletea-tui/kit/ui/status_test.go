package ui

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"tuikit/theme"
)

// An item in progress (a deploy rolling, a tool call running) is its own state, and
// it drew as another one in a different colour: Dot's busy as the current glyph, ok's
// own, and a running tool call as a pending one. Under NO_COLOR or ASCII the two were
// one mark. Both draw the partial rung, so the shape says it, not only the colour.
func TestInProgressHasItsOwnShape(t *testing.T) {
	files := append(testThemes(t), filepath.Join("..", "theme", "testdata", "quire.json"))
	for _, f := range files {
		tf, err := theme.Load(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, ascii := range []bool{false, true} {
			th := theme.Resolve(tf, true, ascii)
			busy := ansi.Strip(Dot(th, "busy"))
			for _, other := range []string{"ok", "error", "warning", "idle"} {
				if o := ansi.Strip(Dot(th, other)); o == busy {
					t.Errorf("%s ascii=%v: Dot busy and %s are both %q", filepath.Base(f), ascii, other, busy)
				}
			}
			if busy != th.G.Partial {
				t.Errorf("%s ascii=%v: Dot busy is %q, want the partial glyph %q", filepath.Base(f), ascii, busy, th.G.Partial)
			}
			icon := func(s ToolState) string {
				return strings.Fields(ansi.Strip(ToolCall(th, Tool{Name: "Edit", State: s}, 60, false)[0]))[0]
			}
			if run, wait := icon(ToolRunning), icon(ToolPending); run == wait {
				t.Errorf("%s ascii=%v: a running and a pending tool call both lead with %q", filepath.Base(f), ascii, run)
			}
		}
	}
}
