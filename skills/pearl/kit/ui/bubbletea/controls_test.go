package ui

import (
	"fmt"
	"image/color"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"

	"tuikit/theme/bubbletea"
)

var sgrFG = regexp.MustCompile(`\x1b\[[0-9;]*38;2;(\d+);(\d+);(\d+)`)

// In ASCII mode a button's ink must read on the background: in brackets a pill's ink was
// its fill's on-colour, often the background itself, which made the focused button
// invisible (found by linting -ascii renders). A filled button is reverse video, its
// first ink the fill, which stands off the page (TestButtonsAreFillsWhereColourShows).
func TestASCIIButtonsReadOnTheBackground(t *testing.T) {
	for _, f := range testThemes(t) {
		tf, err := theme.Load(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, dark := range []bool{true, false} {
			th := theme.Resolve(tf, dark, true)
			for _, c := range []struct{ focused, danger bool }{{true, false}, {false, false}, {true, true}, {false, true}} {
				out := Button(th, "Turn on", c.focused, c.danger, -1)
				m := sgrFG.FindStringSubmatch(out)
				if m == nil {
					continue // no explicit ink: the terminal's foreground
				}
				r, _ := strconv.Atoi(m[1])
				g, _ := strconv.Atoi(m[2])
				b, _ := strconv.Atoi(m[3])
				ink := theme.Hex("#" + hex2(r) + hex2(g) + hex2(b))
				if cr := theme.Contrast(ink, th.P.Bg); cr < 2.3 {
					t.Errorf("%s dark=%v focused=%v danger=%v: ASCII button ink %.2f:1 on bg", filepath.Base(f), dark, c.focused, c.danger, cr)
				}
			}
		}
	}
}

func hex2(v int) string {
	s := strconv.FormatInt(int64(v), 16)
	if len(s) < 2 {
		s = "0" + s
	}
	return s
}

// A warning's sparkline ends in its mark, against its last bar, and the mark takes a
// column every row keeps, so the time axis stays aligned: the newest value of every
// row sits in one column, and a warning row draws the same points as a quiet one (#51
// review: it drew two fewer, two columns left).
func TestStateSparklineKeepsTheTimeAxis(t *testing.T) {
	for _, ascii := range []bool{false, true} {
		th := testTheme(t, ascii)
		vals := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14}
		quiet, warn := ansi.Strip(StateSparkline(th, "ok", vals, 12)), ansi.Strip(StateSparkline(th, "warning", vals, 12))
		if lipgloss.Width(quiet) != 12 || lipgloss.Width(warn) != 12 {
			t.Errorf("ascii=%v: widths %d and %d, want 12 for every state", ascii, lipgloss.Width(quiet), lipgloss.Width(warn))
		}
		qr, wr := []rune(quiet), []rune(warn)
		if string(qr[:10]) != string(wr[:10]) {
			t.Errorf("ascii=%v: the warning row draws %q where a quiet one draws %q: one time axis", ascii, string(wr[:10]), string(qr[:10]))
		}
		if string(wr[10:]) != th.G.Warn+" " || strings.TrimSpace(string(qr[10:])) != "" {
			t.Errorf("ascii=%v: mark column %q (warning) and %q (quiet), want %q, the mark against its last bar, and blank", ascii, string(wr[10:]), string(qr[10:]), th.G.Warn+" ")
		}
	}
}

// Under ASCII a progress bar's empty part is "-", and a 1-cell remainder is too short to
// read as a rule: tuilint frames judged it text in the border colour, 1.52:1 (#61). The
// ASCII track is drawn in an ink that clears the 2.3:1 text floor.
func TestASCIIProgressTrackReadsAsText(t *testing.T) {
	for _, f := range testThemes(t) {
		tf, err := theme.Load(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, dark := range []bool{true, false} {
			th := theme.Resolve(tf, dark, true)
			bar := Progress(th, 0.95, 20) // 19 filled, 1 empty
			c := cellAt(bar, th.G.ProgressEmpty)
			if c == nil {
				t.Fatalf("%s dark=%v: no empty cell in %q", filepath.Base(f), dark, bar)
			}
			if cr := theme.Contrast(c.Style.Fg, th.P.Bg); cr < 2.3 {
				t.Errorf("%s dark=%v: the ASCII remainder %q is %.2f:1 on the page, under the 2.3:1 text floor", filepath.Base(f), dark, c.Content, cr)
			}
		}
	}
}

// shown is what a terminal at p shows for the cell under the first rune of word in s:
// its ink and its fill, reverse video applied, and whether it is reversed.
func shown(s, word string, p colorprofile.Profile) (ink, fill color.Color, reversed bool, ok bool) {
	c := cellAt(downsample(s, p), word)
	if c == nil {
		return nil, nil, false, false
	}
	ink, fill = c.Style.Fg, c.Style.Bg
	reversed = c.Style.Attrs&uv.AttrReverse != 0
	if reversed {
		ink, fill = fill, ink
	}
	return ink, fill, reversed, true
}

// Buttons are marked by a fill where colour shows, not by brackets: a page of [x] marks,
// [PROD] badges and bracketed buttons made the buttons hard to tell apart ("there are a
// lot of brackets on the page"). Wherever colour shows (truecolour, 256 and 16 colours,
// ASCII glyphs included), the focused button of every expression but underline is a
// fill that stands off the page and the dialog, with its ink apart from it, and draws no
// bracket or cursor; under bracket that fill is primary (fg under focus "fg", where the
// hue is a state), and an unfocused button is a plain label. With no colour (NO_COLOR)
// the focused button keeps a mark that needs none: brackets, or reverse video.
func TestButtonsAreFillsWhereColourShows(t *testing.T) {
	type mode struct {
		name        string
		dark, ascii bool
		p           colorprofile.Profile
	}
	modes := []mode{
		{"dark", true, false, colorprofile.TrueColor}, {"light", false, false, colorprofile.TrueColor},
		{"ascii", true, true, colorprofile.TrueColor}, {"ansi256", true, false, colorprofile.ANSI256},
		{"ansi", true, false, colorprofile.ANSI}, {"light-ansi", false, false, colorprofile.ANSI},
		{"nocolor", true, false, colorprofile.Ascii}, {"nocolor-ascii", true, true, colorprofile.Ascii},
	}
	for _, f := range append(testThemes(t), quirePath(t)) {
		tf, err := theme.Load(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, x := range []string{"bracket", "pill", "inverse", "underline"} {
			tf.Expressions.Button = x
			for _, m := range modes {
				th := theme.ResolveProfile(tf, m.dark, m.ascii, m.p)
				name := fmt.Sprintf("%s button=%s %s", filepath.Base(f), x, m.name)
				for _, danger := range []bool{false, true} {
					on, off := Button(th, "Continue", true, danger, -1), Button(th, "Continue", false, danger, -1)
					if m.p == colorprofile.Ascii {
						if x == "underline" {
							continue // bold and underlined: a mark with no colour (TestASCIIUnderlineButton)
						}
						_, _, rev, _ := shown(on, "Continue", m.p)
						if !rev && !strings.Contains(ansi.Strip(on), "[") {
							t.Errorf("%s danger=%v: with no colour the focused button %q has no mark: no bracket, no reverse", name, danger, ansi.Strip(on))
						}
						if downsample(on, m.p) == downsample(off, m.p) {
							t.Errorf("%s danger=%v: with no colour focused and unfocused draw alike: %q", name, danger, ansi.Strip(on))
						}
						continue
					}
					if strings.ContainsAny(ansi.Strip(on+off), "[]") || strings.Contains(ansi.Strip(on), th.G.Cursor) {
						t.Errorf("%s danger=%v: in colour the buttons draw %q and %q: brackets, not a fill", name, danger, ansi.Strip(on), ansi.Strip(off))
					}
					if x == "underline" {
						continue
					}
					ink, fill, _, ok := shown(on, "Continue", m.p)
					if !ok || fill == nil {
						t.Errorf("%s danger=%v: the focused button %q has no fill", name, danger, ansi.Strip(on))
						continue
					}
					for _, under := range []color.Color{th.P.Bg, th.P.BgPanel} {
						if !th.Distinct(fill, under) {
							t.Errorf("%s danger=%v: the focused button's fill %v draws as the surface %v it sits on", name, danger, fill, under)
						}
					}
					if ink == nil || !th.Distinct(ink, fill) {
						t.Errorf("%s danger=%v: the focused button's ink %v draws as its fill %v", name, danger, ink, fill)
					}
					want := th.P.Primary
					if th.X.Focus == "fg" || !th.Distinct(want, th.P.Bg) || !th.Distinct(want, th.P.BgPanel) || !th.Distinct(want, th.P.BgOverlay) {
						want = th.P.Fg
					}
					if x == "bracket" && !danger && th.Distinct(fill, want) {
						t.Errorf("%s: the focused bracket button's fill is %v, want %v (primary, or fg where the hue is a state)", name, fill, want)
					}
					if _, offFill, _, _ := shown(off, "Continue", m.p); offFill != nil && !th.Distinct(offFill, fill) {
						t.Errorf("%s danger=%v: the unfocused button's fill is the focused one's", name, danger)
					}
				}
			}
		}
	}
}
