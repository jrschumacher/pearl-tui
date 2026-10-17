# Architecture: Bubble Tea v2, one root model, a harness from day one

Verified against `charm.land/bubbletea/v2 v2.1.0`, `charm.land/lipgloss/v2 v2.0.6`,
`charm.land/bubbles/v2 v2.2.1`, `charm.land/huh/v2 v2.0.3` and Ultraviolet
`v0.0.0-20261008173134-6b8d4baf91b4` (October 2026). The kit's `go.mod` pins Go 1.26.7
(`GOTOOLCHAIN=auto` fetches it; no library needs it). Your training data is mostly v1: the import paths,
`View()` and the key messages all changed. Copy from the kit, which compiles.

## Pin These

```
go get charm.land/bubbletea/v2@v2.1.0 charm.land/lipgloss/v2@v2.0.6 charm.land/bubbles/v2@v2.2.1 \
       charm.land/huh/v2@v2.0.3 github.com/charmbracelet/ultraviolet@v0.0.0-20261008173134-6b8d4baf91b4
# huh is required once you copy the kit's ui/ (form.go); optional: charm.land/glamour/v2@v2.0.1
```

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

Use Huh for input mechanics and the kit's `ui/form.go` for everything it renders. Never
`huh.NewForm` bare: `tuilint code` fails it [C-HUHTHEME] and Huh's defaults render.

- `ui.NewForm(t, w, h, groups...)` themes the form from `*theme.Theme` (`ui.HuhTheme`),
  sets the kit's keys (`ui.FormKeys`) and honours `ACCESSIBLE=1` (`ui.Accessible`).
- `(*huh.Form).Update` returns `(huh.Model, tea.Cmd)` and `View()` returns a string, so a
  form is not a `tea.Model`: `f, cmd := m.form.Update(msg); m.form = f.(*huh.Form)`, and
  draw `m.form.View()` into a rectangle. Send it every non-key message too: Huh moves
  focus with commands (`NextField`), not in `Update`.
- `ui.Resize(f, w, h)` after a size change; `ui.Settle(update, cmd)` in tests to run
  Huh's follow-up commands.

Huh v2.0.3 quirks the kit already works around (each is a test; don't undo it):

| Quirk | Workaround |
|---|---|
| assumes a light terminal until a `BackgroundColorMsg` reaches the form | `HuhTheme` ignores `isDark` and uses the resolved kit theme |
| a field keeps the first theme it's given | rebuild the step's form on a theme change |
| a blurred `huh.Input` is unthemed and keeps its `> ` prompt (two `>` on screen) | `ui.NewInput` draws the blurred row itself |
| `Confirm` buttons are centred | `ui.Confirm` aligns them left; order is action first |
| the viewport fills only on `Update`, sized with the default theme | `ui.Resize` sends a no-op and sizes to content |
| validation errors land rows below the field, or off screen | `ui.Resize` counts the error rows; the theme indents them with the fail glyph |
| `/` on a select opens an unthemed filter; Huh binds its own `ctrl+c` | `ui.FormKeys` turns both off; the app owns quit |
| no descriptions beside options | `ui.Options(ui.Choice{Name, Desc, Value}...)` |
| the spinner is `#F780E2`, says "Loading...", no gap before the title | `ui.HuhSpinner(t, title)`; `ui.BubblesSpinner(t)` for Bubbles |
| accessible prompts print theme colours as raw escapes | `NewForm` uses Huh's colourless base theme when accessible |
| the input's cursor blink runs its own timer | accepted: the one exception to one clock |
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
reached that action (`kit/cmd/showcase/behaviour_test.go`, `TestHelpIsHandled`). Text
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
`View` and the harness share one `render(w, h)`, so the PNG is the program. See
`kit/cmd/showcase/main.go`. A `_test.go` renders every state at 60×18 (the floor, still
a full layout), 80×24, 100×30, 120×40 and 200×60 in every theme and mode and asserts
the footprint (`showcase_test.go`); `toosmall` is skipped there and checked by
`-golden`/`-out` at 50×14.

## Project Layout

```
cmd/<app>/main.go        flags, harness, tea.NewProgram
internal/theme/          the kit's theme package (copied), themes/<brand>.json
internal/ui/             the kit's components (copied), plus your own
internal/app/            the root model: state, Update, render(w,h), screens
internal/app/fixtures.go realistic data for every state
```
