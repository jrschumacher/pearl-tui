package theme

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/rivo/uniseg"
)

// Styles are the ready-made styles, all derived from the palette. Components
// read these; nothing else in the program builds a colour.
type Styles struct {
	// text levels: hierarchy comes from these four steps, not from colour
	Base, Soft, Muted, Subtle, Bold lipgloss.Style
	// roles as text
	Primary, Secondary, Tertiary, Accent lipgloss.Style
	OK, Err, Warn, Info                  lipgloss.Style
	// structure
	Title   lipgloss.Style // section titles
	Rule    lipgloss.Style // rules (border colour)
	Panel   lipgloss.Style // a recessed block: bgPanel, no border
	Key     lipgloss.Style // help: the key
	KeyDesc lipgloss.Style // help: what it does
	// selection
	SelRow   lipgloss.Style // a selected row (per expressions.selection)
	FocusBar lipgloss.Style // ▌ beside the focused block
	BlurBar  lipgloss.Style // the same column, unfocused
	Match    lipgloss.Style // fuzzy-match characters
	// controls
	Button, ButtonFocus, ButtonDanger lipgloss.Style
	Dialog                            lipgloss.Style
	// tags (chips): a word on a filled block, or bold coloured text
	TagPrimary, TagOK, TagErr, TagWarn, TagInfo, TagMuted lipgloss.Style
	// bars that follow a tag across the width
	BarOK, BarErr, BarWarn, BarInfo lipgloss.Style
}

func build(t *Theme) Styles {
	p, x := t.P, t.X
	var s Styles
	// ink on a fill: under 16 colours a muted grey and a tinted fill can land on one
	// slot and the text vanishes, so take fg or bg instead, whichever stays apart
	ink := func(c, fill color.Color) color.Color { return t.Ink(c, fill) }
	s.Base = lipgloss.NewStyle().Foreground(p.Fg)
	s.Soft = s.Base.Foreground(p.FgSoft)
	s.Muted = s.Base.Foreground(p.FgMuted)
	s.Subtle = s.Base.Foreground(p.FgSubtle)
	s.Bold = s.Base.Bold(true)
	s.Primary = s.Base.Foreground(p.Primary)
	s.Secondary = s.Base.Foreground(p.Secondary)
	s.Tertiary = s.Base.Foreground(p.Tertiary)
	s.Accent = s.Base.Foreground(p.Accent)
	s.OK, s.Err, s.Warn, s.Info = s.Base.Foreground(p.Success), s.Base.Foreground(p.Error), s.Base.Foreground(p.Warning), s.Base.Foreground(p.Info)

	switch x.SectionTitle {
	case "primary":
		s.Title = s.Primary.Bold(true)
	case "muted-upper":
		s.Title = s.Muted.Bold(true)
	case "fg":
		s.Title = s.Bold
	default:
		s.Title = s.Accent.Bold(true)
	}
	s.Rule = lipgloss.NewStyle().Foreground(p.Border)
	s.Panel = lipgloss.NewStyle().Background(p.BgPanel).Foreground(p.FgSoft)
	s.Key = s.Muted
	s.KeyDesc = s.Subtle
	if x.Keys == "boxed" {
		s.Key = s.Base.Foreground(ink(p.Fg, p.BgSubtle)).Background(p.BgSubtle).Padding(0, 1)
	}

	switch x.Selection {
	// filled selections are drawn as reverse video of (fill, ink): the same
	// pixels in colour, and still a visible bar when colour is stripped
	case "fill": // a full-width primary bar, like Crush's dialogs
		s.SelRow = lipgloss.NewStyle().Foreground(p.Primary).Background(ink(p.FgSelected, p.Primary)).Reverse(true).Bold(true)
	case "inverse":
		s.SelRow = lipgloss.NewStyle().Foreground(p.Fg).Background(p.Bg).Reverse(true).Bold(true)
	case "caret":
		s.SelRow = s.Primary.Bold(true)
	default: // bar: bold text beside a focus bar
		s.SelRow = s.Bold
	}
	s.FocusBar = lipgloss.NewStyle().Foreground(p.Primary)
	s.BlurBar = lipgloss.NewStyle().Foreground(p.BgSubtle)
	s.Match = lipgloss.NewStyle().Underline(true)

	switch x.Button {
	case "bracket":
		s.Button = s.Muted
		s.ButtonFocus = s.Primary.Bold(true)
		s.ButtonDanger = s.Err.Bold(true)
	case "underline":
		s.Button = s.Muted
		s.ButtonFocus = s.Primary.Bold(true).Underline(true)
		s.ButtonDanger = s.Err.Bold(true).Underline(true)
	case "inverse":
		s.Button = lipgloss.NewStyle().Foreground(ink(p.Fg, p.BgSubtle)).Background(p.BgSubtle).Padding(0, 2)
		s.ButtonFocus = lipgloss.NewStyle().Foreground(p.Bg).Background(p.Fg).Bold(true).Padding(0, 2)
		s.ButtonDanger = lipgloss.NewStyle().Foreground(ink(p.OnError, p.Error)).Background(p.Error).Bold(true).Padding(0, 2)
	default: // pill
		s.Button = lipgloss.NewStyle().Foreground(ink(p.FgMuted, p.BgSubtle)).Background(p.BgSubtle).Padding(0, 2)
		fill, on := p.Secondary, p.OnSecondary
		if !t.Distinct(fill, p.BgSubtle) {
			fill, on = p.Fg, p.Bg // the focused pill must not look like the rest: fg, as inverse draws it
		}
		s.ButtonFocus = lipgloss.NewStyle().Foreground(ink(on, fill)).Background(fill).Bold(true).Padding(0, 2)
		s.ButtonDanger = lipgloss.NewStyle().Foreground(ink(p.OnError, p.Error)).Background(p.Error).Bold(true).Padding(0, 2)
	}

	var b lipgloss.Border
	switch x.Dialog {
	case "square":
		b = lipgloss.NormalBorder()
	case "double":
		b = lipgloss.DoubleBorder()
	case "heavy":
		b = lipgloss.ThickBorder()
	case "none":
		b = lipgloss.HiddenBorder()
	default:
		b = lipgloss.RoundedBorder()
	}
	if t.ASCII {
		b = lipgloss.ASCIIBorder()
	} else if x.Dialog == "none" && !t.Distinct(p.Bg, p.BgPanel) {
		// the panel step is the dialog's only edge, and at this profile (16 colours,
		// NO_COLOR) it lands on the page's colour: draw a rule above and below instead,
		// in the cells the hidden border kept, so the layout doesn't move
		b = RuleEdge(t.G.Rule)
	}
	s.Dialog = lipgloss.NewStyle().Border(b).BorderForeground(p.BorderFocus).Padding(1, 2)
	if x.Dialog == "none" {
		s.Dialog = s.Dialog.Background(p.BgPanel).BorderBackground(p.BgPanel)
	}

	tag := func(bg, fg color.Color, text lipgloss.Style) lipgloss.Style {
		if x.Tag == "filled" {
			return lipgloss.NewStyle().Background(bg).Foreground(ink(fg, bg)).Bold(true).Padding(0, 1)
		}
		return text.Bold(true)
	}
	s.TagPrimary = tag(p.Primary, p.OnPrimary, s.Primary)
	s.TagOK = tag(p.Success, p.OnSuccess, s.OK)
	s.TagErr = tag(p.Error, p.OnError, s.Err)
	s.TagWarn = tag(p.Warning, p.OnWarning, s.Warn)
	s.TagInfo = tag(p.Info, OnColor(p.Info, p.Bg, p.Fg), s.Info)
	s.TagMuted = tag(p.BgSubtle, p.Fg, s.Muted)
	bar := func(c color.Color) lipgloss.Style {
		fill := Mix(p.Bg, c, 0.18)
		return lipgloss.NewStyle().Background(fill).Foreground(ink(p.Fg, fill)).Padding(0, 1)
	}
	s.BarOK, s.BarErr, s.BarWarn, s.BarInfo = bar(p.Success), bar(p.Error), bar(p.Warning), bar(p.Info)
	return s
}

// RuleEdge is a border of hairline rules above and below and blank cells at the
// sides: an edge for a filled panel that keeps the panel's footprint.
func RuleEdge(rule string) lipgloss.Border {
	return lipgloss.Border{Top: rule, Bottom: rule, TopLeft: rule, TopRight: rule, BottomLeft: rule, BottomRight: rule,
		Left: " ", Right: " "}
}

// Case applies the theme's title case.
func (t *Theme) Case(s string) string {
	switch t.X.TitleCase {
	case "upper":
		return strings.ToUpper(s)
	case "lower":
		return strings.ToLower(s)
	}
	return s
}

// Grad renders text with a per-grapheme foreground gradient from a to b.
// Use it on a handful of cells (a wordmark, a title rule, a spinner), never
// on prose. Under ASCII/NO_COLOR it degrades to the plain text.
func Grad(text string, a, b color.Color, bold bool) string {
	var clusters []string
	g := uniseg.NewGraphemes(text)
	for g.Next() {
		clusters = append(clusters, g.Str())
	}
	if len(clusters) == 0 {
		return ""
	}
	ramp := lipgloss.Blend1D(max(2, len(clusters)), a, b)
	var sb strings.Builder
	for i, c := range clusters {
		st := lipgloss.NewStyle().Foreground(ramp[i])
		if bold {
			st = st.Bold(true)
		}
		sb.WriteString(st.Render(c))
	}
	return sb.String()
}

// Gap is the vertical rhythm: one blank row when airy, none when compact.
func (t *Theme) Gap() int {
	if t.X.Density == "compact" {
		return 0
	}
	return 1
}
