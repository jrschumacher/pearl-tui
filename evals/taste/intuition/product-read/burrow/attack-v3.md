# burrow: attack on read-v3 (final round)

Same grader and the same four interviewees (Hana, typical; Kwame, power user; Ines,
newcomer; Dmitri, skeptic). Each was walked through v3's answer to what they raised in
round 2, then through their first week with v3 as written. Bar for BLOCKING this round:
a customer hits it in their first week.

## Interviews (round 3, first-week walk-through)

### 1. Hana, translator and essayist (typical daily user)
> "Two seconds on a letter and it counts as read: that's what I'd expect, I read things
> where they sit. My editor's letter is beside my reply again, so I don't lose my place.
> Newsletters grouped by who sent them is nicer than Gmail, honestly. The first-day
> question about old unread ones, I'd say no; I know which thirty I'm keeping. Nothing here
> I'd skip past, and nothing I'd go hunting for in week one."

### 2. Kwame, historian, three co-authored projects (power user)
> "If I stop running Thunderbird, my lists land in bulk, one group per list, newest list on
> top. That's my sitting. r to the sender, a to everyone, and if I forget, it tells me
> three co-authors are left off. That's the mistake I made, caught. Search on the webmail
> for now, as agreed. The train: queued, 'two waiting to send' in the corner, gone on the
> next sync. Fine. The 'mark them read?' on day one I'd answer no, because Thunderbird
> sees the same flags and I'd lose my markers there too."

### 3. Ines, first-year PhD student (newcomer, week one)
> "The question on day one is exactly what I wanted: yes, and the two hundred are gone.
> Department mail in bulk, grouped. The PDF lands as 'comments (2).pdf' and the line tells
> me so. I looked around with the arrows and only the ones I stopped on went read, which is
> what I did, so fine. My first hour is still the config file, but that's not this screen's
> fault."

### 4. Dmitri, infrastructure engineer, neomutt since 2014 (skeptic, would churn)
> "r sender, a all: thank you. Send, e, e, e, and the send still sits there with its own
> unsend key. Quit sends it. That's the client I'd keep. Two nits. When I press r on the team
> thread it says 'a adds them', but my cursor's in the body, so a types an a. And
> 'sending 2': I've never sent two mails inside five seconds in my life, but whatever.
> Nothing I'd leave over."

## Round-2 objections: closure

| # | Objection | Closed? | Note |
|---|---|---|---|
| 1 | Preview never marks read | **closed** | 2 s selected or first scroll; arrowing past marks nothing (Hana, Ines both satisfied). |
| 2 | Send pushed out of the notice row | **closed** | Send holds slot one with `U` until it leaves, however many archives follow (Dmitri). |
| 3 | r = all; To too small | **closed** | Convention restored; the "others aren't included" line catches Kwame's mistake; To grows to 4 rows. See minor 1 for the hint's key. |
| 4 | Bulk mixes lists | **closed** | Grouped by List-Id, else sender; server-filed folders assumed and written down. |
| 5 | Enter undefined in drafts/outbox | **closed** | Resumes in compose; r/a on a draft row resumes; late failures go to the outbox without interrupting. |
| 6 | Saves overwrite; inline images | **closed** | Renamed, never overwritten, notice names the file; inline images not saved. |
| 7 | Newcomer's 200 unread | **closed** | One first-load offer; no standing key. See minor 2. |
| 8 | Compose blank at 120x40 | **closed** | Original beside the draft, read-only; editable quote kept. |
| 9 | HTML layout tables | **closed** | Data vs layout rule stated. See minor 3 for its edge. |
| 10 | Status row overflows at 80 | **closed** | Shortening order stated. Worst case (offline, 2 queued, 1 not sent) now fits ~65 cols. |

## Final creep check

Every key and state was traced to an interviewee's weekly or daily need:
- **2-second preview rule**: not creep. It is one number replacing a rule that broke
  Hana's count (round 2). Without it the count lies; with it nobody configures anything.
- **First-load "mark letters before today as read" offer**: not creep. It appears once, only
  when there is a backlog, and replaces a standing key v2 had already cut. It is the
  newcomer's first-hour need (Ines). Hana and Kwame each answer it in one keystroke.
- **Folders overlay, outbox, "2 waiting to send"**: earned (Kwame's lists, archive, train
  sends). Unchanged since round 2.
- **`U` unsend beside `u` undo**: earned. It is the only way to keep the brief's 5-second
  promise when archives follow a send (Dmitri's and Hana's rhythm).
- **Bulk grouped by list; move sender for good**: earned (Kwame's sitting; Hana's bank,
  Dmitri's team list). They use the same headers, so no new machinery.
- **Original beside the draft at 120x40 and the editable quote**: both earned, by different
  people (Hana, and Kwame and Dmitri).
- **"Sending 2 · U unsends the newer"**: the one rule no interviewee needs (minor 4).

Nothing else has grown past the interviews. The cut list still holds, and every departure
now cites a customer.

## Objections

### 1. MINOR: "a adds them" names a key that types text in compose
After `r`, compose shows "3 others on the original aren't included · a adds them", but the
cursor is in the body (or To), where `a` is a letter. Dmitri: "a types an a." Give the hint a
key that compose doesn't treat as text, or make the line clickable and name a ctrl chord.
Builder note; nobody is harmed, since the line still tells them who is missing.

### 2. MINOR: the first-load offer needs a safe default and a stated reach
- The flags it sets are the server's, so Kwame's Thunderbird loses its markers too. Make
  `n` the default (enter or esc declines), so a fast first keystroke can't clear 142
  markers for good. Or give it the same 5-second undo as other bulk actions.
- "142 letters from before today" must count the whole inbox on the server, not only the
  newest 500 headers first load fetched. Otherwise letters past 500 stay unread and Ines's
  count doesn't fall the way the offer promised.

### 3. MINOR: "a header row" will miss plain HTML data tables
"A table with a header row and no table inside it is data." Many hand-made tables sent as
HTML from webmail use only `<td>`, with the first row bold rather than a header row. Those
would flatten into paragraphs. Builder note: treat a small, regular, non-nested table as data
even without `<th>`. Only HTML-only mail is affected; multipart mail uses its plain part.

### 4. MINOR: "sending 2" specifies a case no one has
Two sends inside one 5-second window means writing a whole message in under five seconds.
No interviewee does it (Dmitri: "never in my life"). Keep the behaviour simple (the newer send
takes the slot and the older one leaves at its time), or drop the special copy. Not worth a
test matrix.

### 5. MINOR: sign-in refused needs a short status-row form
The state's copy ("mail.example refused the sign-in at 09:12; stopped retrying") is ~57
characters. In the 80-col status row, next to "3 unread" and an outbox count, it can't fit.
State its short form for the status row (e.g. "sign-in refused"), with the full sentence where
the list would be, the same way "offline since 09:12" shortens.

VERDICT: CONVERGED
