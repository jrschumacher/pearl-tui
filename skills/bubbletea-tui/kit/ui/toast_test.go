package ui

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/key"
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

// A countdown reads whole seconds rounded up: "in 5 s" on the first frame of a 5 s
// window, never "in 0 s" while the send is still held.
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
	long := "Paging payments on-call for INC-2041 checkout-api 502s across three regions in 5 s"
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
					hint := th.G.Sep + " u undo"
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
