# The user's answers

I'm not at the keyboard. Read this before asking anything, take each answer below as
given, and for any question it doesn't answer take the default you would have proposed.
Record what you took as a default under `## Assumed` in TUI.md.

## Round 1, design

- Who uses it, where, for how long: players at home for an evening, the terminal full
  screen; browsing for a few minutes between games. Dark terminals, often large
  (120×40 and up), but it must work at 80×24.
- The brand in three words: bright, generous, showy. It must never feel like a casino:
  no reels, no flashing credits, no gold.
- Assets: the README's hexes are real; the logo is the coin mid-fall into a slot, no
  file. Draw it as cell art.
- Typeface: none. Reference: none. There is no screen to match and no real app to
  imitate; the look is this brand's own.

## Round 2, architecture

- Screens: the arcade floor (the main screen: the cabinets as a list or a grid, your
  call, each with its name, genre, your best score and the house record), a cabinet's
  detail (its high-score table, how to play, the controls), and launch (later: the
  coin-drop animation and the cabinet's name for the ~1 s it loads, then back to the
  floor, since the games are faked).
- First slice: the floor and a cabinet's detail, every state of both; launch later.
- Forms: none.
- Slow work: loading a cabinet takes about a second; scores sync in the background.
- Smallest terminal: 80×24.

## Anything else

Take the default and say so.
