# The user's answers

I'm not at the keyboard. Read this before asking anything, take each answer below as
given, and for any question it doesn't answer take the default you would have proposed.
Record what you took as a default under `## Assumed` in TUI.md.

## Round 1, design

- Who uses it, where, for how long: the party's organiser, planning a weekend session,
  at home on a laptop; ten to twenty minutes, comparing a handful of dens. Dark and
  light terminals both common: design both.
- The brand in three words: warm, shaggy, adventurous. It must never feel like a hotel
  booking form.
- Assets: the README's hexes are real; the logo is the d20 asleep on a rug, no file.
  Draw it as cell art.
- Typeface: none. Reference: none. There is no screen to match and no real app to
  imitate; the look is this brand's own.

## Round 2, architecture

- Screens: search results (the main screen: dens with town, seats, a shed-safe rating,
  price a night, and the dates you searched), booking, and a den's detail (later).
- Booking: the searched dates carried in; party size and the campaign you're running;
  then a review step that is the confirmation; pay with the card on file, no card
  fields. Paying can't be taken back from the TUI. Booking can fail when the den was
  taken meanwhile.
- First slice: search results and booking, every state of both; the den's detail later.
- Forms: the default (Huh, themed).
- Slow work: search and booking each take one to three seconds. Fake the backend with
  fixtures.
- Smallest terminal: 80×24.

## Anything else

Take the default and say so.
