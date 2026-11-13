// Package tape reads tuicap's tape language: one command per line, run in order
// against a program in a pseudo-terminal.
//
//	# a comment
//	WaitFor Services        until the text is on screen (-wait bounds it)
//	Key down down enter     named keys, ctrl+x / alt+x / shift+tab combos; down*3 repeats
//	Type payments           the rest of the line, as typed
//	Wait 200ms              a fixed pause
//	WaitStable [2s]         until the screen is unchanged for 150 ms (bounded)
//	Snapshot services       write frames/services-<W>x<H>.ansi
//	Resize 120x40           resize the terminal (the program gets SIGWINCH)
//	WaitExit [2s]           until the program exits; only Snapshot may follow
package tape

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// Op is a tape command.
type Op int

// The tape's commands.
const (
	OpKey Op = iota
	OpType
	OpWait
	OpWaitStable
	OpWaitFor
	OpSnapshot
	OpResize
	OpWaitExit
)

var opNames = map[string]Op{
	"key": OpKey, "type": OpType, "wait": OpWait, "waitstable": OpWaitStable, "waitfor": OpWaitFor,
	"snapshot": OpSnapshot, "resize": OpResize, "waitexit": OpWaitExit,
}

// Cmd is one line of a tape.
type Cmd struct {
	Line int
	Op   Op
	Src  string        // the line as written, for messages
	Keys []Key         // OpKey
	Text string        // OpType, OpWaitFor; OpSnapshot: the state name
	Dur  time.Duration // OpWait; OpWaitStable and OpWaitExit: 0 means the -wait default
	W, H int           // OpResize
}

// Key is a key as the tape names it.
type Key struct {
	Name             string
	named            string // a namedKeys value, or "" for the rune r
	r                rune
	ctrl, alt, shift bool
}

var stateRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// Parse reads a tape. Errors name the line.
func Parse(r io.Reader) ([]Cmd, error) {
	var cmds []Cmd
	sc := bufio.NewScanner(r)
	n := 0
	exited := false
	for sc.Scan() {
		n++
		line := strings.TrimRight(sc.Text(), " \t\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		word, rest, _ := strings.Cut(trimmed, " ")
		op, ok := opNames[strings.ToLower(word)]
		if !ok {
			return nil, fmt.Errorf("line %d: unknown command %q (want Key, Type, Wait, WaitStable, WaitFor, Snapshot, Resize or WaitExit)", n, word)
		}
		c := Cmd{Line: n, Op: op, Src: trimmed}
		if exited && op != OpSnapshot {
			return nil, fmt.Errorf("line %d: only Snapshot may follow WaitExit", n)
		}
		var err error
		switch op {
		case OpKey:
			for _, f := range strings.Fields(rest) {
				count := 1
				if name, rep, ok := strings.Cut(f, "*"); ok {
					count, err = strconv.Atoi(rep)
					if err != nil || count < 1 || count > 1000 {
						return nil, fmt.Errorf("line %d: %q: repeat count must be 1..1000", n, f)
					}
					f = name
				}
				k, err := ParseKey(f)
				if err != nil {
					return nil, fmt.Errorf("line %d: %w", n, err)
				}
				for range count {
					c.Keys = append(c.Keys, k)
				}
			}
			if len(c.Keys) == 0 {
				return nil, fmt.Errorf("line %d: Key needs at least one key name", n)
			}
		case OpType:
			// the rest of the line verbatim, after the one separating space
			_, c.Text, _ = strings.Cut(strings.TrimLeft(line, " \t"), " ")
			if c.Text == "" {
				return nil, fmt.Errorf("line %d: Type needs text", n)
			}
		case OpWaitFor:
			c.Text = strings.TrimSpace(rest)
			if c.Text == "" {
				return nil, fmt.Errorf("line %d: WaitFor needs text", n)
			}
		case OpWait:
			c.Dur, err = time.ParseDuration(strings.TrimSpace(rest))
			if err != nil || c.Dur <= 0 {
				return nil, fmt.Errorf("line %d: Wait needs a duration such as 200ms", n)
			}
		case OpWaitStable, OpWaitExit:
			if r := strings.TrimSpace(rest); r != "" {
				c.Dur, err = time.ParseDuration(r)
				if err != nil || c.Dur <= 0 {
					return nil, fmt.Errorf("line %d: %s takes an optional timeout such as 2s", n, word)
				}
			}
			exited = exited || op == OpWaitExit
		case OpSnapshot:
			c.Text = strings.TrimSpace(rest)
			if !stateRe.MatchString(c.Text) {
				return nil, fmt.Errorf("line %d: Snapshot needs a state name of letters, digits, '.', '_' or '-'", n)
			}
		case OpResize:
			c.W, c.H, err = ParseSize(strings.TrimSpace(rest))
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", n, err)
			}
		}
		cmds = append(cmds, c)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return cmds, nil
}

// ParseSize reads WxH.
func ParseSize(s string) (int, int, error) {
	a, b, ok := strings.Cut(strings.ToLower(s), "x")
	w, err1 := strconv.Atoi(a)
	h, err2 := strconv.Atoi(b)
	if !ok || err1 != nil || err2 != nil || w < 10 || h < 3 || w > 1000 || h > 1000 {
		return 0, 0, fmt.Errorf("size %q: want WxH such as 80x24 (10..1000 by 3..1000)", s)
	}
	return w, h, nil
}

// ParseKey reads a key name: a named key (enter, esc, tab, space, backspace, up, down,
// left, right, pgup, pgdn, home, end, delete, insert, f1..f12), a single character
// (q, ?, /, j), or either after ctrl+, alt+ or shift+ (ctrl+c, alt+enter, shift+tab,
// ctrl+up). The bytes are what xterm sends, legacy encoding: no Kitty protocol.
func ParseKey(s string) (Key, error) {
	parts := strings.Split(s, "+")
	if strings.HasSuffix(s, "+") && len(parts) >= 2 { // "+" itself, or ctrl++
		parts = append(parts[:len(parts)-2], "+")
	}
	base := parts[len(parts)-1]
	k := Key{Name: s}
	for _, m := range parts[:len(parts)-1] {
		switch strings.ToLower(m) {
		case "ctrl":
			k.ctrl = true
		case "alt":
			k.alt = true
		case "shift":
			k.shift = true
		default:
			return Key{}, fmt.Errorf("key %q: unknown modifier %q (want ctrl, alt or shift)", s, m)
		}
	}
	if nk, ok := namedKeys[strings.ToLower(base)]; ok {
		k.named = nk
	} else if rs := []rune(base); len(rs) == 1 {
		k.r = rs[0]
	} else {
		return Key{}, fmt.Errorf("key %q: unknown key name (a single character, or enter, esc, tab, space, backspace, up, down, left, right, pgup, pgdn, home, end, delete, insert, f1..f12)", s)
	}
	if _, err := k.encode(false); err != nil {
		return Key{}, fmt.Errorf("key %q: %w", s, err)
	}
	return k, nil
}

// Bytes is what the terminal sends for the key; appCursor is DECCKM (?1h), under
// which plain arrows are sent as SS3 (ESC O A).
func (k Key) Bytes(appCursor bool) string {
	b, _ := k.encode(appCursor)
	return b
}

var namedKeys = map[string]string{
	"enter": "enter", "return": "enter", "tab": "tab", "esc": "esc", "escape": "esc",
	"space": "space", "backspace": "backspace", "bs": "backspace",
	"up": "up", "down": "down", "left": "left", "right": "right",
	"pgup": "pgup", "pageup": "pgup", "pgdn": "pgdn", "pgdown": "pgdn", "pagedown": "pgdn",
	"home": "home", "end": "end", "delete": "delete", "del": "delete", "insert": "insert",
	"f1": "f1", "f2": "f2", "f3": "f3", "f4": "f4", "f5": "f5", "f6": "f6",
	"f7": "f7", "f8": "f8", "f9": "f9", "f10": "f10", "f11": "f11", "f12": "f12",
}

// csiKeys are the keys xterm sends as CSI with a final byte (and CSI 1;m X when modified),
// tildeKeys as CSI n ~ (CSI n;m ~ when modified).
var (
	csiKeys   = map[string]byte{"up": 'A', "down": 'B', "right": 'C', "left": 'D', "home": 'H', "end": 'F', "f1": 'P', "f2": 'Q', "f3": 'R', "f4": 'S'}
	ss3Keys   = map[string]bool{"f1": true, "f2": true, "f3": true, "f4": true}
	tildeKeys = map[string]int{"insert": 2, "delete": 3, "pgup": 5, "pgdn": 6, "f5": 15, "f6": 17, "f7": 18, "f8": 19, "f9": 20, "f10": 21, "f11": 23, "f12": 24}
	arrows    = map[string]bool{"up": true, "down": true, "right": true, "left": true}
)

func (k Key) encode(appCursor bool) (string, error) {
	mod := 1
	if k.shift {
		mod++
	}
	if k.alt {
		mod += 2
	}
	if k.ctrl {
		mod += 4
	}
	if f, ok := csiKeys[k.named]; ok {
		switch {
		case mod > 1:
			return fmt.Sprintf("\x1b[1;%d%c", mod, f), nil
		case ss3Keys[k.named] || (appCursor && arrows[k.named]):
			return "\x1bO" + string(f), nil
		}
		return "\x1b[" + string(f), nil
	}
	if n, ok := tildeKeys[k.named]; ok {
		if mod > 1 {
			return fmt.Sprintf("\x1b[%d;%d~", n, mod), nil
		}
		return fmt.Sprintf("\x1b[%d~", n), nil
	}
	prefix := ""
	if k.alt {
		prefix = "\x1b"
	}
	switch k.named {
	case "tab":
		if k.shift && !k.ctrl {
			return prefix + "\x1b[Z", nil
		}
		return prefix + "\t", noMods(k, "tab", false)
	case "enter":
		return prefix + "\r", noMods(k, "enter", false)
	case "esc":
		return prefix + "\x1b", noMods(k, "esc", false)
	case "backspace":
		if k.ctrl {
			return prefix + "\x08", noMods(k, "backspace", true)
		}
		return prefix + "\x7f", noMods(k, "backspace", false)
	case "space":
		if k.ctrl {
			return prefix + "\x00", noMods(k, "space", true)
		}
		return prefix + " ", noMods(k, "space", false)
	}
	r := k.r
	if k.shift {
		r = unicode.ToUpper(r)
	}
	if k.ctrl {
		switch l := unicode.ToLower(r); {
		case l >= 'a' && l <= 'z':
			r = l & 0x1f
		case r >= '@' && r <= '_': // @ [ \ ] ^ _
			r &= 0x1f
		case r == '?':
			r = 0x7f
		default:
			return "", fmt.Errorf("ctrl+%c has no legacy encoding", r)
		}
	}
	return prefix + string(r), nil
}

// noMods rejects shift and ctrl on keys that legacy encoding can't tell apart; ctrlOK
// allows ctrl where it has a byte of its own.
func noMods(k Key, name string, ctrlOK bool) error {
	if k.shift || (k.ctrl && !ctrlOK) {
		return fmt.Errorf("modified %s has no legacy encoding (alt+%s works)", name, name)
	}
	return nil
}
