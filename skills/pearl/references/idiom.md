# The idiom: what makes a terminal app look designed

Read from Crush's source (`internal/ui/`, October 2026), not from screenshots. A
screenshot gives you an impression. The source gives you the rules. Every move below is
implemented in the kit (`kit/ui/bubbletea`), so copy it from there instead of
re-deriving it.

*Bubble Tea: each move's `ui.*` call, and the Lip Gloss, `tea.View` and Ultraviolet names
in §4, §13 and §16, are Bubble Tea v2's; the moves hold for any framework.*

The idiom is fixed. A brand changes *how* each move is drawn (the theme's expressions and
glyphs), never *whether* the screen follows it.

## 1. Regions Are Separated By Space, Rules And One Background Step

No region gets a box. Hierarchy comes from:

- **whitespace**: a blank row between groups (or none, in a compact brand)
- **a section rule**: `Title ─────────── info`, the rule in a border colour barely lighter
  than the background (`ui.Section`)
- **a background step**: tool output, thinking and code sit on `bgPanel`, one step up
  from `bg`, at full width, with no border (`ui.ToolCall`)

The only box is the one thing floating above everything: a dialog. In Crush that is a
rounded border in the primary colour.

## 2. Two Ramps Do The Hierarchy

Four foreground steps (Crush: Sash `#ECEBF0` › Smoke `#BFBCC8` › Squid `#858392` › Oyster
`#605F6B`) and four background steps (Pepper `#201F26` › BBQ `#2D2C36` › Char `#3A3943` ›
Iron `#4D4C57`). Almost every distinction on screen is one of those steps, not bold and
not a border. In the kit: `fg › fgSoft › fgMuted › fgSubtle` and `bg › bgPanel ›
bgSubtle › bgOverlay`.

Values are brighter than keys. Metadata is muted. Hints are subtle. A screen with one
text level is a wall.

## 3. The Brand Lives In One Repeated Motif

Crush's motif is the diagonal `╱`. It fills the header after the wordmark, follows every
dialog title as a gradient run, and frames the logo with a staircase edge. One glyph,
repeated in three places, is an identity. Ten glyphs in ten places is noise.

Pick the brand's motif from the brand read (see `brand.md`), put it in
`glyphs.headerRule`, and let the header and dialog titles carry it.

## 4. Gradients Touch A Handful Of Cells

Crush uses a per-grapheme gradient (`lipgloss.Blend1D` over grapheme clusters) on exactly
five things: the wordmark, the compact header name, dialog-title rules, the queue
triangles and the spinner. Never prose, never a border, never a background.

## 5. Focus Is A Bar That Takes The Same Cells As No Focus

A focused item gets `PaddingLeft(1)` plus a `▌` left border. An unfocused one gets
`PaddingLeft(2)`. Both are two cells, so moving focus never shifts content by a column.
In the kit this is `ui.Gutter`, and every stream item and form field uses it.

Exactly one thing on screen is focused. Blurred-but-selected things stay visible in a
quieter tone, so the user can see where they were.

## 6. The Tool-Call Grammar

```
● Bash git status --short (background=true)
                                                     ← blank row
  M internal/ui/model/ui.go                          ← output on bgPanel, indented 2
  … (12 lines hidden) [click or space to expand]
```

- The status glyph is coloured by state (pending, running, `✓` done, `×` error), the name
  in the info colour, the main parameter muted, `k=v` options subtle.
- The `k=v` group is dropped before the main parameter would drop below 30 cells.
- A right-aligned note in subtle on the head row says how long the call took and how it
  was allowed (`allowed for session · 6s`), dropped before Param is cut, and a
  description is a muted line under the head, never a `desc=` option (`Tool.Meta`,
  `Tool.Desc`; an option with no value draws bare).
- The body is collapsed to N lines with an honest affordance. It never hides exactly one
  line (it shows that line instead).
- An error is an `ERROR` chip plus a one-line message, not a stack trace.

This generalises to any "an action happened" row: a deploy step, a job, a migration.
`ui.ToolCall` in the kit.

## 7. Tags, Not Badges Of Colour

State words on a filled chip: `ERROR`, `WARN`, `PLAN`, `DONE`, `FAILING`. They are padded
by one cell, bold, coloured by role, with on-role text. A tag plus a tinted bar spanning
the width is the terminal's alert banner (`ui.Alert`). The brand decides the shape:
filled, `[bracketed]`, or a bold word (`expressions.tag`).

## 8. Toasts Paint Over The Help Line

Crush doesn't take a new row for a notification. It paints a tag plus a message on a
tinted bar over the footer for a few seconds, then the help comes back (`ui.Toast`). The
layout never jumps.

## 9. Help Tells The Truth

Help labels change with state ("press again to cancel", "focus chat" vs "focus editor")
and with capability (`shift+enter` only if the terminal can tell it from `enter`). It is
packed greedily from the left, ends in `…` when cut, and never ends on a dangling
separator (`ui.Help`).

## 10. Buttons Are Padded Blocks With A Mnemonic

Padded filled blocks (`Padding(0,2)`): a strong fill when focused, the subtle background
otherwise, with the accelerator letter underlined. Hover and click use Lip Gloss
compositor layers as hit regions. Crush uses playful labels (`Yep!` /
`Nope`); your brand decides the voice. Destructive actions are coloured as danger even
when unfocused, and the safe one is the default.

## 11. Dialogs Share Chrome

Title plus a gradient motif rule, an input with a one-cell margin, the list, one help
line, all inside the one border. The selected row is a full-width primary bar with
on-primary text, and fuzzy-match characters are underlined. Right-aligned shortcut hints
hide uniformly when the row gets tight. Only the front dialog takes input; all stacked
dialogs draw. In the kit: `ui.Dialog` + `Canvas.Overlay`, with an optional scrim that
dims what is underneath.

## 12. Responsive By Breakpoint, Not By Squeezing

Crush drops the sidebar and switches to a one-line header below 120 columns or 30 rows,
caps prose at 120 cells, and lets diffs go full width (split view above 120). Decide your
breakpoints up front (`layout.md`).

## 13. The Program Owns Its Edges

`tea.View` sets the window title (`crush ~/path`), the terminal background (a choice:
`expressions.paintBg`), the terminal's own progress indicator while busy (`View.ProgressBar`),
and the program status (`View.ProgramStatus`). These cost nothing and make the app feel native.

## 14. Demote What Matters Less

Thinking text renders as colourless markdown on `bgPanel`, tail-windowed to 10 lines, and
closes with "Thought for 6.1s". Secondary information is not hidden; it is quieter.

## 15. One Frame Clock

All spinners run off one tick that is armed only while a visible item is animating, with
a generation number so stale ticks die. Frames are pre-rendered (`motion.md`). A product
clock (a poll, a sync) is not motion: it owns its own message, and a setting change
re-arms it.

## 16. Draw Into Rectangles

Crush's root model builds an Ultraviolet screen buffer and each region paints into its own
rectangle. Clipping is structural, so a long line can never wrap and tear the frame. Draw
order is z-order (dialogs last). The kit's `ui.Canvas` is this pattern
(`bubbletea-architecture.md`).

## Other apps worth reading for one idea each

- **Soft Serve**: tabbed repo views with a list + pager split, served over SSH.
- **Glow**: stash list + glamour pager, a status bar with scroll percentage.
- **Gum**: every component as a single command; the cleanest themed components.
- **gh-dash**: tabbed sections of tables with a toggleable preview pane.
- **Superfile**: multi-pane file manager; a counter-example of how far boxes can go.
- **Circumflex**: a dense list made readable purely by fg steps.
