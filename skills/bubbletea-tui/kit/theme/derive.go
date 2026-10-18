package theme

import (
	"fmt"
	"image/color"
	"math"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// Default is the fallback theme: monochrome ink with a single cold accent.
// It exists so a partial theme renders. Never ship it: a product gets its own
// theme from its own brand (references/brand.md). It is themes/ink.json as Go,
// one fallback in two forms (TestInkIsTheDefault holds them equal), so the showcase
// with no flag, which embeds ink.json, and a partial theme's roles draw alike. Ink's
// expressions are its own quiet choices (a solid wordmark, fg section titles, a plain
// dialog title: its two blues blend to an off-palette colour at 256 colours); a key a
// theme leaves unset still takes exprDefaults' first value, not ink's.
func Default() File {
	return File{
		Name:  "ink",
		Brand: "app",
		Read: "The fallback: monochrome ink with one cold accent, plain rule header, nothing to remember it by. " +
			"It exists so a partial theme renders. Never ship it; derive a theme from the product's own brand.",
		ANSI16: map[string]map[string]int{"dark": {"warning": 11}, "light": {"warning": 3}},
		Glyphs: Glyphs{Pending: "◌"}, // waiting is not now: the default current is ●
		Expressions: Expressions{Header: "rule", Wordmark: "solid", SectionTitle: "fg", Selection: "bar", Button: "pill",
			Tag: "filled", Dialog: "rounded", DialogTitle: "plain", Tabs: "underline", Footer: "keys", Density: "airy"},
		Roles: map[string]Roles{
			"dark": {
				Primary: "#8FB8DE", Secondary: "#5E81AC", Tertiary: "#A3D9C9", Accent: "#E5E9F0",
				Bg: "#15171C", Fg: "#D8DEE9", FgMuted: "#8A93A3", FgSubtle: "#5D6573",
				Success: "#8FD19E", Error: "#E8828A", Warning: "#EBCB8B", Info: "#8FB8DE",
			},
			"light": {
				Primary: "#2E5E8C", Secondary: "#3B5B85", Tertiary: "#2F7A68", Accent: "#1D232D",
				Bg: "#FAFAF9", Fg: "#1D232D", FgMuted: "#5B6472", FgSubtle: "#8E96A3",
				Success: "#2B7A43", Error: "#B3363F", Warning: "#8A6200", Info: "#2E5E8C",
			},
		},
	}
}

// derive resolves a role set into colours, filling any missing role from the
// ones present so a theme with only the core roles still renders well. pins (the
// theme's ansi16 at 16 colours, its ansi256 at 256, else nil) replace the roles they
// name with one of those colours, before anything is derived from them.
func derive(r Roles, dark bool, pins map[string]int) Palette {
	var p Palette
	if len(pins) > 0 {
		r = pinned(r, pins)
	}
	c := func(s string, fallback color.Color) color.Color {
		if s == "" {
			return fallback
		}
		if n, ok := strings.CutPrefix(s, "ansi:"); ok {
			i, _ := strconv.Atoi(n)
			if i > 15 {
				return ansi.IndexedColor(i) // an ansi256 pin
			}
			return ansi.BasicColor(i)
		}
		return lipgloss.Color(s)
	}
	defBg := lipgloss.Color("#15171C")
	defFg := lipgloss.Color("#D8DEE9")
	if !dark {
		defBg, defFg = lipgloss.Color("#FAFAF9"), lipgloss.Color("#1D232D")
	}
	p.Bg = c(r.Bg, defBg)
	p.Fg = c(r.Fg, defFg)
	p.Primary = c(r.Primary, Mix(p.Fg, p.Bg, 0.2))
	p.Secondary = c(r.Secondary, p.Primary)
	p.Tertiary = c(r.Tertiary, p.Primary)
	p.Accent = c(r.Accent, p.Fg)
	// background ramp: each step a little further from bg, toward fg
	p.BgSubtle = c(r.BgSubtle, Mix(p.Bg, p.Fg, 0.12))
	p.BgPanel = c(r.BgPanel, Mix(p.Bg, p.BgSubtle, 0.5))
	p.BgOverlay = c(r.BgOverlay, Mix(p.Bg, p.Fg, 0.2))
	// text ramp
	p.FgMuted = c(r.FgMuted, Mix(p.Fg, p.Bg, 0.45))
	p.FgSubtle = c(r.FgSubtle, Mix(p.Fg, p.Bg, 0.62))
	p.FgSoft = c(r.FgSoft, Mix(p.Fg, p.FgMuted, 0.3))
	p.Border = c(r.Border, edgeTint(p.Bg, p.Fg, 1.5))
	p.BorderFocus = c(r.BorderFocus, p.Primary)
	p.Success = c(r.Success, lipgloss.Color("#3DBE7F"))
	p.Error = c(r.Error, lipgloss.Color("#E5484D"))
	p.Warning = c(r.Warning, lipgloss.Color("#E2B93B"))
	p.Info = c(r.Info, p.Primary)
	white := c(r.White, nil)
	// floor is the contrast palette check asks of each on-colour on its fill.
	ink := func(s string, fill color.Color, floor float64) color.Color {
		if s != "" {
			return c(s, nil)
		}
		if white != nil && Contrast(white, fill) >= 4.5 {
			return white
		}
		return OnInk(fill, p.Bg, p.Fg, floor)
	}
	p.OnPrimary = ink(r.OnPrimary, p.Primary, 4.5)
	p.OnSecondary = ink(r.OnSecondary, p.Secondary, 3.5)
	p.OnError = ink(r.OnError, p.Error, 3.5)
	p.OnWarning = ink(r.OnWarning, p.Warning, 4.5)
	p.OnSuccess = ink(r.OnSuccess, p.Success, 3.5)
	p.FgSelected = c(r.FgSelected, p.OnPrimary)
	return p
}

// GreySlot16 is where a neutral role goes at 16 colours when the theme doesn't pin it:
// the four greys, by mode. bg is the page (0 dark, 15 light) and fg the opposite; fgSoft
// goes with fg. On a dark page the text ramp keeps three levels: fgMuted 7, fgSubtle 8.
// On a light page fgSubtle joins fgMuted on 8: ANSI 7 is 1.3:1 on xterm's white and
// 1.8:1 on VGA's, below every floor, so a light page keeps two levels (fgSubtle is never
// the only carrier of meaning). bgSubtle takes the grey nearer bg (8 dark, 7 light);
// bgPanel sits on the page (Distinct then gives a panel an edge), bgOverlay and border on
// bgSubtle's grey, the one edge picks for a rule that lands on the page; borderFocus,
// the one dialog's edge, on fgMuted's grey (7 dark, 8 light), a step up from a rule. A
// theme that wants its hue on the dialog's edge at 16 colours pins borderFocus to
// primary's slot. -1 for a role that isn't a neutral.
// Downsampling by distance puts a tinted neutral on a hue instead (a warm grey on red, a
// cool one on blue): hints then read as errors and the page turns yellow. palette check
// reports a pin that undoes this (NeutralRoles, ANSI16Neutrals).
func GreySlot16(role string, dark bool) int {
	n, ok := greys16[dark][role]
	if !ok {
		return -1
	}
	return n
}

// greys16 is GreySlot16's table, by dark.
var greys16 = map[bool]map[string]int{
	true:  {"bg": 0, "bgPanel": 0, "bgSubtle": 8, "bgOverlay": 8, "border": 8, "borderFocus": 7, "fg": 15, "fgSoft": 15, "fgMuted": 7, "fgSubtle": 8},
	false: {"bg": 15, "bgPanel": 15, "bgSubtle": 7, "bgOverlay": 7, "border": 7, "borderFocus": 8, "fg": 0, "fgSoft": 0, "fgMuted": 8, "fgSubtle": 8},
}

// pins16 are the roles derive replaces at 16 colours in mode: every neutral on its grey
// (GreySlot16), then the theme's own ansi16 pins, which win.
func pins16(f File, mode string, dark bool) map[string]int {
	pins := make(map[string]int, len(NeutralRoles)+len(f.ANSI16[mode]))
	for _, role := range NeutralRoles {
		pins[role] = GreySlot16(role, dark)
	}
	for role, n := range f.ANSI16[mode] {
		pins[role] = n
	}
	return pins
}

// pinned writes each pinned role as "ansi:<n>", which derive's c reads as that basic
// colour (0-15) or indexed one (16-255): a hex can't name one (the xterm hex of ANSI 8 downsamples to 7), and a role
// derived from a pinned one then follows it.
func pinned(r Roles, pins map[string]int) Roles {
	v, t := reflect.ValueOf(&r).Elem(), reflect.TypeOf(r)
	for i := range t.NumField() {
		key, _, _ := strings.Cut(t.Field(i).Tag.Get("json"), ",")
		if n, ok := pins[key]; ok && key != "white" {
			v.Field(i).SetString("ansi:" + strconv.Itoa(n))
		}
	}
	return r
}

// edgeTint is the first step of the ramp from bg toward fg that reads at floor against
// bg (1.5:1 for a border: visible but quiet, palette check's warning floor; bg one step
// off, 1.2:1, was a rule nobody saw), or fg when none does.
func edgeTint(bg, fg color.Color, floor float64) color.Color {
	for _, step := range lipgloss.Blend1D(101, bg, fg)[1:] {
		if c := rgb8(step); Contrast(c, bg) >= floor {
			return c
		}
	}
	return fg
}

// Mix blends a toward b by t (0 = a, 1 = b) in CIELAB, the way Crush derives
// diff tints from one hue: lipgloss.Blend1D does the perceptual maths.
func Mix(a, b color.Color, t float64) color.Color {
	if t <= 0 {
		return a
	}
	if t >= 1 {
		return b
	}
	steps := 101
	ramp := lipgloss.Blend1D(steps, a, b)
	return ramp[int(math.Round(t*float64(steps-1)))]
}

// Luminance is WCAG relative luminance.
func Luminance(c color.Color) float64 {
	r, g, b, _ := c.RGBA()
	lin := func(v uint32) float64 {
		x := float64(v) / 65535
		if x <= 0.04045 {
			return x / 12.92
		}
		return math.Pow((x+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(r) + 0.7152*lin(g) + 0.0722*lin(b)
}

// promptFloor is the contrast the focus ink needs on bgOverlay to draw the prompt: the
// prompt is where you type, a focus cue, and a focus indicator holds 3:1 (WCAG 1.4.11).
const promptFloor = 3.0

// Contrast is the WCAG contrast ratio between two colours.
func Contrast(a, b color.Color) float64 {
	la, lb := Luminance(a), Luminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// OnColor picks whichever of two inks reads better on a fill.
func OnColor(fill, inkA, inkB color.Color) color.Color {
	if Contrast(inkA, fill) >= Contrast(inkB, fill) {
		return inkA
	}
	return inkB
}

// OnInk is the ink for text on a filled role. It is OnColor's pick when that clears
// floor. On a mid-luminance fill where neither ink does, the pick is pushed toward
// black or white (whichever side of the fill it is on) in 1% CIELAB steps, keeping its
// hue, until it clears; if even that end cannot, the other ink is pushed toward the
// other end. Pure black or white reads at least 4.58:1 on any fill, so a floor up to
// 4.5 is always met and the brand's fill never moves. palette derive picks its
// on-colours with this same function.
func OnInk(fill, inkA, inkB color.Color, floor float64) color.Color {
	ink := OnColor(fill, inkA, inkB)
	if Contrast(ink, fill) >= floor {
		return ink
	}
	other := inkB
	if ink == inkB {
		other = inkA
	}
	end, otherEnd := color.Color(color.White), color.Color(color.Black)
	if Luminance(ink) < Luminance(fill) {
		end, otherEnd = otherEnd, end
	}
	start := ink
	if Contrast(end, fill) < floor {
		start, end = other, otherEnd
	}
	ramp := lipgloss.Blend1D(101, start, end) // Mix's ramp: ramp[i] is Mix(start, end, i/100)
	for _, step := range ramp[1:100] {
		c := rgb8(step)
		if Contrast(c, fill) >= floor {
			return c
		}
	}
	return rgb8(end)
}

// rgb8 truncates a colour to 8 bits a channel, so the colour drawn is the one checked.
func rgb8(c color.Color) color.RGBA {
	r, g, b, _ := c.RGBA()
	return color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: 0xff}
}

func glyphDefaults(g Glyphs) Glyphs {
	d := func(p *string, v string) {
		if *p == "" {
			*p = v
		}
	}
	d(&g.Rule, "─")
	d(&g.FocusBar, "▌")
	d(&g.Cursor, "›")
	d(&g.Prompt, ">")
	d(&g.Done, "✓")
	d(&g.Current, "●")
	d(&g.Todo, "○")
	d(&g.Fail, "×")
	d(&g.Pending, "●")
	d(&g.Warn, "▲")
	d(&g.Bullet, "•")
	d(&g.Sep, "·")
	d(&g.Arrow, "→")
	d(&g.Ellipsis, "…")
	d(&g.RadioOn, "◉")
	d(&g.RadioOff, "○")
	d(&g.CheckOn, "■")
	d(&g.CheckOff, "□")
	d(&g.TreeMid, "├─")
	d(&g.TreeEnd, "╰─")
	d(&g.TreeRail, "│")
	d(&g.Thumb, "┃")
	d(&g.Track, "│")
	d(&g.FieldA, "◆")
	d(&g.FieldB, "◇")
	d(&g.Caret, "▏")
	d(&g.More, "↑")
	d(&g.UpDown, "↑↓")
	d(&g.Up, "↑")
	d(&g.Down, "↓")
	d(&g.LeftRight, "←→")
	d(&g.Back, "←")
	d(&g.ProgressFill, "━")
	d(&g.ProgressEmpty, "╌")
	d(&g.ProgressHead, "╸")
	d(&g.TurnBar, "│")
	d(&g.TabUnderline, "▔")
	if len(g.Spinner) == 0 {
		g.Spinner = []string{"◜", "◠", "◝", "◞", "◡", "◟"}
	}
	if g.Partial == "" {
		g = defaultPartial(g)
	}
	// HeaderRule and BlurBar may legitimately be "" (no rule / padding only).
	return g
}

// defaultPartial gives a theme that leaves glyphs.partial unset the first of ◐ ◑ ◒ ◓ ◔
// that none of its status marks or spinner frames already is, so the in-progress rung
// never draws as pending (a ◐ pending) or as a spinner frame. Its twin is the kit's
// ("~" for ◐), unless one of the theme's own status marks already draws as that: then
// the first twin no status mark uses, so a default never makes two of the theme's states
// one mark under ASCII.
func defaultPartial(g Glyphs) Glyphs {
	taken := map[string]bool{}
	for _, s := range append([]string{g.Done, g.Current, g.Todo, g.Pending, g.Warn, g.Fail}, g.Spinner...) {
		taken[strings.TrimSpace(s)] = true
	}
	g.Partial = "◐"
	for _, c := range []string{"◐", "◑", "◒", "◓", "◔"} {
		if !taken[c] {
			g.Partial = c
			break
		}
	}
	if _, own := g.ASCII[g.Partial]; own {
		return g
	}
	used := map[string]bool{"!": true, "x": true} // warn and fail, as ASCII draws them
	for _, s := range []string{g.Done, g.Current, g.Todo, g.Pending} {
		if s != "" && s != g.Partial {
			used[twin(g.ASCII, s)] = true
		}
	}
	for _, alt := range []string{"~", "%", "=", "&"} {
		if !used[alt] {
			if alt != asciiTwins[g.Partial] {
				own := make(map[string]string, len(g.ASCII)+1)
				for k, v := range g.ASCII {
					own[k] = v
				}
				own[g.Partial] = alt
				g.ASCII = own
			}
			break
		}
	}
	return g
}

// asciiTwins is ASCII's map: each glyph the kit draws and its tier-0 twin. A theme
// adds its own in glyphs.ascii.
var asciiTwins = map[string]string{
	"╱": "/", "╲": "\\", "─": "-", "━": "=", "═": "=", "╌": "-", "┄": "-", "·": ".", "~": "~",
	"▌": "|", "┃": "|", "│": "|", "▎": "|", "╎": ":", "┆": ":", "┊": ":", "❯": ">", "›": ">", "▸": ">", "→": "->",
	"✓": "+", "✔": "+", "●": "*", "◆": "*", "◇": "o", "○": "o", "◉": "(*)", "×": "x", "✗": "x", "✕": "x", "✖": "x",
	"•": "*", "▪": "*", "…": "...", "■": "[x]", "□": "[ ]", "☑": "[x]", "☐": "[ ]", "»": ">>", "«": "<<",
	"├─": "|-", "╰─": "`-", "└─": "`-", "├──": "|--", "╰──": "`--", "▶": ">", "◌": ".", "◐": "~", "▲": "^", "△": "o", "◢": "/",
	"◦": "o", "–": "-", "—": "-", "█": "#", "░": ".",
	"▏": "'", "↑": "^", "↓": "v", "↑↓": "^v", "←": "<-", "←→": "<>", "╸": "=", "▔": "-",
}

// ASCIITwin returns the twin ASCII draws for glyph, and whether the kit's map has one.
// tuilint spec reads it to skip glyphs the kit already covers.
func ASCIITwin(glyph string) (string, bool) {
	t, ok := asciiTwins[glyph]
	return t, ok
}

// blurFallback is the blurred selection's marker when glyphs.blurBar is unset
// (Theme.BlurBar), and its ASCII twin: a dashed bar beside the focus bar's solid one.
const blurFallback, blurFallbackASCII = "╎", ":"

// asciiForced are the glyphs ASCII replaces whatever the theme says: status marks
// that must stay distinct in shape. They need no twin.
var asciiForced = map[string]bool{"Warn": true, "Fail": true}

// glyphFields calls fn with every string field of g (by Go name and JSON key) and a
// pointer to it, so a glyph field added to Glyphs is mapped and checked like the rest.
func glyphFields(g *Glyphs, fn func(name, key string, p *string)) {
	v, t := reflect.ValueOf(g).Elem(), reflect.TypeOf(*g)
	for i := range t.NumField() {
		if t.Field(i).Type.Kind() != reflect.String {
			continue
		}
		key, _, _ := strings.Cut(t.Field(i).Tag.Get("json"), ",")
		fn(t.Field(i).Name, key, v.Field(i).Addr().Interface().(*string))
	}
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > 127 {
			return false
		}
	}
	return true
}

// twin is the glyph ASCII draws for s: the theme's twin, the kit's, s itself when it
// is already ASCII, else "?": loud (F-ASCII fails it, CheckGlyphs names it) and never
// another mark's twin.
func twin(own map[string]string, s string) string {
	if v, ok := own[s]; ok {
		return v
	}
	if v, ok := asciiTwins[s]; ok {
		return v
	}
	if isASCII(s) {
		return s
	}
	return "?"
}

// glyphMeanings are the glyph fields that mark different states of one thing. Under
// ASCII two of a set that differ as glyphs must differ as twins too, or two states
// become one mark (a ◔ reading and a ● new both "*"). warn and fail are in the status
// set as ASCII draws them, "!" and "x".
var glyphMeanings = [][]string{
	{"Done", "Current", "Todo", "Pending", "Partial", "Warn", "Fail"}, // a step's or an item's state
	{"RadioOn", "RadioOff"},
	{"CheckOn", "CheckOff"},
	{"Thumb", "Track"},
	{"FocusBar", "BlurBar"},
}

// CheckGlyphs reports a theme whose glyphs ASCII can't draw faithfully: a twin in
// glyphs.ascii that isn't ASCII itself, a glyph outside ASCII with no twin in the
// theme's map or the kit's (it would draw as "?"), or two glyphs of one meaning set
// (glyphMeanings) that differ but share a twin; and a spinner frame that is, or whose
// cells hold, a status mark, a bar or the caret (spinnerMarks), which a working row
// beside them would read as. palette check and the kit's tests fail on it; theme.Load
// doesn't, so an app never refuses to start over a glyph.
func CheckGlyphs(g Glyphs) error {
	keys := make([]string, 0, len(g.ASCII))
	for k := range g.ASCII {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if v := g.ASCII[k]; v == "" || !isASCII(v) {
			return fmt.Errorf("glyphs.ascii: %q → %q is not ASCII: a twin is what TERM=linux draws, so write it in ASCII", k, v)
		}
	}
	var err error
	src := glyphDefaults(g)
	glyphFields(&src, func(name, key string, p *string) {
		if err != nil || asciiForced[name] || twin(src.ASCII, *p) != "?" {
			return
		}
		err = fmt.Errorf("glyphs.%s %q has no ASCII twin: -ascii and TERM=linux would draw it as \"?\"; add \"ascii\": {%q: \"<twin>\"} to glyphs", key, *p, *p)
	})
	if err != nil {
		return err
	}
	// the product's own marks: each with a twin, and no two that differ on one twin (a
	// progress pair • ◦ drawn as one ASCII mark has no progress)
	names := make([]string, 0, len(g.Marks))
	for n := range g.Marks {
		names = append(names, n)
	}
	sort.Strings(names)
	twinOf := map[string]string{}
	for _, n := range names {
		m := g.Marks[n]
		if n == "" || m == "" {
			return fmt.Errorf("glyphs.marks: %q → %q: name a mark and give it a glyph", n, m)
		}
		tw := twin(g.ASCII, m)
		if tw == "?" {
			return fmt.Errorf("glyphs.marks.%s %q has no ASCII twin: -ascii and TERM=linux would draw it as \"?\"; add \"ascii\": {%q: \"<twin>\"} to glyphs", n, m, m)
		}
		if other, dup := twinOf[tw]; dup && g.Marks[other] != m {
			return fmt.Errorf("glyphs.marks.%s %q and glyphs.marks.%s %q both draw as %q under ASCII, so two marks read as one: give one of them another twin in glyphs.ascii", other, g.Marks[other], n, m, tw)
		}
		twinOf[tw] = n
	}
	out := ASCII(src)
	sv, ov, t := reflect.ValueOf(src), reflect.ValueOf(out), reflect.TypeOf(src)
	jsonKey := func(name string) string {
		f, _ := t.FieldByName(name)
		k, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		return k
	}
	// a mark is no status mark under ASCII either: reading dots • ◦ twin to * o, the
	// current and todo marks, so a row of them reads as a step ladder once colour is gone
	for _, n := range names {
		for _, s := range glyphMeanings[0] {
			if m, gs := g.Marks[n], sv.FieldByName(s).String(); m != gs && twin(g.ASCII, m) == ov.FieldByName(s).String() {
				return fmt.Errorf("glyphs.marks.%s %q draws as %q under ASCII, glyphs.%s %q's twin, so the mark reads as that state: give it another twin in glyphs.ascii",
					n, m, twin(g.ASCII, m), jsonKey(s), gs)
			}
		}
	}
	sv0 := reflect.ValueOf(src)
	for _, fr := range src.Spinner {
		for _, m := range spinnerMarks {
			if m == "Partial" && g.Partial == "" {
				continue // the default steps aside (defaultPartial); only a partial the theme wrote counts
			}
			if mark := sv0.FieldByName(m).String(); mark != "" && strings.TrimSpace(fr) == mark {
				return fmt.Errorf("glyphs.spinner frame %q is also glyphs.%s: a working row beside it would read as that mark; pick frames no state or focus mark uses",
					fr, jsonKey(m))
			}
		}
		// a multi-cell frame draws each of its cells beside a status column or a bar:
		// a dots spinner "···" beside a "·" pending draws the pending mark three times
		for _, r := range fr {
			if unicode.IsSpace(r) {
				continue
			}
			for _, m := range spinnerMarks {
				if m == "Partial" && g.Partial == "" {
					continue
				}
				if mark := sv0.FieldByName(m).String(); mark != "" && string(r) == mark {
					return fmt.Errorf("glyphs.spinner frame %q contains glyphs.%s %q: a working row beside a status column or the caret would read as that mark; pick frames whose cells are no state or focus mark",
						fr, jsonKey(m), mark)
				}
			}
		}
	}
	for _, set := range glyphMeanings {
		for i, a := range set {
			for _, b := range set[i+1:] {
				ga, gb := sv.FieldByName(a).String(), sv.FieldByName(b).String()
				ta, tb := ov.FieldByName(a).String(), ov.FieldByName(b).String()
				if ga == "" || gb == "" || ga == gb || ta != tb {
					continue
				}
				return fmt.Errorf("glyphs.%s %q and glyphs.%s %q both draw as %q under ASCII, so two states read as one: give one of them another twin in glyphs.ascii",
					jsonKey(a), ga, jsonKey(b), gb, ta)
			}
		}
	}
	// a blurred selection keeps its marker: as the focus bar, or its twin, the frame
	// shows two focuses once colour is gone
	switch {
	case src.BlurBar != "" && src.BlurBar == src.FocusBar:
		return fmt.Errorf("glyphs.blurBar %q is the focusBar: a blurred selection in the focus bar's shape is a second focus under NO_COLOR and ASCII; pick ╎ ┆ or ┊ (ASCII :), or leave it unset for ╎", src.BlurBar)
	case src.BlurBar == "" && (src.FocusBar == blurFallback || out.FocusBar == blurFallbackASCII):
		return fmt.Errorf("glyphs.focusBar %q is the kit's blurred marker (%s, ASCII %s): set glyphs.blurBar to another glyph, or pick another focus bar", src.FocusBar, blurFallback, blurFallbackASCII)
	}
	// the caret sits after typed text; as a bar, or a bar's twin, "> ro|" reads as a
	// focused block
	for _, bar := range []string{"FocusBar", "BlurBar"} {
		if b := sv.FieldByName(bar).String(); b != "" && (src.Caret == b || out.Caret == ov.FieldByName(bar).String()) {
			return fmt.Errorf("glyphs.caret %q draws as %q, the %s's: a caret that is a bar reads as focus; pick ▏ or _ (ASCII ')", src.Caret, out.Caret, jsonKey(bar))
		}
	}
	return nil
}

// CheckSelectionGlyphs reports a theme whose prompt is its cursor under selection
// "caret": a filter input above a caret-selected list (a palette, a picker) then
// shows the one glyph twice, where you type and where you are, and so does its ASCII
// twin. Under bar, fill and inverse the cursor marks only selects and buttons, so
// one glyph for both is fine (Crush's >). palette check and the kit's tests fail on
// it; theme.Load doesn't.
func CheckSelectionGlyphs(f File) error {
	if exprDefaults(f.Expressions).Selection != "caret" {
		return nil
	}
	g := glyphDefaults(f.Glyphs)
	a := ASCII(g)
	switch {
	case g.Prompt == g.Cursor:
		return fmt.Errorf("glyphs.prompt %q is glyphs.cursor under selection \"caret\": a filter input above a list shows it twice, where you type and where you are; give the prompt its own glyph with an ASCII twin that is not the cursor's (» draws as >>, or $)", g.Prompt)
	case a.Prompt == a.Cursor:
		return fmt.Errorf("glyphs.prompt %q and glyphs.cursor %q both draw as %q under ASCII, so a filter input above a caret-selected list shows one mark twice: give the prompt a twin of its own in glyphs.ascii", g.Prompt, g.Cursor, a.Prompt)
	}
	return nil
}

// ASCII maps a glyph set onto tier-0 characters: TERM=linux, --ascii, or a
// font without the glyphs. Every glyph the theme defines must survive this.
func ASCII(g Glyphs) Glyphs {
	glyphFields(&g, func(_, _ string, p *string) { *p = twin(g.ASCII, *p) })
	// status glyphs must stay distinct in shape, or ok/warn/error become one mark
	g.Warn, g.Fail = "!", "x"
	if g.Todo == g.Current {
		g.Todo = "o"
	}
	if g.Current == g.Warn || g.Current == g.Fail {
		g.Current = "*"
	}
	g.Spinner = asciiSpinner(g)
	if len(g.Marks) > 0 {
		marks := make(map[string]string, len(g.Marks))
		for name, m := range g.Marks {
			marks[name] = twin(g.ASCII, m)
		}
		g.Marks = marks
	}
	// ┃ thumb and │ track both twin to "|": without colour the scrollbar would have no
	// position, so the thumb becomes the solid block it stands for
	if g.Thumb == g.Track && g.Thumb != "" {
		g.Thumb = "#"
	}
	return g
}

// exprValues are the values the kit draws for each string expression, by JSON key; the
// first is the default. Anything else (unset, a typo, a value from a newer kit) resolves
// to the default, so a theme always renders; CheckExpressions names it.
var exprValues = map[string][]string{
	"header":       {"rule", "bar", "minimal", "stamp", "banner"},
	"wordmark":     {"gradient", "solid", "inverse", "tracked", "bracket", "upper", "plain"},
	"sectionTitle": {"accent", "primary", "muted-upper", "fg"},
	"titleCase":    {"sentence", "upper", "lower"},
	"labelCase":    {"", "upper", "lower", "as-is"}, // "": each component's own case
	"selection":    {"bar", "fill", "caret", "inverse"},
	"button":       {"pill", "bracket", "underline", "inverse"},
	"tag":          {"filled", "bracket", "bold"},
	"dialog":       {"rounded", "square", "double", "heavy", "none", "rules"},
	"dialogTitle":  {"rule", "plain", "bar"},
	"tabs":         {"underline", "pill", "bracket", "slash"},
	"footer":       {"keys", "bar", "rule"},
	"keys":         {"inline", "boxed"},
	"density":      {"airy", "compact"},
	"backdrop":     {"none", "field"},
	"working":      {"spinner", "scramble"},
	"spinner":      {"focus", "primary", "secondary", "tertiary", "accent", "info", "fg", "muted"},
	"scrim":        {"dim", "none"},
	"focus":        {"primary", "fg"},
}

// ExpressionValues are the values the kit knows for expression key (its JSON name),
// the default first; nil for a key that takes no word (gutter, measure, paintBg). The
// one default that isn't a word is labelCase's "": unset, each component keeps its own
// case (table headers upper, tags as written, summary labels lower).
func ExpressionValues(key string) []string { return exprValues[key] }

// ExpressionKey reports whether key is an expression the kit reads.
func ExpressionKey(key string) bool {
	t := reflect.TypeOf(Expressions{})
	for i := range t.NumField() {
		if k, _, _ := strings.Cut(t.Field(i).Tag.Get("json"), ","); k == key {
			return true
		}
	}
	return false
}

// exprFields calls fn with every string expression's JSON key and a pointer to it.
func exprFields(x *Expressions, fn func(key string, p *string)) {
	v, t := reflect.ValueOf(x).Elem(), reflect.TypeOf(*x)
	for i := range t.NumField() {
		if t.Field(i).Type.Kind() != reflect.String {
			continue
		}
		key, _, _ := strings.Cut(t.Field(i).Tag.Get("json"), ",")
		fn(key, v.Field(i).Addr().Interface().(*string))
	}
}

// CheckExpressions reports an expression value the kit doesn't know, with the values it
// does: a typo ("rule" for dialog "rules") or a theme written for a newer kit would
// otherwise render the default silently. palette check and the kit's tests fail on it;
// theme.Load doesn't, so an app always starts (Resolve draws the default).
func CheckExpressions(x Expressions) error {
	var err error
	exprFields(&x, func(key string, p *string) {
		if err != nil || *p == "" || len(exprValues[key]) == 0 || slices.Contains(exprValues[key], *p) {
			return
		}
		known, def := exprValues[key], exprValues[key][0]
		if def == "" {
			known, def = known[1:], "as if unset"
		}
		err = fmt.Errorf("expressions.%s %q is not a value the kit knows: %s (it would draw %s)",
			key, *p, strings.Join(known, " | "), def)
	})
	return err
}

// spinnerMarks are the glyphs a working spinner sits beside, in a status column, next
// to a focus bar or at the end of an input: no frame, nor a cell of one, may be one of
// them, or a working row reads as that mark (the ASCII caret "|" is the line spinner's
// first frame, so asciiSpinner drops it).
var spinnerMarks = []string{"FocusBar", "BlurBar", "Done", "Current", "Todo", "Fail", "Warn", "Pending", "Partial", "Caret"}

// asciiSpinner is the twin of g's spinner, g's marks already twinned. A spinner wider
// than one cell keeps its frames' width and shape (multiCellSpinner): a three-cell dots
// spinner stays three cells, never a one-cell line. A one-cell spinner is the line
// spinner | / - \ without the frames g's marks already draw as (the "|" of a focus bar
// twinned from ▌): a turning line still, a frame short. If the marks leave it fewer
// than two distinct frames, a pulse . : ' :, then a bounce ( ), then any two
// punctuation marks no mark is.
func asciiSpinner(g Glyphs) []string {
	v := reflect.ValueOf(g)
	taken := map[string]bool{}
	for _, m := range spinnerMarks {
		taken[v.FieldByName(m).String()] = true
	}
	if out := multiCellSpinner(g.ASCII, g.Spinner, taken); out != nil {
		return out
	}
	for _, frames := range [][]string{{"|", "/", "-", "\\"}, {".", ":", "'", ":"}, {"(", ")"}} {
		var out []string
		distinct := map[string]bool{}
		for _, f := range frames {
			if !taken[f] {
				out = append(out, f)
				distinct[f] = true
			}
		}
		if len(distinct) >= 2 {
			return out
		}
	}
	var out []string
	for _, r := range "+=#@%&<>^~;,`" {
		if f := string(r); !taken[f] && len(out) < 2 {
			out = append(out, f)
		}
	}
	return out
}

// spinnerCells are the one-cell twins a multi-cell spinner's glyph takes when its own
// twin is a mark's or another glyph's of the frames: a dot, then denser shapes.
const spinnerCells = ".:=#o+*~-^;,"

// multiCellSpinner twins a spinner whose frames are wider than one cell cell for cell,
// so each frame keeps its width and shape (··· •·· ••· ••• → ... :.. ::. ::: when "."
// and ":" are free). A glyph takes its own twin (own map, then the kit's) unless that
// is taken (a status, bar or caret mark's twin), wider than one cell, or another glyph
// of the frames already has it; then the first free of spinnerCells. Spaces stay. nil
// for a one-cell spinner, or when no free twin is left for a glyph.
func multiCellSpinner(own map[string]string, frames []string, taken map[string]bool) []string {
	wide := false
	for _, f := range frames {
		wide = wide || ansi.StringWidth(f) > 1
	}
	if !wide {
		return nil
	}
	twins, used := map[rune]string{}, map[string]bool{}
	for _, f := range frames {
		for _, r := range f {
			if _, ok := twins[r]; ok || unicode.IsSpace(r) {
				continue
			}
			pick := ""
			for _, c := range append([]string{twin(own, string(r))}, strings.Split(spinnerCells, "")...) {
				if len(c) == 1 && c != "?" && !taken[c] && !used[c] {
					pick = c
					break
				}
			}
			if pick == "" {
				return nil
			}
			twins[r], used[pick] = pick, true
		}
	}
	out := make([]string, len(frames))
	for i, f := range frames {
		var b strings.Builder
		for _, r := range f {
			if unicode.IsSpace(r) {
				b.WriteRune(r)
				continue
			}
			b.WriteString(twins[r])
			b.WriteString(strings.Repeat(" ", ansi.StringWidth(string(r))-1)) // a wide glyph keeps its cells
		}
		out[i] = b.String()
	}
	return out
}

func exprDefaults(x Expressions) Expressions {
	exprFields(&x, func(key string, p *string) {
		if vals := exprValues[key]; len(vals) > 0 && !slices.Contains(vals, *p) {
			*p = vals[0]
		}
	})
	if x.Gutter == 0 {
		x.Gutter = 2
	}
	if x.Measure == 0 {
		x.Measure = 100
	}
	return x
}
