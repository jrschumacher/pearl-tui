# init — interview the user, then write the design and the architecture into TUI.md

Run this first in a project with no `TUI.md`, or when the product, the brand or the stack
has changed. `teach` is an alias. It writes `TUI.md` at the project root, and every
command that changes UI reads it before doing anything.

*Bubble Tea: the Forms and slow-work defaults in §2 and the Architecture half of §3's
template name Bubble Tea v2's libraries; the interview and the design half hold for any
framework.*

**One file, two halves.** Design (what it looks like and why) and architecture (how it's
built) both go in `TUI.md`. Every command reads that file first; a second file gets
skipped, and then the screens drift from the model that draws them. Never split it into
an `ARCH.md`.

**`TUI.md` is the only file `init` writes** (apart from `.tuitools/` and the `.gitignore`
line Setup step 4 adds). The project's own docs (a spec, PRD, design
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
it. Write each departure from `bubbletea-architecture.md` (v1 imports, nested models, no
harness, Huh on its default theme) as a `gap:` line, which `restyle` then closes.

## 2. Interview In Short Rounds

- **At most three questions per round, each with a proposed default.** The user answers
  "yes" or corrects one word; an open list of ten gets half answered.
- **Use a structured question tool** if you have one (AskUserQuestion or similar): the
  default first, marked "(recommended)", plus one or two real alternatives.
- **Ask only what you couldn't infer.** Confirm an inference in the question's default
  instead of asking it open.
- **Stop at two rounds** (a step-0 conflict round doesn't count) unless an answer
  changes the design. Whatever is still unknown takes its default and goes under
  `## Assumed`.
- **Never assumed: who uses it, the brand words, the screens.** Ask those even when
  everything else is inferred: a bare repo can't answer them, and a brand read inferred
  from a package name ships under `## Assumed` without anyone deciding it.

Round 1, design (skip what the repo answered):

1. **Who uses it, where, and for how long?** (field engineers on rugged laptops for
   hours; developers in a split pane for seconds)
2. **The brand in three words, and one thing it must never feel like.**
3. **Brand assets**: primary colours (hex), the logo file, the typeface, a reference
   product ("like Crush but calmer"). Any screen to match: is it exact, or inspiration?
   (`verify.md` § Hold It To The Reference; it goes in § Reference.)

Round 2, architecture (screens always; the rest only what the code didn't answer):

1. **Screens and the job of each. Always ask it**, even when a spec lists them, because
   screens are the backbone every later command reads (`shape`, the harness states,
   the first slice), and a list inferred in silence drops the flows the user cares
   about. Default: one line per screen with its job (`services — see what is
   deployed`), one per command you found, plus each flow the docs describe that has
   no screen yet (an export, a background sync).
2. **Forms.** Default: Huh v2 for input mechanics (select, input, confirm), themed from
   the brand theme through the kit's `ui.NewForm` (`ui/bubbletea/form.go`), because
   Huh's own theme is the indigo every Huh demo ships.
3. **Slow work and the floor.** What is async (network, builds, streaming), can it be
   cancelled, and the smallest terminal. Default: every call is a `tea.Cmd` with a cancel;
   60×18 floor, designed at 80×24.

No brand: write two short brand reads that differ in kind (not in colour) and let the
user pick. If the user can't pick, take the read matching the session's tone (hours on
call: calm and compact; seconds in a split pane: terse and plain), put the other under
`## Assumed` as the alternative, and say so in the hand-off. Never fall back to the
default theme silently.

## 3. Write TUI.md

Every section is one the next command reads. Fill each with facts or a default from the
interview; delete nothing, and write `assumed:` where you guessed.

````markdown
# <Product> TUI

Read before any UI change. Update it when a decision changes.

## Design

### Product
<what it does, for whom, how long a session lasts>
customers: <two or three: who, how often, what they use today>   (cmd-shape.md § Read The Product, Then Let Customers Attack It)
jobs: <ranked by how often a customer does each; the first is the main screen's>

### Users and terminals
<who; sizes (80×24 design floor); light/dark; SSH/tmux; NO_COLOR; screen readers>
colour: truecolor → 256 → 16 → none; under -profile ascii, focus, selection and errors stay visible by <glyph, weight>
dialogs and overlays: where the background step is gone (16 colours, NO_COLOR, ascii), the edge is <border | ─ rule above and below (dialog: rules)>

### Brand read
palette / tone / geometry / density / type / motif   (references/brand.md §1)
move: <one region elsewhere in the screen's order than the kit puts it (above or below a different region, left or right of one), or gone, or added, and the word it follows from; not a one-row shift from a gap or a rule row>, e.g. the open alert fills the top half and the list folds under it (a monitor, urgent) · the answers so far as a column left of the field (a wizard, reassuring) · the composer right under the header and the transcript below it (a chat, eager); take yours from your words, not these
signature: <one element the kit doesn't draw, on the main screen's default frame, not the move's region>, e.g. a 12-cell trace of today's load beside the name, a running-tally column on the list; not a glyph, a rule style, a hue or a restyling theme expression
skeleton: head <…> · status <…> · order <…> · foot <…> · verdict <…> · chart <…> · chartGlyphs <…> · turnEnd <…>   (the main screen's composition, the words in `diversity -h`, - for an axis it doesn't have; planned here, corrected once craft renders the default frame; `diversity TUI.md` reads it)
accents: <role> (<hue>) = <its one meaning>, one line per hue, e.g. primary (amber) = new; focus = <gutter, weight>, never a state hue ("focus": "fg" in the theme when the brand's hue is a state)
glyphs: <glyph> <meaning> (ascii <twin>), one line each; twins the kit lacks go in the theme's glyphs.ascii; width-safe: <narrow glyph | 2-cell column>

### Theme
themes/<brand>.json   (palette derive from <measured hexes, source>)

### Reference
<none | the files; size in cells; the states shown>   (verify.md § Hold It To The Reference)
exact: <composition, contrast, artwork treatment, status placement, footer hierarchy, motion: the axes to match>; inspiration: <the rest>
approved: <date> <deviation> (<who>), one line each; a law the reference breaks and the user kept goes here too

### Screens
One block per screen, the main screen's first (`tuilint spec` reads the first block as the main screen): `shape` fills it, `craft` builds it in § First slice order.

#### <screen> — <its job in one line, and what the eye lands on first>
job: <the task, and its flow: monitoring | checkout | media | browse | configure>   (flows.md)
data: <each region's data by kind: live, background, money, the product's purpose>
carry: <what stays in view from the product's purpose, or nothing, because this screen is it>
80×24:  <region> <x,y w×h> · <region> <x,y w×h> · footer 0,23 80×1     (outer widths; text width = outer − gutters)
120×40: <region> <x,y w×h> · <the pane that exists only here> · footer 0,39 120×1
        (under footer: rule the footer is 0,22 80×2: layout.md § Write The Budget)
collapse: < 110 cols <what becomes what>; < 30 rows <what goes>
components: <ui.Table, ui.Dot, …>; new: <none | name, and why no kit component draws it>
fixtures: <count, the longest name, the failing row, the empty list>
brand: <every block after the main screen's: what the three words change here, copy voice, dialog form or footer phrasing, or nothing, because …>
flow: <main screen: how deep the flow goes against the spec's, the domain fact behind it (who acts, on how many objects, can the act be taken back) and any brand word; another block: only where its steps or preview differ, else delete the line>
sketch (exactly 80 wide, real content):
```
<row 0>
…
<row 23>
```

### Motion
hero: <the one long-running action> → <its animation, ≤ 30 cells, from the brand motif>
rows: <spinners for row-level work>; idle: nothing moves

### Never
<anti-references: what this product must not look or sound like>

## Architecture

### Stack
Go 1.26.7; charm.land/bubbletea/v2 v2.1.0, lipgloss/v2 v2.0.6, bubbles/v2 v2.2.1,
huh/v2 v2.0.3; ultraviolet v0.0.0-20261008173134-6b8d4baf91b4.
The kit's theme/bubbletea and ui/bubbletea copied into internal/theme and internal/ui;
the showcase into cmd/<app>.

### Root model and regions
One *model in cmd/<app> (where the kit's copy block puts the harness and its tests);
regions are structs drawn into rectangles (x,y w×h at 80×24):

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
Footer keys are **bold** (in a long cell, ‖ after the last of them does the same): main actions first, then the way out and ? help (ctrl+g help where a text box types ?). The rest are help-only.
A footer key shown only under a condition ends with it: `**esc clear filter (if a filter is applied)**`, `(when …)`, `(while the body overflows)`; one whose screen comes later in the slice ends with `(later)` or `(later: <screen>)`. The mark takes no footer width, and the golden needn't show that key.
A `confirm-*` row exists only for an action that can't be taken back or that a failing signal makes risky; its § Screens block names the rung and why (`rung 3: ships to prod`, `rung 4: opens on red checks`). A ship started from a form has no `confirm-*` row: its block says the review step is the confirmation.

| state (harness name) | screen / overlay | keys |
|---|---|---|
| services             | services         | **↑↓ move** · **enter open** · **d deploy** · **r roll back** · **/ filter** · **q quit** · **? help** · l logs · c copy id · s sort by status |
| services-empty       | services         | r refresh · ? help · q quit |
| reply                | reply (textarea) | **ctrl+s send** · **esc cancel** · **ctrl+g help** · shift+enter/ctrl+j newline |
| confirm-rollback     | overlay          | ←→ choose · enter confirm · esc cancel |
| toosmall             | any              | q quit |

Global: ctrl+c quit

### Harness and fixtures
cmd/<app>/fixtures.go feeds setState for the harness (-state, -out) and every test.
fakes: <service> → <fake>, one per external CLI or API; no render or test touches the network.

### Async and the clock
<work> → <cmd> → <msg>; cancel: <context>; stale results: <id or generation>.
Frame clock: one gen-stamped tick, armed only while <what> animates; a stale tick dies.
Product clock: <none | a poll, a sample, a sync every <interval>>, its own msg and gen; the handler that changes it (interval, pause) returns the new arm cmd, never a gen bump alone.
largest: <realistic worst case, e.g. 50k log lines, 5k rows> → <virtual list | render cache keyed by …>

### Size floor
Below <floor; 60×18 unless the product raises it>: ui.TooSmall. Below 110 columns: <what collapses>. Below 30 rows: <what>.

### Accessible and non-TTY
ACCESSIBLE=1 or --accessible: no alt screen, Huh WithAccessible(true), status as new
lines. stdout not a TTY: <--json | plain output>, never escape codes.

## First slice
<the main loop's 2–3 screens, every state of those, rendered from fixtures through the harness; no integration>
Then: <the rest of v1, in order>

## Decisions
- <date> <decision> because <reason>
- <date> cut: <idea> — <why no customer needs it in most sessions>
- <date> cuts: [<screen>:] <column header or budget region>, … — <why it has no longest>   (e.g. `- 2026-10-18 cuts: detail: command — no longest; the pager shows it whole`; <screen> is a § Screens block's name, and with none the line holds on every screen; `tuilint frames --spec TUI.md` reads it under this exact `## Decisions` heading [F-TRUNC])

## Assumed
- <default the user didn't confirm>   (confirm it, then move it to Decisions)
````

A brief written before this skill (a spec, a PRD, an earlier TUI.md) gets these sections
wrong. Write them this way:

- **A move and a signature the TUI draws,** not a quality ("feels trustworthy") and not
  a motif alone (`brand.md` § Move A Region). An earlier TUI.md's `remember:` line
  becomes these two (`tuilint spec` warns until it does [S-SLOTS]): ask the user, or
  write them `assumed:`.
- **One meaning per accent hue.** One hue for "new" *and* for focus makes the cursor
  row and a new row look the same. Focus is the gutter and the weight (`interaction.md`
  § Focus); a hue is a state. The kit draws focus in primary unless the theme says
  `"focus": "fg"` (`brand.md` §4), so a brand whose only hue is a state sets it.
- **Keep a meaningful glyph, with an ASCII twin and a width.** `◆ ◇ ▲ ★ ■` are East
  Asian "ambiguous": two cells in CJK locales, and the kit's ASCII map covers only its
  own glyphs. Keep the glyph, name its twin (`◆ ◇ ▲ (ascii * o ^)`), write the twin into
  the theme's `glyphs.ascii` (`palette check` fails a glyph with no twin, or one that
  shares another state's twin), and set it in a
  2-cell column, which the other ambiguous glyphs in that column need anyway. Swap it
  only for a narrow glyph with the same meaning (`✓ ❯ ▸`), never for a word: a set of
  marks read at a glance is part of the brand's look, and dropping one for width merges
  two meanings into one mark.
- **A dialog edge that survives without colour.** A filled panel one background step
  up, with no border, sits on the page in the same colour under 16 colours and
  NO_COLOR, so the dialog's edge vanishes. The kit's `dialog: none` gets a rule above
  and below wherever that happens (`theme.ResolveProfile` with the terminal's profile)
  and an ASCII border under -ascii; `dialog: rules` (a rule above and below, the title
  in the top one) keeps its edge in every profile. Name the edge the brand wants.
- **A fake for every external service.** The harness renders every state offline and
  the tests run in CI; a state that needs a live CLI or API is a state nobody reviews.
- **A first slice.** The smallest milestone that renders the main loop on fixtures: the
  2–3 screens a session spends its time in (list → detail → act), every state of
  those, so the design is judged on renders before the integrations eat the effort.
  "Every state on fixtures" is the whole app, not a slice. If the interview grew v1
  past the brief, say so and propose the slice; don't just order it.

Keep three things true as the project goes, because each one is what a later command
checks:

- **State names equal harness names.** `-state services-empty` must render the row
  called `services-empty`, so the critic, the harness and this file talk about the same
  frame. The kit's too-small state is `toosmall`, one row for the whole app. Overlays
  are named as `cmd-shape.md` step 3 says.
- **Keys here are the action tables.** If the table and `actions()` disagree, the code
  wins and this file is fixed in the same change; `TestHelpIsHandled` holds the code.
  The help-only keys (not bold, or after ‖) are the actions with `show: false`.
- **A decision carries its reason.** "Square dialogs" is a default in disguise; "square
  dialogs because the brand is industrial" survives the next restyle.

## 4. Lint It Before Handing Off

`tuilint spec TUI.md` must exit 0. A cell is `key action` items joined by `·`, one binding
per item (alternates joined by `/`) with the key first (`ctrl+c twice quit now`, not a sentence), because `ui.Help`
draws the cell as written and the lint reads the first token as the key. It fails a
table not in the template's format
[S-TABLE], because nothing else can be checked without one; a key that terminals without key
disambiguation deliver as another, with no fallback in the cell (`shift+enter` arrives
as `enter` in tmux: write `shift+enter/ctrl+j newline`) [S-KEYAMBIG], and one key with
two actions in one state's cell or in Global [S-KEYDUP]; a modifier carries onto arrows
only (`shift+↑↓` and `shift+↑/↓` are shift+up and shift+down, but `ctrl+c/q` is ctrl+c
or q, so spell three ctrl keys `ctrl+d/ctrl+f/ctrl+t`). A state that binds a Global
key to its own action (`esc cancel` over Global `esc back`) overrides it and passes,
because the state's table is dispatched first. Fix its warnings or justify each in
§ Decisions: a Global key a text box eats (`ctrl+k` deletes to end of line in a Bubbles
textarea) [S-KEYSHADOW], an ambiguous-width glyph with no ASCII twin [S-GLYPHWIDTH], and
a state whose footer keys overflow the 76 columns an 80-column footer has inside the
kit's gutters, where `ui.Help` drops the tail behind `…` (a footer badge is the cell's
first item, `[plan] · **enter send** · …`, and takes its width and a space) [S-FOOTER], a marked footer with
help-only keys and no help key in it (end it with `? help`, or `ctrl+g help` in a row
with a text box, as the template's `reply` does: a Global ctrl+g is the textarea's
select-all [S-KEYSHADOW]) [S-FOOTER], and a line that
puts "the accent" or "primary" on something the `accents:` line didn't give it
[S-ACCENT]. Fix that one by
marking the footer, bold or ‖: the state's main actions and its way out, then `? help`,
so the rest are help-only by decision rather than cut by width, and never `esc` or quit.

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
