package theme

import (
	"image/color"
	"strings"
	"unicode"
	"unicode/utf8"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"
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
	// emphasis, per expressions.focus: primary, or fg and bold
	Focus   lipgloss.Style // focus, current and selection marks: cursor, current step, active tab, chosen option
	Prompt  lipgloss.Style // input prompts and carets: where you type is focus, so the focus ink (primary, or fg and bold); tertiary where primary is under 3:1 on bgOverlay
	Mark    lipgloss.Style // the kit's own marks: the progress fill (primary, or fg)
	Working lipgloss.Style // the working marks: spinner frames, a busy row's mark, a running tool (per expressions.spinner; Mark by default)
	Motif   lipgloss.Style // the header rule and dialog-title rule runs (primary, or the rule role)
	// selection
	SelRow   lipgloss.Style // a selected row (per expressions.selection)
	SelBars  lipgloss.Style // a trace's or sparkline's bars on a filled selected row: the fill, bars in its shade (ui.Table)
	FocusBar lipgloss.Style // ▌ beside the focused block
	BlurBar  lipgloss.Style // the same column, unfocused
	Match    lipgloss.Style // fuzzy-match characters
	// controls
	Button, ButtonFocus, ButtonDanger lipgloss.Style
	Dialog                            lipgloss.Style // the one boxed element, filled with bgPanel
	Popover                           lipgloss.Style // Dialog on bgOverlay: the palette, completions
	// tags (chips): a word on a filled block, or bold coloured text
	TagPrimary, TagOK, TagErr, TagWarn, TagInfo, TagMuted lipgloss.Style
	// bars that follow a tag across the width
	BarOK, BarErr, BarWarn, BarInfo lipgloss.Style
}

// warnFloor16 is the contrast under which the warning's slot is drawn as a fill at 16
// colours: 3:1, the floor for a state mark on its page.
const warnFloor16 = 3

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
	if t.plainBrand { // primary pinned to a grey: no hue, so the brand's text is set apart by weight
		s.Primary = s.Primary.Bold(true)
	}
	s.Secondary = s.Base.Foreground(p.Secondary)
	s.Tertiary = s.Base.Foreground(p.Tertiary)
	s.Accent = s.Base.Foreground(p.Accent)
	s.OK, s.Err, s.Warn, s.Info = s.Base.Foreground(p.Success), s.Base.Foreground(p.Error), s.Base.Foreground(p.Warning), s.Base.Foreground(p.Info)
	// the warning keeps yellow at 16 colours (#51), and yellow 3 on a light page is 1.35:1:
	// a yellow hairline vanishes on white where a yellow cell does not, so there the
	// warning is a fill, its ink dark (OnWarning, re-inked to read on the slot): its mark,
	// tags and badges are yellow cells that carry their shape (a sparkline keeps its bars
	// quiet and its mark filled, ui.StateSparkline)
	if t.Profile == colorprofile.ANSI && Contrast16(p.Warning, p.Bg) < warnFloor16 {
		s.Warn = lipgloss.NewStyle().Background(p.Warning).Foreground(p.OnWarning)
	}
	s.Rule = lipgloss.NewStyle().Foreground(p.Border)
	// focus "fg": the hue is the brand's state colour, so the kit's emphasis is weight
	fgFocus := x.Focus == "fg"
	s.Focus, s.Prompt, s.Mark, s.Motif = s.Primary, s.Primary, s.Primary, s.Primary
	if t.promptTertiary {
		s.Prompt = s.Tertiary // the focus ink would vanish in a palette's input (ResolveProfile)
	}
	if fgFocus {
		s.Focus, s.Prompt, s.Mark, s.Motif = s.Bold, s.Bold, s.Base, s.Rule
	}
	// the working marks follow the focus ink unless the theme names a role for them
	s.Working = s.Mark
	switch x.Spinner {
	case "primary":
		s.Working = s.Primary
	case "secondary":
		s.Working = s.Secondary
	case "tertiary":
		s.Working = s.Tertiary
	case "accent":
		s.Working = s.Accent
	case "info":
		s.Working = s.Info
	case "fg":
		s.Working = s.Base
	case "muted":
		s.Working = s.Muted
	}

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
		if fgFocus { // an fg bar: inverse
			s.SelRow = lipgloss.NewStyle().Foreground(p.Fg).Background(p.Bg).Reverse(true).Bold(true)
		}
	case "inverse":
		s.SelRow = lipgloss.NewStyle().Foreground(p.Fg).Background(p.Bg).Reverse(true).Bold(true)
	case "caret":
		s.SelRow = s.Focus.Bold(true)
	default: // bar: bold text beside a focus bar
		s.SelRow = s.Bold
	}
	// a trace's bars on a filled row: in the row's ink, which is the page's own colour on
	// most themes, ▆▇█ are blocks of the page punched through the fill, so they take a
	// shade of the fill instead and the fill runs the whole row (ui.Table)
	s.SelBars = s.SelRow
	if x.Selection == "fill" || x.Selection == "inverse" {
		s.SelBars = s.SelRow.Background(t.selShade(s.SelRow.GetForeground(), s.SelRow.GetBackground()))
	}
	s.FocusBar = lipgloss.NewStyle().Foreground(p.Primary)
	if fgFocus {
		s.FocusBar = s.Bold
	}
	// the blurred bar is a hint level, not a background step: bgSubtle is about 1.2:1 on
	// the page, which vanishes at truecolour and leaves a bare | under NO_COLOR
	s.BlurBar = lipgloss.NewStyle().Foreground(p.FgSubtle)
	s.Match = lipgloss.NewStyle().Underline(true)

	switch x.Button {
	case "bracket":
		s.Button = s.Muted
		s.ButtonFocus = s.Focus.Bold(true)
		s.ButtonDanger = s.Err.Bold(true)
	case "underline":
		s.Button = s.Muted
		s.ButtonFocus = s.Focus.Bold(true).Underline(true)
		s.ButtonDanger = s.Err.Bold(true).Underline(true)
	case "inverse":
		s.Button = lipgloss.NewStyle().Foreground(ink(p.Fg, p.BgSubtle)).Background(p.BgSubtle).Padding(0, 2)
		s.ButtonFocus = lipgloss.NewStyle().Foreground(p.Bg).Background(p.Fg).Bold(true).Padding(0, 2)
		s.ButtonDanger = lipgloss.NewStyle().Foreground(ink(p.OnError, p.Error)).Background(p.Error).Bold(true).Padding(0, 2)
	default: // pill
		s.Button = lipgloss.NewStyle().Foreground(ink(p.FgMuted, p.BgSubtle)).Background(p.BgSubtle).Padding(0, 2)
		fill, on := p.Secondary, p.OnSecondary
		if fgFocus || !t.Distinct(fill, p.BgSubtle) {
			fill, on = p.Fg, p.Bg // the focused pill must not look like the rest: fg, as inverse draws it
		}
		s.ButtonFocus = lipgloss.NewStyle().Foreground(ink(on, fill)).Background(fill).Bold(true).Padding(0, 2)
		s.ButtonDanger = lipgloss.NewStyle().Foreground(ink(p.OnError, p.Error)).Background(p.Error).Bold(true).Padding(0, 2)
	}

	// a dialog is a filled panel whatever its edge, and whether or not the app paints the
	// terminal's background (paintBg): unfilled, the page's colour would show through it
	panel := func(fill color.Color) lipgloss.Style {
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
		} else if x.Dialog == "none" && !t.Distinct(p.Bg, fill) {
			// the panel step is the dialog's only edge, and at this profile (16 colours,
			// NO_COLOR) it lands on the page's colour: draw a rule above and below instead,
			// in the cells the hidden border kept, so the layout doesn't move
			b = RuleEdge(t.G.Rule)
		}
		if x.Dialog == "rules" {
			// the editorial dialog: a filled panel, one cell of padding inside and one cell
			// of page around it, so the page's text never runs into its open sides. ui.Dialog
			// draws its rules above and below (the title in the top one): its edge is glyphs,
			// which need no colour and hold at every profile
			return lipgloss.NewStyle().Background(fill).Padding(1, 1).Margin(1, 1).MarginBackground(p.Bg)
		}
		return lipgloss.NewStyle().Border(b).BorderForeground(p.BorderFocus).Padding(1, 2).Background(fill).BorderBackground(fill)
	}
	s.Dialog, s.Popover = panel(p.BgPanel), panel(p.BgOverlay)

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

// RuleEdge is a border of hairline rules above and below, blank cells at the sides
// and blank corners: an edge for a filled panel that keeps the panel's footprint and
// leaves a cell of space between its rules and the page's text beside them.
func RuleEdge(rule string) lipgloss.Border {
	return lipgloss.Border{Top: rule, Bottom: rule, TopLeft: " ", TopRight: " ", BottomLeft: " ", BottomRight: " ",
		Left: " ", Right: " "}
}

// Case applies the theme's title case. Under lower a word in capitals keeps them
// (SIGTERM, CPU, PRs, HTTP/2: two capitals or more and no small letter but a plural or
// possessive s), so "Send SIGTERM?" is "send SIGTERM?", not "send sigterm?"; a span the
// caller marks with Verbatim keeps its case under lower and upper ("Open in " +
// theme.Verbatim("iTerm2")).
func (t *Theme) Case(s string) string {
	switch t.X.TitleCase {
	case "upper":
		return caseSpans(s, func(w string) string { return strings.ToUpper(w) })
	case "lower":
		return caseSpans(s, func(w string) string {
			if acronym(w) {
				return w
			}
			return strings.ToLower(w)
		})
	}
	return caseSpans(s, func(w string) string { return w })
}

// verbatimOn and verbatimOff mark a span the case expressions leave alone: Unicode
// noncharacters, which no text carries, so the marks never clash with the caller's.
const verbatimOn, verbatimOff = "\uFDD0", "\uFDD1"

// Verbatim marks s to keep its case under titleCase and labelCase (a product name,
// iTerm2, kubectl): Case and Label draw it as written and drop the marks. Pass it only
// in text a kit component cases (a Section or dialog title, a tab, a tag, a header):
// anywhere else the marks reach the terminal.
func Verbatim(s string) string { return verbatimOn + s + verbatimOff }

// caseSpans applies fn to each word of s outside the Verbatim spans, drops the marks,
// and keeps the spaces as they are.
func caseSpans(s string, fn func(string) string) string {
	var b strings.Builder
	for s != "" {
		before, rest, marked := strings.Cut(s, verbatimOn)
		b.WriteString(caseWords(before, fn))
		if !marked {
			break
		}
		span, after, _ := strings.Cut(rest, verbatimOff)
		b.WriteString(span)
		s = after
	}
	return b.String()
}

// caseWords applies fn to each run of non-space runes in s.
func caseWords(s string, fn func(string) string) string {
	var b strings.Builder
	for s != "" {
		i := strings.IndexFunc(s, unicode.IsSpace)
		if i < 0 {
			b.WriteString(fn(s))
			break
		}
		b.WriteString(fn(s[:i]))
		_, n := utf8.DecodeRuneInString(s[i:])
		b.WriteString(s[i : i+n])
		s = s[i+n:]
	}
	return b.String()
}

// acronym reports whether w is a word in capitals: two capital letters or more and no
// small letter but a last letter s (PRs, CPUs, SIGTERM's).
func acronym(w string) bool {
	upper, letters, lowerAt := 0, 0, -1
	var lower rune
	for _, r := range w {
		switch {
		case unicode.IsUpper(r):
			upper++
		case unicode.IsLower(r):
			if lowerAt >= 0 {
				return false
			}
			lowerAt, lower = letters, r
		default:
			continue
		}
		letters++
	}
	return upper >= 2 && (lowerAt < 0 || lower == 's' && lowerAt == letters-1)
}

// Copy applies the theme's case to a label of the kit's own fixed copy: tab names and
// step names, one word or several ("Pull requests", "Sign in with"). labelCase when the
// theme sets one (the brand's case for the small words of its UI), else titleCase, so
// every label in a row takes the same case. A sentence takes Sentence. Text the app
// writes (Huh field descriptions, prose) is the app's own: write it in the brand's case.
func (t *Theme) Copy(s string) string {
	if t.X.LabelCase != "" {
		return t.Label(s, "as-is")
	}
	return t.Case(s)
}

// Sentence applies titleCase to a sentence of the kit's own fixed copy: "Terminal too
// small", a tool call's "Canceled." and "Waiting for permission", Thinking's "Thought
// for". labelCase is for labels, so a sentence-case brand with lower labels never starts
// a line with "canceled.". Pass a sentence whole, full stop included.
func (t *Theme) Sentence(s string) string { return t.Case(s) }

// Label applies the theme's label case (expressions.labelCase) to a table header, tag,
// toast or summary label; own is the component's case when the theme leaves it unset
// ("upper" for table headers, "lower" for summary-strip labels, "as-is" for tags). A
// label is a word of the brand's UI, so labelCase lower lowers capitals too ("CPU" →
// "cpu"); a Verbatim span keeps its case.
func (t *Theme) Label(s, own string) string {
	c := t.X.LabelCase
	if c == "" {
		c = own
	}
	switch c {
	case "upper":
		return caseSpans(s, strings.ToUpper)
	case "lower":
		return caseSpans(s, strings.ToLower)
	}
	return caseSpans(s, func(w string) string { return w })
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

// selShade is the ink of a trace's bars on a filled selection (Styles.SelBars): the fill
// mixed toward the row's ink until the bars read on the fill (1.3:1) and not as the page
// (1.5:1). At 16 colours, where a mix lands on the fill's slot or the ink's, it is the
// grey that reads best on the fill and is not the page. Under NO_COLOR it is the ink:
// reverse video is all there is.
func (t *Theme) selShade(fill, ink color.Color) color.Color {
	page := t.P.Bg
	switch t.Profile {
	case colorprofile.ANSI:
		best, cr := ink, 0.0
		for _, n := range []int{8, 7, 0, 15} {
			g := ansi.BasicColor(n)
			if Slot16(g) == Slot16(page) || Slot16(g) == Slot16(fill) || Contrast16(g, page) < 1.5 {
				continue
			}
			if c := Contrast16(g, fill); c > cr {
				best, cr = g, c
			}
		}
		return best
	case colorprofile.Ascii, colorprofile.NoTTY:
		return ink
	}
	for _, m := range []float64{0.5, 0.6, 0.4, 0.7, 0.3, 0.8} {
		c := Mix(fill, ink, m)
		if Contrast(c, page) >= 1.5 && Contrast(c, fill) >= 1.3 && t.Distinct(c, page) && t.Distinct(c, fill) {
			return c
		}
	}
	return ink
}
