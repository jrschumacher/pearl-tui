# brand — turn a brand into a theme file that is provably not the default

Input: `TUI.md` (run `init` first if missing) and the brand assets. Output:
`themes/<brand>.json`, `themes/<brand>.read.md`, contact sheets in both modes, a passing
diversity lint. Read `references/brand.md` and `references/color.md` first.

## Steps

1. **Measure the colours.** From the logo file, the CSS or the guide. Write the source
   next to each hex in the read.
2. **Derive the roles.**
   `palette derive <primary> [<secondary>] [--accent …] --bg <brand bg> --light-bg <…> --swatch s.ansi --out roles.json`
   Render `s.ansi` with `cellshot` and look. Tune by hand where the brand demands it
   (a calmer tertiary, a warmer bg). Re-run `palette check`.
3. **Write the read** (`brand.md` §1): palette, tone, geometry, density, type, motif,
   remember, then one line per expression tying it to a trait. Use the translation
   table in `brand.md` §2 as a starting point, not an answer.
4. **Choose the signature.** The two or three moves this brand owns (a stamp header, a
   pixel banner, `▲ △` sails, lowercase titles). Everything else may stay quiet.
5. **Convert the assets.** Wordmark: `scripts/wordmark.py "<name>" --font <brand.ttf>
   --style …`. Logo, if the splash needs it: `scripts/logo2cells.py` and then
   `cell-art.md`. Motif glyph and its ASCII fallback.
6. **Write the JSON** (contract in `brand.md` §4). Start from the kit theme closest in
   spirit, then change every axis the read says to.
7. **Render** the kit showcase (or the app) with the theme: dark, light, ascii, at 80×24
   and 120×40. Contact sheets via `cellshot --sheet`.
8. **Lint.** `diversity themes/*.json kit/themes/ink.json
   kit/themes/charm.json` must pass. If it fails on distance from `ink`, identity is
   only in the palette, so go back to step 4. Also `go test` for footprint and contrast.
9. **Judge** with the four questions in `brand.md` §6, then run a critic (`verify.md`)
   focused on identity: "which product is this, and what will you remember?"
10. **Record** the theme path and decisions in `TUI.md`.

## Done Means

- The theme renders every state at both sizes in both modes, footprint tests pass, and
  `palette check` is OK.
- The diversity lint passes against the baselines.
- With colour stripped (`-profile ascii`), the screen is still recognisably this brand.
- The read explains every choice in one line.
