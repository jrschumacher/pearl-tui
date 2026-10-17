# Motion: one clock, deterministic frames, motion only when something moves

Motion in a terminal means "something is happening". If nothing is happening, nothing
moves. Every frame is bytes through a pty, and over SSH at 60 fps a full-screen shimmer
costs a megabyte a minute.

## One Shared Clock

```go
type tickMsg struct{ gen int }

func (m *model) animating() bool { return m.busy || m.screen == splash }  // visible motion only

func (m *model) tick() tea.Cmd {
    if !m.animating() { return nil }                                     // idle = no ticks
    return tea.Tick(ui.FrameInterval, func(time.Time) tea.Msg { return tickMsg{m.gen} })
}

case tickMsg:
    if msg.gen != m.gen { return m, nil }   // a stale clock dies
    m.frame++
    return m, m.tick()
```

- All spinners and pulses read `m.frame`. Never one `tea.Tick` per spinner.
- Re-arm the clock (with a new `gen`) when something starts animating. It stops itself
  when `animating()` turns false.
- `ui.FrameInterval` is 50 ms (20 fps). Spinners can advance every 2nd frame (100 ms).

## Deterministic Frames

Every animated thing renders from `(frame, seed)` only: `Anim.Frame(n, elapsed)`,
`ui.Spinner(t, n)` (beside a label), `ui.Busy(t, n)` (in a status column: one cell like
the dots beside it, the partial glyph when the theme's frames are wider),
`Field{Drift: n/3}`. So the harness can render frame 24 in a
golden, frames 0..40 into a GIF, and the live program shows the same thing. Phase from
wall-clock (`time.Since(start)`) only for things that must stay in sync with real time
(an elapsed timer).

## Choosing A Spinner (It Has A Voice)

| Spinner | Reads as | Use for |
|---|---|---|
| `ui.Anim` (gradient scramble + label + ellipsis/elapsed) | alive, branded, "thinking" | the one long-running hero action (an agent working, a deploy) |
| arc `◜ ◠ ◝ ◞ ◡ ◟` | calm, circular | a background sync, a row-level "in progress" |
| braille `⠋ ⠙ ⠹ ⠸` | busy, technical | dense tables of jobs |
| quadrants `▖ ▘ ▝ ▗` | mechanical, 8-bit | industrial or retro brands |
| dots `·  ··  ···` | soft, patient | calm brands (health, writing) |
| line `▁▃▅▇▅▃` | live audio, throughput | streaming, network |

Put the theme's choice in `glyphs.spinner`. One hero animation per screen, and row-level
spinners in the primary colour. A spinner always has a subject ("deploying 2/3 pods",
not "Loading...") and, after ~3 s, an elapsed time.

## The Hero Animation (Crush's Pattern)

`ui.NewAnim(t, size, label, seed)`: a run of glyphs scrambling through
`0-9a-fA-F~!@#…`, coloured along a primary→secondary→primary ramp that flows one step per
frame. Columns materialise from `.` at staggered, seeded frames (so two spinners don't
march in lockstep), followed by the label in soft fg and an animated ellipsis, or an
elapsed timer. It draws only when the theme sets `expressions.working: scramble`;
otherwise `Anim.Frame` is the theme's spinner + label, because the scramble is Crush's
signature and a brand must choose it. Pre-render frames if you have many on screen.

## Transitions

Terminals don't tween layout, and you shouldn't try. What works:

- **Appear**: draw in place. A dialog appearing over a scrim is enough motion.
- **State change**: change the glyph and colour of the one cell that changed (`●` → `✓`).
  A one-frame flash (the row in `bgSubtle` for 150 ms) is the most you need.
- **Toasts**: appear over the footer and vanish after ~4 s. No slide.
- **Progress**: the fill advances by real progress, never on a timer.

## Brand Motion (Splash, Intros, Idle)

See `cell-art.md` (rules 10–17): colour-field animation over a fixed glyph grid,
transform animation re-quantised per frame, fluid metaballs, decode/typewriter, temporal
lint, and intros that play once and hand off to an identical idle frame.

## Budget

| Thing | Interval | Region |
|---|---|---|
| hero anim | 50 ms | ≤ 30 cells |
| row spinner | 100 ms | 1 cell |
| field drift | 160 ms | full screen (static otherwise) |
| toast TTL | 4 s | footer |
| cursor blink | terminal's own (`tea.Cursor{Blink: true}`) | 1 cell |

## DO / DON'T

- **DO** stop ticking when nothing visible moves (`TestOneClock`, copied with the kit,
  asserts `tick()` is nil in idle states). **DON'T** leave a `tea.Every` running on an
  idle screen, or tick faster than 33 ms [C-FASTTICK].
- **DO** give every animated component a `Frame(n, …)` like the kit's
  `Anim.Frame(n, elapsed)`. **DON'T** animate with state you can't reproduce in a test.
- **DO** say what is happening. **DON'T** ship "Loading..." [M2].
- **DO** keep the hero animation small (a label's width). **DON'T** shimmer the whole
  screen.
