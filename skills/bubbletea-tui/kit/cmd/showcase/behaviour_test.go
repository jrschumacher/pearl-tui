package main

import (
	"image/color"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"

	"tuikit/theme"
)

// press builds the KeyPressMsg a terminal would send for a key name.
func press(k string) tea.KeyPressMsg {
	var mod tea.KeyMod
	if c, ok := strings.CutPrefix(k, "ctrl+"); ok {
		mod, k = tea.ModCtrl, c
	}
	if c, ok := strings.CutPrefix(k, "shift+"); ok {
		mod, k = tea.ModShift, c
	}
	named := map[string]rune{"enter": tea.KeyEnter, "esc": tea.KeyEscape, "tab": tea.KeyTab, "up": tea.KeyUp,
		"down": tea.KeyDown, "left": tea.KeyLeft, "right": tea.KeyRight, "backspace": tea.KeyBackspace,
		"space": tea.KeySpace, "pgup": tea.KeyPgUp, "pgdown": tea.KeyPgDown, "home": tea.KeyHome, "end": tea.KeyEnd}
	if r, ok := named[k]; ok {
		return tea.KeyPressMsg{Code: r, Mod: mod}
	}
	r := []rune(k)[0]
	if mod != 0 {
		return tea.KeyPressMsg{Code: r, Mod: mod}
	}
	return tea.KeyPressMsg{Code: r, Text: k}
}

func fixture(t *testing.T, state string) *app {
	t.Helper()
	m := newApp(testTheme(t, "charm.json"), true, false)
	m.setState(state)
	return m
}

// The footer may only show keys that do something in that state. Every key
// of every action (footer or not) is pressed on a fresh model and must reach
// its handler. This is the test that keeps help honest.
func TestHelpIsHandled(t *testing.T) {
	for _, s := range states {
		if s == "toosmall" {
			continue
		}
		for _, a := range fixture(t, s).actions() {
			for _, k := range a.b.Keys() {
				m := fixture(t, s)
				m.key(press(k))
				if m.last != a.b.Help().Desc {
					t.Errorf("%s: %q (%s) not handled: last action %q", s, k, a.b.Help().Desc, m.last)
				}
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
// new arm cmd itself; a copied app whose bumper only bumps the generation fails here.
func TestClockSurvivesSettingChange(t *testing.T) {
	m := fixture(t, "services")
	if m.armRefresh() == nil {
		t.Fatal("a live list should arm its refresh")
	}
	for range refreshRates {
		pending := m.refresh.gen
		if m.command("Refresh rate") == nil {
			t.Fatalf("a rate change (now %s) returned no arm cmd: the pending tick dies and the list never syncs again", refreshRates[m.refresh.rate])
		}
		if _, cmd := m.Update(refreshMsg{gen: pending}); cmd != nil {
			t.Fatal("the old interval's tick ran as well as the new one")
		}
		if n := m.refresh.syncs; n != 0 {
			t.Fatalf("the old interval's tick synced (%d)", n)
		}
	}
	if _, cmd := m.Update(refreshMsg{gen: m.refresh.gen}); cmd == nil || m.refresh.syncs != 1 {
		t.Fatalf("the current tick should sync and re-arm (synced %d, re-armed %v)", m.refresh.syncs, cmd != nil)
	}
	// pause stops the clock; resume is a setting change too, and arms it again
	pending := m.refresh.gen
	if m.key(press("p")) != nil || !m.paused {
		t.Fatal("pause should stop the refresh")
	}
	if _, cmd := m.Update(refreshMsg{gen: pending}); cmd != nil || m.refresh.syncs != 1 {
		t.Fatal("a paused list synced or re-armed")
	}
	if m.key(press("p")) == nil || m.paused {
		t.Fatal("resume should arm the refresh again")
	}
}

// ASCII mode emits ASCII, whatever a screen or a fixture typed.
func TestASCIIIsASCII(t *testing.T) {
	m := newApp(testTheme(t, "halyard.json"), true, true)
	for _, s := range states {
		m.setState(s)
		txt := ansi.Strip(m.render(100, 30).String())
		for i, r := range txt {
			if r > 127 {
				t.Errorf("%s: non-ASCII %q at byte %d", s, r, i)
				break
			}
		}
	}
}

// In ASCII the status marks stay distinct: ok, warning, error and idle must
// not collapse onto one character.
func TestASCIIStatusDistinct(t *testing.T) {
	files := testThemes(t)
	for _, f := range files {
		tf, _ := theme.Load(f)
		g := theme.Resolve(tf, true, true).G
		seen := map[string]string{}
		for name, v := range map[string]string{"current": g.Current, "warn": g.Warn, "fail": g.Fail, "todo": g.Todo} {
			if o, dup := seen[v]; dup {
				t.Errorf("%s: ASCII %s and %s are both %q", filepath.Base(f), o, name, v)
			}
			seen[v] = name
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
	m.toast = nil
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
