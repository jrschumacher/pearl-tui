# shape — design the screens on paper before any code

Input: `TUI.md` and the user's ask. Output: a screen spec the user agrees to, written
into `TUI.md` § Screens (one block per screen, in the template's shape: job, regions at
both sizes, collapse rule, components, fixtures, brand line, any flow line, sketch) and
§ States and keys, and nowhere else: a second file gets skipped, and the project's own
docs belong to the user. No Go yet.

## Read The Product, Then Let Customers Attack It

A brief lists screens; it doesn't say what the people using them need first. Three
briefs built as written put a dozen money panels on a time tracker's week and a
reading-time column nobody could parse. A grader's think-aloud ideas taken as orders
(an agents' hours view, a health dashboard on a settings tool) added screens no customer
asked for. Before the first block, write the read into § Product:

- `customers:` two or three, each with how often they use it and what they use today
- `jobs:` ranked by how often a customer does each; the first is the main screen's job
- the slice: the brief's screens (no more), and what each shows, ranked; every idea left
  out goes in § Decisions as `cut: <idea> — <why>`

Then have it attacked by simulated customers, with a subagent where you have one (the
fallback is a fresh message of your own, as `verify.md` § The Critic describes):
four interviews (a daily user, a power user, a newcomer in their first week, a skeptic
about to quit) in their voices, then objections tagged blocking or minor, each with the
customer who raised it. Answer each in the read; repeat until nothing blocks, at most
three rounds. Three briefs rebuilt from such a read, graded blind against the literal
builds, won wherever scope was the problem: "much cleaner to process quickly".

**DO** weigh a stakeholder's idea as one voice: test it against the customers, neither
adopt nor dismiss it. One raised twice is still one voice.
**DO** answer an undefined state with a line (`a timer left past midnight counts to the
day it started`), not a feature or a screen.
**DON'T** cut a value because it reads two ways: label it (`3 min read`, apart from the
time received). The read that deleted it lost to the build that kept it unlabelled.
**DON'T** stop at answering objections one by one: re-read each screen whole after. A
reply answered two objections apart showed the original beside the draft and again as
its quote, and lost.

## Depth Follows The Domain, Then The Brand

Before the first block, answer one question: does this domain, or this brand, earn more
or fewer steps than the spec's default flow? Depth is how many steps a task takes, how
much shows before an act (a preview, a diff, a summary), what is one key away against a
screen away, and whether an expert path exists (a command line, batch selection). Read
the domain first (`SPEC.md` or the ask, § Product, § Users and terminals), then the
read's three words: the domain sets the range, and a word picks a place in it.

- **High consequence, many objects** (cluster ops, deploys, infrastructure, finance)
  earns what an act will change, and to what, shown before it runs (`3 replicas v2.4.0
  → v2.4.1 · 1 unchanged`): on the screen its key is pressed from, one key away, or in
  the confirm its rung already has, never as a screen in front of a rung-1 act, whose
  key is the act. One wrong act there reaches many objects.
- **Experts earn the expert path**: users who run it all day (§ Users and terminals:
  operators, on-call) or a word such as "for operators" earn `:` for a command line and
  `space` to mark a batch, because they act on many objects at speed.
- **Personal, low consequence** (mail, notes, a todo list) earns fewer steps and fewer
  screens: the item opens in place and the act happens on its key, because an undo
  covers a slip and every extra step is paid on every use. A screen the ask or
  `SPEC.md` names stays.
- **A word places the flow within that range, and never adds a step to a flow the
  domain keeps light**: "calm" earns one thing at a time on a deep flow, with the rarely
  read fields behind a key, and on a todo list changes nothing. A restyle keeps the
  original's flow unless the user asks.

Write the answer on the main screen's block as a `flow:` line that names the domain
fact (who acts, on how many objects, whether the act can be taken back) and any word
that moves it: `flow: the spec's, because one person's own list: every act is one key
and the same key undoes it`, never `the spec's, because the spec defines it`. Give every
other block whose steps, preview or detail differ from the spec's its own line;
`tuilint spec` flags one left as a placeholder [S-SLOTS]. The same apply step of a DNS
zone editor, two brands:

```
#### apply — publish the edited records              (exact · fast · for operators)
flow: space marks records and a applies the marked set, because for operators; the rung-3 confirm lists each record old → new, because resolvers cache the new answer for its TTL, so republishing the old one can't take it back
```

```
#### apply — publish the edited records              (calm · plain · careful)
flow: the spec's, one record at a time and no batch path, because its users edit one zone at a time and calm asks for nothing more; the rung-3 confirm lists the record old → new, because resolvers cache the new answer for its TTL, so republishing the old one can't take it back
```

**DO** keep what the product does: `SPEC.md` decides which tasks exist and what each
does, and depth changes only how a task is reached and what shows on the way.
**DON'T** set the depth with a confirm (`interaction.md` § Confirm By Consequence).
Depth never moves an act's rung, so no confirm goes in front of a rung-1 or rung-2 act
or a rung-4 act outside its signal's state; a rung-3 act keeps its confirm or its
form's review step, a rung-4 act in its signal's state keeps its confirm, and neither
gets a second one after it.
**DON'T** let a flow differ from the spec's where no domain fact or brand word on its
`flow:` line asks for it: two brands of one spec then differ by chance, which is drift,
not identity, and the critic marks it.

## For Each Screen

1. **Job and data**: the one thing the user comes to this screen to do, and what they
   look at first, then the journey around it. Write three lines in its block: `job:`
   (the task, and its flow from `flows.md`: monitoring, checkout, media, browse,
   configure), `data:` (each region's data by kind: live, background, money, the
   product's purpose) and `carry:` (what this screen keeps in view from the product's
   purpose, or nothing, because it is that purpose). Then load `flows.md` § Name The
   Data Before Placing It and the section the `job:` names: they decide where the
   live pane goes, what becomes a glyph and how loud the brand is, before any region is
   drawn. A flow is a screen's, not the app's: a booking app browses until its pay step.
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
