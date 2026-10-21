# tallyho: product read v3

## Answers to attack v2

| # | Objection | Verdict | Why |
|---|---|---|---|
| 1 | Overnight fix not one step | accept | While the timer has run past midnight, `e` opens the running entry from any day and any selection. |
| 2 | When `clients.toml` is re-read | accept | When the draft opens and whenever the file changes on disk; a parse error names the file, the line and the reason. |
| 3 | New client gets no section | accept | The picker writes the new client's section with empty keys, creating the file if needed. Jo fills in values, not syntax. |
| 4 | The note prompt costs every switch | accept | Enter on the picked row starts at once with no note; tab adds one. space never prompts. |
| 5 | Undo can bill a lunch | accept | Undo lasts until the next change or 5 minutes, whichever comes first; after that, use the entry form. |
| 6 | No fallback client for the invoice | accept | It falls back to the client of the most recent entry. |
| 7 | Invoice rows overflow | accept | Warnings never hide; they come in a fixed order, and the entries list gives up the rows (down to 3). Project lines past what fits fold into one summed `+N more` row. |
| 8 | Provisional invoice number | accept | Before the first export the facts row says `number assigned on export`. |

v1's answers stand as revised in v2, and the researcher upheld both partial answers: no in-app form for rates and bill-to, and no split at midnight.

Units everywhere: durations `7h 30m` (running `1h 24m 07s`), clock times 24h `09:02–11:12`,
money `$6,544.00`, rates `$120/h`, counts with their noun (`19 entries`).

## 1. Customers
- **Dev, freelance backend developer, 3 clients.** A terminal open all day in tmux; switches 8–15 times a day. Her retainer client caps at 20 h a week, which she keeps in her head and checks against that client's week hours daily. Invoices monthly, on the 1st for the month just ended. Today: Toggl in a forgotten tab and a spreadsheet.
- **Sam, freelance product designer, 4–5 clients.** Figma first, the terminal in a side window; 5–8 switches a day. He forgets to stop twice a week and fixes entries on Friday, day by day. Invoices one agency weekly and the rest monthly. Today: Harvest free tier and a notes file.
- **Ola, contract front-end developer, 1 main client plus 1 small one.** The main client audits every line, so she types the note at start or halfway through. Invoices weekly and finds wrong notes while reviewing the invoice. Today: `timew` and a doc template.

## 2. Jobs (by how often)
1. **Switch what I'm timing**: start, stop or move (5–15 a day).  2. **Glance: running, on what, since when** (dozens a day).
3. **Check today's and the week's hours** (daily).  4. **Fix an entry**: project, start, end, note, or a missed one (2–3 a week).
5. **Draft one client's invoice for a period, check it, fix it, export the PDF** (weekly or monthly).
The week screen serves job 1, with job 2 always on top of it. The invoice serves job 5 and, from it, job 4.

## 3. The first slice: two screens plus overlays

**Always visible.** The header (row 1): `tallyho` and where you are; on the week it also holds
the week total (`19–25 oct · 26h 31m this week`). The timer line (row 2): when running,
`1h 24m 07s` elapsed, client / project, note, `since 13:12`, or `since wed 18:02` when the
start was another day. When stopped: `not tracking since 14:36` and what space would resume.
The footer (last row): this screen's keys, with help at the right.

**Overlays.**
- **Picker** (`s`): type to filter, most recent first. Enter on a row starts it at once with no note; tab on the row opens one note line first. A name that matches nothing offers `new: <client> / <project>`, creates both, and writes the client's empty section to `clients.toml`. space (resume) never opens the picker.
- **Entry form** (`e` on the selected entry, `a` to add one on the selected day; while the timer has run past midnight, `e` opens the running entry from any day): client / project, start, end, note, and delete with a confirm. On the running entry, setting the end stops the timer at that time. That is the one-step fix for a forgotten timer.
- **Help** (`?`).

**Switch and undo.** `s` while running stops the current entry and starts the new one at the
same second, with no gap and no overlap. A stop or delete puts `u undo` on the status line,
which stays live until the next change or 5 minutes, whichever comes first, so an undo can't turn a lunch into billed time.

**Midnight.** An entry belongs to the day it started, with all its hours, on the strip, in the
week total and on the invoice. While it runs past midnight, the timer line names the start
day and adds `e sets when it ended`.

### Week (main)
Job: switch the timer, see the day, fix an entry. Shows, ranked:
1. The timer line. space starts or stops; `s` switches.
2. The selected day's entries in time order, the running one last and ticking: `09:02–11:12  northwind / site redesign  hero animation  2h 10m`. Opens on today. At 80 cols the project truncates first, then the client; the note truncates last and keeps at least 20 cols; times and duration never truncate.
3. The day strip: 7 days, each with its hours (`10h 05m` worst case), the selected day marked. ←→ moves the selection.
4. This week's hours per client, largest first (`northwind 17h 51m`).
5. The status line when it applies: unreadable lines (`2 lines unreadable (212, 340), hours not counted`), or undo.

| | 80×24 | 120×40 |
|---|---|---|
| header, timer | rows 1–2 | rows 1–2 |
| day strip | rows 4–5: day names over hours, 7 columns × 10 cols = 70 | left column, 26 cols: one row per day |
| entries | rows 7–20, full width, about 14, scrolls | right, about 90 cols, rows 4–37 |
| client hours | row 22, one row, `+2 more` when it overflows | left column under the days, one row per client |
| status, footer | rows 23, 24 | rows 39, 40 |

States: First run: no entries and no projects. The entries area says to type a client / project name after `s`. Empty week: the strip shows `0h` and one line says space resumes the last project. Running. Running since another day. Just stopped, with undo. Idle. Partial: unreadable lines, named. Error: the entries file can't be read. It shows the path, the reason and `r` retry, and the timer line says the timer's state is unknown.

Not here: money, other months, per-project breakdowns, charts, an entry detail panel.

### Invoice draft (one client, one period)
Job: confirm what I'm billing this client, fix what's wrong, write the PDF. `i` opens it.
- **Client**: the selected entry's client; else the running timer's; else the most recent entry's. `c` switches client in the picker.
- **Period**: that client's cycle (week or month, monthly by default). It opens on the last *complete* period unless that period has an export logged, and then on the current one. ←→ moves one period; esc goes back to the week.

Shows, ranked:
1. **Total**: `$6,544.00 for 54h 32m, 19 entries`.
2. **What changes or blocks the total**, one line each, in this order and never hidden: unconfigured or unreadable config, no rate, running, changed since export. For example:
   - `naming workshop has no rate: 3h 15m left out`
   - `1h 24m running on site redesign not included until you stop it`
   - `changed since export on 22 oct: 2 entries; export again to update the PDF`
3. **Per-project lines** under a column-header row: project, hours, rate, amount. Lines that don't fit fold into one `+N more projects` row with their summed hours and amount, so the column still adds up.
4. **Invoice facts**, one row: `invoice 2026-014 · bill to northwind studio · due 5 nov (14 days)`. Before the first export: `number assigned on export · bill to northwind studio · due 14 days after export`.
5. **Export**: the file path and the action, then the result or the error (what failed, what to do).
6. **Entries in the period**: date, project, hours, note; enter opens the entry form, and the total and lines recompute when it closes. Truncation follows the week's order.

**Money rule.** Each entry is billed to the nearest minute. A line's hours are the sum of its
entries' minutes, and its amount is those minutes × rate ÷ 60, rounded to the cent. The total
is the sum of the line amounts, so the PDF adds up to the cent.

**Invoice facts rule.** The number is the next unused `<year>-<nnn>`, assigned on the first
export of that client and period and kept on every re-export, so a number is never skipped or
reused. The issue date is that first export, and the due date is the issue date plus the
client's terms. Exports are logged with the time and the lines they wrote, which is how the
draft knows it changed.

**Config** is one file, `~/.config/tallyho/clients.toml`: rate per client or project, bill-to,
terms and cycle. It is re-read when the draft opens and whenever it changes on disk, so a fix in another pane shows at once. A client created in the picker gets its section written with the keys present and empty. When keys are missing, the draft names the file and the missing keys
(`northwind: no rate, no bill_to · ~/.config/tallyho/clients.toml`). Export is blocked when
bill-to is missing or the total is $0. A project without a rate is left out and named, and
export is still allowed. If the file can't be parsed, the draft names the file, the line and the reason, and blocks export.

| | 80×24 | 120×40 |
|---|---|---|
| header, timer | rows 1–2 | rows 1–2 |
| total, warnings | row 4; 0–4 warning rows | row 4; 0–4 warning rows |
| project lines | header + 1–4 lines | header + 1–8 lines |
| facts, export | 2 rows | 2 rows |
| entries | the rest to row 23: 8 rows typical, never fewer than 3, scrolls | the rest to row 39: about 20, full width |

States: Ready. Unconfigured client: names the file and keys, and blocks export if bill-to is missing. Config unreadable: names the file, line and reason, and blocks export. Partial: a project with no rate, or unreadable lines. Empty period. Exporting: a spinner for 1–2 s; esc cancels. Exported: persists, with the date of the last export. Changed since export. Export failed.

Not here: editing rates, addresses or terms in the app; sending; paid status; tax; amounts per entry; rounding options.

## 4. Cut list (unchanged; the researcher agreed with all of it)
- **Consultant**: dashboard main screen (the main job is switching); tasks and to-dos (a second product); earnings, goals and motivation (no one acts on them while timing); AI agents and their feed (not in the brief; "I bill my time, not theirs"); concurrent and always-shown recent timers (one person, one timer; the picker shows recent); notification header, 2–3 panes, slide-out panel (nothing to notify, no tasks); journal as the second screen (entry notes cover the audit); focus inference (guesses intent); pie chart (the client-hours row says it); brand placement (visual, not this read's).
- **Old build**: money on the week, the October-so-far, by-week and projects-this-week panels, the entry detail panel and the per-day tally bars (they repeat the rows or show money no one acts on until invoicing); week number (people recognise the date range); entry-count column, export page count and seconds (no decision rides on them); month-only periods (weekly invoicers).
- **New in v2**: in-app editing of rates and bill-to (set once; the config file plus a draft that names the missing keys is cheaper); splitting entries at midnight (rare real work past midnight doesn't justify two rows per entry); rounding options (nobody asked; exact minutes add up).

## 5. Departures from the brief
- **The week leads with the timer and one day, not a grid of the whole week**: jobs 1–2 happen dozens of times a day.
- **Fixing and adding entries in an overlay, from the week and from the invoice**: Sam fixes weekly, Ola finds errors while reviewing; a wrong entry makes a wrong invoice.
- **The invoice period follows each client's cycle and opens on the last complete one**: monthly invoicers draft the month just ended.
- **The invoice shows each entry's note**: Ola's client audits every line.

## Assumed
- Dev's 20 h weekly cap and Ola's auditing client are invented for the personas, not taken from the brief. Interviews corroborate both (Marco: a 16 h cap kept in his head, and an agency that audits).
- The config is a TOML file; the cycle defaults to monthly; the undo window is 5 minutes.
