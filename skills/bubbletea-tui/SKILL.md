---
name: bubbletea-tui
description: Design and build terminal UIs with Bubble Tea v2, Lip Gloss and Bubbles that look as considered as Charm's Crush and unmistakably like the product's own brand, not like every other Bubble Tea app. Use when the user wants to build, design, redesign, restyle, theme, brand, critique, audit, polish, harden or animate a TUI, CLI dashboard, interactive CLI, terminal wizard, chat/agent interface, splash screen, terminal logo or wordmark, or port a v1 Bubble Tea app to v2. Covers layout and column budgets, colour roles and contrast, glyph vocabulary, focus and keys, empty/loading/error states, motion, cell art, a theme contract per brand, and a render-lint-critique loop that lets you see the terminal before the user does. Ships a compile-verified Go kit (theme engine, components, reference app), an honest ANSI-to-PNG renderer, a palette deriver and a slop linter. Not for web UIs or non-Go TUI frameworks.
license: MIT
compatibility: Go ≥ 1.26.7 (Bubble Tea v2.1, Lip Gloss v2.0.6, Bubbles v2.2). Python 3 with Pillow (and numpy for wordmark/logo tools).
metadata:
  version: "0.1.0"
---

# Bubble Tea TUI

You are about to design for a medium you cannot see, with libraries whose examples
taught every model the same pink-and-purple boxes. Left to your defaults you will
produce a TUI that works and that nobody would mistake for a designed product: rounded
borders around every region, `#7D56F4`, emoji for icons, "Loading...", a help paragraph,
and a frame that tears at 79 columns.

This skill replaces those defaults with three things: **an idiom** (how a terminal app
looks designed, read from Crush's source), **a brand system** (one theme file per
product, provably not the default), and **eyes** (a harness that renders every state, a
renderer that turns ANSI into honest PNGs, a linter, and an adversarial critic). Commit
to all three. A half-applied idiom reads as noise.

## Setup (Every Time, Before Any UI Code)

1. **Read the design context.** If `TUI.md` exists at the project root, read it: it holds
   the brand read, the theme path, the screens and past decisions. If it doesn't exist,
   run `teach` first (or, for a one-off screen, ask the 2–3 questions in
   `references/cmd-teach.md` and say what you assumed).
2. **Load the command's reference** from the table below, plus the domain references it
   names. Load nothing else up front.
3. **Start from the kit, never from prose.** `kit/` is a compiling Bubble Tea v2 module:
   `theme/` (engine), `ui/` (components), `cmd/showcase/` (a full reference app with the
   harness), `themes/` (seven brand themes). Copy and adapt it. A cold rebuild from
   descriptions loses half the idiom at once.
4. **Never show the user anything you haven't rendered and looked at** with
   `scripts/termshot.py`. Never ask them for a screenshot.

## The Laws

1. **The cell is the pixel.** Every frame is exactly W×H cells. Measure with
   `lipgloss.Width`, cut with `ansi.Truncate`, and draw into rectangles that clip
   (`ui.Canvas`). One row at W+1 tears the whole screen.
2. **Space and rules, not boxes.** Regions separate by whitespace, a faint section rule,
   or one background step. The only box is the one thing floating above everything (a
   dialog).
3. **Lightness is hierarchy, hue is meaning.** Four text levels and four background steps
   do the hierarchy. Hue is reserved for brand and state.
4. **Every colour and glyph comes from the theme.** Screens hold no hex and no glyph
   literals. The theme comes from the brand, measured.
5. **Exactly one visible focus, and it never moves the layout.** The focus marker takes
   the same cells focused or not.
6. **One action table per state drives both the keys and the help.** A test presses
   every advertised key in every state. Help is never a paragraph and never a lie.
7. **Every state is designed and rendered:** empty, loading, error, partial, too small,
   and every overlay. If the harness can't render it, nobody reviewed it.
8. **Motion means something is happening.** One clock, ticking only while something
   visible moves. A spinner always says what it is doing.
9. **80×24 first, then 120×40, the same design.** Collapse at named breakpoints; use the
   extra room, don't stretch into it.
10. **The brand owns two or three moves.** Pick them in the brand read and make them
    visible on every screen. Everything else stays quiet.

## Domains (Summary, Then Consult The Reference)

### The medium → *Consult `references/medium.md`*
**DO** budget in integer cells, use the glyph ladder (ASCII → box drawing → blocks →
geometric → braille), and design for TrueColor, 256, 16 and no colour.
**DON'T** use emoji or Nerd Font icons in chrome, measure with `len()`, or assume a dark
background.

### The idiom → *Consult `references/idiom.md`*
**DO** use a header with the brand motif, section rules, `▌` focus gutters, the tool-call
grammar (`● Name param (k=v)` + an indented panel body), tag chips, toasts over the help
line, and dialogs with shared chrome and a scrim.
**DON'T** reproduce the Lip Gloss README layout, connect tabs with borders, or nest
boxes.

### Colour → *Consult `references/color.md`*
**DO** derive roles from measured brand colours (`scripts/palette.py derive`), tint the
neutrals toward the brand, check contrast in both modes, and use gradients on five to
forty cells.
**DON'T** ship `#7D56F4`/`#FF5FAF`/205/62/170, pure greys on a tinted bg, colour-only
signals, or holes in filled bars.

### Layout → *Consult `references/layout.md`*
**DO** write the column budget before the code, draw into `uv` rectangles with
`layout.Horizontal/Vertical`, cap prose at the measure, and add panes on wide terminals.
**DON'T** centre lists, hard-code 80, or let content push the footer off screen.

### Components → *Consult `references/components.md`*
**DO** copy `kit/ui`: Header, Tabs, Section, List, Table, KV, Dialog + Overlay, Button,
Tag, Alert, Toast, Footer/Help, ToolCall, Thinking, Progress, Sparkline, Tree, Empty,
TooSmall.
**DON'T** leave a bare "No items", focus the destructive button, or let a blurred list
lose its selection.

### Interaction and copy → *Consult `references/interaction.md`*
**DO** use `key.Binding`s, conventional keys, `esc` everywhere, a command palette, a
Cmd for everything slow, cancel paths, and copy that names the thing and the next step.
**DON'T** bind `q` while an input is focused, block in `Update`, or write "An error
occurred" or "Successfully".

### Motion → *Consult `references/motion.md`*
**DO** run one `gen`-stamped clock, render deterministic frames from `frame`, and keep
one hero animation (`ui.Anim`) with a subject.
**DON'T** tick while idle, fake progress, or shimmer the screen.

### Brand → *Consult `references/brand.md`*
**DO** write the brand read first, translate traits into expressions, convert real
assets (`wordmark.py`, `logo2cells.py`), and prove distance from the default
(`scripts/diversity.py`).
**DON'T** treat a new palette on the default layout as a brand, or move an axis just to
pass the lint.

### Cell art → *Consult `references/cell-art.md`*
**DO** use one vocabulary per edge, floor the faintest level, snap stems to the grid,
measure the source, and keep keyframes as code.
**DON'T** use figlet's "standard", `░` on dark-on-dark, or splashes that outlive the
work.

### Architecture → *Consult `references/architecture.md`*
**DO** use Bubble Tea v2 (`charm.land/*/v2`, pinned), one root model, `View() tea.View`
with declarative fields, a canvas, caches, and the harness flags from the first commit.
**DON'T** write v1 imports, nested `tea.Model` trees for every region, or `fmt.Print`
while running.

## The TUI Slop Test (Run It On Every Screen)

Would someone who lives in a terminal look at this and say "a model made this"? Check
`references/anti-patterns.md` (≈40 named defaults, most linted by
`scripts/tuilint.py`), then:

1. Squint: which product is this? If the answer is "a Bubble Tea app", go back to the
   brand.
2. Count borders. More than one box needs a reason.
3. Find the focus in under a second. Exactly one.
4. `-profile ascii`: are focus, selection and errors still visible?
5. 80×24 and 200×60: the same design, nothing torn, room used.
6. Read every word aloud in the brand's voice.
7. Any pink-purple, emoji or "Loading..."? Remove it.
8. Name the one thing a user will remember. If you can't, it has no identity yet.

## Commands

The user may name a command (`/bubbletea-tui critique`), or you route from intent. Load
the reference, follow it, and hand off to the next command when it says so. Never
auto-run a command the user didn't ask for; suggest it.

| Command | Use when | Reference |
|---|---|---|
| `teach` | no `TUI.md`; new project; the brand changed | `references/cmd-teach.md` |
| `brand` | create or change a theme from brand assets | `references/cmd-brand.md` |
| `shape` | plan screens: budgets, states, keys, sketches | `references/cmd-shape.md` |
| `craft` | build screens from the kit, with the loop | `references/cmd-craft.md` |
| `critique` | design review: scored, cited, two critics | `references/cmd-critique.md` |
| `audit` | technical pass: footprint, profiles, keys, clock, exit | `references/cmd-audit.md` |
| `polish` | the final 10% on a working app | `references/cmd-polish.md` |
| `harden` | sizes, data, terminals, failures, accessibility | `references/cmd-harden.md` |
| `animate` | spinners, transitions, intros, idle motion | `references/cmd-animate.md` |
| `art` | logo, wordmark, field, splash | `references/cmd-art.md` |
| `restyle` | an existing app onto the theme, idiom and v2 | `references/cmd-restyle.md` |

No command named and the user wants a new TUI: `teach` → `brand` → `shape` → `craft`,
checking in with the user after `shape`.

## Tools

| Script | Does |
|---|---|
| `scripts/termshot.py` | ANSI → PNG/GIF/contact sheet, drawn like a modern terminal; reports tofu, emoji, uneven rows |
| `scripts/tuilint.py` | `frames`: width, emoji, default palette, off-palette, contrast, boxes, rainbow, levels… `code`: len(), blocking, hex, emoji, v1, resize, ticks, Print, keys |
| `scripts/palette.py` | `derive` a dark+light role set from brand hexes (OKLCH, tinted neutrals, contrast-tuned); `check` a theme |
| `scripts/diversity.py` | how different are these themes from each other and from the default |
| `scripts/wordmark.py` | the brand name in cells, from a real font: half / shade / ascii / tracked |
| `scripts/logo2cells.py` | a logo raster onto the grid (half, shade or quadrants) with lint |

The verification loop that ties them together is `references/verify.md`. Run it before
every hand-off.

## The Kit

`kit/` (module `tuikit`): `go test ./...` renders seven themes × dark/light/ASCII × every
state × five sizes and asserts the footprint and contrast. `go run ./cmd/showcase
-theme themes/halyard.json` runs the reference app. Its `-out` flag writes every state
for `termshot.py`. Copy `theme/` and `ui/` into the project and rename the import path.

You can build terminal interfaces people screenshot and share. The libraries are
capable of it, and Crush proves it. The difference is decisions: make them in the brand
read, encode them in the theme, and check every one of them with your own eyes before
anyone else sees the screen.
