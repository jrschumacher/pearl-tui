// Command fixture is a tiny full-screen program for tuicap's tests: it draws a styled
// screen at the terminal's size (alt screen, hidden cursor, a title, a link, an unknown
// sequence), and reacts to keys: ↑/↓ move the selection, / starts a filter that takes
// typed text until enter, enter opens the selection, q leaves and restores the terminal.
// Larkspur, its accounts and addresses are fictional.
package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/charmbracelet/x/term"
)

var items = []string{"Larkspur Ledger", "Quarry Notes", "Bellwether Sync"}

type state struct {
	w, h      int
	sel       int
	filtering bool
	filter    string
	opened    string
}

func main() {
	fd := os.Stdin.Fd()
	old, err := term.MakeRaw(fd)
	if err != nil {
		fmt.Fprintln(os.Stderr, "fixture: not a terminal:", err)
		os.Exit(1)
	}
	out := os.Stdout
	// alt screen, hidden cursor, a title, and a sequence no terminal knows
	fmt.Fprint(out, "\x1b[?1049h\x1b[?25l\x1b]2;Larkspur ledger\x07\x1b[?7;1x")
	s := &state{}
	s.w, s.h, _ = term.GetSize(fd)
	draw(s)

	winch := make(chan os.Signal, 1)
	signal.Notify(winch, syscall.SIGWINCH)
	keys := make(chan string)
	go func() {
		buf := make([]byte, 64)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil {
				close(keys)
				return
			}
			keys <- string(buf[:n])
		}
	}()
	for {
		select {
		case <-winch:
			s.w, s.h, _ = term.GetSize(fd)
		case in, ok := <-keys:
			for _, k := range split(in) {
				if !ok || !s.key(k) {
					ok = false
					break
				}
			}
			if !ok {
				fmt.Fprint(out, "\x1b[?25h\x1b[?1049l")
				_ = term.Restore(fd, old)
				return
			}
		}
		draw(s)
	}
}

// split cuts one read of input into keys: an arrow's CSI, or one character.
func split(in string) []string {
	var keys []string
	for in != "" {
		n := len(string([]rune(in)[0]))
		if strings.HasPrefix(in, "\x1b[") && len(in) >= 3 {
			n = 3
		}
		keys = append(keys, in[:n])
		in = in[n:]
	}
	return keys
}

// key handles one key; false quits.
func (s *state) key(k string) bool {
	switch {
	case s.filtering && k == "\r":
		s.filtering = false
	case s.filtering && k == "\x7f":
		if s.filter != "" {
			s.filter = s.filter[:len(s.filter)-1]
		}
	case s.filtering:
		s.filter += k
	case k == "q", k == "\x03":
		return false
	case k == "\x1b[A":
		s.sel = (s.sel + len(items) - 1) % len(items)
	case k == "\x1b[B":
		s.sel = (s.sel + 1) % len(items)
	case k == "/":
		s.filtering = true
	case k == "\r":
		s.opened = items[s.sel]
	}
	return true
}

// row pads text to the width with style around it.
func row(w int, style, text string) string {
	n := len([]rune(text))
	if n > w {
		text, n = string([]rune(text)[:w]), w
	}
	return style + text + strings.Repeat(" ", w-n) + "\x1b[m"
}

func draw(s *state) {
	var b strings.Builder
	b.WriteString("\x1b[H\x1b[2J")
	lines := []string{
		row(s.w, "\x1b[1;38;2;250;240;230;48;2;40;60;90m", " Larkspur Ledger · accounts 12"),
		"",
	}
	for i, it := range items {
		if i == s.sel {
			lines = append(lines, row(s.w, "\x1b[38;2;120;200;160m", " ▌ "+it+" — 42 entries"))
		} else {
			lines = append(lines, row(s.w, "\x1b[38;2;150;150;160m", "   "+it))
		}
	}
	lines = append(lines, "",
		row(s.w, "", " see https://larkspur.example/acct/42 or mail ops@larkspur.example"),
		row(s.w, "", " log in /var/log/larkspur/ledger.log"),
		" \x1b]8;;https://larkspur.example\x1b\\docs\x1b]8;;\x1b\\")
	if s.filtering || s.filter != "" {
		lines = append(lines, row(s.w, "\x1b[38;2;220;180;90m", " filter: "+s.filter))
	}
	if s.opened != "" {
		lines = append(lines, row(s.w, "", " Opened "+s.opened))
	}
	for i := len(lines); i < s.h-1; i++ {
		lines = append(lines, "")
	}
	lines = lines[:s.h-1]
	lines = append(lines, row(s.w, "\x1b[38;2;110;110;120m", " ↑/↓ move · enter open · / filter · ctrl+c quit · q quit · ? help"))
	b.WriteString(strings.Join(lines, "\r\n"))
	fmt.Fprint(os.Stdout, b.String())
}
