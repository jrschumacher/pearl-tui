package cellshot

import (
	"image"
	"image/color"
	"image/gif"
	"io"
	"sort"
)

// WriteGIF encodes frames as a looping animation, delay ms per frame. The palette is
// built from the frames' own colours: exact when they hold 256 or fewer, otherwise the
// most frequent half exactly plus a median cut of the rest, each pixel mapped to the
// nearest entry without dithering (dither noise reads as a rendering defect in cell art).
func WriteGIF(w io.Writer, frames []*image.RGBA, ms int) error {
	hist := map[uint32]int{}
	for _, f := range frames {
		p := f.Pix
		for i := 0; i < len(p); i += 4 {
			hist[uint32(p[i])<<16|uint32(p[i+1])<<8|uint32(p[i+2])]++
		}
	}
	pal := medianCut(hist, 256)
	index := map[uint32]uint8{}
	for i, c := range pal {
		rgba := c.(color.RGBA)
		index[uint32(rgba.R)<<16|uint32(rgba.G)<<8|uint32(rgba.B)] = uint8(i)
	}
	nearest := func(k uint32) uint8 {
		if i, ok := index[k]; ok {
			return i
		}
		r, g, b := int(k>>16), int(k>>8&255), int(k&255)
		best, bd := 0, 1<<30
		for i, c := range pal {
			p := c.(color.RGBA)
			dr, dg, db := r-int(p.R), g-int(p.G), b-int(p.B)
			if d := dr*dr + dg*dg + db*db; d < bd {
				best, bd = i, d
			}
		}
		index[k] = uint8(best)
		return uint8(best)
	}
	anim := &gif.GIF{LoopCount: 0}
	for _, f := range frames {
		b := f.Rect
		pm := image.NewPaletted(image.Rect(0, 0, b.Dx(), b.Dy()), pal)
		for y := 0; y < b.Dy(); y++ {
			src := f.Pix[y*f.Stride:]
			dst := pm.Pix[y*pm.Stride:]
			for x := 0; x < b.Dx(); x++ {
				dst[x] = nearest(uint32(src[x*4])<<16 | uint32(src[x*4+1])<<8 | uint32(src[x*4+2]))
			}
		}
		anim.Image = append(anim.Image, pm)
		anim.Delay = append(anim.Delay, (ms+5)/10)
		anim.Disposal = append(anim.Disposal, gif.DisposalBackground)
	}
	return gif.EncodeAll(w, anim)
}

type colourCount struct {
	c [3]uint8
	n int
}

// medianCut reduces a histogram to at most k colours. The k/2 most frequent colours
// (backgrounds, fills, text colours) are kept exactly; the rest (mostly anti-aliasing
// blends) are split, box with the widest channel range first, at the weighted median
// until the palette is full, and each box is averaged.
func medianCut(hist map[uint32]int, k int) color.Palette {
	all := make([]colourCount, 0, len(hist))
	for key, n := range hist {
		all = append(all, colourCount{[3]uint8{uint8(key >> 16), uint8(key >> 8), uint8(key)}, n})
	}
	// Deterministic order regardless of map iteration.
	sort.Slice(all, func(i, j int) bool {
		a, b := all[i].c, all[j].c
		return uint32(a[0])<<16|uint32(a[1])<<8|uint32(a[2]) < uint32(b[0])<<16|uint32(b[1])<<8|uint32(b[2])
	})
	if len(all) <= k {
		pal := make(color.Palette, len(all))
		for i, cc := range all {
			pal[i] = color.RGBA{cc.c[0], cc.c[1], cc.c[2], 255}
		}
		return pal
	}
	byCount := append([]colourCount(nil), all...)
	sort.SliceStable(byCount, func(i, j int) bool { return byCount[i].n > byCount[j].n })
	exact := byCount[:k/2]
	k -= len(exact)
	boxes := [][]colourCount{byCount[len(exact):]}
	spread := func(b []colourCount) (int, int) {
		lo, hi := [3]int{255, 255, 255}, [3]int{}
		for _, cc := range b {
			for ch := 0; ch < 3; ch++ {
				lo[ch] = min(lo[ch], int(cc.c[ch]))
				hi[ch] = max(hi[ch], int(cc.c[ch]))
			}
		}
		best, ch := -1, 0
		for i := 0; i < 3; i++ {
			if hi[i]-lo[i] > best {
				best, ch = hi[i]-lo[i], i
			}
		}
		return ch, best
	}
	for len(boxes) < k {
		bi, bch, bs := -1, 0, 0
		for i, b := range boxes {
			if len(b) < 2 {
				continue
			}
			ch, s := spread(b)
			if s > bs {
				bi, bch, bs = i, ch, s
			}
		}
		if bi < 0 {
			break
		}
		b := boxes[bi]
		sort.SliceStable(b, func(i, j int) bool { return b[i].c[bch] < b[j].c[bch] })
		total := 0
		for _, cc := range b {
			total += cc.n
		}
		acc, cut := 0, 1
		for i, cc := range b {
			acc += cc.n
			if acc*2 >= total {
				cut = max(1, min(i+1, len(b)-1))
				break
			}
		}
		boxes[bi] = b[:cut]
		boxes = append(boxes, b[cut:])
	}
	pal := make(color.Palette, 0, len(exact)+len(boxes))
	for _, cc := range exact {
		pal = append(pal, color.RGBA{cc.c[0], cc.c[1], cc.c[2], 255})
	}
	for _, b := range boxes {
		var r, g, bl, n int
		for _, cc := range b {
			r += int(cc.c[0]) * cc.n
			g += int(cc.c[1]) * cc.n
			bl += int(cc.c[2]) * cc.n
			n += cc.n
		}
		pal = append(pal, color.RGBA{uint8((r + n/2) / n), uint8((g + n/2) / n), uint8((bl + n/2) / n), 255})
	}
	return pal
}
