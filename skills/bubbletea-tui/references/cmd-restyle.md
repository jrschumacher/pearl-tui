# restyle — bring an existing Bubble Tea app onto the system

For an app that works but looks like every other one (or like five different ones). The
goal is the same behaviour with a theme, the idiom, and a harness, landed in small steps
that each keep it running.

1. **Get eyes on it first** (`cmd-critique.md` §1) and save the "before" sheets.
2. **Add the harness**: `render(w, h)`, `setState`, `-golden`/`-out` flags and the
   footprint test. Fix footprint failures before anything visual.
3. **Migrate to v2** if it's on v1 (`architecture.md`): import paths, `View() tea.View`,
   `tea.KeyPressMsg`, the AltScreen/mouse view fields, Lip Gloss colours, Bubbles style
   setters. Use the official upgrade guides for the mechanics. The renders must match
   before and after.
4. **Pull every colour and glyph into a theme.** `tuilint.py code` lists the hex literals
   [C-HEX] and emoji [C-EMOJI]. Map each literal to a role (most map to 6–8 roles), and
   each glyph to the glyph set. If the product has a brand, run `cmd-brand.md`; if not,
   propose one.
5. **De-box.** Replace region borders with sections, rules and background steps
   (`idiom.md` §1). Keep one border, for dialogs.
6. **Adopt the components** one region at a time (the list, the footer and help from
   bindings, tags, the dialog with a scrim). Re-render after each.
7. **Loop** (`verify.md`) and show before/after sheets side by side.
