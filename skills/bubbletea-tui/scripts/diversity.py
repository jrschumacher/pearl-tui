#!/usr/bin/env python3
"""diversity.py — define "different", or every brand picks the same different.

  diversity.py themes/*.json [--max-shared 9] [--max-popular 0.5] [--baseline ink,charm --min-distance 8]

Tabulates every theme's visible choices (expressions + signature glyphs) and fails when:
  - two themes agree on more than --max-shared of the counted axes (one design, two colours)
  - one value is used by more than --max-popular of the themes (a house style, not a trait);
    two-valued axes are exempt (one of two values always has a majority)
  - a theme sits closer than --min-distance axes to a baseline (the default ink theme, the
    Crush study): it is the default with a new palette, which is exactly the slop it replaces

Dead axes (choices nothing renders) are shown as "-" and never count: headerRule under a
bar/stamp/minimal header, wordmark under a banner/stamp header.

It is a floor, not a target. A plain brand stays plain and spends its differences on the
two or three moves it owns; never move an axis just to pass. Learned over three rounds of
nine brands: parallel agents given a bigger vocabulary converge on the NEWEST options, so
after a first pass the orchestrator writes a seating plan (each contested value goes to
the brand whose product actually has it) and lets each brand finish in its lane.
"""
import argparse
import itertools
import json
import sys

AXES = [  # (group, key, default)
    ("x", "header", "rule"), ("x", "wordmark", "gradient"), ("x", "sectionTitle", "accent"),
    ("x", "titleCase", "sentence"), ("x", "selection", "bar"), ("x", "button", "pill"), ("x", "tag", "filled"),
    ("x", "dialog", "rounded"), ("x", "dialogTitle", "rule"), ("x", "tabs", "underline"), ("x", "footer", "keys"),
    ("x", "keys", "inline"), ("x", "density", "airy"), ("x", "backdrop", "none"), ("x", "scrim", "dim"),
    ("x", "paintBg", False),
    ("g", "headerRule", ""), ("g", "rule", "─"), ("g", "focusBar", "▌"), ("g", "cursor", "›"), ("g", "done", "✓"),
]
TWO_VALUED = {"titleCase", "density", "backdrop", "scrim", "paintBg", "keys"}


def choices(path):
    t = json.load(open(path))
    x, g = t.get("expressions", {}), t.get("glyphs", {})
    out = {}
    for grp, k, d in AXES:
        v = (x if grp == "x" else g).get(k, d)
        out[k] = str(v)
    if out["header"] in ("bar", "stamp", "minimal"):
        out["headerRule"] = "-"
    if out["header"] in ("banner", "stamp"):
        out["wordmark"] = "-"
    return t.get("name", path), out


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("themes", nargs="+")
    ap.add_argument("--max-shared", type=int, default=9)
    ap.add_argument("--max-popular", type=float, default=0.5)
    ap.add_argument("--baseline", default="ink,charm")
    ap.add_argument("--min-distance", type=int, default=8)
    a = ap.parse_args()
    rows = [choices(p) for p in a.themes]
    keys = [k for _, k, _ in AXES]
    names = [n for n, _ in rows]
    w = max(len(n) for n in names) + 2
    print("axis".ljust(13) + "".join(n[:10].ljust(12) for n in names))
    for k in keys:
        print(k.ljust(13) + "".join(r[k][:10].ljust(12) for _, r in rows))
    fails = []
    base = set(a.baseline.split(","))
    for (n1, r1), (n2, r2) in itertools.combinations(rows, 2):
        shared = [k for k in keys if r1[k] == r2[k] and r1[k] != "-"]
        if n1 in base or n2 in base:  # baselines are judged by --min-distance
            continue
        if len(shared) > a.max_shared:
            fails.append(f"{n1} and {n2} share {len(shared)} axes (> {a.max_shared}): {', '.join(shared)}")
    others = [(n, r) for n, r in rows if n not in base]
    for n, r in others:
        for bn, br in rows:
            if bn in base:
                dist = sum(1 for k in keys if r[k] != br[k] and r[k] != "-" and br[k] != "-")
                if dist < a.min_distance:
                    fails.append(f"{n} differs from baseline {bn} on only {dist} axes (< {a.min_distance}): a default with a new palette")
    if len(others) >= 3:
        for k in keys:
            if k in TWO_VALUED:
                continue
            vals = [r[k] for _, r in others if r[k] != "-"]
            for v in set(vals):
                if vals.count(v) / len(others) > a.max_popular and len(vals) >= 3:
                    fails.append(f"{k}={v} used by {vals.count(v)}/{len(others)} brand themes (> {int(a.max_popular * 100)}%): a house style")
    print()
    for f in fails:
        print("FAIL " + f)
    print("OK" if not fails else f"{len(fails)} problem(s)")
    return 1 if fails else 0


if __name__ == "__main__":
    sys.exit(main())
