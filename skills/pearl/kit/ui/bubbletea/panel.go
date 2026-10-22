package ui

import (
	"image/color"
	"regexp"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"

	"tuikit/theme/bubbletea"
)

// panelFloor is the contrast a text cell must keep on its fill (tuilint frames' F-CONTRAST).
const panelFloor = 2.3

// OnPanel re-inks a styled string for a filled panel: a neutral text level that drops
// under 2.3:1 on fill takes the next level up (fgSubtle to fgMuted, fgMuted to fgSoft or
// fg), so hints stay readable on bgPanel and bgOverlay whatever step the theme chose.
// ToolCall and Thinking draw their hints muted for the same reason; Dialog and Popover
// apply it to every row they hold. A rule in the border's ink that lands on the fill
// takes the first level that draws apart from it. A span that sets its own background
// (a filled button, a filled tag, a selected row) sits on that fill, not the panel's,
// and keeps its ink; the hue roles are left alone everywhere, a state's colour on a panel being
// the theme's own contrast decision.
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
	lift := map[string]string{}
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
		lift[fgParam(c)] = fgParam(to)
	}
	// a rule (a scrollbar's track, a section's rule) is quiet, not text: it needs only to
	// draw apart from the fill. At 16 colours a light page's border and bgOverlay are both
	// grey 7, and a palette's track vanished into the popover: it takes the first text
	// level that is apart, as quiet as it can be and still a rail
	apart := func(c color.Color) bool {
		cc, cf := prof.Convert(rgb8(c)), prof.Convert(rgb8(fill))
		return cc == nil || cf == nil || !same8(cc, cf)
	}
	if b := fgParam(t.P.Border); !apart(t.P.Border) && lift[b] == "" {
		for _, up := range levels {
			if apart(up) {
				lift[b] = fgParam(up)
				break
			}
		}
	}
	if len(lift) == 0 {
		return s
	}
	return sgrRe.ReplaceAllStringFunc(s, func(seq string) string {
		params := sgrParams(seq[2 : len(seq)-1])
		for _, p := range params {
			if strings.HasPrefix(p, "48;") || p == "48" || (len(p) == 2 && p[0] == '4' && p[1] >= '0' && p[1] <= '7') || (len(p) == 3 && strings.HasPrefix(p, "10")) {
				return seq // its own fill: judged there, not on the panel
			}
		}
		for i, p := range params {
			if to, ok := lift[p]; ok {
				params[i] = to
			}
		}
		return "\x1b[" + strings.Join(params, ";") + "m"
	})
}

var sgrRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// sgrParams splits an SGR parameter string into its attributes, keeping a colour and
// its arguments (38;2;R;G;B, 38;5;N, 48;…) as one.
func sgrParams(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ";")
	var out []string
	for i := 0; i < len(parts); i++ {
		p := parts[i]
		if (p == "38" || p == "48" || p == "58") && i+1 < len(parts) {
			n := 0
			switch parts[i+1] {
			case "2":
				n = 4
			case "5":
				n = 2
			}
			end := min(len(parts)-1, i+1+n)
			p = strings.Join(parts[i:end+1], ";")
			i = end
		}
		out = append(out, p)
	}
	return out
}

// fgParam is the SGR parameter Lip Gloss writes for a foreground c ("38;2;R;G;B" for a
// hex, "90" for a role pinned to one of the 16 colours), so a replacement matches what
// the styles wrote and nothing else.
func fgParam(c color.Color) string {
	s := lipgloss.NewStyle().Foreground(c).Render("x")
	seq, _, _ := strings.Cut(s, "x")
	return strings.TrimSuffix(strings.TrimPrefix(seq, "\x1b["), "m")
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
