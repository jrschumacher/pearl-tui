# craft — build it from the kit, verify it, then show it

Input: `TUI.md`, the theme, the screen spec (`shape`). Output: a running Bubble Tea v2
program with the harness, rendered and critiqued.

*Bubble Tea: §1, §2 and §5 build on the kit, its harness and `tea.Cmd`, and §4's loop runs
`tuilint code`, all Bubble Tea v2's today; §0, §3 and §6 hold for any framework.*

## 0. Read TUI.md First

It holds the brand read, the screens and the architecture this build must follow. If it's
missing and the job is one screen, run `init`'s short form, saying what you assumed;
otherwise suggest `init` and stop: a build with no plan invents its regions and keys
screen by screen.

## 1. Start From The Kit, Not From Prose

Run the block in `kit/README.md` § Using it in a project, as written: it imports
`theme/bubbletea` and `ui/bubbletea` with `go get` (the kit is a dependency: never copy
those two packages into the project) and copies `cmd/skeleton` (as `cmd/<app>`, its
harness test included) and the donor theme as `themes/<brand>.json` with
`themes/embed.go` (the binary carries the theme: no `-theme` flag, no path that breaks
when it moves); CI proves the result passes its tests, `tuilint code` and `-golden` in a
fresh module with nothing deleted. Skip the `<closest>.json` line when `brand` ran
first; for a root model outside `package main` (an existing app's `internal/app`) follow
the README's note. Then read `cmd/<app>/{main.go,app.go,screens.go}`: the tables, the
model with its action tables, `render(w,h)`, and grow your screens into them. For a
pattern the skeleton lacks (a wizard, scrolling panes, toasts with undo, a grace window)
read the showcase (`kit/cmd/showcase`: `setup.go`, `rollout.go`, `app.go`) and build it
on your states; never copy the showcase in. A cold rebuild from these references loses
half the idiom at once (the focus gutter, the filled-row backgrounds, the help
truncation, the scrim order). The code already has them.

## 2. Skeleton First, With The Harness

Before any real screen:

- the root model (one model, a focus enum, the per-state action table, the clock with
  a generation number), and `main.go`'s package doc rewritten for this product
- the theme from `themes/` (`themes.Default`, `theme.Parse`), `-theme` only overriding it
- `render(w, h) *ui.Canvas` shared by `View` and the harness
- `setState(name)` for every state in the spec, using fixtures, and beside it in
  `main.go` the tables the harness tests read: `overlays` (the states that open a
  dialog, `help` among them), `scrolls` (the overlays whose body may scroll at the
  floor, drawing its position: help, a write's verbatim body), `overlayPages` (the pages
  they open over, a form's steps among them), `unjudgedPages` (only for an app on Huh:
  a Huh form's pages, whose focus can't blur, each with why, never the main screen) and
  `openOverlay`. The tests then judge every overlay
  over every page with no edit, and hold the tables to the states; a test rewritten to
  skip an overlay hides the bug it was written for
- a help key in the action table of every overlay but help itself and every page that
  draws a focus mark (each of `overlayPages`): `? help`, or `ctrl+g help` where a
  focused text box types `?`. The harness tests open help from each of them and fail the
  one where no key does; leaving a focused page out of `overlayPages` to dodge that
  fails `TestOverlayTablesMatchTheStates` instead
- flags: `-golden -w -h`, `-state`, `-out -sizes`, `-theme`, `-light`, `-ascii`,
  `-profile` (the skeleton's), and `-frame` once something animates (the showcase's)
- `_test.go` asserting the footprint at the floor `TUI.md` § Size floor names (60×18 in
  the kit) and the four sizes above it (80×24, 100×30, 120×40, 200×60), and `toosmall`
  one size below it (50×14 in the kit, checked by `-golden`/`-out`)

Run `-golden` at both sizes. It must pass while the screens are still stubs.

## 3. Screens, One At A Time

For each screen in `TUI.md` § First slice, in that order, then the `Then:` list: draw the
fixed chrome, then the body into rectangles from its § Screens block at both sizes
(`layout.md` § Use The Room); use the components it names; add the states; wire the keys
and the help. A form is the kit's fields (`ui.NewInput`, `ui.NewSelect`,
`ui.NewConfirm`; `components.md` § Forms And Wizards), kept in the model and drawn
blurred under a dialog like any page; Huh only where the app already has it or needs a
field the kit lacks, and `TUI.md` § Forms says which. Name in its § Screens block the frame its identity shows on; the main
screen's default frame (nothing pressed) draws the read's move and signature (`brand.md`
§6, question 2), and once it renders, the read's `skeleton:` line says what that frame
shows, axis by axis: fix the frame where it misses the move, and the line where it
guessed. Every other screen words its question, buttons and footer labels (the action
table's help text) as its `brand:` line says, and any screen takes the steps and shows
the preview its `flow:` line names. Read its render against the lines: where they
disagree, change one, since the critic marks the mismatch. After each screen, run
the loop below on that screen before starting the next. The slice is the order because
it is the main loop the design gets judged on; the integrations wait behind it.

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
the critic's verdict, and what you chose not to do. A pick point the work order names
(a splash style, a layout two arrangements pass) ships behind a harness flag per option,
shown as its own sheet, until the person picks (`collaborate.md`). Append decisions to
`TUI.md`, and rewrite § Work orders for what comes next.

Next: `audit` (mechanics), then `polish`; `harden` before a release.
