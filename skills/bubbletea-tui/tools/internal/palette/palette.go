// Package palette is the engine behind the palette command: derive a dark and a light
// role set from one to three brand colours in OKLCH, and check any role set's contrast.
// It is a port of scripts/palette.py and produces the same bytes (see the testdata
// goldens). It differs on purpose in five places:
//
//   - check takes a missing on-colour from the kit's engine (theme.Resolve) rather than
//     re-deriving it, so a theme without bg or fg is checked against the engine's default
//     inks (#15171C/#D8DEE9 dark) where palette.py used #000000/#ffffff;
//   - options also take "--opt=value" (palette.py read "--bg=#000" as an option named
//     "bg=#000" and swallowed the next argument);
//   - the swatch hint names the Go cellshot, not cellshot.py;
//   - bad input (a missing file, a role that is not a hex colour, a missing option value)
//     exits 2 with one line on stderr instead of a traceback (exit 1); check validates
//     every role it may read up front, where palette.py failed only on the one it reached;
//   - -h prints the command's Go doc comment.
package palette

import (
	"encoding/json"
	"fmt"
	"image/color"
	"sort"
	"strings"
	"unicode/utf8"

	"tuikit/theme"
	"tuitools/internal/oklch"
	"tuitools/internal/pyfmt"
)

// Mode is one named role set ("dark" or "light"), role name to hex.
type Mode struct {
	Name  string
	Roles map[string]string
}

// pymax and pymin are Python's two-argument max and min: the first wins a tie.
func pymax(a, b float64) float64 {
	if b > a {
		return b
	}
	return a
}

func pymin(a, b float64) float64 {
	if b < a {
		return b
	}
	return a
}

// tuneL moves lightness from l0 in dir (+1 lighter, -1 darker) in steps of 0.005 until
// the colour's contrast against bg reaches target.
func tuneL(l0, c, h float64, bg string, target, dir float64) float64 {
	l := l0
	for range 200 {
		if oklch.Contrast(oklch.FromLCH(l, c, h), bg) >= target {
			return l
		}
		l = pymax(0.0, pymin(1.0, l+0.005*dir))
		if l == 0 || l == 1 {
			break
		}
	}
	return l
}

var statusHues = []struct {
	role string
	hue  float64
}{{"success", 150}, {"error", 22}, {"warning", 85}, {"info", 240}}

// Derive builds the dark and light role sets. secondary, accent, bg and lightBg may be
// empty: the secondary then sits 40° from the primary, the accent is a light low-chroma
// tint of it, and each background is a near-neutral tinted toward its hue. Every input
// must be a valid hex colour (see oklch.Valid).
func Derive(primary, secondary, accent, bg, lightBg string, tint float64) []Mode {
	lp, cp, hp := oklch.LCH(primary)
	ls, cs, hs := lp, cp, oklch.Mod(hp+40, 360)
	if secondary != "" {
		ls, cs, hs = oklch.LCH(secondary)
	}
	la, ca, ha := 0.88, pymin(0.08, cp*0.5), hp
	if accent != "" {
		la, ca, ha = oklch.LCH(accent)
	}
	var out []Mode
	for _, mode := range []string{"dark", "light"} {
		dark := mode == "dark"
		b := lightBg
		if dark {
			b = bg
		}
		if b == "" {
			if dark {
				b = oklch.FromLCH(0.20, tint*1.2, hp)
			} else {
				b = oklch.FromLCH(0.985, tint*0.6, hp)
			}
		}
		lb, cb, hb := oklch.LCH(b)
		d := 1.0 // direction "away from the background"
		if !dark {
			d = -1
		}
		nc := pymax(tint, pymin(cb, 0.04)) // neutral chroma, tinted toward the brand hue
		hn := hp                           // neutrals lean toward a coloured background's own hue
		if cb > 0.012 {
			hn = hb
		}
		r := map[string]string{"bg": b}
		r["bgSubtle"] = oklch.FromLCH(lb+d*0.075, nc*1.4, hn)
		r["bgOverlay"] = oklch.FromLCH(lb+d*0.13, nc*1.6, hn)
		r["bgPanel"] = oklch.FromLCH(lb+d*0.035, nc*1.2, hn)
		r["border"] = oklch.FromLCH(lb+d*0.07, nc*1.6, hn)
		r["fgSubtle"] = oklch.FromLCH(tuneL(lb+d*0.25, nc*1.8, hn, b, 2.7, d), nc*1.8, hn)
		r["fgMuted"] = oklch.FromLCH(tuneL(lb+d*0.35, nc*1.6, hn, b, 4.6, d), nc*1.6, hn)
		fgT, softT := 11.0, 8.0
		if !dark {
			fgT, softT = 12, 9
		}
		r["fg"] = oklch.FromLCH(tuneL(lb+d*0.62, nc*0.8, hn, b, fgT, d), nc*0.8, hn)
		r["fgSoft"] = oklch.FromLCH(tuneL(lb+d*0.52, nc, hn, b, softT, d), nc, hn)

		role := func(l, c, h, target float64) string {
			l0 := l
			if !dark {
				l0 = pymin(l, 0.62)
			}
			return oklch.FromLCH(tuneL(l0, c, h, b, target, d), c, h)
		}
		r["primary"] = role(lp, cp, hp, 3.5)
		r["secondary"] = role(ls, cs, hs, 3.5)
		if dark {
			r["tertiary"] = role(pymax(lp, 0.78), pymin(cp, 0.14), oklch.Mod(hp+160, 360), 4.5)
			r["accent"] = role(la, ca, ha, 4.5)
		} else {
			r["tertiary"] = role(0.5, pymin(cp, 0.14), oklch.Mod(hp+160, 360), 4.5)
			r["accent"] = role(0.42, pymax(ca, 0.09), ha, 4.5)
		}
		r["borderFocus"] = r["primary"]
		bandL := 0.50
		if dark {
			bandL = pymax(0.68, pymin(0.80, lp))
		}
		bandC := pymax(0.13, pymin(0.19, cp))
		for _, s := range statusHues {
			l, target := bandL, 4.5
			if s.role == "warning" {
				l = 0.55
				if dark {
					l = 0.84
				}
			}
			if s.role == "info" {
				target = 3.5
			}
			r[s.role] = role(l, bandC, s.hue, target)
		}
		lightInk, darkInk := oklch.FromLCH(0.985, nc*0.4, hn), oklch.FromLCH(0.18, nc*1.5, hn)
		on := func(fill string) string { // text on a filled role: whichever ink reads better
			if theme.OnColor(oklch.Color(fill), oklch.Color(lightInk), oklch.Color(darkInk)) == oklch.Color(lightInk) {
				return lightInk
			}
			return darkInk
		}
		r["onPrimary"], r["onSecondary"], r["onWarning"] = on(r["primary"]), on(r["secondary"]), on(r["warning"])
		r["onError"], r["onSuccess"] = on(r["error"]), on(r["success"])
		r["fgSelected"] = r["onPrimary"]
		out = append(out, Mode{mode, r})
	}
	return out
}

// Check is one contrast requirement: role a against role b, within [min, max].
type Check struct {
	A, B     string
	Min, Max float64
	want     string // the range as palette.py prints it
}

// Checks are the contrast requirements, in report order.
var Checks = []Check{
	{"fg", "bg", 7.0, 0, "≥7.0"}, {"fgMuted", "bg", 4.5, 0, "≥4.5"}, {"fgSubtle", "bg", 2.6, 0, "≥2.6"},
	{"fg", "bgSubtle", 4.5, 0, "≥4.5"}, {"fgMuted", "bgSubtle", 3.0, 0, "≥3.0"},
	{"primary", "bg", 3.5, 0, "≥3.5"}, {"secondary", "bg", 3.0, 0, "≥3.0"}, {"tertiary", "bg", 4.5, 0, "≥4.5"},
	{"accent", "bg", 4.5, 0, "≥4.5"}, {"success", "bg", 3.5, 0, "≥3.5"}, {"error", "bg", 3.5, 0, "≥3.5"},
	{"warning", "bg", 3.5, 0, "≥3.5"}, {"info", "bg", 3.0, 0, "≥3.0"}, {"border", "bg", 1.14, 1.9, "≥1.14 ≤1.9"},
	{"bgSubtle", "bg", 1.1, 2.2, "≥1.1 ≤2.2"}, {"onPrimary", "primary", 4.5, 0, "≥4.5"}, {"onSecondary", "secondary", 3.5, 0, "≥3.5"},
	{"onWarning", "warning", 4.5, 0, "≥4.5"}, {"onError", "error", 3.5, 0, "≥3.5"}, {"onSuccess", "success", 3.5, 0, "≥3.5"},
	{"fgSelected", "primary", 3.5, 0, "≥3.5"},
}

// Report checks one role set and returns the number of failures and the report text.
// accept maps "<label>.<role>" to the reason a theme gives in writing for a deviation:
// a failing check on that role is reported as ACCEPT and not counted.
func Report(roles map[string]string, label string, accept map[string]string) (int, string) {
	bad := 0
	lines := []string{"  " + label}
	for _, c := range Checks {
		a, okA := roles[c.A]
		b, okB := roles[c.B]
		if !okA || !okB {
			continue
		}
		ratio := oklch.Contrast(a, b)
		ok := ratio >= c.Min && (c.Max == 0 || ratio <= c.Max)
		why := accept[label+"."+c.A]
		mark := "ok "
		if !ok {
			mark = "FAIL"
			if why != "" {
				mark = "ACCEPT"
			} else {
				bad++
			}
		}
		line := fmt.Sprintf("    %-4s %-12s on %-9s %5.2f  (want %s)  %s / %s", mark, c.A, c.B, ratio, c.want, a, b)
		if why != "" && !ok {
			line += "  — " + why
		}
		lines = append(lines, line)
	}
	return bad, strings.Join(lines, "\n")
}

// referenced are the roles a check reads: every role in Checks plus the inks the
// engine may put on a fill.
var referenced = func() map[string]bool {
	m := map[string]bool{"white": true}
	for _, c := range Checks {
		m[c.A], m[c.B] = true, true
	}
	return m
}()

var onFills = []struct{ on, fill string }{
	{"onPrimary", "primary"}, {"onSecondary", "secondary"}, {"onWarning", "warning"},
	{"onError", "error"}, {"onSuccess", "success"},
}

func same(a, b color.Color) bool {
	if a == nil || b == nil {
		return a == b
	}
	r1, g1, b1, a1 := a.RGBA()
	r2, g2, b2, a2 := b.RGBA()
	return r1 == r2 && g1 == g2 && b1 == b2 && a1 == a2
}

func hexOf(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02X%02X%02X", r>>8, g>>8, b>>8)
}

// FillOnColours adds the on-colours a theme leaves out, taking them from the kit's own
// engine (theme.Resolve), so the check sees the inks the app will draw. An on-colour is
// filled only when its fill role is present; fgSelected defaults to onPrimary. The ink
// is named by the role string it came from (white, bg, fg) where one matches.
func FillOnColours(r map[string]string, dark bool) error {
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	var roles theme.Roles
	if err := json.Unmarshal(b, &roles); err != nil {
		return err
	}
	mode := "dark"
	if !dark {
		mode = "light"
	}
	p := theme.Resolve(theme.File{Roles: map[string]theme.Roles{mode: roles}}, dark, false).P
	engine := map[string]color.Color{
		"onPrimary": p.OnPrimary, "onSecondary": p.OnSecondary, "onWarning": p.OnWarning,
		"onError": p.OnError, "onSuccess": p.OnSuccess,
	}
	name := func(c color.Color) string {
		for _, k := range []string{"white", "bg", "fg"} {
			if s, ok := r[k]; ok && s != "" && same(oklch.Color(s), c) {
				return s
			}
		}
		return hexOf(c)
	}
	for _, f := range onFills {
		if _, ok := r[f.on]; ok {
			continue
		}
		if _, ok := r[f.fill]; !ok {
			continue
		}
		r[f.on] = name(engine[f.on])
	}
	if _, ok := r["fgSelected"]; !ok {
		if v, ok := r["onPrimary"]; ok {
			r["fgSelected"] = v
		}
	}
	return nil
}

// Swatch renders the role sets as a 64-column .ansi file: each hue as a block and a
// sample line, the text ramp, a border rule, and buttons on their fills.
func Swatch(modes []Mode) string {
	sgr := func(code int, h string) string {
		c := oklch.MustRGB(h)
		return fmt.Sprintf("\x1b[%d;2;%d;%d;%dm", code, oklch.Round(c[0]*255), oklch.Round(c[1]*255), oklch.Round(c[2]*255))
	}
	fg := func(h string) string { return sgr(38, h) }
	bgc := func(h string) string { return sgr(48, h) }
	const w = 64
	var rows []string
	for _, m := range modes {
		r := m.Roles
		bg := bgc(r["bg"])
		line := func(s string, vis int) string {
			return bg + s + bg + strings.Repeat(" ", max(0, w-vis)) + "\x1b[0m"
		}
		rows = append(rows, line(fg(r["fgMuted"])+" "+m.Name, utf8.RuneCountInString(m.Name)+1))
		for _, k := range []string{"primary", "secondary", "tertiary", "accent", "success", "error", "warning", "info"} {
			txt := " ██ " + pyfmt.Ljust(k, 10) + " " + r[k] + "  "
			sample := "The quick brown fox"
			rows = append(rows, line(fg(r[k])+txt+sample, utf8.RuneCountInString(txt)+len(sample)))
		}
		for _, k := range []string{"fg", "fgMuted", "fgSubtle"} {
			txt := " Aa " + pyfmt.Ljust(k, 10) + " " + r[k] + "  "
			sample := "body text at this level"
			rows = append(rows, line(fg(r[k])+txt+sample, utf8.RuneCountInString(txt)+len(sample)))
		}
		rule := " " + fg(r["border"]) + strings.Repeat("─", 40) + fg(r["fgSubtle"]) + " border"
		rows = append(rows, line(rule, 48))
		pill := " " + bgc(r["secondary"]) + fg(r["onSecondary"]) + "  Save  " + bg + " " + bgc(r["bgSubtle"]) + fg(r["fg"]) +
			"  Cancel  " + bg + "  " + bgc(r["primary"]) + fg(r["onPrimary"]) + " selected row " + bg
		rows = append(rows, line(pill, 1+8+1+10+2+14))
		rows = append(rows, line("", 0))
	}
	return strings.Join(rows, "\n") + "\n"
}

func sortedRoles(r map[string]string) []string {
	keys := make([]string, 0, len(r))
	for k := range r {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// JSON renders the derive document exactly as palette.py's json.dumps(indent=1):
// {"name", "roles": {mode: {role: hex}}, "ansi256": {mode: {role: index}}}, roles sorted.
func JSON(name string, modes []Mode) string {
	var b strings.Builder
	b.WriteString("{\n \"name\": " + pyfmt.QuoteJSON(name) + ",\n")
	section := func(key string, value func(string) string, last bool) {
		b.WriteString(" " + pyfmt.QuoteJSON(key) + ": {\n")
		for i, m := range modes {
			b.WriteString("  " + pyfmt.QuoteJSON(m.Name) + ": {\n")
			keys := sortedRoles(m.Roles)
			for j, k := range keys {
				b.WriteString("   " + pyfmt.QuoteJSON(k) + ": " + value(m.Roles[k]))
				if j < len(keys)-1 {
					b.WriteByte(',')
				}
				b.WriteByte('\n')
			}
			b.WriteString("  }")
			if i < len(modes)-1 {
				b.WriteByte(',')
			}
			b.WriteByte('\n')
		}
		b.WriteString(" }")
		if !last {
			b.WriteByte(',')
		}
		b.WriteByte('\n')
	}
	section("roles", pyfmt.QuoteJSON, false)
	section("ansi256", func(h string) string { return fmt.Sprint(oklch.Xterm256(h)) }, true)
	b.WriteString("}")
	return b.String()
}
