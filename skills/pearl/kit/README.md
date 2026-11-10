# The reference kit

A compiling Go module (`github.com/jrschumacher/pearl-tui/skills/pearl/kit`) whose one framework today is
Bubble Tea v2. Import it; don't re-derive it from the references.

## Layout

- `themes/` is the theme contract: one JSON file per brand, framework-neutral. Any
  framework's kit reads the same files, and `palette`, `diversity` and `tuilint frames`
  judge them without knowing which one draws them.
- `theme/<framework>/` and `ui/<framework>/` hold one framework's code each: its theme
  engine (the contract loaded into that framework's styles) and its components. Today
  only `bubbletea` exists: `theme/bubbletea` and `ui/bubbletea`, Go packages named
  `theme` and `ui`, imported as `github.com/jrschumacher/pearl-tui/skills/pearl/kit/theme/bubbletea` and `…/ui/bubbletea`.
- `cmd/skeleton` is the smallest app on the kit's shape, which a project copies as its
  own `cmd/<app>`; `cmd/showcase` is the Bubble Tea reference app, the worked example
  to read. Both stay two levels below the module root, where a copied app's `cmd/<app>`
  sits, because the harness reads `../../themes/*.json` in every place.

A project imports one framework's packages and copies only `cmd/skeleton` (§ Using it
in a project).

```
theme/bubbletea/
                theme.go (the JSON contract), derive.go (fills missing roles, ASCII glyphs),
                styles.go (every style from the palette, Grad, Case, Label, Gap)
ui/bubbletea/   canvas.go   Canvas: draw into rectangles, Scrim, Backfill, PaintBackground, ASCIIfy, Fit, OnBg
                chrome.go   Wordmark, Header, Details, Section, Rule, Help, Footer, Toast, Tabs, TooSmall
                filter.go   Filter: a list's filter line (query, caret while typed, "3 of 8")
                toast.go    ToastTTL, UndoTTL, GraceTTL, Frames, SecondsLeft, ToastKey (a toast that
                            offers one key: u undo), Toasts and Note (the toast row: one key, a
                            grace window that holds a send, a note waiting behind it)
                controls.go Button(s), Tag, Alert, Dot, KV, Progress, Sparkline, Scrollbar, Count, Empty (NoMark: no mark)
                list.go     List (selection styles, fuzzy underline, scrollbar), Table (borderless;
                            TableWith{NoRule} without the rule under its header),
                            Follow (scroll-into-view), Fuzzy (subsequence match with indices)
                pane.go     Pane (a region that scrolls both ways: its own offsets and selection,
                            Fit, Wheel, Move, Scroll, View with the cut-edge marks), Hit (the pane
                            under a mouse cell), WheelGate (a wheel burst is one notch)
                ladder.go   Mark (a status rung in its role), Rungs (the theme's ladder, or the kit's),
                            Ladder (the legend the ladder state draws)
                dialog.go   DialogTitle, Dialog (on bgPanel, at most h rows), DialogScrolled, DialogStatus (a
                            status in the bottom edge), Popover (on bgOverlay), Clip (a scrolled body with
                            its "more" rows), Canvas.Overlay/Popover
                overlay.go  what Overlay clears of the page: the band a dialog spans, and the body with no scrim
                chat.go     Gutter, Transcript, UserMessage, AssistantMessage, ToolCall, Thinking, TurnFooter, Tree
                anim.go     FrameInterval, Anim (brand spinner or the Crush scramble), Spinner,
                            Busy (the one-cell working mark for a status column)
                field.go    Field: the two-glyph brand backdrop
                fields.go   the kit's form fields: Input (Bubbles' textinput), Select and Option,
                            Confirm, each drawn into a rectangle with its Height; Prompter
                            (their plain prompts for ACCESSIBLE=1), Accessible
                form.go     Actions (button rows), BubblesSpinner, Settle; no Huh: ui links none
                huhform/    Huh, the optional path for an app with Huh forms: Theme, Styles,
                            NewForm, Spinner (an app that never imports it never links Huh)
                steps.go    Steps (Count, Row, List), Summary, SummaryStrip: wizard chrome
                frame.go    RenderProfile, Downsample, CheckSize: what an app's harness flags need
harness/bubbletea/
                the tests every app runs against itself: an app implements harness.App,
                fills a harness.Spec and calls harness.Run from one test
themes/         ink (the fallback: theme.Default() as a file, TestInkIsTheDefault keeps them equal),
                charm (a Crush study), ferrous, lumen, halyard, moss, cartridge, terrarium;
                embed.go embeds them (themes.Default: the one theme, or ink among several)
cmd/skeleton/   the app a project copies: two pages, a confirm, help, the harness
cmd/showcase/   the Bubble Tea reference app and its harness: the worked example
```

`theme/bubbletea/theme.go` is the contract: every key the kit reads, its values and what
each draws, mirrored in `references/brand.md` §4. `palette check` fails a key or value
the kit doesn't know. `wordmark.style` is the one key nothing reads: wordmark.py records
the style it drew with; the tones come from the glyphs in the rows.

`ladder` (optional) is the product's own status ladder, `[{"glyph", "meaning", "role"}]`
in the order a thing moves through it: a glyph that is a status slot (`current`, never a
chrome glyph such as `focusBar`) or a one-cell literal with a twin, a meaning of the
product's own (`new`; never a kit slot's name such as `done`), and any colour role (or
`focus` / `working`). `ui.Rungs`, `ui.Mark(t, "new")` and the `ladder` state draw those
rungs in those inks instead of the kit's default rungs; the kit's slots keep theirs.
`theme/bubbletea/testdata/quire.json` declares one; `palette check` fails a role, glyph
or meaning the kit can't draw (`theme.CheckLadder`).

## The showcase is the pattern

The skeleton has the shape; the showcase shows it at full size. Read it for how a
pattern is built, then build that pattern on your own states.

- `app.go`: one root model; a per-state **action table** (`actions()`) that both
  dispatches keys and renders the footer; a filter on the services list (`/`, and
  `esc clear filter` only while one is in force); toasts on `ui.Toasts` with their own
  length, whose one key (`u undo`) joins the table while they show (`toastAction`, never
  a key the state binds), and a grace window that holds a send, returned as a command
  (`say`, `sayUndo`, `hold`, each returning its command); text inputs take printable keys first; one
  clock with a generation number; async work finishes by message; selection by identity.
- `screens.go`: `render(w, h)` draws every region into rectangles; header and footer
  first; overlays last; the ASCII pass at the end.
- `rollout.go`: two panes that scroll (enter on a deploy): a `ui.Pane` per pane in the
  root model, fitted from the layout on every resize and draw (`fitPanes`), the wheel
  routed to the pane under the pointer (`ui.Hit`) through a `ui.WheelGate`, a click that
  focuses, `tab` between panes, `←→` offered only while the focused pane is wide, mouse
  on for this screen alone (`View.MouseMode`), and help's Mouse section.
- `setup.go`: the first-run wizard on the kit's fields (`-first-run`): a select and an
  input on one screen, a select with a description column, two confirms, kit screens
  for review and sign-in, a step row that counts past the review, a summary that marks
  defaults, and keys only reaching a field through the action table. `ACCESSIBLE=1`
  asks the same fields as plain prompts (`ui.Prompter`).
- `main.go`: the harness (`-golden`, `-state`, `-out`, `-sizes`, `-theme`, `-light`,
  `-ascii`, `-profile`, `-frame`), `setState` for every named state, and the app's
  tables the harness reads: `states`, `overlays` (the states that open a dialog,
  `help` among them), `scrolls` (the overlays whose body may scroll at the floor, `help`
  among them), `overlayPages` (the states that draw a selection, a focused field
  or a caret), `unjudgedPages` (such states left out, each with why) and `openOverlay`.
- `harness_test.go`: the app as the kit's harness reads it (`spec`: the tables and
  the floor; `harnessApp`: the root model behind `harness.App`) and the one test,
  `TestHarness`, that runs the kit's `harness/bubbletea` package against it. That
  package holds the tests, which name no showcase state: `frames.go` (the footprint at
  5 sizes and the app's floor × every theme × 3 modes, contrast floors, no unstyled
  cells, frame names, every overlay at 16 colours, NO_COLOR and ASCII, dialogs above
  the footer, one focus column, ASCII keeps text), `overlays.go` (the tables match the
  states; every overlay over every page: the page keeps its rows, one focus, help names
  the overlay's keys and a page's), `behaviour.go` (every advertised key is handled,
  every key a toast names is in the table and works and a dialog takes it away, a grace
  window holds its send until undo, quit or its end and no other key cuts it short,
  the too-small screen's keys, ASCII is ASCII with distinct status glyphs) and
  `ansi16.go` (warning glyphs and inks at 16 colours). `go test -run
  TestHarness/HelpIsHandled` runs one.
- The showcase's own tests: `screens_test.go` (actions target the selection, inputs
  take letters, one frame clock, the refresh clock survives a rate change or pause,
  deploys finish, rollbacks don't repeat, layout holds under focus, breakpoints, tab
  stops in a tool body, Silence's undo toast, the page's grace window (its send a
  command that runs once, never once undone, its clock stopped under help), the state's
  key winning a clash with a toast's, the filter narrows the list and esc clears it),
  `rollout_test.go` (wheel, click, keys and resizes as Update receives them: the wheel
  scrolls the pane under the pointer and never the selection, shift+wheel goes
  sideways, `←→` only while wide, a resize clamps, a burst is one notch, help's Mouse
  section) and `setup_test.go` (the wizard's review defaults to Continue,
  button order, the step count past review, esc keeps input, no Bubbles or Huh default
  colour in any frame, the accessible transcript).

```sh
go test ./...
go run ./cmd/showcase -theme themes/halyard.json            # interactive
go run ./cmd/showcase -theme themes/moss.json -light -out frames -sizes 80x24,120x40
cellshot frames/*-80x24.ansi --sheet sheet.png
```

## Using it in a project

Import the kit; don't copy it. `theme` and `ui` are a dependency, so the project's diff
holds its own screens and its one theme, not 12,000 lines of the kit. From the project
root (after `go mod init <your/module>` in a new one), with `<skill>` the directory
holding `SKILL.md` and `<version>` SKILL.md's `metadata.version`:

```sh
go get github.com/jrschumacher/pearl-tui/skills/pearl/kit@v<version>
mkdir -p themes cmd && cp -r <skill>/kit/cmd/skeleton cmd/<app> && cp <skill>/kit/themes/embed.go themes/
cp <skill>/kit/themes/<closest>.json themes/<brand>.json   # skip when brand already wrote themes/<brand>.json
perl -pi -e 's#"github.com/jrschumacher/pearl-tui/skills/pearl/kit/themes"#"<your/module>/themes"#' cmd/<app>/main.go
go mod tidy && go test ./...
```

Nothing is left to delete or unwire: CI runs this block in a fresh module with three
donor themes, and the result passes `go vet`, `go test`, `tuilint code` and `-golden`
as it stands. The kit's `go.mod` brings the versions it is tested with: don't pin them
again. (`perl -pi` because `sed -i` differs on macOS; the import it rewrites sits in a
group of its own, so no `gofmt` is needed after.)

`cmd/<app>` is now the skeleton (`kit/cmd/skeleton`): a list, an item's page, a confirm
and the keys overlay, about 500 lines that are yours to rewrite. Build up from it:

- **states** go in `main.go`'s `states` (`states[0]` the main screen), each reached in
  `setState` by the keys a user presses; a dialog's also in `overlays` and
  `openOverlay`, a page that draws a focus in `overlayPages`. Keep `help` and
  `toosmall`: the tests name those two. Rewrite `main.go`'s package doc first.
- **pages and dialogs** are `screen` and `overlay` constants in `app.go`, each state with
  one action table in `actions()`: it dispatches the keys and draws the footer and help,
  so a key is added there and nowhere else. Name each dialog in `harnessApp.Overlay`.
- **regions** are methods in `screens.go`, each drawing into the rect `render` gives
  it: header and footer first, overlays last, the ASCII pass at the end.
- **fixtures** are `fixtures.go`: fictional data in the real data's shape (the longest
  name, an empty field, every status).
- **tests** of your own laws go beside `skeleton_test.go`, which drives the model by
  keys (`harness.Press`); `harness_test.go` stays.

The comments on the tables in `main.go` say what each holds;
`TestOverlayTablesMatchTheStates` fails a table that disagrees with the states, an
`unjudgedPages` entry anywhere but a Huh form's page, and `TestDialogsStayAboveTheFooter`
a dialog that doesn't fit the floor unless `scrolls` names it.

For a worked example, read the showcase (`kit/cmd/showcase`, above) and rebuild the part
you need on your states: toasts with an undo key, a grace window that holds a send and
one clock in `app.go` (with `TestGraceWindow` in `screens_test.go`); tabs, a table and
a filter in `screens.go`; panes that scroll and the mouse in `rollout.go`; a first-run
wizard on the kit's fields in `setup.go`.

What is imported stays the kit's: `…/kit/theme/bubbletea` (package `theme`) and
`…/kit/ui/bubbletea` (package `ui`). Forms use the kit's fields (`fields.go`);
`…/kit/ui/bubbletea/huhform` only themes Huh, for an app that already has Huh forms, and
an app that doesn't import it never links Huh. A component the kit lacks goes
in the project's own package, drawing with `theme` and `ui.Canvas` as the kit's do; a
change the kit itself needs is a `feedback` issue, and until it lands a `replace`
directive pointing at a patched checkout.

`harness_test.go` is the only harness code in your project. Its `TestHarness` runs the
kit's tests (imported, `harness/bubbletea`) against your app, which they read through:

- `spec` (`harness.Spec`): your tables, your floor (`minW`/`minH`), `frameSize` and
  `frameName`, and `New`, which wraps `newApp` in `harnessApp`; optionally
  `ChangesPage(overlay, page)`, true for an overlay that changes the page it opens over,
  the one pair the overlay-over-page test then skips;
- `harnessApp`, which gives the root model the methods of `harness.App`: `Theme`,
  `SetProfile`, `Render`, `SetState`, `OpenOverlay`, `Overlay` ("" for none, "help" for
  the keys overlay), `CloseOverlay`, `OverlayRect`, `Body`, `FootY`, `Key`, `Actions`
  and `ToastAction` (nil while no toast offers a key), `Toasts`, `Last` (the description
  of the last action `Key` ran), `HelpBody` and `HelpRows`. Change a method's body when
  your model names a field differently; the tests never see the field.

The tests also hold the app to `c.Body` set in `render` to the body's rect (where the
page can't be dimmed a dialog hides those rows and keeps the header); help's shape (one
row per binding, the key then what it does, `esc` back to what it was opened over); and
focus drawn in the theme's marks (the focus bar, cursor, caret or prompt glyph, or
`t.S.SelRow`'s reverse video under a `fill`/`inverse` selection). They find
`themes/*.json` two levels up from `cmd/<app>`, as from the kit's `cmd/skeleton`.

`harness_test.go` is `package main` because it sits beside the root model in
`cmd/<app>`. An app whose root model lives elsewhere (an existing app's `internal/app`)
moves it beside the model and changes its package clause (`package app`). What it reads
moves with it: the tables, `setState`, `openOverlay`, `frameSize` and `frameName` from
`main.go`. The package exports the old `main` as `func Main()`, and
`cmd/<app>/main.go` shrinks to `func main() { app.Main() }`. Keep that package two
levels below the project root, since the tests read `../../themes/*.json`. When
`unjudgedPages` is non-empty, the `huhform.NewForm` call must be in that package too: the
test that checks its entries reads only the package's own `*.go` files.

`themes/<brand>.json` is the one theme in the project: the donor is copied under your
brand's name, so `brand` edits it in place and `diversity themes/*.json` judges one
brand. When `brand` ran first, the theme is already there and the `<closest>.json`
line would overwrite the derived roles with a kit donor's: skip it. `themes/embed.go`
embeds it and `cmd/<app>/main.go` draws it with no `-theme` flag (`themes.Default`),
so the binary carries the brand wherever it is run from.
