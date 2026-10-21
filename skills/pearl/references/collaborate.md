# Collaborate: the person picks taste from renders, and a work order carries what they said

Two things get lost between a person and the agent building their TUI: the choices only
they can make, and what they said in conversation. Show the first as renders at named
pick points; write the second into a work order that a fresh session can run.

## The Person Picks Taste; The Skill Decides The Rest

Across four brands graded blind, no splash style won everywhere: field was picked as the
most refined twice, the wordmark twice, and the brand fit split three ways. Taste is the
person's. The laws are not: a key table or a contrast ratio put to a vote gets a guess.
So stop at these pick points, and only these:

- **the brand read**, when there is no brand: two reads that differ in kind (`cmd-init.md`)
- **the main screen's move and signature** (`brand.md` § Move A Region)
- **a splash or loading style**: field, wordmark, intro (`cmd-art.md`, `cmd-animate.md`)
- **a layout where two arrangements both pass the laws**: where the selection's detail
  goes, a live pane beside or below its list at 80 columns

**DO** decide keys, states, budgets, contrast, confirm rungs and every law yourself,
citing the reference that decides it.
**DON'T** ask "does this look good?": the answer is yes, and it carries no reason.

## Show, Then Ask One Question

- **Two to four options that differ in kind**: composition, motion or density, never only
  hue. A palette swap on one layout is one option.
- **Before code**, sketch each at 80×24 in cells (`cmd-shape.md` § Sketch In Cells) and
  render them to one contact sheet: `cellshot a.txt b.txt c.txt --sheet pick-move.png`.
  **After code**, put each behind a harness flag (`-splash field|wordmark|intro`,
  `-variant a|b`) and render it at 80×24 and 120×40, with a GIF for anything that moves
  (`cellshot f*.ansi --gif out.gif --ms 50`).
- **Show the image**, not a description of it: open or attach it where your harness can,
  or give its path. A person picks from pixels; a paragraph about a layout gets a guess.
- **Ask one question**: which one, with your recommendation first and its reason, and
  ask for one line of theirs.
- **Record it** in `TUI.md` § Decisions as `<date> picked <option> for <point> because
  <their words>`, and write the reason as a rule for later screens. "Great for a very
  small resolution", said of the quiet splash every time it came up, became: quiet is
  every style's fallback below its size floor.
- **Delete the options not picked**, except one that becomes a fallback.
- **The person is away**: build and render the options the same way, take your
  recommendation, and record it under `## Assumed` with its reason, so they can overrule
  it from the sheet later.

## Work Orders Carry The Conversation

What the person said lives in the conversation, and a fresh session, a compacted context
or another model loses it. `TUI.md` holds the product; a work order holds the work. It is
a prompt that stands alone, so a session with only the repo and the skill can run it.
`init` and `shape` end by writing one per next step into `TUI.md` § Work orders:

```
### 1. craft: the setup flow, every state on fixtures
keep: "`lanternfish init --token <t>` keeps working with no screen at all: CI runs it."
keep: "A pasted token is never shown again after it's pasted."
reference: none   (or exact: <axes> and inspiration: <the rest>, from § Reference)
method: pick the step column's form at three options (-variant), then the loop (verify.md)
deliver: frames/ at 80×24 and 120×40, sheet-80.png, sheet-120.png, pick-steps.png
decide: the step column's form; work stops at the sheet until it is picked
```

**DO** quote the person word for word on each `keep:` line: what they said must stay
(a flag CI runs, a step order support docs follow, a key). A default nobody stated is
not a keep; it goes under `## Assumed`.
**DO** point at `TUI.md`'s sections instead of copying them: a screen block or key table
copied into an order drifts from the one the next command edits.
**DO** rewrite the orders at every hand-off and delete the ones done, then print the
next one in the hand-off, ready to paste, instead of a bare `Next: craft`.
**DON'T** write a `decide:` the skill decides (keys, contrast): `decide:` names a pick
point, or `nothing; build it`.
