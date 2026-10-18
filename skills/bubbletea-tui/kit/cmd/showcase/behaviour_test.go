package main

import (
	"path/filepath"
	"strconv"
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
// its handler. This is the test that keeps help honest.
func TestHelpIsHandled(t *testing.T) {
	for _, s := range states {
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
