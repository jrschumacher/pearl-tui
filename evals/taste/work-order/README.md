# Work orders: a cold session against the one that had the interview

The `work-order-lanternfish` eval, run once (skill at PR #91's first commit).

1. **Interview.** One session ran `init` and `shape` on the lanternfish brief, with an
   `answers.md` that said what must stay and stood in for the person at pick points. It
   stopped at two pick points and rendered the options (`picks/`): the main screen's move
   (A, a journey column; B, an accordion) and the connecting style (A, a light on the
   route; B, a spinner line; C, the fish). It took A both times, under `## Assumed`. It
   wrote two work orders, `brand` then `craft`, each with three `keep:` lines quoting
   answers.md.
2. **Cold.** A fresh session got a copy of the repo with answers.md removed and the prompt
   `Run the work orders in TUI.md, in order.` It kept all three `keep:` lines, each held
   by a test, including the one whose quote pointed at a list it couldn't see ("the order
   above"), which it rebuilt from the spec. That miss became a rule: a quote that points
   elsewhere carries what it points at.
3. **Warm.** The interview session ran the same two orders itself.

Both TUI.md files as each session left them are here (`TUI-cold.md`,
`TUI-interview-after.md`). Round 5 (`../rounds/5`) shows them blind, seven states and one
repeat: `cold` is the fresh session, `warm` the one that had the interview.

The orders as the interview wrote them:

```
### 1. brand: themes/lanternfish.json from README § Brand, proven apart from the kit
keep: "`lanternfish init --token <t>` keeps working with no screen at all: CI runs it."
keep: "A pasted token is never shown again after it's pasted, not even masked to its last four characters."
keep: "The steps stay in the order above. Support docs are written against it."
reference: none
method: references/cmd-brand.md: palette derive …; the expressions, glyphs and ascii twins from § Brand read; render the kit showcase with it in dark, light and ascii at 80×24 and 120×40; palette check; diversity …; no pick point left in this step
deliver: themes/lanternfish.json, frames/lanternfish/, frames/lanternfish-light/, sheet-brand.png, sheet-brand-light.png
decide: nothing; build it (the move, the signature and the connecting light were taken under § Assumed from picks/)

### 2. craft: the whole setup flow, every state on fixtures
keep: (the same three)
reference: none
method: references/cmd-craft.md: copy the kit …; build the setup frame of § Screens with the new Journey and Route, then every state of § States and keys from fixtures; one action table per state; the loop (verify.md) in every profile, with a critic; the connecting light as a GIF from frames
deliver: frames/ at 80×24 and 120×40 (dark, light, 256, 16, NO_COLOR, ascii), sheet-80.png, sheet-120.png, sheet-light.png, sheet-ascii.png, connecting.gif
decide: nothing; build it
```
