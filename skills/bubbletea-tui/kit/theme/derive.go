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

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// Default is the fallback theme: monochrome ink with a single cold accent.
// It exists so a partial theme renders. Never ship it: a product gets its own
// theme from its own brand (references/brand.md).
func Default() File {
	return File{
		Name:  "ink",
		Brand: "app",
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
// theme's ansi16 at 16 colours, else nil) replace the roles they name with one of the
// 16 colours, before anything is derived from them.
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
	p.Border = c(r.Border, Mix(p.Bg, p.Fg, 0.1))
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

// pinned writes each pinned role as "ansi:<n>", which derive's c reads as that basic
// colour: a hex can't name one (the xterm hex of ANSI 8 downsamples to 7), and a role
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
	if g.Partial == "" {
		g = defaultPartial(g)
	}
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
	if len(g.Spinner) == 0 {
		g.Spinner = []string{"◜", "◠", "◝", "◞", "◡", "◟"}
	}
	// HeaderRule and BlurBar may legitimately be "" (no rule / padding only).
	return g
}

// defaultPartial gives a theme that leaves glyphs.partial unset the kit's ◐. Its twin is
// "~", unless one of the theme's own status marks already draws as "~" (a ◔ pending
// written before the slot existed): then the first twin no status mark uses, so a
// default never makes two of the theme's states one mark under ASCII.
func defaultPartial(g Glyphs) Glyphs {
	g.Partial = "◐"
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
	"▌": "|", "┃": "|", "│": "|", "▎": "|", "❯": ">", "›": ">", "▸": ">", "→": "->",
	"✓": "+", "✔": "+", "●": "*", "◆": "*", "◇": "o", "○": "o", "◉": "(*)", "×": "x", "✗": "x", "✕": "x", "✖": "x",
	"•": "*", "▪": "*", "…": "...", "■": "[x]", "□": "[ ]", "☑": "[x]", "☐": "[ ]",
	"├─": "|-", "╰─": "`-", "└─": "`-", "├──": "|--", "╰──": "`--", "▶": ">", "◌": ".", "◐": "~", "▲": "^", "△": "o", "◢": "/",
	"◦": "o", "–": "-", "—": "-", "█": "#", "░": ".",
}

// ASCIITwin returns the twin ASCII draws for glyph, and whether the kit's map has one.
// tuilint spec reads it to skip glyphs the kit already covers.
func ASCIITwin(glyph string) (string, bool) {
	t, ok := asciiTwins[glyph]
	return t, ok
}

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
// (glyphMeanings) that differ but share a twin. palette check and the kit's tests
// fail on it; theme.Load doesn't, so an app never refuses to start over a glyph.
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
		if err != nil || asciiForced[name] || twin(g.ASCII, *p) != "?" {
			return
		}
		err = fmt.Errorf("glyphs.%s %q has no ASCII twin: -ascii and TERM=linux would draw it as \"?\"; add \"ascii\": {%q: \"<twin>\"} to glyphs", key, *p, *p)
	})
	if err != nil {
		return err
	}
	out := ASCII(src)
	sv, ov, t := reflect.ValueOf(src), reflect.ValueOf(out), reflect.TypeOf(src)
	jsonKey := func(name string) string {
		f, _ := t.FieldByName(name)
		k, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		return k
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
	return nil
}

// ASCII maps a glyph set onto tier-0 characters: TERM=linux, --ascii, or a
// font without the glyphs. Every glyph the theme defines must survive this.
func ASCII(g Glyphs) Glyphs {
	glyphFields(&g, func(_, _ string, p *string) { *p = twin(g.ASCII, *p) })
	g.Spinner = []string{"|", "/", "-", "\\"}
	// status glyphs must stay distinct in shape, or ok/warn/error become one mark
	g.Warn, g.Fail = "!", "x"
	if g.Todo == g.Current {
		g.Todo = "o"
	}
	if g.Current == g.Warn || g.Current == g.Fail {
		g.Current = "*"
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
	"scrim":        {"dim", "none"},
	"focus":        {"primary", "fg"},
}

// ExpressionValues are the values the kit knows for expression key (its JSON name),
// the default first; nil for a key that takes no word (gutter, measure, paintBg).
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
