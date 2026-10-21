# burrow: attack on read-v1

Grader: adversarial UX research. Sources: brief.md (binding), consultant.md (one voice),
old-screens/ (the literal first build), read-v1.md (under attack). The interviews are
simulated, from the brief's audience: writers, researchers and engineers who live in the
terminal, burrow open for hours, 120x40 typical, 80x24 must work, dark terminals.

## Interviews

### 1. Hana, translator and essayist (typical daily user)
> "It's open all day next to my working file, 120 wide. Maybe 25 messages a day: my two
> editors, a translators' list, four or five newsletters, the bank. I answer three, four.
> Today it's Gmail in a tab, and the tab is the problem: I go to check one thing and lose
> twenty minutes. What annoys me most is that the editor's letter sits between two
> newsletters and I scroll past it. In Gmail the newsletters go to their own tab and I
> read them on Friday. When I reply I write a reply and then I archive the letter right
> away, that's the rhythm, reply then put it away. Half my newsletters are the pretty
> kind with pictures; I don't need the pictures, but I need the words."

### 2. Kwame, historian, three co-authored projects (power user)
> "Sixty a day, easily. Forty of those are from four mailing lists that Thunderbird
> files into folders before I see them; the inbox proper is maybe twenty, co-authors and
> students. I read the lists on the train, every day, and I reply on them, inline, under
> the paragraph I'm answering. Threads with co-authors have three or four people on them,
> and the worst thing I ever did was reply to just one of them for a week without
> noticing. On the train the wifi drops every ten minutes, so I write three replies and
> I want to know, later, that all three left. Once a week I need a table someone sent me
> last month, and I search for it. I archive aggressively, but only because I can find
> things again."

### 3. Ines, first-year PhD student, never used terminal mail (newcomer, week one)
> "My lab uses terminal everything, so I installed it on Monday. First thing: how do I
> even log in? The university uses the Outlook-style sign-in. Then I had 1,400 unread
> because I've never archived anything in my life. The department sends everything as
> those HTML announcements. My supervisor sends PDFs with comments on my chapter, every
> week, and I need to open them. I pressed down a few times to look around and now I
> don't know which ones I actually read. I'd use it every day if it just showed me
> what's new and let me open the PDF."

### 4. Dmitri, infrastructure engineer, neomutt since 2014 (skeptic, would churn)
> "I've got 400 lines of muttrc and I'll leave it for something prettier only if it
> respects how mail works. Eighty columns in a tmux pane, always. I reply inline, trimmed;
> a client that bolts the whole quote under my answer is a client for managers. I reply-all
> to the team list ten times a day. I send, I immediately archive the thread, I quit; if
> quitting eats the send, I'm gone that day. I mark things unread when I open one too
> early; I don't need a sermon about workflows, I need `N`. Pretty is fine. Losing a
> message is not."

## Objections

### 1. BLOCKING: the read never decides which mail burrow shows, and dismisses the categorisation need on a false premise
The cut list says "the brief asks for no AI features". The brief says nothing about AI
either way; the cut has to stand on customers, not on a misquote. The agent, the trends
pane and the projections deserve cutting (no interviewee acts on a forecast; Hana and
Kwame both said they want *fewer* things moving). But the consultant's kernel is real:
people who read a lot of mail separate letters from bulk before choosing what to open.
Hana does it with Gmail's tabs ("the editor's letter sits between two newsletters and I
scroll past it"); Kwame does it with Thunderbird folders ("forty of those are from four
mailing lists that Thunderbird files into folders"). The read cuts folders and labels
and shows "the inbox" without saying what that is. Either answer breaks someone:
- inbox folder only: Kwame's daily list reading (his largest job, every day) vanishes
  from burrow, and he keeps Thunderbird;
- everything: Hana's and Ines's letters drown among newsletters and announcements.
The read must name what the list contains (which server folders, or the inbox plus
which others) and how a person tells a letter from bulk mail at a glance, without an
agent and without a side pane. A single non-AI distinction (list or bulk mail versus a
person writing to you, from headers the server already gives) would serve the customer
need the consultant pointed at at a fraction of the cost. Weighing the consultant: right
about the need, wrong about the machinery.

### 2. BLOCKING: archive is one-way
Departure 2 adds archive, rightly (Hana: "reply then put it away"; Dmitri archives
every thread). But search is cut, folders are cut, and nothing says where archived mail
goes or how one gets a message back after the 5-second undo. Kwame: "I archive
aggressively, but only because I can find things again." With no way back, careful
users stop archiving, and the list "grows without end", which is the problem departure 2
exists to solve. The read must say, for slice one, how an archived message is reached
again (an archived view, a minimal search, or an explicit statement that it is only
reachable outside burrow, with that cost accepted in writing).

### 3. BLOCKING: the core reading loop's transitions are undefined
Job 1-3 together happen 10-50 times a day; the read specifies the screens but not the
moves between them. A builder has to guess all of these:
- **When does a message become read?** At 120x40 the list and reader sit side by side.
  If selecting a row previews it and marks it read, arrowing past six rows marks six
  read. Ines: "I pressed down a few times to look around and now I don't know which ones
  I actually read." The read also cuts the `m` key (Dmitri: "I need `N`"), so the
  mistake can't be undone. Name the rule (on open in reading mode, after N seconds
  shown, or on scroll), and either restore a mark-unread key or justify the cut with a
  rule that makes accidental reads impossible.
- **What does archive select next?** In choosing and in reading mode, at both sizes:
  the next older, the next newer, the next unread, or back to the list?
- **How do you get to the next message from the reader at 80x24?** Reading replaces the
  list there; going esc, down, enter for each of 10-20 messages is three keys where one
  would do. The consultant asked for this too ("hotkeys more focused on reply or next
  message"), and here the customers agree with him.
- **What does a key do at the end of a long message?** Stop, or move on?

### 4. BLOCKING: the send lifecycle breaks on its most common path
"A notice takes this row for 5 seconds with its undo key." The most common sequence in
the interviews is reply, send, archive the original at once (Hana: "reply then put it
away"; Dmitri: "I send, I immediately archive the thread, I quit"). That puts two
notices into one row inside 5 seconds. Undefined, and each guess loses something:
- which notice holds the row, and which action does the undo key undo?
- does quitting during the countdown send now, cancel, or wait? (Dmitri: "if quitting
  eats the send, I'm gone that day.")
- undo send returns you where: compose with the draft as it was, or just a kept draft?
- after the 5 s, a queued offline message and a failed send have no home. Kwame: "I
  write three replies and I want to know, later, that all three left." "Deliberately not
  there: counts of anything but unread" forbids the one line that would tell him. Say
  where queued and failed sends are visible after the notice, what a failed send keeps
  (the draft, restored and marked), and whether burrow must stay open for a queued send
  to leave.

### 5. BLOCKING: a reply's recipients and quote are decided by omission, against the power user and the skeptic
- **Recipients.** The read cuts reply-all and cc, and says To "accepts several
  addresses". It doesn't say whom To is filled with on reply. If it is the sender only,
  every co-author thread silently loses its other members. Kwame: "the worst thing I
  ever did was reply to just one of them for a week." Dmitri reply-alls to the team list
  ten times a day. The brief limits the fields to to, subject and body; it does not stop
  a reply from filling To with the original's other recipients. Decide it, and show who
  will receive the reply before send.
- **Quote.** "Sent quoted below the reply" is top-posting with a quote the writer can't
  trim or answer into. Kwame and Dmitri both reply inline, and on mailing lists that is
  the etiquette, not a taste. Hana and Wren-like writers are fine either way. The cheap
  answer is the quote placed in the body as ordinary editable text, cursor above it;
  that serves both groups. The current one serves neither power user and is Dmitri's
  stated churn reason.

### 6. BLOCKING: day one starts before the first state the read lists
"First load (no local copy yet)" assumes an account that works. Ines's first question
was "how do I even log in?", and her university uses an OAuth-style sign-in. The read
has no state for "no account configured" and none for "the server refused the sign-in".
The second is worse than missing: with only "offline since 09:12, retrying every minute",
an expired password reads as a dead wifi, and Kwame on the train would wait an hour for a
network that isn't the problem. Account setup can stay out of slice one, but the inbox
must say, in its own words, (a) there is no account yet and where it is set, and
(b) sync failed because sign-in failed, distinct from offline.

### 7. BLOCKING: cutting "HTML rendering" leaves HTML-only mail undefined
The cut is right about rendering layouts and images. But Hana reads "four or five
newsletters" daily ("I don't need the pictures, but I need the words"), Ines's department
"sends everything as those HTML announcements", and the read's own Wren reads newsletters.
Many of these have no plain-text part, or one that says only "view this in your browser".
The read must say what the reader shows for an HTML-only message: at minimum its text,
with its links kept whole (the read already promises whole URLs). This is day one for a
newcomer and daily for the typical user, so it can't wait for a later slice.

### 8. MINOR: attachments are a dead end
"Attachment names if any (listed, not opened)." Ines gets a PDF from her supervisor
weekly; Kwame needs a co-author's table; the read's Wren gets marked-up chapters. Listing a
name with no action sends each of them back to the browser tab that Hana and Wren are
trying to leave. Opening can stay out; one key that saves an attachment to a stated
folder (and says where) would close the loop. If it stays cut, say how a user gets the
file today.

### 9. MINOR: the 80x24 row leaves widths a builder must guess
"Sender 18 cols, subject flexes, time 5 cols." Five columns hold "09:41" and "mon" but not
an unambiguous date before this week ("10/12" reads two ways). The unread, replied and
draft marks have no width in the 80-col budget. State the older-date format and the
mark column, and check that subject still gets 40+ cols.

### 10. MINOR: "resumed by r or c" is unclear for new messages
Drafts are "one per message, resumed by r or c, marked on the list row". A reply draft
has a row; a new-message draft has none. Does `c` always reopen the one kept new draft? Then
how does Hana start a second letter while one is kept? Say how many new drafts can be kept
and where they show.

### 11. MINOR: the list's extent is unstated
Ines has 1,400 unread; Kwame's inbox is years deep. "Earlier" has no floor. Say how far
back the list goes (all of the inbox, loaded as you scroll, or a window), so first load and
scrolling have a defined size. Related: "3 unread" in the status row becomes "1,400 unread"
for Ines and a permanent three-digit number for Kwame's lists if they share the list. That
is the inbox that shouts. Consider whether the count counts only what objection 1 calls
letters, or only what arrived since the last look.

### 12. MINOR: an 80x24 tmux pane is promoted to a daily persona without saying so
The brief says "a full-screen tab, often at 120x40 or larger, but it must work at 80x24".
The read's Priya lives at 80x24, which makes the minimum a primary layout. Dmitri confirms
such users exist, so the choice is defensible, but it is a departure and belongs in that
list, so whoever trades off the two layouts knows 80x24 was raised on purpose.

## What the read gets right (no objection)
- Cutting the agent, trends pane, projections, braille graph and minutes-to-zero: no
  interviewee would act on them, and Hana and Kwame both asked for less movement.
- Cutting per-row reading time and the "about 4 min of reading" summary: the old build's
  "3 min" reads as a received time (the consultant tripped on it too).
- Newest-first over processed order; one help key (ctrl+g) over ctrl+? that terminals
  don't send; reading replacing the list at 80x24; esc keeping a draft; send queueing
  offline; completion under To instead of a 40-col contact sidebar.
- Search deferred to slice two is acceptable *only* if objection 2 gets an answer.

VERDICT: REVISE
