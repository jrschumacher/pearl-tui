#!/usr/bin/env python3
"""Decode a taste round's grades against its key: what the grader preferred, by arm.

Usage: python3 decode.py ROUND_DIR

Reads ROUND_DIR/key.json and ROUND_DIR/grades.json ({item: {refined, brand}}, read back
from the page's store). A grade runs from -2 (A, clearly) to +2 (B, clearly); this turns
each into a score toward the skill's build (+ favours the skill), checks the anchor
(+ favours the known-good screen) and each repeat against its original, and prints the
skill's wins, losses and ties per question.
"""

import json
import os
import sys


def toward(key, item, who, v):
    return v if key[item]["b"] == who else -v


def main(argv):
    if len(argv) != 2:
        sys.exit(__doc__)
    key = json.load(open(os.path.join(argv[1], "key.json")))
    grades = json.load(open(os.path.join(argv[1], "grades.json")))
    totals = {}
    for item, k in key.items():
        g = grades.get(item)
        if g is None:
            print(f"{item:5} not graded")
            continue
        if k["kind"] == "anchor":
            print(f"{item:5} anchor: {toward(key, item, 'good', g['refined']):+d} toward the known-good screen")
        elif k["kind"] == "repeat":
            first = grades.get(k["of"], {})
            for q in ("refined", "brand"):
                if q in g and q in first:
                    a, b = toward(key, k["of"], "skill", first[q]), toward(key, item, "skill", g[q])
                    print(f"{item:5} repeat of {k['of']}, {q}: {a:+d} then {b:+d}{'' if abs(a - b) <= 1 else '  (flipped)'}")
        else:
            row = []
            for q in ("refined", "brand"):
                if q in g:
                    v = toward(key, item, "skill", g[q])
                    totals.setdefault(q, []).append(v)
                    row.append(f"{q} {v:+d}")
            print(f"{item:5} " + "  ".join(row))
    for q, vs in totals.items():
        w, l = sum(v > 0 for v in vs), sum(v < 0 for v in vs)
        print(f"{q}: the skill wins {w}, loses {l}, ties {len(vs) - w - l}; mean {sum(vs) / len(vs):+.1f}")


if __name__ == "__main__":
    main(sys.argv)
