# craft — build it from the kit, verify it, then show it

Input: `TUI.md`, the theme, the screen spec (`shape`). Output: a running Bubble Tea v2
program with the harness, rendered and critiqued.

## 1. Start From The Kit, Not From Prose

```
cp -r <skill>/kit/theme  internal/theme      # theme engine
cp -r <skill>/kit/ui     internal/ui         # components
cp <skill>/kit/themes/<closest>.json themes/ # or the brand's theme
# read kit/cmd/showcase/{app.go,screens.go,main.go}: the model, render(w,h), harness
```

Fix the import paths (`tuikit/theme` → `<module>/internal/theme`). Pin the versions from
`architecture.md`. A cold rebuild from these references loses half the idiom at once
(the focus gutter, the filled-row backgrounds, the help truncation, the scrim order).
The code already has them.

## 2. Skeleton First, With The Harness

Before any real screen:

- the root model (one model, a focus enum, the per-state action table, the clock with
  a generation number)
- `render(w, h) *ui.Canvas` shared by `View` and the harness
- `setState(name)` for every state in the spec, using fixtures
- flags: `-golden -w -h`, `-state`, `-out -sizes`, `-theme`, `-light`, `-ascii`,
  `-profile`, `-frame`
- `_test.go` asserting the footprint at 80×24, 100×30, 120×40 and 200×60

Run `-golden` at both sizes. It must pass while the screens are still stubs.

## 3. Screens, One At A Time

For each screen in the spec: draw the fixed chrome, then the body into rectangles from
the budget; use kit components; add the states; wire the keys and the help. After each
screen, run the loop below on that screen before starting the next.

## 4. The Loop (`verify.md`)

Render all states (dark, light, ascii, 256) → golden → `tuilint frames` and
`tuilint code` → contact sheets → read the PNGs → slop test → critic subagent → fix →
again. Two to four rounds.

## 5. Behaviour Checks The Renders Hide

- every key in the help works in that state (`TestHelpIsHandled`: press every action's
  keys in every state), and actions target the selection, not the first fixture row
- the ASCII render is pure ASCII (`Canvas.ASCIIfy` + a test); every visible cell has an
  explicit foreground (`TestNoUnstyledText`)
- `esc` closes or backs out everywhere; `ctrl+c` quits from everywhere
- typing `q` into an input types a q
- resize mid-dialog keeps the dialog centred and clamped
- the clock stops when nothing moves (assert `tick()` returns nil in idle states)
- slow work runs in a `tea.Cmd`; stale results are ignored

## 6. Show The User

The contact sheets (80×24, 120×40, light), the run command, the critic's verdict, and
what you chose not to do. Append decisions to `TUI.md`.
