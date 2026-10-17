package ui

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"

	"tuikit/theme"
)

// panelFloor is the contrast a text cell must keep on its fill (tuilint frames' F-CONTRAST).
const panelFloor = 2.3

// OnPanel re-inks a styled string for a filled panel: a neutral text level that drops
// under 2.3:1 on fill takes the next level up (fgSubtle to fgMuted, fgMuted to fgSoft or
// fg), so hints stay readable on bgPanel and bgOverlay whatever step the theme chose.
// ToolCall and Thinking draw their hints muted for the same reason; Dialog and Popover
// apply it to every row they hold. The hue roles are left alone: a state's colour on a
// panel is the theme's own contrast decision.
func OnPanel(t *theme.Theme, s string, fill color.Color) string {
	// judged as the terminal shows them: at 16 or 256 colours a level and the fill can
	// land on one index (both ANSI 8), which truecolour contrast would call readable
	prof := t.Profile
	if prof == colorprofile.Unknown {
		prof = colorprofile.TrueColor
	}
	holds := func(c color.Color) bool {
		// as written, 8 bits a channel, before the profile's nearest-colour pick: a
		// derived role is a blend with fractional channels the terminal never sees
		cc, cf := prof.Convert(rgb8(c)), prof.Convert(rgb8(fill))
		if cc == nil || cf == nil {
			return true // no colour at all: nothing to re-ink
		}
		return !same8(cc, cf) && theme.Contrast(cc, cf) >= panelFloor
	}
	levels := []color.Color{t.P.FgSubtle, t.P.FgMuted, t.P.FgSoft, t.P.Fg}
	for i, c := range levels[:3] {
		if holds(c) {
			continue
		}
		to := levels[len(levels)-1]
		for _, up := range levels[i+1:] {
			if holds(up) {
				to = up
				break
			}
		}
		s = strings.ReplaceAll(s, sgrFg(c), sgrFg(to))
	}
	return s
}

// sgrFg is the foreground sequence Lip Gloss writes for c ("\x1b[38;2;R;G;Bm" for a
// hex, "\x1b[90m" for a role pinned to one of the 16 colours), so a replacement
// matches what the styles wrote and nothing else.
func sgrFg(c color.Color) string {
	s := lipgloss.NewStyle().Foreground(c).Render("x")
	seq, _, _ := strings.Cut(s, "x")
	return seq
}

// rgb8 is c as Lip Gloss writes it, 8 bits a channel.
func rgb8(c color.Color) color.RGBA {
	r, g, b, _ := c.RGBA()
	return color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: 0xff}
}

// same8 compares two colours as a terminal holds them, 8 bits a channel.
func same8(a, b color.Color) bool {
	if a == nil || b == nil {
		return a == b
	}
	r1, g1, b1, _ := a.RGBA()
	r2, g2, b2, _ := b.RGBA()
	return r1>>8 == r2>>8 && g1>>8 == g2>>8 && b1>>8 == b2>>8
}
