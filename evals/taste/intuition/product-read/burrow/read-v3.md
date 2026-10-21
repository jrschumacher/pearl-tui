# burrow: product read, v3 (for the build)

| # | Objection | Verdict | Why |
|---|---|---|---|
| 1 | Preview never marks read | accept | A previewed message is read after 2 seconds selected, or as soon as its preview is scrolled. |
| 2 | Archives push out the send | accept | A send keeps the first notice slot until it leaves, with its own unsend key. |
| 3 | r = all breaks habit; To row too small | accept | Convention: r sender, a everyone. r warns in one line when others are left off. To grows to fit. |
| 4 | Bulk mixes every list | accept | Bulk groups by list (List-Id, else sender). Assumption written down: server-filed folders also work. |
| 5 | Enter undefined in drafts, outbox | accept | Enter on drafts and outbox resumes in compose; r on a draft row resumes it; a late failure never interrupts. |
| 6 | Saves overwrite; inline images | accept | Never overwrite (renamed "comments (2).pdf", notice names it); inline images aren't saved. |
| 7 | Newcomer's 200 unread | accept | One offer on first load to mark mail from before today read. No standing mark-all key. |
| 8 | Compose blank at 120x40 | accept | The original returns beside the draft at 120x40, read-only; the editable quote stays. |
| 9 | Layout tables in HTML | accept | Only data tables keep their shape; layout tables flatten into paragraphs. |
| 10 | Status row overflows at 80 | accept | A stated order of what shortens; sync state and the outbox count never go. |

## Customers

1. **Wren, novelist.** Daily, 3-5 hours at 120x40; reads 15-25, most in the preview; replies
   to 1-3, re-reading the letter as she writes. Today: Gmail.
2. **Dr. Osei, historian.** Daily, all workday, 120x40+, often on a train. 20 letters from
   co-authors (3-4 per thread), 40 from four lists, read one list at a sitting; replies
   inline, archives hard, searches weekly. Today: Thunderbird.
3. **Priya, infrastructure engineer.** Daily, 10-20 glances, an 80x24 tmux pane. Reply-alls
   to her team ~10 a day, replies to one person as often; trims quotes; send, archive,
   archive, quit. Today: neomutt.

## Jobs, by how often a customer does each

1. Pick what to open (every glance, 10-30 a day). 2. Read it to the end, wide lines and HTML
included (10-50). 3. Put it away (10-30). 4. Reply, to one or all (1-10). 5. Know the list is
current and sends left (every glance). 6. Read one list at a sitting (Osei daily, Wren weekly).
7. Write new (a few a week); get an archived message or attachment back (weekly).
**The main screen serves jobs 1 and 2 as one: read what's waiting.**

## The first slice: two screens and one overlay

### Inbox
Job: read what's waiting. Two modes: **choosing** and **reading**.

**What the list holds.** One server folder at a time, default the inbox, whole, newest first,
loaded as you scroll. Two views of it, from headers alone, one key to flip: **letters**
(default, grouped today / yesterday / earlier) and **bulk** (List-Unsubscribe, Precedence
bulk or list, or Auto-Submitted), grouped by list (List-Id, else sender), the list with the
newest message first. One key moves the selected sender or list to the other view for good.
Assumed: lists already filed into server folders are read from the folders overlay;
burrow files nothing itself.

**Folders overlay** (one key): server folders with "12 unread" beside each, plus **outbox**
while anything waits or failed; type to filter. In drafts or outbox, enter resumes in compose.

Shows, ranked:
1. The body of the selected message (120x40) / the list (80x24, choosing).
2. The list: per row a mark (draft, else unread, else replied), sender, subject, received
   time ("09:41" today, "mon" this week, "12 oct" this year, "2024" before).
3. Message header: subject, sender name and address, everyone else it went to, received
   date and time, attachment names (inline images not listed).
4. "N more lines below" while there are any; "end of message" at the end.
5. Wide lines: prose wraps; data tables and code keep their shape, clip with a mark and
   scroll sideways; URLs wrap whole. HTML-only mail shows as text ("shown as text" in the
   header): a table with a header row and no table inside it is data and keeps its shape;
   every other table flattens into paragraphs; images drop, alt text stays.

The reading loop:
- Read when opened (enter, click, next-message key), or, in the 120x40 preview, after 2
  seconds selected or on the first scroll. Arrowing past marks nothing. One key marks
  unread again.
- Next / previous message: one key each from the reader. Space pages and stops at the end.
- Archive selects the row below (older) and returns to choosing at 80x24. In the archive
  folder the same key moves a message back to the inbox.
- One key saves the attachments to ~/Downloads, never overwriting: a taken name becomes
  "comments (2).pdf", and the notice names each file written.
- First load, once: "142 letters from before today are unread. mark them read?" (y / n).

Proportions:
- **120x40**: list ~44 cols, two-line rows, ~15-17 messages with group headers; reader
  ~74 cols, body at a 72-col measure. 38 rows of content, 2 of chrome.
- **80x24**: one pane at a time. Row: mark 2, sender 16, subject ~50, time 6, gaps 6;
  about 19 rows. Reading: full width, body at 72 cols. 22 rows of content, 2 of chrome.

States: no account (with the config file's path), first load (newest 500 headers, then
the offer), choosing, reading, scrolled, empty folder, empty letters ("nothing from people;
38 in bulk"), syncing, offline with or without a local copy ("offline since 09:12, retrying
every minute"), **sign-in refused** ("mail.example refused the sign-in at 09:12; stopped
retrying", a retry key; never shown as offline), archived, sending, sent, queued, not sent.

### Compose
Job: answer a message, or write a new one, and send it with 5 seconds to take it back.

Shows, ranked:
1. The body. On reply the original is in it as quoted text, cursor above it, to trim or
   answer inline. New message: cursor in To.
2. To, then subject. **r** fills To with the sender (Reply-To if set); **a** with the sender
   and everyone else on it, minus you; the mutt, Gmail and Thunderbird convention. After r on
   a message with others on it, one line under To: "3 others on the original aren't
   included · a adds them". To grows to as many rows as its names need (at most 4, then it
   scrolls and reads "to 9 people"). Completion from people in your mail (at most 5 matches).
   A bad address is named under the field and blocks send.
3. At 120x40, the original beside the draft, read-only, scrolling on its own.

Proportions: **120x40** draft ~74 cols (body at a 72-col measure), original ~44 cols.
**80x24** draft full width, fields 2-5 rows; one key swaps to the original and back.

States: new, reply, reply all, others left off, address invalid, offline (will queue),
resumed draft or outbox item, undone send (draft and cursor as they were). Esc keeps the
draft in drafts and marks the replied row; r or a there resumes it. c always starts new.
Discard has its own key and a 5-second undo.

### On every screen
- **Status row**, left to right: where you are ("inbox · letters"), "3 unread" (letters),
  sync state, and while non-zero "2 waiting to send" / "1 not sent" (both open the
  outbox). When it doesn't fit, in order: "retrying every minute" drops, the view name
  drops, "offline since 09:12" becomes "offline 09:12". Sync and outbox counts never drop.
- **Key row**: the 4-6 keys for this mode and help. Notices take it for 5 seconds, two
  slots. A send holds the first slot until it leaves ("sending to 6 people in 3 s · U
  unsend"), however many archives follow; a second send in the window shares it ("sending
  2 · U unsends the newer"). The second slot shows the newest other notice; u undoes the
  newest undoable action.
- **Sends**: offline, they queue and leave on the next sync while burrow is open. A refused
  send (bad address, too large) never interrupts reading: it goes to the outbox as "not
  sent: <server's reason>", with a notice and the status count. Quit in a countdown sends at
  once, then quits; quit with mail queued asks once ("1 message hasn't left; it sends next
  time burrow is open. quit anyway?").
- ctrl+g help everywhere (`?` too on the inbox). Mouse: click opens or focuses, wheel scrolls.

### Deliberately not there
No stats, projections, categories beyond letters / bulk, reading-time estimates or splash.
New mail joins the list without moving the selection.

## Cut list

- Agent categorization (consultant): headers split letters from bulk with no agent; no one asked for more than two classes.
- Trends pane, projections, minutes to inbox zero, sparkline (consultant): no one acts on a forecast; they move while you read; 30 cols.
- List in order processed (consultant): newest-first is how all three find new mail.
- Notes on a message (consultant): not in the brief; Wren and Osei annotate in their editors.
- Larger logo (consultant): brand visuals come later; it takes list rows at 80x24. ctrl+? help: terminals don't send it.
- Per-row "3 min", reading-time summary, contact sidebar, split view at 80x24 (old build): read as a time; completion does the sidebar's job; reader under 40 cols.
- Search: slice two. Until then old mail is browsed in the archive folder or found in webmail; Osei pays that weekly, and we accept it.
- Standing mark-all key; filing lists inside burrow: the first-load offer and bulk's grouping cover them.
- Opening attachments, HTML in a browser: saving covers the weekly need. "New since last look": a second count.
- Cc and bcc fields, forward, threads view, flags, multiple accounts, account setup screen, notifications: not in the brief, no persona needs them weekly in slice one.

## Departures from the brief

1. **At 80x24 reading replaces the list**: a split reader is under 40 cols.
2. **Archive, with a folders overlay**: all three put away 10-30 a day; the overlay brings
   archive, lists, drafts, sent and outbox back without a third screen.
3. **Letters and bulk**: Wren loses her editor among newsletters; Osei reads a list at a sitting.
4. **Esc keeps drafts; offline sends queue; the countdown runs on the inbox**: nothing is lost.
5. **Two reply keys, quote editable in the body**: the brief's three fields stay. r follows
   convention because Priya's one-person reply reaching her team can't be taken back after
   5 seconds; Osei's opposite mistake is caught by the "others aren't included" line.
6. **80x24 is a daily layout, not a fallback**: Priya lives there all day.
