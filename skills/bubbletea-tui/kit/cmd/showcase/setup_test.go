package main

import (
	"bytes"
	"image/color"
	"io"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"tuikit/theme"
	"tuikit/ui"
)

// drive presses keys the way a running program would deliver them: each
// key, then every message its commands produce (Huh moves between fields
// by Cmd, not in Update).
func drive(m *app, keys ...string) {
	for _, k := range keys {
		cmd := m.key(press(k))
		ui.Settle(func(msg tea.Msg) tea.Cmd { _, c := m.Update(msg); return c }, cmd)
	}
}

func screenText(m *app, w, h int) string { return ansi.Strip(m.render(w, h).String()) }

// header is the header's rows at 120×40: one for most themes, the wordmark rows
// plus the details row for a "banner" header.
func header(m *app) string {
	n := len(ui.Header(m.t, 120, 40, m.t.AppName(), ""))
	return strings.Join(strings.SplitN(screenText(m, 120, 40), "\n", n+1)[:n], "\n")
}

// The review's focused default is the expected forward choice, Continue,
// and enter on it signs in (the user's wizard defaulted to "Edit settings").
func TestSetupReviewDefaultsToContinue(t *testing.T) {
	m := fixture(t, "setup-review")
	if got := m.setup.acts.Selected(); got != "Continue" {
		t.Fatalf("review default = %q, want Continue", got)
	}
	if b := m.footerBindings(); b[0].Help().Desc != "continue" {
		t.Fatalf("footer says enter %q", b[0].Help().Desc)
	}
	drive(m, "enter")
	if m.setup.step != stSignin {
		t.Fatalf("enter on the review went to step %d, want sign-in", m.setup.step)
	}
}

// Buttons: the action first, the negative last, everywhere; and left in the
// field's text column, never centred under the title.
func TestSetupButtonOrder(t *testing.T) {
	for state, labels := range map[string][]string{
		"setup-review": {"Continue", "Edit", "Cancel"},
		"setup-alerts": {"Turn on", "Not now"},
		"setup-done":   {"Yes", "No"},
		"setup-failed": {"Retry", "Back"},
	} {
		for _, w := range []int{80, 120, 200} {
			m := fixture(t, state)
			var row string
			for _, l := range strings.Split(screenText(m, w, 40), "\n") {
				all := true
				for _, lb := range labels {
					all = all && strings.Contains(l, lb)
				}
				if all {
					row = l
				}
			}
			if row == "" {
				t.Fatalf("%s %d: no button row with %v", state, w, labels)
			}
			last := -1
			for _, l := range labels {
				i := strings.Index(row, l)
				if i <= last {
					t.Errorf("%s %d: %q out of order in %q", state, w, l, strings.TrimSpace(row))
				}
				last = i
			}
			// left-aligned: the row starts at the gutter, whatever the width
			if gut := strings.Index(row, m.t.G.FocusBar); gut < 0 || gut > 4 {
				t.Errorf("%s %d: buttons not in the field's column: %q", state, w, row)
			}
		}
	}
}

// The counter keeps counting past the review: sign-in and finishing are
// steps of their own, never "4/4" repeated on four screens.
func TestSetupCounterAdvancesPastReview(t *testing.T) {
	m := fixture(t, "setup-target")
	seen := map[string]bool{}
	step := func(want string) {
		t.Helper()
		h := header(m)
		if !strings.Contains(h, want) {
			t.Fatalf("header %q, want %q", strings.TrimSpace(h), want)
		}
		if seen[want] {
			t.Fatalf("counter repeated %q", want)
		}
		seen[want] = true
	}
	// the step names take the project theme's case (t.Copy)
	step("1/6 " + m.t.Copy("Cluster"))
	drive(m, "enter", "p", "a", "y", "enter")
	step("2/6 " + m.t.Copy("Sign-in"))
	drive(m, "down", "enter")
	step("3/6 " + m.t.Copy("Alerts"))
	drive(m, "enter")
	step("4/6 " + m.t.Copy("Review"))
	drive(m, "enter")
	step("5/6 " + m.t.Copy("Connect"))
	m.Update(signedInMsg{m.setup.gen})
	step("6/6 " + m.t.Copy("Finish"))
	if m.setup.namespace != "pay" || m.setup.auth != "device" || !m.setup.alerts {
		t.Fatalf("values lost on the way: %+v", *m.setup)
	}
	drive(m, "enter")
	if n, ok := m.toasts.Current(); m.screen != scrServices || !ok || !strings.Contains(n.Msg, "default cluster") {
		t.Fatalf("finishing did not land on services with a toast (screen %d)", m.screen)
	}
}

// Esc never silently drops typed input: on the first step it asks first;
// from step 2 on it goes back with every value kept.
func TestSetupEscKeepsInput(t *testing.T) {
	m := fixture(t, "setup-namespace")
	drive(m, "esc")
	if m.screen != scrSetup || m.setup.namespace != "payments" {
		t.Fatal("one esc with typed input left setup")
	}
	if n, ok := m.toasts.Current(); !ok || !strings.Contains(n.Label+n.Msg, "ESC AGAIN") {
		t.Fatal("esc with typed input must say esc again")
	}
	drive(m, "esc")
	if m.screen != scrServices {
		t.Fatal("second esc should leave setup")
	}

	a := fixture(t, "setup-alerts")
	drive(a, "esc", "esc")
	if a.setup.step != stTarget || a.setup.namespace != "payments" {
		t.Fatalf("back to step %d lost the namespace %q", a.setup.step, a.setup.namespace)
	}
	if !strings.Contains(screenText(a, 120, 40), "payments") {
		t.Fatal("the rebuilt first step does not show the typed namespace")
	}
}

// A canceled sign-in can't finish later.
func TestSetupStaleSignin(t *testing.T) {
	m := fixture(t, "setup-signin")
	gen := m.setup.gen
	drive(m, "esc")
	m.Update(signedInMsg{gen})
	if m.setup.step != stReview {
		t.Fatalf("a canceled sign-in moved the flow to step %d", m.setup.step)
	}
	s := fixture(t, "setup-signin")
	s.Update(signinTimeoutMsg{s.setup.gen})
	if !s.setup.failed || s.setup.acts.Selected() != "Retry" {
		t.Fatal("a timeout must fail the step with Retry focused")
	}
	drive(s, "enter")
	if s.setup.failed || s.setup.step != stSignin {
		t.Fatal("retry did not start a new sign-in")
	}
}

// The select cursor and an input prompt never show at once, and a blurred
// input draws no prompt glyph at all.
func TestSetupOneCursor(t *testing.T) {
	files := testThemes(t)
	for _, f := range files {
		tf, _ := theme.Load(f)
		m := newApp(tf, true, false)
		for _, s := range []string{"setup-target", "setup-namespace"} {
			m.setState(s)
			g := m.t.G
			m.render(120, 40) // lays the form out
			body := ansi.Strip(m.setup.form.View())
			cursors := strings.Count(body, g.Cursor+" ") // the select's selector
			prompts := strings.Count(body, g.Prompt+" ")
			if g.Cursor == g.Prompt {
				if cursors != 1 {
					t.Errorf("%s %s: %d cursor/prompt glyphs, want exactly 1", filepath.Base(f), s, cursors)
				}
			} else if cursors+prompts != 1 {
				t.Errorf("%s %s: %d cursors and %d prompts, want one in all", filepath.Base(f), s, cursors, prompts)
			}
		}
	}
}

// No Huh or Bubbles default colour reaches a frame: every colour on screen
// is a theme role (truecolor), never Huh's Charm palette, never an ANSI
// index from a Bubbles default style.
func TestSetupNoDefaultColours(t *testing.T) {
	huhDefaults := map[[3]uint8]string{{0xF7, 0x80, 0xE2}: "fuchsia", {0x75, 0x71, 0xF9}: "indigo", {0x5A, 0x56, 0xE0}: "indigo light"}
	check := func(name string, c color.Color) string {
		switch c.(type) {
		case nil:
			return ""
		case ansi.BasicColor, ansi.IndexedColor:
			return "an ANSI-indexed default colour"
		}
		r, g, b, _ := c.RGBA()
		if d, ok := huhDefaults[[3]uint8{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8)}]; ok {
			return "Huh's " + d
		}
		return ""
	}
	files := testThemes(t)
	for _, f := range files {
		tf, _ := theme.Load(f)
		for _, dark := range []bool{true, false} {
			m := newApp(tf, dark, false)
			for _, s := range states {
				if !strings.HasPrefix(s, "setup-") {
					continue
				}
				for _, sz := range [][2]int{{80, 24}, {120, 40}} {
					m.setState(s)
					c := m.render(sz[0], sz[1])
					b := c.Bounds()
				scan:
					for y := b.Min.Y; y < b.Max.Y; y++ {
						for x := b.Min.X; x < b.Max.X; x++ {
							cell := c.CellAt(x, y)
							if cell == nil {
								continue
							}
							for _, col := range []color.Color{cell.Style.Fg, cell.Style.Bg} {
								if why := check(s, col); why != "" {
									t.Errorf("%s dark=%v %s %dx%d: %s at row %d col %d (%q)", filepath.Base(f), dark, s, sz[0], sz[1], why, y, x, cell.Content)
									break scan
								}
							}
						}
					}
				}
			}
		}
	}
}

// lines hands a reader out one line per Read, like a terminal does. Huh's
// accessible prompts make a new bufio.Scanner per question, so a reader
// that returns everything at once feeds only the first one.
type lines struct{ rest []string }

func (l *lines) Read(p []byte) (int, error) {
	if len(l.rest) == 0 {
		return 0, io.EOF
	}
	n := copy(p, l.rest[0]+"\n")
	l.rest = l.rest[1:]
	return n, nil
}

// ACCESSIBLE=1 runs the same flow as plain prompts and status lines: no
// escape codes, no repainting, every step answered.
func TestAccessibleSetup(t *testing.T) {
	t.Setenv("ACCESSIBLE", "1")
	tf := testTheme(t, "halyard.json")
	th := theme.Resolve(tf, true, true)
	var out bytes.Buffer
	in := &lines{[]string{"1", "payments", "2", "n", "y"}}
	if err := runAccessibleSetup(th, in, &out, 0); err != nil {
		t.Fatal(err)
	}
	txt := out.String()
	if strings.Contains(txt, "\x1b[") {
		t.Errorf("accessible transcript has escape codes:\n%q", txt)
	}
	// the field titles take the project theme's case (t.Copy)
	for _, want := range []string{th.Copy("Cluster"), th.Copy("Namespace"), th.Copy("Sign in with"), "Signing in to prod-us-east-1", "WXKD-4PQM", "Saved. prod-us-east-1 is your default cluster."} {
		if !strings.Contains(txt, want) {
			t.Errorf("transcript lacks %q:\n%s", want, txt)
		}
	}
}

// The wizard's own copy follows the brand's case like the kit's: its step names and the
// Huh field titles go through t.Copy, so a labelCase lower brand (quire) draws
// "cluster", "sign-in", "namespace", and a sentence-case one keeps them as written. A
// title of several words is a label too: quire's "sign in with" once drew "Sign in with"
// beside "cluster" and "namespace", because t.Copy took any text with a space for a
// sentence.
func TestSetupCopyFollowsTheme(t *testing.T) {
	q, err := theme.Load(filepath.Join("..", "..", "theme", "testdata", "quire.json"))
	if err != nil {
		t.Skip("quire.json is the kit's; a copied project has its own themes")
	}
	for _, c := range []struct {
		name            string
		f               theme.File
		step, huh, auth string
	}{
		{"quire", q, "cluster", "namespace", "sign in with"},
		{"sentence", testTheme(t, "halyard.json"), "Cluster", "Namespace", "Sign in with"},
	} {
		if c.f.Expressions.LabelCase == "" && c.f.Expressions.TitleCase == "upper" {
			continue // a copied project's one theme may shout; the quire case still holds
		}
		a := newApp(c.f, true, false)
		a.setState("setup-auth")
		if frame := ansi.Strip(a.render(100, 30).String()); !strings.Contains(frame, c.auth) {
			t.Errorf("%s: setup-auth frame lacks %q in its siblings' case:\n%s", c.name, c.auth, frame)
		}
		m := newApp(c.f, true, false)
		m.setState("setup-namespace")
		frame := ansi.Strip(m.render(100, 30).String())
		for _, want := range []string{c.step, c.huh} {
			if !strings.Contains(frame, want) {
				t.Errorf("%s: setup frame lacks %q:\n%s", c.name, want, frame)
			}
		}
	}
}
