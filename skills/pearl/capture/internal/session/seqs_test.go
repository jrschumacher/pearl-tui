package session

import (
	"testing"
)

func TestTracker(t *testing.T) {
	tr := NewTracker(true)
	tr.Feed([]byte("\x1b[?1049h\x1b[?25l\x1b[?2026h\x1b[2J\x1b[H\x1b[1;31mhi\x1b[m\x1b[?2026l\x1b]2;Ledger\x07\x1b[?1002;1006h\x1b[?2004h\x1b[c"))
	tr.Feed([]byte("\x1b]8;;https://x.test\x1b\\link\x1b]8;;\x1b\\\x1b]52;c;aGk=\x07\x1b[3J\x1b[?9"))
	tr.Feed([]byte("9z")) // split across reads: CSI ?99z, unknown
	if tr.Mode(2026) {
		t.Error("synchronized output still on")
	}
	seg := tr.Cut("first")
	for ev, n := range map[string]int{"alt_screen_on": 1, "cursor_hide": 1, "sync_begin": 1, "sync_end": 1, "clear_screen": 1,
		"title": 1, "mouse_1002_on": 1, "mouse_1006_on": 1, "bracketed_paste_on": 1, "query": 1, "hyperlink": 1,
		"clipboard": 1, "clear_scrollback": 1, "unknown": 1} {
		if seg[ev] != n {
			t.Errorf("%s = %d, want %d (segment %v)", ev, seg[ev], n, seg)
		}
	}
	if tr.Title != "Ledger" || tr.TitleLen != 6 {
		t.Errorf("title = %q (%d)", tr.Title, tr.TitleLen)
	}
	if u := tr.Unknowns(); len(u) != 1 || u[0].Kind != "CSI ?z" || u[0].Bytes != "\x1b[?99z" || u[0].First != "first" {
		t.Errorf("unknowns = %+v", u)
	}
	left := tr.Left()
	if len(left) != 5 {
		t.Errorf("left = %v, want alt screen, mouse 1002, mouse SGR, bracketed paste, cursor", left)
	}
	tr.Feed([]byte("\x1b]2;\x07\x1b[?1002;1006l\x1b[?2004l\x1b[?25h\x1b[?1049l"))
	tr.Cut("(end)")
	if left := tr.Left(); len(left) != 0 {
		t.Errorf("left after restoring = %v", left)
	}
	if tr.Title != "Ledger" {
		t.Errorf("an empty title replaced the last one: %q", tr.Title)
	}
	if tot := tr.Totals()["alt_screen_off"]; tot == nil || tot.First != "(end)" {
		t.Errorf("alt_screen_off total = %+v", tot)
	}

	priv := NewTracker(false)
	priv.Feed([]byte("\x1b]2;Secret\x07\x1b[?99z"))
	priv.Cut("s")
	if priv.Title != "" || priv.TitleLen != 6 || priv.Unknowns()[0].Bytes != "" {
		t.Errorf("privacy tracker kept text: %q %+v", priv.Title, priv.Unknowns())
	}
}
