package session

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// Tracker reads the program's output stream beside the emulator and records what it
// did to the terminal beyond drawing: modes it set, queries it sent, titles, links,
// clipboard writes, clears, and any sequence it doesn't know. Events are counted per
// segment (the output between two snapshots) and in total, with the snapshot each was
// first seen at.
type Tracker struct {
	p    *ansi.Parser
	full bool // keep titles' text and unknown sequences' bytes

	seg   map[string]int
	total map[string]*Total

	unknown map[string]*Unknown

	Title    string // the last non-empty title set (full only)
	TitleLen int    // its length in runes

	modes map[int]bool // DEC private modes as last set (true) or reset (false)
	kitty int          // Kitty keyboard flags pushed minus popped
}

// Total is one event's count over the run and the snapshot it was first seen before.
type Total struct {
	Count int    `json:"count"`
	First string `json:"first"`
}

// Unknown is a sequence the tracker doesn't know: its kind (CSI with its prefix,
// intermediate and final, OSC with its number…), and in full mode its bytes.
type Unknown struct {
	Kind  string `json:"kind"`
	Count int    `json:"count"`
	First string `json:"first"`
	Bytes string `json:"bytes,omitempty"`
}

// NewTracker returns a tracker; full keeps titles and unknown bytes.
func NewTracker(full bool) *Tracker {
	t := &Tracker{p: ansi.NewParser(), full: full, seg: map[string]int{}, total: map[string]*Total{},
		unknown: map[string]*Unknown{}, modes: map[int]bool{}}
	t.p.SetHandler(ansi.Handler{
		Execute:   t.execute,
		HandleCsi: t.csi,
		HandleEsc: t.esc,
		HandleOsc: t.osc,
		HandleDcs: t.dcs,
		HandleApc: func([]byte) { t.other("APC", "") },
		HandlePm:  func([]byte) { t.other("PM", "") },
		HandleSos: func([]byte) { t.other("SOS", "") },
	})
	return t
}

// Feed reads a chunk of output.
func (t *Tracker) Feed(b []byte) { t.p.Parse(b) }

// Mode reports whether DEC private mode n is set.
func (t *Tracker) Mode(n int) bool { return t.modes[n] }

func (t *Tracker) add(ev string) { t.seg[ev]++ }

// Cut closes the segment that ends at the snapshot state and returns its counts.
func (t *Tracker) Cut(state string) map[string]int {
	seg := t.seg
	t.seg = map[string]int{}
	for ev, n := range seg {
		tot := t.total[ev]
		if tot == nil {
			tot = &Total{First: state}
			t.total[ev] = tot
		}
		tot.Count += n
	}
	for _, u := range t.unknown {
		if u.First == "" {
			u.First = state
		}
	}
	return seg
}

// Totals is every event over the run.
func (t *Tracker) Totals() map[string]*Total { return t.total }

// Unknowns lists the unknown sequences, most frequent first.
func (t *Tracker) Unknowns() []Unknown {
	var out []Unknown
	for _, u := range t.unknown {
		out = append(out, *u)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Kind < out[j].Kind
	})
	return out
}

// Left lists what the program left on in the terminal: empty means it ended clean.
func (t *Tracker) Left() []string {
	var left []string
	names := map[int]string{1049: "alt screen", 1047: "alt screen", 47: "alt screen", 1000: "mouse 1000", 1002: "mouse 1002",
		1003: "mouse 1003", 1006: "mouse SGR 1006", 2004: "bracketed paste", 1004: "focus reporting", 2026: "synchronized output", 1: "application cursor keys"}
	seen := map[string]bool{}
	for _, n := range []int{1049, 1047, 47, 1000, 1002, 1003, 1006, 2004, 1004, 2026, 1} {
		if t.modes[n] && !seen[names[n]] {
			seen[names[n]] = true
			left = append(left, names[n]+" still on")
		}
	}
	if hidden, ok := t.modes[25]; ok && !hidden {
		left = append(left, "cursor still hidden")
	}
	if t.kitty > 0 {
		left = append(left, "kitty keyboard flags still pushed")
	}
	return left
}

func (t *Tracker) execute(b byte) {
	if b == ansi.BEL {
		t.add("bell")
	}
}

// the DEC private modes with event names of their own
var modeEvents = map[int]string{1049: "alt_screen", 1047: "alt_screen", 47: "alt_screen", 2026: "sync", 25: "cursor",
	1000: "mouse_1000", 1002: "mouse_1002", 1003: "mouse_1003", 1006: "mouse_1006", 2004: "bracketed_paste", 1004: "focus"}

func (t *Tracker) csi(cmd ansi.Cmd, params ansi.Params) {
	prefix, inter, final := cmd.Prefix(), cmd.Intermediate(), cmd.Final()
	switch {
	case prefix == '?' && inter == 0 && (final == 'h' || final == 'l'):
		on := final == 'h'
		params.ForEach(-1, func(_, n int, _ bool) {
			if n < 0 {
				return
			}
			t.modes[n] = on
			name, ok := modeEvents[n]
			switch {
			case n == 25:
				name = map[bool]string{true: "cursor_show", false: "cursor_hide"}[on]
			case n == 2026:
				name = map[bool]string{true: "sync_begin", false: "sync_end"}[on]
			case ok:
				name += map[bool]string{true: "_on", false: "_off"}[on]
			default:
				name = "mode_" + strconv.Itoa(n) + map[bool]string{true: "_on", false: "_off"}[on]
			}
			t.add(name)
		})
	case prefix == 0 && inter == 0 && final == 'J':
		n, _, _ := params.Param(0, 0)
		switch n {
		case 2:
			t.add("clear_screen")
		case 3:
			t.add("clear_scrollback")
		}
	case inter == '$' && final == 'p', prefix == 0 && inter == 0 && final == 'c', prefix == '>' && final == 'c',
		prefix == '?' && final == 'u', prefix == '>' && final == 'q', prefix == 0 && final == 'n':
		t.add("query") // DECRQM, DA1, DA2, Kitty keyboard flags, XTVERSION, DSR/CPR
	case prefix == '>' && final == 'u':
		t.kitty++
		t.add("kitty_keyboard_push")
	case prefix == '<' && final == 'u':
		n, _, _ := params.Param(0, 1)
		t.kitty = max(0, t.kitty-n)
		t.add("kitty_keyboard_pop")
	case prefix == '=' && final == 'u':
		t.add("kitty_keyboard_set")
	case prefix == '>' && final == 'm':
		t.add("modify_other_keys")
	case prefix == 0 && inter == '!' && final == 'p':
		t.add("soft_reset")
	case prefix == 0 && inter == ' ' && final == 'q':
		t.add("cursor_style")
	case prefix == '?' && final == 'W', prefix == 0 && final == 'g': // DECST8C, TBC: tab stops
	case prefix == 0 && inter == 0 && strings.IndexByte("ABCDEFGHIJKLMPSTXZ@`abdefghlmnrstu", final) >= 0:
		// drawing: cursor moves, erases, scroll, SGR, margins, save/restore, window ops
	default:
		t.other(fmt.Sprintf("CSI %s", cmdName(prefix, inter, final)), csiBytes(cmd, params))
	}
}

func (t *Tracker) esc(cmd ansi.Cmd) {
	inter, final := cmd.Intermediate(), cmd.Final()
	switch {
	case inter == 0 && final == 'c':
		t.add("reset")
	case inter == 0 && strings.IndexByte("78=>DEHMNOZ\\", final) >= 0,
		inter == '(' || inter == ')' || inter == '*' || inter == '+' || inter == '#':
		// save/restore cursor, keypad modes, index, charsets, string terminator
	default:
		raw := "\x1b"
		if inter != 0 {
			raw += string(inter)
		}
		t.other("ESC "+cmdName(0, inter, final), raw+string(final))
	}
}

func (t *Tracker) osc(cmd int, data []byte) {
	_, rest, _ := strings.Cut(string(data), ";")
	switch cmd {
	case 0, 2:
		t.add("title")
		if rest != "" { // the last title set; an empty one (a reset on exit) keeps it
			t.TitleLen = len([]rune(rest))
			if t.full {
				t.Title = rest
			}
		}
	case 1:
		t.add("icon_name")
	case 8:
		if _, url, _ := strings.Cut(rest, ";"); url != "" {
			t.add("hyperlink")
		}
	case 52:
		t.add("clipboard")
	case 10, 11, 12:
		if strings.HasSuffix(rest, "?") {
			t.add("query")
		} else {
			t.add("set_default_color")
		}
	case 4, 104, 110, 111, 112:
		t.add("set_default_color")
	case 7:
		t.add("working_directory")
	case 9, 777:
		t.add("notification")
	case 22:
		t.add("pointer_shape")
	case 133:
		t.add("prompt_mark")
	default:
		t.other("OSC "+strconv.Itoa(cmd), "\x1b]"+trunc(string(data), 64)+"\x07")
	}
}

func (t *Tracker) dcs(cmd ansi.Cmd, params ansi.Params, data []byte) {
	if cmd.Final() == 'q' && (cmd.Intermediate() == '+' || cmd.Intermediate() == '$') {
		t.add("query") // XTGETTCAP, DECRQSS
		return
	}
	t.other("DCS "+cmdName(cmd.Prefix(), cmd.Intermediate(), cmd.Final()), "\x1bP"+paramString(params)+trunc(string(data), 64)+"\x1b\\")
}

// other counts an unknown sequence under its kind.
func (t *Tracker) other(kind, raw string) {
	t.add("unknown")
	u := t.unknown[kind]
	if u == nil {
		u = &Unknown{Kind: kind}
		t.unknown[kind] = u
	}
	u.Count++
	if t.full && u.Bytes == "" && raw != "" {
		u.Bytes = raw
	}
}

func cmdName(prefix, inter, final byte) string {
	var b strings.Builder
	for _, c := range []byte{prefix, inter, final} {
		if c != 0 {
			b.WriteByte(c)
		}
	}
	return b.String()
}

func csiBytes(cmd ansi.Cmd, params ansi.Params) string {
	var b strings.Builder
	b.WriteString("\x1b[")
	if p := cmd.Prefix(); p != 0 {
		b.WriteByte(p)
	}
	b.WriteString(paramString(params))
	if i := cmd.Intermediate(); i != 0 {
		b.WriteByte(i)
	}
	b.WriteByte(cmd.Final())
	return b.String()
}

func paramString(params ansi.Params) string {
	var b strings.Builder
	params.ForEach(-1, func(i, p int, more bool) {
		if p >= 0 {
			b.WriteString(strconv.Itoa(p))
		}
		if i < len(params)-1 {
			if more {
				b.WriteByte(':')
			} else {
				b.WriteByte(';')
			}
		}
	})
	return b.String()
}

func trunc(s string, n int) string {
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}
