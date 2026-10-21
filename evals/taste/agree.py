#!/usr/bin/env python3
"""How often a critic agrees with the person who graded a taste round.

Usage: python3 agree.py CRITIC.json [CRITIC.json ...] -- ROUNDS_DIR

CRITIC.json is a list of {"id": "r<N>-<item>", "refined": -2..2, "brand": -2..2}, the
critic's answers on the same blind pairs (A and B as the page showed them). ROUNDS_DIR
holds N/grades.json for each round. For each question this prints, over the pairs both
graded: the share where the two lean the same way (both A, both B, or both "same"), the
share within one step of each other, the share that lean opposite ways, and the
Pearson correlation of the two scores. The critic is calibrated when it leans the same
way as the person on new pairs, not on the ones it was tuned on.
"""

import json
import os
import sys


def sign(v):
    return (v > 0) - (v < 0)


def pearson(xs, ys):
    n = len(xs)
    if n < 2:
        return float("nan")
    mx, my = sum(xs) / n, sum(ys) / n
    sxy = sum((x - mx) * (y - my) for x, y in zip(xs, ys))
    sxx = sum((x - mx) ** 2 for x in xs)
    syy = sum((y - my) ** 2 for y in ys)
    return sxy / (sxx * syy) ** 0.5 if sxx and syy else float("nan")


def main(argv):
    if "--" not in argv:
        sys.exit(__doc__)
    cut = argv.index("--")
    files, rounds = argv[1:cut], argv[cut + 1]
    critic = {}
    for f in files:
        for row in json.load(open(f)):
            critic[row["id"]] = row
    for q in ("refined", "brand"):
        person, guess, rows = [], [], []
        for cid, row in sorted(critic.items()):
            rnd, item = cid.split("-", 1)
            path = os.path.join(rounds, rnd.lstrip("r"), "grades.json")
            g = json.load(open(path)).get(item, {}) if os.path.exists(path) else {}
            if q in g and q in row:
                person.append(g[q])
                guess.append(row[q])
                rows.append(f"{cid}: person {g[q]:+d}, critic {row[q]:+d}")
        n = len(person)
        if not n:
            print(f"{q}: no pairs graded by both")
            continue
        same = sum(sign(a) == sign(b) for a, b in zip(person, guess))
        near = sum(abs(a - b) <= 1 for a, b in zip(person, guess))
        opposite = sum(sign(a) * sign(b) < 0 for a, b in zip(person, guess))
        print(f"{q}: {n} pairs; same lean {same}/{n}, within one step {near}/{n}, "
              f"opposite {opposite}/{n}, r = {pearson(person, guess):+.2f}")
        if "-v" in files or os.environ.get("AGREE_VERBOSE"):
            print("  " + "\n  ".join(rows))


if __name__ == "__main__":
    main(sys.argv)
