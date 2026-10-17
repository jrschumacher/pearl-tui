# brand — turn a brand into a theme file that is provably not the default

Input: `TUI.md` (missing: suggest `init`; its short form only for a one-screen job) and
the brand assets. Output:
`themes/<brand>.json`, the read in `TUI.md` § Brand read (the only copy: a second file
drifts), contact sheets in both modes, a passing diversity lint. Read
`references/brand.md` and `references/color.md` first.

## Steps

1. **Measure the colours.** From the logo file, the CSS or the guide. Write the source
   next to each hex in the read. No assets: start from the kit theme closest in kind to
   the read, change the hue the read names, and write `source: chosen, no assets` beside
   each hex, so nobody later reads a chosen hue as measured.
2. **Derive the roles.**
   `palette derive <primary> [<secondary>] [--accent …] --bg <brand bg> --light-bg <…> --swatch s.ansi --out roles.json`
   Render `s.ansi` with `cellshot` and look. One hue: pass it alone, and secondary and
   tertiary are lightness steps of it. The `ansi16` block in the output is the pins
   `check` would ask for: keep it. Tune by hand where the brand demands it (a calmer
   tertiary, a warmer bg). Re-run `palette check`.
3. **Write the read** into `TUI.md` § Brand read (`brand.md` §1): palette, tone,
   geometry, density, type, motif, remember, then one line per expression tying it to a
   trait. Use the translation table in `brand.md` §2 as a starting point, not an answer.
4. **Choose the signature.** The two or three moves this brand owns (a stamp header, a
   pixel banner, `▲ △` sails, lowercase titles). Everything else may stay quiet.
5. **Convert the assets.** Wordmark: `scripts/wordmark.py "<name>" --font <brand.ttf>
   --style …`. Logo, if the splash needs it: `scripts/logo2cells.py` and then
   `cell-art.md`. Motif glyph and its ASCII fallback.
6. **Write the JSON** (contract in `brand.md` §4). Start from the kit theme closest in
   spirit, then change every axis the read says to. Once `craft` has run the copy block,
   that donor is already `themes/<brand>.json`: edit it in place, and keep `themes/` to
   that one file, since `diversity` and the kit's tests count every JSON there as a brand.
7. **Render** the kit showcase (or the app) with the theme: dark, light, ascii, at 80×24
   and 120×40 (`brand.md` §6). Before `craft` copies the kit, the showcase runs from the
   skill: `go -C <skill>/kit run ./cmd/showcase -theme $PWD/themes/<brand>.json -out
   $PWD/frames/<brand> -sizes 80x24,120x40`, then the same with `-out
   $PWD/frames/<brand>-light -light` (`-C` runs it from the kit, so paths are
   absolute). Contact sheets via `cellshot --sheet`.
8. **Lint.** `diversity themes/*.json kit/themes/ink.json kit/themes/charm.json
   --baseline ink,charm --min-distance 8` must pass (`kit/` is in the skill directory;
   without the baseline files it exits 2). If it fails on distance from `ink`, identity
   is only in the palette, so go back to step 4. Also `palette check` (every role is
   `#RGB` or `#RRGGBB`), `tuilint frames frames/<brand> --theme themes/<brand>.json --bg
   <bg>` and the light frames with `--bg <light bg>`, and the footprint: `go test` once
   the kit is in the project (`craft` §1); until then the showcase's `-golden -w 80 -h
   24` and `-w 120 -h 40` from step 7's `go -C` form, which tests every state's rows.
9. **Judge** with the four questions in `brand.md` §6 (the `ladder` frame shows every
   status mark in its role), then run a critic (`verify.md`) focused on identity: "which
   product is this, and what will you remember?"
10. **Record** the theme path and decisions in `TUI.md`.

Next: `shape` (no screen spec yet) | `craft` (§ Screens is filled).

## Done Means

- The theme renders every state at both sizes in both modes, the footprint passes
  (`go test`, or the showcase's `-golden` before the kit is copied), and
  `palette check` is OK.
- The diversity lint passes against the baselines.
- With colour stripped (`-profile ascii`), the screen is still recognisably this brand.
- The read explains every choice in one line.
