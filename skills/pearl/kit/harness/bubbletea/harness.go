// Package harness holds the tests every app built on the kit runs against itself: the
// footprint at every size, contrast, one focus, honest help, overlays at every colour
// depth, and that every advertised key is handled. An app implements App on its root
// model, lists its states in a Spec, and calls Run from one test:
//
//	func TestHarness(t *testing.T) {
//		harness.Run(t, harness.Spec{New: newHarnessApp, States: states, …})
//	}
//
// The tests read the project's themes from ../../themes/*.json, relative to the package
// that calls Run, and name no state but "help" and "toosmall".
package harness

import (
	"testing"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	uv "github.com/charmbracelet/ultraviolet"

	"github.com/jrschumacher/pearl-tui/skills/pearl/kit/theme/bubbletea"
	"github.com/jrschumacher/pearl-tui/skills/pearl/kit/ui/bubbletea"
)

// App is what the tests read from an app's root model.
type App interface {
	// Theme is the resolved theme the app draws with. The tests change it in place.
	Theme() *theme.Theme
	// SetProfile sets the colour profile the next Render downsamples to.
	SetProfile(colorprofile.Profile)
	// Render draws the current state at w×h, as View does.
	Render(w, h int) *ui.Canvas
	// SetState puts the app in a named state of Spec.States.
	SetState(s string)
	// OpenOverlay opens the overlay a state of Spec.Overlays shows, over the page shown.
	OpenOverlay(s string)
	// Overlay names the open overlay: "" for none, "help" for the keys overlay, and a
	// name of the app's own for each other one.
	Overlay() string
	// CloseOverlay closes the open overlay and leaves the page as it is.
	CloseOverlay()
	// OverlayRect is where the last Render drew the overlay.
	OverlayRect() uv.Rectangle
	// Body is the rectangle the last Render drew the page's body in, and FootY the
	// footer's row.
	Body() uv.Rectangle
	FootY() int
	// Key handles a key press, as Update does.
	Key(tea.KeyPressMsg) tea.Cmd
	// Actions is the current state's action table, and ToastAction the key the showing
	// toast offers in this state, or nil.
	Actions() []Action
	ToastAction() *Action
	// Toasts is the app's toast row.
	Toasts() *ui.Toasts
	// Last is the description of the last action Key ran.
	Last() string
	// HelpBody is the keys overlay's rows and HelpRows how many of them it can show.
	HelpBody() []string
	HelpRows() int
}

// Action is one entry of an action table: its binding, and whether the footer shows it.
type Action struct {
	B    key.Binding
	Show bool
}

// Spec is the app as the tests see it.
type Spec struct {
	// New makes the app with a theme, dark or light, in the theme's or ASCII glyphs.
	New func(f theme.File, dark, ascii bool) App
	// States is every named state; States[0] is the main screen. "help" is the keys
	// overlay's state and "toosmall" the too-small screen's.
	States []string
	// Overlays is the states that open a dialog, and Scrolls those of them whose body
	// may scroll at the floor.
	Overlays, Scrolls []string
	// OverlayPages is the states that draw a selection, a focused field or a caret, and
	// UnjudgedPages such states left out, each with why.
	OverlayPages  []string
	UnjudgedPages map[string]string
	// MinW×MinH is the app's floor: below it the too-small screen draws.
	MinW, MinH int
	// FrameSize is the size a state renders at when w×h is asked (the too-small state's
	// is fixed, below the floor), and FrameName the file its frame is written to.
	FrameSize func(s string, w, h int) (int, int)
	FrameName func(s string, w, h int) string
	// ANSI16 is more states to hold to the 16-colour checks, beside OverlayPages and
	// Overlays: a page that draws a status or a fill no other state does.
	ANSI16 []string
}

// sp is the app under test: one per test binary, set by Use.
var sp Spec

// Use names the app the package's helpers act on (RenderState, ToastKeysOffered…). Run
// calls it; an app whose own tests call a helper outside Run calls it from an init.
func Use(s Spec) {
	sp = s
	ansi16States = append(append(append([]string{}, s.OverlayPages...), s.Overlays...), s.ANSI16...)
}

// Run runs every harness test against the app.
func Run(t *testing.T, s Spec) {
	Use(s)
	for _, tc := range tests {
		t.Run(tc.name, tc.run)
	}
}
