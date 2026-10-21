# nestkeeper: product read v1

Used a few times a month, not weekly, so the bar is per session: a feature stays if a named
persona needs it **in most sessions**; a number stays only if someone acts on it.

## 1. Customers

**Rhea, the household's admin.** Runs the nest for a family of four. 2-4 times a month,
5-15 min, over SSH from her laptop in the evening, right after something came up ("the
backup ran during the film"). Today: the browser console on :8443, and `sudo vim` in /etc
for the rest, then forgets which file she touched.

**Dev, a new owner in month one.** Setting up address, ports, pool, backup target; several
times a week for a month, then like Rhea, with many changes per session across sections.
Today: the vendor wiki in one tab, a shell in another, pasting commands.

**Maren, who inherited the box.** Her brother set it up and moved away. Every month or two,
when told to ("turn on all updates"), his text on her phone; doesn't know the setting
names. Today: she phones him, or he SSHes in for her.

## 2. Jobs, by how often they come up

1. **Find one setting and change it** (most sessions; Maren by a word she was told, Rhea by
   name, Dev by browsing a section).
2. **Look up a current value** (most sessions: "which port is the console on?").
3. **Check what's pending and apply it together** (every session that changes something).
4. **Undo a change before applying it** (often: a typo, a second thought at review).
5. **Recover when one change fails** (a few times a year).
6. **Move the admin port without locking yourself out** (once ever, but costly if wrong).

**The main screen serves job 1** (find and change); job 2 comes free with it.

## 3. The first slice: two screens

Every screen: **header** (one line: brand top left, "4 to apply", host "nest.local" right)
and **footer** (keys for this state, help last). The pending count links the two screens;
the host tells Maren she's on the right box and the lockout command names it.

### Settings (main)
**Job:** find a setting, see what it is now, change it; the change waits.
Shows, ranked:
1. The settings list: name, current value, or "old → new" with a pending mark when staged.
2. The selected setting's detail: what it does (1-3 lines), what applying it does
   ("restarts sshd"), what's valid ("1-65535; not 80, 443, 8443") or the choices, and for a
   staged one "x revert to 22".
3. Find line (when open): the typed text and "3 match".
4. Sections with a pending count each ("network 2"): where your staged changes are.

Proportions:
- 120x40: header 1, blank 1; rail 14 cols | list ~60 | detail ~44, rows 3-38; footer 1.
- 80x24: header 1; sections as one row with pending counts (~50 cols); list full width,
  rows 4-17 (14 rows, a whole section); detail as a 4-line strip, rows 19-22; footer 1.
  A side pane at 80 would leave the value column ~24 cols, too narrow for
  "security only → all updates".

States: loading; loaded; find typing / find kept / no match; edit text; edit choice; invalid
(reason under the field, value not staged); reverted (with "u undo" in the footer for a few
seconds); can't read (all sections, "r retry"); one section can't read (the others still
editable).

Not there: the pending batch (review's job), history, any live server data.

### Review & apply
**Job:** see every pending change, old and new, and what applying will do; then apply.
Shows, ranked:
1. The changes, grouped by section: name, old → new, effect ("restarts sshd", "takes effect
   at once"). The rows are selectable: x reverts one (departure 1).
2. The admin-port line when it's in the batch: "your next login: ssh -p 2222 rhea@nest.local;
   this session stays open".
3. One summary line: "Restarts resolved and sshd; nothing else stops. Changes wait on the nest, so a
   dropped connection loses none."
4. Apply (named with the count: "Apply 4 changes") and Back.

During and after apply the same rows stay put; the effect column becomes status
(waits / restarting sshd / done / failed: reason). Title becomes "Applying 2 of 4",
"Applied 4 changes", "Applied 3 of 4; dns server failed" or "Nothing applied".

Proportions:
- 120x40: one column ~100 wide (name 24 | old → new 34 | effect/status 30), rows from 3;
  summary, lockout line and actions directly under the last row, not pinned to the bottom.
  The right ~20 cols stay empty.
- 80x24: columns 22 | 30 | 22; rows scroll past 8 changes (rare); summary, lockout line and
  actions pinned at rows 19-22 so Apply is always on screen.

States: empty ("Nothing to apply"); ready; admin-port confirm (overlay: the new ssh
command, "only the console on the nest can set it back", default focus on Back); applying;
all applied (with the next-login command if the port moved); partial (failed row's reason,
"Retry dns server" / Back, which leaves it pending as the old build did); nothing applied (nestd unreachable, all still pending,
"Retry 4 changes").

Not there: per-change descriptions, a "What happens" side pane, timings.

## 4. Cut list
- Health dashboard as main screen; live and historical storage, memory, temperature,
  events; trend alerts (consultant): monitoring, not settings; no persona opens this to watch the box.
- Swappable widgets / custom layouts (consultant): no persona customises a tool used 15 min a month.
- Agent-generated layouts and reports (consultant): AI feature the brief didn't ask for.
- ctrl+p command palette (consultant): / find already jumps to any of the ~60 settings; a
  second jump is a second thing to learn.
- Wizards, state tables, "many second screens" (consultant): the brief asks for two screens.
- Buttons styled by background, not brackets (consultant): a visual call; passed to visual design, not cut.
- "Waiting to apply" block on settings (old build): duplicates review; header count and
  rail counts say where changes are; no room at 80x24.
- "What happens" pane on review (old build): restates each row's effect column.
- Per-row descriptions on review (old build): the name identifies it; doubles the rows.
- History pane, "12 Oct rhea set to..." (old build): no persona acts on it in most sessions.
- "debian 12", "11 settings" per section, "of 58" on find, per-change and total durations
  (old build): numbers nobody acts on.
- Elapsed seconds while applying (old build): "Applying 2 of 4" and the live row show it's moving.
- The reassurance paragraph on every apply state (old build): kept once, on review.
- Auto-revert of the admin port unless confirmed (mine): a once-ever change; the confirm and
  the shown ssh command cover it. Revisit if lockouts get reported.
- Switching between several servers: a household has one nest.

## 5. Departures from the brief (and the old build)
1. **Revert from review.** The brief's review only applies. Rhea spots the wrong value
   at review; making her go back, find it and revert is three steps for one. x on a
   row reverts it, with undo.
2. **Find matches descriptions too, names first.** The brief says "by name". Maren searches
   with the word she was told ("vpn", "password"), not nestkeeper's name for it
   ("remote access"); the old build's "No setting matches vpn" is that miss.

Kept from the brief and old build: the two screens, staged changes, old → new, the
admin-port confirm, every state of both. (The rail becoming one row at 80x24 is layout, not scope.)
