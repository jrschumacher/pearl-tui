# The medium: cells, glyphs, colour profiles

Everything you know from the web is about a continuous canvas. A terminal is a grid of
cells, and every bug and every ugly screen you will produce comes from forgetting that.
Read this before your first layout.

*Bubble Tea: the measuring calls in § Width Is Cells and the profile and background
detection in § Colour Profiles are Lip Gloss's and Bubble Tea v2's; the rules hold for any
framework.*

## The Cell Is The Pixel

One cell holds one glyph, one foreground, one background, and a few attributes (bold,
faint, italic, underline, reverse, strikethrough). Nothing is fractional. A region is an
integer number of columns wide and rows tall. There are no margins of 0.5, no 1px
borders, no anti-aliased edges, no z-index without compositing, no hover without mouse
mode.

A cell is about **1:2** (twice as tall as wide). Do geometry in square units (1 col = 1,
1 row = 2) or circles come out as ovals, a "square" panel is 2× as wide in columns as it
is tall in rows, and diagonals are ~63° instead of 45°.

## Width Is Cells, Never Bytes, Never Runes

- `len("─")` is 3. `len([]rune("日本"))` is 2, but it occupies 4 cells. Styled strings
  carry escape bytes that occupy zero cells.
- Always `lipgloss.Width(s)` / `lipgloss.Height(s)` for measuring, `ansi.Truncate(s, w,
  "…")` and `ansi.TruncateLeft` for cutting (both from `github.com/charmbracelet/x/ansi`),
  `lipgloss.NewStyle().Width(w)` or `MaxWidth(w)` for padding. `tuilint code` fails a
  `len()` or rune count used to size or pad text [C-LEN].
- **Every row of a frame must be exactly W cells and the frame exactly H rows.** One row
  at W+1 wraps, pushes everything down one line, and the top row scrolls off: the whole
  screen "tears". This is the most common bug in model-written TUIs. The verification
  loop enforces it (`tuilint frames --cols W --rows H`).
- Wide glyphs (CJK, most emoji) take 2 cells, and terminals disagree about some
  (emoji with variation selectors, `⚠` vs `⚠️`, some symbols in the East-Asian
  "ambiguous" class). If a glyph's width is in doubt, don't use it in a layout column.

## The Glyph Ladder (portability, safest first)

| Tier | Set | Examples | Use for |
|---|---|---|---|
| 0 | ASCII | `> - * \| + = #` | the fallback for everything below |
| 1 | Box drawing | `─ │ ╭ ╮ ╰ ╯ ┼ ━ ┃ ╌ ═ ╱ ╲` | rules, the one dialog, tree guides |
| 2 | Block elements | `█ ▀ ▄ ▌ ▐ ▔ ▁ ▏ ▎ ▍` | bars, focus markers, meters, half-block art |
| 3 | Shades | `░ ▒ ▓` | texture, cell art edges (never `░` on dark-on-dark) |
| 4 | Geometric + arrows | `● ○ ◆ ◇ ■ □ ▲ △ ▸ › ❯ → ✓ ✗ • · …` | status, steps, bullets, cursors |
| 5 | Braille, sextants, eighths | `⣾ ⠋ 🬀 ▁▂▃▅▇` | spinners, sparklines, sub-cell plots |
| ✗ | Nerd Font private-use | ` ` | only behind a config flag with a tier-0–4 fallback |
| ✗ | Emoji | `🚀 ✅ ❌ 📁` | never in chrome. Widths vary, colour clashes with your palette, and it reads as the AI default. |

Tiers 0–4 render in every modern terminal font. Tier 5 renders almost everywhere but
not in every review font. Ship tier 0 fallbacks for every glyph the theme defines and
use them when `TERM=linux`, when `NO_COLOR` is set *and* the user asked for ASCII, or
when a `--ascii` flag is passed.

**Non-obvious:** box-drawing and block glyphs are drawn by the terminal itself in
Ghostty, Kitty, WezTerm and iTerm2 (not by the font), so they join seamlessly across
cells. `╱╱╱╱` reads as hatching, `▌` as a solid bar, `━━━━━` as one line. That is why
these tiers can carry so much design: they are the terminal's vector graphics.

## Colour Profiles: You Are Designing Four Palettes

| Profile | Where you meet it | What happens to your hex |
|---|---|---|
| TrueColor | Ghostty, Kitty, WezTerm, iTerm2, Windows Terminal, VS Code | exact |
| ANSI256 | macOS Terminal.app, older tmux configs, many SSH sessions | nearest of 240 colours. Tinted greys collapse into the 24-step grey ramp. |
| ANSI (16) | Linux console, CI logs, some remote shells | the *user's* 16 theme colours. Your purple becomes their "magenta". |
| Ascii / NO_COLOR | `NO_COLOR=1`, piped output, dumb terminals | no colour at all. Bold, faint, reverse and glyphs carry everything. |

Bubble Tea v2 detects the profile and downsamples for you, so the screen still renders.
**Downsampling is not design**, though. At ANSI256 a subtle `bgSubtle` step can merge
with `bg` and the focused row disappears. At 16 colours a tinted neutral lands on a hue
(a warm page on yellow, warm hints on error's red), so the kit draws every neutral on
one of the four greys there and `palette check` reports a pin that moves one off. Under
NO_COLOR, a selected row that is marked only by background colour is invisible.

Rules:

- **Nothing by colour alone.** A selected row also gets a marker glyph (`▌`, `›`, `●`).
  An error also says "error". A focused panel also changes its title weight or glyph.
- **Render your key states at ANSI256, 16 colours and NO_COLOR** before you ship
  (`-profile ansi256`, `-profile ansi`, `NO_COLOR=1` in the kit's harness). Look at them.
- **Give the theme the profile.** Keep `tea.ColorProfileMsg` and resolve with
  `theme.ResolveProfile`: where a background step lands on the page's colour
  (`t.Distinct`), the kit draws an edge instead (a borderless dialog gets a rule above
  and below, a bar footer's keys lose the strip that hid them, text on a fill takes fg or
  bg via `t.Ink`). `theme.Resolve` assumes truecolour.
- **Pin the roles that must keep their meaning at 16 colours** with the theme's `ansi16`
  (`color.md` § The Four Palettes); `palette check` names the clash and the pin. Never a
  per-profile hex in a screen: `tuilint code` fails it [C-HEX] and the theme can't see it.
- **Respect the user's background.** Detect it (`tea.RequestBackgroundColor` →
  `tea.BackgroundColorMsg.IsDark()`), choose the light or dark role set (a program with
  no profile, background or `NO_COLOR` handling at all fails `tuilint code` [C-NOCOLOR]), and by default
  **do not paint the whole screen background**. Painting a full-screen bg is a strong
  brand move (Crush paints its own). It is a choice that needs both role sets reviewed, it
  breaks transparent terminals, and it fights the user's theme. If you paint it, paint
  every cell. A painted screen with unpainted holes (the default bg leaking through
  after a reset) is the worst of both.

## Attributes Are Typography

There is one font and one size. Hierarchy comes from:

1. **Lightness**: four text levels (fg › fgSoft › fgMuted › fgSubtle); three of them do
   80% of the work.
2. **Weight**: bold for titles and the one thing that matters on a row. Faint (SGR 2)
   is unreliable across terminals; prefer an explicit muted colour.
3. **Case and tracking**: `SMALL CAPS LABELS`, `T R A C K E D` wordmarks, sentence case
   for content.
4. **Position and space**: indentation, alignment columns, blank rows.
5. **Glyph**: a leading marker (`●`, `▌`, `›`) is the terminal's icon.
6. **Colour**: hue last, and only from roles.

Italic is rendered as oblique or not at all in many fonts. Use it for one voice (a
quotation, a placeholder), never for structure. Underline is for links and accelerator
letters. Reverse video is the honest NO_COLOR fallback for "selected".

## Motion Costs Bytes

Every frame is re-diffed and written to a pty. Bubble Tea's renderer only sends changed
cells, so a still screen costs nothing, and a full-screen gradient animated at 60 fps
costs ~100 KB/s over SSH. Budget:

- Idle screens do not tick. Stop the spinner's ticks when nothing is loading.
- One 50 ms clock (`ui.FrameInterval`, 20 fps); spinners advance every 2nd frame
  (100 ms). `motion.md` § Budget has the table.
- Animate small regions. A gradient sweep across a 10-cell title is fine; a full-screen
  shimmer is a battery bug.
- Phase from the frame index of that one clock (`m.frame`, which the harness renders
  with `-frame N`), so a golden and the live program show the same frame. Wall-clock
  only for an elapsed timer.

## The Two Sizes

Design at **80×24 first** (the default of nearly every terminal and the floor most users
hit in split panes), then **120×40** (a comfortable laptop pane), and check **200×60**
(an external monitor). The same design at all three: more room, not a different app.
Decide in advance what collapses (sidebar → row, long help → short help, details
column → hidden) and at which width. Below the minimum (the kit's is 60×18), show a
centred block (`ui.TooSmall`): the wordmark, "Terminal too small", "52×16, needs 60×18",
the `toosmall` row's key from its action table, never a garbled layout.
