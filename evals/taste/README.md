# Taste rounds

The evals in `evals.json` check what an assertion can say. Taste can't be written down
ahead of time, so a taste round asks a person: the same brief built with the skill and
without it, shown blind, graded pairwise. The grades say whether the skill makes screens
more refined and more like their brand, and the notes become the next assertions.

## A round

1. **Briefs.** `evals/fixtures/taste-*` (README.md and answers.md each, init's format).
   Review a new brief with an adversarial subagent before building from it: real-brand
   proximity, palette clashes with state colours, answers that leave the slice open.
2. **Builds.** For each brief, two cold builds in their own folders, run in parallel as
   background agents:
   - *skill*: the pearl skill, `/pearl init` then craft the first slice answers.md names,
     the harness writing every state to `frames/` at 80×24 and 120×40;
   - *bare*: the same brief and the same pinned stack, no skill and no pearl tools, with a
     flag that writes the same frames.

   Give each build its own scratch folder in its prompt, or both arms write to a shared
   one and can read each other's work. Background agents end when the turn that started
   them is interrupted: check their folders before waiting on them.
3. **Pairs.** Render the same state of each arm with `cellshot` on the brand's background
   (`--bg`), one screen per pair, then `pair.py ROUND_DIR ID …` for each. Add:
   - the anchor, `x01`: the kit showcase's `services-80x24` (halyard) against
     `make_bad_anchor.py`'s copy with the anti-patterns put back (`tuilint frames` fails
     it); grading the bad one above the good one flags the round;
   - repeats, `pair.py ROUND_DIR --repeat z0N OF`: earlier pairs again with sides swapped,
     at least three, to measure how much a grade moves on a second look.
4. **Grade.** Publish `page/index.html` as an Artifact with `capabilities: {db: {}}`, the
   round's `round.json` and PNGs under `rounds/N/`, and `rounds/index.json` naming the
   current round. Never publish `key.json`. Each pair asks which is more refined and which
   feels more like the brand, on a five-step scale, and a lean on brand needs a note.
5. **Decode.** Read the `grades` collection back, save it as `rounds/N/grades.json`
   (`{item: {refined, brand}}`), and run `decode.py rounds/N`. Write `rounds/N/RESULTS.md`:
   the table, the checks, what the notes said, and the change each finding led to.
6. **Fix.** Each finding goes through the loop in AGENTS.md: an assertion first, then the
   smallest fix. Rebuild only the skill arm next round; the bare builds are the control.
