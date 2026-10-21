# The user's answers

I'm not at the keyboard. Read this before asking anything, take each answer below as
given, and for any question it doesn't answer take the default you would have proposed.
Record what you took as a default under `## Assumed` in TUI.md.

## Round 1, design
- Who uses it: writers, researchers and engineers who live in the terminal; open for
  hours, in a full-screen tab, often at 120×40 or larger but it must work at 80×24.
  Dark terminals.
- The brand in three words: cozy, unhurried, private. Never an inbox that shouts.
- Assets: the README's hexes are real; the logo is the burrow door, no file.
- Typeface: none. Reference: none. There is no screen to match and no real app to
  imitate; the look is this brand's own.

## Round 2, architecture
- Screens: the inbox (the main screen: messages, and reading the selected one; long
  messages and wide lines in them) and compose (a reply or a new message, with to,
  subject and body; send can be undone for 5 seconds).
- Mail syncs in the background every minute; a sync can fail while offline.
- First slice: the inbox and compose, every state of both. Mouse on.
- Smallest terminal: 80×24.

## Anything else
Take the default and say so.
