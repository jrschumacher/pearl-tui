# Components: anatomy and rules

Every component here exists in `kit/ui`. Copy it and change what the task needs. Each
entry gives the anatomy, the rules that a cold rebuild got wrong, and the kit function.

## Header — `ui.Header(t, w, h, brand, details)`

One row (or a banner, if the brand earns it and h ≥ 30): the wordmark, the brand motif
filling the gap, and details on the right (`env · region · selection`, the first part
in fg and the rest muted). It is drawn per `expressions.header`: `rule`, `bar` (a filled
strip), `minimal`, `stamp` (the name as an inverse stamp at the right), or `banner`.
Details truncate before the brand does.

## Tabs — `ui.Tabs`

Text, never boxes. The active tab is bold with a marker drawn per `expressions.tabs`
(a `▔` underline row, a pill, `[brackets]`, or a ` / ` path). Counts sit beside the
names in subtle; a count is only red if it needs attention now.

## Section — `ui.Section(t, title, w, info)`

`Title ───────── info`. The title is in the theme's title style and case, the rule is in
the border colour, and right info is subtle. This is how regions separate.

## List — `ui.List(t, items, sel, offset, w, h, focused)`

- The marker column is always 2 cells, so selecting never shifts text.
- Selection follows `expressions.selection`: `bar` (▌ + bold), `fill` (a full-width
  primary bar; every span carries the bg), `caret` (› + primary), `inverse`.
- Blurred lists keep the selection visible in a quieter style.
- Title in fg, description muted after it (dropped first when tight), right-aligned
  hint in subtle, fuzzy matches underlined.
- The scrollbar column appears only when the list overflows.
- Empty list: an empty state, never a blank area.

## Table — `ui.Table`

Borderless. The header row in subtle UPPERCASE (or the theme's case), one rule under it,
2-cell column gaps, numbers right-aligned, the first column taking the rest. The
selected row is marked like a list row. Color a cell only when its value is abnormal (a
red error rate), never a whole column.

## Detail Pane

A `Section` with the item name and its environment as info, a KV block (`ui.KV`, keys
muted and values in fg), then sections for the next most useful things (a sparkline with
its last value and trend, recent events). It appears at ≥ 110 columns. Below that it
collapses to one summary line under the list.

## Dialog / Command Palette — `ui.Dialog` + `Canvas.Overlay`

- The title row per `expressions.dialogTitle` (the motif rule in a gradient, plain, or a
  filled bar), then a blank row, the body, buttons, and one help line.
- Palette: a prompt input with a caret, then a fixed-height list (it doesn't jump while
  typing), with fuzzy matches underlined and shortcuts right-aligned.
- Only the front dialog takes input. `esc` closes it. The scrim dims the rest.
- Max ~64–70 cols wide; fullscreen below the minimum size.

## Confirm — destructive actions

```
Roll back checkout-api? ───────────────
prod returns to the last healthy version.

now     v2.4.0  failing 12 min
after   v2.3.9  healthy for 3 days
impact  3 pods restart, one at a time (~90s)

                         Roll back   [ Cancel ]
```

The title is the question, naming the object. The body says what changes, with before
and after. The danger button is coloured even when unfocused. **Cancel is focused by
default.** `y`/`n` shortcuts for experts. Prefer undo to a confirm when the action is
reversible.

## Form Field

```
▌ Version                                   ← label: bold when focused, muted when not
▌ tag or commit; latest is v2.4.1           ← hint in subtle, always visible
▌ > v2.4.1▏                                 ← input: prompt in tertiary, caret
```

The gutter bar (`ui.Gutter`) marks the focused field and spans all its rows. Radios
`◉ ○`, checks `■ □` (glyphs from the theme), selects as a `›` cursor list with a
description beside each option. Errors appear under the field in error colour, with
words. The primary button is filled because Enter triggers it, and the footer says so.

## Buttons — `ui.Button`, `ui.Buttons`

Per `expressions.button`: `pill` (padded filled blocks), `bracket` (`[ Save ]`, the
focused one with a cursor), `underline` (text), `inverse`. Mnemonic letters are
underlined. Buttons sit in a row, aligned per the layout (right in dialogs, left in
forms).

## Tags And Alerts — `ui.Tag`, `ui.Alert`

`Tag(kind, "FAILING")` is a chip. `Alert(kind, label, msg, w)` is a chip plus a tinted bar
across the width: the banner for "something is wrong with what you're looking at". One
alert per screen. If there are many, it's a list.

## Toast — `ui.Toast`

Paints over the footer row for ~4 s (a tag and a message on a tinted bar), then the help
returns. Never a new row, never a modal.

## Footer / Help — `ui.Footer`, `ui.Help`

An optional left badge (mode, `PROD`, `PLAN`), the help packed from `key.Binding`s by
importance, and right status text (`synced 4s ago`). Per `expressions.footer`: bare,
on a filled strip, or under a rule. Full help (`?`) expands into columns, grouped.

## Stream (chat, logs, events) — `ui.UserMessage`, `ui.AssistantMessage`, `ui.ToolCall`, `ui.Thinking`, `ui.TurnFooter`

- The user's voice carries a primary bar; the assistant's text is indented 2 and wrapped
  to the measure.
- Actions use the tool-call grammar (`idiom.md` §6). Collapsed output gets an honest
  "N lines hidden" affordance.
- Follow mode pins to the bottom. When scrolled up, show "↑ N more lines" at the top and
  a "new below" hint.
- A turn ends with a section-rule footer (model, provider, duration).

## Editor / Prompt

A rule above, a prompt glyph in tertiary (bold when focused, muted when not),
continuation lines with a subtle `:`, and the real cursor via `View.Cursor`. Grow from 1
to ~8 rows with content, then scroll. The placeholder says what to type, in the brand's
voice.

## Progress, Sparkline, Meter — `ui.Progress`, `ui.Sparkline`

- Progress: `━━━━╸╌╌╌` with the fill in a short gradient, then the fraction and what is
  progressing ("3/5 pods"). Indeterminate work gets a spinner, not a fake bar.
- Sparkline: `▁▂▃▅▇` over a fixed column, coloured by state, with a flat series kept
  flat. Always print the current value beside it.

## Empty, Loading, Error — `ui.Empty`

Every list and pane designs all three:

- **Empty**: a quiet glyph, what is missing, why (in context: "checkout-api has been
  failing for 12 min"), and the one action that fixes it, as a key.
- **Loading**: a skeleton of the real layout with dim placeholders, or one spinner with
  a subject. Never a blank screen.
- **Error**: what failed, against what, the likely cause, and the key to retry. Keep the
  rest of the UI usable.

## Tree — `ui.Tree`

Rounded rails `├─ ╰─` in subtle, children indented so wrapped lines keep the rail. Use it
for steps, nested agents and dependencies.

## Splash

See `cell-art.md`. Mark, wordmark, spinner with subject, version bottom-left, quit hint
bottom-right, brand field behind with a clearing. It is shown only while something loads.

## Too Small — `ui.TooSmall`

One centred line with the actual and the needed size. Nothing else.
