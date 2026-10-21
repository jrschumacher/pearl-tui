#!/usr/bin/env python3
"""Write the signin-reference eval's frames: the reference and the candidate.

Usage: python3 make_frames.py OUTDIR

Then, from this directory, with OUTDIR=out:

  python3 make_frames.py out
  cp out/frames/*.ansi frames/
  cellshot out/reference/wait-1-80x24.ansi reference/wait-1.png
  cellshot out/reference/wait-2-80x24.ansi reference/wait-2.png
  cellshot out/reference/wait-*-80x24.ansi --gif reference/wait.gif --ms 500

Writes OUTDIR/reference/wait-{1,2}-80x24.ansi (the design lead's screen to match, two
animation frames) and OUTDIR/frames/wait{,-2}-80x24.ansi and wait-120x40.ansi (the
"restyled" app's harness output). The eval ships the reference as cellshot's PNGs and
GIF, not as .ansi, because the user in the case supplied images.

The candidate is deliberately attractive and off-reference: the same mark, status,
code, URL and keys, but the mark centred above the status instead of left of it, solid
and static instead of tonal with a moving arc, the status moved into the header, the
body copy a step dimmer, and the footer one weight with cancel first. Its elapsed time
reads "12s" where the reference reads "0:12": the deviation BRIEF.md approves.

Standard library only. Nothing here is shipped to the agent under evaluation.
"""

import os
import sys

BG = "#14161b"
PANEL = "#1d2129"
FG = "#e3e6ec"
MUTED = "#a3abb9"
SUBTLE = "#6b7385"
PRIMARY = "#4f7bd9"
DIM = "#2c3a5c"
MID = "#3d5a9a"
LINE = "#2a2f3a"

MARK = [
    "      ▄▄████▄▄      ",
    "   ▄██▀▀    ▀▀██▄   ",
    "  ██   ▄▄██▄▄   ██  ",
    " ██  ▄█▀    ▀█▄  ██ ",
    " ██  ██  ◆   ██  ██ ",
    " ██  ▀█▄    ▄█▀  ██ ",
    "  ██   ▀▀██▀▀   ██  ",
    "   ▀██▄▄    ▄▄██▀   ",
    "      ▀▀████▀▀      ",
]


def ring(x, y):
    """0 for the outer ring, 1 for the inner, 2 for the centre."""
    if MARK[y][x] == "◆":
        return 2
    return 1 if 4 <= x <= 15 and 2 <= y <= 6 else 0


class Grid:
    def __init__(self, w, h):
        self.w, self.h = w, h
        self.cells = [[[" ", FG, BG, False] for _ in range(w)] for _ in range(h)]

    def put(self, x, y, text, fg=FG, bg=None, bold=False):
        for i, ch in enumerate(text):
            if 0 <= x + i < self.w:
                c = self.cells[y][x + i]
                c[0], c[1], c[3] = ch, fg, bold
                if bg:
                    c[2] = bg

    def fill(self, y, bg):
        for c in self.cells[y]:
            c[2] = bg

    def ansi(self):
        def sgr(hexc, base):
            r, g, b = (int(hexc[i : i + 2], 16) for i in (1, 3, 5))
            return f"{base};2;{r};{g};{b}"

        rows = []
        for row in self.cells:
            out, last = [], None
            for ch, fg, bg, bold in row:
                style = (fg, bg, bold)
                if style != last:
                    codes = ["0", sgr(fg, 38), sgr(bg, 48)] + (["1"] if bold else [])
                    out.append("\x1b[" + ";".join(codes) + "m")
                    last = style
                out.append(ch)
            rows.append("".join(out) + "\x1b[0m")
        return "\n".join(rows) + "\n"


def header(g, right):
    g.fill(0, PANEL)
    g.put(1, 0, "◆", PRIMARY, PANEL)
    g.put(3, 0, "kestrel", FG, PANEL, bold=True)
    g.put(11, 0, "setup", MUTED, PANEL)
    width = sum(len(t) for t, _ in right)
    x = g.w - 1 - width
    for text, fg in right:
        g.put(x, 0, text, fg, PANEL, bold=(fg == FG))
        x += len(text)


def stepper(g, y):
    x = 2
    for glyph, name, fg, gfg, bold in [
        ("◆", "Region", MUTED, PRIMARY, False),
        ("◆", "Project", MUTED, PRIMARY, False),
        ("▲", "Sign-in", FG, PRIMARY, True),
        ("△", "Notifications", SUBTLE, SUBTLE, False),
        ("△", "Review", SUBTLE, SUBTLE, False),
    ]:
        g.put(x, y, glyph, gfg)
        g.put(x + 2, y, name, fg, bold=bold)
        x += len(name) + 5


def mark(g, x0, y0, lit, solid):
    """lit: the outer-ring rows the moving arc brightens (reference only)."""
    for y, row in enumerate(MARK):
        for x, ch in enumerate(row):
            if ch == " ":
                continue
            r = ring(x, y)
            if solid:
                fg = PRIMARY
            elif r == 2:
                fg = PRIMARY
            elif r == 1:
                fg = MID
            else:
                fg = PRIMARY if y in lit else DIM
            g.put(x0 + x, y0 + y, ch, fg)


def key(g, x, y, k, label, kfg=FG, lfg=MUTED, box=True):
    if box:
        g.put(x, y, f" {k} ", kfg, PANEL, bold=True)
        x += len(k) + 2
    else:
        g.put(x, y, k, kfg, bold=True)
        x += len(k)
    g.put(x + 1, y, label, lfg)
    return x + 1 + len(label)


SPIN = ["⠋", "⠙"]
URL = "https://kestrel.example/device"


def reference(frame):
    g = Grid(80, 24)
    header(g, [("3/5", MUTED), (" · ", SUBTLE), ("Sign-in", MUTED)])
    stepper(g, 2)
    g.put(2, 4, "─" * 76, LINE)
    mark(g, 4, 7, lit={0, 1, 2} if frame == 0 else {6, 7, 8}, solid=False)
    x = 30
    g.put(x, 8, SPIN[frame], PRIMARY)
    g.put(x + 2, 8, "Waiting for your browser", FG, bold=True)
    g.put(x, 10, "We opened the sign-in page in your browser.", MUTED)
    g.put(x, 11, "Enter this code there:", MUTED)
    g.put(x, 12, " WDJB-MJHT ", FG, PANEL, bold=True)
    g.put(x, 14, "Not there? Open", MUTED)
    g.put(x + 16, 14, URL, PRIMARY)
    g.put(x, 15, f"waiting 0:{12 + frame:02d}", SUBTLE)
    g.put(2, 22, "─" * 76, LINE)
    k = key(g, 1, 23, "o", "open url")
    key(g, k + 2, 23, "c", "copy code")
    key(g, 80 - 1 - len("esc cancel"), 23, "esc", "cancel", SUBTLE, SUBTLE, box=False)
    return g


def candidate(frame, w, h):
    g = Grid(w, h)
    header(
        g,
        [(SPIN[frame], PRIMARY), (" Waiting for your browser", FG), ("  ·  3/5", SUBTLE)],
    )
    stepper(g, 2)
    cx = (w - 20) // 2
    top = 5 if h == 24 else 9
    mark(g, cx, top, lit=set(), solid=True)
    lines = [
        ("We opened the sign-in page in your browser.", SUBTLE),
        ("Enter this code there:", SUBTLE),
        (None, None),
        ("Not there? Open " + URL, SUBTLE),
        (f"waiting {12 + frame}s", SUBTLE),
    ]
    y = top + len(MARK) + 1
    for text, fg in lines:
        if text is None:
            code = " WDJB-MJHT "
            g.put((w - len(code)) // 2, y, code, FG, PANEL, bold=True)
        else:
            g.put((w - len(text)) // 2, y, text, fg)
        y += 1
    x = key(g, 1, h - 1, "esc", "cancel", MUTED, MUTED, box=False)
    g.put(x + 1, h - 1, "·", SUBTLE)
    x = key(g, x + 3, h - 1, "o", "open url", MUTED, MUTED, box=False)
    g.put(x + 1, h - 1, "·", SUBTLE)
    key(g, x + 3, h - 1, "c", "copy code", MUTED, MUTED, box=False)
    return g


def main():
    if len(sys.argv) != 2:
        sys.exit(__doc__)
    out = sys.argv[1]
    os.makedirs(os.path.join(out, "reference"), exist_ok=True)
    os.makedirs(os.path.join(out, "frames"), exist_ok=True)

    def write(path, g):
        with open(os.path.join(out, path), "w", encoding="utf-8") as f:
            f.write(g.ansi())

    for i in (0, 1):
        write(f"reference/wait-{i + 1}-80x24.ansi", reference(i))
    write("frames/wait-80x24.ansi", candidate(0, 80, 24))
    write("frames/wait-2-80x24.ansi", candidate(1, 80, 24))
    write("frames/wait-120x40.ansi", candidate(0, 120, 40))


if __name__ == "__main__":
    main()
