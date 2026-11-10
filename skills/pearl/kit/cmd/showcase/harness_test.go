package main

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	uv "github.com/charmbracelet/ultraviolet"

	"github.com/jrschumacher/pearl-tui/skills/pearl/kit/harness/bubbletea"
	"github.com/jrschumacher/pearl-tui/skills/pearl/kit/theme/bubbletea"
	"github.com/jrschumacher/pearl-tui/skills/pearl/kit/ui/bubbletea"
)

// spec is this app as the harness sees it: the tables in main.go and the root model.
var spec = harness.Spec{
	New:           func(f theme.File, dark, ascii bool) harness.App { return harnessApp{newApp(f, dark, ascii)} },
	States:        states,
	Overlays:      overlays,
	Scrolls:       scrolls,
	OverlayPages:  overlayPages,
	UnjudgedPages: unjudgedPages,
	MinW:          minW,
	MinH:          minH,
	FrameSize:     frameSize,
	FrameName:     frameName,
}

// The kit's harness tests, run against this app: the footprint at every size, contrast,
// one focus, honest help, every overlay at every colour depth, every advertised key
// handled. They read the tables in main.go and the app through harnessApp.
func TestHarness(t *testing.T) { harness.Run(t, spec) }

// harnessApp is the root model as the harness reads it.
type harnessApp struct{ *app }

func (h harnessApp) Theme() *theme.Theme               { return h.t }
func (h harnessApp) SetProfile(p colorprofile.Profile) { h.profile = p }
func (h harnessApp) Render(w, hh int) *ui.Canvas       { return h.render(w, hh) }
func (h harnessApp) SetState(s string)                 { h.setState(s) }
func (h harnessApp) OpenOverlay(s string)              { h.openOverlay(s) }
func (h harnessApp) CloseOverlay()                     { h.ov = ovNone }
func (h harnessApp) OverlayRect() uv.Rectangle         { return h.ovRect }
func (h harnessApp) Body() uv.Rectangle                { return h.body }
func (h harnessApp) FootY() int                        { return h.footY }
func (h harnessApp) Key(msg tea.KeyPressMsg) tea.Cmd   { return h.key(msg) }
func (h harnessApp) Toasts() *ui.Toasts                { return &h.toasts }
func (h harnessApp) Last() string                      { return h.last }
func (h harnessApp) HelpBody() []string                { return h.helpBody() }
func (h harnessApp) HelpRows() int                     { return h.helpRows() }

// Overlay names the open overlay: "" for none, "help" for the keys overlay.
func (h harnessApp) Overlay() string {
	switch h.ov {
	case ovNone:
		return ""
	case ovHelp:
		return "help"
	}
	return "overlay " + string(rune('0'+int(h.ov)))
}

func (h harnessApp) Actions() []harness.Action {
	as := h.actions()
	out := make([]harness.Action, len(as))
	for i, a := range as {
		out[i] = harness.Action{B: a.b, Show: a.show}
	}
	return out
}

func (h harnessApp) ToastAction() *harness.Action {
	if a := h.toastAction(); a != nil {
		return &harness.Action{B: a.b, Show: a.show}
	}
	return nil
}
