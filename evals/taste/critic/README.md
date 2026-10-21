# Calibrating the critic to the grader

The skill's critic scored every taste-round build SHIP 21–22/24, including the ones the
grader rated clearly worse. This folder measures how far a critic is from the grader and
holds the standard that closes the gap.

- `pairs.json`: rounds 1 and 2's twenty brand pairs as the page showed them (A/B, brand,
  words), with no key and no grades. A critic grades these blind.
- `baseline-*.json`: a critic using `verify.md` § The Critic as written.
- `standard.md`: the grader's standard, written from their round 2 notes and answers as
  principles, never as verdicts on these pairs.
- `calibrated-*.json`: a critic given `standard.md` instead.

`python3 ../agree.py baseline-1.json baseline-2.json -- ../rounds`:

| critic | refined, same lean | refined r | brand, same lean | brand r |
|---|---|---|---|---|
| baseline | 12/20 | +0.27 | 6/20 | +0.03 |
| calibrated | 16/20 | +0.75 | 8/20 | +0.21 |
| calibrated, round 2 only | 7/10 | +0.62 | 8/10 | +0.70 |
| calibrated, round 1 only | 9/10 | +0.84 | 0/10 | −0.56 |

The calibrated critic is tuned on these pairs, so these numbers flatter it. It is
judged on round 3: it predicts each pair before the grader sees the page. On brand it
matches round 2 and inverts round 1. The grader's brand standard sharpened between them:
the same bare coinslot floor was the more on-brand in round 1 for its large coin logo,
and in round 2 that logo was "way too large".
