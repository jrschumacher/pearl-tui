#!/usr/bin/env python3
"""termshot.py — render ANSI terminal frames to PNG/GIF, honestly, so you can SEE a TUI.

You cannot look at a terminal. This is your eyes. It draws every cell the way a modern
terminal (Ghostty, Kitty, WezTerm, iTerm2 with built-in box drawing) does: box drawing,
block elements, shades, quadrants, sextants, braille and powerline separators are drawn
geometrically (so they join and fill like a real terminal, never as font tofu), text uses
a real monospace font, and wide characters take two cells.

  termshot.py frame.ansi out.png                    one frame
  termshot.py a.ansi b.ansi c.ansi --sheet out.png  contact sheet, labelled by filename
  termshot.py f*.ansi --gif out.gif --ms 80         animation from frames
  some-cmd | termshot.py - out.png                  from stdin

Options
  --cell WxH        pixel size of one cell (default 12x24; use 16x32 to inspect detail)
  --bg #rrggbb      terminal default background (default #16161a)   --light  = #fafafa/#222
  --fg #rrggbb      terminal default foreground (default #d8d8d8)
  --cols N --rows N expected size; pads the image and reports any mismatch (exit 1)
  --grid            faint cell grid (for cell-art review)
  --scale N         integer upscale after rendering (crisp)
  --label TEXT      caption bar above the image (sheets use file names)

Prints one line per input: size in cells, problems found (rows of unequal width, glyphs no
font could draw, emoji, non-SGR escape sequences). Exit status 1 if --cols/--rows mismatch.
Only dependency: Pillow.
"""
import os
import re
import sys
import unicodedata

from PIL import Image, ImageDraw, ImageFont

# ----------------------------------------------------------------------------- fonts
FONT_DIRS = ["/usr/share/fonts", "/usr/local/share/fonts", os.path.expanduser("~/.local/share/fonts"),
             os.path.expanduser("~/Library/Fonts"), "/Library/Fonts", "/System/Library/Fonts",
             "C:\\Windows\\Fonts"]
REGULAR = ["JetBrainsMono-Regular.ttf", "DejaVuSansMono.ttf", "Menlo.ttc", "LiberationMono-Regular.ttf",
           "consola.ttf", "FreeMono.ttf"]
BOLD = ["JetBrainsMono-Bold.ttf", "DejaVuSansMono-Bold.ttf", "Menlo.ttc", "LiberationMono-Bold.ttf",
        "consolab.ttf", "FreeMonoBold.ttf"]
ITALIC = ["JetBrainsMono-Italic.ttf", "DejaVuSansMono-Oblique.ttf", "LiberationMono-Italic.ttf"]
FALLBACK = ["DejaVuSans.ttf", "NotoSansSymbols2-Regular.ttf", "NotoSansSymbols-Regular.ttf",
            "NotoSansMath-Regular.ttf", "FreeSerif.ttf", "Symbola.ttf", "opens___.ttf",
            "NotoSansCJK-Regular.ttc", "fonts-japanese-gothic.ttf", "wqy-microhei.ttc", "Apple Symbols.ttf"]
EMOJI = ["NotoColorEmoji.ttf", "Apple Color Emoji.ttc", "seguiemj.ttf"]

_index = None


def find_font(names):
    global _index
    if _index is None:
        _index = {}
        for d in FONT_DIRS:
            for root, _, files in os.walk(d):
                for f in files:
                    _index.setdefault(f, os.path.join(root, f))
    for n in names:
        if n in _index:
            return _index[n]
    return None


class Fonts:
    def __init__(self, px):
        self.px = px
        size = int(px * 0.80)
        reg = find_font(REGULAR)
        if not reg:
            sys.exit("termshot: no monospace font found (install DejaVu Sans Mono or JetBrains Mono)")
        self.reg = ImageFont.truetype(reg, size)
        self.bold = ImageFont.truetype(find_font(BOLD) or reg, size)
        self.italic = ImageFont.truetype(find_font(ITALIC) or reg, size)
        self.fallbacks = []
        for n in FALLBACK:
            p = find_font([n])
            if p:
                try:
                    self.fallbacks.append(ImageFont.truetype(p, size))
                except OSError:
                    pass
        self.emoji_path = find_font(EMOJI)
        self._notdef = {}
        self._cache = {}

    def _missing(self, font, ch):
        key = id(font)
        if key not in self._notdef:
            m = font.getmask("\U0010FFFD")
            self._notdef[key] = (m.size, bytes(m))
        m = font.getmask(ch)
        if m.size == (0, 0) and not ch.isspace():
            return True
        return (m.size, bytes(m)) == self._notdef[key]

    def pick(self, ch, bold, italic):
        k = (ch, bold, italic)
        if k in self._cache:
            return self._cache[k]
        base = self.bold if bold else self.italic if italic else self.reg
        f = base
        if self._missing(base, ch):
            f = None
            for fb in self.fallbacks:
                if not self._missing(fb, ch):
                    f = fb
                    break
        self._cache[k] = f
        return f


# ----------------------------------------------------------------------------- parse
BASIC = [(0, 0, 0), (205, 49, 49), (13, 188, 121), (229, 229, 16), (36, 114, 200), (188, 63, 188),
         (17, 168, 205), (229, 229, 229), (102, 102, 102), (241, 76, 76), (35, 209, 139), (245, 245, 67),
         (59, 142, 234), (214, 112, 214), (41, 184, 219), (255, 255, 255)]


def xterm(n):
    if n < 16:
        return BASIC[n]
    if n < 232:
        n -= 16
        r, g, b = n // 36, (n // 6) % 6, n % 6
        return tuple(0 if v == 0 else 55 + v * 40 for v in (r, g, b))
    v = 8 + (n - 232) * 10
    return (v, v, v)


def cell_width(ch):
    if unicodedata.combining(ch) or ch in "\u200d\ufe0f\ufe0e":
        return 0
    if unicodedata.east_asian_width(ch) in ("W", "F"):
        return 2
    return 1


def is_emoji(ch):
    o = ord(ch)
    return 0x1F300 <= o <= 0x1FAFF or 0x2600 <= o <= 0x27BF and unicodedata.east_asian_width(ch) == "W"


class Cell:
    __slots__ = ("ch", "fg", "bg", "bold", "faint", "italic", "under", "strike", "rev", "wide")

    def __init__(self, ch, st, wide=False):
        self.ch = ch
        self.fg, self.bg, self.bold, self.faint, self.italic, self.under, self.strike, self.rev = st
        self.wide = wide


TOKEN = re.compile(r"\x1b\[([0-9;:?<>=]*)([@-~])|\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)|\x1b[PX^_][^\x1b]*\x1b\\|\x1b[@-Z\\-_]")


def parse(text):
    """-> (rows of Cell, list of warnings)"""
    rows, row, warns = [], [], set()
    st = [None, None, False, False, False, 0, False, False]  # fg bg bold faint italic under strike rev
    i, n = 0, len(text)
    while i < n:
        c = text[i]
        if c == "\x1b":
            m = TOKEN.match(text, i)
            if not m:
                i += 1
                continue
            if m.group(2) == "m":
                sgr(m.group(1), st)
            elif m.group(2) is not None:
                if m.group(2) not in "hl":  # mode set/reset (cursor show/hide) are harmless
                    warns.add(f"non-SGR CSI '{m.group(2)}' ignored (cursor movement? frame may not be a flat render)")
            i = m.end()
            continue
        i += 1
        if c == "\n":
            rows.append(row)
            row = []
            continue
        if c == "\r":
            continue
        if c == "\t":
            for _ in range(8 - len(row) % 8):
                row.append(Cell(" ", tuple(st)))
            continue
        w = cell_width(c)
        if w == 0:
            if row:
                row[-1].ch += c
                if c == "\ufe0f" and not row[-1].wide:  # VS16 makes the base emoji wide
                    row[-1].wide = True
                    row.append(Cell("", tuple(st)))
            continue
        if is_emoji(c):
            warns.add("emoji present (rendering varies by terminal and font; usually a design smell)")
        row.append(Cell(c, tuple(st), wide=(w == 2)))
        if w == 2:
            row.append(Cell("", tuple(st)))
    if row:
        rows.append(row)
    return rows, sorted(warns)


def sgr(params, st):
    codes = re.split(r"[;]", params) if params else ["0"]
    j = 0
    while j < len(codes):
        part = codes[j]
        if ":" in part:  # colon sub-params: 38:2::r:g:b, 4:3 (curly underline)
            sub = [s for s in part.split(":")]
            head = int(sub[0] or 0)
            if head in (38, 48, 58):
                vals = [int(s) for s in sub[1:] if s != ""]
                col = None
                if vals and vals[0] == 5 and len(vals) >= 2:
                    col = xterm(vals[1])
                elif vals and vals[0] == 2 and len(vals) >= 4:
                    col = tuple(vals[-3:])
                if head == 38:
                    st[0] = col
                elif head == 48:
                    st[1] = col
            elif head == 4:
                st[5] = int(sub[1] or 1) if len(sub) > 1 else 1
            j += 1
            continue
        c = int(part or 0)
        if c == 0:
            st[:] = [None, None, False, False, False, 0, False, False]
        elif c == 1:
            st[2] = True
        elif c == 2:
            st[3] = True
        elif c == 3:
            st[4] = True
        elif c == 4:
            st[5] = 1
        elif c == 21:
            st[5] = 2
        elif c == 7:
            st[7] = True
        elif c == 9:
            st[6] = True
        elif c == 22:
            st[2] = st[3] = False
        elif c == 23:
            st[4] = False
        elif c == 24:
            st[5] = 0
        elif c == 27:
            st[7] = False
        elif c == 29:
            st[6] = False
        elif 30 <= c <= 37:
            st[0] = BASIC[c - 30]
        elif 90 <= c <= 97:
            st[0] = BASIC[c - 90 + 8]
        elif 40 <= c <= 47:
            st[1] = BASIC[c - 40]
        elif 100 <= c <= 107:
            st[1] = BASIC[c - 100 + 8]
        elif c == 39:
            st[0] = None
        elif c == 49:
            st[1] = None
        elif c in (38, 48, 58) and j + 1 < len(codes):
            mode = int(codes[j + 1] or 0)
            col = None
            if mode == 5 and j + 2 < len(codes):
                col = xterm(int(codes[j + 2] or 0))
                j += 2
            elif mode == 2 and j + 4 < len(codes):
                col = tuple(int(x or 0) for x in codes[j + 2:j + 5])
                j += 4
            if c == 38:
                st[0] = col
            elif c == 48:
                st[1] = col
        j += 1


# ----------------------------------------------------------------------------- geometry
def mix(a, b, t):
    return tuple(int(round(a[k] + (b[k] - a[k]) * t)) for k in range(3))


# box drawing: (up, right, down, left) weights. 1 light, 2 heavy, 3 double
BOX = {}


def _box(chars, spec):
    for ch, s in zip(chars, spec):
        BOX[ch] = s


_box("─━│┃", [(0, 1, 0, 1), (0, 2, 0, 2), (1, 0, 1, 0), (2, 0, 2, 0)])
_box("┌┍┎┏┐┑┒┓└┕┖┗┘┙┚┛", [
    (0, 1, 1, 0), (0, 2, 1, 0), (0, 1, 2, 0), (0, 2, 2, 0), (0, 0, 1, 1), (0, 0, 1, 2), (0, 0, 2, 1), (0, 0, 2, 2),
    (1, 1, 0, 0), (1, 2, 0, 0), (2, 1, 0, 0), (2, 2, 0, 0), (1, 0, 0, 1), (1, 0, 0, 2), (2, 0, 0, 1), (2, 0, 0, 2)])
_box("├┝┞┟┠┡┢┣┤┥┦┧┨┩┪┫", [
    (1, 1, 1, 0), (1, 2, 1, 0), (2, 1, 1, 0), (1, 1, 2, 0), (2, 1, 2, 0), (2, 2, 1, 0), (1, 2, 2, 0), (2, 2, 2, 0),
    (1, 0, 1, 1), (1, 0, 1, 2), (2, 0, 1, 1), (1, 0, 2, 1), (2, 0, 2, 1), (2, 0, 1, 2), (1, 0, 2, 2), (2, 0, 2, 2)])
_box("┬┭┮┯┰┱┲┳┴┵┶┷┸┹┺┻", [
    (0, 1, 1, 1), (0, 1, 1, 2), (0, 2, 1, 1), (0, 2, 1, 2), (0, 1, 2, 1), (0, 1, 2, 2), (0, 2, 2, 1), (0, 2, 2, 2),
    (1, 1, 0, 1), (1, 1, 0, 2), (1, 2, 0, 1), (1, 2, 0, 2), (2, 1, 0, 1), (2, 1, 0, 2), (2, 2, 0, 1), (2, 2, 0, 2)])
_box("┼┽┾┿╀╁╂╃╄╅╆╇╈╉╊╋", [
    (1, 1, 1, 1), (1, 1, 1, 2), (1, 2, 1, 1), (1, 2, 1, 2), (2, 1, 1, 1), (1, 1, 2, 1), (2, 1, 2, 1), (2, 1, 1, 2),
    (2, 2, 1, 1), (1, 1, 2, 2), (1, 2, 2, 1), (2, 2, 1, 2), (1, 2, 2, 2), (2, 1, 2, 2), (2, 2, 2, 1), (2, 2, 2, 2)])
_box("═║╒╓╔╕╖╗╘╙╚╛╜╝╞╟╠╡╢╣╤╥╦╧╨╩╪╫╬", [
    (0, 3, 0, 3), (3, 0, 3, 0), (0, 3, 1, 0), (0, 1, 3, 0), (0, 3, 3, 0), (0, 0, 1, 3), (0, 0, 3, 1), (0, 0, 3, 3),
    (1, 3, 0, 0), (3, 1, 0, 0), (3, 3, 0, 0), (1, 0, 0, 3), (3, 0, 0, 1), (3, 0, 0, 3), (1, 3, 1, 0), (3, 1, 3, 0),
    (3, 3, 3, 0), (1, 0, 1, 3), (3, 0, 3, 1), (3, 0, 3, 3), (0, 3, 1, 3), (0, 1, 3, 1), (0, 3, 3, 3), (1, 3, 0, 3),
    (3, 1, 0, 1), (3, 3, 0, 3), (1, 3, 1, 3), (3, 1, 3, 1), (3, 3, 3, 3)])
_box("╴╵╶╷╸╹╺╻╼╽╾╿", [
    (0, 0, 0, 1), (1, 0, 0, 0), (0, 1, 0, 0), (0, 0, 1, 0), (0, 0, 0, 2), (2, 0, 0, 0), (0, 2, 0, 0), (0, 0, 2, 0),
    (0, 2, 0, 1), (1, 0, 2, 0), (0, 1, 0, 2), (2, 0, 1, 0)])
DASH = {"┄": (0, 3, 1), "┅": (0, 3, 2), "┆": (1, 3, 1), "┇": (1, 3, 2), "┈": (0, 4, 1), "┉": (0, 4, 2),
        "┊": (1, 4, 1), "┋": (1, 4, 2), "╌": (0, 2, 1), "╍": (0, 2, 2), "╎": (1, 2, 1), "╏": (1, 2, 2)}
ARC = {"╭": "dr", "╮": "dl", "╯": "ul", "╰": "ur"}
# block elements: list of (x0, y0, x1, y1) in eighths of the cell
BLOCK = {"█": [(0, 0, 8, 8)], "▀": [(0, 0, 8, 4)], "▄": [(0, 4, 8, 8)], "▌": [(0, 0, 4, 8)], "▐": [(4, 0, 8, 8)],
         "▔": [(0, 0, 8, 1)], "▕": [(7, 0, 8, 8)],
         "▖": [(0, 4, 4, 8)], "▗": [(4, 4, 8, 8)], "▘": [(0, 0, 4, 4)], "▝": [(4, 0, 8, 4)],
         "▙": [(0, 0, 4, 8), (4, 4, 8, 8)], "▚": [(0, 0, 4, 4), (4, 4, 8, 8)], "▛": [(0, 0, 8, 4), (0, 4, 4, 8)],
         "▜": [(0, 0, 8, 4), (4, 4, 8, 8)], "▞": [(4, 0, 8, 4), (0, 4, 4, 8)], "▟": [(4, 0, 8, 8), (0, 4, 4, 8)]}
for k, ch in enumerate("▁▂▃▄▅▆▇"):
    BLOCK[ch] = [(0, 7 - k, 8, 8)]
for k, ch in enumerate("▏▎▍▌▋▊▉"):
    BLOCK[ch] = [(0, 0, k + 1, 8)]
SHADE = {"░": 0.25, "▒": 0.5, "▓": 0.75}


def draw_box(d, x0, y0, cw, rh, ch, col, lw):
    u, r, dn, l = BOX[ch]
    mx, my = x0 + cw // 2, y0 + rh // 2
    x1, y1 = x0 + cw, y0 + rh
    gap = max(2, lw + 1)

    def seg(weight, pts_single, pts_double):
        if weight == 0:
            return
        if weight == 3:
            for p in pts_double:
                d.line(p, fill=col, width=lw)
        else:
            d.line(pts_single, fill=col, width=lw * (2 if weight == 2 else 1))

    # for doubles, run the two strokes to the centre lines of the crossing direction
    seg(u, [(mx, y0), (mx, my)], [[(mx - gap, y0), (mx - gap, my + (gap if l or r else 0))],
                                  [(mx + gap, y0), (mx + gap, my + (gap if l or r else 0))]])
    seg(dn, [(mx, my), (mx, y1)], [[(mx - gap, my - (gap if l or r else 0)), (mx - gap, y1)],
                                   [(mx + gap, my - (gap if l or r else 0)), (mx + gap, y1)]])
    seg(l, [(x0, my), (mx, my)], [[(x0, my - gap), (mx + (gap if u or dn else 0), my - gap)],
                                  [(x0, my + gap), (mx + (gap if u or dn else 0), my + gap)]])
    seg(r, [(mx, my), (x1, my)], [[(mx - (gap if u or dn else 0), my - gap), (x1, my - gap)],
                                  [(mx - (gap if u or dn else 0), my + gap), (x1, my + gap)]])


def draw_cell(img, d, fonts, cell, x0, y0, cw, rh, dfg, dbg, warn):
    fg = cell.fg or dfg
    bg = cell.bg
    if cell.rev:
        fg, bg = (bg or dbg), (cell.fg or dfg)
    w = cw * (2 if cell.wide else 1)
    if bg is not None:
        d.rectangle([x0, y0, x0 + w - 1, y0 + rh - 1], fill=bg)
    eff_bg = bg or dbg
    if cell.faint:
        fg = mix(eff_bg, fg, 0.55)
    ch = cell.ch
    lw = max(1, cw // 8)
    base = ch[:1]
    if base in ("", " "):
        pass
    elif base in BOX:
        draw_box(d, x0, y0, cw, rh, base, fg, lw)
    elif base in DASH:
        vert, nseg, weight = DASH[base]
        L = rh if vert else cw
        step = L / nseg
        for k in range(nseg):
            a, b = k * step + step * 0.15, k * step + step * 0.75
            if vert:
                d.line([(x0 + cw // 2, y0 + a), (x0 + cw // 2, y0 + b)], fill=fg, width=lw * weight)
            else:
                d.line([(x0 + a, y0 + rh // 2), (x0 + b, y0 + rh // 2)], fill=fg, width=lw * weight)
    elif base in ARC:
        mx, my = x0 + cw // 2, y0 + rh // 2
        rad = min(cw, rh) // 2
        kind = ARC[base]
        # straight parts to the cell edge, then a quarter circle around the corner
        if kind == "dr":
            box, ang = [mx, my, mx + 2 * rad, my + 2 * rad], (180, 270)
            d.line([(mx, my + rad), (mx, y0 + rh)], fill=fg, width=lw)
            d.line([(mx + rad, my), (x0 + cw, my)], fill=fg, width=lw)
        elif kind == "dl":
            box, ang = [mx - 2 * rad, my, mx, my + 2 * rad], (270, 360)
            d.line([(mx, my + rad), (mx, y0 + rh)], fill=fg, width=lw)
            d.line([(x0, my), (mx - rad, my)], fill=fg, width=lw)
        elif kind == "ul":
            box, ang = [mx - 2 * rad, my - 2 * rad, mx, my], (0, 90)
            d.line([(mx, y0), (mx, my - rad)], fill=fg, width=lw)
            d.line([(x0, my), (mx - rad, my)], fill=fg, width=lw)
        else:  # ur
            box, ang = [mx, my - 2 * rad, mx + 2 * rad, my], (90, 180)
            d.line([(mx, y0), (mx, my - rad)], fill=fg, width=lw)
            d.line([(mx + rad, my), (x0 + cw, my)], fill=fg, width=lw)
        d.arc(box, ang[0], ang[1], fill=fg, width=lw)
    elif base in "╱╲╳":
        if base in "╱╳":
            d.line([(x0, y0 + rh - 1), (x0 + cw - 1, y0)], fill=fg, width=lw)
        if base in "╲╳":
            d.line([(x0, y0), (x0 + cw - 1, y0 + rh - 1)], fill=fg, width=lw)
    elif base in BLOCK:
        for (a, b, c, e) in BLOCK[base]:
            d.rectangle([x0 + a * cw // 8, y0 + b * rh // 8, x0 + c * cw // 8 - 1, y0 + e * rh // 8 - 1], fill=fg)
    elif base in SHADE:
        d.rectangle([x0, y0, x0 + cw - 1, y0 + rh - 1], fill=mix(eff_bg, fg, SHADE[base]))
    elif 0x2800 <= ord(base) <= 0x28FF:
        bits = ord(base) - 0x2800
        dots = [(0, 0, 0x01), (0, 1, 0x02), (0, 2, 0x04), (1, 0, 0x08), (1, 1, 0x10), (1, 2, 0x20), (0, 3, 0x40), (1, 3, 0x80)]
        rr = max(1, cw // 6)
        for (cx, cy, bit) in dots:
            if bits & bit:
                px = x0 + (cx * 2 + 1) * cw // 4
                py = y0 + (cy * 2 + 1) * rh // 8
                d.ellipse([px - rr, py - rr, px + rr, py + rr], fill=fg)
    elif 0x1FB00 <= ord(base) <= 0x1FB3B:  # sextants 2x3
        k = ord(base) - 0x1FB00 + 1
        if k >= 21:
            k += 1
        if k >= 42:
            k += 1
        for bit in range(6):
            if k & (1 << bit):
                cx, cy = bit % 2, bit // 2
                d.rectangle([x0 + cx * cw // 2, y0 + cy * rh // 3, x0 + (cx + 1) * cw // 2 - 1, y0 + (cy + 1) * rh // 3 - 1], fill=fg)
    elif base in "\ue0b0\ue0b2\ue0b4\ue0b6":
        if base in "\ue0b0\ue0b4":
            d.polygon([(x0, y0), (x0 + cw - 1, y0 + rh // 2), (x0, y0 + rh - 1)], fill=fg)
        else:
            d.polygon([(x0 + cw - 1, y0), (x0, y0 + rh // 2), (x0 + cw - 1, y0 + rh - 1)], fill=fg)
    elif is_emoji(base) and fonts.emoji_path:
        try:
            ef = ImageFont.truetype(fonts.emoji_path, 109)
            tmp = Image.new("RGBA", (136, 128), (0, 0, 0, 0))
            ImageDraw.Draw(tmp).text((0, 0), ch, font=ef, embedded_color=True)
            tmp = tmp.crop(tmp.getbbox() or (0, 0, 1, 1))
            s = min(w / tmp.width, rh / tmp.height) * 0.9
            tmp = tmp.resize((max(1, int(tmp.width * s)), max(1, int(tmp.height * s))), Image.LANCZOS)
            img.paste(tmp, (x0 + (w - tmp.width) // 2, y0 + (rh - tmp.height) // 2), tmp)
        except Exception:
            warn.add(f"could not draw emoji {base!r}")
    else:
        f = fonts.pick(base, cell.bold, cell.italic)
        if f is None:
            warn.add(f"no font has {base!r} U+{ord(base):04X}: drawn as tofu (it may be tofu for users too)")
            d.rectangle([x0 + 1, y0 + 2, x0 + w - 2, y0 + rh - 3], outline=(255, 0, 255))
        else:
            bb = f.getbbox(ch)
            gw = bb[2] - bb[0]
            ox = x0 + max(0, (w - gw) // 2 - bb[0]) if gw > w else x0
            d.text((ox, y0 + int(rh * 0.08)), ch, font=f, fill=fg)
    if cell.under:
        uy = y0 + rh - max(2, rh // 10)
        if cell.under == 3:  # curly
            pts = [(x0 + k, uy + (1 if (k // 2) % 2 else -1)) for k in range(0, w, 1)]
            d.line(pts, fill=fg, width=1)
        else:
            d.line([(x0, uy), (x0 + w - 1, uy)], fill=fg, width=max(1, lw))
            if cell.under == 2:
                d.line([(x0, uy - 3), (x0 + w - 1, uy - 3)], fill=fg, width=1)
    if cell.strike:
        d.line([(x0, y0 + rh // 2), (x0 + w - 1, y0 + rh // 2)], fill=fg, width=max(1, lw))


def render(rows, cw, rh, dfg, dbg, cols=None, nrows=None, grid=False):
    width = max([sum(1 for _ in r) for r in rows] + [cols or 0, 1])
    height = max(len(rows), nrows or 0, 1)
    img = Image.new("RGB", (width * cw, height * rh), dbg)
    d = ImageDraw.Draw(img)
    fonts = Fonts(rh)
    warn = set()
    for y, row in enumerate(rows):
        for x, cell in enumerate(row):
            if cell.ch == "" and not cell.wide:  # trailing half of a wide glyph
                continue
            draw_cell(img, d, fonts, cell, x * cw, y * rh, cw, rh, dfg, dbg, warn)
    if grid:
        gcol = mix(dbg, (128, 128, 128), 0.18)
        for x in range(width + 1):
            d.line([(x * cw, 0), (x * cw, height * rh)], fill=gcol)
        for y in range(height + 1):
            d.line([(0, y * rh), (width * cw, y * rh)], fill=gcol)
    return img, warn


def label_bar(img, text, dbg):
    f = Fonts(22).reg
    bar = Image.new("RGB", (img.width, 30), mix(dbg, (255, 255, 255), 0.12))
    ImageDraw.Draw(bar).text((8, 5), text, font=f, fill=(235, 235, 235))
    out = Image.new("RGB", (img.width, img.height + 30), dbg)
    out.paste(bar, (0, 0))
    out.paste(img, (0, 30))
    return out


def hexcol(s):
    s = s.lstrip("#")
    return (int(s[0:2], 16), int(s[2:4], 16), int(s[4:6], 16))


def main(argv):
    opts = {"cell": "12x24", "bg": "#16161a", "fg": "#d8d8d8", "ms": "100", "scale": "1"}
    flags, files = set(), []
    i = 0
    while i < len(argv):
        a = argv[i]
        if a in ("--sheet", "--gif", "--cell", "--bg", "--fg", "--cols", "--rows", "--ms", "--scale", "--label"):
            opts[a[2:]] = argv[i + 1]
            i += 2
            continue
        if a in ("--grid", "--light"):
            flags.add(a[2:])
        elif a in ("-h", "--help"):
            print(__doc__)
            return 0
        else:
            files.append(a)
        i += 1
    if "light" in flags:
        opts["bg"], opts["fg"] = "#fafafa", "#222222"
    out = opts.get("sheet") or opts.get("gif")
    if not out:
        if len(files) != 2:
            print(__doc__)
            return 2
        files, out = files[:1], files[1]
    cw, rh = (int(v) for v in opts["cell"].split("x"))
    dbg, dfg = hexcol(opts["bg"]), hexcol(opts["fg"])
    cols = int(opts["cols"]) if "cols" in opts else None
    nrows = int(opts["rows"]) if "rows" in opts else None
    status = 0
    images = []
    for f in files:
        text = sys.stdin.read() if f == "-" else open(f, encoding="utf-8", errors="replace").read()
        if text.endswith("\n"):
            text = text[:-1]
        rows, warns = parse(text)
        widths = [len(r) for r in rows]
        problems = list(warns)
        if widths and len(set(widths)) > 1:
            bad = [k for k, w in enumerate(widths) if w != max(widths)]
            problems.append(f"rows of unequal width: {len(bad)} rows short of {max(widths)} (first: row {bad[0]} = {widths[bad[0]]})")
        if cols and widths and max(widths) != cols:
            problems.append(f"width {max(widths)} != expected {cols}")
            status = 1
        if nrows and len(rows) != nrows:
            problems.append(f"height {len(rows)} != expected {nrows}")
            status = 1
        img, w2 = render(rows, cw, rh, dfg, dbg, cols, nrows, "grid" in flags)
        problems += sorted(w2)
        sc = int(opts["scale"])
        if sc > 1:
            img = img.resize((img.width * sc, img.height * sc), Image.NEAREST)
        lab = opts.get("label") or (os.path.splitext(os.path.basename(f))[0] if opts.get("sheet") else None)
        if lab:
            img = label_bar(img, lab, dbg)
        images.append(img)
        size = f"{max(widths) if widths else 0}x{len(rows)}"
        print(f"{f}: {size} cells" + ("" if not problems else "\n  - " + "\n  - ".join(problems)))
    if opts.get("gif"):
        images[0].save(out, save_all=True, append_images=images[1:], duration=int(opts["ms"]), loop=0, disposal=2)
    elif opts.get("sheet"):
        per = 2 if len(images) > 1 else 1
        if len(images) >= 5 and images[0].width < 1100:
            per = 3
        gw = max(im.width for im in images)
        rows_of = [images[k:k + per] for k in range(0, len(images), per)]
        H = sum(max(im.height for im in r) for r in rows_of) + 8 * (len(rows_of) + 1)
        sheet = Image.new("RGB", (per * gw + 8 * (per + 1), H), (40, 40, 44))
        y = 8
        for r in rows_of:
            x = 8
            for im in r:
                sheet.paste(im, (x, y))
                x += gw + 8
            y += max(im.height for im in r) + 8
        sheet.save(out)
    else:
        images[0].save(out)
    print(f"wrote {out}")
    return status


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
