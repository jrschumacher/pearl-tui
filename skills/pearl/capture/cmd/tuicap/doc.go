// Command tuicap captures any terminal program's screens as .ansi frames, so cellshot,
// tuilint frames and the critic can see an app that has no in-process harness (one built
// without the kit, or an existing CLI being restyled). It runs the program in a
// pseudo-terminal at W×H, drives it with a tape, and writes each Snapshot as a frame in
// the kit harness's format: exactly W×H cells, SGR only, one row per line.
//
//	tuicap -tape setup.tape -out capture -- ./bin/app init
//	tuicap -sizes 80x24,120x40 -tape setup.tape -out capture -- ./bin/app init
//	tuicap -profile ansi -tape setup.tape -out capture-16 -- ./bin/app
//	tuicap -level privacy -tape setup.tape -out share -- ./bin/app   frames safe to share
//	cellshot capture/frames/*-80x24.ansi --sheet sheet.png; tuilint frames capture/frames
//
// The tape is a text file, one command per line (# starts a comment):
//
//	WaitFor Services        wait until the text is on screen (fails after -wait)
//	Key down down enter     named keys: enter esc tab space backspace up down left right
//	Key ctrl+c alt+x q ?    pgup pgdn home end delete insert f1..f12, any one character,
//	Key down*3              ctrl+/alt+/shift+ combos; *N repeats
//	Type payments-eu        the rest of the line, one character at a time
//	Wait 200ms              a fixed pause
//	WaitStable [2s]         until the screen is unchanged for 150 ms after the last input;
//	                        a screen still moving at the timeout (-wait) is a warning only
//	Snapshot setup-target   write frames/setup-target-<W>x<H>.ansi at the current size
//	Resize 120x40           resize the terminal; the program gets SIGWINCH
//	WaitExit [2s]           until the program exits (fails after the timeout); only
//	                        Snapshot may follow, of the screen it left behind
//
// Input waits for the program's first output. Prefer WaitFor over WaitStable: a program
// slower than 150 ms to react to a key is otherwise judged stable too early.
//
// Options
//
//	-w W -h H          terminal size (default 80x24)
//	-sizes 80x24,…     run the whole tape once per size (overrides -w/-h)
//	-tape FILE         the tape (required; - reads stdin)
//	-out DIR           the bundle directory (required): empty, or a bundle tuicap wrote,
//	                   whose files it replaces
//	-profile P         truecolor (default) | ansi256 | ansi | ascii: sets TERM, COLORTERM
//	                   and NO_COLOR for the program as a terminal of that depth would
//	                   (truecolor: xterm-256color + COLORTERM=truecolor; ansi256:
//	                   xterm-256color; ansi: xterm; ascii: NO_COLOR=1). The kit's ASCII
//	                   glyphs (TERM=linux) are -env TERM=linux.
//	-env K=V           add or override an environment variable (repeatable); the rest of
//	                   tuicap's environment is passed through, minus the colour ones
//	-light             the terminal's default colours are light (#222222 on #fafafa):
//	                   OSC 11 queries answer light, sheets render --light. Default
//	                   dark, #d8d8d8 on #16161a, cellshot's defaults.
//	-wait D            default timeout for WaitFor, WaitStable, WaitExit (default 5s)
//	-timeout D         limit for one run of the tape (default 2m)
//	-level L           full (default) | privacy (below)
//	-allow w1,w2       privacy: words kept besides the built-in allowlist
//	-raw               full: also write raw-<W>x<H>.log, the program's output bytes
//
// The bundle (-out DIR)
//
//	frames/<state>-<W>x<H>.ansi   one per Snapshot
//	flow.json      the run as data: per snapshot its state, size, profile, the keys,
//	               text and resizes sent since the previous one, elapsed ms, whether
//	               the screen was stable; and how the program ended
//	sequences.json what the program did to the terminal beyond SGR, per snapshot and
//	               in total with the snapshot each was first seen at: alt screen,
//	               synchronized output (2026), cursor hide/show and its final place,
//	               mouse modes (1000/1002/1003/1006), bracketed paste, focus reporting,
//	               titles (OSC 0/2), hyperlinks (OSC 8), clipboard (OSC 52), clears,
//	               queries, unknown sequences; and whether the terminal ended clean
//	sheet-<W>x<H>.png  a contact sheet per size, when cellshot is on PATH
//	raw-<W>x<H>.log    with -raw, full level only
//	README.txt     what the bundle holds and at which level
//
// Levels: full keeps everything as captured (typed text, titles, unknown sequences'
// bytes). privacy is for sharing a private app's screens: each frame keeps its layout
// cell for cell (widths, SGR styles, glyphs, box drawing, punctuation, whitespace) but
// every letter becomes x or X (by case) and every digit 0; URL-, email- and path-shaped
// runs are redacted whole. Key names and generic UI words survive so the frames can
// still be critiqued: enter esc tab space shift ctrl alt pgup pgdn home end backspace
// q help quit back next cancel keys yes no ok, any glyph (↑ ↓ ← → ?), and the letter
// after ctrl+/alt+/shift+; -allow adds words. Typed text and the command line are
// redacted the same way; titles' text, unknown bytes and -raw are not written. State
// names (the tape's) are kept. Privacy frames pass the same size checks and tuilint
// frames runs on them; what reads the words loses its meaning: the critic's copy and
// voice review, and tuilint spec --frames (S-GOLDEN, which looks for each footer's
// action words). The tuilint frames rules judge colour, layout and glyphs, and hold.
//
// The terminal is Charm's VT emulator (github.com/charmbracelet/x/vt v0.1.0) on a pty
// from github.com/charmbracelet/x/xpty v0.1.4, both pinned in go.mod with Ultraviolet at
// the kit's own commit: the emulator's cells are the kit canvas's cells, and a frame is
// written by the same Line.Render, so a capture of a kit app matches its harness frame
// cell for cell. (tuitest v0.2.0 was the alternative: it brings a forked emulator,
// tuios.dev/vt, libghostty and a CLI framework, for features a capture doesn't use.)
// The emulator answers the queries
// Bubble Tea sends (DA1, cursor position, OSC 10/11 colours, DECRQM) and reports
// synchronized output (2026) and grapheme widths (2027) as supported, as Ghostty and
// kitty do; a snapshot waits out a synchronized update in flight. Keys are sent in
// legacy xterm encoding (no Kitty keyboard protocol). Cells the program left at the
// default colours stay default (cellshot and tuilint draw them as the same #16161a)
// unless the program set the defaults with OSC 10/11, which are then painted in.
//
// Limits: Unix only (Linux, macOS, BSD, WSL). The pty's line discipline is the kernel's
// default until the program sets raw mode, so ctrl+c reaches a raw-mode TUI as a key
// and a line-mode program as SIGINT. WaitStable is a heuristic. No mouse input.
//
// tuicap -help, or a bare -h, prints this text (-h N is the height).
//
// Exit 0 when every Snapshot was written; 1 when the program exits before the tape
// ends (outside WaitExit), a WaitFor or WaitExit times out, a run exceeds -timeout, or
// a frame is not W×H; 2 on usage errors. flow.json and sequences.json are written
// either way, with the error.
package main
