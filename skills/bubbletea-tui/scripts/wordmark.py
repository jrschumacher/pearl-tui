#!/usr/bin/env python3
"""wordmark.py — rasterise a brand name into terminal cells, from a real font.

  wordmark.py "crush" --style half  --rows 4                 solid half-block letters (▀▄█)
  wordmark.py "crush" --style shade --rows 5                 coverage → █▓▒ (three tones, no ░)
  wordmark.py "crush" --style ascii --rows 6 --set "@%*"     ASCII fill, the fallback (and a voice)
  wordmark.py "CRUSH" --style tracked                        T R A C K E D  (one row, the quiet option)
      [--font path.ttf | --font bold|regular|condensed|mono|serif] [--upper] [--track N]
      [--go varName] [--json] [--ansi out.ansi --c1 "#hex" --c2 "#hex" --c3 "#hex"]

Prints the rows (every row exactly the same width), then with --go a Go `[]string`
literal, with --json a JSON array (paste into theme.json "wordmark.rows"), and with --ansi
a coloured preview you can render with cellshot. Rules (references/cell-art.md):
shade needs ≥5 rows (4 smears), half needs ≥4 rows for lowercase with ascenders, never
the figlet "standard" look, keep the wordmark quieter than the mark.

Needs Pillow + numpy. Finds DejaVu/Liberation/Noto fonts on Linux and macOS; pass --font
with a .ttf/.otf path to use the brand's own typeface (the best option when you have it).
"""
import argparse
import json
import os
import sys

import numpy as np
from PIL import Image, ImageDraw, ImageFont

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from cellshot import find_font  # noqa: E402

FAMILIES = {
    "bold": ["DejaVuSans-Bold.ttf", "LiberationSans-Bold.ttf", "NotoSans-Bold.ttf", "Arial Bold.ttf", "Helvetica.ttc"],
    "regular": ["DejaVuSans.ttf", "LiberationSans-Regular.ttf", "NotoSans-Regular.ttf", "Arial.ttf", "Helvetica.ttc"],
    "condensed": ["DejaVuSansCondensed-Bold.ttf", "LiberationSansNarrow-Bold.ttf", "NotoSans-CondensedBold.ttf"],
    "mono": ["DejaVuSansMono-Bold.ttf", "LiberationMono-Bold.ttf", "JetBrainsMono-Bold.ttf", "Menlo.ttc"],
    "serif": ["DejaVuSerif-Bold.ttf", "LiberationSerif-Bold.ttf", "NotoSerif-Bold.ttf", "Georgia Bold.ttf"],
}


def load_font(spec, size):
    path = spec if os.path.exists(spec) else find_font(FAMILIES.get(spec, [spec]))
    if not path and spec in FAMILIES and spec != "bold":
        print(f"wordmark: no {spec} font installed, using bold", file=sys.stderr)
        path = find_font(FAMILIES["bold"])
    if not path:
        sys.exit(f"wordmark: font {spec!r} not found; pass --font /path/to/font.ttf")
    return ImageFont.truetype(path, size)


def raster(text, rows_px, font, track=0):
    """Draw text so the ink spans rows_px samples vertically; return float coverage array."""
    scale = 8
    f = load_font(font, int(rows_px * scale * 1.1))
    chars = list(text)
    widths = [f.getbbox(c)[2] - f.getbbox(c)[0] if c.strip() else f.getlength(" ") for c in chars]
    W = int(sum(f.getlength(c) for c in chars) + track * scale * len(chars) + 16 * scale)
    img = Image.new("L", (W, int(rows_px * scale * 2)), 0)
    d = ImageDraw.Draw(img)
    x = 8 * scale
    for c in chars:
        d.text((x, rows_px * scale * 0.3), c, font=f, fill=255)
        x += f.getlength(c) + track * scale
    a = np.array(img)
    ys, xs = np.where(a > 100)
    if len(ys) == 0:
        sys.exit("wordmark: nothing rendered")
    a = a[ys.min():ys.max() + 1, xs.min():xs.max() + 1]
    return a, widths


def to_cells(a, rows, samples_per_row):
    """Box-filter the ink to (rows*samples_per_row) x cols, where a cell is 1 wide, 2 tall."""
    H = rows * samples_per_row
    px_per_sample = a.shape[0] / H
    # a cell is twice as tall as wide: cols = width / (cell height / 2)
    cell_h = px_per_sample * samples_per_row
    cols = max(1, int(round(a.shape[1] / (cell_h / 2))))
    small = np.array(Image.fromarray(a).resize((cols, H), Image.BOX)) / 255.0
    return small


def style_half(text, rows, font, track):
    a, _ = raster(text, rows * 2, font, track)
    s = to_cells(a, rows, 2)
    on = s >= 0.45
    out = []
    for r in range(rows):
        t, b = on[2 * r], on[2 * r + 1]
        out.append("".join("█" if x and y else "▀" if x else "▄" if y else " " for x, y in zip(t, b)))
    return out


def style_fill(text, rows, font, track, glyphs, th=(0.70, 0.38, 0.14)):
    a, _ = raster(text, rows, font, track)
    s = to_cells(a, rows, 1)
    out = []
    for r in s:
        out.append("".join(glyphs[0] if c >= th[0] else glyphs[1] if c >= th[1] else glyphs[2] if c >= th[2] else " " for c in r))
    return out


def trim(rows):
    rows = [r.rstrip() for r in rows]
    w = max(len(r) for r in rows)
    lead = min((len(r) - len(r.lstrip()) for r in rows if r.strip()), default=0)
    return [r[lead:].ljust(w - lead) for r in rows]


def sgr(h):
    h = h.lstrip("#")
    return f"\x1b[38;2;{int(h[0:2], 16)};{int(h[2:4], 16)};{int(h[4:6], 16)}m"


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("text")
    ap.add_argument("--style", default="half", choices=["half", "shade", "ascii", "tracked"])
    ap.add_argument("--rows", type=int, default=0)
    ap.add_argument("--font", default="bold")
    ap.add_argument("--set", default="")
    ap.add_argument("--upper", action="store_true")
    ap.add_argument("--track", type=float, default=0, help="extra letter spacing, in cells")
    ap.add_argument("--go", default="")
    ap.add_argument("--json", action="store_true")
    ap.add_argument("--ansi", default="")
    ap.add_argument("--c1", default="#E8E6F0", help="█ / solid tone")
    ap.add_argument("--c2", default="#8B7CFF", help="▓ tone")
    ap.add_argument("--c3", default="#55506A", help="▒ tone")
    a = ap.parse_args()
    text = a.text.upper() if a.upper else a.text
    if a.style == "tracked":
        rows = [" ".join(text)]
    elif a.style == "half":
        rows = trim(style_half(text, a.rows or 4, a.font, a.track))
    elif a.style == "shade":
        if a.rows and a.rows < 5:
            print("wordmark: warning: shade under 5 rows smears letters together", file=sys.stderr)
        rows = trim(style_fill(text, a.rows or 5, a.font, a.track, a.set or "█▓▒"))
    else:
        rows = trim(style_fill(text, a.rows or 6, a.font, a.track, a.set or "#+."))
    w = len(rows[0])
    print("\n".join(rows))
    print(f"-- {w}×{len(rows)} cells", file=sys.stderr)
    if a.json:
        print(json.dumps(rows, ensure_ascii=False))
    if a.go:
        print(f"var {a.go} = []string{{")
        for r in rows:
            print(f"\t{json.dumps(r, ensure_ascii=False)},")
        print("}")
    if a.ansi:
        tone = {"█": a.c1, "▀": a.c1, "▄": a.c1, "▓": a.c2, "▒": a.c3}
        if a.style == "ascii":
            g = a.set or "#+."
            tone = {g[0]: a.c1, g[1]: a.c2, g[2]: a.c3}
        with open(a.ansi, "w") as f:
            for r in rows:
                f.write("".join((sgr(tone[c]) + c + "\x1b[0m") if c in tone else (sgr(a.c1) + c + "\x1b[0m" if c != " " else " ") for c in r) + "\n")
        print(f"wrote {a.ansi}", file=sys.stderr)


if __name__ == "__main__":
    main()
