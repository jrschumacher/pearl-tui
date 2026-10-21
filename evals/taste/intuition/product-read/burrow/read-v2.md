# burrow: product read, v2

| # | Objection | Verdict | Why |
|---|---|---|---|
| 1 | Which mail, letters vs bulk | accept | Inbox by default; a letters / bulk split from headers the server already sends; one folder overlay for the rest. No agent. |
| 2 | Archive is one-way | accept | Archive is the server's archive folder, reachable from the same folder overlay; moved back with the same key. Search stays slice two, cost written down. |
| 3 | Reading loop moves undefined | accept | Read means opened, not previewed; mark-unread restored; next-message key; archive moves down a row; end of message stops. |
| 4 | Send and archive notices collide | accept | One notice line holds both, newest first; undo undoes the newest; quit sends first; queued and failed sends get a status-row count and a home. |
| 5 | Reply recipients, quoting | accept | r replies to everyone on the message, R to the sender only, all names shown in To; the quote is editable text in the body. Cuts compose's side pane. |
| 6 | No account, failed sign-in | accept | Two states in the inbox, in words, distinct from offline. Setting an account up stays outside the two screens. |
| 7 | HTML-only mail | accept | Shown as text: headings, paragraphs, lists, tables, links whole; images dropped. |
| 8 | Attachments dead end | partly | One key saves them to ~/Downloads and says so; opening stays out. |
| 9 | 80-col row widths | accept | Widths and the older-date format stated below; subject gets ~50 cols. |
| 10 | New-message drafts | accept | c always starts a new letter; drafts live in the server's drafts folder, any number. |
| 11 | List extent, shouting count | partly | Whole folder, loaded as you scroll; the count counts unread letters only. A "since last look" count is rejected: two counts to explain. |
| 12 | 80x24 raised to daily | accept | Added as departure 6. |

## Customers

1. **Wren, novelist.** Daily, 3-5 hours at 120x40 beside her draft. Reads 15-25 (letters among
   newsletters), replies to 1-3, archives. Today: a Gmail tab, newsletters in their own tab.
2. **Dr. Osei, historian.** Daily, all workday, 120x40+, often on a train. 20 letters from
   co-authors (3-4 per thread), 40 from lists filed into folders; replies inline, archives
   hard, searches weekly. Today: Thunderbird.
3. **Priya, infrastructure engineer.** Daily, 10-20 glances, an 80x24 tmux pane. Reply-alls
   to her team list ~10 a day, trimmed inline; archives, quits. Today: neomutt.

## Jobs, by how often a customer does each

1. Pick what to open (every glance, 10-30 a day). 2. Read it to the end, wide lines and HTML
newsletters included (10-50 a day). 3. Put it away (10-30 a day). 4. Reply, to one or to all
(1-10 a day). 5. Know the list is current and sends have left (every glance; acts weekly).
6. Read lists or bulk in their own sitting (Osei daily, Wren weekly). 7. Write new (a few a
week); get an archived message or an attachment back (weekly).

**The main screen serves job 1 and 2 as one job: read what's waiting.**

## The first slice: two screens and one overlay

### Inbox
Job: read what's waiting. Two modes: **choosing** and **reading**.

**What the list holds.** One server folder at a time, default the inbox, whole, newest first,
loaded as you scroll. Within it, two views from headers alone: **letters** (default) and
**bulk** (anything with List-Unsubscribe, Precedence: bulk or list, or Auto-Submitted). One
key flips views. One key moves the selected sender or list to the other view for good
(Priya's team list into letters). The **folders overlay** lists the server's folders (inbox,
archive, drafts, sent, Osei's list folders) with "12 unread" beside each, plus **outbox**
while anything waits to send; type to filter, enter to show that folder.

Shows, ranked:
1. The body of the open message (120x40) / the list (80x24, choosing).
2. The list: per row a mark (unread, replied or draft; one at a time), sender, subject,
   received time ("09:41" today, "mon" this week, "12 oct" this year, "2024" before),
   under today / yesterday / earlier.
3. The message header: subject, sender name and address, everyone else it went to, received
   date and time, attachment names.
4. "N more lines below" while there are any; "end of message" at the end.
5. Wide lines: prose wraps; tables and code keep their shape, clip with a mark, scroll
   sideways. URLs wrap whole. HTML-only mail shows as text, with "shown as text" in the
   header; images are dropped, their alt text kept.

The reading loop:
- Read means opened (enter, click, next-message key), never passed over: at 120x40 the
  reader previews the selected row, which is not reading. One key marks it unread again.
- Next / previous message: one key each from the reader. Space pages and stops at the end.
- Archive moves the selection one row down (older) and returns to choosing at 80x24.
  In the archive folder the same key moves a message back to the inbox.
- One key saves the message's attachments to ~/Downloads; the notice names the path.

Proportions:
- **120x40**: list ~44 cols, two-line rows, ~15-17 messages with group headers; reader
  ~74 cols, body at a 72-col measure. 38 rows of content, 2 of chrome.
- **80x24**: one pane at a time. Row: mark 2, sender 16, subject ~50, time 6, gaps 6;
  about 19 rows. Reading: full width, body at 72 cols. 22 rows of content, 2 of chrome.

States: no account ("no account set up", with the config file's path), first load (newest
500 headers), choosing, reading, scrolled down / sideways, empty folder, empty letters
("nothing from people; 38 in bulk"), syncing, offline with or without a local copy
("offline since 09:12, retrying every minute"), **sign-in refused** ("mail.example refused
the sign-in at 09:12; stopped retrying", a retry key; never shown as offline), archived,
sending, sent, queued, send failed.

### Compose
Job: answer a message, or write a new one, and send it with 5 seconds to take it back.

Shows, ranked:
1. The body. On reply the original is in it as quoted text, cursor above it, to trim or
   answer inline. New message: cursor in To.
2. To and subject, one row each. r fills To with the sender and everyone else on the
   message, minus you, honouring Reply-To; R fills the sender only. Every name is shown,
   each removable. To completes from people in your mail (at most 5 matches). A bad
   address is named under the field in plain words and blocks send.

Proportions: fields 2 rows, body the rest at a 72-col measure; at 120x40 ~46 cols stay empty.

States: new, reply, reply to sender only, address invalid, offline (send will queue),
undone send (back in compose, draft and cursor as they were), failed send (draft restored,
"not sent: <reason>"). Esc keeps the draft in the drafts folder; a reply draft also marks
its row. c always starts a new letter. Discard has its own key and the 5-second undo.

### On every screen
- Status row: where you are ("inbox · letters", "writing to mara okafor"), unread letters
  as "3 unread", sync state in words, and while non-zero "2 waiting to send" or "1 not
  sent" (both open the outbox).
- Key row: the 4-6 keys for this mode and help. Notices take it for 5 seconds: newest first,
  at most two ("archived 'notes on…' · sending to mara in 3 s"); u undoes the newest,
  again undoes the other while its 5 seconds last.
- Quit in a countdown sends at once, then quits; quit with mail queued asks once ("1 message
  hasn't left; it sends next time burrow is open. quit anyway?").
- ctrl+g help everywhere (`?` too on the inbox). Mouse: click opens or focuses, wheel scrolls.

### Deliberately not there
No stats, projections, categories beyond letters / bulk, reading-time estimates, contact
list or splash. New mail joins the list without moving the selection.

## Cut list

- Agent categorization, learning from you (consultant): headers split letters from bulk with no agent; no one asked for more classes than two.
- Trends pane (1/4 width), volume projection, minutes to inbox zero, braille sparkline (consultant): no one acts on a forecast or rate; they move while you read; they cost the reader 30 cols.
- List in order processed (consultant): newest-first is how all three find new mail.
- Notes on a message (consultant): not in the brief; Wren and Osei annotate in their own editors.
- Larger logo (consultant): brand visuals come later; it would take list rows at 80x24.
- ctrl+? for help (consultant): most terminals don't send it; ctrl+g everywhere.
- Per-row "3 min", "about 4 min of reading" (old build): read as a time received; no one decides by it.
- Contact sidebar; original beside the draft (old build, v1): completion under To, and the editable quote, do both.
- Split view and two-line rows at 80x24 (old build): reader under 40 cols; ~10 rows per glance.
- Search: slice two. Until then old mail is reached by browsing the archive folder newest-first or the server's own webmail search; Osei pays that weekly, and we accept it.
- Opening attachments, HTML in a browser: saving covers the weekly need.
- "New since last look" count: a second count to explain beside unread.
- Mark all read: once per person (a newcomer's 1,400); the bulk view holds most of them.
- Cc and bcc fields, forward, threads view, flags, multiple accounts, account setup screen, notifications: not in the brief, no persona needs them weekly in slice one.

## Departures from the brief

1. **At 80x24 reading replaces the list**: one screen, two modes; a split reader is under 40 cols.
2. **Archive, with a folders overlay**: all three put away 10-30 a day; the overlay brings
   archive, lists, drafts, sent and outbox back without a third screen.
3. **Letters and bulk**: Wren scrolls past her editor between newsletters; Osei's lists need
   a sitting of their own.
4. **Esc keeps a draft; offline sends queue; the countdown runs on the inbox**: no message
   is lost to a key, a train or a quit.
5. **Reply goes to everyone, quote editable in the body**: the brief's three fields stay; To
   starts full, because answering one of four co-authors is the costliest silent mistake.
6. **80x24 is a daily layout, not a fallback**: Priya lives there all day.
