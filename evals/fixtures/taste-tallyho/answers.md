# The user's answers

I'm not at the keyboard. Read this before asking anything, take each answer below as
given, and for any question it doesn't answer take the default you would have proposed.
Record what you took as a default under `## Assumed` in TUI.md.

## Round 1, design
- Who uses it: freelance designers and developers, all day; they start and stop timers
  many times a day and invoice weekly or monthly. Dark terminals.
- The brand in three words: brisk, honest, tidy. Never a corporate timesheet.
- Assets: the README's hexes are real; the logo is the tally, no file.
- Typeface: none. Reference: none. There is no screen to match and no real app to
  imitate; the look is this brand's own.

## Round 2, architecture
- Screens: the week (the main screen: the week's entries by day, the timer you can start
  or stop on a project) and an invoice draft for one client (the hours and amounts for a
  period, then export to PDF; exporting can be redone, nothing is sent).
- First slice: the week and the invoice draft, every state of both.
- Slow work: none worth a spinner except the export (1 to 2 seconds).
- Smallest terminal: 80×24.

## Anything else
Take the default and say so.
