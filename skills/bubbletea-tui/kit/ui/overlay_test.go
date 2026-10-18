package ui

import (
	"fmt"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
	uv "github.com/charmbracelet/ultraviolet"

	"tuikit/theme"
)

// overlayPage is a page a dialog is drawn over: table rows whose columns are two blanks
// apart, prose, full-width rules, a rule that starts after a gap, and one row carrying
// the theme's focus bar glyph. It returns the rows as plain text, cell for cell.
func overlayPage(th *theme.Theme, c *Canvas) []string {
	b := c.Bounds()
	w := b.Dx()
	var plain []string
	for y := range b.Dy() {
		var row string
		switch {
		case y == 5:
			row = "  " + Gutter(th, true, th.S.FocusBar) + th.S.Bold.Render("checkout-api  v2.4.0  ok  p99 124ms  error 0.02%  deployed 3h")
		case y%8 == 0:
			row = "  " + th.S.Soft.Render("search-indexer  v1.9.3  busy  p99 88ms  error 0.00%  deployed 1d ago")
		case y%8 == 4:
			row = "  " + th.S.Warn.Render("▲") + " " + th.S.Base.Render("notifications-worker-eu-west  14:03 circuit opened on payments/api for checkout-web")
		case y%8 == 1:
			row = th.S.Soft.Render("lorem ipsum dolor sit amet consectetur adipiscing elit sed do eiusmod tempor")
		case y%8 == 5:
			row = "  " + th.S.Muted.Render("p99 "+strings.Repeat("▁▂▃▄▅▆▇█", 8))
		case y%4 == 2:
			row = th.S.Rule.Render(strings.Repeat(th.G.Rule, w))
		default:
			row = "  " + th.S.Subtle.Render("SERVICE") + strings.Repeat(" ", w-20) + th.S.Rule.Render(strings.Repeat(th.G.Rule, 11)) // starts two cells past the dialog
		}
		c.Draw(row, uv.Rect(0, y, w, 1))
		var sb strings.Builder
		for x := range w {
			if cell := c.CellAt(x, y); cell != nil && cell.Content != "" {
				sb.WriteString(cell.Content)
			} else {
				sb.WriteString(" ")
			}
		}
		plain = append(plain, sb.String())
	}
	return plain
}

func cellText(c *Canvas, x, y int) string {
	if cell := c.CellAt(x, y); cell != nil && cell.Content != "" {
		return cell.Content
	}
	return " "
}

// A dialog's rows are its own, at every size and in every profile: the page outside it
// on the rows it spans is cleared across, to the page's background. Under a scrim the
// kit used to clear only the word the dialog cut and leave the rest, and at 120 columns
// a column of "…to show it" and "…ctx.Err()" stood beside a confirm on its own rows,
// re-cut at its margin, reading as part of the dialog. Where the page can be dimmed
// (a scrim at truecolour or 256 colours), the rows the dialog doesn't span are the
// page's, cell for cell (dimmed), a focus bar's glyph among them: Overlay doesn't look
// for focus marks (the app draws the page unfocused), so a wordmark or a separator
// drawn in the same glyph survives. Where it can't (16 colours, NO_COLOR,
// scrim: none) the page's body is hidden whole (TestOverlayHidesTheBodyWhereItCantDim).
func TestOverlayClearsItsBand(t *testing.T) {
	const W, H = 70, 20
	for _, dialog := range []string{"rounded", "rules", "none"} {
		for _, mode := range []struct {
			name  string
			ascii bool
			p     colorprofile.Profile
			scrim string
		}{
			{"truecolor", false, colorprofile.TrueColor, "dim"},
			{"ansi256", false, colorprofile.ANSI256, "dim"},
			{"ansi", false, colorprofile.ANSI, "dim"},
			{"NO_COLOR", false, colorprofile.Ascii, "dim"},
			{"ascii", true, colorprofile.TrueColor, "dim"},
			{"ascii ansi", true, colorprofile.ANSI, "dim"},
			{"ascii NO_COLOR", true, colorprofile.Ascii, "dim"},
			{"scrim none", false, colorprofile.TrueColor, "none"},
			{"rules in ·", false, colorprofile.TrueColor, "dim"},
		} {
			f := theme.Default()
			f.Expressions.Dialog, f.Expressions.Scrim = dialog, mode.scrim
			if mode.name == "rules in ·" {
				f.Glyphs.Rule = "·"
			}
			th := theme.ResolveProfile(f, true, mode.ascii, mode.p)
			scrimmed := mode.scrim == "dim" && mode.p != colorprofile.ANSI && mode.p != colorprofile.Ascii
			d := Dialog(th, 44, 0, "Roll back checkout-api?", []string{"It goes back to v2.3.0.", "", "3 pods restart."}, "", "")
			for y0 := 1; y0+len(strings.Split(d, "\n")) < H; y0++ {
				c := NewCanvas(th, W, H)
				page := overlayPage(th, c)
				r := c.Overlay(d, &uv.Position{X: -1, Y: y0})
				name := fmt.Sprintf("%s %s y0=%d", dialog, mode.name, y0)
				for y := range H {
					band := y >= r.Min.Y && y < r.Max.Y
				cells:
					for x := range W {
						if uv.Pos(x, y).In(r) {
							continue
						}
						got := cellText(c, x, y)
						switch {
						case band || !scrimmed:
							if got != " " {
								t.Errorf("%s: the page shows %q at %d,%d on a row the dialog spans or with no scrim: %q", name, got, x, y, rowText(c, y))
								break cells
							}
							if cell := c.CellAt(x, y); cell != nil && cell.Style.Bg != nil && !sameColor(cell.Style.Bg, th.P.Bg) {
								t.Errorf("%s: a cleared cell at %d,%d keeps a background %v, not the page's", name, x, y, cell.Style.Bg)
								break cells
							}
						case got != string([]rune(page[y])[x]):
							t.Errorf("%s: row %d, which the dialog doesn't span, reads %q under a scrim, want the page's %q", name, y, rowText(c, y), page[y])
							break cells
						}
					}
				}
			}
		}
	}
}

// Where the page can't be dimmed (16 colours, NO_COLOR, scrim: none, whatever the glyph
// set) it would stand at full ink above and below the dialog, as live as the dialog
// itself: a confirm over a composer showed the reply's last two lines under its bottom
// edge, a second copy of the body on the screen that shows exactly what will be sent.
// There the rows of the page's body (Canvas.Body) are hidden across, and the rows
// outside it, the app's header, are kept; with no Body set, everything outside the
// dialog is cleared (the app draws its footer over the dialog afterwards, as it
// already does). Under a scrim the body stays, dimmed, ASCII glyphs at truecolour or
// 256 colours included: the glyph set is not the terminal's colour.
func TestOverlayHidesTheBodyWhereItCantDim(t *testing.T) {
	const W, H = 70, 20
	for _, mode := range []struct {
		name  string
		ascii bool
		p     colorprofile.Profile
		scrim string
		dims  bool
	}{
		{"truecolor", false, colorprofile.TrueColor, "dim", true},
		{"ansi256", false, colorprofile.ANSI256, "dim", true},
		{"ansi", false, colorprofile.ANSI, "dim", false},
		{"NO_COLOR", false, colorprofile.Ascii, "dim", false},
		{"ascii", true, colorprofile.TrueColor, "dim", true},
		{"ascii ansi256", true, colorprofile.ANSI256, "dim", true},
		{"ascii ansi", true, colorprofile.ANSI, "dim", false},
		{"ascii NO_COLOR", true, colorprofile.Ascii, "dim", false},
		{"scrim none", false, colorprofile.TrueColor, "none", false},
	} {
		f := theme.Default()
		f.Expressions.Scrim = mode.scrim
		th := theme.ResolveProfile(f, true, mode.ascii, mode.p)
		d := Dialog(th, 44, 0, "Discard this reply?", []string{"when the caller gave up first", "errors.Is."}, "", "")
		// Body is rows: an inset rectangle hides the same rows across as a full-width one
		for _, body := range []uv.Rectangle{{}, uv.Rect(0, 3, W, H-5), uv.Rect(2, 3, W-4, H-5)} {
			c := NewCanvas(th, W, H)
			page := overlayPage(th, c)
			c.Body = body
			r := c.Overlay(d, &uv.Position{X: -1, Y: 6})
			hidden := func(y int) bool { return body.Empty() || y >= body.Min.Y && y < body.Max.Y }
			name := fmt.Sprintf("%s body=%v", mode.name, body)
			for y := range H {
				if y >= r.Min.Y && y < r.Max.Y {
					continue // the band: TestOverlayClearsItsBand
				}
			cols:
				for x := range W {
					got, was := cellText(c, x, y), string([]rune(page[y])[x])
					switch {
					case mode.dims || !hidden(y):
						if got != was {
							t.Errorf("%s: %q at %d,%d is %q; the page outside its body (or under a scrim) is kept: %q", name, was, x, y, got, rowText(c, y))
							break cols
						}
					case got != " ":
						t.Errorf("%s: no scrim, and the page's body shows %q at %d,%d at full ink beside the dialog: %q", name, got, x, y, rowText(c, y))
						break cols
					}
				}
			}
		}
	}
}

// Body hides whole rows, so an app with a pane header beside a sidebar (the header on
// rows 1–2 right of a sidebar that runs the full height) couldn't hide the sidebar and
// keep the header: hiding from below the header kept the sidebar's first rows and half
// an item, and hiding every row took the header too. Canvas.Keep names what stays: a
// cell of the body in a Keep rect holds the page's glyph where the page can't be
// dimmed, and the rest of the body's rows are hidden as before. The dialog's own rows
// are its own whatever Keep says, and under a scrim Keep changes nothing.
func TestOverlayKeepsWhatTheAppNames(t *testing.T) {
	const W, H = 70, 20
	for _, mode := range []struct {
		name  string
		p     colorprofile.Profile
		scrim string
		dims  bool
	}{
		{"truecolor", colorprofile.TrueColor, "dim", true},
		{"ansi", colorprofile.ANSI, "dim", false},
		{"NO_COLOR", colorprofile.Ascii, "dim", false},
		{"scrim none", colorprofile.TrueColor, "none", false},
	} {
		f := theme.Default()
		f.Expressions.Scrim = mode.scrim
		th := theme.ResolveProfile(f, true, false, mode.p)
		d := Dialog(th, 30, 0, "Discard this reply?", []string{"when the caller gave up first"}, "", "")
		c := NewCanvas(th, W, H)
		page := overlayPage(th, c)
		c.Body = uv.Rect(0, 1, W, H-2)                       // everything between a title row and the footer
		header := uv.Rect(24, 1, W-24, 2)                    // a pane's header right of a 24-cell sidebar
		c.Keep = []uv.Rectangle{header, uv.Rect(0, 0, 0, 0)} // an empty rect keeps nothing
		r := c.Overlay(d, &uv.Position{X: -1, Y: 8})
		for y := range H {
			if y >= r.Min.Y && y < r.Max.Y {
				continue
			}
		cols:
			for x := range W {
				got, was := cellText(c, x, y), string([]rune(page[y])[x])
				kept := uv.Pos(x, y).In(header) || y < 1 || y >= H-1
				switch {
				case mode.dims || kept:
					if got != was {
						t.Errorf("%s: %q at %d,%d is %q; Keep names it (or it is outside the body, or the page dims): %q", mode.name, was, x, y, got, rowText(c, y))
						break cols
					}
				case got != " ":
					t.Errorf("%s: the body outside Keep shows %q at %d,%d at full ink beside the dialog: %q", mode.name, got, x, y, rowText(c, y))
					break cols
				}
			}
		}
	}
	// the band stays the dialog's: a Keep rect across its rows keeps nothing there
	f := theme.Default()
	th := theme.ResolveProfile(f, true, false, colorprofile.Ascii)
	c := NewCanvas(th, W, H)
	overlayPage(th, c)
	c.Body = uv.Rect(0, 1, W, H-2)
	c.Keep = []uv.Rectangle{uv.Rect(0, 1, W, H-2)}
	r := c.Overlay(Dialog(th, 30, 0, "Discard?", []string{"gone for good"}, "", ""), &uv.Position{X: -1, Y: 8})
	for x := range W {
		if !uv.Pos(x, r.Min.Y).In(r) && cellText(c, x, r.Min.Y) != " " {
			t.Errorf("Keep across the dialog's rows kept %q at %d,%d: %q", cellText(c, x, r.Min.Y), x, r.Min.Y, rowText(c, r.Min.Y))
			break
		}
	}
}

// A popover anchored to its trigger (a completion list above the line being typed,
// anchor.X ≥ 0) is not a modal: it clears the rows it spans and nothing else, in every
// profile, so the line it completes stays on screen where the page can't be dimmed.
// Hiding the body under it blanked the very input the list completes.
func TestAnchoredPopoverKeepsItsTrigger(t *testing.T) {
	const W, H = 70, 20
	for _, mode := range []struct {
		name  string
		ascii bool
		p     colorprofile.Profile
		scrim string
	}{
		{"truecolor", false, colorprofile.TrueColor, "dim"},
		{"ansi", false, colorprofile.ANSI, "dim"},
		{"NO_COLOR", false, colorprofile.Ascii, "dim"},
		{"ascii ansi", true, colorprofile.ANSI, "dim"},
		{"scrim none", false, colorprofile.TrueColor, "none"},
	} {
		f := theme.Default()
		f.Expressions.Scrim = mode.scrim
		th := theme.ResolveProfile(f, true, mode.ascii, mode.p)
		d := Popover(th, 30, 0, "", []string{"retry budget", "retry deadline"}, "", "")
		c := NewCanvas(th, W, H)
		page := overlayPage(th, c)
		c.Body = uv.Rect(0, 1, W, H-2)
		trigger := 12 // the line typed in, under the list
		r := c.Popover(d, &uv.Position{X: 4, Y: trigger - lipgloss.Height(d)})
		for y := range H {
			if y >= r.Min.Y && y < r.Max.Y {
				continue
			}
			if got := rowText(c, y); got != page[y] {
				t.Errorf("%s: row %d (the trigger is %d) reads %q under an anchored popover, want the page's %q", mode.name, y, trigger, got, page[y])
			}
		}
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := range W {
				if !uv.Pos(x, y).In(r) && cellText(c, x, y) != " " {
					t.Errorf("%s: the page shows %q at %d,%d on a row the popover spans", mode.name, cellText(c, x, y), x, y)
					break
				}
			}
		}
	}
}

func rowText(c *Canvas, y int) string {
	var b strings.Builder
	for x := range c.Bounds().Dx() {
		b.WriteString(cellText(c, x, y))
	}
	return b.String()
}

// A chip on the dialog's rows goes whole, its padding too: "FAILING" was cleared and
// its filled padding stayed beside the dialog, an empty block of the chip's colour.
// Under a scrim, at every column the chip can sit across either edge, a side of the
// chip's row with no text left keeps no background but the page's.
func TestOverlayLeavesNoEmptyChip(t *testing.T) {
	const W, H = 70, 9
	th := theme.ResolveProfile(theme.Default(), true, false, colorprofile.TrueColor)
	d := Dialog(th, 44, 0, "Roll back checkout-api?", []string{"It goes back to v2.3.0."}, "", "")
	chip := Tag(th, "error", "FAILING")
	for x0 := 0; x0 < W-lipgloss.Width(chip); x0++ {
		c := NewCanvas(th, W, H)
		c.Draw(strings.Repeat(" ", x0)+chip, uv.Rect(0, 4, W, 1))
		r := c.Overlay(d, &uv.Position{X: -1, Y: 2})
		for _, side := range [][2]int{{0, r.Min.X}, {r.Max.X, W}} {
			text, filled := false, -1
			for x := side[0]; x < side[1]; x++ {
				cell := c.CellAt(x, 4)
				if cell == nil {
					continue
				}
				if strings.TrimSpace(cell.Content) != "" {
					text = true
				} else if cell.Style.Bg != nil && !sameColor(cell.Style.Bg, th.P.Bg) && filled < 0 {
					filled = x
				}
			}
			if !text && filled >= 0 {
				t.Errorf("chip at %d: an empty chip at %d beside the dialog", x0, filled)
			}
		}
	}
}
