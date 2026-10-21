# The user's answers

I'm not at the keyboard. Read this before asking anything, take each answer below as
given, and for any question it doesn't answer take the default you would have proposed.
Record what you took as a default under `## Assumed` in TUI.md.

## Round 1, design

- Who uses it, where, for how long: developers and writers who keep music on while they
  work; open for hours, looked at for seconds. Dark terminals.
- The brand in three words: cheeky, nocturnal, deadpan. It must never feel like a
  spectrum-analyser light show: no bouncing bars.
- Assets: the README's hexes are real; the logo is the vinyl with a moustache, no file.
  Draw it as cell art.
- Typeface: none. Reference: none. There is no screen to match and no real app to
  imitate; the look is this brand's own.

## Round 2, architecture

- Screens: now playing (the main screen: the spoof, its made-up band, the made-up
  original it sends up, progress, what's next) and search (results you can play or
  queue). A rate-this-spoof dialog exists, later.
- First slice: now playing and search, every state of both; the rate dialog later.
- Forms: none in the slice.
- Slow work: search takes up to two seconds; progress ticks every second while a track
  plays and stops when paused. Fake the audio: no sound, a fake clock.
- Smallest terminal: 60×18, a side pane; design at 80×24, and 120×40 when it has a tab
  of its own. Dark only.

## Anything else

Take the default and say so.
