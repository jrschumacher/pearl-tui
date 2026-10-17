# tuikit: the reference kit

A compiling Bubble Tea v2 module. Copy from it; don't re-derive it from the references.

```
theme/          theme.go (the JSON contract), derive.go (fills missing roles, ASCII glyphs),
                styles.go (every style from the palette, Grad, Case, Label, Gap)
ui/             canvas.go   Canvas: draw into rectangles, Scrim, Backfill, PaintBackground, ASCIIfy, Fit, OnBg
                chrome.go   Wordmark, Header, Details, Section, Rule, Help, Footer, Toast, Tabs, TooSmall
                controls.go Button(s), Tag, Alert, Dot, KV, Progress, Sparkline, Scrollbar, Count, Empty
                list.go     List (selection styles, fuzzy underline, scrollbar), Table (borderless),
                            Follow (scroll-into-view), Fuzzy (subsequence match with indices)
                ladder.go   Mark (a status rung in its role), Ladder (the legend the ladder state draws)
                dialog.go   DialogTitle, Dialog (on bgPanel), Popover (on bgOverlay), Canvas.Overlay/Popover
                chat.go     Gutter, UserMessage, AssistantMessage, ToolCall, Thinking, TurnFooter, Tree
                anim.go     FrameInterval, Anim (brand spinner or the Crush scramble), Spinner,
                            Busy (the one-cell working mark for a status column)
                field.go    Field: the two-glyph brand backdrop
                form.go     Huh, themed: HuhTheme, NewForm, Resize, NewInput, Confirm, Options,
                            Actions (button rows), HuhSpinner, BubblesSpinner, Settle, Accessible
                steps.go    Steps (Count, Row, List), Summary, SummaryStrip: wizard chrome
themes/         ink (the fallback: theme.Default() as a file, TestInkIsTheDefault keeps them equal),
                charm (a Crush study), ferrous, lumen, halyard, moss, cartridge;
                embed.go embeds them (themes.Default: the one theme, or ink among several)
cmd/showcase/   the reference app and its harness
```

`theme/theme.go` is the contract: every key the kit reads, its values and what each
draws, mirrored in `references/brand.md` §4. `palette check` fails a key or value the
kit doesn't know. `wordmark.style` is the one key nothing reads: wordmark.py records the
style it drew with; the tones come from the glyphs in the rows.

## The showcase is the pattern

- `app.go`: one root model; a per-state **action table** (`actions()`) that both
  dispatches keys and renders the footer; text inputs take printable keys first; one
  clock with a generation number; async work finishes by message; selection by identity.
- `screens.go`: `render(w, h)` draws every region into rectangles; header and footer
  first; overlays last; the ASCII pass at the end.
- `setup.go`: the first-run wizard on Huh (`-first-run`): one themed Huh form per input
  step, kit screens for review and sign-in, a step row that counts past the review, a
  summary that marks defaults, and keys only reaching Huh through the action table.
  `ACCESSIBLE=1` runs the same fields as plain prompts.
- `main.go`: the harness (`-golden`, `-state`, `-out`, `-sizes`, `-theme`, `-light`,
  `-ascii`, `-profile`, `-frame`) and `setState` for every named state.
- `showcase_test.go`, `behaviour_test.go`: the footprint at 5 sizes × 7 themes × 3
  modes, contrast floors, no unstyled cells, ASCII is ASCII with distinct status glyphs,
  every advertised key is handled, actions target the selection, inputs take letters,
  one clock, deploys finish, rollbacks don't repeat. `setup_test.go`: the wizard's review
  defaults to Continue, button order, the step count past review, esc keeps input, no Huh
  default colour in any frame, the accessible transcript.

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
cp -r <skill>/kit/theme internal/theme && cp -r <skill>/kit/ui internal/ui
cp -r <skill>/kit/cmd/showcase cmd/<app>
cp <skill>/kit/themes/embed.go themes/ && cp <skill>/kit/themes/<closest>.json themes/<brand>.json
perl -pi -e 's#"tuikit/themes"#"<your/module>/themes"#; s#"tuikit/#"<your/module>/internal/#' $(grep -rl '"tuikit/' internal cmd)
go mod edit -go=1.26.7
go get charm.land/bubbletea/v2@v2.1.0 charm.land/lipgloss/v2@v2.0.6 \
  charm.land/bubbles/v2@v2.2.1 charm.land/huh/v2@v2.0.3 \
  github.com/charmbracelet/ultraviolet@v0.0.0-20261008173134-6b8d4baf91b4
go mod tidy && go test ./...
```

Huh is pinned too: `ui/form.go` works around v2.0.3's quirks, each with a test, and a
newer Huh can change them. CI runs this block in a fresh module on every change.

`perl -pi` because `sed -i` takes different arguments on macOS. The tests find
`themes/*.json` from `internal/ui` and `cmd/<app>` as they do from the kit, and use your
theme where they name one of the kit's, so they pass in your project as they do here.

`themes/<brand>.json` is the one theme in the project: the donor is copied under your
brand's name, so `brand` edits it in place and `diversity themes/*.json` judges one
brand. `themes/embed.go` embeds it and `cmd/<app>/main.go` draws it with no `-theme`
flag (`themes.Default`), so the binary carries the brand wherever it is run from.

Then build your root model in `cmd/<app>` on the showcase's shape (action tables,
`render`, `setState`, harness flags, the test files) and replace its screens with yours.
Rewrite the package doc at the top of `main.go` first: it still says what the showcase is.
