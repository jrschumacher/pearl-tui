package theme

import (
	"image/color"
	"math"

	"charm.land/lipgloss/v2"
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
// ones present so a theme with only the core roles still renders well.
func derive(r Roles, dark bool) Palette {
	var p Palette
	c := func(s string, fallback color.Color) color.Color {
		if s == "" {
			return fallback
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
	ink := func(s string, fill color.Color) color.Color {
		if s != "" {
			return lipgloss.Color(s)
		}
		if white != nil && Contrast(white, fill) >= 4.5 {
			return white
		}
		return OnColor(fill, p.Bg, p.Fg)
	}
	p.OnPrimary = ink(r.OnPrimary, p.Primary)
	p.OnSecondary = ink(r.OnSecondary, p.Secondary)
	p.OnError = ink(r.OnError, p.Error)
	p.OnWarning = ink(r.OnWarning, p.Warning)
	p.OnSuccess = ink(r.OnSuccess, p.Success)
	p.FgSelected = c(r.FgSelected, p.OnPrimary)
	return p
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

// ASCII maps a glyph set onto tier-0 characters: TERM=linux, --ascii, or a
// font without the glyphs. Every glyph the theme defines must survive this.
func ASCII(g Glyphs) Glyphs {
	m := map[string]string{
		"╱": "/", "╲": "\\", "─": "-", "━": "=", "═": "=", "╌": "-", "┄": "-", "·": ".", "~": "~",
		"▌": "|", "┃": "|", "│": "|", "▎": "|", "❯": ">", "›": ">", "▸": ">", "→": "->",
		"✓": "+", "✔": "+", "●": "*", "◆": "*", "◇": "o", "○": "o", "◉": "(*)", "×": "x", "✗": "x", "✕": "x",
		"•": "*", "▪": "*", "…": "...", "■": "[x]", "□": "[ ]", "☑": "[x]", "☐": "[ ]",
		"├─": "|-", "╰─": "`-", "└─": "`-", "├──": "|--", "╰──": "`--", "▶": ">", "◌": ".", "▲": "^", "△": "^",
	}
	f := func(s string) string {
		if v, ok := m[s]; ok {
			return v
		}
		for _, r := range s {
			if r > 127 {
				return "*"
			}
		}
		return s
	}
	g.HeaderRule, g.Rule, g.FocusBar, g.BlurBar = f(g.HeaderRule), f(g.Rule), f(g.FocusBar), f(g.BlurBar)
	g.Cursor, g.Prompt, g.Done, g.Current, g.Todo = f(g.Cursor), f(g.Prompt), f(g.Done), f(g.Current), f(g.Todo)
	g.Fail, g.Pending, g.Bullet, g.Sep, g.Arrow, g.Ellipsis = f(g.Fail), f(g.Pending), f(g.Bullet), f(g.Sep), f(g.Arrow), f(g.Ellipsis)
	g.RadioOn, g.RadioOff, g.CheckOn, g.CheckOff = f(g.RadioOn), f(g.RadioOff), f(g.CheckOn), f(g.CheckOff)
	g.TreeMid, g.TreeEnd, g.TreeRail, g.Thumb, g.Track = f(g.TreeMid), f(g.TreeEnd), f(g.TreeRail), f(g.Thumb), f(g.Track)
	g.FieldA, g.FieldB = f(g.FieldA), f(g.FieldB)
	g.Spinner = []string{"|", "/", "-", "\\"}
	return g
}

func exprDefaults(x Expressions) Expressions {
	d := func(p *string, v string) {
		if *p == "" {
			*p = v
		}
	}
	d(&x.Header, "rule")
	d(&x.Wordmark, "gradient")
	d(&x.SectionTitle, "accent")
	d(&x.TitleCase, "sentence")
	d(&x.Selection, "bar")
	d(&x.Button, "pill")
	d(&x.Tag, "filled")
	d(&x.Dialog, "rounded")
	d(&x.DialogTitle, "rule")
	d(&x.Tabs, "underline")
	d(&x.Footer, "keys")
	d(&x.Keys, "inline")
	d(&x.Density, "airy")
	d(&x.Backdrop, "none")
	d(&x.Scrim, "dim")
	if x.Gutter == 0 {
		x.Gutter = 2
	}
	if x.Measure == 0 {
		x.Measure = 100
	}
	return x
}
