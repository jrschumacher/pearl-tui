# nestkeeper: product read v3

| # | Objection (round 2) | Verdict | Why |
|---|---|---|---|
| 1 | Admin port + a dropping change in one apply | accept | One rule: they never apply together. The port stays pending until the next session; only one connection line ever shows. |
| 2 | Inactive dependent settings read as live | accept | Value plus "not used while address mode is dhcp"; still editable. |
| 3 | Account password is a second job | accept | a takes name, then password twice, in one step; no passwordless new accounts. |
| 4 | q on review; failed address change | accept | q behaves the same everywhere; failed row says "still on 192.168.1.37". |

Round 1 answers stand. Bar: a feature stays if a persona needs it **in most sessions** or
leaving it out locks someone out; a number stays only if someone acts on it.

## 1. Customers (interviewees: Rhea is Tomas, Dev is Kofi then Ines, Maren is Gail)

**Rhea, the household's admin.** 2-4 times a month, 5-15 min, SSH from her laptop when something
comes up; often stages, gets called away, returns days later. Today: the :8443 console, `sudo vim`.

**Dev, a new owner in month one.** Fixed address, family accounts, ports, backups: several
evenings a week, then monthly, 8-10 changes per apply. Today: the vendor wiki and pasted commands.

**Maren, who inherited the box.** Every month or two, told by text what to change; does that
one thing and leaves. Box in a cupboard, no screen: a lockout ends her use. Today: phones her brother.

## 2. Jobs, by how often they come up
1. **Find one setting and change it**, by own word, name or section (most sessions). **The main screen serves this.**
2. **Look up a current value** (most sessions; comes free with 1).
3. **Apply what's pending and leave nothing half-done** (every session that changes something).
4. **Undo or discard a staged change** (often: a typo, a second thought, someone else's leftovers).
5. **Add or remove an account** (Dev's week one; then a few times a year).
6. **Get back in after a change that drops the connection** (address, admin port; rare, costly).

## 3. The first slice: two screens plus overlays

Every screen: **header** (brand top left, "4 to apply", host right) and **footer** (keys, help last).

**Value kinds.** One per setting, fixed; a builder never guesses.

| Kind | List shows | Edit | Review shows |
|---|---|---|---|
| toggle | on / off | enter flips it | on → off |
| choice | the choice | ←→ in place, enter sets | dhcp → static |
| text, number | the value | inline field, checked on enter | 22 → 2222 |
| secret | set / not set; never the value | masked field, typed twice | password: changed |
| list | "3 entries" | detail pane: n new, x remove, one entry at a time | + 9.9.9.9 / − 1.1.1.1, one line each |

Keys: n adds (a stays "review" everywhere); x on a staged row reverts it, on an existing entry
or account stages its removal. **Accounts** are the users section's rows, one per account
("ada  password set"). n adds one in one step: name, then password twice; nothing is staged
until both are in. enter changes the password. Review: "+ ada", "− guest", "ada: password
changed"; "password: not set" only on accounts that already existed that way.

**Dependent settings** keep their value and say when it isn't used: "192.168.1.20/24 · not
used while address mode is dhcp". They stay editable, so Dev can set the address first.

### Settings (main)
**Job:** find a setting, see what it is now, change it; the change waits.
Shows, ranked:
1. The settings list: name, value, or old → new with a pending mark when staged.
2. Detail for the selected setting: what it does (1-3 lines); its effect ("restarts
   networking"); what's valid or the choices; for a staged one "x revert"; and **the
   running value where the configured one doesn't say it** ("dhcp, now 192.168.1.37").
3. Find line: the typed text and "3 match". It matches names, then descriptions, across all
   sections; each match row shows its section; the count covers both kinds of match.
4. Sections with a pending count each ("network 2").

Proportions:
- 120x40: header 1, blank 1; rail 14 cols | list ~60 | detail ~44, rows 3-38; footer 1.
- 80x24: header 1; sections as one row with pending counts; list full width, rows 4-17,
  scrolling past 14 (users can exceed it); detail as a 4-line strip, rows 19-22; footer 1.

States: loading; loaded; find typing / kept / no match; edit (per kind above); add account;
invalid (reason under the field, not staged); reverted ("u undo" stays until the next change);
can't read (all, "r retry"); one section can't read (others editable); **quit with pending**
(one footer line, q on either screen: "2 changes not applied; they wait on the nest · a review
· q quit anyway · esc stay"; on review "a review" reads "enter apply").

Not there: the pending batch (review's job), history, health or any other live data.

### Review & apply
**Job:** see every pending change, what applying will do to the house, then apply.
Shows, ranked:
1. The changes, grouped by section: name, old → new (or per kind), effect in daemon words
   ("restarts sshd"). Rows are selectable: x reverts one, with undo. Rows staged in another
   session also say "staged 12 Oct, tomas".
2. **The connection line**, at most one, when the batch will drop or move this session:
   - admin port: "This session stays open. Your next login: ssh -p 2222 rhea@nest.local."
   - address or interface: "This connection will drop. Reconnect: ssh rhea@192.168.1.20"
     (to dhcp: "the router assigns the address; try ssh rhea@nest.local").
   - both staged: the address line, plus "The admin port waits: it moves in its own apply,
     after you reconnect, so you can test it and put it back." Apply then names the count
     without it ("Apply 4 of 5 changes"); the port row reads "waits for next apply".
3. The summary in household words, worst effect first: "The network drops for about 10 s:
   streams and calls stop" or "Name lookups pause for about 1 s; streams keep playing".
4. Apply ("Apply 4 changes"), Back, and d discard all (confirm line: "Discard 4 changes? y / n").

**Rule:** the admin port never applies with a change that drops the connection, so p's
session always exists. Apply order: dropping changes go last, so the others report first.
Once started, apply runs on the nest; a dropped session doesn't stop it. After reconnecting,
the waiting port is an ordinary pending change ("1 to apply").

States: empty; ready; admin-port confirm (overlay, below; not shown when the port waits); applying ("Applying 2 of 4", effect
column becomes status: waits / restarting sshd / done / failed: reason); all applied; partial
(rows stay selectable; Retry the failed one, x drops it, Back leaves it pending); nothing applied
(nestd unreachable, all still pending, "Retry 4 changes"); **result after a dropped session**
(next launch, same result states titled "While you were disconnected: applied 3 of 4").

**Admin port.** The change moves sshd, opens the new port in the firewall and closes the old
one; its effect reads "restarts sshd; firewall opens 2222, closes 22". The confirm overlay:
"Move the admin port to 2222? This session stays open. After applying, test the new port from
another terminal; if it fails, one key puts 22 back from here." Default focus on Back. After
apply: "Before you close this session, try ssh -p 2222 rhea@nest.local in another terminal ·
p put 22 back" (p stages and applies the old port and firewall rule, no second confirm).
Failed: the row says "sshd still on 22". A failed address change says "still on
192.168.1.37; this session is fine", and the session isn't dropped.

Proportions:
- 120x40: one column ~100 wide (name 24 | old → new 34 | effect/status 30), rows from 3;
  connection line, summary, actions directly under the last row.
- 80x24: columns 22 | 30 | 22; rows scroll; connection line, summary and actions pinned at
  rows 18-22 so Apply is always on screen.

Not there: per-change descriptions, a "What happens" side pane, timings, history.

## 4. Cut list
- Consultant: health dashboard, metrics, trend alerts (monitoring; sessions are 5-15 min, begun
  to change something); widgets (nobody customises a 15-minute tool); agent layouts (AI the
  brief didn't ask for); ctrl+p (/ find jumps anywhere); wizards, more screens (brief: two).
- Buttons by background, not brackets (consultant): a visual call, passed to visual design.
- "Waiting to apply" block on settings, "What happens" pane, per-row descriptions on review (old build): duplicates.
- History pane; "debian 12", "11 settings", "of 58", durations, elapsed seconds (old build):
  nobody acts on them; "staged by" covers the one history case that matters.
- Free space and other live facts beyond the running-value rule: no setting needs one.
- Household effect on every row: the decision is per batch; one summary line serves it.
- Timed auto-revert of the port, alone or with an address change: test-then-p and the split rule are cheaper.
- Several servers: a household has one nest.

## 5. Departures from the brief
1. **Revert from review** (x per row, d all). Rhea sees "2223" at review and fixes it there;
   Maren meets changes she didn't make and needs to clear them.
2. **Find matches descriptions too, names first.** People search with their word ("vpn"), not ours.
3. **Next launch after a dropped apply opens on review's result**, not settings: the person
   who lost the connection needs to know what happened before anything else.
Kept: the two screens, staged changes, old → new, the admin-port confirm, every state of both.
