# restyle — bring an existing Bubble Tea app onto the system

For an app that works but looks like every other one (or like five different ones). The
goal is the same behaviour **at the idiom's richness**: a theme, a harness, and the moves
that make the kit's own screens look designed, landed in small steps that each keep it
running. A recolour is not a restyle: swapping a wizard's colours and keeping everything
else leaves the richness missing.

**Keep the libraries that work.** Huh for forms, Bubbles for lists and inputs, Glamour
for markdown: theme them from the brand theme (`components.md` § Forms And Wizards) and draw them
inside your own chrome. Rebuilding a working widget costs behaviour you'll then get wrong.

0. **Read `TUI.md` first.** If it's missing, run `init` (or its short form, saying what
   you assumed): restyling without the brand read and the screen plan only recolours.
1. **Get eyes on it first** (`cmd-critique.md` §1) and save the "before" sheets.
2. **Add the harness from the kit.** Copy `theme/` and `ui/` with the README block
   (`cmd-craft.md` §1; skip `cmd/showcase`, the app exists), then lift `main.go`'s
   harness (`render(w, h)`, `setState`, the `-golden`/`-out` flags) and
   `showcase_test.go`/`behaviour_test.go` into the app's package. Written by hand, the
   harness loses the kit's tests (`TestHelpIsHandled`, footprint, contrast). Fix
   footprint failures before anything visual.
3. **Migrate to v2** if it's on v1 (`architecture.md`): import paths, `View() tea.View`,
   `tea.KeyPressMsg`, the AltScreen/mouse view fields, Lip Gloss colours, Bubbles style
   setters. Use the official upgrade guides for the mechanics. The renders must match
   before and after.
4. **Pull every colour and glyph into a theme.** `tuilint code` lists the hex literals
   [C-HEX] and emoji [C-EMOJI]. Map each literal to a role (most map to 6–8 roles), and
   each glyph to the glyph set. If the product has a brand, run `cmd-brand.md`; if not,
   propose one.
5. **De-box.** Replace region borders with sections, rules and background steps
   (`idiom.md` §1). Keep one border, for dialogs.
6. **Adopt the components** one region at a time (the list, the footer and help from
   bindings, tags, the dialog with a scrim). Re-render after each.
7. **Raise it to the idiom.** Put the "after" sheet beside the kit's services screen and
   check each move. A restyled screen shows at least four of them; say in one line why
   any missing one doesn't fit this product.

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
9. **Loop** (`verify.md`) and show before/after sheets side by side.
