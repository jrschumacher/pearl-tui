package main

import (
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"charm.land/bubbles/v2/key"
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
		"space": tea.KeySpace, "pgup": tea.KeyPgUp, "pgdown": tea.KeyPgDown, "home": tea.KeyHome, "end": tea.KeyEnd}
	for i := range 12 { // f1 … f12: the help key where ? would be typed
		named["f"+strconv.Itoa(i+1)] = tea.KeyF1 + rune(i)
	}
	if r, ok := named[k]; ok {
		return tea.KeyPressMsg{Code: r, Mod: mod}
	}
	r := []rune(k)[0]
	if mod != 0 {
		return tea.KeyPressMsg{Code: r, Mod: mod}
	}
	return tea.KeyPressMsg{Code: r, Text: k}
}

// press names every key an action table may bind, the function keys among them: f1 is
// the spec's help key where ? would be typed, and press("f1") was the letter f with the
// text "f1", a key no terminal sends.
func TestPressNamesKeys(t *testing.T) {
	for _, k := range []string{"f1", "f5", "f12", "shift+f2", "ctrl+f3", "enter", "ctrl+j", "shift+tab", "?", "q"} {
		if got := press(k).String(); got != k {
			t.Errorf("press(%q) is %q", k, got)
		}
	}
	for i := range 12 {
		if p := press("f" + strconv.Itoa(i+1)); p.Code != tea.KeyF1+rune(i) || p.Text != "" {
			t.Errorf("press(\"f%d\") is code %d text %q, want tea.KeyF%d", i+1, p.Code, p.Text, i+1)
		}
	}
}

func fixture(t *testing.T, state string) *app {
	t.Helper()
	m := newApp(testTheme(t, "charm.json"), true, false)
	m.setState(state)
	return m
}

// The footer may only show keys that do something in that state. Every key
// of every action (footer or not) is pressed on a fresh model and must reach
// its handler. This is the test that keeps help honest. A toast that offers a key
// (u undo) after any of them has that key in actions() too: the row names it.
func TestHelpIsHandled(t *testing.T) {
	for _, s := range states {
		for _, a := range fixture(t, s).actions() {
			for _, k := range a.b.Keys() {
				m := fixture(t, s)
				m.key(press(k))
				if m.last != a.b.Help().Desc {
					t.Errorf("%s: %q (%s) not handled: last action %q", s, k, a.b.Help().Desc, m.last)
				}
				if ta := m.toastAction(); ta != nil && !inActions(m, ta.b.Help()) {
					t.Errorf("%s then %q: the toast offers %s %s, which actions() doesn't hold", s, k, ta.b.Help().Key, ta.b.Help().Desc)
				}
			}
		}
	}
}

// inActions reports whether the state's action table holds a binding with help h.
func inActions(m *app, h key.Help) bool {
	return slices.ContainsFunc(m.actions(), func(a action) bool { return a.b.Help() == h })
}

// toastRow is the footer row as drawn at w×h, where a toast stands while it shows.
func toastRow(m *app, w, h int) string {
	return rowText(m.render(w, h), h-1)
}

// pressToastKeys checks the toast m shows (if it offers a key): the footer row names
// the key and what it does, the state's action table binds it, every one of its keys,
// pressed on a fresh copy of the state, reaches the toast's handler, and a dialog over
// it (help) takes the key away: under a dialog u would act behind it, unseen. fresh
// rebuilds the state the toast was raised in, from s.
func pressToastKeys(t *testing.T, s, name string, m *app, fresh func() *app) {
	t.Helper()
	a := m.toastAction()
	if a == nil {
		return
	}
	h := a.b.Help()
	for _, sz := range [][2]int{{minW, minH}, {80, 24}, {120, 40}} {
		if row := toastRow(m, sz[0], sz[1]); !strings.Contains(row, h.Key+" "+h.Desc) {
			t.Errorf("%s %dx%d: the toast offers %s %s but its row reads %q", name, sz[0], sz[1], h.Key, h.Desc, row)
		}
	}
	if !inActions(m, h) {
		t.Errorf("%s: the toast names %s %s, which the state's action table doesn't hold", name, h.Key, h.Desc)
	}
	for _, k := range a.b.Keys() {
		f := fresh()
		f.key(press(k))
		if f.last != h.Desc {
			t.Errorf("%s: %q, which the toast names (%s), reached %q", name, k, h.Desc, f.last)
		}
	}
	f := fresh()
	f.key(press(helpKey(t, s)))
	if f.ov == ovNone {
		return // the help key was the toast's: nothing covers it
	}
	if f.toastAction() != nil || inActions(f, h) {
		t.Errorf("%s: under a dialog the toast still offers %s %s, which would act behind it", name, h.Key, h.Desc)
	}
}

// toastKeysOffered presses every key of every state and checks each toast that then
// offers a key (pressToastKeys). It returns how many it reached.
func toastKeysOffered(t *testing.T) int {
	offered := 0
	for _, s := range states {
		if s == "toosmall" {
			continue
		}
		m := fixture(t, s)
		if m.toastAction() != nil {
			offered++
			pressToastKeys(t, s, s, m, func() *app { return fixture(t, s) })
		}
		for _, a := range m.actions() {
			k := a.b.Keys()[0]
			raise := func() *app { f := fixture(t, s); f.key(press(k)); return f }
			if r := raise(); r.toastAction() != nil && r.ov == ovNone {
				offered++
				pressToastKeys(t, s, s+" then "+k, r, raise)
			}
		}
	}
	return offered
}

// Law 6 for the one key a toast names: every key a toast offers (u undo, u unsilence)
// is in the state's action table while the toast shows, its row names it at every size
// down to the floor, pressing it does what the row says, and a dialog over the toast
// takes it away. An app whose toasts offer no key passes.
func TestToastKeysAreHandled(t *testing.T) { toastKeysOffered(t) }

// A grace window (a toast holding a send: a page, an email) holds it until undo or its
// end, wherever the app raises one (every key of every state is tried). Its key undoes
// it; quitting inside the window sends it first (Flush); no other key of the state ends
// it early; and no key that starts a window is offered while one holds, since
// ui.Toasts refuses a second (Raise returns false) and the app would have offered a key
// that does nothing. (ui's toast tests hold the clock, the sends and the refusal.)
func TestGraceWindowsHold(t *testing.T) {
	type window struct {
		name  string
		raise func() *app
	}
	var windows []window
	starts := map[string]bool{} // what the keys that open a window say they do
	for _, s := range states {
		if s == "toosmall" {
			continue
		}
		if fixture(t, s).toasts.Holding() { // a harness state that opens on a window
			windows = append(windows, window{s, func() *app { return fixture(t, s) }})
			continue
		}
		for _, a := range fixture(t, s).actions() {
			k := a.b.Keys()[0]
			raise := func() *app { m := fixture(t, s); m.key(press(k)); return m }
			if raise().toasts.Holding() {
				starts[a.b.Help().Desc] = true
				windows = append(windows, window{s + " then " + k, raise})
			}
		}
	}
	for _, w := range windows {
		m := w.raise()
		undo := m.toastAction()
		if undo == nil {
			t.Errorf("%s: a grace window offers no key to undo it", w.name)
			continue
		}
		for _, a := range m.actions() {
			if starts[a.b.Help().Desc] {
				t.Errorf("%s: %s %s, which opens a grace window, is offered while one holds: the second would be refused", w.name, a.b.Help().Key, a.b.Help().Desc)
			}
		}
		m.key(press(undo.b.Keys()[0]))
		if m.toasts.Holding() {
			t.Errorf("%s: %s didn't undo the window", w.name, undo.b.Keys()[0])
		}
		m = w.raise()
		if cmd := m.key(press("ctrl+c")); cmd == nil || m.toasts.Holding() {
			t.Errorf("%s: ctrl+c inside the window left the send held: it never goes", w.name)
		}
		for _, b := range w.raise().actions() {
			o := b.b.Keys()[0]
			if b.b.Help().Desc == "quit" || slices.Contains(undo.b.Keys(), o) {
				continue
			}
			f := w.raise()
			f.key(press(o))
			if !f.toasts.Holding() {
				t.Errorf("%s then %q: the window was cut short", w.name, o)
			}
		}
	}
}

// The too-small screen's key is the toosmall action table's, drawn from it: ui.TooSmall
// hard-coded "q quit" and TestHelpIsHandled skipped toosmall, so an app whose table
// bound only ctrl+c showed q quit, a help lie in a lint-clean frame. A table that shows
// no key (a spec whose too-small screen is its sentence and nothing else, ui.TooSmallText)
// draws no key row, and ctrl+c must still leave.
func TestTooSmallKeysAreTheTable(t *testing.T) {
	m := fixture(t, "toosmall")
	var want []string
	for _, a := range m.actions() {
		if a.show {
			want = append(want, a.b.Help().Key+" "+a.b.Help().Desc)
		}
	}
	if len(want) == 0 {
		cmd := m.key(press("ctrl+c"))
		if cmd == nil {
			t.Fatal("toosmall shows no key and ctrl+c does nothing: the screen has no way out")
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Fatal("toosmall shows no key and ctrl+c does not quit: the screen has no way out")
		}
		return
	}
	w, h := frameSize("toosmall", 80, 24)
	rows := strings.Split(ansi.Strip(m.render(w, h).String()), "\n")
	last := ""
	for _, r := range rows {
		if r = strings.TrimSpace(r); r != "" {
			last = r
		}
	}
	if got := strings.Join(strings.Fields(last), " "); got != strings.Join(want, " ") {
		t.Errorf("toosmall %dx%d: the key row reads %q, the toosmall action table shows %q", w, h, last, strings.Join(want, "  "))
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
