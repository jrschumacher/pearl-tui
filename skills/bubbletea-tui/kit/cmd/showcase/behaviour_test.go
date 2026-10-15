package main

import (
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
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
		"space": tea.KeySpace, "pgup": tea.KeyPgUp, "pgdown": tea.KeyPgDown}
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
	f, err := theme.Load("../../themes/charm.json")
	if err != nil {
		t.Fatal(err)
	}
	m := newApp(f, true, false)
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
	if txt := ansi.Strip(m.render(100, 30).String()); !strings.Contains(txt, "Roll back "+want+"?") {
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
}

// ASCII mode emits ASCII, whatever a screen or a fixture typed.
func TestASCIIIsASCII(t *testing.T) {
	f, _ := theme.Load("../../themes/halyard.json")
	m := newApp(f, true, true)
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
	files, _ := filepath.Glob("../../themes/*.json")
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
	for range 3 {
		m.key(press("j"))
		want := m.events[m.depSel].Svc
		if got := m.svcs[m.sel].Name; got != want {
			t.Fatalf("deploys highlights %q but actions target %q", want, got)
		}
		if txt := ansi.Strip(m.render(120, 40).String()); !strings.Contains(strings.SplitN(txt, "\n", 2)[0], want) {
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
