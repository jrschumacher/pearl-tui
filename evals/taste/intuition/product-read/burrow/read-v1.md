# burrow: product read, v1

## Customers

1. **Wren, novelist.** Daily, 3-5 hours open, full-screen tab at 120x40 beside her draft. Long
   letters from editor and friends, newsletters; reads 15-25 a day, replies to 1-3, slowly.
   Today: a Gmail browser tab she keeps closing because it pulls her out of the draft.
2. **Dr. Osei, historian.** Daily, all workday, 120x40+, often on a train with patchy wifi.
   Mailing lists, threads with co-authors carrying tables; reads 30-50, replies to 3-5.
   Today: Thunderbird; tried aerc and dropped it because wide tables were unreadable.
3. **Priya, backend engineer.** Daily, 10-20 glances between builds, in an 80x24 tmux pane.
   Short messages; reads 10-20, replies to 1-2 in a line, writes new ones a few times a week.
   Today: neomutt, which she finds noisy, or the Gmail tab.

## Jobs, by how often a customer does each

1. See what has arrived and pick one to open (every glance: 10-30 times a day, all three).
2. Read a message to the end, long ones and wide lines included (10-50 a day).
3. Put a finished message away (10-30 a day; read-but-kept means "come back to this").
4. Reply (1-5 a day).
5. Know whether the list is current or burrow is offline (looks every glance; acts on it
   weekly, mostly Osei).
6. Write a new message (a few a week).
7. Take back a send (about weekly).

**The main screen serves job 1 and 2 as one job: read what's waiting.** Everything on the
inbox is ranked by whether it helps pick the next message and read it.

## The first slice: two screens

### Inbox
Job: read what's waiting. Two modes on one screen: **choosing** and **reading**.

Shows, ranked:
1. The reading pane's body text (at 120x40) / the message list (at 80x24, choosing).
2. The list: per message sender, subject, received time (clock time today, weekday this
   week, date before), unread mark, replied mark, draft mark. Grouped under today /
   yesterday / earlier.
3. The open message's header: subject, sender name and address, received date and time,
   attachment names if any (listed, not opened).
4. Position in a long message: "N more lines below" while there are any.
5. Wide lines: prose wraps to the pane; tables and code keep their shape and clip at the
   edge with a mark, scrolled sideways with a key or shift+wheel. URLs wrap, so they stay
   whole for copy and click.

Proportions:
- **120x40**: list ~44 cols, two-line rows (sender + time / subject), ~15-17 messages with
  group headers; reader the other ~74 cols, body at a 72-col measure. 38 rows + 2 chrome.
- **80x24**: one pane at a time. Choosing: full-width list, one-line rows (sender 18 cols,
  subject flexes, time 5 cols), about 19 messages. Reading: the message full width, body
  at 72 cols, 1-2 header rows then body. Esc or the list key returns to the same row.
  22 rows of content, 2 rows of chrome.

States: first load (no local copy yet), choosing, reading, scrolled down, scrolled sideways,
empty, syncing (list stays usable), offline with a local copy ("offline since 09:12,
retrying every minute"), offline with no copy (retry, write), archived + undo, sending
countdown + undo, sent, queued offline ("sends when back online"), send failed.

### Compose
Job: answer a message, or write a new one, and send it with 5 seconds to take it back.

Shows, ranked:
1. The body, cursor in it on reply; cursor in To on a new message.
2. To and subject, one row each. To accepts several addresses and completes from people
   already in your mail (at most 5 matches, under the field). A bad address is named under
   the field in plain words and blocks send.
3. On reply, the message being answered, for reference (sent quoted below the reply).

Proportions:
- **120x40**: draft ~78 cols (fields, then body), original ~40 cols to the right, scrolls alone.
- **80x24**: draft full width (fields 2 rows, body ~19); the original is one key away, full
  width, and the same key returns to the cursor.

States: new, reply, address invalid, offline (send will queue), draft kept on esc (one per
message, resumed by r or c, marked on the list row), discard (own key, 5-second undo).
Send hands back to the inbox, where the countdown runs.

### On every screen
- One status row: the app name, where you are ("inbox", "writing to mara okafor"), the
  unread count as "3 unread", and the sync state in words (synced / syncing / offline
  since 09:12).
- One key row with the 4-6 keys for this mode and help. A notice (archived, sending in 5 s,
  sent) takes this row for 5 seconds with its undo key, then the keys return.
- One help key that works on both screens: ctrl+g, because `?` is text in compose (`?`
  also opens help on the inbox).
- Mouse: click a row to open it, click a field to focus it, wheel scrolls the pane under
  the pointer.

### Deliberately not there
No stats, counts of anything but unread, reading-time estimates, projections, categories,
folder tree, contact list, toolbar or splash. Nothing moves or blinks while you read; the
syncing state is a word, and new mail appears in the list without moving the selection.

## Cut list

- Agent-driven categorization (consultant): the brief asks for no AI features; no persona sorts mail into categories weekly in a terminal client.
- Trends / categorization side pane, 1/4 width (consultant): no one acts on it; it costs the reader 30 cols.
- Volume projection, "minutes to inbox zero", braille sparkline (consultant): no one acts on a forecast or a rate; a clock contradicts "unhurried"; a graph moves while you read.
- List ordered by when you processed each message (consultant): newest-first is how all three find new mail.
- Notes / annotations on a message (consultant): not in the brief; Wren and Osei annotate in their own editors.
- Larger logo in the corner (consultant): visual brand, decided later; it would take rows from the list at 80x24.
- ctrl+? for help (consultant): most terminals don't send it; ctrl+g everywhere instead (same intent: one help key).
- Per-row reading time "3 min" (old build): read as received time; no one decided by it; it costs 6 cols per row.
- "3 waiting, about 4 min of reading" list summary (old build): the unread count already sits in the status row.
- "People in your inbox" contact sidebar on new message (old build): 40 cols for a list you need once per message; completion under To does it.
- List beside the reader, and two-line rows, at 80x24 (old build): reader under 40 cols; ~10 messages per glance.
- Mark read / unread key `m` (old build): read-but-not-archived already means "come back later".
- Different help keys per screen, `?` and ctrl+g (old build): one key everywhere.
- Search: not in the brief; first candidate for the next slice (Osei, weekly).
- Reply-all, forward, cc, bcc, opening attachments: brief names to, subject, body; To takes several addresses.
- Threaded conversation view, folders, labels, flags, multiple accounts, HTML rendering, desktop notifications: not in the brief, and no persona needs them weekly in slice one.

## Departures from the brief

1. **At 80x24 reading replaces the list instead of sitting beside it.** The brief puts
   messages and reading on one screen; it still is one screen, but in two modes. Priya's
   tmux pane would give a split reader under 40 cols, and Osei's tables would not fit at all.
2. **Archive is added.** The brief names reading, replying and writing only. All three
   personas finish with 10-30 messages a day; without a way to put them away, unread is
   the only marker of done and the list grows without end.
3. **Esc keeps a draft.** The brief doesn't say what leaving compose does. Wren leaves a
   reply mid-sentence to re-read the letter; losing it on esc would cost her the reply.
4. **Sending while offline queues.** The brief covers a failed sync, not a failed send.
   Osei writes on the train; queueing is the only answer that doesn't lose a message.
5. **The undo countdown runs on the inbox, not in compose.** Compose closes on send so Priya
   is back to reading at once; the 5 seconds and the undo key stay in the key row.
