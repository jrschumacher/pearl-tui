package main

import (
	"image/color"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	uv "github.com/charmbracelet/ultraviolet"

	"github.com/jrschumacher/pearl-tui/skills/pearl/kit/harness/bubbletea"
	"github.com/jrschumacher/pearl-tui/skills/pearl/kit/theme/bubbletea"
	"github.com/jrschumacher/pearl-tui/skills/pearl/kit/ui/bubbletea"
)

// The showcase's own tests (screens_test.go, rollout_test.go, setup_test.go) call the
// harness's helpers outside harness.Run, so the app is named before any test runs. The
// 16-colour checks also judge every status mark (ladder) and a brand title over a
// filled header (setup-signin), which only the showcase draws.
func init() {
	spec.ANSI16 = []string{"ladder", "setup-signin"}
	harness.Use(spec)
}

// The harness's helpers under the names the showcase's own tests call, on *app.

func testThemes(t *testing.T) []string               { return harness.Themes(t) }
func testTheme(t *testing.T, name string) theme.File { return harness.ThemeFile(t, name) }
func press(k string) tea.KeyPressMsg                 { return harness.Press(k) }
func cellText(c *ui.Canvas, x, y int) string         { return harness.CellText(c, x, y) }
func rowText(c *ui.Canvas, y int) string             { return harness.RowText(c, y) }
func footerText(c *ui.Canvas, fy int) string         { return harness.FooterText(c, fy) }
func toastRow(m *app, w, h int) string               { return harness.ToastRow(harnessApp{m}, w, h) }
func toastKeysOffered(t *testing.T) int              { return harness.ToastKeysOffered(t) }
func slot16(c, def color.Color) int                  { return harness.Slot16(c, def) }
func (m *app) focusMarks(c *ui.Canvas) []uv.Position { return harness.FocusMarks(harnessApp{m}, c) }

func fixture(t *testing.T, state string) *app {
	t.Helper()
	m := newApp(testTheme(t, "charm.json"), true, false)
	m.setState(state)
	return m
}

func dialogFits(t *testing.T, name string, m *app, s string, w, h int) {
	t.Helper()
	harness.DialogFits(t, name, harnessApp{m}, s, w, h)
}

func pressToastKeys(t *testing.T, s, name string, m *app, fresh func() *app) {
	t.Helper()
	harness.PressToastKeys(t, s, name, harnessApp{m}, func() harness.App { return harnessApp{fresh()} })
}

func renderState(f theme.File, dark bool, prof colorprofile.Profile, state string, w, h int) (*app, *ui.Canvas) {
	m, c := harness.RenderState(f, dark, prof, state, w, h)
	return m.(harnessApp).app, c
}
