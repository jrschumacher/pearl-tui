# Round 2

The same four briefs. The skill arm rebuilt with round 1's fix (what follows the
selection stays in one region); the bare arm is round 1's builds, unchanged. Ten brand
pairs, four splash picks (each brand's splash in four styles, built with the skill), the
anchor, three pair repeats and one pick repeat, graded by one person. `decode.py`:

| pair | screen | refined | brand |
|---|---|---|---|
| s01 | spoofify, now playing | −1 | −1 |
| s02 | spoofify, search | +2 | −2 |
| b01 | bisque, runs | +2 | +1 |
| b02 | bisque, a failed run | +1 | −2 |
| h01 | hairdnd, den search | +1 | −2 |
| h02 | hairdnd, booking review | −2 | −2 |
| h03 | hairdnd, den search, light | +1 | −1 |
| c01 | coinslot, the floor | +2 | +2 |
| c02 | coinslot, a cabinet | +2 | +2 |
| c03 | coinslot, the floor at 120×40 | +2 | +2 |
| **total** | | **8 wins, 2 losses** (round 1: 5, 5) | **4 wins, 6 losses** (round 1: 0, 5) |

Each round's skill builds are fresh runs, so part of the swing is build-to-build
variation, not the fix alone.

## Checks

- **Anchor**: +2 toward the known-good screen.
- **Repeats**: b01 and c02 held (+2/+2, +2/+2 on refinement; brand within one step).
  s02 flipped on refinement (+2, then −2), the pair whose note called itself "torn… a
  split". The hairdnd splash pick, reshuffled, got the same two picks.

## Splash styles

| brand | fits the brand | most refined |
|---|---|---|
| spoofify | wordmark | field |
| bisque | field | field |
| hairdnd | intro (twice) | wordmark (twice) |
| coinslot | field | wordmark |

Quiet was never picked, and was called "great for a very small resolution" three times:
it is the small-terminal fallback of the other styles, not a style. On hairdnd's field:
"I don't think the placement of those little squiggles is great."

## What the grader said, and asked about it

1. **Live data beside its list.** bisque's bare build put the log right of the steps: "as
   an engineer this is so much easier to process… in A my eye would be jumping between
   the middle and the bottom." Asked whether the skill was too specific: "if you
   consider the data type in user experience… the journey, one would have always put
   the live data to the right."
2. **Background status hides.** "Scores synced 2 min ago. No one cares." Asked: anything
   background is "encapsulated in an icon and maybe there's a hidden hotkey that you can
   derive through the help… If it's something that should be blocking, then yes, it
   should be visible."
3. **The mark small on working screens.** coinslot's bare header logo was "way too large";
   the skill's header "just right". "People are there to do their job. They're not there
   to look at your brand." A player differs: spoofify's name in the bottom right with a
   small gradient, no logo, "was really refined", because the listener wants the
   artwork, not the logo.
4. **Money gets a receipt.** hairdnd's bare review highlighted the total and set the
   numbers right, with the card apart from the booking: "specific for receipt style
   payments… maybe there's a router: what type of product is this… even on the screens…
   hairdnd, which is you're looking for rentals, and then at the very end it turns into
   a financial thing. We shouldn't conflate the whole app experience to a specific flow."
5. **General footer words**: "esc back", not a shortcut that duplicates enter
   ("y book & pay" read as slop). Confirmed.
6. **Context that carries the product's purpose persists**: the now-playing strip under
   spoofify's search, "because the goal of the application is music. Maybe that's
   something we're missing… a user journey… what is the purpose of this thing, and
   what's the purpose of this particular screen as opposed to other scenes."
7. **The default follows the intent.** hairdnd's skill review focused Cancel on paying:
   "I'm trying to close a sale, I want it to fall on the affirmative… if I'm saying
   destroy this pod I'm gonna default to cancel… The reason why it was frustrating is
   because: does that lose all my data? What does cancel do, does it go back?"

## What changes

- A journey pass per screen (its job, its data, what it carries from the product's
  purpose), from which 1, 2, 3 and 6 follow, with short pattern sheets loaded per flow
  type (checkout, monitoring, media, browse, configure), never per app.
- Confirm by consequence gains the intent: the flow's own purpose defaults to the act,
  destruction to the safe choice, and no button says a bare "Cancel".
- Splash: quiet becomes the fallback below the other styles' size floor.
