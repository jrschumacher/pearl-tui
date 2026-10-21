# tallyho: product read v1

Units used everywhere: durations as `7h 30m` (running timer `1h 24m 07s`), clock times
24h `09:02–11:12`, money `$6,544.00`, rates `$120/h`, counts with their noun (`19 entries`).

## 1. Customers

- **Dev, freelance backend developer, 3 clients.** Terminal open all day in a tmux pane;
  switches project 8–15 times a day (calls, reviews, support pings). Retainer with one
  client caps at 20 hours a week, so she checks that client's week hours daily. Invoices
  monthly. Today: Toggl in a browser tab she forgets is there, plus a spreadsheet to
  turn a CSV export into an invoice.
- **Sam, freelance product designer, 4–5 clients.** Lives in Figma; the terminal is a
  side window. Starts and stops 5–8 times a day and forgets to start or stop at least
  twice a week, so fixes entries every Friday. Invoices one agency weekly, the rest
  monthly. Today: a Harvest free tier he's outgrowing, and a notes file for what he did.
- **Ola, contract front-end developer, one main client plus one small one.** The main
  client audits hours: every invoice line must say what the time was. Tracks 4–6 entries
  a day with a note on each. Invoices weekly. Today: `timew` (Timewarrior) plus a
  hand-made invoice in a doc template, re-typing notes.

## 2. Jobs (by how often)

1. **Switch what I'm timing**: start, stop, or move to another project (5–15 times a day, all three).
2. **Glance: is it running, on what, since when** (dozens of times a day, all three).
3. **Check today's and this week's hours** (daily: Dev against her cap, everyone at end of day).
4. **Fix an entry**: wrong project, forgot to stop, forgot to start, add a note (2–3 times a week: Sam, Ola).
5. **Draft an invoice for one client and period, check it, export the PDF** (weekly: Sam, Ola; monthly: Dev).

**The main screen (the week) serves job 1**, with job 2 always on top of it. Jobs 3 and 4
live on the same screen; job 5 is the second screen.

## 3. The first slice: two screens

### Always visible, both screens
- **Timer line** (row 2, full width): running: elapsed `1h 24m 07s`, client / project,
  note, `since 13:12`. Stopped: `not tracking since 14:36` and what space would resume.
- **Footer** (last row): the keys for this screen, help key at the right.
- **Header** (row 1): `tallyho` and where you are (`19–25 oct 2026`, or `invoice · northwind studio · oct 2026`).

### Week (main)
Job: switch the timer, see today, fix an entry. Shows, ranked:
1. The timer line (above). The eye lands here; space starts/stops, `s` picks a project.
2. The selected day's entries, newest at the bottom: `09:02–11:12  northwind / site redesign  hero animation  2h 10m`.
   The running entry is the last row and ticks. Opens on today.
3. The day strip: the 7 days with each day's hours, the selected day marked, then the week total (`26h 31m this week`).
4. This week's hours by client, hours only, largest first (`northwind 17h 51m`). Dev reads her cap off this.
5. One status line when it applies: `2 lines unreadable (212, 340), their hours not counted`, or `stopped · u undo`.

Overlays (not screens): **project picker** (type to filter, recent first, ~6 rows), **entry
form** (client / project, start, end, note; same form adds a missed entry; delete with a
confirm), **help**.

| | 80×24 | 120×40 |
|---|---|---|
| header + timer | rows 1–2, blank row 3 | rows 1–2, blank row 3 |
| days | one row across (row 4): `mon 7h 30m` ×7, ~76 cols | left column, 26 cols: 7 day rows + week total |
| entries | full width, rows 6–20 (~14 entries, scrolls) | right, ~90 cols, rows 5–36 |
| client hours | one row (row 22), wraps to `+2 more` | left column under the days, one row per client |
| status / footer | row 23 / row 24 | row 38 / row 40 |

States: running; stopped (just now, with undo); idle (not tracking, nothing to undo);
empty week (one line: space or `s` to start, the day strip still shows 0h); partial
(unreadable lines, counted and named); error (can't read the entries file: path, the
reason, `r` retry; timer line says the timer state is unknown).

Deliberately not here: money of any kind, months, weeks other than via ←→, per-project
breakdowns, bars or charts, an entry detail panel (the row already holds every field).

### Invoice draft (one client, one period)
Job: confirm what I'm billing this client and write the PDF. Shows, ranked:
1. **Total**: `$6,544.00 for 54h 32m, 19 entries`, the line the eye lands on.
2. **Anything that changes the total**, one line each, above the table when present:
   `naming workshop has no rate: 3h 15m left out`; `1h 24m running on site redesign is not included until you stop it`.
3. **Per-project lines**: project, hours, rate, amount. Usually 1–4 rows.
4. **Invoice facts** in one row: `invoice 2026-014 · bill to northwind studio · due in 14 days`.
5. **Export**: the file path and the action; then its result (`written 14:36 · ~/invoices/northwind-2026-10.pdf`, enter exports again) or its error (what failed, what to do, try again).
6. **Entries in the period**: date, project, hours, note (the note is what Ola's client audits), scrolls.

Period: the client's billing cycle, week or month (set per client in config); ←→ moves
one period. `c` switches client via the same picker as projects. esc returns to the week.

| | 80×24 | 120×40 |
|---|---|---|
| header + timer | rows 1–2 | rows 1–2 |
| total + warnings | rows 4–6 | rows 4–6 |
| project lines | rows 7–10 | rows 8–13 |
| invoice facts + export | rows 11–13 | rows 15–17 |
| entries | rows 15–23, ~8 rows, full width, scrolls | rows 19–39, ~20 rows, full width (notes need the width) |

States: ready; empty period (no hours for this client in this period, ←→ to move);
partial (no rate on a project, unreadable lines); exporting (spinner, 1–2 s, esc cancels);
exported; export failed.

Deliberately not here: editing rates, addresses or terms (config, set once per client);
sending; marking paid; tax lines; per-entry amounts.

## 4. Cut list

- Dashboard as the main screen (consultant): the main job is switching the timer, not reviewing.
- Tasks / to-dos per client (consultant): a task manager is a second product; they already have one.
- Earnings, rates and goals "motivation" view (consultant): no one acts on it while timing; money lives on the invoice.
- AI agents tracking their own hours, live agent feed (consultant): not in the brief; no persona runs agents.
- Several concurrent timers, recently stopped timers always visible (consultant): one person, one timer; the picker shows recent.
- Header notification area, 2–3 pane layout (consultant): nothing in the slice needs to notify.
- Slide-out task panel on the left (consultant): no tasks to show.
- Journal / audit log as the second screen (consultant): the brief's second screen is the invoice; notes on entries cover the audit.
- "Pulled away" / focus highlighting (consultant): needs inference about intent; nobody asked.
- Pie chart of time across clients (consultant): the client-hours list says the same in one row.
- Brand mark bottom right (consultant): visual decision, not this read's.
- Money per client this week (old build): nobody acts on it until invoicing.
- October-so-far panel with hours and money (old build): monthly money belongs on the invoice.
- By-week panel for the month (old build): looked at monthly at most, and the invoice covers it.
- Projects-this-week panel with bars (old build): duplicates entries and client hours.
- Entry detail panel: client, rate, "bills $168 so far", "october 41:52 logged" (old build): repeats the row; the rest is money.
- Tally bars next to each day (old build): the day's hours number already says it.
- Week number `week 43` (old build): the date range is what people recognise.
- Entries count column per project on the invoice (old build): no one acts on it; the total keeps the count.
- `1 page` and elapsed seconds on export (old build): no decision rides on them.
- Month-only invoice periods (old build): Sam and Ola invoice weekly.

## 5. Departures from the brief

- **Week screen leads with the timer and one day, not a grid of the whole week.** Jobs 1–2
  happen dozens of times a day; the other six days are a row of totals you step into with ←→.
- **Adds fixing and adding entries (an overlay on the week).** The brief names only
  start/stop, but Sam and Ola fix entries weekly; a wrong entry means a wrong invoice.
- **Invoice period follows the client's cycle, week or month.** The brief says "a period"
  and that customers invoice weekly or monthly; month-only fails Sam and Ola.
- **Invoice shows each entry's note.** Ola's client audits every line; without notes
  she re-types them from memory.
