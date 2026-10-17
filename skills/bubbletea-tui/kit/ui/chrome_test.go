package ui

import (
	"path/filepath"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"

	"tuikit/theme"
)

// cellAt is the cell under the first rune of word in the styled row s.
func cellAt(s, word string) *uv.Cell {
	plain := ansi.Strip(s)
	x := strings.Index(plain, word)
	if x < 0 {
		return nil
	}
	buf := uv.NewScreenBuffer(lipgloss.Width(s), 1)
	uv.NewStyledString(s).Draw(buf, buf.Bounds())
	return buf.CellAt(len([]rune(plain[:x])), 0)
}

// Under ASCII the active tab's underline is the theme's rule, not a run of carets, and
// an underline button stays an underline button: bold and underline survive ASCII, and
// the bracket with a cursor it turned into was a second and third focus cue beside the
// form's focus bar.
func TestASCIIChromeKeepsOneCue(t *testing.T) {
	files := append(testThemes(t), filepath.Join("..", "theme", "testdata", "quire.json"))
	for _, f := range files {
		tf, err := theme.Load(f)
		if err != nil {
			t.Fatal(err)
		}
		tf.Expressions.Tabs, tf.Expressions.Button = "underline", "underline"
		th := theme.Resolve(tf, true, true)
		tabs := strings.Join(Tabs(th, []string{"Inbox", "Saved"}, 0, nil), "\n")
		if strings.Contains(ansi.Strip(tabs), "^") {
			t.Errorf("%s: ASCII tabs underline with carets:\n%s", filepath.Base(f), ansi.Strip(tabs))
		}
		if !strings.Contains(ansi.Strip(tabs), strings.Repeat(th.G.Rule, 5)) {
			t.Errorf("%s: no rule under the active ASCII tab:\n%s", filepath.Base(f), ansi.Strip(tabs))
		}
		b := Button(th, "Continue", true, false, -1)
		plain := ansi.Strip(b)
		if strings.ContainsAny(plain, "[]") || strings.Contains(plain, th.G.Cursor) {
			t.Errorf("%s: an ASCII underline button draws %q: a bracket or a cursor is a second cue", filepath.Base(f), plain)
		}
		c := cellAt(b, "Continue")
		if c == nil || c.Style.Attrs&uv.AttrBold == 0 || c.Style.Underline == uv.UnderlineStyleNone {
			t.Errorf("%s: the focused ASCII underline button is not bold and underlined: %q", filepath.Base(f), b)
		}
	}
}

// A dialog is a filled panel whatever its edge, and whether or not the app paints the
// terminal background (paintBg: false): every cell inside it samples as bgPanel after
// an Overlay, and as bgOverlay after a Popover, in a bordered style, a none and a rules one.
func TestOverlaysPaintThePanel(t *testing.T) {
	for _, dialog := range []string{"rounded", "square", "none", "rules"} {
		f := theme.Default()
		f.Expressions.Dialog, f.Expressions.PaintBg = dialog, false
		th := theme.Resolve(f, true, false)
		for _, c := range []struct {
			name string
			draw func(c *Canvas) uv.Rectangle
			fill interface {
				RGBA() (uint32, uint32, uint32, uint32)
			}
		}{
			{"overlay", func(c *Canvas) uv.Rectangle {
				return c.Overlay(Dialog(th, 40, "Roll back?", []string{"It goes back."}, "", ""), nil)
			}, th.P.BgPanel},
			{"popover", func(c *Canvas) uv.Rectangle {
				return c.Popover(Popover(th, 40, "Commands", []string{"> ro"}, "", ""), nil)
			}, th.P.BgOverlay},
		} {
			cv := NewCanvas(th, 70, 20)
			cv.Draw(strings.Repeat(th.S.Soft.Render(strings.Repeat("lorem ipsum ", 6))+"\n", 20), cv.Bounds())
			r := c.draw(cv)
			cv.PaintBackground()
			r = Inset(r, th.S.Dialog.GetMarginLeft(), th.S.Dialog.GetMarginTop())
			for y := r.Min.Y; y < r.Max.Y; y++ {
				for x := r.Min.X; x < r.Max.X; x++ {
					if bg := cv.CellAt(x, y).Style.Bg; !same8(bg, c.fill) {
						t.Errorf("dialog %s %s: cell %d,%d has background %v, want the panel's %v", dialog, c.name, x, y, bg, c.fill)
						y = r.Max.Y
						break
					}
				}
			}
		}
	}
}

// selection "bar" and "caret" bold the selected row's title in List as in Table: under
// focus "fg" the bold is the only cue beside a one-cell bar.
func TestListEmphasisesTheSelectedRow(t *testing.T) {
	for _, sel := range []string{"bar", "caret"} {
		f := quire(t)
		f.Expressions.Selection = sel
		th := theme.Resolve(f, true, false)
		rows := List(th, []Item{{Title: "ledger", Desc: "payments"}, {Title: "auth", Desc: "identity"}}, 0, 0, 30, 2, true)
		if c := cellAt(rows[0], "ledger"); c == nil || c.Style.Attrs&uv.AttrBold == 0 {
			t.Errorf("selection %s: the selected row's title isn't bold: %q", sel, ansi.Strip(rows[0]))
		}
		if c := cellAt(rows[1], "auth"); c == nil || c.Style.Attrs&uv.AttrBold != 0 {
			t.Errorf("selection %s: an unselected row is bold: %q", sel, ansi.Strip(rows[1]))
		}
	}
}

// Tree continues a multi-line item under the rail (glyphs.treeRail) while items follow,
// and under air after the last one, each continuation starting where the item's text does.
func TestTreeRail(t *testing.T) {
	f := theme.Default()
	for _, ascii := range []bool{false, true} {
		th := theme.Resolve(f, true, ascii)
		rows := Tree(th, []string{"build image\ncached", "watch error rate\n5 min"})
		want := []string{th.G.TreeMid + " build image", Fit(th.G.TreeRail, lipgloss.Width(th.G.TreeMid)) + " cached",
			th.G.TreeEnd + " watch error rate", strings.Repeat(" ", lipgloss.Width(th.G.TreeEnd)) + " 5 min"}
		for i := range want {
			if i >= len(rows) || ansi.Strip(rows[i]) != want[i] {
				t.Errorf("ascii=%v row %d: %q, want %q", ascii, i, ansi.Strip(rows[i]), want[i])
			}
		}
		if ascii && th.G.TreeRail != "|" {
			t.Errorf("ASCII rail is %q", th.G.TreeRail)
		}
	}
}

// Empty's mark above the title is the theme's bullet unless the screen names a glyph,
// and an Alert whose tag has no bar marks where its message starts with the bullet.
func TestBulletIsDrawn(t *testing.T) {
	f := theme.Default()
	f.Glyphs.Bullet = "▪"
	f.Expressions.Tag = "bold"
	th := theme.Resolve(f, true, false)
	if got := strings.TrimSpace(ansi.Strip(Empty(th, 40, "", "No incidents", "Nothing is paging.", "")[0])); got != "▪" {
		t.Errorf("Empty's mark is %q, want the bullet", got)
	}
	if got := strings.TrimSpace(ansi.Strip(Empty(th, 40, "◇", "No incidents", "Nothing is paging.", "")[0])); got != "◇" {
		t.Errorf("Empty's mark is %q, want the glyph the screen named", got)
	}
	if got := ansi.Strip(Alert(th, "warning", "PROD", "customer traffic", 40)); !strings.HasPrefix(got, "PROD ▪ customer traffic") {
		t.Errorf("Alert = %q, want the bullet between tag and message", got)
	}
}

// A dialog's rows are re-inked for its fill: charm's fgSubtle holds 2.21:1 on its bgPanel,
// under the 2.3 floor, so a subtle hint inside its dialog draws muted; on a page it stays
// subtle, and a level that holds is left alone.
func TestDialogLiftsHintsOnThePanel(t *testing.T) {
	tf, err := theme.Load(filepath.Join("..", "themes", "charm.json"))
	if err != nil {
		t.Skip("charm.json is the kit's; a copied project has its own themes")
	}
	th := theme.Resolve(tf, true, false)
	if theme.Contrast(th.P.FgSubtle, th.P.BgPanel) >= panelFloor {
		t.Fatal("charm's fgSubtle now holds on its bgPanel: pick another theme for this test")
	}
	hint := th.S.Subtle.Render("deployed 12m ago")
	rows := strings.Split(Dialog(th, 40, "Roll back?", []string{hint}, "", ""), "\n")
	var row string
	for _, r := range rows {
		if strings.Contains(ansi.Strip(r), "deployed") {
			row = r
		}
	}
	c := cellAt(row, "deployed")
	if c == nil || !same8(c.Style.Fg, th.P.FgMuted) {
		t.Errorf("a subtle hint in a dialog draws %v, want fgMuted %v", c.Style.Fg, th.P.FgMuted)
	}
	if c := cellAt(OnPanel(th, hint, th.P.Bg), "deployed"); !same8(c.Style.Fg, th.P.FgSubtle) {
		t.Errorf("on the page a subtle hint draws %v, want fgSubtle", c.Style.Fg)
	}
	if c := cellAt(OnPanel(th, th.S.Base.Render("deployed"), th.P.BgPanel), "deployed"); !same8(c.Style.Fg, th.P.Fg) {
		t.Errorf("fg on the panel draws %v, want fg", c.Style.Fg)
	}
}
