# Colour: roles, ramps, and the four palettes you are really designing

## Roles, Never Hex

A screen asks for `t.S.Muted` or `t.P.Error`, never `"#8A93A3"`. The theme file owns
every colour (`kit/themes/*.json`), and `theme.Resolve` turns it into styles. If a screen
needs a colour the roles can't express, add a role to the contract. Do not reach for a
literal.

| Role | Job | Floor (WCAG vs its bg) |
|---|---|---|
| `primary` | the brand hue: focus bar, selection, current, header motif | 3.5 as text |
| `secondary` | second voice: primary button fill, gradient end | 3.0 |
| `tertiary` | prompts and carets | 4.5 |
| `accent` | section titles: reads "heading", never "alert" | 4.5 |
| `bg › bgPanel › bgSubtle › bgOverlay` | the background ramp | border/bgSubtle 1.15–1.9 |
| `fg › fgSoft › fgMuted › fgSubtle` | the text ramp | 7 › 8 › 4.5 › 2.6 |
| `border` | rules: barely there, but there | 1.15–1.9 |
| `success error warning info` | state only | 3.5 |
| `onPrimary onSecondary onError…` | text on a filled role | 4.5 |

## Derive The Palette From The Brand, Then Check It

```
scripts/palette.py derive "#FF5A1F" "#3D7CC9" --bg "#0D1724" --light-bg "#F4F6F8" --swatch s.ansi
scripts/termshot.py s.ansi s.png      # look at it
scripts/palette.py check theme.json   # on any existing theme
```

`derive` works in OKLCH so lightness steps are perceptual. It:

- keeps the brand's hue and chroma for `primary`/`secondary` and moves only lightness
  until the colour clears its floor
- **tints every neutral toward the background's hue** (or the primary's, on a neutral
  bg). This is the single biggest difference between a palette that reads "designed" and
  one that reads "default grey".
- pulls status colours into one lightness and chroma band, so green, red and amber sit
  together instead of shouting at different volumes
- picks the on-colour ink (light or dark) by contrast for every fill
- writes both a dark and a light set, plus the nearest xterm-256 index per role

Then tune by hand. The tool gets you a correct palette; the brand read decides whether
the tertiary should be a complement or a quiet tint.

**Measure, don't guess.** Take brand colours from the logo file, the site's CSS or the
style guide. A "close enough" blue is a different brand.

## Two Ramps, Small Steps

Crush's four background steps are ~5–8 L apart in OKLCH, and its border is the second
step. Small steps look calm. If you can see the panel edge from across the room, the step
is too big.

- `bgPanel` (+~0.035 L): recessed blocks (tool output, code, thinking, cards)
- `bgSubtle` (+~0.075 L): unselected buttons, inactive chips, a dim focus bar
- `bgOverlay` (+~0.13 L): popovers, completions, hover

## Light Mode Is Not An Inversion

Keep the hue and shift the lightness. Primary usually gets darker and more saturated to
clear 3.5:1 on a near-white. Warnings move from yellow to a dark amber (yellow text on
white is invisible). Borders become *darker* than bg. Test the light set as carefully as
the dark one; most themes ship with an unreviewed light set.

Detect the terminal: `tea.RequestBackgroundColor` in `Init`, then on
`tea.BackgroundColorMsg` call `theme.Resolve(file, msg.IsDark(), ascii)`. Never assume
dark.

## Painting The Background Is A Brand Decision

`expressions.paintBg: true` sets `tea.View.BackgroundColor`, so the whole terminal takes
the brand's bg (Crush does this). It is a strong identity move, but it:

- ignores the user's terminal theme and transparency
- obliges you to review both role sets (light users get your bg too)
- needs every filled span to carry its own background, or resets punch holes. Draw on
  the canvas and `Backfill`, or use `ui.OnBg` for strings.

Most tools should not. A product with a strong visual brand (a game tool, an AI agent with
a mascot palette) may.

## Gradients

- `theme.Grad(text, a, b, bold)` blends per grapheme cluster in CIELAB
  (`lipgloss.Blend1D`).
- Use it on 5–40 cells: the wordmark, a motif rule, a spinner, a progress fill. Never on
  prose, borders, or backgrounds.
- Two stops from the brand, adjacent in hue. A three-hue rainbow is a toy.
- Under ASCII/NO_COLOR it degrades to plain bold text.

## Derived Tints Instead Of New Colours

Need a diff background, a selection tint, an alert bar? Blend a role toward bg
(`theme.Mix(p.Success, p.Bg, 0.82)`) instead of inventing a hex. Crush derives its diff
backgrounds this way from one hue (20%/13% for inserts, 15%/8% for deletes), which is why
any theme gets harmonious diffs.

## Foreign Colour

Subprocess output (`ls --color`, test runners, logs) arrives with the ANSI-16 palette of
someone else's terminal. Crush remaps the 16 basic colours onto on-brand legible ones
before drawing. At minimum, render foreign output on `bgPanel` so it reads as a quotation.

## The Four Palettes

TrueColor is what you design. ANSI256, ANSI16 and none are what many users get. Check:

```
go run ./cmd/<app> -out frames -profile ansi256
go run ./cmd/<app> -out frames -profile ascii
```

and look at the PNGs. What usually breaks:

- the `bg`/`bgSubtle` step merges at 256 colours, so selection disappears. Move it further
  apart, or add the glyph marker.
- `fgMuted` and `fgSubtle` collapse to one grey at 16 colours. Acceptable if nothing
  depends on the difference.
- at `ascii`, everything is the same colour. Focus, selection, error and disabled must
  still be visible through glyphs, bold, reverse video and words.

## DO / DON'T

- **DO** derive every role from 1–3 measured brand colours. **DON'T** start from Lip
  Gloss's example purple or Bubbles' default pink.
- **DO** tint neutrals toward the brand. **DON'T** use `#888`.
- **DO** keep hue for state and brand, and lightness for hierarchy. **DON'T** colour
  columns to tell them apart.
- **DO** check contrast for every text role on every background it sits on (fg on
  bgSubtle too). **DON'T** trust a palette you haven't run through `palette.py check`.
- **DO** give every filled element an explicit on-colour. **DON'T** put white on yellow.
