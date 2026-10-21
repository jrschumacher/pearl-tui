# tallyho: attack on read v2

v2 answers ten of my twelve objections fully, and the answers hold. I re-ran each one
past the same four interviewees. Two accepted fixes fail on a detail that defeats the
customer problem they were meant to solve. Of the two partial answers, "no split at
midnight" holds, and "config file, no form" holds only if the app re-reads the file.
Creep check: v2 added five states and four rules, and I can trace every one to an
interviewee. Only the note prompt in the picker costs the majority something.

## Interviews, round 2 (same four people, reacting to v2)

### Priya, typical daily user (full-stack developer, 3 clients, monthly on the 1st)
> "Opening on the last complete month on the 1st: yes, that's the click I wanted gone.
> Week total in the header is fine; I just need it somewhere I don't hunt for. The note
> prompt after every pick: I switch ten times a day and almost never write a note, so
> that's ten extra enters a day. I'll live with it if a single enter both picks and skips
> the note. If it's pick, enter, *then* enter again, it'll annoy me by Wednesday. On the
> 1st I open the app before I've started anything, so nothing's running and nothing's
> selected today. Whose invoice opens then?"

### Marco, power user (backend/devops, 5 clients, a 16 h cap, an auditing agency)
> "Enter on an invoice entry to fix it: that's the one. Midnight: I do late deploys,
> 22:00 to 01:30. I call that Tuesday's work and I'd put it on Tuesday's invoice line
> anyway, so no split is right. Splitting would give me two half-rows with the same note
> and my agency would ask why. A TOML file for rates is fine, better than a form; I keep
> it in my dotfiles. But this thing sits open in tmux for days. If I fix a rate in the
> other pane and the draft still says 'no rate', I'll assume it's broken."

### Jo, newcomer (web designer, first week, came from a spreadsheet)
> "Typing a new client and project in the picker: good, that answers day one. The file:
> I can edit a file if it tells me which one and what to type. I don't know what TOML
> looks like, though. If the file doesn't exist yet, or my client isn't in it, I'd be
> guessing at brackets and quotes. And the overnight thing: next morning it opens on
> today, today is empty, and the timer line says `e sets when it ended`. I press `e`.
> What does it open? There's nothing on today's list."

### Theo, the skeptic (product designer, Harvest, weekly agency plus monthly others)
> "The money rule adds up to the cent, so my first churn reason is gone. One number per
> client and period, kept on re-export, so my accountant's happy. 'Changed since export
> on 22 oct' is exactly what I asked for. Two things. One: the forgotten-timer fix has to
> really be one step from wherever I am, not 'go back to Wednesday, find it, press `e`'.
> Two: 'undo until the next change'. If I stop for lunch at 12:00 and don't touch
> anything, is `u` still sitting there at 14:00? Pressing it then would bill my lunch."

## Objections

### 1. BLOCKING: The overnight fix isn't one step, because `e` targets the selected entry and the running entry isn't on today's list
v2's own rules collide. The week opens on today. An entry belongs to the day it started.
The entry form opens "`e` on a selected entry". The morning after a forgotten stop,
today's list is empty and the running entry sits on yesterday, yet the timer line
advertises `e sets when it ended`. As written, the fix is ←, select the last row, `e`:
three steps, and the hint on screen lies about it. A builder has to guess whether `e`
with nothing selected opens the running entry.
*Evidence:* Jo: "I press `e`. What does it open? There's nothing on today's list." Theo:
"one step from wherever I am".
*Needs:* one rule. While the timer has run past midnight, `e` opens the running entry
from any day. Or the week opens on the running entry's day in that state. Either works;
say which.

### 2. BLOCKING: Nothing says when `clients.toml` is re-read, and the app stays open all day
The partial answer to v1 #1 only works if fixing the file fixes the draft. The read
names the file and the missing keys, then sends the user to edit it in another pane.
It never says whether the running app sees the edit. The audience keeps tallyho open in
a tmux pane for days, so "read at startup" means restarting to see a rate. A builder will
default to exactly that. Jo's and Marco's first experience of the config path would
then be a draft that still says "no rate" after they fixed it.
*Evidence:* Marco: "If I fix a rate in the other pane and the draft still says 'no rate',
I'll assume it's broken." Jo will edit it on her first Friday.
*Needs:* the file is re-read whenever the invoice draft opens, and when it changes on
disk while the draft is open. If the file can't be parsed, the draft names the file,
the line and the reason, the same as the entries-file error.

With that line added, the config-file answer holds. No interviewee asked for an in-app
form. Marco prefers the file, and Jo will edit a file if it's named and shows what to
type (see #3).

### 3. MINOR: A new client gets no section in the config file to fill in
Jo doesn't know TOML syntax. When the picker creates a client, the app could write that
client's section into `clients.toml` with the keys present and empty (rate, bill-to,
terms, cycle). Jo would then fill in values, not syntax, and the "missing keys" message
points at lines that already exist. One sentence in the read. It isn't a form.
*Evidence:* Jo: "I'd be guessing at brackets and quotes."

### 4. MINOR: The note prompt adds a keystroke to every switch
Most switches carry no note (Priya: "almost never"), and switches happen 5–15 times a
day. State that one enter on the picked row starts the timer without a note, and that
typing a note is one extra step only for those who want it. Also state that space (resume)
never prompts. Ola and Marco keep their note-at-start, and Priya keeps one key.
*Evidence:* Priya: "ten extra enters a day… it'll annoy me by Wednesday."

### 5. MINOR: An undo that is live "until the next change" can resume a stopped timer hours later
Undo of a stop removes the stop, so the time in between becomes tracked time. A `u`
still live after a two-hour lunch bills the lunch. Bound it. Undo stays live until the
next change *or* a few minutes, whichever comes first. After that, the entry form is
the fix.
*Evidence:* Theo: "Pressing it then would bill my lunch."

### 6. MINOR: The invoice's opening client has no fallback when idle and nothing is selected
The rule is the selected entry's client, else the running timer's. On the 1st, before
anything has started, today is empty and nothing runs. Name the fallback, such as the
client of the most recent entry.
*Evidence:* Priya: "nothing's running and nothing's selected today. Whose invoice opens
then?"

### 7. MINOR: The invoice rows overflow when several warnings apply at once
At 80×24, rows 4–6 hold the total and two warning lines. v2 now has four warning kinds:
no rate, running, changed since export, and unconfigured. Unconfigured and no rate, or
running and changed since export, can plausibly co-occur. State the order (whatever
blocks export first) and the overflow (`+1 more`). Do the same for project lines past
four at 80×24, where rows 7–10 also need a column-header row.

### 8. MINOR: Nothing says what the invoice number shows before the first export
The number is assigned at first export. Until then the facts row can't show
`invoice 2026-014` as if it were settled: two unexported drafts would both show the same
"next" number. Say what shows before export, for example `invoice number on export`.
*Evidence:* Theo's duplicate-number history; he will read a provisional number as a
real one.

### Partial answers: verdicts
- **#1 config file, no form: holds, given #2 (re-read) and preferably #3 (stub).**
  Blocking export only when bill-to is missing or the total is $0, and letting an unrated
  project through with a named warning, is the right line. A wrong address is a wrong
  invoice; a missing project is visible and fixable.
- **#3 no split at midnight: holds.** Marco's real late work belongs to the start day on
  his own invoices, and a split would give his auditing agency two rows with one note.
  Jo's forgotten timer is fixed by setting the end, after which it is a same-day entry
  anyway. One rule covers the strip, the week total and the invoice period edge. The only
  gap is #1 above, which is about reaching the entry, not about splitting it.

### Accepted fixes that hold (no objection)
#4 opening period (Priya; Sam's weekly flow lands on the current week once last week is
exported, which is right), #5 fix from the invoice (Marco), #6 the two-row strip (70 of
76 cols, `10h 05m` fits a 10-col column; the week total in the header satisfies Priya),
#7 the money rule (Theo, checked: 2,512 min × $120 ÷ 60 = $5,024.00), #8 numbering kept
on re-export, #9 the changed-since-export state (Theo), #10 truncation order (Marco),
#11 the switch with no gap, #12 assumptions labelled.

### Creep check: nothing to cut
The new states (first run, running since another day, unconfigured client, changed since
export, exported persisting) and rules (money, numbering, opening period, midnight) each
trace to one interviewee's stated need. The export log is heavier machinery, but it is
the only way to produce Theo's "changed since export", so it earns its place. The one
cost to the majority is the note prompt (#4), which is a wording fix, not a cut.

VERDICT: REVISE
