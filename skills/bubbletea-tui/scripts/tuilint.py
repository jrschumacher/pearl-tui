#!/usr/bin/env python3
"""tuilint.py — catch TUI slop deterministically, in rendered frames and in Go source.

Two modes. Run both before you show anything.

  tuilint.py frames <dir|files...> [--cols W --rows H] [--theme theme.json] [--json out.json]
      Lints rendered truecolor ANSI frames (the output of your `-state X -w W -h H` flag).
  tuilint.py code <dir>  [--theme-pkg path/fragment]  [--json out.json]
      Lints Go source for the mistakes that make frames wrong.

Every rule has an ID. A finding prints as `ID  file:where  message`. Rules marked [P0]/[P1]
fail the run (exit 1); [P2] are warnings you must either fix or justify in one line.

FRAME RULES
  F-WIDTH     [P0] a row is not exactly W cells / frame is not H rows (overflow wraps, the screen tears)
  F-EMOJI     [P1] emoji glyphs (width differs across terminals and fonts; reads as AI default)
  F-DEFAULT   [P1] colours from library examples/defaults (#7D56F4, #FF5FAF/205, 62/63/170/212, #EE6FF8, #874BFD)
  F-OFFPAL    [P1] (with --theme) a colour not in the theme's roles (hard-coded hex somewhere)
  F-CONTRAST  [P1] text whose fg/bg contrast is < 2.3:1 (unreadable; ≥3 needed for anything but hints)
  F-BOXES     [P1] more than one bordered box in a frame, or boxes nested/adjacent (box soup)
  F-RAINBOW   [P2] more than 7 distinct foreground hues in one frame (no hierarchy, just colour)
  F-LEVELS    [P2] fewer than 3 text levels (fg/muted/subtle) on a dense frame: a flat wall of text
  F-BOLD      [P2] more than 30% of text cells bold (bold stops meaning anything)
  F-EDGE      [P2] content touching column 0 or the last column with no gutter on most rows
  F-EMPTY     [P2] frame is >92% blank (an empty state with nothing to say?)
  F-TOFU      [P2] replacement or private-use glyphs (Nerd Font icons) with no fallback evidence

CODE RULES
  C-LEN       [P1] len()/utf8.RuneCountInString used to size/pad styled or user text: use lipgloss.Width
  C-SLEEP     [P0] time.Sleep / blocking I/O inside Update or View: return a tea.Cmd
  C-HEX       [P1] lipgloss.Color("#...") / hex literal outside the theme package (screens use roles)
  C-EMOJI     [P1] emoji in string literals
  C-V1        [P1] Bubble Tea / Lip Gloss v1 import paths (github.com/charmbracelet/bubbletea) in new code
  C-SIZE      [P1] a root model that never handles tea.WindowSizeMsg
  C-VIEWIO    [P1] os./http./exec. calls inside a View method
  C-FASTTICK  [P2] tea.Tick/Every faster than 33ms (CPU burn; nobody sees 60fps in a terminal)
  C-PRINT     [P2] fmt.Print* inside a Bubble Tea program (corrupts the frame; use tea.Println or a log file)
  C-KEYS      [P2] key strings matched inline in Update with no key.Binding / help (keys drift from help)
  C-NOCOLOR   [P2] no reference to colour profile / NO_COLOR handling anywhere
"""
import colorsys
import unicodedata
import glob
import json
import os
import re
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from termshot import parse  # noqa: E402

DEFAULT_COLOURS = {  # from Bubble Tea / Lip Gloss / Bubbles examples and default delegates
    (0x7D, 0x56, 0xF4): "#7D56F4 (lipgloss README purple)", (0xFF, 0x5F, 0xAF): "#FF5FAF / 205 (bubbles default pink)",
    (0xEE, 0x6F, 0xF8): "#EE6FF8 (list default selected)", (0x87, 0x4B, 0xFD): "#874BFD (lipgloss example border)",
    (0xAD, 0x58, 0xB4): "#AD58B4 (list default selected desc)", (0x5f, 0x5f, 0xd7): "62 (example purple)",
    (0x5f, 0x5f, 0xff): "63 (example purple)", (0xd7, 0x5f, 0xd7): "170 (example magenta)",
    (0xff, 0x87, 0xd7): "212 (example pink)", (0x04, 0xB5, 0x75): "#04B575 (example green)",
    (0xF2, 0x5D, 0x94): "#F25D94 (lipgloss example pink)", (0xFF, 0x5F, 0x87): "204 (example pink)",
}
CORNERS = set("╭╮╰╯┌┐└┘╔╗╚╝┏┓┗┛")
TL = set("╭┌╔┏")


def lum(c):
    def lin(v):
        v /= 255
        return v / 12.92 if v <= 0.04045 else ((v + 0.055) / 1.055) ** 2.4
    r, g, b = c
    return 0.2126 * lin(r) + 0.7152 * lin(g) + 0.0722 * lin(b)


def contrast(a, b):
    la, lb = lum(a), lum(b)
    return (max(la, lb) + 0.05) / (min(la, lb) + 0.05)


def hexc(s):
    s = s.lstrip("#")
    return tuple(int(s[i:i + 2], 16) for i in (0, 2, 4))


def theme_colours(path):
    t = json.load(open(path))
    roles = t.get("roles", t)
    cols = set()

    def walk(v):
        if isinstance(v, str) and re.fullmatch(r"#[0-9A-Fa-f]{6}", v):
            cols.add(hexc(v))
        elif isinstance(v, dict):
            for x in v.values():
                walk(x)
        elif isinstance(v, list):
            for x in v:
                walk(x)
    walk(roles)
    # gradients, dims and fades legitimately produce blends of two roles (a lerp in RGB or
    # in a perceptual space bows a little, hence the tolerance in near())
    base = list(cols)
    for i, a in enumerate(base):
        la = _oklab(a)
        for b in base[i + 1:]:
            lb = _oklab(b)
            for k in range(1, 20):
                t = k / 20
                cols.add(tuple(round(a[j] + (b[j] - a[j]) * t) for j in range(3)))
                cols.add(_from_oklab(tuple(la[j] + (lb[j] - la[j]) * t for j in range(3))))
    # dims, fields and scrims blend a blend toward a background: one more step
    bgs = [hexc(v) for k, v in _flat(roles) if k.startswith("bg") and re.fullmatch(r"#[0-9A-Fa-f]{6}", v)]
    for c in list(cols):
        for bgc in bgs:
            for k in range(1, 10):
                t = k / 10
                cols.add(tuple(round(c[j] + (bgc[j] - c[j]) * t) for j in range(3)))
    return {(c[0] // 6, c[1] // 6, c[2] // 6) for c in cols}


def _flat(roles):
    for v in roles.values():
        if isinstance(v, dict):
            yield from v.items()


def _oklab(c):
    def lin(v):
        v /= 255
        return v / 12.92 if v <= 0.04045 else ((v + 0.055) / 1.055) ** 2.4
    r, g, b = (lin(v) for v in c)
    l = (0.4122214708 * r + 0.5363325363 * g + 0.0514459929 * b) ** (1 / 3)
    m = (0.2119034982 * r + 0.6806995451 * g + 0.1073969566 * b) ** (1 / 3)
    s = (0.0883024619 * r + 0.2817188376 * g + 0.6299787005 * b) ** (1 / 3)
    return (0.2104542553 * l + 0.7936177850 * m - 0.0040720468 * s,
            1.9779984951 * l - 2.4285922050 * m + 0.4505937099 * s,
            0.0259040371 * l + 0.7827717662 * m - 0.8086757660 * s)


def _from_oklab(lab):
    L, a, b = lab
    l = (L + 0.3963377774 * a + 0.2158037573 * b) ** 3
    m = (L - 0.1055613458 * a - 0.0638541728 * b) ** 3
    s = (L - 0.0894841775 * a - 1.2914855480 * b) ** 3
    rgb = (4.0767416621 * l - 3.3077115913 * m + 0.2309699292 * s,
           -1.2684380046 * l + 2.6097574011 * m - 0.3413193965 * s,
           -0.0041960863 * l - 0.7034186147 * m + 1.7076147010 * s)

    def de(v):
        v = max(0.0, min(1.0, v))
        return 12.92 * v if v <= 0.0031308 else 1.055 * v ** (1 / 2.4) - 0.055
    return tuple(round(de(v) * 255) for v in rgb)


_near_cache = {}


def near(c, palette):
    """palette is a set of colours quantised to 6-unit buckets; accept the bucket or a neighbour."""
    if c not in _near_cache:
        q = (c[0] // 6, c[1] // 6, c[2] // 6)
        _near_cache[c] = any((q[0] + a, q[1] + b, q[2] + d) in palette
                             for a in (-1, 0, 1) for b in (-1, 0, 1) for d in (-1, 0, 1))
    return _near_cache[c]


def lint_frame(path, cols, rows, palette, findings, bg_default):
    text = open(path, encoding="utf-8", errors="replace").read()
    if text.endswith("\n"):
        text = text[:-1]
    grid, warns = parse(text)
    name = os.path.basename(path)

    def add(rule, sev, where, msg):
        findings.append({"rule": rule, "sev": sev, "file": name, "where": where, "msg": msg})

    widths = [len(r) for r in grid]
    W = cols or (max(widths) if widths else 0)
    bad = [i for i, w in enumerate(widths) if w != W]
    if bad:
        add("F-WIDTH", "P0", f"row {bad[0]}", f"{len(bad)} rows are not {W} cells wide (row {bad[0]} is {widths[bad[0]]})")
    if rows and len(grid) != rows:
        add("F-WIDTH", "P0", "frame", f"{len(grid)} rows, expected {rows}")
    for w in warns:
        if w.startswith("emoji"):
            add("F-EMOJI", "P1", "frame", "emoji glyphs present: use a glyph from the brand's vocabulary (✓ ● ◆ › ╱) instead")
    text_cells = bold = 0
    hues, levels, defaults, offpal, lowc, offpal_cells = {}, set(), {}, {}, [], []
    corners = []
    nonblank_rows = 0
    edge_left = edge_right = 0
    for y, row in enumerate(grid):
        line = "".join(c.ch or "" for c in row)
        if line.strip():
            nonblank_rows += 1
            if row and row[0].ch not in (" ", "") and row[0].bg is None:
                edge_left += 1
            if row and row[-1].ch not in (" ", "") and row[-1].bg is None:
                edge_right += 1
        for x, c in enumerate(row):
            for col in (c.fg, c.bg):
                if col is None:
                    continue
                for dc, label in DEFAULT_COLOURS.items():
                    if max(abs(col[k] - dc[k]) for k in range(3)) <= 3:
                        defaults.setdefault(label, (y, x))
                if palette and not near(col, palette):
                    offpal_cells.append(("#%02X%02X%02X" % col, y, x))
            ch = c.ch[:1] if c.ch else ""
            if ch in CORNERS:
                corners.append((y, x, ch))
            if not ch or ch.isspace() or ch in "─│╱╲━┃═║┄┈╌╍▔▁":
                continue
            o = ord(ch)
            if 0xE000 <= o <= 0xF8FF or 0xF0000 <= o:
                add("F-TOFU", "P2", f"r{y}c{x}", f"private-use glyph U+{o:04X} (Nerd Font icon): needs a fallback for users without the font")
            if ch == "�":
                add("F-TOFU", "P2", f"r{y}c{x}", "replacement character in output (broken encoding)")
            text_cells += 1
            if c.bold:
                bold += 1
            fg = c.fg or (216, 216, 216)
            bg = c.bg or bg_default
            if c.rev:
                fg, bg = bg, fg
            decorative = 0x2500 <= o <= 0x25FF or 0x2800 <= o <= 0x28FF or 0x1FB00 <= o <= 0x1FBFF or ch in "•·"
            if not decorative:
                cr = contrast(fg, bg)
                if cr < 2.3:
                    lowc.append((y, x, cr, fg, bg))
            h, l, s = colorsys.rgb_to_hls(*(v / 255 for v in fg))
            if s > 0.25 and 0.15 < l < 0.92:
                hues[int(h * 24)] = hues.get(int(h * 24), 0) + 1
            else:
                levels.add(round(lum(fg), 2))
    if defaults:
        for label, (y, x) in list(defaults.items())[:4]:
            add("F-DEFAULT", "P1", f"r{y}c{x}", f"library-default colour {label}: the 'every Bubble Tea app is pink and purple' look")
    tls_all = [(y, x) for y, x, ch in corners if ch in TL]
    overlay_name = re.search(r"(dialog|palette|confirm|modal|overlay|popup|help|keys)", name)
    if overlay_name and len(tls_all) != 1 and (lowc or offpal_cells):
        # a borderless dialog over a scrim: we can't find its edges, so don't judge the scrim
        add("F-CONTRAST", "P2", "frame", "overlay frame without a bordered dialog: contrast and palette checked only by eye here")
        lowc, offpal_cells = [], []
    if len(tls_all) == 1 and lowc:
        # one box open = a dialog over a scrim: only its inside (and the footer) must be readable
        y0, x0 = tls_all[0]
        bl = [(y, x) for y, x, ch in corners if ch in "╰└╚┗" and x == x0 and y > y0]
        y1 = bl[0][0] if bl else len(grid) - 1
        row = grid[y0]
        x1 = next((k for k in range(x0 + 1, len(row)) if (row[k].ch or "")[:1] in "╮┐╗┓"), len(row) - 1)
        last = len(grid) - 1
        inside = lambda y, x: (y0 <= y <= y1 and x0 <= x <= x1) or y == last
        lowc = [c for c in lowc if inside(c[0], c[1])]
        offpal_cells = [c for c in offpal_cells if inside(c[1], c[2])]
    for col, y, x in offpal_cells:
        offpal.setdefault(col, (y, x))
    if offpal:
        items = sorted(offpal.items(), key=lambda kv: kv[1])[:6]
        add("F-OFFPAL", "P1", f"r{items[0][1][0]}c{items[0][1][1]}",
            f"{len(offpal)} colours not in the theme roles: " + ", ".join(k for k, _ in items))
    if lowc:
        rows_hit = sorted(set(y for y, *_ in lowc))
        y, x, cr, fg, bg = lowc[0]
        add("F-CONTRAST", "P1", f"r{y}c{x}", f"{len(lowc)} text cells under 2.3:1 on rows {rows_hit[:8]} (first {cr:.2f}:1, fg #%02X%02X%02X on bg #%02X%02X%02X)" % (fg + bg))
    tls = [(y, x) for y, x, ch in corners if ch in TL]
    if len(tls) > 1:
        boxes = []
        for (y, x) in tls:
            row = grid[y]
            x2 = x + 1
            while x2 < len(row) and (row[x2].ch[:1] if row[x2].ch else "") not in "╮┐╗┓":
                x2 += 1
            boxes.append((y, x, x2))
        nested = any(a != b and a[0] < b[0] and a[1] < b[1] and b[2] < a[2] for a in boxes for b in boxes)
        adjacent = any(a != b and abs(a[2] - b[1]) <= 1 and a[0] == b[0] for a in boxes for b in boxes)
        add("F-BOXES", "P1", f"r{tls[0][0]}c{tls[0][1]}",
            f"{len(tls)} bordered boxes" + (" (nested)" if nested else "") + (" (side by side)" if adjacent else "") +
            ": separate regions with whitespace, rules and background steps; keep borders for the one thing that floats (a dialog)")
    big = [h for h, n in hues.items() if n >= 4]
    if len(big) > 7:
        add("F-RAINBOW", "P2", "frame", f"{len(big)} distinct foreground hues: hierarchy should come from lightness levels, colour from roles")
    if text_cells > 400 and len(levels) < 3:
        add("F-LEVELS", "P2", "frame", f"only {len(levels)} neutral text level(s) on a dense frame: use fg / fgMuted / fgSubtle")
    if text_cells and bold / text_cells > 0.30:
        add("F-BOLD", "P2", "frame", f"{100 * bold // text_cells}% of text is bold")
    if nonblank_rows and (edge_left > 0.6 * nonblank_rows or edge_right > 0.6 * nonblank_rows):
        add("F-EDGE", "P2", "frame", "content touches the terminal edge on most rows: give it a 1–2 cell gutter")
    total = sum(widths) or 1
    if text_cells / total < 0.015:
        add("F-EMPTY", "P2", "frame", "almost nothing on screen: if this is an empty state, say what is missing and what to do")


GO_EMOJI = re.compile("[\U0001F300-\U0001FAFF☀-⛿✀-➿]")
EMOJI_OK = set("✓✔✗✘★☆☐☑☒✦✧✱✳❯❮➜➤⚠")  # dingbats that are text glyphs in most mono fonts


def lint_code(root, theme_pkg, findings):
    files = [f for f in glob.glob(os.path.join(root, "**", "*.go"), recursive=True)
             if "/vendor/" not in f and not f.endswith("_test.go")]
    saw_size = saw_profile = saw_binding = False
    inline_keys = []
    for f in files:
        rel = os.path.relpath(f, root)
        src = open(f, encoding="utf-8", errors="replace").read()
        lines = src.split("\n")

        def add(rule, sev, ln, msg):
            findings.append({"rule": rule, "sev": sev, "file": rel, "where": f"{ln}", "msg": msg})

        if "WindowSizeMsg" in src:
            saw_size = True
        if re.search(r"colorprofile|NO_COLOR|HasDarkBackground|BackgroundColorMsg|lipgloss\.Complete|LightDark|isDark", src):
            saw_profile = True
        if "key.NewBinding" in src or "key.Binding" in src:
            saw_binding = True
        is_theme = theme_pkg and theme_pkg in rel or re.search(r"(^|/)(theme|styles?|palette|colors?)(/|\.go$|_)", rel)
        func = None
        method = False
        for i, ln in enumerate(lines, 1):
            if ln.startswith("func "):
                m = re.match(r"func \((\w+) \*?(\w+)\) (\w+)\(", ln)
                func = m.group(3) if m else None
                method = bool(m)
            code = re.sub(r"//.*$", "", ln)
            for arg in re.findall(r"\b(?:len|utf8\.RuneCountInString)\(([^()]*(?:\([^()]*\))?[^()]*)\)", code):
                a = arg.strip()
                last = re.split(r"[.\[]", a)[-1] if a else a
                if not re.search(r"(Repeat|Width|[pP]ad|width|cols|Cols)", code):
                    continue
                if a.startswith("[]rune") or last.endswith("s") or last in ("Banner", "top", "pre", "post", "src", "grid", "m", "b", "buf", "data", "out", "res", "parts", "args", "frames"):
                    continue
                add("C-LEN", "P1", i, f"len({a}) in width maths: if {a} is text, use lipgloss.Width (cells, not bytes); ansi.Truncate to cut it")
                break
            if func in ("Update", "View", "update", "view") and re.search(r"time\.Sleep\(|http\.(Get|Post|Do)\(|\.ReadFile\(|exec\.Command\(.*\)\.(Run|Output)", code):
                add("C-SLEEP" if "Sleep" in code else "C-VIEWIO", "P0" if "Sleep" in code else "P1", i,
                    f"blocking call in {func}: move it into a tea.Cmd and send the result back as a Msg")
            if not is_theme and re.search(r'(lipgloss\.Color\(\s*"#|"#[0-9A-Fa-f]{6}")', code):
                add("C-HEX", "P1", i, "hex colour outside the theme: screens take roles (t.Primary, t.FgMuted), never literals")
            for em in GO_EMOJI.findall(code):
                if em not in EMOJI_OK and '"' in code and unicodedata.east_asian_width(em) in ("W", "F"):
                    add("C-EMOJI", "P1", i, f"emoji {em!r} in a string: emoji width varies by terminal; use the brand glyph set")
                    break
            if re.search(r'"github\.com/charmbracelet/(bubbletea|lipgloss|bubbles|huh)(/[\w/]+)?"', code):
                add("C-V1", "P1", i, "v1 import: new code uses charm.land/bubbletea/v2, charm.land/lipgloss/v2, charm.land/bubbles/v2")
            mt = re.search(r"tea\.(Tick|Every)\(\s*(\d+)\s*\*\s*time\.Millisecond", code)
            if mt and int(mt.group(2)) < 33:
                add("C-FASTTICK", "P2", i, f"{mt.group(2)}ms tick: 30–80ms is plenty for motion; idle screens should not tick at all")
            if re.search(r"\bfmt\.Print(f|ln)?\(", code) and "tea." in src and method:
                add("C-PRINT", "P2", i, "fmt.Print inside a running program corrupts the frame: tea.Println, a log file, or tea.LogToFile")
            if func in ("Update", "update") and re.search(r'case\s+"(ctrl\+\w|[a-z]|enter|esc|tab|up|down)"', code):
                inline_keys.append((rel, i))
    if files and not saw_size:
        findings.append({"rule": "C-SIZE", "sev": "P1", "file": root, "where": "-", "msg": "no tea.WindowSizeMsg handled anywhere: layout cannot respond to the terminal"})
    if files and not saw_profile:
        findings.append({"rule": "C-NOCOLOR", "sev": "P2", "file": root, "where": "-", "msg": "no colour-profile / dark-background handling found: check light terminals, 256 and 16 colours, NO_COLOR"})
    if inline_keys and not saw_binding:
        rel, i = inline_keys[0]
        findings.append({"rule": "C-KEYS", "sev": "P2", "file": rel, "where": f"{i}", "msg": f"{len(inline_keys)} keys matched as raw strings and no key.Binding: the help view will drift from reality"})


def main(argv):
    if not argv or argv[0] in ("-h", "--help") or argv[0] not in ("frames", "code"):
        print(__doc__)
        return 0 if argv and argv[0] in ("-h", "--help") else 2
    mode, rest = argv[0], argv[1:]
    opts, pos = {}, []
    i = 0
    while i < len(rest):
        if rest[i].startswith("--"):
            opts[rest[i][2:]] = rest[i + 1]
            i += 2
        else:
            pos.append(rest[i])
            i += 1
    findings = []
    if mode == "frames":
        paths = []
        for p in pos:
            paths += sorted(glob.glob(os.path.join(p, "*.ansi"))) if os.path.isdir(p) else [p]
        palette = theme_colours(opts["theme"]) if "theme" in opts else None
        bgd = hexc(opts.get("bg", "#16161a"))
        for p in paths:
            lint_frame(p, int(opts["cols"]) if "cols" in opts else None, int(opts["rows"]) if "rows" in opts else None,
                       palette, findings, bgd)
        print(f"linted {len(paths)} frame(s)")
    else:
        for p in pos or ["."]:
            lint_code(p, opts.get("theme-pkg"), findings)
    order = {"P0": 0, "P1": 1, "P2": 2}
    findings.sort(key=lambda f: (order[f["sev"]], f["rule"], f["file"]))
    for f in findings:
        print(f"[{f['sev']}] {f['rule']:<11} {f['file']}:{f['where']}  {f['msg']}")
    fails = sum(1 for f in findings if f["sev"] in ("P0", "P1"))
    print(("FAIL" if fails else "OK") + f"  {fails} blocking, {len(findings) - fails} warnings")
    if "json" in opts:
        json.dump(findings, open(opts["json"], "w"), indent=1)
    return 1 if fails else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
