# audit — the technical pass, with evidence

Mechanical correctness, not taste (that's `critique`). Output: a P0–P3 table with
evidence and the command that reproduces each finding.

## Checks

| Area | How | P0 if |
|---|---|---|
| Footprint | `-golden` at 80×24, 120×40; `go test` at the kit's five sizes (60×18 … 200×60); resize to 250×70 | any row ≠ W or rows ≠ H |
| Resize | is `tea.WindowSizeMsg` handled; what renders below the minimum | the layout is hard-coded |
| Blocking | `tuilint code` C-SLEEP/C-VIEWIO; grep `Update`/`View` for I/O | anything slow on the UI goroutine |
| Version | imports: `charm.land/*/v2` | v1 in new code (P1) |
| Colour | `tuilint frames --theme` F-OFFPAL/F-DEFAULT; `tuilint code` C-HEX; `palette check` | text under 2.3:1 |
| Profiles | renders at ansi256, ansi, ascii; `NO_COLOR=1` | focus or selection invisible without colour |
| Light | `-light` renders; `BackgroundColorMsg` handled | unreadable on a light terminal |
| Glyphs | emoji, Nerd Font PUA, width-ambiguous symbols in columns; ASCII fallbacks exist | a column misaligns |
| Keys | every help key wired per state; ctrl+c everywhere; q vs inputs; esc everywhere | a key in the help does nothing |
| Clock | ticks while idle (log the tick count over 5 s idle) | CPU burn while idle (P1) |
| Output | `fmt.Print` while running; logs to stdout | frame corruption |
| Exit | the terminal is restored after quit, panic and `tea.Suspend` (if ctrl+z is bound); program status cleared | broken terminal after exit |
| Pipes | behaviour when stdout is not a TTY; `--json` path | a TUI forced into a pipe |
| Perf | render time at 200×60 with `TUI.md` § Async `largest:` (10k items if it names none; `testing.B` on `render`) | > 16 ms per frame (P1) |

## Report

```
P0  Footprint   deploy@100×30 row 17 is 101 cells    go run ./cmd/app -golden -w 100 -h 30
P1  Clock       ticks while idle on services (20/s)  log tickMsg count; animating() always true
…
```

Every finding has the command that reproduces it. Fix P0s before anything else; P1s
before showing the user.
