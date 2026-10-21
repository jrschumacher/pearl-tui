# tallyho: product read v2

## Answers to attack v1

| # | Objection | Verdict | Why |
|---|---|---|---|
| 1 | First run undefined | partly | Typing a new name in the picker creates it; rates and bill-to stay in one named config file, not an in-app form. Export is blocked only when it would produce a wrong invoice. |
| 2 | Note and start of the running entry | accept | The picker asks for an optional note; `e` opens the form on any entry, the running one included. |
| 3 | Timer running past midnight | partly | Defined: the timer line names the start day and `e` sets the end in one step. Entries count toward the day they started, with no split at midnight: one entry is one row. |
| 4 | Invoice's opening client and period | accept | Opens on the selected entry's client and that client's last complete period, unless that period is already exported. |
| 5 | Fix an entry from the invoice | accept | Enter on an entry row opens the same form; the total recomputes. This adds no screen. |
| 6 | 80×24 day strip doesn't fit | accept | The strip takes two rows of 10-col columns, and the week total moves to the header. |
| 7 | Money rule | accept | Stated once under Invoice. |
| 8 | Invoice number and due date | accept | The number is assigned on first export and kept on re-export; the due date counts from that first export. |
| 9 | Exported state persists or goes stale | accept | Each export is logged per client and period, and the draft says when its entries changed since then. |
| 10 | 80-col row truncation | accept | The project gives way first and the note last; times and duration never truncate. |
| 11 | Switch and undo | accept | A switch is one instant with no gap; undo lasts until the next change. |
| 12 | Label the persona specifics | accept | Under `## Assumed`. |

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
- **Picker** (`s`): type to filter, most recent first. A name that matches nothing offers `new: <client> / <project>` and creates both. After the pick, one optional note line; enter skips it.
- **Entry form** (`e` on a selected entry, `a` to add one on the selected day): client / project, start, end, note, and delete with a confirm. On the running entry, setting the end stops the timer at that time. That is the one-step fix for a forgotten timer.
- **Help** (`?`).

**Switch and undo.** `s` while running stops the current entry and starts the new one at the
same second, with no gap and no overlap. A stop or delete puts `u undo` on the status line,
which stays live until the next change to the entries.

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
- **Client**: the selected entry's client, or the running timer's if nothing is selected. `c` switches client in the picker.
- **Period**: that client's cycle (week or month, monthly by default). It opens on the last *complete* period unless that period has an export logged, and then on the current one. ←→ moves one period; esc goes back to the week.

Shows, ranked:
1. **Total**: `$6,544.00 for 54h 32m, 19 entries`.
2. **What changes or blocks the total**, one line each:
   - `naming workshop has no rate: 3h 15m left out`
   - `1h 24m running on site redesign not included until you stop it`
   - `changed since export on 22 oct: 2 entries; export again to update the PDF`
3. **Per-project lines**: project, hours, rate, amount.
4. **Invoice facts**, one row: `invoice 2026-014 · bill to northwind studio · due 5 nov (14 days)`.
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
terms and cycle. When keys are missing, the draft names the file and the missing keys
(`northwind: no rate, no bill_to · ~/.config/tallyho/clients.toml`). Export is blocked when
bill-to is missing or the total is $0. A project without a rate is left out and named, and
export is still allowed.

| | 80×24 | 120×40 |
|---|---|---|
| header, timer | rows 1–2 | rows 1–2 |
| total, warnings | rows 4–6 | rows 4–7 |
| project lines | rows 7–10 | rows 9–14 |
| facts, export | rows 11–13 | rows 16–18 |
| entries | rows 15–23, about 8, full width, scrolls | rows 20–39, about 20, full width |

States: Ready. Unconfigured client: names the file and keys, and blocks export if bill-to is missing. Partial: a project with no rate, or unreadable lines. Empty period. Exporting: a spinner for 1–2 s; esc cancels. Exported: persists, with the date of the last export. Changed since export. Export failed.

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
- The config is a TOML file, the cycle defaults to monthly, and the undo window is until the next change.
