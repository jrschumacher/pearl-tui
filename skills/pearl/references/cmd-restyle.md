# restyle — bring an existing Bubble Tea app onto the system

*Bubble Tea: steps 2 and 3 (the kit's harness, the v1 → v2 migration), step 4's
`tuilint code` and steps 6 and 7's kit components are Bubble Tea's; steps 0, 1, 5, 8
and 9 hold for any framework.*

For an app that works but looks like every other one (or like five different ones). The
goal is the same behaviour **at the idiom's richness**: a theme, a harness, and the moves
that make the kit's own screens look designed, landed in small steps that each keep it
running. A recolour is not a restyle: swapping a wizard's colours and keeping everything
else leaves the richness missing.

**Keep the libraries that work.** Huh for the forms the app already has, Bubbles for
lists and inputs, Glamour for markdown: theme them from the brand theme (`components.md`
§ Forms And Wizards) and draw them inside your own chrome. Rebuilding a working widget
costs behaviour you'll then get wrong, and lines somebody has to review. An existing Huh
form stays on Huh, themed with `huhform.NewForm`: the kit's fields are the default for a new
form, not a reason to migrate one that works.

## Start From What The App Has

A restyle adds a theme and a harness to an app that works. It does not add a second
foundation beside the first. Before step 2, write these into the work order's `size:`
line (`collaborate.md` § Work Orders Carry The Conversation):

- **What exists**, listed: the libraries' own components in use, the app's helpers, its
  colour mapping, its root model, its tests. Each is reused unless you can name what it
  can't do. A kit component comes in for a named gap, one line each.
- **One style mapping.** `huhform.Theme` is the whole mapping of the theme onto Huh. A
  feature that needs a difference states that difference; it never copies the mapping.
- **Huh forms stay Huh forms.** Move one to the kit's fields (`ui.NewInput`,
  `ui.NewSelect`, `ui.NewConfirm`) only when the user asks, or when a Huh gap blocks the
  design (`bubbletea-architecture.md` § Huh v2: a blurred input beside another field,
  an error that must stay inside the form's height), and say which in § Decisions.
  A migration nobody asked for is a rewrite of working input code.
- **One rule, each delivery its own.** Where the app has a rich UI and a plain one (a
  line-by-line fallback), a rule both follow (what is valid, when to retry, what a
  result means) is one function both call. How keys and messages arrive stays separate:
  no shared wizard engine, no second copy of the draft.
- **What goes.** When a layout or screen is replaced, list its callers and delete what
  has none left, with any test that only kept it alive. A branch whose condition is
  never true is deleted, not kept for later.

**DON'T** wrap a library's widget, read its rendered text, or redraw part of it to match
the kit: where it falls short of the theme, say so in § Decisions and leave it native
(`bubbletea-architecture.md` § Huh v2).

0. **Read `TUI.md` first.** If it's missing and the job is one screen, run `init`'s
   short form, saying what you assumed; otherwise suggest `init` and stop: restyling
   without the brand read and the screen plan only recolours.
1. **Get eyes on it first** (`cmd-critique.md` §1) and save the "before" sheets. If the
   user gave a screen to match, write its terms in TUI.md § Reference now and ask about
   its conflicts with the laws in one message (`verify.md` § Hold It To The Reference):
   from here on it is the target, not an ingredient list.
2. **Add the harness from the kit.** Import `theme/bubbletea` and `ui/bubbletea` with the
   README block's `go get` (`cmd-craft.md` §1; skip the `cmd/showcase` and theme lines
   when the app and its theme exist; never copy the two packages in), then lift
   `main.go`'s harness (`render(w, h)`, `setState`, `states`, `overlays`, `scrolls`,
   `overlayPages`, `unjudgedPages`, `openOverlay`, the `-golden`/`-out` flags) into the
   app's package, then copy `harness_test.go` (77 lines) and point its `harnessApp`
   methods at the app's model (`kit/README.md`): its `TestHarness` runs the kit's tests,
   imported, against the app. Written by hand, the harness loses those tests
   (`HelpIsHandled`, footprint, contrast). Never copy the kit's test files into the
   app. Fix footprint failures before anything visual.
3. **Migrate to v2** if it's on v1 (`bubbletea-architecture.md`): import paths,
   `View() tea.View`, `tea.KeyPressMsg`, the AltScreen/mouse view fields, Lip Gloss
   colours, Bubbles style setters. Use the official upgrade guides for the mechanics. The
   renders must match before and after.
4. **Pull every colour and glyph into a theme.** `tuilint code` lists the hex literals
   [C-HEX] and emoji [C-EMOJI]. Map each literal to a role (most map to 6–8 roles), and
   each glyph to the glyph set. If the product has a brand, run `cmd-brand.md`; if not,
   propose one.
5. **De-box.** Replace region borders with sections, rules and background steps
   (`idiom.md` §1). Keep one border, for dialogs.
6. **Adopt the components** one region at a time (the list, the footer and help from
   bindings, tags, the dialog with a scrim). Re-render after each.
7. **Raise it to the idiom.** With an `exact` reference, put the "after" sheet beside
   the reference at its size and state instead, and take the moves below only where it
   leaves room: a move it leaves out is its choice, not a gap. Otherwise put the "after"
   sheet beside the kit's services screen and check each move. A restyled screen shows
   at least four of them; say in one line why any missing one doesn't fit this product.

   | Move | On a dashboard | On a wizard |
   |---|---|---|
   | brand header bar | `ui.Header`, product + context | the same bar, with the step: `2/6 · Authentication` |
   | progress you can read | tabs, sections | a stepper: `◆` done, `▲` current, `△` to do (theme glyphs) |
   | three or four text levels | name / value / meta / hint | label fgMuted over value fg; untouched defaults fgSubtle |
   | a background step | selected row fill, a panel | the summary on `bgPanel`, the focused field's gutter |
   | filled tags | `FAILING`, `PROD` | `DRAFT`, `DEFAULT`, `SIGNED IN` |
   | state glyphs | `▲ △ ! ×` per row | per step and per check |
   | boxed or styled keys | footer from the action table | the same footer, changing per step |
   | the motif | header rule, dialog titles | the splash or the done screen, not every step |

8. **Fix the flow while you're there.** One button order on every screen (the action
   first, the negative last), the forward action as the default on a review step, a step
   counter that keeps counting after review, and a wait that says what to do if it never
   ends (the URL to open by hand, elapsed time, cancel).
9. **Loop** (`verify.md`) and show before/after sheets side by side, and the reference
   beside the "after" when there is one, with the critic's fidelity verdict.
