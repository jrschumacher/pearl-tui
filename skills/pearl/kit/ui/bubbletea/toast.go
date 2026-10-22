package ui

import (
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"tuikit/theme/bubbletea"
)

// How long each kind of toast stands over the footer row (references/motion.md
// § Budget). A plain toast is read and gone. An undo toast stays long enough for the
// hand to reach the key it names. A grace window holds a send that can't be recalled
// (a page, an email) long enough to catch the mistake and short enough that the send
// still goes out fast.
const (
	ToastTTL = 4 * time.Second
	UndoTTL  = 8 * time.Second
	GraceTTL = 5 * time.Second
)

// Frames is d on the frame clock (FrameInterval), rounded up: the frame a toast ends
// on is now + Frames(ToastTTL). The clock that animates is the one that ends toasts,
// so a test drives both with the same tick messages.
func Frames(d time.Duration) int {
	return int((d + FrameInterval - 1) / FrameInterval)
}

// SecondsLeft is a countdown's whole seconds for frames left on the clock, rounded up:
// "in 5 s" on a grace window's first frame, "in 1 s" on its last, never "in 0 s"
// while the send is still held.
func SecondsLeft(frames int) int {
	if frames <= 0 {
		return 0
	}
	per := int(time.Second / FrameInterval)
	return (frames + per - 1) / per
}

// toastStyles are the tag and the tinted bar of a toast of kind: ok ("" too), error,
// warning or info, in any case, and the names the kit's status slots use for them
// (done, fail, warn) or that people write (success, err, failed). Any other kind is
// info: a notice, never a success, so a mistyped kind can't paint a failure green.
func toastStyles(t *theme.Theme, kind string) (tag, bar lipgloss.Style) {
	switch strings.ToLower(kind) {
	case "", "ok", "success", "done":
		return t.S.TagOK, t.S.BarOK
	case "error", "err", "fail", "failed":
		return t.S.TagErr, t.S.BarErr
	case "warning", "warn":
		return t.S.TagWarn, t.S.BarWarn
	}
	return t.S.TagInfo, t.S.BarInfo
}

// hintGap reports whether sep, the theme's separator, misreads between a message and
// the key a toast offers: ".", "," (a sentence's end) or "/" (a rate).
func hintGap(sep string) bool { return sep == "." || sep == "," || sep == "/" }

// ToastKey is a Toast that offers one key while it shows: the message, the theme's
// separator, then the binding's key (bold) and what it does, in the bar's ink:
//
//	DONE  Silenced checkout-api alerts for 1h · u unsilence
//	HELD  Paging payments on-call in 5s · u undo
//
// Where the separator would misread before a key, a two-cell gap stands in for it, as
// between the footer's keys (hintGap): sentence punctuation (the ASCII twin of ·, ".")
// read "in 5s . u undo" as a sentence ending on its unit, and a slash read "in 5s / u
// undo" as a rate.
//
// Pass the binding the state's action table holds while the toast is up, so the row
// names exactly the key that works (an undo key the table doesn't bind is a lie). A
// narrow row cuts the message, never the key. With an unset binding it is Toast.
func ToastKey(t *theme.Theme, kind, label, msg string, k key.Binding, w int) string {
	h := k.Help()
	if h.Key == "" {
		return Toast(t, kind, label, msg, w)
	}
	g := t.X.Gutter
	tag, bar := toastStyles(t, kind)
	tg := tag.Render(t.Label(label, "as-is"))
	rest := max(0, w-2*g-lipgloss.Width(tg))
	in := bar.UnsetPadding()
	pl, pr := bar.GetPaddingLeft(), bar.GetPaddingRight()
	sep := " " + t.G.Sep + " "
	if hintGap(t.G.Sep) {
		sep = "  "
	}
	hint := sep + h.Key + " " + h.Desc
	room := max(0, rest-pl-pr-lipgloss.Width(hint))
	m := Trunc(t, msg, room)
	body := in.Render(strings.Repeat(" ", pl)+m+sep) + in.Bold(true).Render(h.Key) +
		in.Render(" "+h.Desc+strings.Repeat(" ", max(0, rest-pl-lipgloss.Width(m)-lipgloss.Width(hint))))
	return strings.Repeat(" ", g) + tg + Fit(body, rest) + strings.Repeat(" ", g)
}

// Note is one toast: what to say, how long, and at most one key it offers.
//
//	ui.Note{Kind: "ok", Label: "DONE", Msg: "Rolled back checkout-api"}               a plain toast, ToastTTL
//	ui.Note{…, For: ui.UndoTTL, Key: u, Act: unsilence}                               an undo toast (rung 1)
//	ui.Note{…, For: ui.GraceTTL, Key: u, Act: cancel, Send: page}                     a grace window (rung 2)
type Note struct {
	Kind, Label, Msg string
	// For is how long it stands: ToastTTL (the default, 0), UndoTTL or GraceTTL.
	For time.Duration
	// Key is the one key it offers while it shows (u undo), or unset. Bind it in the
	// state's action table while the note shows, and draw the row with ToastKey.
	Key key.Binding
	// Act is what Key does. The note goes first, then Act runs; it may raise the next
	// note ("Alerts back on"). On a grace window it is the undo: Send then never runs.
	Act func() tea.Cmd
	// Send is a grace window's held send (a page, an email): it runs once, when the window
	// ends or on Flush (quit), never after Act. The row counts down to it. Its returned
	// command is the send itself, so it comes back out of Tick and Flush for the app to
	// return: a send that ran in place but whose command was dropped never went.
	Send func() tea.Cmd
}

// Toasts is the toast row on the frame clock: the note that shows, and the latest note
// waiting behind a grace window. Tick it on every frame and return what it returns; bind
// the showing note's Key while Current reports one; call Act when that key is pressed and
// Flush before quitting.
//
// A grace window is never cut short: while a send is held its note keeps the row until
// undo or its end, and a note raised meanwhile waits and shows after the window's own
// confirmation (the latest waiting one only), so an unrelated key never sends early and
// "Declared INC-2041" is never covered by "No incidents". One window at a time: Raise
// refuses a second while a send is held (it returns false and changes nothing, never
// running the refused window's Act or Send), so offer the key that starts one only while
// Holding is false, and check what Raise returns.
type Toasts struct {
	cur, wait *Note
	left      int // frames the showing note has left
	sends     int // held sends run so far (toast_test.go reads it)
}

// Raise shows n now, or, while a grace window holds a send, keeps it waiting behind that
// window (the latest waiting one wins). It never runs anything: not a held send, not an
// Act. It returns false, and does nothing, for a grace window (a Note with a Send) raised
// while another holds: the caller still owns that send, and says it didn't start.
func (q *Toasts) Raise(n Note) bool {
	if n.For <= 0 {
		n.For = ToastTTL
	}
	if q.Holding() {
		if n.Send != nil {
			return false // refused: one window at a time
		}
		q.wait = &n
		return true
	}
	q.cur, q.wait, q.left = &n, nil, Frames(n.For)
	return true
}

// Current is the note that shows, if any.
func (q *Toasts) Current() (Note, bool) {
	if q.cur == nil {
		return Note{}, false
	}
	return *q.cur, true
}

// Holding reports whether a grace window holds a send.
func (q *Toasts) Holding() bool { return q.cur != nil && q.cur.Send != nil }

// Text is the showing note's message as drawn now: a grace window's counts down to its
// send in whole seconds, in the kit's unit style ("Paging payments on-call in 5s").
func (q *Toasts) Text() string {
	if q.cur == nil {
		return ""
	}
	if q.cur.Send != nil {
		return q.cur.Msg + " in " + strconv.Itoa(SecondsLeft(q.left)) + "s"
	}
	return q.cur.Msg
}

// Tick advances the showing note one frame and ends it when its time is up, returning a
// grace window's send. Pass unreachable true whenever the app doesn't offer the note's
// key right now (a dialog covers the row, an input takes letters, the screen is below
// its floor, the state binds the key: toastAction returns nil): a note that offers a key
// then waits, so a window never sends, and an undo never lapses, while its key is out of
// reach. A plain note runs on.
func (q *Toasts) Tick(unreachable bool) tea.Cmd {
	if q.cur == nil || (unreachable && q.cur.Key.Help().Key != "") {
		return nil
	}
	if q.left--; q.left > 0 {
		return nil
	}
	return q.end(true)
}

// Act runs the showing note's action (its key was pressed): the note goes, then Act runs,
// and a held send never runs.
func (q *Toasts) Act() tea.Cmd {
	c, w := q.cur, q.wait
	if c == nil || c.Act == nil {
		return nil
	}
	q.cur, q.wait, q.left = nil, nil, 0
	cmd := c.Act()
	q.resume(w)
	return cmd
}

// Flush ends the row before quitting: a held send goes now (moving on is not undo), and
// what waits is dropped. Return its command ahead of tea.Quit (tea.Sequence).
func (q *Toasts) Flush() tea.Cmd {
	if !q.Holding() {
		return nil
	}
	cmd := q.end(false)
	q.cur, q.wait = nil, nil
	return cmd
}

// end takes the showing note down, runs its held send, and shows what waited, after the
// send's own confirmation if it raised one.
func (q *Toasts) end(next bool) tea.Cmd {
	c, w := q.cur, q.wait
	q.cur, q.wait, q.left = nil, nil, 0
	var cmd tea.Cmd
	if c.Send != nil {
		q.sends++
		cmd = c.Send()
	}
	if next {
		q.resume(w)
	}
	return cmd
}

// resume puts back what waited: next if nothing shows, else waiting behind what does.
func (q *Toasts) resume(w *Note) {
	switch {
	case w == nil:
	case q.cur == nil:
		q.cur, q.left = w, Frames(w.For)
	default:
		q.wait = w
	}
}
