# Architecture: Bubble Tea v2, one root model, a harness from day one

Verified against `charm.land/bubbletea/v2 v2.1.0`, `charm.land/lipgloss/v2 v2.0.6`,
`charm.land/bubbles/v2 v2.2.1`, `charm.land/huh/v2 v2.0.3` and Ultraviolet
`v0.0.0-20261008173134-6b8d4baf91b4` (October 2026). The kit's `go.mod` pins Go 1.26.7
(`GOTOOLCHAIN=auto` fetches it; no library needs it). Your training data is mostly v1: the import paths,
`View()` and the key messages all changed. Build on the kit, which compiles.

## Pin These

```
go get github.com/jrschumacher/pearl-tui/skills/pearl/kit@v<version>   # <version>: SKILL.md's metadata.version
# optional: charm.land/glamour/v2@v2.0.1
```

The kit's `go.mod` requires the versions above, so importing it brings them; an app
that doesn't import the kit pins them itself.

Never `github.com/charmbracelet/bubbletea` (v1) in new code [C-V1].

## v2 In One Screen

```go
type model struct{ w, h int; t *theme.Theme; file theme.File; prof colorprofile.Profile }

func (m *model) Init() tea.Cmd { return tea.RequestBackgroundColor }   // still only a Cmd

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:     m.w, m.h = msg.Width, msg.Height
    case tea.ColorProfileMsg:   m.prof = msg.Profile; m.t = theme.ResolveProfile(m.file, m.t.Dark, false, m.prof)
    case tea.BackgroundColorMsg: m.t = theme.ResolveProfile(m.file, msg.IsDark(), false, m.prof)
    case tea.KeyPressMsg:        // msg.String(): "ctrl+c", "shift+enter", "space" (not " ")
        if key.Matches(msg, m.keys.Quit) { return m, tea.Quit }
    case tea.PasteMsg:           _ = msg.Content            // paste is no longer a key
    case tea.MouseClickMsg:      _ = msg.X                  // also MouseReleaseMsg, MouseWheelMsg, MouseMotionMsg
    }
    return m, nil
}

func (m *model) View() tea.View {                            // returns tea.View, not string
    v := tea.NewView(m.render(m.w, m.h).String())
    v.AltScreen = true                                      // was tea.WithAltScreen()
    v.MouseMode = tea.MouseModeCellMotion                   // was tea.WithMouseCellMotion()
    v.WindowTitle = "app · " + m.ctx                        // was tea.SetWindowTitle
    v.Cursor = m.input.Cursor()                             // real cursor; nil hides it
    // v.BackgroundColor = m.t.P.Bg                         // only if expressions.paintBg
    // v.ProgressBar = tea.NewProgressBar(tea.ProgressBarIndeterminate, 0)  // while busy
    return v
}
```

- `tea.NewProgram(m).Run()` is the only way to run. `tea.Sequentially` is now
  `tea.Sequence`.
- Bubble Tea v2 downsamples colours to the terminal's profile for you. Outside a program,
  print with `lipgloss.Println` so the same happens.
- `tea.Println`/`tea.Printf` print above the program. `tea.LogToFile("debug.log", "")`
  for logs. Never `fmt.Print` while running [C-PRINT].
- `tea.Suspend` (ctrl+z), `tea.ExecProcess` (open `$EDITOR`), `tea.SetClipboard`
  (OSC 52).
- `tea.KeyboardEnhancementsMsg.SupportsKeyDisambiguation()` tells you whether
  `shift+enter` can be told apart from `enter`. Set `t.KeyDisambiguation` from it after
  every `Resolve` (the showcase's `resolve` does), and `ui.Help` draws a
  `shift+enter/ctrl+j` help key as the alternative that works.
- Mouse is per frame: set `v.MouseMode = tea.MouseModeCellMotion` in `View` only for the
  states that take it, so the terminal keeps text selection elsewhere. Clicks arrive as
  `tea.MouseClickMsg`, the wheel as `tea.MouseWheelMsg`; `msg.Mouse()` gives `X`, `Y`,
  `Button` (`tea.MouseLeft`, `MouseWheelUp`/`Down`, and `MouseWheelLeft`/`Right` for a
  trackpad's side scroll) and `Mod` (`tea.ModShift` for shift+wheel). The kit routes them
  with `ui.Hit`, `ui.WheelGate` and `ui.Pane` (components.md § Panes That Scroll; the
  showcase's `rollout.go`).

### Lip Gloss v2

- `lipgloss.Color("#hex")` returns `color.Color`. There is no `AdaptiveColor` and no
  renderer: choose light/dark with `lipgloss.LightDark(isDark)(light, dark)`, or resolve
  the whole theme on `BackgroundColorMsg` (the kit's way).
- `Blend1D`, `Blend2D`, `Lighten`, `Darken`, `Alpha`; `BorderForegroundBlend` for
  gradient borders (use sparingly); `StyleRanges` to style parts of a string;
  `Hyperlink(url)`.
- `NewLayer(...).X().Y().Z().ID()` + `NewCompositor(...)` with `.Hit(x, y)` for mouse hit
  regions over buttons.

### Bubbles v2 (gotchas the upgrade guide gets wrong)

- Textarea/textinput styles: `ta.SetStyles(textarea.DefaultStyles(isDark))`. Assigning
  `ta.Styles = …` does not compile.
- Components' `View()` still return `string`. Compose them into your canvas.
- `SetWidth`/`SetHeight` setters. `DefaultKeyMap()` is a function.
- For a real cursor: `ta.SetVirtualCursor(false)` and put `ta.Cursor()` in
  `tea.View.Cursor`, offset by where you drew the textarea.

### Huh v2

**Huh is the optional path.** A new form uses the kit's fields (`ui.NewInput`,
`ui.NewSelect`, `ui.NewConfirm`; `components.md` § Forms And Wizards): plain structs on
Bubbles' `textinput` and the kit's buttons, themed focused and blurred, with no
commands of their own to route. The showcase's wizard runs on them and doesn't import
Huh. This section is for an app that already has Huh forms, or needs a field the kit
lacks (a multi-select, a file picker, options that change as the user types): it keeps
Huh, themed through `huhform.NewForm`. An app that uses Huh imports
`github.com/jrschumacher/pearl-tui/skills/pearl/kit/ui/bubbletea/huhform`; an app that doesn't never links Huh
(`ui` imports none). These were `NewForm`, `HuhTheme`, `HuhStyles` and `HuhSpinner` in `ui`; they are
now `huhform.NewForm`, `huhform.Theme`, `huhform.Styles` and `huhform.Spinner`.

Huh keeps its own mechanics and layout: typing, focus order, validation, sizing, button
alignment. The kit gives it only the theme, through Huh's public theme API. Never
`huh.NewForm` bare: `tuilint code` fails it [C-HUHTHEME] and Huh's defaults render.
Don't wrap a field, read a form's rendered text to size it, or redraw part of it to
match the kit's look: a wrapper pins the app to one Huh release and costs more than the
difference it hides. Where Huh falls short of the theme, report it to Huh.

- `huhform.NewForm(t, w, h, groups...)` is `huh.NewForm` with the theme (`huhform.Theme`),
  Huh's help line off (the footer is the help) and `ACCESSIBLE=1` honoured
  (`ui.Accessible`). Fields are Huh's own: `huh.NewInput`, `huh.NewSelect`,
  `huh.NewConfirm`.
- `(*huh.Form).Update` returns `(huh.Model, tea.Cmd)` and `View()` returns a string, so a
  form is not a `tea.Model`: `f, cmd := m.form.Update(msg); m.form = f.(*huh.Form)`, and
  draw `m.form.View()` into a rectangle. Send it every non-key message too: Huh moves
  focus with commands (`NextField`), not in `Update`.
- A reply from a dismissed form can reach the form under it, because Huh moves focus by
  commands: check which form a reply belongs to (a generation number, as the showcase's
  sign-in does) before forwarding it (issue #96).
- Size it with its own `WithWidth` and `WithHeight`; `ui.Settle(update, cmd)` in tests to
  run Huh's follow-up commands. It runs them at once and drops whatever doesn't answer within
  `ui.SettleWait` (40 ms: timers, blinks, ticks), so a test that waits on a tick must
  send the tick's message itself.

What Huh v2.0.3 does that the theme can't change, and what to do in the app if you use
Huh (the kit's fields have none of these gaps):

| Huh | In the app |
|---|---|
| assumes a light terminal until a `BackgroundColorMsg` reaches the form | nothing: `huhform.Theme` returns the resolved theme whatever `isDark` says |
| a field keeps the first theme it's given | rebuild the step's form on a theme change |
| its theme reaches only a focused `huh.Input`; a blurred one draws in Bubbles' default colours | give a text input a group of its own, so it is never on screen blurred |
| leaves room for a group's error rows only when `WithHeight` runs, and draws one row past the height | call `WithHeight` on every draw, with one row less than the rectangle |
| the spinner is `#F780E2`, says "Loading...", no gap before the title | `huhform.Spinner(t, title)`; `ui.BubblesSpinner(t)` for Bubbles |
| accessible prompts print theme colours as raw escapes | nothing: `huhform.NewForm` uses Huh's colourless base theme when accessible |
| the input's cursor blink runs its own timer | accepted: the one exception to one frame clock (`ui.Input`'s cursor doesn't blink) |
| `Select.Value` before `Options` selects nothing | call `Options` first |

- Rebuild the form when a choice changes which fields exist. Bound values must be
  pointers that outlive the form.

## One Root Model, Regions As Plain Structs

Crush has one `*UI` model. Regions (header, list, detail, editor, status, dialogs) are
structs with a `Draw(c, rect)` or a render function, not nested `tea.Model`s with their
own `Update` you must forward to and keep in sync. Benefits: one place to route keys,
one focus enum, one clock, and no "which child gets this message" bugs.

```go
type focus int
const ( focusList focus = iota; focusDetail; focusEditor )
// keys: overlays first (only the front one), then the focused region, then globals
```

Use a nested `tea.Model` only for a self-contained Bubbles component (textarea,
viewport), and forward to it only when it is focused.

## Keys: One Action Table Per State

The most common way a TUI lies is help that drifts from handling: the footer says
`enter send` and enter does nothing, or `q` quits while the user is typing. The fix is
structural. Each state returns **one** list, and the footer and the dispatcher both
read it:

```go
type action struct {
    b    key.Binding                                  // keys + help text
    show bool                                         // in the footer (all show in the ? overlay)
    do   func(m *model, msg tea.KeyPressMsg) tea.Cmd  // what it does here
}

func (m *model) actions() []action { /* switch on overlay, then screen; most important first */ }

func (m *model) key(msg tea.KeyPressMsg) tea.Cmd {
    if msg.String() == "ctrl+c" { return tea.Quit }
    for _, a := range m.actions() {
        if key.Matches(msg, a.b) { return a.do(m, msg) }
    }
    if p := m.textTarget(); p != nil && msg.Text != "" { *p += msg.Text }  // inputs get letters
    return nil
}
```

Then a test presses every key of every action in every harness state and asserts it
reached that action (`kit/harness/bubbletea/behaviour.go`, `TestHarness/HelpIsHandled`). Text
inputs have no single-letter actions, so letters type. The `?` overlay lists every
action, including the ones not in the footer.

## Rendering: Canvas, Rectangles, Caches

- `View` creates a canvas the size of the terminal, computes the layout, and draws each
  region into its rectangle. Overlays come last, at the end of the draw order.
- Every rectangle clips, so a frame can't tear.
- Cache expensive renders (markdown, syntax highlighting, finished list items) keyed by
  (content version, width). Crush freezes finished chat items and re-renders only the
  streaming tail.
- Long lists: render only the visible slice (a virtual list). Bubbles' `list` is fine to
  ~1k items; past that, write the 60-line virtual list.

## Async: Everything Slow Is A Cmd

```go
func fetch(id string) tea.Cmd {
    return func() tea.Msg {
        res, err := api.Get(id)            // runs off the UI goroutine
        return fetchedMsg{id, res, err}
    }
}
```

- Mark the region busy and start the clock. On the result message, clear busy and stop
  the clock.
- Ignore stale results (carry an id or a generation number).
- Cancelling is a context you cancel in `Update`, plus a "Canceled." state.
- A permission prompt or dialog that opens on an async event gets a grace period: ignore
  keys typed in the ~400 ms before it could have been seen (Crush does this).

## The Harness (Non-Negotiable)

Every app built with this skill exposes, from the first commit:

```
go run ./cmd/<app>                                  interactive (the default; flags are extra)
go run ./cmd/<app> -golden -w 80 -h 24              every state as plain text; exit 1 if any row ≠ W cells or rows ≠ H
go run ./cmd/<app> -state <name> -w 120 -h 40       one state as truecolor ANSI to stdout
go run ./cmd/<app> -out frames -sizes 80x24,120x40  every state × size into frames/*.ansi
         [-theme t.json] [-light] [-ascii] [-profile ansi256|ansi|ascii] [-frame N]
```

The `states` list names every screen and overlay the app can show (including empty,
loading, error, too-small). `setState(name)` builds each from scratch with fixture data.
`overlays` names the states that open a dialog, `scrolls` those whose body may scroll
at the floor (help, and a write's verbatim body), and `overlayPages` the pages they open
over (`unjudgedPages` the Huh-form pages whose focus can't blur, with why);
`openOverlay` opens one over the page shown. The harness tests read these tables, hold
them to the states,
and name no state of an app's but `help` and `toosmall`: they are the kit's
`harness/bubbletea` package, which an app imports and runs from the one test in its
`harness_test.go` (`kit/README.md`).
`View` and the harness share one `render(w, h)`, so the PNG is the program. See
`kit/cmd/skeleton/main.go`, the smallest app that has all of it (`kit/cmd/showcase/main.go`
has the same at full size). A `_test.go` renders every state at the floor (60×18 in
the kit, still a full layout; `TUI.md` § Size floor may raise it), 80×24, 100×30,
120×40 and 200×60 in every theme and mode and asserts the footprint
(`kit/harness/bubbletea/frames.go`, the floor read from `minW`/`minH`); `toosmall` is skipped there
and checked by `-golden`/`-out` one size below the floor (50×14 in the kit).

## Project Layout

```
cmd/<app>/main.go        flags, harness, tea.NewProgram
cmd/<app>/app.go         the root model: state, Update, the action tables, the clock
cmd/<app>/screens.go     render(w,h): every region into its rectangle, overlays last
cmd/<app>/fixtures.go    realistic data for every state, in your domain's shape
cmd/<app>/harness_test.go  runs the kit's footprint, contrast and behaviour tests on this app
cmd/<app>/*_test.go      your own tests: what your screens do
themes/<brand>.json      the one theme, embedded by themes/embed.go
go.mod                   requires the kit: its theme and ui packages are imported, never copied
```

A component the kit lacks is a file in `cmd/<app>`, drawn with `theme` and `ui.Canvas`.
This is the layout of `kit/cmd/skeleton`, which `kit/README.md`'s block copies as
`cmd/<app>` and which passes the harness as copied; grow your screens into it, and read
the showcase for a pattern it lacks (a wizard, panes, toasts). One package,
`cmd/<app>`, because that is where the block puts the skeleton and its tests: a model in
`internal/app` splits the harness from its states.
