# harden — make it survive real terminals, real data, real users

Each item becomes a harness state or a test, so it stays fixed.

## Sizes

- [ ] 60×18 minimum (or your stated floor): the too-small screen below it.
- [ ] 80×24, 100×30, 120×40, 200×60, 250×70: same design, room used, nothing stretched.
- [ ] Resize mid-dialog, mid-scroll, mid-input: state survives, and the dialog re-centres.
- [ ] Height under 30: banners collapse; the footer never leaves the screen.

## Data

- [ ] Empty, one item, and `TUI.md` § Async `largest:` (10,000 items if it names none):
      virtualise; measure render time. The user's worst case, not a round number.
- [ ] The longest realistic name (60+ chars), CJK names (2 cells per rune), emoji in
      user data (render it, but never let it break a column: truncate by cells).
- [ ] Multi-line values, tabs, ANSI escape codes inside user data (strip or remap them),
      control characters.
- [ ] Partial failure: 3 of 50 items failed to load. Show what loaded and mark the rest.
- [ ] Stale data: a timestamp and a refresh key.

## Terminals

- [ ] Light background (`-light`, and real detection via `BackgroundColorMsg`).
- [ ] `-profile ansi256`, `-profile ansi`, `NO_COLOR=1`, `TERM=linux` (ASCII glyphs).
- [ ] tmux and screen (no focus events, `ctrl+arrow` may not arrive), SSH with latency
      (no full-screen animation), Windows Terminal.
- [ ] Fonts without box-drawing coverage: the ASCII glyph set is one flag away.
- [ ] Mouse off: every action still has a key.

## Behaviour

- [ ] Every async action can be cancelled; cancelling leaves a clear state.
- [ ] Network down and network slow (10 s): errors say what and offer retry; no frozen UI.
- [ ] Double-submit: Enter twice on Deploy does one deploy.
- [ ] Paste a 5,000-line blob into an input: bounded, no hang.
- [ ] `ctrl+z` (bound to `tea.Suspend`) and `fg` restore the screen; `ctrl+c` during a
      write asks first.
- [ ] Panics restore the terminal and print the error after exit, not in the frame.
- [ ] Not a TTY (`app | cat`): plain output or a clear error, never escape-code soup.

## Accessibility

- [ ] Nothing by colour alone (check the ascii renders).
- [ ] Contrast floors (`palette check`), in both modes.
- [ ] No information only in motion or only in a transient toast. Toasts also go to a log
      or history the user can open.
- [ ] An accessible mode (`--accessible`, `ACCESSIBLE=1`): no alt screen, line-by-line
      prompts, status as new lines, no repainting spinners (`interaction.md` § Screen
      Readers), plus the non-interactive `--json`/plain path for every task.
