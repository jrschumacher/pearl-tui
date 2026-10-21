# burrow: attack on read-v2

Same grader, same four interviewees as attack-v1 (Hana, typical; Kwame, power user; Ines,
newcomer; Dmitri, skeptic). This round each was walked through v2's answer to the thing
they raised, and then through their own week again.

## Interviews (round 2, follow-ups)

### 1. Hana, translator and essayist (typical daily user)
> "Letters and bulk, yes, that's my Gmail tabs, and my newsletters all have the unsubscribe
> line, so they'd sort. The bank wouldn't, but one key to send it to bulk, fine, once. What
> worries me is the side-by-side. I'm at 120 wide; I'd read almost everything right there
> on the right without pressing enter. You're telling me it stays unread? Then that '3
> unread' is a lie by lunchtime and I'll stop looking at it. And when I answer my editor
> I go back to her letter four or five times while I write. Last time the letter sat next
> to my reply; now it's under my reply and half the screen is blank? Why?"

### 2. Kwame, historian, three co-authored projects (power user)
> "Archive I can get back now, good, and the drafts and sent folders. Search on the
> webmail once a week, I can live with that for a while; I'm not losing anything, I'm
> walking to another room for it. Reply to everyone by default is right for my co-authors.
> The lists: Thunderbird does that filing when Thunderbird is running. If I stop running
> it, all four lists land in the inbox, and your bulk view is the four of them shuffled
> together by time. I read one list at a sitting. On the train I write three replies,
> close the lid at the station; 'it sends next time burrow is open' is honest, at least."

### 3. Ines, first-year PhD student (newcomer, week one)
> "Pressing down doesn't mark things read any more, good. The department mail goes to
> bulk, which is where it belongs. But I still had about two hundred real ones unread
> from people, from before. There's no 'mark all read', so it says '200 unread' forever,
> or I archive two hundred times. Saving the PDF to Downloads is fine, I can open it from
> there. My supervisor names it 'comments.pdf' every single week, though. Does the new one
> replace last week's?"

### 4. Dmitri, infrastructure engineer, neomutt since 2014 (skeptic, would churn)
> "Inline quote, good. Quit sends the pending one, good, that's the right call. But r is
> reply-all? In mutt r is the sender, g is the group. In Gmail r is the sender, a is all.
> Every client I've touched. I'll press r to tell one person 'your patch broke staging' and
> it goes to the team list. You say every name is shown: in one row, at 80 columns, with
> six people and a list address? And I archive fast: send, e, e, e. Does my send's undo
> survive three archives?"

## Objections

### 1. BLOCKING: the fix for objection 3 overcorrects; at 120x40 nothing read in the preview becomes read
v2: "Read means opened (enter, click, next-message key), never passed over: at 120x40 the
reader previews the selected row, which is not reading." That fixes Ines's complaint
(arrowing past no longer marks) but breaks the typical user at the typical size. At 120x40
the preview *is* where Hana reads: a short letter fits whole in the 74-col pane, and she
never presses enter. So her read letters stay unread, the status row's "3 unread" (the only
count v2 keeps) climbs through the day, and she stops trusting it (Hana: "a lie by
lunchtime"). The read must give a rule that serves both: a message shown in the preview
becomes read after it has stayed selected for a stated time (Thunderbird, which Kwame and
the read's Osei use today, does this), or when the preview is scrolled. Say the number. The
mark-unread key covers the rest.

### 2. BLOCKING: the two-notice row can push a live send countdown out of reach
v2: "Notices take it for 5 seconds: newest first, at most two; u undoes the newest, again
undoes the other." Dmitri's sequence is send, then `e`, `e`, `e` inside five seconds, and
Hana archives the letter she answered and then the newsletter under it. The third notice
pushes the send's notice out while its countdown still runs. Its undo is then unreachable,
which breaks the one thing the brief promises about sending ("send can be undone for 5
seconds"). State the rule: a send's countdown keeps its place in the row until it ends,
and the newest other notice takes the second slot (or say how else the send stays undoable).
The two-notice row is otherwise earned. It exists for the reply-then-archive rhythm that
both Hana and Dmitri described, and it isn't creep.

### 3. BLOCKING: r means everyone, guarded by a To row that cannot show everyone
Departure 5 makes `r` reply to all and `R` to the sender. The reason is sound (Kwame: replying
to one of four co-authors is the costly silent mistake). But:
- **The guard doesn't fit.** Compose gives To "one row" and promises "every name is shown,
  each removable". Dmitri's team thread has six people and a list address, and Kwame's
  co-author threads have four plus a list. That's 120+ characters of names in a ~70-col row
  at 80x24. A builder will truncate ("+3 more"), and then the guard is gone. Say what To does
  when the names don't fit (it grows to as many rows as needed on reply, or something
  equivalent), and make the recipient count visible at send.
- **The mapping inverts every interviewee's habit.** mutt (`r`/`g`), Gmail (`r`/`a`) and
  Thunderbird (reply/reply all) all put "sender" on the plain key. Dmitri: "I'll press r to
  tell one person 'your patch broke staging' and it goes to the team list." That mistake is
  as costly as Kwame's, and it isn't silent: it can't be taken back once the 5 seconds pass.
  The designer may keep `r` = all, but the read must name this cost and rely on the guard
  above. It can't rest on departure 5's one-sided reason.
The two-key split itself is not creep. Both directions are weekly for Kwame (co-authors
versus an off-list word to one person) and daily for Dmitri.

### 4. MINOR: the bulk view mixes every list into one stream
The letters/bulk split meets Hana's need outright (her newsletters carry List-Unsubscribe,
and the bank is one "move sender" press). It meets Kwame's only if his server files his
lists into folders. Thunderbird's filters run while Thunderbird runs, so if burrow replaces
it, his four lists land in the inbox and bulk shows them interleaved by time. Kwame: "I read
one list at a sitting." Either state the assumption (lists are filed server-side; the
overlay reaches them), or group bulk by list (List-Id), which uses the same headers. Without
one of the two, Kwame's daily list sitting is worse than today. That's why this is MINOR and
not blocking: the mail is all there, just badly ordered.

### 5. MINOR: the folders overlay and outbox are earned, but enter is undefined in three places
The overlay is not creep. It serves archive retrieval (Kwame weekly), list folders (Kwame
daily), sent ("did it leave", Kwame), drafts and outbox, without a third screen. But a
builder must guess:
- enter on a message in **drafts**: read it, or resume it in compose?
- enter on a **queued** or **not sent** message in the outbox: compose, or a reader with
  retry/cancel? And a send that fails *after* compose closed: v2 says "draft restored, 'not
  sent: <reason>'", so does burrow pull Hana out of a letter into compose, or wait in the
  outbox?
- `r` on a row marked with a draft: resume that draft, or start a second reply?
One line each.

### 6. MINOR: #8 holds, if it says what happens to a second file with the same name
Ines can open a PDF from ~/Downloads, which closes the loop she had, and opening can stay
out. Missing: Ines's supervisor sends "comments.pdf" weekly, so say that a save never
overwrites (it renames, and the notice names the file actually written). Also say whether
"saves the message's attachments" includes inline images such as signature logos, which
Kwame's co-authors' mail is full of. Saving those clutters Downloads.

### 7. MINOR: #11 holds for Kwame and Hana but not for the newcomer's first hour
The letters-only count fixes the shouting for Kwame. Ines still arrives with about 200 unread
letters. "Mark all read" is cut as "once per person", but once per person is exactly the
newcomer's first hour, and every customer is a newcomer once. The only exit is to archive
200 times, or to accept "200 unread" for weeks. One offer on first load ("mark the 200 before
today as read?") or a mark-all key in the bulk and letters views would do it. Otherwise
write the cost down next to search's.

### 8. MINOR: compose at 120x40 drops the original and leaves 46 cols blank
v2 moves the quote into the body (right for Kwame and Dmitri) and cuts the original beside
the draft. Hana goes back to the letter "four or five times" while she writes; v1's own
departure 3 was built on Wren doing the same. With the quote below a growing reply, she
scrolls her own text away to re-read it, while "~46 cols stay empty". The editable quote and
a read-only original beside it don't conflict. Keep the original beside the draft at 120x40
and keep the editable quote, or say why the empty columns serve Hana better.

### 9. MINOR: HTML "tables" will catch newsletter layout tables
"Shown as text: headings, paragraphs, lists, tables." Nearly every HTML newsletter (Hana's
daily reading) is built from nested layout tables. Kept as tables, they clip sideways into
a grid of fragments. Say that only tables of data keep their shape, and that layout tables
flatten into paragraphs.

### 10. MINOR: the status row overflows 80 cols in the states that matter most
At 80x24 offline with queued mail: "inbox · letters", "3 unread", "offline since 09:12,
retrying every minute", "2 waiting to send" comes to ~90 characters before gaps. Say what
yields first. Kwame's "waiting to send" and the sync state should be the last to go.

## Checked and holding (no objection)
- **Letters/bulk from List-Unsubscribe, Precedence, Auto-Submitted, plus "move sender"**:
  meets Hana's need in full and Ines's (department mail goes to bulk). It meets Kwame's
  subject to objection 4. Two classes and one correction key is the consultant's
  categorisation kernel without his machinery. The cut of more classes and the agent stands.
- **Search in slice two**: holds. Archive is now reversible and browsable, so nothing is
  lost. Kwame pays a weekly trip to webmail and accepts it ("walking to another room for
  it"). No interviewee needs it daily. The written-down cost is the right form.
- **Quit in a countdown sends at once; quit with mail queued asks once**: Dmitri's churn
  reason is gone, and Kwame's train case is honest.
- **Sign-in refused distinct from offline; no-account state with the config path**: closes
  objection 6. Setting up an account outside the two screens is within the brief.
- **Next/previous keys, archive moves down, end of message stops, mark unread restored**:
  close objection 3's other parts.
- **Outbox and "2 waiting to send"**: earned by Kwame's three train replies. A count beside
  unread is justified here because someone acts on it.
- **Cuts**: notes, trends, projections, larger logo, ctrl+?, reading time, contact sidebar,
  cc/bcc/forward/threads remain right.

VERDICT: REVISE
