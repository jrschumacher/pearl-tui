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

Then `cellshot.py --sheet`, `tuilint.py frames` and `tuilint.py code`.

## 2. Two Independent Reviews

Spawn two critic subagents with the same files but no shared context (`verify.md` has
the prompt): one with the full rubric, one with only "you are a terminal power user
seeing this for the first time: what is confusing, slow, ugly or untrustworthy?". Where
they agree, you have high confidence. Where only one sees it, judge for yourself.

## 3. Score

| # | Dimension | 0 | 2 | 4 |
|---|---|---|---|---|
| 1 | Identity | any default (pink-purple, emoji, figlet) | branded palette only | recognisable with colour stripped |
| 2 | Hierarchy | one text level, everything bold | levels used inconsistently | the eye lands where it should |
| 3 | Focus & flow | none or two focuses, lying help | works but surprises | one focus, honest help, esc everywhere |
| 4 | Layout | tears, box soup, fixed width | works at one size | same design 80×24 → 200×60, room used |
| 5 | States | blank empty/loading/error | some designed | all designed, useful copy |
| 6 | Terminal craft | breaks on light/NO_COLOR/ascii | degrades badly | degrades gracefully, idle is still |

Total out of 24. 20+ ship, 14–19 fix, below 14 rethink.

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
