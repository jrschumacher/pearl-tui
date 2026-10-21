# The user's answers

I'm not at the keyboard. Read this before asking anything, take each answer below as
given, and for any question it doesn't answer take the default you would have proposed.
Record what you took as a default under `## Assumed` in TUI.md.

## Round 1, design

- Who uses it, where, for how long: engineers, right after a push, in a tmux pane beside
  the editor; one to five minutes while a run finishes, often over SSH. Dark terminals.
- The brand in three words: hot, terse, workmanlike. It must never be a traffic-light
  board: no column of red and green dots, one per run.
- Assets: the README's hex is real; the logo is the kiln's mouth with a flame, no file.
  Draw it as cell art.
- Colour: ember means firing (running) and nothing else. Focus is fg and weight
  (`focus: fg`). A failure is error red with its own glyph, never ember.
- Typeface: none. Reference: none. There is no screen to match and no real app to
  imitate; the look is this brand's own.

## Round 2, architecture

- Screens: runs (the main screen: recent runs with branch, commit, author, status and
  duration, the failing one easy to find) and a run's detail (its steps live, the
  failing step's log tail). A cancel confirmation: a cancelled run can't be resumed.
  Retry happens on its key with no confirm, since it re-runs only the failed steps.
- First slice: runs and the run's detail with its log tail, plus the cancel confirm;
  the full log pager later.
- Slow work: runs stream step updates every second or two; logs can be 20k lines. Fake
  the backend with fixtures.
- Smallest terminal: 80×24.

## Anything else

Take the default and say so.
