# init — interview the user, then write the design and the architecture into TUI.md

Run this first in a project with no `TUI.md`, or when the product, the brand or the stack
has changed. `teach` is an alias. It writes `TUI.md` at the project root, and every
command that changes UI reads it before doing anything.

**One file, two halves.** Design (what it looks like and why) and architecture (how it's
built) both go in `TUI.md`. Every command reads that file first; a second file gets
skipped, and then the screens drift from the model that draws them. Never split it into
an `ARCH.md`.

**`TUI.md` is the only file `init` writes.** The project's own docs (a spec, PRD, design
doc, README), if it has any, are the user's: an answer that changes one goes in the
hand-off as a proposed edit, applied only when they say yes, because a silent rewrite
hides who decided what. Layout, keys, colours and screens never leave `TUI.md`; a
breakpoint written in two files drifts in one. Point at those docs ("product behaviour:
docs/prd.md") instead of restating them; with none, § Product holds what the TUI needs.

## 0. Read The Project's Own Docs First

Look for whatever the project already decided, under any name: a spec or PRD
(`SPEC.md`, `PRD.md`, `docs/`), the README, `CLAUDE.md`/`AGENTS.md`, an issue the user
points to. Often there is none, and the interview is the only source. What you find,
confirm in a question's default; don't ask it again. If one routes screen design
to another method, tool or skill ("screens are designed with the X method"), ask which
one governs this TUI and record the answer, with its reason, in § Decisions. A build
agent follows whichever instruction it read last, so two methods give two designs. Ask
it in its own one-question round before round 2, or in place of a lower-value round-2
question (forms, the floor), never in place of the screens question.

## 1. Explore Before Asking

Every question the repo could have answered spends the user's patience on nothing. Read
first:

- the README, the CLI's `--help`, its commands, and any screens
  (`grep -rln "bubbletea\|lipgloss\|huh" --include=*.go`)
- brand assets: logos (`*.svg *.png`), CSS variables, a style guide, fonts
- the product's own copy (docs, site text, error messages). This is the voice.
- if a TUI exists: `go.mod` (v1 or v2, which versions), the root model, how `Update`
  routes keys, which Bubbles and Huh components it uses, its `tea.Cmd`s and ticks. Render
  it (`cmd-critique.md` §1).

When a TUI exists, the architecture half describes the code as it is. Infer it; don't ask
it. Write each departure from `architecture.md` (v1 imports, nested models, no harness,
Huh on its default theme) as a `gap:` line, which `restyle` then closes.

## 2. Interview In Short Rounds

- **At most three questions per round, each with a proposed default.** The user answers
  "yes" or corrects one word; an open list of ten gets half answered.
- **Use a structured question tool** if you have one (AskUserQuestion or similar): the
  default first, marked "(recommended)", plus one or two real alternatives.
- **Ask only what you couldn't infer.** Confirm an inference in the question's default
  instead of asking it open.
- **Stop at two rounds** unless an answer changes the design. Whatever is still unknown
  takes its default and goes under `## Assumed`.

Round 1, design (skip what the repo answered):

1. **Who uses it, where, and for how long?** (field engineers on rugged laptops for
   hours; developers in a split pane for seconds)
2. **The brand in three words, and one thing it must never feel like.**
3. **Brand assets**: primary colours (hex), the logo file, the typeface, a reference
   product ("like Crush but calmer").

Round 2, architecture (only what the code didn't answer):

1. **Screens and the job of each. Always ask it**, even when a spec lists them, because
   screens are the backbone every later command reads (`shape`, the harness states,
   the first slice), and a list inferred in silence drops the flows the user cares
   about. Default: one line per screen with its job (`services — see what is
   deployed`), one per command you found, plus each flow the docs describe that has
   no screen yet (a bulk action, an alert answered from any screen).
2. **Forms.** Default: Huh v2 for input mechanics (select, input, confirm), themed from
   the brand theme through the kit's `ui.NewForm` (`ui/form.go`), because Huh's own theme is the indigo
   every Huh demo ships.
3. **Slow work and the floor.** What is async (network, builds, streaming), can it be
   cancelled, and the smallest terminal. Default: every call is a `tea.Cmd` with a cancel;
   60×18 floor, designed at 80×24.

No brand: write two short brand reads that differ in kind (not in colour) and let the
user pick. Never fall back to the default theme silently.

## 3. Write TUI.md

Every section is one the next command reads. Fill each with facts or a default from the
interview; delete nothing, and write `assumed:` where you guessed.

````markdown
# <Product> TUI

Read before any UI change. Update it when a decision changes.

## Design

### Product
<what it does, for whom, the jobs the TUI is for, how long a session lasts>

### Users and terminals
<who; sizes (80×24 design floor); light/dark; SSH/tmux; NO_COLOR; screen readers>
colour: truecolor → 256 → 16 → none; under -profile ascii, focus, selection and errors stay visible by <glyph, weight>

### Brand read
palette / tone / geometry / density / type / motif / remember   (references/brand.md §1)
accents: <hue> = <its one meaning>, one line per hue; focus = <gutter, weight>, never a state hue
glyphs: <glyph> <meaning> (ascii <twin>), one line each; width-safe: <narrow glyph | 2-cell column>

### Theme
themes/<brand>.json   (palette derive from <measured hexes, source>)

### Screens
<screen> — <its job in one line>

### Motion
hero: <the one long-running action> → <its animation, ≤ 30 cells, from the brand motif>
rows: <spinners for row-level work>; idle: nothing moves

### Never
<anti-references: what this product must not look or sound like>

## Architecture

### Stack
Go 1.26.7; charm.land/bubbletea/v2 v2.1.0, lipgloss/v2 v2.0.6, bubbles/v2 v2.2.1,
huh/v2 v2.0.3; ultraviolet v0.0.0-20261008173134-6b8d4baf91b4.
theme/ and ui/ copied from the kit into internal/.

### Root model and regions
One *model in internal/app; regions are structs drawn into rectangles (x,y w×h at 80×24):

    header   0,0    80×1
    list     0,2    32×20
    detail   34,2   46×20
    footer   0,23   80×1

focus: list | detail        overlays: palette, confirm, help (front one gets keys)

### Forms
Huh v2 for <setup, config, confirm>, themed with the kit's `ui.NewForm` from
*theme.Theme. Hand-written: <field, because Huh can't show it>.

### Components
Bubbles: <textarea, viewport, …>. Kit ui: <List, Table, Dialog, Footer, …>.

### States and keys
One action table per state; footer and dispatcher both read it. Every screen that
loads data has its empty, loading, error and partial (some sources failed) rows.
Keys before ‖ are the footer, most used first, ending in ? help; keys after ‖ are help-only.

| state (harness name) | screen / overlay | keys |
|---|---|---|
| services             | services         | ↑↓ move · enter open · / filter · ? help ‖ r refresh · q quit |
| services-empty       | services         | r refresh · ? help · q quit |
| confirm              | overlay          | ←→ choose · enter confirm · esc cancel |

Global: ctrl+c quit

### Harness and fixtures
internal/app/fixtures.go feeds setState for the harness (-state, -out) and every test.
fakes: <service> → <fake>, one per external CLI or API; no render or test touches the network.

### Async and the clock
<work> → <cmd> → <msg>; cancel: <context>; stale results: <id or generation>.
One gen-stamped tick, armed only while <what> animates.
largest: <realistic worst case, e.g. 50k log lines, 5k rows> → <virtual list | render cache keyed by …>

### Size floor
Below 60×18: ui.TooSmall. Below 110 columns: <what collapses>. Below 30 rows: <what>.

### Accessible and non-TTY
ACCESSIBLE=1 or --accessible: no alt screen, Huh WithAccessible(true), status as new
lines. stdout not a TTY: <--json | plain output>, never escape codes.

## First slice
<the core screens, every state, rendered from fixtures through the harness; no integration>
Then: <the rest of v1, in order>

## Decisions
- <date> <decision> because <reason>

## Assumed
- <default the user didn't confirm>   (confirm it, then move it to Decisions)
````

A brief written before this skill (a spec, a PRD, an earlier TUI.md) gets four sections
wrong. Write them this way:

- **One meaning per accent hue.** One hue for "new" *and* for focus makes the cursor
  row and a new row look the same. Focus is the gutter and the weight (`interaction.md`
  § Focus); a hue is a state.
- **An ASCII twin and a width for every motif glyph.** `◆ ★ ▲ ■` are East Asian
  "ambiguous": two cells in CJK locales, and the kit's ASCII map covers only its own
  glyphs. Name the twin (`◆ ◇ (ascii * o)`), and pick a narrow glyph (`✓ ❯ ▸`) or a
  2-cell column.
- **A fake for every external service.** The harness renders every state offline and
  the tests run in CI; a state that needs a live CLI or API is a state nobody reviews.
- **A first slice.** The smallest milestone that renders the core screens on fixtures,
  so the design is judged on renders before the integrations eat the effort. If the
  interview grew v1 past the brief, say so and propose the slice; don't just order it.

Keep three things true as the project goes, because each one is what a later command
checks:

- **State names equal harness names.** `-state services-empty` must render the row
  called `services-empty`, so the critic, the harness and this file talk about the same
  frame.
- **Keys here are the action tables.** If the table and `actions()` disagree, the code
  wins and this file is fixed in the same change; `TestHelpIsHandled` holds the code.
  Keys after ‖ are the actions with `show: false`.
- **A decision carries its reason.** "Square dialogs" is a default in disguise; "square
  dialogs because the brand is industrial" survives the next restyle.

## 4. Lint It Before Handing Off

`tuilint spec TUI.md` must exit 0. It fails a key that terminals without key
disambiguation deliver as another, with no fallback in the cell (`shift+enter` arrives
as `enter` in tmux: write `shift+enter/ctrl+j newline`) [S-KEYAMBIG], and one key with
two actions in one state's cell or in Global [S-KEYDUP]. A state that binds a Global
key to its own action (`esc cancel` over Global `esc back`) overrides it and passes,
because the state's table is dispatched first. Fix its warnings or justify each in
§ Decisions: a Global key a text box eats (`ctrl+k` deletes to end of line in a Bubbles
textarea) [S-KEYSHADOW], an ambiguous-width glyph with no ASCII twin [S-GLYPHWIDTH], and
a state whose footer keys overflow the 76 columns an 80-column footer has inside the
kit's gutters, where `ui.Help` drops the tail behind `…` [S-FOOTER]. Fix that one with
a ‖ after the keys a user needs most, ending in `? help`, so the rest are help-only by
decision rather than cut by width.

## 5. Hand Off In Three Lines

```
Captured: <product, users, brand read, N screens, stack>. Assumed: <the defaults nobody confirmed>.
Proposed for <their doc>: <product changes from the interview, one per line> (apply? y/n)   ← only if an answer changes one of their docs
Next: brand (no theme yet) | shape (theme exists) | restyle (an existing app).
```

## The Short Form

When another command finds no `TUI.md` and the job is one screen, ask round 1 only (two
or three questions, each with a default), write the template with the rest marked
`assumed:`, and tell the user what you assumed. A full `init` can replace it later.
