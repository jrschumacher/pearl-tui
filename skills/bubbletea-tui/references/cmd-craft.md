# craft — build it from the kit, verify it, then show it

Input: `TUI.md`, the theme, the screen spec (`shape`). Output: a running Bubble Tea v2
program with the harness, rendered and critiqued.

## 0. Read TUI.md First

It holds the brand read, the screens and the architecture this build must follow. If it's
missing and the job is one screen, run `init`'s short form, saying what you assumed;
otherwise suggest `init` and stop: a build with no plan invents its regions and keys
screen by screen.

## 1. Start From The Kit, Not From Prose

Run the copy block in `kit/README.md` § Copying it into a project, as written: it copies
`theme/`, `ui/` and `cmd/showcase` (as `cmd/<app>`, tests included), the donor theme as
`themes/<brand>.json` with `themes/embed.go` (the binary carries the theme: no `-theme`
flag, no path that breaks when it moves), rewrites the import paths and pins the
versions, and CI proves it passes in a fresh module. Follow its notes on skipping the
`<closest>.json` line when `brand` ran first, on what to delete once your screens are
in, and on a root model outside `package main` (an existing app's `internal/app`: the
kept tests move beside it and change only their package clause). Then read
`cmd/<app>/{app.go,screens.go,main.go}`: the model, `render(w,h)`, the harness. A cold
rebuild from these references loses half the idiom at once (the focus gutter, the
filled-row backgrounds, the help truncation, the scrim order). The code already has
them.

## 2. Skeleton First, With The Harness

Before any real screen:

- the root model (one model, a focus enum, the per-state action table, the clock with
  a generation number), and `main.go`'s package doc rewritten for this product
- the theme from `themes/` (`themes.Default`, `theme.Parse`), `-theme` only overriding it
- `render(w, h) *ui.Canvas` shared by `View` and the harness
- `setState(name)` for every state in the spec, using fixtures, and beside it in
  `main.go` the tables the kept harness tests read: `overlays` (the states that open a
  dialog, `help` among them), `scrolls` (the overlays whose body may scroll at the
  floor, drawing its position: help, a write's verbatim body), `overlayPages` (the pages
  they open over), `unjudgedPages` (Huh-form pages whose focus can't blur, each with
  why, never the main screen) and `openOverlay`. The tests then judge every overlay
  over every page with no edit, and hold the tables to the states; a test rewritten to
  skip an overlay hides the bug it was written for
- a help key in the action table of every overlay and every page that draws a focus
  mark (each of `overlayPages`): `? help`, or `ctrl+g help` where a focused text box
  types `?`. The kept tests open help from each of them and fail the one where no key
  does; leaving a focused page out of `overlayPages` to dodge that fails
  `TestOverlayTablesMatchTheStates` instead
- flags: `-golden -w -h`, `-state`, `-out -sizes`, `-theme`, `-light`, `-ascii`,
  `-profile`, `-frame`
- `_test.go` asserting the footprint at the floor `TUI.md` § Size floor names (60×18 in
  the kit) and the four sizes above it (80×24, 100×30, 120×40, 200×60), and `toosmall`
  one size below it (50×14 in the kit, checked by `-golden`/`-out`)

Run `-golden` at both sizes. It must pass while the screens are still stubs.

## 3. Screens, One At A Time

For each screen in `TUI.md` § First slice, in that order, then the `Then:` list: draw the
fixed chrome, then the body into rectangles from its § Screens block at both sizes
(`layout.md` § Use The Room); use the components it names; add the states; wire the keys
and the help. Name in its § Screens block the frame its identity shows on; the main
screen's default frame (nothing pressed) draws the read's move and signature (`brand.md`
§6, question 2), and every other screen words its question, buttons and footer labels
(the action table's help text) as its `brand:` line says. Read its render against the
line: where they disagree, change one, since the critic marks the mismatch. After each
screen, run the loop below on that screen before starting the next. The slice is the
order because it is the main loop the design gets judged on; the integrations wait
behind it.

## 4. The Loop (`verify.md`)

Render all states (dark, light, ascii, 256, 16, NO_COLOR) → golden → `tuilint frames` and
`tuilint code` → contact sheets → read the PNGs → slop test → critic subagent (or its
fallback in `verify.md` § The Critic) → fix → again. Two to four rounds.

## 5. Behaviour Checks The Renders Hide

- every key in the help works in that state (`TestHelpIsHandled`: press every action's
  keys in every state), and actions target the selection, not the first fixture row
- `tuilint spec TUI.md` passes again whenever a key changes: the table is the contract
- the ASCII render is pure ASCII (`Canvas.ASCIIfy` + a test); every visible cell has an
  explicit foreground (`TestNoUnstyledText`)
- `esc` closes or backs out everywhere; `ctrl+c` quits from everywhere
- typing `q` into an input types a q
- resize mid-dialog keeps the dialog centred and clamped
- the clock stops when nothing moves (assert `tick()` returns nil in idle states)
- a product clock survives a setting change (change the interval, deliver the armed tick,
  assert it syncs and re-arms)
- slow work runs in a `tea.Cmd`; stale results are ignored

## 6. Show The User

Show it after the first slice, before the rest, so the design is judged before the
integrations eat the effort: the contact sheets (80×24, 120×40, light), the run command,
the critic's verdict, and what you chose not to do. Append decisions to `TUI.md`.

Next: `audit` (mechanics), then `polish`; `harden` before a release.
