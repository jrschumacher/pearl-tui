# Architecture: Bubble Tea v2, one root model, a harness from day one

Verified against `charm.land/bubbletea/v2 v2.1.0`, `charm.land/lipgloss/v2 v2.0.6`,
`charm.land/bubbles/v2 v2.2.1`, `charm.land/huh/v2 v2.0.3` and Ultraviolet
`v0.0.0-20261008173134-6b8d4baf91b4` (October 2026). Lip Gloss v2.0.6 needs Go ≥ 1.26.7
(`GOTOOLCHAIN=auto` fetches it). Your training data is mostly v1: the import paths,
`View()` and the key messages all changed. Copy from the kit, which compiles.

## Pin These

```
go get charm.land/bubbletea/v2@v2.1.0 charm.land/lipgloss/v2@v2.0.6 charm.land/bubbles/v2@v2.2.1 \
       github.com/charmbracelet/ultraviolet@v0.0.0-20261008173134-6b8d4baf91b4
# optional: charm.land/huh/v2@v2.0.3  charm.land/glamour/v2@v2.0.1
```

Never `github.com/charmbracelet/bubbletea` (v1) in new code [C-V1].

## v2 In One Screen

```go
type model struct{ w, h int; t *theme.Theme; file theme.File }

func (m *model) Init() tea.Cmd { return tea.RequestBackgroundColor }   // still only a Cmd

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:     m.w, m.h = msg.Width, msg.Height
    case tea.BackgroundColorMsg: m.t = theme.Resolve(m.file, msg.IsDark(), false)
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
  `shift+enter` can be told apart from `enter`. Change your help when it can.

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

- `(*huh.Form).Update` returns `(huh.Model, tea.Cmd)` and its `View()` returns a string,
  so a form is not a `tea.Model`. Embed it with `f, cmd := m.form.Update(msg); m.form =
  f.(*huh.Form)` and draw `m.form.View()` into a rectangle.
- Themes: `form.WithTheme(huh.ThemeFunc(myTheme))` where `myTheme(isDark bool)
  *huh.Styles` maps your roles onto Huh's styles. The focused field's `Base` is your focus
  bar.
- Rebuild the form when a choice changes which fields exist. Bound values must be
  pointers that outlive the form. Set the form height after theming, because Huh sizes
  its viewport with the base theme.
- For anything Huh can't show (descriptions beside options, sections that aren't pages),
  write the field yourself. The kit's deploy form is ~60 lines and fully themed.

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
`View` and the harness share one `render(w, h)`, so the PNG is the program. See
`kit/cmd/showcase/main.go`. A `_test.go` renders every state at
80×24/100×30/120×40/200×60 in every theme and mode and asserts the footprint
(`showcase_test.go`).

## Project Layout

```
cmd/<app>/main.go        flags, harness, tea.NewProgram
internal/theme/          the kit's theme package (copied), themes/<brand>.json
internal/ui/             the kit's components (copied), plus your own
internal/app/            the root model: state, Update, render(w,h), screens
internal/app/fixtures.go realistic data for every state
```
