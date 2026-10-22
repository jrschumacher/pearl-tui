// Package session runs one program in a pseudo-terminal, interprets its output with
// Charm's VT emulator, and answers what tuicap asks of the screen: is a text on it, has
// it stopped changing, what are its cells.
package session

import (
	"errors"
	"fmt"
	"image/color"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/vt"
	"github.com/charmbracelet/x/xpty"
)

// StableFor is how long the screen must stay unchanged, after the last input, to count
// as stable.
const StableFor = 150 * time.Millisecond

const poll = 10 * time.Millisecond

// Config starts a session.
type Config struct {
	Argv   []string
	Env    []string
	W, H   int
	Fg, Bg color.Color // the terminal's default colours, answered to OSC 10/11 queries
	Full   bool        // the tracker keeps titles' text and unknown sequences' bytes
	Raw    io.Writer   // optional: every byte the program writes
}

// Exit is how the program ended.
type Exit struct {
	Exited bool   `json:"exited"`           // it exited on its own
	Code   int    `json:"code"`             // its exit code (-1 when killed by a signal)
	Signal string `json:"signal,omitempty"` // the signal that ended it
	Killed bool   `json:"killed_by_tuicap"` // tuicap ended it after the tape
	AtMS   int64  `json:"at_ms"`
}

// Session is one program on one terminal.
type Session struct {
	cfg   Config
	start time.Time

	mu        sync.Mutex // guards the emulator, the tracker and the fields below
	emu       *vt.Emulator
	tr        *Tracker
	gotOutput bool
	prev      []uv.Cell // the grid at the last poll
	changed   time.Time // when the grid last changed
	input     time.Time // when tuicap last sent input

	wmu sync.Mutex // serialises writes to the pty: keys and the emulator's replies
	pty xpty.Pty
	cmd *exec.Cmd

	done     chan struct{} // closed when the program exits
	pumpDone chan struct{} // closed when its output is drained
	exit     Exit
	stop     chan struct{}
	final    Cursor // where the cursor was when the session closed
}

// Start spawns the program.
func Start(cfg Config) (*Session, error) {
	if len(cfg.Argv) == 0 {
		return nil, errors.New("no program to run")
	}
	p, err := xpty.NewPty(cfg.W, cfg.H)
	if err != nil {
		return nil, fmt.Errorf("open a pty: %w", err)
	}
	s := &Session{cfg: cfg, pty: p, tr: NewTracker(cfg.Full), done: make(chan struct{}), pumpDone: make(chan struct{}), stop: make(chan struct{})}
	s.emu = vt.NewEmulator(cfg.W, cfg.H)
	s.emu.SetDefaultForegroundColor(cfg.Fg)
	s.emu.SetDefaultBackgroundColor(cfg.Bg)
	// Answer DECRQM for synchronized output (2026) and grapheme widths (2027) as a
	// modern terminal (Ghostty, kitty, WezTerm) does: supported. The emulator alone
	// says "not recognised", and Bubble Tea would then draw without either.
	s.emu.RegisterCsiHandler(ansi.Command('?', '$', 'p'), func(params ansi.Params) bool {
		n, _, ok := params.Param(0, 0)
		if !ok || (n != 2026 && n != 2027) {
			return false
		}
		setting := ansi.ModeReset
		if s.tr.Mode(n) {
			setting = ansi.ModeSet
		}
		// through the emulator's own reply pipe, so replies keep their order
		_, _ = io.WriteString(s.emu.InputPipe(), ansi.ReportMode(ansi.DECMode(n), setting))
		return true
	})

	s.cmd = exec.Command(cfg.Argv[0], cfg.Argv[1:]...)
	s.cmd.Env = cfg.Env
	if err := spawn(p, s.cmd); err != nil {
		p.Close()
		return nil, fmt.Errorf("start %s: %w", cfg.Argv[0], err)
	}
	s.start = time.Now()
	s.changed, s.input = s.start, s.start

	go s.pump()
	go s.replies()
	go s.watch()
	go func() {
		err := s.cmd.Wait()
		s.exit = exitOf(s.cmd, err)
		s.exit.AtMS = time.Since(s.start).Milliseconds()
		close(s.done)
	}()
	return s, nil
}

// pump feeds the program's output to the tracker and the emulator.
func (s *Session) pump() {
	defer close(s.pumpDone)
	buf := make([]byte, 32*1024)
	for {
		n, err := s.pty.Read(buf)
		if n > 0 {
			s.mu.Lock()
			if s.cfg.Raw != nil {
				_, _ = s.cfg.Raw.Write(buf[:n])
			}
			s.gotOutput = true
			s.tr.Feed(buf[:n])
			_, _ = s.emu.Write(buf[:n])
			s.mu.Unlock()
		}
		if err != nil {
			return
		}
	}
}

// replies forwards the emulator's answers to the program's queries.
func (s *Session) replies() {
	buf := make([]byte, 4096)
	for {
		n, err := s.emu.Read(buf)
		if n > 0 {
			s.write(string(buf[:n]))
		}
		if err != nil {
			return
		}
	}
}

func (s *Session) write(b string) {
	s.wmu.Lock()
	defer s.wmu.Unlock()
	_, _ = io.WriteString(s.pty, b)
}

// watch notes when the grid last changed.
func (s *Session) watch() {
	t := time.NewTicker(poll)
	defer t.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-t.C:
		}
		s.mu.Lock()
		w, h := s.emu.Width(), s.emu.Height()
		same := len(s.prev) == w*h
		if !same {
			s.prev = make([]uv.Cell, w*h)
		}
		for y := range h {
			for x := range w {
				c := s.emu.CellAt(x, y)
				i := y*w + x
				if c == nil {
					c = &uv.Cell{}
				}
				if same && !s.prev[i].Equal(c) {
					same = false
				}
				s.prev[i] = *c
			}
		}
		if !same {
			s.changed = time.Now()
		}
		s.mu.Unlock()
	}
}

// Elapsed is the time since the program started.
func (s *Session) Elapsed() time.Duration { return time.Since(s.start) }

// Done is closed when the program exits.
func (s *Session) Done() <-chan struct{} { return s.done }

// ExitStatus is how the program ended; valid once Done is closed.
func (s *Session) ExitStatus() Exit { return s.exit }

func (s *Session) exited() bool {
	select {
	case <-s.done:
		return true
	default:
		return false
	}
}

// Send writes keys, choosing arrow encodings by the program's cursor-key mode.
func (s *Session) Send(enc func(appCursor bool) string) error {
	if s.exited() {
		return errors.New("the program has exited")
	}
	s.mu.Lock()
	app := s.tr.Mode(1)
	s.input = time.Now()
	s.mu.Unlock()
	s.write(enc(app))
	return nil
}

// Type writes text as typed: one write per character, as a terminal sends keys.
func (s *Session) Type(text string) error {
	for _, r := range text {
		if err := s.Send(func(bool) string { return string(r) }); err != nil {
			return err
		}
	}
	return nil
}

// Resize resizes the terminal; the program gets SIGWINCH.
func (s *Session) Resize(w, h int) error {
	if s.exited() {
		return errors.New("the program has exited")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.emu.Resize(w, h)
	s.input = time.Now()
	return s.pty.Resize(w, h)
}

// Size is the terminal's size.
func (s *Session) Size() (int, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.emu.Width(), s.emu.Height()
}

// WaitOutput waits for the program's first byte.
func (s *Session) WaitOutput(timeout time.Duration) error {
	return s.until(timeout, func() bool { return s.gotOutput })
}

// WaitFor waits until text is on screen, on one row or across the screen's text.
func (s *Session) WaitFor(text string, timeout time.Duration) error {
	return s.until(timeout, func() bool { return strings.Contains(s.text(), text) })
}

// WaitStable waits until the screen has been unchanged for StableFor since both the
// last change and the last input. It reports false on timeout: the screen animates.
func (s *Session) WaitStable(timeout time.Duration) bool {
	return s.until(timeout, s.stable) == nil
}

// stable is called with s.mu held.
func (s *Session) stable() bool {
	last := s.changed
	if s.input.After(last) {
		last = s.input
	}
	return s.gotOutput && time.Since(last) >= StableFor
}

// WaitExit waits for the program to exit and its output to drain.
func (s *Session) WaitExit(timeout time.Duration) error {
	select {
	case <-s.done:
	case <-time.After(timeout):
		return fmt.Errorf("the program was still running after %s", timeout)
	}
	select {
	case <-s.pumpDone:
	case <-time.After(500 * time.Millisecond):
	}
	return nil
}

var errTimeout = errors.New("timed out")

// until polls cond (called with s.mu held) until it holds, the program exits, or timeout.
func (s *Session) until(timeout time.Duration, cond func() bool) error {
	deadline := time.Now().Add(timeout)
	for {
		s.mu.Lock()
		ok := cond()
		s.mu.Unlock()
		if ok {
			return nil
		}
		if s.exited() {
			// let the last output land, then look once more
			select {
			case <-s.pumpDone:
			case <-time.After(200 * time.Millisecond):
			}
			s.mu.Lock()
			ok = cond()
			s.mu.Unlock()
			if ok {
				return nil
			}
			return errExited
		}
		if time.Now().After(deadline) {
			return errTimeout
		}
		time.Sleep(poll)
	}
}

var errExited = errors.New("the program exited")

// IsExited reports whether err means the program exited during a wait.
func IsExited(err error) bool { return errors.Is(err, errExited) }

// text is the screen as plain text, rows joined by newlines; called with s.mu held.
func (s *Session) text() string {
	var b strings.Builder
	for y := range s.emu.Height() {
		for x := range s.emu.Width() {
			switch c := s.emu.CellAt(x, y); {
			case c == nil:
				b.WriteByte(' ')
			case c.Content != "":
				b.WriteString(c.Content)
			case c.Width != 0:
				b.WriteByte(' ')
			} // else the second half of a wide character
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// Cursor is where the terminal's cursor is.
type Cursor struct {
	X      int  `json:"x"`
	Y      int  `json:"y"`
	Hidden bool `json:"hidden"`
}

// Frame is the screen at a snapshot.
type Frame struct {
	W, H   int
	Lines  []uv.Line
	Stable bool // unchanged for StableFor since the last change and the last input
	Cursor Cursor
}

// Snapshot copies the screen. It waits (up to half a second) for a synchronized
// update (mode 2026) in flight to end, so a frame is never half drawn. Cells the
// program left at the terminal's default colours stay default, unless it set the
// defaults (OSC 10/11): then they are painted with them, as the terminal shows them.
func (s *Session) Snapshot() Frame {
	deadline := time.Now().Add(500 * time.Millisecond)
	for {
		s.mu.Lock()
		if !s.tr.Mode(2026) || time.Now().After(deadline) {
			break
		}
		s.mu.Unlock()
		time.Sleep(poll)
	}
	defer s.mu.Unlock()
	w, h := s.emu.Width(), s.emu.Height()
	bg, fg := s.emu.BackgroundColor(), s.emu.ForegroundColor()
	paintBg, paintFg := !sameColor(bg, s.cfg.Bg), !sameColor(fg, s.cfg.Fg)
	f := Frame{W: w, H: h, Stable: s.stable(), Lines: make([]uv.Line, h)}
	for y := range h {
		line := make(uv.Line, w)
		for x := range w {
			c := s.emu.CellAt(x, y)
			if c == nil {
				line[x] = uv.EmptyCell
				continue
			}
			nc := *c
			nc.Link = uv.Link{} // frames are SGR only
			if paintBg && nc.Style.Bg == nil && nc.Width > 0 {
				nc.Style.Bg = bg
			}
			if paintFg && nc.Style.Fg == nil && nc.Content != " " && nc.Content != "" {
				nc.Style.Fg = fg
			}
			line[x] = nc
		}
		f.Lines[y] = line
	}
	pos := s.emu.CursorPosition()
	f.Cursor = Cursor{X: pos.X, Y: pos.Y, Hidden: s.emu.CursorHidden()}
	return f
}

func sameColor(a, b color.Color) bool {
	if a == nil || b == nil {
		return a == b
	}
	r1, g1, b1, a1 := a.RGBA()
	r2, g2, b2, a2 := b.RGBA()
	return r1 == r2 && g1 == g2 && b1 == b2 && a1 == a2
}

// Cut closes the tracker's segment at the snapshot state and returns its events.
func (s *Session) Cut(state string) map[string]int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.tr.Cut(state)
}

// FinalCursor is where the cursor was when the session closed.
func (s *Session) FinalCursor() Cursor { return s.final }

// Tracker is read once the session is closed.
func (s *Session) Tracker() *Tracker { return s.tr }

// Close ends the program if it still runs (SIGHUP to its process group, then SIGKILL)
// and releases the terminal. It returns how the program ended.
func (s *Session) Close() Exit {
	killed := false
	if !s.exited() {
		killed = true
		hangup(s.cmd)
		select {
		case <-s.done:
		case <-time.After(time.Second):
			kill(s.cmd)
			<-s.done
		}
	}
	select {
	case <-s.pumpDone:
	case <-time.After(500 * time.Millisecond):
	}
	close(s.stop)
	s.mu.Lock()
	pos := s.emu.CursorPosition()
	s.final = Cursor{X: pos.X, Y: pos.Y, Hidden: s.emu.CursorHidden()}
	s.mu.Unlock()
	_ = s.pty.Close()
	closeReplies(s.emu)
	s.exit.Killed = killed
	return s.exit
}

// closeReplies ends the emulator's reply pipe, which unblocks a pending Read. Not
// Emulator.Close: in vt v0.1.0 it sets an unguarded flag that a blocked Read also reads,
// a data race.
func closeReplies(e *vt.Emulator) {
	if c, ok := e.InputPipe().(io.Closer); ok {
		_ = c.Close()
	}
}
