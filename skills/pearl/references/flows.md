# Flows: the screen follows the job and the data on it

Layout rules say where a region can go; the job says where it should. Put a screen's
data where the user's task needs it and most layout decisions make themselves: a grader
who saw a run's log stacked under its steps said that anyone who thought about the data
"would have always put the live data to the right". This file is loaded per screen, by
the flow its `job:` line names (`cmd-shape.md` § For Each Screen), never per app: a
rental app is browsing until its last step turns into a payment, and only that step
reads § Checkout.

## Name The Data Before Placing It

Each region holds one kind of data, and the kind decides its place:

- **Live** (a log tail, a running step, a progress stream) goes beside what it belongs
  to, in reading order after it: right of the list at 80 columns when the list fits in
  about 30, below it only when it can't. A live pane under its list sends the eye
  between the middle and the bottom on every update.
- **Background** (a sync, a refresh, a token's expiry) is a glyph, not a sentence: one
  cell in the header's details or the footer's status, with a key on the help screen
  that opens the details. It takes words and a place in the body only while the user is
  waiting on it. "Scores synced 2 min ago" in a header was read as "no one cares".
- **Money** gets § Checkout's receipt wherever it appears.
- **The product's purpose** (the track playing, the deploy in flight, the booking being
  made) stays in view on every screen that isn't about it, as a one-row strip: a
  player's search keeps a now-playing bar along its bottom. It is the context a user
  carries from screen to screen; dropping it makes them go back to check.

**DO** write a screen's data kinds on its `data:` line, region by region.
**DON'T** give background status a sentence in the header; the header is for where you
are and what you're acting on.

## Brand Recedes Where People Work

People are on a working screen to do their job, not to look at the brand. On a list, a
form or a monitor the mark is the one-row signature beside the name (`brand.md`); a tall
mark in the header was read as "way too large", and pushes the job down. A product
whose content is a picture can give up the mark there: a player whose screen belongs to
the artwork puts its name in a corner with a short brand gradient, and was called
"really refined" for it. The splash is where the mark is big. The exception is a header
band built to carry it: a taller mark set into a band that holds the screen's context
beside it ("the header band actually works") was the grader's pick over a one-row mark,
because the band, not the mark, is the header.

## Monitoring And Live Work

CI runs, deploys, queues, logs. The user is waiting for something to finish or fail.

- The failing thing is findable from the list without opening anything: its row's
  status, and one line naming where it broke.
- A run's detail is two panes: its steps left, the selected step's log right, following
  the running step until the user moves.
- Each failure on its own line: two runs' states on one line read as one.

## Checkout And Money

Any step that charges, pays or quotes a total.

- **A receipt**: line items with their amounts right-aligned in one column, the total
  last in the accent, the payment method on its own line apart from what is being
  bought. Amounts on the left of their labels read as a form, not a bill.
- **The act defaults to the purchase.** The user came to buy; the review step is the
  confirmation, its button names the act and the amount (`Pay £240 and book`), and
  focus arrives on it (`interaction.md` § Confirm By Consequence, rung 3).
- **The other button says what it does**: `Back to edit` or `Keep looking`, never a bare
  `Cancel`, which leaves the user asking whether their answers are lost.

## Media And Playback

Players, readers, galleries: the user is there for the content.

- The content leads (artwork, the track, the page); chrome is quiet.
- What is playing persists on every other screen, as a strip with its progress.
- No motion that competes with the content: no bouncing bars, no idle animation.

## Browse And Search

Lists to compare and choose from.

- The selected item's detail is one region beside or on one side of the list
  (`brand.md` § Move A Region), never split across both sides.
- A search highlights the matched text in each result.
- The number the user compares by (price, score, size) is a right-aligned column, and
  the one that decides the choice is accented on the selection.

## Configure

Setup wizards and settings: the user wants to be done, and to know how far that is.

- **The whole journey stays in view**: every step this user will see, as a column or a
  row, the done ones carrying their answer and the current one lit. "It can get really
  frustrating when you don't know… how many pages am I gonna be going through." A branch
  that adds or removes a step updates the list and the count as soon as it's chosen.
- **Going back keeps every answer**; changing one that removes a later step drops only
  that step's answer.
- The steps, the summary and the review are in `components.md` § Forms And Wizards.
