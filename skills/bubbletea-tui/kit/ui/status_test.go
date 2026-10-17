package ui

import (
	"path/filepath"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
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
		labelCase                          string
		header, tag, toast, strip, summary string
	}{
		{"", "SERVICE", "SETUP", "DEPLOYED", "namespace", ""},
		{"lower", "service", "setup", "deployed", "namespace", "namespace"},
		{"upper", "SERVICE", "SETUP", "DEPLOYED", "NAMESPACE", "NAMESPACE"},
		{"as-is", "Service", "SETUP", "DEPLOYED", "Namespace", "Namespace"},
	} {
		f := base
		f.Expressions.LabelCase = c.labelCase
		th := theme.Resolve(f, true, false)
		got := map[string]string{
			"header":  strings.Fields(ansi.Strip(Table(th, []string{"Service", "when"}, nil, []int{10, 0}, "lr", -1, 40, false)[0]))[0],
			"tag":     strings.TrimSpace(ansi.Strip(Tag(th, "info", "SETUP"))),
			"toast":   strings.Fields(ansi.Strip(Toast(th, "ok", "DEPLOYED", "ledger v7.1.0", 60)))[0],
			"strip":   strings.Fields(ansi.Strip(SummaryStrip(th, []SummaryRow{{Label: "Namespace", Value: "payments", Set: true}}, 60)))[0],
			"summary": strings.Fields(ansi.Strip(Summary(th, "Summary", "", []SummaryRow{{Label: "Namespace", Value: "payments", Set: true}}, 40)[2]))[0],
		}
		if c.labelCase == "" {
			c.summary = "Namespace" // the sidebar's labels keep the app's case unless the theme sets one
		}
		for what, want := range map[string]string{"header": c.header, "tag": c.tag, "toast": c.toast, "strip": c.strip, "summary": c.summary} {
			if got[what] != want {
				t.Errorf("labelCase %q: %s %q, want %q", c.labelCase, what, got[what], want)
			}
		}
	}
}

// A status column holds one cell: the dot of each row, and the working mark of a row in
// progress. Spinner draws the theme's frames, which may be wider (lumen's "···"), so
// that row's name sat two cells right of the rest. Busy is one cell in every theme.
func TestBusyIsOneCell(t *testing.T) {
	files := append(testThemes(t), filepath.Join("..", "theme", "testdata", "quire.json"))
	for _, f := range files {
		tf, err := theme.Load(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, ascii := range []bool{false, true} {
			th := theme.Resolve(tf, true, ascii)
			for n := range 12 {
				if w := lipgloss.Width(Busy(th, n)); w != 1 {
					t.Errorf("%s ascii=%v frame %d: Busy is %d cells (%q)", filepath.Base(f), ascii, n, w, ansi.Strip(Busy(th, n)))
				}
			}
		}
	}
}

// selection "bar" and "caret" mark the selected row with a glyph and bold text, as List
// and Huh's selects do (theme.Styles.SelRow). Table drew only the glyph, so under focus
// "fg", where the glyph is fg too, a selected table row was one bar away from the rest.
func TestTableEmphasisesTheSelectedRow(t *testing.T) {
	for _, sel := range []string{"bar", "caret"} {
		f := quire(t)
		f.Expressions.Selection = sel
		th := theme.Resolve(f, true, false)
		rows := Table(th, []string{"service", "when"}, [][]string{{th.S.Base.Render("ledger"), th.S.Subtle.Render("2d")}, {"auth", "8d"}}, []int{10, 0}, "lr", 0, 30, true)
		buf := uv.NewScreenBuffer(30, len(rows))
		uv.NewStyledString(strings.Join(rows, "\n")).Draw(buf, buf.Bounds())
		bold := func(y int, word string) bool {
			x := strings.Index(ansi.Strip(rows[y]), word)
			c := buf.CellAt(len([]rune(ansi.Strip(rows[y])[:x])), y)
			return c != nil && c.Style.Attrs&uv.AttrBold != 0
		}
		if !bold(2, "ledger") || !bold(2, "2d") {
			t.Errorf("selection %s: the selected row's text isn't bold", sel)
		}
		if bold(3, "auth") {
			t.Errorf("selection %s: an unselected row is bold", sel)
		}
	}
}

// Under focus "fg" the active tab is marked in fg and weight in every tab style, the
// underline row included (quire's own tabs are slash, which has none).
func TestFocusFgTabs(t *testing.T) {
	for _, tabs := range []string{"underline", "pill", "bracket", "slash"} {
		f := quire(t)
		f.Expressions.Tabs = tabs
		for _, p := range []colorprofile.Profile{colorprofile.TrueColor, colorprofile.ANSI256, colorprofile.ANSI} {
			th := theme.ResolveProfile(f, true, false, p)
			if hits := primaryCells(th, strings.Join(Tabs(th, []string{"Inbox", "Saved"}, 0, nil), "\n"), p); len(hits) > 0 {
				t.Errorf("tabs %s %v: primary at %v", tabs, p, hits)
			}
		}
	}
}

// The blurred selection gutter (S.BlurBar: an unfocused list or table under a dialog)
// drew in bgSubtle, about 1.2:1 on the page: invisible at truecolour, and under NO_COLOR
// a bare | told from the focused bar by bold alone. It is fgSubtle, a text level that
// holds the 2.3:1 floor on bg in every theme and mode, and Huh's blurred field bar is
// the same ink.
func TestBlurBarIsVisible(t *testing.T) {
	files := append(testThemes(t), filepath.Join("..", "theme", "testdata", "quire.json"))
	for _, f := range files {
		tf, err := theme.Load(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, dark := range []bool{true, false} {
			th := theme.Resolve(tf, dark, false)
			ink := th.S.BlurBar.GetForeground()
			if !sameColor(ink, th.P.FgSubtle) {
				t.Errorf("%s dark=%v: BlurBar draws %v, want fgSubtle %v", filepath.Base(f), dark, ink, th.P.FgSubtle)
			}
			if r := theme.Contrast(ink, th.P.Bg); r < 2.3 {
				t.Errorf("%s dark=%v: BlurBar is %.2f:1 on bg, under the 2.3 floor", filepath.Base(f), dark, r)
			}
			if huh := HuhStyles(th).Blurred.Base.GetBorderLeftForeground(); th.G.BlurBar != "" && !sameColor(huh, ink) {
				t.Errorf("%s dark=%v: Huh's blurred field bar draws %v, want BlurBar's %v", filepath.Base(f), dark, huh, ink)
			}
		}
	}
}

// Case reached section and dialog titles, table headers and tags, but not the kit's
// own words: ui.Tabs never called t.Case (Services 8  Deploys beside lowercase labels),
// ui.TooSmall hard-coded "Terminal too small", and a tool call's "Canceled." and
// "Waiting for permission" and Thinking's "Thought for" were fixed. Under
// labelCase/titleCase lower they read as a second brand. The kit's fixed copy takes
// labelCase when the theme sets one, else titleCase (t.Copy); the data after it (an
// elapsed time, a size) stays as it is.
func TestFixedCopyFollowsTheme(t *testing.T) {
	for _, c := range []struct {
		name, titleCase, labelCase                        string
		tabs, small, canceled, waiting, thought, thinking string
	}{
		{"quire", "sentence", "lower", "services / deploys", "terminal too small", "canceled.", "waiting for permission", "thought for 6.1s", ""},
		{"default", "sentence", "", "Services / Deploys", "Terminal too small", "Canceled.", "Waiting for permission", "Thought for 6.1s", ""},
		{"upper titles", "upper", "", "SERVICES / DEPLOYS", "TERMINAL TOO SMALL", "CANCELED.", "WAITING FOR PERMISSION", "THOUGHT FOR 6.1s", ""},
		{"upper titles, lower labels", "upper", "lower", "services / deploys", "terminal too small", "canceled.", "waiting for permission", "thought for 6.1s", ""},
	} {
		f := quire(t)
		f.Expressions.TitleCase, f.Expressions.LabelCase = c.titleCase, c.labelCase
		th := theme.Resolve(f, true, false)
		plain := func(rows []string) string { return strings.TrimSpace(ansi.Strip(strings.Join(rows, "\n"))) }
		got := map[string]string{
			"tabs":     plain(Tabs(th, []string{"Services", "Deploys"}, 0, nil)),
			"canceled": plain(ToolCall(th, Tool{Name: "Edit", State: ToolCanceled}, 60, false)[2:]),
			"waiting":  strings.TrimSuffix(plain(ToolCall(th, Tool{Name: "Edit", State: ToolPending}, 60, false)[2:]), th.G.Ellipsis),
			"thought":  plain(Thinking(th, []string{"a"}, 2, "6.1s", 60)[1:]),
		}
		for what, want := range map[string]string{"tabs": c.tabs, "canceled": c.canceled, "waiting": c.waiting, "thought": c.thought} {
			if got[what] != want {
				t.Errorf("%s: %s %q, want %q", c.name, what, got[what], want)
			}
		}
		if small := ansi.Strip(TooSmall(th, 52, 16, 60, 18)); !strings.Contains(small, c.small) || !strings.Contains(small, "52x16, needs 60x18") {
			t.Errorf("%s: TooSmall draws %q, want %q and the sizes", c.name, strings.TrimSpace(small), c.small)
		}
	}
}
