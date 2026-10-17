# Layout: column budgets, breakpoints, rectangles

## Write The Budget Before Any Code

For every screen, at 80×24 (the floor) and 120×40 (comfortable), write down:

```
80×24                                120×40
row 0      header (1)                header (1, or a 5-row banner if h ≥ 30)
row 1      gap                       gap
row 2–3    tabs + underline (2)      tabs (2)
row 4      gap                       gap
row 5–21   body (17)                 body (33): table 74 | gap 3 | detail 40
row 22     gap                       gap
row 23     footer / help (1)         footer (1)
cols       gutter 2 | 76 | gutter 2  gutter 2 | 116 | gutter 2
```

The footer is one row bare or on a bar and two under `footer: rule` (the rule, then
the keys: rows 22–23 at 80×24), so a rule-footer brand's body budget is one row shorter.
Measure it, `len(ui.Footer(…))`, rather than writing 1.

Every pane's outer width, inner width and text width are integers that sum to the
parent. If you can't write this table, you can't build the screen. It is also the first
thing to check when a frame tears.

## Breakpoints Collapse On Purpose

Decide what changes and at which width, and write it down:

| Width | Change |
|---|---|
| < 60 or < 18 rows | `ui.TooSmall`: "Terminal too small", "52×16, needs 60×18", "q quit", centred, nothing else |
| 60–109 | single column. The detail pane becomes one summary line under the list. The sidebar becomes a header row. Long help becomes short help. |
| ≥ 110 | detail pane / sidebar on the right (30–40 cols) |
| ≥ 140 | diffs default to side by side. A product toggle (`t unified`) may offer it narrower; below ~100 columns each half gets < 45 cells, so the dialog says the toggle and cuts the long lines honestly with `…` |
| h ≥ 30 | a banner header is allowed; below that it collapses to one row |

The rule: **one design at every size, more or less room**. Nothing exists at one size
that is absent at the other except what the table says collapses.

## Draw Into Rectangles, Not Strings

String layout (`JoinVertical`, `JoinHorizontal`, `Place`) is fine for a component. For a
screen, use a canvas:

```go
c := ui.NewCanvas(t, w, h)                     // uv.ScreenBuffer underneath
var main, side uv.Rectangle
layout.Horizontal(layout.Fill(1), layout.Len(3), layout.Len(40)).
    Split(body).Assign(&main, new(uv.Rectangle), &side)
c.Draw(strings.Join(tableRows, "\n"), main)    // clipped to main: can't wrap, can't tear
c.Draw(detail, side)
c.Overlay(dialog, nil)                         // last: draw order is z-order
```

`github.com/charmbracelet/ultraviolet/layout` takes `Len`, `Min`, `Max`, `Percent`,
`Ratio{Num, Den}` and `Fill`. Recompute the layout in `View` (it's cheap) and resize
child components only when a rectangle changes.

## Fixed Chrome First, Body Last

Draw (or measure) the header and footer first. The body gets whatever is left. Never
let body content push the footer off the screen. Overflow scrolls inside the body, with a
visible "↑ 25 more lines" affordance.

## Rows Go By Content

- **DO** let a stacked pair share rows by content: the upper region takes the rows it
  fills (capped at a share), the lower takes the rest. **DON'T** give each a fixed
  half: a five-row collection over a long response leaves rows 6–10 blank while the
  response is cut at row 11.
- **DO** keep the working surface (the list the user acts on) at least half the body
  rows at the floor, and fold a readings panel's secondary rows into the row above
  (`cpu 42% · mem 61%`, not two rows) before taking rows from it. **DON'T** let "one
  row per reading" win: at 80×24 it takes a 13-row list down to 8.

## Gutters And Rhythm

- One gutter for the whole app (`expressions.gutter`, usually 2). Content never touches
  column 0.
- Lists and tables start at the gutter; centring is for a splash, an empty state and a
  dialog (`anti-patterns.md` S3). A centred list has no column to scan down.
- Vertical rhythm is one blank row between groups (`t.Gap()`), or none in a compact
  brand. Never two blank rows. If a screen looks empty with one, the content is too
  thin, not the spacing too tight.
- Align on columns. Labels in a KV block share a key width. Numbers right-align. Times
  are fixed-width.

## The Measure

Prose is capped at 72–100 cells (`expressions.measure`) even in a 200-column terminal.
Tables, logs and diffs may use the full width. A wide terminal is room for a second pane,
not for longer lines.

## Use The Room

At 120×40 a screen with 30 blank rows is under-designed. Tall terminals get the next most
useful thing: an activity feed under the table, a preview pane, the full help. Wide
terminals get a detail pane. Don't stretch; add. `tuilint frames` warns on a frame over
98.5% blank [F-EMPTY].

## Overlays

- Centred dialogs with a max size (Crush: 70×20), clamped to the screen.
- Popovers anchored to their trigger (a completion list above the cursor line), clamped
  to the screen edges.
- Dim what's underneath (`Canvas.Scrim`) unless the brand says otherwise, and redraw the
  footer above the scrim because it now describes the dialog's keys.
- Below the minimum size, dialogs go fullscreen.

## DO / DON'T

- **DO** budget columns for 80×24 first. **DON'T** design at your own 200-column terminal.
- **DO** draw into rectangles. **DON'T** concatenate screen-height strings and hope.
- **DO** collapse panes at named breakpoints. **DON'T** shrink every pane proportionally
  until all of them are useless.
- **DO** handle `tea.WindowSizeMsg` from the first commit. **DON'T** default to 80
  and forget.
