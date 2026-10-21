# Round 1

Four briefs (spoofify, hairdnd, bisque, coinslot), each built once with the skill
(v1.0.0-rc.1 plus the fidelity changes of #89) and once without it, on Claude Code. Ten
brand pairs, the anchor and one repeat, graded by one person. `decode.py`:

| pair | screen | refined | brand |
|---|---|---|---|
| s01 | spoofify, now playing | −1 | 0 |
| s02 | spoofify, search | −2 | −2 |
| b01 | bisque, runs | +2 | 0 |
| b02 | bisque, a failed run | +1 | −2 |
| h01 | hairdnd, den search | −2 | −1 |
| h02 | hairdnd, booking review | −2 | 0 |
| h03 | hairdnd, den search, light | −2 | 0 |
| c01 | coinslot, the floor | +2 | −2 |
| c02 | coinslot, a cabinet | +2 | −2 |
| c03 | coinslot, the floor at 120×40 | +1 | 0 |
| **total** | | **5 wins, 5 losses** | **0 wins, 5 losses, 5 ties** |

+ favours the skill's build. Each skill build's own critic (the no-subagent fallback)
scored it SHIP 21–22/24.

## Checks

- **Anchor**: +1 toward the known-good screen ("consistent colors is easier to read").
  Passed, narrowly.
- **Repeat**: h01 again with its sides swapped went from −2 to +1 on refinement. One
  repeat can't say how noisy the grades are; round 2 carries three.

## What the grader said

- **hairdnd's skill build, den search**: "mixes a list that impacts data above and below.
  Genuinely hard on the eyes", and, asked what: "felt like if I changed row it would
  change content above and below". Its brand move put the selected den's card above the
  list, with that den's reviews below it.
- **Brand fit**: asked what made the bare builds more on-brand, "skill felt generic".
  Shown the four skill builds together: three read as one house style, but "it feels
  like the best way to present information given the limited options. We have the right
  things highlighted and prioritized and the right things muted without eliminating the
  brand." The shared structure is good information design, not the defect.

## What changed

- `brand.md` § Move A Region: a move never splits what follows the selection across both
  sides of the list; the critic's layout item checks it (`verify.md`). Assertion in each
  `taste-*` eval.
- The brand-fit gap has no named cause yet. Round 2's page needs a note whenever the
  grade leans on brand, so the next fix comes from the grader's words.
- Not changed: tightening `diversity` on the axes a main screen shows. The kit's own moss
  and lumen differ from ink on two or three of them, quiet on purpose, and coinslot
  differed on five and still read as generic; the count doesn't measure what the grader
  saw.
- From the builds' own reports, to verify: every non-error kit toast is green; the
  `muted-upper` section title doesn't uppercase; the gradient wordmark lands on red at 16
  colours; `tuilint spec` reads a spinner in the footer's status as a key.
