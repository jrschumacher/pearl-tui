package ui

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"tuikit/theme/bubbletea"
)

// testThemes finds the theme JSONs two levels up: in the kit (ui/bubbletea/../../themes)
// or in a project that copied ui/bubbletea to internal/ui and its theme to themes/
// (kit/README.md). Finding none fails: a glob that matches nothing would pass every test without running one.
func testThemes(t *testing.T) []string {
	t.Helper()
	if files, _ := filepath.Glob("../../themes/*.json"); len(files) > 0 {
		return files
	}
	t.Fatal("no theme JSON in ../../themes")
	return nil
}

// quirePath is the kit's focus "fg" test theme, in the theme package's testdata: beside
// ui/bubbletea in the kit (../../theme/bubbletea), beside internal/ui in a project that
// copied both packages flat (../theme).
func quirePath(t *testing.T) string {
	t.Helper()
	for _, p := range []string{
		filepath.Join("..", "..", "theme", "bubbletea", "testdata", "quire.json"),
		filepath.Join("..", "theme", "testdata", "quire.json"),
	} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	t.Fatal("no quire.json in ../../theme/bubbletea/testdata or ../theme/testdata")
	return ""
}

// testTheme is halyard in the kit, and the project's own theme after a copy.
func testTheme(t *testing.T, ascii bool) *theme.Theme {
	t.Helper()
	files := testThemes(t)
	path := files[0]
	for _, f := range files {
		if filepath.Base(f) == "halyard.json" {
			path = f
		}
	}
	f, err := theme.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	return theme.Resolve(f, true, ascii)
}

// Both spinners turn the brand's frames in the theme's working ink (S.Working:
// the focus ink, so fg under focus "fg", or the role expressions.spinner names),
// never Bubbles' Line or Huh's #F780E2 Dot. Every kit theme and quire (focus fg),
// each as written and with focus forced to fg: a project whose one theme sets
// focus fg failed this test out of the box when it compared against primary.
func TestSpinnersAreThemed(t *testing.T) {
	files := append(testThemes(t), quirePath(t))
	for _, f := range files {
		tf, err := theme.Load(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, focus := range []string{tf.Expressions.Focus, "fg"} {
			tf.Expressions.Focus = focus
			th := theme.Resolve(tf, true, false)
			name := filepath.Base(f) + " focus=" + focus
			bs := BubblesSpinner(th)
			if strings.Join(bs.Spinner.Frames, "") != strings.Join(th.G.Spinner, "") {
				t.Errorf("%s: bubbles spinner frames %v", name, bs.Spinner.Frames)
			}
			if v := bs.View(); v != th.S.Working.Render(th.G.Spinner[0]) {
				t.Errorf("%s: bubbles spinner renders %q, want the working ink %q", name, v, th.S.Working.Render(th.G.Spinner[0]))
			}
			hs := HuhSpinner(th, "Connecting to staging")
			v := hs.View()
			if !strings.HasPrefix(ansi.Strip(v), th.G.Spinner[0]+" Connecting") {
				t.Errorf("%s: huh spinner renders %q", name, ansi.Strip(v))
			}
			if got, want := firstInk(v), firstInk(th.S.Working.Render(th.G.Spinner[0])); !sameColor(got, want) {
				t.Errorf("%s: huh spinner frame is %v, want the working ink %v", name, got, want)
			}
			if strings.Contains(v, "247;128;226") || strings.Contains(ansi.Strip(v), "Loading") {
				t.Errorf("%s: huh spinner kept a default: %q", name, v)
			}
		}
	}
}

// A blurred input shows its value behind blank prompt cells; focused, the
// prompt glyph comes back in the same cells, so the text never moves.
func TestInputBlurredHasNoPrompt(t *testing.T) {
	th := testTheme(t, false)
	v := "payments"
	in := NewInput(th).Title("Namespace").Value(&v)
	in.WithWidth(40)
	rows := strings.Split(ansi.Strip(in.View()), "\n")
	last := rows[len(rows)-1]
	if strings.Contains(last, th.G.Prompt) || !strings.Contains(last, "payments") {
		t.Fatalf("blurred input row %q", last)
	}
	in.Focus()
	rows = strings.Split(ansi.Strip(in.View()), "\n")
	focused := rows[len(rows)-1]
	col := func(r string) int { return lipgloss.Width(r[:strings.Index(r, "payments")]) }
	if col(focused) != col(last) {
		t.Fatalf("focus moved the text: %q vs %q", focused, last)
	}
}

// Confirm puts the action first, aligned left, with the stated default.
func TestConfirmOrderAndDefault(t *testing.T) {
	th := testTheme(t, false)
	var v bool
	c := Confirm(th, "Delete it?", "Delete", "Keep", false, &v)
	f := NewForm(th, 60, 8, huh.NewGroup(c))
	f.Init()
	txt := ansi.Strip(f.View())
	row := ""
	for _, l := range strings.Split(txt, "\n") {
		if strings.Contains(l, "Delete") && strings.Contains(l, "Keep") {
			row = l
		}
	}
	if row == "" || strings.Index(row, "Delete") > strings.Index(row, "Keep") || strings.Index(row, "Delete") > 8 {
		t.Fatalf("button row %q: want Delete first, left in the field's column", row)
	}
	if v {
		t.Fatal("the default was false")
	}
}

type slowMsg struct{ n int }
type tickMsg struct{}
type quickMsg struct{}

// Settle delivers a command that answers in a few tens of milliseconds, and the command
// its message returns: under load a real command (Huh's NextField) took longer than the
// 5 ms Settle gave each one, was dropped as a timer, and the harness drew focus on the
// field before. A 500 ms tick (a cursor blink, a spinner) is still dropped, and Settle
// doesn't wait for it to fire: every command of a round runs at once against one window.
func TestSettleDeliversSlowCommandsAndDropsTimers(t *testing.T) {
	var got []tea.Msg
	sleep := func(n int) tea.Cmd {
		return func() tea.Msg { time.Sleep(20 * time.Millisecond); return slowMsg{n} }
	}
	tick := tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg { return tickMsg{} })
	update := func(msg tea.Msg) tea.Cmd {
		got = append(got, msg)
		if m, ok := msg.(slowMsg); ok && m.n == 1 {
			return sleep(2) // the next field's own command, as slow
		}
		return nil
	}
	start := time.Now()
	Settle(update, tea.Batch(tick, sleep(1), func() tea.Msg { return quickMsg{} },
		tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg { return tickMsg{} }),
		tea.Tick(time.Second, func(time.Time) tea.Msg { return tickMsg{} })))
	took := time.Since(start)
	want := []tea.Msg{slowMsg{1}, slowMsg{2}, quickMsg{}}
	for _, w := range want {
		if !slices.Contains(got, w) {
			t.Errorf("Settle dropped %T%v: delivered %v", w, w, got)
		}
	}
	if slices.Contains(got, tea.Msg(tickMsg{})) {
		t.Errorf("Settle delivered a 500 ms tick: %v", got)
	}
	// three timers in one round cost one window, not three, and never the tick's 500 ms
	if took > 300*time.Millisecond {
		t.Errorf("Settle took %v over three timers and two 20 ms commands: it waited on the timers", took)
	}
}
