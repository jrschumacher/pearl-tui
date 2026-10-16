package theme

import (
	"image/color"
	"math/rand/v2"
	"testing"
)

// OnInk clears its floor on any fill with any pair of inks, and keeps OnColor's pick
// whenever that already clears it, so a theme that passed before draws the same.
func TestOnInkAlwaysClearsTheFloor(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	rgb := func() color.Color {
		return color.RGBA{R: uint8(rng.IntN(256)), G: uint8(rng.IntN(256)), B: uint8(rng.IntN(256)), A: 0xff}
	}
	for range 20000 {
		fill, a, b := rgb(), rgb(), rgb()
		for _, floor := range []float64{3.5, 4.5} {
			ink := OnInk(fill, a, b, floor)
			if c := Contrast(ink, fill); c < floor {
				t.Fatalf("OnInk(%v, %v, %v, %.1f) = %v at %.2f", fill, a, b, floor, ink, c)
			}
			if pick := OnColor(fill, a, b); Contrast(pick, fill) >= floor && ink != pick {
				t.Fatalf("OnInk(%v, %v, %v, %.1f) = %v, but OnColor's %v already clears", fill, a, b, floor, ink, pick)
			}
		}
	}
}

// A mid-luminance brand fill (#2E8B57) keeps its hex; the near-black ink deepens to
// clear 4.5:1 and keeps its hue.
func TestOnInkDeepensOnMidFill(t *testing.T) {
	fill := color.RGBA{0x2E, 0x8B, 0x57, 0xff}
	dark, light := color.RGBA{0x00, 0x17, 0x09, 0xff}, color.RGBA{0xF4, 0xFD, 0xF7, 0xff}
	ink := OnInk(fill, light, dark, 4.5)
	if Contrast(ink, fill) < 4.5 || Luminance(ink) >= Luminance(dark) {
		t.Fatalf("ink %v at %.2f; want darker than %v and ≥ 4.5", ink, Contrast(ink, fill), dark)
	}
	if r, g, b, _ := ink.RGBA(); !(g > r && g > b) {
		t.Errorf("ink %v lost the green hue", ink)
	}
	p := Resolve(File{Roles: map[string]Roles{"dark": {Primary: "#2E8B57", Bg: "#1A1A1A", Fg: "#D0D0D0"}}}, true, false).P
	if c := Contrast(p.OnPrimary, p.Primary); c < 4.5 {
		t.Errorf("engine onPrimary %v on #2E8B57 at %.2f; want ≥ 4.5", p.OnPrimary, c)
	}
}
