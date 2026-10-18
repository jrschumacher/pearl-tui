# Cell art: marks, wordmarks, fields, splash, animation

For anything that is a *picture* printed into cells: a logo or brand mark, a wordmark, a
splash or loading screen, a repeating backdrop field, a badge, a spinner, an intro
animation. Screens (forms, lists, dashboards) are in `components.md` and `layout.md`.
Web and pixel instincts are wrong here. Follow these rules and run the loop before
showing anything.

## Glyph = Shape, Colour = Tone

A cell has one glyph, one fg and one bg. The glyph says how much of the cell is covered,
and the colour says how bright that coverage is. **Never use a glyph to express a colour
change** (shading a lit surface with `░▒▓`), and never use colour to express shape.

Sample geometry in square units (1 col = 1, 1 row = 2) so circles are round.

## Rules For Stills

1. **One vocabulary per edge.** The eye reads an edge as a line. A silhouette drawn with
   crisp blocks in some cells and `▒▓` in others reads as noise, even when each cell is
   locally right. Pick one treatment for the whole silhouette.
2. **Less information looks better.** Quadrants + eighths everywhere produce the worst
   result. Three coverage levels applied consistently beat them.
3. **Floor the faintest level.** `░` in a dark ink on a dark background is ~1.1:1: it is
   invisible as an edge and visible as dirt. For low-contrast inks use `▒ ▓ █` at
   ~0.3/0.6/0.85 coverage and never emit `░`. A high-contrast accent is never dithered.
   Apply it only to `█`/`▓` cells and leave `▒` edge cells in the base ink.
4. **Snap letterforms to the grid.** Stems start on a column boundary, tops and
   baselines on row boundaries, and strokes are an integer number of cells. Stems and top
   cuts are hard (in or out at 50%); only curves may take `▓`. A half-column fringe on a
   vertical stroke is the worst artefact a 1:2 grid can produce.
5. **Measure the source.** Derive the bounding box, radii, segment angles (histogram the
   pixel angles) and stroke widths from the asset before drawing. Drift (36° becoming 60°,
   a third tone that isn't in the brand) comes from eyeballing. Reproduce, don't redesign.
6. **Symmetry is exact.** If the source is symmetric, quantise one quadrant and mirror
   it. Left and right quantised on different sub-cell offsets is the first thing a viewer
   sees.
7. **Sizes.** Likeness needs ~40 columns. Below ~24, an inner letterform is a real
   character (`ν`, `v`), not a drawn stroke. Below ~12, use box drawing plus a glyph
   (`╭─╮│ν│╰─╯`, inline `(ν)`). Design the 40-column version first and the small ones
   by hand.
8. **Raster sampling has a limit.** A curve spanning fewer than ~8 rows cannot survive
   resampling. If likeness matters, draw it as explicit paths for that grid.
9. **Half-blocks with fg+bg are your 2× vertical resolution.** `▀` with fg = top colour
   and bg = bottom colour gives two independent "pixels" per cell. That is how image
   previews and crisp small marks are drawn. It needs truecolor or 256 colours, so
   provide a plain fallback.

## Wordmarks

`scripts/wordmark.py "name" --style half|shade|ascii|tracked --rows N` rasterises a real
font into cells and prints rows, an ANSI preview, and a Go `[]string`.

- **half** (`▀▄█`, two samples per cell): solid ink. Lowercase names with ascenders need
  4 rows.
- **shade** (`█▓▒` by coverage, no `░`) with three tones (pale/primary/muted) gives
  depth. Five rows is the floor (2-cell stems); four rows smears letters together.
- **ascii** (`#+.` or `@%*`): the fallback, and a voice of its own (BBS, NFO).
- **tracked text** (`T R U C T L` in bold primary): the quiet option when a mark does the
  work. Often the best one.
- Figlet fonts are portable but carry someone else's voice: `slant` is classic CLI and
  `ansi_shadow` is loud. Never the default `standard` font. It is the most recognisable
  "a model made this" banner there is.
- Keep a wordmark quieter than the mark: pale, not white. In an app header, a banner
  costs 4–6 rows. Earn them on a splash or a wizard's first step, and collapse to a
  one-row wordmark after.

## Logos And Marks

`scripts/logo2cells.py logo.png --cols 40 --mode half|shade|quad --colors 3` samples a
raster logo (export the SVG at ≥ 800px first) onto the grid. It quantises to the brand's
own colours (measured, k-means), applies rules 1–6, and prints rows plus an ANSI preview
you can render with `cellshot --grid`. Then **hand-fix** the cells the rules flag. The
generator is a starting point, not the answer.

## Fields (Repeating Backdrops)

A field is the terminal version of a tiled-icon brand background: two glyphs on a
lattice, one bright and one subtle, angled, with a gradient. It is in the kit as
`kit/ui/bubbletea/field.go`.

- **The angle is a shear.** A cell grid cannot rotate. Instead, each row of the lattice
  shifts by `shear` cells. A period of 8–10 columns × 2 rows with shear 3–5 reads as
  diagonal lanes. Shear 1 on a 1-row period turns `╱ ╲` into rails. Choose the period and
  shear before the glyphs.
- **The pair shares a shape.** `◆ ◇` and `▲ △` work because they differ only in fill. The
  subtle partner needs ink: `◇ △ ◦ ✧ ×`, never `·` or `∘`, which vanish at any tone.
- **Subtle means the same hue, pulled toward the background.** Mix the subtle glyph 50–55%
  toward the hue. At 35% it reads grey and the gradient looks as if it touched only the
  bright glyphs.
- **There are two gradient families.** Hue ramps between two brand colours are for
  identity. Fades (brightness ramps) are for getting out of the way.
- **Content sits in a clearing, not a box.** Dim the field under a mark with a quadratic
  radial falloff (squashed vertically for 1:2 cells) and extend it one lattice period
  past the content. Under a form, keep the field out of the content column entirely;
  glyphs in the blank rows between fields read as noise.
- **Cost.** An 80×24 field is ~240 styled cells, ~5 KB per frame. That is fine static, or
  drifting one cell per ~160 ms. Quantise it to two palette entries for 16 colours, and
  drop it entirely under NO_COLOR.

## Animation

10. **A colour-field animation changes colour, never shape.** Sweeps, pulses, highlight
    passes and CRT afterglow: precompute the glyph grid once, then restyle cells per
    frame. Take the phase from the frame index with easing (at 50 ms a frame: hold ~36
    frames, sweep ~16 ease-in-out for a mark).
11. **A transform animation (2D/3D) re-quantises every frame.** Apply every still rule to
    every frame, then:
    - Interior edges between two surfaces are **half-blocks with fg/bg** (`▌▐▀▄`), chosen
      by which half the dominant face fills. Never use a shade there: `▒▓` blend two
      tones into a smear.
    - **Prune detached slivers.** Keep a silhouette cell under 85% only if a neighbour is
      at least 75%.
    - **Lighting is continuous** (8 tones of one hue). Faces that share a visible edge
      must differ by at least 2 levels.
    - **Budget the angular step by cell travel**: an edge should move less than half a
      cell per frame.
    - No empty mid-clip frame. The resting pose must come from the same quantiser as the
      still logo.
12. **An organic/fluid animation** is a scalar field (metaballs) thresholded per frame.
    The seed count stays constant, centroids move at most 1 cell per frame, blobs merge
    on contact (falloff ≈ `(r²/d²)^1.5`), and cells of the target shape only ever gain
    (`level = max(prev, now)`). Apply hysteresis of ±0.05 and prune components under 3
    cells. At 40×20, a blob is ~3×2 cells and a neck is one cell. Be honest about whether
    the metaphor survives at this scale.
13. **A glyph-resolve animation** (decode, typewriter): noise glyphs in a dim tone that
    still clears ~1.6:1, re-rolled on alternate frames, locking along a wavefront with a
    one-frame flash. Keep the end hold short.
14. **Moving heads** (beam, sweep, wipe) run past the last cell by their full tail
    length. The penultimate frame equals the final one.
15. **Temporal lint**: look for cells that flip and flip back, empty mid-clip frames,
    identical-frame runs longer than 3 at ≤ 50 ms, change spikes, and a last frame that
    differs from the target.
16. **Keyframes are the code.** The GIF and the Go embed the same frame list. The Go
    component ticks and restyles; it never computes geometry at runtime unless the clip
    would exceed ~80 frames. Intros play ONCE and hand off to the idle state on an
    identical frame (test that in Go).
17. **Deterministic frames.** Every animated component renders from a frame number, as
    the kit's `Anim.Frame(n, elapsed)` and `Field{Drift: n/3}` do, sharing the renderer
    with `Update`. Goldens and GIFs pass `-frame N`; the live program ticks `m.frame`.
    A wall-clock phase can't be rendered by the harness.

## Spinners

- Bubbles' `spinner.Dot`/`MiniDot` (braille) are fine in real terminals. Choose by voice:
  braille is "busy, technical", the arc `◜◠◝◞◡◟` is "calm, circular", a line `▁▃▅▇▅▃` is
  "audio, live", and a gradient-cycling glyph run (Crush's thinking animation) is
  "alive, branded".
- One spinner per screen. A spinner always has a label saying *what* is happening ("indexing
  1,204 files", not "Loading..."), and it stops the moment the work does.
- Brand the spinner: colour it from the primary→secondary ramp, or cycle its glyphs
  through the brand's own vocabulary.

## Splash / Loading Composition

Centred block, top to bottom: the mark (24×12 at 80×24) · a blank row · the wordmark · a
blank row · spinner + status. Put the version bottom-left and the quit hint
bottom-right, with a brand field behind and a clearing under the block, drifting slowly.
Use the mark's own hold-and-sweep as the idle motion; the field drift is the only other
movement. The same wordmark rows become the app's header banner so the splash and the
app match. A splash that lasts longer than the work it covers is a lie. Show it only
while something actually loads, or for ≤ 600 ms on first run.

## The Loop (Run It; Never Ask The User For A Screenshot)

1. **Sample** the geometry or source raster at 4×8 per cell, centred on cell boundaries.
2. **Map** coverage to a glyph with one thresholds table per ink.
3. **Emit** `name-WxH.txt` (rows exactly W cells), `.ansi`, and Go data.
4. **Render honestly**: `cellshot name.ansi name.png --grid --cell 16x32`, plus a
   contact sheet (`--sheet`) of 6–8 frames at the phase boundaries and the GIF (`--gif`).
   Look at them with the Read tool.
5. **Lint**: unequal rows, mixed vocabulary along an edge, orphan cells, `░` under
   1.6:1, stems off column boundaries, broken symmetry, and the temporal rules.
6. **Critique adversarially** with a subagent (see `verify.md`). Give it the images,
   grids and source, and ask for ≤ 8 ranked defects that cite cells. Fix, re-render, and
   resume the same critic. Expect 2–4 rounds.
7. **Verify in Go**: a test renders every size and frame, asserts `lipgloss.Width(row) ==
   cols`, and checks the intro → idle hand-off.
