# tuikit: the reference kit

A compiling Bubble Tea v2 module. Copy from it; don't re-derive it from the references.

```
theme/          theme.go (the JSON contract), derive.go (fills missing roles, ASCII glyphs),
                styles.go (every style from the palette, Grad, Case, Gap)
ui/             canvas.go   Canvas: draw into rectangles, Scrim, Backfill, PaintBackground, ASCIIfy, Fit, OnBg
                chrome.go   Wordmark, Header, Details, Section, Rule, Help, Footer, Toast, Tabs, TooSmall
                controls.go Button(s), Tag, Alert, Dot, KV, Progress, Sparkline, Scrollbar, Count, Empty
                list.go     List (selection styles, fuzzy underline, scrollbar), Table (borderless)
                dialog.go   DialogTitle, Dialog, Canvas.Overlay
                chat.go     Gutter, UserMessage, AssistantMessage, ToolCall, Thinking, TurnFooter, Tree
                anim.go     FrameInterval, Anim (brand spinner or the Crush scramble), Spinner
                field.go    Field: the two-glyph brand backdrop
themes/         ink (the fallback), charm (a Crush study), ferrous, lumen, halyard, moss, cartridge
cmd/showcase/   the reference app and its harness
```

## The showcase is the pattern

- `app.go`: one root model; a per-state **action table** (`actions()`) that both
  dispatches keys and renders the footer; text inputs take printable keys first; one
  clock with a generation number; async work finishes by message; selection by identity.
- `screens.go`: `render(w, h)` draws every region into rectangles; header and footer
  first; overlays last; the ASCII pass at the end.
- `main.go`: the harness (`-golden`, `-state`, `-out`, `-sizes`, `-theme`, `-light`,
  `-ascii`, `-profile`, `-frame`) and `setState` for every named state.
- `showcase_test.go`, `behaviour_test.go`: the footprint at 5 sizes × 7 themes × 3
  modes, contrast floors, no unstyled cells, ASCII is ASCII with distinct status glyphs,
  every advertised key is handled, actions target the selection, inputs take letters,
  one clock, deploys finish, rollbacks don't repeat.

```sh
go test ./...
go run ./cmd/showcase -theme themes/halyard.json            # interactive
go run ./cmd/showcase -theme themes/moss.json -light -out frames -sizes 80x24,120x40
cellshot frames/*-80x24.ansi --sheet sheet.png
```

## Copying it into a project

```sh
cp -r kit/theme internal/theme && cp -r kit/ui internal/ui && cp kit/themes/<closest>.json themes/
sed -i 's#"tuikit/#"<your/module>/internal/#' $(grep -rl '"tuikit/' internal)
```

Then build your root model on the showcase's shape (action tables, `render`, `setState`,
harness flags, the two test files) and replace its screens with yours.
