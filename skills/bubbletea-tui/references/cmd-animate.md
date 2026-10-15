# animate — motion that means something

Read `motion.md` first, and `cell-art.md` rules 10–17 for anything bigger than a
spinner.

1. **List what actually happens over time** on each screen: work that takes > 300 ms,
   state changes the user should notice, the first launch. Only those get motion.
2. **Wire the one clock** (`motion.md`): a `gen`-stamped tick, armed only while
   something visible animates. Add a test that idle states return a nil tick.
3. **Choose the voice**: one hero animation for the main long-running action (`ui.Anim`
   with the brand ramp and a real subject), and the theme's spinner glyphs for rows.
4. **State changes**: change the glyph and colour of the cell that changed. At most a
   one-frame flash.
5. **Brand motion** (splash, intro, idle mark): follow `cell-art.md`. Precompute the
   frames, play intros once, and hand off on an identical frame.
6. **Make it deterministic**: everything renders from `frame`. Dump frames 0..N through
   the harness (`-frame`), build a GIF (`cellshot.py f*.ansi --gif out.gif --ms 50`) and
   a contact sheet, and look at them.
7. **Temporal lint**: no cell flips and flips back, no empty mid-clip frame, no run of
   identical frames > 3, and the last frame equals the resting state.
8. **Budget**: measure SSH bytes per second for the busiest state (count output bytes
   over 5 s). A spinner should cost hundreds of bytes per second, not tens of kilobytes.
