#!/usr/bin/env python3
"""Write the known-bad anchor: the showcase's services screen with the anti-patterns put back.

Usage: python3 make_bad.py OUT.ansi

Same rows of data as the kit's services frame at 80x24, drawn the way a model draws a
TUI without the skill: a rounded box around every region (box soup), Charm's default
pink and purple everywhere, a rainbow of column colours, emoji for status, bold on
everything, a centred title with rockets, and a footer of equal-weight key: action pairs.
"""

import sys

PINK, PURPLE, CYAN, YEL, GRN, RED, WHITE, GREY = (
    "#FF5FD2", "#7D56F4", "#00D7FF", "#FFD700", "#5FFF5F", "#FF4040", "#FFFFFF", "#9E9E9E")
BG = "#0d1724"
W, H = 80, 24

grid = [[(" ", WHITE, False)] * W for _ in range(H)]


def put(x, y, s, fg=WHITE, bold=True):
    for ch in s:
        if x >= W:
            break
        grid[y][x] = (ch, fg, bold)
        x += 1


def box(x, y, w, h, fg):
    put(x, y, "╭" + "─" * (w - 2) + "╮", fg)
    for r in range(y + 1, y + h - 1):
        put(x, r, "│", fg)
        put(x + w - 1, r, "│", fg)
    put(x, y + h - 1, "╰" + "─" * (w - 2) + "╯", fg)


box(0, 0, W, 3, PURPLE)
put(26, 1, "** HALYARD DASHBOARD **", PINK)
box(0, 3, W, 3, PINK)
put(2, 4, "[ Services ]  Deploys  Incidents  Assistant", PURPLE)
box(0, 6, W, 12, PURPLE)
cols = [(2, "STATUS", RED), (10, "SERVICE", PINK), (40, "VERSION", CYAN),
        (51, "ERRORS", YEL), (61, "DEPLOYED", GRN)]
for x, name, c in cols:
    put(x, 7, name, c)
rows = [("FAIL", "checkout-api", "v2.4.0", "4.81%", "12m ago"),
        ("OK", "inventory-sync", "v1.19.2", "0.02%", "3d ago"),
        ("DEPLOY", "search-indexer", "v3.0.1", "0.10%", "now"),
        ("OK", "auth-gateway", "v5.2.7", "0.00%", "8d ago"),
        ("WARN", "notifications-worker-eu-west", "v0.9.14", "0.92%", "1d ago"),
        ("OK", "ledger", "v7.1.0", "0.01%", "21d ago"),
        ("OK", "image-resizer", "v2.0.3", "0.05%", "5d ago"),
        ("???", "recommendations", "v4.4.4", "-", "34d ago")]
stat = {"FAIL": RED, "OK": GRN, "DEPLOY": CYAN, "WARN": YEL, "???": GREY}
for i, (s, name, ver, err, dep) in enumerate(rows):
    y = 8 + i
    put(2, y, s, stat[s])
    put(10, y, name, PINK if i == 0 else WHITE)
    put(40, y, ver, CYAN)
    put(51, y, err, YEL)
    put(61, y, dep, GRN)
box(0, 18, W, 3, PINK)
put(2, 19, "!!! ALERT: checkout-api 502s 4.8% failing since 14:02 !!!", RED)
box(0, 21, W, 3, PURPLE)
put(2, 22, "up/down: select - d: deploy - r: rollback - ?: help - q: quit", PINK)


def sgr(hexc, base):
    r, g, b = (int(hexc[i:i + 2], 16) for i in (1, 3, 5))
    return f"{base};2;{r};{g};{b}"


out = []
for row in grid:
    line, last = [], None
    for ch, fg, bold in row:
        if (fg, bold) != last:
            line.append("\x1b[0;" + sgr(fg, 38) + ";" + sgr(BG, 48) + (";1" if bold else "") + "m")
            last = (fg, bold)
        line.append(ch)
    out.append("".join(line) + "\x1b[0m")
open(sys.argv[1], "w", encoding="utf-8").write("\n".join(out) + "\n")
