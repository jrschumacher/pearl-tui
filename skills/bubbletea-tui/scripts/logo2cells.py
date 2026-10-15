#!/usr/bin/env python3
"""logo2cells.py — sample a raster logo onto a terminal cell grid (a starting point to hand-fix).

  logo2cells.py logo.png --cols 40 --mode half  [--colors 2] [--palette "#1f5aa6,#6a9fdc"]
  logo2cells.py logo.png --cols 40 --mode shade                coverage → ▒▓█ edge vocabulary
  logo2cells.py logo.png --cols 24 --mode quad                 2×2 quadrants, two colours per cell
      [--bg "#0b1220"] [--out name] [--go varName] [--invert]

Export SVG logos to PNG at ≥ 800 px first (any tool: rsvg-convert, inkscape, a browser).
Transparent pixels (or pixels close to the top-left corner colour when there is no alpha)
are background. Ink colours are the brand's own: --palette if you pass it (measured from
the brand guide), else k-means on the logo's opaque pixels (--colors k).

Writes  <out>.txt   glyph rows, every row exactly --cols cells
        <out>.ansi  truecolor preview → cellshot.py <out>.ansi <out>.png --grid --cell 16x32
        <out>.go    (with --go) []string glyph rows + []string colour-index rows (0 = bg)
and prints a lint: rows of unequal width, orphan cells (inked, no inked neighbour), and
asymmetry when the logo itself is left/right symmetric.

Modes (references/cell-art.md):
  half   two vertical samples per cell (▀ ▄ █, fg = top, bg = bottom). Crisp, 2× vertical
         resolution, needs 256/truecolor. Best for flat multi-colour logos.
  shade  one coverage sample per cell → ▒ ▓ █ (never ░). Soft edges, one ink per cell.
         Best for single-colour marks with curves; needs ~40 cols for likeness.
  quad   2×2 samples per cell → quadrant glyphs. Sharp corners; noisy on curves.
"""
import argparse
import os
import sys

import numpy as np
from PIL import Image


def hex2rgb(h):
    h = h.strip().lstrip("#")
    return np.array([int(h[i:i + 2], 16) for i in (0, 2, 4)], dtype=float)


def kmeans(px, k, iters=20):
    rng = np.random.default_rng(1)
    cent = px[rng.choice(len(px), size=min(k, len(px)), replace=False)]
    for _ in range(iters):
        d = ((px[:, None, :] - cent[None, :, :]) ** 2).sum(-1)
        lab = d.argmin(1)
        for j in range(len(cent)):
            if (lab == j).any():
                cent[j] = px[lab == j].mean(0)
    return cent


def load(path, invert):
    im = Image.open(path).convert("RGBA")
    a = np.array(im).astype(float)
    rgb, alpha = a[..., :3], a[..., 3] / 255
    if (alpha < 0.99).mean() < 0.01:  # no real alpha: key out the corner colour
        corner = rgb[0, 0]
        dist = np.sqrt(((rgb - corner) ** 2).sum(-1))
        alpha = np.clip((dist - 30) / 60, 0, 1)
    if invert:
        alpha = 1 - alpha
    ys, xs = np.where(alpha > 0.5)
    y0, y1, x0, x1 = ys.min(), ys.max() + 1, xs.min(), xs.max() + 1
    return rgb[y0:y1, x0:x1], alpha[y0:y1, x0:x1]


def grid(rgb, alpha, cols, sy, sx):
    """Supersample: returns per-sample (alpha, rgb) at (rows*sy, cols*sx)."""
    h, w = alpha.shape
    cell_w = w / cols
    rows = max(1, int(round(h / (cell_w * 2))))
    H, W = rows * sy, cols * sx
    A = np.array(Image.fromarray((alpha * 255).astype(np.uint8)).resize((W, H), Image.BOX)) / 255
    C = np.stack([np.array(Image.fromarray(rgb[..., k].astype(np.uint8)).resize((W, H), Image.BOX)) for k in range(3)], -1).astype(float)
    return rows, A, C


def nearest(c, pal):
    return int(((pal - c) ** 2).sum(-1).argmin())


def sgr(c, bg=False):
    r, g, b = (int(v) for v in c)
    return f"\x1b[{48 if bg else 38};2;{r};{g};{b}m"


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("image")
    ap.add_argument("--cols", type=int, default=40)
    ap.add_argument("--mode", default="half", choices=["half", "shade", "quad"])
    ap.add_argument("--colors", type=int, default=2)
    ap.add_argument("--palette", default="")
    ap.add_argument("--bg", default="#16161a")
    ap.add_argument("--out", default="")
    ap.add_argument("--go", default="")
    ap.add_argument("--invert", action="store_true")
    a = ap.parse_args()
    out = a.out or os.path.splitext(os.path.basename(a.image))[0] + f"-{a.cols}"
    rgb, alpha = load(a.image, a.invert)
    ink = rgb[alpha > 0.5].reshape(-1, 3)
    pal = np.array([hex2rgb(h) for h in a.palette.split(",")]) if a.palette else kmeans(ink, a.colors)
    bgc = hex2rgb(a.bg)
    glyphs, idx, ansi = [], [], []

    if a.mode == "half":
        rows, A, C = grid(rgb, alpha, a.cols, 2, 1)
        for r in range(rows):
            g, ci, line = "", "", ""
            for c in range(a.cols):
                top, bot = A[2 * r, c] >= 0.5, A[2 * r + 1, c] >= 0.5
                kt = nearest(C[2 * r, c], pal) + 1 if top else 0
                kb = nearest(C[2 * r + 1, c], pal) + 1 if bot else 0
                if not top and not bot:
                    g += " "; ci += "0"; line += " "
                elif top and bot and kt == kb:
                    g += "█"; ci += str(kt); line += sgr(pal[kt - 1]) + "█\x1b[0m"
                elif top and bot:
                    g += "▀"; ci += str(kt); line += sgr(pal[kt - 1]) + sgr(pal[kb - 1], True) + "▀\x1b[0m"
                elif top:
                    g += "▀"; ci += str(kt); line += sgr(pal[kt - 1]) + "▀\x1b[0m"
                else:
                    g += "▄"; ci += str(kb); line += sgr(pal[kb - 1]) + "▄\x1b[0m"
            glyphs.append(g); idx.append(ci); ansi.append(line)
    elif a.mode == "shade":
        rows, A, C = grid(rgb, alpha, a.cols, 8, 4)
        for r in range(rows):
            g, ci, line = "", "", ""
            for c in range(a.cols):
                blk = A[r * 8:(r + 1) * 8, c * 4:(c + 1) * 4]
                cov = blk.mean()
                ch = "█" if cov >= 0.85 else "▓" if cov >= 0.6 else "▒" if cov >= 0.3 else " "
                if ch == " ":
                    g += " "; ci += "0"; line += " "
                    continue
                m = blk > 0.5
                col = C[r * 8:(r + 1) * 8, c * 4:(c + 1) * 4][m].mean(0) if m.any() else C[r * 8, c * 4]
                k = nearest(col, pal) + 1
                g += ch; ci += str(k); line += sgr(pal[k - 1]) + ch + "\x1b[0m"
            glyphs.append(g); idx.append(ci); ansi.append(line)
    else:  # quad
        Q = {(1, 0, 0, 0): "▘", (0, 1, 0, 0): "▝", (0, 0, 1, 0): "▖", (0, 0, 0, 1): "▗", (1, 1, 0, 0): "▀",
             (0, 0, 1, 1): "▄", (1, 0, 1, 0): "▌", (0, 1, 0, 1): "▐", (0, 1, 1, 0): "▞", (1, 0, 0, 1): "▚",
             (1, 1, 1, 0): "▛", (1, 1, 0, 1): "▜", (1, 0, 1, 1): "▙", (0, 1, 1, 1): "▟", (1, 1, 1, 1): "█"}
        rows, A, C = grid(rgb, alpha, a.cols, 2, 2)
        for r in range(rows):
            g, ci, line = "", "", ""
            for c in range(a.cols):
                q = tuple(int(A[2 * r + dy, 2 * c + dx] >= 0.5) for dy in (0, 1) for dx in (0, 1))
                if not any(q):
                    g += " "; ci += "0"; line += " "
                    continue
                pts = [C[2 * r + dy, 2 * c + dx] for dy in (0, 1) for dx in (0, 1)]
                col = np.mean([p for p, on in zip(pts, q) if on], 0)
                k = nearest(col, pal) + 1
                ch = Q[q]
                g += ch; ci += str(k); line += sgr(pal[k - 1]) + ch + "\x1b[0m"
            glyphs.append(g); idx.append(ci); ansi.append(line)

    # lint
    warn = []
    widths = {len(g) for g in glyphs}
    if len(widths) > 1:
        warn.append(f"unequal row widths {sorted(widths)}")
    orphans = []
    for r, g in enumerate(glyphs):
        for c, ch in enumerate(g):
            if ch == " ":
                continue
            nb = [glyphs[rr][cc] for rr in (r - 1, r, r + 1) for cc in (c - 1, c, c + 1)
                  if (rr, cc) != (r, c) and 0 <= rr < len(glyphs) and 0 <= cc < len(g)]
            if all(x == " " for x in nb):
                orphans.append((r, c))
    if orphans:
        warn.append(f"{len(orphans)} orphan cells (inked, no inked neighbour), first at row {orphans[0][0]} col {orphans[0][1]}")
    src_sym = np.abs(alpha - alpha[:, ::-1]).mean() < 0.02
    if src_sym:
        mirror = {"▌": "▐", "▐": "▌", "▘": "▝", "▝": "▘", "▖": "▗", "▗": "▖", "▛": "▜", "▜": "▛", "▙": "▟", "▟": "▙", "▞": "▚", "▚": "▞"}
        asym = sum(1 for g in glyphs for c in range(len(g)) if g[c] != mirror.get(g[-1 - c], g[-1 - c]))
        if asym:
            warn.append(f"source is symmetric but {asym} cells are not: quantise one half and mirror it")
    open(out + ".txt", "w").write("\n".join(glyphs) + "\n")
    open(out + ".ansi", "w").write("\n".join(ansi) + "\n")
    print("\n".join(ansi))
    print(f"-- {a.cols}×{len(glyphs)} cells, palette " + ", ".join("#%02X%02X%02X" % tuple(int(v) for v in p) for p in pal), file=sys.stderr)
    for w in warn:
        print("lint: " + w, file=sys.stderr)
    if a.go:
        with open(out + ".go", "w") as f:
            f.write(f"package ui\n\n// {a.go}Glyphs is generated by logo2cells.py from {os.path.basename(a.image)} ({a.mode}, {a.cols} cols).\n")
            f.write(f"var {a.go}Glyphs = []string{{\n" + "".join(f"\t{g!r},\n".replace("'", '"') for g in glyphs) + "}\n\n")
            f.write(f"// {a.go}Tones: per cell, 0 = background, n = palette[n-1].\nvar {a.go}Tones = []string{{\n" + "".join(f'\t"{t}",\n' for t in idx) + "}\n")
        print(f"wrote {out}.go", file=sys.stderr)
    print(f"wrote {out}.txt {out}.ansi", file=sys.stderr)


if __name__ == "__main__":
    main()
