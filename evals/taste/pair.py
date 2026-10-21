#!/usr/bin/env python3
"""Add a blind pair to a taste round: two renders of one screen, sides shuffled.

Usage:
  python3 pair.py ROUND_DIR ID BRAND SCREEN WORDS NEVER SKILL_PNG BARE_PNG [ask ...]
  python3 pair.py ROUND_DIR --repeat ID OF
  python3 pair.py ROUND_DIR --pick ID BRAND SCREEN WORDS NEVER NAME=PNG NAME=PNG ...

ROUND_DIR holds round.json (what the page shows) and key.json (which side is which,
never published). The first form copies the two renders to ROUND_DIR/ID-a.png and
ID-b.png in a random order and records the order in key.json; ask defaults to
"refined brand". The second shows pair OF again as ID with its sides swapped, to
measure how consistent the grader is. The third shows several renders of one brand
(splash styles, say) as A, B, C… in a random order and asks which fits the brand best
and which is most refined; key.json records which NAME each letter was. Publish
ROUND_DIR's round.json and renders with the page (README.md); keep key.json out of
the published files.
"""

import json
import os
import random
import shutil
import sys


def load(path, empty):
    return json.load(open(path)) if os.path.exists(path) else empty


def save(path, data):
    with open(path, "w") as f:
        json.dump(data, f, indent=1)
        f.write("\n")


def main(argv):
    if len(argv) < 4:
        sys.exit(__doc__)
    rd = argv[1]
    rj, kj = os.path.join(rd, "round.json"), os.path.join(rd, "key.json")
    rnd = load(rj, {"round": int(os.path.basename(os.path.abspath(rd))), "items": []})
    rnd.setdefault("title", f"Round {rnd['round']}")
    key = load(kj, {})
    if argv[2] == "--pick":
        pid, brand, screen, words, never = argv[3:8]
        opts = [a.split("=", 1) for a in argv[8:]]
        random.shuffle(opts)
        options, names = [], {}
        for tag, (name, src) in zip("ABCDEFGH", opts):
            ext = os.path.splitext(src)[1]
            shutil.copy(src, os.path.join(rd, f"{pid}-{tag.lower()}{ext}"))
            options.append({"tag": tag, "src": f"{pid}-{tag.lower()}{ext}"})
            names[tag] = name
        item = {"id": pid, "kind": "pick", "brand": brand, "screen": screen, "words": words,
                "never": never, "options": options, "ask": ["brand", "refined"]}
        key[pid] = {"kind": "pick", "options": names}
    elif argv[2] == "--repeat":
        pid, of = argv[3], argv[4]
        src = next(i for i in rnd["items"] if i["id"] == of)
        shutil.copy(os.path.join(rd, src["b"]), os.path.join(rd, f"{pid}-a.png"))
        shutil.copy(os.path.join(rd, src["a"]), os.path.join(rd, f"{pid}-b.png"))
        item = dict(src, id=pid, a=f"{pid}-a.png", b=f"{pid}-b.png")
        key[pid] = {"kind": "repeat", "of": of, "a": key[of]["b"], "b": key[of]["a"]}
    else:
        if len(argv) < 9:
            sys.exit(__doc__)
        pid, brand, screen, words, never, skill, bare = argv[2:9]
        sides = [("skill", skill), ("bare", bare)]
        random.shuffle(sides)
        for tag, (_, src) in zip("ab", sides):
            shutil.copy(src, os.path.join(rd, f"{pid}-{tag}.png"))
        item = {"id": pid, "brand": brand, "screen": screen, "words": words, "never": never,
                "a": f"{pid}-a.png", "b": f"{pid}-b.png", "ask": argv[9:] or ["refined", "brand"]}
        key[pid] = {"kind": "skill-vs-bare", "a": sides[0][0], "b": sides[1][0]}
    rnd["items"] = [i for i in rnd["items"] if i["id"] != pid] + [item]
    save(rj, rnd)
    save(kj, key)
    print(pid, "added")


if __name__ == "__main__":
    main(sys.argv)
