package ui

import (
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"

	"tuikit/theme"
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

// toastStyles are the tag and the tinted bar of a toast of kind: ok (the default),
// error, warning or info.
func toastStyles(t *theme.Theme, kind string) (tag, bar lipgloss.Style) {
	switch kind {
	case "error":
		return t.S.TagErr, t.S.BarErr
	case "warning":
		return t.S.TagWarn, t.S.BarWarn
	case "info":
		return t.S.TagInfo, t.S.BarInfo
	}
	return t.S.TagOK, t.S.BarOK
}

// ToastKey is a Toast that offers one key while it shows: the message, the theme's
// separator, then the binding's key (bold) and what it does, in the bar's ink:
//
//	DONE  Silenced checkout-api alerts for 1h · u unsilence
//	HELD  Paging payments on-call in 5 s · u undo
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
	hint := sep + h.Key + " " + h.Desc
	room := max(0, rest-pl-pr-lipgloss.Width(hint))
	m := Trunc(t, msg, room)
	body := in.Render(strings.Repeat(" ", pl)+m+sep) + in.Bold(true).Render(h.Key) +
		in.Render(" "+h.Desc+strings.Repeat(" ", max(0, rest-pl-lipgloss.Width(m)-lipgloss.Width(hint))))
	return strings.Repeat(" ", g) + tg + Fit(body, rest) + strings.Repeat(" ", g)
}
