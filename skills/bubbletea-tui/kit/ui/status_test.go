package ui

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/colorprofile"
	uv "github.com/charmbracelet/ultraviolet"
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

// A section rule, a table's header rule and the footer rule are drawn in the border
// role, a step off bg. At 16 colours that step downsampled onto the page and every rule
// vanished. Read back as the terminal holds them, their cells are never bg's colour.
func TestRulesVisibleAt16And256Colours(t *testing.T) {
	files := append(testThemes(t), filepath.Join("..", "theme", "testdata", "quire.json"))
	for _, f := range files {
		tf, err := theme.Load(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, dark := range []bool{true, false} {
			for _, p := range []colorprofile.Profile{colorprofile.ANSI, colorprofile.ANSI256} {
				th := theme.ResolveProfile(tf, dark, false, p)
				rows := []string{Section(th, "Activity", 40, "last hour")}
				rows = append(rows, Table(th, []string{"service", "when"}, [][]string{{"ledger", "2d"}}, []int{0, 6}, "lr", -1, 40, false)...)
				s := strings.Join(rows, "\n")
				buf := uv.NewScreenBuffer(40, len(rows))
				uv.NewStyledString(downsample(s, p)).Draw(buf, buf.Bounds())
				bg := p.Convert(th.P.Bg)
				for y := range len(rows) {
					for x := range 40 {
						c := buf.CellAt(x, y)
						if c == nil || c.Content != th.G.Rule {
							continue
						}
						if c.Style.Fg == nil || sameColor(c.Style.Fg, bg) {
							t.Errorf("%s dark=%v %v: rule cell %d,%d draws in the page's colour", filepath.Base(f), dark, p, x, y)
							break
						}
					}
				}
			}
		}
	}
}

// The kit forced case on a brand: table headers always upper (SERVICE), so a lowercase
// editorial brand couldn't keep its type there, and tags drew what the app wrote
// (SETUP). expressions.labelCase sets table headers, tags, toast labels and summary
// labels together; unset, each keeps the kit's own case, so themes render as before.
func TestLabelCase(t *testing.T) {
	base := theme.Default()
	for _, c := range []struct {
		labelCase                 string
		header, tag, toast, strip string
	}{
		{"", "SERVICE", "SETUP", "DEPLOYED", "namespace"},
		{"lower", "service", "setup", "deployed", "namespace"},
		{"upper", "SERVICE", "SETUP", "DEPLOYED", "NAMESPACE"},
		{"as-is", "Service", "SETUP", "DEPLOYED", "Namespace"},
	} {
		f := base
		f.Expressions.LabelCase = c.labelCase
		th := theme.Resolve(f, true, false)
		got := map[string]string{
			"header": strings.Fields(ansi.Strip(Table(th, []string{"Service", "when"}, nil, []int{10, 0}, "lr", -1, 40, false)[0]))[0],
			"tag":    strings.TrimSpace(ansi.Strip(Tag(th, "info", "SETUP"))),
			"toast":  strings.Fields(ansi.Strip(Toast(th, "ok", "DEPLOYED", "ledger v7.1.0", 60)))[0],
			"strip":  strings.Fields(ansi.Strip(SummaryStrip(th, []SummaryRow{{Label: "Namespace", Value: "payments", Set: true}}, 60)))[0],
		}
		for what, want := range map[string]string{"header": c.header, "tag": c.tag, "toast": c.toast, "strip": c.strip} {
			if got[what] != want {
				t.Errorf("labelCase %q: %s %q, want %q", c.labelCase, what, got[what], want)
			}
		}
	}
}
