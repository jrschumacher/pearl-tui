package ui

import (
	"hash/fnv"
	"image/color"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"tuikit/theme"
)

// FrameInterval is the animation clock: 20 fps. Nobody sees 60 fps in a
// terminal, and every frame is bytes over SSH.
const FrameInterval = 50 * time.Millisecond

// Anim is a branded "working" indicator in the spirit of Crush's: a run of
// scrambling glyphs whose colour flows along the brand gradient, columns
// materialising one by one, then a label and an optional elapsed timer. The
// label gets no animated ellipsis: the spinner already moves, and "Loading..."
// is the default the skill's slop test removes.
//
// It is deterministic: Frame(n) depends only on n and the seed, so the same
// frame renders in a golden test and in the live program. Drive it from ONE
// shared clock (see the showcase's tick), and stop ticking when nothing on
// screen is animating.
type Anim struct {
	T      *theme.Theme
	Size   int    // scrambling cells (8–15)
	Label  string // "Thinking", "Deploying checkout-api"
	Seed   string // different seeds don't march in lockstep
	ramp   []color.Color
	births []int
}

const scramble = "0123456789abcdefABCDEF~!@#$%^&*+=_"

func NewAnim(t *theme.Theme, size int, label, seed string) *Anim {
	a := &Anim{T: t, Size: size, Label: label, Seed: seed}
	// A→B→A over 3× the width, so the colour can flow by offsetting into it
	n := size * 3
	half := lipgloss.Blend1D(n/2+1, t.P.Primary, t.P.Secondary)
	back := lipgloss.Blend1D(n-n/2+1, t.P.Secondary, t.P.Primary)
	a.ramp = append(half, back[1:]...)
	h := fnv.New32a()
	h.Write([]byte(seed))
	s := h.Sum32()
	a.births = make([]int, size)
	for i := range a.births {
		s = s*1664525 + 1013904223
		a.births[i] = int(s>>24) % 20
	}
	return a
}

// Frame renders frame n. elapsed (optional) follows the label.
func (a *Anim) Frame(n int, elapsed string) string {
	t := a.T
	suffix := ""
	if elapsed != "" {
		suffix = t.S.Subtle.Render(" " + elapsed)
	}
	if t.ASCII || t.X.Working != "scramble" {
		// the brand's own spinner, then the label: scramble is Crush's signature
		return Spinner(t, n/2) + t.S.Soft.Render(" "+a.Label) + suffix
	}
	var b strings.Builder
	h := fnv.New32a()
	for i := 0; i < a.Size; i++ {
		col := a.ramp[(i+n)%len(a.ramp)]
		st := lipgloss.NewStyle().Foreground(col)
		if n < a.births[i] {
			b.WriteString(st.Render("."))
			continue
		}
		h.Reset()
		h.Write([]byte{byte(i), byte(n / 2), byte(n / 512)}) // re-roll every 2 frames
		h.Write([]byte(a.Seed))
		ch := scramble[int(h.Sum32())%len(scramble)]
		b.WriteString(st.Render(string(ch)))
	}
	return b.String() + t.S.Soft.Render(" "+a.Label) + suffix
}

// Spinner is the small inline spinner (the theme's frames) at frame n.
func Spinner(t *theme.Theme, n int) string {
	return t.S.Working.Render(t.G.Spinner[n%len(t.G.Spinner)])
}

// BusyAnimates reports whether Busy changes between frames for this theme: true when
// the theme's own spinner frames are one cell (ASCII twins included, since the ASCII
// spinner is one cell whenever the theme's is). A clock driven for a busy row ticks
// only when it does (Law 8: tick while something visible moves).
func BusyAnimates(t *theme.Theme) bool {
	for _, f := range t.Unicode.Spinner {
		if lipgloss.Width(f) != 1 {
			return false
		}
	}
	return len(t.G.Spinner) > 1
}

// Busy is the working mark for a status column, one cell like the Dot beside it: the
// theme's spinner at frame n when its frames are one cell, else Dot's busy mark (the
// partial glyph, still), so a row in progress never pushes its name out of line. The
// theme's own frames decide (BusyAnimates), so unicode and ASCII agree: a three-cell
// spinner holds the partial mark in both, never a still ◐ here and a turning / there.
func Busy(t *theme.Theme, n int) string {
	if !BusyAnimates(t) {
		return Dot(t, "busy")
	}
	return Spinner(t, n)
}
