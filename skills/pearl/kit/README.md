# tuikit: the reference kit

A compiling Go module whose one framework today is Bubble Tea v2. Copy from it; don't
re-derive it from the references.

## Layout

- `themes/` is the theme contract: one JSON file per brand, framework-neutral. Any
  framework's kit reads the same files, and `palette`, `diversity` and `tuilint frames`
  judge them without knowing which one draws them.
- `theme/<framework>/` and `ui/<framework>/` hold one framework's code each: its theme
  engine (the contract loaded into that framework's styles) and its components. Today
  only `bubbletea` exists: `theme/bubbletea` and `ui/bubbletea`, Go packages still named
  `theme` and `ui`, imported as `tuikit/theme/bubbletea` and `tuikit/ui/bubbletea`.
- `cmd/showcase` is the Bubble Tea showcase: the reference app and its harness. It
  stays two levels below the module root, where a copied app's `cmd/<app>` sits, because
  its kept tests read `../../themes/*.json` in both places.

A project copies one framework's packages flat (`internal/theme`, `internal/ui`), so its
own imports never name the framework.

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
                form.go     Huh, themed: HuhTheme, NewForm, Resize, NewInput, Confirm, Options,
                            Actions (button rows), HuhSpinner, BubblesSpinner, Settle, Accessible
                steps.go    Steps (Count, Row, List), Summary, SummaryStrip: wizard chrome
themes/         ink (the fallback: theme.Default() as a file, TestInkIsTheDefault keeps them equal),
                charm (a Crush study), ferrous, lumen, halyard, moss, cartridge, terrarium;
                embed.go embeds them (themes.Default: the one theme, or ink among several)
cmd/showcase/   the Bubble Tea reference app and its harness
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

- `app.go`: one root model; a per-state **action table** (`actions()`) that both
  dispatches keys and renders the footer; a filter on the services list (`/`, and
  `esc clear filter` only while one is in force); toasts on `ui.Toasts` with their own
  length, whose one key (`u undo`) joins the table while they show (`toastAction`, never
  a key the state binds), and a grace window that holds a send, returned as a command
  (`say`, `sayUndo`, `hold`, each returning its command); text inputs take printable keys first; one
  clock with a generation number; async work finishes by message; selection by identity.
- `screens.go`: `render(w, h)` draws every region into rectangles; header and footer
  first; overlays last; the ASCII pass at the end.
- `setup.go`: the first-run wizard on Huh (`-first-run`): one themed Huh form per input
  step, kit screens for review and sign-in, a step row that counts past the review, a
  summary that marks defaults, and keys only reaching Huh through the action table.
  `ACCESSIBLE=1` runs the same fields as plain prompts.
- `main.go`: the harness (`-golden`, `-state`, `-out`, `-sizes`, `-theme`, `-light`,
  `-ascii`, `-profile`, `-frame`), `setState` for every named state, and the app's
  tables the harness tests read: `states`, `overlays` (the states that open a dialog,
  `help` among them), `scrolls` (the overlays whose body may scroll at the floor, `help`
  among them), `overlayPages` (the states that draw a selection, a focused field
  or a caret), `unjudgedPages` (such states left out, each with why) and `openOverlay`.
- The harness tests, which read those tables and name no showcase state:
  `showcase_test.go` (the footprint at 5 sizes and the app's floor × every theme × 3
  modes, contrast floors, no unstyled cells, frame names, every overlay at 16 colours,
  NO_COLOR and ASCII, dialogs above the footer, one focus column, ASCII keeps text),
  `overlay_test.go` (the tables match the states; every overlay over every page: the
  page keeps its rows, one focus, help names the overlay's keys and a page's), `behaviour_test.go` (every advertised key is handled,
  every key a toast names is in the table and works and a dialog takes it away, a grace
  window holds its send until undo, quit or its end and no other key cuts it short,
  the too-small screen's keys, ASCII is ASCII with distinct status glyphs) and
  `ansi16_test.go` (warning glyphs and inks at 16 colours).
- The showcase's own tests: `screens_test.go` (actions target the selection, inputs
  take letters, one frame clock, the refresh clock survives a rate change or pause,
  deploys finish, rollbacks don't repeat, layout holds under focus, breakpoints, tab
  stops in a tool body, Silence's undo toast, the page's grace window (its send a
  command that runs once, never once undone, its clock stopped under help), the state's
  key winning a clash with a toast's, the filter narrows the list and esc clears it) and `setup_test.go` (the wizard's review defaults to Continue,
  button order, the step count past review, esc keeps input, no Huh default colour in
  any frame, the accessible transcript).

```sh
go test ./...
go run ./cmd/showcase -theme themes/halyard.json            # interactive
go run ./cmd/showcase -theme themes/moss.json -light -out frames -sizes 80x24,120x40
cellshot frames/*-80x24.ansi --sheet sheet.png
```

## Copying it into a project

From the project root (after `go mod init <your/module>` in a new one), with `<skill>` the
directory holding `SKILL.md`:

```sh
mkdir -p internal themes cmd
cp -r <skill>/kit/theme/bubbletea internal/theme && cp -r <skill>/kit/ui/bubbletea internal/ui
cp -r <skill>/kit/cmd/showcase cmd/<app>
cp <skill>/kit/themes/embed.go themes/
cp <skill>/kit/themes/<closest>.json themes/<brand>.json   # skip when brand already wrote themes/<brand>.json
perl -pi -e 's#"tuikit/themes"#"<your/module>/themes"#; s#"tuikit/(theme|ui)/bubbletea"#"<your/module>/internal/$1"#' $(grep -rl '"tuikit/' internal cmd)
gofmt -w internal cmd
go mod edit -go=1.26.7
go get charm.land/bubbletea/v2@v2.1.0 charm.land/lipgloss/v2@v2.0.6 \
  charm.land/bubbles/v2@v2.2.1 charm.land/huh/v2@v2.0.3 \
  github.com/charmbracelet/ultraviolet@v0.0.0-20261008173134-6b8d4baf91b4
go mod tidy && go test ./...
```

Huh is pinned too: `ui/bubbletea/form.go` (`internal/ui/form.go` once copied) works
around v2.0.3's quirks, each with a test, and a newer Huh can change them. CI runs this
block in a fresh module on every change.

The rewrite drops `/bubbletea` from the imports: the project holds one framework's
packages flat, as `internal/theme` and `internal/ui`, with the same package names.
`perl -pi` because `sed -i` takes different arguments on macOS, and `gofmt -w` after it
because the rewrite moves `<your/module>/themes` below `<your/module>/internal/ui` in
the import order, which `gofmt -l` would flag in CI. The tests find `themes/*.json` two
levels up from `internal/theme`, `internal/ui` and `cmd/<app>`, as they do from the
kit's `theme/bubbletea`, `ui/bubbletea` and `cmd/showcase`, and use your theme where
they name one of the kit's, so they pass in your project as they do here.

`themes/<brand>.json` is the one theme in the project: the donor is copied under your
brand's name, so `brand` edits it in place and `diversity themes/*.json` judges one
brand. When `brand` ran first, the theme is already there and the `<closest>.json`
line would overwrite the derived roles with a kit donor's: skip it. `themes/embed.go`
embeds it and `cmd/<app>/main.go` draws it with no `-theme` flag (`themes.Default`),
so the binary carries the brand wherever it is run from.

Then build your root model in `cmd/<app>` on the showcase's shape (action tables,
`render`, `setState`, harness flags, the test files) and replace its screens with yours.
Rewrite the package doc at the top of `main.go` first: it still says what the showcase is.
Once your screens are in, delete what is the showcase's alone:

- `data.go`, its fixtures (yours are `fixtures.go`), and `screens_test.go`, its tests.
  Rewrite on your states the ones whose law you share (one clock, typing never
  commands, actions target the selection; a grace window's send observed as a command
  that runs once, as `TestGraceWindow` does with `advance` and `runCmd`; and, if a
  palette command raises a toast with a key, its key checked as
  `TestShowcaseToastKeysAreHandled` does: the kept `TestToastKeysAreHandled` reaches
  the toasts your keys raise, not your commands').
- `setup.go` and `setup_test.go`, the first-run wizard, with its wiring: each `scrSetup`,
  `m.setup` and `firstRun` branch in `app.go` and `screens.go`, and in `main.go` the
  `setup-*` states, `setupState`, `-first-run` and the `runAccessibleSetup` call.
- In `main.go`, set `states`, `overlays`, `scrolls`, `overlayPages`, `unjudgedPages`
  and `openOverlay` to yours. Keep `help` as the keys overlay's state and `toosmall` as the
  too-small one: the tests name those two. `TestOverlayTablesMatchTheStates` holds the
  tables to your states: a state opens a dialog exactly when `overlays` names it, and a
  page that draws a focus mark is in `overlayPages` or renders as one that is.
  `unjudgedPages` (state → reason, repeated in TUI.md § Decisions) is only for a page
  whose focus the kit can't draw blurred, a Huh form's field: the test fails any entry
  in an app that never calls `ui.NewForm`, an empty reason, your main screen
  (`states[0]`), and a page whose focus does blur (unless its reason names Huh).
  `scrolls` is for a dialog whose body must be shown whole and has no length bound (a
  write's verbatim text): it may scroll at the floor, and `TestDialogsStayAboveTheFooter`
  holds it to drawing its position while rows are hidden (Clip's rows, or
  `ui.DialogLines` set into its bottom edge with `ui.DialogStatus`) and to reaching the
  body's end by its scroll key. Every other overlay must fit the floor.

Keep `showcase_test.go`, `overlay_test.go`, `behaviour_test.go` and `ansi16_test.go`
whole. They read your tables and these from your app:

- `newApp`, `render`, `key`, `helpBody`, `helpRows`, `minW`/`minH` (your floor), and
  `actions()`, whose entries carry the binding as `b` and whether the footer shows it
  as `show`;
- `toastAction()`, the key the showing toast offers in this state or nil, and the
  field `toasts` (`ui.Toasts`): an app that raises no toast keeps both and returns nil;
- the fields `t`, `profile`, `ov`, `last`, `ovRect`, `body` and `footY`, and the
  overlay values `ovNone` and `ovHelp`;
- the canvas's `Body`, set in `render` to the body's rect (`c.Body = body`, the same
  rect as `m.body`): where the page can't be dimmed a centred dialog hides its rows and
  keeps the header (unset, the header goes too), and the tests fail a `Body` that
  doesn't span `m.body`'s rows;
- help's shape: one row per binding, the key then what it does (a key name holds no
  space), and `esc` closing help back to whatever it was opened over;
- focus drawn in the theme's marks: the focus bar, cursor, caret or prompt glyph in its
  ink, or under a `fill`/`inverse` selection `t.S.SelRow`'s reverse video, which is
  therefore the selection's style only (a badge in it reads as a focus).

The four are `package main` because they sit beside the root model in `cmd/<app>`. An
app whose root model lives elsewhere (an existing app's `internal/app`) moves them
beside it and changes their package clause (`package app`), and that is the only edit
they take. What they read moves with them: the tables, `setState` and `openOverlay`
from `main.go`, with its `frameSize`, `frameName`, `checkSize`, `renderProfile` and
`downsample`. The package exports the old `main` as `func Main()`, and
`cmd/<app>/main.go` shrinks to `func main() { app.Main() }`. Keep that package two
levels below the project root, since the tests read `../../themes/*.json`. When
`unjudgedPages` is non-empty, the `ui.NewForm` call must be in that package too: the
test that checks its entries reads only the package's own `*.go` files.

Then delete each import `go vet ./...` reports unused. CI deletes these files from a
copy, puts a two-page app with one overlay of its own in place of the screens
(`.github/kit-stub`) and runs the kept tests with three donors, so they hold for an app
that isn't the showcase.
