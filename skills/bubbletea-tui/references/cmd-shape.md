# shape — design the screens on paper before any code

Input: `TUI.md` and the user's ask. Output: a screen spec the user agrees to, written
into `TUI.md` § Screens (one block per screen, in the template's shape: job, regions at
both sizes, collapse rule, components, fixtures, brand line, any flow line, sketch) and
§ States and keys, and nowhere else: a second file gets skipped, and the project's own
docs belong to the user. No Go yet.

## Depth Follows The Domain, Then The Brand

Before the first block, answer one question: does this domain, or this brand, earn more
or fewer steps than the spec's default flow? Depth is how many steps a task takes, how
much shows before an act (a preview, a diff, a summary), what is one key away against a
screen away, and whether an expert path exists (a command line, batch selection). Read
the domain first (`SPEC.md` or the ask, § Product, § Users and terminals), then the
read's three words and any `flow:` line `brand` wrote: the domain sets the range, and a
word picks a place in it.

- **High consequence, many objects** (cluster ops, deploys, infrastructure, finance)
  earns a step that shows what will change, and to what, before the act (`3 replicas
  v2.4.0 → v2.4.1 · 1 unchanged`), and an expert path (`:` for a command line, `space`
  to mark a batch): one wrong act reaches many objects, and its users run it all day.
- **Personal, low consequence** (mail, notes, a todo list) earns fewer steps and fewer
  screens: the item opens in place and the act happens on its key, because an undo
  covers a slip and every extra step is paid on every use.
- **A word pushes within that range**: "for operators" earns dense rows, keyboard-first
  and a batch path; "calm" earns one thing at a time, with the detail behind a key.

Write the answer on the main screen's block as a `flow:` line with its reason (`flow:
the spec's, because a personal list: each act is one key and undo covers it`), and give
every other block whose steps, preview or detail differ from the spec's its own. The
same apply step of a DNS zone editor, two brands:

```
#### apply — publish the edited records              (exact · dense · for operators)
flow: space marks records and a applies the marked set, because for operators; the confirm lists each record old → new, because a zone change reaches every resolver
```

```
#### apply — publish the edited records              (calm · plain · careful)
flow: one record at a time, its history behind h, because calm; the confirm lists the record old → new, because a zone change reaches every resolver
```

Both show old → new, since the domain asks it of both; only the operators' brand has
the batch path, since only its word asks for one.

**DO** keep what the product does: `SPEC.md` decides which tasks exist and what each
does, and depth changes only how a task is reached and what shows on the way.
**DON'T** add or drop a confirm to set the depth. The ladder (`interaction.md`
§ Confirm By Consequence) puts each act on its rung by consequence, and no word moves
it: a deep flow puts its diff in the confirm a rung-3 act already has, or on the screen
the key is pressed from, never in a new dialog before a rung-1 or rung-2 act; a light
flow never skips the confirm a rung-3 act needs.
**DON'T** let a flow differ from the spec's where no domain fact or brand word on its
`flow:` line asks for it: two brands of one spec then differ by chance, which is drift,
not identity, and the critic marks it.

## For Each Screen

1. **Job**: the one thing the user comes to this screen to do, and what they look at
   first.
2. **Regions and budgets** at 80×24 and 120×40, with every pane's outer, inner and text
   width as integers (`layout.md`). Say what collapses below 110 columns and below 30
   rows.
3. **States**: populated, empty, loading, error, partial (a chat screen adds thinking
   with no tool call yet, and prose streaming in), plus every overlay (palette,
   confirm, help). Each becomes a harness state name; `toosmall` is one row for the app.
   An overlay's name carries `dialog`, `palette`, `confirm`, `modal`, `overlay`,
   `popup`, `help` or `keys` (`confirm-discard`, not `discard`): `tuilint frames` finds
   a dialog by shape only where its scrim dims or hides the page, so name every overlay
   frame so it is judged as one, and not as a page whose scrim fails [F-CONTRAST].
   A write gets a `confirm-*` state only if it can't be taken back or is risky in that
   state (rungs 3 and 4, `interaction.md` § Confirm By Consequence); a reversible one
   gets a done state with its undo toast instead (`closed-undo`), and a sent message its
   `sent`. A ship started from a form has no `confirm-*` state: its review step is it.
4. **Keys**: the bindings, the help order, what `esc` does, and where focus starts and
   moves. A footer key shown only sometimes ends with its condition, `(if …)`, `(when …)`
   or `(while …)` (`(if a filter is applied)`), and one whose screen comes later in the
   slice with `(later)` or `(later: <screen>)`, as in `cmd-init.md`'s template, so
   `tuilint spec` neither counts the mark's width nor asks the golden to show the key.
5. **Content**: realistic fixture data, including the longest name, the failing row and
   the empty list. No lorem ipsum. Each column's budget fits its longest fixture: more
   than 3 rows cut to `…` in one column is a budget too narrow [F-TRUNC]. A column of
   paths, URLs or code has no longest: cut it and say so in § Decisions, on a `cuts:`
   line naming its header or budget region, after a § Screens block's name and a colon
   to hold it to that screen (`- 2026-10-18 cuts: detail: command — no longest; the
   pager shows it whole`; any other word before a colon starts the reason, and a line
   with no screen holds on every screen), which `tuilint frames --spec TUI.md` honours.
6. **Components**: which kit component draws each region (`components.md`), and any new
   component you'll need. A new component is a reason to stop and think: is it a real
   need, or a box you're reaching for?
7. **Identity**: where the brand shows on this screen (header, motif, selection style,
   copy). On the main screen, whose block comes first, the sketch draws the read's move
   and signature. Every other screen and overlay block gets one `brand:` line (§ Behind
   The Main Screen), and a `flow:` line where its flow differs from the spec's (§ Depth
   Follows The Domain, Then The Brand).

## Behind The Main Screen

The move and the signature reach the main screen only. Two brands built cold from one
spec had main screens with nothing in common, and a detail, a picker and a confirm that
matched word for word: the same footers, the same target line, and one sentence about
the action that neither the spec nor the kit held. A user meets those screens after one
keypress. So each secondary screen and overlay block says, in one `brand:` line, what
the brand's three words change on it: its copy voice, its dialog form, its footer
phrasing. Like `move:`, it follows from one word and is judged by what the user reads,
not by paint. `tuilint spec` warns on a block after the first without one [S-SLOTS].

The same confirm, two brands (a backup tool deleting one snapshot):

```
#### confirm-delete — delete one snapshot            (terse · sharp · fast)
brand: terse: fewer words, never fewer facts: the object, its size and the consequence on one line; the buttons are the verb and its no; y and n answer it

  delete nightly-0412? 41.2 GB, no undo
                                        [ delete ]  › [ keep ]
  y delete · n/esc keep · ? help
```

```
#### confirm-delete — delete one snapshot            (patient · thorough · plain)
brand: thorough: the question says what goes and what stays; the buttons name the outcome; the footer names each way out once

  Delete snapshot nightly-0412?
  It holds 41.2 GB from 12 April and can't be restored once deleted.
  The 13 snapshots after it keep their own copies; nothing else is removed.
                                  Delete snapshot   › Keep it
  ←→ choose · enter confirm · esc keep it
```

Same snapshot, same facts, same `ui.Dialog`, same theme slots, Keep focused in both;
what differs is the question's length and register, the button words and the footer's
verbs and keys (the terse one leaves ←→ and enter to ? help). The detail behind it
differs the same way: `r restore · d delete · esc back` against `r restore to a folder ·
d delete snapshot · esc back to the list`. Both keep the spec's flow, one key to one
confirm; a brand that changes it says why on a `flow:` line.

**DO** write each line from one of the three words, as what the user reads: the
question, the button labels, the footer's verbs (the action table's help text), the
detail's labels and empty copy.
Where the spec gives the words, keep its facts (what is acted on, the number, the
consequence) and word them in the brand's voice (`brand.md` § The Read Reaches Past The
Main Screen).
**DON'T** drop a fact to sound terse: terse means fewer words, never fewer facts, and a
confirm still names the object and what changes (`interaction.md` § Confirm By Consequence).
**DON'T** count what the theme already paints (`dialog: rules`, underline buttons, a
hue) as a brand line, or write `brand: nothing` because the spec has copy. `nothing,
because …` is for a screen whose words are not the brand's to choose: `brand: nothing,
because help lists the action tables as written`. A bare `nothing`, `none` or `-` gives
no reason, and `tuilint spec` warns on it [S-SLOTS].

## Sketch In Cells

Draw the 80×24 frame as text in the screen's block, with real content, exactly 80 wide.
It is cheap to change and it exposes width problems before code does:

```
  halyard                                                      prod / us-east-1
  Services 8 / Deploys / Incidents / Assistant
  SERVICE                                   VERSION   P99            ERRORS
  ──────────────────────────────────────────────────────────────────────────
▌ ▲ checkout-api                            v2.4.0    ▁▁▂▃▅▇██       4.81%
```

For a large or unusual screen, you may also mock it on the web inside a cell-locked
stylesheet (only `ch` and row units), then port by vocabulary. Most screens don't need
this; the kit's components are the vocabulary.

## Check With The User

Run `tuilint spec TUI.md` after every change to § States and keys, and fix what it
reports before the user sees the table.

Show the 80×24 sketches and the state list, and ask the one or two questions that change
the design (not "does this look good?"). Record decisions in `TUI.md` § Decisions. Then
hand off to `craft`.
