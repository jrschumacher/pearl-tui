package ui

import (
	"path/filepath"
	"strings"
	"testing"

	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"tuikit/theme"
)

// testThemes finds the theme JSONs in the kit (ui/../themes) or in a project that
// copied ui/ to internal/ui and its theme to themes/ (kit/README.md). Finding none
// fails: a glob that matches nothing would pass every test without running one.
func testThemes(t *testing.T) []string {
	t.Helper()
	for _, dir := range []string{"../themes", "../../themes"} {
		if files, _ := filepath.Glob(filepath.Join(dir, "*.json")); len(files) > 0 {
			return files
		}
	}
	t.Fatal("no theme JSON in ../themes or ../../themes")
	return nil
}

// testTheme is halyard in the kit, and the project's own theme after a copy.
func testTheme(t *testing.T, ascii bool) *theme.Theme {
	t.Helper()
	files := testThemes(t)
	path := files[0]
	for _, f := range files {
		if filepath.Base(f) == "halyard.json" {
			path = f
		}
	}
	f, err := theme.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	return theme.Resolve(f, true, ascii)
}

// Both spinners turn the brand's frames in the primary colour, never
// Bubbles' Line or Huh's #F780E2 Dot.
func TestSpinnersAreThemed(t *testing.T) {
	th := testTheme(t, false)
	bs := BubblesSpinner(th)
	if strings.Join(bs.Spinner.Frames, "") != strings.Join(th.G.Spinner, "") {
		t.Fatalf("bubbles spinner frames %v", bs.Spinner.Frames)
	}
	if v := bs.View(); v != th.S.Primary.Render(th.G.Spinner[0]) {
		t.Fatalf("bubbles spinner renders %q", v)
	}
	hs := HuhSpinner(th, "Connecting to staging")
	v := hs.View()
	if !strings.HasPrefix(ansi.Strip(v), th.G.Spinner[0]+" Connecting") {
		t.Fatalf("huh spinner renders %q", ansi.Strip(v))
	}
	if strings.Contains(v, "247;128;226") || strings.Contains(ansi.Strip(v), "Loading") {
		t.Fatalf("huh spinner kept a default: %q", v)
	}
}

// A blurred input shows its value behind blank prompt cells; focused, the
// prompt glyph comes back in the same cells, so the text never moves.
func TestInputBlurredHasNoPrompt(t *testing.T) {
	th := testTheme(t, false)
	v := "payments"
	in := NewInput(th).Title("Namespace").Value(&v)
	in.WithWidth(40)
	rows := strings.Split(ansi.Strip(in.View()), "\n")
	last := rows[len(rows)-1]
	if strings.Contains(last, th.G.Prompt) || !strings.Contains(last, "payments") {
		t.Fatalf("blurred input row %q", last)
	}
	in.Focus()
	rows = strings.Split(ansi.Strip(in.View()), "\n")
	focused := rows[len(rows)-1]
	col := func(r string) int { return lipgloss.Width(r[:strings.Index(r, "payments")]) }
	if col(focused) != col(last) {
		t.Fatalf("focus moved the text: %q vs %q", focused, last)
	}
}

// Confirm puts the action first, aligned left, with the stated default.
func TestConfirmOrderAndDefault(t *testing.T) {
	th := testTheme(t, false)
	var v bool
	c := Confirm(th, "Delete it?", "Delete", "Keep", false, &v)
	f := NewForm(th, 60, 8, huh.NewGroup(c))
	f.Init()
	txt := ansi.Strip(f.View())
	row := ""
	for _, l := range strings.Split(txt, "\n") {
		if strings.Contains(l, "Delete ]") {
			row = l
		}
	}
	if strings.Index(row, "Delete") > strings.Index(row, "Keep") || strings.Index(row, "[") > 6 {
		t.Fatalf("button row %q", row)
	}
	if v {
		t.Fatal("the default was false")
	}
}
