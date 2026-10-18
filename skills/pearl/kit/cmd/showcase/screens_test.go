package main

// The showcase's own tests: its screens, fixtures and behaviours (the services list,
// deploys, the deploy form, the assistant, the ladder, the palette's commands). They
// name its states and data, so a copied app deletes this file with data.go and
// setup.go once its screens are in (kit/README.md); the harness tests it keeps
// (showcase_test.go, overlay_test.go, behaviour_test.go, ansi16_test.go) read the app's
// own tables: states, setState, overlays, overlayPages, openOverlay and the action tables.

import (
	"fmt"
	"image/color"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"

	"tuikit/theme"
	"tuikit/ui"
)

// The 16-colour tests (ansi16_test.go) also judge every status mark (ladder) and a
// brand title over a filled header (setup-signin), which only the showcase draws.
func init() { ansi16States = append(ansi16States, "ladder", "setup-signin") }

// The ladder's reading keeps its text under -ascii: a unit (21.5°C), a product (3×4)
// and a name with an accent (Zoë) read 21.5 C, 3x4 and Zoe, where the sweep drew "?"
// (TestASCIIKeepsTextRunes holds every state to no new "?"; this holds the fixture's
// words). The sample holds no ±: its one cell (+) would change the value.
func TestASCIIKeepsTheLaddersText(t *testing.T) {
	for _, f := range testThemes(t) {
		tf, err := theme.Load(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, sz := range [][2]int{{80, 24}, {120, 40}} {
			m := newApp(tf, true, false)
			m.setState("ladder")
			uni := ansi.Strip(m.render(sz[0], sz[1]).String())
			a := newApp(tf, true, true)
			a.setState("ladder")
			asc := ansi.Strip(a.render(sz[0], sz[1]).String())
			if !strings.Contains(uni, "21.5°C") {
				t.Fatalf("%s ladder %dx%d: no reading with a unit on the unicode frame", filepath.Base(f), sz[0], sz[1])
			}
			for _, want := range []string{"21.5 C", "3x4", "Zoe"} {
				if !strings.Contains(asc, want) {
					t.Errorf("%s ladder %dx%d: ASCII frame lacks %q:\n%s", filepath.Base(f), sz[0], sz[1], want, asc)
				}
			}
		}
	}
}

// A dialog is judged by its own cells (readPos), not the page's: the assistant's
// transcript under it is cut ("↑ n more", "… (n lines hidden)"), and a test that scanned
// the whole frame failed under every overlay of an app whose page has one. Each overlay
// over that cut transcript, every theme, at 80×24 and the floor, passes dialogFits.
func TestDialogsOverACutTranscript(t *testing.T) {
	for _, f := range testThemes(t) {
		tf, err := theme.Load(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, sz := range [][2]int{{80, 24}, {minW, minH}} {
			w, h := sz[0], sz[1]
			for _, s := range overlays {
				m := newApp(tf, true, false)
				m.screen, m.sent = scrAssistant, []string{"ship it as v2.4.1"}
				if full := ansi.Strip(m.render(w, h).String()); !strings.Contains(full, m.t.G.More+" ") || !strings.Contains(full, " more") {
					t.Fatalf("%s %s %dx%d: the transcript under the dialog has no cut row to be fooled by", filepath.Base(f), s, w, h)
				}
				m.openOverlay(s)
				dialogFits(t, fmt.Sprintf("%s %s over the cut transcript %dx%d", filepath.Base(f), s, w, h), m, s, w, h)
			}
		}
	}
}

// go test prints tabs; lipgloss measures one as 0 cells, so the assistant's go test body
// was fitted short and the canvas drew it with a cut ("0.087s   …") mid-panel. The kit
// expands a tab to its 8-column stop before fitting: every row of the body ends in its
// time, uncut, with the stops aligned.
func TestToolBodyTabsKeepTheirStops(t *testing.T) {
	for _, f := range testThemes(t) {
		tf, err := theme.Load(f)
		if err != nil {
			t.Fatal(err)
		}
		m := newApp(tf, true, false)
		m.setState("assistant")
		frame := ansi.Strip(m.render(120, 40).String())
		for _, want := range []string{"ok      checkout-api/internal/config    0.087s", "ok      checkout-api/internal/payments  0.412s"} {
			i := strings.Index(frame, want)
			if i < 0 {
				t.Errorf("%s assistant 120x40: no %q:\n%s", filepath.Base(f), want, frame)
				continue
			}
			row := frame[i:]
			if j := strings.IndexByte(row, '\n'); j >= 0 {
				row = row[:j]
			}
			if strings.Contains(row, m.t.G.Ellipsis) {
				t.Errorf("%s assistant 120x40: the go test row is cut: %q", filepath.Base(f), row)
			}
		}
	}
}

// Destructive actions act on the selection, not on whatever the fixture had first.
func TestActionsTargetSelection(t *testing.T) {
	m := fixture(t, "services")
	for range 5 {
		m.key(press("j"))
	}
	want := m.svcs[m.sel].Name
	m.key(press("r"))
	// any theme's titleCase (moss is lower) may set the dialog title's case
	if txt := strings.ToLower(ansi.Strip(m.render(100, 30).String())); !strings.Contains(txt, strings.ToLower("Roll back "+want+"?")) {
		t.Fatalf("confirm does not name the selected service %q", want)
	}
	m.key(press("enter")) // Cancel is the default
	if m.svcs[m.sel].Version != services[m.sel].Version {
		t.Fatal("enter on the default button must cancel, not roll back")
	}
	m.key(press("r"))
	m.key(press("left"))
	m.key(press("enter"))
	if m.svcs[m.sel].Version != services[m.sel].Prev {
		t.Fatalf("rollback did not change %s", want)
	}
}

// Typing in an input types: global single-letter keys yield to it.
func TestInputsTakeLetters(t *testing.T) {
	m := fixture(t, "assistant")
	m.input = ""
	for _, k := range "quit, ready" {
		m.key(press(string(k)))
	}
	if m.input != "quit, ready" || m.screen != scrAssistant {
		t.Fatalf("assistant input = %q screen = %v", m.input, m.screen)
	}
	p := fixture(t, "palette")
	p.query = ""
	for _, k := range "jk" {
		p.key(press(string(k)))
	}
	if p.query != "jk" {
		t.Fatalf("palette query = %q: j/k must type, arrows move", p.query)
	}
	d := fixture(t, "deploy")
	d.version = ""
	for _, k := range "v2é" {
		d.key(press(string(k)))
	}
	d.key(press("backspace"))
	if d.version != "v2" {
		t.Fatalf("deploy version = %q after backspace (rune-safe?)", d.version)
	}
}

// One clock: key presses don't start extra tick chains, and idle states stop.
func TestOneClock(t *testing.T) {
	m := fixture(t, "assistant-busy")
	if m.startClock() == nil {
		t.Fatal("busy assistant should animate")
	}
	for range 10 {
		if _, cmd := m.Update(press("pgup")); cmd != nil && m.tick.gen > 1 {
			t.Fatal("a key press started a second clock")
		}
	}
	idle := fixture(t, "incidents")
	if idle.startClock() != nil {
		t.Fatal("an idle screen must not tick")
	}
	stale := fixture(t, "assistant-busy")
	stale.startClock()
	if _, cmd := stale.Update(tickMsg{gen: 999}); cmd != nil {
		t.Fatal("a stale tick must die")
	}
	// a deploying row animates only where Busy does (Law 8: tick while something visible
	// moves): with a spinner wider than one cell the row holds the still partial mark,
	// in unicode and under ASCII alike, and the clock stays off. Both spinners are set
	// here: a copied project's one theme may have either (lumen's is three cells)
	for _, ascii := range []bool{false, true} {
		f := testTheme(t, "charm.json")
		f.Glyphs.Spinner = []string{"◜", "◠", "◝", "◞", "◡", "◟"}
		one := newApp(f, true, ascii)
		one.setState("services")
		if !one.hasDeploying() {
			t.Fatal("the services fixture has no deploying row")
		}
		if one.startClock() == nil {
			t.Errorf("ascii=%v: a deploying row with a one-cell spinner should tick", ascii)
		}
		f.Glyphs.Spinner = []string{"·  ", " · ", "  ·"}
		wide := newApp(f, true, ascii)
		wide.setState("services")
		if wide.startClock() != nil {
			t.Errorf("ascii=%v: a deploying row with a three-cell spinner draws a still mark, and ticked anyway", ascii)
		}
	}
}

// The product clock survives a setting change. The frame clock may let a stale tick
// die (TestOneClock): the next key press re-arms it. The list's refresh has no second
// chance, so a handler that changes its interval or pauses and resumes it returns the
// new arm cmd itself. The test delivers that cmd's tick: a copied app that only bumps
// the generation, or arms a tick stamped with the old one, never syncs again and fails.
func TestClockSurvivesSettingChange(t *testing.T) {
	saved := refreshRates
	refreshRates = []refreshRate{{time.Millisecond, "1ms"}, {2 * time.Millisecond, "2ms"}, {3 * time.Millisecond, "3ms"}}
	t.Cleanup(func() { refreshRates = saved })
	m := fixture(t, "services")
	m.refresh.rate = 0
	if m.armRefresh() == nil {
		t.Fatal("a live list should arm its refresh")
	}
	// deliver runs an arm cmd and feeds its tick back: it must sync once and re-arm
	deliver := func(what string, arm tea.Cmd) tea.Cmd {
		t.Helper()
		if arm == nil {
			t.Fatalf("%s returned no arm cmd: the pending tick dies and the list never syncs again", what)
		}
		before := m.refresh.syncs
		_, next := m.Update(arm())
		if m.refresh.syncs != before+1 || next == nil {
			t.Fatalf("%s: its tick should sync once and re-arm (synced %d, re-armed %v)", what, m.refresh.syncs-before, next != nil)
		}
		return next
	}
	for range refreshRates {
		pending := m.refresh.gen
		arm := m.command("Refresh rate")
		what := "a rate change (now " + refreshRates[m.refresh.rate].label + ")"
		if _, cmd := m.Update(refreshMsg{gen: pending}); cmd != nil || m.refresh.syncs != 0 {
			t.Fatalf("%s: the old interval's tick ran as well as the new one", what)
		}
		deliver(what+", then its own re-arm", deliver(what, arm))
		m.refresh.syncs = 0
	}
	// pause stops the clock; resume is a setting change too, and arms it again
	pending := m.refresh.gen
	if m.key(press("p")) != nil || !m.paused {
		t.Fatal("pause should stop the refresh")
	}
	if _, cmd := m.Update(refreshMsg{gen: pending}); cmd != nil || m.refresh.syncs != 0 {
		t.Fatal("a paused list synced or re-armed")
	}
	resume := m.key(press("p"))
	if m.paused {
		t.Fatal("resume should unpause the list")
	}
	deliver("resume", resume)
}

// The rate a toast names is the one the palette lists, as people write it: "every 1m",
// never time.Duration's "every 1m0s".
func TestRefreshRateToastNamesTheRate(t *testing.T) {
	m := fixture(t, "services")
	i := slices.IndexFunc(commands, func(c command) bool { return c.Title == "Refresh rate" })
	if i < 0 {
		t.Fatal("the palette has no Refresh rate command")
	}
	for range refreshRates {
		m.command("Refresh rate")
		r := refreshRates[m.refresh.rate]
		if n, ok := m.toasts.Current(); !ok || n.Msg != "every "+r.label {
			t.Fatalf("the toast for %s reads %+v, want \"every %s\"", r.every, n, r.label)
		}
		if strings.HasSuffix(r.label, "0s") && r.every >= time.Minute {
			t.Errorf("the %s label %q prints the zero seconds", r.every, r.label)
		}
		if !strings.Contains(commands[i].Desc, r.label) {
			t.Errorf("the palette's %q does not list the %s rate", commands[i].Desc, r.label)
		}
	}
}

// Every list selects by identity: the highlighted row's service is the one the
// header names and every action uses (the round-2 Deploys bug).
func TestSelectionIsIdentity(t *testing.T) {
	m := fixture(t, "deploys")
	for range len(m.events) - 1 { // every row, including the non-prod one
		m.key(press("j"))
		want := m.events[m.depSel].Svc
		if got := m.svcs[m.sel].Name; got != want {
			t.Fatalf("deploys highlights %q but actions target %q", want, got)
		}
		if !strings.Contains(header(m), want) {
			t.Fatalf("header does not name the selected service %q", want)
		}
	}
}

// The deploy form's review step is a ship's confirmation (interaction.md § Confirm By
// Consequence, rung 3): the button names the act and the target, and on prod the
// buttons start on Cancel, so enter, enter, enter… never ships. Staging starts on the act.
func TestDeployReviewIsTheConfirm(t *testing.T) {
	m := fixture(t, "deploy")
	for i := 0; i < 10 && m.screen == scrDeploy; i++ {
		m.key(press("enter"))
	}
	if m.screen == scrDeploy || m.svcs[m.sel].State == "deploying" {
		t.Fatalf("enter alone shipped to prod or never left the form (screen %d)", m.screen)
	}
	m = fixture(t, "deploy-review")
	if m.focus != 4 || m.formBtn != 1 {
		t.Fatal("deploy-review must render the prod review step with Cancel focused")
	}
	if out := strings.ToLower(ansi.Strip(m.render(120, 40).String())); !strings.Contains(out, strings.ToLower("Deploy "+m.version+" to prod")) {
		t.Fatal("the deploy button must name the act and where it lands")
	}
	// the act is drawn in danger ink while Cancel holds the focus, as a confirm's is
	c, found := m.render(120, 40), false
	for y, line := range strings.Split(strings.ToLower(ansi.Strip(c.String())), "\n") {
		x := strings.Index(line, "deploy "+m.version)
		if x < 0 {
			continue
		}
		found = true
		cell := c.CellAt(len([]rune(line[:x])), y)
		if cell == nil || cell.Style.Fg == nil {
			t.Fatal("a prod deploy's act button must be drawn in danger ink, not the default")
		}
		r1, g1, b1, _ := cell.Style.Fg.RGBA()
		r2, g2, b2, _ := m.t.P.Error.RGBA()
		if r1 != r2 || g1 != g2 || b1 != b2 {
			t.Fatal("a prod deploy's act button must be drawn in danger ink")
		}
	}
	if !found {
		t.Fatal("no act button on the deploy review step")
	}
	m.env, m.focus = 1, 3
	m.key(press("tab"))
	if m.focus != 4 || m.formBtn != 0 {
		t.Fatal("a staging deploy's buttons start on the act, the forward choice")
	}
}

// Enter in a field moves on; it never ships to prod. Deploys finish by message.
func TestDeployLifecycle(t *testing.T) {
	m := fixture(t, "deploy")
	m.key(press("enter"))
	if m.screen != scrDeploy || m.focus != 2 {
		t.Fatal("enter in the version field must move to the next field")
	}
	for m.focus != 4 {
		m.key(press("tab"))
	}
	// the review step is the confirmation: a prod deploy arrives on Cancel
	if m.formBtn != 1 {
		t.Fatal("a prod deploy's buttons must start on Cancel, so enter alone never ships")
	}
	m.key(press("right"))
	cmd := m.key(press("enter"))
	s := m.svcs[m.sel]
	if s.State != "deploying" || cmd == nil {
		t.Fatalf("deploy did not start (state %q)", s.State)
	}
	m.Update(deployDoneMsg{s.Name})
	if m.svcs[m.sel].State != "ok" {
		t.Fatal("deploy never finished")
	}
	m.Update(deployDoneMsg{"search-indexer"})
	m.toasts = ui.Toasts{}
	if m.animating() {
		t.Fatal("nothing is deploying and no toast: the clock must stop")
	}
}

// A colon or question mark typed into the composer is text, not a command.
func TestTypingNeverCommands(t *testing.T) {
	m := fixture(t, "assistant")
	m.input = ""
	for _, k := range "a:?q" {
		m.key(press(string(k)))
	}
	if m.input != "a:?q" || m.ov != ovNone {
		t.Fatalf("input %q overlay %v", m.input, m.ov)
	}
}

// Rolling back twice can't claim to go from a version to itself.
func TestRollbackOnce(t *testing.T) {
	m := fixture(t, "services")
	m.key(press("r"))
	m.key(press("left"))
	m.key(press("enter"))
	m.key(press("r"))
	if m.ov == ovConfirm {
		t.Fatal("second rollback opened a confirm with nothing to roll back to")
	}
}

// Law 6: a toggle's help names what the key does now. After one press the description
// must change (pause → resume, expand → collapse) and a second press must bring it back;
// a footer that still says "pause" while paused lies in one of its two states.
func TestToggleHelpIsTruthful(t *testing.T) {
	desc := func(m *app, k string) string {
		for _, a := range m.actions() {
			if slices.Contains(a.b.Keys(), k) {
				return a.b.Help().Desc
			}
		}
		return ""
	}
	for _, c := range []struct{ state, key string }{{"services", "p"}, {"assistant", "ctrl+o"}} {
		m := fixture(t, c.state)
		before := desc(m, c.key)
		m.key(press(c.key))
		after := desc(m, c.key)
		switch {
		case before == "":
			t.Errorf("%s: %s is not bound", c.state, c.key)
		case after == before:
			t.Errorf("%s: %s says %q before and after a press: a toggle's help names what the key does now", c.state, c.key, before)
		}
		m.key(press(c.key))
		if got := desc(m, c.key); got != before {
			t.Errorf("%s: %s says %q after two presses, want %q again", c.state, c.key, got, before)
		}
	}
}

// shape is a frame as layout: ANSI stripped, the focus and selection marks and the input
// caret blanked, so two frames that differ only in where focus sits compare equal.
func shape(m *app, w, h int) []string {
	g := m.t.G
	var pairs []string
	for _, mark := range []string{g.FocusBar, g.BlurBar, g.Cursor, g.Caret} {
		if mark != "" {
			pairs = append(pairs, mark, strings.Repeat(" ", lipgloss.Width(mark)))
		}
	}
	txt := strings.NewReplacer(pairs...).Replace(ansi.Strip(m.render(w, h).String()))
	return strings.Split(txt, "\n")
}

// tableRows are the rows of the first table in a frame (its header, rule and n rows), cut
// to the list's own width when a detail pane shares them.
func tableRows(rows []string, n, w int) []string {
	for i, r := range rows {
		if l := strings.ToLower(r); strings.Contains(l, "service") && strings.Contains(l, "version") {
			out := rows[i:min(len(rows), i+2+n)]
			if w >= wideW {
				for k, r := range out {
					out[k] = string([]rune(r)[:w-45])
				}
			}
			return out
		}
	}
	return nil
}

// Law 5: moving focus or the selection changes the marker column and the text's weight,
// never where anything sits. Every focus state of the deploy form, every palette row and
// every services and deploys row draws onto the same cells, marks and caret aside, in
// every theme (each selection and button expression) at both sizes.
func TestFocusKeepsLayout(t *testing.T) {
	for _, f := range testThemes(t) {
		tf, err := theme.Load(f)
		if err != nil {
			t.Fatal(err)
		}
		m := newApp(tf, true, false)
		name := filepath.Base(f)
		for _, sz := range [][2]int{{80, 24}, {120, 40}} {
			w, h := sz[0], sz[1]
			same := func(what string, base, got []string) {
				t.Helper()
				for i := range base {
					if i < len(got) && got[i] != base[i] {
						t.Errorf("%s %dx%d %s: row %d moved:\n%q\n%q", name, w, h, what, i, base[i], got[i])
						return
					}
				}
			}
			// the footer names the focused field's keys, so it is the one row allowed to change
			body := func(rows []string) []string { return rows[:h-3] }
			m.setState("deploy")
			base := body(shape(m, w, h))
			for focus := 1; focus < 5; focus++ {
				m.focus = focus
				same("deploy focus "+itoa(focus), base, body(shape(m, w, h)))
			}
			m.setState("palette")
			m.query = ""
			base = shape(m, w, h)
			for sel := 1; sel < len(m.filtered()); sel++ {
				m.palSel = sel
				same("palette row "+itoa(sel), base, shape(m, w, h))
			}
			for _, s := range []string{"services", "deploys"} {
				m.setState(s)
				n := len(m.svcs)
				if s == "deploys" {
					n = len(m.events)
				}
				base = tableRows(shape(m, w, h), n, w)
				if base == nil {
					t.Fatalf("%s %dx%d %s: no table", name, w, h, s)
				}
				for sel := 1; sel < n; sel++ {
					m.key(press("j"))
					same(s+" row "+itoa(sel), base, tableRows(shape(m, w, h), n, w))
				}
			}
		}
	}
}

// Law 9: a second pane joins the list at wideW columns and not below. At 80×24 the
// services and deploys lists, the deploy form and the assistant have the whole width; at
// 120×40 the detail pane, the preview and the sidebar appear.
func TestBreakpoints(t *testing.T) {
	for _, f := range testThemes(t) {
		tf, err := theme.Load(f)
		if err != nil {
			t.Fatal(err)
		}
		m := newApp(tf, true, false)
		for _, c := range []struct{ state, pane string }{
			{"services", "p99 latency"}, {"deploys", "p99 latency"}, {"deploy", "what happens"}, {"assistant", "touched"},
		} {
			m.setState(c.state)
			narrow := strings.ToLower(ansi.Strip(m.render(80, 24).String()))
			m.setState(c.state)
			wide := strings.ToLower(ansi.Strip(m.render(120, 40).String()))
			if strings.Contains(narrow, c.pane) {
				t.Errorf("%s %s at 80x24: the %q pane is there; it belongs above %d columns", filepath.Base(f), c.state, c.pane, wideW)
			}
			if !strings.Contains(wide, c.pane) {
				t.Errorf("%s %s at 120x40: no %q pane", filepath.Base(f), c.state, c.pane)
			}
		}
	}
}

// The palette's list scrolls: on a short terminal the selection stays on screen as it
// moves past the rows the list has (ui.Follow), and comes back when it moves up.
func TestPaletteScrollsToSelection(t *testing.T) {
	m := fixture(t, "palette")
	m.query = ""
	last := commands[len(commands)-1].Title
	first := commands[0].Title
	frame := func() string { return strings.ToLower(ansi.Strip(m.render(60, 18).String())) }
	if strings.Contains(frame(), strings.ToLower(last)) {
		t.Skipf("every command fits at 60x18: nothing to scroll")
	}
	for range len(commands) - 1 {
		m.key(press("down"))
	}
	if f := frame(); !strings.Contains(f, strings.ToLower(last)) {
		t.Fatalf("the selected last command is off screen:\n%s", f)
	}
	for range len(commands) - 1 {
		m.key(press("up"))
	}
	if f := frame(); !strings.Contains(f, strings.ToLower(first)) {
		t.Fatalf("the selected first command is off screen after scrolling back:\n%s", f)
	}
}

func sameColor(a, b color.Color) bool {
	if a == nil || b == nil {
		return a == b
	}
	r1, g1, b1, _ := a.RGBA()
	r2, g2, b2, _ := b.RGBA()
	return r1 == r2 && g1 == g2 && b1 == b2
}

// A blurred chat turn carries its bar in muted, never the focus ink: in assistant-busy
// the input is disabled, yet the user's turn drew a bold focus-coloured bar, a focus
// that wasn't there (under NO_COLOR and ASCII a bold | is the one focus cue). Every
// theme, both modes, ASCII too.
func TestBlurredTurnBarIsNotFocus(t *testing.T) {
	for _, f := range testThemes(t) {
		tf, err := theme.Load(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, mode := range []struct{ dark, ascii bool }{{true, false}, {false, false}, {true, true}} {
			m := newApp(tf, mode.dark, mode.ascii)
			m.setState("assistant-busy")
			c := m.render(80, 24)
			bars := 0
			for y := range 24 {
				row := ""
				for x := range 80 {
					if cell := c.CellAt(x, y); cell != nil {
						row += cell.Content
					}
				}
				if !strings.Contains(row, "ship it as v2.4.1") {
					continue
				}
				cell := c.CellAt(m.t.X.Gutter, y)
				if cell == nil || strings.TrimSpace(cell.Content) == "" {
					t.Errorf("%s dark=%v ascii=%v row %d: no turn bar before the user's text", filepath.Base(f), mode.dark, mode.ascii, y)
					continue
				}
				bars++
				if cell.Style.Attrs&uv.AttrBold != 0 {
					t.Errorf("%s dark=%v ascii=%v row %d: the blurred turn bar %q is bold", filepath.Base(f), mode.dark, mode.ascii, y, cell.Content)
				}
				if !sameColor(cell.Style.Fg, m.t.P.FgMuted) {
					t.Errorf("%s dark=%v ascii=%v row %d: the blurred turn bar draws %v, want fgMuted %v", filepath.Base(f), mode.dark, mode.ascii, y, cell.Style.Fg, m.t.P.FgMuted)
				}
			}
			if bars == 0 {
				t.Errorf("%s dark=%v ascii=%v: the user's turn is not on screen", filepath.Base(f), mode.dark, mode.ascii)
			}
		}
	}
}

// Where the warning is a fill (yellow 3 on a white page), only its mark, tags and badges
// take it: a degraded row's sparkline drawn as filled bars was a solid strip of black on
// yellow, the heaviest thing on the services screen, in the black of the light page's
// focus and selection (#59 critic). Its bars are drawn on the page, in an ink that is not
// the focus's, and its mark still sits against the last bar.
func TestFaintWarningSparklineStaysQuiet(t *testing.T) {
	for _, path := range testThemes(t) {
		tf, err := theme.Load(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, dark := range []bool{true, false} {
			for _, size := range [][2]int{{80, 24}, {120, 40}} {
				m, c := renderState(tf, dark, colorprofile.ANSI, "services", size[0], size[1])
				if theme.Contrast16(m.t.P.Warning, m.t.P.Bg) >= 3 {
					continue
				}
				warn, page := theme.Slot16(m.t.P.Warning), theme.Slot16(m.t.P.Bg)
				focus := theme.Slot16(m.t.S.Focus.GetForeground())
				rows := 0
				for y := range size[1] {
					// the degraded row's sparkline: the run of bar cells against a warning mark
					// drawn as a fill, after the row's own leading mark
					for x := size[0] - 1; x > 4; x-- {
						cell := c.CellAt(x, y)
						if cell == nil || cell.Content != m.t.G.Warn || slot16(cell.Style.Bg, m.t.P.Bg) != warn {
							continue
						}
						prev := c.CellAt(x-1, y)
						if prev == nil || !strings.ContainsAny(prev.Content, theme.SparkRamp) {
							continue
						}
						rows++
						for bx := x - 1; bx >= 0; bx-- {
							b := c.CellAt(bx, y)
							if b == nil || !strings.ContainsAny(b.Content, theme.SparkRamp) {
								break
							}
							if bg, fg := slot16(b.Style.Bg, m.t.P.Bg), slot16(b.Style.Fg, m.t.P.Fg); bg != page || fg == focus {
								t.Errorf("%s dark=%v %dx%d: sparkline bar %q at %d,%d draws ANSI %d on ANSI %d: want the page (%d) and not the focus's ink (%d)",
									filepath.Base(path), dark, size[0], size[1], b.Content, bx, y, fg, bg, page, focus)
								break
							}
						}
						break
					}
				}
				if rows == 0 {
					t.Errorf("%s dark=%v %dx%d: no degraded sparkline with its mark: the test reached nothing", filepath.Base(path), dark, size[0], size[1])
				}
			}
		}
	}
}

// advance runs the frame clock n frames (step, what each tick does before it re-arms)
// and runs every command they return, a grace window's send among them.
func advance(m *app, n int) {
	for range n {
		runCmd(m, m.step())
	}
}

// runCmd runs cmd as Bubble Tea would, batches and sequences included, and feeds each
// message back to the model, except quit. It is how a test sees a command that was
// returned rather than dropped. (Only for commands that answer at once: a timer's would
// wait out its time.)
func runCmd(m *app, cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	got := cmd()
	switch msg := got.(type) {
	case nil, tea.QuitMsg:
	case tea.BatchMsg:
		for _, c := range msg {
			runCmd(m, c)
		}
	default:
		if v := reflect.ValueOf(msg); v.Kind() == reflect.Slice && v.Type().Elem() == reflect.TypeOf(tea.Cmd(nil)) {
			for i := range v.Len() { // tea.Sequence's message: its commands, in order
				runCmd(m, v.Index(i).Interface().(tea.Cmd))
			}
			return
		}
		if _, isKey := msg.(tea.KeyPressMsg); !isKey {
			m.Update(msg) // its own command (a clock re-arm) is the clock's: advance drives it
		}
	}
}

// Every toast the showcase raises that offers a key is reached and its key works: the
// kept TestToastKeysAreHandled walks each state's keys, and this adds each palette
// command (Silence alerts raises the undo toast), and holds that some toast offers one.
func TestShowcaseToastKeysAreHandled(t *testing.T) {
	offered := toastKeysOffered(t)
	for _, c := range commands {
		raise := func() *app { f := fixture(t, "services"); f.command(c.Title); return f }
		if r := raise(); r.toastAction() != nil {
			offered++
			pressToastKeys(t, "services", "services then "+c.Title, r, raise)
		}
	}
	if offered == 0 {
		t.Fatal("no toast offers a key: the undo toast and the grace window are gone")
	}
}

// Silence is rung 1 of the confirm ladder: it happens on the command, the screen shows
// it, and an undo toast names the key that takes it back, for UndoTTL. The key leaves
// the action table when the toast goes; the palette then names the way back.
func TestSilenceUndoes(t *testing.T) {
	m := fixture(t, "services")
	name := m.svcs[m.sel].Name
	m.command("Silence alerts")
	if !m.silenced[name] {
		t.Fatal("Silence alerts didn't silence the selected service")
	}
	if n, ok := m.toasts.Current(); !ok || n.For != ui.UndoTTL {
		t.Fatalf("the silence toast %+v should stand for UndoTTL", n)
	}
	if row := toastRow(m, 80, 24); !strings.Contains(row, "Silenced "+name+" alerts for 1h") || !strings.Contains(row, "u unsilence") {
		t.Errorf("the silence toast reads %q", row)
	}
	m.key(press("u"))
	if m.silenced[name] {
		t.Fatal("u unsilence left the alerts silenced")
	}
	if m.toastAction() != nil {
		t.Error("the toast after an undo offers a key again")
	}
	// the undo key lives as long as its toast
	m.command("Silence alerts")
	advance(m, ui.Frames(ui.UndoTTL))
	if _, ok := m.toasts.Current(); ok || m.toastAction() != nil {
		t.Fatal("the undo toast outlived UndoTTL")
	}
	m.last = ""
	m.key(press("u"))
	if !m.silenced[name] || m.last == "unsilence" {
		t.Error("u still unsilenced after its toast had gone")
	}
	if !slices.ContainsFunc(m.filtered(), func(h hit) bool { return h.c.Title == "Unsilence alerts" }) {
		t.Error("with the toast gone the palette offers no way back: want \"Unsilence alerts\"")
	}
}

// A page is rung 2: `n declare incident` is the judgement, so no confirm, but the page
// waits out a grace window under an undo toast that counts down. The send is a command
// (page), observed by its message arriving: it runs once when the window ends, once on
// quit (q or ctrl+c), and never once undone. No other key cuts the window short (h's
// toast waits and shows after the confirmation), and help over it stops the clock.
func TestGraceWindow(t *testing.T) {
	press1 := func(m *app, k string) { runCmd(m, m.key(press(k))) }
	m := fixture(t, "incidents")
	press1(m, "n")
	if m.declared || m.pages != 0 {
		t.Fatal("the incident was declared at once: the page should wait out its grace window")
	}
	if row := toastRow(m, 80, 24); !strings.Contains(row, "Paging payments on-call in 5s") || !strings.Contains(row, "u undo") {
		t.Errorf("the grace toast reads %q", row)
	}
	advance(m, ui.Frames(time.Second))
	if row := toastRow(m, 80, 24); !strings.Contains(row, "in 4s") {
		t.Errorf("a second in, the grace toast reads %q: it should count down", row)
	}
	press1(m, "u")
	advance(m, 2*ui.Frames(ui.GraceTTL))
	press1(m, "ctrl+c")
	if m.declared || m.pages != 0 {
		t.Fatalf("u undo inside the grace window, and the page went anyway (%d pages)", m.pages)
	}

	// the window runs out: the page goes once, and the row says it went
	m = fixture(t, "incidents")
	press1(m, "n")
	advance(m, ui.Frames(ui.GraceTTL))
	if !m.declared || m.pages != 1 {
		t.Fatalf("the grace window ended: declared %v, %d pages, want one", m.declared, m.pages)
	}
	if row := toastRow(m, 80, 24); !strings.Contains(row, "Declared INC-2041") || !strings.Contains(row, "paged payments on-call") {
		t.Errorf("after the window the toast reads %q, want it to say the page went", row)
	}
	if m.toastAction() != nil {
		t.Error("the toast that says the page went still offers undo")
	}
	if slices.ContainsFunc(m.actions(), func(a action) bool { return a.b.Help().Desc == "declare incident" }) {
		t.Error("n declare incident is still offered with INC-2041 declared")
	}
	advance(m, 3*ui.Frames(ui.GraceTTL))
	press1(m, "q")
	if m.pages != 1 {
		t.Errorf("the page went %d times", m.pages)
	}

	// quitting inside the window sends it, once
	for _, k := range []string{"q", "ctrl+c"} {
		m = fixture(t, "incidents")
		press1(m, "n")
		press1(m, k)
		if !m.declared || m.pages != 1 {
			t.Errorf("%s during the grace window: declared %v, %d pages, want the page sent once", k, m.declared, m.pages)
		}
	}

	// another key doesn't cut it short: h's toast waits behind it and the confirmation
	m = fixture(t, "incidents")
	press1(m, "n")
	advance(m, ui.Frames(time.Second))
	press1(m, "h")
	if m.declared || m.pages != 0 || !strings.Contains(toastRow(m, 80, 24), "in 4s") {
		t.Fatalf("h a second into the window: declared %v, %d pages, row %q", m.declared, m.pages, toastRow(m, 80, 24))
	}
	advance(m, ui.Frames(ui.GraceTTL)-ui.Frames(time.Second))
	if m.pages != 1 || !strings.Contains(toastRow(m, 80, 24), "Declared INC-2041") {
		t.Fatalf("at the window's end: %d pages, row %q, want the page and its confirmation", m.pages, toastRow(m, 80, 24))
	}
	advance(m, ui.Frames(ui.ToastTTL))
	if !strings.Contains(toastRow(m, 80, 24), "No incidents in the last 30 days") {
		t.Errorf("after the confirmation the row reads %q, want h's toast, which waited", toastRow(m, 80, 24))
	}

	// wherever the toast's key isn't offered its clock waits: tab to Assistant, where the
	// input takes u as a letter, and the page doesn't go while u types (review of #76,
	// round 2: it went 5 s later with no key on the row); below the floor the same
	m = fixture(t, "incidents")
	press1(m, "n")
	for m.screen != scrAssistant {
		press1(m, "tab")
	}
	press1(m, "u")
	if m.input != "u" || m.toastAction() != nil {
		t.Fatalf("on Assistant u should be typed (input %q) and the toast offer no key", m.input)
	}
	advance(m, 3*ui.Frames(ui.GraceTTL))
	if m.declared || m.pages != 0 || !m.toasts.Holding() {
		t.Fatalf("the window ran out on Assistant, its u typed into the input: %d pages", m.pages)
	}
	m = fixture(t, "incidents")
	press1(m, "n")
	m.Update(tea.WindowSizeMsg{Width: minW - 1, Height: minH})
	advance(m, 3*ui.Frames(ui.GraceTTL))
	if m.declared || m.pages != 0 {
		t.Fatal("the window ran out below the floor, where no row shows it")
	}
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	if row := toastRow(m, 80, 24); !strings.Contains(row, "in 5s") || !strings.Contains(row, "u undo") {
		t.Errorf("back at 80x24 the row reads %q, want the window where it was", row)
	}

	// help over the window stops its clock: it never sends unseen, without its key
	m = fixture(t, "incidents")
	press1(m, "n")
	press1(m, "?")
	if m.ov != ovHelp || m.toastAction() != nil {
		t.Fatalf("help over the window: ov %v, toast key %v, want help and no toast key", m.ov, m.toastAction())
	}
	advance(m, 3*ui.Frames(ui.GraceTTL))
	if m.declared || m.pages != 0 {
		t.Fatal("the window ran out under help, where its undo key can't be reached")
	}
	press1(m, "esc")
	if row := toastRow(m, 80, 24); !strings.Contains(row, "in 5s") || !strings.Contains(row, "u undo") {
		t.Errorf("back from help the row reads %q, want the window where it was", row)
	}
}

// The state's own key wins a clash with a toast's key, and the toast then offers none:
// a toast whose key the state binds (an undo on h where incidents binds h history)
// names no key, and h does what the state's table says. With the toast's key first in
// actions() the row named "h undo" and h undid, while the footer said history.
func TestStateKeyWinsAClashWithAToast(t *testing.T) {
	m := fixture(t, "incidents")
	undone := false
	m.sayUndo("ok", "DONE", "Archived the postmortem", bind([]string{"h"}, "h", "unarchive"), func(*app) tea.Cmd { undone = true; return nil })
	if a := m.toastAction(); a != nil {
		t.Errorf("the toast offers %q, which the state binds", a.b.Keys())
	}
	if row := toastRow(m, 80, 24); strings.Contains(row, "unarchive") {
		t.Errorf("the toast row names a key the state binds: %q", row)
	}
	m.key(press("h"))
	if undone || m.last != "history" {
		t.Errorf("h undid the toast (%v) or reached %q, want the state's history", undone, m.last)
	}
}

// The services footer says how fresh the list is from the product clock: it read
// "synced 4s ago" whatever the rate, at 1s and at 1m. It names the rate the list syncs
// at, after every change, and "paused" while paused.
func TestFooterNamesTheRefreshRate(t *testing.T) {
	m := fixture(t, "services")
	for range refreshRates {
		m.command("Refresh rate")
		r := refreshRates[m.refresh.rate]
		got := ansi.Strip(m.footerRight())
		if !strings.Contains(got, "every "+r.label) || strings.Contains(got, "ago") {
			t.Errorf("at %s the footer's status reads %q, want the rate (every %s)", r.every, got, r.label)
		}
	}
	m.key(press("p"))
	if got := ansi.Strip(m.footerRight()); got != "paused" {
		t.Errorf("paused, the footer's status reads %q", got)
	}
}

// visibleNames are the service names the services table draws at w×h.
func visibleNames(m *app, w, h int) []string {
	txt := ansi.Strip(m.render(w, h).String())
	var out []string
	for _, s := range m.svcs {
		if strings.Contains(txt, " "+s.Name+" ") {
			out = append(out, s.Name)
		}
	}
	return out
}

// The filter pattern (#70: both cold monitors invented their own): / types a filter on
// the line above the list, the list narrows as it is typed and says how many it shows
// of how many, enter keeps it and gives the keys back, and esc clears it. esc clear
// filter is offered exactly while a filter is typed or in force. The selection stays
// on a row the filter shows, and the actions target it.
func TestFilterNarrowsTheList(t *testing.T) {
	has := func(m *app, desc string) bool {
		return slices.ContainsFunc(m.actions(), func(a action) bool { return a.b.Help().Desc == desc })
	}
	m := fixture(t, "services")
	if has(m, "clear filter") {
		t.Fatal("esc clear filter is offered with no filter")
	}
	m.key(press("/"))
	for _, k := range "er" {
		m.key(press(string(k)))
	}
	var want []string
	for _, s := range m.svcs {
		if strings.Contains(s.Name, "er") {
			want = append(want, s.Name)
		}
	}
	if got := visibleNames(m, 80, 30); !slices.Equal(got, want) {
		t.Errorf("filtering \"er\" shows %v, want %v", got, want)
	}
	if !strings.Contains(ansi.Strip(m.render(80, 30).String()), itoa(len(want))+" of "+itoa(len(m.svcs))) {
		t.Error("the filter line doesn't say how many rows it shows of how many")
	}
	if !strings.Contains(m.svcs[m.sel].Name, "er") {
		t.Errorf("the selection %q is a row the filter hides", m.svcs[m.sel].Name)
	}
	m.key(press("q")) // typed: a letter is text while the filter has the focus
	if m.filter != "erq" {
		t.Fatalf("q while typing the filter left it %q", m.filter)
	}
	m.key(press("backspace"))
	if !has(m, "clear filter") {
		t.Error("esc clear filter is not offered while the filter is typed")
	}
	m.key(press("enter"))
	if m.filtering || m.filter != "er" {
		t.Fatalf("enter should keep the filter and leave the input: filtering %v, filter %q", m.filtering, m.filter)
	}
	if !has(m, "clear filter") {
		t.Error("esc clear filter is not offered while the filter is in force")
	}
	m.key(press("j"))
	if !strings.Contains(m.svcs[m.sel].Name, "er") {
		t.Errorf("j moved the selection to %q, a row the filter hides", m.svcs[m.sel].Name)
	}
	m.key(press("esc"))
	if m.filter != "" || has(m, "clear filter") {
		t.Errorf("esc left the filter %q (clear filter offered: %v)", m.filter, has(m, "clear filter"))
	}
	if got := visibleNames(m, 80, 30); len(got) != len(m.svcs) {
		t.Errorf("with the filter cleared the list shows %d of %d", len(got), len(m.svcs))
	}
	// nothing left: the list says so, and the way out
	m.key(press("/"))
	for _, k := range "zz" {
		m.key(press(string(k)))
	}
	if txt := ansi.Strip(m.render(80, 24).String()); !strings.Contains(txt, "0 of "+itoa(len(m.svcs))) || !strings.Contains(txt, "No service") {
		t.Errorf("a filter that leaves nothing draws:\n%s", txt)
	}
}

// In a caret theme the input's prompt (») and the list's cursor (›) stand in one column,
// both in the focus ink, >> over > under ASCII (#56). Judged clear, on three conditions
// this holds: the two glyphs differ in both modes (theme.Validate refuses a prompt that
// is the cursor), they are never on adjacent rows (the palette keeps a blank row under
// its input, the filter line a row of air and the header), and a filter in force draws
// its prompt quiet (ui.TestFilterLine), so the focus ink marks two places only while
// both take keys: letters go to the query, ↑↓ to the selection.
func TestPromptNeverSitsOnTheCursor(t *testing.T) {
	sawColumn, caret := false, false
	for _, f := range testThemes(t) {
		tf, err := theme.Load(f)
		if err != nil {
			t.Fatal(err)
		}
		caret = caret || tf.Expressions.Selection == "caret"
		for _, ascii := range []bool{false, true} {
			for _, s := range []string{"palette", "services-filter", "services-filtered"} {
				for _, sz := range [][2]int{{80, 24}, {120, 40}} {
					m := newApp(tf, true, ascii)
					m.setState(s)
					c := m.render(sz[0], sz[1])
					name := fmt.Sprintf("%s ascii=%v %s %dx%d", filepath.Base(f), ascii, s, sz[0], sz[1])
					p, cur := m.t.G.Prompt, m.t.G.Cursor
					if p == cur && tf.Expressions.Selection == "caret" {
						t.Fatalf("%s: the prompt and the cursor are both %q", name, p)
					}
					for y := range sz[1] {
						x := slices.Index(strings.Split(rowText(c, y), ""), string([]rune(p)[0]))
						if x < 0 || !strings.HasPrefix(string([]rune(rowText(c, y))[x:]), p+" ") {
							continue
						}
						for _, dy := range []int{-1, 1} {
							if ny := y + dy; ny >= 0 && ny < sz[1] && cellText(c, x, ny) == cur {
								t.Errorf("%s: the prompt %q on row %d sits on the cursor %q on row %d:\n%s\n%s", name, p, y, cur, ny, rowText(c, min(y, ny)), rowText(c, max(y, ny)))
							}
						}
						for ny := y + 2; ny < min(sz[1], y+5); ny++ {
							sawColumn = sawColumn || (tf.Expressions.Selection == "caret" && cellText(c, x, ny) == cur)
						}
					}
				}
			}
		}
	}
	if caret && !sawColumn {
		t.Error("no caret theme drew its cursor in the prompt's column: this test no longer sees the case it guards")
	}
}

// No key acts on a row the filter hides, however the selection got there (review of
// #76). Tabbing away to Deploys moves the selection to a deploy's service, which the
// filter may hide: back on Services the selection is on a shown row again, so the
// header, r roll back and the palette name a row on screen. A filter that leaves no
// row selects none: the header and the side pane name no service, and no key (the
// palette's commands among them) reaches the hidden one, typed or in force.
func TestFilterNeverActsOnAHiddenRow(t *testing.T) {
	shownSel := func(m *app) bool { return slices.Contains(m.shown(), m.sel) }
	m := fixture(t, "services")
	m.key(press("/"))
	for _, k := range "ledg" {
		m.key(press(string(k)))
	}
	m.key(press("enter"))
	m.key(press("tab"))
	m.key(press("j"))
	m.key(press("j"))
	m.key(press("shift+tab"))
	if m.screen != scrServices || !shownSel(m) {
		t.Fatalf("back on Services after the deploys moved it, the selection is %q, which the filter %q hides", m.svcs[m.sel].Name, m.filter)
	}
	if head := ansi.Strip(m.render(80, 24).String()); !strings.Contains(strings.Split(head, "\n")[0], "ledger") {
		t.Errorf("the header names %q, not the one shown row", strings.Split(head, "\n")[0])
	}

	for _, typing := range []bool{true, false} {
		m := fixture(t, "services")
		m.key(press("/"))
		for _, k := range "zz" {
			m.key(press(string(k)))
		}
		if !typing {
			m.key(press("enter"))
		}
		if len(m.shown()) != 0 {
			t.Fatalf("\"zz\" leaves %d rows", len(m.shown()))
		}
		for _, a := range m.actions() {
			if d := a.b.Help().Desc; d == "commands" || d == "deploy" || d == "roll back" {
				t.Errorf("typing=%v: with no row shown, %s is bound, which acts on the hidden %q", typing, d, m.svcs[m.sel].Name)
			}
		}
		for _, sz := range [][2]int{{80, 24}, {120, 40}} {
			txt := ansi.Strip(m.render(sz[0], sz[1]).String())
			head := strings.Split(txt, "\n")[0]
			for _, s := range m.svcs {
				if strings.Contains(head, s.Name) {
					t.Errorf("typing=%v %dx%d: with no row shown the header names %q: %q", typing, sz[0], sz[1], s.Name, head)
				}
			}
			// the detail line (80) and the side pane (120) both draw owner and p99
			if strings.Contains(txt, "owner") || strings.Contains(txt, "p99 latency") {
				t.Errorf("typing=%v %dx%d: with no row shown the screen still draws a service's detail:\n%s", typing, sz[0], sz[1], txt)
			}
		}
		m.key(press("ctrl+p"))
		if m.ov == ovPalette {
			t.Errorf("typing=%v: ctrl+p opened the palette, whose commands act on the hidden %q", typing, m.svcs[m.sel].Name)
		}
	}
}
