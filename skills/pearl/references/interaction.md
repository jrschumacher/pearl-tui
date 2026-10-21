# Interaction and copy: keys, focus, states, words

*Bubble Tea: the `tea.*`, Bubbles and Huh names in § Keys Are A Contract, § Async And Long
Work, § Terminal Citizenship and § Screen Readers are Bubble Tea v2's; the rules hold for
any framework.*

## Keys Are A Contract

- Define every key once, in a per-state **action table** (binding + handler), which both
  the dispatcher and the footer read (`bubbletea-architecture.md` § Keys). A key in the
  help that doesn't work is a bug [C-KEYS], and `TestHelpIsHandled` makes it a failing
  test.
- Actions act on **the selection**, not on the first item of the fixture. Test it:
  select the fifth row, press the key, and check the dialog names the fifth item.
- **Conventions users already have:** `↑/↓` and `j/k` move, `enter` selects/submits,
  `esc` goes back or closes, `tab`/`shift+tab` cycle focus, `/` filters, `?` toggles full
  help, `ctrl+c` quits (or confirms quitting when work would be lost), `q` quits only
  when no text input is focused, `g/G` jumps to the top/bottom, `pgup/pgdn` scrolls, and
  `ctrl+z` suspends (bind it to `tea.Suspend`; v2 and the kit don't on their own).
- **Every overlay but help, and every page that draws a focus mark, binds a help
  key** in its own action table: `?`, or `ctrl+g` where a focused text box types `?`.
  A Bubbles textarea binds `ctrl+g` to select-all, so binding it to help costs
  select-all in that state; bind it in that state's table, never in Global, so the
  other text boxes keep it. Help is the way to the keys a footer cuts, and the kept
  harness tests open it from each such state and fail the one where no key does.
- **Command palette** (`ctrl+p`, or `:`) for everything else. It lets you keep the
  footer to the 5–7 most useful keys.
- **Mnemonic single keys** for frequent actions (`d` deploy, `r` roll back), shown in the
  footer and as right-aligned hints in the palette.
- **Help changes with context.** In a dialog, the footer shows the dialog's keys. Pressing
  `esc` once with typed content shows "esc again to discard". Capability matters too:
  bind `shift+enter` and `ctrl+j` with the help key `shift+enter/ctrl+j` and set
  `t.KeyDisambiguation` from `SupportsKeyDisambiguation()`: `ui.Help` draws the one that
  works there (ctrl+j, or shift+enter), never the 18-cell pair.
- **A toggle's help names what the key does now:** `f pause` while following, `f follow`
  while paused, as the kit's `enter` reads `deploy` or `cancel` from the focused button.
  `TestHelpIsHandled` proves the key fires, not that the label is true, so render both
  states and read the footer against the header: `following` beside `f follow` is a lie.
- Key order in the help: the primary action, navigation, secondary actions, then
  `? help`/`q quit` last. It truncates from the right with `…`, and `ui.Help` pins the
  first `?`/`ctrl+g` binding last wherever the table lists it, so the way to the rest
  survives the cut.

## Focus

- Exactly one visible focus. It moves with `tab`, with a click, and to a dialog when one
  opens.
- Focus is shown by the gutter bar, the label weight, and the prompt colour. It never
  changes layout.
- Unfocused-but-selected stays visible in a quieter tone and its own glyph, never the
  focus bar recoloured: `t.BlurBar()`, `╎` unless the theme sets `glyphs.blurBar`, `:`
  under ASCII, so NO_COLOR still shows one focus.
- Overlays trap focus. Keys go to the front overlay only, and `esc` closes it and
  restores the previous focus.

## Mouse

Opt in per frame (`View.MouseMode`). `CellMotion` gives clicks and wheel. Turn mouse
off when the user disables it so terminal text selection works. Make the obvious things
clickable (tabs, list rows, buttons via compositor hit regions, "N lines hidden" to
expand). Coalesce wheel events; some terminals send 3 per notch. Mouse is a convenience:
every action must also have a key.

The wheel and the keys answer to different things, because a desktop taught both:

- **The wheel scrolls the pane under the pointer**, and focus stays where it is. A click
  is what moves focus. Scrolling the focused pane from across the screen, or stealing
  focus on a wheel, leaves the user typing into a pane they weren't looking at.
- **The wheel moves the view, never the selection.** The selected row may scroll out of
  sight; the next arrow key brings the view back to it before moving. A wheel that walks
  the selection runs actions on a row the user scrolled past.
- **Shift+wheel and a trackpad's side scroll move sideways** in a pane that overflows.

## Panes That Scroll

- **`tab` moves focus between panes; arrows act inside the focused one.** `↑↓` move the
  selection or the view, and `←→` scroll sideways when the focused pane's content is
  wider than it. So `←→` never mean both "scroll" and "change pane".
- **Show what's hidden.** A pane scrolled sideways marks its cut edges with `‹` `›` (ASCII
  `<` `>`) on the rows that continue; a pane with more rows than fit carries the
  scrollbar or a "N more" line. A user who can't see that a pane scrolls never scrolls it.
- **Each pane keeps its own offsets**, vertical and horizontal, across focus changes and
  resizes; a resize clamps them, never resets them.
- **Help names the mouse once.** The footer shows keys only. The full help screen ends
  with a short Mouse section: click focuses, the wheel scrolls the pane under the
  pointer, shift+wheel scrolls sideways.

## Async And Long Work

- Start the work as a `tea.Cmd`, mark the region busy, show a spinner with a subject, and
  keep the rest of the UI usable.
- Cancel with `esc` or `ctrl+c` (with a confirm if it would leave something half-done),
  then show "Canceled." in the region.
- Over 3 s, show elapsed time. Over 10 s with a known total, show a progress fraction.
  Over 30 s, consider letting the user leave the screen while it continues (a badge on
  the tab, a toast when done, `View.ProgramStatus` so the terminal tab shows it).
- Results that arrive for a region the user has left are stored, not drawn over the
  current screen.

## Confirm By Consequence, Not By Write

A confirm is a request for a human's judgement, so ask only when there is a judgement
left for the human to make. A confirm costs a keypress and a read, and each one that
asks for nothing trains the user to press `y` without reading; the one in front of a
delete then gets the same reflex. Sending a write to a server is not a reason to ask.
Nor is "consequential" alone: when automation has already judged an action a revert
can take back (green checks, a passing gate), or the gesture was the judgement (an
explicit send), a confirm adds nothing. Put every action on the first rung that fits:

1. **Reversible in one gesture: the action is the confirmation.** It happens on the
   key, the screen shows it at once (the reaction appears, the thread folds as
   resolved, the row's mark flips), and the same key undoes it: a reaction,
   resolve/unresolve, close/reopen, star, mute, any mark (pin, label). The footer
   names what the key does now (`x close` becomes `x reopen`). At most a toast that
   names what happened and the undo ("Closed #812 · x reopen"). A bulk one stays here:
   "Marked 12 done · u undo" undoes the batch.
2. **Composed, then sent: the send is the confirmation.** A comment, reply, message or
   page the user wrote and sent with an explicit send key (`enter`, `ctrl+s`) goes
   without a dialog. They read it while writing it; "Post this comment?" only asks them
   to read it again. The same holds for a send to people that one labelled key starts
   (`n declare incident` pages on-call): the label is the judgement. This rung is for
   sends to people only; a one-key delete or ship stays on rung 3. It holds when the
   send can't be recalled (a page, an email, a post): sending was the judgement.
   Such a send gets a grace window by default, since the TUI can almost always hold
   it: it waits ~5 s under an undo toast ("Paging payments on-call in 5 s · u undo",
   as mail apps offer undo send), then goes. Skip the window only when holding defeats
   the point (an emergency page where seconds matter): then it goes at once, still with
   no confirm. Where the service allows, offer edit or delete on the sent item. A
   failing signal still moves a send to rung 4: paging a rotation that's off duty, or
   an incident already open for the same thing.
3. **Can't be taken back, so nothing vouches for it: confirm.** Deleting, purging,
   force-pushing, and shipping to production (a release, a deploy, a rollback: a
   rollback is a deploy) even with green checks: traffic meets it the moment it lands,
   and the fix is another ship, not an undo. Hard to undo isn't can't be undone: a
   merge reverts, an issue reopens. Only the user knows whether this is the object to
   lose, so the dialog names it and what can't be taken back ("Delete nightly-0412?
   41.2 GB, no undo"), focuses the safe button, colours the destructive button as
   danger, and accepts `y` for experts. Where focus starts follows the intent, not the
   rung: an act that destroys, removes or replaces what exists (a delete, a purge, a
   force-push, a deploy or rollback over what's running) starts on the safe button; the
   act a flow exists for (a purchase, a booking, a sign-up) starts on the act, because
   the user came to do it and focusing away from it reads as the app arguing (§
   Checkout in `flows.md`). The safe button says what it does, never a bare `Cancel`:
   `Keep v2.4.0`, `Back to edit`, `Keep looking`. "Cancel" on a review left the user
   asking whether their answers were lost. For what destroys production data (dropping a
   database, deleting a prod bucket), make the user type its name; a ship doesn't
   need it, since its fix is another ship. A ship started from a form is confirmed by
   the form's last step when that step says exactly what ships where (version, target,
   what changes), its button is named for the act (`Deploy v2.4.1 to prod`, not
   `Submit` or `OK`) and drawn as danger, and focus arrives on the safe button
   (`Keep v2.4.0`) so enter alone doesn't ship over what's running. No dialog after it: it would repeat what the user just read. The
   dialog is for a ship one key starts (`r roll back` from a list).
4. **Normally safe, but a signal says no: confirm only in that state.** The signal is
   one the user should weigh: red checks, a blocking review, an unsaved draft, a stale
   view of what they're acting on. On a code host, merging with green checks takes no
   confirm: the checks already judged it, and a revert undoes it. That holds where
   main deploys on merge: the pipeline's gate judges that ship. Merging with red
   checks asks, and the question names the signal ("2 checks failing: lint, e2e. Merge
   anyway?"), so the user judges the failure, not "are you sure?". A failing signal
   escalates; a passing one stands in for the confirm only on an action a revert takes
   back. On rung 3 the confirm stays, and a failing signal goes in its question.

**DO** prefer undo to a confirm wherever the action can be held or reversed: a delete
that goes to a trash, a send held for its grace window. Undo lets people act at speed
and recover, so they learn the UI is safe to use; a confirm teaches them it is
dangerous, and to press `y` without reading.
**DO** choose the rung per action when you list the states: a `confirm-*` state exists
only for rungs 3 and 4, and a rung-4 confirm says which signal opens it.
**DO** write the question about the one thing to judge: the object and what can't be
taken back (rung 3), or the failing signal (rung 4).
**DO** bind the toast's undo key in the state's action table while the toast shows, so
the key it advertises works (Law 6).
**DON'T** ask a question that fits any action ("Are you sure?", "Continue?"): it asks for
no judgement, so it gets none.
**DON'T** offer undo for what can't be undone: an undo toast after a purge is a lie,
and that action needed the confirm.

## States Every Screen Designs

Populated, empty, loading, error, partial (some items failed), offline/disconnected,
too small, and the first run. Each is a named harness state
(`bubbletea-architecture.md`), so each gets rendered and reviewed.

## Terminal Citizenship

- Restore the terminal on exit, panic and `tea.Suspend`. Bubble Tea does this; `ctrl+z`
  suspends only when you bind it to `tea.Suspend`.
- Set the window title. Set the program status/progress while busy and clear it before
  exiting.
- Respect `NO_COLOR` (glyphs and words still carry meaning), `TERM=dumb`/`linux` (ASCII
  glyphs), and a light background.
- Offer a non-interactive path (`--json`, plain output when stdout isn't a TTY) for
  anything scriptable. A TUI is not an API.
- Works inside tmux/screen (no reliance on focus events or exotic keys), over SSH, and
  in a 80×24 split pane.

## Screen Readers: The Canvas Is The Hard Case

A full-screen TUI is hostile to screen readers in ways that a line-oriented CLI is not
(see "The text mode lie", xogium.me, 2026). Screen readers follow the cursor and read
linear output. A canvas app moves the cursor around to repaint spinners and timers,
redraws large regions on every keystroke, and lays text out in two dimensions. Users
hear fragments, or their reader crashes as the history grows. No standard (WCAG
included) covers terminals yet, so hold yourself to these:

- **Ship an accessible mode** for every task the TUI performs. Use `--accessible`, and
  honour `ACCESSIBLE=1`, the variable Huh's accessible forms already read. In that mode:
  no alt screen; plain line-by-line prompts (Huh's `WithAccessible(true)`); status
  printed as new lines ("deploying 2/3 pods", then "deployed"), never repainted in
  place; no spinners or timers; no colour-only meaning.
- **In the canvas UI**: the real cursor (`View.Cursor`) sits where input goes and stays
  there, and is nil when there's no input. Animate only while something is happening, and
  keep animations small (`motion.md`). Don't re-render a long history on every keystroke:
  cache finished items (`bubbletea-architecture.md`).
- **Offer the non-interactive path** (`--json`, plain output when stdout isn't a TTY).
  For many screen-reader users that is the accessible interface.
- **Test it**: run the accessible mode with a screen reader once per release (Orca,
  VoiceOver, NVDA in Windows Terminal), and keep a golden test of its plain-text
  transcript.

## Copy

Terminal copy is read at speed by people mid-task. Every word earns its cells.

- **Labels**: nouns, sentence case (or the brand's case), no colons. "Version", not
  "Version:".
- **Hints**: what is valid and what happens, in subtle, always visible. "tag or commit;
  latest is v2.4.1 (2 min ago)".
- **Errors**: what failed + against what + likely cause + what to do. "Can't reach
  prod-eu (timeout after 10s). Check the VPN, then press r to retry." Never "An error
  occurred", never a stack trace in the UI (log it, and offer `ctrl+o` to view it).
- **Empty states**: what is missing, why, in this context, and the one action. "No
  incidents declared. checkout-api has been failing for 12 min. n declare incident".
- **Spinners**: present participle + object + progress. "Deploying checkout-api 2/3
  pods". Never "Loading...", "Please wait", or "Processing".
- **Confirmations**: only when a judgement is left to the user (§ Confirm By
  Consequence). The question names what they are judging: the object and what changes
  ("Roll back checkout-api?", rung 3: it ships to production), or the signal that made
  it risky ("2 checks failing: lint, e2e. Merge anyway?"). The buttons are verbs
  ("Roll back" / "Cancel"), never "OK" / "Yes".
- **Success**: past tense + object + the fact that matters. "Rolled back checkout-api
  to v2.3.9 · 3/3 pods healthy". No "Successfully".
- **Voice** comes from the brand read: Crush's `Yep!` / `Nope` and rotating
  placeholders are a voice. A calm health product says less, more softly. A blunt
  industrial tool says it in caps and stops.
- **Numbers**: tabular and aligned, with units (`705ms`, `4.81%`, `1.2k`), relative times
  for recent things ("12m ago") and absolute for logs (`14:02:41`).
