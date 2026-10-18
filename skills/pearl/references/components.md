# Components: anatomy and rules

*Bubble Tea: the `ui.*` calls in the headings and the code are the Bubble Tea kit's; each
component's anatomy and rules hold for any framework.*

Every component here exists in `kit/ui/bubbletea`. Copy it and change what the task
needs. Each entry gives the anatomy, the rules that a cold rebuild got wrong, and the kit
function.

## Header — `ui.Header(t, w, h, brand, details[, right])`

One row (or a banner, if the brand earns it and h ≥ 30): the wordmark, the brand motif
filling the gap, and details on the right (`env · region · selection`, the first part
in fg and the rest muted). It is drawn per `expressions.header`: `rule`, `bar` (a filled
strip), `minimal` (`brand · details`, and `right` at the edge: a count, a status),
`stamp` (the name as an inverse stamp at the right), or `banner`. Details truncate
before the brand does. The one-row header is the kit's default; a brand read's move may
replace it (`brand.md` § Move A Region).

## Tabs — `ui.Tabs`

Text, never boxes, the names in the theme's case (`labelCase` when set, else `titleCase`).
The active tab is bold with a marker drawn per `expressions.tabs`
(a `▔` underline row, a pill, `[brackets]`, or a ` / ` path). Counts sit beside the
names in subtle; a count is only red if it needs attention now.

## Section — `ui.Section(t, title, w, info)`

`Title ───────── info`. The title is in the theme's title style and case, the rule is in
the border colour, and right info is subtle. Style a span of the info to give it its own
ink (`t.S.Err.Render(t.G.Fail)+" 4 of 4 pinned"`, a figure in `t.S.Base`); the unstyled
rest stays subtle. Don't rebuild Section for it. This is how regions separate.

## List — `ui.List(t, items, sel, offset, w, h, focused)`

- The marker column is always 2 cells, so selecting never shifts text.
- Selection follows `expressions.selection`: `bar` (▌ + bold), `fill` (a full-width
  primary bar; every span carries the bg), `caret` (› + primary), `inverse`.
  `t.S.SelRow` is that selection's style and nothing else's: a badge or banner drawn in
  it reads as a second focus, and the kept harness tests count its reverse video as
  the page's focus mark under `fill` and `inverse`. Draw a badge with `ui.Tag`.
- Blurred lists keep the selection visible in a quieter style and its own shape:
  `t.BlurBar()` (`glyphs.blurBar`, else `╎`, ASCII `:`) in fgSubtle, never the focus bar
  or cursor in grey, which under NO_COLOR is a second focus. `CheckGlyphs` refuses a
  `blurBar` that is the `focusBar`.
- Title in fg, description muted after it (dropped first when tight), right-aligned
  hint in subtle, fuzzy matches underlined.
- Before the title, a status column (`Item.Mark`, a rung of the theme's ladder by its
  meaning, else `Item.State`'s Dot) and a muted `Item.Lead` column (a repo#n, an id) as
  wide as the widest lead, so titles line up. Don't hand-write a row for either.
- The scrollbar column appears only when the list overflows.
- Empty list: an empty state, never a blank area.

## Table — `ui.Table`, `ui.TableWith`

Borderless. The header row in subtle UPPERCASE (or the theme's case), one rule under it
(`ui.TableWith(…, ui.TableOpts{NoRule: true})` leaves it out where the app draws its own
divider there; don't slice the output),
2-cell column gaps (a last argument of 1 for a dense table: 10 columns at 80 cells;
`TableOpts.Gap`, where 0 is the default 2 and -1 is none),
numbers right-aligned, the first column taking the rest. The selected row is marked like
a list row. On a filled selection every block or braille cell (a trace, a sparkline, or a
history the app draws itself) takes a shade of the fill (`Styles.SelBars`): in the row's
ink they are blocks of the page punched through the bar. The theme's own marks among them
(a `▖` spinner frame, a status mark) stay in the row's ink. Under NO_COLOR the bars stay,
reversed with the rest of the row, so the fill is unbroken and the selected history
still reads, as marks in the page's ink. Color a cell only when its value is abnormal (a
red error rate), never a whole column.

## Filter — `ui.Filter`

A list's filter (`/`) stands on the list's own first row, directly above its header,
while it is typed or in force, with a row of air under it; the rows move down to make
room. Never in the footer: a toast or the keys cover it there, and the two cold builds
that put it there each drew it differently. The line is the prompt, the query and, at
the right edge, how many rows it leaves of how many (`3 of 8`), so a narrowed list
never passes for the whole one. Typed, the prompt takes the prompt's ink and a caret
follows the query; in force, the prompt is quiet and the caret's cell blank, so the
query doesn't move. Typing narrows the list live, letters are the query (the list moves
on the arrows), `enter keep filter` gives the keys back with the filter in force, and
`esc clear filter` drops it: offer `esc clear filter` only while a filter is typed or in
force, early in the table so the footer keeps it. The selection stays on a row the
filter leaves, after every key, a tab back from another view among them (clamp it in
one place, as the showcase's `key` does); when it leaves none, say so under the header
(`ui.Empty` with `ui.NoMark`), select nothing (the header and a detail pane name no
row), and bind nothing that acts on a hidden row: not `d` or `r`, and not the palette,
whose commands act on the selection. In a caret theme the prompt
(`»`, ASCII `>>`) stands in the cursor's (`›`) column in the same ink while both take
keys; keep a row between them (the filter's air, the palette's blank row under its
input), never adjacent, so they read as two marks, not a nested selection. The
showcase's services list is the pattern (`services-filter`, `services-filtered`).

## Detail Pane

A `Section` with the item name and its environment as info, a KV block (`ui.KV`, keys
muted and values in fg), then sections for the next most useful things (a sparkline with
its last value and trend, recent events). It appears at ≥ 110 columns. Below that it
collapses to one summary line under the list.

A body viewer in it (JSON, code, a response, a log) levels its text at two lightnesses:
names and keys in fg, punctuation and values in muted (or the reverse, chosen once),
never one flat level and never a rainbow of token colours. Two levels let the eye find
the keys; hue stays for state (color.md).

## Dialog / Command Palette — `ui.Dialog` + `Canvas.Overlay`

- The title row per `expressions.dialogTitle` (the motif rule in a gradient, plain, or a
  filled bar), then a blank row, the body, buttons, and one help line.
- The edge per `expressions.dialog`: a border (rounded, square, double, heavy), a filled
  panel (`none`, which gains a rule above and below where 16 colours or NO_COLOR drop the
  fill), or `rules`, the editorial one: a filled panel between two hairline rules
  (`── Send reply ──────`), a one-cell margin inside, no sides, the title in the top rule.
  Its edge is glyphs, so it holds in every profile.
- `ui.Dialog(t, w, h, …)` takes the rows it may use (the showcase passes those between
  the tabs and a row of air above the footer) and scrolls its body when it doesn't fit
  (`ui.Clip`: `↑ n more` above, `… n more` below; `DialogScrolled` for a screen with an
  offset, `DialogBodyRows` and `ClipMax` to clamp it), so no dialog reaches the footer.
  A body that must be shown whole and has no length bound (a write's verbatim text)
  scrolls between edges that stay, with `ui.DialogLines` (`lines 1–10 of 76`) set into
  the bottom edge: nothing unsent is hidden without a position. List it in the
  harness's `scrolls`.
- A dialog's status (the sending dots, `3 of 12`) is set right-aligned into its bottom
  edge, as a rules title is into the top one, so it takes no row and never crowds the
  buttons (`ui.DialogStatus`).
- Palette: a prompt input with a caret, then a fixed-height list (it doesn't jump while
  typing), with fuzzy matches underlined and shortcuts right-aligned.
- Only the front dialog takes input. `esc` closes it. The scrim dims the rest. Draw the
  page under it unfocused (`focused=false` to its lists, tables and fields, a subtle
  prompt and no caret in its input): one focus per frame, and a list keeps its
  selection under the blurred mark.
  `Canvas.Overlay` (and `Popover`) clears the rows the dialog spans, across, at every
  size and in every profile: a page cut at the dialog's margin left `…to show it` beside
  a confirm on its own rows, reading as its text. Where no scrim is drawn (16 colours,
  NO_COLOR, `scrim: none`; ASCII glyphs at truecolour still dim) a centred dialog hides
  the rows of the page's body too, since at full ink a page under a confirm showed the
  reply's tail a second time: set `c.Body = body`, the rect you draw the body in, so the
  header stays (unset, everything goes but the dialog and the footer you redraw). A
  popover anchored to its trigger (`anchor.X ≥ 0`, a completion list) clears only its
  own rows, so the line it completes stays.
- Max ~64–70 cols wide; fullscreen below the minimum size.

## Confirm — what can't be taken back, or is risky now

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
default.** `y`/`n` shortcuts for experts. It asks for a judgement, so it guards rungs 3
and 4 of `interaction.md` § Confirm By Consequence only: what can't be taken back, or
what a failing signal makes risky ("2 checks failing: lint, e2e. Merge anyway?"). This
roll back is rung 3: it ships to production, and the fix would be another ship, so it
confirms even when the target is healthy. A ship started from a form gets no dialog:
the form's review step is its confirmation (the showcase's `deploy-review`: what ships
where, a danger button named `Deploy v2.4.1 to prod`, focus arriving on Cancel).
Anything a revert or the same key takes back gets no confirm: one on every write
teaches `y` before the one that matters.

## Form Field

```
▌ Version                                   ← label: bold when focused, muted when not
▌ tag or commit; latest is v2.4.1           ← hint in subtle, always visible
▌ > v2.4.1▏                                 ← input: prompt in the focus ink, caret
```

The gutter bar (`ui.Gutter`) marks the focused field and spans all its rows. Radios
`◉ ○`, checks `■ □` (glyphs from the theme), selects as a `›` cursor list with a
description beside each option. Errors appear under the field in error colour, with
words. The primary button is filled because Enter triggers it, and the footer says so.

## Forms And Wizards — `ui.NewForm`, `ui.Confirm`, `ui.Steps`, `ui.Summary`

Huh does the input; the kit does the look and the frame around it. The reference is the
showcase's first-run setup (`go run ./cmd/showcase -first-run`, harness states `setup-*`).
Huh field titles are the app's text: the kit re-cases only its own fixed copy (`t.Copy`
for labels such as tab names in `labelCase`, `t.Sentence` for a sentence such as "Canceled."
in `titleCase`), so write a field's title in the brand's case.

```
H A L Y A R D                                         setup / 2/6 Sign-in
◆ Cluster ── ▲ Sign-in ── △ Alerts ── △ Review ── △ Connect ── △ Finish
Sign-in ──────────────────── staging-eu-west-1   Summary ──────── DRAFT
▌ Sign in with                                   Cluster
▌ ▸ Browser        SSO in your browser            staging-eu-west-1
▌   Device code    approve on another device      Sign-in        default
▌   Access token   paste one from the console     Browser
```

- **The frame is yours:** header bar with `Steps.Count()` ("2/6 Sign-in"), the step row
  (`Steps.Row`, done/current/todo glyphs from the theme), the form on the left at its
  measure, `ui.Summary` on the right (label fgMuted over value fg; untouched defaults in
  fgSubtle marked `default`; a `DRAFT`/`SAVED` tag). Below ~100 columns the summary becomes
  `ui.SummaryStrip`, one line.
- **Every step counts.** Connecting and finishing are steps, so the counter never repeats
  the review's number on three screens.
- **Review defaults to the forward action** (`ui.NewActions(0, "Continue", "Edit",
  "Cancel")`), unless the review ships what can't be taken back: then it is the
  confirmation, and focus arrives on Cancel (`interaction.md` § Confirm By Consequence,
  rung 3). `ui.Confirm(t, title, action, negative, def, &v)` for yes/no, action first.
- **Waits say what to do if nothing happens**: the URL to open by hand, the code, elapsed
  time, `esc cancel`; a timeout becomes a failure state with Retry.
- **Esc goes back** from step 2 on; on step 1 it asks before dropping typed input.

## Buttons — `ui.Button`, `ui.Buttons`

Per `expressions.button`: `pill` (padded filled blocks), `bracket` (`[ Save ]`, the
focused one with a cursor), `underline` (text), `inverse`. Mnemonic letters are
underlined. Buttons sit in a row, aligned per the layout (right in dialogs, left in
forms).

**One order, everywhere: the action first, the negative last** (`Roll back  Cancel`,
`Yes  No`, `Continue  Back`). It matches Huh's confirm and Crush, and a user who learns it
on one screen can press it blind on the next. A two-option select is not a confirm:
use the confirm, so the order can't flip. The focused default is the safe choice when the
action is destructive or irreversible, and the forward choice otherwise (Continue on a
review step, not Edit).

## Tags And Alerts — `ui.Tag`, `ui.Alert`

`Tag(kind, "FAILING")` is a chip. `Alert(kind, label, msg, w)` is a chip plus a tinted bar
across the width: the banner for "something is wrong with what you're looking at". One
alert per screen. If there are many, it's a list.

## Toast — `ui.Toast`, `ui.ToastKey`, `ui.Toasts`

Paints over the footer row for ~4 s, an undo toast ~8 s and a grace window ~5 s
(`ui.ToastTTL`, `ui.UndoTTL`, `ui.GraceTTL`; `motion.md` § Budget), as a tag and a
message on a tinted bar, then the help returns. Never a new row, never a modal. Keep the
row in a `ui.Toasts` and raise `ui.Note`s on it: it runs on the frame clock (`Tick` each
frame and return what it returns), so one tick both animates and ends a toast, and a
test drives both.

An undo toast is what a reversible action gets instead of a confirm: it names what
happened and the key that takes it back (`Closed #812 · x reopen`, `Marked 12 done · u
undo`), stays the longer ~8 s so the hand can reach the key, and that key is bound in
the state's action table while it shows. Never for something that can't be undone:
that one had a confirm, and an undo that can't is a lie. Copy the showcase's
`toastAction` (`kit/cmd/showcase/app.go`): the note's `Key` joins `actions()` while it
shows and leaves with it, so `TestHelpIsHandled` presses it, and `ui.ToastKey` draws the
row from that binding, cutting the message, never the key. Under a dialog, or while an
input takes letters, the toast offers no key and its row names none; pick a key the
state doesn't bind, since where it does the state's key wins and the toast offers none.

A grace window is the default undo toast for a send that can't be recalled once it
goes (a page, an email), skipped only where seconds matter: the note's `Send` is
pending until it ends, its key cancels it, and the row says what will happen and when
(`Paging payments on-call in 5s · u undo`, counting down). When it ends the send goes,
and a plain toast says it went (`Declared INC-2041 · paged payments on-call`). The send
is a command: return it from `Tick` and `Flush`, and every call that raises a note
returns one too, since a send whose command is dropped never went while the row says
it did. Nothing cuts the window short: a toast raised meanwhile waits and shows after
the confirmation (the latest only), so an unrelated key never sends early (an undo
toast raised then waits too, about 9 s with the confirmation, and its key with it:
accepted, since a window is short; keep another way back, as the showcase's palette offers Unsilence); quitting
sends it first (`tea.Sequence(toasts.Flush(), tea.Quit)`); and wherever its key isn't
offered (help or the palette over the row, an input taking letters, below the floor, a
state that binds the key) its clock stops (`Tick(toastAction() == nil)`), so it never
goes unseen with its key out of reach. One window at a time: `Raise` refuses a second
while one holds (it returns false and runs nothing, so the caller still owns that
send), and the kept `TestGraceWindowsHold` fails a key that opens a window offered while
one holds, so offer it only while `Holding` is false.

## Footer / Help — `ui.Footer`, `ui.Help`

An optional left badge (mode, `PROD`, `PLAN`), the help packed from `key.Binding`s by
importance, and right status text (`syncs every 5s`: from the product clock, never a fixed
`synced 4s ago`). Per `expressions.footer`: bare,
on a filled strip, or under a rule. Full help (`?`) expands into columns, grouped. The
badge's width and a space come out of the keys' 76 columns at 80: name it first in the
state's keys cell (`[plan] · **enter send** · …`) so `tuilint spec` [S-FOOTER] counts it.

## Stream (chat, logs, events) — `ui.Transcript`, `ui.UserMessage`, `ui.AssistantMessage`, `ui.ToolCall`, `ui.Thinking`, `ui.TurnFooter`

- The user's voice carries a bar: the focus bar while the turn is focused, the turn bar
  in muted when it isn't (a bold or coloured bar means focus, and a sent turn has none);
  the assistant's text is indented 2 and wrapped to the measure.
- Actions use the tool-call grammar (`idiom.md` §6). Collapsed output gets an honest
  "N lines hidden" affordance.
- Follow mode pins to the bottom. When scrolled up, show "↑ N more" at the top and
  a "new below" hint. The cut falls on a block boundary: where it would split a block, the
  marker row carries that block's head with the count right-aligned
  (`✓ Edit config/timeouts.go        ↑ 25 more  pgup`), so no output shows with its head
  off-screen (`ui.Transcript`).
- A turn ends with a section-rule footer (model, provider, duration): the kit's
  default; a brand read's move may replace it.

## Editor / Prompt

A rule above (the kit's default; a brand read's move may replace it), a prompt glyph
in the focus ink (`t.S.Prompt`: primary, or bold fg under `focus: fg`; tertiary where
primary is under 3:1 on `bgOverlay`; subtle when not focused), since where you type is
the focus,
continuation lines with a subtle `:`, and the real cursor via `View.Cursor`. Grow from 1
to ~8 rows with content, then scroll. The placeholder says what to type, in the brand's
voice.

## Status Column — `ui.Dot`, `ui.Busy`

One cell per row, before the name. `ui.Dot(t, state)` draws each state in its own shape
and role colour: ok `done`, current `current` (the open or loaded one: ● in the focus
ink), busy `partial`, warning `warn`, error `fail`, idle `todo`, so the shape says it
under NO_COLOR too, and `●` (`current`) keeps one meaning: now. A
state of the product's own (`new`, `reviewed`) is a rung of the theme's `ladder`, drawn
with `ui.Mark(t, "new")` (brand.md §4). A row in progress takes `ui.Busy(t, frame)`: the
theme's spinner when its frames are one cell, else the still partial glyph (in unicode
and ASCII alike), never a multi-cell spinner that pushes the name out of line; the
clock for such a row runs only when `ui.BusyAnimates(t)`. The label or a tag beside it
says the state in words.

## Progress, Trace, Sparkline — `ui.Progress`, `ui.Trace`, `ui.Sparkline`

- Progress: `━━━━╸╌╌╌` with the fill in a short gradient, then the fraction and what is
  progressing ("3/5 pods"). Indeterminate work gets a spinner, not a fake bar.
- A reading (a meter, a rate, a percentage) past its warning level carries the theme's
  warn mark beside the value (`ui.Mark(t, "warn")`, `t.G.Warn`: `▲`, ascii `!`), and
  past its error level the fail mark, not only amber or red: at 16 colours amber is often
  red's slot and under NO_COLOR it is the text colour, so the band is the mark's to say.
  The `ladder` frame shows the marks a reading can carry.
- Trace: a history of a reading with a fixed range (cpu %, a disk, a load) is
  `ui.Trace(t, vals, w, ui.TraceOpts{Min, Max, Per, Rows, Warn, Fail})` (one row:
  `ui.TraceRow`), never a Sparkline, which scales each series to itself and draws an
  idle 3 % full height. The zero `Min, Max` is 0..100; set `Max` for bytes or a rate.
  Newest sample at the right; a cell holds `Per` samples drawn as their max (60 s at Per
  4 is 15 cells, and a 1 s spike survives); cells before the first sample are blank, not
  zero; a cell at or over `Warn` takes the warning's ink and over `Fail` the error's,
  or `Ink` decides every cell above `Min` (a cell at `Min` is the floor in fgSubtle
  whatever `Ink` says, so mark a down state beside the trace, not in it);
  `Rows: 3` draws 21 bar heights above the floor on the bottom row. A bar's top row stops at `▇`
  (7/8), in a Sparkline too: a `█` sits flush on the row above, and under NO_COLOR the
  two rows fuse into one shape.
- Idle is dots, never a line. A sample under 1/8 of the scale draws the floor `⣀` (`⠤`
  beside a rule drawn in braille), so a 4–9 % band is one even row, not `⣀▁⣀▁`, and a
  quiet row is never a hairline: `▁▁▁▁` reads as one more rule beside any rule, and over
  the next row's bars it closes a box. A Trace's zeros are a reading (an idle queue, a
  healthy error rate) and draw the floor in fgSubtle, a step under a low reading's
  fgMuted; its no data is NaN or no samples, which draw blank, never 0. ASCII draws heights
  `.,:;%&#@` (punctuation thickening upward) and the floor `.` (`,` beside a `.` rule),
  never `-`, `_` or `=`, whose runs are rules.
- Sparkline: `▁▂▃▅▇` over a fixed column, scaled to the series (latency, throughput),
  coloured by state. A series within 10 % of its peak is noise and draws the floor; past
  it the peak rises a level per ~8 % of spread (`▂` at 10 %, `▄` at a third, `▇` from
  about a half), so a +20 % p99 step shows as a step, never as a wall, and 34 % is not
  twice 32 %. A series of zeros (a p99 with no requests) is no data and draws blank:
  the floor a quiet series draws would be the same row as 0.01 % under NO_COLOR and in
  ASCII. Always print the current value
  beside it. A warning's ends in the warn mark (`ui.StateSparkline`: `▁▃▅▆▇▲`): at 16
  colours a warm brand's yellow and the warning's are one hue, so bars in yellow alone
  read as the selection. A sparkline taller than one row grows upward from the label's
  row: label, value and the bar's base share the bottom row, the rows above hold the
  peaks, so a two-row graph never splits the label from the bar it names (a `▂▆▆▂`
  fragment alone on a row reads as noise).

## Empty, Loading, Error — `ui.Empty`

Every list and pane designs all three:

- **Empty**: a quiet glyph, what is missing, why (in context: "checkout-api has been
  failing for 12 min"), and the one action that fixes it, as a key. Where the state's
  mark or motion is elsewhere (a spinner in the footer), pass `ui.NoMark`: the title
  is the first row. Don't cut `Empty`'s rows.
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

## Too Small — `ui.TooSmall(t, w, h, minW, minH, bindings)`, `ui.TooSmallText(t, w, h, say, bindings)`

The wordmark, one centred line with the actual and the needed size, then the keys of the
`toosmall` row of the action table (`m.footerBindings()` while the size is below the
minimum), never a key typed into the screen: `TestHelpIsHandled` presses every one. A
spec that words the screen itself ("Terminal is 60×18; needs 80×24") passes its sentence
to `ui.TooSmallText`, which wraps it at 56 cells and draws no wordmark. A spec that asks
for the sentence and nothing else leaves the `toosmall` row without a shown key: no key
row is drawn, and ctrl+c still quits (`TestTooSmallKeysAreTheTable` checks it).
