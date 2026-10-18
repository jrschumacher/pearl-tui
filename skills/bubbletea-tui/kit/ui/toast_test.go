package ui

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"tuikit/theme"
)

// Each kind of toast has its own length (references/motion.md § Budget), held on the
// one frame clock: a plain toast ~4 s, an undo toast ~8 s, a grace window ~5 s. The
// showcase had one TTL (frame + 80), so an undo toast left before the hand reached
// its key.
func TestToastLengths(t *testing.T) {
	for _, c := range []struct {
		d      time.Duration
		frames int
	}{{ToastTTL, 80}, {UndoTTL, 160}, {GraceTTL, 100}} {
		if got := Frames(c.d); got != c.frames {
			t.Errorf("Frames(%s) = %d, want %d at %s a frame", c.d, got, c.frames, FrameInterval)
		}
	}
	if !(ToastTTL < GraceTTL && GraceTTL < UndoTTL) {
		t.Errorf("toast %s, grace %s, undo %s: want plain < grace < undo", ToastTTL, GraceTTL, UndoTTL)
	}
}

// A countdown reads whole seconds rounded up: "in 5s" on the first frame of a 5 s
// window, never "in 0s" while the send is still held.
func TestSecondsLeft(t *testing.T) {
	for _, c := range []struct{ frames, want int }{
		{Frames(GraceTTL), 5}, {Frames(GraceTTL) - 1, 5}, {Frames(4 * time.Second), 4}, {1, 1}, {0, 0}, {-3, 0},
	} {
		if got := SecondsLeft(c.frames); got != c.want {
			t.Errorf("SecondsLeft(%d) = %d, want %d", c.frames, got, c.want)
		}
	}
}

// A toast that offers a key names it whole at every width: the message is cut first,
// never the key, so the footer row never advertises "u und…" or drops the key the
// action table binds. It keeps the row's exact width and draws every cell on the bar.
func TestToastKeyNamesItsKey(t *testing.T) {
	b := key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "undo"))
	long := "Paging payments on-call for INC-2041 checkout-api 502s across three regions in 5s"
	files := append(testThemes(t), filepath.Join("..", "theme", "testdata", "quire.json"))
	for _, f := range files {
		tf, err := theme.Load(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, ascii := range []bool{false, true} {
			th := theme.Resolve(tf, true, ascii)
			for _, w := range []int{60, 80, 120} {
				for _, msg := range []string{"Silenced checkout-api alerts for 1h", long} {
					row := ToastKey(th, "warning", "HELD", msg, b, w)
					plain := ansi.Strip(row)
					name := filepath.Base(f) + " ascii=" + map[bool]string{false: "no", true: "yes"}[ascii]
					if got := lipgloss.Width(row); got != w {
						t.Errorf("%s w=%d: toast is %d cells: %q", name, w, got, plain)
					}
					hint := " " + th.G.Sep + " u undo"
					if !strings.Contains(plain, hint) {
						t.Errorf("%s w=%d: toast %q does not name %q whole", name, w, plain, hint)
					}
					if msg == long && w < 120 && !strings.Contains(plain, th.G.Ellipsis) {
						t.Errorf("%s w=%d: a long message should be cut with %q: %q", name, w, th.G.Ellipsis, plain)
					}
					if strings.Index(plain, hint) < strings.Index(plain, msg[:8]) {
						t.Errorf("%s w=%d: the key comes before the message: %q", name, w, plain)
					}
				}
			}
		}
	}
}

// With no key the toast is the plain one: ToastKey with an unset binding is Toast.
func TestToastKeyWithNoKeyIsToast(t *testing.T) {
	th := testTheme(t, false)
	if a, b := ToastKey(th, "ok", "DONE", "Rolled back", key.Binding{}, 80), Toast(th, "ok", "DONE", "Rolled back", 80); a != b {
		t.Errorf("ToastKey with no key:\n%q\nToast:\n%q", a, b)
	}
}

// tickN runs n frames of the clock, uncovered, and returns every command it gave back.
func tickN(q *Toasts, n int) []tea.Cmd {
	var out []tea.Cmd
	for range n {
		if c := q.Tick(false); c != nil {
			out = append(out, c)
		}
	}
	return out
}

type sentMsg struct{}

// grace is a grace window whose send is a real command (it returns sentMsg) and raises
// its confirmation, and whose undo raises its own.
func grace(q *Toasts, ran *int) Note {
	u := key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "undo"))
	return Note{Kind: "warning", Label: "HELD", Msg: "Paging payments on-call", For: GraceTTL, Key: u,
		Act: func() tea.Cmd { q.Raise(Note{Label: "CANCELED", Msg: "Nobody was paged"}); return nil },
		Send: func() tea.Cmd {
			*ran++
			q.Raise(Note{Label: "SENT", Msg: "Declared INC-2041"})
			return func() tea.Msg { return sentMsg{} }
		}}
}

// A grace window's send is a command, and it comes back out exactly once: when the window
// ends (Tick), or on quit (Flush), never when another note is raised, and never once
// undone. A send whose command is dropped never went (review of #76: raise() threw the
// command away when a toast replaced the window, while the row said it went).
func TestGraceWindowSendsOnce(t *testing.T) {
	isSent := func(cs []tea.Cmd) int {
		n := 0
		for _, c := range cs {
			if _, ok := c().(sentMsg); ok {
				n++
			}
		}
		return n
	}
	// the window ends: one command, the send, and its confirmation shows
	var q Toasts
	ran := 0
	if c := q.Raise(grace(&q, &ran)); c != nil {
		t.Fatal("raising a window returned a command: nothing was sent yet")
	}
	if !q.Holding() || q.Text() != "Paging payments on-call in 5s" {
		t.Fatalf("a fresh window reads %q (holding %v), want the countdown from 5s", q.Text(), q.Holding())
	}
	tickN(&q, Frames(time.Second))
	if q.Text() != "Paging payments on-call in 4s" {
		t.Errorf("a second in, the window reads %q", q.Text())
	}
	cmds := tickN(&q, Frames(GraceTTL))
	if ran != 1 || q.Sends() != 1 || isSent(cmds) != 1 {
		t.Fatalf("the window ended: send ran %d times, returned %d send commands, want 1 and 1", ran, isSent(cmds))
	}
	if n, _ := q.Current(); n.Label != "SENT" {
		t.Errorf("after the window the row shows %q, want the send's confirmation", n.Label)
	}

	// undo inside the window: it never runs, not at the end, not on quit
	q, ran = Toasts{}, 0
	q.Raise(grace(&q, &ran))
	tickN(&q, Frames(time.Second))
	if c := q.Act(); c != nil {
		t.Error("undo returned a command")
	}
	cmds = append(tickN(&q, 3*Frames(GraceTTL)), q.Flush())
	if ran != 0 || q.Sends() != 0 || isSent(cmds[:len(cmds)-1]) != 0 {
		t.Fatalf("undone inside the window, and the send ran %d times", ran)
	}
	if n, ok := q.Current(); ok && n.Label != "CANCELED" {
		t.Errorf("after undo the row shows %q", n.Label)
	}

	// quit inside the window: Flush returns the send, once
	q, ran = Toasts{}, 0
	q.Raise(grace(&q, &ran))
	if c := q.Flush(); c == nil || isSent([]tea.Cmd{c}) != 1 || ran != 1 {
		t.Fatalf("Flush inside the window returned %v and ran the send %d times, want its command once", c, ran)
	}
	if c := q.Flush(); c != nil || ran != 1 {
		t.Error("a second Flush ran the send again")
	}
}

// A grace window keeps the row until undo or its end (review of #76: any toast cut it
// short and sent at once, and its "Declared" confirmation was never seen). A note raised
// meanwhile waits, the latest one only, and shows after the window's confirmation; a
// second window raised while one holds is undone at once, its send never run.
func TestGraceWindowIsNeverCutShort(t *testing.T) {
	var q Toasts
	ran := 0
	q.Raise(grace(&q, &ran))
	tickN(&q, Frames(time.Second))
	second := 0
	q.Raise(grace(&q, &second)) // refused: its undo raises "Nobody was paged", which waits
	if second != 0 {
		t.Fatal("a second window raised while one holds ran its send")
	}
	q.Raise(Note{Label: "HISTORY", Msg: "first"})
	q.Raise(Note{Label: "HISTORY", Msg: "No incidents in the last 30 days"})
	if n, _ := q.Current(); n.Label != "HELD" || ran != 0 {
		t.Fatalf("a toast raised in the window replaced it (%q) or sent it (%d)", n.Label, ran)
	}
	tickN(&q, Frames(GraceTTL)-Frames(time.Second))
	if n, _ := q.Current(); n.Label != "SENT" || ran != 1 {
		t.Fatalf("at the window's end the row shows %q (sent %d), want SENT once", n.Label, ran)
	}
	tickN(&q, Frames(ToastTTL))
	if n, _ := q.Current(); n.Msg != "No incidents in the last 30 days" {
		t.Errorf("after the confirmation the row shows %q, want the latest note that waited", n.Msg)
	}
	if second != 0 {
		t.Error("the refused second window sent")
	}
}

// While a dialog covers the row, a note that offers a key waits: a window never sends,
// and an undo never lapses, while its key is out of reach (review of #76: help over the
// window let it run out unseen). A plain note runs on.
func TestToastWaitsUnderADialog(t *testing.T) {
	var q Toasts
	ran := 0
	q.Raise(grace(&q, &ran))
	for range 3 * Frames(GraceTTL) {
		q.Tick(true)
	}
	if ran != 0 || !q.Holding() || q.Text() != "Paging payments on-call in 5s" {
		t.Fatalf("covered, the window ran on: sent %d, reads %q", ran, q.Text())
	}
	var p Toasts
	p.Raise(Note{Msg: "Rolled back"})
	for range Frames(ToastTTL) {
		p.Tick(true)
	}
	if _, ok := p.Current(); ok {
		t.Error("a plain note under a dialog outlived ToastTTL")
	}
}
