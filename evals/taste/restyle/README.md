# A restyle with no harness, run cold

The `restyle-setup-no-harness` eval, run once on the skill at PR #94's capture commit
(before `F-FOOTWRAP` and `F-SPREAD` existed), by a fresh session with only the skill,
the fictional `quayside` app, its brief and the lanternfish reference.

Why it exists: a person restyled their own CLI's setup wizard toward the lanternfish
composition, and the agent, with no harness and no captures, shipped steps and a form
spread over the height, an answer orphaned under a breadcrumb and a footer wrapped at 80
columns. The fixture reproduces those flaws (`before-*.png`, captured with `tuicap`).

The cold run:
- captured every flow before touching code (5 tapes, 31 frames: `tuilint frames` 52
  blocking, all Huh's default colours and white on fuchsia; the two new rules, added
  later, find 35 more: the wrapped footer and the spread layout);
- copied the kit, added a harness, built a theme from the brief's brand, and rebuilt the
  layout from the kit's parts;
- replayed the same tapes: 0 blocking, 0 warnings on all 31 after-frames (and with the
  two new rules), in dark, light, 16 colours and no colour; the terminal left clean;
- reported fidelity per exact axis: composition, footer hierarchy, status placement and
  motion all MATCH, with the frames compared (`pairs-80.png`, `pairs-120.png`: `a` the
  reference, `b` the restyle); the critic (the session's own fallback) SHIP 22/24.

Its two notes for the skill: `verify.md` told it to lint 16-colour captures against the
theme, which an emulator's own 16 colours can't pass (fixed); and the kit's
`unjudgedPages` can't hold a Huh main screen (open).
