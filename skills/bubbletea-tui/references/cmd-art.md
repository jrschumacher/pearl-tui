# art — marks, wordmarks, fields, splash screens

Everything is in `cell-art.md`; this is the order to do it in.

1. **Source.** Get the real logo (SVG exported to PNG at ≥ 800 px), the typeface file,
   and the measured colours. Measure the angles, stroke widths and radii before drawing.
2. **Wordmark.** `scripts/wordmark.py "<name>" --font <brand.ttf> --style half|shade|ascii
   --rows N --ansi wm.ansi`. Try 2–3 styles and row counts, render a sheet
   (`cellshot --sheet --cell 16x32`), and pick by legibility at 80×24. Tracked text is
   often the right answer.
3. **Mark.** `scripts/logo2cells.py logo.png --cols 40 --mode half|shade|quad --palette
   "<hexes>"`, render with `--grid`, then hand-fix by the still rules (one vocabulary per
   edge, floor the faintest level, snap stems, exact symmetry). Draw 24- and 12-column
   versions by hand.
4. **Field.** Choose the motif pair (same shape, different fill), the period, the shear
   and the gradient. Draw it with `ui.Field` behind a clearing.
5. **Splash.** The composition in `cell-art.md`. Render it at 80×24 and 120×40.
6. **Motion** (if any): `cmd-animate.md`.
7. **Loop.** Render → lint → critic with the images and the source → fix → resume the
   critic. Expect 2–4 rounds. A Go test asserts every frame's footprint and the intro →
   idle hand-off.
8. **Wire it in.** Put the wordmark rows in the theme (`wordmark.rows`) so the header
   banner and the splash match, and the mark grid in `ui/<name>_grid.go`.
