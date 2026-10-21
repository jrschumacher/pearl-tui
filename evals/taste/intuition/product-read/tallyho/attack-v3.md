# tallyho: attack on read v3 (final round)

All eight round-2 objections are closed, and each fix solves the customer's problem as
stated, not only the letter of the objection. The final creep check across the whole read
finds nothing to cut. What remains is five builder notes, none of which a customer hits
as a broken path in their first week.

## Interviews, round 3 (same four people, reacting to v3)

### Priya, typical daily user
> "Enter starts it, tab if I want a note. That's my one key back. And on the 1st, with
> nothing running, it opens on the client I worked on last. That's usually who I'm
> invoicing first anyway. Fine."

### Marco, power user
> "Re-read on change, and the parse error gives me a line number: that's how my other
> tools behave. Fix a rate in the other pane and the draft updates, good. One thing I'd
> trip on eventually: when I add a missed entry on Friday, can it overlap one I already
> have? I wouldn't notice until the agency did."

### Jo, newcomer
> "The picker writes my client into the file with empty slots, so I just type the rate
> and the address. Overnight: `e` from today opens the running one and I set 18:00. That's
> what I wanted. Day one I'd probably press `i` just to see what it does, before I'd
> tracked anything."

### Theo, skeptic
> "Undo dies after five minutes, so lunch is safe. 'Number assigned on export' instead of
> a fake number, good. Warnings never hide behind the entry list, good. Nothing here tells
> me how motivated to be. I'd keep it."

## Round-2 objections: closure check

| # | Objection | Closed? | Check |
|---|---|---|---|
| 1 | Overnight fix not one step | yes | `e` opens the running entry from any day and selection while past midnight, and the timer line's hint is now true. Jo: one step. |
| 2 | Config re-read | yes | Re-read when the draft opens and on change on disk; a parse error names the file, line and reason, and blocks export. Marco. |
| 3 | No section for a new client | yes | The section is written with empty keys and the file created if needed. Jo fills in values, not syntax. |
| 4 | Note prompt on every switch | yes | Enter starts with no note, tab adds one, space never prompts. Priya keeps one key; Ola and Marco keep note-at-start. |
| 5 | Undo can bill a lunch | yes | Live until the next change or 5 minutes, whichever comes first. Theo. |
| 6 | No fallback client | yes | The fallback is the most recent entry's client. Priya. (First run, with no entries at all, is minor #1 below.) |
| 7 | Invoice rows overflow | yes | Warnings in a fixed order, never hidden; entries yield down to 3 rows; extra projects fold into a summed `+N more` row. Worst case at 80×24 checks out: total row 4, four warnings rows 5–8, header plus four lines rows 9–13, facts and export rows 14–15, entries from row 16 to row 23, which is more than 3. |
| 8 | Provisional number | yes | `number assigned on export` and `due 14 days after export`. Theo. |

## Final creep check

The read's surface is two screens and three overlays (picker, entry form, help). Each
screen element maps to a job with a stated frequency. The states (week 8, invoice 10)
each trace to an interviewee or to the brief's "every state of both". The rules (money,
numbering, opening period, midnight, undo window, config re-read) each answer a question a
builder would otherwise guess at. The only addition since v2 is the "config unreadable"
state, and it was asked for. Nothing to cut. The cut list still holds: no money on the
week, no dashboard, no agents, no tasks, no journal, no charts.

## Objections

### 1. MINOR: `i` with no entries at all is undefined
The fallback chain (selected entry, then running timer, then most recent entry) ends
empty on first run. Jo will press `i` on day one to explore. Builder note: `i` on first
run shows the invoice's empty state, saying there are no clients yet and that `s` creates
one. Do not open a blank client. Minor, not blocking: nothing is lost and no wrong
invoice results, so the builder's job is only to choose words.

### 2. MINOR: The entry form has no overlap rule
`a` (add a missed entry) and editing a start or end can overlap an existing entry. Nothing
says whether that is refused, allowed or flagged. Overlap counts the same minutes twice,
on the strip and on the invoice. Builder note: refuse end-before-start, and on overlap
name the other entry and refuse the save, so the invoice can never double-bill. This
wasn't raised in earlier rounds. It's minor because a builder's safe default is obvious
and it typically shows up in Friday fixing, not on day one.
*Evidence:* Marco: "I wouldn't notice until the agency did."

### 3. MINOR: Several no-rate projects don't fit "one line each" in four warning rows
With two unrated projects, "one line each" plus the other three kinds gives five warning
lines against a 0–4 row budget. Builder note: unrated projects share one line
(`no rate: naming workshop, research · 5h 10m left out`).

### 4. MINOR: The format of "stopped since another day" isn't stated
The `since wed 18:02` form is defined only for a running timer. Builder note: apply the
same rule to `not tracking since …` when the stop was on another day.

### 5. MINOR: An export error may need more than the two rows the table allots
"What failed, what to do" can run to two lines at 80 cols. Builder note: the export
error takes rows from the entries list, the same way warnings do, down to the 3-row
minimum.

No BLOCKING objections remain.

VERDICT: CONVERGED
