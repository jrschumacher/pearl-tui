# Verify: you cannot see a terminal, so build eyes and use them

A model writing a TUI is designing blind. The code compiles, `View()` returns a string,
and nobody has looked at it. Every defect in `anti-patterns.md` survives that. This loop
is how you see. Run it before you show the user anything, and never ask the user for a
screenshot.

## The Loop

```
1. RENDER   go run ./cmd/<app> -out frames -sizes 80x24,120x40          (every state)
            go run ./cmd/<app> -out frames/light -light
            go run ./cmd/<app> -out frames/256 -profile ansi256
            go run ./cmd/<app> -out frames/16 -profile ansi
            NO_COLOR=1 go run ./cmd/<app> -out frames/nocolor
            go run ./cmd/<app> -out frames/ascii -profile ascii -ascii
2. GOLDEN   go run ./cmd/<app> -golden -w 80 -h 24 && … -w 120 -h 40      (exit 0 or stop)
            go test ./...                                                 (footprint at the kit's 5 sizes × themes)
3. LINT     tuilint frames frames frames/256 frames/16 frames/nocolor --theme themes/<brand>.json --bg <bg>
            tuilint frames frames/ascii --ascii --theme themes/<brand>.json --bg <bg>
            tuilint frames frames/light --theme themes/<brand>.json --bg <light bg>
            (lint every directory you rendered: light, 256, 16, no colour and ASCII break where dark
            truecolour passes, and LOOK at the overlays there: a dialog's edge is the first to go;
            --ascii fails any glyph that leaked into an ASCII frame [F-ASCII];
            a frame is linted as a dialog only when its name carries dialog, palette, confirm,
            modal, overlay, popup, help or keys: any other overlay name lints the scrim and
            fails F-CONTRAST on it, so rename the state, not the theme)
            tuilint code .                                                (tuivet ./... in go vet or an editor: same rules)
            tuilint spec TUI.md                                           (after any key-table change)
4. LOOK     cellshot frames/*-80x24.ansi frames/toosmall-*.ansi --sheet sheet-80.png
            cellshot frames/*-120x40.ansi --sheet sheet-120.png --cell 8x16
            cellshot frames/light/*-80x24.ansi --sheet sheet-light.png --light
            cellshot frames/ascii/*-80x24.ansi --sheet sheet-ascii.png
            (globs, not a state list: toosmall renders at its own size, 50x14, and a
            named list drops it)
            → Read the PNGs. Run the slop test (anti-patterns.md) on each.
5. CRITIQUE spawn a critic subagent (below) with the PNGs, goldens, TUI.md and code
6. FIX      one change per defect; re-run 1–4; resume the SAME critic for round 2
```

Expect 2–4 rounds. Stop when the critic says ship and your own slop test passes.

## Reading A Render Honestly

- `cellshot` draws box drawing, blocks, shades, braille and sextants geometrically, the
  way Ghostty, Kitty, WezTerm and iTerm2 do. Treat what you see as what users see. It
  reports glyphs no font could draw (users may see tofu too) and emoji.
- `--cell 16x32` to inspect detail. `--grid` for cell art. `--sheet` for contact sheets.
  `--gif --ms 50` for animation frames.
- Look at 80×24 at full size first. That's the floor most users hit.
- Zoom into the focus. Is there exactly one marked thing? Does the selected row's
  background reach the full width with no holes?

## What The Lint Catches (And Doesn't)

`tuilint` catches the mechanical defects: row widths, emoji, default palettes,
off-palette colours, contrast under 2.3:1, box soup, rainbows, missing text levels,
excessive bold, edge-touching content, blocking calls, hex in views, v1 imports, fast
ticks, fmt.Print, raw-string key matching, unhandled resize, and in `TUI.md` keys a
terminal can't tell apart, duplicated keys, globals a text box eats and ambiguous-width
glyphs with no ASCII twin.

It cannot see a weak hierarchy, a screen with no identity, help that lies about what a
key does in this state, copy in the wrong voice, or a layout that wastes the 120×40
room. That's the critic's job, and yours.

## The Critic

Spawn a fresh subagent. It has not seen your reasoning, so it judges the screen, not the
intent. Give it the files, not a description. No subagent tool: re-read only the PNGs
and goldens (not your code) in a fresh message with the prompt below, and say so in the
hand-off, because a self-review that has the code in context grades the intent.

```
You are reviewing a terminal UI built with Bubble Tea. Be adversarial: your job is to
find what a demanding terminal user and a brand designer would reject. You cannot run
it; you have renders.

Files:
  - sheet-80.png, sheet-120.png: every state at 80×24 and 120×40 (honest renders)
  - sheet-light.png, sheet-ascii.png: the light role set; ASCII glyphs with no colour
  - golden-80x24.txt: the plain-text frames
  - TUI.md (§ Brand read, § Never, § Screens) and themes/<brand>.json: what the brand
    and the screens are supposed to be
  - the app's render code: <paths>
Rubric (score each 0–4, cite rows/cells, not impressions; 0 = a default or a break,
2 = works but surprises, 4 = designed):
  1. Identity: with colour stripped, is it recognisably this brand? Is the read's
     "remember" line visible? Any default (pink-purple, emoji, figlet, Loading...)?
  2. Hierarchy: 3–4 text levels used correctly; what the eye lands on first is what
     matters most; one alert at most.
  3. Focus and flow: exactly one visible focus; where am I, what can I do, how do I go
     back; help keys match what the screen actually does in this state (read the code;
     a toggle's label names the next state).
  4. Layout: budgets respected; 80×24, 120×40 and 200×60 are the same design; room
     used, not stretched (layout.md § Use The Room: a blank sidebar or unused columns
     beside the body at 120×40 is a stretch); nothing touching edges; no box soup.
  5. States: empty, loading, error, partial, too-small all designed; copy says what
     happened and what to do.
  6. Terminal craft: NO_COLOR and ASCII still readable; light set reviewed; no holes in
     filled bars; nothing animates while idle.
Output:
  - verdict first: SHIP (20+) / FIX (14–19) / RETHINK (below 14), and a total out of 24
  - ≤ 10 defects ranked P0 (broken) → P3 (polish), each with the state, the cells, why,
    and the fix in renderer terms (which component, which theme key)
  - the one change that would help most
  - anything you suspect is a behaviour bug the renders hide
```

For round two, resume the **same** critic (SendMessage to it) with the new renders and
"here is what changed". It grades its own items, which keeps the bar from moving.

## Reporting To The User

Show the user the result, not the process:

- the contact sheets (80×24 and 120×40, plus light) as files they can open
- what the brand read decided, in 3–5 lines
- the critic's last verdict and score, and anything you chose not to fix (with the reason)
- how to run it: `go run ./cmd/<app>` and the harness flags
