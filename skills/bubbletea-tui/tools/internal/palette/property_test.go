package palette

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"

	"tuitools/internal/oklch"
)

// deriveInput is one palette derive invocation.
type deriveInput struct {
	primary, secondary, accent, bg, lightBg string
	tint                                    float64
}

// deriveCorpus is a broad sweep of derive's inputs: an OKLCH grid of primaries
// (lightness × chroma × hue, gamut-mapped), each with a secondary, accent and tint
// cycled in, then 8,000 random hex inputs with and without a secondary and accent.
func deriveCorpus() []deriveInput {
	var in []deriveInput
	tints := []float64{0.012, 0, 0.03, 0.06}
	i := 0
	for l := 0.02; l <= 0.99; l += 0.06 {
		for _, c := range []float64{0, 0.04, 0.08, 0.13, 0.19, 0.26, 0.37} {
			for h := 0.0; h < 360; h += 20 {
				d := deriveInput{primary: oklch.FromLCH(l, c, h), tint: tints[i%len(tints)]}
				if i%3 == 1 {
					d.secondary = oklch.FromLCH(1-l, c, h+90)
				}
				if i%5 == 2 {
					d.accent = oklch.FromLCH(0.5+l/2, c/2, h+200)
				}
				in = append(in, d)
				i++
			}
		}
	}
	rng := rand.New(rand.NewPCG(8000, 1))
	for range 8000 {
		d := deriveInput{primary: randHex(rng), tint: rng.Float64() * 0.06}
		if rng.IntN(2) == 0 {
			d.secondary = randHex(rng)
		}
		if rng.IntN(3) == 0 {
			d.accent = randHex(rng)
		}
		in = append(in, d)
	}
	return in
}

func randHex(rng *rand.Rand) string { return fmt.Sprintf("#%06X", rng.IntN(1<<24)) }

// problems lists the checks a role set fails; onOnly keeps the on-colour checks.
func problems(r map[string]string, onOnly bool) []string {
	var out []string
	for _, c := range Checks {
		if onOnly && !strings.HasPrefix(c.A, "on") && c.A != "fgSelected" {
			continue
		}
		if x := oklch.Contrast(r[c.A], r[c.B]); x < c.Min || (c.Max != 0 && x > c.Max) {
			out = append(out, fmt.Sprintf("%s %s on %s %s: %.2f (want %s)", c.A, r[c.A], c.B, r[c.B], x, c.want))
		}
	}
	return out
}

// derive's output passes its own check, dark and light, for any brand colours with the
// backgrounds derive picks: the brand's primary keeps its hex and the on-colours move
// instead (theme.OnInk). With the on-colours left out, the engine's own inks
// (FillOnColours, every 8th input) pass too.
func TestDeriveAlwaysPassesCheck(t *testing.T) {
	t.Parallel()
	corpus := deriveCorpus()
	if len(corpus) < 10000 {
		t.Fatalf("corpus has %d inputs", len(corpus))
	}
	failed := 0
	report := func(d deriveInput, mode, what string, p []string) {
		if failed++; failed <= 5 {
			t.Errorf("derive %+v %s%s:\n  %s", d, mode, what, strings.Join(p, "\n  "))
		}
	}
	for i, d := range corpus {
		for _, m := range Derive(d.primary, d.secondary, d.accent, d.bg, d.lightBg, d.tint) {
			if p := problems(m.Roles, false); len(p) > 0 {
				report(d, m.Name, "", p)
				continue
			}
			if i%8 != 0 {
				continue
			}
			bare := map[string]string{}
			for k, v := range m.Roles {
				if !strings.HasPrefix(k, "on") && k != "fgSelected" {
					bare[k] = v
				}
			}
			if err := FillOnColours(bare, m.Name == "dark"); err != nil {
				t.Fatal(err)
			}
			if p := problems(bare, false); len(p) > 0 {
				report(d, m.Name, ", engine on-colours", p)
			}
		}
	}
	if failed > 0 {
		t.Errorf("%d of %d role sets fail check", failed, 2*len(corpus))
	}
}

// With a background of the brand's own, the background roles are the brand's call (a
// mid-grey bg cannot carry fg at 7:1, see testdata/derive/fails), but the on-colours
// still always clear their floors: they depend only on the fill.
func TestDeriveOnColoursPassWithAnyBackground(t *testing.T) {
	t.Parallel()
	rng := rand.New(rand.NewPCG(2000, 2))
	for range 2000 {
		d := deriveInput{primary: randHex(rng), bg: randHex(rng), lightBg: randHex(rng), tint: rng.Float64() * 0.06}
		if rng.IntN(2) == 0 {
			d.secondary = randHex(rng)
		}
		for _, m := range Derive(d.primary, d.secondary, d.accent, d.bg, d.lightBg, d.tint) {
			if p := problems(m.Roles, true); len(p) > 0 {
				t.Fatalf("derive %+v %s:\n  %s", d, m.Name, strings.Join(p, "\n  "))
			}
		}
	}
}

// The brand's primary stays the primary: on #2E8B57 (the miss, where neither default
// ink reached 4.5:1) derive keeps the hex and deepens the ink instead.
func TestDeriveKeepsMidLuminancePrimary(t *testing.T) {
	for _, m := range Derive("#2E8B57", "", "", "", "", 0.012) {
		if m.Name == "dark" && m.Roles["primary"] != "#2E8B57" {
			t.Errorf("dark primary %s, want the brand's #2E8B57", m.Roles["primary"])
		}
		if r := oklch.Contrast(m.Roles["onPrimary"], m.Roles["primary"]); r < 4.5 {
			t.Errorf("%s onPrimary %s on %s: %.2f, want ≥ 4.5", m.Name, m.Roles["onPrimary"], m.Roles["primary"], r)
		}
	}
}
