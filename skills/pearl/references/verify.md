# Verify: you cannot see a terminal, so build eyes and use them

A model writing a TUI is designing blind. The code compiles, `View()` returns a string,
and nobody has looked at it. Every defect in `anti-patterns.md` survives that. This loop
is how you see. Run it before you show the user anything, and never ask the user for a
screenshot.

*Bubble Tea: the RENDER and GOLDEN commands in § The Loop run the kit's Bubble Tea harness,
and LINT's `tuilint code` lints Bubble Tea code; any harness that writes the same `.ansi`
frames takes the rest of the loop unchanged.*

## The Loop

```
1. RENDER   go run ./cmd/<app> -out frames -sizes 80x24,120x40          (every state)
            go run ./cmd/<app> -out frames/light -light
            go run ./cmd/<app> -out frames/256 -profile ansi256
            go run ./cmd/<app> -out frames/16 -profile ansi
            NO_COLOR=1 go run ./cmd/<app> -out frames/nocolor
            go run ./cmd/<app> -out frames/ascii -profile ascii -ascii
            no harness (an existing app, another framework): capture it as it runs,
            a tape per flow (cmd-critique.md §1); the frames are in capture/frames/
            tuicap -sizes 80x24,120x40 -tape <flow>.tape -out capture -- ./bin/<app>
            tuicap -profile ansi -tape <flow>.tape -out capture-16 -- ./bin/<app>
            (lint capture-16 without --theme: the emulator draws the 16 slots in its
            own colours, so off-palette there means nothing; the harness's -profile
            ansi frames are the ones to hold to the theme)
2. GOLDEN   go run ./cmd/<app> -golden -w 80 -h 24 && … -w 120 -h 40      (exit 0 or stop)
            go test ./...                                                 (footprint at the kit's 5 sizes × themes)
3. LINT     tuilint frames frames frames/256 frames/16 frames/nocolor --theme themes/<brand>.json --bg <bg> --spec TUI.md
            tuilint frames frames/ascii --ascii --unicode frames --theme themes/<brand>.json --bg <bg> --spec TUI.md
            tuilint frames frames/light --theme themes/<brand>.json --bg <light bg> --spec TUI.md
            (--spec reads § Decisions' `cuts: [<screen>:] <column header or region>` lines:
            a column cut on purpose, paths, URLs, code, is not a budget too narrow [F-TRUNC])
            (lint every directory you rendered: light, 256, 16, no colour and ASCII break where dark
            truecolour passes, and LOOK at the overlays there: a dialog's edge is the first to go;
            --ascii fails any glyph that leaked into an ASCII frame, and --unicode a "?" where the
            unicode frame of the state has a glyph or a letter [F-ASCII]; an overlay
            failing F-CONTRAST on its scrim may be misnamed: the lint finds a dialog by
            shape only where its scrim dims or hides the page, so name every overlay frame
            as cmd-shape.md step 3 says)
            tuilint code .                                                (tuivet ./... in go vet or an editor: same rules)
            tuilint spec TUI.md --frames frames                           (after any key-table change;
            each state's footer in its golden must be its row's keys [S-GOLDEN])
4. LOOK     cellshot frames/*-80x24.ansi frames/toosmall-*.ansi --sheet sheet-80.png
            cellshot frames/*-120x40.ansi --sheet sheet-120.png --cell 8x16
            cellshot frames/light/*-80x24.ansi --sheet sheet-light.png --light
            cellshot frames/ascii/*-80x24.ansi --sheet sheet-ascii.png
            (globs, not a state list: toosmall renders at its own size, 50x14, and a
            named list drops it)
            → Read the PNGs. Run the slop test (anti-patterns.md) on each.
5. CRITIQUE spawn a critic subagent (below; or its fallback) with the PNGs, goldens, TUI.md and code,
            and the user's reference when TUI.md § Reference names one (§ Hold It To The Reference)
6. FIX      one change per defect; re-run 1–4; resume the SAME critic for round 2
```

Expect 2–4 rounds. Stop when the critic says ship and your own slop test passes, and,
with a reference, its fidelity verdict says MATCH.

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

## Hold It To The Reference

When the user gives you a screen to match (an image, a GIF, frames, a mockup), the
rubric stops being the whole question: it asks "is this good?", and a coherent screen
built from the reference's ingredients scores SHIP while it misses the reference's
composition, contrast and footer. So fidelity gets its own verdict.

- **Write its terms down first**, in TUI.md § Reference: the files, the size in cells,
  the states it shows, what is `exact` and what is `inspiration`, and the deviations the
  user approved. A reference with no terms is a question for the user, not a guess.
- **Ask about conflicts once, before building.** List every place the reference breaks
  a law or a lint rule (a border the idiom drops, contrast under 2.3:1, fewer than four
  restyle moves) in one message, and record each answer in § Reference as approved or as
  the law winning. Asked one at a time, they cost the user a round each.
- **Compare at its size and state.** Render the app at the reference's size in the
  state it shows, and set the two side by side. An image's size is its pixels over the
  cell: cellshot's 12×24 default makes 960×576 an 80×24 frame. A 120×40 frame against an
  80×24 reference compares nothing: say there's no reference at that size and judge the
  larger frame as an adaptation of the matched one. For motion, compare two or more
  frames of each, or the GIF frame by frame.
- **Ask for frames when the reference came from a TUI.** `.ansi` frames (a harness's
  `-out`) render through the same cellshot and compare cell for cell; a GIF of them has
  lost the cells.
- **Claim MATCH only from a side-by-side you looked at.** A SHIP score, a passing test
  run or an earlier review says nothing about fidelity; without the comparison, say the
  fidelity is unchecked.

## The Critic

Spawn a fresh subagent. It has not seen your reasoning, so it judges the screen, not the
intent. Give it the files, not a description. With no subagent tool, the documented
fallback is a fresh message of your own: re-read the PNGs and goldens under the prompt
below and write the verdict in its format, citing only rows and cells you can see there.
It can hold you to the rubric and to what the frames show; it cannot forget your code
and reasoning, which stay in context in one session, so it grades the intent more than a
subagent would. Say in the hand-off which critic ran.

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
  - only when the user gave one: the reference (<paths>, at <size>, in <states>) and
    TUI.md § Reference: what is exact, what is inspiration, and the approved deviations
Rubric (score each 0–4, cite rows/cells, not impressions; 0 = a default or a break,
2 = works but surprises, 4 = designed):
  1. Identity: with colour stripped, is it recognisably this brand? Does the main
     screen's default frame show the read's move (a region placed where the kit
     doesn't) and its signature (an element the kit doesn't draw)? Read § Brand
     read's `skeleton:` line against that frame, axis by axis (order, verdict,
     status, head, foot, chart): a value the frame contradicts is a P1, since the
     line declares the frame as built and `diversity` trusts it; words edited to
     pass the lint, with the frame unchanged, are that P1. Then read each
     secondary screen (the dialogs, the detail) against its § Screens `brand:` line:
     do its question, buttons and footer carry the line, or are they the kit's and
     the spec's words, which another brand on the same spec would match? A line
     that promises what its frame doesn't do ("no descriptions" over a picker that
     draws one, a sentence worded otherwise in the dialog) is a defect: name the
     line and the frame. Identity that stops at the main screen scores at most 2.
     Any default (pink-purple, emoji, figlet, Loading...)?
  2. Hierarchy: 3–4 text levels used correctly; what the eye lands on first is what
     matters most; one alert at most.
  3. Focus and flow: exactly one visible focus; where am I, what can I do, how do I go
     back; help keys match what the screen actually does in this state (read the code;
     a toggle's label names the next state). Read each § Screens `flow:` line against
     its frames: a step, preview or expert path that differs from the spec's flow with
     no domain fact or brand word on its line is drift (P2); a main block's line that
     names no domain fact (who acts, on how many objects, whether the act can be taken
     back) is P2, and so is an act that reaches many objects or can't be taken back
     and shows nothing of what it will change before it runs. Hold every confirm to
     interaction.md § Confirm By Consequence, P1 on a break: depth never moves an
     act's rung, so no confirm goes in front of a rung-1 or rung-2 act or a rung-4 act
     outside its signal's state; a rung-3 act keeps its confirm or its form's review
     step, a rung-4 act in its signal's state keeps its confirm, and neither gets a
     second one after it.
  4. Layout: budgets respected; what follows the selection is one region, not split
     above and below the list (brand.md § Move A Region); 80×24, 120×40 and 200×60 are
     the same design; room used, not stretched (layout.md § Use The Room: a blank
     sidebar or unused columns beside the body at 120×40 is a stretch); nothing
     touching edges; no box soup.
  5. States: empty, loading, error, partial, too-small all designed; copy says what
     happened and what to do.
  6. Terminal craft: NO_COLOR and ASCII still readable; light set reviewed; no holes in
     filled bars; nothing animates while idle.
Output:
  - verdict first: SHIP (20+) / FIX (14–19) / RETHINK (below 14), and a total out of 24
  - with a reference, a second verdict the score can't offset: FIDELITY MATCH or
    DEVIATES, from frames of the same size and state only. For each exact axis
    (composition, contrast, artwork treatment, status placement, footer hierarchy,
    motion), the cells that differ in each frame; an approved deviation is not one
  - ≤ 10 defects ranked P0 (broken) → P3 (polish), each with the state, the cells, why,
    and the fix in renderer terms (which component, which theme key)
  - the one change that would help most
  - anything you suspect is a behaviour bug the renders hide
```

For round two, resume the **same** critic (SendMessage to it; the fallback pastes its
last verdict above the new sheets) with the new renders and "here is what changed". It
grades its own items, which keeps the bar from moving.

## Reporting To The User

Show the user the result, not the process:

- the contact sheets (80×24 and 120×40, plus light) as files they can open
- what the brand read decided, in 3–5 lines
- the critic's last verdict and score, and anything you chose not to fix (with the reason)
- with a reference, the fidelity verdict and the sheet beside it at its size; DEVIATES
  is not ready, whatever the score
- how to run it: `go run ./cmd/<app>` and the harness flags
