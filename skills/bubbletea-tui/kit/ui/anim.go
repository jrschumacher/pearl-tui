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
// materialising one by one, then a label and an animated ellipsis or an
// elapsed timer.
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

// Frame renders frame n. elapsed (optional) replaces the ellipsis.
func (a *Anim) Frame(n int, elapsed string) string {
	t := a.T
	if t.ASCII {
		return t.S.Muted.Render(t.G.Spinner[n%len(t.G.Spinner)] + " " + a.Label + "...")
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
	label := t.S.Soft.Render(" " + a.Label)
	if elapsed != "" {
		return b.String() + label + t.S.Subtle.Render(" "+elapsed)
	}
	dots := []string{"", ".", "..", "..."}[(n/8)%4]
	return b.String() + label + t.S.Subtle.Render(Fit(dots, 3))
}

// Spinner is the small inline spinner (the theme's frames) at frame n.
func Spinner(t *theme.Theme, n int) string {
	return t.S.Primary.Render(t.G.Spinner[n%len(t.G.Spinner)])
}
