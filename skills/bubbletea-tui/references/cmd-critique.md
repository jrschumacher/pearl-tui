# critique — an adversarial design review, scored

Use on an existing TUI (yours or the user's). Output: a verdict, a score, ranked
defects that cite cells, and the one change that helps most. Don't fix anything unless
asked; hand off to `polish`, `harden` or `brand`.

## 1. Get Eyes On It

If the app has the harness: `-out frames` for every state (dark, light, ascii) at 80×24
and 120×40. If it doesn't, that is finding #1. Get frames anyway:

- run it in a pty at a fixed size and capture the screen. A short Go test driving the
  model's `View()` with `tea.WindowSizeMsg{80, 24}` and key messages, writing
  `View().Content` to `.ansi`, is usually 20 lines.
- or render the model's `View()` directly from a test for each state you can reach.

Then `cellshot --sheet`, `tuilint frames` and `tuilint code`.

## 2. Two Independent Reviews

Spawn two critic subagents with the same files but no shared context (`verify.md` has
the prompt): one with the full rubric, one with only "you are a terminal power user
seeing this for the first time: what is confusing, slow, ugly or untrustworthy?". Where
they agree, you have high confidence. Where only one sees it, judge for yourself. No
subagent tool: `verify.md` § The Critic's fallback, once per prompt, in two fresh
messages.

## 3. Score

The rubric in `verify.md` § The Critic: six dimensions (identity, hierarchy, focus and
flow, layout, states, terminal craft), 0–4 each, total out of 24. Its bands are the
verdict: 20+ ship, 14–19 fix, below 14 rethink. One rubric, so two critics score the
same screen against the same bar.

## 4. Report

```
VERDICT  FIX   17/24   (identity 2, hierarchy 3, focus 3, layout 4, states 2, craft 3)

P0  services@80×24  rows 4–12: selection is a bg change only; invisible at -profile ascii
    → selection: "bar" or add the ▌ gutter (ui.List handles both)
P1  palette@120×40  dialog has 6 blank rows below 2 results; list height jumps…
…
One change that helps most: <…>
Hand off: polish (P1–P3), brand (identity), harden (craft)
```

Defects cite the state, the size and the cells. "Looks cluttered" is not a defect; "rows
3–5 have three competing colours and no blank row between groups" is.
