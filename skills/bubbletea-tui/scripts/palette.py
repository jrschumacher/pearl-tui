#!/usr/bin/env python3
"""palette.py — derive a terminal role palette from brand colours, and check one.

Brand colours are measured from the brand (logo, site CSS, style guide), never invented.
This turns 1–3 of them into the full role set a TUI needs, in OKLCH so lightness steps
are perceptual, with neutrals tinted toward the brand hue (the single biggest reason a
palette reads "designed" instead of "default grey"), status colours pulled into the
brand's lightness/chroma band so they sit with it, and every text role checked for
contrast against the background it will actually sit on.

  palette.py derive "#6B50FF" "#FF60FF" [--accent "#E8FE96"] [--bg "#201F26"] [--light-bg "#FBFAF7"]
                    [--tint 0.012] [--name crush] [--out theme-roles.json] [--swatch swatch.ansi]
  palette.py check theme.json        contrast report for an existing theme's roles (exit 1 on fail)
                                     a theme may accept a deviation in writing:
                                     "contrastAccept": {"dark.primary": "the brand's own hex, used as a fill"}

`derive` prints the roles as JSON (`{"dark": {...}, "light": {...}}`, the theme contract's
shape), a contrast table, and the nearest xterm-256 index for each role (for the
ANSI256 fallback). `--swatch` writes an .ansi file you can render with cellshot.py.

Thresholds (WCAG 2 contrast ratio against the role's background):
  fg ≥ 7      fgMuted ≥ 4.5     fgSubtle ≥ 2.6 (hints only, never the only carrier of meaning)
  primary/secondary/accent/status as text ≥ 3.5     border 1.25–1.7 (barely there, but there)
  fgSelected on primary ≥ 4.5   text on bgSubtle checked too
  `check` adds the pairs a theme's expressions put on screen: "footer": "bar" draws the
  key hints (fgSubtle) on bgSubtle, so that pair must clear tuilint's 2.3 floor
"""
import json
import math
import sys


# ----------------------------------------------------------------------------- colour maths
def hex2rgb(h):
    h = h.strip().lstrip("#")
    if len(h) == 3:
        h = "".join(c * 2 for c in h)
    return tuple(int(h[i:i + 2], 16) / 255 for i in (0, 2, 4))


def rgb2hex(c):
    return "#" + "".join(f"{max(0, min(255, round(v * 255))):02X}" for v in c)


def lin(v):
    return v / 12.92 if v <= 0.04045 else ((v + 0.055) / 1.055) ** 2.4


def delin(v):
    return 12.92 * v if v <= 0.0031308 else 1.055 * v ** (1 / 2.4) - 0.055


def rgb2oklab(c):
    r, g, b = (lin(v) for v in c)
    l = 0.4122214708 * r + 0.5363325363 * g + 0.0514459929 * b
    m = 0.2119034982 * r + 0.6806995451 * g + 0.1073969566 * b
    s = 0.0883024619 * r + 0.2817188376 * g + 0.6299787005 * b
    l, m, s = (math.copysign(abs(x) ** (1 / 3), x) for x in (l, m, s))
    return (0.2104542553 * l + 0.7936177850 * m - 0.0040720468 * s,
            1.9779984951 * l - 2.4285922050 * m + 0.4505937099 * s,
            0.0259040371 * l + 0.7827717662 * m - 0.8086757660 * s)


def oklab2rgb(L, a, b):
    l = (L + 0.3963377774 * a + 0.2158037573 * b) ** 3
    m = (L - 0.1055613458 * a - 0.0638541728 * b) ** 3
    s = (L - 0.0894841775 * a - 1.2914855480 * b) ** 3
    r = 4.0767416621 * l - 3.3077115913 * m + 0.2309699292 * s
    g = -1.2684380046 * l + 2.6097574011 * m - 0.3413193965 * s
    bb = -0.0041960863 * l - 0.7034186147 * m + 1.7076147010 * s
    return tuple(delin(v) for v in (r, g, bb))


def to_lch(hexc):
    L, a, b = rgb2oklab(hex2rgb(hexc))
    return L, math.hypot(a, b), math.degrees(math.atan2(b, a)) % 360


def in_gamut(c):
    return all(-1e-4 <= v <= 1 + 1e-4 for v in c)


def from_lch(L, C, H):
    """OKLCH -> hex, reducing chroma until it fits sRGB (keeps hue and lightness)."""
    lo, hi = 0.0, C
    rgb = oklab2rgb(L, C * math.cos(math.radians(H)), C * math.sin(math.radians(H)))
    if in_gamut(rgb):
        return rgb2hex(rgb)
    for _ in range(24):
        mid = (lo + hi) / 2
        rgb = oklab2rgb(L, mid * math.cos(math.radians(H)), mid * math.sin(math.radians(H)))
        if in_gamut(rgb):
            lo = mid
        else:
            hi = mid
    return rgb2hex(oklab2rgb(L, lo * math.cos(math.radians(H)), lo * math.sin(math.radians(H))))


def luminance(hexc):
    r, g, b = (lin(v) for v in hex2rgb(hexc))
    return 0.2126 * r + 0.7152 * g + 0.0722 * b


def contrast(a, b):
    la, lb = luminance(a), luminance(b)
    hi, lo = max(la, lb), min(la, lb)
    return (hi + 0.05) / (lo + 0.05)


def tune_L(L0, C, H, bg, target, direction):
    """Move lightness from L0 in `direction` (+1 lighter / -1 darker) until contrast vs bg ≥ target."""
    L = L0
    for _ in range(200):
        if contrast(from_lch(L, C, H), bg) >= target:
            return L
        L = max(0.0, min(1.0, L + 0.005 * direction))
        if L in (0.0, 1.0):
            break
    return L


def xterm256(hexc):
    r, g, b = (round(v * 255) for v in hex2rgb(hexc))
    steps = [0, 95, 135, 175, 215, 255]
    best, bi = 1e9, 0
    for i in range(16, 256):
        if i < 232:
            n = i - 16
            c = (steps[n // 36], steps[(n // 6) % 6], steps[n % 6])
        else:
            v = 8 + (i - 232) * 10
            c = (v, v, v)
        L1, a1, b1 = rgb2oklab(tuple(x / 255 for x in c))
        L2, a2, b2 = rgb2oklab((r / 255, g / 255, b / 255))
        d = (L1 - L2) ** 2 + (a1 - a2) ** 2 + (b1 - b2) ** 2
        if d < best:
            best, bi = d, i
    return bi


# ----------------------------------------------------------------------------- derive
STATUS_HUES = {"success": 150, "error": 22, "warning": 85, "info": 240}


def derive(primary, secondary=None, accent=None, bg=None, light_bg=None, tint=0.012):
    Lp, Cp, Hp = to_lch(primary)
    if secondary:
        Ls, Cs, Hs = to_lch(secondary)
    else:  # analogous partner, 40° away, so a two-stop gradient has somewhere to go
        Ls, Cs, Hs = Lp, Cp, (Hp + 40) % 360
    if accent:
        La, Ca, Ha = to_lch(accent)
    else:  # a light, low-chroma tint of the primary: headings that read as "title", not "alert"
        La, Ca, Ha = 0.88, min(0.08, Cp * 0.5), Hp
    out = {}
    for mode in ("dark", "light"):
        dark = mode == "dark"
        if dark:
            B = bg or from_lch(0.20, tint * 1.2, Hp)
        else:
            B = light_bg or from_lch(0.985, tint * 0.6, Hp)
        LB, CB, HB = to_lch(B)
        d = 1 if dark else -1  # direction "away from the background"
        nC = max(tint, min(CB, 0.04))  # neutral chroma: tinted toward the brand hue
        # neutrals lean toward the background's own hue when it has one (a navy
        # bg wants navy greys, not greys tinted toward an orange primary)
        Hn = HB if CB > 0.012 else Hp
        r = {"bg": B}
        r["bgSubtle"] = from_lch(LB + d * 0.075, nC * 1.4, Hn)
        r["bgOverlay"] = from_lch(LB + d * 0.13, nC * 1.6, Hn)
        r["bgPanel"] = from_lch(LB + d * 0.035, nC * 1.2, Hn)
        r["border"] = from_lch(LB + d * 0.07, nC * 1.6, Hn)
        r["fgSubtle"] = from_lch(tune_L(LB + d * 0.25, nC * 1.8, Hn, B, 2.7, d), nC * 1.8, Hn)
        r["fgMuted"] = from_lch(tune_L(LB + d * 0.35, nC * 1.6, Hn, B, 4.6, d), nC * 1.6, Hn)
        r["fg"] = from_lch(tune_L(LB + d * 0.62, nC * 0.8, Hn, B, 11 if dark else 12, d), nC * 0.8, Hn)
        r["fgSoft"] = from_lch(tune_L(LB + d * 0.52, nC, Hn, B, 8 if dark else 9, d), nC, Hn)

        def role(L, C, H, target):
            L0 = L if dark else min(L, 0.62)
            return from_lch(tune_L(L0, C, H, B, target, d), C, H)

        r["primary"] = role(Lp, Cp, Hp, 3.5)
        r["secondary"] = role(Ls, Cs, Hs, 3.5)
        r["tertiary"] = role(max(Lp, 0.78) if dark else 0.5, min(Cp, 0.14), (Hp + 160) % 360, 4.5)
        r["accent"] = role(La if dark else 0.42, Ca if dark else max(Ca, 0.09), Ha, 4.5)
        r["borderFocus"] = r["primary"]
        band_L = max(0.68, min(0.80, Lp)) if dark else 0.50
        band_C = max(0.13, min(0.19, Cp))
        for k, h in STATUS_HUES.items():
            r[k] = role(band_L if k != "warning" else (0.84 if dark else 0.55), band_C, h, 4.5 if k != "info" else 3.5)
        light_ink, dark_ink = from_lch(0.985, nC * 0.4, Hn), from_lch(0.18, nC * 1.5, Hn)

        def on(fill):  # text on a filled role: whichever ink reads better
            return light_ink if contrast(light_ink, fill) >= contrast(dark_ink, fill) else dark_ink

        r["onPrimary"], r["onSecondary"], r["onWarning"] = on(r["primary"]), on(r["secondary"]), on(r["warning"])
        r["onError"], r["onSuccess"] = on(r["error"]), on(r["success"])
        r["fgSelected"] = r["onPrimary"]  # text on a selected (primary-filled) row
        out[mode] = dict(sorted(r.items()))
    return out


CHECKS = [  # (role, against, minimum, maximum)
    ("fg", "bg", 7.0, None), ("fgMuted", "bg", 4.5, None), ("fgSubtle", "bg", 2.6, None),
    ("fg", "bgSubtle", 4.5, None), ("fgMuted", "bgSubtle", 3.0, None),
    ("primary", "bg", 3.5, None), ("secondary", "bg", 3.0, None), ("tertiary", "bg", 4.5, None),
    ("accent", "bg", 4.5, None), ("success", "bg", 3.5, None), ("error", "bg", 3.5, None),
    ("warning", "bg", 3.5, None), ("info", "bg", 3.0, None), ("border", "bg", 1.14, 1.9),
    ("bgSubtle", "bg", 1.1, 2.2), ("onPrimary", "primary", 4.5, None), ("onSecondary", "secondary", 3.5, None),
    ("onWarning", "warning", 4.5, None), ("onError", "error", 3.5, None), ("onSuccess", "success", 3.5, None),
    ("fgSelected", "primary", 3.5, None),
]


def expression_checks(expressions):
    """Pairs that exist only because of an expression choice (mirrors kit/ui and kit/theme)."""
    extra = []
    if expressions.get("footer") == "bar":  # ui.Footer: key hints in fgSubtle on a bgSubtle strip
        extra.append(("fgSubtle", "bgSubtle", 2.3, None))
    return extra


def report(roles, label, accept=None, extra=()):
    """accept: {"dark.primary": "reason"} — a deviation the theme justifies in writing.
    extra: more (role, against, min, max) checks, from expression_checks."""
    accept = accept or {}
    bad = 0
    lines = [f"  {label}"]
    for a, b, lo, hi in CHECKS + list(extra):
        if a not in roles or b not in roles:
            continue
        c = contrast(roles[a], roles[b])
        ok = c >= lo and (hi is None or c <= hi)
        rng = f"≥{lo}" + (f" ≤{hi}" if hi else "")
        why = accept.get(f"{label}.{a}")
        mark = "ok " if ok else ("ACCEPT" if why else "FAIL")
        bad += not ok and not why
        lines.append(f"    {mark:<4} {a:<12} on {b:<9} {c:5.2f}  (want {rng})  {roles[a]} / {roles[b]}" + (f"  — {why}" if why and not ok else ""))
    return bad, "\n".join(lines)


def fill_on_colours(r):
    """Mirror the Go engine (kit/theme/derive.go): a missing on-colour is the legacy
    `white` if it clears 4.5:1, else whichever of bg/fg reads better on the fill."""
    for on, fill in (("onPrimary", "primary"), ("onSecondary", "secondary"), ("onWarning", "warning"),
                     ("onError", "error"), ("onSuccess", "success")):
        if on in r or fill not in r:
            continue
        w = r.get("white")
        if w and contrast(w, r[fill]) >= 4.5:
            r[on] = w
        else:
            r[on] = max((r.get("bg", "#000000"), r.get("fg", "#ffffff")), key=lambda c: contrast(c, r[fill]))
    r.setdefault("fgSelected", r.get("onPrimary"))
    return r


def swatch(roles_by_mode):
    def sgr_fg(h):
        r, g, b = (round(v * 255) for v in hex2rgb(h))
        return f"\x1b[38;2;{r};{g};{b}m"

    def sgr_bg(h):
        r, g, b = (round(v * 255) for v in hex2rgb(h))
        return f"\x1b[48;2;{r};{g};{b}m"

    rows = []
    for mode, r in roles_by_mode.items():
        bg = sgr_bg(r["bg"])
        W = 64

        def line(s, vis):
            return bg + s + bg + " " * max(0, W - vis) + "\x1b[0m"

        rows.append(line(sgr_fg(r["fgMuted"]) + f" {mode}", len(mode) + 1))
        for k in ["primary", "secondary", "tertiary", "accent", "success", "error", "warning", "info"]:
            txt = f" ██ {k:<10} {r[k]}  "
            sample = "The quick brown fox"
            rows.append(line(sgr_fg(r[k]) + txt + sample, len(txt) + len(sample)))
        for k in ["fg", "fgMuted", "fgSubtle"]:
            txt = f" Aa {k:<10} {r[k]}  "
            sample = "body text at this level"
            rows.append(line(sgr_fg(r[k]) + txt + sample, len(txt) + len(sample)))
        rule = " " + sgr_fg(r["border"]) + "─" * 40 + sgr_fg(r["fgSubtle"]) + " border"
        rows.append(line(rule, 48))
        pill = " " + sgr_bg(r["secondary"]) + sgr_fg(r["onSecondary"]) + "  Save  " + bg + " " + sgr_bg(r["bgSubtle"]) + sgr_fg(r["fg"]) + "  Cancel  " + bg + "  " + sgr_bg(r["primary"]) + sgr_fg(r["onPrimary"]) + " selected row " + bg
        rows.append(line(pill, 1 + 8 + 1 + 10 + 2 + 14))
        rows.append(line("", 0))
    return "\n".join(rows) + "\n"


def main(argv):
    if not argv or argv[0] in ("-h", "--help"):
        print(__doc__)
        return 0
    if argv[0] == "check":
        t = json.load(open(argv[1]))
        roles = t.get("roles", t)
        bad = 0
        for mode, r in roles.items():
            if not isinstance(r, dict):
                continue
            r = fill_on_colours(dict(r))
            b, txt = report(r, mode, t.get("contrastAccept", {}), expression_checks(t.get("expressions", {})))
            bad += b
            print(txt)
        print("FAIL" if bad else "OK", f"({bad} contrast problems)")
        return 1 if bad else 0
    if argv[0] != "derive":
        print(__doc__)
        return 2
    pos, opts = [], {}
    i = 1
    while i < len(argv):
        if argv[i].startswith("--"):
            opts[argv[i][2:]] = argv[i + 1]
            i += 2
        else:
            pos.append(argv[i])
            i += 1
    roles = derive(pos[0], pos[1] if len(pos) > 1 else None, opts.get("accent"), opts.get("bg"),
                   opts.get("light-bg"), float(opts.get("tint", 0.012)))
    doc = {"name": opts.get("name", "brand"), "roles": roles,
           "ansi256": {m: {k: xterm256(v) for k, v in r.items()} for m, r in roles.items()}}
    js = json.dumps(doc, indent=1)
    if "out" in opts:
        open(opts["out"], "w").write(js + "\n")
        print(f"wrote {opts['out']}")
    else:
        print(js)
    bad = 0
    for m, r in roles.items():
        b, txt = report(r, m)
        bad += b
        print(txt, file=sys.stderr)
    if "swatch" in opts:
        open(opts["swatch"], "w").write(swatch(roles))
        print(f"wrote {opts['swatch']} (render: cellshot.py {opts['swatch']} swatch.png)", file=sys.stderr)
    print(("FAIL" if bad else "OK") + f" ({bad} contrast problems)", file=sys.stderr)
    return 1 if bad else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
