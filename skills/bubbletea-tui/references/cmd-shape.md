# shape — design the screens on paper before any code

Input: `TUI.md` and the user's ask. Output: a screen spec the user agrees to, written
into `TUI.md` § Screens and a `SPEC.md` beside the code. No Go yet.

## For Each Screen

1. **Job**: the one thing the user comes to this screen to do, and what they look at
   first.
2. **Regions and budgets** at 80×24 and 120×40, with every pane's outer, inner and text
   width as integers (`layout.md`). Say what collapses below 110 columns and below 30
   rows.
3. **States**: populated, empty, loading, error, partial, too-small, plus every overlay
   (palette, confirm, help). Each becomes a harness state name.
4. **Keys**: the bindings, the help order, what `esc` does, and where focus starts and
   moves.
5. **Content**: realistic fixture data, including the longest name, the failing row and
   the empty list. No lorem ipsum.
6. **Components**: which kit component draws each region (`components.md`), and any new
   component you'll need. A new component is a reason to stop and think: is it a real
   need, or a box you're reaching for?
7. **Identity**: where the brand's signature shows on this screen (header, motif,
   selection style, copy).

## Sketch In Cells

Draw the 80×24 frame as text in the spec, with real content, exactly 80 wide. It is cheap
to change and it exposes width problems before code does:

```
  halyard                                                      prod / us-east-1
  Services 8 / Deploys / Incidents / Assistant
  SERVICE                                   VERSION   P99            ERRORS
  ──────────────────────────────────────────────────────────────────────────
▌ ▲ checkout-api                            v2.4.0    ▁▁▂▃▅▇██       4.81%
```

For a large or unusual screen, you may also mock it on the web inside a cell-locked
stylesheet (only `ch` and row units), then port by vocabulary. Most screens don't need
this; the kit's components are the vocabulary.

## Check With The User

Show the 80×24 sketches and the state list, and ask the one or two questions that change
the design (not "does this look good?"). Record decisions in `TUI.md` § Decisions. Then
hand off to `craft`.
