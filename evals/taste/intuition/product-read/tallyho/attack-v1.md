# tallyho: attack on read v1

Grader's stance: the read gets the big calls right. Timer first, one day of entries,
money only on the invoice, and the consultant's dashboard, agents, tasks and journal are
all cut. I agree with every line of the cut list, and the interviews below back it. The
objections are about what the read leaves undefined on paths customers hit every week,
and about two places where it cut too much.

## Interviews (simulated, grounded in the brief's audience)

### Priya, typical daily user: freelance full-stack developer, 3 clients
> "Toggl desktop, in the menu bar. I switch maybe six to ten times a day, mostly when a
> client pings me. The thing I look at most is just: is it on, and on what. End of day I
> check today's total, and on long days that's ten, eleven hours, so don't assume single
> digits. I invoice monthly, always on the 1st or 2nd, *for the month that just ended*.
> Toggl opens its report on the current month, so every month I click back once. Annoying
> every time. Money while I'm working? No. I look at money when I invoice and when my
> accountant asks. Twice a month I leave it running over lunch and fix the end time after."

### Marco, power user: freelance backend/devops developer, 5 clients
> "Timewarrior, plus a shell script that makes a CSV, plus a doc template. One client is a
> retainer capped at 16 hours a week, so I check that client's week hours most days. I
> don't need the app to know the cap, I know it's 16. Two clients get weekly invoices, an
> agency gets monthly, and the agency audits: every line needs what I did. Half the time I
> notice a missing note or a wrong project *while reviewing the invoice*, and with my
> script that means leave, `timew` edit, regenerate. That's the part I hate. I usually type
> the note when I start, sometimes I add it halfway through. Yes, I run coding agents, but
> I bill my time, not theirs. Tracking agents is a gimmick to me."

### Jo, newcomer in their first week: freelance web designer, 2 clients, came from a spreadsheet
> "Day one I opened it and there was nothing to pick. My first question was how to add my
> client. My second was where my rate goes. Day two I forgot to stop before bed and the next
> morning it said seventeen hours. I didn't know if that was yesterday's hours or today's
> or both, and I wanted to say 'it actually ended at 18:00' in one go. I'll invoice my
> first client on Friday. If the invoice comes out with no address on it or a $0 total, I
> need it to tell me exactly what file to put that in."

### Theo, the skeptic who would churn: freelance product designer, 4 clients, Harvest
> "The terminal's a side window for me. I'd churn on three things. One, if fixing a
> forgotten timer takes more than one stop. Two, if my numbers don't add up. My accountant
> once caught two invoices with the same number, and if the PDF total doesn't match the
> lines to the cent, I'm gone. Three, any dashboard telling me how motivated to be. I
> invoice one agency weekly and the rest monthly. I check the week on Friday, day by day,
> and fix what's wrong. I like that an invoice can be re-exported, but I need to know if
> the PDF I already sent is now out of date."

## Objections

### 1. BLOCKING: The newcomer's first run is undefined (no projects, no rates, no bill-to)
The read's empty-week state says "space or `s` to start", but on first run the picker has
nothing in it, and the read never says how a client or project comes into existence.
Typing a new name could create it, or a config file could be required first; a builder
has to guess. The invoice has the same hole. Rates, bill-to and the cycle are "config, set
once per client", but nothing says what the draft shows when that config is missing. The
"no rate" partial covers one project, not "this client has no rate, address or cycle at
all". It also doesn't say whether export is allowed when bill-to is empty.
*Evidence:* Jo, day one: "how to add my client… where my rate goes"; on her first invoice
she needs to be told "exactly what file to put that in".
*Needs:* (a) how a new client / project is created from the picker; (b) a first-run week
state; (c) an unconfigured-client invoice state that names the file and the missing keys,
and says whether export is blocked or allowed.

### 2. BLOCKING: The running entry's note and start time have no defined way to set them
The timer line shows a note, and the invoice depends on notes, which is the read's own
departure #4. Yet the read never says when a note is entered. It could be at start, in the
picker, or later by opening the running entry in the entry form. It also doesn't say
whether the entry form can open on the *running* entry to fix its start ("forgot to start
20 minutes ago"). Job 4 lists "forgot to start", but the only path the read gives is
"add a missed entry", and that doesn't fit a timer that is still running.
*Evidence:* Marco: "I usually type the note when I start, sometimes I add it halfway
through." Jo/Priya: they fix start and end times on live or just-stopped timers.
*Needs:* one line each: the note on start (picker, or straight after), editing the running
entry (note and start), and the key that opens the entry form.

### 3. BLOCKING: A timer left running past midnight is an undefined state
Priya, Jo and Sam (the read's own persona) all forget to stop, and Sam does it twice a
week. The read doesn't say:
- how the timer line reads when the start was yesterday (`since 13:12` is wrong; it needs
  the day);
- which day's total in the strip the hours count toward (the start day, or split at midnight);
- what the one-step fix is ("it ended at 18:00").
These choices change the day strip, the week total and the invoice, so a builder can't
safely guess them.
*Evidence:* Jo: "it said seventeen hours. I didn't know if that was yesterday's hours or
today's or both." Theo churns "if fixing a forgotten timer takes more than one stop".
*Needs:* a "running since a previous day" state, the rule for attributing entries that
cross midnight, and a stop-at-a-time path (in the entry form is fine).

### 4. BLOCKING: The invoice draft's opening client and period are undefined, and "current period" is wrong for monthly invoicers
`i` opens "an invoice draft for one client", but the read doesn't say which client (the
running timer's, the selected entry's, or the last one invoiced) or which period. Monthly
invoicers draft on the 1st–2nd for the month that *just ended*. If the draft opens on the
current period, they land on the "empty period" state every month and have to press ←
every time.
*Evidence:* Priya: "always on the 1st or 2nd, for the month that just ended… every month
I click back once. Annoying every time." Theo invoices weekly and monthly in the same week.
*Needs:* the opening client rule and the opening period rule (for example: the most
recent period with unexported hours for that client, or the last *complete* period when
the current one has just started).

### 5. BLOCKING: You can't fix an entry from the invoice, where errors are found
The read keeps the entry list on the invoice so the client can audit notes, but makes it
look-only. Checking is half of job 5, and the check is where wrong projects and missing
notes turn up. To fix one today, you press esc, step ←→ back day by day (up to ~30 presses
for a monthly invoice), edit, press `i`, re-pick the client with `c`, and re-find the
period. The entry form overlay already exists, so opening it from an invoice entry row adds
no screen.
*Evidence:* Marco: "Half the time I notice a missing note or a wrong project while
reviewing the invoice… That's the part I hate." Theo: "I check… and fix what's wrong."
*Needs:* enter on an invoice entry opens the same entry form, and the draft's total and
lines update when it closes.

### 6. BLOCKING: The 80×24 day strip doesn't fit what the read puts in it
`mon 7h 30m` is 10 cols, so 7 days take 70 cols plus 6 separators. That leaves only one
space between days (`30m tue`, which reads as one run) and no room for:
- the week total (`26h 31m this week`, job 3), which has no row at 80×24;
- the selected-day marker;
- any day of 10 hours or more (`mon 10h 05m` is 11 cols, which pushes the strip past 76).
*Evidence:* Priya: "on long days that's ten, eleven hours, so don't assume single digits."
Everyone checks the week total daily.
*Needs:* a row budget at 80×24 that holds 7 days at worst-case width, the selected day and
the week total. Either drop the minutes in the strip, use two rows, or move the total to
its own row and say which.

### 7. MINOR: The money rule is unstated (rounding, and how lines sum to the total)
`$5,024.00` for `41h 52m` at `$120/h` means exact minutes, no rounding. But entries are
stored to the second and shown to the minute, so line hours can disagree with the total by
a minute and amounts by cents. State the rule once: bill exact minutes, or seconds; each
line's amount comes from that line's own time; the total is the sum of the lines. A
rounding *option* is not needed in this slice.
*Evidence:* Theo: "if the PDF total doesn't match the lines to the cent, I'm gone."

### 8. MINOR: The invoice facts have no rules behind them
`invoice 2026-014` and `due in 14 days` appear with no rule. Say whether the number is
assigned on first export and kept on re-export of the same client and period (so
re-exporting never burns or duplicates a number), and say what the due date counts from
(the export date).
*Evidence:* Theo: "two invoices with the same number."

### 9. MINOR: The "exported" state doesn't say if it persists or goes stale
Re-export is allowed and nothing is sent, so the PDF the customer already mailed can fall
behind the draft after an entry is fixed. Say whether reopening the draft days later
shows `written 22 oct` for that client and period, and whether it shows that entries
changed since then. Objection 5 makes this state more likely.
*Evidence:* Theo: "I need to know if the PDF I already sent is now out of date."

### 10. MINOR: The 80-col entry row has no truncation order
`marlow / spring catalogue retouching` is 36 cols before the note and duration. Say which
field gives way first at 80 cols. It should be the project, never the duration, and the
note last, since the note is what audited clients read.
*Evidence:* Marco's agency audits the notes.

### 11. MINOR: The switch and undo semantics are implied, not written down
Say that `s` while running stops the current entry and starts the next at the same
instant, with no gap or overlap. Say how long `u undo` stays live: until the next action,
or for N seconds.

### 12. MINOR: The persona specifics are the designer's own, so label them
Dev's 20 h cap and Ola's auditing client aren't in the brief. Interviews here back both
(Marco has a cap and an auditing agency), and back the cap staying in the user's head
rather than the app. So the client-hours row and the invoice notes are earned. Still, the
read should say these came from assumed personas, not from the brief, under `## Assumed`.

### Agreed (no objection)
- Cutting all money from the week (Priya: "Money while I'm working? No.").
- Cutting the dashboard, the motivation view and the pie chart (Theo, objection three:
  "any dashboard telling me how motivated to be").
- Cutting agents (Marco: "I bill my time, not theirs").
- Cutting tasks, the journal and focus inference.
- Leading the week with today rather than a grid (Theo steps through day by day on
  Friday; ←→ serves him).
- Invoice periods following each client's cycle (Priya monthly; Marco and Theo mixed).

VERDICT: REVISE
