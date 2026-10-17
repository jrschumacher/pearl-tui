# Brand: consistent across every screen, unmistakably this product's

Two failures look opposite and have the same cause. **Inconsistent:** every screen
invents its own colours and glyphs, so the app looks like five apps. **Generic:** every
screen is consistent, with the same pink-purple defaults as every other Bubble Tea app.
Both come from making visual decisions in screens instead of in one place, from the
brand.

The fix has three parts: a **brand read** (decisions, with reasons), a **theme file**
(the decisions as data), and a **lint** (proof the decisions differ from the default).

## 1. The Brand Read (Write It First)

Gather what exists: logo files, the site's CSS, the style guide, the typeface, the
product's own words, screenshots of its other surfaces. Measure; don't guess. Then write
`TUI.md` § Brand read (the only copy; the theme's `read` field holds it condensed):

```
palette   <the hues, measured, and which one does the pointing>
tone      <how the product talks; what it never does>
geometry  <round or square, light or heavy, soft or hard>
density   <airy or compact, and why the users need that>
type      <case, tracking, weight; what the wordmark looks like>
motif     <the one shape the brand repeats: a slash, a diamond, a sail, a dot>
remember  <the one move a user will remember, drawn on every screen: a glyph, a rule, a layout move>

<one line per expression below, tying the choice to a product trait>
```

A choice with no reason is a default in disguise. "Pill buttons because pills are nice"
is not a reason. "Pill buttons: the rounded button, because the brief says 'rounded
everything'" is.

## 2. Translate Traits Into Terminal Moves

| The brand is… | Terminal moves that say it |
|---|---|
| soft, calm, caring | `╌` dashed rules, `›` caret selection, `•` dots for steps, underline or pill buttons, a bold-word tag, minimal header, airy, sentence case, no scrim drama, dots spinner |
| industrial, blunt, exact | `━`/`═` heavy rules, a stamp header, `fill` selection, inverse block buttons, `[BRACKET]` tags, heavy dialog, UPPERCASE labels, compact, quadrant spinner, no gradients |
| precise, technical, nautical | a bar header, `┄` chart rules, slash tabs, `[ bracket ]` buttons, boxed keys, `▲ △` motifs, compact |
| playful, loud, retro | a half-block banner, `inverse` selection, double dialog, pill tabs, owned background, a gradient motif rule, field backdrop |
| editorial, bookish, quiet | lowercase titles and labels (`titleCase`, `labelCase: lower`), `·` dotted leaders, a thin `│` bar, filled-panel dialogs (no border, or `dialog: rules`: a rule above and below), underline buttons, measure 72; when the one hue is a state, `focus: fg` and `wordmark: plain` |
| a developer tool with personality (Crush) | a `╱` motif in the header and every dialog title, a gradient wordmark, `▌` bars, pill buttons, filled tags, playful copy (`Yep!`) |

These are starting points. Two brands with the same trait still differ on palette, motif,
wordmark and copy.

## 3. Brand Assets, Converted Honestly

- **Colours**: `palette derive <primary> [secondary] --bg <brand bg>` produces
  both role sets, tinted neutrals and contrast checks (`color.md`).
- **Typeface**: `scripts/wordmark.py "name" --font brand.ttf --style shade|half|ascii`
  rasterises the real typeface into cells. Tracked caps (`--style tracked`) when the mark
  is the hero.
- **Logo**: `scripts/logo2cells.py logo.png --cols 40` as a starting grid, then the
  rules in `cell-art.md`. The likeness needs ~40 columns. Smaller marks are drawn by hand.
- **Motif**: take it from the logo's geometry. Crush's logo is slanted, so its motif is
  `╱`. A diamond logo gives `◆ ◇`, a sail gives `▲ △`, a wave gives `∿` (with an ASCII
  fallback). One glyph, repeated in the header rule, dialog titles and the backdrop
  field.

## 4. The Theme File (The Contract)

One JSON per brand. Every screen reads it and none holds a literal. A non-Go renderer can
read it too.

```jsonc
{
  "name": "halyard", "brand": "halyard",           // brand = the CLI's own name
  "read": "one paragraph: the brand read, condensed",
  "roles": { "dark": { "primary": "#FF5A1F", … }, "light": { … } },   // color.md
  "glyphs": {
    "headerRule": "", "rule": "┄", "focusBar": "▎", "blurBar": "", "cursor": "▸", "prompt": "▸",
    "done": "◆", "current": "▲", "todo": "△", "fail": "✕", "pending": "●",
    "partial": "◭",              // in progress, between current and done (default ◐, ascii ~)
    "warn": "!",                 // degraded: must differ in shape from current and fail
    "bullet": "▸", "sep": "/", "arrow": "→", "ellipsis": "…",
    "radioOn": "◉", "radioOff": "○", "checkOn": "■", "checkOff": "□",
    "treeMid": "├─", "treeEnd": "╰─", "treeRail": "│", "thumb": "┃", "track": "│",
    "spinner": ["◢","◣","◤","◥"], "fieldA": "▲", "fieldB": "△",
    "ascii": { "◭": "~" }         // twins for glyphs the kit's map lacks (theme.ASCIITwin)
  },
  "expressions": {
    "header": "bar",            // rule | bar | minimal | stamp | banner
    "wordmark": "tracked",      // gradient | solid | inverse | tracked | bracket | upper | plain (fg, bold)
    "sectionTitle": "primary",  // accent | primary | muted-upper | fg
    "titleCase": "sentence",    // sentence | upper | lower
    "labelCase": "lower",       // upper | lower | as-is: table headers, tags, toast and summary
                                //   labels (unset: headers upper, tags as the app writes them)
    "selection": "fill",        // bar | fill | caret | inverse
    "button": "bracket",        // pill | bracket | underline | inverse
    "tag": "filled",            // filled | bracket | bold
    "dialog": "square",         // rounded | square | double | heavy | none (a filled panel)
                                //   | rules (a filled panel between two rules, title in the top one)
    "dialogTitle": "bar",       // rule (motif in a gradient) | plain | bar (unused under rules)
    "tabs": "slash",            // underline | pill | bracket | slash
    "footer": "rule",           // keys | bar | rule
    "keys": "boxed",            // inline | boxed
    "density": "compact",       // airy | compact
    "backdrop": "field",        // none | field
    "working": "spinner",       // spinner | scramble (Crush's hero; without it ui.Anim draws the spinner + label)
    "scrim": "dim",             // dim | none
    "focus": "primary",         // primary | fg (focus, cursor, prompt, selection in fg + bold)
    "paintBg": false,           // own the terminal background
    "gutter": 2, "measure": 100
  },
  "wordmark": { "rows": ["…"], "style": "half" },   // optional, from wordmark.py
  "ansi16": { "dark": { "warning": 11 }, "light": { "warning": 3 } }  // optional, color.md
}
```

**`focus: fg` is for a brand whose hue means a state** (amber = new, and nothing else).
One meaning per hue means focus can't be that hue, so the kit's own emphasis (focus bar,
cursor, prompt, current step, active tab, focused button, the fill selection) moves to
fg and weight and its marks go neutral; what the theme opts into (a bar or stamp
header, a gradient wordmark, `sectionTitle: primary`, filled tags) and what the app asks
for (`t.S.Primary`) keep primary, because the brand chose it.
`TestFocusFgKeepsPrimaryOffChrome` holds the kit to that split. At 16 colours pin the
hue with `ansi16` (color.md § The Four Palettes) so it doesn't land on red, error's slot;
`palette check` fails a focus-fg theme whose primary shares a slot with a state.

**A state ladder takes the status slots, by meaning.** `● new · ◐ reading · ○ waiting ·
✓ reviewed` is `current · partial · todo · done`: the app draws `t.G.Current`,
`t.G.Partial`, `t.G.Todo`, `t.G.Done`, never literals, so `-ascii` draws `* ~ o +` and
`palette check` keeps the rungs apart. `partial` is the in-progress rung (default `◐`);
the kit draws it for a busy `ui.Dot` and a running tool call, which would otherwise be
another state's glyph in another colour, and one mark under NO_COLOR.

**Keep a state's glyph and hue exclusive.** The kit's own components draw the same slots
with the same meanings: `ui.Steps` done/current/todo/fail, `ui.ToolCall`
pending/partial/done/fail, `ui.Dot` current (ok), partial (busy), warn, fail and todo
(idle), and `ui.Busy` the spinner (one cell, the partial glyph when the frames are wider).
So a ladder mapped onto the slots by meaning stays consistent everywhere. Where a rung
means something no kit component means (welp's `●` is "new", and `ui.Dot` draws
`current` for "healthy"), keep that component off the ladder's screens, or give the rung
its own field in the copied `Glyphs` (`ASCII` maps it and `CheckGlyphs` checks its twin;
add it to `glyphMeanings`' status set so the rungs stay apart). The hue stays off the
kit's chrome with `focus: fg`; a spinner frame may not be a status or focus mark
(`CheckGlyphs`), and under `-ascii` the kit's spinner leaves out the focus bar's `|`.

Missing roles are derived, and missing glyphs and expressions default, so a theme can
start small. An expression the kit doesn't know (`"dialog": "rule"` for `rules`, a key
from a newer kit) also draws the default, so `palette check` fails it and lists the
values it knows; `theme.Load` loads it, so an app never refuses to start over a word. Write every role as `#RRGGBB` or `#RGB`, `#` included: Lip Gloss reads
`6B50FF` as an ANSI index or no colour, so `theme.Load` and `palette check` reject it. A theme that keeps a role below the contrast floors on purpose (a brand's
exact hex, used mostly as a fill) says so in writing, and `palette check` reports it
as ACCEPT instead of FAIL:

```json
"contrastAccept": { "dark.primary": "Charple #6B50FF exactly as Crush ships it; used for fills and bars" }
```

Every glyph needs an ASCII twin, the theme's own in `glyphs.ascii` or the kit's map;
`palette check` and the kit's theme tests name a glyph with neither and two marks of one
meaning that share a twin (half-filled circles take `~` by convention). `theme.Load`
still loads it, so an app always starts; an unmapped glyph then draws as `?`, which
`tuilint frames --ascii` fails [F-ASCII]. If you add a key: add it to the Go structs,
default it, document it here, probe-render it at 80×24 and 120×40, and decide whether
`diversity` should count it; `theme.ASCII` and `theme.CheckGlyphs` cover every string in
`Glyphs` on their own.

**What never changes, whatever the brand:** whitespace and rules instead of boxes, one
box for the floating thing, three to four text levels, exactly one visible focus that
doesn't shift layout, buttons in a row, alerts as tag + message, help from real
bindings, and every frame exactly W×H.

## 5. Prove It's Different

```
diversity themes/*.json kit/themes/ink.json kit/themes/charm.json --baseline ink,charm --min-distance 8
```

Pass the baseline files (`kit/` is in the skill directory): a baseline is matched by
theme name, so without its file the min-distance rule has nothing to measure against,
and `diversity` exits 2 rather than pass a recoloured default. It fails when a brand theme is within 8 axes of the default or the Crush study (that's
the default with a new palette), when two brand themes share more than 9 axes, or when
one value is used by more than half the brands. It is a floor, not a target. A plain
brand stays plain and spends its differences on the two or three moves it owns; never
move an axis just to pass.

**Many brands at once** (a theme gallery, a design-system rollout across products):
parallel agents converge on the newest options. Six of nine picked the same new header
when it was added. After the first pass, write a **seating plan**: give each contested
value to the brand whose product actually has it, fix those columns, and let each brand
finish in its lane. Agents can't coordinate; the orchestrator must.

## 6. Render The Brand, Then Judge It

```
go run ./cmd/<app> -theme themes/<brand>.json -out frames/<brand> -sizes 80x24,120x40
go run ./cmd/<app> -theme themes/<brand>.json -out frames/<brand>-light -light
cellshot frames/<brand>/*-120x40.ansi --sheet <brand>.png --cell 8x16
```

Put it beside the baseline (`ink`) and one other brand. Ask:

1. With the colour stripped, can you still tell which brand it is? If not, identity is
   only in the palette. Move the expressions.
2. Is the "remember" line from the read visible in the first second?
3. Does every choice in the read trace to something on screen?
4. Does the light set look like the same brand, not a different app?

## 7. Keep It Consistent Over Time

- Record the brand read, the theme path and the decisions in the project's design context
  file (`TUI.md`, written by `init`). Every later change reads it first.
- `tuilint code` fails on hex outside the theme package [C-HEX] and on emoji
  [C-EMOJI]. `tuilint frames --theme` fails on any rendered colour that isn't a role
  or a blend of two roles [F-OFFPAL].
- New component? It takes `*theme.Theme` and nothing else. If it needs a new colour, the
  contract gets a new role, with a reason in the read.
