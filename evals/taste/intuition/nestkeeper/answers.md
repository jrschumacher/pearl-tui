# The user's answers

I'm not at the keyboard. Read this before asking anything, take each answer below as
given, and for any question it doesn't answer take the default you would have proposed.
Record what you took as a default under `## Assumed` in TUI.md.

## Round 1, design
- Who uses it: the person who runs the household's server, a few times a month, over
  SSH, for five to fifteen minutes. Dark terminals.
- The brand in three words: orderly, reassuring, exact. Never a registry editor.
- Assets: the README's hexes are real; the logo is the nest, no file.
- Typeface: none. Reference: none. There is no screen to match and no real app to
  imitate; the look is this brand's own.

## Round 2, architecture
- Screens: settings (the main screen: about 60 settings in six sections; find one by
  name; change values, which stay pending until applied) and review & apply (every
  pending change, old value and new, then apply; some changes restart a service, one
  (the admin port) can lock you out if wrong).
- First slice: settings and review & apply, every state of both.
- Applying takes 3 to 10 seconds and can fail on one change while others succeed.
- Smallest terminal: 80×24.

## Anything else
Take the default and say so.
