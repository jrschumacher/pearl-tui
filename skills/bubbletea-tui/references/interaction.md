# Interaction and copy: keys, focus, states, words

## Keys Are A Contract

- Define every key once as a `key.Binding` with help text. `Update` matches with
  `key.Matches`, and the footer renders the same bindings. A key in the help that doesn't
  work is a bug [C-KEYS].
- **Conventions users already have:** `↑/↓` and `j/k` move, `enter` selects/submits,
  `esc` goes back or closes, `tab`/`shift+tab` cycle focus, `/` filters, `?` toggles full
  help, `ctrl+c` quits (or confirms quitting when work would be lost), `q` quits only
  when no text input is focused, `g/G` jumps to the top/bottom, `pgup/pgdn` scrolls, and
  `ctrl+z` suspends.
- **Command palette** (`ctrl+p`, or `:`) for everything else. It lets you keep the
  footer to the 5–7 most useful keys.
- **Mnemonic single keys** for frequent actions (`d` deploy, `r` roll back), shown in the
  footer and as right-aligned hints in the palette.
- **Help changes with context.** In a dialog, the footer shows the dialog's keys. Pressing
  `esc` once with typed content shows "esc again to discard". Capability matters too:
  only advertise `shift+enter` if `SupportsKeyDisambiguation()`.
- Key order in the help: the primary action, navigation, secondary actions, then
  `? help`/`q quit` last. It truncates from the right with `…`.

## Focus

- Exactly one visible focus. It moves with `tab`, with a click, and to a dialog when one
  opens.
- Focus is shown by the gutter bar, the label weight, and the prompt colour. It never
  changes layout.
- Unfocused-but-selected stays visible in a quieter tone.
- Overlays trap focus. Keys go to the front overlay only, and `esc` closes it and
  restores the previous focus.

## Mouse

Opt in per frame (`View.MouseMode`). `CellMotion` gives clicks and wheel. Turn mouse
off when the user disables it so terminal text selection works. Make the obvious things
clickable (tabs, list rows, buttons via compositor hit regions, "N lines hidden" to
expand). Coalesce wheel events; some terminals send 3 per notch. Mouse is a convenience:
every action must also have a key.

## Async And Long Work

- Start the work as a `tea.Cmd`, mark the region busy, show a spinner with a subject, and
  keep the rest of the UI usable.
- Cancel with `esc` or `ctrl+c` (with a confirm if it would leave something half-done),
  then show "Canceled." in the region.
- Over 3 s, show elapsed time. Over 10 s with a known total, show a progress fraction.
  Over 30 s, consider letting the user leave the screen while it continues (a badge on
  the tab, a toast when done, `View.ProgramStatus` so the terminal tab shows it).
- Results that arrive for a region the user has left are stored, not drawn over the
  current screen.

## Destructive Actions

Prefer undo ("Deleted 3 files · u undo" as a toast for 8 s). When the action can't be
undone: a confirm dialog that names the object, says what changes, focuses **Cancel**,
colours the destructive button as danger, and accepts `y` for experts. For irreversible
actions on production, make the user type the object's name.

## States Every Screen Designs

Populated, empty, loading, error, partial (some items failed), offline/disconnected,
too small, and the first run. Each is a named harness state (`architecture.md`), so each
gets rendered and reviewed.

## Terminal Citizenship

- Restore the terminal on exit, panic and `ctrl+z`. Bubble Tea does this unless you
  disable it.
- Set the window title. Set the program status/progress while busy and clear it before
  exiting.
- Respect `NO_COLOR` (glyphs and words still carry meaning), `TERM=dumb`/`linux` (ASCII
  glyphs), and a light background.
- Offer a non-interactive path (`--json`, plain output when stdout isn't a TTY) for
  anything scriptable. A TUI is not an API.
- Works inside tmux/screen (no reliance on focus events or exotic keys), over SSH, and
  in a 80×24 split pane.

## Copy

Terminal copy is read at speed by people mid-task. Every word earns its cells.

- **Labels**: nouns, sentence case (or the brand's case), no colons. "Version", not
  "Version:".
- **Hints**: what is valid and what happens, in subtle, always visible. "tag or commit;
  latest is v2.4.1 (2 min ago)".
- **Errors**: what failed + against what + likely cause + what to do. "Can't reach
  prod-eu (timeout after 10s). Check the VPN, then press r to retry." Never "An error
  occurred", never a stack trace in the UI (log it, and offer `ctrl+o` to view it).
- **Empty states**: what is missing, why, in this context, and the one action. "No
  incidents declared. checkout-api has been failing for 12 min. n declare incident".
- **Spinners**: present participle + object + progress. "Deploying checkout-api 2/3
  pods". Never "Loading...", "Please wait", or "Processing".
- **Confirmations**: the question names the object ("Roll back checkout-api?") and the
  buttons are verbs ("Roll back" / "Cancel"), never "OK" / "Yes".
- **Success**: past tense + object + the fact that matters. "Rolled back checkout-api
  to v2.3.9 · 3/3 pods healthy". No "Successfully".
- **Voice** comes from the brand read: Crush's `Yep!` / `Nope` and rotating
  placeholders are a voice. A calm health product says less, more softly. A blunt
  industrial tool says it in caps and stops.
- **Numbers**: tabular and aligned, with units (`705ms`, `4.81%`, `1.2k`), relative times
  for recent things ("12m ago") and absolute for logs (`14:02:41`).
