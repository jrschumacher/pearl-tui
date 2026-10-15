# polish — the last 10% that makes it feel made

Run after the screens work and the critique's P0/P1s are fixed. Go through every state at
80×24 and 120×40 with the renders open. Change one thing at a time and re-render.

## Alignment And Rhythm

- [ ] One gutter everywhere. Nothing touches column 0 or the last column.
- [ ] One blank row between groups (none in compact brands). Never two.
- [ ] KV keys share a width. Numbers right-align. Times are fixed-width.
- [ ] Section rules end at the same column as the content.
- [ ] Right-aligned info (counts, times, hints) lines up down the screen.

## Hierarchy

- [ ] The first thing the eye lands on is the thing that matters (the failing row, the
      focused field, the question in the dialog).
- [ ] Values brighter than keys, metadata muted, hints subtle. Bold only for titles and
      the one important word on a row.
- [ ] At most one alert per screen. Colour only on what's abnormal.

## Identity

- [ ] The brand's signature is visible on this screen (motif, selection style, header).
- [ ] Glyphs come from the theme. One glyph per meaning across the app.
- [ ] Copy is in the brand's voice (`interaction.md` § Copy). No "Loading...", no
      "Successfully", no "An error occurred".

## Edges

- [ ] Long names truncate with `…` at a sensible point and still identify the thing.
- [ ] Filled bars (selection, footer, header) have no holes and reach the full width.
- [ ] Help ends cleanly (`…`, never a dangling separator).
- [ ] The scrollbar appears only when content overflows.
- [ ] The "N more lines" and "N lines hidden" affordances are honest.

## Motion

- [ ] Exactly one hero animation at a time, with a subject. Row spinners in primary.
- [ ] Nothing moves on an idle screen.

## Then

Re-run the loop (`verify.md`) and resume the same critic. Polish is done when the critic
has nothing above P3 and the slop test passes on every state.
